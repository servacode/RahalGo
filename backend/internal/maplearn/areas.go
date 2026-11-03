package maplearn

// ══════════════════════════════════════════════════════════════════════
// **أسماءُ الأحياء والشوارع من عناوين الزبائن** (قرارُ المالك ٢٠٢٦-١١-٠٣:
// «بالضبط من عناوين الزباين راح نستفيد من اسماء الاحياء والشوارع»)
// ══════════════════════════════════════════════════════════════════════
//
// كاشفان صافيان فوق نصوص العناوين ونقاط التسليم، لا يلمسان القاعدة:
//
//   ١ · الأحياء — «حي X» / «حارة X» / «منطقة X» (وبـ«ال» وبدونها وبالهمزات والتاء المربوطة…):
//       تُجمَّع نقاطُ كلِّ اسمٍ (ربطٌ بالجوار ≤ ٤٠٠ م)، ويُقبل ما ذكره **٣ زبائن مختلفين** فأكثر،
//       ومنه: اسمٌ + نقطةُ تسمية (الأوسط الحقيقيّ medoid) + امتدادٌ (الغلافُ المحدَّب) + عدد الأدلّة.
//   ٢ · الشوارع — «شارع X» / «طريق X» / «اوتوستراد X»: تُلصق نقاطُ التسليم بأقرب حافّة طريقٍ
//       (`Locator`)، ويُختار الطريقُ (أو الطرقُ) الذي يتّفق عليه **زبونان مختلفان** فأكثر وبحصّةٍ أغلبيّة.
//
// الحفظُ والاستبعادُ (الموجودُ في البلاطات أو المقرَّرُ عليه) في `areas_job.go`.

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
)

// أنواعُ المرشَّحين.
const (
	AreaHood   = "neighbourhood"
	AreaStreet = "street"
)

// AreaParams عتبات الكشف.
type AreaParams struct {
	// HoodLinkM بعدُ ربط نقطتَي تسليمٍ لاسمٍ واحدٍ في حيٍّ واحد.
	HoodLinkM float64
	// HoodMinCustomers أقلُّ زبائنَ مختلفين يذكرون الحيَّ.
	HoodMinCustomers int
	// StreetMinCustomers أقلُّ زبائنَ مختلفين يلصقهم الطريقُ نفسُه.
	StreetMinCustomers int
	// SnapM أبعدُ مسافةٍ بين نقطة تسليمٍ وأقرب طريقٍ ليُحسب لصقاً.
	SnapM float64
	// MinShare أقلُّ حصّةٍ من الملصَقين تقع على الطرق المختارة (أغلبيّة).
	MinShare float64
	// MaxLocatePerStreet سقفُ نقاط الاسم الواحد في الطلب.
	MaxLocatePerStreet int
	// MaxLocatePerRun سقفُ كلّ النداءات في الجولة.
	MaxLocatePerRun int
}

// DefaultAreaParams العتبات المعتمدة.
func DefaultAreaParams() AreaParams {
	return AreaParams{HoodLinkM: 400, HoodMinCustomers: 3, StreetMinCustomers: 2, SnapM: 80,
		MinShare: 0.5, MaxLocatePerStreet: 40, MaxLocatePerRun: 3000}
}

func (p AreaParams) filled() AreaParams {
	d := DefaultAreaParams()
	if p.HoodLinkM <= 0 {
		p.HoodLinkM = d.HoodLinkM
	}
	if p.HoodMinCustomers <= 0 {
		p.HoodMinCustomers = d.HoodMinCustomers
	}
	if p.StreetMinCustomers <= 0 {
		p.StreetMinCustomers = d.StreetMinCustomers
	}
	if p.SnapM <= 0 {
		p.SnapM = d.SnapM
	}
	if p.MinShare <= 0 {
		p.MinShare = d.MinShare
	}
	if p.MaxLocatePerStreet <= 0 {
		p.MaxLocatePerStreet = d.MaxLocatePerStreet
	}
	if p.MaxLocatePerRun <= 0 {
		p.MaxLocatePerRun = d.MaxLocatePerRun
	}
	return p
}

// ══════════════════════════════════════════════════════════════════════
// الاستخراج
// ══════════════════════════════════════════════════════════════════════

// areaTypes كلماتُ نوع الموضع (بعد التوحيد وحذف «ال») ونوعُها.
var areaTypes = map[string][2]string{
	"حي":        {AreaHood, "حي"},
	"حاره":      {AreaHood, "حاره"},
	"منطقه":     {AreaHood, "منطقه"},
	"شارع":      {AreaStreet, "شارع"},
	"طريق":      {AreaStreet, "طريق"},
	"اوتوستراد": {AreaStreet, "اوتوستراد"},
	"اوتستراد":  {AreaStreet, "اوتوستراد"},
	"اتوستراد":  {AreaStreet, "اوتوستراد"},
	"اتستراد":   {AreaStreet, "اوتوستراد"},
}

// genericAreaNames أسماءٌ لا تُسمّي موضعاً («الشارع الرئيسي»).
var genericAreaNames = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range []string{"رئيسي", "فرعي", "عام", "جانبي", "خلفي", "اول", "ثاني", "ثالث", "رابع", "خامس",
		"هذا", "هاد", "هادا", "هاي", "هذه", "ذاك", "نفس", "نفسه", "كبير", "صغير", "ضيق", "واسع", "طويل", "جديد"} {
		m[Normalize(w)] = true
	}
	return m
}()

const maxAreaTokens = 3

// areaTypeOf يعرف كلمةَ النوع بما يلصق بها من «و/ب/ل» و«ال».
func areaTypeOf(nt string) (kind, typ string, ok bool) {
	cands := []string{nt}
	if r := []rune(nt); len(r) > 2 && (r[0] == 'و' || r[0] == 'ب' || r[0] == 'ل') {
		cands = append(cands, string(r[1:]))
	}
	for _, c := range cands {
		for _, v := range []string{c, stripAl(c)} {
			if t, ok := areaTypes[v]; ok {
				return t[0], t[1], true
			}
		}
	}
	return "", "", false
}

// AreaHit موضعٌ مسمّى وُجد في نصّ.
type AreaHit struct {
	Kind string   // neighbourhood | street
	Type string   // حي · حاره · منطقه · شارع · طريق · اوتوستراد
	Disp []string // كلماتُ الاسم كما كتبها الزبون (حتى ٣)
	Norm []string // موحَّدةً بلا «ال»
}

// prefixKey مفتاحُ أوّل n كلمات.
func (h AreaHit) prefixKey(n int) string {
	if n > len(h.Norm) {
		n = len(h.Norm)
	}
	return strings.Join(h.Norm[:n], " ")
}

func (h AreaHit) prefixDisp(n int) string {
	if n > len(h.Disp) {
		n = len(h.Disp)
	}
	return strings.Join(h.Disp[:n], " ")
}

// ExtractAreas يستخرج الأحياء والشوارع من نصّ عنوانٍ أو ملاحظة.
func ExtractAreas(text string) []AreaHit {
	disp := make([]rune, 0, len(text))
	for _, r := range cleanDisplay(text) {
		if strings.ContainsRune(breakRunes, r) {
			disp = append(disp, ' ', '|', ' ')
		} else {
			disp = append(disp, r)
		}
	}
	dtoks := strings.Fields(string(disp))
	ntoks := make([]string, len(dtoks))
	for i, t := range dtoks {
		ntoks[i] = Normalize(t)
	}
	var out []AreaHit
	for i := 0; i < len(ntoks); i++ {
		kind, typ, ok := areaTypeOf(ntoks[i])
		if !ok {
			continue
		}
		j := i + 1
		if j < len(ntoks) && ntoks[j] == "|" { // «حي: الدرعية» · «شارع - الوادي»
			j++
		}
		var dn, nn []string
		for ; j < len(ntoks) && len(dn) < maxAreaTokens; j++ {
			t := ntoks[j]
			if t == "|" || prefixWords[t] || prefixWords[strings.TrimPrefix(t, "و")] || stopWords[t] ||
				stopWords[stripAl(t)] || joinedStop(t) || strings.ContainsAny(t, "0123456789") {
				break
			}
			if _, _, isType := areaTypeOf(t); isType {
				break
			}
			if len(dn) > 0 {
				if _, isKind := typeKinds[stripAl(t)]; isKind {
					break
				}
			}
			dn = append(dn, dtoks[j])
			nn = append(nn, stripAl(t))
		}
		if len(nn) == 0 || genericAreaNames[nn[0]] || len([]rune(nn[0])) < 2 {
			continue
		}
		out = append(out, AreaHit{Kind: kind, Type: typ, Disp: dn, Norm: nn})
		i = j - 1
	}
	return out
}

// groupKey مفتاحُ التجميع الأوّل: الحيُّ بأوّل كلمة، والشارعُ بنوعه وأوّل كلمة.
func (h AreaHit) groupKey() string {
	if h.Kind == AreaStreet {
		return h.Type + " " + h.Norm[0]
	}
	return h.Norm[0]
}

// ══════════════════════════════════════════════════════════════════════
// دمجُ الأخطاء الإملائيّة
// ══════════════════════════════════════════════════════════════════════

// withinOneEdit هل بين الكلمتين حرفٌ واحدٌ فأقلّ من الفرق (إبدالٌ أو حذفٌ أو زيادة)؟
func withinOneEdit(a, b string) bool {
	if a == b {
		return true
	}
	ra, rb := []rune(a), []rune(b)
	if len(ra) < len(rb) {
		ra, rb = rb, ra
	}
	if len(ra)-len(rb) > 1 {
		return false
	}
	i := 0
	for i < len(rb) && ra[i] == rb[i] {
		i++
	}
	if len(ra) == len(rb) {
		return string(ra[i+1:]) == string(rb[i+1:])
	}
	return string(ra[i+1:]) == string(rb[i:])
}

// mergeNearKeys يردّ لكلّ مفتاحٍ مفتاحَه القانونيّ: الأكثرُ تكراراً بين المتقاربين
// (فرقُ حرفٍ واحد في كلمةٍ بطول ٥ فأكثر وبالحرف الأوّل نفسِه).
func mergeNearKeys(counts map[string]int) map[string]string {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		return keys[i] < keys[j]
	})
	canon := map[string]string{}
	var heads []string
	for _, k := range keys {
		hit := ""
		if len([]rune(k)) >= 5 {
			for _, h := range heads {
				if []rune(h)[0] == []rune(k)[0] && withinOneEdit(h, k) {
					hit = h
					break
				}
			}
		}
		if hit == "" {
			heads = append(heads, k)
			hit = k
		}
		canon[k] = hit
	}
	return canon
}

// ══════════════════════════════════════════════════════════════════════
// أدواتٌ هندسيّة
// ══════════════════════════════════════════════════════════════════════

// medoid أقربُ نقطةٍ فعليّةٍ إلى الباقين (مركزٌ متينٌ لا يجرّه شاذّ). فوق ٤٠٠ نقطةٍ يُؤخذ عيّنةٌ منتظمة.
func medoid(pts []AddrEvidence) int {
	cand := pts
	step := 1
	if len(pts) > 400 {
		step = (len(pts) + 399) / 400
	}
	best, bestSum := 0, math.MaxFloat64
	for i := 0; i < len(cand); i += step {
		sum := 0.0
		for j := 0; j < len(pts); j += step {
			sum += distM(cand[i].Lat, cand[i].Lng, pts[j].Lat, pts[j].Lng)
		}
		if sum < bestSum {
			best, bestSum = i, sum
		}
	}
	return best
}

// convexHull الغلافُ المحدَّب لنقاط (lng,lat) — سلسلة أندرو. أقلُّ من ٣ رؤوسٍ = nil. مغلقٌ (الأوّل = الأخير).
func convexHull(pts [][2]float64) [][2]float64 {
	if len(pts) < 3 {
		return nil
	}
	p := append([][2]float64(nil), pts...)
	sort.Slice(p, func(i, j int) bool {
		if p[i][0] != p[j][0] {
			return p[i][0] < p[j][0]
		}
		return p[i][1] < p[j][1]
	})
	cross := func(o, a, b [2]float64) float64 {
		return (a[0]-o[0])*(b[1]-o[1]) - (a[1]-o[1])*(b[0]-o[0])
	}
	var h [][2]float64
	for _, q := range p {
		for len(h) >= 2 && cross(h[len(h)-2], h[len(h)-1], q) <= 0 {
			h = h[:len(h)-1]
		}
		h = append(h, q)
	}
	lo := len(h) + 1
	for i := len(p) - 2; i >= 0; i-- {
		for len(h) >= lo && cross(h[len(h)-2], h[len(h)-1], p[i]) <= 0 {
			h = h[:len(h)-1]
		}
		h = append(h, p[i])
	}
	if len(h) < 4 { // نقاطٌ على استقامةٍ واحدة
		return nil
	}
	return h
}

// links يربط النقاط المتقاربة (≤ linkM) مكوّناتٍ متّصلة — بحثٌ بالاتحاد على فهرسٍ مكانيّ.
func links(ev []AddrEvidence, linkM float64) [][]int {
	idx := newSpatialIndex(linkM)
	for i, e := range ev {
		idx.add(i, e.Lat, e.Lng)
	}
	parent := make([]int, len(ev))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	for i, e := range ev {
		for _, j := range idx.near(e.Lat, e.Lng) {
			if j > i && distM(e.Lat, e.Lng, ev[j].Lat, ev[j].Lng) <= linkM {
				parent[find(j)] = find(i)
			}
		}
	}
	comps := map[int][]int{}
	for i := range ev {
		r := find(i)
		comps[r] = append(comps[r], i)
	}
	roots := make([]int, 0, len(comps))
	for r := range comps {
		roots = append(roots, r)
	}
	sort.Ints(roots)
	out := make([][]int, 0, len(roots))
	for _, r := range roots {
		out = append(out, comps[r])
	}
	return out
}

// ══════════════════════════════════════════════════════════════════════
// الأحياء
// ══════════════════════════════════════════════════════════════════════

// HoodCluster حيٌّ اتّفق على اسمه ٣ زبائنَ فأكثر.
type HoodCluster struct {
	Name      string
	Key       string
	Lat, Lng  float64      // نقطةُ التسمية
	RadiusM   float64      // أبعدُ نقطةٍ مقبولةٍ عن نقطة التسمية
	Hull      [][2]float64 // (lng,lat) مغلقٌ — nil إن قلّت النقاط
	Customers int
	Evidence  int
	Samples   []string
}

type areaItem struct {
	hit AreaHit
	ev  AddrEvidence
}

// bestPrefix يختار الاسمَ الأطول الذي يتّفق عليه أغلبُ الأعضاء (٦٠٪ فأكثر وبعدد الزبائن الأدنى):
// «الرقة القديمة» إن كتبها أغلبُهم، وإلّا «الرقة».
func bestPrefix(items []areaItem, minCustomers int) (key, display string) {
	for n := maxAreaTokens; n >= 1; n-- {
		cnt := map[string]int{}
		custs := map[string]map[string]bool{}
		forms := map[string]map[string]int{}
		for _, it := range items {
			if len(it.hit.Norm) < n {
				continue
			}
			k := it.hit.prefixKey(n)
			cnt[k]++
			if custs[k] == nil {
				custs[k] = map[string]bool{}
				forms[k] = map[string]int{}
			}
			custs[k][it.ev.Customer] = true
			forms[k][it.hit.prefixDisp(n)]++
		}
		bk, bn := "", 0
		for k, c := range cnt {
			if c > bn || (c == bn && k < bk) {
				bk, bn = k, c
			}
		}
		if bk == "" {
			continue
		}
		if n > 1 && (float64(bn) < 0.6*float64(len(items)) || len(custs[bk]) < minCustomers) {
			continue
		}
		bd, bdn := "", -1
		for f, c := range forms[bk] {
			if c > bdn || (c == bdn && (len(f) < len(bd) || (len(f) == len(bd) && f < bd))) {
				bd, bdn = f, c
			}
		}
		return bk, bd
	}
	return "", ""
}

func sampleSet(items []areaItem, limit int) []string {
	var out []string
	seen := map[string]bool{}
	for _, it := range items {
		if s := SanitizeSample(it.ev.Text); s != "" && !seen[s] && len(out) < limit {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func customersOf(items []areaItem) map[string]bool {
	m := map[string]bool{}
	for _, it := range items {
		m[it.ev.Customer] = true
	}
	return m
}

// groupAreas يجمع أدلّة نوعٍ واحدٍ بمفتاحها الأوّل بعد دمج الأخطاء الإملائيّة، ثم يردّ المجموعات مرتّبةً.
func groupAreas(ev []AddrEvidence, kind string) (groups map[string][]areaItem, order []string) {
	raw := map[string][]areaItem{}
	counts := map[string]int{}
	for _, e := range ev {
		seen := map[string]bool{}
		for _, h := range ExtractAreas(e.Text) {
			if h.Kind != kind {
				continue
			}
			k := h.groupKey()
			if seen[k] {
				continue
			}
			seen[k] = true
			raw[k] = append(raw[k], areaItem{h, e})
			counts[k]++
		}
	}
	canon := mergeNearKeys(counts)
	groups = map[string][]areaItem{}
	for k, items := range raw {
		groups[canon[k]] = append(groups[canon[k]], items...)
	}
	for k := range groups {
		order = append(order, k)
	}
	sort.Strings(order)
	return groups, order
}

// ClusterHoods يجمع كلَّ اسمٍ على نقاط تسليمه المتّصلة ويُبقي ما ذكره `HoodMinCustomers` زبائنَ فأكثر.
func ClusterHoods(ev []AddrEvidence, p AreaParams) []HoodCluster {
	p = p.filled()
	groups, order := groupAreas(ev, AreaHood)
	var out []HoodCluster
	for _, gk := range order {
		items := groups[gk]
		pts := make([]AddrEvidence, len(items))
		for i, it := range items {
			pts[i] = it.ev
		}
		for _, comp := range links(pts, p.HoodLinkM) {
			cItems := make([]areaItem, len(comp))
			cPts := make([]AddrEvidence, len(comp))
			for i, ci := range comp {
				cItems[i], cPts[i] = items[ci], pts[ci]
			}
			custs := customersOf(cItems)
			if len(custs) < p.HoodMinCustomers {
				continue
			}
			m := medoid(cPts)
			dists := make([]float64, len(cPts))
			for i, q := range cPts {
				dists[i] = distM(cPts[m].Lat, cPts[m].Lng, q.Lat, q.Lng)
			}
			sorted := append([]float64(nil), dists...)
			sort.Float64s(sorted)
			limit := math.Max(3*sorted[len(sorted)/2], 250)
			var hullIn [][2]float64
			radius := 0.0
			for i, q := range cPts {
				if dists[i] > limit {
					continue
				}
				hullIn = append(hullIn, [2]float64{q.Lng, q.Lat})
				radius = math.Max(radius, dists[i])
			}
			key, name := bestPrefix(cItems, p.HoodMinCustomers)
			if key == "" {
				continue
			}
			out = append(out, HoodCluster{Name: name, Key: key, Lat: cPts[m].Lat, Lng: cPts[m].Lng,
				RadiusM: radius, Hull: convexHull(hullIn), Customers: len(custs), Evidence: len(cItems),
				Samples: sampleSet(cItems, 4)})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Customers != out[j].Customers {
			return out[i].Customers > out[j].Customers
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Lat < out[j].Lat
	})
	return out
}

// ══════════════════════════════════════════════════════════════════════
// الشوارع
// ══════════════════════════════════════════════════════════════════════

// StreetGroup اسمُ شارعٍ ذكره زبونان فأكثر، ونقاطُ تسليمهم.
type StreetGroup struct {
	Name      string // «شارع الوادي»
	Key       string // النوعُ + الاسمُ موحَّدين
	Bare      string // الاسمُ موحَّداً بلا النوع
	Type      string
	Points    []AddrEvidence
	Customers int
	Evidence  int
	Samples   []string
}

// GroupStreets يجمع أسماءَ الشوارع (بلا مكان — المكانُ من اللصق على الطرق).
func GroupStreets(ev []AddrEvidence, p AreaParams) []StreetGroup {
	p = p.filled()
	groups, order := groupAreas(ev, AreaStreet)
	var out []StreetGroup
	for _, gk := range order {
		items := groups[gk]
		custs := customersOf(items)
		if len(custs) < p.StreetMinCustomers {
			continue
		}
		key, name := bestPrefix(items, p.StreetMinCustomers)
		if key == "" {
			continue
		}
		typ := items[0].hit.Type // المفتاحُ يحمل النوعَ، فهو واحدٌ في المجموعة
		pts := make([]AddrEvidence, len(items))
		for i, it := range items {
			pts[i] = it.ev
		}
		out = append(out, StreetGroup{Name: typDisplay(items[0].hit, name), Key: typ + " " + key, Bare: key, Type: typ,
			Points: pts, Customers: len(custs), Evidence: len(items), Samples: sampleSet(items, 4)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Customers != out[j].Customers {
			return out[i].Customers > out[j].Customers
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// typDisplay «شارع» + الاسم المعروض — بصورة النوع الأكثر شيوعاً للعرض.
func typDisplay(h AreaHit, name string) string {
	switch h.Type {
	case "طريق":
		return "طريق " + name
	case "اوتوستراد":
		return "اوتوستراد " + name
	}
	return "شارع " + name
}

// LocatedEdge أقربُ طريقٍ لنقطة.
type LocatedEdge struct {
	WayID int64
	Names []string
	DistM float64
	Line  [][2]float64 // (lng,lat) هندسةُ الحافّة؛ قد تكون فارغة
}

// Locator يلصق نقاطاً بأقرب طريق. يردّ عنصراً لكلّ نقطةٍ (WayID = 0: لا طريقَ قريب).
type Locator interface {
	Locate(ctx context.Context, pts [][2]float64) ([]LocatedEdge, error)
}

// ErrLocateBudget نفدت نداءاتُ الجولة — يبقى الباقي للجولة التالية ولا يُحذف قديمٌ.
var ErrLocateBudget = errors.New("maplearn: نفدت نداءات اللصق في هذه الجولة")

// StreetResolved شارعٌ ثبت على طريقٍ أو أكثر.
type StreetResolved struct {
	Name      string
	Key       string
	WayIDs    []int64
	Lines     [][][2]float64
	Lat, Lng  float64
	Customers int // الملصَقون على الطرق المختارة
	Evidence  int
	Samples   []string
	Named     bool // كلُّ الطرق المختارة تحمل هذا الاسمَ أصلاً في الخريطة
}

// ResolveStreet يلصق نقاطَ الاسم بأقرب الطرق ويختار الأكثرَ اتّساقاً. nil = لا اتّفاق.
// `used` يُعدّ ما استُهلك من نداءات الجولة.
func ResolveStreet(ctx context.Context, loc Locator, g StreetGroup, p AreaParams, used *int) (*StreetResolved, error) {
	p = p.filled()
	// حتى ثلاث نقاطٍ لكلّ زبون، وسقفٌ للاسم.
	perCust := map[string]int{}
	var pts []AddrEvidence
	for _, e := range g.Points {
		if perCust[e.Customer] >= 3 || len(pts) >= p.MaxLocatePerStreet {
			continue
		}
		perCust[e.Customer]++
		pts = append(pts, e)
	}
	if used != nil {
		if *used+len(pts) > p.MaxLocatePerRun {
			return nil, ErrLocateBudget
		}
		*used += len(pts)
	}
	coords := make([][2]float64, len(pts))
	for i, e := range pts {
		coords[i] = [2]float64{e.Lat, e.Lng}
	}
	got, err := loc.Locate(ctx, coords)
	if err != nil {
		return nil, err
	}
	// أقربُ طريقٍ لكلّ زبون.
	type snap struct {
		edge LocatedEdge
		ev   AddrEvidence
	}
	best := map[string]snap{}
	for i, e := range pts {
		if i >= len(got) || got[i].WayID == 0 || got[i].DistM > p.SnapM {
			continue
		}
		if b, ok := best[e.Customer]; !ok || got[i].DistM < b.edge.DistM {
			best[e.Customer] = snap{got[i], e}
		}
	}
	total := len(best)
	if total < p.StreetMinCustomers {
		return nil, nil
	}
	byWay := map[int64][]string{}
	for c, s := range best {
		byWay[s.edge.WayID] = append(byWay[s.edge.WayID], c)
	}
	ways := make([]int64, 0, len(byWay))
	for w, cs := range byWay {
		if len(cs) >= p.StreetMinCustomers {
			ways = append(ways, w)
		}
	}
	sort.Slice(ways, func(i, j int) bool {
		if len(byWay[ways[i]]) != len(byWay[ways[j]]) {
			return len(byWay[ways[i]]) > len(byWay[ways[j]])
		}
		return ways[i] < ways[j]
	})
	covered := 0
	for _, w := range ways {
		covered += len(byWay[w])
	}
	if len(ways) == 0 || float64(covered) < p.MinShare*float64(total) {
		return nil, nil
	}
	chosen := map[int64]bool{}
	for _, w := range ways {
		chosen[w] = true
	}
	r := &StreetResolved{Name: g.Name, Key: g.Key, WayIDs: ways, Customers: covered, Evidence: g.Evidence, Samples: g.Samples}
	// هندسةُ الطرق (حافّةٌ واحدةٌ بلا تكرارٍ بين الاتجاهَين) وأسماؤها.
	seenEdge := map[string]bool{}
	var mine []AddrEvidence
	named := true
	namedSeen := map[int64]bool{}
	custs := make([]string, 0, len(best))
	for c := range best {
		custs = append(custs, c)
	}
	sort.Strings(custs)
	for _, c := range custs {
		s := best[c]
		w := s.edge.WayID
		if !chosen[w] {
			continue
		}
		mine = append(mine, s.ev)
		if len(s.edge.Line) >= 2 {
			k, _, canon := CanonEdge(w, s.edge.Line)
			if !seenEdge[k] {
				seenEdge[k] = true
				r.Lines = append(r.Lines, canon)
			}
		}
		if !namedSeen[w] {
			namedSeen[w] = true
			if !streetNamed(s.edge.Names, g.Type, g.Bare) {
				named = false
			}
		}
	}
	r.Named = named
	m := medoid(mine)
	r.Lat, r.Lng = mine[m].Lat, mine[m].Lng
	return r, nil
}

// streetNamed هل أحدُ أسماء الطريق في الخريطة هو هذا الاسم؟ («شارع الوادي» أو «الوادي» بلا نوع.)
func streetNamed(names []string, typ, bare string) bool {
	for _, n := range names {
		var ws []string
		for _, w := range strings.Fields(Normalize(n)) {
			ws = append(ws, stripAl(w))
		}
		full := strings.Join(ws, " ")
		if full == bare || full == typ+" "+bare || strings.HasPrefix(full, typ+" "+bare+" ") {
			return true
		}
	}
	return false
}
