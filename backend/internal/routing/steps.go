package routing

import (
	"math"
	"strings"
)

// ══════════════════════════════════════════════════════════════════════
// **بناءُ المسار من خطواته — مرّةً واحدةً عند التحويل**
// ══════════════════════════════════════════════════════════════════════
//
// (المرحلة ٢، تصحيحُ المالك ٢٠٢٦-٠٨-٢٠: «أريد بناءَ `atDistanceM`
//
//	بطريقةٍ ثابتةٍ أثناء تحويل Route من OSRM، **وليس عن طريق بحثٍ
//	عشوائيٍّ متكرّرٍ عن إحداثيّة المناورة داخل Overview Geometry**».)
//
// # ولماذا تُبنى الهندسةُ من الخطوات لا من `overview`
//
// **`overview=full` خطٌّ مبسَّطٌ قليلاً، وهندسةُ الخطوات هي الأصل** —
// **وبينهما فروقُ رؤوسٍ صغيرة.** فلو أخذنا الخطَّ من الأولى والمسافاتِ
// من الثانية **لَما تطابقا**، وصار `atDistanceM` يشير إلى موضعٍ ليس
// عليه.
//
// **فالخطُّ والمسافاتُ والمناوراتُ كلُّها من مصدرٍ واحد** — والاتّساقُ
// مضمونٌ بالبناء لا بالرجاء.
//
// # وأوّلُ رأسٍ في كلّ خطوةٍ هو آخرُ رأسٍ في سابقتها
//
// **ومن ألصقهما بلا حذفٍ** كرّر الرؤوسَ فزادت المسافةُ صفراً في كلّ
// مفصل — **ورقمٌ لا يتغيّر لا يُكتشف بالعين.**

// osrmStep خطوةٌ واحدةٌ كما يردّها المحرّك — **وما لا نستعمله لا يُفكّ.**
type osrmStep struct {
	Name       string  `json:"name"`
	RotaryName string  `json:"rotary_name"`
	Distance   float64 `json:"distance"`
	Duration   float64 `json:"duration"`
	Geometry   struct {
		Coordinates [][]float64 `json:"coordinates"`
	} `json:"geometry"`
	Maneuver struct {
		Type     string  `json:"type"`
		Modifier *string `json:"modifier"`
		Exit     *int    `json:"exit"`
	} `json:"maneuver"`
}

// buildFromSteps يبني الخطَّ والمسافاتِ والمناورات — أو `nil`.
//
// **ويردّ `nil` حين لا خطوات** — فيعود المنادي إلى الخطّ الإجماليّ.
func buildFromSteps(steps []osrmStep) *Route {
	if len(steps) == 0 {
		return nil
	}
	out := &Route{}
	// **والرأسُ الأخيرُ من كلّ خطوةٍ يُترك لأوّل التالية** — انظر أعلاه.
	for _, st := range steps {
		coords := st.Geometry.Coordinates
		for i, c := range coords {
			if len(c) < 2 {
				continue
			}
			p := Point{Lat: c[1], Lng: c[0]}
			if i == 0 && len(out.Geometry) > 0 && samePoint(out.Geometry[len(out.Geometry)-1], p) {
				continue
			}
			out.Geometry = append(out.Geometry, p)
		}
	}
	if len(out.Geometry) < 2 {
		return nil
	}

	// **والتراكميّةُ تُحسب مرّةً** — وبها يصير المتبقّي طرحاً واحداً.
	out.CumulativeM = make([]float64, len(out.Geometry))
	for i := 1; i < len(out.Geometry); i++ {
		out.CumulativeM[i] = out.CumulativeM[i-1] +
			MetersBetween(out.Geometry[i-1], out.Geometry[i])
	}

	// **وموضعُ كلّ مناورةٍ هو أوّلُ رأسٍ في خطوتها** — يُعرف بالبناء
	// لا بالبحث.
	idx := 0
	for _, st := range steps {
		m := Maneuver{
			Kind:          mapKind(st.Maneuver.Type, st.Maneuver.Modifier),
			Modifier:      st.Maneuver.Modifier,
			AtIndex:       idx,
			AtDistanceM:   out.CumulativeM[minInt(idx, len(out.CumulativeM)-1)],
			StepDistanceM: st.Distance,
			StepDurationS: st.Duration,
		}
		if n := strings.TrimSpace(st.Name); n != "" {
			m.StreetName = &n
		}
		if n := strings.TrimSpace(st.RotaryName); n != "" {
			m.RoundaboutName = &n
		}
		// **ورقمُ المخرج يُؤخذ للدوّار وحدَه** — `exit` تظهر في أنواعٍ
		// أخرى بمعنىً مختلف، **ورقمٌ يُعرض في غير موضعه يُربك.**
		if st.Maneuver.Exit != nil &&
			(m.Kind == KindRoundabout || m.Kind == KindExitRoundabout) {
			e := *st.Maneuver.Exit
			m.RoundaboutExit = &e
		}
		out.Maneuvers = append(out.Maneuvers, m)

		// **ويُقدَّم الفهرسُ بعدد رؤوس الخطوة** — ناقصاً الرأسَ
		// المشترك.
		n := countNew(st, idx, out.Geometry)
		idx += n
		if idx > len(out.Geometry)-1 {
			idx = len(out.Geometry) - 1
		}
	}
	return out
}

// countNew كم رأساً أضافت هذه الخطوة.
func countNew(st osrmStep, at int, geom []Point) int {
	c := st.Geometry.Coordinates
	if len(c) == 0 {
		return 0
	}
	n := len(c)
	// **والرأسُ الأوّلُ مشتركٌ إلّا في أوّل خطوة.**
	if at > 0 {
		n--
	}
	if n < 0 {
		n = 0
	}
	return n
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func samePoint(a, b Point) bool {
	return math.Abs(a.Lat-b.Lat) < 1e-9 && math.Abs(a.Lng-b.Lng) < 1e-9
}

// MetersBetween المسافةُ بالأمتار — هافرساين.
//
// **وتُصدَّر لأنّ الاختبارَ يقيس بها ما بناه التحويل** — ومقياسان
// مختلفان يجعلان الاختبارَ يقيس نفسَه.
func MetersBetween(a, b Point) float64 {
	const r = 6371000.0
	p1 := a.Lat * math.Pi / 180
	p2 := b.Lat * math.Pi / 180
	dp := (b.Lat - a.Lat) * math.Pi / 180
	dl := (b.Lng - a.Lng) * math.Pi / 180
	h := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * r * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}
