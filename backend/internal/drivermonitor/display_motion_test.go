package drivermonitor

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

// **مسارات حقيقية الشكل لنقطة العرض** (قرار المالك ٢٠٢٦-١١-١٨: «السائق الواقف عم يبين ماشي»).
//
// جوّا البيت الجوال بدقة ٣٠–٥٠ م بيرجف ٢٠–٤٠ م وسرعته عشوائية — لازم ما يتحرّك. والمشي الحقيقي
// (سيارة كل ٤ ث، أو زحف بالزحمة ١–٢ م/ث) لازم يتحرّك.

// fixNE نقطة بعد `sec` ثانية من displayT0، مزاحة `north` و`east` متر.
func fixNE(sec float64, north, east float64, acc, speed *float64) Point {
	return Point{At: displayT0.Add(time.Duration(sec * float64(time.Second))),
		Lat: 35.95 + north/111195, Lng: 39.0 + east/(111195*math.Cos(35.95*math.Pi/180)),
		AccuracyM: acc, SpeedMps: speed}
}

// L-T8 — واقف جوّا بيت: دقة ٣٠–٥٠ م، رجفة لحد ٤٠ م، وسرعات عشوائية فيها قفزات لـ٣ م/ث
// (ما بتجي مرتين ورا بعض). **والموضع المعروض ما لازم يتحرّك ولا مرة** خلال خمس دقايق.
//
// على الشيفرة القديمة (عتبة max(٨، نص الدقة) والسرعة < ١) بيسقط: رجفة ٣٠ م بسرعة ٠٫٤ كانت حركة.
func TestNextShown_IndoorJitterNeverMoves(t *testing.T) {
	cfg := DefaultConfig()
	r := rand.New(rand.NewSource(20261118))
	s := NextShown(Shown{}, fixNE(0, 0, 0, f64(40), f64(0)), cfg)
	home := Point{Lat: s.Lat, Lng: s.Lng}
	moves, speedy := 0, 0
	for i := 1; i <= 60; i++ { // كل ٥ ثواني، خمس دقايق
		acc := 30 + r.Float64()*20
		ang := r.Float64() * 2 * math.Pi
		rad := (0.4 + 0.4*r.Float64()) * acc // ١٢–٤٠ م
		if rad > 40 {
			rad = 40
		}
		spd := r.Float64() * 0.9
		if i%4 == 0 { // قفزة سرعة وحدة كل أربع نقاط
			spd = 2 + r.Float64()
		}
		s = NextShown(s, fixNE(float64(5*i), rad*math.Cos(ang), rad*math.Sin(ang), f64(acc), f64(spd)), cfg)
		if d := Dist(home, Point{Lat: s.Lat, Lng: s.Lng}); d > 0.01 {
			moves++
			home = Point{Lat: s.Lat, Lng: s.Lng}
		}
		if !s.At.Equal(displayT0.Add(time.Duration(5*i) * time.Second)) {
			t.Fatalf("النقطة %d ما حدّثت وقت آخر ظهور", i)
		}
		if s.SpeedMps != nil && *s.SpeedMps != 0 {
			speedy++
		}
	}
	if moves > 0 {
		t.Fatalf("سائق واقف جوّا بيت انعرض ماشي %d مرة من ٦٠", moves)
	}
	if speedy > 0 {
		t.Fatalf("موضع ممسوك انبثّ بسرعة %d مرة — بيطلع سهم الاتجاه على واقف", speedy)
	}
}

// L-T9 — مشوار حقيقي بالسيارة: ٨ م/ث (~٢٩ كم/س)، نقطة كل ٤ ث، دقة ٨–١٥ م ورجفة لحد ٥ م.
// **والمعروض لازم يمشي معه** — كل نقطة تقريباً بتحرّكه وما بيتأخّر أكتر من ٣٠ م.
func TestNextShown_RealRideMoves(t *testing.T) {
	cfg := DefaultConfig()
	r := rand.New(rand.NewSource(7))
	s := NextShown(Shown{}, fixNE(0, 0, 0, f64(10), f64(8)), cfg)
	last := Point{Lat: s.Lat, Lng: s.Lng}
	moves := 0
	const n = 45 // ثلاث دقايق
	for i := 1; i <= n; i++ {
		truth := 8.0 * 4 * float64(i)
		fx := fixNE(float64(4*i), truth+(r.Float64()*10-5), r.Float64()*10-5, f64(8+r.Float64()*7), f64(7+r.Float64()*2))
		s = NextShown(s, fx, cfg)
		cur := Point{Lat: s.Lat, Lng: s.Lng}
		if Dist(last, cur) > 0.01 {
			moves++
		}
		last = cur
		if d := Dist(cur, fixNE(0, truth, 0, nil, nil)); d > 30 {
			t.Fatalf("النقطة %d: المعروض متأخّر %.0f م عن السيارة", i, d)
		}
	}
	if moves < n-2 {
		t.Fatalf("السيارة تحرّكت %d مرة بس من %d", moves, n)
	}
}

// L-T10 — زحف بالزحمة: ١٫٥ م/ث، نقطة كل ٥ ث، دقة ١٠ م، والجهاز بيقول ١–١٫٨ م/ث (تحت عتبة
// السرعة). **الإزاحة بتتجمّع من الموضع الممسوك** لحد ما تقطع ٢٠ م فبيتحرّك — ما بيعلق.
func TestNextShown_SlowCrawlEventuallyMoves(t *testing.T) {
	cfg := DefaultConfig()
	s := NextShown(Shown{}, fixNE(0, 0, 0, f64(10), f64(1.5)), cfg)
	start := Point{Lat: s.Lat, Lng: s.Lng}
	firstMove := -1
	for i := 1; i <= 24; i++ { // دقيقتين
		truth := 1.5 * 5 * float64(i)
		spd := 1.0 + 0.8*float64(i%3)/2
		s = NextShown(s, fixNE(float64(5*i), truth, 0, f64(10), f64(spd)), cfg)
		cur := Point{Lat: s.Lat, Lng: s.Lng}
		if firstMove < 0 && Dist(start, cur) > 0.01 {
			firstMove = i
		}
		if d := Dist(cur, fixNE(0, truth, 0, nil, nil)); d > ShownMoveMinM+0.5 {
			t.Fatalf("النقطة %d: الزاحف متأخّر %.1f م — أكتر من عتبة الحركة", i, d)
		}
	}
	if firstMove < 0 || firstMove > 3 {
		t.Fatalf("الزاحف بـ١٫٥ م/ث لازم يتحرّك بالنقطة التالتة (٢٢٫٥ م)، تحرّك بـ%d", firstMove)
	}
	// وبلا سرعة من الجهاز ودقة ١٥ م: بس الإزاحة — ١ م/ث كل ٥ ث بيتحرّك بعد ٢٠ م.
	s = NextShown(Shown{}, fixNE(0, 0, 0, f64(15), nil), cfg)
	start = Point{Lat: s.Lat, Lng: s.Lng}
	moved := false
	for i := 1; i <= 8; i++ {
		s = NextShown(s, fixNE(float64(5*i), 5*float64(i), 0, f64(15), nil), cfg)
		if Dist(start, Point{Lat: s.Lat, Lng: s.Lng}) > 0.01 {
			moved = true
			if i != 5 {
				t.Fatalf("بلا سرعة لازم يتحرّك أول ما تقطع الإزاحة ٢٠ م (النقطة ٥)، تحرّك بـ%d", i)
			}
			break
		}
	}
	if !moved {
		t.Fatal("زحف بلا سرعة ما تحرّك أبداً")
	}
}

// L-T11 — **السرعة على نقطتين ورا بعض**: نقطة وحدة سريعة ما بتحرّك، والتانية بتحرّك (إذا
// في إزاحة فعلية فوق ٨ م)؛ وسرعتان عشوائيتان بنفس المكان ما بيحرّكوه.
func TestNextShown_SpeedNeedsTwoFixes(t *testing.T) {
	cfg := DefaultConfig()
	s := NextShown(Shown{}, fixNE(0, 0, 0, f64(10), f64(0)), cfg)
	home := s.Lat
	if s = NextShown(s, fixNE(3, 9, 0, f64(10), f64(3)), cfg); s.Lat != home {
		t.Fatalf("نقطة سريعة وحدة حرّكته: %+v", s)
	}
	if s = NextShown(s, fixNE(6, 18, 0, f64(10), f64(3)), cfg); s.Lat == home {
		t.Fatalf("نقطتين سريعتين ورا بعض وإزاحة ١٨ م ما حرّكوه: %+v", s)
	}
	// سرعتان عشوائيتان ورا بعض بس الإزاحة ٥ م: واقف.
	s = NextShown(Shown{}, fixNE(0, 0, 0, f64(10), f64(0)), cfg)
	home = s.Lat
	s = NextShown(s, fixNE(5, 4, 0, f64(10), f64(2.5)), cfg)
	s = NextShown(s, fixNE(10, -5, 0, f64(10), f64(2.5)), cfg)
	if s.Lat != home {
		t.Fatalf("سرعتان عشوائيتان بنفس المكان حرّكوه: %+v", s)
	}
}
