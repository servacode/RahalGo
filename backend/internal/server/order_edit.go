package server

// ══════════════════════════════════════════════════════════════════════
// **تعديلُ الطلب العاديّ** — قرارُ المالك ٢٠٢٦-١١-٠٣
// ══════════════════════════════════════════════════════════════════════
//
// بابان: **دليلُ ما يُضاف** (أصنافُ متاجر الطلب)، **والتعديلُ نفسُه** — بمعاينةٍ
// (`dry_run`) تُحسب كلُّها في الخادم ثمّ تُطوى، فيرى المكتبُ الرقمَ قبل أن يقع.
// والحسبةُ والقيودُ في `orders/order_edit.go`.

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// handleOrderEditCatalog **أصنافُ متاجر الطلب** — ما يُختار منه للإضافة.
func (s *Server) handleOrderEditCatalog(w http.ResponseWriter, r *http.Request) {
	cat, err := s.orders.EditCatalog(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, cat)
}

// handleOrderEdit **يعدّل الطلبَ أو يعاينه.**
func (s *Server) handleOrderEdit(w http.ResponseWriter, r *http.Request) {
	var req orders.EditRequest
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16)).Decode(&req); err != nil {
		s.respondErr(w, errValidation)
		return
	}
	orderID := chi.URLParam(r, "id")
	// **وبعد الاستلام قدرةٌ أضيق** (قرارُ المالك، البند ٤) — يقرّرها الخادمُ بالطلب لا الشاشة.
	o, res, err := s.orders.EditOrder(r.Context(), userIDFrom(r), orderID, req,
		s.hasCapability(r, authz.OrdersEditAfterPickup),
		func(ctx context.Context, q dbtx.Querier, res orders.EditResult) error {
			if res.DryRun {
				return nil
			}
			return s.auditTx(ctx, q, r, "ops.order_edit", "order", orderID, map[string]any{
				"reason": res.Reason, "after_pickup": res.AfterPickup,
				"old_total": res.OldTotal, "new_total": res.NewTotal,
				"old_cash_due": res.OldCashDue, "cash_due": res.CashDue,
				"refunded": res.Refunded, "charged": res.Charged,
				"changes": res.Changes, "stores": res.Stores,
			})
		})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.DryRun {
		httpx.JSON(w, http.StatusOK, map[string]any{"result": res})
		return
	}
	s.touch("order", "ops")
	view, err := orderView(orders.AudienceOps, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"order": view, "result": res})
}
