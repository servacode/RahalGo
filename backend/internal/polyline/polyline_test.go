package polyline

import (
	"math"
	"testing"
)

// نُقل من `maplearn` مع المفكّك.
func TestPolyline6_RoundTrip(t *testing.T) {
	in := [][2]float64{{38.400000, 35.400000}, {38.401234, 35.400567}, {38.399000, 35.402000}}
	out := Decode6(Encode6(in))
	if len(out) != len(in) {
		t.Fatalf("الطول %d", len(out))
	}
	for i := range in {
		if math.Abs(in[i][0]-out[i][0]) > 1e-6 || math.Abs(in[i][1]-out[i][1]) > 1e-6 {
			t.Errorf("النقطة %d: %v ≠ %v", i, out[i], in[i])
		}
	}
}

func TestProjectAndCut(t *testing.T) {
	// خطّ شرقاً ١٠٠٠ م تقريباً ثمّ شمالاً.
	line := [][2]float64{{38.40, 35.40}, {38.41, 35.40}, {38.41, 35.41}}
	cum := Cumulative(line)
	east := cum[1]
	if math.Abs(east-Meters(line[0], line[1])) > 1e-9 {
		t.Fatal("التراكم")
	}
	// نقطة فوق منتصف القطعة الأولى بـ ٢٠ م تقريباً.
	pos, d := Project(line, cum, [2]float64{38.405, 35.40018}, 0, 2)
	if math.Abs(pos-east/2) > 2 || math.Abs(d-20) > 1 {
		t.Errorf("الإسقاط: pos=%.1f d=%.1f", pos, d)
	}
	// مقصورة على القطعة الثانية (شمالاً): الإسقاط ٢٠ م بعد الزاوية.
	pos, _ = Project(line, cum, [2]float64{38.405, 35.40018}, 1, 2)
	if math.Abs(pos-(east+20)) > 1 {
		t.Errorf("الإسقاط المقصور: %.1f", pos)
	}
	cut := Cut(line, cum, east/2, east+100)
	if len(cut) != 3 || math.Abs(Length(cut)-(east/2+100)) > 0.5 {
		t.Errorf("القصّ: %v طول %.1f", cut, Length(cut))
	}
}
