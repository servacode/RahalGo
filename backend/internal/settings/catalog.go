package settings

// كتالوج الإعدادات — المصدر الواحد لما تعرفه المنصة عن مفاتيحها.
//
// كانت `Set` تقبل أيّ مفتاح بأيّ قيمة: `INSERT ... ON CONFLICT`. ومعنى ذلك
// ثلاثة أعطاب لا واحد:
//
//  1. **خطأٌ مطبعيّ في المفتاح يُنشئ مفتاحاً جديداً** بدل أن يُعدّل القديم —
//     فيمضي النظام بالقيمة الافتراضية ولا يقول شيئاً، والمالك يظنّ أنه غيّر.
//  2. **قيمةٌ خارج المعقول تُقبل**: `drivers.share_value = 200` يجعل المنصة
//     تدفع ضعف رسم التوصيل لكل سائق في كل طلب، صامتةً.
//  3. **قيمةٌ من نوعٍ خاطئ تكسر خطّ التوصيل**: `(value#>>'{}')::float8` على
//     نصٍّ يرمي خطأً **داخل معاملة التسليم** — فينكسر النظام بحرفٍ في مربّع نص.
//
// فصار لكل مفتاح **تعريفٌ يعرفه الخادم**: نوعه، ومداه، وخياراته، ومن يملكه.
// والمفتاح الذي ليس في هذا الكتالوج مرفوض.
//
// والتسميات ليست هنا بل في `packages/i18n` تحت `admin.settings.*` — النصّ
// الظاهر للمستخدم يعيش في مكانٍ واحد (القاعدة الأولى في المشروع).

import (
	"encoding/json"
	"fmt"
)

// Kind نوع الإعداد — يحدّد الحقل الذي تعرضه اللوحة والتحقق الذي يجريه الخادم.
type Kind string

const (
	KindInt    Kind = "int"    // عدد صحيح ضمن [Min, Max]
	KindBool   Kind = "bool"   // مفتاح تبديل
	KindChoice Kind = "choice" // واحدٌ من Options
	KindText   Kind = "text"   // نصّ حرّ بطول أقصى
	KindMoney  Kind = "money"  // مبلغ بالليرة — عدد صحيح غير سالب
)

// Group مجموعة العرض في اللوحة — كي لا تكون خمسة وعشرون مفتاحاً في عمودٍ واحد.
type Group string

const (
	GroupOrders    Group = "orders"
	GroupDrivers   Group = "drivers"
	GroupMerchants Group = "merchants"
	GroupSales     Group = "sales"
	GroupCustomers Group = "customers"
	GroupPayouts   Group = "payouts"
	GroupSecurity  Group = "security"
	GroupSupport   Group = "support"
	GroupPlatform  Group = "platform"
	// GroupNew **القسمُ الذي يُبنى على مراحل.**
	//
	// الإعداداتُ نمت مفتاحاً مفتاحاً، **فصار المفتاحُ يقع حيث وقع لا حيث
	// يُبحث عنه**: عمولةُ المتاجر في «المتاجر» وهامشُ المنصة في «التسعير»
	// وحصّةُ السائق في «السائقين» — **ومن أراد أن يرى ربحَه يفتح ثلاثة.**
	//
	// فهذا موضعٌ نُعيد فيه البناءَ بقرارٍ لا بتراكم: يُذكر الإعدادُ، **فإن
	// كان في القديم نُقل وصُحّح، وإن لم يكن أُضيف.**
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-٠٤.)
	GroupNew Group = "new"
)

// Groups **ترتيبُ الأقسام في اللوحة — ومصدرُه الواحد.**
//
// كان يُشتقّ من ترتيب `Catalog`: أوّلُ ظهورٍ للمجموعة هو موضعُها. **فقسمٌ بلا
// مفاتيحَ لا يظهر أصلاً** — ولا يُبنى قسمٌ يُملأ على مراحل، إذ لا سبيل إلى
// رؤيته قبل أن يمتلئ.
//
// **والاشتقاقُ يُخفي قراراً أيضاً**: موضعُ القسم صار أثراً جانبيّاً لترتيب
// المفاتيح — **فنقلُ مفتاحٍ ينقل قسماً معه** بلا أن يقصد أحد.
var Groups = []Group{
	GroupNew,
}

// Default افتراضُ مفتاحٍ رقميّ — **لمن لا مخزنَ لديه.**
//
// **ولا يُكتب الرقمُ في القارئ**: خدمةٌ تُبنى قبل حقن المخزن (اختبارٌ أو
// إقلاعٌ نصفُ مهيَّأ) تحتاج رقماً، **وكتابتُه عندها تُنشئ نسخةً ثانيةً من
// الافتراض** تفترق عن الفهرس بلا صوت.
//
// **ومفتاحٌ مجهولٌ يعيد صفراً** — لا يُخترع له رقم.
func Default(key string) int64 {
	if def, ok := Lookup(key); ok {
		if n, ok := toNumber(def.Default); ok {
			return int64(n)
		}
	}
	return 0
}

// Def تعريف مفتاح واحد.
type Def struct {
	Key     string   `json:"key"`
	Group   Group    `json:"group"`
	Kind    Kind     `json:"kind"`
	Min     float64  `json:"min,omitempty"`
	Max     float64  `json:"max,omitempty"`
	Options []string `json:"options,omitempty"`
	// Unit وحدة القياس — تُعرض بجانب الحقل: دقيقة، ثانية، ليرة، بالمئة، طلب.
	// وغيابُها هو ما يجعل «٧٠» و«٥٠٠٠٠٠» متشابهين في عين من يقرأ.
	Unit string `json:"unit,omitempty"`
	// Default القيمة التي يعمل بها النظام إن غاب المفتاح — تُعرض للمالك كي
	// يعرف إلى أين يعود إن أخطأ.
	Default any `json:"default"`
	// Sensitive يمسّ المال مباشرةً — تُبرزه اللوحة ويُطلب تأكيدٌ قبل حفظه.
	Sensitive bool `json:"sensitive,omitempty"`
}

// Catalog **فارغٌ عمداً — ولا مفتاحَ فيه.**
//
// # القرار
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «احذف كلَّ الإعدادات بشكلٍ كاملٍ ونهائيّ، لا أريد
// أن يبقى أيُّ إعداد» — بعد «سنعود لنعمل الإعداداتِ بشكلٍ صحيحٍ ونهائيّ».)
//
// **والإعداداتُ نمت مفتاحاً مفتاحاً على مدى شهر**: كلُّ حاجةٍ أضافت مفتاحاً
// حيث وقعت الحاجة. فصار سبعةٌ وأربعون مفتاحاً في عشر مجموعات، **ومنها ما لم
// يُقرَّر يوماً بل وُلد افتراضاً في شيفرة.**
//
// **فيُبدأ من فارغٍ ويُبنى بقرار**: يُذكر الإعدادُ، ويُتّفق على معناه وحدّه
// ونمطه، **ثمّ يُكتب هنا.** وما لم يُذكر لا وجودَ له.
//
// # وماذا يعني الفراغُ للمحرّك
//
// **المفتاحُ الذي ليس هنا لا يُقرأ ولا يُكتب**: `Validate` ترفض كتابتَه،
// و`GetInt` تعيد صفراً، و`GetString` فراغاً. **والقراءةُ لا تنهار** — تعيد
// العدم، وهو ما يجب: **لا إعدادَ يعني لا قاعدة، لا قاعدةً مخبّأة.**
//
// **وأثرُ ذلك موثَّقٌ في `docs/SETTINGS-REBUILD.md`** — ما كان، وما يتوقّف
// بغيابه، وما يجب أن يعود أوّلاً. **فلا يُعاد البناءُ من الذاكرة ناقصاً.**
var Catalog = []Def{}

// byKey فهرسٌ يُبنى مرّة — البحث الخطّي في كل كتابة إعداد ترفٌ لا داعي له.
var byKey = func() map[string]Def {
	return indexCatalog()
}()

// indexCatalog يبني فهرسَ البحث من `Catalog`.
//
// **ويُنادى عند التحميل وحين يُبدَّل الفهرس** — والاختباراتُ تُبدّله لتفحص
// آلةَ التحقّق بمفاتيحها هي. **وخريطةٌ تُبنى مرّةً ولا تُعاد تجعل التبديلَ
// بلا أثر**، فتُقرأ نتيجةُ الاختبار من فهرسٍ لم يعد قائماً.
func indexCatalog() map[string]Def {
	m := make(map[string]Def, len(Catalog))
	for _, d := range Catalog {
		m[d.Key] = d
	}
	return m
}

// ReindexCatalog يُعيد بناءَ الفهرس بعد تبديل `Catalog` — **للاختبارات.**
func ReindexCatalog() { byKey = indexCatalog() }

// Lookup تعريف المفتاح، أو false إن كان مجهولاً.
func Lookup(key string) (Def, bool) {
	d, ok := byKey[key]
	return d, ok
}

// ErrUnknownKey مفتاحٌ ليس في الكتالوج — يُرفض ولا يُنشأ.
type ErrUnknownKey struct{ Key string }

func (e ErrUnknownKey) Error() string { return "settings: مفتاح مجهول: " + e.Key }

// ErrInvalidValue قيمةٌ لا تطابق تعريف مفتاحها.
type ErrInvalidValue struct {
	Key    string
	Reason string
}

func (e ErrInvalidValue) Error() string {
	return fmt.Sprintf("settings: قيمة غير صالحة لـ%s: %s", e.Key, e.Reason)
}

// Validate يتحقق أن القيمة تطابق تعريف المفتاح، ويعيدها **مطبَّعة**.
//
// والتطبيع مقصود: JSON لا يفرّق بين ٧٠ و٧٠٫٠، والقاعدة تقرأ `::float8` فتقبل
// الاثنين — لكنّ اللوحة تعرض `70.0` فيظنّ القارئ أن ثمّة كسراً. فيُعاد الصحيح
// صحيحاً.
func Validate(key string, v any) (any, error) {
	d, ok := Lookup(key)
	if !ok {
		return nil, ErrUnknownKey{Key: key}
	}
	switch d.Kind {
	case KindInt, KindMoney:
		n, ok := toNumber(v)
		if !ok {
			return nil, ErrInvalidValue{Key: key, Reason: "ليست رقماً"}
		}
		if n != float64(int64(n)) {
			return nil, ErrInvalidValue{Key: key, Reason: "ليست عدداً صحيحاً"}
		}
		if n < d.Min || n > d.Max {
			return nil, ErrInvalidValue{Key: key,
				Reason: fmt.Sprintf("خارج المدى [%v..%v]", d.Min, d.Max)}
		}
		return int64(n), nil

	case KindBool:
		b, ok := v.(bool)
		if !ok {
			return nil, ErrInvalidValue{Key: key, Reason: "ليست نعم/لا"}
		}
		return b, nil

	case KindChoice:
		s, ok := v.(string)
		if !ok {
			return nil, ErrInvalidValue{Key: key, Reason: "ليست نصاً"}
		}
		for _, o := range d.Options {
			if o == s {
				return s, nil
			}
		}
		return nil, ErrInvalidValue{Key: key, Reason: "ليست من الخيارات المتاحة"}

	case KindText:
		s, ok := v.(string)
		if !ok {
			return nil, ErrInvalidValue{Key: key, Reason: "ليست نصاً"}
		}
		// الطول بالمحارف لا بالبايتات: الحرف العربي بايتان، فقياسُه بالبايت
		// يقصّ نصّاً عربياً عند نصف ما يقصّ عنده نصّاً لاتينياً.
		if d.Max > 0 && float64(len([]rune(s))) > d.Max {
			return nil, ErrInvalidValue{Key: key,
				Reason: fmt.Sprintf("أطول من %v محرفاً", d.Max)}
		}
		return s, nil
	}
	return nil, ErrInvalidValue{Key: key, Reason: "نوعٌ غير معروف"}
}

// toNumber يقبل ما يصل من JSON رقماً — وهو `float64` دائماً بعد فكّ الترميز.
func toNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
