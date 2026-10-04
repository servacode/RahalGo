package server

// ══════════════════════════════════════════════════════════════════════
// **مثالٌ ماليٌّ حيٌّ بجانب مفاتيح المال** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (الإعدادات، البند ١٧)
// ══════════════════════════════════════════════════════════════════════
//
// «طلبٌ بخمسين ألفاً وأجرةُ عشرة آلاف ← الزبونُ يدفع… المتجرُ… السائقُ…
// المندوبُ… المنصّة…» — **محسوبٌ بدوالّ المحرّك نفسِها** (`pricing` و
// `orders.DriverFeeAfterShare`) **لا في الشاشة**: حسبتان تفترقان يومَ يتبدّل
// شيء، فيقرأ المالكُ رقماً ويُحاسَب الناسُ بغيره.
//
// **وهو تقديرٌ لصنفٍ واحدٍ بلا تجاوزٍ خاصٍّ للمتجر ولا عرض** — يقول القاعدةَ
// العامّةَ بالأرقام، لا فاتورةَ طلبٍ بعينه.

import (
	"net/http"
	"strconv"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// moneyExample ما يُعرض تحت مفاتيح المال.
type moneyExample struct {
	Amount             int64 `json:"amount"`               // سعرُ الأصناف عند المتجر
	DeliveryFee        int64 `json:"delivery_fee"`         // أجرةُ التوصيل النافذة
	Margin             int64 `json:"margin"`               // الهامشُ فوق السعر
	CustomerPays       int64 `json:"customer_pays"`        // السعرُ مع الهامش + الأجرة
	MerchantCommission int64 `json:"merchant_commission"`  // عمولةُ المنصّة من المتجر
	MerchantGets       int64 `json:"merchant_gets"`        // ما يقبضه المتجر
	DriverGets         int64 `json:"driver_gets"`          // أجرُ السائق بعد حصّة المنصّة
	PlatformFeeShare   int64 `json:"platform_fee_share"`   // حصّةُ المنصّة من الأجرة
	RepGets            int64 `json:"rep_gets"`             // عمولةُ المندوب من ربح المنصّة
	PlatformGets       int64 `json:"platform_gets"`        // ما يبقى للمنصّة
	PlatformDeliveryPc int64 `json:"platform_delivery_pc"` // نسبةُ الحصّة النافذة
}

// computeMoneyExample **الحسبةُ نفسُها التي تجري على الطلب.**
func computeMoneyExample(r *http.Request, st pricing.Store, amount int64, platformPct int64) moneyExample {
	ctx := r.Context()
	ex := moneyExample{Amount: amount}
	sale := pricing.RuleFrom(ctx, st).SalePrice(amount, nil, nil)
	ex.Margin = sale - amount
	ex.DeliveryFee = pricing.DeliveryFee(ctx, st)
	ex.CustomerPays = sale + ex.DeliveryFee
	ex.MerchantCommission = pricing.MerchantCommission(ctx, st, nil).Of(amount)
	ex.MerchantGets = amount - ex.MerchantCommission
	ex.PlatformDeliveryPc = platformPct
	ex.DriverGets = orders.DriverFeeAfterShare(ex.DeliveryFee, platformPct)
	ex.PlatformFeeShare = ex.DeliveryFee - ex.DriverGets
	base := pricing.RepCommissionBase(pricing.RepCommissionSourceFixed, ex.MerchantCommission, ex.Margin)
	ex.RepGets = pricing.RepCommission(ctx, st).Of(base)
	ex.PlatformGets = ex.Margin + ex.MerchantCommission + ex.PlatformFeeShare - ex.RepGets
	return ex
}

// handleSettingsMoneyExample `GET /admin/settings/money-example?amount=50000`.
func (s *Server) handleSettingsMoneyExample(w http.ResponseWriter, r *http.Request) {
	amount := int64(50000)
	if raw := r.URL.Query().Get("amount"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 || n > 100_000_000 {
			s.respondErr(w, errValidation)
			return
		}
		amount = n
	}
	pct := s.settings.GetInt(r.Context(), orders.SettingPlatformDeliveryPercent)
	if pct < 0 {
		pct = 0
	}
	if pct > 90 {
		pct = 90
	}
	httpx.JSON(w, http.StatusOK, computeMoneyExample(r, s.settings, amount, pct))
}
