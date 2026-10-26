package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleWaiveCommission **«ما استلمت العمولة»** — قرارُ المالك ٢٠٢٦-١٠-٢٦.
//
// المتجرُ أخذ من السائق ثمنَ البضاعة كاملاً فلم تبقَ العمولةُ بيده: تُمحى من الطلب
// نهائيّاً، **ويُصحَّح السائقُ والمتجرُ والمندوبُ والخزينة بقيودٍ جديدةٍ في معاملةٍ
// واحدة** (`orders.WaiveCommissionTx`)، **والتدقيقُ فيها معها** — من ومتى ولماذا.
func (s *Server) handleWaiveCommission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
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
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	res, err := s.orders.WaiveCommissionTx(ctx, tx, id, userIDFrom(r), req.Reason)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.auditTx(ctx, tx, r, "finance.commission_waived", "order", id, map[string]any{
		"number": res.Number, "waived": res.Waived, "reason": req.Reason,
		"driver_id": res.DriverID, "driver_held_after": res.DriverHeld,
		"rep_id": res.RepID, "rep_reversed": res.RepReversed, "rep_debt": res.RepDebt,
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("wallet", "ops")
	s.touchUser(res.DriverID, "wallet")
	if res.RepID != "" {
		s.touchUser(res.RepID, "wallet")
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"order_id": res.OrderID, "number": res.Number,
		"waived": res.Waived, "driver_held": res.DriverHeld,
	})
}
