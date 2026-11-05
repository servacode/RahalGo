package drivermonitor

import (
	"testing"
	"time"
)

// نقطة العرض والحالة الحيّة (L1b، طلب المالك ٢٠٢٦-١١-٠٥: «ما بدنا نطّات»).

var t0 = time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)

func f64(v float64) *float64 { return &v }

// fixAt نقطة بعد `sec` ثانية من t0، مزاحة `north` متر شمالاً (الدرجة ≈ ١١١٫٢ كم).
func fixAt(sec int, north float64, acc, speed *float64) Point {
	return Point{At: t0.Add(time.Duration(sec) * time.Second),
		Lat: 35.95 + north/111195, Lng: 39.0, AccuracyM: acc, SpeedMps: speed}
}

// L-T1
func TestNextShown_PoorAccuracyHeldThenReleased(t *testing.T) {
	cfg := DefaultConfig()
	s := NextShown(Shown{}, fixAt(0, 0, f64(5), f64(0)), cfg)
	if s.Rough || s.Lat != fixAt(0, 0, nil, nil).Lat {
		t.Fatalf("أول نقطة دقيقة ما انعرضت متل ما هي: %+v", s)
	}
	home := s.Lat
	// نقطة دقتها ٣٠٠ م وبعيدة ٣٠٠ م: الموضع بيضل، والوقت بيتحدّث (فبيضل «حي»).
	s = NextShown(s, fixAt(10, 300, f64(300), nil), cfg)
	if s.Lat != home || s.Rough || !s.At.Equal(t0.Add(10*time.Second)) {
		t.Fatalf("نقطة ضعيفة حرّكت السائق أو ما حدّثت وقته: %+v", s)
	}
	s = NextShown(s, fixAt(40, 280, f64(300), nil), cfg)
	if s.Lat != home {
		t.Fatalf("نقطة ضعيفة قبل الدقيقة حرّكته: %+v", s)
	}
	// دقيقة كاملة بلا نقطة دقيقة ⇒ منعرضها، باهتة.
	s = NextShown(s, fixAt(60, 300, f64(300), nil), cfg)
	if s.Lat == home || !s.Rough {
		t.Fatalf("بعد دقيقة من النقاط الضعيفة بس لازم تنعرض باهتة: %+v", s)
	}
	// وأول نقطة دقيقة بترجّع الموضع الصافي، ولو بعيدة عن الباهت.
	s = NextShown(s, fixAt(64, 0, f64(6), f64(0)), cfg)
	if s.Rough || s.Lat != home {
		t.Fatalf("نقطة دقيقة بعد الباهت ما انعرضت: %+v", s)
	}
}

// L-T2
func TestNextShown_StandingJitterKeepsPoint(t *testing.T) {
	cfg := DefaultConfig()
	s := NextShown(Shown{}, fixAt(0, 0, f64(10), f64(0)), cfg)
	home := s.Lat
	s = NextShown(s, fixAt(4, 7, f64(10), f64(0.2)), cfg)
	if s.Lat != home {
		t.Fatalf("رجفة ٧ م وهو واقف حرّكت السائق: %+v", s)
	}
	if !s.At.Equal(t0.Add(4*time.Second)) || !s.PosAt.Equal(t0.Add(4*time.Second)) {
		t.Fatalf("الموضع ما تحدّث وقته: %+v", s)
	}
	// بلا سرعة متل السرعة الصفر.
	if s = NextShown(s, fixAt(8, -6, f64(10), nil), cfg); s.Lat != home {
		t.Fatalf("رجفة بلا سرعة حرّكته: %+v", s)
	}
	// ونفس المسافة وهو ماشي (٥ م/ث) حركة حقيقية.
	if s = NextShown(s, fixAt(12, 7, f64(10), f64(5)), cfg); s.Lat == home {
		t.Fatalf("حركة بسرعة ٥ م/ث انمسكت كرجفة: %+v", s)
	}
}

// L-T3
func TestNextShown_ImpossibleHopHeld30s(t *testing.T) {
	cfg := DefaultConfig()
	s := NextShown(Shown{}, fixAt(0, 0, f64(5), f64(8)), cfg)
	home := s.Lat
	// ٢ كم بـ١٠ ثواني (٧٢٠ كم/س): بتنمسك.
	s = NextShown(s, fixAt(10, 2000, f64(5), f64(8)), cfg)
	if s.Lat != home {
		t.Fatalf("قفزة ٢ كم بعشر ثواني انعرضت: %+v", s)
	}
	s = NextShown(s, fixAt(25, 2000, f64(5), f64(8)), cfg)
	if s.Lat != home {
		t.Fatalf("القفزة انقبلت قبل ٣٠ ثانية: %+v", s)
	}
	// ٣٠ ثانية من أول قفزة ⇒ بتنقبل: موضع معروض غلط ما لازم يعلّقه.
	s = NextShown(s, fixAt(40, 2000, f64(5), f64(8)), cfg)
	if s.Lat == home {
		t.Fatalf("القفزة ما انقبلت بعد ٣٠ ثانية — السائق معلّق بموضع قديم: %+v", s)
	}
	// ونقطة شاذّة وحدة بترجع لمكانها ما بتحرّك شي.
	s2 := NextShown(Shown{}, fixAt(0, 0, f64(5), f64(8)), cfg)
	s2 = NextShown(s2, fixAt(4, 1500, f64(5), f64(8)), cfg)
	s2 = NextShown(s2, fixAt(8, 30, f64(5), f64(8)), cfg)
	if d := Dist(Point{Lat: s2.Lat, Lng: s2.Lng}, fixAt(8, 30, nil, nil)); d > 1 {
		t.Fatalf("بعد النقطة الشاذّة ما رجع للحركة العادية: %.0f م", d)
	}
	if !s2.hopSince.IsZero() {
		t.Fatal("القفزة الممسوكة ما انمسحت بعد ما رجع")
	}
}

// L-T7
func TestDeriveLive_BadAccuracyIsNotMoving(t *testing.T) {
	now := t0.Add(10 * time.Minute)
	at := func(ago time.Duration, north, acc float64) Point {
		return Point{At: now.Add(-ago), Lat: 35.95 + north/111195, Lng: 39.0, AccuracyM: f64(acc), SpeedMps: f64(0)}
	}
	pts := []Point{
		at(5*time.Minute, 0, 8),
		at(90*time.Second, 300, 300),
		at(60*time.Second, 0, 300),
	}
	lv := DeriveLive(pts, now)
	if lv.State != LiveStopped {
		t.Fatalf("نقطتان دقتهما ٣٠٠ م وبينهما ٣٠٠ م انقروا «%s» — لازم «متوقف»", lv.State)
	}
	if lv.LastSeen == nil || !lv.LastSeen.Equal(now.Add(-60*time.Second)) {
		t.Fatalf("آخر ظهور لازم يكون من النقاط الخام: %v", lv.LastSeen)
	}
	// ونقاط ضعيفة بس: متوقف، مو غير متصل.
	only := DeriveLive(pts[1:], now)
	if only.State != LiveStopped {
		t.Fatalf("جوال بيبعت نقاط ضعيفة بس انقرا «%s»", only.State)
	}
	// والحركة الحقيقية بنقاط دقيقة بتضل «يمشي».
	walk := []Point{at(100*time.Second, 0, 8), at(70*time.Second, 120, 8), at(40*time.Second, 240, 8)}
	if lv := DeriveLive(walk, now); lv.State != LiveMoving {
		t.Fatalf("حركة دقيقة انقرت «%s»", lv.State)
	}
}

// Clean طلع من Analyze بلا تغيير: نفس النقاط.
func TestClean_DropsPoorAndImpossible(t *testing.T) {
	pts := []Point{
		fixAt(0, 0, f64(5), nil),
		fixAt(10, 100, f64(80), nil), // دقة ضعيفة
		fixAt(20, 200, f64(5), nil),
		fixAt(21, 5000, f64(5), nil), // قفزة مستحيلة
		fixAt(30, 300, f64(5), nil),
	}
	got := Clean(pts, DefaultConfig())
	if len(got) != 3 || !got[1].At.Equal(pts[2].At) || !got[2].At.Equal(pts[4].At) {
		t.Fatalf("التنظيف غلط: %d نقاط", len(got))
	}
}
