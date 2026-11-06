package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleMerchantNoPhotoItems lists the store's items without a photo for the fast photo mode
// (owner decision 2026-11-06). Own store only; a keyset page via `after` (last item id shown).
// The photo itself is saved with the existing PATCH /merchant/menu/items/{itemID} (image_media_id).
func (s *Server) handleMerchantNoPhotoItems(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	page, err := s.catalog.ListNoPhotoItems(r.Context(), merchantID, r.URL.Query().Get("after"), limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, page)
}
