package server

import (
	"net/http"
	"slices"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// كشف محفظة مستخدم — أدمن/مالية/عمليات (قراءة).
func (s *Server) handleAdminWalletStatement(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	st, err := s.wallet.StatementFor(r.Context(), chi.URLParam(r, "id"), limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

// حركة يدوية على المحفظة (شحن/تعويض/تسوية/سحب) — أدمن/مالية فقط.
func (s *Server) handleAdminWalletApply(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Amount int64  `json:"amount"`
		Kind   string `json:"kind"`
		Debit  bool   `json:"debit"` // للتسوية فقط: سحب بدل إيداع
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// الحركات اليدوية المسموحة من اللوحة فقط — حركات الطلبات يصدرها المحرك
	if !slices.Contains([]string{"topup", "compensation", "adjustment", "payout"}, req.Kind) {
		s.respondErr(w, errValidation)
		return
	}
	// المبلغ يُرسل موجباً دائماً، والإشارة تُشتق في الخادم من النوع —
	// لا نثق بإشارة العميل (دفتر قيود سلامته حرجة).
	if req.Amount <= 0 {
		s.respondErr(w, wallet.ErrInvalidAmount)
		return
	}
	amount := req.Amount
	if req.Kind == "payout" || (req.Kind == "adjustment" && req.Debit) {
		amount = -amount
	}
	if !isUUID(chi.URLParam(r, "id")) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	actor := userIDFrom(r)
	balance, err := s.wallet.Apply(r.Context(), chi.URLParam(r, "id"),
		amount, req.Kind, "", req.Note, &actor)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// صاحب المحفظة يعرف فوراً بأي إيداع/خصم — شفافية مالية بلا تحديث صفحة.
	title := notifTitles.walletCredit
	if amount < 0 {
		title = notifTitles.walletDebit
	}
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: chi.URLParam(r, "id"), Kind: notifications.KindWallet,
		Title: title, Body: req.Note,
		Entity: "wallet", Href: "/wallet",
	})
	httpx.JSON(w, http.StatusOK, map[string]any{"balance": balance})
}
