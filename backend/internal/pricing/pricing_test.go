package pricing_test

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// TestSalePrice الهامشُ يُورَث، والتجاوزُ فراغٌ لا صفر، والتقريبُ لأعلى.
//
// **وأخطرُ ما فيه سطرُ الصفر.** التجاوزُ `NULL` يعني «اتبع ما فوقك»، و`0` يعني
// «لا هامشَ على هذا» — **ومن خلط بينهما جعل كلَّ صنفٍ لم يُلمس بلا هامش**،
// فتبيع المنصةُ ألفَ صنفٍ بسعر شرائها ولا يظهر ذلك في أيّ خطأ.
func TestSalePrice(t *testing.T) {
	n := func(v int64) *int64 { return &v }

	cases := []struct {
		name     string
		rule     pricing.MarginRule
		cost     int64
		item     *int64
		category *int64
		want     int64
	}{
		{"نسبةٌ عامّة بلا تجاوز",
			pricing.MarginRule{Mode: "percent", Value: 20}, 10_000, nil, nil, 12_000},
		{"تجاوزُ التصنيف يسبق العام",
			pricing.MarginRule{Mode: "percent", Value: 20}, 10_000, nil, n(50), 15_000},
		{"وتجاوزُ الصنف يسبقهما",
			pricing.MarginRule{Mode: "percent", Value: 20}, 10_000, n(10), n(50), 11_000},

		// **الصفرُ قرارٌ لا فراغ.**
		{"صفرُ الصنف يعني لا هامش",
			pricing.MarginRule{Mode: "percent", Value: 20}, 10_000, n(0), n(50), 10_000},
		{"وصفرُ التصنيف كذلك",
			pricing.MarginRule{Mode: "percent", Value: 20}, 10_000, nil, n(0), 10_000},

		// **الثابتُ يوافق تكلفتَك**: سائقٌ وعملياتٌ لا يتضاعفان بتضاعف الطعام.
		{"الثابتُ يُضاف كما هو",
			pricing.MarginRule{Mode: "fixed", Value: 3_000}, 10_000, nil, nil, 13_000},
		{"والثابتُ على الغالي لا يتضاعف",
			pricing.MarginRule{Mode: "fixed", Value: 3_000}, 60_000, nil, nil, 63_000},

		// **والتقريبُ لأعلى لا لأقرب**: لأقرب يبتلع من الهامش نصفَ الخانة.
		{"التقريبُ إلى ٥٠٠ لأعلى",
			pricing.MarginRule{Mode: "percent", Value: 15, Rounding: 500}, 8_000, nil, nil, 9_500},
		{"وما استوى لا يُزاد",
			pricing.MarginRule{Mode: "percent", Value: 25, Rounding: 500}, 8_000, nil, nil, 10_000},
		{"وبلا تقريبٍ يُترك كما حُسب",
			pricing.MarginRule{Mode: "percent", Value: 15}, 8_125, nil, nil, 9_343},

		// **ولا يُباع بأقلّ من ثمنه.**
		//
		// هامشٌ سالبٌ بالخطأ يجعل المنصةَ تدفع من جيبها عن كلّ بيعة —
		// **وخسارةٌ تتكرّر بلا حدث تُكتشف في آخر الشهر.**
		{"الهامشُ السالب لا يُنقص الثمن",
			pricing.MarginRule{Mode: "fixed", Value: -5_000}, 10_000, nil, nil, 10_000},

		// **وصفرُ الهامش هو حالُ الترحيل** — سعرُ البيع = سعرُ الشراء.
		{"بلا هامشٍ سعرُ البيع سعرُ الشراء",
			pricing.MarginRule{Mode: "percent"}, 10_000, nil, nil, 10_000},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.rule.SalePrice(c.cost, c.item, c.category); got != c.want {
				t.Errorf("سعرُ البيع = %d، والمتوقّع %d", got, c.want)
			}
		})
	}
}
