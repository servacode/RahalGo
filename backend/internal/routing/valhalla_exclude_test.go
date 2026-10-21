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
		_ = json.Unmarshal(raw, &got)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"NoRoute"}`))
	}))
	defer ts.Close()

	v := NewValhalla(ts.URL)
	_, _ = v.Route(context.Background(), Point{Lat: 35.95, Lng: 39.0}, Point{Lat: 35.96, Lng: 39.01})
	if _, ok := got["exclude_locations"]; ok {
		t.Fatal("بلا مصدرٍ لا يُرسل exclude_locations")
	}
	v.SetExcluder(func(context.Context, Point, Point) []Point { return []Point{{Lat: 35.955, Lng: 39.005}} })
	_, _ = v.Route(context.Background(), Point{Lat: 35.95, Lng: 39.0}, Point{Lat: 35.96, Lng: 39.01})
	ex, _ := got["exclude_locations"].([]any)
	if len(ex) != 1 {
		t.Fatalf("الطريقُ المسكَّرُ لم يصل المحرّك: %v", got["exclude_locations"])
	}
}
