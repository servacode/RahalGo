package server

import (
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// **رابطُ تطبيق الزبون في رسالة الدخول** (قرارُ المالك ٢٠٢٦-١٠-٠٥): المندوبُ والمتجرُ
// والسائقُ يتسوّقون بالحساب نفسِه — **والزبونُ والموظّفُ بلا سطر التسوّق.**
func TestWelcome_ShopLinkPerRole(t *testing.T) {
	s := &Server{}
	const link = "https://x/download/"
	for _, tc := range []struct {
		key      string
		wantShop bool
		wantApp  string
	}{
		{"rep", true, "تطبيق المندوب"},
		{"merchant", true, "تطبيق المتجر"},
		{"driver", true, "تطبيق السائق"},
		{"customer", false, "تطبيق رحّال غو"},
		{"", false, "لوحة التحكّم"},
	} {
		msg := welcomeText(settings.WelcomeTemplateDefault, "+963900000000", "pw123456",
			link+tc.key, appPurpose(tc.key), s.shopLinkFor(tc.key), 72, false)
		if strings.Contains(msg, "{") {
			t.Fatalf("%q: موضعٌ لم يُستبدل:\n%s", tc.key, msg)
		}
		if !strings.Contains(msg, tc.wantApp) || !strings.Contains(msg, link+tc.key) {
			t.Fatalf("%q: رابطُ تطبيق الدور أو اسمُه غائب:\n%s", tc.key, msg)
		}
		if got := strings.Contains(msg, "تتسوّق"); got != tc.wantShop {
			t.Fatalf("%q: سطرُ التسوّق = %v، والمطلوب %v:\n%s", tc.key, got, tc.wantShop, msg)
		}
		if tc.wantShop && !strings.Contains(msg, "/download/customer") {
			t.Fatalf("%q: رابطُ تطبيق الزبون غائب:\n%s", tc.key, msg)
		}
	}

	// **وقالبٌ قديمٌ بلا `{shop_link}`** يُضاف إليه السطرُ للمندوب.
	old := "{title}\nكلمة: {password}\nحمّل من: {link}"
	msg := welcomeText(old, "+963900000000", "pw", link+"rep", appPurpose("rep"), s.shopLinkFor("rep"), 72, false)
	if !strings.Contains(msg, "/download/customer") {
		t.Fatalf("قالبٌ قديم: سطرُ التسوّق لم يُضَف:\n%s", msg)
	}
}

// **فيديو الشرح لصاحب المتجر وحدَه** (طلبُ المالك ٢٠٢٦-١٠-٠٨) — وفارغُه يحذف السطر.
func TestWelcome_MerchantTutorialLine(t *testing.T) {
	const url = "https://youtu.be/cIQ_dXUjZ_g"
	if got := tutorialLine("merchant", url); !strings.Contains(got, url) {
		t.Fatalf("صاحبُ المتجر بلا رابط الشرح: %q", got)
	}
	for _, k := range []string{"rep", "driver", "customer", ""} {
		if got := tutorialLine(k, url); got != "" {
			t.Fatalf("%q: سطرُ الشرح وصل غيرَ المتجر: %q", k, got)
		}
	}
	if got := tutorialLine("merchant", "  "); got != "" {
		t.Fatalf("رابطٌ فارغ ولم يُحذف السطر: %q", got)
	}
}
