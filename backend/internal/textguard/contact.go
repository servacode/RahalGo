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
//	«0 9 9 1 2 3 4 5 6 7» «099-123-4567»        مفصولاً بمسافةٍ أو شَرطةٍ أو نقطة
//	«wa.me/…» «api.whatsapp.com/…»             رابطُ الواتساب
//	«واتس» «الواتس» «بالواتساب» «WhatsApp»      الكلمةُ بسوابقها، و«و ا ت س» مفرّقة
//
// **ولا يُمسّ العدديُّ العاديّ**: سعرٌ («25000») وكميّةٌ ورقمُ بناءٍ ورقمُ طلبٍ
// («#123456») — كلُّها دون سبع خانات — **والفاصلةُ والشَّرطةُ المائلةُ
// والنقطتان ليست فواصلَ رقم**: «1,250,000» و«5/11/2026» و«10:30» تمرّ.

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
	spans = append(spans, linkSpans(rs)...)
	spans = append(spans, contactWordSpans(rs)...)
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

// phoneSep **فاصلٌ يُقبل بين خانات الرقم** — مسافةٌ وشَرطةٌ ونقطةٌ وقوسٌ
// وشَرطةٌ سفليّة. **لا الفاصلةُ ولا المائلةُ ولا النقطتان** — تلك أعدادٌ
// وتواريخُ وأوقات. **ولا السطرُ الجديد** — رقمٌ على سطرين ثغرةٌ معروفة،
// وإمساكُه يُمسك «السعر 25000» فوق «12 قطعة».
func phoneSep(r rune) bool {
	switch r {
	case '-', '.', '(', ')', '_', '\u2212':
		return true
	case '\n', '\r':
		return false
	}
	if r >= 0x2010 && r <= 0x2015 {
		return true
	}
	return unicode.IsSpace(r)
}

// phoneIgnorable **ما يلتصق بالخانة ولا يفصلها** — محدِّدُ الإيموجي ومفتاحُها
// («0\uFE0F\u20E39…»).
func phoneIgnorable(r rune) bool { return r == 0xFE0F || r == 0x20E3 }

func isPlus(r rune) bool { return r == '+' || r == '\uFF0B' }

// phoneSpans **مواضعُ الأرقام الطويلة** — خاناتٌ (بأيّ نظام أرقام) بينها
// فاصلان على الأكثر، **وسبعُ خاناتٍ فأكثر.** **و`+` قبلها يُخفى معها.**
func phoneSpans(rs []rune) []cspan {
	var out []cspan
	for i := 0; i < len(rs); {
		start := -1
		switch {
		case unicode.IsDigit(rs[i]):
			start = i
		case isPlus(rs[i]):
			j := i + 1
			if j < len(rs) && unicode.IsSpace(rs[j]) && rs[j] != '\n' {
				j++
			}
			if j < len(rs) && unicode.IsDigit(rs[j]) {
				start = i
			}
		}
		if start < 0 {
			i++
			continue
		}
		digits, last, seps := 0, -1, 0
		j := start
		if isPlus(rs[j]) {
			j++
		}
		for j < len(rs) {
			r := rs[j]
			if unicode.IsDigit(r) {
				digits++
				last = j
				seps = 0
				j++
				continue
			}
			if phoneIgnorable(r) {
				j++
				continue
			}
			if phoneSep(r) {
				seps++
				if seps > 2 {
					break
				}
				j++
				continue
			}
			break
		}
		if digits >= MinPhoneDigits && last >= 0 {
			end := last + 1
			for end < len(rs) && phoneIgnorable(rs[end]) {
				end++
			}
			out = append(out, cspan{start: start, end: end, kind: "phone"})
			i = end
			continue
		}
		// **ولا رجوعَ إلى الوراء** — ما بعد آخر خانةٍ يُفحص من جديد.
		next := last + 1
		if next <= start {
			next = start + 1
		}
		i = next
	}
	return out
}

// ── الروابط ──────────────────────────────────────────────────────────

func linkSpans(rs []rune) []cspan {
	s := string(rs)
	locs := contactLinkRe.FindAllStringIndex(s, -1)
	if len(locs) == 0 {
		return nil
	}
	out := make([]cspan, 0, len(locs))
	for _, l := range locs {
		start := utf8.RuneCountInString(s[:l[0]])
		end := start + utf8.RuneCountInString(s[l[0]:l[1]])
		out = append(out, cspan{start: start, end: end, kind: "link"})
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

// contactWordSpans **مواضعُ كلمة الواتساب** — بتقطيع فلتر الشتائم وتطبيعه
// نفسيهما، **والحروفُ المفرّقةُ تُجمع** («و ا ت س»)، **والذيلُ يُضمّ**
// («whats app»).
func contactWordSpans(rs []rune) []cspan {
	tokens := tokenize(string(rs))
	var out []cspan
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
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
	return out
}
