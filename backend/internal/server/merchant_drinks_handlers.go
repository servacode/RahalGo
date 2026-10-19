package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/textguard"
)

// handleMerchantGetDrinks **قائمةُ مشروبات المتجر** (قرارُ المالك ٢٠٢٦-١٠-١٩).
func (s *Server) handleMerchantGetDrinks(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	drinks, err := s.catalog.Drinks(r.Context(), merchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"drinks": drinks})
}

// handleMerchantSetDrinks **يستبدل القائمةَ وينسخها إلى كلّ الأصناف.**
//
// **والموقوفُ لا يكتبها** (A4) — **والأسماءُ تمرّ بحارس النصّ** كأسماء الأصناف.
func (s *Server) handleMerchantSetDrinks(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if err := s.merchantWriteGuard(r, merchantID); err != nil {
		s.respondErr(w, err)
		return
	}
	req, err := decode[struct {
		Drinks []catalog.Drink `json:"drinks"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for i := range req.Drinks {
		if _, err := s.guardText(r.Context(),
			tf("name", &req.Drinks[i].Name, s.itemNameMax(r.Context()), textguard.Name),
		); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	drinks, err := s.catalog.SetDrinks(r.Context(), userIDFrom(r), merchantID, req.Drinks, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("menu", "merchant:"+merchantID)
	httpx.JSON(w, http.StatusOK, map[string]any{"drinks": drinks})
}
