package pricing_test

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// TestAmount_RoundsToNearestLira **النسبةُ تُقرَّب لأقرب ليرة** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «إذا تحت الـ٫٥
// للـ١، وإذا أكبر للـ٢») — كان الكسرُ يُحذف دائماً فتخسر المنصّةُ حتّى ٠٫٩٩ في الطلب.
func TestAmount_RoundsToNearestLira(t *testing.T) {
	cases := []struct {
		name string
		a    pricing.Amount
		base int64
		want int64
	}{
		{"٦٪ من ٣٢ = ١٫٩٢ ← ٢", pricing.Amount{Mode: "percent", Value: 6}, 32, 2},
		{"٦٪ بمنزلتين من ٣٢ ← ٢", pricing.Amount{Mode: pricing.ModePercent2, Value: 600}, 32, 2},
		{"١٠٪ من ١٤ = ١٫٤ ← ١", pricing.Amount{Mode: "percent", Value: 10}, 14, 1},
		{"١٠٪ من ١٥ = ١٫٥ ← ٢", pricing.Amount{Mode: "percent", Value: 10}, 15, 2},
		{"١٠٪ من ١٩ = ١٫٩ ← ٢", pricing.Amount{Mode: "percent", Value: 10}, 19, 2},
		{"١٫٥٪ من ١٠٠ = ١٫٥ ← ٢", pricing.Amount{Mode: pricing.ModePercent2, Value: 150}, 100, 2},
		{"١٫٢٥٪ من ١٠٠ = ١٫٢٥ ← ١", pricing.Amount{Mode: pricing.ModePercent2, Value: 125}, 100, 1},
		{"والصحيحُ كما هو: ١٠٪ من ٤٠ = ٤", pricing.Amount{Mode: "percent", Value: 10}, 40, 4},
		{"ولا يتجاوز الأساس: ١٠٠٪ من ٧", pricing.Amount{Mode: "percent", Value: 100}, 7, 7},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.a.Of(c.base); got != c.want {
				t.Fatalf("⇒ %d والمنتظَر %d", got, c.want)
			}
		})
	}
}
