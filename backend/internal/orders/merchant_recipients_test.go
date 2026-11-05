package orders

import "testing"

// TestRecipientPhoneKey **رقمُ المستلِم الواحدُ بصيغه كلِّها مفتاحٌ واحد** — وبه يُخفى
// ويُجمَع (قرارُ المالك ٢٠٢٦-١١-٠٥: صفٌّ لكلّ مستلِم).
func TestRecipientPhoneKey(t *testing.T) {
	const want = "+963935111222"
	for _, raw := range []string{
		"0935111222", "+963935111222", "00963935111222", "963935111222", "935111222",
		" 0935 111 222 ", "0935-111-222",
		"٠٩٣٥١١١٢٢٢", "+٩٦٣٩٣٥١١١٢٢٢", "۰۹۳۵۱۱۱۲۲۲",
	} {
		got, ok := RecipientPhoneKey(raw)
		if !ok || got != want {
			t.Fatalf("%q ⇒ %q (%v) والمنتظَر %q", raw, got, ok, want)
		}
	}
	for _, bad := range []string{"", "12", "0211234567", "+905321234567", "abc"} {
		if got, ok := RecipientPhoneKey(bad); ok {
			t.Fatalf("%q قُبل رقماً: %q", bad, got)
		}
	}
}
