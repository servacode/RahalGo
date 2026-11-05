package roadmatch

import (
	"math"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/polyline"
)

// عتبات «برّا الخريطة» (B4): حارة مو مرسومة أو ساحة — Valhalla ما بيقول «ما انطابق»، بيلصقها بشارع
// ضمن نصف قطر البحث وبيلفّ حول البلوك. فنقطتين متتاليتين أو أكتر انلصقوا أبعد من
// max(٢٥ م، ١٫٥ × الدقة) عن مكانهن ⇒ ما إلهن موضع ع الطريق (خط مستقيم متقطّع).
const (
	OffMapMinM      = 25.0
	OffMapAccFactor = 1.5
	// UnmatchedRunChord ٣ نقاط ما انطابقت ورا بعض أو أكتر ⇒ خط مستقيم؛ أقل ⇒ بتاخد موضع اللي قبلها.
	UnmatchedRunChord = 3
)

// Row **سطر بـ`driver_road_chunks`**: قطعة (أو جزء منها لما يكون فيها انقطاع بالطريق).
type Row struct {
	FromAt, ToAt time.Time
	Closed       bool
	Status       string // ok | no_match
	ErrorCode    string
	PointAt      []time.Time
	PosM         []float64 // NaN = بلا موضع ع الطريق
	Line6        string
	RoadM        float64
	StraightM    float64
	TookMs       int
}

// SentPositions **موضع كل نقطة مبعوتة ع خط الجواب** (متر من أوله) — دالة صافية.
//
//   - نقطة انطابقت (أو انحسبت بالتوسيط) بتنسقط ع **قطعة حافّتها هي** من الخط
//     (`begin_shape_index … end_shape_index`)، فالداخل والطالع من نفس الزقاق ما بيتلخبطوا. وبلا رقم
//     حافّة: إسقاط لقدّام من الموضع اللي قبلها.
//   - **لقدّام بس**: موضع أصغر من اللي قبله بيصير متله (رجفة لورا = صفر متر).
//   - نقطة ما انطابقت لحالها (أو تنتين) بتاخد موضع اللي قبلها، و٣ أو أكتر ورا بعض ⇒ NaN.
//   - برّا الخريطة: نقطتين أو أكتر انلصقوا بعاد ⇒ NaN؛ وحدة لحالها بتاخد موضع اللي قبلها.
//
// `breakBefore[k]` = في انقطاع بالطريق قبل النقطة k ⇒ السطر بينقسم هونيك.
func SentPositions(sent []drivermonitor.Point, a Answer) (pos []float64, breakBefore []bool, line [][2]float64, cum []float64) {
	m := len(sent)
	line = polyline.Decode6(a.Shape)
	cum = polyline.Cumulative(line)
	raw := make([]float64, m)
	unmatched := make([]bool, m)
	far := make([]bool, m)
	breakBefore = make([]bool, m)
	last := 0.0
	for k := 0; k < m; k++ {
		raw[k] = math.NaN()
		if k >= len(a.MatchedPoints) || len(line) == 0 {
			unmatched[k] = true
			continue
		}
		mp := a.MatchedPoints[k]
		if k > 0 && k-1 < len(a.MatchedPoints) &&
			(mp.BeginRouteDiscontinuity || a.MatchedPoints[k-1].EndRouteDiscontinuity) {
			breakBefore[k] = true
		}
		if mp.Type != "matched" && mp.Type != "interpolated" {
			unmatched[k] = true
			continue
		}
		p := [2]float64{mp.Lon, mp.Lat}
		var at float64
		if mp.EdgeIndex != nil && *mp.EdgeIndex >= 0 && *mp.EdgeIndex < int64(len(a.Edges)) {
			e := a.Edges[*mp.EdgeIndex]
			at, _ = polyline.Project(line, cum, p, e.BeginShapeIndex, e.EndShapeIndex)
		} else {
			at, _ = polyline.Project(line, cum, p, vertexBefore(cum, last), len(line)-1)
		}
		if math.IsNaN(at) {
			unmatched[k] = true
			continue
		}
		raw[k] = at
		last = math.Max(last, at)
		lim := OffMapMinM
		if acc := sent[k].AccuracyM; acc != nil {
			lim = math.Max(lim, OffMapAccFactor**acc)
		}
		if mp.DistanceFromTracePoint != nil && *mp.DistanceFromTracePoint > lim {
			far[k] = true
		}
	}
	nan := make([]bool, m)
	carry := make([]bool, m)
	runs := func(flag []bool, minNaN int) {
		for i := 0; i < m; {
			if !flag[i] {
				i++
				continue
			}
			j := i
			for j+1 < m && flag[j+1] {
				j++
			}
			for k := i; k <= j; k++ {
				if j-i+1 >= minNaN {
					nan[k] = true
				} else {
					carry[k] = true
				}
			}
			i = j + 1
		}
	}
	runs(unmatched, UnmatchedRunChord)
	runs(far, 2)

	pos = make([]float64, m)
	prev := math.NaN()
	for k := 0; k < m; k++ {
		switch {
		case nan[k]:
			pos[k] = math.NaN()
		case carry[k]:
			pos[k] = prev
		default:
			v := raw[k]
			if !math.IsNaN(prev) && v < prev {
				v = prev
			}
			pos[k] = v
			prev = v
		}
	}
	return pos, breakBefore, line, cum
}

// vertexBefore فهرس الرأس اللي قبل الموضع.
func vertexBefore(cum []float64, at float64) int {
	i := 0
	for i+1 < len(cum) && cum[i+1] <= at {
		i++
	}
	return i
}

// sentPoints النقاط المبعوتة من القطعة.
func sentPoints(c Chunk) []drivermonitor.Point {
	out := make([]drivermonitor.Point, 0, len(c.Points))
	for i, p := range c.Points {
		if c.Sent[i] {
			out = append(out, p)
		}
	}
	return out
}

// Rows **من قطعة وجواب Valhalla لأسطر التخزين** — دالة صافية.
//
// النقطة اللي ما انبعتت بتاخد موضع اللي قبلها (صفر متر)، والسطر بينقسم عند كل انقطاع بالطريق:
// المقطع اللي عليه الانقطاع ما إله سطر ⇒ خط مستقيم. كل سطر إله خطّه المقصوص ومواضعه من أوله.
func Rows(c Chunk, a Answer) []Row {
	sent := sentPoints(c)
	sp, brk, line, cum := SentPositions(sent, a)
	n := len(c.Points)
	all := make([]float64, n)
	split := make([]bool, n)
	prev := math.NaN()
	k := 0
	for i := 0; i < n; i++ {
		if c.Sent[i] {
			all[i] = sp[k]
			split[i] = brk[k]
			k++
			prev = all[i]
			continue
		}
		all[i] = prev
	}
	var out []Row
	start := 0
	for i := 1; i <= n; i++ {
		if i < n && !split[i] {
			continue
		}
		out = append(out, makeRow(c, start, i, all, line, cum))
		start = i
	}
	return out
}

// makeRow سطر للنقاط [s, e) بخطّه المقصوص ومواضعه من أوله.
func makeRow(c Chunk, s, e int, all []float64, line [][2]float64, cum []float64) Row {
	r := Row{FromAt: c.Points[s].At, ToAt: c.Points[e-1].At, Closed: c.Closed, Status: "ok"}
	lo, hi := math.Inf(1), math.Inf(-1)
	for i := s; i < e; i++ {
		if v := all[i]; !math.IsNaN(v) {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
	}
	for i := s; i < e; i++ {
		r.PointAt = append(r.PointAt, c.Points[i].At)
		v := all[i]
		if !math.IsNaN(v) {
			v -= lo
		}
		r.PosM = append(r.PosM, v)
		if i > s {
			r.StraightM += drivermonitor.Dist(c.Points[i-1], c.Points[i])
			if a, b := r.PosM[i-s-1], r.PosM[i-s]; !math.IsNaN(a) && !math.IsNaN(b) {
				r.RoadM += b - a
			}
		}
	}
	if hi > lo && len(line) >= 2 {
		r.Line6 = polyline.Encode6(polyline.Cut(line, cum, lo, hi))
	}
	return r
}

// FlatRow سطر بلا مطابقة لقطعة (`no_match`، أو قطعة فيها أقل من نقطتين مبعوتتين ⇒ كلها صفر متر).
func FlatRow(c Chunk, status, code string) Row {
	r := Row{FromAt: c.From(), ToAt: c.To(), Closed: c.Closed, Status: status, ErrorCode: code}
	for i, p := range c.Points {
		r.PointAt = append(r.PointAt, p.At)
		v := math.NaN()
		if status == "ok" {
			v = 0
		}
		r.PosM = append(r.PosM, v)
		if i > 0 {
			r.StraightM += drivermonitor.Dist(c.Points[i-1], p)
		}
	}
	return r
}
