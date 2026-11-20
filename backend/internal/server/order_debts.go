package server

// «ديون الطلبات» — قرار المالك ٢٠٢٦-١١-٢٠ (orders/order_driver_debts.go).
//
//	GET  /admin/finance/order-debts                    كل كابتن إله شي عالمكتب
//	GET  /admin/finance/order-debts/{driverID}         طلباته
//	POST /admin/finance/order-debts/{driverID}/pay     «تم الدفع» — {ids?: [...]} وبلاها كل المفتوح
//	GET  /driver/order-debts                           الكابتن: قدّيش إله عالمكتب وطلباته

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

func (s *Server) handleOrderDebtsSummary(w http.ResponseWriter, r *http.Request) {
	rows, err := s.orders.OrderDebtsSummary(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var open int64
	for _, d := range rows {
		open += d.OpenTotal
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"drivers": rows, "open_total": open})
}

func (s *Server) handleOrderDebtsOf(w http.ResponseWriter, r *http.Request) {
	rows, err := s.orders.OrderDebtsOf(r.Context(), chi.URLParam(r, "driverID"), 200)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": rows})
}

func (s *Server) handlePayOrderDebts(w http.ResponseWriter, r *http.Request) {
	driverID := chi.URLParam(r, "driverID")
	req, err := decode[struct {
		IDs []string `json:"ids"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var total int64
	var n int
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		var err error
		total, n, err = s.orders.PayOrderDebtsTx(ctx, q, driverID, req.IDs, userIDFrom(r))
		if err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "finance.order_debts_paid", "user", driverID, map[string]any{
			"amount": total, "count": n,
		})
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"paid_total": total, "paid_count": n})
}

func (s *Server) handleDriverOrderDebts(w http.ResponseWriter, r *http.Request) {
	me := userIDFrom(r)
	total, n, err := s.orders.OrderDebtsOpenTotal(r.Context(), me)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.orders.OrderDebtsOf(r.Context(), me, 50)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"open_total": total, "open_count": n, "items": rows})
}
