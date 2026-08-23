package routing

import (
	"context"
	"os"
	"testing"
)

// TestLiveSelfSimilarity **المسارُ الحقيقيُّ مع نفسِه.**
func TestLiveSelfSimilarity(t *testing.T) {
	base := os.Getenv("OSRM_URL")
	if base == "" {
		t.Skip("لا محرّك")
	}
	c := New(base)
	cases := []struct {
		name     string
		from, to Point
	}{
		{"الرقّة→دمشق", Point{35.9500, 39.0050}, Point{33.5138, 36.2765}},
		{"الرقّة→ديرالزور", Point{35.9500, 39.0050}, Point{35.3350, 40.1400}},
		{"دمشق→اللاذقيّة", Point{33.5138, 36.2765}, Point{35.5200, 35.7900}},
		{"حلب→حمص", Point{36.2021, 37.1343}, Point{34.7308, 36.7090}},
	}
	for _, tc := range cases {
		all, err := c.RouteSet(context.Background(), tc.from, tc.to)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		for i, r := range all {
			s := SharedRatio(r.Geometry, r.Geometry)
			d := FirstDivergenceM(r.Geometry, r.Geometry)
			t.Logf("%-18s [%d] نقاط=%5d ذاتيّ=%.4f تفرّعٌ ذاتيّ=%.0f",
				tc.name, i, len(r.Geometry), s, d)
			if s < 0.99 {
				t.Errorf("%s[%d]: ذاتيّ %.4f", tc.name, i, s)
			}
			if d >= 0 {
				t.Errorf("%s[%d]: تفرّعٌ كاذبٌ عند %.0fم", tc.name, i, d)
			}
		}
	}
}
