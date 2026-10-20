package orders

import "testing"

// TestAmountUnits **الليرةُ صحيحة والدولارُ بالسنتات** (٢٠٢٦-١٠-٢٠).
func TestAmountUnits(t *testing.T) {
	cases := []struct {
		v    float64
		cur  string
		want int64
		ok   bool
	}{
		{300, "SYP", 300, true},
		{300, "", 300, true},
		{12.5, "SYP", 0, false},
		{25, "USD", 2500, true},
		{12.5, "USD", 1250, true},
		{12.75, "USD", 1275, true},
		{12.345, "USD", 0, false},
		{10, "EUR", 0, false},
	}
	for _, c := range cases {
		got, ok := AmountUnits(c.v, c.cur)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("AmountUnits(%v, %q) = %d %v — والمتوقَّع %d %v", c.v, c.cur, got, ok, c.want, c.ok)
		}
	}
}

func TestAmountText(t *testing.T) {
	for _, c := range []struct {
		u    int64
		cur  string
		want string
	}{{2500, "USD", "25$"}, {1250, "USD", "12.50$"}, {300, "SYP", "300 ل.س"}, {300, "", "300 ل.س"}} {
		if got := AmountText(c.u, c.cur); got != c.want {
			t.Errorf("AmountText(%d, %q) = %q — والمتوقَّع %q", c.u, c.cur, got, c.want)
		}
	}
}
