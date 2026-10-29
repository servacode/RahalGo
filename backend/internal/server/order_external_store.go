package server

// **«من متجر غير تابع لنا»** — قرارُ المالك ٢٠٢٦-١٠-٢٩.
//
// **البابُ رقيقٌ والمنطقُ في المحرّك** (`orders.ConvertToExternalStore`): الشروطُ والقلبُ
// والحالُ والإشعارات — **والتدقيقُ في معاملته نفسِها.** ومن بعدها يمشي الطلبُ في مسار
// المشتريات القائم — **فلا مالَ يُحسب هنا.**

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// رموزُ القلب — **للوحة وحدَها** (`SERVER_ADMIN` في `web/scripts/check-app-error-codes.mjs`).
var extStoreErrors = []struct {
	sentinel error
	reply    error
}{
	{orders.ErrExtStoreNotStandard, httpx.NewError(http.StatusConflict,
		"ext_store_not_standard", "errors.ext_store_not_standard")},
	{orders.ErrExtStoreBadStatus, httpx.NewError(http.StatusConflict,
		"ext_store_bad_status", "errors.ext_store_bad_status")},
	{orders.ErrExtStoreWallet, httpx.NewError(http.StatusConflict,
		"ext_store_wallet", "errors.ext_store_wallet")},
	{orders.ErrExtStoreMoneyMoved, httpx.NewError(http.StatusConflict,
		"ext_store_money_moved", "errors.ext_store_money_moved")},
}

// handleOrderExternalStore `POST /admin/orders/{id}/external-store` — `{"reason": "…"}`.
func (s *Server) handleOrderExternalStore(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	o, _, err := s.orders.ConvertToExternalStore(r.Context(), userIDFrom(r), orderID, req.Reason,
		func(ctx context.Context, q dbtx.Querier, ch orders.ExtStoreChange) error {
			return s.auditTx(ctx, q, r, "ops.order_external_store", "order", orderID, map[string]any{
				"from_merchant":      ch.FromMerchantID,
				"from_merchant_name": ch.FromMerchantName,
				"item_stores":        ch.ItemStores,
				"from_status":        ch.FromStatus,
				"to_status":          ch.ToStatus,
				"reason":             ch.Reason,
				"released_promo":     ch.ReleasedPromo,
				"fee_snapshot":       ch.FeeSnapshot,
			})
		})
	if err != nil {
		for _, e := range extStoreErrors {
			if errors.Is(err, e.sentinel) {
				err = e.reply
				break
			}
		}
		s.respondErr(w, err)
		return
	}
	// **وغرفُ المتاجر التي كانت تراه أُشيرت في المحرّك** (`ExtStoreChange.Stores`) — فتُسقطه شاشاتُها.
	s.touch("order", "ops")
	view, err := orderView(orders.AudienceOps, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"order": view})
}
