package server

import "testing"

// TestGoodsUnits **ثمنُ البضاعة كما يُحفظ بعملته** (قرارُ المالك ٢٠٢٦-١١-٠٩) — والتطبيقُ القديمُ بلا عملةٍ
// ليرةٌ كما كان؛ والدولارُ سنتاتٌ في حقله والليرةُ صفر؛ وخطوةُ الأجرة لا ثمنَ فيها.
func TestGoodsUnits(t *testing.T) {
	cases := []struct {
		name     string
		v        float64
		cur      string
		feeStep  bool
		goods    int64
		usd      int64 // -1: nil
		accepted bool
	}{
		{"تطبيقٌ قديم", 500, "", false, 500, -1, true},
		{"ليرةٌ صريحة", 500, "SYP", false, 500, -1, true},
		{"ليرةٌ بكسر", 12.5, "", false, 0, -1, false},
		{"دولارٌ بكسر", 12.5, "USD", false, 0, 1250, true},
		{"دولارٌ صحيح", 12, "USD", false, 0, 1200, true},
		{"دولارٌ صفر", 0, "USD", false, 0, -1, false},
		{"ثلاثُ منازل", 1.255, "USD", false, 0, -1, false},
		{"عملةٌ مجهولة", 5, "EUR", false, 0, -1, false},
		{"خطوةُ الأجرة", 12.5, "USD", true, 0, -1, true},
	}
	for _, c := range cases {
		goods, usd, ok := goodsUnits(c.v, c.cur, c.feeStep)
		if ok != c.accepted {
			t.Errorf("%s: قُبل=%v", c.name, ok)
			continue
		}
		if !ok {
			continue
		}
		if goods != c.goods || (usd == nil) != (c.usd == -1) || (usd != nil && *usd != c.usd) {
			t.Errorf("%s: goods=%d usd=%v", c.name, goods, usd)
		}
	}
}
