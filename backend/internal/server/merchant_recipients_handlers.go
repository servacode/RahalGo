package server

// ══════════════════════════════════════════════════════════════════════
// **مستلمو «لدي توصيلة» السابقون — بابا صاحب المتجر** (قرارُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
//	المتجر  GET    /merchant/stores/{id}/delivery-recipients?q=…         ثمانيةٌ على الأكثر
//	المتجر  DELETE /merchant/stores/{id}/delivery-recipients/{phone}     «شيلو من القائمة»
//
// **والحارسُ حارسُ إنشاء التوصيلة نفسُه** (`ownsMerchant`) — صاحبُ المتجر وموظّفوه
// (يعملون باسمه)، **ومن لا يملك المتجرَ يُردّ ٤٠٣** ولا يعرف أنّ لأحدٍ مستلِمين.
// **ولا بوّابةَ دوام**: الاقتراحُ قراءةٌ تُعين على الكتابة، والإرسالُ وحدَه يُحرَس بها.
// والمنطقُ في `orders/merchant_recipients.go`.

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleMerchantDeliveryRecipients **مستلمو المتجر السابقون الأقربُ لما كُتب.**
func (s *Server) handleMerchantDeliveryRecipients(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	list, err := s.orders.MerchantRecipients(r.Context(), merchantID, r.URL.Query().Get("q"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"recipients": list})
}

// handleMerchantHideDeliveryRecipient **«شيلو من القائمة»** — ويرجع إن بعث له من جديد.
func (s *Server) handleMerchantHideDeliveryRecipient(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	// **و`+` يصل مرمَّزاً من المتصفّح** (`%2B963…`) — وchi يردّ الجزءَ كما وصل حينها.
	phone := chi.URLParam(r, "phone")
	if p, err := url.PathUnescape(phone); err == nil {
		phone = p
	}
	if err := s.orders.HideMerchantRecipient(r.Context(), merchantID, phone); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"hidden": true})
}
