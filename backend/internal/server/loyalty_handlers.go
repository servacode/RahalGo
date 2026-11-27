package server

// ══════════════════════════════════════════════════════════════════════
// **بطاقة رحّال غو وكرت اليوم** — قرارُ المالك ٢٠٢٦-١١-٢٧
// ══════════════════════════════════════════════════════════════════════
//
// `GET /my/stamp-card` للزبون و`GET /merchant/stores/{id}/stamp-card` للمتجر — الردُّ نفسُه:
// بطاقةُ رحّال غو في الجذر، **وكرتُ اليوم في `daily`**. الحسابُ في `orders/loyalty_cards.go`.

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleMyStampCard **بطاقتا الزبون.**
func (s *Server) handleMyStampCard(w http.ResponseWriter, r *http.Request) {
	v, err := s.orders.CustomerLoyalty(r.Context(), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}

// handleMerchantStampCard **بطاقتا المتجر** — لصاحبه وموظّفيه وحدَهم (`ownsMerchant`).
func (s *Server) handleMerchantStampCard(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	v, err := s.orders.MerchantLoyalty(r.Context(), merchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, v)
}
