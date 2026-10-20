package orders

import (
	"fmt"
	"math"
)

// AmountUnits **المبلغُ كما يُحفظ** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — الليرةُ صحيحةٌ كما هي، **والدولارُ
// بالسنتات** (١٢٫٥ ⇒ ١٢٥٠). ويردّ «لا» لليرةٍ بكسرٍ أو لدولارٍ بأكثر من منزلتين أو لعملةٍ مجهولة.
//
// **وفارغٌ ليرة** — تطبيقٌ قديمٌ لا يرسل العملة.
func AmountUnits(v float64, currency string) (int64, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	switch currency {
	case "", "SYP":
		if v != math.Trunc(v) {
			return 0, false
		}
		return int64(v), true
	case "USD":
		cents := math.Round(v * 100)
		if math.Abs(cents-v*100) > 1e-6 {
			return 0, false
		}
		return int64(cents), true
	}
	return 0, false
}

// AmountText **المبلغُ نصّاً بعملته** — «١٢٫٥٠$» أو «٣٠٠ ل.س» (أرقامٌ لاتينيّةٌ كما في الإشعارات).
func AmountText(units int64, currency string) string {
	if currency == "USD" {
		if units%100 == 0 {
			return fmt.Sprintf("%d$", units/100)
		}
		return fmt.Sprintf("%d.%02d$", units/100, units%100)
	}
	return fmt.Sprintf("%d ل.س", units)
}
