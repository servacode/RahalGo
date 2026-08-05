package server

// نقاطُ العروض — **بابٌ عند الإدارة وبابٌ عند الزبون.**
//
// # والزبونُ يرى الساريَ وحدَه
//
// **عرضٌ انتهى يُعرض ثمّ لا يُطبَّق خدعةٌ لا خطأ**: يضغطه فيجد السعرَ كما كان،
// **فيظنّ أنّنا نكذب** — وهو محقّ.
//
// # وسعرُ العرض من الخادم
//
// **الهامشُ يُضاف على سعر الشراء** (`pricing`)، والخصمُ يُطرح من الناتج.
// **وحسبةٌ في متصفّحٍ تفترق عمّا يُقيَّد في الطلب** — فيرى سعراً ويُحاسَب بآخر.

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/offers"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// saleOf سعرُ البيع من سعر الشراء — **بالقاعدة نفسِها التي يُبنى بها الطلب.**
func (s *Server) saleOf(r *http.Request) func(int64) int64 {
	rule := pricing.RuleFrom(r.Context(), s.settings)
	return func(cost int64) int64 { return rule.SalePrice(cost, nil, nil) }
}

// handleAdminOffers العروضُ كلُّها — الساريةُ والمنتهيةُ والمنزَّلة.
func (s *Server) handleAdminOffers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.offers.List(r.Context(), false, s.saleOf(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"offers": rows})
}

// handleCreateOffer ينزّل عرضاً — لافتةً أو خصماً.
func (s *Server) handleCreateOffer(w http.ResponseWriter, r *http.Request) {
	in, err := decode[offers.Input](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o, err := s.offers.Create(r.Context(), userIDFrom(r), *in, s.saleOf(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "catalog.offer_create", "offer", o.ID, map[string]any{
		"kind": o.Kind, "title": o.Title,
	})
	// **والزبائنُ يرونه فوراً** — عرضٌ يُنزَل ولا يظهر حتى يُحدَّث المتصفّحُ
	// عرضٌ نصفُ منزَّل.
	s.touch("offer", "ops")
	httpx.JSON(w, http.StatusOK, o)
}

// handleSetOfferActive يرفع العرضَ أو ينزله — **ولا حذف.**
//
// **عرضٌ حُذف لا يُقرأ في تقريرٍ لاحق** — ومن سأل «كم خسرنا على عروض رمضان؟»
// لم يجد ما يقرؤه.
func (s *Server) handleSetOfferActive(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Active bool `json:"active"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o, err := s.offers.SetActive(r.Context(), chi.URLParam(r, "id"), req.Active, s.saleOf(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "catalog.offer_active", "offer", o.ID, map[string]any{"active": req.Active})
	s.touch("offer", "ops")
	httpx.JSON(w, http.StatusOK, o)
}

// handlePublicOffers ما يراه الزبون — **الساريَ وحدَه.**
func (s *Server) handlePublicOffers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.offers.List(r.Context(), true, s.saleOf(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"offers": rows})
}
