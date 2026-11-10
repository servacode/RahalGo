package server

// ══════════════════════════════════════════════════════════════════════
// **طلب تعديل من الكابتن — والمكتب يوافق أو يرفض** (قرارُ المالك ٢٠٢٦-١١-١٠)
// ══════════════════════════════════════════════════════════════════════
//
// الكابتن يبعت (`POST /driver/orders/{id}/edit-request`) ويقرأ آخر طلب إله (`GET`). والمكتب يقرأ
// المعلّق بمعاينة المحرّك (`GET /admin/orders/{id}/edit-request`) ويوافق أو يرفض. **والموافقة بتمشي
// من باب تعديل الطلب نفسه** وبقدرته بعد الاستلام (`orders.edit_after_pickup`). والحسبة كلها في
// `orders/order_edit_request.go`.

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// handleDriverEditRequest **الكابتن يبعت طلب تعديل** — والجديد بيحلّ محل المعلّق.
func (s *Server) handleDriverEditRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[orders.DriverEditInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	// **الفحص ومعاينة المحرّك قبل المعاملة**، والحفظ بمعاملة منع التكرار (`WithIdempotentTx`):
	// ردّ ضايع بيُعاد بنفس المفتاح فيرجع نفس الطلب ولا ينحفظ مرّتين.
	p, err := s.orders.PrepareOrderEdit(r.Context(), orderID, userIDFrom(r), *req)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		v, err := s.orders.SaveOrderEditTx(ctx, q, p)
		if err != nil {
			return IdempotentBody{}, err
		}
		if err := s.auditTx(ctx, q, r, "driver.order_edit_request", "order", orderID, map[string]any{
			"request_id": v.ID, "old_total": v.OldTotal, "new_total": v.NewTotal,
			"lines": v.Lines, "adds": v.Adds, "note": v.Note,
		}); err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{
			Status:  http.StatusOK,
			Payload: map[string]any{"request": v},
			AfterCommit: func() {
				s.orders.AfterOrderEdit(context.WithoutCancel(ctx), p)
				s.touch("order", "ops")
			},
		}, nil
	})
}

// handleDriverGetEditRequest **آخر طلب تعديل بعته الكابتن على هالطلب** — و`null`: ما في.
func (s *Server) handleDriverGetEditRequest(w http.ResponseWriter, r *http.Request) {
	v, err := s.orders.DriverEditRequest(r.Context(), chi.URLParam(r, "id"), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"request": v})
}

// handleAdminGetEditRequest **الطلب المعلّق بمعاينة المحرّك** — ومين بيقدر يوافق.
func (s *Server) handleAdminGetEditRequest(w http.ResponseWriter, r *http.Request) {
	v, err := s.orders.PendingEditRequest(r.Context(), chi.URLParam(r, "id"), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"request": v,
		// **بعد الاستلام للأدمن والمالك بس** (البند ٤) — الشاشة بتقول هالشي بدل زر بيرد ٤٠٣.
		"can_approve_after_pickup": s.hasCapability(r, authz.OrdersEditAfterPickup),
	})
}

// handleAdminApproveEditRequest **المكتب يوافق** — التعديل من باب `EditOrder` بالتدقيق نفسه.
func (s *Server) handleAdminApproveEditRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		RequestID string `json:"request_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	o, res, err := s.orders.ApproveEditRequest(r.Context(), userIDFrom(r), orderID, req.RequestID,
		s.hasCapability(r, authz.OrdersEditAfterPickup),
		func(ctx context.Context, q dbtx.Querier, res orders.EditResult) error {
			return s.auditTx(ctx, q, r, "ops.order_edit_request_approved", "order", orderID, map[string]any{
				"request_id": req.RequestID,
				"reason":     res.Reason, "after_pickup": res.AfterPickup,
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
	s.touch("order", "ops")
	view, err := orderView(orders.AudienceOps, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"order": view, "result": res})
}

// handleAdminRejectEditRequest **المكتب يرفض بسبب** — والطلب ما بيتغيّر.
func (s *Server) handleAdminRejectEditRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		RequestID string `json:"request_id"`
		Reason    string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	if err := s.orders.RejectEditRequest(r.Context(), userIDFrom(r), orderID, req.RequestID, req.Reason,
		func(ctx context.Context, q dbtx.Querier) error {
			return s.auditTx(ctx, q, r, "ops.order_edit_request_rejected", "order", orderID, map[string]any{
				"request_id": req.RequestID, "reason": req.Reason,
			})
		}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"rejected": true})
}
