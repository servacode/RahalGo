package settings_test

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// TestCommissionPercentAcceptsTwoDecimals **١٫٥ تُقبل · ١٫٥٥٥ و١٠١ تُردّان.**
//
// (قرارُ المالك ٢٠٢٦-١٠-١٨.) **وقبله كان المفتاحُ `KindInt`** فيُردّ ١٫٥
// «ليست عدداً صحيحاً» — ولا سبيلَ إلى نسبةٍ بكسر.
func TestCommissionPercentAcceptsTwoDecimals(t *testing.T) {
	const key = "merchants.commission_percent"
	for _, v := range []float64{1.5, 2.25, 0, 100, 0.01, 99.99} {
		if _, err := settings.Validate(key, v); err != nil {
			t.Errorf("%v رُدّت: %v", v, err)
		}
	}
	for _, v := range []float64{1.555, 101, -0.5, 100.01} {
		if _, err := settings.Validate(key, v); err == nil {
			t.Errorf("%v قُبلت — والمنتظَر رفضُها", v)
		}
	}
	// **والصحيحُ يُحفظ صحيحاً** — `10` لا `10.0`، فقيمٌ قديمةٌ تبقى كما هي.
	if got, _ := settings.Validate(key, float64(10)); got != int64(10) {
		t.Errorf("١٠ حُفظت %#v — والمنتظَر int64(10)", got)
	}
	if got, _ := settings.Validate(key, 1.5); got != 1.5 {
		t.Errorf("١٫٥ حُفظت %#v", got)
	}
}
