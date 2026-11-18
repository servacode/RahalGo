package server

// ══════════════════════════════════════════════════════════════════════
// **سعرُ طلب الواتساب من الكارت ومن الكابتن** (قرارُ المالك ٢٠٢٦-١١-١٨)
// ══════════════════════════════════════════════════════════════════════
//
//	POST /admin/whatsapp-orders/{id}/goods        المكتب يكتب ثمن الطلب أو يصحّحه (بسبب) — والزبون برسالة
//	POST /driver/orders/{id}/fee-request          الكابتن يطلب أجرة توصيل جديدة بسبب
//	GET  /driver/orders/{id}/fee-request          آخر طلب إله
//	GET  /admin/orders/{id}/fee-request           المعلّق للمكتب
//	POST /admin/orders/{id}/fee-request/approve   موافقة (بمعاينة `dry_run`) — من باب «تعديل الأجرة» نفسه
//	POST /admin/orders/{id}/fee-request/reject    رفض بسبب
//
// والحسبة كلها في `orders/order_fee_request.go` و`orders/fetch_amount.go`.

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// handleOfficeWhatsAppGoods **المكتب يكتب ثمن طلب الواتساب أو يصحّحه.**
func (s *Server) handleOfficeWhatsAppGoods(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Amount   *float64 `json:"amount"`
		Currency string   `json:"currency"`
		Reason   string   `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Amount == nil || (req.Currency != "" && req.Currency != orders.CurrencySYP && req.Currency != orders.CurrencyUSD) {
		s.respondErr(w, errValidation)
		return
	}
	orderID := chi.URLParam(r, "id")
	currency := req.Currency
	if currency == "" {
		_ = s.pg.QueryRow(r.Context(), `SELECT amount_currency FROM orders WHERE id::text = $1`, orderID).Scan(&currency)
	}
	units, ok := orders.AmountUnits(*req.Amount, currency)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	o, err := s.orders.OfficeSetWhatsAppGoods(r.Context(), userIDFrom(r), orderID, units, req.Currency, req.Reason)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.whatsapp_goods_edit", "order", orderID, map[string]any{
		"amount": units, "currency": currency, "reason": req.Reason,
	})
	s.sendWhatsAppGoodsNotice(context.WithoutCancel(r.Context()), orderID)
	s.touch("order", "ops")
	view, err := orderView(orders.AudienceOps, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"order": view})
}

// handleDriverFeeRequest **الكابتن يطلب أجرة جديدة** — والجديد بيحلّ محل المعلّق.
func (s *Server) handleDriverFeeRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Fee    int64  `json:"fee"`
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	v, err := s.orders.RequestFeeChange(r.Context(), orderID, userIDFrom(r), req.Fee, req.Reason)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "driver.order_fee_request", "order", orderID, map[string]any{
		"request_id": v.ID, "old_fee": v.OldFee, "fee": v.Fee, "reason": v.Reason,
	})
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"request": v})
}

// handleDriverGetFeeRequest **آخر طلب أجرة للكابتن على هالطلب** — `request: null` ما في.
func (s *Server) handleDriverGetFeeRequest(w http.ResponseWriter, r *http.Request) {
	v, err := s.orders.DriverFeeRequest(r.Context(), chi.URLParam(r, "id"), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"request": v})
}

// handleAdminGetFeeRequest **المعلّق للمكتب** — `request: null` ما في.
func (s *Server) handleAdminGetFeeRequest(w http.ResponseWriter, r *http.Request) {
	v, err := s.orders.PendingFeeRequest(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"request": v})
}

// handleAdminApproveFeeRequest **الموافقة** — معاينةٌ بـ`dry_run` وإلّا تُطبَّق.
func (s *Server) handleAdminApproveFeeRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		RequestID string `json:"request_id"`
		DryRun    bool   `json:"dry_run"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	o, res, err := s.orders.ApproveFeeRequest(r.Context(), userIDFrom(r), orderID, req.RequestID, req.DryRun,
		func(ctx context.Context, q dbtx.Querier, res orders.EditFeeResult) error {
			return s.auditTx(ctx, q, r, "ops.order_fee_request_approve", "order", orderID, map[string]any{
				"request_id": req.RequestID, "reason": res.Reason,
				"old_fee": res.OldFee, "new_fee": res.NewFee,
				"old_driver_fee": res.OldDriverFee, "driver_fee": res.DriverFee,
				"old_total": res.OldTotal, "new_total": res.NewTotal,
				"old_cash_due": res.OldCashDue, "cash_due": res.CashDue,
			})
		})
	if err != nil {
		if err == orders.ErrFeeRequestStale {
			s.touch("order", "ops")
		}
		s.respondErr(w, err)
		return
	}
	if req.DryRun {
		httpx.JSON(w, http.StatusOK, map[string]any{"result": res})
		return
	}
	if o.Origin == orders.OriginWhatsApp {
		s.sendWhatsAppGoodsNotice(context.WithoutCancel(r.Context()), orderID)
	}
	s.touch("order", "ops")
	view, err := orderView(orders.AudienceOps, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"order": view, "result": res})
}

// handleAdminRejectFeeRequest **الرفض بسبب.**
func (s *Server) handleAdminRejectFeeRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		RequestID string `json:"request_id"`
		Reason    string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	if err := s.orders.RejectFeeRequest(r.Context(), userIDFrom(r), orderID, req.RequestID, req.Reason); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.order_fee_request_reject", "order", orderID, map[string]any{
		"request_id": req.RequestID, "reason": req.Reason,
	})
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
