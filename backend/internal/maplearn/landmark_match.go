package maplearn

// ══════════════════════════════════════════════════════════════════════
// **«موجود من قبل» — هل المعلمُ المرسَلُ معلمٌ نعرفه؟** (قرارُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «لازم كل سائق باليوم يرسل اقل شي 5 معالم، واذا ارسلو شي قديم طالعه».
//
// **دالّتان صافيتان لا تلمسان القاعدة** — والمسافةُ والمصادرُ (المعالمُ ومتاجرُنا
// واقتراحاتُ السائقين) في `server/landmark_quota.go`.
//
//   - `LandmarkKeyOf` تفكّ الاسمَ: كلماتُه موحَّدةً (`Normalize`) بلا «ال» ولا همزةٍ
//     منفردة، **وكلمةُ النوع في أوّله تُرفع عنه** فـ«جامع النور» و«مسجد النور» كلاهما «نور».
//   - `SameLandmark` تحكم: **الاسمان سواء والعائلةُ واحدة** ⇒ واحد. **وأحدُهما داخلَ الآخر
//     بكلماتٍ كاملة، أو بينهما حرفٌ والاسمُ خمسةُ أحرفٍ فأكثر** ⇒ واحدٌ إن كانت العائلةُ
//     نفسَها بالضبط. **و«غير ذلك» لا يُطابَق إلّا بالاسم كاملاً.**
//
// **والعائلةُ أنواعُ السائق السبعة** (mosque · pharmacy · school · hospital · shop ·
// roundabout · other) — وصنفُ المعلم المستورد يُترجَم إليها في الخادم.

import (
	"strings"
	"unicode"
)

// LandmarkKey **اسمُ معلمٍ مفكوكٌ للمقارنة.**
type LandmarkKey struct {
	// Full كلُّ الكلمات — موحَّدةً، بلا «ال»، بلا همزةٍ منفردة.
	Full string
	// Core بلا كلمات النوع في أوّله — «الجامع الكبير» و«المسجد الكبير» كلاهما «كبير».
	Core string
	// Kind نوعُ كلمة النوع في أوّله (`typeKinds`) — وفارغٌ إن لم تكن.
	Kind string
	// Group **مترادفُ كلمة النوع**: «جامع» و«مسجد» واحد، **و«فرن» غيرُ «سوبرماركت»** وكلاهما
	// محلّ. وفارغٌ إن لم تكن كلمةُ نوعٍ أو كانت عامّة («محل»، «دكان»).
	Group string
}

// typeGroups **كلماتُ النوع المترادفة** — وما ليس هنا من `typeKinds` مجموعتُه نفسُه.
// **والفارغُ عامٌّ** يوافق أيَّ مجموعة («محل الأمل» قد يكون «سوبرماركت الأمل»).
var typeGroups = map[string]string{
	"جامع": "mosque", "مسجد": "mosque", "مصلي": "mosque",
	"صيدليه": "pharmacy",
	"مشفي":   "hospital", "مستشفي": "hospital",
	"مستوصف": "clinic", "عياده": "clinic",
	"مدرسه": "school", "ثانويه": "school", "اعداديه": "school",
	"روضه": "kindergarten", "حضانه": "kindergarten",
	"جامعه": "university", "كليه": "university",
	"فرن": "bakery", "مخبز": "bakery",
	"سوبرماركت": "grocery", "ماركت": "grocery", "بقاليه": "grocery",
	"مقهي": "cafe", "كافيه": "cafe",
	"محطه": "fuel", "كازيه": "fuel",
	"بنك": "bank", "مصرف": "bank",
	"دوار": "roundabout", "ساحه": "roundabout",
	"محل": "", "دكان": "",
}

// leadModifiers **كلماتٌ تسبق كلمةَ النوع ولا تسمّي** — «سوبر ماركت الأمل» = «ماركت الأمل».
var leadModifiers = setOf("سوبر", "هايبر", "ميني")

func typeGroupOf(w string) string {
	if g, ok := typeGroups[w]; ok {
		return g
	}
	return w
}

// landmarkWords **كلماتُ الاسم للمقارنة** — `Normalize` ثمّ بلا همزةٍ منفردة («الشفاء» =
// «الشفا») ولا علامات، وبلا «ال» في أوّل كلّ كلمة.
func landmarkWords(name string) []string {
	var b strings.Builder
	for _, r := range Normalize(name) {
		switch {
		case r == 'ء':
			continue
		case r == 'ی': // ياءٌ فارسيّة
			b.WriteRune('ي')
		case r == 'ک': // كافٌ فارسيّة
			b.WriteRune('ك')
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	fields := strings.Fields(b.String())
	out := make([]string, 0, len(fields))
	for _, w := range fields {
		out = append(out, stripAl(w))
	}
	return out
}

// LandmarkKeyOf **يفكّ اسمَ معلمٍ للمقارنة.**
//
// **وكلمةُ النوع الأولى تُرفع وحدَها** — «جامع الحديقة» اسمُه «الحديقة» لا كلمةَ نوعٍ ثانية.
// **إلّا كلمةً عامّةً قبل نوعٍ أخصّ**: «مركز صيدلية الشفاء» ⇒ «شفا»، ونوعُه الصيدليّة.
// **ولا يُرفع آخرُ كلمة** — «صيدلية» وحدَها اسمُها ونوعُها.
func LandmarkKeyOf(name string) LandmarkKey {
	words := landmarkWords(name)
	k := LandmarkKey{Full: strings.Join(words, " ")}
	if len(words) == 0 {
		return k
	}
	i := 0
	for i < len(words)-1 && leadModifiers[words[i]] {
		i++
	}
	typeWord := ""
	if _, ok := typeKinds[words[i]]; ok {
		typeWord = words[i]
		// **«مركز» قبل «صيدلية»** — العامُّ يُرفع ويُؤخذ الأخصُّ بعده.
		if typeKinds[typeWord] == "other" && i+1 < len(words) {
			if next, ok := typeKinds[words[i+1]]; ok && next != "other" {
				i++
				typeWord = words[i]
			}
		}
		if i < len(words)-1 {
			i++
		}
	}
	k.Core = strings.Join(words[i:], " ")
	if typeWord != "" {
		k.Kind = typeKinds[typeWord]
		k.Group = typeGroupOf(typeWord)
	}
	return k
}

// DriverFamily **عائلةُ ما أرسله السائق** — نوعُه، **و«غير ذلك» يأخذ نوعَ كلمته** إن قالت
// نوعاً («دوار النعيم» بنوع «غير ذلك» دوّار).
func DriverFamily(kind string, k LandmarkKey) string {
	if kind == "other" && k.Kind != "" && k.Kind != "other" {
		return k.Kind
	}
	return kind
}

// SameLandmark **أهما معلمٌ واحد؟** — `famA` و`famB` من أنواع السائق السبعة، **و«other»
// لصنفٍ معروفٍ لا نوعَ له عندنا** (حديقة، مخفر…)، **وفارغٌ لصنفٍ لا يُرسم** (مقبرة، حيّ، مكتب).
func SameLandmark(a LandmarkKey, famA string, b LandmarkKey, famB string) bool {
	if a.Full == "" || b.Full == "" {
		return false
	}
	// **وكلمتا نوعٍ لا تترادفان مكانان** — «فرن الأمل» غيرُ «سوبرماركت الأمل».
	if a.Group != "" && b.Group != "" && a.Group != b.Group {
		return false
	}
	// **و«غير ذلك» والصنفُ الذي لا يُرسم بالاسم كاملاً وحدَه** (مراجعة ٢٠٢٦-١١-٠٥: «مقبرة تل
	// البيعة» بحرفها كانت تُحسب جديدة) — **ولا يكفيه احتواءٌ ولا حرف**: «النور» ليس «حي النور».
	if famA == "" || famB == "" || famA == "other" || famB == "other" {
		return a.Full == b.Full
	}
	// **وعائلتان مختلفتان مكانان** — «صيدلية الشفاء» غيرُ «مشفى الشفاء».
	if famA != famB {
		return false
	}
	if a.Core == b.Core {
		return true
	}
	if wordsInside(a.Core, b.Core) || wordsInside(b.Core, a.Core) {
		return true
	}
	// **وحرفٌ واحدٌ في اسمٍ طويل** — `withinOneEdit` نفسُها التي تدمج أخطاءَ أسماء الأحياء.
	return min(len([]rune(a.Core)), len([]rune(b.Core))) >= 5 && withinOneEdit(a.Core, b.Core)
}

// SameOwnLandmark **أهو ما أرسله السائقُ نفسُه من قبل؟** — بالاسم، **ونوعُ القائمة لا يفرّق**
// (مراجعة ٢٠٢٦-١١-٠٥): «جامع النور» بنوع جامع ثمّ بنوع محلّ كان يُحسب مرّتين.
//
//   - **الاسمُ كاملاً سواء** ⇒ هو، أيّاً كان النوعُ المختار.
//   - **أو لبُّه سواءٌ** («نور») **وكلمتا النوع مترادفتان** («جامع» ثمّ «مسجد»)، **أو العائلةُ
//     واحدة، أو أحدُهما بلا كلمة نوع** («جامع النور» ثمّ «النور» بنوع محلّ).
//   - **و«محل النور» بجانب «جامع النور» مكانان** — المحلُّ يأخذ اسمَ الجامع الذي بجانبه كثيراً.
func SameOwnLandmark(a LandmarkKey, famA string, b LandmarkKey, famB string) bool {
	if a.Full == "" || b.Full == "" {
		return false
	}
	if a.Full == b.Full {
		return true
	}
	if a.Core == "" || a.Core != b.Core {
		return false
	}
	if a.Group != "" && b.Group != "" {
		return a.Group == b.Group
	}
	return famA == famB || a.Kind == "" || b.Kind == ""
}

// wordsInside **أكلماتُ `inner` متتاليةٌ داخل `outer`؟** — بكلماتٍ كاملةٍ لا بحروف، **ورقمٌ وحدَه
// لا يُحتوى**: «مدرسة 15» ليست «مدرسة الشهيد 15».
func wordsInside(outer, inner string) bool {
	if inner == "" || outer == "" || !strings.ContainsFunc(inner, unicode.IsLetter) {
		return false
	}
	return strings.Contains(" "+outer+" ", " "+inner+" ")
}
