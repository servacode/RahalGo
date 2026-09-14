package qa

// ══════════════════════════════════════════════════════════════════════
// **هويّةُ الاشتراك — ما وعد به الزرّ** (`SI`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **والزرُّ قال «أخبرني عند توفّر الخدمة في دمشق»** — **لا «في هذه
// الخليّة».** **ودمشقُ نصفُ قطرها ٢٥ كم والخليّةُ ١٫١** — **فألفُ
// خليّةٍ وأكثرُ داخلَ مدينةٍ واحدة.**
//
// **ومن ضغطها من عنوانين صار هدفَين فيُخبَر مرّتين** — **وهذا ما
// يمنعه هذا الملفّ.**

import (
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/opsmap"
)

// نقطتان متباعدتان داخلَ مدينة دمشق (نصفُ القطر ٢٥ كم).
//
// **والفرقُ بينهما ≈ ٨ كم** — **سبعُ خلايا فأكثر**، **ومدينةٌ واحدة.**
const (
	damA_Lat, damA_Lng = 33.5138, 36.2765
	damB_Lat, damB_Lng = 33.5600, 36.3300
)

// حلبُ — مدينةٌ أخرى مبذورةٌ مُطفأة.
const aleppoLat, aleppoLng = 36.2021, 37.1343

// نقطتان في البادية لا مدينةَ لهما — وخليّتاهما مختلفتان.
const (
	wildA_Lat, wildA_Lng = 34.2000, 38.6000
	wildB_Lat, wildB_Lng = 34.4000, 38.9000
)

// targets يقرأ أهدافَ حسابٍ من نوعٍ بعينه.
func targets(t *testing.T, h *Harness, userID, kind string) map[string]bool {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT target_key, active FROM coverage_requests
		 WHERE user_id = $1::uuid AND kind = $2`, userID, kind)
	if err != nil {
		t.Fatalf("قراءةُ الأهداف: %v", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		var a bool
		if err := rows.Scan(&k, &a); err != nil {
			t.Fatalf("هدف: %v", err)
		}
		out[k] = a
	}
	return out
}

// ═════════════════ SI-01 · SI-10 — مدينةٌ واحدةٌ هدفٌ واحد ═════════════════

// TestSI01_SI10_TwoPointsOneCityOneTarget **عنوانان في دمشقَ اشتراكٌ
// واحد.**
func TestSI01_SI10_TwoPointsOneCityOneTarget(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ SI-01")
	u := hh.Customer()

	first := demand(t, hh, u.Token, damA_Lat, damA_Lng, "")
	if first.Code != http.StatusOK || first.JSON()["outcome"] != "created" {
		t.Fatalf("الأوّل: %d / %v", first.Code, first.JSON())
	}
	second := demand(t, hh, u.Token, damB_Lat, damB_Lng, "")
	if second.Code != http.StatusOK {
		t.Fatalf("الثاني: %d / %s", second.Code, second.Err())
	}
	if second.JSON()["outcome"] != "already_registered" {
		t.Fatalf("**عنوانٌ ثانٍ في دمشقَ أنشأ هدفاً ثانياً**: %v — "+
			"**فيُخبَر مرّتين يومَ تُطلَق**", second.JSON()["outcome"])
	}

	// **SI-10 · وضغطاتٌ مكرّرةٌ لا تؤذي.**
	for i := 0; i < 3; i++ {
		if r := demand(t, hh, u.Token, damA_Lat, damA_Lng, ""); r.Code != http.StatusOK {
			t.Fatalf("**ضغطةٌ مكرّرةٌ رُدّت**: %d / %s", r.Code, r.Err())
		}
	}

	got := targets(t, hh, u.ID, "service_interest")
	if len(got) != 1 {
		t.Fatalf("**أهدافٌ متعدّدةٌ لمدينةٍ واحدة**: %v", got)
	}
	for k := range got {
		if len(k) < 5 || k[:5] != "city:" {
			t.Fatalf("**هدفُ مدينةٍ معروفةٍ ليس بالمدينة**: %q", k)
		}
	}
}

// ═════════════════ SI-02 · SI-03 — مدينتان هدفان ═════════════════

// TestSI02_SI03_DamascusAndAleppoAreDistinct **ودمشقُ غيرُ حلب —
// وإلغاءُ إحداهما لا يمسّ الأخرى.**
func TestSI02_SI03_DamascusAndAleppoAreDistinct(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ SI-02")
	u := hh.Customer()

	if r := demand(t, hh, u.Token, damA_Lat, damA_Lng, ""); r.Code != http.StatusOK {
		t.Fatalf("دمشق: %d / %s", r.Code, r.Err())
	}
	aleppo := demand(t, hh, u.Token, aleppoLat, aleppoLng, "")
	if aleppo.Code != http.StatusOK {
		t.Fatalf("حلب: %d / %s", aleppo.Code, aleppo.Err())
	}
	if aleppo.JSON()["outcome"] != "created" {
		t.Fatalf("**حلبُ دُمجت في دمشق**: %v", aleppo.JSON()["outcome"])
	}
	if got := targets(t, hh, u.ID, "service_interest"); len(got) != 2 {
		t.Fatalf("**مدينتان ولم يُصنَع هدفان**: %v", got)
	}

	// **SI-03 · وإلغاءُ دمشقَ لا يمسّ حلب.**
	if r := hh.POST("/api/v1/me/demand/cancel", u.Token,
		map[string]any{"lat": damB_Lat, "lng": damB_Lng}); r.Code != http.StatusOK {
		t.Fatalf("**الإلغاءُ من عنوانٍ ثانٍ في دمشقَ رُدّ**: %d / %s", r.Code, r.Err())
	}
	got := targets(t, hh, u.ID, "service_interest")
	dam, alp := 0, 0
	for _, active := range got {
		if active {
			alp++
		} else {
			dam++
		}
	}
	if dam != 1 || alp != 1 {
		t.Fatalf("**الإلغاءُ لم يُصِب هدفَه وحدَه**: %v", got)
	}
}

// ═════════════════ SI-04 · SI-05 — المجهولُ بالخليّة ═════════════════

// TestSI04_SI05_UnknownAreasStayCellScoped **وموضعان مجهولان هدفان.**
//
// **ولا مكانَ يُسمّى ليكون هدفاً** — **فالخليّةُ هي كلُّ ما نعرف.**
func TestSI04_SI05_UnknownAreasStayCellScoped(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ SI-04")
	u := hh.Customer()

	for _, p := range [][2]float64{{wildA_Lat, wildA_Lng}, {wildB_Lat, wildB_Lng}} {
		r := demand(t, hh, u.Token, p[0], p[1], "")
		if r.Code != http.StatusOK {
			t.Fatalf("%v: %d / %s", p, r.Code, r.Err())
		}
		if r.JSON()["reason"] != "area_not_supported" {
			t.Fatalf("%v: **سببٌ غيرُ متوقَّع**: %v", p, r.JSON()["reason"])
		}
	}
	got := targets(t, hh, u.ID, "service_interest")
	if len(got) != 2 {
		t.Fatalf("**موضعان مجهولان دُمجا**: %v", got)
	}
	for k := range got {
		if len(k) < 5 || k[:5] != "cell:" {
			t.Fatalf("**موضعٌ مجهولٌ نُسب إلى مكان**: %q", k)
		}
	}

	// **SI-05 · وإلغاءُ إحداهما لا يمسّ الأخرى.**
	if r := hh.POST("/api/v1/me/demand/cancel", u.Token,
		map[string]any{"lat": wildA_Lat, "lng": wildA_Lng}); r.Code != http.StatusOK {
		t.Fatalf("**الإلغاءُ رُدّ**: %d / %s", r.Code, r.Err())
	}
	on, off := 0, 0
	for _, active := range targets(t, hh, u.ID, "service_interest") {
		if active {
			on++
		} else {
			off++
		}
	}
	if on != 1 || off != 1 {
		t.Fatalf("**الإلغاءُ تجاوز خليّتَه**: سارٍ=%d مُلغىً=%d", on, off)
	}
}

// ═════════════════ SI-06 · SI-07 — طلبُ التغطية مكانيٌّ كما كان ═════════════════

// TestSI06_SI07_CoverageStaysCellScoped **وحيّان خارجَ نطاق الرقّة
// طلبان.**
//
// **ولو جُمعا بالمدينة لصارت الرقّةُ صفّاً واحداً لا يقول أين
// يُوسَّع.**
func TestSI06_SI07_CoverageStaysCellScoped(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ SI-06")
	u := hh.Customer()

	// **نقطتان في مدينة الرقّة خارجَ المنطقة، وفي خليّتين مختلفتين.**
	for _, p := range [][2]float64{{raqqaLat + 0.08, raqqaLng}, {raqqaLat + 0.10, raqqaLng + 0.03}} {
		r := demand(t, hh, u.Token, p[0], p[1], "")
		if r.Code != http.StatusOK {
			t.Fatalf("%v: %d / %s", p, r.Code, r.Err())
		}
		if r.JSON()["kind"] != "coverage_request" {
			t.Fatalf("%v: **نيّةٌ غيرُ متوقَّعة**: %v", p, r.JSON()["kind"])
		}
	}
	got := targets(t, hh, u.ID, "coverage_request")
	if len(got) != 2 {
		t.Fatalf("**حيّان دُمجا في طلبٍ واحد** — **فلا يُعرَف أين يُوسَّع**: %v", got)
	}
	for k := range got {
		if len(k) < 5 || k[:5] != "cell:" {
			t.Fatalf("**طلبُ التغطية نُسب إلى مدينةٍ لا إلى حيّ**: %q", k)
		}
	}
}

// ═════════════════ SI-08 · SI-09 — سؤالُ الدفعة الثامنة ═════════════════

// TestSI08_SI09_TargetQueryIsUniqueAndRespectsCancel **من يُخبَر يومَ
// تُطلَق دمشق؟**
//
// **ولا تنقيةَ عند الإرسال** — **والسؤالُ يُجاب مرّةً واحدةً لكلّ
// حساب.**
func TestSI08_SI09_TargetQueryIsUniqueAndRespectsCancel(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ SI-08")

	var damascus string
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT id::text FROM cities
		 WHERE ST_DWithin(center, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography, radius_m)
		 LIMIT 1`, damA_Lat, damA_Lng).Scan(&damascus); err != nil {
		t.Skipf("لا مدينةَ تحوي النقطة: %v", err)
	}

	before := len(mustTargets(t, hh, damascus))

	u := hh.Customer()
	// **عنوانان في دمشق.**
	for _, p := range [][2]float64{{damA_Lat, damA_Lng}, {damB_Lat, damB_Lng}} {
		if r := demand(t, hh, u.Token, p[0], p[1], ""); r.Code != http.StatusOK {
			t.Fatalf("%v: %d / %s", p, r.Code, r.Err())
		}
	}

	list := mustTargets(t, hh, damascus)
	seen := 0
	for _, id := range list {
		if id == u.ID {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("**الحسابُ يظهر %d مرّةً في قائمة من يُخبَر** — "+
			"**فيصله إشعاران**", seen)
	}
	if len(list) != before+1 {
		t.Fatalf("**عددُ من يُخبَر تبدّل بأكثر من حساب**: %d ← %d", before, len(list))
	}

	// **SI-09 · والمُلغي لا يُستهدَف.**
	if r := hh.POST("/api/v1/me/demand/cancel", u.Token,
		map[string]any{"lat": damA_Lat, "lng": damA_Lng}); r.Code != http.StatusOK {
		t.Fatalf("**الإلغاءُ رُدّ**: %d / %s", r.Code, r.Err())
	}
	for _, id := range mustTargets(t, hh, damascus) {
		if id == u.ID {
			t.Fatal("**من ألغى اشتراكَه ما زال يُستهدَف** — " +
				"**ومن طلب ألّا يُخبَر لا يُخبَر**")
		}
	}
}

// mustTargets ينادي استعلامَ الاستهداف نفسَه الذي ستقرؤه الدفعةُ الثامنة.
func mustTargets(t *testing.T, h *Harness, cityID string) []string {
	t.Helper()
	ids, err := opsmap.InterestedInCity(ctxBG(), h.Pool, cityID)
	if err != nil {
		t.Fatalf("استعلامُ الاستهداف: %v", err)
	}
	return ids
}
