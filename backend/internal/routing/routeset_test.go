package routing

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **رفائدُ هندسيّةٌ مصنوعة**
// ══════════════════════════════════════════════════════════════════════

// **مترٌ في خطّ العرض ≈ 0.000009°، وفي خطّ الطول ≈ 0.0000111° عند
// خطّ عرض الرقّة.** — والرفائدُ تُبنى بهما فتكون المسافاتُ معلومة.
const (
	lat0 = 35.9500
	lng0 = 39.0050
)

func north(m float64) float64 { return lat0 + m*0.000009 }
func east(m float64) float64  { return lng0 + m*0.0000111 }

// straightEast **خطٌّ مستقيمٌ شرقاً** — نقطةٌ كلَّ `stepM`.
func straightEast(lengthM, stepM float64) []Point {
	var out []Point
	for d := 0.0; d <= lengthM; d += stepM {
		out = append(out, Point{Lat: lat0, Lng: east(d)})
	}
	return out
}

// offsetNorth **المسارُ نفسُه مُزاحاً شمالاً.**
func offsetNorth(g []Point, m float64) []Point {
	out := make([]Point, len(g))
	for i, p := range g {
		out[i] = Point{Lat: p.Lat + m*0.000009, Lng: p.Lng}
	}
	return out
}

// detour **يتفرّع عند `atM` ثمّ يعود عند `rejoinM`.**
func detour(lengthM, atM, rejoinM, offsetM float64) []Point {
	var out []Point
	for d := 0.0; d <= lengthM; d += 20 {
		lat := lat0
		if d > atM && d < rejoinM {
			lat = north(offsetM)
		}
		out = append(out, Point{Lat: lat, Lng: east(d)})
	}
	return out
}

func route(g []Point, distanceM, durationS float64) *Route {
	cum := make([]float64, len(g))
	for i := 1; i < len(g); i++ {
		cum[i] = cum[i-1] + MetersBetween(g[i-1], g[i])
	}
	return &Route{
		DistanceM:   distanceM,
		DurationS:   durationS,
		Geometry:    g,
		CumulativeM: cum,
		Maneuvers:   []Maneuver{{Kind: KindArrive}},
		WeightName:  "routability",
	}
}

// ══════════════════════════════════════════════════════════════════════
// **التشابه — البند ٩**
// ══════════════════════════════════════════════════════════════════════

func TestSimilarityIgnoresSampling(t *testing.T) {
	// **أمرُ المالك**: «Route A وB نفس الطريق تقريبًا مع Vertices
	// مختلفة → similarity عالية».
	dense := straightEast(2000, 10)
	sparse := straightEast(2000, 100)
	veryS := straightEast(2000, 400)

	for _, tc := range []struct {
		name string
		g    []Point
	}{
		{"نقطةٌ كلَّ ١٠٠م", sparse},
		{"نقطةٌ كلَّ ٤٠٠م", veryS},
	} {
		got := SharedRatio(dense, tc.g)
		if got < 0.95 {
			t.Errorf("%s: التشابهُ %.3f — والتقطيعُ لا يجوز أن يؤثّر", tc.name, got)
		}
	}
}

func TestSimilaritySelfIsOne(t *testing.T) {
	g := straightEast(3000, 25)
	if got := SharedRatio(g, g); got < 0.99 {
		t.Errorf("المسارُ مع نفسِه %.3f", got)
	}
}

func TestSimilarityParallelRoad(t *testing.T) {
	// **طريقٌ موازٍ** — داخلَ التسامح يُعدُّ واحداً، وخارجَه لا.
	g := straightEast(2000, 25)
	if got := SharedRatio(g, offsetNorth(g, 20)); got < 0.9 {
		t.Errorf("إزاحةُ ٢٠م داخلَ التسامح: %.3f", got)
	}
	if got := SharedRatio(g, offsetNorth(g, 60)); got > 0.2 {
		t.Errorf("إزاحةُ ٦٠م خارجَ التسامح: %.3f — ولا يجوز أن تُعدَّ واحداً", got)
	}
}

func TestSimilarityDifferentRoutes(t *testing.T) {
	a := straightEast(2000, 25)
	b := make([]Point, 0, 80)
	for d := 0.0; d <= 2000; d += 25 {
		b = append(b, Point{Lat: north(d), Lng: lng0})
	}
	if got := SharedRatio(a, b); got > 0.1 {
		t.Errorf("مساران متعامدان: %.3f", got)
	}
}

func TestSimilarityIsNotFingerprint(t *testing.T) {
	// **البند ٩**: «أبق IDENTITY/FINGERPRINT مختلفة تمامًا عن
	// DIVERSITY/SIMILARITY».
	//
	// **مساران بالهندسة نفسِها وتقطيعٍ مختلف**: التشابهُ عالٍ
	// **والمعرِّفُ مختلف** — وذلك صحيح: الهويّةُ تطابقٌ والتنوّعُ تدرّج.
	dense := route(straightEast(2000, 10), 2000, 200)
	sparse := route(straightEast(2000, 100), 2000, 200)
	if SharedRatio(dense.Geometry, sparse.Geometry) < 0.95 {
		t.Fatal("التشابهُ يجب أن يكون عالياً")
	}
	if RouteID(dense, "o1", "pickup") == RouteID(sparse, "o1", "pickup") {
		t.Error("المعرِّفُ يجب أن يفرّق بين هندستين مختلفتَي الرؤوس")
	}
}

func TestFirstDivergence(t *testing.T) {
	base := straightEast(3000, 20)
	alt := detour(3000, 500, 1500, 200)

	// **والقياسُ على المسار الموصى به** — فهو الذي يقوده السائق.
	// **ولو قِيس على البديل لزاد بقدر انحرافه** (٧٠١م بدل ٥٢٠).
	d := FirstDivergenceM(base, alt)
	if d < 480 || d > 620 {
		t.Errorf("نقطةُ التفرّع %.0fم — والمنتظَرُ نحو ٥٢٠", d)
	}
	if got := FirstDivergenceM(base, base); got >= 0 {
		t.Errorf("مسارٌ مع نفسِه لا يتفرّع: %.0f", got)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **الهويّةُ الثابتة — البند ١٤**
// ══════════════════════════════════════════════════════════════════════

func TestRouteIDStable(t *testing.T) {
	r := route(straightEast(2000, 25), 2000, 200)
	a := RouteID(r, "order-1", "pickup")
	b := RouteID(r, "order-1", "pickup")
	if a != b || a == "" {
		t.Fatalf("المعرِّفُ غيرُ ثابت: %q ثمّ %q", a, b)
	}
}

func TestRouteIDSeparatesTarget(t *testing.T) {
	// **البند ٢١** — بديلٌ إلى الاستلام لا يصلح بعد `picked_up`.
	r := route(straightEast(2000, 25), 2000, 200)
	if RouteID(r, "o1", "pickup") == RouteID(r, "o1", "dropoff") {
		t.Error("الوجهةُ يجب أن تدخل المعرِّف")
	}
	if RouteID(r, "o1", "pickup") == RouteID(r, "o2", "pickup") {
		t.Error("الطلبُ يجب أن يدخل المعرِّف")
	}
}

func TestRouteIDIgnoresGeneration(t *testing.T) {
	// **أمرُ المالك**: «routeGeneration لا تدخل في routeId».
	//
	// **ولا سبيلَ لتمريره أصلاً** — التوقيعُ لا يقبله، وهذا هو الحفظ.
	r := route(straightEast(2000, 25), 2000, 200)
	if len(RouteID(r, "o1", "pickup")) != 16 {
		t.Error("طولُ المعرِّف ١٦")
	}
}

func TestRouteIDDiffersByGeometry(t *testing.T) {
	a := route(straightEast(2000, 25), 2000, 200)
	b := route(detour(2000, 500, 1500, 200), 2100, 220)
	if RouteID(a, "o", "pickup") == RouteID(b, "o", "pickup") {
		t.Error("هندستان مختلفتان ومعرِّفٌ واحد")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **الترشيح — البنود ٧ و٨ و١٠ و١٢**
// ══════════════════════════════════════════════════════════════════════

func TestFilterDropsNearDuplicate(t *testing.T) {
	base := route(straightEast(3000, 20), 3000, 300)
	// **يتفرّع خمسين متراً ثمّ يعود** — تشابهٌ عالٍ.
	dup := route(detour(3000, 1000, 1050, 15), 3010, 305)
	kept, dropped := FilterAlternatives(base, []*Route{dup})
	if len(kept) != 0 {
		t.Errorf("شبهُ المكرَّر يجب أن يُحذف — نجا %d", len(kept))
	}
	if len(dropped) != 1 || dropped[0] != string(ReasonNearDuplicate) {
		t.Errorf("سببُ الحذف: %v", dropped)
	}
}

func TestFilterKeepsDominatedButDiverse(t *testing.T) {
	// ══════════════════════════════════════════════════════════════
	// **البندان ٧ و١١ — والهيمنةُ ليست سببَ حذف**
	// ══════════════════════════════════════════════════════════════
	//
	// **أمرُ المالك نصّاً**: «المسافة والزمن لا يمثلان كل معرفة
	// الطريق، والسائق قد يستفيد من Route مختلفة بوضوح إذا كان يعرف
	// إغلاقًا أو مشكلةً محليةً غير موجودة في البيانات».
	//
	// **والقياسُ يسنده**: ٤٤ من ٥٦ بديلاً مهيمَناً عليه **تشابهُها
	// دونَ ٥٠٪** — طرقٌ مختلفةٌ حقّاً.
	base := route(straightEast(3000, 20), 3000, 300)
	worse := route(detour(3000, 200, 2800, 400), 3400, 360)

	if !(worse.DistanceM >= base.DistanceM && worse.DurationS >= base.DurationS) {
		t.Fatal("الرفيدةُ يجب أن تكون مهيمَناً عليها")
	}
	kept, _ := FilterAlternatives(base, []*Route{worse})
	if len(kept) != 1 {
		t.Error("مهيمَنٌ عليه ومتنوّعٌ هندسيّاً يجب أن يبقى")
	}
}

func TestFilterDropsGrossPenalty(t *testing.T) {
	base := route(straightEast(3000, 20), 3000, 300)
	// **زمنٌ فاحش** — ضِعفُ الأساس.
	slow := route(detour(3000, 200, 2800, 400), 3100, 700)
	kept, dropped := FilterAlternatives(base, []*Route{slow})
	if len(kept) != 0 {
		t.Error("عقوبةُ زمنٍ فاحشةٌ يجب أن تُحذف")
	}
	if len(dropped) != 1 || dropped[0] != string(ReasonGrossTime) {
		t.Errorf("سببٌ غيرُ متوقَّع: %v", dropped)
	}

	long := route(detour(3000, 200, 2800, 400), 6000, 320)
	kept, dropped = FilterAlternatives(base, []*Route{long})
	if len(kept) != 0 || dropped[0] != string(ReasonGrossDistance) {
		t.Errorf("عقوبةُ مسافةٍ فاحشة: نجا %d سبب %v", len(kept), dropped)
	}
}

func TestFilterCapsAtTwo(t *testing.T) {
	// **البند ٢٢** — سقفُ ثلاثةِ مساراتٍ إجمالاً.
	base := route(straightEast(4000, 20), 4000, 400)
	alts := []*Route{
		route(detour(4000, 200, 3800, 300), 4200, 430),
		route(detour(4000, 300, 3700, 600), 4300, 440),
		route(detour(4000, 400, 3600, 900), 4400, 450),
	}
	kept, dropped := FilterAlternatives(base, alts)
	if len(kept) != MaxAlternatives {
		t.Errorf("السقف %d ونجا %d", MaxAlternatives, len(kept))
	}
	if len(dropped) == 0 || dropped[len(dropped)-1] != string(ReasonOverCap) {
		t.Errorf("سببُ الحذف الأخير: %v", dropped)
	}
}

func TestFilterDropsDuplicateAmongAlternatives(t *testing.T) {
	// **بديلان متشابهان بينهما** — لا يُعرضان معاً ولو اختلفا عن الأساس.
	base := route(straightEast(4000, 20), 4000, 400)
	a := route(detour(4000, 500, 3500, 400), 4200, 430)
	b := route(detour(4000, 520, 3480, 405), 4205, 432)
	kept, _ := FilterAlternatives(base, []*Route{a, b})
	if len(kept) != 1 {
		t.Errorf("بديلان متشابهان — نجا %d", len(kept))
	}
}

func TestFilterDropsInvalid(t *testing.T) {
	base := route(straightEast(2000, 25), 2000, 200)
	for _, bad := range []*Route{
		nil,
		{DistanceM: 0, DurationS: 100, Geometry: straightEast(2000, 25)},
		{DistanceM: 2000, DurationS: 0, Geometry: straightEast(2000, 25)},
		{DistanceM: 2000, DurationS: 200, Geometry: []Point{{Lat: lat0, Lng: lng0}}},
	} {
		kept, _ := FilterAlternatives(base, []*Route{bad})
		if len(kept) != 0 {
			t.Errorf("بديلٌ فاسدٌ نجا: %+v", bad)
		}
	}
}

func TestFilterBadAlternativeDoesNotKillGoodOne(t *testing.T) {
	// **البند ٣٣ من التحليل**: «ولا Alternative فاسدة تسقط RouteSet
	// كلها إذا Primary سليمة».
	base := route(straightEast(4000, 20), 4000, 400)
	good := route(detour(4000, 500, 3500, 400), 4200, 430)
	kept, _ := FilterAlternatives(base, []*Route{nil, good})
	if len(kept) != 1 {
		t.Errorf("السليمُ يجب أن ينجو رغم الفاسد — نجا %d", len(kept))
	}
}

func TestFilterEmptyIsNotFailure(t *testing.T) {
	// **البند ٣٤ من التحليل** — «primary + alternatives=[] طبيعي».
	base := route(straightEast(2000, 25), 2000, 200)
	kept, dropped := FilterAlternatives(base, nil)
	if len(kept) != 0 || len(dropped) != 0 {
		t.Errorf("لا بدائلَ: نجا %d حُذف %d", len(kept), len(dropped))
	}
}

// ══════════════════════════════════════════════════════════════════════
// **الأساسُ لا يُرقّى — البندان ٢ و٣٢**
// ══════════════════════════════════════════════════════════════════════

func TestPrimaryStaysRecommendedEvenWhenOutranked(t *testing.T) {
	// ══════════════════════════════════════════════════════════════
	// **الحالتان المقيستان — ولا يُبدَّل الأساس**
	// ══════════════════════════════════════════════════════════════
	//
	// **أمرُ المالك نصّاً**: «حتى لو routes[1] كانت أقصر + أقل
	// duration، لا ترقيها تلقائيًا إلى Primary».
	//
	// **ووقع فعلاً مرّتين في ١٣٤ طلباً** (٢٠٢٦-٠٨-٢١):
	//
	//	الأساس ٣٩٨٫٣كم/٣٠٦٫٤د  →  البديل ٣٦١٫٣كم/٢٩٩٫١د
	//	الأساس ٤٨٩٫٥كم/٣٧٤٫٨د  →  البديل ٤٥٣٫٨كم/٣٦٣٫٥د
	//
	// **وهذا الاختبارُ يمنع مطوّراً قادماً** من أن يرتّب بالمدّة
	// ويظنّه تحسيناً.
	base := route(straightEast(4000, 20), 398340, 18384)
	better := route(detour(4000, 500, 3500, 400), 361340, 17946)

	if !PrimaryOutranked(base, []*Route{better}) {
		t.Fatal("الحالةُ يجب أن تُرصد")
	}

	kept, _ := FilterAlternatives(base, []*Route{better})
	if len(kept) != 1 {
		t.Fatalf("البديلُ يجب أن ينجو — نجا %d", len(kept))
	}
	// **والترتيبُ لم يُقلب** — `FilterAlternatives` لا تردّ أساساً،
	// **فلا سبيلَ لترقيةٍ من هنا أصلاً.** وذلك هو الحفظُ البنيويّ.
	if kept[0] != better {
		t.Error("البديلُ تبدّل")
	}
}

func TestPrimaryOutrankedFalseWhenNormal(t *testing.T) {
	base := route(straightEast(3000, 20), 3000, 300)
	worse := route(detour(3000, 200, 2800, 400), 3400, 360)
	if PrimaryOutranked(base, []*Route{worse}) {
		t.Error("لا يجوز الرصدُ حين يكون الأساسُ أفضل")
	}
}

func TestFilterPreservesEngineOrder(t *testing.T) {
	// **البند ١٣** — «حافظ قدر الإمكان على OSRM recommendation order».
	base := route(straightEast(5000, 20), 5000, 500)
	first := route(detour(5000, 400, 4600, 300), 5600, 560)
	second := route(detour(5000, 600, 4400, 900), 5100, 505)

	kept, _ := FilterAlternatives(base, []*Route{first, second})
	if len(kept) != 2 {
		t.Fatalf("نجا %d", len(kept))
	}
	// **والثاني أفضلُ في المحورين** — ولا يُقدَّم.
	if kept[0] != first {
		t.Error("الترتيبُ أُعيد بالمدّة أو بالمسافة — والمطلوبُ ترتيبُ المحرّك")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **الرفيدةُ الذهبيّة — البند ٣٠**
// ══════════════════════════════════════════════════════════════════════

type goldenAlt struct {
	Area      string  `json:"area"`
	PrimM     float64 `json:"prim_m"`
	PrimS     float64 `json:"prim_s"`
	M         float64 `json:"m"`
	S         float64 `json:"s"`
	DM        float64 `json:"dm"`
	DS        float64 `json:"ds"`
	Shared    float64 `json:"shared"`
	Dominated bool    `json:"dominated"`
}

func loadGolden(t *testing.T) []goldenAlt {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "syria-alternatives.golden.json"))
	if err != nil {
		t.Fatalf("الرفيدةُ الذهبيّة: %v", err)
	}
	var body struct {
		Alts []goldenAlt `json:"alts"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("الرفيدةُ الذهبيّة: %v", err)
	}
	return body.Alts
}

// TestGoldenDatasetShape **الرفيدةُ كما قِيست** — فلا تشيخ صامتةً.
func TestGoldenDatasetShape(t *testing.T) {
	alts := loadGolden(t)
	if len(alts) != 87 {
		t.Fatalf("الرفيدةُ %d بديلاً والمقيسُ ٨٧", len(alts))
	}
	dominated := 0
	dominatedDiverse := 0
	nearDup := 0
	shorterSlower := 0
	fasterLonger := 0
	betterBoth := 0
	for _, a := range alts {
		if a.Dominated {
			dominated++
			if a.Shared < 0.50 {
				dominatedDiverse++
			}
		}
		if a.Shared >= NearDuplicateShared {
			nearDup++
		}
		switch {
		case a.DM < 0 && a.DS > 0:
			shorterSlower++
		case a.DS < 0 && a.DM > 0:
			fasterLonger++
		case a.DM < 0 && a.DS < 0:
			betterBoth++
		}
	}
	if dominated != 56 {
		t.Errorf("المهيمَنُ عليها %d والمقيسُ ٥٦", dominated)
	}
	if nearDup != 2 {
		t.Errorf("شبهُ المكرَّر %d والمقيسُ ٢", nearDup)
	}
	if betterBoth != 2 {
		t.Errorf("أفضلُ في المحورين %d والمقيسُ ٢", betterBoth)
	}
	if fasterLonger != 0 {
		t.Errorf("«أسرعُ وأطول» %d والمقيسُ ٠ — فلا وسمَ لها", fasterLonger)
	}
	t.Logf("الرفيدة: %d بديلاً · مهيمَنٌ عليها %d (منها %d متنوّعة) · "+
		"شبهُ مكرَّر %d · أقصرُ وأبطأ %d · أفضلُ في المحورين %d",
		len(alts), dominated, dominatedDiverse, nearDup, shorterSlower, betterBoth)
}

// TestPolicyCalibration **السياسةُ المعتمدةُ على البيانات نفسِها.**
//
// **البند ١٢**: «اختَر أبسط Policy تمنع Routes العبثية ولا تهبط
// التغطية بلا داعٍ». **وهذا الاختبارُ يُثبت أنّ ما اعتُمد ما زال يفعل
// ما قِيس.**
func TestPolicyCalibration(t *testing.T) {
	alts := loadGolden(t)

	keep := func(a goldenAlt) bool {
		if a.Shared >= NearDuplicateShared {
			return false
		}
		if a.PrimS > 0 && a.DS/a.PrimS > GrossPenaltyRatio {
			return false
		}
		return !(a.PrimM > 0 && a.DM/a.PrimM > GrossPenaltyRatio)
	}

	kept := 0
	maxTime, maxDist := 0.0, 0.0
	for _, a := range alts {
		if !keep(a) {
			continue
		}
		kept++
		maxTime = math.Max(maxTime, 100*a.DS/a.PrimS)
		maxDist = math.Max(maxDist, 100*a.DM/a.PrimM)
	}

	// **قِيس ٨٤ ناجياً من ٨٧** — تُحذف اثنتان شبهَ مكرَّرٍ وواحدةٌ
	// بعقوبة مسافةٍ +٥٢٪.
	if kept != 84 {
		t.Errorf("نجا %d والمعايَرُ ٨٤", kept)
	}
	if maxTime > 100*GrossPenaltyRatio {
		t.Errorf("أقصى عقوبةِ زمنٍ %.1f%% تجاوزت الحدّ", maxTime)
	}
	if maxDist > 100*GrossPenaltyRatio {
		t.Errorf("أقصى عقوبةِ مسافةٍ %.1f%% تجاوزت الحدّ", maxDist)
	}
	t.Logf("السياسةُ «ج»: نجا %d من %d · أقصى عقوبةِ زمن %.1f%% · "+
		"أقصى عقوبةِ مسافة %.1f%%", kept, len(alts), maxTime, maxDist)
}

// TestDominatedNotHardRejected **الرفضُ الصريحُ لحذف المهيمَن.**
func TestDominatedNotHardRejected(t *testing.T) {
	alts := loadGolden(t)
	diverse := 0
	for _, a := range alts {
		if a.Dominated && a.Shared < 0.50 {
			diverse++
		}
	}
	if diverse < 40 {
		t.Fatalf("مهيمَنٌ عليه ومتنوّع: %d — والمقيسُ ٤٤", diverse)
	}
	t.Logf("**حذفُ المهيمَنِ عليه كان سيرمي %d بديلاً مختلفَ الطريق** "+
		"— ولذلك رُفض (البند ٧)", diverse)
}

// ══════════════════════════════════════════════════════════════════════
// **رؤوسٌ متباعدة — العيبُ الذي أُصلح في إغلاق الصحّة**
// ══════════════════════════════════════════════════════════════════════

// sparseHighway **قطعٌ طويلةٌ بلا رؤوسَ بينها** — كطريقٍ سريع.
//
// **رأسٌ كلَّ ثلاثةِ كيلومترات**، والخليّةُ `0.005°` ≈ ٥٥٠م. **فبين
// الرأسين عشرُ خلايا لا رأسَ فيها.**
func sparseHighway(lengthM, stepM float64) []Point {
	var out []Point
	for d := 0.0; d <= lengthM; d += stepM {
		out = append(out, Point{Lat: lat0, Lng: east(d)})
	}
	return out
}

// TestSimilaritySparseVertices **مسارٌ مع نفسِه ورؤوسُه متباعدة.**
//
// **قِيس على المحرّك الحقيقيّ** (٢٠٢٦-٠٨-٢١): مساران متطابقان
// بايتاً ببايت من الرقّة إلى دمشق **أعطيا ٠٫٩٠٦ و٠٫٨٤٩** — **وكانت
// الشبكةُ تسجّل القطعةَ عند طرفيها فقط**، فتسقط الخلايا الوسطى.
func TestSimilaritySparseVertices(t *testing.T) {
	for _, step := range []float64{600, 1500, 3000, 8000} {
		g := sparseHighway(60000, step)
		if got := SharedRatio(g, g); got < 0.99 {
			t.Errorf("رأسٌ كلَّ %.0fم: المسارُ مع نفسِه %.4f — والخلايا الوسطى تسقط",
				step, got)
		}
	}
}

// TestSimilaritySparseVsDense **الطريقُ نفسُه بتقطيعين متباعدين.**
func TestSimilaritySparseVsDense(t *testing.T) {
	dense := sparseHighway(60000, 20)
	for _, step := range []float64{1500, 5000} {
		sparse := sparseHighway(60000, step)
		if got := SharedRatio(dense, sparse); got < 0.98 {
			t.Errorf("كثيفٌ مقابل رأسٍ كلَّ %.0fم: %.4f", step, got)
		}
	}
}

// TestFirstDivergenceSparseNoFalseEarly **ولا تفرّعَ كاذبٌ مبكّر.**
//
// **وهذا أثرُ العيب الأخطر**: نقطةُ تفرّعٍ كاذبةٌ **تُبطل بديلاً ما
// زال صالحاً** — والسائقُ يفقد خياراً بلا سبب.
func TestFirstDivergenceSparseNoFalseEarly(t *testing.T) {
	g := sparseHighway(60000, 3000)
	if got := FirstDivergenceM(g, g); got >= 0 {
		t.Errorf("مسارٌ مع نفسِه أعلن تفرّعاً عند %.0fم", got)
	}
}
