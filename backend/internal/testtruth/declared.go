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

// Flows الخمسةُ والثلاثون.
var Flows = []FlowDecl{
	{ID: "F-01", Title: "إنشاءُ طلبٍ عاديّ", Actor: "customer", Apps: []string{"customer", "merchant", "admin"}, Money: true, Realtime: true, Partial: true, Severity: "BLOCKER", Defects: []string{"D4", "D20", "D23"}, Risks: []string{"R8"}, Gaps: []string{"XG-9"}},
	{ID: "F-02", Title: "إنشاءُ طلبٍ خاصّ", Actor: "customer", Apps: []string{"customer", "admin"}, Money: true, Partial: true, Severity: "BLOCKER", Defects: []string{"D6", "D8", "D9", "D22"}, Risks: []string{"R11"}},
	{ID: "F-03", Title: "ردُّ إنشاءٍ ضائع", Actor: "customer", Apps: []string{"customer"}, Money: true, Partial: true, Severity: "CRITICAL", Risks: []string{"R8"}},
	{ID: "F-04", Title: "قبولُ المتجر", Actor: "merchant", Apps: []string{"merchant", "customer"}, Realtime: true, Severity: "CRITICAL", Defects: []string{"D19", "D20"}, Risks: []string{"R21"}},
	{ID: "F-05", Title: "القبولُ التلقائيّ", Actor: "watchdog", Apps: []string{"merchant", "customer"}, Realtime: true, Partial: true, Severity: "CRITICAL", Gaps: []string{"XG-7"}},
	{ID: "F-06", Title: "رفضُ المتجر", Actor: "merchant", Apps: []string{"merchant", "customer", "admin"}, Money: true, Realtime: true, Severity: "HIGH"},
	{ID: "F-07", Title: "عرضُ الطلب على سائق", Actor: "engine", Apps: []string{"driver"}, Realtime: true, Partial: true, Race: true, Severity: "CRITICAL", Defects: []string{"D27"}, Risks: []string{"R10", "R23"}},
	{ID: "F-08", Title: "قبولُ السائق", Actor: "driver", Apps: []string{"driver", "customer", "merchant"}, Money: true, Realtime: true, Partial: true, Race: true, Severity: "BLOCKER", Defects: []string{"D7", "D24"}, Risks: []string{"R7", "R10"}},
	{ID: "F-09", Title: "انقضاءُ العرض ودورانُه", Actor: "watchdog", Apps: []string{"driver", "admin"}, Realtime: true, Partial: true, Race: true, Severity: "HIGH", Defects: []string{"D1", "D26"}, Risks: []string{"R1", "R22"}},
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
	{ID: "F-21", Title: "تسجيلُ مرشَّحٍ ثمّ تحويلُه", Actor: "rep", Apps: []string{"rep", "merchant", "admin"}, Money: true, Partial: true, Race: true, Severity: "BLOCKER", Defects: []string{"D2", "D25"}, Gaps: []string{"XG-18", "XG-29", "XG-30"}},
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

	// ══════════════════════════════════════════════════════════════
	// **`F-35` — خريطةُ العمليات**
	// ══════════════════════════════════════════════════════════════
	//
	// **وتدفّقٌ خامسٌ وثلاثون لا رابعٌ وثلاثون** (٢٠٢٦-٠٩-٠٦): **ميزةٌ
	// جديدةٌ عابرةٌ للأنظمة بُنيت بطلب المالك**، **ولا يُخفى عددٌ ليبقى
	// الرقمُ جميلاً.**
	//
	// **وتمسُّ الإدارةَ وحدَها في الواجهة** — **لكنّها تقرأ من الأربعة
	// كلِّها**: مواضعُ السائقين، ومتاجرُ المندوبين، وطلباتُ الزبائن.
	//
	// **و`XG-20` تمسُّها**: **كتاباتُها الحسّاسةُ تُدقَّق بنمط المنصّة
	// القائم — وهو `best-effort` خارجَ المعاملة.** **ولم يُصلَح نظامُ
	// التدقيق هنا** (خارجَ النطاق)، **والنقصُ معلَنٌ لا مخبوء.**
	{ID: "F-35", Title: "خريطةُ العمليات — قراءةُ الأرض وإدارةُ التغطية",
		Actor: "admin", Apps: []string{"admin"},
		Severity: "HIGH", Gaps: []string{"XG-20"}},
}

// ══════════════════════════════════════════════════════════════════════
// **فجواتُ العقد — وشدّتُها من `TQ-1`**
// ══════════════════════════════════════════════════════════════════════

// GapDecl فجوةٌ معلَنةٌ بشدّتها.
type GapDecl struct {
	ID       string
	Title    string
	Severity string
	// Fixed **دليلُ الإصلاح** — فارغٌ يعني «ما زالت قائمة».
	//
	// # ولماذا نصٌّ لا علَمٌ منطقيّ
	//
	// **`true` لا تقول من أصلحها ولا بأيّ دليل** — **وفجوةٌ تُغلَق
	// بعلَمٍ تُفتح بعلَمٍ**، ولا يبقى ما يُراجَع.
	//
	// **والنصُّ يُقرأ في البوّابة والتقرير** — فمن سأل «لماذا لم تعد
	// مانعة؟» وجد الجواب في مكانٍ واحد.
	//
	// **ولا تُملأ إلّا وحارسُ انحدارٍ دائمٌ يؤكّد العقدَ ويمرّ.**
	Fixed string
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
	{ID: "XG-10", Title: "لا عكسَ لعمولة المندوب عند الاسترداد", Severity: "BLOCKER",
		// **دورةُ إصلاحٍ ٢ · ٢٠٢٦-٠٩-٠٦ — وقبلها مصالحةُ عقدين.**
		Fixed: "**العقدُ النافذ `RQ-5`** (قرارُ المالك ٢٠٢٦-٠٩-٠٥): " +
			"`ORDER REVENUE REVERSED → RELATED REP COMMISSION REVERSED`. " +
			"**وكان في `TRUTH.md` قرارٌ أقدمُ (٢٠٢٦-٠٨-٠٣) يقول «تبقى " +
			"عمولتُه»** — **وسياقُه نزاعُ الطعام الفاسد لا قفلُ دفترِ " +
			"استرداد** — **فنُسخ وصُولحت الوثيقتان.** " +
			"و`reverseCommissions` صارت تعكس ما قُيّد للمندوب من الدفتر، " +
			"**وما عجز عنه رصيدُه يصير التزاماً في `users.commission_debt`** " +
			"يُقتطَع من أوّل عمولةٍ قادمة (`offsetRepDebt`) — **وهو ما " +
			"يوجبه العقد**: «تُعالَج التزاماً — ولا يُمحى التاريخ». " +
			"وثلاثةُ حرّاسٍ تؤكّده، و`FI-09.a` صار `PROVABLE_NOW`."},
	// **والوصفُ صُحّح في `P-5` بقرار المالك** — والجوهرُ لم يتبدّل.
	//
	// **المُثبَتُ فعلاً**: **المتجرُ** لا المندوب. `reverseCommissions`
	// تعكس مستحقَّ المتجر، **فإن كان قد سحبه أسقط `CHECK (balance >= 0)`
	// المعاملةَ كلَّها** ⇒ `409` والمستردُّ للزبون صفر.
	// **وحالُ المندوب تمرّ اليومَ بالعَرَض** لأنّ `XG-10` يعني ألّا عكسَ
	// لقيده أصلاً.
	//
	// **والعقدُ العامُّ يشمل الجميع**:
	//
	//	CUSTOMER REFUND ENTITLEMENT MUST NOT DEPEND ON CURRENT DOWNSTREAM ACTOR BALANCES
	{ID: "XG-11", Title: "حقُّ الاسترداد مشروطٌ برصيدِ مستفيدٍ تالٍ — والمتجرُ مُثبَت",
		Severity: "BLOCKER",
		// **دورةُ إصلاحٍ ١ · ٢٠٢٦-٠٩-٠٦.**
		Fixed: "**عكسُ مستحقّ المتجر صار جزئيّاً بالدَّين** — " +
			"`reverseCommissions` تأخذ ما تحتمله المحفظةُ وتُقيّد الباقيَ في " +
			"`merchants.debt`، **وهي الآليّةُ نفسُها المُثبَتةُ في مسار ردّ " +
			"البضاعة** (`goods.go`) — **فالمساران افترقا وأُعيدا.** " +
			"والحارسُ `TestFIN_XG11_RefundIndependentOfMerchantBalance` " +
			"يؤكّد العقدَ: كان `409` والمستردُّ صفراً، وصار الزبونُ يستردّ " +
			"كاملاً والمستحقُّ لا يتبخّر (عُكس + دُيّن = ما قُيّد). " +
			"**وحَكَمُ `P-4` يشهد ألّا خرقَ جديداً.**"},
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

	// ══════════════════════════════════════════════════════════════
	// **`XG-31` — التزامٌ بلا أثرٍ محاسبيّ**
	// ══════════════════════════════════════════════════════════════
	//
	// **كشفه تدقيقُ المالك لدورة الإصلاح ٢** (٢٠٢٦-٠٩-٠٦).
	//
	// **الدَّينُ رقمٌ جارٍ لا واقعة**: `merchants.debt` و
	// `users.commission_debt` عمودان يُزادان ويُنقَصان، **ولا سطرَ
	// يقول متى نشأ الالتزامُ ولا عن أيّ طلب.**
	//
	// **والتسويةُ متتبَّعة** — سطرٌ في الدفتر بمرجع الطلب الجديد
	// ونصٍّ يقول «اقتطاعُ التزام». **والنشأةُ ليست**: حين لا يحتمل
	// الرصيدُ شيئاً **لا يُكتب سطرٌ أصلاً** — يُزاد العمودُ وحدَه.
	//
	// **فمن رأى دَيناً قدرُه ألفٌ ومئتان لا يستطيع أن يقول من أين.**
	// **ورقمٌ لا يُفسَّر لا يُراجَع ولا يُنازَع فيه.**
	//
	// **وهي علّةٌ سابقةٌ للمتجر** (`goods.go` منذ ردّ البضاعة)
	// **وُرِّثت إلى المندوب في دورة الإصلاح ٢** — **والنمطُ نُقل بعلّته.**
	{ID: "XG-31", Title: "التزامُ متجرٍ أو مندوبٍ يُقيَّد رقماً بلا واقعةٍ تُفسّره",
		Severity: "CRITICAL",
		// **دورةُ إصلاحٍ ٣ · ٢٠٢٦-٠٩-٠٦.**
		Fixed: "**صار الالتزامُ واقعةً تُقيَّد** — `financial_obligations` " +
			"تحمل الطرفَ والمبلغَ الأصليَّ والسببَ والطلبَ والوقتَ والمنشئ، " +
			"و`obligation_settlements` تحمل كلَّ اقتطاعٍ بمقداره وطلبِه " +
			"والباقي بعده **وقيدِه في الدفتر**. " +
			"**ولم يُقَم عالمٌ محاسبيٌّ موازٍ**: `wallet_transactions` " +
			"دفترُ محفظةٍ يحرّك الرصيدَ حتماً، **والالتزامُ ليس حركةَ مال** " +
			"— فقيدُه فيه يُفسد كلَّ رصيدٍ ومصالحة. " +
			"**والعمودان `merchants.debt` و`users.commission_debt` صارا " +
			"صورةً محفوظة** لا تُكتب إلّا من `internal/obligations`، " +
			"**ويحرس تطابقَها مع الوقائع `FI-13`** بأربعة فحوص. " +
			"**والنشأةُ في معاملة الاسترداد نفسِها** فلا دَينٌ بلا أصل، " +
			"**وفهرسٌ فريدٌ في القاعدة يمنع نشأةً ثانيةً للطلب نفسِه** — " +
			"حارسٌ في المخطَّط لا في ترتيب الشيفرة. " +
			"**والأقدمُ يُسدَّد أوّلاً** — سياسةٌ مُعلَنة. " +
			"**وأُغلقت للمتجر والمندوب معاً** — والعلّةُ كانت في " +
			"`merchants.debt` منذ ردّ البضاعة. عشرةُ حرّاسٍ دائمين."},
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

	// ── `P-3` · مصنعُ البيانات ────────────────────────────────────
	//
	// **وهي بنيةٌ تحتيّةٌ لا تحرس ميزةً** — **فلا تُعدّ يتيمةً ولا
	// تُحسَب في تغطية التدفّقات.**
	"TestFactoryNamespaceIsDeterministic":         harness(),
	"TestFactoryNamespacesDoNotCollide":           harness(),
	"TestFactoryNamesCarryScenario":               harness(),
	"TestFactorySeedChangesOutput":                harness(),
	"TestFactoryClockIsFixed":                     harness(),
	"TestFactoryRejectsUnusedLedgerKinds":         harness(),
	"TestDatabaseAvailability":                    infra(),
	"TestFactoryBuildsUserStates":                 infra(),
	"TestFactoryDriverStates":                     infra(),
	"TestFactoryFinanciallyConsistent":            infra(),
	"TestUnsafeFixtureBreaksConsistencyOnPurpose": harness(),
	"TestFactoryMerchantAndRep":                   infra(),
	"TestFactoryCleansUpAfterItself":              infra(),

	// ── `P-3V` · إثباتاتُ القاعدة ─────────────────────────────────
	//
	// **وثلاثةٌ منها أمنٌ لا بنية**: **هويّةُ القاعدة والعزلُ والتنظيف** —
	// **لأنّها ما يمنع اختباراً من الكتابة حيث لا يجوز.**
	"TestDatabaseIdentityProof":                 security(),
	"TestDatabaseIsolationBetweenScenarios":     security(),
	"TestCleanupRemovesOnlyItsOwnScenario":      security(),
	"TestFinancialFixtureConsistencyOnDatabase": infra(),
	"TestCorruptFixtureIsExplicitOnly":          infra(),
	"TestParallelFactoryScenarios":              infra(),
	"TestMigrationsApplied":                     infra(),

	// **وحارسُ الإنتاج أمنٌ لا بنية** — **يمنع كتابةً في قاعدةٍ حيّة.**
	"TestProductionDatabaseGuard": {
		Level: L11, Purpose: PurposeHarnessSelf,
		Modes: []string{"FAST", "FULL", "SECURITY", "RELEASE"},
	},
	"TestGuardRejectsActualProductionTarget": {
		Level: L11, Purpose: PurposeHarnessSelf,
		Modes: []string{"FAST", "FULL", "SECURITY", "RELEASE"},
	},

	// ── عيّنةُ النقل — **تحرس ما كانت تحرسه أصولُها** ──────────────
	"TestSampleFactory_ValidToken": {
		Level: L4, Flows: []string{"F-30"},
		Risks: []string{"R15"}, Modes: []string{"FULL"},
	},
	"TestSampleFactory_SuspendedIsRefused": {
		Level: L4, Flows: []string{"F-29"},
		Defects: []string{"D14"}, Modes: []string{"FULL", "SECURITY"},
	},
	"TestSampleFactory_DriverOnShiftWithCash": {
		Level: L4, Flows: []string{"F-08", "F-27"},
		Defects: []string{"D7"}, Modes: []string{"FULL"},
	},
	"TestSampleFactory_ForeignRoleDenied": {
		Level: L11, Flows: []string{"F-30"},
		Modes: []string{"FULL", "SECURITY"},
	},
	"TestSampleFactory_WalletMatchesLedger": {
		Level: L3, Flows: []string{"F-25"},
		Modes: []string{"FULL", "FINANCIAL"},
	},

	// ── `P-4` · محرّكُ الثوابت الماليّة ───────────────────────────
	//
	// **ولا اختبارَ يتيمٌ هنا** (البند ٢٤): كلُّ واحدٍ مربوطٌ بتدفّقه
	// وبما يمسّه من السجلّات المجمَّدة وبالإعدادات التي يقودها.

	// المحرّكُ نفسُه — بلا قاعدة.
	"TestEveryCheckIsComplete":          harness(),
	"TestEveryFamilyHasChecks":          harness(),
	"TestSelectDefaultsToProvable":      harness(),
	"TestUnprovenChecksAreInert":        harness(),
	"TestEveryKindHasContract":          harness(),
	"TestCreatorSitesExist":             harness(),
	"TestChecksReferenceKnownFlows":     harness(),
	"TestSnapshotCounts":                harness(),
	"TestContractedKindsSQLIsGenerated": harness(),
	"TestKindDriftDetectsBothDirections": {
		Level: L1, Purpose: PurposeHarnessSelf,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	},

	// الأمانُ وحارسُ الأنواع.
	"TestFIN_DatabaseSafetyBeforeWrites": security(),
	"TestFIN_LedgerKindAllowlistGuard": {
		Level: L3, Purpose: PurposeInfrastructure,
		Modes: []string{"FULL", "FINANCIAL", "RELEASE"},
	},
	"TestFIN_EngineRunsOnRealDatabase": infra(),
	"TestFIN_LedgerKindGuardFailsOnNewKind": {
		Level: L3, Purpose: PurposeHarnessSelf,
		Modes: []string{"FULL", "FINANCIAL", "RELEASE"},
	},

	// حفظُ اقتصاد الطلب.
	"TestFIN_OrderEconomicConservation": {
		Level: L4, Flows: []string{"F-01", "F-12", "F-14"},
		Settings: []string{"merchants.commission_percent", "delivery.fee"},
		Modes:    []string{"FULL", "FINANCIAL", "RELEASE"},
	},
	"TestFIN_ConservationSurvivesRefund": {
		Level: L4, Flows: []string{"F-14", "F-15"},
		Modes: []string{"FULL", "FINANCIAL", "RELEASE"},
	},
	"TestFIN_MoneyTimingByStatus": {
		Level: L4, Flows: []string{"F-12", "F-13", "F-14", "F-23"},
		Settings: []string{"merchants.commission_percent", "sales.commission_percent",
			"pricing.margin_fixed", "delivery.fee", "sales.activation_orders"},
		Modes: []string{"FULL", "FINANCIAL"},
	},

	// اختباراتُ الحارسِ لنفسِه — **إفسادٌ مقصودٌ ثمّ تنظيف.**
	"TestFIN_CorruptionDetectionSelfTest": harness(),
	"TestFIN_MissingReferenceSelfTest":    harness(),
	"TestFIN_DuplicateEffectSelfTest":     harness(),

	// منعُ التكرار — بالتسلسل.
	"TestFIN_RepeatedRefundIsIdempotent": {
		Level: L4, Flows: []string{"F-15"},
		Modes: []string{"FULL", "FINANCIAL", "RELEASE"},
	},
	"TestFIN_RepeatedPayoutDecision": {
		Level: L4, Flows: []string{"F-24"},
		Settings: []string{"payouts.min_amount"},
		Modes:    []string{"FULL", "FINANCIAL", "RELEASE"},
	},

	// السقفُ النقديّ — `D7`.
	"TestFIN_CashExposureContract": {
		Level: L4, Flows: []string{"F-19", "F-27"},
		Defects:  []string{"D7"},
		Settings: []string{"drivers.cash_limit"},
		Modes:    []string{"FULL", "FINANCIAL"},
	},

	// المندوبُ والاسترداد — `XG-10` · `XG-11` · `XG-13`.
	"TestFIN_RepCommissionReversal": {
		Level: L4, Flows: []string{"F-15", "F-23"},
		Gaps:     []string{"XG-10"},
		Settings: []string{"sales.commission_percent", "sales.activation_orders"},
		Modes:    []string{"FULL", "FINANCIAL", "RELEASE"},
	},
	"TestFIN_RefundNotConditionedOnRepBalance": {
		Level: L4, Flows: []string{"F-15"},
		Gaps:     []string{"XG-10", "XG-11"},
		Settings: []string{"sales.commission_percent"},
		Modes:    []string{"FULL", "FINANCIAL", "RELEASE"},
	},
	"TestFIN_MerchantWithdrewThenRefund": {
		Level: L4, Flows: []string{"F-15", "F-24"},
		Gaps:  []string{"XG-11"},
		Modes: []string{"FULL", "FINANCIAL", "RELEASE"},
	},

	// مصدرُ العمولة واللقطة — `XG-13` · `XQ-2`.
	"TestFIN_CommissionSourceMatrix": {
		Level: L4, Flows: []string{"F-14", "F-23"},
		Gaps: []string{"XG-13"},
		Settings: []string{"merchants.commission_percent", "sales.commission_percent",
			"pricing.margin_fixed", "sales.activation_orders"},
		Modes: []string{"FULL", "FINANCIAL"},
	},
	"TestFIN_SnapshotVsLiveEconomics": {
		Level: L4, Flows: []string{"F-14", "F-23", "F-33"},
		Gaps: []string{"XG-13"},
		Settings: []string{"merchants.commission_percent", "sales.commission_percent",
			"pricing.margin_fixed", "delivery.fee"},
		Modes: []string{"FULL", "FINANCIAL", "RELEASE"},
	},

	// الخزينةُ والمصروفُ والترتيبُ والحدود — `D5` · `D2`.
	"TestFIN_ExpenseTreasuryInvariant": {
		Level: L4, Flows: []string{"F-26"},
		Defects: []string{"D5"},
		Modes:   []string{"FULL", "FINANCIAL", "RELEASE"},
	},
	"TestFIN_TargetRewardPrecedesCommit": {
		Level: L2, Flows: []string{"F-21"},
		Defects: []string{"D2"},
		Modes:   []string{"FAST", "FULL", "FINANCIAL"},
	},
	"TestFIN_TransactionBoundaries": {
		Level: L2, Flows: []string{"F-14", "F-21", "F-24", "F-26"},
		Defects: []string{"D2", "D5"},
		Modes:   []string{"FAST", "FULL", "FINANCIAL"},
	},

	// ── `P-5` · التزامنُ ومنعُ التكرار ───────────────────────────
	//
	// **والخريطةُ الآليّةُ في `internal/racemap`** — وحارسٌ فيها يسقط
	// إن أشارت إلى اختبارٍ لا وجودَ له.

	// مِسنَدُ التزامن يختبر نفسَه.
	"TestCONC_BarrierReleasesTogether": harness(),
	"TestCONC_ProbeMeasuresOverlap":    harness(),
	"TestCONC_TimeoutIsReportedNotHung": {
		Level: L1, Purpose: PurposeHarnessSelf,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	},
	"TestCONC_TrueOverlapProvenByDatabase": {
		Level: L3, Purpose: PurposeHarnessSelf,
		Modes: []string{"FULL", "CONCURRENCY", "RELEASE"},
	},
	"TestTenFlowsMapped": harness(),
	"TestEveryMappedTestExists": {
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	},
	"TestNoFalsePass": {
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	},

	// السباقاتُ على الطلب.
	"TestRACE_TwoDriversSameOrder": {
		Level: L5, Flows: []string{"F-07", "F-08"},
		Risks:    []string{"R10"},
		Settings: []string{"drivers.assignment_mode", "drivers.max_active_orders", "drivers.cash_limit"},
		Modes:    []string{"FULL", "CONCURRENCY", "RELEASE"},
	},
	"TestRACE_MaxActiveOrders": {
		Level: L5, Flows: []string{"F-08", "F-10"},
		// **`R10` بقي في سجلّ المخاطر للتتبّع** — `R10 → CONFIRMED → D24`.
		Defects:  []string{"D24"},
		Risks:    []string{"R10"},
		Settings: []string{"drivers.max_active_orders"},
		Modes:    []string{"FULL", "CONCURRENCY", "RELEASE"},
	},
	"TestRACE_AdminVsAppTransition": {
		Level: L5, Flows: []string{"F-19", "F-08"},
		Risks: []string{"R7"},
		Modes: []string{"FULL", "CONCURRENCY"},
	},
	"TestRACE_FinancialTruthAfterConcurrentDeliveries": {
		Level: L5, Flows: []string{"F-13", "F-14"},
		Settings: []string{"merchants.commission_percent", "delivery.fee"},
		Modes:    []string{"FULL", "CONCURRENCY", "FINANCIAL", "RELEASE"},
	},
	"TestRACE_SettingChangeDuringSettlement": {
		Level: L5, Flows: []string{"F-14", "F-33"},
		Settings: []string{"merchants.commission_percent"},
		Modes:    []string{"FULL", "CONCURRENCY", "FINANCIAL"},
	},
	"TestRACE_DuplicateLeadConversion": {
		Level: L5, Flows: []string{"F-21", "F-22"},
		// **`XG-18` يبقى فجوةَ عقدٍ حاجبة** — `XG-18 → D25`.
		Defects: []string{"D2", "D25"}, Gaps: []string{"XG-18"},
		Modes: []string{"FULL", "CONCURRENCY", "RELEASE"},
	},
	"TestRACE_DuplicatePayout": {
		Level: L5, Flows: []string{"F-24"},
		Settings: []string{"payouts.min_amount"},
		Modes:    []string{"FULL", "CONCURRENCY", "FINANCIAL", "RELEASE"},
	},
	"TestRACE_RefundVsPayout": {
		Level: L5, Flows: []string{"F-15", "F-24"},
		Gaps:  []string{"XG-10", "XG-11"},
		Modes: []string{"FULL", "CONCURRENCY", "FINANCIAL", "RELEASE"},
	},
	"TestRACE_SessionRevokedDuringRequest": {
		Level: L11, Flows: []string{"F-30"},
		Risks: []string{"R15", "R16"},
		Modes: []string{"FULL", "CONCURRENCY", "SECURITY"},
	},
	"TestRACE_LocationQueueIsClientSide": {
		Level: L5, Flows: []string{"F-11", "F-13"},
		Risks: []string{"R19", "R20"},
		Modes: []string{"FULL", "CONCURRENCY"},
	},

	// منعُ التكرار.
	"TestIDEM_SameKeySamePayloadSequential": {
		Level: L4, Flows: []string{"F-01", "F-03"},
		Risks: []string{"R8"}, Modes: []string{"FULL", "RELEASE"},
	},
	"TestIDEM_SameKeySamePayloadConcurrent": {
		Level: L5, Flows: []string{"F-01", "F-03"},
		Risks: []string{"R8"}, Modes: []string{"FULL", "CONCURRENCY", "RELEASE"},
	},
	"TestIDEM_SameKeyDifferentPayload": {
		Level: L4, Flows: []string{"F-01", "F-03"},
		Risks: []string{"R8"}, Modes: []string{"FULL", "SECURITY"},
	},
	"TestIDEM_InProgressOverlap": {
		Level: L5, Flows: []string{"F-03"},
		Risks: []string{"R8"}, Modes: []string{"FULL", "CONCURRENCY", "RELEASE"},
	},
	"TestIDEM_ErrorReleasesClaim": {
		Level: L4, Flows: []string{"F-03"},
		Risks: []string{"R8"}, Modes: []string{"FULL", "RELEASE"},
	},
	"TestIDEM_CommitThenLostResponse": {
		Level: L4, Flows: []string{"F-03"},
		Risks: []string{"R8"}, Modes: []string{"FULL", "RELEASE"},
	},
	"TestIDEM_CleanupAfterTTL": {
		Level: L4, Flows: []string{"F-03"},
		Risks: []string{"R8"}, Modes: []string{"FULL", "RELEASE"},
	},

	// ── الشائخُ — عقدٌ بُدِّل ──────────────────────────────────────
	//
	// **`TestRepTarget_GrantedOnDelivery` كُتب ٢٠٢٦-٠٨-٣٠** حين كان هدفُ
	// المندوب يعدّ **طلبات متاجره المسلَّمة**. **وبدّل المالكُ العقدَ في
	// اليوم التالي** (٢٠٢٦-٠٨-٣١، التزام `5da3dbed`): **الهدفُ يعدّ
	// العملاءَ المسجَّلين لا طلباتِهم.**
	//
	// **فحُذف النداءُ من مسار التسليم عمداً**، وفي موضعه تعليقٌ يقول:
	// «**ولا هدفَ للمندوب هنا — ومكانُه تبدّل**… ونداءٌ لا يغيّر شيئاً
	// أسوأُ من غيابه». **والاختبارُ لم يُحدَّث معه.**
	//
	// **فسقوطُه ليس عيبَ منتجٍ ولا يُجمَّد** — **هو اختبارٌ يقيس عقداً
	// أُلغي.** والحذفُ أو إعادةُ الصياغة قرارُ منتجٍ للمالك.
	"TestRepTarget_GrantedOnDelivery": {
		Level: L4, Purpose: PurposeStale, Flows: []string{"F-21", "F-23"},
		Modes: []string{},
	},

	// ── `P-6` · حقنُ الفشل المُحكَم ───────────────────────────────
	//
	// **والخريطةُ الآليّةُ في `internal/failmap`** — وحارسٌ فيها يسقط
	// إن أشارت إلى اختبارٍ لا وجودَ له.

	"TestFAIL_SelfTest_HitsOnlyTheNamedStep": harness(),
	"TestFAIL_SelfTest_ScopeDoesNotLeak":     harness(),
	"TestFAIL_SelfTest_CleanupLeavesNothing": harness(),

	"TestFAIL_D2_ConvertLeadPartialStates": {
		Level: L6, Flows: []string{"F-21", "F-22"},
		Defects: []string{"D2", "D25"}, Gaps: []string{"XG-18"},
		Settings: []string{"sales.target_reward"},
		Modes:    []string{"FULL", "FAILURE", "RELEASE"},
	},
	"TestFAIL_D5_ExpenseTreasuryPartial": {
		Level: L6, Flows: []string{"F-26"},
		Defects: []string{"D5"},
		Modes:   []string{"FULL", "FAILURE", "FINANCIAL", "RELEASE"},
	},
	"TestFAIL_D15_AdminCreateUserPartial": {
		Level: L6, Flows: []string{"F-30"},
		Defects: []string{"D15"},
		Modes:   []string{"FULL", "FAILURE"},
	},
	"TestFAIL_R7_DriverAcceptPartialState": {
		Level: L6, Flows: []string{"F-08"},
		Defects: []string{"D24"}, Risks: []string{"R7", "R10"},
		Modes: []string{"FULL", "FAILURE"},
	},
	"TestFAIL_R8_OrphanClaimUnderRealFailure": {
		Level: L6, Flows: []string{"F-03", "F-01"},
		Risks: []string{"R8"},
		Modes: []string{"FULL", "FAILURE", "RELEASE"},
	},
	"TestFAIL_R22_WatchdogMarkerBeforeNotify": {
		Level: L6, Flows: []string{"F-09"},
		Risks: []string{"R22"},
		Modes: []string{"FULL", "FAILURE"},
	},
	"TestFAIL_AQ4_AuditAtomicity": {
		Level: L6, Flows: []string{"F-25", "F-29", "F-33"},
		Modes: []string{"FULL", "FAILURE", "SECURITY", "RELEASE"},
	},
	"TestFAIL_XOB7_AutoTransferFireAndForget": {
		Level: L6, Flows: []string{"F-01", "F-18"},
		Risks: []string{"R21"},
		Modes: []string{"FULL", "FAILURE"},
	},
	"TestFAIL_P7SeamRequired_FCMTransport": {
		Level: L2, Flows: []string{"F-07", "F-14"},
		Risks: []string{"R23"},
		Modes: []string{"FAST", "FULL", "FAILURE"},
	},

	"TestNineFlowsMapped": harness(),

	// ── `P-7` · عقودُ البثّ والإشعار ──────────────────────────────
	//
	// **والمصفوفةُ الآليّةُ في `internal/eventmap`** — ومواضعُ الإطلاق
	// تُستخرَج من الشيفرة فلا رقمَ يُثبَّت بيد.

	"TestSeam_ProductionWiringUnchanged": {
		Level: L2, Purpose: PurposeHarnessSelf,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	},
	"TestEV_EmitSitesAndPublishers": {
		Level: L1, Purpose: PurposeGenerator, Modes: []string{"FAST", "FULL"},
	},
	"TestEV_ContractDriftGuard": {
		Level: L1, Purpose: PurposeGenerator, Modes: []string{"FAST", "FULL", "RELEASE"},
	},

	"TestEV_MerchantRealtimePrivacy": {
		Level: L7, Flows: []string{"F-01"},
		Defects: []string{"D20"},
		Modes:   []string{"FULL", "REALTIME", "SECURITY", "RELEASE"},
	},
	"TestEV_CustomOrderOwnerRealtime": {
		Level: L7, Flows: []string{"F-02"},
		Defects: []string{"D22"},
		Modes:   []string{"FULL", "REALTIME", "RELEASE"},
	},
	"TestEV_CustomerDriverAssignment": {
		Level: L7, Flows: []string{"F-08", "F-13"},
		Defects: []string{"D21", "D23"},
		Modes:   []string{"FULL", "REALTIME", "SECURITY", "RELEASE"},
	},
	"TestEV_RealtimeAuthorization": {
		Level: L11, Flows: []string{"F-34"},
		Risks: []string{"R14"},
		Modes: []string{"FULL", "REALTIME", "SECURITY", "RELEASE"},
	},
	"TestEV_MultiOrderRouting": {
		Level: L7, Flows: []string{"F-01"}, Modes: []string{"FULL", "REALTIME"},
	},
	"TestEV_OrderAudience": {
		Level: L7, Flows: []string{"F-01"},
		Modes: []string{"FULL", "REALTIME", "RELEASE"},
	},
	"TestEV_WalletAudience": {
		Level: L7, Flows: []string{"F-25"}, Modes: []string{"FULL", "REALTIME"},
	},
	"TestEV_MerchantDriverAssignment": {
		Level: L7, Flows: []string{"F-08", "F-19"},
		Defects: []string{"D20"},
		Modes:   []string{"FULL", "REALTIME", "SECURITY"},
	},
	"TestEV_AutoAcceptMerchantAwareness": {
		Level: L7, Flows: []string{"F-05"},
		Settings: []string{"orders.auto_accept_sec"},
		Modes:    []string{"FULL", "REALTIME"},
	},
	"TestEV_PushTokenTargeting": {
		Level: L7, Flows: []string{"F-07"},
		Defects: []string{"D12"},
		Modes:   []string{"FULL", "REALTIME", "SECURITY"},
	},
	"TestEV_R23PushFailureIsLost": {
		Level: L7, Flows: []string{"F-07", "F-14"},
		// **`R23` بقي للتتبّع** — `R23 → CONFIRMED → D27`.
		Defects: []string{"D27"},
		Risks:   []string{"R23"},
		Modes:   []string{"FULL", "REALTIME", "FAILURE", "RELEASE"},
	},
	"TestEV_R21AutoTransferAwareness": {
		Level: L7, Flows: []string{"F-18"},
		Risks:    []string{"R21"},
		Settings: []string{"orders.auto_transfer_amount"},
		Modes:    []string{"FULL", "REALTIME"},
	},
	"TestEV_R22WatchdogMarkerSuppressesRetry": {
		Level: L7, Flows: []string{"F-09"},
		// **`R22` بقي للتتبّع** — `R22 → CONFIRMED → D26` · و`XOB-6` دليلٌ فيه.
		Defects: []string{"D26"},
		Risks:   []string{"R22"},
		Modes:   []string{"FULL", "REALTIME", "FAILURE", "RELEASE"},
	},
	"TestEV_XOB5_WatchdogComparisonKey": {
		Level: L2, Flows: []string{"F-09"},
		Risks: []string{"R22"},
		Modes: []string{"FAST", "FULL"},
	},

	// ── مصالحةُ `D15` ─────────────────────────────────────────────
	"TestFAIL_D15_Reconciliation": {
		Level: L6, Flows: []string{"F-30"},
		Defects: []string{"D15"},
		Modes:   []string{"FULL", "FAILURE", "RELEASE"},
	},

	// ── `P-8` · أندرويد ──────────────────────────────────────────
	//
	// **والمصفوفتان الآليّتان في `internal/androidmap`.**
	// **واختباراتُ Kotlin لا يراها هذا المستخرِج** — يقرأ Go وحدَها،
	// **فتُحصى في `ANDROID_TEST_MATRIX.json` لا هنا.**
	"TestNoMissingDeviceCountsAsPass": harness(),
	"TestDeviceCasesAreNotClaimedRun": {
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	},
	"TestUniqueIDs": harness(),

	// ── `P-9` · محرّكُ أثر التغيير ────────────────────────────────
	//
	// **وقواعدُه في `CHANGE_IMPACT_RULES.json`** — ولا تعيش في Markdown.
	"TestHistoricalChangeCases": {
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	},
	"TestDefectImpactCases": {
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	},
	"TestNegativeControl":              harness(),
	"TestUnknownChangeFallsBackSafely": harness(),
	"TestStaleMappingGuard": {
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	},
	"TestWhySelectedTrace":      harness(),
	"TestRepeatabilityAndSpeed": harness(),
	"TestChangeInputModes":      harness(),
	"TestFileCategories":        harness(),

	// ── `P-10` · بوّابةُ الإطلاق بالدليل ─────────────────────────
	//
	// **وحكمُها `RELEASE`** — **بوّابةٌ لا تُشغَّل عند الإطلاق ليست بوّابة.**
	"TestGate_AllPassMeansYes":                      gateTest(),
	"TestGate_OneBlockerMeansNo":                    gateTest(),
	"TestGate_DeviceNotRunMeansNo":                  gateTest(),
	"TestGate_StagingNotRunMeansNo":                 gateTest(),
	"TestGate_StaleEvidenceMeansNo":                 gateTest(),
	"TestGate_HighNonBlockingAllowed":               gateTest(),
	"TestGate_NoDoubleCount":                        gateTest(),
	"TestGate_LatentGapPolicy":                      gateTest(),
	"TestGate_UnmappedCriticalCannotPassSilently":   gateTest(),
	"TestGate_WaiverRegistryEmptyAndPolicyEnforced": gateTest(),
	"TestGate_CurrentTruthProducesJudgment":         gateTest(),
	"TestGate_NoRuleWithoutContractOrEvidence":      gateTest(),
	"TestGate_NonPassStatesNeverCountAsPass":        gateTest(),
	"TestGate_ExitCodesDocumented":                  gateTest(),

	// ── `F-35` · خريطةُ العمليات ────────────────────────────────
	//
	// **والتصريحُ أوّلاً** — **صفحةٌ تقرأ مواضعَ الناس وأرقامَ المال
	// تُحرَس قبل أن تُعرَض.**
	"TestOpsMap_RequiresAdminPanelRole":              mapTest(L2, "SECURITY"),
	"TestOpsMap_MetaGrantsNamedPermissions":          mapTest(L2, "SECURITY"),
	"TestOpsMap_LayerPermissionsAreEnforcedPerLayer": mapTest(L2, "SECURITY"),
	"TestOpsMap_PrivacyMoneyHiddenWithoutPermission": mapTest(L2, "SECURITY"),
	"TestSearch_ScopedToPermissions":                 mapTest(L2, "SECURITY"),

	// ── الطبقاتُ والمشهد ────────────────────────────────────────
	"TestOpsMap_DriversLayerReturnsShape":                mapTest(L4, ""),
	"TestOpsMap_BadBBoxIsRejected":                       mapTest(L4, ""),
	"TestOpsMap_MerchantMarkerCarriesOperationalSummary": mapTest(L4, ""),
	"TestOpsMap_MerchantBBoxExcludesFarAway":             mapTest(L4, ""),
	"TestOpsMap_ActiveOrderMarkerAndRelations":           mapTest(L4, ""),
	"TestOpsMap_ClosedOrderIsNotActive":                  mapTest(L4, ""),

	// ── التغطية — **وأخطرُ ما في الميزة** ──────────────────────
	//
	// **`ZoneAt` تقرّر من تصله المنصّةُ أصلاً** — **وخطأٌ فيها يردّ
	// زبائنَ حقيقيّين.** فوضعُها `RELEASE` في كلّ الأوضاع.
	"TestCoverage_LegacyRadiusUnchanged":                  coverageTest(),
	"TestCoverage_PolygonInsideOutsideBoundary":           coverageTest(),
	"TestCoverage_DisabledZoneDoesNotServe":               coverageTest(),
	"TestCoverage_OverlappingZonesPickNearestCentre":      coverageTest(),
	"TestCoverage_MultipleZonesEachMeasuredByItsOwnShape": coverageTest(),
	"TestCoverage_EmptyTableStaysOpen":                    coverageTest(),
	"TestCoverage_GeometryValidationRejectsBadShapes":     coverageTest(),
	"TestCoverage_LegacyScreenNeverSeesPolygons":          coverageTest(),
	"TestCoverage_ManagePermissionRequired":               mapTest(L2, "SECURITY"),

	// ── طلباتُ التغطية ─────────────────────────────────────────
	"TestCoverageRequest_AnonymousMayAskAndLifecycleRuns": mapTest(L4, ""),
	"TestCoverageRequest_UnknownStatusRejected":           mapTest(L4, ""),
	"TestCoverageRequest_BadPointRejected":                mapTest(L4, ""),

	// ── الفروعُ والمناطقُ التشغيليّة ──────────────────────────
	"TestBranch_CityGetsOnePrimaryAndManySubs": mapTest(L4, ""),
	"TestBranch_HierarchyRulesEnforced":        mapTest(L4, ""),
	"TestBranch_DistrictNeverBecomesBranch":    mapTest(L4, ""),
	"TestArea_RelationsAndNotADistrict":        mapTest(L4, ""),
	"TestBranch_ManagePermissionRequired":      mapTest(L2, "SECURITY"),

	// ── المندوبون والتحليلات ──────────────────────────────────
	"TestRep_ActivityFromExistingDataOnly":         mapTest(L4, ""),
	"TestRep_MapNeverGatesConversion":              mapTest(L4, ""),
	"TestDemand_RequestCellCountsMatchSeed":        mapTest(L4, ""),
	"TestDemand_LayersAreSeparateNotMerged":        mapTest(L4, ""),
	"TestDemand_UnservedShrinksWhenCoverageDrawn":  mapTest(L4, ""),
	"TestDemand_TimeRangeFilters":                  mapTest(L4, ""),
	"TestOpportunity_ScoreIsExplainedNotPredicted": mapTest(L4, ""),

	// ── وحداتُ الحزمة ─────────────────────────────────────────
	"TestPerm_NoRoleNoMap":                     mapTest(L1, "SECURITY"),
	"TestPerm_RoleMatrix":                      mapTest(L1, "SECURITY"),
	"TestPerm_GrantedIsStableAndDeduped":       mapTest(L1, ""),
	"TestFreshness_DerivedNotInvented":         mapTest(L1, ""),
	"TestFreshness_FollowsSettingNotConstant":  mapTest(L1, ""),
	"TestFreshness_ZeroPingFallsBackNotPanics": mapTest(L1, ""),
	"TestBBox_RejectsNonsense":                 mapTest(L1, ""),
	"TestBBox_SQLUsesParametersNotLiterals":    mapTest(L1, "SECURITY"),

	// ── دورةُ إصلاحٍ ١ · `XG-11` ────────────────────────────────
	//
	// **حارسُ عقدٍ يؤكّد ولا يسجّل** — **ويسقط إن عاد العيبُ بعد سنة.**
	//
	// **ويُربَط بـ`F-15`** (الاسترداد) **وبـ`F-01`**: تبدُّلُ مسار
	// العكس يمسّ من يستطيع أن يطلب ويُستردَّ له.
	"TestFIN_XG11_RefundIndependentOfMerchantBalance": {
		Level: L4, Flows: []string{"F-15", "F-01"},
		Gaps:  []string{"XG-11"},
		Modes: []string{"FULL", "FINANCIAL", "CROSS_SYSTEM", "RELEASE"},
	},
	"TestFIN_XG11_RefundUntouchedWhenMerchantSolvent": {
		Level: L4, Flows: []string{"F-15"},
		Gaps:  []string{"XG-11"},
		Modes: []string{"FULL", "FINANCIAL", "RELEASE"},
	},

	// ── دورةُ إصلاحٍ ٢ · `XG-10` ────────────────────────────────
	//
	// **ثلاثةُ حرّاسٍ لعقدٍ ذي شقّين**: **العكسُ** حين يملك،
	// **والالتزامُ** حين سحب، **والتسويةُ** من عمولةٍ قادمة.
	"TestFIN_XG10_RepCommissionReversedOnRefund": {
		Level: L4, Flows: []string{"F-15", "F-23"},
		Gaps:     []string{"XG-10"},
		Settings: []string{"sales.commission_percent"},
		Modes:    []string{"FULL", "FINANCIAL", "CROSS_SYSTEM", "RELEASE"},
	},
	"TestFIN_XG10_RepWithdrewThenRefund": {
		Level: L4, Flows: []string{"F-15", "F-23"},
		Gaps:     []string{"XG-10", "XG-11"},
		Settings: []string{"sales.commission_percent"},
		Modes:    []string{"FULL", "FINANCIAL", "CROSS_SYSTEM", "RELEASE"},
	},
	"TestFIN_XG10_DebtOffsetFromNextCommission": {
		Level: L4, Flows: []string{"F-23"},
		Gaps:     []string{"XG-10"},
		Settings: []string{"sales.commission_percent"},
		Modes:    []string{"FULL", "FINANCIAL", "RELEASE"},
	},

	// ── `P-0` · بيئةُ التجهيز ──────────────────────────────────
	//
	// **وحرّاسُ البيئة ليست اختباراتِ ميزة** — **هي ما يمنع أمراً
	// هدّاماً أن يمسّ الإنتاج**، فوضعُها `RELEASE` في كلّ وضع.
	"TestGuard_CleanStagingPasses":                    infraTest(),
	"TestGuard_EachGuardBlocksAlone":                  infraTest(),
	"TestGuard_ProductionSubdomainsBlocked":           infraTest(),
	"TestGuard_NoBypassExists":                        infraTest(),
	"TestGuard_MustBeSafeNamesEveryFailure":           infraTest(),
	"TestIdentity_CarriesNoSecret":                    infraTest(),
	"TestIdentity_DefaultsToDevelopmentNotProduction": infraTest(),
	"TestStagingConfigNeverNamesProduction":           infraTest(),
	"TestStagingComposeIsFullyIsolated":               infraTest(),
	"TestStagingProvidersAreSilentByDefault":          infraTest(),
	"TestNoSecretsCommitted":                          infraTest(),

	// ── وتجاربُ التكامل على خدماتٍ حقيقيّة ─────────────────────
	//
	// **وتتخطّى بهدوءٍ بلا بيئة تجهيز** — ولا تُحمَّر الحزمةُ لسببٍ بيئيّ.
	"TestStaging_R14_SessionLifecycleServerSide": stagingTest([]string{"R14"}, nil),
	"TestStaging_SecurityBaseline":               stagingTest(nil, nil),
	// **وهذان يوثّقان ما تأكّد** — **وينجحان ما دام العيبُ قائماً،
	// ويسقطان يومَ يُصلَح** فيُقرأ سقوطُهما أمراً بتحديث السجلّ.
	"TestFAIL_R16_RedisDownFailsOpen":        stagingTest([]string{"R16"}, nil),
	"TestFAIL_D13_MediaDirectoryListingOpen": stagingTest(nil, []string{"D13"}),

	// **والبوّابةُ لا تُوسّخ ما تقيسه** — نظافةُ بنيةٍ لا إصلاحُ منتَج.
	"TestGateRunLeavesTreeUnchanged": infraTest(),

	// ── ذرّيّةُ عمليّات الإدارة (دورةُ إصلاحٍ ٤) ────────────────────
	//
	// **ثلاثُ عمليّاتٍ كانت تكتب مرّاتٍ بلا معاملة** — `PF-01` · `PF-02`
	// · `PF-03`. **وهذه تسقط ما دام العطبُ قائماً**، بخلاف حرّاسِ
	// `TestFAIL_*` التي توثّقه بالسجلّ.
	"TestATOMIC_ExpenseAndTreasuryAreOneUnit": atomicTest([]string{"D5"}),
	"TestATOMIC_LeadConversionIsOneUnit":      atomicTest([]string{"D2", "D25"}),
	"TestATOMIC_AdminUserCreationIsOneUnit":   atomicTest([]string{"D15"}),

	// ── `XG-31` · تتبّعُ الالتزامات (دورةُ إصلاحٍ ٣) ──────────────
	//
	// **عشرةُ حرّاسٍ يسألون سؤالاً واحداً**: **من أين جاء هذا الرقم؟**
	"TestOBL_MerchantSufficient_NoObligation":      oblTest(),
	"TestOBL_MerchantInsufficient_OriginTraceable": oblTest(),
	"TestOBL_RepAvailable_NoObligation":            oblTest(),
	"TestOBL_RepWithdrawn_OriginTraceable":         oblTest(),
	"TestOBL_BothInsufficient_AtomicOrigins":       oblTest(),
	"TestOBL_ReplayCreatesNoDuplicate":             oblTest(),
	"TestOBL_FutureEarningsSettleWithEvidence":     oblTest(),
	"TestOBL_MultipleObligationsFIFO":              oblTest(),

	// **وتدقيقُ دورةِ ٢ يحرس اقتصادَ العكس نفسَه.**
	"TestFIN_XG10_RefundReplayDoesNotDoubleCharge":     xg10Test(),
	"TestFIN_XG10_DebtSettlementArithmetic":            xg10Test(),
	"TestFIN_XG10_CombinedMerchantAndRepInsufficiency": xg10Test(),
	"TestFIN_XG10_ConservationAcrossRefund":            xg10Test(),
}

// atomicTest حارسُ ذرّيّةِ عمليّةٍ إداريّة — `PF-01` · `PF-02` · `PF-03`.
//
// **والوضعُ `FAILURE` أصلُه**: **لا يُثبَت إلّا بحقنِ عطبٍ في منتصف
// العمليّة** — **ونداءٌ ناجحٌ لا يقول شيئاً عن الذرّيّة.**
func atomicTest(defects []string) TestDecl {
	return TestDecl{
		Level: L4, Purpose: PurposeFeature,
		Flows:   []string{"F-21", "F-26", "F-30"},
		Defects: defects,
		Gaps:    []string{"XG-18"},
		Modes:   []string{"FAILURE", "FINANCIAL", "FULL", "RELEASE"},
	}
}

// oblTest حارسُ تتبّعِ التزامٍ ماليّ — `XG-31`.
//
// **ويمسّ تدفّقَي التسوية والاسترداد** — **والمالُ فيهما لا يُقرأ من
// عمودٍ بل من واقعة.**
func oblTest() TestDecl {
	return TestDecl{
		Level: L4, Purpose: PurposeFeature,
		Flows: []string{"F-14", "F-17"},
		Gaps:  []string{"XG-31"},
		Modes: []string{"FINANCIAL", "FULL", "RELEASE"},
	}
}

// xg10Test حارسُ عكسِ عمولةِ المندوب — `XG-10` · `XG-31`.
func xg10Test() TestDecl {
	return TestDecl{
		Level: L4, Purpose: PurposeFeature,
		Flows: []string{"F-14", "F-17"},
		Gaps:  []string{"XG-10", "XG-31"},
		Modes: []string{"FINANCIAL", "FULL", "RELEASE"},
	}
}

// infraTest حارسُ بنيةٍ تحتيّة — **يعمل في كلّ وضعٍ بلا بيئةٍ خارجيّة.**
func infraTest() TestDecl {
	return TestDecl{
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL", "SECURITY", "RELEASE"},
	}
}

// stagingTest تجربةُ تكاملٍ تحتاج بيئةَ تجهيزٍ حيّة.
//
// **ولا تُعَدُّ يتيمةً لأنّها تتخطّى** — **الربطُ بالسجلّ يبقى.**
func stagingTest(risks, defects []string) TestDecl {
	return TestDecl{
		Level: L9, Risks: risks, Defects: defects,
		Modes: []string{"FULL", "SECURITY", "RELEASE"},
	}
}

// mapTest اختبارُ خريطةِ عمليات — **مربوطٌ بـ`F-35`.**
func mapTest(level Level, extra string) TestDecl {
	modes := []string{"FULL"}
	if extra != "" {
		modes = append(modes, extra)
	}
	return TestDecl{Level: level, Flows: []string{"F-35"}, Modes: modes}
}

// coverageTest **انحدارُ خدمةِ التغطية** — ويمسُّ إنشاءَ الطلب نفسَه.
//
// **ولذلك يُربَط بـ`F-01`** أيضاً: **تبدُّلُ `ZoneAt` يبدّل من يستطيع
// أن يطلب أصلاً**، **ومن ربطه بالخريطة وحدَها أخفى أثرَه في المحرّك.**
func coverageTest() TestDecl {
	return TestDecl{
		Level: L4, Flows: []string{"F-35", "F-01"},
		Modes: []string{"FULL", "CROSS_SYSTEM", "RELEASE"},
	}
}

// gateTest فحصُ بوّابةٍ ذاتيّ — **يُشغَّل في كلّ وضعٍ إلزاميّ.**
func gateTest() TestDecl {
	return TestDecl{
		Level: L1, Purpose: PurposeGenerator,
		Modes: []string{"FAST", "FULL", "RELEASE"},
	}
}

// harness اختبارٌ يُثبت المِسنَدَ نفسَه — **لا يحرس ميزة.**
func harness() TestDecl {
	return TestDecl{Level: L1, Purpose: PurposeHarnessSelf, Modes: []string{"FAST", "FULL"}}
}

// security حارسٌ يمنع كتابةً حيث لا يجوز — **أمنٌ لا بنية.**
func security() TestDecl {
	return TestDecl{Level: L11, Purpose: PurposeHarnessSelf,
		Modes: []string{"FULL", "SECURITY", "RELEASE"}}
}

// infra اختبارُ بنيةٍ تحتيّةٍ يحتاج قاعدةً.
func infra() TestDecl {
	return TestDecl{Level: L3, Purpose: PurposeInfrastructure, Modes: []string{"FULL"}}
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
