package maplearn

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestExtractAreas_HoodsAndVariants(t *testing.T) {
	cases := []struct {
		text string
		kind string
		norm string // أوّل كلمتين موحَّدتين
	}{
		{"حي الدرعيه، جانب الجامع", AreaHood, "درعيه"},
		{"في حي الدرعية بناء 5", AreaHood, "درعيه"},
		{"حيّ الدرعيّة", AreaHood, "درعيه"},
		{"بحي الدرعيه", AreaHood, "درعيه"},
		{"الحي: الدرعية", AreaHood, "درعيه"},
		{"حارة الجزرة قرب الفرن", AreaHood, "جزره"},
		{"منطقة المشلب الشقة 3", AreaHood, "مشلب"},
		{"حي الرقة القديمة شارع الوادي", AreaHood, "رقه قديمه"},
		{"شارع الوادي مقابل صيدلية الشفاء", AreaStreet, "وادي"},
		{"طريق الرصافة", AreaStreet, "رصافه"},
		{"بالشارع الوادي", AreaStreet, "وادي"},
		{"اوتوستراد حلب", AreaStreet, "حلب"},
		{"أوتوستراد الرقة", AreaStreet, "رقه"},
	}
	for _, c := range cases {
		hits := ExtractAreas(c.text)
		if len(hits) == 0 {
			t.Errorf("%q: لا نتيجة", c.text)
			continue
		}
		var got *AreaHit
		for i := range hits {
			if hits[i].Kind == c.kind {
				got = &hits[i]
				break
			}
		}
		if got == nil {
			t.Errorf("%q: لا %s في %+v", c.text, c.kind, hits)
			continue
		}
		if k := got.prefixKey(2); k != c.norm && got.prefixKey(1) != c.norm {
			t.Errorf("%q: المفتاح %q والمتوقَّع %q", c.text, k, c.norm)
		}
	}
	for _, text := range []string{
		"الحي الرئيسي", "الشارع الرئيسي", "شارع 8", "حي", "جانب صيدلية الشفاء", "شارع قرب الجامع", "حي الطابق الثاني",
		"", "منطقة 5",
	} {
		if h := ExtractAreas(text); len(h) != 0 {
			t.Errorf("%q: لا يجب أن يُستخرج شيء، وُجد %+v", text, h)
		}
	}
}

func TestExtractAreas_StopsAtBoundaries(t *testing.T) {
	h := ExtractAreas("حي الدرعية شارع الوادي مقابل الفرن")
	if len(h) != 2 || h[0].Kind != AreaHood || h[1].Kind != AreaStreet {
		t.Fatalf("متوقَّع حيّ ثم شارع: %+v", h)
	}
	if len(h[0].Norm) != 1 || len(h[1].Norm) != 1 {
		t.Errorf("الاسمان لا يمتدّان خلف الفاصل: %+v", h)
	}
	if d := h[0].Disp[0]; d != "الدرعية" {
		t.Errorf("الاسم المعروض %q", d)
	}
}

func TestWithinOneEditAndMerge(t *testing.T) {
	if !withinOneEdit("درعيه", "درعيا") || !withinOneEdit("مشلب", "مشلبب") || withinOneEdit("درعيه", "جزره") {
		t.Fatal("فرق الحرف الواحد")
	}
	m := mergeNearKeys(map[string]int{"درعيه": 5, "درعيا": 1, "جزره": 3, "ورد": 1, "ورق": 1})
	if m["درعيا"] != "درعيه" || m["جزره"] != "جزره" {
		t.Errorf("دمج الأخطاء الإملائية: %+v", m)
	}
	if m["ورق"] == m["ورد"] {
		t.Error("كلمات أقصر من ٥ حروف لا تُدمج")
	}
}

func ev(cust, text string, lat, lng float64) AddrEvidence {
	return AddrEvidence{Customer: cust, Text: text, Lat: lat, Lng: lng}
}

func TestClusterHoods_ThreeCustomersAndVariants(t *testing.T) {
	// ٤ زبائن يذكرون «الدرعية» بصيغٍ مختلفة قرب نقطةٍ واحدة (+ ملاحظةٌ شاذّةٌ بعيدة).
	e := []AddrEvidence{
		ev("a", "حي الدرعية، جوال 0933123456", 35.9500, 39.0000),
		ev("b", "حي الدرعيه قرب الفرن", 35.9510, 39.0010),
		ev("c", "بحي الدرعيّة", 35.9490, 39.0005),
		ev("d", "الحي الدرعيا", 35.9505, 38.9995), // خطأ إملائيّ
		ev("a", "حي الدرعية مرة ثانية", 35.9501, 39.0001),
		// حيٌّ بزبونَين فقط → لا
		ev("e", "حارة الجزرة", 35.9700, 39.0100),
		ev("f", "حارة الجزرة", 35.9701, 39.0101),
		// الاسم نفسه في مكانٍ آخر بعيد بثلاثة زبائن → حيٌّ ثانٍ
		ev("g", "حي الدرعية", 35.8000, 38.8000),
		ev("h", "حي الدرعية", 35.8005, 38.8002),
		ev("i", "حي الدرعية", 35.8002, 38.8004),
	}
	cs := ClusterHoods(e, DefaultAreaParams())
	if len(cs) != 2 {
		t.Fatalf("المتوقَّع حيّان، والناتج %+v", cs)
	}
	var big, far *HoodCluster
	for i := range cs {
		if cs[i].Lat > 35.9 {
			big = &cs[i]
		} else {
			far = &cs[i]
		}
	}
	if big == nil || far == nil {
		t.Fatalf("الموضعان: %+v", cs)
	}
	if big.Customers != 4 || big.Evidence != 5 || big.Name != "الدرعية" || big.Key != "درعيه" {
		t.Errorf("الحيّ الأول: %+v", big)
	}
	if big.Lat < 35.949 || big.Lat > 35.9511 || big.RadiusM <= 0 {
		t.Errorf("نقطة التسمية: %+v", big)
	}
	if len(big.Hull) < 4 {
		t.Errorf("الغلاف: %+v", big.Hull)
	}
	if far.Customers != 3 || len(far.Hull) < 4 {
		t.Errorf("الحيّ الثاني: %+v", far)
	}
	for _, s := range big.Samples {
		if strings.Contains(s, "0933123456") {
			t.Errorf("هاتفٌ في الدليل: %q", s)
		}
	}
}

func TestClusterHoods_LongerNameWhenMajorityAgrees(t *testing.T) {
	var e []AddrEvidence
	for i, c := range []string{"a", "b", "c", "d"} {
		e = append(e, ev(c, "حي الرقة القديمة", 35.95+float64(i)*0.0003, 39.0))
	}
	e = append(e, ev("e", "حي الرقة", 35.9502, 39.0001))
	cs := ClusterHoods(e, DefaultAreaParams())
	if len(cs) != 1 || cs[0].Name != "الرقة القديمة" {
		t.Fatalf("الاسم الأطول بالأغلبية: %+v", cs)
	}
	// وبلا أغلبيةٍ يبقى الاسمُ القصير.
	e = []AddrEvidence{
		ev("a", "حي الرقة القديمة", 35.95, 39.0), ev("b", "حي الرقة الجديدة", 35.9501, 39.0),
		ev("c", "حي الرقة", 35.9502, 39.0), ev("d", "حي الرقة", 35.9503, 39.0),
	}
	cs = ClusterHoods(e, DefaultAreaParams())
	if len(cs) != 1 || cs[0].Name != "الرقة" {
		t.Fatalf("الاسم القصير: %+v", cs)
	}
}

func TestConvexHullAndMedoid(t *testing.T) {
	h := convexHull([][2]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0.5, 0.5}})
	if len(h) != 5 || h[0] != h[len(h)-1] {
		t.Fatalf("الغلاف المغلق للمربع: %+v", h)
	}
	if convexHull([][2]float64{{0, 0}, {1, 1}, {2, 2}}) != nil {
		t.Error("نقاطٌ على استقامة: لا غلاف")
	}
	pts := []AddrEvidence{{Lat: 35, Lng: 39}, {Lat: 35.0001, Lng: 39}, {Lat: 35.0002, Lng: 39}, {Lat: 36, Lng: 39}}
	if m := medoid(pts); m != 1 && m != 2 {
		t.Errorf("الشاذّ لا يجرّ المركز: %d", m)
	}
}

// fakeLocator يلصق كلَّ نقطةٍ بأقرب شارعٍ من قائمة (خطوطٌ أفقيّة عند خطّ العرض).
type fakeLocator struct {
	ways  map[int64]float64 // رقم الطريق → خطُّ عرضه
	names map[int64][]string
	calls int
}

func (f *fakeLocator) Locate(_ context.Context, pts [][2]float64) ([]LocatedEdge, error) {
	f.calls++
	out := make([]LocatedEdge, len(pts))
	for i, p := range pts {
		best := LocatedEdge{}
		for w, lat := range f.ways {
			d := distM(p[0], p[1], lat, p[1])
			if best.WayID == 0 || d < best.DistM || (d == best.DistM && w < best.WayID) {
				best = LocatedEdge{WayID: w, DistM: d, Names: f.names[w],
					Line: [][2]float64{{p[1] - 0.001, lat}, {p[1] + 0.001, lat}}}
			}
		}
		out[i] = best
	}
	return out, nil
}

func TestGroupAndResolveStreets(t *testing.T) {
	e := []AddrEvidence{
		ev("a", "شارع الوادي مقابل الفرن", 35.9501, 39.00),
		ev("b", "شارع الوادي", 35.9502, 39.01),
		ev("c", "الشارع الوادي رقم 3", 35.9500, 39.02),
		ev("d", "شارع النهر", 35.9800, 39.00), // زبونٌ واحد → لا
		ev("e", "شارع الكورنيش", 35.9900, 39.00),
		ev("f", "شارع الكورنيش", 35.9902, 39.01),
		ev("g", "شارع الكورنيش", 35.9980, 39.02), // بعيدٌ على طريقٍ آخر
		ev("h", "شارع الرئيسي", 35.95, 39.0),
		ev("i", "شارع الرئيسي", 35.95, 39.0),
	}
	groups := GroupStreets(e, DefaultAreaParams())
	if len(groups) != 2 {
		t.Fatalf("مجموعتان: %+v", groups)
	}
	loc := &fakeLocator{ways: map[int64]float64{100: 35.9501, 200: 35.9900, 300: 35.9980}}
	used := 0
	byName := map[string]*StreetResolved{}
	for _, g := range groups {
		r, err := ResolveStreet(context.Background(), loc, g, DefaultAreaParams(), &used)
		if err != nil {
			t.Fatal(err)
		}
		if r != nil {
			byName[g.Name] = r
		}
	}
	w := byName["شارع الوادي"]
	if w == nil || len(w.WayIDs) != 1 || w.WayIDs[0] != 100 || w.Customers != 3 || len(w.Lines) == 0 || w.Named {
		t.Fatalf("الوادي: %+v", w)
	}
	// الكورنيش: ثلاثة زبائن على طريقين؛ الطريق ٢٠٠ فيه اثنان (أغلبية) → يُقبل بطريقٍ واحد.
	k := byName["شارع الكورنيش"]
	if k == nil || len(k.WayIDs) != 1 || k.WayIDs[0] != 200 || k.Customers != 2 {
		t.Fatalf("الكورنيش: %+v", k)
	}
	if used == 0 {
		t.Error("عدّاد النداءات")
	}

	// موزَّعون بلا أغلبية → لا اتّفاق.
	spread := StreetGroup{Name: "شارع س", Key: "شارع س", Bare: "س", Type: "شارع", Points: []AddrEvidence{
		ev("a", "", 35.9501, 39), ev("b", "", 35.9900, 39), ev("c", "", 35.9980, 39)}, Customers: 3}
	if r, _ := ResolveStreet(context.Background(), loc, spread, DefaultAreaParams(), nil); r != nil {
		t.Errorf("موزَّعٌ بلا اتّفاق: %+v", r)
	}
	// الاسم موجودٌ أصلاً على الطريق → Named.
	loc.names = map[int64][]string{100: {"شارع الوادي"}}
	r, _ := ResolveStreet(context.Background(), loc, groups[0], DefaultAreaParams(), nil)
	if groups[0].Name == "شارع الوادي" && (r == nil || !r.Named) {
		t.Errorf("الاسم موجود أصلاً: %+v", r)
	}
}

func TestResolveStreet_BudgetAndFarPoints(t *testing.T) {
	loc := &fakeLocator{ways: map[int64]float64{1: 35.0}}
	far := StreetGroup{Name: "شارع ص", Key: "شارع ص", Bare: "ص", Type: "شارع",
		Points: []AddrEvidence{ev("a", "", 35.5, 39), ev("b", "", 35.5, 39)}, Customers: 2}
	if r, _ := ResolveStreet(context.Background(), loc, far, DefaultAreaParams(), nil); r != nil {
		t.Errorf("بعيدٌ عن الطريق أكثر من الحدّ: %+v", r)
	}
	p := DefaultAreaParams()
	p.MaxLocatePerRun = 1
	used := 0
	_, err := ResolveStreet(context.Background(), loc, far, p, &used)
	if err != ErrLocateBudget {
		t.Errorf("نفاد الميزانية: %v", err)
	}
}

func TestHullAndLinesWKT(t *testing.T) {
	if hullWKT(nil) != nil || linesWKT(nil) != nil {
		t.Fatal("فارغ")
	}
	h := hullWKT([][2]float64{{1, 2}, {3, 2}, {3, 4}, {1, 2}})
	if h == nil || !strings.HasPrefix(*h, "POLYGON((1.0000000 2.0000000,") {
		t.Errorf("%v", h)
	}
	l := linesWKT([][][2]float64{{{1, 2}, {3, 4}}, {{5, 6}}})
	if l == nil || *l != fmt.Sprintf("MULTILINESTRING((%.7f %.7f,%.7f %.7f))", 1.0, 2.0, 3.0, 4.0) {
		t.Errorf("%v", l)
	}
}
