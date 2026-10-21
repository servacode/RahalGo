package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// handleDriverRequestLiveLocation **السائقُ يطلب موقعَ الزبون المباشر** — فيُسأل الزبونُ
// ثانيةً ويصله إشعار. POST /driver/orders/{id}/request-live-location
func (s *Server) handleDriverRequestLiveLocation(w http.ResponseWriter, r *http.Request) {
	o, err := s.orders.RequestLiveLocation(r.Context(), chi.URLParam(r, "id"), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"order":     orders.ViewFor(orders.AudienceDriver, o),
		"remaining": orders.LiveRequestMax - o.LiveRequestCount,
	})
}
