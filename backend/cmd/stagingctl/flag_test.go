package main

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// TestQAFlags_EveryAllowedKeyIsARealCatalogKey **قائمةُ السماح لا تخترع مفتاحاً.**
//
// **مفتاحٌ في القائمة ليس في الكتالوج يُقبَل ثمّ يُرفَض عند الكتابة** —
// **فالأولى أن يُكتشَف هنا لا على التجهيز.**
func TestQAFlags_EveryAllowedKeyIsARealCatalogKey(t *testing.T) {
	for key := range qaFlags {
		if _, ok := settings.Lookup(key); !ok {
			t.Errorf("مفتاحُ قائمة السماح %q ليس في الكتالوج", key)
		}
	}
}

// TestQAFlags_OnlyExpectedKeys **لا يتضخّم البابُ صامتاً.**
//
// **قائمةُ سماحٍ تكبر بلا انتباهٍ تصير «افعل أيَّ شيء»** — **فيُثبَّت عددُها
// ومحتواها، ومن زادها زاد هذا الاختبارَ معه عمداً.**
func TestQAFlags_OnlyExpectedKeys(t *testing.T) {
	want := map[string]bool{
		"launch.customer_signup":        true,
		"launch.customer_browse":        true,
		"launch.customer_orders":        true,
		"launch.customer_custom_orders": true,
		"launch.notice":                 true,
		"hours.platform_enforced":       true,
		"app.min_version.customer":      true,
		"platform.support_phone":        true,
		"platform.whatsapp":             true,
		"platform.facebook":             true,
		"platform.instagram":            true,
		"platform.telegram":             true,
		"platform.address":              true,
	}
	for k := range qaFlags {
		if !want[k] {
			t.Errorf("مفتاحٌ غيرُ متوقَّعٍ في قائمة السماح: %q", k)
		}
	}
	for k := range want {
		if _, ok := qaFlags[k]; !ok {
			t.Errorf("مفتاحٌ متوقَّعٌ غائبٌ من قائمة السماح: %q", k)
		}
	}
}

// TestQAFlags_NoWildcardOrSQL **لا حرفَ بدلٍ ولا مسافة في مفتاح.**
func TestQAFlags_NoWildcardOrSQL(t *testing.T) {
	for k := range qaFlags {
		for _, bad := range []string{"*", "%", " ", ";", "'", "\""} {
			if containsRune(k, bad) {
				t.Errorf("مفتاحٌ فيه محرفٌ خطر %q: %q", bad, k)
			}
		}
	}
}

func containsRune(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestParseByKind **تحويلُ النوع يفشل مغلقاً على الغموض.**
func TestParseByKind(t *testing.T) {
	if v, err := parseByKind(settings.KindBool, "true"); err != nil || v != true {
		t.Errorf("bool true: %v %v", v, err)
	}
	if v, err := parseByKind(settings.KindBool, "false"); err != nil || v != false {
		t.Errorf("bool false: %v %v", v, err)
	}
	if _, err := parseByKind(settings.KindBool, "maybe"); err == nil {
		t.Error("bool maybe: توقّعتُ خطأً")
	}
	if v, err := parseByKind(settings.KindInt, "42"); err != nil || v.(int64) != 42 {
		t.Errorf("int 42: %v %v", v, err)
	}
	if _, err := parseByKind(settings.KindInt, "4.2"); err == nil {
		t.Error("int 4.2: توقّعتُ خطأً")
	}
	if v, err := parseByKind(settings.KindText, "0999"); err != nil || v.(string) != "0999" {
		t.Errorf("text: %v %v", v, err)
	}
	// **ونوعٌ لا يُقلَب بالأداة يُرفَض** — الجغرافيا مثلاً.
	if _, err := parseByKind(settings.KindGeo, "1,2"); err == nil {
		t.Error("geo: توقّعتُ رفضاً")
	}
}
