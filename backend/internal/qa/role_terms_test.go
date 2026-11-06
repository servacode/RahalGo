package qa

import (
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

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
	// **نصُّ اللوحة يغلب الأصل، وبندُ «التعامل برّا المنصّة» حاضرٌ دائماً** (قرارُ المالك
	// ٢٠٢٦-١١-٠٥) — يُلحَق بأيّ نصٍّ لا يحويه، مرّةً واحدة (`settings.EffectiveDriverTerms`،
	// وحارسُه `TestDriverTerms_ClauseAlwaysPresent`). **فالحقيقةُ اثنتان معاً لا نصُّ اللوحة وحده.**
	v := str(body["driver_terms_text"])
	if !strings.HasPrefix(v, "شروطٌ من اللوحة") || strings.Contains(v, settings.DefaultDriverTerms) {
		t.Fatalf("RTX02 نصُّ اللوحة لم يغلب الأصل: %q", v)
	}
	if strings.Count(v, settings.DriverTermsClause) != 1 {
		t.Fatalf("RTX02 بندُ التعامل برّا المنصّة ليس مرّةً واحدة: %q", v)
	}
	if v != settings.EffectiveDriverTerms("شروطٌ من اللوحة") {
		t.Fatalf("RTX02 النصُّ غيرُ ما يُلحقه البند: %q", v)
	}
}

// **وسطرُ نقاط الخريطة في خصوصيّة السائق ولو عدّلها المالك** (قرارُ المالك ٢٠٢٦-١١-٠٦) — نصُّ
// اللوحة يغلب الأصل، ويُلحَق به السطرُ مرّةً واحدة (`settings.EffectiveDriverPrivacy`).
func TestRTX03_DriverPrivacyAlwaysMentionsMapPoints(t *testing.T) {
	h := New(t)
	h.Setting("page.driver_privacy_text", `"خصوصيّةٌ من اللوحة"`)
	got := h.GET("/api/v1/public/contact", "")
	body := got.JSON()
	if d, ok := body["data"].(map[string]any); ok {
		body = d
	}
	v := str(body["driver_privacy_text"])
	if !strings.HasPrefix(v, "خصوصيّةٌ من اللوحة") || strings.Count(v, settings.DriverMapTraceLine) != 1 {
		t.Fatalf("RTX03 خصوصيّةُ السائق بلا سطر نقاط الخريطة: %q", v)
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
