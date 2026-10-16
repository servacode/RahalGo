package orders

import "testing"

// TestFetchAmountBlocks **حارسُ «استلمت» في «إحضار»** (قرارُ المالك ٢٠٢٦-١٠-١٦) — بلا قاعدة.
func TestFetchAmountBlocks(t *testing.T) {
	cases := []struct {
		mode     string
		estimate bool
		status   string
		want     bool
	}{
		// **«توصيل» لا يُوقَف أبداً.**
		{DeliveryModeSend, true, "", false},
		{DeliveryModeSend, true, FetchAmountPending, false},
		{"", false, "", false},
		// **تقديرٌ بلا مبلغٍ موافَقٍ عليه يوقف.**
		{DeliveryModeFetch, true, "", true},
		{DeliveryModeFetch, true, FetchAmountPending, true},
		{DeliveryModeFetch, true, FetchAmountRejected, true},
		{DeliveryModeFetch, true, FetchAmountApproved, false},
		// **وبلا تقديرٍ لا خطوة — إلّا مبلغاً كُتب فينتظر كغيره.**
		{DeliveryModeFetch, false, "", false},
		{DeliveryModeFetch, false, FetchAmountPending, true},
		{DeliveryModeFetch, false, FetchAmountRejected, true},
		{DeliveryModeFetch, false, FetchAmountApproved, false},
	}
	for _, c := range cases {
		if got := FetchAmountBlocks(c.mode, c.estimate, c.status); got != c.want {
			t.Errorf("FetchAmountBlocks(%q, %v, %q) = %v — والمنتظَر %v", c.mode, c.estimate, c.status, got, c.want)
		}
	}
}
