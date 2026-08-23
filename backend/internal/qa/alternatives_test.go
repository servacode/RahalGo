package qa

import (
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **بدائلُ المسار — البند ٤٥**
// ══════════════════════════════════════════════════════════════════════
//
// (المرحلة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
//
// **والمحرّكُ لا يعمل في بيئة الاختبار** — `OSRM_URL` فارغةٌ فيردّ
// `available:false`. **وذلك بعينه ما يُفحص هنا**: العقدُ والصلاحيّاتُ
// والتوافقُ الخلفيّ **تعمل بلا محرّك**، **ولا تسقط شاشةٌ لأنّ خدمةَ
// مساراتٍ نامت.**
//
// **وسلوكُ الترشيح مُختبَرٌ في `internal/routing`** على رفيدةٍ ذهبيّةٍ
// من ٨٧ بديلاً مقيساً — **وهناك موضعُه**، لا في نداءٍ عبرَ الشبكة.

// ALT-001 **العقدُ القديمُ لم يتغيّر.**
//
// **أمرُ المالك نصّاً**: «old client لا يحمل payload البدائل إلا إذا
// طلبها».
func TestALT_001_LegacyContractUnchanged(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)

	res := h.GET(routePath(oid, ""), drv.Token)
	if res.Code != 200 {
		t.Fatalf("ALT-001: ردّ %d", res.Code)
	}
	body := res.JSON()

	for _, forbidden := range []string{
		"alternatives", "alternatives_meta", "route_id", "target",
		"weight_name", "engine_weight",
	} {
		if _, ok := body[forbidden]; ok {
			t.Errorf("ALT-001: من لم يطلب البدائلَ لا يحمل %q", forbidden)
		}
	}
}

// ALT-002 **ومن طلبها يجدها** — ولو فارغة.
//
// **البند ٣٤ من التحليل**: «primary + alternatives=[] هذا طبيعي وليس
// Failure».
func TestALT_002_RequestedShapeIsPresent(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)

	res := h.GET(routePath(oid, "alternatives=true"), drv.Token)
	if res.Code != 200 {
		t.Fatalf("ALT-002: ردّ %d", res.Code)
	}
	body := res.JSON()

	// **ولا محرّكَ في بيئة الاختبار** — فيردّ `available:false`،
	// **وذلك سلوكٌ محفوظٌ لا خلل.**
	if avail, _ := body["available"].(bool); !avail {
		if _, ok := body["alternatives"]; ok {
			t.Error("ALT-002: لا مسارَ ومع ذلك حقلُ بدائل")
		}
		return
	}

	if _, ok := body["alternatives"]; !ok {
		t.Error("ALT-002: طُلبت البدائلُ ولم يُردّ حقلُها")
	}
	if _, ok := body["route_id"]; !ok {
		t.Error("ALT-002: لا معرِّفَ مسار")
	}
	if target, _ := body["target"].(string); target != "pickup" && target != "dropoff" {
		t.Errorf("ALT-002: وجهةٌ غيرُ معروفة: %q", target)
	}
}

// ALT-003 **والصلاحيّةُ كما هي** — سائقٌ آخرُ لا يرى.
func TestALT_003_ForeignDriverDenied(t *testing.T) {
	h := New(t)
	oid, _ := h.assignedOrder(t)
	other := h.NewUser("driver")

	res := h.GET(routePath(oid, "alternatives=true"), other.Token)
	if res.Code == 200 {
		t.Errorf("ALT-003: سائقٌ آخرُ حصل على البدائل: %s", res)
	}
}

// ALT-004 **ولا رمزَ لا مسار.**
func TestALT_004_AnonymousDenied(t *testing.T) {
	h := New(t)
	oid, _ := h.assignedOrder(t)

	if res := h.GET(routePath(oid, "alternatives=true"), ""); res.Code != 401 {
		t.Errorf("ALT-004: بلا رمزٍ ردَّ %d", res.Code)
	}
}

// ALT-005 **والوجهةُ يحدّدها الخادم.**
//
// **البند ٨ من التحليل**: «الهاتف لا يحدد destination… ولا يقبل
// الهاتف: duration · distance · route score · make this primary».
//
// **فمعاملاتٌ تحاول ذلك تُتجاهَل** — لا تُقبل ولا تُسقط.
func TestALT_005_ClientCannotDictateRoute(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)

	base := h.GET(routePath(oid, "alternatives=true"), drv.Token)
	if base.Code != 200 {
		t.Fatalf("ALT-005: ردّ %d", base.Code)
	}

	injected := h.GET(
		routePath(oid, "alternatives=true&target=dropoff&duration_s=1&distance_m=1&primary=alt-1"),
		drv.Token,
	)
	if injected.Code != 200 {
		t.Fatalf("ALT-005: ردّ %d", injected.Code)
	}

	a, b := base.JSON(), injected.JSON()
	if a["target"] != b["target"] {
		t.Errorf("ALT-005: الهاتفُ بدّل الوجهة: %v ← %v", a["target"], b["target"])
	}
	if a["distance_m"] != b["distance_m"] {
		t.Errorf("ALT-005: الهاتفُ بدّل المسافة: %v ← %v", a["distance_m"], b["distance_m"])
	}
	if a["duration_s"] != b["duration_s"] {
		t.Errorf("ALT-005: الهاتفُ بدّل المدّة: %v ← %v", a["duration_s"], b["duration_s"])
	}
}

// ALT-006 **والأصلُ المحلّيُّ يُفحص كما في ٣ب.**
func TestALT_006_LocalOriginValidated(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)

	if res := h.GET(routePath(oid, "alternatives=true&lat=35.95&lng=39.005"), drv.Token); res.Code != 200 {
		t.Errorf("ALT-006: أصلٌ صحيحٌ ردَّ %d", res.Code)
	}
	// **ونصفُ زوجٍ خطأُ تحقّق** — كما في المرحلة ٣ب.
	if res := h.GET(routePath(oid, "alternatives=true&lat=35.95"), drv.Token); res.Code != 400 {
		t.Errorf("ALT-006: نصفُ زوجٍ ردَّ %d", res.Code)
	}
	if res := h.GET(routePath(oid, "alternatives=true&lat=999&lng=39"), drv.Token); res.Code != 400 {
		t.Errorf("ALT-006: خطُّ عرضٍ خارجَ المدى ردَّ %d", res.Code)
	}
}

// ALT-007 **والمخبآن لا يتشاركان** — البند ١٧.
//
// **فطلبٌ مفردٌ ثمّ طلبُ بدائلَ لا يردّ حمولةَ الأوّل.**
func TestALT_007_CacheModesSeparated(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)

	// **المفردُ أوّلاً** — فيملأ مخبأه.
	single := h.GET(routePath(oid, ""), drv.Token)
	if single.Code != 200 {
		t.Fatalf("ALT-007: ردّ %d", single.Code)
	}
	if _, ok := single.JSON()["alternatives"]; ok {
		t.Error("ALT-007: المفردُ حمل بدائل")
	}

	// **ثمّ البدائل** — ولا يقرأ مخبأ المفرد.
	withAlts := h.GET(routePath(oid, "alternatives=true"), drv.Token)
	if withAlts.Code != 200 {
		t.Fatalf("ALT-007: ردّ %d", withAlts.Code)
	}
	body := withAlts.JSON()
	if avail, _ := body["available"].(bool); avail {
		if _, ok := body["alternatives"]; !ok {
			t.Error("ALT-007: البدائلُ قرأت مخبأ المفرد فضاع حقلُها")
		}
	}
}

// ALT-008 **ومعاملٌ غيرُ `true` لا يُفعّل البدائل.**
func TestALT_008_OnlyExplicitTrueEnables(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)

	for _, q := range []string{"alternatives=1", "alternatives=yes", "alternatives=", "alternatives=TRUE"} {
		res := h.GET(routePath(oid, q), drv.Token)
		if res.Code != 200 {
			t.Fatalf("ALT-008 %q: ردّ %d", q, res.Code)
		}
		if _, ok := res.JSON()["alternatives"]; ok {
			t.Errorf("ALT-008: %q فعّلت البدائل — والصريحُ وحدَه يفعل", q)
		}
	}
}

// ALT-009 **وحبيبتان لا واحدة** — إغلاقُ صحّة ٧، البندان ١ و٢.
//
// **أمرُ المالك نصّاً**: «لا أقبل أن request B يستلم Cached RouteSet
// المبنية للأصل A لمجرد أن A وB وقعا في خلية 4 decimals ≈11m».
//
// **والمفردُ لا يُمسّ** — «لا أريد تغيير السلوك القديم بلا داعٍ».
func TestALT_009_CacheGrainSeparated(t *testing.T) {
	src := readSource(t, "../server/driver_route.go")

	for _, needle := range []string{
		"cellDecimalsSingle = 4",
		"cellDecimalsAlts = 5",
		"altOriginToleranceM",
	} {
		if !strings.Contains(src, needle) {
			t.Errorf("ALT-009: %q غائب", needle)
		}
	}

	// **والأصلُ يُحفظ مع المجموعة** — فالحمايةُ قطعيّةٌ لا إحصائيّة.
	if !strings.Contains(src, "OriginLat") || !strings.Contains(src, "cacheEntryUsable") {
		t.Error("ALT-009: المجموعةُ لا تحفظ أصلَها ولا تُفحص عنده")
	}
}

// ALT-010 **وموضعُ القرار يُرسل** — إغلاقُ صحّة ٧، البنود ٦ إلى ١٠.
func TestALT_010_DecisionDivergenceInContract(t *testing.T) {
	src := readSource(t, "../server/driver_route.go")
	if !strings.Contains(src, "decision_divergence_m") {
		t.Error("ALT-010: موضعُ القرار غيرُ مرسَل")
	}
	if !strings.Contains(src, "routing.MeaningfulDivergence") {
		t.Error("ALT-010: يُحسب بغير `MeaningfulDivergence`")
	}
}
