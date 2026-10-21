package routing

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestValhalla_ExcludesClosedRoads **الطرقُ المسكَّرةُ تصل المحرّكَ ليتجنّبها** (قرارُ المالك
// ٢٠٢٦-١٠-٢١) — وبلا مصدرٍ لا حقلَ في الطلب.
func TestValhalla_ExcludesClosedRoads(t *testing.T) {
	var got map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var one map[string]any
		_ = json.Unmarshal(raw, &one)
		// **أوّلُ طلبٍ فيه الاستثناء** — والإعادةُ بدونه لا تمحو ما رأيناه.
		if got == nil || one["exclude_locations"] != nil {
			got = one
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"NoRoute"}`))
	}))
	defer ts.Close()

	v := NewValhalla(ts.URL)
	_, _ = v.Route(context.Background(), Point{Lat: 35.95, Lng: 39.0}, Point{Lat: 35.96, Lng: 39.01})
	if _, ok := got["exclude_locations"]; ok {
		t.Fatal("بلا مصدرٍ لا يُرسل exclude_locations")
	}
	got = nil
	v.SetExcluder(func(context.Context, Point, Point) []Point { return []Point{{Lat: 35.955, Lng: 39.005}} })
	_, _ = v.Route(context.Background(), Point{Lat: 35.95, Lng: 39.0}, Point{Lat: 35.96, Lng: 39.01})
	ex, _ := got["exclude_locations"].([]any)
	if len(ex) != 1 {
		t.Fatalf("الطريقُ المسكَّرُ لم يصل المحرّك: %v", got["exclude_locations"])
	}
}

// TestValhalla_ClosureNeverKillsRouting **والطريقُ المسكَّرُ لا يقتل التوجيه** (شُهد على الجهاز
// ٢٠٢٦-١٠-٢١: بلاغٌ عند السائق ⇒ «لا طريق» ⇒ خطٌّ مستقيم) — لا طريقَ يتجنّبه ⇒ يُعاد بدونه.
func TestValhalla_ClosureNeverKillsRouting(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		raw, _ := io.ReadAll(r.Body)
		var one map[string]any
		_ = json.Unmarshal(raw, &one)
		if one["exclude_locations"] != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":"NoRoute"}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[{"distance":500,"duration":60,` +
			`"geometry":{"type":"LineString","coordinates":[[39.0,35.95],[39.01,35.96]]},"legs":[]}]}`))
	}))
	defer ts.Close()

	v := NewValhalla(ts.URL)
	v.SetExcluder(func(context.Context, Point, Point) []Point { return []Point{{Lat: 35.95, Lng: 39.0}} })
	rt, err := v.Route(context.Background(), Point{Lat: 35.95, Lng: 39.0}, Point{Lat: 35.96, Lng: 39.01})
	if err != nil || rt == nil {
		t.Fatalf("مسكَّرٌ عند البداية قتل التوجيه: %v (نداءات %d)", err, calls)
	}
	if calls != 2 {
		t.Fatalf("المتوقَّع نداءان (مع الاستثناء ثمّ بدونه)، صار %d", calls)
	}
}
