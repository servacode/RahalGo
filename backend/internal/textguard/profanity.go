package textguard

import (
	"strings"
	"unicode"
)

// ══════════════════════════════════════════════════════════════════════
// **فلترُ الشتائم — لا يُخدَع بالحيل المعروفة**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٣: «فلترُ الشتائم بالفصحى واللهجة السوريّة
//  والإنكليزيّة، لا يُخدَع — حروفٌ مفرّقة · تكرار · تشكيل · أرقامٌ مكان
//  حروف — وقائمتُه تُعدَّل من لوحة الإدارة».)
//
// # كيف يُخدَع فلترٌ ساذج — وكيف لا يُخدَع هذا
//
//	«كــلـب» «كَلْب»        التطويلُ والتشكيلُ يُسقطان قبل المقارنة
//	«كلللب»               التكرارُ يُطوى إلى حرفٍ واحد
//	«ك ل ب» «ك.ل.ب»       الحروفُ المفرّقةُ تُجمع كلمةً واحدة
//	«أ/ا/إ» «ة/ه» «ى/ي»    الأشكالُ تُردّ إلى شكلٍ واحد
//	«7مار» «kalb» «3rs»   أرقامُ العربيزي حروفٌ في الكلمة العربيّة
//	«ياكلب» «الكلب» «كلبك» السوابقُ واللواحقُ الشائعةُ تُنزع
//
// # ولا يمسك البريء
//
// **الفحصُ على الكلمة كاملةً لا على الاحتواء**: «كسرت» تحوي «كس»، و«كباب»
// تبدأ بـ«كب»، و«زبون» تبدأ بـ«زب». **والسوابقُ واللواحقُ قائمةٌ قصيرةٌ
// معدودة** — لا «كلُّ ما بدأ بلفظ» — فـ«حيوانات» (متجرُ حيواناتٍ أليفة)
// و«زبيب» و«خراب» و«إيران» تمرّ.

// SettingKey **مفتاحُ القائمة في الإعدادات** — تُعدَّل من لوحة الإدارة.
const SettingKey = "moderation.banned_words"

// DefaultWords **القائمةُ المدمجة** — افتراضُ `moderation.banned_words`.
//
// **تُكتب بأيّ شكل** — تُطبَّع عند التحميل. **والقائمةُ تُعدَّل من اللوحة**؛
// وهذه ما يعمل به المحرّكُ إن فرغ الإعداد.
var DefaultWords = []string{
	// فصحى ولهجة
	"كلب", "كلاب", "حمار", "حمير", "حيوان", "جحش", "بغل", "خنزير", "غبي", "اغبياء",
	"احمق", "حقير", "حقيرة", "نذل", "سافل", "منحط", "قذر", "وسخ", "زبالة", "تفو",
	"لعنة", "يلعن", "ديوث", "قحبة", "قحاب", "شرموط", "شرموطة", "شراميط", "عرص",
	"معرص", "منيك", "منيوك", "نيك", "انيك", "كس", "كسمك", "كسختك", "طيز", "خرا",
	"زق", "زب", "اير", "لوطي",
	// إنكليزيّة
	"fuck", "fucker", "fucking", "motherfucker", "shit", "bitch", "asshole",
	"bastard", "dick", "pussy", "cunt", "whore", "slut", "idiot", "stupid",
	// عربيزي
	"kalb", "7mar", "hmar", "7ayawan", "khanzir", "5anzir", "ghabi", "7a2er", "7aqer",
	"kos", "kus", "teez", "tiz", "sharmouta", "sharmota", "sharmoota", "3ars", "3rs",
	"m3ars", "manyak", "manyok", "neek", "nik", "a7a", "khara", "5ara", "zeb", "ayre",
	"ayri", "kosomak", "kosom",
}

// ParseList **قائمةُ الإعداد نصّاً** — لفظٌ في كلّ سطر، أو مفصولةٌ بفاصلة.
func ParseList(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == '،' || r == ';'
	})
}

// Filter **قائمةٌ مطبَّعةٌ جاهزةٌ للمقارنة.**
type Filter struct {
	words map[string]bool
}

var defaultFilter = NewFilter(DefaultWords)

// NewFilter يطبّع القائمةَ مرّةً — **والفراغُ والقصيرُ جدّاً يُسقطان.**
func NewFilter(list []string) *Filter {
	f := &Filter{words: map[string]bool{}}
	for _, w := range list {
		n := normalizeWord(strings.TrimSpace(w))
		if len([]rune(n)) >= 2 {
			f.words[n] = true
		}
	}
	return f
}

// Find **أوّلُ لفظٍ مسيءٍ في النصّ** — أو فراغ.
func (f *Filter) Find(s string) string {
	for _, h := range f.hits(s) {
		return h.word
	}
	return ""
}

// Mask **يُخفي كلَّ لفظٍ مسيءٍ بـ`token`** — ويردّ ما أُخفي (مطبَّعاً).
func (f *Filter) Mask(s, token string) (string, []string) {
	hits := f.hits(s)
	if len(hits) == 0 {
		return s, nil
	}
	rs := []rune(s)
	var b strings.Builder
	words := make([]string, 0, len(hits))
	at := 0
	for _, h := range hits {
		b.WriteString(string(rs[at:h.start]))
		b.WriteString(token)
		at = h.end
		words = append(words, h.word)
	}
	b.WriteString(string(rs[at:]))
	return b.String(), words
}

type span struct {
	start, end int // بالحروف، `end` خارجه
	norm       string
}

type hit struct {
	start, end int
	word       string
}

// hits **مواضعُ الألفاظ المسيئة بالترتيب** — كلماتٌ، ثمّ حروفٌ مفرّقةٌ مجموعة.
func (f *Filter) hits(s string) []hit {
	tokens := tokenize(s)
	var out []hit
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		// **الحروفُ المفرّقة** — «ك ل ب» ثلاثُ كلماتٍ من حرفٍ واحد.
		if len([]rune(t.norm)) == 1 {
			j := i
			var joined strings.Builder
			for j < len(tokens) && len([]rune(tokens[j].norm)) == 1 {
				joined.WriteString(tokens[j].norm)
				j++
			}
			if j-i >= 2 {
				if w := f.match(collapse(joined.String())); w != "" {
					out = append(out, hit{tokens[i].start, tokens[j-1].end, w})
					i = j - 1
					continue
				}
			}
		}
		if w := f.match(t.norm); w != "" {
			out = append(out, hit{t.start, t.end, w})
		}
	}
	return out
}

// tokenize **كلماتُ النصّ ومواضعُها** — والكلمةُ حروفٌ وأرقامٌ وتشكيلٌ وتطويل.
func tokenize(s string) []span {
	rs := []rune(s)
	var out []span
	for i := 0; i < len(rs); {
		if !inWord(rs[i]) {
			i++
			continue
		}
		j := i
		for j < len(rs) && inWord(rs[j]) {
			j++
		}
		out = append(out, span{start: i, end: j, norm: normalizeWord(string(rs[i:j]))})
		i = j
	}
	return out
}

func inWord(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r)
}

// prefixes وsuffixes **ما يلتصق بالكلمة في العربيّة** — معدودٌ لا «كلُّ ما بدأ».
var (
	prefixes = []string{"وال", "يال", "بال", "لل", "ال", "يا", "و"}
	suffixes = []string{"هم", "كم", "ها", "ه", "ك", "ي"}
	// enSuffixes **لواحقُ الإنكليزيّة** — «shits» و«fucked».
	enSuffixes = []string{"ing", "ers", "er", "ed", "s", "y"}
)

// match **أهذه الكلمةُ لفظٌ مسيء؟** — بنفسها، أو بعد نزع سابقةٍ أو لاحقة.
func (f *Filter) match(w string) string {
	if len([]rune(w)) < 2 {
		return ""
	}
	if f.words[w] {
		return w
	}
	cands := []string{w}
	for _, p := range prefixes {
		if strings.HasPrefix(w, p) && len([]rune(w))-len([]rune(p)) >= 2 {
			cands = append(cands, strings.TrimPrefix(w, p))
		}
	}
	for _, c := range cands {
		if f.words[c] {
			return c
		}
		for _, sx := range suffixes {
			if strings.HasSuffix(c, sx) && len([]rune(c))-len([]rune(sx)) >= 2 {
				if base := strings.TrimSuffix(c, sx); f.words[base] {
					return base
				}
			}
		}
	}
	for _, sx := range enSuffixes {
		if strings.HasSuffix(w, sx) && len(w)-len(sx) >= 3 {
			if base := strings.TrimSuffix(w, sx); f.words[base] {
				return base
			}
		}
	}
	return ""
}

// arabizi **أرقامُ العربيزي حروفاً** — في كلمةٍ فيها حرفٌ عربيّ («7مار»).
var arabizi = map[rune]rune{
	'2': 'ا', '3': 'ع', '5': 'خ', '6': 'ط', '7': 'ح', '8': 'ق', '9': 'ص',
}

// normalizeWord **يردّ الكلمةَ إلى شكلٍ واحدٍ يُقارَن به.**
func normalizeWord(s string) string {
	hasArabic := false
	for _, r := range s {
		if unicode.Is(unicode.Arabic, r) && unicode.IsLetter(r) {
			hasArabic = true
			break
		}
	}
	var b strings.Builder
	for _, r := range s {
		r = unicode.ToLower(r)
		switch r {
		case 'أ', 'إ', 'آ', 'ٱ', 'ء':
			r = 'ا'
		case 'ة', 'ۀ', 'ھ':
			r = 'ه'
		case 'ى', 'ی', 'ئ':
			r = 'ي'
		case 'ؤ':
			r = 'و'
		case 'ک':
			r = 'ك'
		case 'ـ': // تطويل
			continue
		}
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if hasArabic {
			if m, ok := arabizi[r]; ok {
				r = m
			}
		} else {
			// **وفي اللاتينيّة: الصفرُ «o» والواحدُ «i»** — «sh1t».
			switch r {
			case '0':
				r = 'o'
			case '1':
				r = 'i'
			}
		}
		b.WriteRune(r)
	}
	return collapse(b.String())
}

// collapse **التكرارُ يُطوى إلى حرفٍ واحد** — «كلللب» تصير «كلب».
func collapse(s string) string {
	var b strings.Builder
	var last rune = -1
	for _, r := range s {
		if r == last {
			continue
		}
		last = r
		b.WriteRune(r)
	}
	return b.String()
}
