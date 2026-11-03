package maplearn

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
)

// ══════════════════════════════════════════════════════════════════════
// **تعلّم الطرق: اتّجاه واحد وسرعة حقيقية** (قرارُ المالك ٢٠٢٦-١١-٠٣)
// ══════════════════════════════════════════════════════════════════════
//
// الجزءُ الصافي: تنظيفُ أثر السائق وتقطيعُه، تحويلُ المطابقة على الحافّات إلى «عبورات»، كشفُ
// الاتّجاه الواحد، وتصنيفُ البطء. **لا قاعدةَ ولا شبكةَ هنا** — الحافّاتُ تأتي من `Matcher`
// (Valhalla `trace_attributes` في الإنتاج، ومزيَّفٌ في الاختبار). **ولا يُغيَّر التوجيهُ ولا الـETA.**

// RoadParams عتباتُ التنظيف والتقطيع.
type RoadParams struct {
	// MaxAccM أسوأُ دقّةٍ مقبولة؛ ونقطةٌ بلا دقّةٍ معروفةٍ تُهمل.
	MaxAccM float64
	// MinMoveMps أبطأُ من هذا بين نقطتين = واقف (لا يدخل التقطيعَ ولا السرعة).
	MinMoveMps float64
	// MaxMps أسرعُ من هذا = قفزةُ GPS.
	MaxMps float64
	// MaxGap صمتٌ أطولُ منه يقطع المسار.
	MaxGap time.Duration
	// MinPts أقلُّ نقاطِ مقطعٍ يُرسَل للمطابقة.
	MinPts int
	// MaxPts أكثرُ نقاطِ طلبٍ واحد.
	MaxPts int
	// MinEdgeM حافّةٌ أقصرُ من هذا (وصلات التقاطعات) لا تُحسب.
	MinEdgeM float64
}

// DefaultRoadParams الافتراضات.
func DefaultRoadParams() RoadParams {
	return RoadParams{MaxAccM: 20, MinMoveMps: 0.8, MaxMps: 40, MaxGap: 90 * time.Second, MinPts: 4, MaxPts: 300, MinEdgeM: 12}
}

// OneWayParams شروطُ الاقتراح: عبوراتٌ وسائقون وأيّامٌ كافية، واتّجاهٌ يغلب بنسبةٍ عالية.
type OneWayParams struct {
	MinPasses  int
	MinDrivers int
	MinDays    int
	MinShare   float64
}

// DefaultOneWayParams ≥٢٠ عبوراً، ≥٥ سائقين، ≥٥ أيّام، ≥٩٥٪ باتّجاه.
func DefaultOneWayParams() OneWayParams {
	return OneWayParams{MinPasses: 20, MinDrivers: 5, MinDays: 5, MinShare: 0.95}
}

// SlowParams شروطُ «بطيءٌ باستمرار».
type SlowParams struct {
	// MinPasses عبوراتٌ بسرعةٍ مقيسةٍ لا أقلّ على الحافّة كلِّها.
	MinPasses int
	// MinDays أيّامٌ مختلفة.
	MinDays int
	// BucketPasses أقلُّ عبوراتِ خانةِ (ساعة، نهايةِ أسبوع) لتُحسب.
	BucketPasses int
	// MinBuckets أقلُّ عددِ خاناتٍ محسوبة.
	MinBuckets int
	// SlowMps متوسطُ الخانةِ تحتَه = بطيئة (٤ م/ث ≈ ١٤ كم/س).
	SlowMps float64
	// SlowShare حصّةُ الخانات البطيئة التي تجعل الحافّةَ «بطيئةً باستمرار».
	SlowShare float64
}

// DefaultSlowParams الافتراضات.
func DefaultSlowParams() SlowParams {
	return SlowParams{MinPasses: 20, MinDays: 3, BucketPasses: 4, MinBuckets: 3, SlowMps: 4.0, SlowShare: 0.75}
}

// SlowFromCounts **هل الحافّةُ بطيئةٌ باستمرار؟** — من عدّاداتٍ يحسبها SQL (انظر `SpeedLayer`).
// بطيئةٌ في ٧٥٪ فأكثر من خاناتها المحسوبة (لا في ساعةِ ذروةٍ وحدَها)، وبأيّامٍ وعبوراتٍ كافية.
func SlowFromCounts(passes, days, qualifying, slowBuckets int, p SlowParams) bool {
	if passes < p.MinPasses || days < p.MinDays || qualifying < p.MinBuckets {
		return false
	}
	return float64(slowBuckets) >= p.SlowShare*float64(qualifying)
}

// ───────────────────────── الحافّات والمطابقة ─────────────────────────

// MatchedEdge حافّةٌ مطابَقةٌ على الطريق.
type MatchedEdge struct {
	// Key مفتاحٌ ثابتٌ للحافّة (انظر CanonEdge).
	Key    string
	WayID  int64
	Name   string
	Length float64
	// Dir اتّجاهُ العبور بالنسبة للترتيب القانوني: ١ أو -١.
	Dir int
	// OneWayKnown الخريطةُ تعرفها باتّجاهٍ واحد أصلاً — لا تُقترح.
	OneWayKnown bool
	// Line هندسةُ الحافّة بالترتيب القانوني (lng,lat).
	Line [][2]float64
}

// MatchResult حصيلةُ مطابقة مقطعٍ واحد.
type MatchResult struct {
	Edges []MatchedEdge
	// PointEdge لكلّ نقطةِ إدخالٍ فهرسُ حافّتها في Edges، أو -١.
	PointEdge []int
}

// ErrNoMatch لا مطابقةَ لهذا المقطع (نقاطٌ بعيدةٌ عن الطرق) — يُتخطّى ولا يُعدّ عطلاً.
var ErrNoMatch = errors.New("maplearn: لا مطابقة")

// Matcher يطابق مقطعاً مرتّباً بالزمن على الطرق.
type Matcher interface {
	Match(ctx context.Context, pts []drivermonitor.Point) (MatchResult, error)
}

// CanonEdge **مفتاحُ الحافّة واتّجاهُ العبور.** الترتيبُ القانونيُّ: أيُّ طرفَي الخطّ أصغرُ (خطّ العرض ثم الطول
// بخمس منازل). فتُعدّ العبورات باتّجاهَين على حافّةٍ واحدةٍ بمفتاحٍ واحد.
// `travel` هندسةُ العبور بترتيب السير (lng,lat). تردّ المفتاح والاتّجاه (١ = السيرُ بالترتيب القانوني) والخطَّ قانونيّاً.
func CanonEdge(wayID int64, travel [][2]float64) (key string, dir int, canon [][2]float64) {
	if len(travel) < 2 {
		return "", 1, travel
	}
	first, last := travel[0], travel[len(travel)-1]
	r := func(v float64) float64 { return math.Round(v*1e5) / 1e5 }
	dir = 1
	if r(last[1]) < r(first[1]) || (r(last[1]) == r(first[1]) && r(last[0]) < r(first[0])) {
		dir = -1
	}
	canon = travel
	if dir == -1 {
		canon = make([][2]float64, len(travel))
		for i, p := range travel {
			canon[len(travel)-1-i] = p
		}
	}
	a, b := canon[0], canon[len(canon)-1]
	return fmt.Sprintf("%d|%.5f,%.5f|%.5f,%.5f", wayID, r(a[1]), r(a[0]), r(b[1]), r(b[0])), dir, canon
}

// Chunks **ينظّف أثرَ سائقٍ ويقطّعه مقاطعَ للمطابقة**: يُسقط المزيَّفَ وغيرَ الدقيق (> MaxAccM أو بلا دقّة) وقفزاتِ GPS
// والوقوف، ويقطع عند كلّ صمتٍ أطولَ من MaxGap، ويرمي القصيرَ جدّاً. النقاطُ مرتّبةٌ بالزمن.
func Chunks(pts []drivermonitor.Point, p RoadParams) [][]drivermonitor.Point {
	var out [][]drivermonitor.Point
	var cur []drivermonitor.Point
	flush := func() {
		if len(cur) >= p.MinPts {
			out = append(out, cur)
		}
		cur = nil
	}
	for _, pt := range pts {
		if pt.Mocked || pt.AccuracyM == nil || *pt.AccuracyM > p.MaxAccM {
			continue
		}
		if len(cur) > 0 {
			prev := cur[len(cur)-1]
			dt := pt.At.Sub(prev.At)
			if dt <= 0 {
				continue
			}
			if dt > p.MaxGap {
				flush()
			} else {
				v := drivermonitor.Dist(prev, pt) / dt.Seconds()
				if v < p.MinMoveMps {
					continue // واقف
				}
				if v > p.MaxMps {
					continue // قفزة
				}
			}
		}
		cur = append(cur, pt)
		if len(cur) >= p.MaxPts {
			flush()
		}
	}
	flush()
	return out
}

// Pass عبورٌ واحد: سائقٌ مرَّ على حافّةٍ باتّجاه، ومعه (إن قُيست) سرعتُه وساعتُه.
type Pass struct {
	Edge     int // فهرس في MatchResult.Edges
	Dir      int
	HasSpeed bool
	SpeedMps float64
	Hour     int  // بتوقيت دمشق
	Weekend  bool // الجمعة والسبت
}

// IsWeekend نهايةُ الأسبوع في سوريا: الجمعة والسبت.
func IsWeekend(t time.Time) bool {
	d := t.In(drivermonitor.Loc()).Weekday()
	return d == time.Friday || d == time.Saturday
}

// Passes **يحوّل مقطعاً مطابَقاً إلى عبورات.** كلُّ حافّةٍ طويلةٍ كفايةً في المطابقة = عبور. والسرعةُ: لكلّ زوجِ نقطتين
// متتاليتين مسافةٌ/زمن، تُنسب إلى كلّ الحافّات بين حافّتَي النقطتين؛ وعبورُ الحافّة يأخذ متوسّطَ ما نُسب إليها،
// وساعتُه ساعةُ أوّلِ ما نُسب. النقاطُ هنا هي نفسُها المرسَلة للمطابقة (بعد `Chunks`).
func Passes(pts []drivermonitor.Point, res MatchResult, p RoadParams) []Pass {
	n := len(res.Edges)
	if n == 0 || len(res.PointEdge) != len(pts) {
		return nil
	}
	sum := make([]float64, n)
	cnt := make([]int, n)
	first := make([]time.Time, n)
	for i := 1; i < len(pts); i++ {
		e0, e1 := res.PointEdge[i-1], res.PointEdge[i]
		if e0 < 0 || e1 < 0 || e0 >= n || e1 >= n {
			continue
		}
		dt := pts[i].At.Sub(pts[i-1].At)
		if dt <= 0 || dt > p.MaxGap {
			continue
		}
		v := drivermonitor.Dist(pts[i-1], pts[i]) / dt.Seconds()
		lo, hi := e0, e1
		if lo > hi {
			lo, hi = hi, lo
		}
		for e := lo; e <= hi; e++ {
			if cnt[e] == 0 {
				first[e] = pts[i-1].At
			}
			sum[e] += v
			cnt[e]++
		}
	}
	var out []Pass
	for i, e := range res.Edges {
		if e.Key == "" || e.Length < p.MinEdgeM || (e.Dir != 1 && e.Dir != -1) {
			continue
		}
		ps := Pass{Edge: i, Dir: e.Dir}
		if cnt[i] > 0 {
			ps.HasSpeed = true
			ps.SpeedMps = sum[i] / float64(cnt[i])
			ps.Hour = first[i].In(drivermonitor.Loc()).Hour()
			ps.Weekend = IsWeekend(first[i])
		}
		out = append(out, ps)
	}
	return out
}

// ───────────────────────── كشفُ الاتّجاه الواحد ─────────────────────────

// EdgeDirStat إحصاءُ حافّةٍ عبر السائقين والأيّام.
type EdgeDirStat struct {
	Key      string
	Fwd, Rev int // عبوراتُ الاتّجاه القانوني ومعاكسِه
	Drivers  int
	Days     int
}

// OneWayFinding حافّةٌ يُقترح أنّها باتّجاه واحد.
type OneWayFinding struct {
	Key     string
	Dir     int // اتّجاهُ الغلبة: ١ = القانوني
	Main    int
	Opp     int
	Drivers int
	Days    int
	Share   float64
}

// DetectOneWay **حافّاتٌ يعبرها ≥ ٩٥٪ باتّجاه واحد** بعبوراتٍ وسائقين وأيّامٍ كافية. صافية، مرتّبةٌ بالعبورات.
func DetectOneWay(stats []EdgeDirStat, p OneWayParams) []OneWayFinding {
	var out []OneWayFinding
	for _, s := range stats {
		total := s.Fwd + s.Rev
		if total < p.MinPasses || s.Drivers < p.MinDrivers || s.Days < p.MinDays {
			continue
		}
		main, opp, dir := s.Fwd, s.Rev, 1
		if s.Rev > s.Fwd {
			main, opp, dir = s.Rev, s.Fwd, -1
		}
		share := float64(main) / float64(total)
		if share < p.MinShare {
			continue
		}
		out = append(out, OneWayFinding{Key: s.Key, Dir: dir, Main: main, Opp: opp, Drivers: s.Drivers, Days: s.Days, Share: share})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Main != out[j].Main {
			return out[i].Main > out[j].Main
		}
		return out[i].Key < out[j].Key
	})
	return out
}
