package maplearn

import (
	"math"
	"math/rand"
	"strings"
	"testing"
)

const (
	lat0 = 35.95
	lng0 = 39.00
)

func mLng() float64 { return mPerDegLat * math.Cos(lat0*math.Pi/180) }

// walk نقاطٌ على قطعةٍ مستقيمةٍ من (x0,y0) إلى (x1,y1) بالمتر، كلّ stepM، بتشويشٍ ±jitter، في يومٍ.
func walk(r *rand.Rand, x0, y0, x1, y1, stepM, jitter float64, day string) []Point {
	n := int(math.Hypot(x1-x0, y1-y0) / stepM)
	var out []Point
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x := x0 + (x1-x0)*t + (r.Float64()*2-1)*jitter
		y := y0 + (y1-y0)*t + (r.Float64()*2-1)*jitter
		out = append(out, Point{Lat: lat0 + y/mPerDegLat, Lng: lng0 + x/mLng(), Day: day})
	}
	return out
}

func TestDetect_StraightStreetOnThreeDays(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	var pts []Point
	for _, d := range []string{"2026-10-01", "2026-10-02", "2026-10-05"} {
		pts = append(pts, walk(r, 0, 0, 240, 0, 9, 4, d)...)
	}
	got := Detect(pts, DefaultParams())
	if len(got) != 1 {
		t.Fatalf("متوقّع خطٌّ واحد، وجدتُ %d", len(got))
	}
	c := got[0]
	if c.Days != 3 || c.FirstDay != "2026-10-01" || c.LastDay != "2026-10-05" {
		t.Fatalf("الأيّام: %+v", c)
	}
	if c.LengthM < 180 || c.LengthM > 270 {
		t.Fatalf("الطول %.0f خارج المتوقّع حول ٢٤٠", c.LengthM)
	}
	if len(c.Line) < 2 {
		t.Fatal("خطٌّ بلا رأسين")
	}
}

func TestDetect_OneDayOnlyIsNotCandidate(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	pts := walk(r, 0, 0, 240, 0, 6, 3, "2026-10-01")
	pts = append(pts, walk(r, 0, 0, 240, 0, 6, 3, "2026-10-01")...)
	if got := Detect(pts, DefaultParams()); len(got) != 0 {
		t.Fatalf("يومٌ واحدٌ لا يكفي: %d", len(got))
	}
}

func TestDetect_BlobAndScatterRejected(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	var pts []Point
	// ساحةٌ ~٣٠ م عبر ثلاثة أيام: كثيرةُ النقاط لكنّها ليست طريقاً.
	for _, d := range []string{"2026-10-01", "2026-10-02", "2026-10-03"} {
		for i := 0; i < 25; i++ {
			x, y := 1000+(r.Float64()*2-1)*15, 1000+(r.Float64()*2-1)*15
			pts = append(pts, Point{Lat: lat0 + y/mPerDegLat, Lng: lng0 + x/mLng(), Day: d})
		}
	}
	// ونقاطٌ شاردةٌ متفرّقة كلُّ واحدةٍ وحدَها.
	for i := 0; i < 40; i++ {
		x, y := 3000+float64(i)*400, 3000+float64(i)*300
		pts = append(pts, Point{Lat: lat0 + y/mPerDegLat, Lng: lng0 + x/mLng(), Day: "2026-10-0" + string(rune('1'+i%3))})
	}
	if got := Detect(pts, DefaultParams()); len(got) != 0 {
		t.Fatalf("لا يُقترَح شيءٌ هنا: %+v", got)
	}
}

func TestDetect_CurvedAlleyFollowsTheTurn(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	var pts []Point
	for _, d := range []string{"2026-10-01", "2026-10-02"} {
		pts = append(pts, walk(r, 0, 0, 150, 0, 8, 3, d)...)
		pts = append(pts, walk(r, 150, 8, 150, 160, 8, 3, d)...)
	}
	got := Detect(pts, DefaultParams())
	if len(got) != 1 {
		t.Fatalf("متوقّع خطٌّ واحد: %d", len(got))
	}
	// الخطّ يلتفّ: طولُه قريبٌ من ٣١٠ لا من وترٍ ٢٢٠.
	if got[0].LengthM < 250 {
		t.Fatalf("الطول %.0f — الخطّ لم يتبع المنعطف", got[0].LengthM)
	}
}

func TestDetect_TwoSeparateStreets(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	var pts []Point
	for _, d := range []string{"2026-10-01", "2026-10-02"} {
		pts = append(pts, walk(r, 0, 0, 200, 0, 8, 3, d)...)
		pts = append(pts, walk(r, 0, 600, 200, 600, 8, 3, d)...)
	}
	if got := Detect(pts, DefaultParams()); len(got) != 2 {
		t.Fatalf("متوقّع خطّان: %d", len(got))
	}
}

func TestDetect_DeterministicAndEmpty(t *testing.T) {
	if Detect(nil, DefaultParams()) != nil {
		t.Fatal("بلا نقاطٍ بلا خطوط")
	}
	r := rand.New(rand.NewSource(6))
	var pts []Point
	for _, d := range []string{"2026-10-01", "2026-10-02"} {
		pts = append(pts, walk(r, 0, 0, 200, 40, 8, 3, d)...)
	}
	a, b := Detect(pts, DefaultParams()), Detect(pts, DefaultParams())
	if len(a) != 1 || len(a) != len(b) || len(a[0].Line) != len(b[0].Line) || a[0].LengthM != b[0].LengthM {
		t.Fatal("النتيجة غير ثابتة بين جولتين")
	}
}

func TestExport_OSMAndGeoJSON(t *testing.T) {
	lines := []Line{{ID: 7, Name: "حارة <الأمل> & ", Coords: [][2]float64{{39.0, 35.95}, {39.001, 35.951}, {39.002, 35.951}}},
		{ID: 8, Coords: [][2]float64{{39.0, 35.95}}}} // نقطةٌ واحدة تُهمل
	x := OSM(lines)
	for _, want := range []string{"<node id=\"25000000001\"", "<way id=\"16000000001\"", "highway", "residential",
		"rahalgo:candidate_id\" v=\"7\"", "&lt;الأمل&gt; &amp;"} {
		if !strings.Contains(x, want) {
			t.Fatalf("ينقص %q في:\n%s", want, x)
		}
	}
	if strings.Contains(x, "16000000002") {
		t.Fatal("خطٌّ بنقطةٍ واحدةٍ دخل التصدير")
	}
	if strings.Index(x, "<way") < strings.LastIndex(x, "<node") {
		t.Fatal("الطرقُ يجب أن تلي كلَّ العقد")
	}
	j, err := GeoJSON(lines)
	if err != nil || !strings.Contains(string(j), "LineString") {
		t.Fatalf("geojson: %v %s", err, j)
	}
}
