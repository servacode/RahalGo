package server

// **اتّجاهُ السائق ومفتاحُ المخبأ بمحرّكه** — طلبُ المالك ٢٠٢٦-١٠-٠٢:
// الموتور يدخل كلّ الطرق مو سيارة.

import (
	"net/http/httptest"
	"testing"
)

func TestClientHeadingIgnoresInvalid(t *testing.T) {
	for raw, want := range map[string]float64{"0": 0, "92.5": 92.5, "360": 360} {
		r := httptest.NewRequest("GET", "/x?heading="+raw, nil)
		got := clientHeading(r)
		if got == nil || *got != want {
			t.Errorf("heading=%s → %v", raw, got)
		}
	}
	for _, raw := range []string{"", "abc", "-1", "361", "NaN", "Inf"} {
		r := httptest.NewRequest("GET", "/x?heading="+raw, nil)
		if got := clientHeading(r); got != nil {
			t.Errorf("heading=%q قُبل: %v", raw, *got)
		}
	}
	if clientHeading(httptest.NewRequest("GET", "/x", nil)) != nil {
		t.Error("بلا معاملٍ صار اتّجاهاً")
	}
}

func TestHeadingCellSplitsOpposites(t *testing.T) {
	h := func(v float64) *float64 { return &v }
	if headingCell(nil) != "" {
		t.Fatal("بلا اتّجاهٍ تبدّل المفتاح")
	}
	if headingCell(h(350)) != headingCell(h(10)) || headingCell(h(0)) != headingCell(h(360)) {
		t.Error("اتّجاهان متقاربان افترقا")
	}
	if headingCell(h(0)) == headingCell(h(180)) {
		t.Error("المتعاكسان تشاركا الجواب")
	}
}

func TestRouteKeyPrefixKeepsOSRMKeys(t *testing.T) {
	for engine, want := range map[string]string{"": "", "osrm": "", "valhalla": "valhalla:"} {
		if got := (&Server{routeEngine: engine}).routeKeyPrefix(); got != want {
			t.Errorf("%q → %q", engine, got)
		}
	}
}
