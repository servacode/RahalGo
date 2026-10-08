package settings

// ══════════════════════════════════════════════════════════════════════
// **أقسامُ صفحة الإعدادات بالموضوع** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (الإعدادات، البند ١٧)
// ══════════════════════════════════════════════════════════════════════
//
// **كانت ستّةَ عشرَ تبويباً بالدور**: أجرةُ التوصيل التي يدفعها الزبونُ تحت
// «السائقون»، والهامشُ تحت «المنصّة»، ونافذةُ الإلغاء تحت «الزبائن» — **فمن
// أراد «المال» فتّش أربعةَ تبويبات.** وصار عموداً جانبيّاً بالموضوع:
//
//	المال والعمولات · الطلبات والتوصيل · السائقون · المتاجر · المندوبون ·
//	الزبائن · الدعم والسحوبات · الأمان والدخول · الرسائل · الإطلاق والتشغيل ·
//	التطبيقات والتنزيل · الموقع والمحتوى والتواصل · (التغطية · واتساب والإعلان)
//
// # ولماذا قواعدُ بالبادئة لا قائمةُ مفاتيح
//
// **المفاتيحُ تُضاف كلَّ يوم** — وأقسامٌ أخرى تكتب مفاتيحَها الآن (المصاريف،
// التسويات، السحوبات…). **وقائمةٌ تُكتب مفتاحاً مفتاحاً تُنسى في واحد**،
// فيقع مفتاحٌ جديدٌ في غير بيته أو لا يظهر. **فالبادئةُ تُسكنه بنفسه**:
// `expenses.*` في المال، `support.*` في الدعم — **والاستثناءُ وحدَه يُكتب باسمه.**
//
// **والمجموعةُ القديمةُ (`Group`) تبقى** — يقرؤها ما سواها، **وهي الملجأُ
// الأخيرُ هنا**: مفتاحٌ لا تطابقه قاعدةٌ يسكن موضوعَ مجموعته.

import "strings"

// Topic موضوعٌ في العمود الجانبيّ.
type Topic string

const (
	TopicMoney     Topic = "money"
	TopicOrders    Topic = "orders"
	TopicDrivers   Topic = "drivers"
	TopicMerchants Topic = "merchants"
	TopicReps      Topic = "reps"
	TopicCustomers Topic = "customers"
	TopicSupport   Topic = "support"
	TopicSecurity  Topic = "security"
	TopicMessages  Topic = "messages"
	TopicLaunch    Topic = "launch"
	TopicApps      Topic = "apps"
	TopicSite      Topic = "site"
)

// Topics **ترتيبُ العمود الجانبيّ** — مصدرُه الواحد.
//
// **والتغطيةُ وواتساب والإعلان ألواحٌ لا مفاتيح** — تُضاف في الشاشة بعد هذه.
var Topics = []Topic{
	TopicMoney, TopicOrders, TopicDrivers, TopicMerchants, TopicReps, TopicCustomers,
	TopicSupport, TopicSecurity, TopicMessages, TopicLaunch, TopicApps, TopicSite,
}

// Placement **أين يسكن المفتاحُ في الصفحة.**
type Placement struct {
	Topic Topic `json:"topic"`
	// Section **صندوقٌ داخل الموضوع** — اسمُه في المعجم `admin.settings.topicSections.*`.
	//
	// **واسمُه غيرُ `Section`** — يُضمَّن بجانب `Def` في الردّ، **وحقلان باسمٍ
	// واحدٍ في عمقٍ واحدٍ يُسقطهما ترميزُ JSON كليهما بصمت.**
	TopicSection string `json:"topic_section,omitempty"`
	// Panel **لا بطاقةَ له — يُضبط في لوحٍ مخصَّص** (البند ١٢: «كلُّ إعدادٍ في
	// مكانٍ واحد»). **ويبقى في البحث يقود إلى لوحه.**
	//
	//	appStatus  أبوابُ الإطلاق ونصُّه — حالاتٌ جاهزةٌ بتأكيد
	//	hours      سريانُ الدوام — مع الجدول والإيقاف
	//	release    أدنى نسخةٍ للتطبيق — بخطوةِ تحقّق
	Panel string `json:"panel,omitempty"`
	// Hidden **مخفيٌّ عن الصفحة** — إعداداتُ الموقع العامّ بعد أن صار الويبُ
	// للموظّفين (البند ١٤). **يبقى في الفهرس ويعمل بقيمته**، ولا يُعرض.
	Hidden bool `json:"hidden,omitempty"`
}

type placeRule struct {
	prefix  string // بادئةٌ أو مفتاحٌ كامل
	exact   bool
	topic   Topic
	section string
	panel   string
	hidden  bool
}

// placeRules **الأدقُّ أوّلاً** — أوّلُ مطابقةٍ تحسم.
var placeRules = []placeRule{
	// ── المال والعمولات: ربحُ المنصّة ← المندوب ← السائق ← الزبون ← الصندوق ──
	{prefix: "pricing.", topic: TopicMoney, section: "money.profit"},
	{prefix: "merchants.commission_percent", exact: true, topic: TopicMoney, section: "money.profit"},
	{prefix: "delivery.platform_percent", exact: true, topic: TopicMoney, section: "money.profit"},
	{prefix: "sales.commission_percent", exact: true, topic: TopicMoney, section: "money.rep"},
	{prefix: "delivery.fee", exact: true, topic: TopicMoney, section: "money.driver"},
	{prefix: "delivery.merchant_fee", exact: true, topic: TopicMoney, section: "money.driver"},
	{prefix: "delivery.merchant_debt_open", exact: true, topic: TopicMoney, section: "money.driver"},
	{prefix: "delivery.by_distance", exact: true, topic: TopicMoney, section: "money.driver"},
	{prefix: "delivery.per_km", exact: true, topic: TopicMoney, section: "money.driver"},
	{prefix: "delivery.max_fee", exact: true, topic: TopicMoney, section: "money.driver"},
	{prefix: "drivers.failed_compensation_percent", exact: true, topic: TopicMoney, section: "money.driver"},
	{prefix: "drivers.cash_limit", exact: true, topic: TopicMoney, section: "money.driver"},
	{prefix: "customers.signup_bonus", exact: true, topic: TopicMoney, section: "money.customer"},
	{prefix: "customers.cod_limit", exact: true, topic: TopicMoney, section: "money.customer"},
	{prefix: "referral.", topic: TopicMoney, section: "money.customer"},
	// **ومفاتيحُ أقسام المال الأخرى تسكن هنا بأنفسها** — المصاريف والصندوق
	// والتسويات والالتزامات والخسائر والحوافز.
	{prefix: "finance.", topic: TopicMoney, section: "money.treasury"},
	{prefix: "treasury.", topic: TopicMoney, section: "money.treasury"},
	{prefix: "cash.", topic: TopicMoney, section: "money.treasury"},
	{prefix: "cashbox.", topic: TopicMoney, section: "money.treasury"},
	{prefix: "expenses.", topic: TopicMoney, section: "money.treasury"},
	{prefix: "obligations.", topic: TopicMoney, section: "money.treasury"},
	{prefix: "losses.", topic: TopicMoney, section: "money.treasury"},
	{prefix: "profits.", topic: TopicMoney, section: "money.treasury"},
	{prefix: "compensation.", topic: TopicMoney, section: "money.treasury"},
	{prefix: "compensations.", topic: TopicMoney, section: "money.treasury"},
	{prefix: "incentives.", topic: TopicMoney, section: "money.treasury"},
	{prefix: "promos.", topic: TopicMoney, section: "money.treasury"},

	// ── الدعم والسحوبات ──
	{prefix: "support.", topic: TopicSupport, section: "support.complaints"},
	{prefix: "complaints.", topic: TopicSupport, section: "support.complaints"},
	{prefix: "payouts.", topic: TopicSupport, section: "support.payouts"},
	{prefix: "payout.", topic: TopicSupport, section: "support.payouts"},

	// ── الرسائل ──
	{prefix: "whatsapp.", topic: TopicMessages, section: "messages.templates"},
	{prefix: "auth.otp_channel", exact: true, topic: TopicMessages, section: "messages.templates"},
	{prefix: "auth.sms_template", exact: true, topic: TopicMessages, section: "messages.templates"},
	{prefix: "accounts.welcome_template", exact: true, topic: TopicMessages, section: "messages.templates"},
	{prefix: "accounts.merchant_tutorial_url", exact: true, topic: TopicMessages, section: "messages.templates"},
	{prefix: "customers.tutorial_url", exact: true, topic: TopicMessages, section: "messages.templates"},
	{prefix: "customers.welcome_template", exact: true, topic: TopicMessages, section: "messages.templates"},
	{prefix: "customers.welcome_whatsapp", exact: true, topic: TopicMessages, section: "messages.templates"},
	{prefix: "notify.", topic: TopicMessages, section: "messages.notifications"},
	{prefix: "meals.", topic: TopicMessages, section: "messages.meals"},
	{prefix: "app_text.", topic: TopicMessages, section: "messages.app"},

	// ── الأمان والدخول ──
	{prefix: "security.", topic: TopicSecurity, section: "security.login"},
	{prefix: "auth.otp_login", exact: true, topic: TopicSecurity, section: "security.login"},
	{prefix: "auth.signup_verify", exact: true, topic: TopicSecurity, section: "security.login"},
	{prefix: "auth.require_whatsapp", exact: true, topic: TopicSecurity, section: "security.login"},
	{prefix: "moderation.", topic: TopicSecurity, section: "security.safety"},
	{prefix: "safety.", topic: TopicSecurity, section: "security.safety"},
	{prefix: "media.", topic: TopicSecurity, section: "security.safety"},

	// ── الإطلاق والتشغيل — الأبوابُ بطاقاتٌ بتأكيد، ونصُّها وسريانُ الدوام في ألواحهما ──
	{prefix: "launch.notice", exact: true, topic: TopicLaunch, panel: "appStatus"},
	{prefix: "launch.", topic: TopicLaunch, section: "launch.gates"},
	{prefix: "hours.platform_enforced", exact: true, topic: TopicLaunch, panel: "hours"},
	{prefix: "hours.", topic: TopicLaunch, section: "launch.ops"},
	{prefix: "platform.orders_mode", exact: true, topic: TopicLaunch, section: "launch.ops"},
	{prefix: "ops.", topic: TopicLaunch, section: "launch.ops"},

	// ── التطبيقات والتنزيل — وأدنى نسخةٍ في لوح «الإصدار» وحدَه ──
	{prefix: "app.min_version.", topic: TopicApps, panel: "release"},
	{prefix: "release.", topic: TopicApps, section: "apps.files"},
	{prefix: "app.", topic: TopicApps, section: "apps.files"},

	// ── الموقع والمحتوى والتواصل — الهويّةُ والتواصلُ والصفحاتُ القانونيّة وحدَها ──
	{prefix: "platform.location", exact: true, topic: TopicSite, section: "site.office"},
	{prefix: "platform.name", exact: true, topic: TopicSite, section: "site.identity"},
	{prefix: "platform.logo", exact: true, topic: TopicSite, section: "site.identity"},
	{prefix: "platform.background", topic: TopicSite, hidden: true},
	{prefix: "auth.background", topic: TopicSite, hidden: true},
	{prefix: "site.", topic: TopicSite, hidden: true},
	// **وتقليبُ سلايدر التطبيق يرجع مع لوحه** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — التطبيقُ يقرؤه.
	{prefix: "home.banner_enabled", exact: true, topic: TopicSite, panel: "slider"},
	{prefix: "home.banner_auto", exact: true, topic: TopicSite, panel: "slider"},
	{prefix: "home.banner_seconds", exact: true, topic: TopicSite, panel: "slider"},
	{prefix: "home.", topic: TopicSite, hidden: true},
	{prefix: "shop.", topic: TopicSite, hidden: true},
	{prefix: "platform.", topic: TopicSite, section: "site.contact"},
	{prefix: "page.", topic: TopicSite, section: "site.pages"},

	// ── الطلبات والتوصيل: الأجرةُ الخاصّة والمهلُ والتوزيع وحدودُ الزبون ──
	// **وكودُ التسليم مع صورة التسليم** — يُضبطان معاً (قرارُ المالك ٢٠٢٦-١٠-٠٦).
	{prefix: "delivery.code_required", exact: true, topic: TopicDrivers, section: "drivers.work"},
	{prefix: "delivery.code_channel", exact: true, topic: TopicDrivers, section: "drivers.work"},
	{prefix: "delivery.custom_", topic: TopicOrders, section: "orders.custom"},
	{prefix: "delivery.", topic: TopicOrders, section: "orders.dispatch"},
	{prefix: "orders.max_sources", exact: true, topic: TopicOrders, section: "orders.limits"},
	{prefix: "orders.max_open_per_customer", exact: true, topic: TopicOrders, section: "orders.limits"},
	{prefix: "orders.extra_source_fee", exact: true, topic: TopicOrders, section: "orders.limits"},
	{prefix: "orders.source_proximity_m", exact: true, topic: TopicOrders, section: "orders.limits"},
	{prefix: "orders.customer_cancel_window_sec", exact: true, topic: TopicOrders, section: "orders.timeouts"},
	{prefix: "orders.auto_dispatch", exact: true, topic: TopicOrders, section: "orders.dispatch"},
	{prefix: "orders.route_margin_pct", exact: true, topic: TopicOrders, section: "orders.dispatch"},
	{prefix: "orders.", topic: TopicOrders, section: "orders.timeouts"},
	{prefix: "drivers.assignment_mode", exact: true, topic: TopicOrders, section: "orders.dispatch"},
	{prefix: "drivers.direct_assign", exact: true, topic: TopicOrders, section: "orders.dispatch"},
	{prefix: "drivers.assigned_silence_sec", exact: true, topic: TopicOrders, section: "orders.dispatch"},
	{prefix: "drivers.offer_timeout_sec", exact: true, topic: TopicOrders, section: "orders.dispatch"},
	{prefix: "drivers.proximity_", topic: TopicOrders, section: "orders.dispatch"},
	{prefix: "drivers.dispatch_radius_", topic: TopicOrders, section: "orders.dispatch"},
	{prefix: "drivers.location_fresh_sec", exact: true, topic: TopicOrders, section: "orders.dispatch"},
	{prefix: "drivers.same_route_", topic: TopicOrders, section: "orders.dispatch"},
	{prefix: "drivers.zone_gate_enabled", exact: true, topic: TopicOrders, section: "orders.dispatch"},

	// ── الأدوار ──
	{prefix: "drivers.target_", topic: TopicDrivers, section: "drivers.targets"},
	{prefix: "drivers.monthly_target", exact: true, topic: TopicDrivers, section: "drivers.targets"},
	{prefix: "drivers.reward_", topic: TopicDrivers, section: "drivers.targets"},
	{prefix: "drivers.", topic: TopicDrivers, section: "drivers.work"},
	{prefix: "merchants.", topic: TopicMerchants},
	{prefix: "sales.", topic: TopicReps},
	{prefix: "customers.", topic: TopicCustomers},
}

// groupTopic **الملجأُ الأخير** — موضوعُ المجموعة القديمة.
var groupTopic = map[Group]Topic{
	GroupPlatform:  TopicLaunch,
	GroupLaunch:    TopicLaunch,
	GroupSite:      TopicSite,
	GroupApp:       TopicApps,
	GroupCustomers: TopicCustomers,
	GroupDispatch:  TopicOrders,
	GroupDrivers:   TopicDrivers,
	GroupMerchants: TopicMerchants,
	GroupSales:     TopicReps,
}

// PlacementOf **بيتُ المفتاح في الصفحة** — قاعدةٌ بالبادئة ثمّ مجموعتُه.
func PlacementOf(d Def) Placement {
	for _, r := range placeRules {
		hit := d.Key == r.prefix
		if !r.exact {
			hit = strings.HasPrefix(d.Key, r.prefix)
		}
		if hit {
			return Placement{Topic: r.topic, TopicSection: r.section, Panel: r.panel, Hidden: r.hidden}
		}
	}
	if t, ok := groupTopic[d.Group]; ok {
		return Placement{Topic: t}
	}
	return Placement{Topic: TopicLaunch}
}
