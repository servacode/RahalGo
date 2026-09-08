package orders

// ══════════════════════════════════════════════════════════════════════
// **لقطةُ اقتصادِ الطلب** — `XQ-2` · `XG-25`…`XG-28`
// ══════════════════════════════════════════════════════════════════════
//
// # العقد
//
//	EXISTING ORDER USES ITS FINANCIAL SNAPSHOT
//
// **تُلتقَط لحظةَ نشوء الالتزام الماليّ — وهو إنشاءُ الطلب** — **ولا
// يبدّل إعدادٌ لاحقٌ اقتصادَ طلبٍ قائم.**
//
// # ولماذا الإنشاءُ هو اللحظة
//
// **سعرُ البيع وسعرُ الشراء يُلتقطان في `order_items` عند الإنشاء**،
// **وأجرةُ التوصيل في الطلب نفسِه** — **فالمقدّماتُ الأربعُ الباقيةُ
// تُلتقَط معها أو تفترق عن مصدرها.**
//
// # وقراءةٌ واحدةٌ متماسكة
//
// **أربعُ قراءاتٍ مستقلّةٍ قد تتخطّى تبديلاً إداريّاً** — **فتخرج
// لقطةٌ نصفُها من عقدٍ ونصفُها من آخر**: نسبةٌ قديمةٌ ومصدرٌ جديد.
//
// **فتُقرأ الأربعُ من صورةٍ واحدةٍ للإعدادات** (`snapshotAt`)، **ثمّ
// تُكتب مع الطلب في معاملته.**

import (
	"context"
	"fmt"

	"github.com/servacode/rahalgo/backend/internal/pricing"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// EconomicsSnapshot المقدّماتُ الماليّةُ الملقوطةُ للطلب.
//
// **ولا يدخلها إعدادُ واجهةٍ ولا إشعارٍ ولا تشغيل** — **لقطةُ اقتصادٍ
// لا مستودعُ إعدادات.**
type EconomicsSnapshot struct {
	// MerchantCommissionPercent نسبةُ عمولة المنصّة — `XG-25`.
	MerchantCommissionPercent int64
	// RepCommissionPercent نسبةُ المندوب — `XG-26`.
	RepCommissionPercent int64
	// CommissionSource مصدرُ احتساب عمولة المندوب — `XG-27`.
	CommissionSource string
	// ActivationOrders عتبةُ تفعيل المتجر — `XG-28`.
	ActivationOrders int64
}

// ErrNoSnapshot **طلبٌ يُسوّى بلا لقطةِ اقتصاد.**
//
// **ولا ارتدادَ إلى إعدادات اليوم** — **وذاك بعينه ما يمنعه `XQ-2`**:
// **اقتصادٌ يُخترَع بعد شهرٍ ليس اقتصادَ الطلب.**
var ErrNoSnapshot = fmt.Errorf("orders: طلبٌ بلا لقطةِ اقتصاد — ولا تُقرأ إعداداتُ اليوم بدلاً منها")

// snapshotNow يقرأ المقدّماتِ الأربعَ من صورةٍ واحدةٍ للإعدادات.
//
// **ومجهولُ الوضع يُردّ هنا** — **فلا تُكتب لقطةٌ لا تُقرأ.**
func (s *Service) snapshotNow(ctx context.Context) (EconomicsSnapshot, error) {
	if s.settings == nil {
		return EconomicsSnapshot{}, fmt.Errorf("orders: لا مخزنَ إعدادات — ولا لقطةَ تُقرأ")
	}
	// ══════════════════════════════════════════════════════════════
	// **والأربعُ من صورةٍ واحدة** — `XQ-2`
	// ══════════════════════════════════════════════════════════════
	//
	// **وقيس بأربع قراءاتٍ مستقلّة**: **لقطةٌ خرجت `40/10`** —
	// **نسبةُ المنصّة من الإعداد الجديد ونسبةُ المندوب من القديم**،
	// **وذاك عقدٌ لم يوافق عليه أحد.**
	c, err := s.settings.ReadCoherent(ctx,
		"merchants.commission_percent", "sales.commission_percent",
		pricing.CommissionSourceKey, "sales.activation_orders")
	if err != nil {
		return EconomicsSnapshot{}, err
	}
	src := pricing.CommissionSource(c.String(pricing.CommissionSourceKey))
	switch src {
	case pricing.SourcePlatformCommission, pricing.SourcePricingMargin, pricing.SourceBoth:
	default:
		// **ومجهولُ الوضع يُردّ** — **فلا تُكتب لقطةٌ لا تُقرأ.**
		return EconomicsSnapshot{}, fmt.Errorf("orders: وضعُ احتسابٍ مجهول: %q", src)
	}
	return EconomicsSnapshot{
		MerchantCommissionPercent: c.Int("merchants.commission_percent"),
		RepCommissionPercent:      c.Int("sales.commission_percent"),
		CommissionSource:          string(src),
		ActivationOrders:          c.Int("sales.activation_orders"),
	}, nil
}

// snapshotOf يقرأ لقطةَ طلبٍ قائم — **وهي حقيقتُه الماليّة.**
//
// **ولا تُقرأ منها قيمةٌ ناقصة**: القيدُ في الجدول يوجب أن تكون كلُّها
// أو لا شيءَ منها، **فغيابُ واحدةٍ غيابُ اللقطة.**
func (s *Service) snapshotOf(ctx context.Context, q wallet.Querier, orderID string) (EconomicsSnapshot, error) {
	var snap EconomicsSnapshot
	var mp, rp, act *int64
	var src *string
	if err := q.QueryRow(ctx, `
		SELECT snap_merchant_commission_percent, snap_rep_commission_percent,
		       snap_commission_source, snap_activation_orders
		  FROM orders WHERE id = $1`, orderID).Scan(&mp, &rp, &src, &act); err != nil {
		return EconomicsSnapshot{}, err
	}
	if mp == nil || rp == nil || src == nil || act == nil {
		return EconomicsSnapshot{}, ErrNoSnapshot
	}
	snap.MerchantCommissionPercent = *mp
	snap.RepCommissionPercent = *rp
	snap.CommissionSource = *src
	snap.ActivationOrders = *act
	return snap, nil
}

// orderPct **نسبةُ عمولة المنصّة لهذا المتجر في هذا الطلب.**
//
// **وتجاوزُ المتجر يغلب** — صفةٌ فيه لا إعدادٌ عامّ، **ولا تشيخ
// بتبديل مفتاح.** **وحيث لا تجاوزَ تُقرأ نسبةُ اللقطة** لا نسبةُ
// اليوم.
func orderPct(override *int64, snap EconomicsSnapshot) *int64 {
	if override != nil {
		return override
	}
	v := snap.MerchantCommissionPercent
	return &v
}
