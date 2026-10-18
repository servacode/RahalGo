package pricing_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// TestPercent2_ParseAndPrint **منزلتان تُقبلان وثلاثٌ تُردّ — بلا عائم.**
func TestPercent2_ParseAndPrint(t *testing.T) {
	ok := map[string]int64{"1.5": 150, "2.25": 225, "10": 1000, "0": 0, "100": 10000,
		"1.50": 150, "0.01": 1, "-1": -100, ".5": 50}
	for in, want := range ok {
		got, err := pricing.ParsePercent2(in)
		if err != nil || got.Hundredths() != want {
			t.Errorf("%q ⇒ %d, %v — والمنتظَر %d", in, got, err, want)
		}
	}
	for _, in := range []string{"1.555", "abc", "", "1.", "1e2", "1,5"} {
		if _, err := pricing.ParsePercent2(in); err == nil {
			t.Errorf("%q قُبلت — والمنتظَر رفضُها", in)
		}
	}
	for p, want := range map[pricing.Percent2]string{150: "1.5", 225: "2.25", 1000: "10", 5: "0.05", 0: "0"} {
		if got := p.String(); got != want {
			t.Errorf("%d ⇒ %q والمنتظَر %q", p, got, want)
		}
	}
	// **والعميلُ القديمُ عددٌ صحيحٌ مقرَّبٌ نصفيّاً.**
	for p, want := range map[pricing.Percent2]int64{150: 2, 149: 1, 225: 2, 250: 3, 1000: 10, 0: 0} {
		if got := p.RoundedPercent(); got != want {
			t.Errorf("%s مقرَّبةً ⇒ %d والمنتظَر %d", p, got, want)
		}
	}
	var in struct {
		P *pricing.Percent2 `json:"p"`
	}
	if err := json.Unmarshal([]byte(`{"p":1.5}`), &in); err != nil || in.P == nil || *in.P != 150 {
		t.Errorf("فكُّ 1.5 ⇒ %v, %v", in.P, err)
	}
	if err := json.Unmarshal([]byte(`{"p":1.555}`), &in); err == nil {
		t.Error("فكُّ 1.555 قُبل")
	}
	b, _ := json.Marshal(map[string]pricing.Percent2{"p": 150})
	if string(b) != `{"p":1.5}` {
		t.Errorf("ترميزُ ١٫٥ ⇒ %s", b)
	}
}

// TestPercent2_CommissionTruncatesLikeBefore **البترُ نفسُه — والصحيحُ ليرةً بليرة.**
func TestPercent2_CommissionTruncatesLikeBefore(t *testing.T) {
	ctx := context.Background()
	st := fakeStore{"merchants.commission_percent": 1.5}
	if got := pricing.MerchantCommission(ctx, st, nil).Of(12_345); got != 185 {
		t.Errorf("١٫٥٪ من ١٢٣٤٥ ⇒ %d والمنتظَر 185", got)
	}
	p := pricing.Percent2(225)
	if got := pricing.MerchantCommission(ctx, st, &p).Of(10_000); got != 225 {
		t.Errorf("٢٫٢٥٪ من ١٠٠٠٠ ⇒ %d والمنتظَر 225", got)
	}
	// **كلُّ نسبةٍ صحيحةٍ وكلُّ ثمنٍ صغيرٍ كما كان.**
	for pct := int64(0); pct <= 100; pct++ {
		q := pricing.Pct(pct)
		for base := int64(0); base <= 3_000; base += 7 {
			old := pricing.Amount{Mode: "percent", Value: pct}.Of(base)
			if got := pricing.MerchantCommission(ctx, st, &q).Of(base); got != old {
				t.Fatalf("%d٪ من %d ⇒ %d وكان %d", pct, base, got, old)
			}
		}
	}
}
