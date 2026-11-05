// Package roadmatch **مطابقة أثر السائق ع الطرق اللي مشي فيها فعلاً** (طلب المالك ٢٠٢٦-١١-٠٥).
//
// «والطريق يحسب حسب طريق السائق مو ع كيفنا».
//
// كل قطعة من `driver_trail` بتنطابق **مرّة وحدة** بنداء Valhalla `trace_attributes` (دراجة نارية،
// **الاتجاه الواحد محترَم**، بلا تفضيلات توجيه) وبيتخزّن موضع كل نقطة ع الخط المطابق والخط نفسه
// بـ`driver_road_chunks`. القراءة بـ`drivermonitor` (`road.go`) — وكل كيلومتر بالمنصّة من هونيك.
//
//   - `chunks.go`    التقطيع: شو منبعت وشو منحتفظ فيه بلا ما نبعته (صافية)
//   - `request.go`   الطلب والعميل وأصناف الأخطاء
//   - `positions.go` من الجواب لموضع كل نقطة ع الطريق (صافية)
//   - `gate.go`      **نداء تتبّع واحد بالمرّة** للمحرّك كله (المطابق وتعلّم الطرق)
//   - `worker.go`    الدورة: مؤشّر حيّ ومؤشّر رجوع، قاطع، والكتابة
//
// **مطفي افتراضياً** (`drivers.road_match_enabled`) — المالك بيشغّله بعد تجربة التجهيز.
// وإطفاؤه بيوقف المطابقة الجديدة بس: اللي انطابق بيضل محسوب.
package roadmatch

import (
	"math"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
)

// ParamsVersion **نسخة قواعد التقطيع والطلب والمواضع.** لما تتغيّر، القطع الأقدم بتنعاد مطابقتها
// بسرعة الرجوع (`worker.go`).
const ParamsVersion = 1

// عتبات التقطيع (B2).
const (
	// MaxSent أكتر نقاط بتنبعت بالنداء الواحد — نداءات قصيرة بتخلّي خيط Valhalla الوحيد فاضي.
	MaxSent = 150
	// MaxChunkDur أطول قطعة.
	MaxChunkDur = 30 * time.Minute
	// HopSplitM قفزة أطول من هيك ⇒ قطع: `breakage_distance` بالمحرّك ٢ كم وما بينغيّر بالطلب.
	HopSplitM = 2000.0
	// JitterMinM و JitterSpeedMps: ضمن max(١٠ م، الدقة) من آخر نقطة مبعوتة وسرعة الجهاز أقل من
	// ١٫٥ م/ث (أو بلا سرعة) ⇒ رجفة واقف، ما بتنبعت.
	JitterMinM     = 10.0
	JitterSpeedMps = 1.5
	// SpikeLegM و SpikeSpeedMps و SpikeReturn: نتوء ⇒ الضلعين أطول من ٥٠ م وأسرع من ١٥ م/ث،
	// والنقطة اللي بعده رجعت ضمن ٤٠٪ من الضلع.
	SpikeLegM     = 50.0
	SpikeSpeedMps = 15.0
	SpikeReturn   = 0.4
)

// Chunk **قطعة أثر للمطابقة**: كل النقاط المنظّفة بالمدى، ومين منها بينبعت.
type Chunk struct {
	Points []drivermonitor.Point
	// Sent النقطة بتنبعت؟ اللي ما بتنبعت (مزيَّفة، رجفة، نتوء) بتاخد موضع اللي قبلها ⇒ صفر متر.
	Sent []bool
	// Closed القطعة خلصت (بعدها انقطاع أو وقفة طويلة أو قفزة أو بلغت الحدّ). لأ = آخر الأثر
	// والسائق ممكن يكمّل ⇒ مطابقة مؤقّتة.
	Closed bool
}

// SentCount كم نقطة بتنبعت.
func (c Chunk) SentCount() int {
	n := 0
	for _, s := range c.Sent {
		if s {
			n++
		}
	}
	return n
}

// From و To وقت أول وآخر نقطة.
func (c Chunk) From() time.Time { return c.Points[0].At }
func (c Chunk) To() time.Time   { return c.Points[len(c.Points)-1].At }

// Chunks **بيقطّع أثر سائق** مرتّب بالزمن (خام من `driver_trail`) — دالة صافية.
//
//  1. نفس تنظيف `Analyze` (`drivermonitor.Clean`).
//  2. قطع عند صمت أطول من `GapAfter`، وعند الوقفات الطويلة (`drivermonitor.LongStops` — نفس وقفات
//     التحليل): القطعة بتخلص بأول نقطة بالوقفة واللي بعدها بتبلّش بآخر نقطة فيها، وعند قفزة > ٢ كم.
//  3. ≤ `MaxSent` نقطة مبعوتة و ≤ `MaxChunkDur` للقطعة (بنقطة مشتركة بين القطعتين).
//
// `until` آخر وقت مقروء: آخر قطعة «خالصة» بس إذا مرّ بعدها أكتر من `GapAfter`.
func Chunks(raw []drivermonitor.Point, cfg drivermonitor.Config, until time.Time) []Chunk {
	if cfg.GapAfter == 0 {
		cfg = drivermonitor.DefaultConfig()
	}
	pts := drivermonitor.Clean(raw, cfg)
	if len(pts) == 0 {
		return nil
	}
	stops := drivermonitor.LongStops(pts, cfg)
	// بداية ونهاية كل وقفة بالفهارس.
	stopStart := map[int]int{} // أول نقطة بالوقفة ← آخر نقطة فيها
	si := 0
	for i := range pts {
		for si < len(stops) && stops[si].Start.Before(pts[i].At) {
			si++
		}
		if si < len(stops) && stops[si].Start.Equal(pts[i].At) {
			for j := i; j < len(pts); j++ {
				if pts[j].At.Equal(stops[si].End) {
					stopStart[i] = j
					break
				}
			}
		}
	}

	var out []Chunk
	cur := Chunk{}
	var lastSent *drivermonitor.Point
	closeCur := func() {
		if len(cur.Points) > 0 {
			cur.Closed = true
			out = append(out, cur)
		}
		cur = Chunk{}
		lastSent = nil
	}
	add := func(i int) {
		p := pts[i]
		send := sendable(pts, i, lastSent)
		cur.Points = append(cur.Points, p)
		cur.Sent = append(cur.Sent, send)
		if send {
			q := p
			lastSent = &q
		}
	}
	for i := 0; i < len(pts); i++ {
		if len(cur.Points) > 0 {
			prev := cur.Points[len(cur.Points)-1]
			p := pts[i]
			switch {
			case p.At.Sub(prev.At) > cfg.GapAfter:
				closeCur()
			case drivermonitor.Dist(prev, p) > HopSplitM:
				closeCur()
			case cur.SentCount() >= MaxSent || p.At.Sub(cur.From()) > MaxChunkDur:
				// قطعة جديدة بتبلّش بآخر نقطة بالقديمة.
				last := cur.Points[len(cur.Points)-1]
				closeCur()
				cur.Points = []drivermonitor.Point{last}
				cur.Sent = []bool{true}
				lastSent = &last
			}
		}
		if j, ok := stopStart[i]; ok {
			// وقفة طويلة: القطعة بتخلص بأولها، واللي بعدها بتبلّش بآخرها.
			add(i)
			closeCur()
			i = j - 1
			continue
		}
		add(i)
	}
	if len(cur.Points) > 0 {
		cur.Closed = until.Sub(cur.To()) > cfg.GapAfter
		out = append(out, cur)
	}
	// أول نقطة بكل قطعة بتنبعت دايماً (منها بيبلّش الخط).
	for k := range out {
		if !out[k].Points[0].Mocked {
			out[k].Sent[0] = true
		}
	}
	return out
}

// sendable النقطة i بتنبعت؟ لا مزيَّفة، ولا رجفة واقف، ولا نتوء.
func sendable(pts []drivermonitor.Point, i int, lastSent *drivermonitor.Point) bool {
	p := pts[i]
	if p.Mocked {
		return false
	}
	if lastSent == nil {
		return true
	}
	// رجفة واقف.
	r := JitterMinM
	if p.AccuracyM != nil {
		r = math.Max(r, *p.AccuracyM)
	}
	if drivermonitor.Dist(*lastSent, p) <= r && (p.SpeedMps == nil || *p.SpeedMps < JitterSpeedMps) {
		return false
	}
	// نتوء: رايح وراجع بسرعة مستحيلة للدرّاج بالمدينة.
	if i+1 < len(pts) {
		n := pts[i+1]
		leg1 := drivermonitor.Dist(*lastSent, p)
		leg2 := drivermonitor.Dist(p, n)
		dt1 := p.At.Sub(lastSent.At).Seconds()
		dt2 := n.At.Sub(p.At).Seconds()
		if leg1 > SpikeLegM && leg2 > SpikeLegM && dt1 > 0 && dt2 > 0 &&
			leg1/dt1 > SpikeSpeedMps && leg2/dt2 > SpikeSpeedMps &&
			drivermonitor.Dist(*lastSent, n) <= SpikeReturn*leg1 {
			return false
		}
	}
	return true
}
