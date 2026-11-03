package maplearn

// ══════════════════════════════════════════════════════════════════════
// **أماكن بلا اسم** (قرارُ المالك ٢٠٢٦-١١-٠٣ «اي موافق»)
// ══════════════════════════════════════════════════════════════════════
//
// كاشفان صافيان، لا يلمسان القاعدة:
//
//   ١ · `ClusterStops` — وقفاتُ السائقين تتكرّر عند مكانٍ واحد (٣ أيّامٍ أو ٣ سائقين) ولا معلمَ هناك.
//   ٢ · `ClusterNames` — عبارات «جانب صيدلية الشفاء» في عناوين الزبائن، يتّفق عليها زبونان فأكثر
//       على نقطةٍ واحدة (≤ ١٥٠ م).
//
// الحفظُ والاستبعادُ (ما قربه معلمٌ أو متجر) في `places_job.go`.

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// StopObs وقفةُ سائقٍ واحدة.
type StopObs struct {
	Lat, Lng float64
	Driver   string
	Day      string // YYYY-MM-DD
}

// PlaceParams عتبات كشف الأماكن.
type PlaceParams struct {
	// ClusterM نصفُ قطر تجميع الوقفات.
	ClusterM float64
	// MinDays أو MinDrivers — أيٌّ منهما يكفي.
	MinDays    int
	MinDrivers int
	// NameClusterM نصفُ قطر تجميع نقاط التسليم لاسمٍ واحد.
	NameClusterM float64
	// MinCustomers أقلُّ زبائنَ مختلفين يذكرون الاسمَ نفسَه عند النقطة نفسِها.
	MinCustomers int
}

// DefaultPlaceParams العتبات المعتمدة.
func DefaultPlaceParams() PlaceParams {
	return PlaceParams{ClusterM: 40, MinDays: 3, MinDrivers: 3, NameClusterM: 150, MinCustomers: 2}
}

// PlaceCluster مكانٌ تتكرّر فيه الوقفات.
type PlaceCluster struct {
	Lat, Lng          float64
	Visits            int
	Drivers           int
	Days              int
	FirstDay, LastDay string
}

type spatialIndex struct {
	cellM float64
	cells map[[2]int][]int
}

func newSpatialIndex(cellM float64) *spatialIndex {
	return &spatialIndex{cellM: cellM, cells: map[[2]int][]int{}}
}

func (s *spatialIndex) key(lat, lng float64) [2]int {
	mPerLng := mPerDegLat * math.Cos(lat*math.Pi/180)
	return [2]int{int(math.Floor(lat * mPerDegLat / s.cellM)), int(math.Floor(lng * mPerLng / s.cellM))}
}

func (s *spatialIndex) add(i int, lat, lng float64) {
	k := s.key(lat, lng)
	s.cells[k] = append(s.cells[k], i)
}

func (s *spatialIndex) near(lat, lng float64) []int {
	k := s.key(lat, lng)
	var out []int
	for di := -1; di <= 1; di++ {
		for dj := -1; dj <= 1; dj++ {
			out = append(out, s.cells[[2]int{k[0] + di, k[1] + dj}]...)
		}
	}
	return out
}

func distM(aLat, aLng, bLat, bLng float64) float64 {
	dy := (aLat - bLat) * mPerDegLat
	dx := (aLng - bLng) * mPerDegLat * math.Cos((aLat+bLat)/2*math.Pi/180)
	return math.Hypot(dx, dy)
}

type geoAcc struct {
	sLat, sLng float64
	n          int
	seedLat    float64
	seedLng    float64
}

func (g *geoAcc) center() (float64, float64) { return g.sLat / float64(g.n), g.sLng / float64(g.n) }

// ClusterStops يجمع الوقفات (أقربُ مركزٍ ضمن `ClusterM`) ويُبقي ما تكرّر.
func ClusterStops(obs []StopObs, p PlaceParams) []PlaceCluster {
	if p.ClusterM <= 0 {
		p = DefaultPlaceParams()
	}
	idx := newSpatialIndex(p.ClusterM)
	var accs []*geoAcc
	var members [][]int
	for oi, o := range obs {
		best, bestD := -1, p.ClusterM+1
		for _, ci := range idx.near(o.Lat, o.Lng) {
			la, ln := accs[ci].center()
			if d := distM(la, ln, o.Lat, o.Lng); d <= p.ClusterM && d < bestD {
				best, bestD = ci, d
			}
		}
		if best < 0 {
			accs = append(accs, &geoAcc{seedLat: o.Lat, seedLng: o.Lng})
			members = append(members, nil)
			best = len(accs) - 1
			idx.add(best, o.Lat, o.Lng)
		}
		a := accs[best]
		a.sLat += o.Lat
		a.sLng += o.Lng
		a.n++
		members[best] = append(members[best], oi)
	}
	var out []PlaceCluster
	for ci, a := range accs {
		drivers, days := map[string]bool{}, map[string]bool{}
		first, last := "", ""
		for _, oi := range members[ci] {
			o := obs[oi]
			drivers[o.Driver] = true
			days[o.Day] = true
			if first == "" || o.Day < first {
				first = o.Day
			}
			if o.Day > last {
				last = o.Day
			}
		}
		if len(days) < p.MinDays && len(drivers) < p.MinDrivers {
			continue
		}
		la, ln := a.center()
		out = append(out, PlaceCluster{Lat: la, Lng: ln, Visits: a.n, Drivers: len(drivers), Days: len(days),
			FirstDay: first, LastDay: last})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Visits != out[j].Visits {
			return out[i].Visits > out[j].Visits
		}
		return out[i].Lat < out[j].Lat
	})
	return out
}

// ══════════════════════════════════════════════════════════════════════
// الأسماء من العناوين
// ══════════════════════════════════════════════════════════════════════

// Normalize يوحّد العربيّة للمقارنة: الألفات، التاء المربوطة، الألف المقصورة، الهمزات، التشكيل والتطويل،
// الأرقام الهنديّة، والمسافات.
func Normalize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 0x064B && r <= 0x065F, r == 0x0670, r == 0x0640:
			continue
		case r == 'أ' || r == 'إ' || r == 'آ' || r == 'ٱ':
			b.WriteRune('ا')
		case r == 'ة':
			b.WriteRune('ه')
		case r == 'ى':
			b.WriteRune('ي')
		case r == 'ؤ':
			b.WriteRune('و')
		case r == 'ئ':
			b.WriteRune('ي')
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// cleanDisplay يحذف التشكيل والتطويل فقط (يبقي صورة الكلمة كما كتبها الزبون).
func cleanDisplay(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 0x064B && r <= 0x065F) || r == 0x0670 || r == 0x0640 {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

var prefixWords = setOf("جانب", "بجانب", "قرب", "بقرب", "مقابل", "بمقابل", "قبال", "قباله", "بقبال", "بقباله",
	"خلف", "بخلف", "وراء", "ورا", "عند", "جنب", "بجنب", "مجاور", "بجوار", "جوار", "ملاصق", "امام", "بامام")

var stopWords = setOf("و", "ثم", "بعد", "الطابق", "طابق", "الدور", "دور", "شقه", "الشقه", "بنايه", "البنايه", "بناء",
	"عماره", "العماره", "بيت", "منزل", "رقم", "باب", "الباب", "ضمن", "داخل", "تحت", "فوق", "يمين", "يسار",
	"علي", "الي", "من", "في", "لو", "سمحت", "فضلك", "لطفا", "شكرا", "ممكن", "اذا", "بدون", "مع", "هون", "هناك", "تماما", "مباشره", "بالقرب", "الاول", "الثاني", "الثالث", "حي", "شارع", "الشارع", "حاره", "الحاره")

// typeKinds كلماتُ نوع المكان (بعد التوحيد وحذف «ال») ونوعُها في جدول المعالم.
var typeKinds = map[string]string{
	"جامع": "mosque", "مسجد": "mosque", "مصلي": "mosque",
	"صيدليه": "pharmacy",
	"مدرسه":  "school", "معهد": "school", "روضه": "school", "حضانه": "school", "ثانويه": "school",
	"اعداديه": "school", "جامعه": "school", "كليه": "school",
	"مشفي": "hospital", "مستشفي": "hospital", "مستوصف": "hospital", "عياده": "hospital",
	"فرن": "shop", "مخبز": "shop", "مطعم": "shop", "سوبرماركت": "shop", "ماركت": "shop", "بقاليه": "shop",
	"محل": "shop", "دكان": "shop", "معرض": "shop", "مقهي": "shop", "كافيه": "shop", "صالون": "shop",
	"مول": "shop", "مغسله": "shop", "مكتبه": "shop", "مطحنه": "shop", "محطه": "shop", "كازيه": "shop",
	"بنك": "shop", "مصرف": "shop", "فندق": "shop", "مكتب": "shop", "شركه": "shop", "موسسه": "shop",
	"مصنع": "shop", "كراج": "shop", "مخزن": "shop", "ستوديو": "shop",
	"دوار": "roundabout", "ساحه": "roundabout",
	"حديقه": "other", "ملعب": "other", "مخفر": "other", "بلديه": "other", "مديريه": "other", "دايره": "other",
	"مركز": "other", "محكمه": "other", "نادي": "other", "موقف": "other",
}

const maxNameTokens = 4

func setOf(ws ...string) map[string]bool {
	m := make(map[string]bool, len(ws))
	for _, w := range ws {
		m[w] = true
	}
	return m
}

func stripAl(w string) string {
	if strings.HasPrefix(w, "ال") && len([]rune(w)) > 3 {
		return strings.TrimPrefix(w, "ال")
	}
	return w
}

// NameHit اسمُ معلمٍ وُجد في نصّ.
type NameHit struct {
	Display string // كما كتبه الزبون (بلا تشكيل)
	Key     string // أوّلُ كلمتين موحَّدتين بلا «ال»
	Kind    string // مفتاحُ النوع (mosque, pharmacy, …)
}

var breakRunes = ",،.;؛:/\\-–—()[]{}|!؟?\"'«»\n\r\t"

// ExtractNames يستخرج أسماء المعالم من عبارات «جانب/قرب/مقابل/خلف/عند …».
func ExtractNames(text string) []NameHit {
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
	var out []NameHit
	for i := 0; i < len(ntoks); i++ {
		if !prefixWords[ntoks[i]] && !prefixWords[strings.TrimPrefix(ntoks[i], "و")] {
			continue
		}
		j := i + 1
		var dn, nn []string
		for ; j < len(ntoks) && len(dn) < maxNameTokens; j++ {
			t := ntoks[j]
			if t == "|" || prefixWords[t] || stopWords[t] || stopWords[stripAl(t)] || joinedStop(t) || strings.ContainsAny(t, "0123456789") {
				break
			}
			dn = append(dn, dtoks[j])
			nn = append(nn, stripAl(t))
		}
		if len(nn) < 2 {
			continue
		}
		kind, ok := typeKinds[nn[0]]
		if !ok {
			continue
		}
		out = append(out, NameHit{Display: strings.Join(dn, " "), Key: nn[0] + " " + nn[1], Kind: kind})
		i = j - 1
	}
	return out
}

// NameKey مفتاحُ مقارنةٍ لأيّ اسم (معلمٍ أو متجر): أوّلُ كلمتين موحَّدتين بلا «ال».
func NameKey(name string) string {
	var ws []string
	for _, w := range strings.Fields(Normalize(name)) {
		ws = append(ws, stripAl(w))
		if len(ws) == 2 {
			break
		}
	}
	return strings.Join(ws, " ")
}

// NameMatches هل اسمٌ موجودٌ (معلمٌ أو متجر) هو الاسمُ المقترَح؟ تساوي المفتاحين، أو احتواءُ الاسم الموجود للمفتاح.
func NameMatches(existing, key string) bool {
	if key == "" {
		return false
	}
	if NameKey(existing) == key {
		return true
	}
	var ws []string
	for _, w := range strings.Fields(Normalize(existing)) {
		ws = append(ws, stripAl(w))
	}
	return strings.Contains(" "+strings.Join(ws, " ")+" ", " "+key+" ")
}

// AddrEvidence نصُّ عنوانٍ أو ملاحظةٍ ونقطةُ تسليمه وصاحبُه (معرّفٌ فقط — لا اسمَ ولا هاتف).
type AddrEvidence struct {
	Text     string
	Customer string
	Lat, Lng float64
}

// NameCluster اسمٌ اتّفق عليه أكثرُ من زبونٍ عند نقطةٍ واحدة.
type NameCluster struct {
	Name      string
	Key       string
	Kind      string
	Lat, Lng  float64
	Evidence  int // عددُ العبارات
	Customers int
	Samples   []string
}

// ClusterNames يجمع الاسمَ الموحَّد على نقاط التسليم المتقاربة (≤ `NameClusterM`) ويُبقي ما ذكره
// `MinCustomers` زبائنَ مختلفين فأكثر.
func ClusterNames(ev []AddrEvidence, p PlaceParams) []NameCluster {
	if p.NameClusterM <= 0 || p.MinCustomers <= 0 {
		d := DefaultPlaceParams()
		p.NameClusterM, p.MinCustomers = d.NameClusterM, d.MinCustomers
	}
	type item struct {
		hit NameHit
		ev  AddrEvidence
	}
	byKey := map[string][]item{}
	for _, e := range ev {
		seen := map[string]bool{}
		for _, h := range ExtractNames(e.Text) {
			if seen[h.Key] {
				continue
			}
			seen[h.Key] = true
			byKey[h.Key] = append(byKey[h.Key], item{h, e})
		}
	}
	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out []NameCluster
	for _, k := range keys {
		items := byKey[k]
		idx := newSpatialIndex(p.NameClusterM)
		var accs []*geoAcc
		var members [][]int
		for ii, it := range items {
			best, bestD := -1, p.NameClusterM+1
			for _, ci := range idx.near(it.ev.Lat, it.ev.Lng) {
				la, ln := accs[ci].center()
				if d := distM(la, ln, it.ev.Lat, it.ev.Lng); d <= p.NameClusterM && d < bestD {
					best, bestD = ci, d
				}
			}
			if best < 0 {
				accs = append(accs, &geoAcc{})
				members = append(members, nil)
				best = len(accs) - 1
				idx.add(best, it.ev.Lat, it.ev.Lng)
			}
			accs[best].sLat += it.ev.Lat
			accs[best].sLng += it.ev.Lng
			accs[best].n++
			members[best] = append(members[best], ii)
		}
		for ci, a := range accs {
			custs := map[string]bool{}
			forms := map[string]int{}
			var samples []string
			sampleSeen := map[string]bool{}
			for _, ii := range members[ci] {
				it := items[ii]
				custs[it.ev.Customer] = true
				forms[it.hit.Display]++
				if s := SanitizeSample(it.ev.Text); s != "" && !sampleSeen[s] && len(samples) < 4 {
					sampleSeen[s] = true
					samples = append(samples, s)
				}
			}
			if len(custs) < p.MinCustomers {
				continue
			}
			best, bestN := "", -1
			for f, n := range forms {
				if n > bestN || (n == bestN && (len(f) < len(best) || (len(f) == len(best) && f < best))) {
					best, bestN = f, n
				}
			}
			la, ln := a.center()
			out = append(out, NameCluster{Name: best, Key: k, Kind: items[members[ci][0]].hit.Kind,
				Lat: la, Lng: ln, Evidence: a.n, Customers: len(custs), Samples: samples})
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

// SanitizeSample يجهّز نصَّ عنوانٍ للعرض دليلاً: يمسح كلَّ رقمٍ طويل (هاتف) ويقصّ الطول.
func SanitizeSample(s string) string {
	s = strings.Join(strings.Fields(cleanDisplay(s)), " ")
	var b strings.Builder
	run := 0
	var pending []rune
	flush := func() {
		if run < 6 {
			for _, r := range pending {
				b.WriteRune(r)
			}
		}
		pending, run = pending[:0], 0
	}
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= '٠' && r <= '٩') || (r == '+' && run == 0) {
			pending = append(pending, r)
			if r != '+' {
				run++
			}
			continue
		}
		flush()
		b.WriteRune(r)
	}
	flush()
	out := strings.TrimSpace(b.String())
	if r := []rune(out); len(r) > 120 {
		out = string(r[:120])
	}
	return out
}

// joinedStop كلمةٌ ملصقٌ بها واو العطف («وعند»، «ورقم») وأصلُها كلمةُ موضعٍ أو قطع.
func joinedStop(t string) bool {
	if !strings.HasPrefix(t, "و") {
		return false
	}
	rest := strings.TrimPrefix(t, "و")
	return prefixWords[rest] || stopWords[rest] || stopWords[stripAl(rest)]
}
