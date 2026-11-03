// Package maplearn **يتعلّم الخريطة من حركة السائقين** (قرارُ المالك ٢٠٢٦-١١-٠٣).
//
// نقاطُ `map_trace_points` بلا سائقٍ ولا ساعة. هنا: قياسُ بُعد كلّ نقطةٍ عن أقرب طريقٍ في خريطتنا
// (`roads.go`)، ثمّ تجميعُ البعيدِ المتكرّر في خطوطٍ مقترحة (`detect.go`)، ثمّ حفظُها للموافقة
// (`job.go`) وتصديرُ الموافَق عليه (`export.go`).
package maplearn

import (
	"math"
	"sort"
)

// Point نقطةٌ بعيدةٌ عن الطرق: موضعُها ويومُها.
type Point struct {
	Lat, Lng float64
	Day      string // YYYY-MM-DD
}

// Params عتباتُ الكشف.
type Params struct {
	// CellM ضلعُ خليّة الشبكة بالمتر — نصفُ عرضِ ما يُعدّ طريقاً واحداً تقريباً.
	CellM float64
	// MinCellPoints أقلُّ نقاطٍ لتُعدّ الخليّةُ «مسلوكة» — يسقط الشاردُ المنفرد.
	MinCellPoints int
	// MinPoints أقلُّ نقاطٍ في الخطّ كلِّه.
	MinPoints int
	// MinDays أقلُّ أيّامٍ مختلفةٍ — **حركةٌ في يومٍ واحدٍ قد تكون خطأ GPS أو التفافاً عابراً.**
	MinDays int
	// MinLengthM أقصرُ خطٍّ يُقترَح.
	MinLengthM float64
	// MinLinearShare أقلُّ نسبةٍ من النقاط تقع على امتداد الخطّ — يُسقط البقعَ (موقفٌ أو ساحة).
	MinLinearShare float64
	// SimplifyM تبسيطُ الخطّ (دوغلاس-بوكر).
	SimplifyM float64
}

// DefaultParams العتباتُ المبدئيّة — ويُعدَّلُ منها عند القياس على بياناتٍ حقيقيّة.
func DefaultParams() Params {
	return Params{CellM: 25, MinCellPoints: 2, MinPoints: 12, MinDays: 2, MinLengthM: 60,
		MinLinearShare: 0.6, SimplifyM: 6}
}

// Candidate خطٌّ مقترح.
type Candidate struct {
	Line     [][2]float64 // (lng, lat)
	Points   int
	Days     int
	FirstDay string
	LastDay  string
	LengthM  float64
}

const mPerDegLat = 111320.0

type xy struct{ x, y float64 }

type cellKey struct{ i, j int }

type cell struct {
	n      int
	sx, sy float64
}

// Detect **يجمّع النقاطَ البعيدةَ في خطوط.** خالصةٌ بلا قاعدةٍ ولا شبكة — فتُختبر بنقاطٍ مصطنعة.
func Detect(pts []Point, p Params) []Candidate {
	if len(pts) == 0 {
		return nil
	}
	if p.CellM <= 0 {
		p = DefaultParams()
	}
	// إسقاطٌ مستوٍ حول متوسّط العرض (كافٍ لمدينةٍ).
	var meanLat float64
	for _, q := range pts {
		meanLat += q.Lat
	}
	meanLat /= float64(len(pts))
	mPerDegLng := mPerDegLat * math.Cos(meanLat*math.Pi/180)
	proj := func(q Point) xy { return xy{q.Lng * mPerDegLng, q.Lat * mPerDegLat} }

	// ── ١ · الشبكة ───────────────────────────────────────────────
	cells := map[cellKey]*cell{}
	keyOf := func(v xy) cellKey {
		return cellKey{int(math.Floor(v.x / p.CellM)), int(math.Floor(v.y / p.CellM))}
	}
	for _, q := range pts {
		v := proj(q)
		k := keyOf(v)
		c := cells[k]
		if c == nil {
			c = &cell{}
			cells[k] = c
		}
		c.n++
		c.sx += v.x
		c.sy += v.y
	}
	for k, c := range cells {
		if c.n < p.MinCellPoints {
			delete(cells, k)
		}
	}

	// ── ٢ · المكوّناتُ المتّصلة (ثمانية جيران) ────────────────────
	seen := map[cellKey]bool{}
	var comps [][]cellKey
	for k := range cells {
		if seen[k] {
			continue
		}
		var comp []cellKey
		stack := []cellKey{k}
		seen[k] = true
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			comp = append(comp, cur)
			for di := -1; di <= 1; di++ {
				for dj := -1; dj <= 1; dj++ {
					nk := cellKey{cur.i + di, cur.j + dj}
					if _, ok := cells[nk]; ok && !seen[nk] {
						seen[nk] = true
						stack = append(stack, nk)
					}
				}
			}
		}
		comps = append(comps, comp)
	}

	// ── ٣ · لكلّ مكوّنٍ: حدودُه ثمّ خطُّه ───────────────────────
	compOf := map[cellKey]int{}
	for ci, comp := range comps {
		for _, k := range comp {
			compOf[k] = ci
		}
	}
	type stat struct {
		n    int
		days map[string]bool
	}
	stats := make([]stat, len(comps))
	for i := range stats {
		stats[i].days = map[string]bool{}
	}
	pos := make([][]xy, len(comps)) // نقاطُ كلّ مكوّن (لنسبة الاستقامة)
	for _, q := range pts {
		v := proj(q)
		ci, ok := compOf[keyOf(v)]
		if !ok {
			continue
		}
		stats[ci].n++
		stats[ci].days[q.Day] = true
		pos[ci] = append(pos[ci], v)
	}
	// الأيّامُ الأولى والأخيرة.
	first := make([]string, len(comps))
	last := make([]string, len(comps))
	for _, q := range pts {
		ci, ok := compOf[keyOf(proj(q))]
		if !ok {
			continue
		}
		if first[ci] == "" || q.Day < first[ci] {
			first[ci] = q.Day
		}
		if q.Day > last[ci] {
			last[ci] = q.Day
		}
	}

	var out []Candidate
	for ci, comp := range comps {
		if stats[ci].n < p.MinPoints || len(stats[ci].days) < p.MinDays {
			continue
		}
		path := longestPath(comp)
		line := make([]xy, 0, len(path))
		for _, k := range path {
			c := cells[k]
			line = append(line, xy{c.sx / float64(c.n), c.sy / float64(c.n)})
		}
		line = simplify(line, p.SimplifyM)
		length := polyLen(line)
		if length < p.MinLengthM {
			continue
		}
		// **نسبةُ الاستقامة**: كم من النقاط قريبٌ من الخطّ؟ بقعةٌ عريضةٌ لا تُقترَح طريقاً.
		near := 0
		for _, v := range pos[ci] {
			if distToPolyline(v, line) <= p.CellM*1.5 {
				near++
			}
		}
		if float64(near)/float64(len(pos[ci])) < p.MinLinearShare {
			continue
		}
		ll := make([][2]float64, len(line))
		for i, v := range line {
			ll[i] = [2]float64{v.x / mPerDegLng, v.y / mPerDegLat}
		}
		out = append(out, Candidate{Line: ll, Points: stats[ci].n, Days: len(stats[ci].days),
			FirstDay: first[ci], LastDay: last[ci], LengthM: length})
	}
	// ترتيبٌ ثابت: الأطولُ ثمّ الأكثرُ أيّاماً.
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].LengthM != out[b].LengthM {
			return out[a].LengthM > out[b].LengthM
		}
		return out[a].Days > out[b].Days
	})
	return out
}

// longestPath **أطولُ مسارٍ بين خليّتين** في المكوّن (قطرُ الرسم) — بحثان بالعرض.
// وما تفرّع منه جانباً لا يدخل الخطّ (يُلتقط في جولةٍ لاحقة إن تكرّر).
func longestPath(comp []cellKey) []cellKey {
	in := make(map[cellKey]bool, len(comp))
	for _, k := range comp {
		in[k] = true
	}
	bfs := func(src cellKey) (far cellKey, parent map[cellKey]cellKey) {
		parent = map[cellKey]cellKey{src: src}
		dist := map[cellKey]int{src: 0}
		queue := []cellKey{src}
		far = src
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			if dist[cur] > dist[far] || (dist[cur] == dist[far] && lessKey(cur, far)) {
				far = cur
			}
			for di := -1; di <= 1; di++ {
				for dj := -1; dj <= 1; dj++ {
					nk := cellKey{cur.i + di, cur.j + dj}
					if !in[nk] {
						continue
					}
					if _, ok := dist[nk]; ok {
						continue
					}
					dist[nk] = dist[cur] + 1
					parent[nk] = cur
					queue = append(queue, nk)
				}
			}
		}
		return far, parent
	}
	// **بدايةٌ ثابتة** (أصغرُ مفتاح) فيتكرّر الناتجُ بين جولتين.
	start := comp[0]
	for _, k := range comp {
		if lessKey(k, start) {
			start = k
		}
	}
	u, _ := bfs(start)
	v, parent := bfs(u)
	var path []cellKey
	for cur := v; ; cur = parent[cur] {
		path = append(path, cur)
		if cur == u {
			break
		}
	}
	return path
}

func lessKey(a, b cellKey) bool {
	if a.i != b.i {
		return a.i < b.i
	}
	return a.j < b.j
}

func polyLen(l []xy) float64 {
	var s float64
	for i := 1; i < len(l); i++ {
		s += math.Hypot(l[i].x-l[i-1].x, l[i].y-l[i-1].y)
	}
	return s
}

func distToSeg(p, a, b xy) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	d2 := dx*dx + dy*dy
	if d2 == 0 {
		return math.Hypot(p.x-a.x, p.y-a.y)
	}
	t := ((p.x-a.x)*dx + (p.y-a.y)*dy) / d2
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(p.x-(a.x+t*dx), p.y-(a.y+t*dy))
}

func distToPolyline(p xy, l []xy) float64 {
	if len(l) == 1 {
		return math.Hypot(p.x-l[0].x, p.y-l[0].y)
	}
	best := math.Inf(1)
	for i := 1; i < len(l); i++ {
		if d := distToSeg(p, l[i-1], l[i]); d < best {
			best = d
		}
	}
	return best
}

// simplify دوغلاس-بوكر.
func simplify(l []xy, tol float64) []xy {
	if len(l) < 3 || tol <= 0 {
		return l
	}
	idx, dmax := 0, 0.0
	for i := 1; i < len(l)-1; i++ {
		if d := distToSeg(l[i], l[0], l[len(l)-1]); d > dmax {
			idx, dmax = i, d
		}
	}
	if dmax <= tol {
		return []xy{l[0], l[len(l)-1]}
	}
	left := simplify(l[:idx+1], tol)
	right := simplify(l[idx:], tol)
	return append(append([]xy{}, left...), right[1:]...)
}
