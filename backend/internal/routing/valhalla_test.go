package routing

// **Valhalla محرّكاً بديلاً** — طلبُ المالك ٢٠٢٦-١٠-٠٢: الموتور يدخل كلّ
// الطرق مو سيارة.
//
// والتثبيتُ `testdata/valhalla_raqqa_route.json` ردٌّ حقيقيٌّ سُجّل من
// Valhalla 3.9.0 (سوريا) ٢٠٢٦-١٠-٠٢: الرقّة 35.9607,39.0140 → 35.9513,39.0117
// بتكلفة `motor_scooter` و`alternates: 2`.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func f64(v float64) *float64 { return &v }

// captureValhalla خادمٌ مزيّفٌ يحفظ الطلبَ ويردّ ما أُعطي.
func captureValhalla(t *testing.T, status int, reply []byte) (*httptest.Server, *map[string]any, *string) {
	t.Helper()
	got := map[string]any{}
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.Method + " " + r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(reply)
	}))
	t.Cleanup(srv.Close)
	return srv, &got, &path
}

func raqqaFixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/valhalla_raqqa_route.json")
	if err != nil {
		t.Fatalf("التثبيت: %v", err)
	}
	return raw
}

var (
	raqqaFrom = Point{Lat: 35.9607078, Lng: 39.0140076}
	raqqaTo   = Point{Lat: 35.9513478, Lng: 39.0116516}
)

// TestValhallaRequestCostsAMotorcycle **التكلفةُ والصيغةُ وحدُّ الالتقاط.**
func TestValhallaRequestCostsAMotorcycle(t *testing.T) {
	srv, got, path := captureValhalla(t, 200, raqqaFixture(t))
	if _, err := NewValhalla(srv.URL).Route(context.Background(), raqqaFrom, raqqaTo); err != nil {
		t.Fatalf("مسار: %v", err)
	}
	if *path != "POST /route" {
		t.Fatalf("النداء: %q", *path)
	}
	body := *got
	if body["costing"] != "motor_scooter" || body["format"] != "osrm" || body["shape_format"] != "geojson" {
		t.Fatalf("تكلفةٌ أو صيغةٌ خاطئة: %v", body)
	}
	opts := body["costing_options"].(map[string]any)["motor_scooter"].(map[string]any)
	want := map[string]float64{
		"top_speed": 50, "use_primary": 0.2, "use_living_streets": 1.0,
		"use_tracks": 0.5, "use_distance": 0.5, "service_penalty": 0, "service_factor": 1,
	}
	for k, v := range want {
		if opts[k] != v {
			t.Errorf("%s = %v، والمطلوب %v", k, opts[k], v)
		}
	}
	// **ولا عكسَ للاتّجاه أبداً** — بلاغُ المالك ٢٠٢٦-١٠-٠٢: جانبا الطريق المفصول
	// كلٌّ منهما باتّجاهٍ واحد، فعكسُه سيرٌ عكسَ السير.
	if v, ok := opts["ignore_oneways"]; ok && v != false {
		t.Errorf("ignore_oneways = %v — السيرُ عكسَ السير ممنوع", v)
	}
	if body["directions_options"].(map[string]any)["units"] != "kilometers" {
		t.Errorf("الوحدات: %v", body["directions_options"])
	}
	if _, ok := body["alternates"]; ok {
		t.Errorf("المفردُ طلب بدائل: %v", body["alternates"])
	}
	locs := body["locations"].([]any)
	o, d := locs[0].(map[string]any), locs[1].(map[string]any)
	// **العرضُ عرضٌ والطولُ طول** — Valhalla يسمّيهما ولا يرتّبهما.
	if o["lat"] != raqqaFrom.Lat || o["lon"] != raqqaFrom.Lng || d["lat"] != raqqaTo.Lat {
		t.Fatalf("إحداثيّاتٌ مقلوبة: %v", locs)
	}
	if o["search_cutoff"] != 150.0 || d["search_cutoff"] != 250.0 {
		t.Fatalf("بلا حدِّ التقاط: %v", locs)
	}
	if _, ok := o["heading"]; ok {
		t.Fatalf("اتّجاهٌ لم يُعطَ أُرسل: %v", o)
	}
}

// TestValhallaHeadingOnOriginOnly **الاتّجاهُ على الأصل وحدَه بسماحة ٦٠°.**
func TestValhallaHeadingOnOriginOnly(t *testing.T) {
	srv, got, _ := captureValhalla(t, 200, raqqaFixture(t))
	from := raqqaFrom
	from.Bearing = f64(360)
	if _, err := NewValhalla(srv.URL).Route(context.Background(), from, raqqaTo); err != nil {
		t.Fatalf("مسار: %v", err)
	}
	locs := (*got)["locations"].([]any)
	o, d := locs[0].(map[string]any), locs[1].(map[string]any)
	if o["heading"] != 0.0 || o["heading_tolerance"] != 60.0 {
		t.Fatalf("اتّجاهُ الأصل: %v", o)
	}
	if _, ok := d["heading"]; ok {
		t.Fatalf("الوجهةُ حملت اتّجاهاً: %v", d)
	}
}

// TestValhallaRouteSetAsksAlternates **والبدائلُ تُطلب بعدد OSRM.**
func TestValhallaRouteSetAsksAlternates(t *testing.T) {
	srv, got, _ := captureValhalla(t, 200, raqqaFixture(t))
	routes, err := NewValhalla(srv.URL).RouteSet(context.Background(), raqqaFrom, raqqaTo)
	if err != nil {
		t.Fatalf("مجموعة: %v", err)
	}
	if (*got)["alternates"] != float64(EngineMaxAlternatives) {
		t.Fatalf("alternates = %v", (*got)["alternates"])
	}
	if len(routes) != 2 {
		t.Fatalf("التثبيتُ فيه مساران، وقُرئ %d", len(routes))
	}
	// **وترتيبُ المحرّك محفوظ.**
	if routes[0].DistanceM != 1331.802 || routes[1].DistanceM != 1386.803 {
		t.Fatalf("ترتيبٌ أو قراءةٌ خاطئة: %v %v", routes[0].DistanceM, routes[1].DistanceM)
	}
}

// TestValhallaFixtureParsesWithOSRMReader **الردُّ الحقيقيُّ يُقرأ بقارئ OSRM.**
func TestValhallaFixtureParsesWithOSRMReader(t *testing.T) {
	srv, _, _ := captureValhalla(t, 200, raqqaFixture(t))
	r, err := NewValhalla(srv.URL).Route(context.Background(), raqqaFrom, raqqaTo)
	if err != nil {
		t.Fatalf("مسار: %v", err)
	}
	if r.DistanceM != 1331.802 || r.DurationS != 132.313 {
		t.Fatalf("مسافةٌ أو مدّة: %v %v", r.DistanceM, r.DurationS)
	}
	if r.WeightName != "motor_scooter" || r.EngineWeight != 326.031 {
		t.Fatalf("الوزن: %v %v", r.WeightName, r.EngineWeight)
	}
	if !r.HasNavigation() || len(r.Geometry) < 2 {
		t.Fatalf("بلا ملاحة: %d مناورة، %d نقطة", len(r.Maneuvers), len(r.Geometry))
	}
	// **والهندسةُ عرضٌ ثمّ طول** — تبدأ قربَ الأصل.
	if MetersBetween(r.Geometry[0], raqqaFrom) > 50 {
		t.Fatalf("الهندسةُ لا تبدأ عند الأصل: %v", r.Geometry[0])
	}
	if last := r.CumulativeM[len(r.CumulativeM)-1]; last < 1200 || last > 1450 {
		t.Fatalf("الطولُ التراكميّ %v بعيدٌ عن ١٣٣٢م", last)
	}
}

// TestValhallaErrorsAreNamed **الرفضُ يُسمّى كما في OSRM.**
func TestValhallaErrorsAreNamed(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"osrm-nosegment", 400, `{"code":"NoSegment","message":"x"}`, ErrNoSegment},
		{"osrm-noroute", 400, `{"code":"NoRoute","message":"x"}`, ErrNoRoute},
		{"171", 400, `{"error_code":171,"error":"No suitable edges near location"}`, ErrNoSegment},
		{"442", 400, `{"error_code":442,"error":"No path could be found for input"}`, ErrNoRoute},
	}
	for _, c := range cases {
		srv, _, _ := captureValhalla(t, c.status, []byte(c.body))
		_, err := NewValhalla(srv.URL).Route(context.Background(), raqqaFrom, raqqaTo)
		if !errors.Is(err, c.want) {
			t.Errorf("%s: %v، والمطلوب %v", c.name, err, c.want)
		}
	}
	srv, _, _ := captureValhalla(t, 500, []byte(`oops`))
	_, err := NewValhalla(srv.URL).Route(context.Background(), raqqaFrom, raqqaTo)
	if err == nil || errors.Is(err, ErrNoSegment) || errors.Is(err, ErrNoRoute) {
		t.Fatalf("العطبُ سُمّي رفضاً: %v", err)
	}
}

// TestValhallaEmptyIsNoEngine **وبلا عنوانٍ لا يُنادى شيء.**
func TestValhallaEmptyIsNoEngine(t *testing.T) {
	v := NewValhalla("")
	if v.Enabled() {
		t.Fatal("عنوانٌ فارغٌ عُدّ محرّكاً")
	}
	if _, err := v.RouteSet(context.Background(), raqqaFrom, raqqaTo); err != ErrNoEngine {
		t.Fatalf("%v", err)
	}
}

// TestOSRMSendsOriginBearing **وOSRM يُرسل اتّجاهَ الأصل حين يُعرف.**
func TestOSRMSendsOriginBearing(t *testing.T) {
	var raw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[{"distance":1,"duration":1,
			"geometry":{"coordinates":[[39.01,35.95],[39.02,35.96]]}}]}`))
	}))
	defer srv.Close()

	from := Point{Lat: 35.95, Lng: 39.01, Bearing: f64(92.6)}
	if _, err := New(srv.URL).Route(context.Background(), from, Point{Lat: 35.96, Lng: 39.02}); err != nil {
		t.Fatalf("مسار: %v", err)
	}
	if !strings.Contains(raw, "bearings=93,45%3B") {
		t.Fatalf("بلا اتّجاه: %q", raw)
	}

	from.Bearing = nil
	if _, err := New(srv.URL).Route(context.Background(), from, Point{Lat: 35.96, Lng: 39.02}); err != nil {
		t.Fatalf("مسار: %v", err)
	}
	if strings.Contains(raw, "bearings") {
		t.Fatalf("اتّجاهٌ لم يُعطَ أُرسل: %q", raw)
	}
}

// TestLiveValhallaRaqqa **نداءٌ حيّ** — يُتخطّى بلا `VALHALLA_URL`.
func TestLiveValhallaRaqqa(t *testing.T) {
	base := os.Getenv("VALHALLA_URL")
	if base == "" {
		t.Skip("لا محرّك")
	}
	v := NewValhalla(base)
	routes, err := v.RouteSet(context.Background(), raqqaFrom, raqqaTo)
	if err != nil {
		t.Fatalf("مجموعة: %v", err)
	}
	for i, r := range routes {
		t.Logf("[%d] %.1fم %.1fث وزن=%.1f مناورات=%d نقاط=%d",
			i, r.DistanceM, r.DurationS, r.EngineWeight, len(r.Maneuvers), len(r.Geometry))
		if !r.HasNavigation() {
			t.Errorf("[%d] بلا ملاحة", i)
		}
	}
	// **والاتّجاهُ يُقبل حيّاً** — جنوباً كما يسير المسار.
	from := raqqaFrom
	from.Bearing = f64(180)
	if _, err := v.Route(context.Background(), from, raqqaTo); err != nil {
		t.Fatalf("باتّجاه: %v", err)
	}
	// **والبحرُ المتوسّطُ خارجَ الحدّ** — لا مسارٌ واثقٌ من إحداثيّةٍ فاسدة.
	_, err = v.Route(context.Background(), Point{Lat: 35.0, Lng: 34.0}, raqqaTo)
	if !errors.Is(err, ErrNoSegment) {
		t.Fatalf("البحرُ لم يُرفض: %v", err)
	}
}

// TestBearingMissFallsBackOnce **اتّجاهٌ لا يُطابق طريقاً يُسقَط مرّةً**
// — ولا يُعاد بلا اتّجاهٍ من لم يرسل اتّجاهاً.
func TestBearingMissFallsBackOnce(t *testing.T) {
	ok := raqqaFixture(t)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		raw, _ := io.ReadAll(r.Body)
		if strings.Contains(string(raw), "heading") || strings.Contains(r.URL.RawQuery, "bearings") {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"code":"NoSegment"}`))
			return
		}
		if r.Method == http.MethodGet && strings.Contains(r.URL.RawQuery, "unlimited") {
			t.Error("الحدُّ وُسّع")
		}
		_, _ = w.Write(ok)
	}))
	defer srv.Close()

	from := raqqaFrom
	from.Bearing = f64(90)
	for name, b := range map[string]Backend{"valhalla": NewValhalla(srv.URL), "osrm": New(srv.URL)} {
		calls = 0
		if _, err := b.Route(context.Background(), from, raqqaTo); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if calls != 2 {
			t.Fatalf("%s: %d نداء", name, calls)
		}
	}
	srv2, _, _ := captureValhalla(t, 400, []byte(`{"code":"NoSegment"}`))
	if _, err := NewValhalla(srv2.URL).Route(context.Background(), raqqaFrom, raqqaTo); !errors.Is(err, ErrNoSegment) {
		t.Fatalf("%v", err)
	}
}

// TestPointWithoutBearingSerializesAsBefore **والمخبأُ لا يتبدّل بلا اتّجاه.**
func TestPointWithoutBearingSerializesAsBefore(t *testing.T) {
	raw, _ := json.Marshal(Point{Lat: 1, Lng: 2})
	if string(raw) != `{"Lat":1,"Lng":2}` {
		t.Fatalf("صيغةُ النقطة تبدّلت: %s", raw)
	}
}
