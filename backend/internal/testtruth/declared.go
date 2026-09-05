// المُعلَن — **ما لا تقوله الشيفرةُ عن نفسها.**
//
// **وهذا وحدَه ما يُكتب بيد** — **وكلُّ ما عداه يُستخرَج** (`extract.go`).
//
// # ولماذا لا يُستخرَج هذا
//
// **«أيَّ عيبٍ يحرس هذا الاختبار» قصدٌ لا يقوله الكود.** **واسمُ الدالّة
// يلمّح ولا يُثبت** — ومن بنى الربطَ على الأسماء بنى على رملٍ يتبدّل مع
// أوّلِ إعادةِ تسمية.
//
// # والحارسُ يمنع الشيخوخة
//
// **مرجعٌ إلى اختبارٍ لم يعد موجوداً ⇒ `STALE` ⇒ سقوط.**
// **وعيبٌ في السجلّ لا يعرفه هذا الملفّ ⇒ سقوط.**
// **فالإعلانُ يدويٌّ والصيانةُ مفروضة.**
package testtruth

// ══════════════════════════════════════════════════════════════════════
// **التدفّقاتُ الأربعةُ والثلاثون**
// ══════════════════════════════════════════════════════════════════════
//
// **مصدرُها** `docs/testing/CROSS_SYSTEM_INTERACTION_CLOSURE.md` §١.
// **والطبقاتُ اللازمةُ تُشتقّ بقواعد** (`deriveLevels`) **لا تُكتب.**

// FlowDecl إعلانُ تدفّقٍ — **الحدُّ الأدنى الذي لا يُستخرَج.**
type FlowDecl struct {
	ID       string
	Title    string
	Actor    string
	Apps     []string
	Money    bool
	Realtime bool
	Partial  bool
	Race     bool
	Severity string
	Defects  []string
	Risks    []string
	Gaps     []string
}

// Flows الأربعةُ والثلاثون.
var Flows = []FlowDecl{
	{ID: "F-01", Title: "إنشاءُ طلبٍ عاديّ", Actor: "customer", Apps: []string{"customer", "merchant", "admin"}, Money: true, Realtime: true, Partial: true, Severity: "BLOCKER", Defects: []string{"D4", "D20", "D23"}, Risks: []string{"R8"}, Gaps: []string{"XG-9"}},
	{ID: "F-02", Title: "إنشاءُ طلبٍ خاصّ", Actor: "customer", Apps: []string{"customer", "admin"}, Money: true, Partial: true, Severity: "BLOCKER", Defects: []string{"D6", "D8", "D9", "D22"}, Risks: []string{"R11"}},
	{ID: "F-03", Title: "ردُّ إنشاءٍ ضائع", Actor: "customer", Apps: []string{"customer"}, Money: true, Partial: true, Severity: "CRITICAL", Risks: []string{"R8"}},
	{ID: "F-04", Title: "قبولُ المتجر", Actor: "merchant", Apps: []string{"merchant", "customer"}, Realtime: true, Severity: "CRITICAL", Defects: []string{"D19", "D20"}, Risks: []string{"R21"}},
	{ID: "F-05", Title: "القبولُ التلقائيّ", Actor: "watchdog", Apps: []string{"merchant", "customer"}, Realtime: true, Partial: true, Severity: "CRITICAL", Gaps: []string{"XG-7"}},
	{ID: "F-06", Title: "رفضُ المتجر", Actor: "merchant", Apps: []string{"merchant", "customer", "admin"}, Money: true, Realtime: true, Severity: "HIGH"},
	{ID: "F-07", Title: "عرضُ الطلب على سائق", Actor: "engine", Apps: []string{"driver"}, Realtime: true, Partial: true, Race: true, Severity: "CRITICAL", Risks: []string{"R10"}},
	{ID: "F-08", Title: "قبولُ السائق", Actor: "driver", Apps: []string{"driver", "customer", "merchant"}, Money: true, Realtime: true, Partial: true, Race: true, Severity: "BLOCKER", Defects: []string{"D7"}, Risks: []string{"R7", "R10"}},
	{ID: "F-09", Title: "انقضاءُ العرض ودورانُه", Actor: "watchdog", Apps: []string{"driver", "admin"}, Realtime: true, Partial: true, Race: true, Severity: "HIGH", Defects: []string{"D1"}, Risks: []string{"R1"}},
	{ID: "F-10", Title: "عرضُ طلبٍ على الطريق نفسِه", Actor: "engine", Apps: []string{"driver", "customer"}, Money: true, Realtime: true, Partial: true, Race: true, Severity: "HIGH", Defects: []string{"D7"}, Risks: []string{"R10"}},
	{ID: "F-11", Title: "الوصولُ للاستلام", Actor: "driver", Apps: []string{"driver", "customer"}, Realtime: true, Severity: "MEDIUM", Gaps: []string{"XG-6"}},
	{ID: "F-12", Title: "الاستلام", Actor: "driver", Apps: []string{"driver", "customer", "merchant"}, Realtime: true, Severity: "HIGH"},
	{ID: "F-13", Title: "التسليمُ وإثباتُه", Actor: "driver", Apps: []string{"driver", "customer", "merchant", "admin"}, Money: true, Realtime: true, Partial: true, Severity: "BLOCKER", Defects: []string{"D13"}},
	{ID: "F-14", Title: "تسويةُ التسليم", Actor: "engine", Apps: []string{"merchant", "driver", "rep", "admin"}, Money: true, Realtime: true, Partial: true, Severity: "BLOCKER", Defects: []string{"D20", "D23"}, Risks: []string{"R4", "R23"}, Gaps: []string{"XG-25", "XG-26"}},
	{ID: "F-15", Title: "الاسترداد", Actor: "admin", Apps: []string{"customer", "merchant", "rep", "admin"}, Money: true, Realtime: true, Partial: true, Race: true, Severity: "BLOCKER", Gaps: []string{"XG-10", "XG-11"}},
	{ID: "F-16", Title: "تعذّرُ التسليم", Actor: "driver", Apps: []string{"driver", "customer", "admin"}, Money: true, Realtime: true, Severity: "HIGH"},
	{ID: "F-17", Title: "إلغاءُ الزبون", Actor: "customer", Apps: []string{"customer", "merchant", "driver"}, Money: true, Realtime: true, Race: true, Severity: "HIGH"},
	{ID: "F-18", Title: "تحويلُ الطلب لمتجرٍ آخر", Actor: "admin", Apps: []string{"merchant", "customer", "admin"}, Money: true, Realtime: true, Partial: true, Severity: "HIGH"},
	{ID: "F-19", Title: "إسنادٌ إداريٌّ وإعادةُ إسناد", Actor: "admin", Apps: []string{"driver", "customer", "admin"}, Realtime: true, Partial: true, Race: true, Severity: "CRITICAL", Defects: []string{"D1"}, Gaps: []string{"XG-5"}},
	{ID: "F-20", Title: "إرسالُ الطلب بواتساب", Actor: "admin", Apps: []string{"merchant", "admin"}, Partial: true, Severity: "CRITICAL", Risks: []string{"R21", "R24"}},
	{ID: "F-21", Title: "تسجيلُ مرشَّحٍ ثمّ تحويلُه", Actor: "rep", Apps: []string{"rep", "merchant", "admin"}, Money: true, Partial: true, Race: true, Severity: "BLOCKER", Defects: []string{"D2"}, Gaps: []string{"XG-18", "XG-29", "XG-30"}},
	{ID: "F-22", Title: "نقلُ متجرٍ بين مندوبين", Actor: "admin", Apps: []string{"rep", "admin"}, Money: true, Severity: "HIGH", Gaps: []string{"XG-17"}},
	{ID: "F-23", Title: "عمولةُ المندوب", Actor: "engine", Apps: []string{"rep", "admin"}, Money: true, Realtime: true, Severity: "BLOCKER", Gaps: []string{"XG-13", "XG-14", "XG-15", "XG-26", "XG-27", "XG-28"}},
	{ID: "F-24", Title: "طلبُ سحبٍ وقرارُه", Actor: "multi", Apps: []string{"driver", "merchant", "rep", "admin"}, Money: true, Realtime: true, Race: true, Severity: "BLOCKER", Risks: []string{"R8"}, Gaps: []string{"XG-12"}},
	{ID: "F-25", Title: "قيدٌ يدويٌّ في محفظة", Actor: "admin", Apps: []string{"admin"}, Money: true, Realtime: true, Severity: "CRITICAL", Gaps: []string{"XG-20"}},
	{ID: "F-26", Title: "مصروفٌ وخزينة", Actor: "admin", Apps: []string{"admin"}, Money: true, Partial: true, Severity: "CRITICAL", Defects: []string{"D5"}},
	{ID: "F-27", Title: "تسويةُ نقد السائق", Actor: "admin", Apps: []string{"driver", "admin"}, Money: true, Realtime: true, Severity: "CRITICAL", Defects: []string{"D7"}},
	{ID: "F-28", Title: "تعليقُ متجر", Actor: "admin", Apps: []string{"merchant", "customer", "admin"}, Realtime: true, Race: true, Severity: "HIGH", Defects: []string{"D14"}},
	{ID: "F-29", Title: "تعليقُ سائقٍ أو زبون", Actor: "admin", Apps: []string{"driver", "customer", "admin"}, Realtime: true, Race: true, Severity: "CRITICAL", Defects: []string{"D14"}, Gaps: []string{"XG-22", "XG-23", "XG-24"}},
	{ID: "F-30", Title: "إعادةُ كلمةِ مرورٍ وإخراجٌ شامل", Actor: "admin", Apps: []string{"customer", "driver", "merchant", "rep"}, Realtime: true, Race: true, Severity: "CRITICAL", Defects: []string{"D10", "D11", "D12"}, Risks: []string{"R13", "R15", "R16"}},
	{ID: "F-31", Title: "شكوى أو بلاغٌ ثمّ حلٌّ بتعويض", Actor: "multi", Apps: []string{"customer", "driver", "merchant", "admin"}, Money: true, Severity: "HIGH", Gaps: []string{"XG-19"}},
	{ID: "F-32", Title: "مراجعةُ صنفٍ معلَّق", Actor: "admin", Apps: []string{"merchant", "rep", "customer", "admin"}, Money: true, Severity: "MEDIUM"},
	{ID: "F-33", Title: "تبديلُ إعدادٍ حسّاس", Actor: "admin", Apps: []string{"customer", "driver", "merchant", "rep"}, Money: true, Realtime: true, Race: true, Severity: "BLOCKER", Gaps: []string{"XG-25", "XG-26", "XG-27", "XG-28"}},
	{ID: "F-34", Title: "بثٌّ للأدوار", Actor: "admin", Apps: []string{"customer", "driver", "merchant", "rep"}, Partial: true, Severity: "LOW", Gaps: []string{"XG-9"}},
}

// ══════════════════════════════════════════════════════════════════════
// **فجواتُ العقد — وشدّتُها من `TQ-1`**
// ══════════════════════════════════════════════════════════════════════

// GapDecl فجوةٌ معلَنةٌ بشدّتها.
type GapDecl struct {
	ID       string
	Title    string
	Severity string
	// WokenBy **الإعدادُ الذي يوقظ النائمة** — و`LATENT` وحدَها تملؤه.
	WokenBy string
}

// Gaps السّتُّ والعشرون.
var Gaps = []GapDecl{
	{ID: "XG-5", Title: "لا إشعارَ للمتجر عند الإسناد", Severity: "HIGH"},
	{ID: "XG-6", Title: "لا إشعارَ للمتجر عند وصول السائق", Severity: "HIGH"},
	{ID: "XG-7", Title: "القبولُ التلقائيُّ صامتٌ عن المتجر", Severity: "HIGH", WokenBy: "orders.auto_accept_min"},
	{ID: "XG-8", Title: "الإشعارُ يفتح الشاشةَ الأولى لا الكيان", Severity: "MEDIUM"},
	{ID: "XG-9", Title: "٢٤ إشعاراً بلا توجيهِ تطبيق", Severity: "MEDIUM"},
	{ID: "XG-10", Title: "لا عكسَ لعمولة المندوب عند الاسترداد", Severity: "BLOCKER"},
	{ID: "XG-11", Title: "الاستردادُ يفشل إن سحب المندوبُ عمولتَه", Severity: "BLOCKER"},
	{ID: "XG-12", Title: "لا طبقاتِ رصيد", Severity: "CRITICAL"},
	{ID: "XG-13", Title: "لا مفتاحَ لمصدر احتساب العمولة", Severity: "HIGH"},
	{ID: "XG-14", Title: "بوّابةُ platformCommission مثبَّتةٌ في الشيفرة", Severity: "CRITICAL"},
	{ID: "XG-15", Title: "عتبةُ التفعيل تُسقط الطلباتِ السابقة", Severity: "HIGH", WokenBy: "sales.activation_orders"},
	{ID: "XG-16", Title: "المندوبُ يرى عمولةَ المنصّة", Severity: "HIGH"},
	{ID: "XG-17", Title: "لا سجلَّ نقلِ متجرٍ بين مندوبين", Severity: "HIGH"},
	{ID: "XG-18", Title: "لا فحصَ ازدواجٍ ولا قيدَ فريدٍ للمرشَّحين", Severity: "BLOCKER"},
	{ID: "XG-19", Title: "لا مدخلَ دعمٍ للمندوب والتذكرةُ تشترط طلباً", Severity: "HIGH"},
	{ID: "XG-20", Title: "التدقيقُ الحسّاسُ خارجَ المعاملة", Severity: "CRITICAL"},
	{ID: "XG-21", Title: "سردُ /media/ مكشوفٌ وفيه إثباتُ التسليم", Severity: "BLOCKER"},
	{ID: "XG-22", Title: "التعليقُ العاديُّ يشلّ إتمامَ طلبٍ قائم", Severity: "BLOCKER"},
	{ID: "XG-23", Title: "لا مسارَ تدخّلٍ استثنائيٍّ مسمّى", Severity: "HIGH"},
	{ID: "XG-24", Title: "لا استعلامَ لطلباتٍ بيد موقوفين", Severity: "HIGH"},
	{ID: "XG-25", Title: "عمولةُ المنصّة تُحسب لحظةَ التسليم", Severity: "CRITICAL"},
	{ID: "XG-26", Title: "نسبةُ المندوب تُقرأ لحظةَ التسليم", Severity: "CRITICAL"},
	{ID: "XG-27", Title: "مصدرُ الاحتساب مثبَّتٌ ولا يُلتقَط", Severity: "CRITICAL"},
	{ID: "XG-28", Title: "عتبةُ التفعيل تُقرأ لحظةَ التسليم", Severity: "CRITICAL"},
	{ID: "XG-29", Title: "المكافأةُ تُدفع قبل تثبيت التحويل", Severity: "BLOCKER"},
	{ID: "XG-30", Title: "لا حالَ «مؤهَّلة» للمكافأة في النموذج", Severity: "HIGH"},
}

// ══════════════════════════════════════════════════════════════════════
// **استراتيجيّةُ تحقّقِ المخاطر**
// ══════════════════════════════════════════════════════════════════════
//
// **الخطرُ لا يصير عيباً بلا دليل** — **وهذا ما يُثبته أو ينفيه.**

// RiskStrategy الاختبارُ الذي يحسم كلَّ خطر.
var RiskStrategy = map[string]string{
	"R1":  "انتقالٌ يكتب الحالةَ خارجَ البوّابة — L3",
	"R2":  "جردُ المنادين آليّاً — L1",
	"R3":  "EnsureUserWithRole بلا موافقةٍ صريحة — L4",
	"R4":  "نداءُ الخزينة خارجَ معاملة — L3",
	"R5":  "PATCH stores/{id}/settings بلا تدقيق — L3",
	"R6":  "حِملُ قراءة الإعدادات تحت ضغط — L10",
	"R7":  "إسقاطٌ بين كتابات القبول الثلاث — L9",
	"R8":  "مفتاحٌ عالقٌ ثمّ تقليمُ ٢٤ ساعة — L4",
	"R9":  "إغراقُ بابٍ بلا حدِّ معدّل — L11",
	"R10": "سباقُ قبولٍ بحاجزٍ حقيقيّ — L9",
	"R11": "إسقاطُ إنشاء الطلب الخاصّ — L9",
	"R12": "قطعُ سياق أثناء تدقيقٍ بسياق الطلب — L9",
	"R13": "إعادةُ كلمةٍ إداريّةٍ ثمّ فحصُ الجلسة — L4",
	"R14": "إبطالٌ بعد مصافحة البثّ — L5",
	"R15": "سحبُ دورٍ ثمّ قياسُ مهلة السريان — L4",
	"R16": "إسقاطُ Redis ثمّ إبطالُ جلسة — L9",
	"R17": "إعادةُ إقلاعٍ ثمّ فحصُ سؤال الورديّة — L7",
	"R18": "انتهاءُ توكن أثناء رفعِ ملفّ — L4",
	"R19": "دفعتا مواقعَ متداخلتان — L7",
	"R20": "نقطةٌ بطابعٍ مستقبليّ — L1",
	"R21": "إسقاطُ واتساب بعد القبول — L9",
	"R22": "إسقاطُ الإشعار بعد وسم alerted_at — L9",
	"R23": "إسقاطُ FCM وقياسُ أثر الفعل — L9",
	"R24": "جردُ أنماط فشل التحويل السبعة — L4",
}

// ══════════════════════════════════════════════════════════════════════
// **ما يحرسه كلُّ اختبار — الربطُ اليدويُّ الوحيد**
// ══════════════════════════════════════════════════════════════════════
//
// **ويبدأ بما بناه `P-1`** — **ولا يُنسَخ منه شيءٌ**، بل يُشار إليه.

// TestDecl إعلانُ اختبارٍ واحد.
type TestDecl struct {
	Level    Level
	Purpose  Purpose
	Flows    []string
	Defects  []string
	Risks    []string
	Gaps     []string
	Settings []string
	Modes    []string
	Evidence []string
}

// TestMap **الاختباراتُ المعلَنة** — بمفتاح `اسمُ الدالّة`.
//
// **ويُملأ مرحلةً بعد مرحلة** — **و`P-1` أوّلُ ساكنيه.**
var TestMap = map[string]TestDecl{
	// ── `P-1` · عقدُ الخصوصيّة ────────────────────────────────────
	"TestOrderFieldsAllClassified": {
		Level: L5, Purpose: PurposeGenerator,
		Defects: []string{"D20", "D21", "D22", "D23"},
		Modes:   []string{"FAST", "FULL", "SECURITY", "RELEASE"},
	},
	"TestPrivacyContractHasNoGhosts": {
		Level: L5, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL"},
	},
	"TestEveryFieldCoversEveryRole": {
		Level: L5, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL"},
	},
	"TestUnknownFieldFailsClosed": {
		Level: L5, Purpose: PurposeHarnessSelf,
		Modes: []string{"FAST", "FULL", "SECURITY"},
	},
	"TestForbiddenFieldGuardCatchesLeak": {
		Level: L5, Purpose: PurposeHarnessSelf,
		Defects: []string{"D21"},
		Modes:   []string{"FAST", "FULL", "SECURITY"},
	},
	"TestD21_CustomerRedactionAgainstContract": {
		Level: L5, Flows: []string{"F-01", "F-14"},
		Defects:  []string{"D21", "D23"},
		Modes:    []string{"FULL", "SECURITY", "RELEASE"},
		Evidence: []string{"payload"},
	},
	"TestD20_MerchantRealtimeVsREST": {
		Level: L5, Flows: []string{"F-01", "F-04", "F-14"},
		Defects:  []string{"D20", "D23"},
		Modes:    []string{"FULL", "SECURITY", "RELEASE"},
		Evidence: []string{"payload"},
	},
	"TestD20_CustomerRealtimeVsREST": {
		Level: L5, Flows: []string{"F-01"},
		Defects:  []string{"D20"},
		Modes:    []string{"FULL", "SECURITY", "RELEASE"},
		Evidence: []string{"payload"},
	},
	"TestD22_CustomOrderOwnerChannelContract": {
		Level: L5, Flows: []string{"F-02"},
		Defects: []string{"D22"},
		Modes:   []string{"FULL", "CROSS_SYSTEM", "RELEASE"},
	},
	"TestPushPayloadScope": {
		Level: L5, Flows: []string{"F-01"},
		Modes: []string{"FULL", "SECURITY"},
	},

	// ── `P-2` · حارسُ الحقيقة ─────────────────────────────────────
	"TestTruthIsCurrent": {
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	},
	"TestEveryDefectIsKnown": {
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	},
	"TestEveryRiskHasStrategy": {
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL"},
	},
	"TestNoStaleReferences": {
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	},
	"TestAllFlowsDeclared": {
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL"},
	},
	"TestExtractorsAgreeWithSource": {
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL"},
	},
}

// ══════════════════════════════════════════════════════════════════════
// **الحزمُ التي اختباراتُها بنيةٌ تحتيّةٌ بطبيعتها**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا تُعدّ يتيمةً** — **لكنّها تُحصى وتُسمّى، فلا تختفي مئاتٌ بلا
// تفسير.** (البند ١٠ من طلب المالك.)

// InfraPackages حزمٌ غرضُها البنيةُ لا الميزة.
var InfraPackages = map[string]Purpose{
	"testdb":    PurposeInfrastructure,
	"httpx":     PurposeInfrastructure,
	"migrate":   PurposeInfrastructure,
	"testtruth": PurposeGenerator,
}
