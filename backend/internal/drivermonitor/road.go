package drivermonitor

// ══════════════════════════════════════════════════════════════════════
// **المسافة ع الطرق اللي مشي فيها السائق** (طلب المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «والطريق يحسب حسب طريق السائق مو ع كيفنا».
//
// المطابق (`internal/roadmatch`) بيطابق كل قطعة من الأثر **مرّة وحدة** وبيخزّن بـ`driver_road_chunks`
// موضع كل نقطة ع الخط المطابق. هون **منقرأ** هالقطع ومنعطي كل مقطع بين نقطتين أمتاره ع الطريق:
//
//   - المقطع جوّا قطعة مطابقة ⇒ أمتار الطريق والخط المقصوص (`road`؛ وأبعد من ٥٠٠ م بين النقطتين ⇒
//     الشارع مقدَّر، `inferred`، بيترسم منقّط).
//   - جوّا المطابق بس ما انطابق (لا مطابقة، برّا الخريطة، أو أمتار مستحيلة) ⇒ خط مستقيم متقطّع (`straight`).
//   - بعد آخر قطعة والمطابقة شغّالة ⇒ خط مستقيم و«لسّا عم تنحسب» (`pending`).
//   - غير هيك (المطابقة مطفية وما في شي مخزّن) ⇒ **متل قبل بالضبط**: خط مستقيم عادي.
//
// **والقراءة بس هون** — الكتابة بـ`roadmatch` وحدها، وما في استيراد من هون لهونيك.

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/servacode/rahalgo/backend/internal/polyline"
)

// عتبات القراءة — **بتنطبق وقت القراءة**، فبتتعدّل بلا إعادة مطابقة.
const (
	// RoadInferredM مقطع مطابق نقطتيه أبعد من هيك ⇒ الشارع بيناتهن تخمين المطابق.
	RoadInferredM = 500.0
	// RoadLoopSec و RoadLoopFactor و RoadLoopExtraM: مقطع ≤ ١٠ ث أمتاره أكتر من ٢ × الخط + ٣٠ م ⇒
	// لفّة أو نتوء ما بيقدر الدرّاج يعمله بثواني ⇒ خط مستقيم.
	RoadLoopSec    = 10.0
	RoadLoopFactor = 2.0
	RoadLoopExtraM = 30.0
)

// SegKind نوع المقطع بعد قراءة الطريق.
type SegKind int

const (
	// SegNone ما في معلومات طريق ⇒ خط مستقيم متل قبل.
	SegNone SegKind = iota
	// SegRoad أمتار الطريق المطابق.
	SegRoad
	// SegInferred أمتار الطريق، بس النقطتين بعاد ⇒ الشارع مقدَّر.
	SegInferred
	// SegStraight المطابق حاول وما زبط ⇒ خط مستقيم متقطّع.
	SegStraight
	// SegPending لسّا ما انطابق والمطابقة شغّالة ⇒ خط مستقيم والرقم تقريبي.
	SegPending
)

// RoadSeg مقطع بين نقطتين بعد قراءة الطريق.
type RoadSeg struct {
	Kind SegKind
	// M الأمتار المحسوبة (طريق أو خط مستقيم).
	M float64
	// Line الخط المرسوم [lat, lng] — الطريق المقصوص، أو الخط المستقيم.
	Line [][2]float64
}

// RoadChunk قطعة مخزّنة كما انقرأت.
type RoadChunk struct {
	FromAt, ToAt time.Time
	Closed       bool
	OK           bool // status = ok
	PointAt      []time.Time
	PosM         []float64 // NaN = ما إلها موضع ع الطريق
	Line         [][2]float64
	cum          []float64
}

// Road **معلومات الطريق لسائق** — مدخل `Analyze` (`Input.Road`). nil = متل قبل.
type Road struct {
	// Chunks مرتّبة بـ`FromAt` وما بتتداخل.
	Chunks []RoadChunk
	// On المطابقة شغّالة (`drivers.road_match_enabled`) — بس وقتها في «لسّا عم تنحسب».
	On bool
	// CoverFrom و CoverTo أول وآخر وقت غطّاه المطابق لهالسائق (كل قطعه، مو بس بالمدى المقروء).
	CoverFrom, CoverTo time.Time
}

func (r *Road) prepare() {
	sort.Slice(r.Chunks, func(i, j int) bool { return r.Chunks[i].FromAt.Before(r.Chunks[j].FromAt) })
	for i := range r.Chunks {
		if r.Chunks[i].cum == nil {
			r.Chunks[i].cum = polyline.Cumulative(r.Chunks[i].Line)
		}
	}
}

// covered المقطع ضمن ما غطّاه المطابق؟
func (r *Road) covered(a, b time.Time) bool {
	return !r.CoverFrom.IsZero() && !a.Before(r.CoverFrom) && !b.After(r.CoverTo)
}

// indexAt فهرس الوقت بالقطعة، أو -1.
func (c *RoadChunk) indexAt(t time.Time) int {
	i := sort.Search(len(c.PointAt), func(k int) bool { return !c.PointAt[k].Before(t) })
	if i < len(c.PointAt) && c.PointAt[i].Equal(t) {
		return i
	}
	return -1
}

// Segment **أمتار المقطع a→b وخطّه** — دالة صافية. النقطتين مرتّبتين بالزمن.
func (r *Road) Segment(a, b Point, cfg Config) RoadSeg {
	chord := Dist(a, b)
	straight := [][2]float64{{a.Lat, a.Lng}, {b.Lat, b.Lng}}
	if r == nil {
		return RoadSeg{Kind: SegNone, M: chord, Line: straight}
	}
	if cfg.GapAfter == 0 {
		cfg = DefaultConfig()
	}
	// القطعة اللي بتحوي النقطتين: آخر قطعة بلّشت قبل (أو عند) a.
	k := sort.Search(len(r.Chunks), func(i int) bool { return r.Chunks[i].FromAt.After(a.At) }) - 1
	// القطع ما بتتداخل إلا بنقطة الحدّ، فبتكفي القطعة وما قبلها.
	for j := k; j >= 0 && j >= k-1; j-- {
		c := &r.Chunks[j]
		if c.ToAt.Before(b.At) {
			break
		}
		ia, ib := c.indexAt(a.At), c.indexAt(b.At)
		if ia < 0 || ib < 0 || ib <= ia {
			continue
		}
		if !c.OK {
			return RoadSeg{Kind: SegStraight, M: chord, Line: straight}
		}
		pa, pb := c.PosM[ia], c.PosM[ib]
		if math.IsNaN(pa) || math.IsNaN(pb) {
			return RoadSeg{Kind: SegStraight, M: chord, Line: straight}
		}
		m := math.Max(0, pb-pa)
		dt := b.At.Sub(a.At).Seconds()
		// معقوليّة: أسرع من ١٢٠ كم/س ع الطريق، أو لفّة بثواني ⇒ خط مستقيم.
		if (dt > 0 && m/dt > cfg.MaxSpeedMps) ||
			(dt <= RoadLoopSec && m > RoadLoopFactor*chord+RoadLoopExtraM) {
			return RoadSeg{Kind: SegStraight, M: chord, Line: straight}
		}
		kind := SegRoad
		if chord > RoadInferredM {
			kind = SegInferred
		}
		line := straight
		if m >= 1 && len(c.Line) >= 2 {
			cut := polyline.Cut(c.Line, c.cum, pa, pb)
			line = make([][2]float64, len(cut))
			for i, q := range cut {
				line[i] = [2]float64{q[1], q[0]}
			}
		}
		return RoadSeg{Kind: kind, M: m, Line: line}
	}
	switch {
	case r.covered(a.At, b.At):
		return RoadSeg{Kind: SegStraight, M: chord, Line: straight}
	case r.On:
		return RoadSeg{Kind: SegPending, M: chord, Line: straight}
	}
	return RoadSeg{Kind: SegNone, M: chord, Line: straight}
}

// LoadRoad **يقرأ قطع الطريق المخزّنة لسائق** بالمدى [from, to] — مدخل `Analyze`.
// `on` = المطابقة شغّالة. **وبلا قطع والمطابقة مطفية بيرجّع nil** ⇒ التحليل متل قبل بالضبط.
func LoadRoad(ctx context.Context, db DB, driverID string, from, to time.Time, on bool) (*Road, error) {
	r := &Road{On: on}
	var cf, ct *time.Time
	if err := db.QueryRow(ctx, `
		SELECT (SELECT min(from_at) FROM driver_road_chunks WHERE driver_id = $1::uuid),
		       (SELECT to_at FROM driver_road_chunks WHERE driver_id = $1::uuid ORDER BY from_at DESC LIMIT 1)`,
		driverID).Scan(&cf, &ct); err != nil {
		return nil, err
	}
	if cf == nil || ct == nil {
		if !on {
			return nil, nil
		}
		return r, nil
	}
	r.CoverFrom, r.CoverTo = *cf, *ct
	rows, err := db.Query(ctx, `
		SELECT from_at, to_at, closed, status = 'ok', point_at, pos_m, line6
		  FROM driver_road_chunks
		 WHERE driver_id = $1::uuid AND to_at >= $2 AND from_at <= $3
		 ORDER BY from_at`, driverID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c RoadChunk
		var pos []float32
		var line6 string
		if err := rows.Scan(&c.FromAt, &c.ToAt, &c.Closed, &c.OK, &c.PointAt, &pos, &line6); err != nil {
			return nil, err
		}
		c.PosM = make([]float64, len(pos))
		for i, v := range pos {
			c.PosM[i] = float64(v)
		}
		c.Line = polyline.Decode6(line6)
		r.Chunks = append(r.Chunks, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.prepare()
	return r, nil
}

// NewRoad يبني معلومات الطريق من قطع جاهزة (للاختبارات وللي عنده القطع بالذاكرة).
func NewRoad(chunks []RoadChunk, on bool) *Road {
	r := &Road{Chunks: append([]RoadChunk(nil), chunks...), On: on}
	for _, c := range r.Chunks {
		if r.CoverFrom.IsZero() || c.FromAt.Before(r.CoverFrom) {
			r.CoverFrom = c.FromAt
		}
		if c.ToAt.After(r.CoverTo) {
			r.CoverTo = c.ToAt
		}
	}
	r.prepare()
	return r
}
