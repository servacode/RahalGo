package server

import (
	"strings"
	"testing"
)

// TestIphoneLine **صاحبُ المتجر يصله رابطُ الآيفون** (٢٠٢٦-١٠-٢٠) — لوحةُ متجره والتسوّقُ من
// سفاري — **وغيرُه لا**، ولا سطرَ بلا أصلِ موقع.
func TestIphoneLine(t *testing.T) {
	got := iphoneLine("merchant", "https://rahalgo.com")
	for _, want := range []string{"آيفون", "https://rahalgo.com/store", "https://rahalgo.com/shop"} {
		if !strings.Contains(got, want) {
			t.Fatalf("سطرُ الآيفون بلا %q: %q", want, got)
		}
	}
	if !strings.HasPrefix(got, "\n") {
		t.Fatal("السطرُ لا يبدأ بسطرٍ جديد — يلتصق بما قبله")
	}
	for _, k := range []string{"rep", "driver", "customer", ""} {
		if iphoneLine(k, "https://rahalgo.com") != "" {
			t.Fatalf("سطرُ آيفون لـ%q — ولا نسخةَ آيفون له", k)
		}
	}
	if iphoneLine("merchant", "") != "" {
		t.Fatal("سطرٌ بلا أصلِ موقع — روابطُ مكسورة")
	}
}
