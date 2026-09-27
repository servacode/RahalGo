package server

// أبوابُ تسوية مستحقّات المتجر نقداً/محفظةً — للأدمن ذي الصلاحيّة الماليّة.
//
//   - تغييرُ الطريقة (نقد/محفظة): إعدادٌ ماليٌّ — صلاحيّةٌ + خطوةُ تحقّقٍ + تدقيق.
//   - كشفُ المستحقّات النقديّة: قراءةٌ ماليّة.
//   - تأكيدُ الدفع نقداً: فعلٌ ماليٌّ — صلاحيّةٌ + خطوةُ تحقّقٍ + تدقيقٌ + إشعار.

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

type setSettlementMethodReq struct {
	Method string `json:"method"`
}

// handleSetMerchantSettlementMethod يبدّل طريقةَ تسويةِ المتجر — للطلبات الجديدة فقط.
func (s *Server) handleSetMerchantSettlementMethod(w http.ResponseWriter, r *http.Request) {
	if !isUUID(chi.URLParam(r, "id")) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[setSettlementMethodReq](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	m, err := s.catalog.SetSettlementMethod(r.Context(), userIDFrom(r),
		chi.URLParam(r, "id"), req.Method, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, m)
}

// handleMerchantCashSettlements كشفُ المستحقّات النقديّة لمتجرٍ والمجموعُ القائم.
func (s *Server) handleMerchantCashSettlements(w http.ResponseWriter, r *http.Request) {
	if !isUUID(chi.URLParam(r, "id")) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	sum, err := s.orders.MerchantCashSettlements(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sum)
}

type markCashPaidReq struct {
	Note string `json:"note"`
}

// handleMarkCashSettlementPaid يؤكّد أنّ الأدمنَ سلّم المتجرَ مستحقَّه نقداً.
func (s *Server) handleMarkCashSettlementPaid(w http.ResponseWriter, r *http.Request) {
	if !isUUID(chi.URLParam(r, "id")) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[markCashPaidReq](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.orders.MarkCashSettlementPaid(r.Context(),
		chi.URLParam(r, "id"), userIDFrom(r), req.Note, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والتدقيقُ صار داخلَ معاملة الدفع** (`AQ-4`/`PF-06`): يُكتب ذرّياً مع
	// خروجِ النقد لا بأفضل جهدٍ بعده. **وإعادةُ التأكيدِ لمدفوعٍ سلفاً لا تُشعِر
	// ثانيةً** (البند ٥) — الإشعارُ وحدَه يبقى هنا (لا يمسّ الدفتر).
	if !res.AlreadyPaid && s.notify != nil && res.OwnerUserID != "" {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: res.OwnerUserID, Kind: notifications.KindWallet,
			Title:  "تم تسديد مستحقاتك النقدية",
			Body:   fmt.Sprintf("%d ل.س", res.Amount),
			Entity: "order", EntityID: res.OrderID, Href: "/portal",
			Apps: []string{notifications.AppMerchant},
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"settlement_id": res.SettlementID,
		"state":         "cash_paid",
		"amount":        res.Amount,
		"already_paid":  res.AlreadyPaid,
	})
}
