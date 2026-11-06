package server

// ══════════════════════════════════════════════════════════════════════
// **أفعالُ المكتب على قسم متجرٍ في الطلب بعدّة محطّات** (عقدُ اللوحة §2)
// ══════════════════════════════════════════════════════════════════════
//
//	POST /admin/orders/{id}/parts/{storeID}/accept    {prep_minutes?}   قبولُ القسم والطلبُ «بانتظار»
//	POST /admin/orders/{id}/parts/{storeID}/reject    {reason}          رفضُه قبل قبول الطلب
//	POST /admin/orders/{id}/parts/{storeID}/remove    {reason, fault}   شيلُه — والذنبُ يختاره المكتب
//	POST /admin/orders/{id}/parts/{storeID}/transfer  {to_merchant_id}  تحويلُه لمتجرٍ آخر
//
// **البابُ رقيقٌ والمنطقُ في المحرّك** (`orders/parts_remove.go`) — **والتدقيقُ في معاملة
// الفعل نفسِه** حيث يُتاح (الشيلُ والتحويل). **وقدرتُها `orders.intervene`** كجيرانها.

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

var (
	// errPartCollected **السائقُ استلم قسمَ هذا المتجر** — لا يُشال ولا يُحوَّل.
	errPartCollected = httpx.NewError(http.StatusConflict, "part_collected", "errors.part_collected")
	// errTransferTargetInOrder **المتجرُ الجديدُ في الطلب أصلاً.**
	errTransferTargetInOrder = httpx.NewError(http.StatusConflict,
		"transfer_target_in_order", "errors.transfer_target_in_order")
	// errUsePartTransfer **الطلبُ من أكثر من متجر يُحوَّل قسماً قسماً.**
	errUsePartTransfer = httpx.NewError(http.StatusConflict, "use_part_transfer", "errors.use_part_transfer")
)

// partAppErr **أخطاءُ المحرّك في الأقسام برموزها** — من `respondErr` لكلّ باب، فلا يُنسى بابٌ.
func partAppErr(err error) *httpx.AppError {
	var unmatched *orders.PartTransferUnmatched
	switch {
	case errors.Is(err, orders.ErrPartCollected):
		return errPartCollected
	case errors.Is(err, orders.ErrTransferTargetInOrder):
		return errTransferTargetInOrder
	case errors.Is(err, orders.ErrPartFault):
		return errValidation
	case errors.As(err, &unmatched):
		return &httpx.AppError{
			Status: http.StatusConflict, Code: "transfer_items_unmatched",
			MessageKey: "errors.transfer_items_unmatched",
			Details:    map[string]any{"items": unmatched.Items},
		}
	}
	return nil
}

// partTarget **الطلبُ ومتجرُ القسم من المسار** — ومعرّفٌ لا شكلَ له «غير موجود».
func partTarget(r *http.Request) (orderID, storeID string, ok bool) {
	orderID, storeID = chi.URLParam(r, "id"), strings.ToLower(chi.URLParam(r, "storeID"))
	return orderID, storeID, isUUID(orderID) && isUUID(storeID)
}

// adminPartDone **ردُّ فعل القسم** — حمولةُ المكتب بأقسامها.
func (s *Server) adminPartDone(w http.ResponseWriter, r *http.Request, orderID string) {
	s.touch("order", "ops")
	o, err := s.orders.GetByID(r.Context(), orderID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	view, err := orderView(orders.AudienceOps, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, view)
}

// handleAdminPartAccept `POST /admin/orders/{id}/parts/{storeID}/accept`.
func (s *Server) handleAdminPartAccept(w http.ResponseWriter, r *http.Request) {
	orderID, storeID, ok := partTarget(r)
	if !ok {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		PrepMinutes *int `json:"prep_minutes"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.PrepMinutes != nil && (*req.PrepMinutes < 0 || *req.PrepMinutes > 600) {
		s.respondErr(w, errValidation)
		return
	}
	if _, err := s.orders.AnswerParts(r.Context(), userIDFrom(r), rolesFrom(r), orderID,
		[]string{storeID}, true, req.PrepMinutes, ""); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.part_accepted", "order", orderID, map[string]any{
		"merchant_id": storeID, "prep_minutes": req.PrepMinutes,
	})
	s.adminPartDone(w, r, orderID)
}

// handleAdminPartReject `POST /admin/orders/{id}/parts/{storeID}/reject` — قبل قبول الطلب.
func (s *Server) handleAdminPartReject(w http.ResponseWriter, r *http.Request) {
	orderID, storeID, ok := partTarget(r)
	if !ok {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		s.respondErr(w, errReasonRequired)
		return
	}
	if _, err := s.orders.AnswerParts(r.Context(), userIDFrom(r), rolesFrom(r), orderID,
		[]string{storeID}, false, nil, reason); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.part_rejected", "order", orderID, map[string]any{
		"merchant_id": storeID, "reason": reason,
	})
	s.adminPartDone(w, r, orderID)
}

// handleAdminPartRemove `POST /admin/orders/{id}/parts/{storeID}/remove` — `fault` «merchant» أو «platform».
func (s *Server) handleAdminPartRemove(w http.ResponseWriter, r *http.Request) {
	orderID, storeID, ok := partTarget(r)
	if !ok {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Reason string `json:"reason"`
		Fault  string `json:"fault"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		s.respondErr(w, errReasonRequired)
		return
	}
	if req.Fault != orders.FaultMerchant && req.Fault != orders.FaultPlatform {
		s.respondErr(w, errValidation)
		return
	}
	if _, err := s.orders.RemovePart(r.Context(), userIDFrom(r), rolesFrom(r), orderID, storeID,
		reason, req.Fault, func(ctx context.Context, q dbtx.Querier) error {
			return s.auditTx(ctx, q, r, "ops.part_removed", "order", orderID, map[string]any{
				"merchant_id": storeID, "reason": reason, "fault": req.Fault,
			})
		}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.adminPartDone(w, r, orderID)
}

// handleAdminPartTransfer `POST /admin/orders/{id}/parts/{storeID}/transfer` — والمطابقةُ في المحرّك.
func (s *Server) handleAdminPartTransfer(w http.ResponseWriter, r *http.Request) {
	orderID, storeID, ok := partTarget(r)
	if !ok {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		ToMerchantID string `json:"to_merchant_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	to := strings.ToLower(strings.TrimSpace(req.ToMerchantID))
	if !isUUID(to) {
		s.respondErr(w, errValidation)
		return
	}
	if to == storeID {
		s.respondErr(w, errSameMerchant)
		return
	}
	_, res, err := s.orders.TransferPart(r.Context(), userIDFrom(r), rolesFrom(r), orderID, storeID, to, "",
		func(ctx context.Context, q dbtx.Querier) error {
			return s.auditTx(ctx, q, r, "ops.part_transferred", "order", orderID, map[string]any{
				"from": storeID, "to": to,
			})
		})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Info("الأقسام: حُوّل قسمٌ لمتجرٍ آخر", "order", orderID, "items", res.Items)
	s.adminPartDone(w, r, orderID)
}
