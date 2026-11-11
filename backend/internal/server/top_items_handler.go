package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleMerchantTopItems GET /merchant/stores/{id}/top-items — «الأكتر مبيعاً هالأسبوع»: the
// five best sellers of the own store over the last 7 days, by delivered quantity.
// It also carries `share_url`, the store's page on the customer site (/stores/<id>) that the
// «شارك متجرك» button shares — built from the server's site URL, so staging shares staging.
func (s *Server) handleMerchantTopItems(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	items, err := s.catalog.TopSellingItems(r.Context(), merchantID, 7, 5)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"days": 7, "items": items,
		"share_url": s.siteURL() + "/stores/" + merchantID,
	})
}
