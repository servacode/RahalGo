package pricing_test

// قواعدُ المال — **نسبةٌ أو مقطوع، وأجرةُ توصيلٍ بثلاثة أنماط.**
//
// **وهي اختباراتٌ بلا قاعدةِ بيانات** وتحرس ثلاثةَ أموالٍ في كلّ طلب: عمولةَ
// المنصة من المتجر، وحصّةَ المندوب منها، وأجرةَ التوصيل.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// fakeStore مخزنُ إعداداتٍ في الذاكرة — **يُغيَّر بين نداءين ليُثبَت السريان.**
type fakeStore map[string]any

func (f fakeStore) GetString(_ context.Context, key string) string {
	if v, ok := f[key].(string); ok {
		return v
	}
	return ""
}
func (f fakeStore) GetInt(_ context.Context, key string) int64 {
	if v, ok := f[key].(int64); ok {
		return v
	}
	return 0
}

// TestAmount_Of النسبةُ والمقطوع — **ولا يتجاوز أحدُهما الأساس.**
//
// **وأخطرُ سطرٍ فيه المقطوعُ الأكبرُ من الطلب.** عمولةٌ مقطوعةٌ بعشرين ألفاً
// على طلبٍ بخمسة عشر **تجعل المتجرَ يدفع لنا كي يبيع** — ورقمٌ سالبٌ يخرج من
// هنا يقلب قيداً محاسبيّاً بلا أن يصرخ أحد.
func TestAmount_Of(t *testing.T) {
	cases := []struct {
		name string
		a    pricing.Amount
		base int64
		want int64
	}{
		{"نسبةٌ عاديّة", pricing.Amount{Mode: "percent", Value: 10}, 26_000, 2_600},
		{"مقطوعٌ عاديّ", pricing.Amount{Mode: "fixed", Value: 3_000}, 26_000, 3_000},
		{"نسبةُ صفرٍ لا عمولة", pricing.Amount{Mode: "percent", Value: 0}, 26_000, 0},
		{"مقطوعٌ أكبرُ من الأساس يُحدّ به",
			pricing.Amount{Mode: "fixed", Value: 20_000}, 15_000, 15_000},
		{"نسبةٌ فوق المئة تُحدّ بالأساس",
			pricing.Amount{Mode: "percent", Value: 150}, 10_000, 10_000},
		{"أساسٌ صفرٌ لا يُشتقّ منه شيء",
			pricing.Amount{Mode: "fixed", Value: 3_000}, 0, 0},
		{"وأساسٌ سالبٌ كذلك", pricing.Amount{Mode: "percent", Value: 10}, -5_000, 0},
		{"ونمطٌ مجهولٌ يُقرأ نسبةً — لا يُهمَل المال",
			pricing.Amount{Mode: "لا-شيء", Value: 10}, 10_000, 1_000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.a.Of(c.base); got != c.want {
				t.Fatalf("من %d: %d والمنتظَر %d", c.base, got, c.want)
			}
		})
	}
}

// TestCommissions_LiveOnEveryRead **التغييرُ يسري على القراءة التالية.**
//
// وهو جوهرُ ما طُلب (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «لازم كلُّ المشروع يأخذ
// الإعداداتِ هذه بحيث تُطبَّق بشكلٍ حقيقيٍّ فوريٍّ عند أيّ تغيّر»).
//
// **ومفتاحٌ يُقرأ مرّةً عند الإقلاع يجعل المالكَ يغيّر الرقمَ ويرى القديمَ
// يعمل** — فيغيّره ثانيةً وثالثة، ثمّ يشكّ في الشاشة كلِّها.
func TestCommissions_LiveOnEveryRead(t *testing.T) {
	ctx := context.Background()
	st := fakeStore{
		"merchants.commission_percent": int64(10),
		"sales.commission_mode":        "percent",
		"sales.commission_value":       int64(10),
	}

	if got := pricing.MerchantCommission(ctx, st, nil).Of(26_000); got != 2_600 {
		t.Fatalf("عمولةُ المتجر %d والمنتظَر ٢٦٠٠", got)
	}

	// **يُغيَّر المفتاحُ ولا يُعاد بناءُ شيء.**
	st["merchants.commission_percent"] = int64(15)
	if got := pricing.MerchantCommission(ctx, st, nil).Of(26_000); got != 3_900 {
		t.Fatalf("بعد التغيير %d والمنتظَر ٣٩٠٠ — الإعدادُ لم يسرِ", got)
	}

	// **وحصّةُ المندوب من عمولتنا لا من البيع.**
	st["sales.commission_value"] = int64(25)
	if got := pricing.RepCommission(ctx, st).Of(4_000); got != 1_000 {
		t.Fatalf("حصّةُ المندوب %d والمنتظَر ١٠٠٠", got)
	}
}

// TestMerchantCommission_OverrideBeatsGlobal **التجاوزُ يسبق العامّ، وفراغُه يتبعه.**
//
// **والفراغُ غيرُ الصفر**: الصفرُ قرارٌ («لا عمولةَ على هذا المتجر») والفراغُ
// غيابُ قرار. **ومن خلط بينهما جعل كلَّ متجرٍ لم يُلمس بلا عمولة** — فتعمل
// المنصةُ بلا دخلٍ ولا يظهر ذلك إلّا في آخر الشهر.
func TestMerchantCommission_OverrideBeatsGlobal(t *testing.T) {
	ctx := context.Background()
	st := fakeStore{"merchants.commission_percent": int64(10)}
	n := func(v int64) *int64 { return &v }

	if got := pricing.MerchantCommission(ctx, st, n(2)).Of(100_000); got != 2_000 {
		t.Fatalf("التجاوزُ ٢٪ أعطى %d والمنتظَر ٢٠٠٠", got)
	}
	if got := pricing.MerchantCommission(ctx, st, n(0)).Of(100_000); got != 0 {
		t.Fatalf("تجاوزُ الصفر أعطى %d — والصفرُ قرارٌ «بلا عمولة»", got)
	}
	if got := pricing.MerchantCommission(ctx, st, nil).Of(100_000); got != 10_000 {
		t.Fatalf("الفراغُ أعطى %d والمنتظَر ١٠٠٠٠ — الفراغُ «اتبع العامّ»", got)
	}
}

// TestDeliveryFee_FlatOnly **رقمٌ مقطوعٌ واحد — لا نمطَ ولا حسبة.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٤.)
func TestDeliveryFee_FlatOnly(t *testing.T) {
	ctx := context.Background()
	st := fakeStore{"delivery.fee": int64(10_000)}
	if got := pricing.DeliveryFee(ctx, st); got != 10_000 {
		t.Fatalf("الأجرة %d والمنتظَر ١٠٠٠٠", got)
	}

	// **والتغييرُ يسري على الطلب التالي.**
	st["delivery.fee"] = int64(7_500)
	if got := pricing.DeliveryFee(ctx, st); got != 7_500 {
		t.Fatalf("بعد التغيير %d والمنتظَر ٧٥٠٠ — الإعدادُ لم يسرِ", got)
	}

	// **وبلا مخزنٍ صفر** — لا يُخترع رسمٌ لم يقرّره أحد.
	if got := pricing.DeliveryFee(ctx, nil); got != 0 {
		t.Fatalf("بلا مخزنٍ %d والمنتظَر صفراً", got)
	}

	// **وسالبٌ يُقرأ صفراً** — رسمٌ سالبٌ يجعل المنصةَ تدفع للزبون ليطلب.
	st["delivery.fee"] = int64(-5_000)
	if got := pricing.DeliveryFee(ctx, st); got != 0 {
		t.Fatalf("سالبٌ أعطى %d والمنتظَر صفراً", got)
	}
}
