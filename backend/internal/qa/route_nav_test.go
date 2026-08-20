package qa

// ══════════════════════════════════════════════════════════════════════
// **الطبقةُ الرابعة عشرة — عقدُ مسار الملاحة**
// ══════════════════════════════════════════════════════════════════════
//
// المعرّفات: `RNAV-*` · الوسم: `@api @release`
//
// (المرحلة ٢، أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
//
// # وما يُقاس
//
// **أنّ الحقولَ الأربعةَ القديمةَ لم تُمسّ** — والنسخةُ المنشورةُ
// تعمل، **وأنّ الملاحةَ حقولٌ إضافيّةٌ لا بديلة.**

import (
	"os"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/routing"
)

// TestRNAV_001_CacheKeyIsVersioned **ومفتاحُ المخبأ يحمل نسختَه.**
//
// **ومسارٌ خُزّن قبل الخطوات يُفكّ بلا مناورات** — فيصير «متاحاً» بلا
// إرشاد، **ولا خطأَ يظهر.**
func TestRNAV_001_CacheKeyIsVersioned(t *testing.T) {
	src := readSource(t, "../server/driver_route.go")
	if !strings.Contains(src, `routeCacheVersion = "v2"`) {
		t.Error("RNAV-001 **لا نسخةَ في مفتاح المخبأ**")
	}
	if !strings.Contains(src, `"route:" + routeCacheVersion`) {
		t.Error("RNAV-001 **المفتاحُ لا يحمل النسخة**")
	}
}

// TestRNAV_002_StepsAreRequested **و`steps=true` في نداء المحرّك.**
func TestRNAV_002_StepsAreRequested(t *testing.T) {
	src := readSource(t, "../routing/osrm.go")
	if strings.Contains(src, "steps=false") {
		t.Error("RNAV-002 **ما زال يطلب بلا خطوات**")
	}
	if !strings.Contains(src, "steps=true") {
		t.Error("RNAV-002 **لا يطلب خطوات**")
	}
}

// TestRNAV_010_LegacyFieldsUnchanged **والحقولُ الأربعةُ كما كانت.**
//
// **ونسخةُ السائق المنشورةُ تقرأ هذه وحدَها** — ومن بدّل معناها كسر
// من لم يحدّث.
func TestRNAV_010_LegacyFieldsUnchanged(t *testing.T) {
	src := readSource(t, "../server/driver_route.go")
	for _, f := range []string{
		`"available":  true`, `"distance_m"`, `"duration_s"`, `"points"`,
	} {
		if !strings.Contains(src, f) {
			t.Errorf("RNAV-010 **حقلٌ قديمٌ غاب**: %s", f)
		}
	}
	// **و`available:false` عند سقوط المحرّك تبقى.**
	if !strings.Contains(src, `"available": false`) {
		t.Error("RNAV-010 **احتياطُ سقوط المحرّك غاب**")
	}
}

// TestRNAV_011_NavFieldsOnlyWhenPresent **ولا تُرسَل الملاحةُ فارغة.**
func TestRNAV_011_NavFieldsOnlyWhenPresent(t *testing.T) {
	src := readSource(t, "../server/driver_route.go")
	if !strings.Contains(src, "if route.HasNavigation()") {
		t.Error("RNAV-011 **تُرسَل الملاحةُ بلا شرط**")
	}
}

// TestRNAV_020_ContractHasNoOsrmWords **ولا كلمةَ OSRM في عقد الجوّال.**
//
// **ومن مرّرها ربط تطبيقَ السائق بمحرّكٍ بعينه** — فيومَ يُبدَّل
// يُعاد بناءُ التطبيق كلِّه.
func TestRNAV_020_ContractHasNoOsrmWords(t *testing.T) {
	src := readSource(t, "../routing/maneuver.go")
	// **وكلماتُ المحرّك تُذكر في التحويل وحدَه** — لا في التعداد.
	for _, bad := range []string{
		`ManeuverKind = "exit rotary"`, `ManeuverKind = "slight left"`,
	} {
		if strings.Contains(src, bad) {
			t.Errorf("RNAV-020 **كلمةُ محرّكٍ في عقدنا**: %s", bad)
		}
	}
	// **وكلُّ ما نعرفه بمفاهيمنا.**
	for _, want := range []string{"TURN_LEFT", "ROUNDABOUT", "EXIT_ROUNDABOUT", "UNKNOWN"} {
		if !strings.Contains(src, want) {
			t.Errorf("RNAV-020 مفهومٌ ناقص: %s", want)
		}
	}
}

// TestRNAV_030_NoNavWhenNoSteps **ومسارٌ بلا خطواتٍ لا يدّعي ملاحة.**
func TestRNAV_030_NoNavWhenNoSteps(t *testing.T) {
	bare := &routing.Route{DistanceM: 900, DurationS: 80}
	if bare.HasNavigation() {
		t.Error("RNAV-030 **ادّعى ملاحةً بلا مناورات**")
	}
	// **وتراكميّةٌ لا تطابق الهندسةَ ليست ملاحة** — انظر `HasNavigation`.
	mismatch := &routing.Route{
		Geometry:    []routing.Point{{Lat: 1}, {Lat: 2}, {Lat: 3}},
		CumulativeM: []float64{0, 1},
		Maneuvers:   []routing.Maneuver{{Kind: routing.KindDepart}},
	}
	if mismatch.HasNavigation() {
		t.Error("RNAV-030 **تراكميّةٌ مختلفةُ الطول عُدّت ملاحة**")
	}
}

// readSource **يقرأ مصدراً للفحص** — والحارسُ يقرأ ما كُتب لا ما يُظنّ.
func readSource(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ %s: %v", path, err)
	}
	return string(b)
}
