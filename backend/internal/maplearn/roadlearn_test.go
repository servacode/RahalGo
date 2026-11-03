package maplearn

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
)

func acc(v float64) *float64 { return &v }

// pt نقطةٌ على خطّ عرضٍ ثابتٍ بمسافةٍ شرقيّةٍ (م) من نقطة الأصل.
func pt(base time.Time, secs int, eastM float64, a *float64) drivermonitor.Point {
	lat := 35.4
	return drivermonitor.Point{
		At: base.Add(time.Duration(secs) * time.Second), Lat: lat,
		Lng: 38.4 + eastM/(111320*math.Cos(lat*math.Pi/180)), AccuracyM: a,
	}
}

func TestChunks_CleansAndSplits(t *testing.T) {
	base := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	p := DefaultRoadParams()
	var pts []drivermonitor.Point
	for i := 0; i < 6; i++ { // مقطعٌ سليم: ٦ نقاط بـ٨ م/ث
		pts = append(pts, pt(base, i*20, float64(i)*160, acc(5)))
	}
	mocked := pt(base, 130, 900, acc(5))
	mocked.Mocked = true
	pts = append(pts, mocked)
	pts = append(pts, pt(base, 150, 1100, acc(60))) // دقّةٌ سيّئة
	pts = append(pts, pt(base, 170, 1200, nil))     // بلا دقّة
	// صمتٌ ٥ دقائق ثم مقطعٌ ثانٍ من ٥ نقاط
	for i := 0; i < 5; i++ {
		pts = append(pts, pt(base, 500+i*20, 2000+float64(i)*160, acc(5)))
	}
	// واقفٌ: نقاطٌ في مكانٍ واحد بعدها لا تُحسب
	for i := 0; i < 3; i++ {
		pts = append(pts, pt(base, 600+i*20, 2640+float64(i), acc(5)))
	}
	got := Chunks(pts, p)
	if len(got) != 2 {
		t.Fatalf("المتوقَّع مقطعان، والناتج %d", len(got))
	}
	if len(got[0]) != 6 {
		t.Errorf("المقطع الأول: %d نقاط والمتوقَّع ٦ (المزيَّف وسيّئ الدقّة وبلا الدقّة ساقطة)", len(got[0]))
	}
	if len(got[1]) != 5 {
		t.Errorf("المقطع الثاني: %d والمتوقَّع ٥ (الواقف ساقط)", len(got[1]))
	}
	// قصيرٌ جدّاً يُرمى
	if n := len(Chunks(pts[:3], p)); n != 0 {
		t.Errorf("مقطعٌ من ٣ نقاط يجب أن يُرمى، والناتج %d", n)
	}
	// قفزة GPS
	jump := []drivermonitor.Point{
		pt(base, 0, 0, acc(5)), pt(base, 20, 160, acc(5)), pt(base, 40, 50000, acc(5)),
		pt(base, 60, 320, acc(5)), pt(base, 80, 480, acc(5)),
	}
	if c := Chunks(jump, p); len(c) != 1 || len(c[0]) != 4 {
		t.Errorf("القفزة يجب أن تسقط وحدَها: %v", c)
	}
}

func TestCanonEdge_SameKeyBothDirections(t *testing.T) {
	east := [][2]float64{{38.400, 35.4}, {38.401, 35.4}, {38.402, 35.4}}
	west := [][2]float64{{38.402, 35.4}, {38.401, 35.4}, {38.400, 35.4}}
	k1, d1, c1 := CanonEdge(77, east)
	k2, d2, c2 := CanonEdge(77, west)
	if k1 != k2 {
		t.Fatalf("مفتاحا الاتّجاهين يجب أن يتطابقا: %q %q", k1, k2)
	}
	if d1 != 1 || d2 != -1 {
		t.Errorf("الاتّجاهان %d,%d والمتوقَّع ١ و-١", d1, d2)
	}
	if c1[0] != c2[0] || c1[2] != c2[2] {
		t.Errorf("الهندسة القانونيّة يجب أن تتطابق: %v %v", c1, c2)
	}
	// شمالاً أيضاً قانونيٌّ بخطّ العرض
	if _, d, _ := CanonEdge(1, [][2]float64{{38.4, 35.40}, {38.4, 35.41}}); d != 1 {
		t.Errorf("الشمال اتّجاهٌ قانونيّ: %d", d)
	}
}

func TestDetectOneWay(t *testing.T) {
	p := DefaultOneWayParams()
	stats := []EdgeDirStat{
		{Key: "one", Fwd: 60, Rev: 1, Drivers: 8, Days: 9},         // واضح
		{Key: "rev", Fwd: 2, Rev: 40, Drivers: 6, Days: 7},         // واضح باتّجاه معاكس
		{Key: "two", Fwd: 30, Rev: 28, Drivers: 8, Days: 9},        // اتّجاهان
		{Key: "edge", Fwd: 94, Rev: 6, Drivers: 8, Days: 9},        // ٩٤٪ فقط
		{Key: "few-pass", Fwd: 15, Rev: 0, Drivers: 8, Days: 9},    // عبورات قليلة
		{Key: "few-drivers", Fwd: 50, Rev: 0, Drivers: 4, Days: 9}, // سائقون قليلون
		{Key: "few-days", Fwd: 50, Rev: 0, Drivers: 8, Days: 4},    // أيّامٌ قليلة
	}
	got := DetectOneWay(stats, p)
	if len(got) != 2 || got[0].Key != "one" || got[1].Key != "rev" {
		t.Fatalf("المتوقَّع one ثم rev، والناتج %+v", got)
	}
	if got[0].Dir != 1 || got[1].Dir != -1 {
		t.Errorf("الاتّجاهان %d,%d", got[0].Dir, got[1].Dir)
	}
	if got[0].Share < 0.95 || got[0].Main != 60 || got[0].Opp != 1 {
		t.Errorf("الأدلّة: %+v", got[0])
	}
}

func TestSlowFromCounts(t *testing.T) {
	p := DefaultSlowParams()
	cases := []struct {
		name                    string
		passes, days, qual, slw int
		want                    bool
	}{
		{"بطيئةٌ في كلّ الخانات", 80, 6, 4, 4, true},
		{"بطيئةٌ في ٣ من ٤", 80, 6, 4, 3, true},
		{"بطيئةٌ في نصف الخانات (ذروةٌ فقط)", 80, 6, 4, 2, false},
		{"عبوراتٌ قليلة", 10, 6, 4, 4, false},
		{"أيّامٌ قليلة", 80, 2, 4, 4, false},
		{"خاناتٌ قليلة", 80, 6, 2, 2, false},
	}
	for _, c := range cases {
		if got := SlowFromCounts(c.passes, c.days, c.qual, c.slw, p); got != c.want {
			t.Errorf("%s: %v والمتوقَّع %v", c.name, got, c.want)
		}
	}
}

func TestPasses_SpeedHourAndEdges(t *testing.T) {
	// الجمعة ١٣ ١١ ٢٠٢٦ عند ١٤:٠٠ بتوقيت دمشق = نهايةُ أسبوع، والساعة ١٤.
	loc := drivermonitor.Loc()
	base := time.Date(2026, 11, 6, 14, 0, 0, 0, loc)
	if base.Weekday() != time.Friday {
		t.Fatalf("التاريخ ليس جمعة: %v", base.Weekday())
	}
	var pts []drivermonitor.Point
	for i := 0; i < 5; i++ { // ٥ م/ث
		pts = append(pts, pt(base, i*20, float64(i)*100, acc(5)))
	}
	line := [][2]float64{{38.4, 35.4}, {38.401, 35.4}}
	key, dir, canon := CanonEdge(5, line)
	short := MatchedEdge{Key: "short", Length: 5, Dir: 1}
	res := MatchResult{
		Edges: []MatchedEdge{
			{Key: key, WayID: 5, Length: 90, Dir: dir, Line: canon},
			{Key: key + "x", WayID: 5, Length: 90, Dir: -1, Line: canon},
			short,
		},
		PointEdge: []int{0, 0, 1, 1, 1},
	}
	got := Passes(pts, res, DefaultRoadParams())
	if len(got) != 2 {
		t.Fatalf("عبوران (القصيرُ ساقط)، والناتج %d", len(got))
	}
	for _, p := range got {
		if !p.HasSpeed || math.Abs(p.SpeedMps-5) > 0.1 {
			t.Errorf("السرعة %v والمتوقَّع ٥ م/ث", p.SpeedMps)
		}
		if p.Hour != 14 || !p.Weekend {
			t.Errorf("الساعة/نهاية الأسبوع: %d %v", p.Hour, p.Weekend)
		}
	}
	if got[1].Dir != -1 {
		t.Errorf("اتّجاه الثاني %d", got[1].Dir)
	}
}

// encodePolyline6 مُشفِّر اختباريّ لصيغة Valhalla (دقّة ٦ منازل) من (lng,lat).
func encodePolyline6(pts [][2]float64) string {
	var out []byte
	var pLat, pLng int
	enc := func(v int) {
		v <<= 1
		if v < 0 {
			v = ^v
		}
		for v >= 0x20 {
			out = append(out, byte((0x20|(v&0x1f))+63))
			v >>= 5
		}
		out = append(out, byte(v+63))
	}
	for _, p := range pts {
		lat, lng := int(math.Round(p[1]*1e6)), int(math.Round(p[0]*1e6))
		enc(lat - pLat)
		enc(lng - pLng)
		pLat, pLng = lat, lng
	}
	return string(out)
}

func TestPolyline6_RoundTrip(t *testing.T) {
	in := [][2]float64{{38.400000, 35.400000}, {38.401234, 35.400567}, {38.399000, 35.402000}}
	out := decodePolyline6(encodePolyline6(in))
	if len(out) != len(in) {
		t.Fatalf("الطول %d", len(out))
	}
	for i := range in {
		if math.Abs(in[i][0]-out[i][0]) > 1e-6 || math.Abs(in[i][1]-out[i][1]) > 1e-6 {
			t.Errorf("النقطة %d: %v ≠ %v", i, out[i], in[i])
		}
	}
}

func TestValhallaMatcher_ParsesTraceAttributes(t *testing.T) {
	shape := [][2]float64{{38.400, 35.4}, {38.401, 35.4}, {38.402, 35.4}, {38.401, 35.4}, {38.400, 35.4}}
	one := 0
	two := 1
	body := map[string]any{
		"shape": encodePolyline6(shape),
		"edges": []map[string]any{
			{"way_id": 42, "names": []string{"شارع الاختبار"}, "length": 0.09, "begin_shape_index": 0, "end_shape_index": 1, "traversability": "both"},
			// حافّةٌ تُعبَر غرباً (عكس الترتيب القانوني) ومعروفةٌ باتّجاه واحد
			{"way_id": 43, "names": []string{}, "length": 0.18, "begin_shape_index": 2, "end_shape_index": 4, "traversability": "forward"},
			{"way_id": 44, "length": 0.09, "begin_shape_index": 1, "end_shape_index": 2, "traversability": "both"},
		},
		"matched_points": []map[string]any{
			{"type": "matched", "edge_index": one},
			{"type": "unmatched"},
			{"type": "matched", "edge_index": two},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/trace_attributes" || r.Method != http.MethodPost {
			t.Errorf("نداءٌ غير متوقَّع: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()
	pts := []drivermonitor.Point{{}, {}, {}}
	res, err := NewMatcher(srv.URL).Match(context.Background(), pts)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Edges) != 3 {
		t.Fatalf("حافّات: %d", len(res.Edges))
	}
	e0 := res.Edges[0]
	if e0.Key == "" || e0.Dir != 1 || e0.Name != "شارع الاختبار" || math.Abs(e0.Length-90) > 0.01 || e0.OneWayKnown {
		t.Errorf("الحافّة ٠: %+v", e0)
	}
	e1 := res.Edges[1]
	if e1.Dir != -1 || !e1.OneWayKnown || len(e1.Line) != 3 || e1.Line[0][0] > e1.Line[2][0] {
		t.Errorf("الحافّة ١ (غرباً ومعروفةٌ باتّجاه واحد، وهندستها قانونيّةٌ شرقاً): %+v", e1)
	}
	if res.PointEdge[0] != 0 || res.PointEdge[1] != -1 || res.PointEdge[2] != 1 {
		t.Errorf("نقاط→حافّات: %v", res.PointEdge)
	}

	// ٤٠٠ = لا مطابقة، و٥٠٠ = عطل.
	for code, wantNoMatch := range map[int]bool{400: true, 503: false} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }))
		_, err := NewMatcher(s.URL).Match(context.Background(), pts)
		s.Close()
		if wantNoMatch && err != ErrNoMatch {
			t.Errorf("%d: %v والمتوقَّع ErrNoMatch", code, err)
		}
		if !wantNoMatch && (err == nil || err == ErrNoMatch) {
			t.Errorf("%d: %v والمتوقَّع عطل", code, err)
		}
	}
	if NewMatcher("") != nil {
		t.Error("بلا عنوانٍ يجب أن يكون المطابقُ nil")
	}
}
