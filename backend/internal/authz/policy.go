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
	// **مَن يحمل الدور، وأثرُ المنح قبل وقوعه** (قرارُ المالك ٢٠٢٦-١٠-٠٤).
	{"GET", "/roles/{code}/members", RolesManage},
	{"GET", "/roles/{code}/impact", RolesManage},
	{"", "/capabilities", RolesManage},
	{"POST", "/users/{id}/roles", RolesManage},
	{"DELETE", "/users/{id}/roles/{role}", RolesManage},

	// ── صحّةُ المنصّة الداخليّة ───────────────────────────────────
	//
	// **وبابٌ واحدٌ يُقرأ ولا يُكتب** (دورةُ ٧٠أ).
	{"GET", "/ops/health", ObservabilityRead},
	// **وشاشةُ المراقب — سيرُ الطلبات بكلمات الخادم بلا أرقامه** (قرارُ المالك
	// ٢٠٢٦-١٠-٠٤ — «مراقبة التشغيل»، البند ٢): **موظّفُ العمليّات يراها،
	// والتفاصيلُ التقنيّةُ تبقى خلف `observability.read`.**
	{"GET", "/ops/monitor", OrdersRead},
	// **وأخطاءُ الخادم مجمَّعةً بالمسار والرمز** (مراقبةُ المنصّة ٢٠٢٦-١٠-٠٩) —
	// تفصيلٌ تقنيٌّ كصحّة المنصّة، فيبقى خلف `observability.read`.
	{"GET", "/monitoring/errors", ObservabilityRead},

	// ── حساباتُ الموظّفين والمستخدمين ────────────────────────────
	{"POST", "/users", UsersStatusManage},
	{"PATCH", "/users/{id}", UsersStatusManage},
	{"POST", "/users/{id}/password", UsersStatusManage},
	{"POST", "/users/{id}/logout-all", UsersStatusManage},
	// **قسمُ الحسابات** (قراراتُ المالك ٢٠٢٦-١٠-٠٤).
	{"POST", "/users/{id}/resend-welcome", UsersStatusManage},
	{"GET", "/phone-requests", UsersRead},
	{"POST", "/phone-requests/{id}/approve", UsersStatusManage},
	{"POST", "/phone-requests/{id}/reject", UsersStatusManage},
	{"GET", "/users/{id}/notes", UsersRead},
	{"POST", "/users/{id}/notes", UsersRead},
	{"GET", "/users/{id}/cash-ban", UsersRead},
	{"POST", "/users/{id}/cash-ban/lift", UsersCashBanLift},
	{"POST", "/users/{id}/driver-ok", DriversManage},
	{"PATCH", "/users/{id}/vehicle", DriversManage},
	{"PATCH", "/users/{id}/cash-limit", SettingsFinancialManage},
	{"POST", "/users/{id}/warnings", SafetyManage},
	{"GET", "/users/{id}/warnings", SafetyManage},
	{"GET", "/users/{id}/warn-reasons", SafetyManage},
	{"GET", "/users/{id}/financials", FinanceRead},
	{"GET", "/users/{id}/wallet", FinanceRead},
	{"POST", "/users/{id}/wallet", FinanceManage},
	{"GET", "/wallet-requests", FinanceRead},
	{"POST", "/wallet-requests/{id}/approve", FinanceManage},
	{"POST", "/wallet-requests/{id}/reject", FinanceManage},
	{"POST", "/users/{id}/incentive", FinanceManage},
	{"GET", "/users/{id}/incentives", FinanceRead},
	{"GET", "/incentive-requests", FinanceRead},
	{"POST", "/incentive-requests/{id}/approve", FinanceManage},
	{"POST", "/incentive-requests/{id}/reject", FinanceManage},
	{"POST", "/incentive-alerts/{id}/decide", FinanceManage},
	{"POST", "/incentive-failures/{id}/retry", FinanceManage},
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
	{"POST", "/orders/{id}/custom-quote", OrdersIntervene}, // تدخّلُ الأدمن على عرض المخصَّص — Batch 2a
	{"POST", "/orders/{id}/assign", OrdersIntervene},
	{"POST", "/orders/{id}/transfer", OrdersIntervene},
	// **ومرشّحو التحويل يُقرؤون لمن يحوّل** — فيهم أسعارُ الشراء.
	{"GET", "/orders/{id}/transfer-candidates", OrdersIntervene},
	// **وإعادةُ حساب التسوية للمالك والأدمن وحدَهما** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ١٥).
	{"POST", "/orders/{id}/recompute", FinanceRecompute},
	// **ومرشّحو الإسناد اليدويّ** — بالقرب والنقد والطلبات (البند ١٠).
	{"GET", "/orders/{id}/assign-candidates", OrdersIntervene},
	// **و«استلمتها» على الطلب الجديد** — يُسكت رنينَه عند المكتب كلِّه (البند ٥).
	{"POST", "/orders/{id}/seen", OrdersIntervene},
	// **و«أنا عليه» على العالق** — يوقف تكرارَ تذكيره («مراقبة التشغيل»، البند ٤).
	{"POST", "/orders/{id}/alert-ack", OrdersIntervene},
	{"POST", "/orders/{id}/proof-exception", OrdersIntervene}, // إذنُ تسليمٍ بلا صورةٍ — عملياتٌ مُخوَّلةٌ لا السائق (٢٠٢٦-٠٩-٢٧)
	{"POST", "/orders/{id}/compensate-driver", FinanceManage},
	{"POST", "/orders/{id}/compensation/reject", FinanceManage},
	{"GET", "/compensations/pending", FinanceRead},
	// **صفحةُ «التعويضات» الواحدة** (قرارُ المالك ٢٠٢٦-١٠-٠٤).
	{"GET", "/compensations", FinanceRead},
	{"POST", "/compensations", FinanceManage},
	{"POST", "/compensations/{id}/approve", FinanceManage},
	{"POST", "/compensations/{id}/reject", FinanceManage},
	{"POST", "/orders/{id}/settle-goods", FinanceManage},
	// **العمليّاتُ تقرّر أين البضاعة، والماليّةُ تكتب التعويض** (البند ١٢).
	{"POST", "/orders/{id}/goods", OrdersIntervene},
	{"POST", "/orders/{id}/goods/compensation", FinanceManage},
	// **إنهاءُ الإدارة عند باب الزبون** — سلّم الآن أو عُد إلى المكتب (مساءَ ٢٠٢٦-١٠-٠٢).
	{"POST", "/orders/{id}/door-resolution", OrdersIntervene},
	{"POST", "/orders/{id}/whatsapp", OrdersIntervene},
	{"GET", "/orders/{id}/breakdown", FinanceRead},
	// **وتصديرُ الطلبات كشفُ محاسبةٍ لا تقرير** — فيه **اسمُ الزبون
	// وهاتفُه** وأنصبةُ كلّ طلبٍ من الدفتر. **وحارسُه قبلَ `ADG-2`
	// كان `admin,finance`** — **ودورةُ ٢٥ وسّعته إلى التحليلات
	// فأدخلت التحليلاتِ على أرقام الهواتف.** (مصالحةُ دورةِ ٢٦.)
	{"GET", "/orders/export", FinanceExport},
	{"GET", "/orders", OrdersRead},
	{"GET", "/orders/alerts", OrdersRead},
	// **عدّاداتُ اللوحة بالشرط الذي تُرشِّح به** (البند ٣).
	{"GET", "/orders/board", OrdersRead},
	{"GET", "/orders/{id}", OrdersRead},
	// **وكلامُ الناس صنفٌ بذاته** — ومن يسوّي حساباً لا يقرؤه.
	{"GET", "/orders/{id}/chat", OrdersCommunicationsRead},
	// **ورسالةُ المنصّة إلى المتجر ليست كلامَ الناس** — نصٌّ تبنيه المنصّةُ ورابطُ
	// واتساب لزرّ «حوّل للمتجر». **وكانت بـ`orders.communications.read`** فموظّفُ
	// العمليّات يُردّ ٤٠٣ عند التحويل (بلاغُ المالك من التجهيز ٢٠٢٦-١٠-٠٥).
	// **فهي بقدرة الفعل الذي تخدمه** — كأختها `POST /orders/{id}/whatsapp`.
	{"GET", "/orders/{id}/message", OrdersIntervene},

	// ── المتاجر: إدارةٌ · وسلامةٌ · وتوثيق ───────────────────────
	{"POST", "/merchants/{id}/suspend", SafetyManage},
	{"POST", "/merchants/{id}/warnings", SafetyManage},
	{"POST", "/merchants/{id}/clear-violations", SafetyManage},
	{"GET", "/merchants/{id}/violations", SafetyManage},
	{"GET", "/merchants/{id}/warnings", SafetyManage},
	{"POST", "/leads/{id}/status", MerchantsVerify},
	{"GET", "/leads", MerchantsVerify},
	{"POST", "/merchants", MerchantsManage},
	{"PATCH", "/merchants/{id}", MerchantsManage},
	// **إعدادُ المتاجر — مَن ضبط ومَن لم يضبط، والتذكيرُ بيد الموظّف** (٢٠٢٦-١٠-٠٨).
	{"GET", "/merchants-setup", MerchantsRead},
	{"POST", "/merchants/{id}/setup-reminder", MerchantsManage},
	// **تسويةُ مستحقّات المتجر — صلاحيّةٌ ماليّة لا إدارةُ متجر.**
	{"PATCH", "/merchants/{id}/settlement-method", SettingsFinancialManage},
	// **وسقفُ دينِ التوصيلة كذلك** — رقمٌ يُدين به المتجرُ المنصّة (الخطوة ١٨).
	{"GET", "/merchants/{id}/delivery-credit", FinanceRead},
	{"PATCH", "/merchants/{id}/delivery-credit", SettingsFinancialManage},
	{"GET", "/merchants/{id}/cash-settlements", FinanceRead},
	{"POST", "/merchant-cash-settlements/{id}/pay", FinanceManage},
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
	{"GET", "/cash/outstanding/export", FinanceExport},
	{"GET", "/cash/merchant-dues", FinanceRead},
	{"POST", "/drivers/{id}/end-shift", DriversManage},
	{"GET", "/drivers", DriversRead},

	// ── المال ───────────────────────────────────────────────────
	{"POST", "/payouts/{id}/decide", PayoutsDecide},
	{"GET", "/payouts", FinanceRead},
	// **والديون** — القراءةُ ماليّة، والتصديرُ بقدرته، والدفعُ والشطبُ اقتراحٌ
	// بـ`finance.manage`. **وموافقةُ الشطب تشترط `finance.writeoff.approve` داخلَ
	// الباب** (الدفعةُ بـ`finance.manage`)، فالمسارُ واحدٌ بحدّه الأدنى.
	{"GET", "/obligations", FinanceRead},
	{"GET", "/obligations/export", FinanceExport},
	{"POST", "/obligations/{id}/office-cash", FinanceManage},
	{"POST", "/obligations/{id}/write-off", FinanceManage},
	{"GET", "/obligation-requests", FinanceRead},
	{"POST", "/obligation-requests/{id}/approve", FinanceManage},
	{"POST", "/obligation-requests/{id}/reject", FinanceManage},
	{"GET", "/profits", FinanceRead},
	{"GET", "/expenses", FinanceRead},
	{"GET", "/expenses/categories", FinanceRead},
	{"POST", "/expenses", FinanceManage},
	{"POST", "/expenses/categories", FinanceManage},
	{"POST", "/expenses/{id}/void", FinanceManage},
	{"GET", "/expenses/export", FinanceExport},
	{"POST", "/expenses/receipt", FinanceManage},
	{"GET", "/expense-requests", FinanceRead},
	{"POST", "/expense-requests/{id}/approve", FinanceManage},
	{"POST", "/expense-requests/{id}/reject", FinanceManage},
	{"GET", "/ledger/export", FinanceExport},
	{"GET", "/treasury-candidates", FinanceManage},
	{"GET", "/reports/losses", FinanceRead},
	// **وتعويضُ البضاعة الراجعة من «الخسائر والنزاعات»** (قرارُ المالك ٢٠٢٦-١٠-٠٥) —
	// قائمةٌ تُقرأ، **والتعويضُ من بابه** (`/orders/{id}/goods/compensation`، `finance.manage`).
	{"GET", "/losses/goods-compensations", FinanceRead},
	// **و«شحن محفظة» من الخزينة** — بحثٌ بخمسة حقول بقدرة من يقترح الحركة،
	// **لا بـ`users.read`**: الماليّةُ لا ترى الحسابات (قرارُ المالك ٢٠٢٦-١٠-٠٥).
	{"GET", "/treasury/wallet-lookup", FinanceManage},
	// **الخزينةُ وصندوقُ المكتب والإغلاقُ اليوميّ والموافقاتُ الموحّدة**
	// (قراراتُ المالك ٢٠٢٦-١٠-٠٤ — الخزينة).
	{"GET", "/treasury/overview", FinanceRead},
	{"GET", "/treasury/statement", FinanceRead},
	{"GET", "/treasury/statement/export", FinanceExport},
	{"GET", "/treasury/withdrawals", FinanceRead},
	{"POST", "/treasury/withdrawals", TreasuryManage},
	{"GET", "/treasury/health", FinanceRead},
	{"GET", "/cashbox", FinanceRead},
	{"GET", "/cashbox/closes", FinanceRead},
	{"POST", "/cashbox/closes", FinanceManage},
	{"POST", "/cashbox/closes/{id}/approve", FinanceManage},
	{"POST", "/cashbox/closes/{id}/reject", FinanceManage},
	{"POST", "/cashbox/shortfalls/{id}/resolve", FinanceManage},
	{"POST", "/cashbox/shortfalls/{id}/approve", TreasuryManage},
	{"POST", "/cashbox/shortfalls/{id}/reject", TreasuryManage},
	{"GET", "/approvals", FinanceRead},

	// ── الدعمُ والنزاعاتُ والطوارئ ───────────────────────────────
	{"GET", "/tickets", SupportManage},
	{"POST", "/tickets", SupportManage},
	{"GET", "/tickets/{id}", SupportManage},
	{"POST", "/tickets/{id}/replies", SupportManage},
	{"POST", "/tickets/{id}/resolve", SupportManage},
	// **والطوارئُ بقدرتها** — يبلغها الدعمُ والعمليّاتُ معاً (قرارُ المالك ٢٠٢٦-١٠-٠٤،
	// قسمُ «الطلبات»، البند ٧): **من يوزّع الطلبات يرى الحادث ويستلمه.**
	{"GET", "/emergencies", EmergenciesManage},
	{"POST", "/emergencies/{id}/resolve", EmergenciesManage},
	// **وشريطُ الطوارئ أعلى كلّ صفحةٍ وزرُّ «استلمتها»** (٢٠٢٦-١٠-٠٤).
	{"GET", "/emergencies/banner", EmergenciesManage},
	{"POST", "/emergencies/{id}/ack", EmergenciesManage},
	{"POST", "/emergencies/stores/{id}/ack", EmergenciesManage},
	// **وغرفةُ الطوارئ بخطواتها** (٢٠٢٦-١٠-٠٤) — والمالُ طلبُ تعويضٍ لا دفع،
	// **والموافقةُ عليه في الماليّة** (`/orders/{id}/compensate-driver`).
	{"GET", "/emergencies/count", EmergenciesManage},
	{"GET", "/emergencies/map", EmergenciesManage},
	{"GET", "/emergencies/{id}", EmergenciesManage},
	{"POST", "/emergencies/{id}/driver-ok", EmergenciesManage},
	{"POST", "/emergencies/{id}/outcome", EmergenciesManage},
	{"POST", "/emergencies/{id}/money", EmergenciesManage},
	{"POST", "/emergencies/{id}/notes", EmergenciesManage},
	// **النزاعاتُ بقدرتها** (قرارُ المالك ٢٠٢٦-١٠-٠٤، «الخسائر والنزاعات» البندان ١ و٢):
	// الدعمُ والماليّةُ يريان ويفتحان، **والحسمُ اقتراحٌ من الماليّة وموافقةُ غيرِ المقترِح.**
	{"GET", "/disputes", DisputesManage},
	{"POST", "/disputes", DisputesManage},
	{"GET", "/disputes/parties", DisputesManage},
	{"POST", "/disputes/{id}/propose", FinanceManage},
	{"GET", "/dispute-resolutions", FinanceRead},
	{"POST", "/dispute-resolutions/{id}/approve", FinanceManage},
	{"POST", "/dispute-resolutions/{id}/reject", FinanceManage},
	{"GET", "/ratings", SupportManage},

	// ── المحتوى والتسويق ────────────────────────────────────────
	// **«العروض والخصومات» بقدرتها لا بقدرة المحتوى** (قرارُ المالك ٢٠٢٦-١٠-٠٥):
	// قسمٌ للماليّة — **ولا تنال معه اللافتاتِ ولا الحملاتِ ولا البثَّ ولا صورَ المنصّة.**
	// **ملخّصُ العروض** — قبل `/promos/{id}` (الأخصُّ أوّلاً).
	{"GET", "/promos/summary", OffersManage},
	{"", "/promos", OffersManage},
	{"", "/promos/{id}", OffersManage},
	{"", "/banners", ContentManage},
	{"", "/banners/{id}", ContentManage},
	{"GET", "/offers/audience", OffersManage},
	{"", "/offers", OffersManage},
	// **«ادعُ صديقاً»** — تبويبٌ في صفحة العروض (قرارُ المالك ٢٠٢٦-١٠-٠٤).
	{"GET", "/referrals", OffersManage},
	// **موافقاتُ الماليّة على العروض فوق حدّ المحتوى** — على عقد الموافقات.
	{"GET", "/promo-approvals", FinanceRead},
	{"POST", "/promo-approvals/{id}/approve", FinanceManage},
	{"POST", "/promo-approvals/{id}/reject", FinanceManage},
	{"", "/offers/{id}/active", OffersManage},
	// **وترتيبُ الأقسام بالسحب قبل `{id}`** — الأخصُّ أوّلاً.
	{"PUT", "/sections/order", MarketManage},
	{"", "/sections", MarketManage},
	{"", "/sections/{id}", MarketManage},
	{"GET", "/sections/{id}/items", MarketManage},
	// ── «السوق» — أصنافُ كلّ المتاجر (قرارُ المالك ٢٠٢٦-١٠-٠٤) ──
	// **وبقدرتها لا بقدرة المحتوى** (قرارُ المالك ٢٠٢٦-١٠-٠٥): موظّفُ العمليّات
	// يرتّب السوقَ ولا يبلغ العروضَ ولا اللافتاتِ ولا الحملات.
	// **وصورُ السوق من بابها** — يرفع صورةَ صنفٍ أو قسمٍ لا شعارَ المنصّة.
	{"POST", "/market/media", MarketManage},
	// **وصنفُ السوق يُضاف ويُعدَّل ويُطفأ من بابه** (قرارُ المالك ٢٠٢٦-١٠-٠٥): كانت
	// صفحةُ القسم تنادي بابَي إدارة المتاجر (`/merchants/{id}/menu/items` ·
	// `/menu/items/{id}`) — **فموظّفُ العمليّات يرى الزرَّ ويُردّ ٤٠٣.** والمعالِجُ
	// نفسُه، **ولا تُمنح `merchants.manage`** (إنشاءُ المتاجر وتحريرُها).
	{"POST", "/market/stores/{id}/items", MarketManage},
	{"PATCH", "/market/items/{itemID}", MarketManage},
	{"GET", "/market/items", MarketManage},
	{"POST", "/market/items/bulk", MarketManage},
	{"GET", "/market/new-count", MarketManage},
	{"POST", "/market/seen", MarketManage},
	{"GET", "/market/stores", MarketManage},
	{"GET", "/market/quality", MarketManage},
	{"POST", "/market/test-data/delete", MarketManage},
	{"", "/categories", MarketManage},
	{"", "/categories/{id}", MarketManage},
	{"POST", "/media", ContentManage},
	{"GET", "/media/sign", ContentManage},
	// **ومركزُ الإشعارات بالقدرة نفسِها** — **وهي قدرةُ من يخاطب
	// الناسَ باسم المنصّة**: **لا تُخترَع قدرةٌ ثانيةٌ لفعلٍ من صنفها.**
	{"", "/campaigns", ContentManage},
	{"GET", "/campaigns/preview", ContentManage},
	{"", "/campaigns/{id}/send", ContentManage},
	{"", "/campaigns/{id}/cancel", ContentManage},
	{"POST", "/broadcast", ContentManage},
	{"GET", "/broadcast/count", ContentManage},
	// **وملفُّ التطبيق إعدادٌ لا محتوى** (قرارُ المالك ٢٠٢٦-١٠-٠٤، الإعدادات):
	// كان بقدرة المحتوى — **فموظّفُ المحتوى يبدّل تطبيقَ السائق عند كلّ الكباتن.**
	{"", "/app-file", SettingsGeneralManage},

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
	// **وقراءةُ أسماء المحافظات قراءةُ لوح** (٢٠٢٦-١٠-٠٥): صفحةُ طلبات الانضمام
	// ترشّح بها — **ورسمُها وتحريرُها يبقيان بالإعدادات العامّة.**
	{"GET", "/governorates", SettingsRead},
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
	// **وكثافةُ الطلب بالمكان الإداريّ** — `CR`. **ولا قدرةَ جديدة.**
	// **وكانت تحليليّةً ثمّ صارت بـ`orders.read`** (قرارُ المالك ٢٠٢٦-١٠-٠٥): «طلباتُ التوسّع» قسمٌ
	// لموظّف العمليّات، **ولا يُمنَح `analytics.read` لأجلها** — فتلك تقاريرُ المدير.
	{"GET", "/ops-map/coverage-demand/places", OrdersRead},
	{"GET", "/ops-map/coverage-requests", OrdersRead},
	{"", "/ops-map/coverage-requests/{id}", SettingsGeneralManage},
	// **«طلباتُ التوسّع»** — قراءتُها بـ`orders.read` كأختها، **وإبلاغُ المنتظرين
	// لمن يملك التغطية** (قرارُ المالك ٢٠٢٦-١٠-٠٤).
	{"GET", "/ops-map/expansion", OrdersRead},
	{"GET", "/ops-map/expansion/reminder", OrdersRead},
	{"", "/ops-map/expansion/notify", SettingsGeneralManage},
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
	{"GET", "/ops-map/demand", OrdersRead},
	{"GET", "/ops-map/opportunities", OrdersRead},
	// **ونشاطُ المندوبين لمن يبني شبكةَ المتاجر** — **ولا يُقرأ
	// بـ`users.read`**: **الماليّةُ تملكها ولا تُراقب مندوباً.**
	{"GET", "/ops-map/reps", MerchantsManage},
	{"GET", "/ops-map/meta", OrdersRead},
	// **شريطُ العدّادات والزبائنُ المجمَّعون والمكتب** (قرارُ المالك ٢٠٢٦-١٠-٠٥) —
	// قراءاتٌ تشغيليّةٌ لموظّف العمليّات: **الزبائنُ خلايا لا بيوت** (حدٌّ أدنى
	// ثلاثة)، **والمكتبُ اسمٌ ودورٌ لمن حضر** بلا هاتف.
	{"GET", "/ops-map/summary", OrdersRead},
	{"GET", "/ops-map/customers", OrdersRead},
	{"GET", "/ops-map/office", OrdersRead},
	{"GET", "/ops-map/search", OrdersRead},

	// ── الحوافز ─────────────────────────────────────────────────
	{"GET", "/incentives/{role}", FinanceRead},
	{"GET", "/incentives/{role}/export", FinanceRead},

	// ── الإعدادات والتقارير والتشخيص ─────────────────────────────
	//
	// **و`PUT /settings/{key}` قدرتُه تتبع المفتاحَ لا المسار** —
	// **يُحسَم في المعالِج** (`settingCapability`). **وهو مستثنىً
	// هنا عمداً ومكتوب.**
	// **وقراءةُ اللوح ليست تبديلَ مفتاح** (٢٠٢٦-٠٩-١٣): **شاشةُ
	// الطلبات تقرأ مفتاحين تشغيليّين** — **وكانت تطلب لأجلهما رسمَ
	// المناطق والمدن والمحافظات.**
	{"GET", "/settings", SettingsRead},
	{"GET", "/settings/money-example", SettingsRead},
	// **وحالُ التطبيق قراءتُها قراءةُ لوح**، **وتبديلُها تبديلُ
	// مفاتيحِه** — **ولا قدرةَ جديدةً لبابٍ يكتب ما يكتبه `PUT`.**
	{"GET", "/launch", SettingsRead},
	{"POST", "/launch/preset", SettingsGeneralManage},
	{"GET", "/audit", AuditRead},
	{"GET", "/audit/actors", AuditRead},
	// **وتصديرُ السجلّ لمدير المنصّة ومالكها وحدَهما** (قرارُ المالك 2026-10-04)
	// — والتصديرُ نفسُه يُقيَّد فيه.
	{"GET", "/audit/export", AuditExport},
	{"GET", "/stats", AnalyticsRead},
	// **ورئيسيّةُ المدير بقدرتها** — فيها المالُ كلُّه (قرارُ المالك ٢٠٢٦-١٠-٠٤).
	{"GET", "/overview", PlatformOverview},
	// **وزوّارُ الموقع والتحميلاتُ على الرئيسيّة نفسِها** (طلبُ المالك ٢٠٢٦-١٠-٠٧).
	{"GET", "/site-stats", PlatformOverview},
	{"GET", "/reports", AnalyticsRead},
	{"GET", "/whatsapp", SettingsSecurityManage},
	{"POST", "/whatsapp/pair", SettingsSecurityManage},
	{"POST", "/whatsapp/unpair", SettingsSecurityManage},
}

// Exempt **مساراتٌ لا تُحكَم بالجدول — ولكلٍّ سببٌ مكتوب.**
var Exempt = map[string]string{
	// **وحالُ الخادم لكلّ موظّفٍ في اللوحة** — كلماتٌ بلا أرقام، يقرؤها
	// الشريطُ الأحمر (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «مراقبة التشغيل»، البند ٣).
	"/ops/status": "**حالُ الخادم بكلمات لكلّ موظّف** — الشريطُ الأحمرُ أعلى " +
		"اللوحة؛ يكفيه `RequireAnyCapability`، **ولا رقمَ ولا سببَ تقنيّاً في ردّه.**",
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

// HandlerRule **قدرةٌ يشترطها البابُ من داخله فوق قدرةِ مساره.**
//
// **والمسارُ واحدٌ بحدّه الأدنى** (`Rules`)، **والقدرةُ الأعلى تُسأل داخلَ
// المعالِج** لأنّها تتبع الصفَّ لا المسار — كموافقةِ الشطب: الطلبُ نفسُه
// دفعةٌ أو شطب، **والدفعةُ بـ`finance.manage` والشطبُ بقدرته.**
type HandlerRule struct {
	Method  string
	Pattern string
	// Need القدرةُ التي يسألها المعالِجُ فوق قدرة المسار.
	Need Capability
	// Where موضعُ السؤال في الشيفرة — ليُقرأ ولا يُظنّ.
	Where string
}

// handlerChecks **القدراتُ المسؤولةُ داخلَ الأبواب** — كلُّ سطرٍ مسارٌ قائمٌ
// في `adminPolicy` (يحرسه `TestHandlerChecksHaveRoutes`).
var handlerChecks = []HandlerRule{
	{"POST", "/obligation-requests/{id}/approve", FinanceWriteoffApprove,
		"server/obligations_actions.go handleDecideObligationRequest"},
}

// HandlerChecks القدراتُ التي تُسأل داخلَ المعالِج — **للعقد والحرّاس.**
func HandlerChecks() []HandlerRule { return handlerChecks }
