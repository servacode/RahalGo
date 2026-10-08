package server

// **رابطُ موقع المستلِم ← نقطة** (طلبُ المالك ٢٠٢٦-١٠-٠٨): المتجرُ يلصق ما شاركه
// الزبونُ بواتساب في «لدي توصيلة»، **فيُقرأ هنا نقطةً تُرسَل مع التوصيلة** —
// والسائقُ يرى نقطةً على خريطتنا كأيّ طلب. انظر `geolink`.

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/geolink"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

func (s *Server) handleMerchantResolveLocation(w http.ResponseWriter, r *http.Request) {
	if !s.ownsMerchant(r, chi.URLParam(r, "id")) {
		s.respondErr(w, errForbidden)
		return
	}
	req, err := decode[struct {
		URL string `json:"url"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if len(req.URL) > 2000 {
		s.respondErr(w, errValidation)
		return
	}
	lat, lng, err := geolink.Resolve(r.Context(), nil, req.URL)
	if err != nil {
		if !errors.Is(err, geolink.ErrNoLocation) {
			s.logger.Warn("رابطُ الموقع لم يُقرأ", "error", err)
		}
		// **ردٌّ صريحٌ لا خطأ** — الشاشةُ تقول «هالرابط ما فيه موقع».
		httpx.JSON(w, http.StatusOK, map[string]any{"found": false})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"found": true, "lat": lat, "lng": lng})
}
