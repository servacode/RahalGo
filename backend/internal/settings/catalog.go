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
	GroupPlatform  Group = "platform"
)

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

// Catalog كل ما تعرفه المنصة. الترتيب هنا هو ترتيب العرض داخل المجموعة.
var Catalog = []Def{
	// ── الطلبات ───────────────────────────────────────────────────────────
	{Key: "orders.accept_timeout_min", Group: GroupOrders, Kind: KindInt,
		Min: 1, Max: 120, Unit: "minute", Default: 5},
	{Key: "orders.driver_timeout_min", Group: GroupOrders, Kind: KindInt,
		Min: 1, Max: 120, Unit: "minute", Default: 10},
	{Key: "orders.delivery_timeout_min", Group: GroupOrders, Kind: KindInt,
		Min: 5, Max: 480, Unit: "minute", Default: 60},
	{Key: "orders.delivery_estimate_min", Group: GroupOrders, Kind: KindInt,
		Min: 1, Max: 240, Unit: "minute", Default: 15},
	// النافذة بالثواني لا بالدقائق: دقيقةٌ واحدة أقصر من أن تُقاس بالدقائق،
	// و١٢٠ ثانية قرارٌ أدقّ من «دقيقتين».
	{Key: "orders.customer_cancel_window_sec", Group: GroupOrders, Kind: KindInt,
		Min: 0, Max: 1800, Unit: "second", Default: 120},

	// ── السائقون ──────────────────────────────────────────────────────────
	{Key: "drivers.share_mode", Group: GroupDrivers, Kind: KindChoice,
		Options: []string{"percent", "fixed"}, Default: "percent", Sensitive: true},
	// **مفتاحان لا مفتاح واحد.**
	//
	// كان `drivers.share_value` واحداً يخدم المعنيين: نسبةً مئوية في نمط، ومبلغاً
	// بالليرة في نمطٍ آخر. فوجب أن يتّسع مداه لكليهما — فلم يحرس أيّاً منهما:
	// المئتان تمرّ لأنها مبلغٌ معقول، وهي نسبةٌ كارثية تجعل المنصة تدفع ضعف ما
	// قبضت في كل طلب. **والمفتاح الذي يعني معنيين لا يُحرَس.**
	//
	// (كشفه اختبارُ المدى على الكتالوج نفسه قبل أن يصل إلى أحد.)
	{Key: "drivers.share_percent", Group: GroupDrivers, Kind: KindInt,
		Min: 0, Max: 100, Unit: "percent", Default: 70, Sensitive: true},
	{Key: "drivers.share_fixed", Group: GroupDrivers, Kind: KindMoney,
		Min: 0, Max: 10000000, Unit: "currency", Default: 5000, Sensitive: true},
	{Key: "drivers.cash_limit", Group: GroupDrivers, Kind: KindMoney,
		Min: 0, Max: 100000000, Unit: "currency", Default: 500000, Sensitive: true},
	// **جديد**: كان السائق يأخذ ما شاء من الطلبات ما دام السقف النقدي يتّسع.
	// وخمسة طلبات بيد سائقٍ واحد تعني أربعة زبائن ينتظرون ساعة.
	{Key: "drivers.max_active_orders", Group: GroupDrivers, Kind: KindInt,
		Min: 1, Max: 20, Unit: "order", Default: 2},

	// ── المتاجر ───────────────────────────────────────────────────────────
	{Key: "merchants.default_commission_percent", Group: GroupMerchants, Kind: KindInt,
		Min: 0, Max: 100, Unit: "percent", Default: 10, Sensitive: true},
	{Key: "merchants.menu_requires_approval", Group: GroupMerchants, Kind: KindBool,
		Default: false},
	// **جديد**: كان ٢٠ دقيقة مكتوباً في الترحيل كافتراضي عمود. والمتجر الجديد
	// يرثه بلا أن يملك المالك تغييره لمن يأتي بعده.
	{Key: "merchants.default_prep_minutes", Group: GroupMerchants, Kind: KindInt,
		Min: 1, Max: 240, Unit: "minute", Default: 20},

	// ── المندوبون ─────────────────────────────────────────────────────────
	{Key: "sales.commission_percent", Group: GroupSales, Kind: KindInt,
		Min: 0, Max: 100, Unit: "percent", Default: 10, Sensitive: true},
	{Key: "sales.activation_orders", Group: GroupSales, Kind: KindInt,
		Min: 1, Max: 100, Unit: "order", Default: 5, Sensitive: true},
	{Key: "sales.monthly_target", Group: GroupSales, Kind: KindInt,
		Min: 1, Max: 100, Unit: "merchant", Default: 5},

	// ── الزبائن ───────────────────────────────────────────────────────────
	// **جديد**: كان `maxAddresses = 10` ثابتاً في المعالِج.
	{Key: "customers.max_addresses", Group: GroupCustomers, Kind: KindInt,
		Min: 1, Max: 50, Unit: "address", Default: 10},
	// **إعدادٌ لا شرطٌ مضمَّن**: قاعدةُ عملٍ يملك المالك تشديدها وإرخاءها. وقد
	// يحتاج إرخاءها يوماً تعطّل فيه بوت واتساب — وقاعدةٌ مضمَّنة في الشيفرة
	// تعني توقّف المنصة حتى نشرٍ جديد.
	{Key: "customers.require_whatsapp", Group: GroupCustomers, Kind: KindBool,
		Default: true},

	// ── السحوبات ──────────────────────────────────────────────────────────
	// **جديد**: لم يكن حدٌّ أدنى. فطلبُ سحبٍ بليرةٍ واحدة يمرّ بدورة الموافقة
	// كاملةً ويشغل المالية — وقيمة القرار أكبر من قيمة المبلغ.
	{Key: "payouts.min_amount", Group: GroupPayouts, Kind: KindMoney,
		Min: 0, Max: 100000000, Unit: "currency", Default: 50000},

	// ── الأمان ────────────────────────────────────────────────────────────
	// **جديد**: كانا ثابتين في `identity`. ورفعُ طول كلمة المرور قرارُ أمانٍ
	// يتّخذه المالك لا قرارُ نشرٍ ينتظر مبرمجاً.
	{Key: "security.password_min_length", Group: GroupSecurity, Kind: KindInt,
		Min: 6, Max: 64, Unit: "letter", Default: 8},
	{Key: "security.login_max_attempts", Group: GroupSecurity, Kind: KindInt,
		Min: 3, Max: 50, Unit: "attempt", Default: 5},

	// ── المنصة ────────────────────────────────────────────────────────────
	{Key: "platform.invite_code", Group: GroupPlatform, Kind: KindText, Max: 32,
		Default: "RAHALGO"},
	// **جديد**: رقمٌ يظهر للزبون حين لا يجد جواباً. وكان لا وجود له أصلاً،
	// فالشكوى تذهب إلى التذاكر وحدها ومن لا يعرف التذاكر لا يجد باباً.
	{Key: "platform.support_phone", Group: GroupPlatform, Kind: KindText, Max: 20,
		Default: ""},
	{Key: "whatsapp.otp_template", Group: GroupPlatform, Kind: KindText, Max: 500,
		Default: "رمز التحقق الخاص بك في رحال غو هو: {code}\n\nلا تشارك هذا الرمز مع أي شخص."},
}

// byKey فهرسٌ يُبنى مرّة — البحث الخطّي في كل كتابة إعداد ترفٌ لا داعي له.
var byKey = func() map[string]Def {
	m := make(map[string]Def, len(Catalog))
	for _, d := range Catalog {
		m[d.Key] = d
	}
	return m
}()

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
