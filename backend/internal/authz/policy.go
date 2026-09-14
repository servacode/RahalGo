package authz

import "strings"

// ══════════════════════════════════════════════════════════════════════
// **سياسةٌ واحدةٌ لكلّ مسارٍ إداريّ** — `ADG-2` · `AQ-1`
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا جدولٌ لا حارسٌ عند كلّ مسار
//
// **مئةٌ وستّةَ عشرَ مساراً في السطح الإداريّ.** **وحارسٌ يُكتب عند
// كلٍّ منها يُنسى عند واحد** — **ومسارٌ نُسي حارسُه لا يُكتشَف إلّا
// حين يُستعمَل.**
//
// **والجدولُ يُقرأ كلَّه في مكانٍ واحد**، **ومشّاءٌ على الموجِّه يُثبت
// أنّ كلَّ مسارٍ مسجَّلٍ له سياسةٌ أو استثناءٌ مكتوب.**
//
// # والافتراضُ منع
//
// **مسارٌ لا سياسةَ له ولا استثناءَ ⇒ يُمنَع** — **ولا يُقرأ الصمتُ
// إذناً.**

// Rule سياسةُ مسارٍ واحد.
type Rule struct {
	// Method فعلُ HTTP — وفارغٌ يعني كلَّها.
	Method string
	// Pattern نمطُ المسار بعد `/api/v1/admin` — و`{}` معلَمةٌ متغيّرة.
	Pattern string
	// Need القدرةُ المطلوبة.
	Need Capability
}

// adminPolicy **الجدولُ الكانونيّ** — مرتَّبٌ من الأخصّ إلى الأعمّ.
//
// **ولا خانةَ مظنونة**: كلُّ سطرٍ مقروءٌ من مسارٍ قائمٍ في الموجِّه.
var adminPolicy = []Rule{
	// ── الأدوارُ والقدرات — أشدُّ ما في المنصّة ──────────────────
	{"", "/roles", RolesManage},
	{"", "/roles/{code}", RolesManage},
	{"", "/roles/{code}/capabilities", RolesManage},
	{"", "/roles/{code}/capabilities/{cap}", RolesManage},
	{"", "/capabilities", RolesManage},
	{"POST", "/users/{id}/roles", RolesManage},
	{"DELETE", "/users/{id}/roles/{role}", RolesManage},

	// ── صحّةُ المنصّة الداخليّة ───────────────────────────────────
	//
	// **وبابٌ واحدٌ يُقرأ ولا يُكتب** (دورةُ ٧٠أ).
	{"GET", "/ops/health", ObservabilityRead},

	// ── حساباتُ الموظّفين والمستخدمين ────────────────────────────
	{"POST", "/users", UsersStatusManage},
	{"PATCH", "/users/{id}", UsersStatusManage},
	{"POST", "/users/{id}/password", UsersStatusManage},
	{"POST", "/users/{id}/logout-all", UsersStatusManage},
	{"POST", "/users/{id}/warnings", SafetyManage},
	{"GET", "/users/{id}/warnings", SafetyManage},
	{"GET", "/users/{id}/warn-reasons", SafetyManage},
	{"GET", "/users/{id}/financials", FinanceRead},
	{"GET", "/users/{id}/wallet", FinanceRead},
	{"POST", "/users/{id}/wallet", FinanceManage},
	{"POST", "/users/{id}/incentive", FinanceManage},
	{"GET", "/users/{id}/incentives", FinanceRead},
	{"GET", "/users", UsersRead},
	{"GET", "/users/stats", UsersRead},
	// **والتصديرُ إخراجُ القاعدة لا قراءةٌ أكثر.**
	{"GET", "/users/export", UsersExport},
	{"GET", "/users/{id}", UsersRead},
	{"GET", "/users/{id}/activity", UsersSensitiveRead},
	{"GET", "/users/{id}/feedback", UsersRead},
	// **ودفترُ بيوتِ المرء غيرُ عنوانِ طلبه.**
	{"GET", "/users/{id}/addresses", UsersSensitiveRead},
	{"GET", "/users/{id}/chats", OrdersCommunicationsRead},
	{"GET", "/customers", UsersRead},
	{"GET", "/salesreps", UsersRead},

	// ── الطلبات ─────────────────────────────────────────────────
	{"POST", "/orders/{id}/transition", OrdersIntervene},
	{"POST", "/orders/{id}/assign", OrdersIntervene},
	{"POST", "/orders/{id}/transfer", OrdersIntervene},
	{"POST", "/orders/{id}/recompute", OrdersIntervene},
	{"POST", "/orders/{id}/compensate-driver", FinanceManage},
	{"POST", "/orders/{id}/settle-goods", FinanceManage},
	{"POST", "/orders/{id}/goods", OrdersIntervene},
	{"POST", "/orders/{id}/whatsapp", OrdersIntervene},
	{"GET", "/orders/{id}/breakdown", FinanceRead},
	// **وتصديرُ الطلبات كشفُ محاسبةٍ لا تقرير** — فيه **اسمُ الزبون
	// وهاتفُه** وأنصبةُ كلّ طلبٍ من الدفتر. **وحارسُه قبلَ `ADG-2`
	// كان `admin,finance`** — **ودورةُ ٢٥ وسّعته إلى التحليلات
	// فأدخلت التحليلاتِ على أرقام الهواتف.** (مصالحةُ دورةِ ٢٦.)
	{"GET", "/orders/export", FinanceExport},
	{"GET", "/orders", OrdersRead},
	{"GET", "/orders/alerts", OrdersRead},
	{"GET", "/orders/{id}", OrdersRead},
	// **وكلامُ الناس صنفٌ بذاته** — ومن يسوّي حساباً لا يقرؤه.
	{"GET", "/orders/{id}/chat", OrdersCommunicationsRead},
	{"GET", "/orders/{id}/message", OrdersCommunicationsRead},

	// ── المتاجر: إدارةٌ · وسلامةٌ · وتوثيق ───────────────────────
	{"POST", "/merchants/{id}/suspend", SafetyManage},
	{"POST", "/merchants/{id}/warnings", SafetyManage},
	{"POST", "/merchants/{id}/clear-violations", SafetyManage},
	{"GET", "/merchants/{id}/violations", SafetyManage},
	{"GET", "/merchants/{id}/warnings", SafetyManage},
	{"POST", "/leads/{id}/status", MerchantsVerify},
	{"POST", "/menu/items/{itemID}/review", MerchantsVerify},
	{"GET", "/menu/pending", MerchantsVerify},
	{"GET", "/leads", MerchantsVerify},
	{"POST", "/merchants", MerchantsManage},
	{"PATCH", "/merchants/{id}", MerchantsManage},
	{"POST", "/merchants/{id}/menu/sections", MerchantsManage},
	{"POST", "/merchants/{id}/menu/items", MerchantsManage},
	{"PATCH", "/menu/sections/{sectionID}", MerchantsManage},
	{"DELETE", "/menu/sections/{sectionID}", MerchantsManage},
	{"PATCH", "/menu/items/{itemID}", MerchantsManage},
	{"DELETE", "/menu/items/{itemID}", MerchantsManage},
	{"PUT", "/merchants/{id}/hours", MerchantsManage},
	// ══════════════════════════════════════════════════════════════
	// **وقراءةُ سجلّ المتاجر ليست إدارتَها** (٢٠٢٦-٠٩-١٣)
	// ══════════════════════════════════════════════════════════════
	//
	// **وكانت الأربعُ بـ`merchants.manage`** — **فشاشةُ الطلبات
	// تنادي `GET /merchants` لتبني قائمةَ التحويل**، **فمن أراد
	// سطراً يقرؤه نال إنشاءَ المتاجر وتحريرَ قوائم غيره وحذفَ
	// أصنافها.**
	//
	// **والكتابةُ فوقها تبقى كما هي** — ومن ملك `manage` مُنح
	// `read` في الهجرة، **فلا أحدَ فقد ما كان يراه.**
	{"GET", "/merchants", MerchantsRead},
	{"GET", "/merchants/{id}", MerchantsRead},
	{"GET", "/merchants/{id}/menu", MerchantsRead},
	{"GET", "/merchants/{id}/hours", MerchantsRead},

	// ── السائقون ────────────────────────────────────────────────
	{"POST", "/drivers/{id}/settle", FinanceManage},
	{"GET", "/drivers/{id}/cash", FinanceRead},
	{"GET", "/cash/outstanding", FinanceRead},
	{"POST", "/drivers/{id}/end-shift", DriversManage},
	{"GET", "/drivers", DriversRead},

	// ── المال ───────────────────────────────────────────────────
	{"POST", "/payouts/{id}/decide", PayoutsDecide},
	{"GET", "/payouts", FinanceRead},
	{"GET", "/profits", FinanceRead},
	{"GET", "/expenses", FinanceRead},
	{"GET", "/expenses/categories", FinanceRead},
	{"POST", "/expenses", FinanceManage},
	{"POST", "/expenses/categories", FinanceManage},
	{"POST", "/expenses/{id}/void", FinanceManage},
	{"GET", "/ledger/export", FinanceExport},
	{"GET", "/treasury-candidates", FinanceManage},
	{"GET", "/reports/losses", FinanceRead},

	// ── الدعمُ والنزاعاتُ والطوارئ ───────────────────────────────
	{"GET", "/tickets", SupportManage},
	{"POST", "/tickets", SupportManage},
	{"GET", "/tickets/{id}", SupportManage},
	{"POST", "/tickets/{id}/replies", SupportManage},
	{"POST", "/tickets/{id}/resolve", SupportManage},
	{"GET", "/emergencies", SupportManage},
	{"POST", "/emergencies/{id}/resolve", SupportManage},
	{"GET", "/disputes", SupportManage},
	{"POST", "/disputes", SupportManage},
	{"POST", "/disputes/{id}/settle", FinanceManage},
	{"GET", "/ratings", SupportManage},

	// ── المحتوى والتسويق ────────────────────────────────────────
	{"", "/promos", ContentManage},
	{"", "/promos/{id}", ContentManage},
	{"", "/banners", ContentManage},
	{"", "/banners/{id}", ContentManage},
	{"", "/offers", ContentManage},
	{"", "/offers/{id}/active", ContentManage},
	{"", "/sections", ContentManage},
	{"", "/sections/{id}", ContentManage},
	{"GET", "/sections/{id}/items", ContentManage},
	{"", "/categories", ContentManage},
	{"", "/categories/{id}", ContentManage},
	{"POST", "/media", ContentManage},
	{"GET", "/media/sign", ContentManage},
	{"POST", "/broadcast", ContentManage},
	{"GET", "/broadcast/count", ContentManage},
	{"", "/app-file", ContentManage},

	// ── الجغرافيا ───────────────────────────────────────────────
	// **ودوامُ المنصّة وإيقافُها المؤقّت من باب المناطق نفسِه** —
	// **تهيئةُ تشغيلٍ عامّةٌ لا مالٌ ولا أمن.**
	{"", "/platform/hours", SettingsGeneralManage},
	{"", "/platform/closure", SettingsGeneralManage},
	{"", "/zones", SettingsGeneralManage},
	{"", "/zones/{id}", SettingsGeneralManage},
	// **وأوقاتُ المنطقة من بابها نفسِه** — `ZH`.
	{"", "/zones/{id}/hours", SettingsGeneralManage},
	{"", "/cities", SettingsGeneralManage},
	{"", "/cities/{id}", SettingsGeneralManage},
	{"", "/governorates", SettingsGeneralManage},
	{"", "/governorates/{id}", SettingsGeneralManage},
	{"", "/districts", SettingsGeneralManage},
	{"", "/districts/{id}", SettingsGeneralManage},

	// ── خريطةُ العمليّات — موجِّهٌ فرعيّ ─────────────────────────
	//
	// **وهي خارجُ نطاق دورةِ ٢٥ عملاً** — **ولا يُعفى مسارُها من
	// التصنيف**: **مسارٌ بلا سياسةٍ يُمنَع، وتصنيفُه ليس بناءَه.**
	//
	// **والتغطيةُ والفروعُ والمناطقُ رسمُ عملٍ جغرافيّ** — قدرتُها
	// `settings.general.manage`. **وقراءاتُها تشغيليّة.**
	// ══════════════════════════════════════════════════════════════
	// **خريطةُ العمليّات — تُقرأ لمن يفتحها وتُكتب لمن يملكها**
	// ══════════════════════════════════════════════════════════════
	//
	// **ودورةُ ٢٥ سوّت القراءةَ بالكتابة** فيها — **فصار موظّفُ
	// العمليّات محجوباً عن قراءة الفروع والتغطية**، **وهي شاشتُه.**
	// (مصالحةُ دورةِ ٢٦.)
	//
	// **ورسمُ مضلَّعٍ يبدّل من تصله المنصّةُ أصلاً** — فيبقى بقدرة
	// الإعدادات العامّة.
	{"GET", "/ops-map/coverage", OrdersRead},
	{"", "/ops-map/coverage", SettingsGeneralManage},
	{"", "/ops-map/coverage/{id}", SettingsGeneralManage},
	{"", "/ops-map/coverage/{id}/active", SettingsGeneralManage},
	// **وكثافةُ الطلب بالمكان الإداريّ قراءةٌ تحليليّةٌ كأختها** — `CR`.
	// **ولا قدرةَ جديدة.**
	{"GET", "/ops-map/coverage-demand/places", AnalyticsRead},
	{"GET", "/ops-map/coverage-requests", AnalyticsRead},
	{"", "/ops-map/coverage-requests/{id}", SettingsGeneralManage},
	{"GET", "/ops-map/branches", OrdersRead},
	{"", "/ops-map/branches", SettingsGeneralManage},
	{"", "/ops-map/branches/{id}", SettingsGeneralManage},
	{"GET", "/ops-map/areas", OrdersRead},
	{"", "/ops-map/areas", SettingsGeneralManage},
	{"", "/ops-map/areas/{id}", SettingsGeneralManage},
	{"GET", "/ops-map/orders", OrdersRead},
	{"GET", "/ops-map/drivers", DriversRead},
	// **ودبّوسُ المتجر سياقُ تشغيلٍ لا إدارةَ متجر** — **والماليّةُ
	// تقرأ الخريطةَ ولا تُحرّر قائمة.**
	{"GET", "/ops-map/merchants", OrdersRead},
	{"GET", "/ops-map/demand", AnalyticsRead},
	{"GET", "/ops-map/opportunities", AnalyticsRead},
	// **ونشاطُ المندوبين لمن يبني شبكةَ المتاجر** — **ولا يُقرأ
	// بـ`users.read`**: **الماليّةُ تملكها ولا تُراقب مندوباً.**
	{"GET", "/ops-map/reps", MerchantsManage},
	{"GET", "/ops-map/meta", OrdersRead},
	{"GET", "/ops-map/search", OrdersRead},

	// ── الحوافز ─────────────────────────────────────────────────
	{"GET", "/incentives/{role}", FinanceRead},

	// ── الإعدادات والتقارير والتشخيص ─────────────────────────────
	//
	// **و`PUT /settings/{key}` قدرتُه تتبع المفتاحَ لا المسار** —
	// **يُحسَم في المعالِج** (`settingCapability`). **وهو مستثنىً
	// هنا عمداً ومكتوب.**
	// **وقراءةُ اللوح ليست تبديلَ مفتاح** (٢٠٢٦-٠٩-١٣): **شاشةُ
	// الطلبات تقرأ مفتاحين تشغيليّين** — **وكانت تطلب لأجلهما رسمَ
	// المناطق والمدن والمحافظات.**
	{"GET", "/settings", SettingsRead},
	{"GET", "/audit", AuditRead},
	{"GET", "/stats", AnalyticsRead},
	{"GET", "/reports", AnalyticsRead},
	{"GET", "/whatsapp", SettingsSecurityManage},
	{"POST", "/whatsapp/pair", SettingsSecurityManage},
	{"POST", "/whatsapp/unpair", SettingsSecurityManage},
}

// Exempt **مساراتٌ لا تُحكَم بالجدول — ولكلٍّ سببٌ مكتوب.**
var Exempt = map[string]string{
	"/settings/{key}": "**القدرةُ تتبع المفتاحَ لا المسار** — عامٌّ أو " +
		"ماليٌّ أو أمنيّ. **وتُحسَم في المعالِج** (`settingCapability`).",
	// **وبابُ التأكيد ليس فعلاً بذاته** — **قدرتُه قدرةُ الفعل الذي
	// يُؤكَّد**، وتُقاس في المعالِج من الجدول نفسِه. (`ADG-3`.)
	//
	// **ولا يُصدَر إثباتٌ لفعلٍ لا يملكه صاحبُه** — فلا يصير البابُ
	// آلةَ تخمينِ كلماتٍ على مسارٍ محجوب.
	"/step-up": "**قدرتُه قدرةُ الفعل المُؤكَّد** — تُقاس في المعالِج " +
		"من `LookupAdmin` نفسِها. (`ADG-3`.)",
}

// LookupAdmin **قدرةُ مسارٍ إداريٍّ** — و`ok=false` تعني «لا سياسةَ له».
//
// **والأخصُّ يغلب**: **مقاطعُ ثابتةٌ تسبق المعلَمات**، **وفعلٌ مسمّىً
// يسبق الفارغ.**
func LookupAdmin(method, pattern string) (Capability, bool) {
	best, bestScore, found := Capability(""), -1, false
	for _, r := range adminPolicy {
		if r.Method != "" && r.Method != method {
			continue
		}
		if !patternsEqual(r.Pattern, pattern) {
			continue
		}
		score := literalSegments(r.Pattern) * 2
		if r.Method != "" {
			score++
		}
		if score > bestScore {
			best, bestScore, found = r.Need, score, true
		}
	}
	return best, found
}

// IsExempt أهذا المسارُ مستثنىً بسببٍ مكتوب؟
//
// **ويُطابَق كالأنماط لا كنصّ** — **فـ`{key}` و`{}` سواءٌ في موضعهما**،
// **ومطابقةُ نصٍّ حرفيّةٌ تجعل الاستثناءَ لا يُصاب أبداً.**
func IsExempt(pattern string) (string, bool) {
	for p, why := range Exempt {
		if patternsEqual(p, pattern) {
			return why, true
		}
	}
	return "", false
}

// patternsEqual **مطابقةٌ باتّجاه**: نمطُ السياسة يُطابق النمطَ الواقع.
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا لا تُقارَن المعلَماتُ تناظراً**
// ══════════════════════════════════════════════════════════════════════
//
// **`DELETE /users/{id}/roles/{role}`** — **و`{role}` قيمتُه `analytics`**،
// **وهي كلمةٌ لاتينيّةٌ لا تُشبه معرّفاً** فتبقى حرفيّةً في النمط
// الواقع. **فتناظرٌ يقول «لا تطابق» ويُمنَع مسارٌ مصنَّف.**
//
// **فمعلَمةُ السياسة تُطابق أيَّ مقطع** — **والحرفيّةُ لا تُطابق إلّا
// مثلَها.** **والأخصُّ يغلب بالنقاط**، فلا تبتلع `/users/{id}` مسارَ
// `/users/stats`.
func patternsEqual(policy, actual string) bool {
	ps := strings.Split(strings.Trim(policy, "/"), "/")
	as := strings.Split(strings.Trim(actual, "/"), "/")
	if len(ps) != len(as) {
		return false
	}
	for i := range ps {
		if isParam(ps[i]) {
			continue // معلَمةٌ تُطابق أيَّ شيء
		}
		if isParam(as[i]) || ps[i] != as[i] {
			return false
		}
	}
	return true
}

func isParam(s string) bool {
	return strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
}

func literalSegments(p string) int {
	n := 0
	for _, s := range strings.Split(strings.Trim(p, "/"), "/") {
		if !isParam(s) {
			n++
		}
	}
	return n
}

// PolicyCount عددُ أسطر الجدول — يحرسه فحصٌ فلا ينكمش.
func PolicyCount() int { return len(adminPolicy) }

// Rules الجدولُ كما هو — **ليُقابَل بالموجِّه في فحصٍ دائم.**
//
// **وصفٌّ لا مسارَ له لا يُمنَح شيئاً** — **لكنّه يُقرأ عقداً
// وهو وهم.** **وخمسةٌ منه وُجدت في دورةِ ٢٦** (`/demand` ·
// `/meta` · `/opportunities` · `/reps` · `/search`) **كتبتُها
// من ذاكرةِ مسارٍ لا من الموجِّه.**
func Rules() []Rule { return adminPolicy }
