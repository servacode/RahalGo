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
	s.respondDrinks(w, r, merchantID)
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
	s.writeDrinks(w, r, merchantID)
}

// handleRepGetDrinks **قائمةُ مشروبات عميله** — والحارسُ `repMenuGuard` على المسار
// (طلبُ المالك ٢٠٢٦-١٠-١٩: «تطبيق المندوب لازم يكون مطابق للمتجر»).
func (s *Server) handleRepGetDrinks(w http.ResponseWriter, r *http.Request) {
	s.respondDrinks(w, r, chi.URLParam(r, "id"))
}

// respondDrinks **القائمةُ ومعها `allowed`** — هل هذا المتجرُ مطعم (قرارُ المالك
// ٢٠٢٦-١٠-٣٠: المشروباتُ للمطاعم وحدَها). **فتُخفي التطبيقاتُ أداةَ المشروبات
// ومفتاحَ «أضف مشروباتي» لغيره**، ولغيره القائمةُ فارغة.
func (s *Server) respondDrinks(w http.ResponseWriter, r *http.Request, merchantID string) {
	allowed, err := s.catalog.DrinksAllowed(r.Context(), merchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	drinks, err := s.catalog.Drinks(r.Context(), merchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"drinks": drinks, "allowed": allowed})
}

// handleRepSetDrinks **يكتبها المندوبُ لعميله كما يكتبها صاحبُه** — المنطقُ واحد.
func (s *Server) handleRepSetDrinks(w http.ResponseWriter, r *http.Request) {
	s.writeDrinks(w, r, chi.URLParam(r, "id"))
}

// writeDrinks **الكتابةُ نفسُها للمتجر والمندوب** — حارسُ النصّ ثمّ النسخُ إلى الأصناف.
func (s *Server) writeDrinks(w http.ResponseWriter, r *http.Request, merchantID string) {
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
	allowed, err := s.catalog.DrinksAllowed(r.Context(), merchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"drinks": drinks, "allowed": allowed})
}

// handleAdminGetDrinks **قائمةُ مشروبات المتجر للإدارة** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «المجموعات بتطبيق
// المتجر نفسهن لازم يكونو باللوحة»).
func (s *Server) handleAdminGetDrinks(w http.ResponseWriter, r *http.Request) {
	s.handleRepGetDrinks(w, r)
}

// handleAdminSetDrinks **تكتبها الإدارةُ كما يكتبها صاحبُ المتجر** — والكتابةُ نفسُها.
func (s *Server) handleAdminSetDrinks(w http.ResponseWriter, r *http.Request) {
	s.writeDrinks(w, r, chi.URLParam(r, "id"))
}
