package roadmatch

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/polyline"
)

// ══════════════════════════════════════════════════════════════════════
// **مطابقة مسار السائق** — طلب المالك ٢٠٢٦-١١-٠٥. كل الاختبارات بمحرّك مزيَّف (لا Valhalla حقيقي).
// ══════════════════════════════════════════════════════════════════════

const (
	tLat = 35.40
	tLng = 38.40
	mLng = 1.0 / 90700 // درجة طول لكل متر تقريباً عند خط عرض الرقة
	mLat = 1.0 / 111200
)

var t0 = time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)

func f(v float64) *float64 { return &v }

// east نقطة ع الشارع الشرقي على بعد `m` متر، بعد `sec` ثانية.
func east(sec float64, m float64) drivermonitor.Point {
	return drivermonitor.Point{At: t0.Add(time.Duration(sec * float64(time.Second))), Lat: tLat, Lng: tLng + m*mLng,
		AccuracyM: f(8), SpeedMps: f(8)}
}

// drive نقاط كل `step` ثانية بسرعة `v` م/ث من `m0` لمدة `dur` ثانية.
func drive(start, dur, step, m0, v float64) []drivermonitor.Point {
	var out []drivermonitor.Point
	for s := 0.0; s <= dur+1e-9; s += step {
		out = append(out, east(start+s, m0+v*s))
	}
	return out
}

func sentIdx(c Chunk) []int {
	var out []int
	for i, s := range c.Sent {
		if s {
			out = append(out, i)
		}
	}
	return out
}

// RM1 — قطع عند الصمت والوقفة الطويلة (نفس وقفات التحليل) والقفزة > ٢ كم، و≤ ١٥٠ مبعوتة، والمزيَّف والدقة
// السيّئة ما بتنبعت.
func TestChunks_SplitAtGapsStopsAndLongHops(t *testing.T) {
	var pts []drivermonitor.Point
	pts = append(pts, drive(0, 120, 4, 0, 8)...) // ٠–١٢٠ ث
	// نقطة دقتها ٢٠٠ م بنص الطريق ⇒ التنظيف بيشيلها.
	bad := east(122, 976)
	bad.AccuracyM = f(200)
	pts = append(pts, bad)
	// صمت ٧ دقايق.
	pts = append(pts, drive(540, 60, 4, 1500, 8)...) // ٥٤٠–٦٠٠ ث، لحد ١٩٨٠ م
	// وقفة ٤ دقايق بمكانه، وفيها نقطة مزيَّفة.
	for s := 620.0; s <= 860; s += 20 {
		p := east(s, 1980+math.Mod(s, 3))
		p.SpeedMps = f(0)
		if s == 700 {
			p.Mocked = true
		}
		pts = append(pts, p)
	}
	pts = append(pts, drive(880, 60, 4, 1990, 8)...) // لحد ٢٤٧٠ م
	// قفزة ٢٫٢ كم بدقيقتين (٦٦ كم/س — التنظيف بيقبلها).
	pts = append(pts, drive(1060, 900, 4, 4670, 8)...) // ١٥ دقيقة = ٢٢٦ نقطة

	until := pts[len(pts)-1].At.Add(time.Minute)
	chunks := Chunks(pts, drivermonitor.DefaultConfig(), until)
	clean := drivermonitor.Clean(pts, drivermonitor.DefaultConfig())
	stops := drivermonitor.LongStops(clean, drivermonitor.DefaultConfig())
	if len(stops) != 1 {
		t.Fatalf("الوقفات الطويلة: %d", len(stops))
	}
	// ١: قبل الصمت · ٢: بعده لأول الوقفة · ٣: من آخر الوقفة للقفزة · ٤ و٥: بعد القفزة (١٥٠ + الباقي).
	if len(chunks) != 5 {
		for i, c := range chunks {
			t.Logf("قطعة %d: %v → %v (%d نقطة، %d مبعوتة)", i, c.From().Sub(t0), c.To().Sub(t0), len(c.Points), c.SentCount())
		}
		t.Fatalf("عدد القطع: %d", len(chunks))
	}
	if !chunks[0].To().Equal(t0.Add(120*time.Second)) || !chunks[1].From().Equal(t0.Add(540*time.Second)) {
		t.Errorf("القطع عند الصمت: %v / %v", chunks[0].To().Sub(t0), chunks[1].From().Sub(t0))
	}
	if !chunks[1].To().Equal(stops[0].Start) || !chunks[2].From().Equal(stops[0].End) {
		t.Errorf("القطع عند الوقفة: %v / %v والوقفة %v→%v", chunks[1].To().Sub(t0), chunks[2].From().Sub(t0),
			stops[0].Start.Sub(t0), stops[0].End.Sub(t0))
	}
	if !chunks[2].To().Equal(t0.Add(940*time.Second)) || !chunks[3].From().Equal(t0.Add(1060*time.Second)) {
		t.Errorf("القطع عند القفزة: %v / %v", chunks[2].To().Sub(t0), chunks[3].From().Sub(t0))
	}
	if chunks[3].SentCount() != MaxSent || !chunks[4].From().Equal(chunks[3].To()) {
		t.Errorf("حدّ الـ١٥٠: %d مبعوتة، والقطعة التالية من %v (المتوقَّع %v)", chunks[3].SentCount(),
			chunks[4].From().Sub(t0), chunks[3].To().Sub(t0))
	}
	for i, c := range chunks {
		if i < 4 && !c.Closed {
			t.Errorf("القطعة %d لازم تكون مسكّرة", i)
		}
		for k, p := range c.Points {
			if p.AccuracyM != nil && *p.AccuracyM > 50 {
				t.Errorf("نقطة دقتها سيّئة بالقطعة %d", i)
			}
			if p.Mocked && c.Sent[k] {
				t.Errorf("نقطة مزيَّفة انبعتت بالقطعة %d", i)
			}
		}
	}
	if chunks[4].Closed {
		t.Error("آخر قطعة (بعدها دقيقة بس) لازم تكون مفتوحة")
	}
}

// RM2 — رجفة واقف ٢٥–٤٦ م كل ٢٠ ث لـ١٠ دقايق ⇒ ≤ ٢ مبعوتة والباقي بياخد موضعها (صفر متر)؛
// ونتوء ١٥٠ م بـ٥ ث ما بينبعت.
func TestChunks_JitterAndSpikesCarryPosition(t *testing.T) {
	var pts []drivermonitor.Point
	for i := 0; i <= 30; i++ {
		p := east(float64(i*20), 0)
		// شمال وجنوب بالتناوب ٢٥–٤٦ م: نقطتين ورا بعض أبعد من ٤٠ م ⇒ ولا وقفة طويلة بيشوفها التحليل.
		off := (25 + float64((i/2)%4)*7) * mLat
		switch {
		case i == 0:
		case i%2 == 1:
			p.Lat += off
		default:
			p.Lat -= off
		}
		p.AccuracyM, p.SpeedMps = f(48), nil
		pts = append(pts, p)
	}
	chunks := Chunks(pts, drivermonitor.DefaultConfig(), pts[len(pts)-1].At)
	if len(chunks) != 1 {
		t.Fatalf("قطع الرجفة: %d", len(chunks))
	}
	if n := chunks[0].SentCount(); n > 2 {
		t.Errorf("رجفة الواقف: %d نقطة مبعوتة", n)
	}
	rows := Rows(chunks[0], fakeAnswer(sentPoints(chunks[0])))
	for _, r := range rows {
		if r.RoadM > 1e-6 {
			t.Errorf("رجفة الواقف انحسبت %.1f م ع الطريق", r.RoadM)
		}
	}

	// نتوء: درّاج ٣ م/ث كل ٥ ث، ونقطة وحدة ١٥٠ م شمالاً.
	sp := drive(0, 100, 5, 0, 3)
	for i := range sp {
		sp[i].AccuracyM = f(5)
	}
	sp[10].Lat += 150 * mLat
	chunks = Chunks(sp, drivermonitor.DefaultConfig(), sp[len(sp)-1].At)
	if len(chunks) != 1 || len(chunks[0].Points) != len(sp) {
		t.Fatalf("النتوء لازم يضل بالقطعة (التنظيف ما بيشيله): %d قطع", len(chunks))
	}
	if chunks[0].Sent[10] {
		t.Error("النتوء انبعت")
	}
	if !chunks[0].Sent[9] || !chunks[0].Sent[11] {
		t.Error("جيران النتوء لازم ينبعتوا")
	}
}

// fakeAnswer محرّك مزيَّف ع الشارع الشرقي: كل نقطة بتنلصق ع خط العرض، والخط من أصغر لأكبر طول.
func fakeAnswer(sent []drivermonitor.Point) Answer {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, p := range sent {
		lo, hi = math.Min(lo, p.Lng), math.Max(hi, p.Lng)
	}
	line := [][2]float64{{lo, tLat}, {hi, tLat}}
	a := Answer{Shape: polyline.Encode6(line), Edges: []Edge{{Length: polyline.Length(line) / 1000, BeginShapeIndex: 0, EndShapeIndex: 1}}}
	for _, p := range sent {
		e := int64(0)
		a.MatchedPoints = append(a.MatchedPoints, MatchedPoint{Lat: tLat, Lon: p.Lng, Type: "matched", EdgeIndex: &e,
			DistanceFromTracePoint: f(drivermonitor.Dist(p, drivermonitor.Point{Lat: tLat, Lng: p.Lng}))})
	}
	return a
}

// RM3 — جسم الطلب بالحرف.
func TestRequest_Golden(t *testing.T) {
	a := east(0, 0)
	a.AccuracyM = f(5) // ٢ × ٥ = ١٠ ⇒ ٢٥
	b := east(4, 40)
	b.AccuracyM = f(30) // ٦٠
	c := east(8, 80)
	c.AccuracyM = f(80) // ١٦٠ ⇒ ١٠٠
	d := east(12, 120)
	d.AccuracyM = nil // مجهولة ⇒ ٥٠
	got := Request([]drivermonitor.Point{a, b, c, d})
	want := `{"costing":"motor_scooter",` +
		`"costing_options":{"motor_scooter":{"ignore_access":false,"ignore_closures":true,"ignore_oneways":false,"ignore_restrictions":true}},` +
		`"filters":{"action":"include","attributes":["shape","matched.point","matched.type","matched.edge_index",` +
		`"matched.distance_along_edge","matched.distance_from_trace_point","matched.begin_route_discontinuity",` +
		`"matched.end_route_discontinuity","edge.length","edge.begin_shape_index","edge.end_shape_index"]},` +
		`"shape":[` +
		`{"lat":35.4,"lon":38.4,"time":1793872800,"radius":25},` +
		`{"lat":35.4,"lon":38.40044101433296,"time":1793872804,"radius":60},` +
		`{"lat":35.4,"lon":38.40088202866593,"time":1793872808,"radius":100},` +
		`{"lat":35.4,"lon":38.4013230429989,"time":1793872812,"radius":50}],` +
		`"shape_match":"map_snap",` +
		`"trace_options":{"gps_accuracy":30,"interpolation_distance":10,"search_radius":50,"turn_penalty_factor":100},` +
		`"use_timestamps":true}`
	if string(got) != want {
		t.Errorf("الطلب:\n%s\nالمتوقَّع:\n%s", got, want)
	}
	var body map[string]any
	_ = json.Unmarshal(got, &body)
	if _, ok := body["trace_options"].(map[string]any)["breakage_distance"]; ok {
		t.Error("breakage_distance ما بينبعت (المحرّك ما بيقبل تغييرها)")
	}
}

// lineAnswer جواب بخط معيّن وحافّات بمداها ونقاط مطابقة ع حافّاتها.
type mp struct {
	lng, lat float64
	edge     int64
	typ      string
	far      float64
	begin    bool
}

func lineAnswer(line [][2]float64, edges [][2]int, pts []mp) Answer {
	a := Answer{Shape: polyline.Encode6(line)}
	for _, e := range edges {
		a.Edges = append(a.Edges, Edge{BeginShapeIndex: e[0], EndShapeIndex: e[1]})
	}
	for _, p := range pts {
		e := p.edge
		typ := p.typ
		if typ == "" {
			typ = "matched"
		}
		m := MatchedPoint{Lat: p.lat, Lon: p.lng, Type: typ, DistanceFromTracePoint: f(p.far), BeginRouteDiscontinuity: p.begin}
		if typ != "unmatched" {
			m.EdgeIndex = &e
		}
		a.MatchedPoints = append(a.MatchedPoints, m)
	}
	return a
}

func sentAt(n int) []drivermonitor.Point {
	out := make([]drivermonitor.Point, n)
	for i := range out {
		out[i] = east(float64(i*4), float64(i)*30)
	}
	return out
}

// RM4 — خط فيه لفّة حول بلوك: رجفة لورا = صفر، والمجموع = طول الخط.
func TestPositions_ForwardOnly(t *testing.T) {
	// شرقاً ٢٠٠ م، لفّة (شمال ١٠٠، غرب ١٠٠، جنوب ١٠٠)، ثم شرقاً ٣٠٠ م.
	x := func(m float64) float64 { return tLng + m*mLng }
	y := func(m float64) float64 { return tLat + m*mLat }
	line := [][2]float64{{x(0), y(0)}, {x(200), y(0)}, {x(200), y(100)}, {x(100), y(100)}, {x(100), y(0)}, {x(400), y(0)}}
	edges := [][2]int{{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 5}}
	pts := []mp{
		{x(0), y(0), 0, "", 0, false},
		{x(150), y(0), 0, "", 0, false},
		{x(140), y(0), 0, "", 0, false}, // رجفة لورا
		{x(200), y(50), 1, "", 0, false},
		{x(150), y(100), 2, "", 0, false},
		{x(100), y(50), 3, "", 0, false},
		{x(150), y(0), 4, "", 0, false}, // نفس مكان النقطة ١ بس بالمرور التاني
		{x(400), y(0), 4, "", 0, false},
	}
	pos, _, line2, cum := SentPositions(sentAt(len(pts)), lineAnswer(line, edges, pts))
	total := cum[len(cum)-1]
	if math.Abs(polyline.Length(line2)-total) > 1e-6 || math.Abs(total-800) > 3 {
		t.Fatalf("طول الخط %.1f", total)
	}
	if pos[2] != pos[1] {
		t.Errorf("الرجفة لورا لازم تكون صفر: %.1f بعد %.1f", pos[2], pos[1])
	}
	if math.Abs(pos[6]-(pos[5]+50+50)) > 3 || pos[6] <= 400 {
		t.Errorf("المرور التاني ع نفس النقطة: %.1f", pos[6])
	}
	if math.Abs(pos[7]-total) > 1e-6 {
		t.Errorf("آخر نقطة %.1f والطول %.1f", pos[7], total)
	}
}

// RM4b — داخل وطالع من زقاق مسدود والتسليم بآخره: كل نقطة ع مرورها، والأمتار قبل التسليم = الطريق لجوّا.
func TestPositions_DeadEndInAndOut(t *testing.T) {
	x := func(m float64) float64 { return tLng + m*mLng }
	line := [][2]float64{{x(0), tLat}, {x(200), tLat}, {x(0), tLat}}
	edges := [][2]int{{0, 1}, {1, 2}}
	var pts []mp
	for _, m := range []float64{0, 50, 100, 150, 200} {
		pts = append(pts, mp{lng: x(m), lat: tLat, edge: 0})
	}
	for _, m := range []float64{150, 100, 50, 0} {
		pts = append(pts, mp{lng: x(m), lat: tLat, edge: 1})
	}
	pos, _, _, _ := SentPositions(sentAt(len(pts)), lineAnswer(line, edges, pts))
	if math.Abs(pos[4]-200) > 2 {
		t.Errorf("الأمتار لحد التسليم %.1f (المتوقَّع ٢٠٠)", pos[4])
	}
	for i := 5; i < len(pos); i++ {
		want := 200 + float64(i-4)*50
		if math.Abs(pos[i]-want) > 2 {
			t.Errorf("نقطة الطلعة %d: %.1f (المتوقَّع %.0f)", i, pos[i], want)
		}
	}
}

// RM5 — انقطاع بالطريق بيقسم السطر، ونقطة لحالها ما انطابقت بتاخد موضع اللي قبلها، و٣ ورا بعض ⇒ وتر.
func TestParse_DiscontinuityAndOutliers(t *testing.T) {
	x := func(m float64) float64 { return tLng + m*mLng }
	line := [][2]float64{{x(0), tLat}, {x(1000), tLat}}
	edges := [][2]int{{0, 1}}
	pts := []mp{
		{lng: x(0), lat: tLat}, {typ: "unmatched"}, {lng: x(100), lat: tLat}, {lng: x(150), lat: tLat},
		{typ: "unmatched"}, {typ: "unmatched"}, {typ: "unmatched"},
		{lng: x(500), lat: tLat, begin: true}, {lng: x(600), lat: tLat},
	}
	sent := sentAt(len(pts))
	pos, brk, _, _ := SentPositions(sent, lineAnswer(line, edges, pts))
	if pos[1] != pos[0] {
		t.Errorf("نقطة لحالها لازم تاخد موضع اللي قبلها: %.1f", pos[1])
	}
	for i := 4; i <= 6; i++ {
		if !math.IsNaN(pos[i]) {
			t.Errorf("٣ ورا بعض لازم بلا موضع: %d = %.1f", i, pos[i])
		}
	}
	if !brk[7] {
		t.Error("الانقطاع ما انقرأ")
	}
	c := Chunk{Points: sent, Sent: make([]bool, len(sent)), Closed: true}
	for i := range c.Sent {
		c.Sent[i] = true
	}
	rows := Rows(c, lineAnswer(line, edges, pts))
	if len(rows) != 2 || !rows[0].ToAt.Equal(sent[6].At) || !rows[1].FromAt.Equal(sent[7].At) {
		t.Fatalf("السطور عند الانقطاع: %d", len(rows))
	}
	if rows[1].PosM[0] != 0 || math.Abs(rows[1].PosM[1]-100) > 1 || math.Abs(rows[1].RoadM-100) > 1 {
		t.Errorf("السطر التاني من أوله: %v (%.1f)", rows[1].PosM, rows[1].RoadM)
	}
	if math.Abs(rows[0].RoadM-150) > 1 {
		t.Errorf("أمتار السطر الأول %.1f (المتوقَّع ١٥٠: بلا المقاطع اللي بلا موضع)", rows[0].RoadM)
	}
}

// RM10 — برّا الخريطة: نقطتين ورا بعض انلصقوا ٤٠ م بعاد (دقة ١٠) ⇒ بلا موضع؛ وحدة لحالها بتاخد موضع اللي قبلها.
func TestOffMap_FarMatchedPoints(t *testing.T) {
	x := func(m float64) float64 { return tLng + m*mLng }
	line := [][2]float64{{x(0), tLat}, {x(1000), tLat}}
	edges := [][2]int{{0, 1}}
	pts := []mp{
		{lng: x(0), lat: tLat}, {lng: x(100), lat: tLat, far: 40}, {lng: x(200), lat: tLat},
		{lng: x(300), lat: tLat, far: 40}, {lng: x(400), lat: tLat, far: 40}, {lng: x(500), lat: tLat},
	}
	sent := sentAt(len(pts))
	for i := range sent {
		sent[i].AccuracyM = f(10)
	}
	pos, _, _, _ := SentPositions(sent, lineAnswer(line, edges, pts))
	if pos[1] != pos[0] {
		t.Errorf("بعيدة لحالها لازم تاخد موضع اللي قبلها: %.1f", pos[1])
	}
	if !math.IsNaN(pos[3]) || !math.IsNaN(pos[4]) {
		t.Errorf("بعيدتين ورا بعض لازم بلا موضع: %.1f %.1f", pos[3], pos[4])
	}
	if math.Abs(pos[5]-500) > 1 {
		t.Errorf("بعد برّا الخريطة: %.1f", pos[5])
	}
}

// ───────────────────────── الأخطاء ─────────────────────────

type script struct {
	calls int
	fn    func(n int, sent []drivermonitor.Point) (Answer, error)
}

func (s *script) Match(_ context.Context, sent []drivermonitor.Point) (Answer, error) {
	s.calls++
	return s.fn(s.calls, sent)
}

// RM8 — 400 ⇒ `no_match` برمزه؛ 503 ⇒ ولا شي وبتنعاد؛ مهلة ⇒ نصّين؛ مهلة تانية لقطعة صغيرة ⇒
// `no_match('timeout')`؛ ٥ 400 ورا بعض ⇒ عاصفة (ولا شي بينخزّن والقاطع بيفتح)؛ ٣ أعطال ⇒ قاطع.
func TestClient_ErrorClasses(t *testing.T) {
	// العميل الحقيقي ع محرّك مزيَّف.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Shape []json.RawMessage `json:"shape"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch len(body.Shape) {
		case 2:
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error_code":442,"error":"No path could be found for input","status_code":400}`))
		case 3:
			w.WriteHeader(503)
		case 4:
			time.Sleep(300 * time.Millisecond)
		default:
			_, _ = w.Write([]byte(`{"shape":"","edges":[],"matched_points":[]}`))
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	c.Timeout = 100 * time.Millisecond
	ctx := context.Background()
	var nm *NoMatchError
	if _, err := c.Match(ctx, sentAt(2)); !errors.As(err, &nm) || nm.Code != "442" {
		t.Errorf("400: %v", err)
	}
	if _, err := c.Match(ctx, sentAt(3)); !errors.Is(err, ErrDown) {
		t.Errorf("503: %v", err)
	}
	if _, err := c.Match(ctx, sentAt(4)); !errors.Is(err, ErrTimeout) {
		t.Errorf("مهلة: %v", err)
	}
	if _, err := c.Match(ctx, sentAt(5)); err != nil {
		t.Errorf("200: %v", err)
	}

	var res TickResult
	// ٥ 400 ورا بعض ⇒ عاصفة.
	bad := &script{fn: func(int, []drivermonitor.Point) (Answer, error) { return Answer{}, &NoMatchError{Code: "171"} }}
	w := &Worker{Matcher: bad, Opt: DefaultOptions()}
	tl := &tally{budget: 60}
	small := Chunk{Points: drive(0, 40, 4, 0, 8), Closed: true}
	small.Sent = make([]bool, len(small.Points))
	for i := range small.Sent {
		small.Sent[i] = true
	}
	for i := 0; i < 5 && !tl.storm; i++ {
		if _, err := w.matchChunk(ctx, small, t0, w.opt(), tl); err != nil {
			t.Fatal(err)
		}
	}
	res = TickResult{}
	if !w.judge(tl, t0, w.opt(), &res) || !res.Storm {
		t.Error("٥ 400 ورا بعض لازم تكون عاصفة")
	}
	if st := w.Status(t0.Add(time.Minute)); st.PausedUntil == nil || !st.PausedUntil.Equal(t0.Add(5*time.Minute)) {
		t.Errorf("القاطع بعد العاصفة: %+v", st)
	}
	if st := w.Status(t0); st.LastError != "storm" {
		t.Errorf("آخر خطأ بعد العاصفة لازم يكون صنفه storm مو نصّ: %q", st.LastError)
	}

	// ٣ أعطال ورا بعض ⇒ قاطع.
	down := &script{fn: func(int, []drivermonitor.Point) (Answer, error) { return Answer{}, ErrDown }}
	w = &Worker{Matcher: down, Opt: DefaultOptions()}
	tl = &tally{budget: 60}
	var last error
	for i := 0; i < 3; i++ {
		_, last = w.matchChunk(ctx, small, t0, w.opt(), tl)
	}
	if !errors.Is(last, errBreaker) {
		t.Errorf("العطل التالت لازم يفتح القاطع: %v", last)
	}
	if st := w.Status(t0); st.LastError != "down" {
		t.Errorf("آخر خطأ بعد العطل لازم يكون صنفه down مو نصّ: %q", st.LastError)
	}
	if res, err := w.Tick(ctx, t0.Add(time.Minute), true); err != nil || res.Calls != 0 {
		t.Errorf("القاطع مفتوح ⇒ ولا نداء: %+v %v", res, err)
	}
}

// Q-R10 (جزء) — المطابقة مطفية، أو جولة تعلّم الطرق شغّالة ⇒ ولا نداء (وما في قاعدة أصلاً).
func TestWorker_IdleWhenOffOrRoadsBusy(t *testing.T) {
	sc := &script{fn: func(int, []drivermonitor.Point) (Answer, error) { return Answer{}, nil }}
	w := &Worker{Matcher: sc, Opt: DefaultOptions()}
	if res, err := w.Tick(context.Background(), t0, false); err != nil || res.Calls != 0 || w.Status(t0).State != "off" {
		t.Errorf("مطفية: %+v %v %s", res, err, w.Status(t0).State)
	}
	w.Busy = func() bool { return true }
	if res, err := w.Tick(context.Background(), t0, true); err != nil || res.Calls != 0 || w.Status(t0).State != "waiting" {
		t.Errorf("تعلّم الطرق شغّال: %+v %v %s", res, err, w.Status(t0).State)
	}
	if sc.calls != 0 {
		t.Errorf("نداءات: %d", sc.calls)
	}
}

// RM8b (مراجعة ٢٠٢٦-١١-٠٦) — **المهلة بتفتح القاطع فوراً**: خيط Valhalla الوحيد بيضل شغّال ع النداء
// المتروك، فلا تقسيم ولا إعادة بنفس الدورة (كانت المهلة تكبّر النداءات بالضبط وقت المحرّك بطيء).
// والقطعة اللي خلصت مهلتها بتنقسم **بالمحاولة الجاية** (بعد الوقفة)، والنص الصغير اللي خلصت مهلته
// مرّتين بيتخزّن `no_match('timeout')` بلا نداء تالت.
func TestWorker_TimeoutOpensBreakerSplitsNextTime(t *testing.T) {
	ctx := context.Background()
	ch := Chunk{Points: drive(0, 39*4, 4, 0, 8), Closed: true}
	ch.Sent = make([]bool, len(ch.Points))
	for i := range ch.Sent {
		ch.Sent[i] = true
	}
	// النص الأول بينجح، وكل شي تاني مهلة.
	sc := &script{fn: func(_ int, sent []drivermonitor.Point) (Answer, error) {
		if len(sent) < 40 && sent[0].At.Equal(ch.Points[0].At) {
			return fakeAnswer(sent), nil
		}
		return Answer{}, ErrTimeout
	}}
	w := &Worker{Matcher: sc, Opt: DefaultOptions(), Rest: func(context.Context, time.Duration) {}}
	o := w.opt()

	rows, err := w.matchChunk(ctx, ch, t0, o, &tally{budget: 60})
	if !errors.Is(err, errBreaker) || rows != nil || sc.calls != 1 {
		t.Fatalf("أول مهلة: خطأ %v، نداءات %d (لازم ١ بلا تقسيم ولا إعادة)، سطور %d", err, sc.calls, len(rows))
	}
	st := w.Status(t0.Add(time.Minute))
	if st.State != "" && st.State != "paused" || st.PausedUntil == nil || !st.PausedUntil.Equal(t0.Add(o.BreakerPause)) {
		t.Errorf("المهلة لازم توقّف المطابق: %+v", st)
	}
	if st.LastError != "timeout" {
		t.Errorf("صنف آخر خطأ: %q", st.LastError)
	}
	if res, err := w.Tick(ctx, t0.Add(time.Minute), true); err != nil || res.Calls != 0 {
		t.Errorf("القاطع مفتوح ⇒ ولا نداء: %+v %v", res, err)
	}

	// المحاولة الجاية: بتنقسم قبل النداء ⇒ النص الأول بينجح والتاني مهلة ⇒ وقفة.
	later := t0.Add(10 * time.Minute)
	if _, err := w.matchChunk(ctx, ch, later, o, &tally{budget: 60}); !errors.Is(err, errBreaker) || sc.calls != 3 {
		t.Fatalf("التانية: خطأ %v، نداءات %d (لازم ٣: كاملة ثم نصّين)", err, sc.calls)
	}
	// التالتة: النص التاني (٢١ < ٣٠) بيتجرّب مرّة كمان ⇒ مهلة.
	if _, err := w.matchChunk(ctx, ch, later.Add(10*time.Minute), o, &tally{budget: 60}); !errors.Is(err, errBreaker) || sc.calls != 5 {
		t.Fatalf("التالتة: خطأ %v، نداءات %d", err, sc.calls)
	}
	// الرابعة: النص التاني خلصت مهلته مرّتين ⇒ `no_match('timeout')` بلا نداء.
	rows, err = w.matchChunk(ctx, ch, later.Add(20*time.Minute), o, &tally{budget: 60})
	if err != nil || sc.calls != 6 || len(rows) != 2 {
		t.Fatalf("الرابعة: خطأ %v، نداءات %d، سطور %d", err, sc.calls, len(rows))
	}
	if rows[0].Status != "ok" || rows[1].Status != "no_match" || rows[1].ErrorCode != "timeout" {
		t.Errorf("السطور: %s/%s %s/%s", rows[0].Status, rows[0].ErrorCode, rows[1].Status, rows[1].ErrorCode)
	}
	if !rows[0].ToAt.Equal(rows[1].FromAt) || !rows[0].FromAt.Equal(ch.From()) || !rows[1].ToAt.Equal(ch.To()) {
		t.Error("النصّين لازم يغطّوا القطعة بنقطة مشتركة")
	}
}

// RM8c (مراجعة ٢٠٢٦-١١-٠٦) — **المطابق بيترك مكان للملاحة**: سقف النداءات بالدورة ≤ ٢٠، وبعد كل نداء
// بيرتاح قدّ ما أخد النداء (فخيط Valhalla فاضي نص الوقت عالأقل).
func TestWorker_RestsBetweenCalls(t *testing.T) {
	if n := DefaultOptions().CallsPerTick; n > 20 {
		t.Errorf("سقف النداءات بالدورة %d — لازم ≤ ٢٠", n)
	}
	small := Chunk{Points: drive(0, 40, 4, 0, 8), Closed: true}
	small.Sent = make([]bool, len(small.Points))
	for i := range small.Sent {
		small.Sent[i] = true
	}
	sc := &script{fn: func(_ int, sent []drivermonitor.Point) (Answer, error) {
		time.Sleep(30 * time.Millisecond)
		return fakeAnswer(sent), nil
	}}
	var rests []time.Duration
	w := &Worker{Matcher: sc, Opt: DefaultOptions(), Rest: func(_ context.Context, d time.Duration) { rests = append(rests, d) }}
	if _, err := w.matchChunk(context.Background(), small, t0, w.opt(), &tally{budget: 60}); err != nil {
		t.Fatal(err)
	}
	if len(rests) != 1 || rests[0] < 30*time.Millisecond {
		t.Errorf("الراحة بعد نداء أخد ≥ ٣٠ م ث: %v", rests)
	}
}
