// Package authz **معجمُ القدرات والقرارُ المركزيّ** — `ADG-1` · `AQ-1`.
//
// ══════════════════════════════════════════════════════════════════════
// **اسمُ الدور ليس صلاحيّة**
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **التخويلُ يسأل «أدورُك `admin`؟»** — **أربعةٌ وأربعون نداءً لـ
// `RequireRoles` وسبعةَ عشرَ فحصاً يدويّاً** في مسارات الخادم.
//
// **فمن أراد أن يمنح موظّفاً صلاحيّةً واحدةً منح دوراً كاملاً** —
// **ومن أراد أن يمنعه من واحدةٍ نزع الدورَ كلَّه.** **وذاك عكسُ «أقلِّ
// صلاحيّة».**
//
// # وعقدُ المالك (٢٠٢٦-٠٩-٠٧)
//
//	NO-CODE FOR OPERATIONS · CODE FOR NEW CAPABILITIES
//
//	معجمُ القدرات   **الشيفرةُ**   — مهندس
//	دورٌ ← قدرة      **القاعدةُ**   — الأدمن من اللوحة
//	حسابٌ ← دور      **القاعدةُ**   — الأدمن من اللوحة
//
// **وقدرةٌ جديدةٌ تحتاج مهندساً** — **وتبديلُ من يملكها لا يحتاجه.**
//
// # والافتراضُ منعٌ
//
// **دورٌ يُنشَأ اليومَ لا يملك شيئاً** · **ومجهولُ الدور يُمنَع** ·
// **ومجهولُ القدرة يُمنَع.** **ولا «كلُّ من دخل بابَ الإدارة يمرّ».**
package authz

import "sort"

// Capability معرّفُ قدرةٍ مستقرّ.
//
// **ونصٌّ لا رقم**: **يُقرأ في سجلٍّ وفي صفٍّ في القاعدة وفي رسالة
// خطأ** — **ورقمٌ يُبدَّل معناه بلا أن يُلاحَظ.**
type Capability string

// ══════════════════════════════════════════════════════════════════════
// **المعجمُ — مشتقٌّ من المسارات القائمة لا مخترَع**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا قدرةٌ لا يحرسها مسارٌ اليوم** — **ومعجمٌ فيه ما لا يُستعمَل
// يُقرأ عقداً وهو أمنية.**
const (
	// ── الطلبات ──────────────────────────────────────────────────
	OrdersRead      Capability = "orders.read"
	OrdersIntervene Capability = "orders.intervene"

	// ── الحسابات والموظّفون ──────────────────────────────────────
	UsersRead         Capability = "users.read"
	UsersStatusManage Capability = "users.status.manage"
	RolesManage       Capability = "roles.manage"

	// ── المال ───────────────────────────────────────────────────
	FinanceRead   Capability = "finance.read"
	FinanceManage Capability = "finance.manage"
	PayoutsDecide Capability = "payouts.decide"

	// ── المتاجر والسائقون ───────────────────────────────────────
	MerchantsManage Capability = "merchants.manage"
	DriversManage   Capability = "drivers.manage"
	// DriversRead **قراءةُ سجلّ السائقين ومواضعهم** — دون تشغيلهم.
	//
	// **ومراجعُ السائقين يقرأ ولا ينهي وردية** — **و`drivers.manage`
	// كانت تجمع الاثنين**، فمن وُظّف للتوثيق ملك إخراجَ سائقٍ من
	// عمله. (`ADG-2` — مصالحةُ دورةِ ٢٦.)
	DriversRead Capability = "drivers.read"

	// ── الإعدادات — ثلاثُ درجاتٍ بحسب الأثر ──────────────────────
	//
	// **وتصنيفُها من دورةِ ٢١ لا يُخترَع ثانيةً** — انظر
	// `server.criticalSettingKey`.
	SettingsGeneralManage   Capability = "settings.general.manage"
	SettingsFinancialManage Capability = "settings.financial.manage"
	SettingsSecurityManage  Capability = "settings.security.manage"

	// ── المحتوى والتقارير ───────────────────────────────────────
	ContentManage Capability = "content.manage"
	AnalyticsRead Capability = "analytics.read"

	// ══════════════════════════════════════════════════════════════
	// **وثلاثٌ أُضيفت في دورةِ ٢٥ — تفرضها مساراتٌ قائمة**
	// ══════════════════════════════════════════════════════════════
	//
	// **ولا واحدةَ أُضيفت لأنّ دوراً وُجد** — **بل لأنّ حدَّ
	// الصلاحيّة القائمَ كان أخشنَ ممّا يلزم.**

	// SupportManage **التذاكرُ والنزاعاتُ والطوارئ.**
	//
	// **وكان `POST /tickets/{id}/resolve` بحارس `admin,finance`** —
	// **فموظّفُ الدعم لا يُغلق تذكرةً، والماليّةُ تُغلقها.** **وذاك
	// عكسُ التخصّص.**
	SupportManage Capability = "support.manage"

	// SafetyManage **الإنذاراتُ والمخالفاتُ وتعليقُ المتاجر.**
	//
	// **وكانت داخلَ `merchants.manage`** — **ومعها تحريرُ القائمة
	// والساعات.** **فمن أراد أن يُعلّق متجراً مخالفاً نال تحريرَ
	// قوائمه**، **ومن أراد تحريرَ قائمةٍ نال تعليقَ المتاجر.**
	SafetyManage Capability = "safety.manage"

	// MerchantsVerify **مراجعةُ المرشَّحين والقوائم قبل النشر.**
	//
	// **وهي غيرُ `merchants.manage`**: **المراجعُ يوافق ويرفض ولا
	// يُنشئ متجراً ولا يُحرّر قوائمَ غيرِه.**
	MerchantsVerify Capability = "merchants.verify"

	// AuditRead **قراءةُ سجلّ التدقيق.**
	//
	// **وكانت بحارس `admin,finance`** — **وهي قراءةُ أمنٍ لا مال.**
	AuditRead Capability = "audit.read"

	// ══════════════════════════════════════════════════════════════
	// **وأربعٌ أُضيفت في دورةِ ٢٦ — «يقرأ» ليست صنفاً واحداً**
	// ══════════════════════════════════════════════════════════════
	//
	// **قراءةُ صفٍّ في عملٍ جارٍ غيرُ سحبِ الجدول كلِّه** —
	// **ومحادثةُ زبونٍ غيرُ رقم طلبه** — **وعنوانُ بيته غيرُ اسمه.**
	//
	// **وكانت الثلاثةُ في قدرتين** (`users.read` · `orders.read`)،
	// **فمن احتاج اسماً نال دفترَ العناوين ومحفوظَ المحادثات.**

	// UsersExport **سحبُ دليل الحسابات ملفّاً.**
	//
	// **والتصديرُ ليس قراءةً أكثر** — **هو إخراجُ القاعدة من
	// المنصّة**: **صفٌّ يُقرأ يبقى في الشاشة، وملفٌّ يُسحَب يمشي
	// في واتساب.**
	UsersExport Capability = "users.export"

	// FinanceExport **سحبُ الدفتر وكشفِ الطلبات ملفّاً.**
	//
	// **وكشفُ الطلبات فيه اسمُ الزبون وهاتفُه** مع أنصبة كلّ طلب.
	FinanceExport Capability = "finance.export"

	// OrdersCommunicationsRead **محادثاتُ الطلب ورسائلُه.**
	//
	// **وهي كلامُ الناس** — **ومن يسوّي حساباً لا يقرؤه**، ومن
	// يحقّق في شكوى يقرؤه. **وكانت في `orders.read` مع رقم الطلب.**
	OrdersCommunicationsRead Capability = "orders.communications.read"

	// UsersContactRead **رقمُ الاتّصال** — `XG-42`.
	//
	// **وهو حقلٌ في ردٍّ لا بابٌ في موجِّه**: **من ملك `users.read`
	// ليجد حساباً لقيدٍ ماليٍّ نال هاتفَه أيضاً** — **ولا يتّصل
	// بأحد.**
	//
	// **ويملكه من يتّصل**: العمليّاتُ والدعمُ والتحقيقُ وتوثيقُ
	// السائقين. **ولا تملكه الماليّةُ ولا التحليل** — **قيدٌ لا
	// مكالمة.**
	//
	// **ولا يُخلَط بـ`users.sensitive.read`**: تلك **دفترُ العناوين
	// والأثر**، وهذه **رقمٌ يُطلَب عليه.**
	UsersContactRead Capability = "users.contact.read"

	// UsersSensitiveRead **دفترُ عناوين المرء وأثرُه.**
	//
	// **وعنوانُ الطلب في الطلب** — **وهذا دفترُ بيوته كلِّها.**
	// **ومن يوزّع طلباً قائماً لا يحتاجه.**
	UsersSensitiveRead Capability = "users.sensitive.read"
)

// FieldPolicy **معجمُ الحقول المحميّة وقدرةُ كلٍّ** — `XG-42`.
//
// **وقدرةٌ تحرس حقلاً لا باباً**: **`ADG-2` يحكم من يبلغ المسار،
// وهذه تحكم ما يصل من بلغه.** **فموضعٌ واحدٌ يقرؤه المنتَجُ والحارس** —
// **ولا سياسةٌ تُكتب مرّتين فتفترقا.**
//
// **ولا يُدرَج حقلٌ لاسمه**: **`address_text` في الطلب عنوانُ الطلب
// وهو من عمل من يوزّعه** — **ودفترُ عناوين المرء بابٌ آخرُ يحرسه
// `UsersSensitiveRead`.** **و`invite_code` رمزٌ يوزّعه المندوبُ بنفسه
// ليُدعى به** — **فليس سرّاً.**
var FieldPolicy = map[string]Capability{
	"phone":          UsersContactRead,
	"customer_phone": UsersContactRead,
	"driver_phone":   UsersContactRead,
}

// catalog **المعجمُ المُعرَّفُ في الشيفرة** — ووصفٌ لكلٍّ يُقرأ في اللوحة.
var catalog = map[Capability]string{
	OrdersRead:               "قراءةُ الطلبات ولوحةِ العمليّات",
	OrdersIntervene:          "تدخّلٌ في طلبٍ نيابةً عن طرفه",
	UsersRead:                "قراءةُ الحسابات",
	UsersStatusManage:        "إيقافُ حسابٍ أو حظرُه أو تبديلُ بياناته",
	RolesManage:              "منحُ الأدوار وسحبُها",
	FinanceRead:              "قراءةُ المال والتقارير الماليّة",
	FinanceManage:            "قيدُ محفظةٍ ومصروفٌ وخزينة",
	PayoutsDecide:            "قرارُ السحب",
	MerchantsManage:          "إدارةُ المتاجر وتعليقُها",
	DriversManage:            "إدارةُ السائقين وتشغيلُهم",
	DriversRead:              "قراءةُ سجلّ السائقين ومواضعهم",
	SettingsGeneralManage:    "إعداداتٌ عامّةٌ ومحتوى",
	SettingsFinancialManage:  "إعداداتٌ تدخل حساباً ماليّاً",
	SettingsSecurityManage:   "إعداداتُ الأمن والجلسات",
	ContentManage:            "لافتاتٌ وعروضٌ ومحتوى",
	AnalyticsRead:            "قراءةُ التحليلات",
	SupportManage:            "التذاكرُ والنزاعاتُ والطوارئ",
	SafetyManage:             "الإنذاراتُ والمخالفاتُ وتعليقُ المتاجر",
	MerchantsVerify:          "مراجعةُ المرشَّحين والقوائم",
	AuditRead:                "قراءةُ سجلّ التدقيق",
	UsersExport:              "سحبُ دليل الحسابات ملفّاً",
	FinanceExport:            "سحبُ الدفتر وكشفِ الطلبات ملفّاً",
	OrdersCommunicationsRead: "قراءةُ محادثات الطلب ورسائله",
	UsersContactRead:         "قراءةُ رقم الاتّصال",
	UsersSensitiveRead:       "قراءةُ عناوين المرء وأثرِه",
}

// Known **أهذه قدرةٌ مسجَّلة؟** — **ومجهولُها يُمنَع.**
func Known(c Capability) bool { _, ok := catalog[c]; return ok }

// Describe وصفُ القدرة — فارغٌ للمجهولة.
func Describe(c Capability) string { return catalog[c] }

// All المعجمُ مرتَّباً — تقرؤه اللوحةُ والحرّاسُ والهجرات.
func All() []Capability {
	out := make([]Capability, 0, len(catalog))
	for c := range catalog {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Count عددُ القدرات — يحرسه فحصٌ فلا تختفي واحدةٌ صامتةً.
func Count() int { return len(catalog) }
