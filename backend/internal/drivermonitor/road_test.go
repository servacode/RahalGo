package drivermonitor

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/polyline"
)

// ══════════════════════════════════════════════════════════════════════
// **المسافة ع الطريق المطابق** — طلب المالك ٢٠٢٦-١١-٠٥ («الطريق يحسب حسب طريق السائق»)
// ══════════════════════════════════════════════════════════════════════

const (
	rdLat = 35.40
	rdLng = 38.40
	// ٣٠٠ م شرقاً وشمالاً تقريباً عند خط عرض الرقة.
	rdEast  = 0.0033077
	rdNorth = 0.0026980
)

var rdT0 = time.Date(2026, 11, 5, 10, 0, 0, 0, time.UTC)

func rdAcc(v float64) *float64 { return &v }

func rdPt(sec int, lat, lng float64) Point {
	return Point{At: rdT0.Add(time.Duration(sec) * time.Second), Lat: lat, Lng: lng, AccuracyM: rdAcc(8), SpeedMps: rdAcc(8)}
}

// rdChunk قطعة مطابقة: النقاط ع الخط بمواضعها من أوله (مسقطة فعلياً).
func rdChunk(line [][2]float64, pts []Point, ok bool) RoadChunk {
	c := RoadChunk{FromAt: pts[0].At, ToAt: pts[len(pts)-1].At, Closed: true, OK: ok, Line: line}
	cum := polyline.Cumulative(line)
	for _, p := range pts {
		c.PointAt = append(c.PointAt, p.At)
		if !ok {
			c.PosM = append(c.PosM, math.NaN())
			continue
		}
		pos, _ := polyline.Project(line, cum, [2]float64{p.Lng, p.Lat}, 0, len(line)-1)
		c.PosM = append(c.PosM, pos)
	}
	return c
}

// rdL ركوبة على شكل L بلا نقطة الزاوية: شرقاً ٣٠٠ م ثم شمالاً ٣٠٠ م بدقيقة ونص.
func rdL() ([]Point, [][2]float64) {
	a := rdPt(0, rdLat, rdLng)
	b := rdPt(90, rdLat+rdNorth, rdLng+rdEast)
	line := [][2]float64{{rdLng, rdLat}, {rdLng + rdEast, rdLat}, {rdLng + rdEast, rdLat + rdNorth}}
	return []Point{a, b}, line
}

// DM1 — المقطع بياخد أمتار الطريق (٦٠٠) مو الوتر (٤٢٤)، و`km_straight` للمقارنة.
func TestAnalyze_RoadMetresCredited(t *testing.T) {
	pts, line := rdL()
	road := NewRoad([]RoadChunk{rdChunk(line, pts, true)}, false)
	res := Analyze(Input{Points: pts, Road: road})
	if math.Abs(res.Summary.KmNoOrder-0.6) > 0.005 || math.Abs(res.Summary.KmTotal-0.6) > 0.005 {
		t.Errorf("كم الطريق = %.3f (المتوقَّع ٠٫٦)", res.Summary.KmNoOrder)
	}
	if math.Abs(res.Summary.KmStraight-0.424) > 0.005 {
		t.Errorf("km_straight = %.3f (المتوقَّع ٠٫٤٢٤)", res.Summary.KmStraight)
	}
	if math.Abs(res.Summary.RoadShare-1) > 1e-9 || res.Summary.PendingKm != 0 || res.Summary.InferredKm != 0 {
		t.Errorf("الحصّة/المعلّق/المقدَّر: %+v", res.Summary)
	}
}

// DM2 — الخط المرسوم هو الطريق المقصوص (بالزاوية)، والمقطع اللي ما انطابق خط مستقيم متقطّع.
func TestAnalyze_PathFollowsMatchedLine(t *testing.T) {
	pts, line := rdL()
	// مقطع تاني بعد ٩٠ ث: لقدّام شمالاً ٢٠٠ م بقطعة ما انطابقت.
	c := rdPt(150, rdLat+rdNorth*5/3, rdLng+rdEast)
	ok := rdChunk(line, pts, true)
	bad := rdChunk(nil, []Point{pts[1], c}, false)
	road := NewRoad([]RoadChunk{ok, bad}, false)
	res := Analyze(Input{Points: append(pts, c), Road: road})
	if len(res.Path) != 2 {
		t.Fatalf("أجزاء المسار: %d (%+v)", len(res.Path), res.Path)
	}
	p0, p1 := res.Path[0], res.Path[1]
	if !p0.Road || p0.Straight || len(p0.Coords) != 3 {
		t.Fatalf("الجزء الأول لازم يكون ع الطريق بالزاوية: %+v", p0)
	}
	if math.Abs(p0.Coords[1][0]-rdLat) > 1e-6 || math.Abs(p0.Coords[1][1]-(rdLng+rdEast)) > 1e-6 {
		t.Errorf("الزاوية مو ع الخط: %v", p0.Coords[1])
	}
	if p1.Road || !p1.Straight || len(p1.Coords) != 2 {
		t.Errorf("الجزء التاني لازم يكون خط مستقيم متقطّع: %+v", p1)
	}
	if !p0.From.Equal(pts[0].At) || !p0.To.Equal(pts[1].At) || !p1.To.Equal(c.At) {
		t.Errorf("أوقات الأجزاء: %v→%v و%v", p0.From, p0.To, p1.To)
	}
}

// DM3 — **بلا معلومات طريق النتيجة متل قبل بالضبط**: نفس الأرقام ونفس الخط.
func TestAnalyze_NoRoadDataIsToday(t *testing.T) {
	var pts []Point
	for i := 0; i <= 40; i++ {
		pts = append(pts, rdPt(i*20, rdLat+float64(i%7)*0.0003, rdLng+float64(i)*0.0004))
	}
	a := Analyze(Input{Points: pts})
	b := Analyze(Input{Points: pts, Road: nil})
	if !reflect.DeepEqual(a, b) {
		t.Fatal("نتيجتين مختلفتين لنفس المدخل")
	}
	// المجموع بالوتر نفسه، والحقول الجديدة صفر/فاضية.
	sum := 0.0
	for i := 1; i < len(pts); i++ {
		sum += Dist(pts[i-1], pts[i])
	}
	if math.Abs(a.Summary.KmNoOrder-sum/1000) > 1e-9 || math.Abs(a.Summary.KmStraight-sum/1000) > 1e-9 {
		t.Errorf("الكيلومترات تغيّرت: %.4f / %.4f / %.4f", a.Summary.KmNoOrder, a.Summary.KmStraight, sum/1000)
	}
	if a.Summary.RoadShare != 0 || a.Summary.PendingKm != 0 || a.Summary.InferredKm != 0 {
		t.Errorf("حقول الطريق بلا طريق: %+v", a.Summary)
	}
	for _, p := range a.Path {
		if p.Road || p.Inferred || p.Straight || p.Pending {
			t.Errorf("جزء معلَّم بلا طريق: %+v", p)
		}
	}
	// والمطابقة شغّالة بلا ولا قطعة ⇒ كله «لسّا عم تنحسب».
	c := Analyze(Input{Points: pts, Road: NewRoad(nil, true)})
	if math.Abs(c.Summary.PendingKm-sum/1000) > 1e-9 {
		t.Errorf("المعلّق والمطابقة شغّالة = %.4f", c.Summary.PendingKm)
	}
}

// DM4 — بعد آخر قطعة مخزّنة: «لسّا عم تنحسب» والمطابقة شغّالة، ومطفية ⇒ خط مستقيم عادي بلا تقريبي.
func TestAnalyze_PendingOnlyWhileMatchingOn(t *testing.T) {
	pts, line := rdL()
	tail := rdPt(150, rdLat+rdNorth*5/3, rdLng+rdEast)
	chunks := []RoadChunk{rdChunk(line, pts, true)}
	all := append(append([]Point(nil), pts...), tail)

	on := Analyze(Input{Points: all, Road: NewRoad(chunks, true)})
	tailKm := Dist(pts[1], tail) / 1000
	if math.Abs(on.Summary.PendingKm-tailKm) > 1e-6 {
		t.Errorf("المعلّق = %.4f (المتوقَّع %.4f)", on.Summary.PendingKm, tailKm)
	}
	if last := on.Path[len(on.Path)-1]; !last.Pending {
		t.Errorf("الذيل لازم يكون معلّق: %+v", last)
	}
	off := Analyze(Input{Points: all, Road: NewRoad(chunks, false)})
	if off.Summary.PendingKm != 0 {
		t.Errorf("المطابقة مطفية والمعلّق = %.4f", off.Summary.PendingKm)
	}
	if last := off.Path[len(off.Path)-1]; last.Pending || last.Straight || last.Road {
		t.Errorf("الذيل والمطابقة مطفية لازم يكون متل قبل: %+v", last)
	}
}

// DM5 — طلبين ورا بعض: المسار بينقسم عند حدّ الطلب، وكل جزء بأوقاته ⇒ طريق طلب واحد بيتلوّن لحاله.
func TestAnalyze_PartsSplitAtHolds(t *testing.T) {
	var pts []Point
	for i := 0; i <= 40; i++ {
		pts = append(pts, rdPt(i*30, rdLat, rdLng+float64(i)*0.0004))
	}
	at := func(sec int) *time.Time { v := rdT0.Add(time.Duration(sec) * time.Second); return &v }
	wins := []Window{
		{OrderID: "a", AssignedAt: at(0), PickedUpAt: at(0), EndAt: at(600)},
		{OrderID: "b", AssignedAt: at(600), PickedUpAt: at(600), EndAt: at(1200)},
	}
	res := Analyze(Input{Points: pts, Windows: wins, Now: rdT0.Add(2 * time.Hour)})
	if len(res.Path) != 2 {
		t.Fatalf("أجزاء: %d", len(res.Path))
	}
	for _, p := range res.Path {
		if p.Class != ClassWithOrder {
			t.Errorf("صنف: %s", p.Class)
		}
	}
	if !res.Path[0].To.Equal(*at(600)) || !res.Path[1].From.Equal(*at(600)) || !res.Path[1].To.Equal(*at(1200)) {
		t.Errorf("حدود الأجزاء: %v→%v | %v→%v", res.Path[0].From, res.Path[0].To, res.Path[1].From, res.Path[1].To)
	}
}

// DM7 — نقطتين مطابقتين ٨٠٠ م بعاد ⇒ الشارع بيناتهن مقدَّر: محسوب ع الطريق، معلَّم ومجموع لحاله.
func TestAnalyze_InferredFlag(t *testing.T) {
	a := rdPt(0, rdLat, rdLng)
	b := rdPt(90, rdLat, rdLng+rdEast*8/3) // ~٨٠٠ م
	line := [][2]float64{{rdLng, rdLat}, {rdLng + rdEast*8/3, rdLat}}
	res := Analyze(Input{Points: []Point{a, b}, Road: NewRoad([]RoadChunk{rdChunk(line, []Point{a, b}, true)}, true)})
	if res.Summary.InferredKm < 0.79 || len(res.Path) != 1 || !res.Path[0].Inferred || !res.Path[0].Road {
		t.Errorf("المقدَّر: %.3f %+v", res.Summary.InferredKm, res.Path)
	}
	if math.Abs(res.Summary.RoadShare-1) > 1e-9 {
		t.Errorf("المقدَّر ع الطريق: الحصّة %.3f", res.Summary.RoadShare)
	}
}

// DM8 — المطابقة مطفية والقطع موجودة ⇒ أمتار الطريق بتضل محسوبة، وما في معلّق.
func TestAnalyze_StoredChunksUsedWhenOff(t *testing.T) {
	pts, line := rdL()
	res := Analyze(Input{Points: pts, Road: NewRoad([]RoadChunk{rdChunk(line, pts, true)}, false)})
	if math.Abs(res.Summary.KmTotal-0.6) > 0.005 || res.Summary.PendingKm != 0 {
		t.Errorf("مطفية: كم %.3f معلّق %.3f", res.Summary.KmTotal, res.Summary.PendingKm)
	}
}

// RM6 — معقوليّة وقت القراءة: لفّة ١٥٠ م بـ٤ ث ⇒ وتر؛ ٢٥٠ م بـ٥ ث (١٨٠ كم/س) ⇒ وتر؛
// واستدارة حقيقيّة بنص البلوك (وتر ١٠ م، طريق ٢٠ م بـ٤ ث) بتضل.
func TestPlausibility(t *testing.T) {
	seg := func(dt int, chordM, roadM float64) RoadSeg {
		a := rdPt(0, rdLat, rdLng)
		b := rdPt(dt, rdLat, rdLng+chordM/90700*1.0)
		c := RoadChunk{FromAt: a.At, ToAt: b.At, Closed: true, OK: true,
			PointAt: []time.Time{a.At, b.At}, PosM: []float64{0, roadM},
			Line: [][2]float64{{rdLng, rdLat}, {rdLng + roadM/90700, rdLat}}}
		return NewRoad([]RoadChunk{c}, true).Segment(a, b, DefaultConfig())
	}
	if s := seg(4, 30, 180); s.Kind != SegStraight || math.Abs(s.M-30) > 1 {
		t.Errorf("لفّة ١٥٠ م بـ٤ ث لازم تصير وتر: %+v", s.Kind)
	}
	if s := seg(5, 240, 250); s.Kind != SegStraight {
		t.Errorf("١٨٠ كم/س ع الطريق لازم يصير وتر: %+v", s.Kind)
	}
	if s := seg(4, 10, 20); s.Kind != SegRoad || math.Abs(s.M-20) > 1e-6 {
		t.Errorf("الاستدارة الحقيقيّة لازم تضل ع الطريق: %+v %.1f", s.Kind, s.M)
	}
}
