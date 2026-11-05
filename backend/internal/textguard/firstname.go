package textguard

import (
	"strings"
	"unicode"
)

// ══════════════════════════════════════════════════════════════════════
// **الاسمُ الأوّل — ما يراه السائقُ من اسم الزبون**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-١١-٠٥: «السائق بيشوف الاسم الأول للزبون بس».) **الاسمُ
// الكاملُ مع الحيّ والطلبات المتكرّرة قائمةُ زبائن** — والأوّلُ وحده لا يكفي
// لها. **وهنا لأنّ `textguard` لا يستورد غيرَ `httpx`**: تستعمله `orders`
// و`comms` و`server` بلا حلقة.

// firstNameJoiners **كلمةٌ لا تقوم وحدها اسماً** — «عبد الله» و«أبو محمد»
// و«أم علي» اسمٌ واحدٌ من كلمتين. **وتُقارَن مطبَّعةً** (`normalizeWord`):
// «أبو» و«ابو» واحد.
var firstNameJoiners = map[string]bool{
	"عبد": true, "ابو": true, "ام": true, "ابن": true, "بن": true, "الحاج": true, "الحاجه": true,
	"abd": true, "abdul": true, "abu": true, "abo": true, "um": true, "umm": true, "ibn": true, "bin": true,
}

// FirstName **الاسمُ الأوّل** — الكلمةُ الأولى، **أو الأوليان** إن كانت
// الأولى «عبد/أبو/أم/ابن/بن/الحاج» وبعدها كلمة.
//
// **وكلُّ كلمةٍ فيها رقمٌ تُسقط** — فمن كتب رقمَه مكانَ اسمه (أو بجانبه)
// لا يصل رقمُه إلى السائق من هذا الباب. **والفارغُ فارغ.**
func FirstName(full string) string {
	var words []string
	for _, w := range strings.Fields(full) {
		if strings.IndexFunc(w, unicode.IsDigit) >= 0 {
			continue
		}
		words = append(words, w)
	}
	if len(words) == 0 {
		return ""
	}
	if len(words) >= 2 && firstNameJoiners[normalizeWord(words[0])] {
		return words[0] + " " + words[1]
	}
	return words[0]
}
