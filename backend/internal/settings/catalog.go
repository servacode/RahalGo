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
	GroupOrders, GroupDrivers, GroupMerchants, GroupSales, GroupPlatform,
	GroupCustomers, GroupPayouts, GroupSupport, GroupSecurity,
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

// Catalog كل ما تعرفه المنصة. الترتيب هنا هو ترتيب العرض داخل المجموعة.
var Catalog = []Def{
	// ── القسم الجديد ──────────────────────────────────────────────────────
	//
	// **قواعدُ المال في موضعٍ واحد** — تُبنى بقرارٍ لا بتراكم.
	//
	// **وكلُّها تُقرأ عند كلّ استعمالٍ لا مرّةً عند الإقلاع**: تغييرٌ هنا يسري
	// على الطلب التالي. (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «لازم كلُّ المشروع يأخذ
	// الإعداداتِ هذه بحيث تُطبَّق بشكلٍ حقيقيٍّ فوريٍّ عند أيّ تغيّر».)

	// **أجرةُ التوصيل — ثلاثةُ أنماط.**
	//
	//	مقطوعة    ←  رقمٌ واحدٌ للجميع              ← بسيطٌ ويُفهَم
	//	بالمنطقة  ←  أجرةُ الدائرة التي يقع فيها    ← القائمُ قبلها
	//	بالمسافة  ←  أساسٌ + رقمٌ لكلّ كيلومتر       ← أعدلُ للبعيد والقريب
	//
	// **والتغطيةُ تبقى بالمناطق في الأنماط الثلاثة**: خارجَ الدوائر يُرفض
	// الطلب. **ونمطُ الأجرة غيرُ حدّ التغطية** — ومن خلطهما فتح المدينةَ
	// كلَّها بمجرّد أن جعل الأجرةَ مقطوعة.
	{Key: "delivery.fee_mode", Group: GroupNew, Kind: KindChoice,
		Options: []string{"flat", "zone", "distance"}, Default: "zone", Sensitive: true},
	{Key: "delivery.flat_fee", Group: GroupNew, Kind: KindMoney,
		Min: 0, Max: 10000000, Unit: "currency", Default: 0, Sensitive: true},
	// **أساسُ النمط المسافيّ** — ما يُدفع قبل أن يتحرّك السائقُ متراً.
	//
	// **وصفرٌ فيه يجعل الجارَ يُوصَّل بلا شيء**: خروجُ السائق تكلفةٌ ولو كان
	// الباب مقابلَ الباب.
	{Key: "delivery.base_fee", Group: GroupNew, Kind: KindMoney,
		Min: 0, Max: 10000000, Unit: "currency", Default: 0, Sensitive: true},
	{Key: "delivery.per_km", Group: GroupNew, Kind: KindMoney,
		Min: 0, Max: 1000000, Unit: "currency", Default: 0, Sensitive: true},

	// **الهامشُ الربحيّ** — ما تضيفه المنصةُ فوق سعر الشراء، ويدفعه الزبون.
	{Key: "pricing.margin_mode", Group: GroupNew, Kind: KindChoice,
		Options: []string{"percent", "fixed"}, Default: "percent", Sensitive: true},
	{Key: "pricing.margin_value", Group: GroupNew, Kind: KindInt,
		Min: 0, Max: 1000000, Default: 0, Sensitive: true},

	// **عمولةُ المنصة من المتاجر** — تُقتطع من سعر شراء المتجر.
	//
	// **وحيّةٌ لا لقطة**: كانت تُنسخ في عمود المتجر لحظةَ إنشائه، **فتغييرُ
	// المفتاح لا يمسّ متجراً قائماً** — يظنّ المالكُ أنّه رفع العمولةَ على
	// الجميع وهو لم يرفعها على أحد. (الترحيل ٠٠٦٧.)
	//
	// **ويبقى للمتجر تجاوزٌ خاصٌّ** حين يُتّفق معه على غير العامّ — كتجاوزِ
	// هامش الصنف: **فراغُه «اتبع العام» لا «بلا عمولة».**
	{Key: "merchants.commission_mode", Group: GroupNew, Kind: KindChoice,
		Options: []string{"percent", "fixed"}, Default: "percent", Sensitive: true},
	{Key: "merchants.commission_value", Group: GroupNew, Kind: KindInt,
		Min: 0, Max: 1000000, Default: 10, Sensitive: true},

	// **عمولةُ المندوب** — حصّتُه ممّا تقبضه المنصةُ من متجرِه.
	//
	// **وهي حصّةٌ من حصّتنا لا من البيع**: نسبةٌ من قيمة الطلب تجعل المندوبَ
	// يأخذ أكثرَ ممّا نأخذ في المتاجر منخفضةِ العمولة.
	{Key: "sales.commission_mode", Group: GroupNew, Kind: KindChoice,
		Options: []string{"percent", "fixed"}, Default: "percent", Sensitive: true},
	{Key: "sales.commission_value", Group: GroupNew, Kind: KindInt,
		Min: 0, Max: 1000000, Default: 10, Sensitive: true},

	// **والتقريبُ ليس تجميلاً** — قائمةٌ فيها ٨٬١٢٥ و١١٬٣٧٥ تقول «هذه آلةٌ
	// تحسب»، وقائمةٌ فيها ٨٬٠٠٠ و١١٬٥٠٠ تقول «هذا سعرُنا».
	//
	// **وهو واحدٌ للبيع والتوصيل**: خانتان تجعلان المجموعَ رقماً لا يُحسب
	// في الجيب — وهو ما التقريبُ كلُّه من أجله.
	{Key: "pricing.rounding", Group: GroupNew, Kind: KindInt,
		Min: 0, Max: 10000, Default: 500},

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
	// **الطلبُ ينزل إلى السائقين وحده.**
	//
	// كان الانتقال إلى الطابور بيد العمليات: موظّفٌ يضغط «طلب سائق» لكل طلب.
	// **فيتأخّر الطلبُ بقدر ما يتأخّر انتباهُه** — وهو ينظر إلى عشرين في الساعة.
	//
	// والسائقُ يرى في بطاقة الطابور «جاهز خلال ن دقيقة»، فيقرّر بنفسه أيأخذه
	// الآن أم يدع غيرَه. **فالقرارُ عند من يعرف موقعَه لا عند موظّفٍ يخمّن.**
	{Key: "orders.auto_dispatch", Group: GroupOrders, Kind: KindBool, Default: true},
	// **مهلةُ ظهور زرّ الإسناد اليدويّ.**
	//
	// **والإسنادُ اليدويّ احتياطٌ لا أصل**: زرٌّ متاحٌ دائماً يُستعمل دائماً،
	// **فيصير هو الطريقَ ويصير ترتيبُ السائقين زينة**. فلا يظهر إلّا حين
	// يعجز الطابور عن التقاط الطلب.
	//
	// ومستقلٌّ عن مهلة تنبيه «بلا سائق» عمداً: **قد يريد المالكُ أن يعلم قبل
	// أن يتدخّل.**
	{Key: "orders.manual_assign_after_min", Group: GroupOrders, Kind: KindInt,
		Min: 0, Max: 120, Unit: "minute", Default: 10},

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
	// **نظاما التوزيع.**
	//
	// `queue` الأسرعُ التقاطاً: سريعٌ في الذروة **ويُجوّع البطيء** — سائقٌ
	// بهاتفٍ قديم أو حيٍّ ضعيف الشبكة لا يصل قبل غيره أبداً.
	// `rotation` بالترتيب: يُعرض على واحدٍ في دوره، **عدلٌ وثمنُه ثوانٍ**.
	//
	// `direct` **إسنادٌ مباشر**: لا عرضَ ولا انتظارَ قبول — الطلبُ يصير مهمّةَ
	// صاحبِ الدور في اللحظة، **ويُنبَّه إليه تنبيهاً متكرّراً.**
	//
	// **ولماذا يلزم**: السائقُ على درّاجته لا أمام شاشته. **وعرضٌ ينتظر ضغطةً
	// ينقضي وقتُه ثمّ ينتقل** — فيدور الطلبُ على ثلاثةٍ ويضيع دقيقتان قبل أن
	// يتحرّك أحد. **والمهمّةُ التي وقعت تُقرأ حين يُنظر، والعرضُ الذي انقضى
	// لا يعود.**
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «لا ننتظر السائقَ يأخذ الطلب بل يصبح فوراً
	// بمهامّه مع تنبيه مهمّةٍ جديدةٍ متكرّرٍ لمدّة ٢٠ ثانية».)
	//
	// **والدورُ هو دورُ `rotation` نفسُه** — بأهليّته وترتيبه: من طال انتظارُه
	// أوّلاً، ولا يُسنَد لمن بلغ سقفَ نقده أو طلباته. **وقاعدةُ الاختيار مكتوبةٌ
	// مرّةً** (`nextEligibleDriver`)، فلا يفترق المُسنَدُ عن المعروض عليه.
	{Key: "drivers.assignment_mode", Group: GroupDrivers, Kind: KindChoice,
		Options: []string{"queue", "rotation", "direct"}, Default: "queue"},
	// **مدّةُ تنبيه المهمّة الجديدة** — يتكرّر حتى تنقضي أو يفتحها السائق.
	//
	// **ورنّةٌ واحدةٌ لا تكفي من يقود**: تمرّ في ضجيج الشارع، **وتنبيهٌ لا
	// يُسمع مهمّةٌ لا تبدأ.**
	{Key: "drivers.alert_seconds", Group: GroupDrivers, Kind: KindInt,
		Min: 0, Max: 120, Unit: "second", Default: 20},
	// **مهلةُ العرض**: قصيرةٌ تُتعب السائق وهو يقود، وطويلةٌ تُبرّد الطعام.
	{Key: "drivers.offer_timeout_sec", Group: GroupDrivers, Kind: KindInt,
		Min: 10, Max: 300, Unit: "second", Default: 45},

	// ── المتاجر ───────────────────────────────────────────────────────────
	{Key: "merchants.menu_requires_approval", Group: GroupMerchants, Kind: KindBool,
		Default: false},
	// **من يدير الطلبات: المتجر أم المنصة؟**
	//
	// ليس كلُّ متجرٍ يجلس إلى شاشة. مطعمٌ صغير في الرقة لا يملك جهازاً في
	// المطبخ ولا من يراقبه — يعمل على واتساب كما يعمل يومَه كلَّه. وإلزامُه
	// ببوابةٍ يفتحها يعني **طلباتٍ تتأخّر حتى يتذكّر أحدهم أن ينظر**.
	//
	// فحين تُطفأ: تقبل العملياتُ الطلب نيابةً عنه ثم تُرسله إليه على واتساب،
	// فيقرؤه في المكان الذي يعمل فيه أصلاً.
	{Key: "merchants.self_manage_orders", Group: GroupNew, Kind: KindBool,
		Default: true},
	// **حظرُ كثيرِ الإلغاء — والزرُّ ذكيٌّ لأن له وضعين لا حالتين.**
	//
	// «آليّ» يحظر بنفسه، و«يدويّ» يُنبّه العملياتِ وتقرّر هي. **وحظرٌ يقع
	// ليلاً بلا من يراه يُفقد المنصةَ متجراً ويُفقد المتجرَ رزقاً** — فالوضعُ
	// اختيارُ المالك لا حتمُ النظام.
	{Key: "merchants.cancel_ban_mode", Group: GroupMerchants, Kind: KindChoice,
		Options: []string{"manual", "auto"}, Default: "manual"},
	// **العتبةُ داخل نافذة لا مدى الحياة**: متجرٌ سلّم ثلاثمئة وألغى ستّاً في
	// سنة ليس سيّئاً، والعدُّ التراكميّ يحظره يوماً حتماً.
	{Key: "merchants.cancel_ban_count", Group: GroupMerchants, Kind: KindInt,
		Min: 1, Max: 100, Default: 5},
	{Key: "merchants.cancel_ban_days", Group: GroupMerchants, Kind: KindInt,
		Min: 1, Max: 365, Unit: "day", Default: 30},
	// **جديد**: كان ٢٠ دقيقة مكتوباً في الترحيل كافتراضي عمود. والمتجر الجديد
	// يرثه بلا أن يملك المالك تغييره لمن يأتي بعده.
	{Key: "merchants.default_prep_minutes", Group: GroupMerchants, Kind: KindInt,
		Min: 1, Max: 240, Unit: "minute", Default: 20},

	// ── المندوبون ─────────────────────────────────────────────────────────
	{Key: "sales.activation_orders", Group: GroupSales, Kind: KindInt,
		Min: 1, Max: 100, Unit: "order", Default: 5, Sensitive: true},
	{Key: "sales.monthly_target", Group: GroupSales, Kind: KindInt,
		Min: 1, Max: 100, Unit: "merchant", Default: 5},

	// ── المنصة ────────────────────────────────────────────────────────────
	// **حسابُ الخزينة** — الأدمن أو المدير المالي، يحدّده المالك.
	//
	// ولم يُختَر بالدور بل بالحساب: **الأدوارُ يحملها أكثرُ من واحد، والخزينةُ
	// واحدة.** وفارغٌ يعني «لم تُختَر»، فلا يُكتب قيدُ خزينةٍ **ولا يُعطَّل
	// تسليم**: طلبٌ يُرفض لأن المالك لم يفتح صفحةَ الإعدادات خسارةٌ لا تُحتمَل.
	{Key: "platform.treasury_user_id", Group: GroupPlatform, Kind: KindText,
		Max: 64, Default: "", Sensitive: true},

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

	// **سقفُ المصادر — والقيدُ الحقيقيُّ «قريب» لا «كم».**
	//
	// **ولا سقفَ للأصناف**: عائلةٌ تطلب شاورما وبطاطا وسلطة وعصيراً وحلواً من
	// **مطبخٍ واحد** — خمسةُ أصنافٍ ووقفةٌ واحدة، **فيُعاقَب الطلبُ الطبيعيّ
	// لمنع النادر.** والقيدُ على عدد المطابخ لا على ما في السلّة.
	{Key: "orders.max_sources", Group: GroupOrders, Kind: KindInt,
		Min: 1, Max: 5, Default: 2},
	// **مجّانيٌّ حين لا يكلّف، محسوبٌ حين يكلّف.**
	//
	// والعملاقان (DoorDash · Uber Eats) يبتلعان الرسمَ لأنهما يشتريان زبائنَ
	// بأموال مستثمرين — **ورحّال لا يستطيع ولا يجب أن يحاول.**
	{Key: "orders.extra_source_fee", Group: GroupOrders, Kind: KindMoney,
		Min: 0, Max: 100000, Default: 0},
	// **التكلفةُ في المسافة بين المطبخين لا في عددهما.** وألفٌ وخمسمئة متر
	// مشوارُ دقائق في مدينةٍ كالرقّة.
	{Key: "orders.source_proximity_m", Group: GroupOrders, Kind: KindInt,
		Min: 100, Max: 20000, Unit: "meter", Default: 1500},

	// ── الدعم ─────────────────────────────────────────────────────────────
	// **مهلةُ الشكوى — لأن الذاكرةَ تُنسى والدليلَ يذهب.**
	//
	// شكوى بعد شهرٍ لا تُحقَّق: السائقُ لا يذكر، والبضاعةُ ذهبت، **ولا يبقى
	// إلّا كلمةٌ ضدّ كلمة** — فتُقبل بلا بيّنة أو تُردّ بلا بيّنة، وكلاهما ظلمٌ
	// لأحدهما. **وأربعٌ وعشرون ساعةً تكفي من نسي أن يفتح الطلبَ ليلَتَه**
	// (قرار المالك ٢٠٢٦-٠٨-٠١).
	{Key: "support.complaint_window_hours", Group: GroupSupport, Kind: KindInt,
		Min: 1, Max: 720, Unit: "hour", Default: 24},

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
