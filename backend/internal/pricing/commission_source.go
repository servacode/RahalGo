package pricing

// ══════════════════════════════════════════════════════════════════════
// **مصدرُ احتساب عمولة المندوب** — `RQ-6` · `XG-13` · `XG-14`
// ══════════════════════════════════════════════════════════════════════
//
// # العقدُ المعتمد
//
//	platform_commission   عمولةُ المنصّة وحدَها
//	pricing_margin        هامشُ التسعير وحدَه
//	both                  مجموعُهما
//
// **وكلُّ مركّبٍ قد يكون صفراً وحدَه** — **والصفرُ في أحدهما لا يمنع
// الحسابَ من الآخر** حيث يسمح الوضع.
//
// # ولماذا هنا
//
// **قواعدُ المال تُقرأ من `pricing` لا تُكتب في المعالِجات** — وهي
// القاعدةُ التي جمعت `RepCommission` من ثلاثة مواضعَ متفرّقة. **ومن
// كتب القاعدةَ ثانيةً دفع غيرَ ما يُعرَض.**
//
// # ولا وضعَ مجهولٌ يُقرأ صامتاً
//
// **مجهولُ الوضع يُردّ خطأً ولا يرتدّ إلى افتراض** — **وقيمةٌ فاسدةٌ
// تُقرأ افتراضاً تدفع مالاً لا يقصده أحد**، ولا يُكتشف إلّا في كشفٍ
// شهريّ.

import (
	"context"
	"fmt"
)

// CommissionSource وضعُ احتساب قاعدة عمولة المندوب.
type CommissionSource string

const (
	// SourcePlatformCommission عمولةُ المنصّة وحدَها.
	SourcePlatformCommission CommissionSource = "platform_commission"
	// SourcePricingMargin هامشُ التسعير وحدَه.
	SourcePricingMargin CommissionSource = "pricing_margin"
	// SourceBoth مجموعُهما.
	SourceBoth CommissionSource = "both"
)

// CommissionSourceKey مفتاحُ الوضع — **واحدٌ لا اثنان.**
const CommissionSourceKey = "sales.commission_source"

// RepCommissionSource يقرأ الوضعَ المعتمد.
//
// **ويردّ خطأً لمجهولٍ** — ولا يُخمَّن.
func RepCommissionSource(ctx context.Context, st Store) (CommissionSource, error) {
	if st == nil {
		return "", fmt.Errorf("pricing: لا مخزنَ إعدادات — ولا وضعَ يُقرأ")
	}
	switch v := CommissionSource(st.GetString(ctx, CommissionSourceKey)); v {
	case SourcePlatformCommission, SourcePricingMargin, SourceBoth:
		return v, nil
	default:
		return "", fmt.Errorf("pricing: وضعُ احتسابٍ مجهول: %q", v)
	}
}

// RepCommissionBase **قاعدةُ الحساب بحسب الوضع** — لا بوّابةَ فيها.
//
// **وكانت البوّابةُ مثبَّتةً**: `platformCommission == 0` تمنع كلَّ
// شيءٍ **ثمّ يُحسَب من الهامش** — **فيُبوَّب بوضعٍ ويُحسَب بآخر.**
//
// **فصارت القاعدةُ تقول ما تجمع، والوضعُ يقول ماذا يدخلها.**
func RepCommissionBase(src CommissionSource, platformCommission, pricingMargin int64) int64 {
	var base int64
	switch src {
	case SourcePlatformCommission:
		base = platformCommission
	case SourcePricingMargin:
		base = pricingMargin
	case SourceBoth:
		base = platformCommission + pricingMargin
	}
	if base < 0 {
		return 0
	}
	return base
}
