package routing

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **تحويلُ OSRM إلى عقد رحّال غو — على ردٍّ حقيقيّ**
// ══════════════════════════════════════════════════════════════════════
//
// (المرحلة ٢، أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
//
// **والردُّ هنا منسوخٌ من محرّكنا نفسِه** (قِيس على `short-raqqa`
// ٢٠٢٦-٠٨-٢٠): دوّاران باسمين عربيّين، وسبعُ خطواتٍ من ثمانٍ بلا اسم.
//
// **وردٌّ أخترعه أنا يختبر خيالي** — **وهذا يختبر ما يصل فعلاً.**

// raqqaBody ردٌّ مصغَّرٌ بأسماءِ حقولِ المحرّك وقيمِه الحقيقيّة.
const raqqaBody = `{"code":"Ok","routes":[{"distance":1894,"duration":133,
 "geometry":{"coordinates":[[39.008585,35.950563],[39.009108,35.954243]]},
 "legs":[{"steps":[
  {"name":"Adnan Malki Street","rotary_name":"","distance":440,"duration":25,
   "geometry":{"coordinates":[[39.008585,35.950563],[39.008600,35.952000],[39.009108,35.954243]]},
   "maneuver":{"type":"depart","modifier":"right","exit":null},
   "intersections":[{"in":0,"out":1}]},
  {"name":"","rotary_name":"دوار النعيم","distance":123,"duration":7,
   "geometry":{"coordinates":[[39.009108,35.954243],[39.009462,35.954790]]},
   "maneuver":{"type":"rotary","modifier":"right","exit":2}},
  {"name":"","rotary_name":"","distance":303,"duration":17,
   "geometry":{"coordinates":[[39.009462,35.954790],[39.012075,35.955995]]},
   "maneuver":{"type":"exit rotary","modifier":"straight","exit":2}},
  {"name":"","rotary_name":"","distance":0,"duration":0,
   "geometry":{"coordinates":[[39.012075,35.955995]]},
   "maneuver":{"type":"arrive","modifier":null,"exit":null}}
 ]}]}]}`

func serve(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
}

func routeOf(t *testing.T, body string) *Route {
	t.Helper()
	s := serve(body)
	t.Cleanup(s.Close)
	c := New(s.URL)
	r, err := c.Route(context.Background(),
		Point{Lat: 35.9506, Lng: 39.0094}, Point{Lat: 35.9560, Lng: 39.0120})
	if err != nil {
		t.Fatalf("التحويلُ فشل: %v", err)
	}
	return r
}

// ── طلبُ الخطوات ──────────────────────────────────────────────────────

// TestROUTE_001_AsksForSteps **و`steps=true` منذ المرحلة ٢.**
func TestROUTE_001_AsksForSteps(t *testing.T) {
	var got string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.RawQuery
		_, _ = w.Write([]byte(raqqaBody))
	}))
	defer s.Close()
	if _, err := New(s.URL).Route(context.Background(),
		Point{Lat: 1, Lng: 1}, Point{Lat: 2, Lng: 2}); err != nil {
		t.Fatalf("فشل: %v", err)
	}
	if !contains(got, "steps=true") {
		t.Errorf("ROUTE-001 **لم تُطلب الخطوات**: %s", got)
	}
	// **ولا `annotations`** — تضاعف الردَّ ولا تُستعمل.
	if contains(got, "annotations=true") {
		t.Errorf("ROUTE-001 طُلبت `annotations` بلا حاجة: %s", got)
	}
}

// ── العقد ────────────────────────────────────────────────────────────

// TestROUTE_010_ContractBuilt **والمناوراتُ تُبنى بمفاهيمنا.**
func TestROUTE_010_ContractBuilt(t *testing.T) {
	r := routeOf(t, raqqaBody)
	if !r.HasNavigation() {
		t.Fatal("ROUTE-010 **لا ملاحةَ في العقد**")
	}
	if len(r.Maneuvers) != 4 {
		t.Fatalf("ROUTE-010 مناورات=%d لا ٤", len(r.Maneuvers))
	}
	want := []ManeuverKind{KindDepart, KindRoundabout, KindExitRoundabout, KindArrive}
	for i, w := range want {
		if r.Maneuvers[i].Kind != w {
			t.Errorf("ROUTE-010 المناورة %d: %s لا %s", i, r.Maneuvers[i].Kind, w)
		}
	}
}

// TestROUTE_011_CumulativeMatchesGeometry **والتراكميّةُ بطول الخطّ.**
//
// **وطولان مختلفان يجعلان `atDistanceM` يشير إلى موضعٍ ليس عليه.**
func TestROUTE_011_CumulativeMatchesGeometry(t *testing.T) {
	r := routeOf(t, raqqaBody)
	if len(r.CumulativeM) != len(r.Geometry) {
		t.Fatalf("ROUTE-011 تراكميّة=%d وهندسة=%d", len(r.CumulativeM), len(r.Geometry))
	}
	if r.CumulativeM[0] != 0 {
		t.Errorf("ROUTE-011 أوّلُ تراكميّةٍ %v لا صفر", r.CumulativeM[0])
	}
	// **وتصاعديّةٌ أبداً** — ورأسٌ مكرّرٌ يجعلها تقف، وخللٌ يجعلها تنقص.
	for i := 1; i < len(r.CumulativeM); i++ {
		if r.CumulativeM[i] < r.CumulativeM[i-1] {
			t.Fatalf("ROUTE-011 **التراكميّةُ نقصت** عند %d", i)
		}
	}
	// **وتطابق المسافةَ المحسوبةَ بأنفسنا.**
	var sum float64
	for i := 1; i < len(r.Geometry); i++ {
		sum += MetersBetween(r.Geometry[i-1], r.Geometry[i])
	}
	if math.Abs(sum-r.CumulativeM[len(r.CumulativeM)-1]) > 0.5 {
		t.Errorf("ROUTE-011 الإجماليُّ %v والمحسوبُ %v",
			r.CumulativeM[len(r.CumulativeM)-1], sum)
	}
}

// TestROUTE_012_NoDuplicateVertexAtJoins **ولا رأسَ مكرَّرٌ في المفاصل.**
//
// **ومن ألصق الخطواتِ بلا حذفٍ** كرّر الرؤوسَ فزادت المسافةُ صفراً في
// كلّ مفصل — **ورقمٌ لا يتغيّر لا يُكتشف بالعين.**
func TestROUTE_012_NoDuplicateVertexAtJoins(t *testing.T) {
	r := routeOf(t, raqqaBody)
	for i := 1; i < len(r.Geometry); i++ {
		if samePoint(r.Geometry[i-1], r.Geometry[i]) {
			t.Fatalf("ROUTE-012 **رأسٌ مكرَّرٌ عند %d**", i)
		}
	}
}

// TestROUTE_013_AtDistanceIsMonotonic **ومواضعُ المناورات تتقدّم.**
func TestROUTE_013_AtDistanceIsMonotonic(t *testing.T) {
	r := routeOf(t, raqqaBody)
	for i := 1; i < len(r.Maneuvers); i++ {
		if r.Maneuvers[i].AtDistanceM < r.Maneuvers[i-1].AtDistanceM {
			t.Fatalf("ROUTE-013 المناورة %d قبل سابقتها", i)
		}
	}
	if r.Maneuvers[0].AtDistanceM != 0 {
		t.Errorf("ROUTE-013 الانطلاقُ عند %v لا صفر", r.Maneuvers[0].AtDistanceM)
	}
}

// ── الدوّارات ────────────────────────────────────────────────────────

// TestROUTE_020_RoundaboutKeepsExitAndName **ورقمُ المخرج والاسمُ يبقيان.**
func TestROUTE_020_RoundaboutKeepsExitAndName(t *testing.T) {
	r := routeOf(t, raqqaBody)
	m := r.Maneuvers[1]
	if m.Kind != KindRoundabout {
		t.Fatalf("ROUTE-020 %s لا دوّار", m.Kind)
	}
	if m.RoundaboutExit == nil || *m.RoundaboutExit != 2 {
		t.Errorf("ROUTE-020 **رقمُ المخرج ضاع**: %v", m.RoundaboutExit)
	}
	if m.RoundaboutName == nil || *m.RoundaboutName != "دوار النعيم" {
		t.Errorf("ROUTE-020 **اسمُ الدوّار ضاع**: %v", m.RoundaboutName)
	}
}

// TestROUTE_021_RotaryAndRoundaboutAreOne **والدوّاران واحدٌ عند السائق.**
func TestROUTE_021_RotaryAndRoundaboutAreOne(t *testing.T) {
	for _, tt := range []struct {
		osrm string
		want ManeuverKind
	}{
		{"roundabout", KindRoundabout},
		{"rotary", KindRoundabout},
		{"exit roundabout", KindExitRoundabout},
		{"exit rotary", KindExitRoundabout},
	} {
		if got := mapKind(tt.osrm, ptr("right")); got != tt.want {
			t.Errorf("ROUTE-021 %q → %s لا %s", tt.osrm, got, tt.want)
		}
	}
}

// ── الأسماءُ والمعدِّلاتُ الغائبة ────────────────────────────────────

// TestROUTE_030_MissingNameStaysMissing **والاسمُ الغائبُ لا يُخترع.**
//
// **وقِيس أنّ ٨٣٪ من خطوات الرقّة بلا اسم** — فهي الحالُ العاديّة.
func TestROUTE_030_MissingNameStaysMissing(t *testing.T) {
	r := routeOf(t, raqqaBody)
	if r.Maneuvers[0].StreetName == nil || *r.Maneuvers[0].StreetName != "Adnan Malki Street" {
		t.Errorf("ROUTE-030 اسمٌ موجودٌ ضاع: %v", r.Maneuvers[0].StreetName)
	}
	for _, i := range []int{1, 2, 3} {
		if r.Maneuvers[i].StreetName != nil {
			t.Errorf("ROUTE-030 **اخترع اسماً للمناورة %d**: %v", i, *r.Maneuvers[i].StreetName)
		}
	}
}

// TestROUTE_031_NilModifierIsUnknownNotRight **ومعدِّلٌ غائبٌ لا يُخمَّن.**
//
// **قِيس أنّه `null` فعلاً** في `depart` بالأزقّة — **ومن خمّنه يميناً
// وجّه السائقَ إلى شارعٍ آخر.**
func TestROUTE_031_NilModifierIsUnknownNotRight(t *testing.T) {
	if got := mapKind("turn", nil); got != KindUnknown {
		t.Errorf("ROUTE-031 معدِّلٌ غائبٌ صار %s", got)
	}
	// **والانطلاقُ والوصولُ لا يحتاجان معدِّلاً.**
	if got := mapKind("depart", nil); got != KindDepart {
		t.Errorf("ROUTE-031 الانطلاقُ صار %s", got)
	}
	if got := mapKind("arrive", nil); got != KindArrive {
		t.Errorf("ROUTE-031 الوصولُ صار %s", got)
	}
}

// TestROUTE_032_UnknownTypeIsSafe **ونوعٌ مجهولٌ لا يُسقط شيئاً.**
//
// **وOSRM يضيف أنواعاً في نسخه** — والتطبيقُ في يد سائقٍ على درّاجة.
func TestROUTE_032_UnknownTypeIsSafe(t *testing.T) {
	for _, weird := range []string{"teleport", "", "exit ferry", "notify"} {
		if got := mapKind(weird, ptr("left")); got != KindUnknown {
			t.Errorf("ROUTE-032 %q صار %s لا UNKNOWN", weird, got)
		}
	}
}

// TestROUTE_033_AllMeasuredTypesMap **وكلُّ ما قِيس من محرّكنا يُحوَّل.**
func TestROUTE_033_AllMeasuredTypesMap(t *testing.T) {
	measured := map[string]ManeuverKind{
		"depart": KindDepart, "arrive": KindArrive,
		"continue": KindStraight, "new name": KindStraight,
		"merge": KindMerge, "fork": KindFork, "off ramp": KindOffRamp,
		"roundabout": KindRoundabout, "exit roundabout": KindExitRoundabout,
		"rotary": KindRoundabout, "exit rotary": KindExitRoundabout,
	}
	for typ, want := range measured {
		if got := mapKind(typ, ptr("right")); got != want {
			t.Errorf("ROUTE-033 %q → %s لا %s", typ, got, want)
		}
	}
	mods := map[string]ManeuverKind{
		"straight": KindStraight, "left": KindTurnLeft, "right": KindTurnRight,
		"slight left": KindSlightLeft, "slight right": KindSlightRight,
		"sharp left": KindSharpLeft, "sharp right": KindSharpRight,
		"uturn": KindUTurn,
	}
	for mod, want := range mods {
		if got := mapKind("turn", ptr(mod)); got != want {
			t.Errorf("ROUTE-033 turn/%q → %s لا %s", mod, got, want)
		}
	}
}

// ── التوافقُ الخلفيّ ─────────────────────────────────────────────────

// TestROUTE_040_NoStepsStillDraws **ومحرّكٌ بلا خطواتٍ يُرسم ولا يُرشد.**
//
// **ولا يسقط شيء** — والشاشةُ ترسم كما كانت.
func TestROUTE_040_NoStepsStillDraws(t *testing.T) {
	body := `{"code":"Ok","routes":[{"distance":900,"duration":80,
	 "geometry":{"coordinates":[[39.0094,35.9506],[39.0120,35.9560]]},"legs":[]}]}`
	r := routeOf(t, body)
	if len(r.Geometry) != 2 {
		t.Fatalf("ROUTE-040 الهندسةُ %d لا ٢", len(r.Geometry))
	}
	if r.HasNavigation() {
		t.Error("ROUTE-040 **ادّعى ملاحةً بلا خطوات**")
	}
	if r.DistanceM != 900 {
		t.Errorf("ROUTE-040 المسافةُ %v", r.DistanceM)
	}
}

// TestROUTE_041_EmptyStepGeometryDoesNotBreak **وخطوةٌ بلا هندسةٍ لا تكسر.**
func TestROUTE_041_EmptyStepGeometryDoesNotBreak(t *testing.T) {
	body := `{"code":"Ok","routes":[{"distance":10,"duration":2,
	 "geometry":{"coordinates":[[39.0094,35.9506],[39.0120,35.9560]]},
	 "legs":[{"steps":[{"name":"","rotary_name":"","distance":0,"duration":0,
	   "geometry":{"coordinates":[]},"maneuver":{"type":"depart","modifier":null}}]}]}]}`
	r := routeOf(t, body)
	if len(r.Geometry) < 2 {
		t.Fatal("ROUTE-041 ضاعت الهندسةُ الاحتياطيّة")
	}
}

// TestROUTE_042_ContractSerializesWithoutNavWhenAbsent **والحقولُ تُحذف.**
//
// **وحقلٌ فارغٌ يُرسَل في كلّ نداءٍ يكبّر الردَّ بلا معنى.**
func TestROUTE_042_ContractSerializesWithoutNavWhenAbsent(t *testing.T) {
	raw, err := json.Marshal(&Route{DistanceM: 1, DurationS: 1})
	if err != nil {
		t.Fatal(err)
	}
	if contains(string(raw), "Maneuvers") || contains(string(raw), "CumulativeM") {
		t.Errorf("ROUTE-042 حقولٌ فارغةٌ تُرسَل: %s", raw)
	}
}

func ptr(s string) *string { return &s }

func contains(h, n string) bool {
	return len(h) >= len(n) && (func() bool {
		for i := 0; i+len(n) <= len(h); i++ {
			if h[i:i+len(n)] == n {
				return true
			}
		}
		return false
	})()
}
