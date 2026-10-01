package qa

import "testing"

// ══════════════════════════════════════════════════════════════════════
// **شروطٌ وخصوصيّةٌ لكلّ دور** — `RTX` (قرارُ المالك ٢٠٢٦-١٠-٠١)
// ══════════════════════════════════════════════════════════════════════
//
// («اكيد لازم يكون نصّ خاصّ للمتجر ونصّ خاصّ للمندوب ونصّ خاصّ للسائق».)
//
// **كان نصٌّ واحدٌ للجميع مكتوباً لمن يشتري** — فقرأ صاحبُ المتجر
// «تختار من قائمةٍ نعرضها». **والمقيسُ هنا أنّ الستّةَ تصل في النداء
// العامّ، ولكلٍّ نصُّه لا نصُّ الزبون، وما كُتب في اللوحة يغلب الأصل.**

var rtxKeys = []string{
	"merchant_terms_text", "merchant_privacy_text",
	"rep_terms_text", "rep_privacy_text",
	"driver_terms_text", "driver_privacy_text",
}

func TestRTX01_EachRoleHasItsOwnTexts(t *testing.T) {
	h := New(t)
	got := h.GET("/api/v1/public/contact", "")
	if got.Code != 200 {
		t.Fatalf("النداءُ العامّ: %s", got)
	}
	body := got.JSON()
	if d, ok := body["data"].(map[string]any); ok {
		body = d
	}
	general := map[string]string{
		"terms":   str(body["terms_text"]),
		"privacy": str(body["privacy_text"]),
	}
	seen := map[string]bool{}
	for _, k := range rtxKeys {
		v := str(body[k])
		if v == "" {
			t.Errorf("RTX01 %s فارغ — يقرأ الدورُ نصَّ الزبون", k)
			continue
		}
		if v == general["terms"] || v == general["privacy"] {
			t.Errorf("RTX01 %s هو النصُّ العامُّ نفسُه", k)
		}
		if seen[v] {
			t.Errorf("RTX01 %s يكرّر نصَّ دورٍ آخر", k)
		}
		seen[v] = true
	}
}

func TestRTX02_PanelTextOverridesDefault(t *testing.T) {
	h := New(t)
	h.Setting("page.driver_terms_text", `"شروطٌ من اللوحة"`)
	got := h.GET("/api/v1/public/contact", "")
	body := got.JSON()
	if d, ok := body["data"].(map[string]any); ok {
		body = d
	}
	if v := str(body["driver_terms_text"]); v != "شروطٌ من اللوحة" {
		t.Fatalf("RTX02 نصُّ اللوحة لم يغلب الأصل: %q", v)
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
