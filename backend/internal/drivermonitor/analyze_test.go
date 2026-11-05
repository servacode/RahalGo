package drivermonitor

// ══════════════════════════════════════════════════════════════════════
// **القيادة الفعلية ومسافة كل طلب** — قرار المالك ٢٠٢٦-١١-٠٥
// ══════════════════════════════════════════════════════════════════════
//
// «الطريق يحسب حسب طريق السائق مو ع كيفنا.» اختبارات صافية بلا قاعدة: القاعدة
// المعتمدة (`IsDrive`)، ورجفة الواقف ما بتصير كيلومترات، ومسافة كل طلب للمتجر
// وللزبون، وطلبان بيده بيتقاسموا المقطع، والصمت جوّا الطلب بينذكر.

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

// metersPerDegLat متر لكل درجة عرض بنصف القطر نفسه (هافرساين على خط الطول).
const metersPerDegLat = 6371000.0 * math.Pi / 180

var t0 = time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)

func f64(v float64) *float64 { return &v }

// north نقطة على بعد `m` متر شمال `p` بعد `dt`.
func north(p Point, m float64, dt time.Duration) Point {
	return Point{At: p.At.Add(dt), Lat: p.Lat + m/metersPerDegLat, Lng: p.Lng,
		AccuracyM: f64(5), SpeedMps: f64(m / dt.Seconds())}
}

func origin() Point {
	return Point{At: t0, Lat: 35.95, Lng: 39.01, AccuracyM: f64(5), SpeedMps: f64(0)}
}

// drive يقود `km` كيلو شمالاً بسرعة `mps` بنقطة كل `step`.
func drive(from Point, km, mps float64, step time.Duration) []Point {
	n := int(math.Round(km * 1000 / (mps * step.Seconds())))
	out := make([]Point, 0, n)
	p := from
	for i := 0; i < n; i++ {
		p = north(p, mps*step.Seconds(), step)
		out = append(out, p)
	}
	return out
}

// stand يقف مكانه `dur` بنقطة كل دقيقة.
func stand(from Point, dur time.Duration) []Point {
	var out []Point
	p := from
	for i := 0; i < int(dur/time.Minute); i++ {
		p.At = p.At.Add(time.Minute)
		p.SpeedMps = f64(0)
		out = append(out, p)
	}
	return out
}

func last(pts []Point) Point { return pts[len(pts)-1] }

func tp(t time.Time) *time.Time { return &t }

// TestIsDrive_Thresholds **القاعدة المعتمدة حرفاً**: ٣ م/ث، والجوال ١٫٥ إن أرسلها،
// والدقة ٣٠ م (الغائبة تمرّ)، والمزيّف ليس قيادة.
func TestIsDrive_Thresholds(t *testing.T) {
	cfg := DefaultConfig()
	a := origin()
	seg := func(mps float64) (Point, Point) {
		b := north(a, mps*10, 10*time.Second)
		return a, b
	}
	if x, y := seg(2.99); IsDrive(x, y, cfg) {
		t.Errorf("٢٫٩٩ م/ث انحسبت قيادة")
	}
	if x, y := seg(3.01); !IsDrive(x, y, cfg) {
		t.Errorf("٣٫٠١ م/ث ما انحسبت قيادة")
	}
	// سرعة الجوال للنقطة الثانية
	x, y := seg(8)
	y.SpeedMps = f64(1.49)
	if IsDrive(x, y, cfg) {
		t.Errorf("جوال ١٫٤٩ م/ث انحسب قيادة")
	}
	y.SpeedMps = f64(1.5)
	if !IsDrive(x, y, cfg) {
		t.Errorf("جوال ١٫٥ م/ث ما انحسب قيادة")
	}
	y.SpeedMps = nil
	if !IsDrive(x, y, cfg) {
		t.Errorf("بلا سرعة جوال ما انحسب قيادة — الغائبة تمرّ")
	}
	// الدقة: ٣٠ تمرّ، ٣٠٫١ لا، والغائبة تمرّ — للنقطتين
	for _, c := range []struct {
		acc  *float64
		want bool
	}{{f64(30), true}, {f64(30.1), false}, {nil, true}} {
		x, y := seg(8)
		x.AccuracyM = c.acc
		if got := IsDrive(x, y, cfg); got != c.want {
			t.Errorf("دقة الأولى %v: %v والمتوقَّع %v", c.acc, got, c.want)
		}
		x, y = seg(8)
		y.AccuracyM = c.acc
		if got := IsDrive(x, y, cfg); got != c.want {
			t.Errorf("دقة الثانية %v: %v والمتوقَّع %v", c.acc, got, c.want)
		}
	}
	// المزيّف
	x, y = seg(8)
	y.Mocked = true
	if IsDrive(x, y, cfg) {
		t.Errorf("نقطة مزيّفة انحسبت قيادة")
	}
	// زمن صفر
	x, y = seg(8)
	y.At = x.At
	if IsDrive(x, y, cfg) {
		t.Errorf("مقطع بزمن صفر انحسب قيادة")
	}
}

// TestAnalyze_StandingJitterIsNotDriving **رجفة الواقف ما بتصير كيلومترات**: ساعة وقوف
// برجفة ٢٥–٤٥ م كل دقيقة ⇒ كيلومترات محسوبة (رجفة) وصفر قيادة.
func TestAnalyze_StandingJitterIsNotDriving(t *testing.T) {
	c := origin()
	pts := []Point{c}
	for i := 1; i <= 60; i++ {
		off := 25.0 + float64(i%5)*5 // ٢٥..٤٥ م
		if i%2 == 0 {
			off = -off
		}
		pts = append(pts, Point{At: t0.Add(time.Duration(i) * time.Minute), Lat: c.Lat + off/metersPerDegLat,
			Lng: c.Lng, AccuracyM: f64(20), SpeedMps: f64(0.3)})
	}
	res := Analyze(Input{Points: pts, Cfg: DefaultConfig()})
	if res.Summary.KmTotal <= 0 {
		t.Fatalf("km_total = %v — الرجفة لازم تبيّن بالمحسوب", res.Summary.KmTotal)
	}
	if res.Summary.DriveKmTotal != 0 {
		t.Fatalf("قيادة = %v كم لواقفٍ مكانه", res.Summary.DriveKmTotal)
	}
	if math.Abs(res.Summary.NotDrivingKm-res.Summary.KmTotal) > 1e-9 {
		t.Fatalf("حركة ما انحسبت = %v والمحسوب %v", res.Summary.NotDrivingKm, res.Summary.KmTotal)
	}
	if res.Alert(1) {
		t.Fatalf("الرجفة طلّعت تنبيه «بلا طلب»")
	}
}

// TestAnalyze_DriveAndNotDrivingAddUp **الأجزاء = المجموع** (فحص الذات بالـSQL المعتمد):
// قيادة بطلب + للاستلام + بلا طلب + ما انحسب = كل المحسوب، ومجموع الطلبات = القيادة بطلب.
func TestAnalyze_DriveAndNotDrivingAddUp(t *testing.T) {
	rng := rand.New(rand.NewSource(20261105))
	for run := 0; run < 20; run++ {
		p := origin()
		pts := []Point{p}
		for i := 0; i < 400; i++ {
			dt := time.Duration(2+rng.Intn(70)) * time.Second
			if rng.Intn(60) == 0 {
				dt = time.Duration(7+rng.Intn(10)) * time.Minute // انقطاع
			}
			m := rng.Float64() * 300
			q := Point{At: p.At.Add(dt), Lat: p.Lat + (rng.Float64()-0.3)*m/metersPerDegLat,
				Lng: p.Lng + (rng.Float64()-0.5)*m/metersPerDegLat}
			switch rng.Intn(4) {
			case 0:
				q.AccuracyM = nil
			case 1:
				q.AccuracyM = f64(rng.Float64() * 80)
			default:
				q.AccuracyM = f64(rng.Float64() * 30)
			}
			if rng.Intn(3) > 0 {
				q.SpeedMps = f64(rng.Float64() * 15)
			}
			q.Mocked = rng.Intn(50) == 0
			pts = append(pts, q)
			p = q
		}
		var wins []Window
		end := pts[len(pts)-1].At
		for k := 0; k < 4; k++ {
			a := t0.Add(time.Duration(rng.Int63n(int64(end.Sub(t0)))))
			pk := a.Add(time.Duration(rng.Intn(20)) * time.Minute)
			e := pk.Add(time.Duration(5+rng.Intn(40)) * time.Minute)
			wins = append(wins, Window{OrderID: string(rune('A' + k)), AssignedAt: tp(a), PickedUpAt: tp(pk), EndAt: tp(e)})
		}
		res := Analyze(Input{Points: pts, Windows: wins, Now: end, Cfg: DefaultConfig()})
		s := res.Summary
		sum := s.DriveKmWithOrder + s.DriveKmToPickup + s.DriveKmNoOrder + s.NotDrivingKm
		if math.Abs(sum-s.KmTotal) > 1e-9 {
			t.Fatalf("جولة %d: الأجزاء %v ≠ المجموع %v", run, sum, s.KmTotal)
		}
		var orders float64
		for _, tk := range res.Tenures {
			orders += tk.ToPickupM + tk.WithOrderM
		}
		if math.Abs(orders/1000-(s.DriveKmWithOrder+s.DriveKmToPickup)) > 1e-9 {
			t.Fatalf("جولة %d: مجموع الطلبات %v ≠ القيادة بطلب %v", run, orders/1000, s.DriveKmWithOrder+s.DriveKmToPickup)
		}
		if s.DriveKmTotal > s.KmTotal+1e-12 {
			t.Fatalf("جولة %d: القيادة %v أكثر من المحسوب %v", run, s.DriveKmTotal, s.KmTotal)
		}
	}
}

// TestAnalyze_OverlappingTenuresSplitEqually **طلبان بيده على مشوار ٢ كم** ⇒ كيلو لكلٍّ
// منهما، ومعلَّمان «مشتركة»، والمجموع = القيادة بطلب.
func TestAnalyze_OverlappingTenuresSplitEqually(t *testing.T) {
	p := origin()
	pts := append([]Point{p}, drive(p, 2, 10, 10*time.Second)...)
	end := last(pts).At.Add(time.Minute)
	before := t0.Add(-time.Minute)
	wins := []Window{
		{OrderID: "a", AssignedAt: tp(before.Add(-5 * time.Minute)), PickedUpAt: tp(before), EndAt: tp(end)},
		{OrderID: "b", AssignedAt: tp(before.Add(-3 * time.Minute)), PickedUpAt: tp(before), EndAt: tp(end)},
	}
	res := Analyze(Input{Points: pts, Windows: wins, Cfg: DefaultConfig()})
	for _, id := range []string{"a", "b"} {
		tk := res.Tenures[id]
		if tk == nil {
			t.Fatalf("لا مسافة للطلب %s", id)
		}
		if math.Abs(tk.WithOrderM-1000) > 10 {
			t.Errorf("الطلب %s: %v م والمتوقَّع ~١٠٠٠", id, tk.WithOrderM)
		}
		if !tk.Shared {
			t.Errorf("الطلب %s ما تعلّم «مشترك»", id)
		}
	}
	got := res.Tenures["a"].WithOrderM + res.Tenures["b"].WithOrderM
	if math.Abs(got/1000-res.Summary.DriveKmWithOrder) > 1e-9 {
		t.Fatalf("المجموع %v ≠ القيادة بطلب %v — انعدّ المقطع مرّتين", got/1000, res.Summary.DriveKmWithOrder)
	}
}

// TestAnalyze_ToPickupThenWithOrder **«من بداية استلام الطلب الى انتهاء الطلب»**: ١ كم
// للمتجر ثم ١٫٥ كم للزبون (±١٪) — والوقوف عند المتجر ما بيزيد شي.
func TestAnalyze_ToPickupThenWithOrder(t *testing.T) {
	p := origin()
	pts := []Point{p}
	pts = append(pts, drive(p, 1, 10, 10*time.Second)...)
	atStore := last(pts)
	pts = append(pts, stand(atStore, 4*time.Minute)...)
	pts = append(pts, drive(last(pts), 1.5, 10, 10*time.Second)...)
	door := last(pts).At
	pts = append(pts, drive(last(pts), 0.8, 10, 10*time.Second)...) // بعد التسليم: بلا طلب
	wins := []Window{{OrderID: "o", AssignedAt: tp(t0), PickedUpAt: tp(atStore.At.Add(2 * time.Minute)), EndAt: tp(door)}}
	res := Analyze(Input{Points: pts, Windows: wins, Cfg: DefaultConfig()})
	tk := res.Tenures["o"]
	if tk == nil {
		t.Fatal("لا مسافة للطلب")
	}
	if math.Abs(tk.ToPickupM-1000) > 10 {
		t.Errorf("للمتجر = %v م والمتوقَّع ١٠٠٠ ±١٪", tk.ToPickupM)
	}
	if math.Abs(tk.WithOrderM-1500) > 15 {
		t.Errorf("للزبون = %v م والمتوقَّع ١٥٠٠ ±١٪", tk.WithOrderM)
	}
	if tk.Shared {
		t.Errorf("طلب وحيد تعلّم «مشترك»")
	}
	if math.Abs(res.Summary.DriveKmNoOrder-0.8) > 0.01 {
		t.Errorf("بلا طلب = %v كم والمتوقَّع ٠٫٨", res.Summary.DriveKmNoOrder)
	}
}

// TestAnalyze_GapInsideTenureReported **٨ دقائق صمت أثناء الطلب** ⇒ `gap_min = 8`،
// والقفزة فوق الصمت ما بتنحسب.
func TestAnalyze_GapInsideTenureReported(t *testing.T) {
	p := origin()
	pts := []Point{p}
	pts = append(pts, drive(p, 1, 10, 10*time.Second)...)
	silent := north(last(pts), 1200, 8*time.Minute) // رجع بعد ٨ د وهو أبعد ١٫٢ كم
	pts = append(pts, silent)
	pts = append(pts, drive(silent, 0.5, 10, 10*time.Second)...)
	wins := []Window{{OrderID: "o", AssignedAt: tp(t0), PickedUpAt: tp(t0), EndAt: tp(last(pts).At)}}
	res := Analyze(Input{Points: pts, Windows: wins, Cfg: DefaultConfig()})
	tk := res.Tenures["o"]
	if tk == nil {
		t.Fatal("لا مسافة للطلب")
	}
	if tk.GapMin != 8 {
		t.Errorf("gap_min = %d والمتوقَّع ٨", tk.GapMin)
	}
	if math.Abs(tk.WithOrderM-1500) > 15 {
		t.Errorf("بطلب = %v م والمتوقَّع ١٥٠٠ (القفزة فوق الصمت ما بتنحسب)", tk.WithOrderM)
	}
	if tk.Points != len(pts) {
		t.Errorf("نقاط الطلب = %d والمتوقَّع %d", tk.Points, len(pts))
	}
}

// TestAnalyze_SpanCountsOnlyInside **مدى الدوام**: المقطع اللي منتصفه برّا المدى ما بينحسب.
func TestAnalyze_SpanCountsOnlyInside(t *testing.T) {
	p := origin()
	pts := append([]Point{p}, drive(p, 3, 10, 10*time.Second)...) // ٣ كم بـ٣٠٠ ث
	from, to := t0.Add(100*time.Second), t0.Add(200*time.Second)
	res := Analyze(Input{Points: pts, From: from, To: to, Cfg: DefaultConfig()})
	if math.Abs(res.Summary.DriveKmTotal-1.0) > 0.011 {
		t.Fatalf("قيادة المدى = %v كم والمتوقَّع ١", res.Summary.DriveKmTotal)
	}
	all := Analyze(Input{Points: pts, Cfg: DefaultConfig()})
	if math.Abs(all.Summary.DriveKmTotal-3.0) > 0.011 {
		t.Fatalf("قيادة كل النقاط = %v كم والمتوقَّع ٣", all.Summary.DriveKmTotal)
	}
}

// TestMergeTenures **سطرا إسنادٍ للطلب نفسه بلا خروجٍ بينهما = ولاية واحدة**.
func TestMergeTenures(t *testing.T) {
	a, b, c := t0, t0.Add(10*time.Minute), t0.Add(30*time.Minute)
	got := mergeTenures([]Window{
		{OrderID: "x", AssignedAt: tp(b), EndAt: tp(c), EndKind: "delivered"},
		{OrderID: "x", AssignedAt: tp(a), EndAt: tp(c), EndKind: "delivered"},
		{OrderID: "y", AssignedAt: tp(a), EndAt: tp(b), EndKind: "released"},
		{OrderID: "y", AssignedAt: tp(c), EndKind: ""},
	})
	if len(got) != 3 {
		t.Fatalf("الولايات = %d والمتوقَّع ٣: %+v", len(got), got)
	}
	if got[0].OrderID != "x" && got[0].OrderID != "y" {
		t.Fatalf("الترتيب: %+v", got)
	}
	var x int
	for _, w := range got {
		if w.OrderID == "x" {
			x++
			if !w.AssignedAt.Equal(a) {
				t.Errorf("ولاية x المدموجة تبدأ %v والمتوقَّع %v", w.AssignedAt, a)
			}
		}
	}
	if x != 1 {
		t.Fatalf("x = %d ولاية والمتوقَّع ١", x)
	}
}

// TestShiftTenures_CarriedOnlyWhenEndedInShift **ولايةٌ بدأت قبل الدوام** (دوامٌ سابقٌ سُكّر
// والطلبُ بيده) تدخل الدوامَ الجديد **إن أنهاها فيه تسليماً أو فشلاً** وحدَه: ما أعادته
// العمليّات للطابور، أو أُلغي، أو بقي مفتوحاً، أو انتهى قبل الفتح أو بعد السكّر — لا يدخل.
// ومشوارُ إرجاعٍ يُقاس بوقت الفشل لا بنهاية المشوار. وما بدأ في الدوام يدخل كلُّه.
func TestShiftTenures_CarriedOnlyWhenEndedInShift(t *testing.T) {
	open, closed := t0.Add(time.Hour), t0.Add(3*time.Hour)
	s := ShiftSpan{OpenedAt: open, ClosedAt: &closed}
	before := t0.Add(10 * time.Minute)
	in := t0.Add(2 * time.Hour)
	after := t0.Add(4 * time.Hour)
	ws := []Window{
		{OrderID: "delivered_in", AssignedAt: tp(before), EndAt: tp(in), EndEventAt: tp(in), EndKind: "delivered"},
		{OrderID: "failed_in", AssignedAt: tp(before), EndAt: tp(in), EndEventAt: tp(in), EndKind: "failed"},
		// فشل داخل الدوام ومشوارُ الإرجاع بعد سكّره — يدخل (بوقت الفشل)
		{OrderID: "returned_in", AssignedAt: tp(before), EndAt: tp(after), EndEventAt: tp(in), EndKind: "returned"},
		{OrderID: "requeued", AssignedAt: tp(before), EndAt: tp(in), EndEventAt: tp(in), EndKind: "released"},
		{OrderID: "cancelled", AssignedAt: tp(before), EndAt: tp(in), EndEventAt: tp(in), EndKind: "cancelled"},
		{OrderID: "still_open", AssignedAt: tp(before)},
		{OrderID: "delivered_after", AssignedAt: tp(before), EndAt: tp(after), EndEventAt: tp(after), EndKind: "delivered"},
		{OrderID: "delivered_before", AssignedAt: tp(before), EndAt: tp(before.Add(time.Minute)), EndEventAt: tp(before.Add(time.Minute)), EndKind: "delivered"},
		// بدأ في الدوام — يدخل أيّاً كانت نهايتُه
		{OrderID: "own_released", AssignedAt: tp(in), EndAt: tp(in.Add(time.Minute)), EndKind: "released"},
		{OrderID: "own_open", AssignedAt: tp(in)},
	}
	got := map[string]bool{}
	for _, w := range ShiftTenures(ws, s) {
		got[w.OrderID] = true
	}
	want := map[string]bool{"delivered_in": true, "failed_in": true, "returned_in": true,
		"own_released": true, "own_open": true}
	for _, w := range ws {
		if got[w.OrderID] != want[w.OrderID] {
			t.Errorf("%s: دخل=%v والمتوقَّع %v", w.OrderID, got[w.OrderID], want[w.OrderID])
		}
	}
	// **ودوامٌ مفتوح**: تسليمٌ بعد فتحه يدخل
	if n := len(ShiftTenures(ws[:1], ShiftSpan{OpenedAt: open})); n != 1 {
		t.Errorf("دوامٌ مفتوح وتسليمٌ بعد فتحه: %d والمتوقَّع ١", n)
	}
}
