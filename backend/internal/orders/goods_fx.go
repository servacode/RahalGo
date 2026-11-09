package orders

// ══════════════════════════════════════════════════════════════════════
// **ثمنُ البضاعة بالدولار — وما يعادله بالليرة** (قرارُ المالك ٢٠٢٦-١١-٠٩)
// ══════════════════════════════════════════════════════════════════════
//
// السائقُ يكتب ثمنَ البضاعة **بالليرة أو بالدولار**، والزبونُ يرى ما كُتب بعملته — **لا تحويلَ
// للعرض ولا جمعَ للعملتين.** **والمحفظةُ ليرةٌ وحدَها**: ثمنٌ بالدولار يُدفع نقداً للسائق، والمحفظةُ
// تدفع الأجرةَ وحدَها.
//
// # شكلُ البيانات
//
//	custom_goods_usd   الثمنُ بالسنتات — **و`custom_goods_amount` معه صفر**، فالمجموعُ والمستحقُّ
//	                   النقديُّ وحجزُ المحفظة الأجرةُ وحدَها، **ولا يُمسّ حسابُ الليرة.**
//	goods_fx_rate      السعرُ الملقوط — سعرُ المتجر (التابع في المشتريات، متجرنا في الواتساب)،
//	                   **وإلّا «سعرُ الدولار المرجعيّ»** (`merchants.usd_reference_rate`).
//	goods_usd_syp      ما يعادله باللقطة (مدوَّراً لأقرب ٥) — **للمشتريات وحدَها.**
//
// # وما يُحسب بالمعادل (القرار ٤ و٥)
//
//	عمولةُ المتجر التابع  ←  نسبتُه من المعادل (`customStoreCut`) — يقبضها صندوقُ السائق ليرةً
//	                         (الدولارُ بيده)، **والخزينةُ تقرأ المعادلَ مقبوضاً ثمّ تطرح ما دُفع للمتجر**
//	عمولةُ متجرنا (واتساب) ←  بالسعر الملقوط على الطلب، وإلّا سعرِ المتجر، وإلّا المرجعيّ
//	سقفُ نقد الزبون        ←  المعادلُ نقدٌ بذمّته أيّاً كانت طريقةُ دفع الأجرة
//	سقفُ نقد السائق        ←  المعادلُ يمرّ بيده كالمستحقّ النقديّ
//	خسارةُ البضاعة         ←  ما دفعه للمتجر بالمعادل — كالليرة تماماً
//
// **وبلا سعرٍ أصلاً**: المعادلُ صفرٌ — لا عمولةَ ولا سقف (تحذيرٌ في السجلّ)، كما كان الواتساب.

import (
	"context"
	"fmt"
	"math"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// CurrencyUSD وCurrencySYP **عملتا ثمن البضاعة** — والفارغُ ليرةٌ (تطبيقٌ قديم).
const (
	CurrencySYP = "SYP"
	CurrencyUSD = "USD"
)

// usdToSYP **سنتاتٌ بسعرٍ إلى ليرةٍ مدوَّرةٍ لأقرب ٥** — كـ`catalog.USDToSYP`. وبلا سعرٍ صفر.
func usdToSYP(cents int64, rate float64) int64 {
	if cents <= 0 || rate <= 0 {
		return 0
	}
	return int64(math.Round(float64(cents)*rate/100/5)) * 5
}

// usdReferenceRate **سعرُ الدولار المرجعيّ** — من منفّذ المعاملة نفسِه (`XG-46`)، وصفرٌ بلا ضبط.
func (s *Service) usdReferenceRate(ctx context.Context, q dbtx.Querier) float64 {
	if s.settings == nil {
		return 0
	}
	return float64(s.settings.On(q).GetHundredths(ctx, "merchants.usd_reference_rate")) / 100
}

// goodsRateFor **السعرُ لطلبٍ بعينه الآن** — سعرُ متجره (التابعِ في المشتريات، ومتجرِنا في الواتساب)
// إن ضبطه، وإلّا المرجعيّ. **وصفرٌ: لا سعر.**
func (s *Service) goodsRateFor(ctx context.Context, q dbtx.Querier, orderID string) (float64, error) {
	var store *float64
	if err := q.QueryRow(ctx, `
		SELECT m.exchange_rate::float8
		  FROM orders o
		  LEFT JOIN merchants m ON m.id = COALESCE(o.partner_merchant_id, o.pickup_merchant_id)
		 WHERE o.id = $1`, orderID).Scan(&store); err != nil {
		return 0, err
	}
	if store != nil && *store > 0 {
		return *store, nil
	}
	return s.usdReferenceRate(ctx, q), nil
}

// snapshotCustomGoodsFXTx **يلقط السعرَ والمعادلَ على طلب مشترياتٍ بثمنٍ بالدولار.**
//
// `force`: يُعاد اللقطُ (تأكيدُ الزبون — كلُّ تأكيدٍ يلقط سعرَ لحظته). **وبلاه لا يُلقط إلّا ما لم
// يُلقط** (خسارةُ بضاعةٍ قبل التأكيد). **وثمنٌ بالليرة: يُصفَّر الاثنان.** ويردّ المعادل.
func (s *Service) snapshotCustomGoodsFXTx(ctx context.Context, q dbtx.Querier, orderID string, force bool) (int64, error) {
	var usd *int64
	var rate *float64
	var equiv int64
	if err := q.QueryRow(ctx, `
		SELECT custom_goods_usd, goods_fx_rate::float8, goods_usd_syp FROM orders WHERE id = $1`,
		orderID).Scan(&usd, &rate, &equiv); err != nil {
		return 0, err
	}
	if usd == nil {
		if rate != nil || equiv != 0 {
			_, err := q.Exec(ctx, `UPDATE orders SET goods_fx_rate = NULL, goods_usd_syp = 0 WHERE id = $1`, orderID)
			return 0, err
		}
		return 0, nil
	}
	if rate != nil && !force {
		return equiv, nil
	}
	r, err := s.goodsRateFor(ctx, q, orderID)
	if err != nil {
		return 0, err
	}
	if r <= 0 {
		s.logger.Warn("ثمن بالدولار بلا سعر صرف (لا للمتجر ولا مرجعي) — المعادل صفر", "order", orderID)
		_, err := q.Exec(ctx, `UPDATE orders SET goods_fx_rate = NULL, goods_usd_syp = 0 WHERE id = $1`, orderID)
		return 0, err
	}
	equiv = usdToSYP(*usd, r)
	if _, err := q.Exec(ctx, `UPDATE orders SET goods_fx_rate = $2, goods_usd_syp = $3 WHERE id = $1`,
		orderID, r, equiv); err != nil {
		return 0, err
	}
	return equiv, nil
}

// customAmountText **ما يدفعه الزبونُ نصّاً** — المجموعُ بالليرة، **أو البضاعةُ بالدولار والتوصيلُ بالليرة
// منفصلَين** (لا جمعَ لعملتين). و`total` في الدولار الأجرةُ وحدَها.
func customAmountText(total int64, usd *int64) string {
	if usd != nil {
		return fmt.Sprintf("البضاعة %s + التوصيل %s ل.س", AmountText(*usd, CurrencyUSD), groupDigits(total))
	}
	return groupDigits(total) + " ل.س"
}

// goodsText **ثمنُ البضاعة نصّاً بعملته** — للحديث والإشعارات.
func goodsText(goods int64, usd *int64) string {
	if usd != nil {
		return AmountText(*usd, CurrencyUSD)
	}
	return groupDigits(goods) + " ل.س"
}
