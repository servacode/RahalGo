// Package itemmatch **مطابقةُ اسمِ صنفٍ باسمِ صنفٍ في متجرٍ آخر — بالمعنى لا بالحرف.**
//
// # لماذا
//
// كان تحويلُ الطلب إلى متجرٍ آخر يشترط **الاسمَ نفسَه حرفاً بحرف** — فـ«رز مصري»
// لا يجد «أرز مصري»، **ويقف التحويلُ كلُّه على همزة.** وقرارُ المالك (٢٠٢٦-١٠-٠٣):
// «المفروض ما يكون نفس الاسم بالضبط، لأنّه ممكن يكون نفسه بتسميةٍ مختلفة».
//
// # وما تفعله — وما لا تفعله
//
// **تقترح ولا تقرّر**: الموظّفُ يرى المقترحَ ويؤكّده أو يختار غيرَه بيده.
// **فالخطأُ هنا يكلّف نظرةً لا شحنة** — ولذلك تُفضَّل المطابقةُ الحذرة:
// «شاورما دجاج» لا تُقترح لـ«شاورما لحم»، **فالسعرُ والطعمُ غيرُهما.**
//
// # القواعد
//
//   - **التشكيلُ والتطويلُ يُحذفان**: «أَرُزّ» = «ارز».
//   - **الألفاتُ واحدة**: أ إ آ ٱ ← ا · **والتاءُ المربوطةُ هاء**: ة ← ه ·
//     **والألفُ المقصورةُ ياء**: ى ← ي · ؤ ← و · ئ ← ي.
//   - **«ال» في أوّل كلّ كلمةٍ تُحذف**: «الرز» = «رز».
//   - **وألفٌ في أوّل الكلمة تُحذف**: «أرز» = «رز» — **وهي القاعدةُ العامّة
//     لمثالِ المالك**: كلمتان تتساويان بعد حذف ألفٍ أولى فهما واحدة.
//   - **والترقيمُ والفراغاتُ فاصلٌ واحد**، والأرقامُ الهنديّةُ عربيّة.
//   - **وكلمتان طويلتان (٥ أحرفٍ فأكثر) بينهما حرفٌ واحدٌ تتساويان**:
//     «شاورما» = «شاورمة».
//
// # والدرجة
//
// **تُقاس بالكلمات المشتركة**: كم من كلمات الصنف المطلوب وُجد في المقابل
// (وزنُه ٠٫٨)، وكم من كلمات المقابل كان مطلوباً (وزنُه ٠٫٢). **فالكلماتُ
// الزائدةُ في المقابل مقبولةٌ وتنقص قليلاً**: «رز مصري» ← «رز مصري حبة طويلة»
// درجتُه ٠٫٩، **والاسمُ نفسُه بعد التسوية ١.**
package itemmatch

import (
	"strings"
	"unicode"
)

// Threshold **أدنى درجةٍ تُقترح** — تحتها لا مقابل.
//
// **ومحسوبةٌ لتردّ نصفَ الاسم**: «شاورما دجاج» و«شاورما لحم» يشتركان في كلمةٍ
// من اثنتين ← ٠٫٥ — **فلا تُقترح**. وثلاثُ كلماتٍ يشترك منها اثنتان ← ٠٫٦٧ —
// **فتُقترح**: «بيبسي علبه كبيره» ← «بيبسي كبيره».
const Threshold = 0.6

// Normalize يردّ الاسمَ كلماتٍ مسوّاة — **وهي ما يُقارَن لا الاسمُ كما كُتب.**
func Normalize(s string) []string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		// **التشكيلُ** (فتحتان … سكون، والمدّةُ والهمزةُ فوقُ وتحت) **والألفُ الخنجريّة.**
		case r >= 0x064B && r <= 0x065F, r == 0x0670:
			continue
		case r == 0x0640: // التطويل
			continue
		case r == 'أ', r == 'إ', r == 'آ', r == 'ٱ':
			b.WriteRune('ا')
		case r == 'ة':
			b.WriteRune('ه')
		case r == 'ى', r == 'ئ':
			b.WriteRune('ي')
		case r == 'ؤ':
			b.WriteRune('و')
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			// **والترقيمُ فاصل** — «رز-مصري» و«رز (مصري)» كـ«رز مصري».
			b.WriteRune(' ')
		}
	}
	out := []string{}
	for _, w := range strings.Fields(b.String()) {
		w = canonWord(w)
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}

// canonWord **يحذف «ال» ثمّ ألفاً أولى** — ويُبقي حرفين على الأقلّ.
//
// **والحدُّ حرفان** كي لا تُمحى كلمةٌ قصيرة: «ال» وحدَها تبقى، و«اب» تبقى.
func canonWord(w string) string {
	rs := []rune(w)
	if len(rs) >= 4 && rs[0] == 'ا' && rs[1] == 'ل' {
		rs = rs[2:]
	}
	if len(rs) >= 3 && rs[0] == 'ا' {
		rs = rs[1:]
	}
	return string(rs)
}

// sameWord **كلمتان واحدةٌ بعد التسوية** — أو طويلتان بينهما حرف.
func sameWord(a, b string) bool {
	if a == b {
		return true
	}
	ra, rb := []rune(a), []rune(b)
	if len(ra) < 5 || len(rb) < 5 {
		return false
	}
	return withinOneEdit(ra, rb)
}

// withinOneEdit **أبدالٌ أو حذفٌ أو زيادةُ حرفٍ واحد** لا أكثر.
func withinOneEdit(a, b []rune) bool {
	if len(a) < len(b) {
		a, b = b, a
	}
	if len(a)-len(b) > 1 {
		return false
	}
	i, j, diff := 0, 0, 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			i++
			j++
			continue
		}
		diff++
		if diff > 1 {
			return false
		}
		if len(a) == len(b) {
			j++
		}
		i++
	}
	return diff+(len(a)-i) <= 1
}

// Score **درجةُ مطابقة اسمِ الصنف المطلوب لاسمِ صنفٍ في المتجر** — بين ٠ و١.
func Score(want, have string) float64 {
	return scoreWords(Normalize(want), Normalize(have))
}

func scoreWords(w, h []string) float64 {
	if len(w) == 0 || len(h) == 0 {
		return 0
	}
	// **كلُّ كلمةٍ في المقابل تُحسب مرّةً** — كي لا تُغطّي «رز» واحدةٌ
	// «رز رز» مكرّرة.
	used := make([]bool, len(h))
	common := 0
	for _, x := range w {
		for k, y := range h {
			if !used[k] && sameWord(x, y) {
				used[k] = true
				common++
				break
			}
		}
	}
	if common == 0 {
		return 0
	}
	return 0.8*float64(common)/float64(len(w)) + 0.2*float64(common)/float64(len(h))
}

// Candidate صنفٌ متاحٌ في المتجر المقترَح.
type Candidate struct {
	ID    string
	Name  string
	Price int64 // سعرُ الشراء — ما سيُدفع للمتجر الجديد
}

// Best **أفضلُ مقابلٍ لصنفٍ مطلوب** — ويردّ `-1` إن لم يبلغ أحدٌ الحدّ.
//
// **والتعادلُ يُحسم بالسعر الأقرب** إلى ما كنّا ندفعه: صنفان بالدرجة نفسِها،
// **فالأقربُ سعراً أقربُ أن يكون هو.**
func Best(want string, price int64, cands []Candidate) (int, float64) {
	w := Normalize(want)
	best, bestScore := -1, 0.0
	var bestGap int64
	for i, c := range cands {
		sc := scoreWords(w, Normalize(c.Name))
		if sc < Threshold-1e-9 {
			continue
		}
		gap := c.Price - price
		if gap < 0 {
			gap = -gap
		}
		switch {
		case best < 0, sc > bestScore+1e-9:
		case sc > bestScore-1e-9 && gap < bestGap:
		default:
			continue
		}
		best, bestScore, bestGap = i, sc, gap
	}
	return best, bestScore
}
