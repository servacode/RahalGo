package qa

import (
	"os"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حرّاسُ المرحلة ٨ب — ما لا يجوز أن ينزلق**
// ══════════════════════════════════════════════════════════════════════

// TestPAR_010_LegacyRouteUnchanged **`/route` القديم لا يتبدّل.**
//
// **البند ٢**: المرحلةُ ٧ ثبّتت أنّ الردَّ يبقى كما هو، **ومن أراد
// معرّفاً طلبه.**
func TestPAR_010_LegacyRouteUnchanged(t *testing.T) {
	src := readSource(t, "../server/driver_route.go")
	if !strings.Contains(src, `r.URL.Query().Get("correlation") == "true"`) {
		t.Fatal("لا اشتراكَ صريحٌ لبيانات الارتباط")
	}
	// **ولا يُرسَل المعرّفُ إلّا لمن طلب** — أحدَ الشرطين.
	i := strings.Index(src, `out["route_id"]`)
	if i < 0 {
		t.Fatal("لا معرّف")
	}
	head := src[:i]
	if !strings.Contains(head, "if wantCorrelation {") &&
		!strings.Contains(head, "if wantAlternatives {") {
		t.Fatal("المعرّفُ يُرسَل بلا اشتراك")
	}
}

// TestPAR_011_MobileKnowsNoEngine **الجوّالُ لا يعرف محرّكاً.**
//
// **البند ١٤**: لا `OSRM` ولا `matchings` ولا `hint` ولا عقدُ OSM.
func TestPAR_011_MobileKnowsNoEngine(t *testing.T) {
	roots := []string{
		"../../../mobile/driver-navigation/src/main/kotlin/com/rahalgo/navigation",
		"../../../mobile/shared/src/main/kotlin/com/rahalgo/shared/model",
		"../../../mobile/shared/src/main/kotlin/com/rahalgo/shared/driver",
	}
	bad := []string{"osrm", "matchings", "tracepoints", "alternatives_count",
		"osm_node", "hint="}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Skipf("لا مجلّد: %s", root)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".kt") {
				continue
			}
			raw, err := os.ReadFile(root + "/" + e.Name())
			if err != nil {
				t.Fatalf("قراءة %s: %v", e.Name(), err)
			}
			low := strings.ToLower(stripKotlinComments(string(raw)))
			for _, b := range bad {
				if strings.Contains(low, b) {
					t.Fatalf("%s فيه %q — والجوّالُ لا يعرف المحرّك", e.Name(), b)
				}
			}
		}
	}
}

// TestPAR_012_NoRerouteFromHttp **لا إعادةَ حسابٍ من ردّ شبكة.**
//
// **البند ٢١**: الردُّ يُغيّر حالَ المُحلِّل، **والحالُ وحدَها تُنتج
// سبباً.**
func TestPAR_012_NoRerouteFromHttp(t *testing.T) {
	src := readSource(t,
		"../../../mobile/driver-navigation/src/main/kotlin/com/rahalgo/navigation/ParallelResolver.kt")
	code := stripKotlinComments(src)
	for _, bad := range []string{"reroute(", "RerouteEngine", "setRoute("} {
		if strings.Contains(code, bad) {
			t.Fatalf("المُحلِّلُ ينادي %q — والحدودُ يجب أن تبقى", bad)
		}
	}
	// **ولا شبكةَ في `driver-navigation`.**
	for _, bad := range []string{"http", "Ktor", "okhttp", "URL("} {
		if strings.Contains(strings.ToLower(code), strings.ToLower(bad)) {
			t.Fatalf("المُحلِّلُ يعرف الشبكة: %q", bad)
		}
	}
}

// TestPAR_013_FlagDefaultsOff **الرايةُ مطفأةٌ في المصدر.**
//
// **البند ١**: التشغيلُ ينتظر قياساً ميدانيّاً.
func TestPAR_013_FlagDefaultsOff(t *testing.T) {
	src := readSource(t,
		"../../../mobile/driver-navigation/src/main/kotlin/com/rahalgo/navigation/NavFeatures.kt")
	if !strings.Contains(stripKotlinComments(src), "var parallelResolver: Boolean = false") {
		t.Fatal("رايةُ المُحلِّل ليست مطفأةً افتراضاً")
	}
	res := readSource(t,
		"../../../mobile/driver-navigation/src/main/kotlin/com/rahalgo/navigation/ParallelResolver.kt")
	if !strings.Contains(stripKotlinComments(res), "var enabled: Boolean = false") {
		t.Fatal("المُحلِّلُ مشتعلٌ افتراضاً")
	}
}

// TestPAR_014_DetectorsUntouched **الكواشفُ لم تُمَسّ — البند ٣٧.**
//
// **ولا `parallel` ولا `correlation` في منطقها.**
func TestPAR_014_DetectorsUntouched(t *testing.T) {
	files := []string{"OffRouteDetector.kt", "WrongWayDetector.kt", "RerouteEngine.kt"}
	for _, f := range files {
		src := stripKotlinComments(readSource(t,
			"../../../mobile/driver-navigation/src/main/kotlin/com/rahalgo/navigation/"+f))
		for _, bad := range []string{"parallel", "Parallel", "correlation", "Correlation"} {
			if strings.Contains(src, bad) {
				t.Fatalf("%s فيه %q — والمُحلِّلُ لا يدخل الكواشف", f, bad)
			}
		}
	}
}

// TestPAR_015_ProgressIsPassThroughOnly **`RouteProgress` تمريرٌ لا منطق.**
//
// **تغييرُ مُحوِّلٍ مصرَّحٌ به** (البند ٣٧): حقلٌ يُمرَّر، **ولا حكمَ
// يُبنى عليه هناك.**
func TestPAR_015_ProgressIsPassThroughOnly(t *testing.T) {
	src := stripKotlinComments(readSource(t,
		"../../../mobile/driver-navigation/src/main/kotlin/com/rahalgo/navigation/RouteProgress.kt"))
	if !strings.Contains(src, "lateralSignedM") {
		t.Fatal("الإزاحةُ الموقّعةُ لا تُمرَّر")
	}
	// **ولا شرطَ ولا مقارنةَ عليها** — تمريرٌ محض.
	for _, bad := range []string{
		"if (lateralSignedM", "lateralSignedM >", "lateralSignedM <",
		"abs(lateralSignedM",
	} {
		if strings.Contains(src, bad) {
			t.Fatalf("منطقٌ على الإزاحة الموقّعة داخلَ RouteProgress: %q", bad)
		}
	}
	if strings.Contains(src, "ParallelSuspicion") || strings.Contains(src, "ArrayDeque") {
		t.Fatal("RouteProgress صارت مخزنَ تاريخٍ — والبند ١٧ يمنع")
	}
}

// TestPAR_016_ServerValidatesOwnership **المعرّفُ ليس إذناً — البند ٤.**
func TestPAR_016_ServerValidatesOwnership(t *testing.T) {
	src := readSource(t, "../server/road_correlation.go")
	if !strings.Contains(src, "o.driver_id = $2::uuid") {
		t.Fatal("المنفذُ لا يتحقّق أنّ الطلبَ لهذا السائق")
	}
	if !strings.Contains(src, "MaxBytesReader") {
		t.Fatal("لا سقفَ لحجم الحمولة")
	}
	if !strings.Contains(src, "routing.MaxTraceFixes") {
		t.Fatal("لا سقفَ لعدد القراءات")
	}
	// **ولا يُردّ شيءٌ يخصّ محرّكاً** — البند ١٦.
	for _, bad := range []string{`"nodes"`, `"confidence"`, `"margin"`, `"offset_m"`} {
		if strings.Contains(src, "httpx.JSON") && strings.Contains(src, bad) {
			t.Fatalf("الردُّ يحمل %s — والعقدُ محايد", bad)
		}
	}
}
