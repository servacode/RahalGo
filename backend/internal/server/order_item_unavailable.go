package server

// **صنفٌ غيرُ متوفّر — المكتبُ يحذفه من الطلب** (قرارُ المالك ٢٠٢٦-١٠-١٥).
//
// **البابُ رقيقٌ والمنطقُ في المحرّك** (`orders.MarkItemUnavailable`): الحذفُ وإعادةُ
// الحساب وردُّ المحفظة والحدثُ والإشعار — **والتدقيقُ في معاملته نفسِها.**

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// handleItemUnavailable `POST /admin/orders/{id}/items/{itemID}/unavailable`.
//
// **الجسمُ اختياريّ**: `{"qty": n}` يحذف بعضَ الكمّيّة، **وبلاه يُحذف السطرُ كلُّه.**
func (s *Server) handleItemUnavailable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Qty int `json:"qty"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16)).Decode(&req); err != nil &&
		!errors.Is(err, io.EOF) {
		s.respondErr(w, errValidation)
		return
	}
	if req.Qty < 0 {
		s.respondErr(w, errValidation)
		return
	}
	orderID, itemID := chi.URLParam(r, "id"), chi.URLParam(r, "itemID")
	o, res, err := s.orders.MarkItemUnavailable(r.Context(), userIDFrom(r), orderID, itemID, req.Qty,
		func(ctx context.Context, q dbtx.Querier, res orders.ItemUnavailableResult) error {
			return s.auditTx(ctx, q, r, "ops.item_unavailable", "order", orderID, map[string]any{
				"item_id": itemID, "item_name": res.ItemName, "removed_qty": res.RemovedQty,
				"old_total": res.OldTotal, "new_total": res.NewTotal,
				"refunded": res.Refunded, "source_gone": res.SourceGone,
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
