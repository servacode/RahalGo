package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// ══════════════════════════════════════════════════════════════════════
// **المندوبُ يضبط نسبةَ عمولة متجر عميله** (قرارُ المالك ٢٠٢٦-١١-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// «بين ال0 وال10 بالمية» — **والحدّان إعدادان** (`sales.rep_commission_min` و
// `sales.rep_commission_max`) يغيّرهما المالكُ متى شاء. **وتسري فوراً بلا موافقة المكتب.**
//
// **والكتابةُ طريقُ الأدمن نفسُه** (`catalog.SetMerchantCommissionByRep` ← `updateMerchant`)،
// وكلُّ تبديلٍ يُقيَّد بقيمتيه في `merchant.commission_change`. **والطلبُ المفتوحُ يُسوّى بالنسبة
// الحاليّة عند التسليم، والمسلَّمُ لا يُمسّ** — كتعديل الأدمن تماماً.
//
// **وما يراه المندوبُ نسبةُ المتجر وحدَها** — ما يُقتطع من صاحبه ويعرفه صاحبُه — **لا هامشَ
// المنصّة ولا ربحَها** (`RQ-7`).

// errRepCommissionRange **نسبةٌ خارجَ ما يجوز للمندوب.**
var errRepCommissionRange = httpx.NewError(http.StatusBadRequest,
	"rep_commission_out_of_range", "errors.rep_commission_out_of_range")

// repCommissionView **نسبةُ عمولة المتجر كما يراها المندوب** — وحدّا ما يضبطه.
type repCommissionView struct {
	// Percent النسبةُ التي يُقتطع بها من المتجر الآن — **خاصّتُه إن كانت وإلّا العامّة.**
	Percent pricing.Percent2 `json:"store_commission_percent"`
	Min     pricing.Percent2 `json:"store_commission_min"`
	Max     pricing.Percent2 `json:"store_commission_max"`
}

// repCommissionBounds **حدّا المندوب من الإعدادات.**
func (s *Server) repCommissionBounds(ctx context.Context) (pricing.Percent2, pricing.Percent2) {
	return pricing.Percent2(s.settings.GetHundredths(ctx, "sales.rep_commission_min")),
		pricing.Percent2(s.settings.GetHundredths(ctx, "sales.rep_commission_max"))
}

// repCommissionOf **نسبةُ المتجر الحاليّةُ وحدّاها** — للتفصيل ولردّ الحفظ.
func (s *Server) repCommissionOf(ctx context.Context, override *pricing.Percent2) repCommissionView {
	lo, hi := s.repCommissionBounds(ctx)
	return repCommissionView{
		Percent: pricing.MerchantCommission(ctx, s.settings, override).Percent(),
		Min:     lo,
		Max:     hi,
	}
}

// handleRepSetCommission `PATCH /rep/stores/{id}/commission` — والحارسُ `repMenuGuard` قبله:
// **متجرُ عميله الحاليّ وحدَه**، وغيرُه ٤٠٣.
func (s *Server) handleRepSetCommission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	merchantID := chi.URLParam(r, "id")
	req, err := decode[struct {
		Percent *pricing.Percent2 `json:"percent"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Percent == nil {
		s.respondErr(w, errValidation)
		return
	}
	lo, hi := s.repCommissionBounds(ctx)
	if *req.Percent < lo || *req.Percent > hi {
		s.respondErr(w, errRepCommissionRange)
		return
	}
	m, err := s.catalog.SetMerchantCommissionByRep(ctx, userIDFrom(r), merchantID, *req.Percent, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **وعمولتُه الجديدةُ تصل طلباتِ المشتريات المربوطة به التي لم تُسلَّم** — كالأدمن.
	if _, err := s.orders.RefreshPartnerCommission(ctx, merchantID); err != nil {
		s.logger.Error("تحديثُ عمولة المتجر التابع على طلباته المفتوحة", "err", err)
	}
	httpx.JSON(w, http.StatusOK, s.repCommissionOf(ctx, m.CommissionExact))
}
