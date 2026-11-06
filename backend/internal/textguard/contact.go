package textguard

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ══════════════════════════════════════════════════════════════════════
// **إخفاءُ وسائل التواصل — رقمُ الجوال ورابطُ الواتساب وكلمتُه**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-١١-٠٥: «موافق على كلشي» — سبعُ حماياتٍ حتّى لا يأخذ
// السائقُ الزبونَ برّا المنصّة.) **في حديث الطلب بين السائق والزبون** يُخفى
// كلُّ ما يشبه رقمَ جوّال، وكلُّ رابطِ واتساب، وكلمةُ «واتس/واتساب/whatsapp»
// — بالاتّجاهين — **ويُوسَم السطرُ للإدارة كالشتيمة** («في الحديث إخفاءٌ
// وتنبيهٌ للإدارة»).
//
// # ما يُمسك وما لا يُمسك
//
//	«0991234567» «099 123 4567» «+963 991…»   رقمٌ من سبع خاناتٍ فأكثر
//	«٠٩٩١٢٣٤٥٦٧» «۰۹۹…» «٠٩٩1234567»          بأيّ أرقامٍ كُتب، ومخلوطاً
//	«0 9 9 1 2 3 4 5 6 7» «099 - 123 - 4567»    مفصولاً بمسافةٍ أو شَرطةٍ أو نقطة
//	«099/123/4567» «٠٩٩،١٢٣،٤٥٦٧»              أو بمائلةٍ أو فاصلة
//	«wa.me/…» «api.whatsapp.com/…»             رابطُ الواتساب
//	«واتس» «الواتس» «بالواتساب» «WhatsApp»      الكلمةُ بسوابقها، و«و ا ت س» مفرّقة
//
// **ولا يُمسّ العدديُّ العاديّ**: سعرٌ («25000») وكميّةٌ ورقمُ بناءٍ ورقمُ طلبٍ
// («#123456») — كلُّها دون سبع خانات. **والمفصولُ يُخفى إن أشبه رقماً سوريّاً
// وحده** (`phoneLike`): «25000-30000» و«1,250,000» و«1 250 000» و«5-11-2026»
// و«05.11.2026» و«10.30-11.45» و«10:30» تمرّ.

// ContactHit **ما أُخفي ونصُّه الأصليّ** — للإدارة وحدها، لا للطرف الآخر.
type ContactHit struct {
	Kind string // "phone" · "link" · "word"
	Text string
}

// MinPhoneDigits **أقلُّ خاناتٍ يُعدّ بها الرقمُ رقمَ تواصل.** — وما دونها
// سعرٌ أو كميّةٌ أو رقمُ بناء.
const MinPhoneDigits = 7

// contactLinkRe **رابطُ الواتساب** — بلا حساسيّةِ حروف، ومعه ما يلحقه من مسار.
var contactLinkRe = regexp.MustCompile(`(?i)\b(?:https?://)?(?:www\.)?(?:wa\.me|wa\.link|(?:api|chat|web)\.whatsapp\.com|whatsapp\.com)(?:/\S*)?`)

// ══════════════════════════════════════════════════════════════════════
// **وتلغرام وفيسبوك وماسنجر وانستغرام والبريد** (قرارُ المالك ٢٠٢٦-١١-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// **بابٌ برّا المنصّة كالواتساب تماماً** — فتُخفى روابطُها وأسماؤها بالاتّجاهين وتُوسَم
// للإدارة والأصلُ في `flag_word`:
//
//	«t.me/…» «telegram.me» «telegram.dog»                 رابطُ تلغرام
//	«facebook.com» «fb.com» «fb.me» «m.me» «messenger.com» رابطُ فيسبوك وماسنجر
//	«instagram.com» «instagr.am»                          رابطُ انستغرام
//	«تلغرام» «تيليجرام» «فيسبوك» «فيس بوك» «ماسنجر» «انستغرام»  الكلمةُ بسوابقها
//	«ahmad@gmail.com»                                     البريدُ الإلكترونيّ
//	«@ahmad_99» بعد كلمة تلغرام أو انستغرام                المعرّف — **وبلاها لا يُمسّ**
//
// **ولا يُمسّ «الفيس» وحدَها** (وجهٌ أو فيسبوك؟ لا يُعرف) **ولا «@» عاديّة.**

// socialLinkRe **روابطُ تلغرام وفيسبوك وماسنجر وانستغرام.**
var socialLinkRe = regexp.MustCompile(`(?i)\b(?:https?://)?(?:www\.)?(?:t\.me|telegram\.(?:me|dog)|(?:[a-z]{1,3}\.)?facebook\.com|fb\.(?:com|me|watch)|m\.me|messenger\.com|instagram\.com|instagr\.am)(?:/\S*)?`)

// emailRe **البريدُ الإلكترونيّ** — وسيلةُ تواصلٍ كالرقم.
var emailRe = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9\-]+(?:\.[A-Za-z0-9\-]+)*\.[A-Za-z]{2,}`)

// handleRe **معرّفٌ «@…» لاتينيّ** — يُخفى بعد كلمة تلغرام أو انستغرام وحدَها.
var handleRe = regexp.MustCompile(`@[A-Za-z0-9_][A-Za-z0-9_.]{2,}`)

// socialWordHas **ما إن احتوته الكلمةُ (مطبَّعةً مطويّةَ التكرار) فهي تطبيقُ تواصل** —
// والقيمةُ «cue»: أيُطلق «@…» بعدها. **وصورٌ مطويّةٌ معها** («facebok» «mesenger»).
var socialWordHas = map[string]bool{
	"تلغرام": true, "تليغرام": true, "تيليغرام": true, "تيلغرام": true, "تلجرام": true,
	"تليجرام": true, "تيليجرام": true, "تيلجرام": true, "تلقرام": true, "تليقرام": true,
	"تيليقرام": true, "telegram": true,
	"انستغرام": true, "انستقرام": true, "انستجرام": true, "انستاغرام": true, "انستاجرام": true,
	"انستكرام": true, "instagram": true,
	"فيسبوك": false, "فيسبك": false, "فسبوك": false, "facebok": false,
	"ماسنجر": false, "مسنجر": false, "ماسينجر": false, "mesenger": false,
}

// socialWordIs **ما كانته الكلمةُ بعينه** — قصيرٌ لا يُبحث عنه داخلَ غيره («insta» في «install»).
var socialWordIs = map[string]bool{"انستا": true, "insta": true, "fb": false, "تلي": false}

// socialPairs **كلمتان هما اسمٌ واحد** — «فيس بوك» و«face book» و«insta gram».
var socialPairs = map[string]map[string]bool{
	"فيس": {"بوك": false, "بك": false}, "face": {"bok": false},
	"انستا": {"غرام": true, "جرام": true, "قرام": true}, "insta": {"gram": true},
	"تلي": {"غرام": true, "جرام": true},
}

// contactWordHas **ما إن احتوته الكلمةُ (مطبَّعةً) فهي الواتساب** — والتكرارُ
// يُطوى قبلها، فـ«whatsapp» تصير «whatsap».
var contactWordHas = []string{"واتس", "وتساب", "واتصاب", "whatsap", "whatsab", "watsap", "watsab"}

// contactWordIs **ما كانته الكلمةُ بعينه** — قصيرٌ لا يُبحث عنه داخلَ غيره.
var contactWordIs = map[string]bool{"whats": true, "wats": true, "wtsp": true, "وتس": true}

// contactWordNot **كلماتٌ تحوي «واتس» وليست منه** — «واتّسعت» و«واتّسخ»
// و«واتّسم» و«واتّسق» و«واتّساع».
var contactWordNot = []string{"اتسع", "اتساع", "اتسخ", "اتسم", "اتسق"}

// contactWordTail **ذيلٌ يُكمل الكلمة** — «whats app» و«واتس آب» كلمةٌ واحدة.
var contactWordTail = map[string]bool{"ap": true, "ab": true, "اب": true}

type cspan struct {
	start, end int // بالحروف، `end` خارجه
	kind       string
}

// MaskContacts **يُخفي كلَّ وسيلةِ تواصلٍ بـ`token`** — ويردّ ما أُخفي بنصّه
// الأصليّ. **ونصٌّ لا شيءَ فيه يعود كما هو حرفاً بحرف.**
func MaskContacts(s, token string) (string, []ContactHit) {
	if s == "" {
		return s, nil
	}
	// **المحارفُ الخفيّةُ تُسقط أوّلاً** — «099\u200c1234567» بفاصلٍ لا يُرى رقمٌ واحد.
	rs := make([]rune, 0, utf8.RuneCountInString(s))
	for _, r := range s {
		if r != '\n' && r != '\r' && r != '\t' && isInvisible(r) {
			continue
		}
		rs = append(rs, r)
	}
	spans := phoneSpans(rs)
	spans = append(spans, linkSpans(rs, contactLinkRe, "link")...)
	spans = append(spans, linkSpans(rs, socialLinkRe, "link")...)
	spans = append(spans, linkSpans(rs, emailRe, "email")...)
	words, cueAt := contactWordSpans(rs)
	spans = append(spans, words...)
	// **والمعرّفُ بعد كلمة تلغرام أو انستغرام وحدَها** — «@احمد» و«شكراً @ahmad» تمرّان.
	if cueAt >= 0 {
		for _, h := range linkSpans(rs, handleRe, "handle") {
			if h.start >= cueAt {
				spans = append(spans, h)
			}
		}
	}
	if len(spans) == 0 {
		return s, nil
	}
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	// **المتداخلُ يُضمّ** — «wa.me/963…» رابطٌ ورقمٌ في موضعٍ واحد.
	merged := []cspan{spans[0]}
	for _, sp := range spans[1:] {
		last := &merged[len(merged)-1]
		if sp.start < last.end {
			if sp.end > last.end {
				last.end = sp.end
			}
			continue
		}
		merged = append(merged, sp)
	}
	var b strings.Builder
	hits := make([]ContactHit, 0, len(merged))
	at := 0
	for _, sp := range merged {
		b.WriteString(string(rs[at:sp.start]))
		b.WriteString(token)
		hits = append(hits, ContactHit{Kind: sp.kind, Text: string(rs[sp.start:sp.end])})
		at = sp.end
	}
	b.WriteString(string(rs[at:]))
	return b.String(), hits
}

// ContactSummary **ما أُخفي سطراً واحداً للإدارة** — «رقم: 0991234567 · كلمة: واتس».
func ContactSummary(h []ContactHit) string {
	parts := make([]string, 0, len(h))
	for _, x := range h {
		label := "كلمة"
		switch x.Kind {
		case "phone":
			label = "رقم"
		case "link":
			label = "رابط"
		case "email":
			label = "بريد"
		case "handle":
			label = "معرّف"
		}
		text := x.Text
		if rs := []rune(text); len(rs) > 80 {
			text = string(rs[:80]) + "…"
		}
		parts = append(parts, label+": "+text)
	}
	return strings.Join(parts, " · ")
}

// ── الأرقام ──────────────────────────────────────────────────────────

// phoneJoin **ما يُقبل بين مجموعتَي خانات** — مسافةٌ وشَرطةٌ (بأنواعها)
// ونقطةٌ وقوسٌ وشَرطةٌ سفليّةٌ ومائلةٌ وفاصلةٌ (لاتينيّةً وعربيّةً وفاصلةَ
// آلاف). **ولا النقطتان** — «10:30» وقتٌ لا يُضمّ. **ولا السطرُ الجديد** —
// رقمٌ على سطرين ثغرةٌ معروفة، وإمساكُه يُمسك «السعر 25000» فوق «12 قطعة».
func phoneJoin(r rune) bool {
	switch r {
	case '-', '.', '(', ')', '_', '/', ',', '\u2212', '\u060C', '\u066B', '\u066C', '\uFF0C', '\u2044':
		return true
	case '\n', '\r':
		return false
	}
	if r >= 0x2010 && r <= 0x2015 {
		return true
	}
	return unicode.IsSpace(r)
}

// phoneJoinMax **أطولُ فاصلٍ بين مجموعتين** — «099 - 123» فاصلٌ واحدٌ من
// ثلاثة حروف، و«099) - (123» من خمسة. **وأكثرُ من ثلاث علاماتٍ ليس فاصلاً.**
const (
	phoneJoinMax     = 6
	phoneJoinMaxMark = 3
)

// phoneIgnorable **ما يلتصق بالخانة ولا يفصلها** — محدِّدُ الإيموجي ومفتاحُها
// («0\uFE0F\u20E39…»).
func phoneIgnorable(r rune) bool { return r == 0xFE0F || r == 0x20E3 }

func isPlus(r rune) bool { return r == '+' || r == '\uFF0B' }

// digitValue **قيمةُ الخانة بأيّ نظام أرقام** — «٧» و«۷» و«7» سبعة. **وكتلُ
// الأرقام في يونيكود عشراتٌ متتالية**: يُرجع إلى أوّل الكتلة ويُؤخذ الباقي.
func digitValue(r rune) int {
	if r >= '0' && r <= '9' {
		return int(r - '0')
	}
	z := r
	for k := 0; k < 60 && unicode.IsDigit(z-1); k++ {
		z--
	}
	return int(r-z) % 10
}

// dgroup **مجموعةُ خاناتٍ متلاصقة** — بموضعها ونصِّها بالأرقام اللاتينيّة،
// **وعلاماتِ الفاصل الذي قبلها** (بلا مسافاته): «-» في «099 - 123».
type dgroup struct {
	start, end int // بالحروف، `end` خارجه (ومعه ما لصق بالخانة الأخيرة)
	digits     string
	join       string
}

// phoneSpans **مواضعُ أرقام التواصل.** — تُقرأ السلسلةُ مجموعاتٍ بينها فواصل،
// **ثمّ يُبحث فيها عن أطول قطعةٍ تشبه رقماً** (`phoneLike`) — فعددٌ قبل الرقم
// («السعر 25 099 123 4567») لا يحميه. **و`+` قبلها يُخفى معها.**
func phoneSpans(rs []rune) []cspan {
	var out []cspan
	for i := 0; i < len(rs); {
		plus, start := -1, i
		switch {
		case unicode.IsDigit(rs[i]):
		case isPlus(rs[i]):
			j := i + 1
			if j < len(rs) && unicode.IsSpace(rs[j]) && rs[j] != '\n' && rs[j] != '\r' {
				j++
			}
			if j >= len(rs) || !unicode.IsDigit(rs[j]) {
				i++
				continue
			}
			plus, start = i, j
		default:
			i++
			continue
		}
		groups, next := readDigitRun(rs, start)
		out = append(out, runPhones(groups, plus)...)
		i = next
	}
	return out
}

// readDigitRun **يقرأ مجموعاتِ الخانات من `start`** — ويردّ أين يُستأنف البحث:
// **بعد آخر خانة**، فما بعدها يُفحص من جديد.
func readDigitRun(rs []rune, start int) ([]dgroup, int) {
	var groups []dgroup
	join := ""
	j := start
	for {
		gs := j
		var d strings.Builder
		for j < len(rs) && (unicode.IsDigit(rs[j]) || (j > gs && phoneIgnorable(rs[j]))) {
			if unicode.IsDigit(rs[j]) {
				d.WriteByte(byte('0' + digitValue(rs[j])))
			}
			j++
		}
		groups = append(groups, dgroup{start: gs, end: j, digits: d.String(), join: join})
		k, n, marks := j, 0, 0
		var mb strings.Builder
		for k < len(rs) && phoneJoin(rs[k]) {
			n++
			if !unicode.IsSpace(rs[k]) {
				marks++
				mb.WriteRune(rs[k])
			}
			k++
		}
		if n == 0 || n > phoneJoinMax || marks > phoneJoinMaxMark || k >= len(rs) || !unicode.IsDigit(rs[k]) {
			return groups, j
		}
		join = mb.String()
		j = k
	}
}

// runPhones **أطولُ قطعةٍ تشبه رقماً من كلّ موضع** — وما أُخفي لا يُعاد فحصُه.
func runPhones(gs []dgroup, plus int) []cspan {
	var out []cspan
	for i := 0; i < len(gs); {
		found := -1
		for j := len(gs) - 1; j >= i; j-- {
			if phoneLike(gs[i:j+1], plus >= 0 && i == 0) && !hasDate(gs[i:j+1]) {
				found = j
				break
			}
		}
		if found < 0 {
			i++
			continue
		}
		s := gs[i].start
		if i == 0 && plus >= 0 {
			s = plus
		}
		out = append(out, cspan{start: s, end: gs[found].end, kind: "phone"})
		i = found + 1
	}
	return out
}

// phoneLike **هل تشبه هذه المجموعاتُ رقمَ تواصل؟**
//
//	ملتصقةً بلا فاصل، أو بعد «+»         سبعُ خاناتٍ فأكثر (كما كانت)
//	مفصولةً وتبدأ بـ«00»                  ١٠–١٥ خانة   «00963 991 234 567»
//	مفصولةً وتبدأ بـ«0»                   ٩–١١ خانة    «099 123 4567» «022 123456»
//	مفصولةً وتبدأ بـ«963»                 ١١–١٣ خانة   «963 991 234 567»
//	مفصولةً وتبدأ بـ«9»                   تسعُ خانات   «991 234 567»
//
// **وما سوى ذلك مفصولاً عددٌ لا رقم**: «25000-30000» و«1 250 000» و«10.30-11.45».
func phoneLike(gs []dgroup, plus bool) bool {
	var b strings.Builder
	for _, g := range gs {
		b.WriteString(g.digits)
	}
	n := b.String()
	d := len(n)
	if len(gs) == 1 {
		return d >= MinPhoneDigits
	}
	if plus {
		return d >= MinPhoneDigits && d <= 15
	}
	switch {
	case strings.HasPrefix(n, "00"):
		return d >= 10 && d <= 15
	case strings.HasPrefix(n, "0"):
		return d >= 9 && d <= 11
	case strings.HasPrefix(n, "963"):
		return d >= 11 && d <= 13
	case strings.HasPrefix(n, "9"):
		return d == 9
	}
	return false
}

// hasDate **هل في القطعة تاريخٌ بسنةٍ كاملة؟** — «05.11.2026» و«2026/11/05»:
// ثلاثُ مجموعاتٍ بفاصلٍ واحدٍ متكرّر، يومٌ (١–٣١) وشهرٌ (١–١٢) وسنةٌ تبدأ
// بـ«19» أو «20». **والتاريخُ لا يُخفى** ولو تلاه ما يكمّله إلى عشر خانات
// («05.11.2026 10:30»).
func hasDate(gs []dgroup) bool {
	for k := 0; k+2 < len(gs); k++ {
		a, m, c := gs[k].digits, gs[k+1].digits, gs[k+2].digits
		if gs[k+1].join == "" || gs[k+1].join != gs[k+2].join {
			continue
		}
		if isDay(a) && isMonth(m) && isYear(c) || isYear(a) && isMonth(m) && isDay(c) {
			return true
		}
	}
	return false
}

func smallNum(s string, lo, hi int) bool {
	if len(s) < 1 || len(s) > 2 {
		return false
	}
	v := 0
	for _, c := range s {
		v = v*10 + int(c-'0')
	}
	return v >= lo && v <= hi
}

func isDay(s string) bool   { return smallNum(s, 1, 31) }
func isMonth(s string) bool { return smallNum(s, 1, 12) }
func isYear(s string) bool {
	return len(s) == 4 && (strings.HasPrefix(s, "19") || strings.HasPrefix(s, "20"))
}

// ── الروابط ──────────────────────────────────────────────────────────

func linkSpans(rs []rune, re *regexp.Regexp, kind string) []cspan {
	s := string(rs)
	locs := re.FindAllStringIndex(s, -1)
	if len(locs) == 0 {
		return nil
	}
	out := make([]cspan, 0, len(locs))
	for _, l := range locs {
		start := utf8.RuneCountInString(s[:l[0]])
		end := start + utf8.RuneCountInString(s[l[0]:l[1]])
		out = append(out, cspan{start: start, end: end, kind: kind})
	}
	return out
}

// ── الكلمة ───────────────────────────────────────────────────────────

func isContactWord(norm string) bool {
	if contactWordIs[norm] {
		return true
	}
	for _, n := range contactWordNot {
		if strings.Contains(norm, n) {
			return false
		}
	}
	for _, w := range contactWordHas {
		if strings.Contains(norm, w) {
			return true
		}
	}
	return false
}

// socialWord **أهذه الكلمةُ اسمُ تطبيق تواصل؟** — والثانيةُ: أيُطلق «@…» بعدها.
func socialWord(norm string) (bool, bool) {
	if cue, ok := socialWordIs[norm]; ok {
		return true, cue
	}
	for w, cue := range socialWordHas {
		if strings.Contains(norm, w) {
			return true, cue
		}
	}
	return false, false
}

// contactWordSpans **مواضعُ كلمة الواتساب وأسماءِ تطبيقات التواصل** — بتقطيع فلتر
// الشتائم وتطبيعه نفسيهما، **والحروفُ المفرّقةُ تُجمع** («و ا ت س»)، **والذيلُ يُضمّ**
// («whats app»). **والثاني أوّلُ موضعِ كلمةِ تلغرام أو انستغرام** (`-1` إن لم تكن) —
// فما بعده من «@…» معرّفٌ يُخفى.
func contactWordSpans(rs []rune) ([]cspan, int) {
	tokens := tokenize(string(rs))
	var out []cspan
	cueAt := -1
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		// **كلمتان هما اسمٌ واحد** — «فيس بوك».
		if tails, ok := socialPairs[t.norm]; ok && i+1 < len(tokens) {
			if cue, ok := tails[tokens[i+1].norm]; ok {
				out = append(out, cspan{start: t.start, end: tokens[i+1].end, kind: "word"})
				if cue && cueAt < 0 {
					cueAt = t.start
				}
				i++
				continue
			}
		}
		if ok, cue := socialWord(t.norm); ok {
			out = append(out, cspan{start: t.start, end: t.end, kind: "word"})
			if cue && cueAt < 0 {
				cueAt = t.start
			}
			continue
		}
		end, next := -1, i+1
		if len([]rune(t.norm)) == 1 {
			j := i
			var joined strings.Builder
			for j < len(tokens) && len([]rune(tokens[j].norm)) == 1 {
				joined.WriteString(tokens[j].norm)
				j++
			}
			if j-i >= 2 && isContactWord(collapse(joined.String())) {
				end, next = tokens[j-1].end, j
			}
		}
		if end < 0 && isContactWord(t.norm) {
			end = t.end
		}
		if end < 0 {
			continue
		}
		if next < len(tokens) && contactWordTail[tokens[next].norm] {
			end = tokens[next].end
			next++
		}
		out = append(out, cspan{start: t.start, end: end, kind: "word"})
		i = next - 1
	}
	return out, cueAt
}
