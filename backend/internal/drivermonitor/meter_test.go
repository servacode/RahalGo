package drivermonitor

// ══════════════════════════════════════════════════════════════════════
// **المقياس `Moved`** — طلب المالك ٢٠٢٦-١١-١٨: «الكيلومترات اللي مشيتها الدراجة لازم تكون رقم حقيقي»
// ══════════════════════════════════════════════════════════════════════
//
// كل اختبار بيقيس القاعدة القديمة (`RealDrive`) كمان: **بتسقط** بالزحمة والانقطاع ورجفة جوّا المحل،
// والجديدة بتنجح. ومختبر الحقيقة الكامل ع شوارع الرقة بـ`roadmatch/truth_test.go`.

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

func sumMeters(m []float64) float64 {
	t := 0.0
	for _, v := range m {
		t += v
	}
	return t
}

func meterBoth(pts []Point) (moved, old float64) {
	cfg := DefaultConfig()
	return sumMeters(Moved{Cfg: cfg}.Meters(pts)), sumMeters(RealDrive{Cfg: cfg}.Meters(pts))
}

// crawl يمشي شمالاً `m` متر بسرعة `mps` بنقطة كل `step` ودقّة ٦ م.
func crawl(from Point, m, mps float64, step time.Duration) []Point {
	out := []Point{}
	p := from
	for done := 0.0; done < m-1e-9; done += mps * step.Seconds() {
		p = north(p, mps*step.Seconds(), step)
		p.AccuracyM = f64(6)
		out = append(out, p)
	}
	return out
}

// TestMoved_SlowTrafficCounts **الزحمة ٢ م/ث (٧ كم/س)** ٦٠٠ م ⇒ ٦٠٠ م. القديمة: صفر (تحت ١٠٫٨ كم/س).
func TestMoved_SlowTrafficCounts(t *testing.T) {
	p := origin()
	pts := append([]Point{p}, stand(p, 2*time.Minute)...)
	pts = append(pts, crawl(last(pts), 600, 2, 10*time.Second)...)
	pts = append(pts, stand(last(pts), 3*time.Minute)...)
	got, old := meterBoth(pts)
	if math.Abs(got-600) > 30 {
		t.Errorf("الزحمة: %.0f م والمتوقَّع ٦٠٠ ±٥٪", got)
	}
	if old > 100 {
		t.Errorf("القديمة طلعت %.0f م — المفروض تسقط هون (هيك كانت)", old)
	}
}

// TestMoved_IndoorJitterIsZero **٤٥ دقيقة جوّا محل**: رجفة مترابطة σ ٢٥ م ودقّة ١٨–٣٠ وسرعة جوال عشوائية
// ⇒ ≤ ٢٠ م. القديمة: مئات الأمتار.
func TestMoved_IndoorJitterIsZero(t *testing.T) {
	rng := rand.New(rand.NewSource(18112026))
	c := origin()
	var pts []Point
	var ex, ey float64
	for s := 0; s < 45*60; s += 5 {
		phi := math.Exp(-5.0 / 20)
		ex = ex*phi + math.Sqrt(1-phi*phi)*25*rng.NormFloat64()
		ey = ey*phi + math.Sqrt(1-phi*phi)*25*rng.NormFloat64()
		p := Point{At: c.At.Add(time.Duration(s) * time.Second), Lat: c.Lat + ey/metersPerDegLat,
			Lng:       c.Lng + ex/(metersPerDegLat*math.Cos(c.Lat*math.Pi/180)),
			AccuracyM: f64(18 + 12*rng.Float64()), SpeedMps: f64(2.5 * rng.Float64())}
		pts = append(pts, p)
	}
	pts = Filter(nil, pts) // نفس تخفيف الأثر
	got, old := meterBoth(Clean(pts, DefaultConfig()))
	if got > 20 {
		t.Errorf("رجفة جوّا المحل صارت %.0f م قيادة", got)
	}
	if old < 100 {
		t.Errorf("القديمة طلعت %.0f م بس — الاختبار ما عاد يميّز", old)
	}
}

// TestMoved_SpikeIsZero **نتوء واحد ٦٠ م ورجع** والسائق واقف برّا ⇒ صفر.
func TestMoved_SpikeIsZero(t *testing.T) {
	p := origin()
	pts := []Point{p}
	for i := 1; i <= 30; i++ {
		q := p
		q.At = p.At.Add(time.Duration(i) * 20 * time.Second)
		q.AccuracyM = f64(8)
		if i == 12 {
			q.Lat += 60 / metersPerDegLat
			q.SpeedMps = f64(4)
		}
		pts = append(pts, q)
	}
	if got, _ := meterBoth(pts); got > 0 {
		t.Errorf("النتوء صار %.0f م", got)
	}
}

// TestMoved_GapWithHeartbeatsBridged **انقطاع إشارة ٤ دقايق والتطبيق بيعيد آخر موضع** (نبضة الواقف، بلا دقّة
// ولا سرعة) ثمّ رجع ١٢٠٠ م أبعد ⇒ القيادة ≈ ١ كم قبل + ١٢٠٠ م فوق الانقطاع + ٥٠٠ م بعد.
func TestMoved_GapWithHeartbeatsBridged(t *testing.T) {
	p := origin()
	pts := append([]Point{p}, drive(p, 1, 8, 5*time.Second)...)
	lastFix := last(pts)
	for i := 1; i <= 12; i++ {
		pts = append(pts, Point{At: lastFix.At.Add(time.Duration(i) * 20 * time.Second), Lat: lastFix.Lat, Lng: lastFix.Lng})
	}
	back := north(lastFix, 1200, 250*time.Second)
	pts = append(pts, back)
	pts = append(pts, drive(back, 0.5, 8, 5*time.Second)...)
	if !Resent(pts, len(pts)-len(drive(back, 0.5, 8, 5*time.Second))-2) {
		t.Fatal("نبضة الواقف ما انعرفت")
	}
	got, _ := meterBoth(Clean(pts, DefaultConfig()))
	if math.Abs(got-2700) > 40 {
		t.Errorf("القيادة = %.0f م والمتوقَّع ~٢٧٠٠ (الانقطاع بينحسب من آخر موضع حقيقي)", got)
	}
}

// TestMoved_MockedNotCounted **المزيَّف ما بيتحسب** ولا بيصير مرساة.
func TestMoved_MockedNotCounted(t *testing.T) {
	p := origin()
	pts := []Point{p}
	fake := drive(p, 2, 10, 5*time.Second)
	for i := range fake {
		fake[i].Mocked = true
	}
	pts = append(pts, fake...)
	if got, _ := meterBoth(pts); got > 0 {
		t.Errorf("المزيَّف صار %.0f م", got)
	}
}

// TestMoved_OnRoadUsesWholeStep **مع المطابقة الخطوة كلها ع الشارع**، والطريق المستحيل (لفّة) ⇒ الخط.
func TestMoved_OnRoadUsesWholeStep(t *testing.T) {
	p := origin()
	pts := append([]Point{p}, drive(p, 1, 10, 5*time.Second)...)
	m := Moved{Cfg: DefaultConfig()}
	// الطريق ١٫٢ × الخط
	on := sumMeters(m.MetersOnRoad(pts, func(i, j int) (float64, bool) { return 1.2 * Dist(pts[i], pts[j]), true }))
	if math.Abs(on-1200) > 15 {
		t.Errorf("ع الطريق = %.0f م والمتوقَّع ١٢٠٠", on)
	}
	// الطريق ٥ × الخط (لفّة حول بلوك ع رجفة) ⇒ الخط المستقيم
	loop := sumMeters(m.MetersOnRoad(pts, func(i, j int) (float64, bool) { return 5*Dist(pts[i], pts[j]) + 200, true }))
	if math.Abs(loop-1000) > 15 {
		t.Errorf("طريق مستحيل = %.0f م والمتوقَّع الخط ١٠٠٠", loop)
	}
	// ما انطابق ⇒ الخط
	none := sumMeters(m.MetersOnRoad(pts, func(i, j int) (float64, bool) { return 0, false }))
	if math.Abs(none-1000) > 15 {
		t.Errorf("بلا طريق = %.0f م والمتوقَّع ١٠٠٠", none)
	}
}
