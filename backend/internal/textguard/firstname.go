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

// firstNameTitles **لقبٌ قبل الاسم لا يقوم اسماً** — «د. أحمد» يراه السائقُ
// «أحمد» لا «د.». **ويُقارَن مطبَّعاً بلا نقطته.**
var firstNameTitles = map[string]bool{
	"دكتور": true, "الدكتور": true, "مهندس": true, "المهندس": true, "استاذ": true, "الاستاذ": true,
	"استاذه": true, "الاستاذه": true, "الشيخ": true, "شيخ": true, "السيد": true, "السيده": true,
	"dr": true, "eng": true, "mr": true, "mrs": true, "ms": true, "miss": true, "prof": true,
}

// isNameTitle **هل الكلمةُ لقب؟** — من القائمة، **أو حرفٌ أو حرفان بعدهما
// نقطة** («د.» «م.» «أ.»)، **أو حرفٌ واحدٌ وحده** (لا اسمَ من حرف).
func isNameTitle(w string) bool {
	bare := strings.TrimRight(w, ".")
	n := len([]rune(bare))
	if n <= 1 || (bare != w && n <= 2) {
		return true
	}
	return firstNameTitles[normalizeWord(bare)]
}

// FirstName **الاسمُ الأوّل** — الكلمةُ الأولى، **أو الأوليان** إن كانت
// الأولى «عبد/أبو/أم/ابن/بن/الحاج» وبعدها كلمة. **واللقبُ قبله يُتخطّى**
// («د. أحمد علي» ← «أحمد») — **إلّا أن يكون وحده** فيبقى كما كُتب.
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
	k := 0
	for k < len(words) && isNameTitle(words[k]) {
		k++
	}
	if k < len(words) {
		words = words[k:]
	}
	if len(words) >= 2 && firstNameJoiners[normalizeWord(words[0])] {
		return words[0] + " " + words[1]
	}
	return words[0]
}
