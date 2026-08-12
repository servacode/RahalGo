package routing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRouteReadsStreetsNotAir **يحرس ترتيبَ الإحداثيّات وقراءةَ الجواب.**
//
// **وOSRM يكتب الطولَ قبل العرض** — عكسَ القاعدة والشاشة. **ومن قلبها**
// سأل عن مسارٍ في بلدٍ آخر: نقطةُ الرقّة (35.95, 39.00) تصير (39.00, 35.95)
// **وهي في بحر العرب** — فيردّ «لا مسار» ولا شيءَ يقول لماذا.
func TestRouteReadsStreetsNotAir(t *testing.T) {
	var asked string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[{"distance":1123.4,"duration":421.0,
			"geometry":{"coordinates":[[39.0079,35.9528],[39.0100,35.9550]]}}]}`))
	}))
	defer srv.Close()

	got, err := New(srv.URL).Route(context.Background(),
		Point{Lat: 35.9528, Lng: 39.0079}, Point{Lat: 35.9550, Lng: 39.0100})
	if err != nil {
		t.Fatalf("مسار: %v", err)
	}

	// **والطولُ أوّلا في العنوان.**
	if !strings.Contains(asked, "39.007900,35.952800;39.010000,35.955000") {
		t.Fatalf("ترتيبُ الإحداثيّات خطأ: %s", asked)
	}
	if got.DistanceM != 1123.4 || got.DurationS != 421 {
		t.Fatalf("قراءةٌ خاطئة: %v", got)
	}
	// **والخطُّ يُقلب إلى عرضٍ ثمّ طول** — كما ترسمه الشاشة.
	if len(got.Geometry) != 2 || got.Geometry[0].Lat != 35.9528 || got.Geometry[0].Lng != 39.0079 {
		t.Fatalf("هندسةٌ مقلوبة: %v", got.Geometry)
	}
}

// TestNoRouteIsNotSuccess **«لا مسار» تأتي بحالة ٢٠٠** — ومن قرأ الحالةَ
// وحدَها ظنّ أنّه نجح ثمّ قسم على صفر.
func TestNoRouteIsNotSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":"NoRoute","routes":[]}`))
	}))
	defer srv.Close()

	if _, err := New(srv.URL).Route(context.Background(), Point{}, Point{}); err == nil {
		t.Fatal("«لا مسار» مرّت نجاحاً")
	}
}

// TestNoEngineIsQuiet **وبلا عنوانٍ لا يُنادى شيء** — ولا يسقط.
func TestNoEngineIsQuiet(t *testing.T) {
	c := New("")
	if c.Enabled() {
		t.Fatal("عنوانٌ فارغٌ عُدّ محرّكا")
	}
	if _, err := c.Route(context.Background(), Point{}, Point{}); err != ErrNoEngine {
		t.Fatalf("خطأٌ غيرُ متوقّع: %v", err)
	}
}
