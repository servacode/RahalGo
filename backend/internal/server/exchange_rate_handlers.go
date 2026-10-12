package server

// ══════════════════════════════════════════════════════════════════════
//  **سعرُ صرف المتجر** (طلبُ المالك ٢٠٢٦-١٠-١٢) — انظر `catalog/usd.go`
// ══════════════════════════════════════════════════════════════════════
//
// المتجرُ يكتبه من «متجري»، والإدارةُ من ملفّ المتجر. **ومتى تبدّل أُعيد حسابُ
// كلّ صنفٍ دولاريٍّ وإضافاته لأقرب ٥ ليرات** — والليريُّ لا يُمسّ. **وقفزةٌ فوق
// ١٠٪ تُردّ بـ`exchange_rate_jump` ومثالٍ من أصنافه** حتّى تُؤكَّد بـ`confirm`.

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type exchangeRateReq struct {
	Rate    float64 `json:"rate"`
	Confirm bool    `json:"confirm"`
}

// PATCH /merchant/stores/{id}/exchange-rate
func (s *Server) handleMerchantExchangeRate(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	if err := s.merchantWriteGuard(r, merchantID); err != nil {
		s.respondErr(w, err)
		return
	}
	s.setExchangeRate(w, r, merchantID)
}

// PATCH /admin/merchants/{id}/exchange-rate
func (s *Server) handleAdminExchangeRate(w http.ResponseWriter, r *http.Request) {
	s.setExchangeRate(w, r, chi.URLParam(r, "id"))
}

func (s *Server) setExchangeRate(w http.ResponseWriter, r *http.Request, merchantID string) {
	if !isUUID(merchantID) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[exchangeRateReq](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	n, err := s.catalog.SetExchangeRate(r.Context(), userIDFrom(r), merchantID, req.Rate, req.Confirm, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("menu")
	httpx.JSON(w, http.StatusOK, map[string]any{"exchange_rate": req.Rate, "items_repriced": n})
}
