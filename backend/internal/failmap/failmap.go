// Package failmap خريطةُ تدفّقات الفشل الجزئيّ — **`P-6`.**
//
// **ولا Markdown مصدراً وحيداً** (`P-6` التوثيق): `P-9` يسأل «أيُّ نقطةِ
// فشلٍ يمسّها هذا الملفّ؟» و`P-10` يسأل «أكلُّ تدفّقٍ جزئيٍّ حرِجٍ له
// اختبارٌ شُغّل؟» — **وسؤالان كهذان لا يُجابان من نصٍّ منسَّق.**
package failmap

import (
	"sort"
	"strings"
)

// Where أين يُثبَت.
type Where string

const (
	Local      Where = "LOCAL"
	RequiresP0 Where = "REQUIRES_P0"
	RequiresP7 Where = "REQUIRES_P7"
	RequiresP8 Where = "REQUIRES_P8"
)

// Result ما قِيس — **ولا `PASS` زور.**
type Result string

const (
	Pass             Result = "PASS"
	DefectReproduced Result = "DEFECT_REPRODUCED"
	RiskConfirmed    Result = "RISK_CONFIRMED"
	ExpectedFail     Result = "EXPECTED_FAIL"
	Partial          Result = "PARTIAL"
	Unproven         Result = "UNPROVEN"
)

// Visibility أترى العمليّاتُ الحالَ الجزئيّةَ؟ (البند ٢٤)
type Visibility string

const (
	Visible   Visibility = "VISIBLE"
	Invisible Visibility = "INVISIBLE"
	PartialV  Visibility = "PARTIAL"
)

// Flow تدفّقٌ ذو فشلٍ جزئيّ.
type Flow struct {
	ID string `json:"id"`
	// Flows تدفّقاتُ المنظومة التي يمسّها.
	Flows []string `json:"flows"`
	Title string   `json:"title"`
	// Steps خطواتُه كما قِيست من الشيفرة.
	Steps []string `json:"steps"`
	// AtomicBoundary حدُّ المعاملة الفعليّ — من `P-4`.
	AtomicBoundary string `json:"atomic_boundary"`
	// Failpoints نقاطُ الفشل المسمّاة.
	Failpoints []string `json:"failpoints"`
	// Expected ما يجب أن يبقى وما يجب أن يُرجَع.
	Expected string `json:"expected"`
	// Observed ما بقي فعلاً.
	Observed string `json:"observed"`
	// FinInv ثوابتُ `P-4` التي شُغّلت بعد الفشل.
	FinInv []string `json:"financial_invariants,omitempty"`
	// UserVisible ما يراه صاحبُ الفعل.
	UserVisible string `json:"user_visible"`
	// AdminVisible أترى العمليّاتُ الحالَ الجزئيّة.
	AdminVisible Visibility `json:"admin_visible"`
	// Recovery ما يقع عند الإعادة.
	Recovery string `json:"recovery"`
	// Tests الاختباراتُ التي تحرسه.
	Tests []string `json:"tests"`
	// Result النتيجة.
	Result Result `json:"result"`
	// Registers ما يمسّه من السجلّات المجمَّدة.
	Registers []string `json:"registers,omitempty"`
	// Where أين يُثبَت.
	Where Where `json:"where"`
	// Evidence **سطرُ الدليل كما طُبع في التشغيل** — ولا نتيجةَ بلا دليل.
	//
	// **وأوّلُ صيغةٍ لهذه الخريطة كُتبت قبل التشغيل** فأعلنت ثلاثَ نتائجَ
	// تخالف ما قِيس (`PF-03` و`PF-04` و`PF-05`). **وخريطةٌ تسبق القياسَ
	// تكذب** — فصار الدليلُ حقلاً إلزاميّاً يحرسه اختبار.
	Evidence string `json:"evidence"`
}

// All التدفّقاتُ التسعة — **`PARTIAL-FAILURE FLOWS = 9`.**
var All = []Flow{
	{
		ID: "PF-01", Title: "تحويلُ مرشَّحٍ إلى متجر",
		Flows: []string{"F-21"}, Where: Local, Result: Pass,
		Steps: []string{
			"١ CreateMerchant — خطؤها يُردّ",
			"٢ UPDATE users full_name — خطؤها مُهمَل",
			"٣ UPDATE users password_hash — خطؤها مُهمَل",
			"٤ UPDATE merchants district_id — خطؤها مُهمَل",
			"٥ grantSalesTargetIfAny — **تدفع مالاً** · خطؤها مُهمَل",
			"٦ UPDATE merchant_leads converted — **التثبيت** · خطؤه يُردّ",
		},
		AtomicBoundary: "ATOMIC — ستُّ كتاباتٍ في معاملةٍ واحدة (دورةُ إصلاحٍ ٤)",
		Failpoints:     []string{"D2/step-1-create-merchant", "D2/step-6-commit-conversion"},
		Expected:       "لا مكافأةَ قبل تثبيتِ التحويل (XQ-3)",
		Observed:       "لا أثرَ بعد الحقن — ولا متجرَ ولا مكافأةَ والمرشَّحُ new",
		FinInv:         []string{"FI-03", "FI-05"},
		UserVisible:    "500 internal — بلا بيانٍ لِما وقع",
		AdminVisible:   PartialV,
		Recovery:       "RETRY SAFETY = RECOVERS — الإعادةُ تثبّت التحويلَ بلا متجرٍ ثانٍ",
		Tests:          []string{"TestFAIL_D2_ConvertLeadPartialStates"},
		Registers:      []string{"D2", "D25", "XG-18"},
		Evidence:       "XQ-3 FORBIDDEN STATE = ABSENT — متاجرُ=0 · مكافأةٌ=0 · المرشَّحُ new · الردّ 500 · TestATOMIC_LeadConversionIsOneUnit",
	},
	{
		ID: "PF-02", Title: "مصروفُ تشغيلٍ وخصمُ خزينة",
		Flows: []string{"F-26"}, Where: Local, Result: Pass,
		Steps: []string{
			"١ INSERT INTO expenses — يُردّ خطؤه",
			"٢ wallet.Apply(operating_expense) — يُردّ خطؤه بعد وقوع الأولى",
		},
		AtomicBoundary: "ATOMIC — الكتابتان في معاملةٍ واحدة (دورةُ إصلاحٍ ٤)",
		Failpoints:     []string{"D5/step-2-treasury-debit"},
		Expected:       "المصروفُ وقيدُه يقعان معاً أو لا يقعان (FI-12.c)",
		Observed:       "لا مصروفَ ولا قيد — والعمليّةُ ارتدّت كاملةً",
		FinInv:         []string{"FI-04.d", "FI-01.f", "FI-12.b"},
		UserVisible:    "500 internal — والمصروفُ باقٍ في القائمة",
		AdminVisible:   PartialV,
		Recovery:       "CLEAN — الإعادةُ تُنشئ مصروفاً وخصمَه، ولا يتيمَ قبله",
		Tests:          []string{"TestFAIL_D5_ExpenseTreasuryPartial", "TestFIN_ExpenseTreasuryInvariant"},
		Registers:      []string{"D5"},
		Evidence:       "مصاريفُ=0 · قيودٌ=0 · وFI-04.d صامتٌ إذ لا خرق · TestATOMIC_ExpenseAndTreasuryAreOneUnit",
	},
	{
		ID: "PF-03", Title: "إنشاءُ مستخدمٍ إداريّاً",
		Flows: []string{"F-30"}, Where: Local, Result: Pass,
		Steps: []string{
			"١ CreateUserWithRole — مستخدمٌ + الدورُ الأوّلُ في **معاملةٍ واحدة**",
			"٢ GrantRole لبقيّة الأدوار — ولا تُنادى إن كان الدورُ واحداً",
			"٣ SetTempPassword — **خارجَ أيّ معاملة**",
			"٤ Audit",
		},
		AtomicBoundary: "ATOMIC — الإنشاءُ والأدوارُ والكلمةُ في معاملةٍ واحدة (دورةُ إصلاحٍ ٤)",
		Failpoints:     []string{"D15/set-temp-password", "D15/grant-second-role", "D15/create-user"},
		Expected:       "مستخدمٌ بلا دورٍ لا يُترَك",
		Observed:       "لا مستخدِمَ ولا دور — والرقمُ يبقى متاحاً",
		UserVisible:    "خطأٌ — والهاتفُ صار محجوزاً",
		AdminVisible:   PartialV,
		Recovery:       "CLEAN — الرقمُ حرٌّ والإعادةُ تُنشئ حساباً كاملاً",
		Tests:          []string{"TestFAIL_D15_Reconciliation", "TestFAIL_D15_AdminCreateUserPartial"},
		Registers:      []string{"D15"},
		Evidence:       "مستخدمون=0 · أدوارٌ=0 — لم يبقَ شيءٌ بعد فشل منح الدور",
	},
	{
		ID: "PF-04", Title: "قبولُ السائق",
		Flows: []string{"F-08"}, Where: Local, Result: Pass,
		Steps: []string{
			"١ UPDATE orders SET driver_id — شرطيٌّ ذرّيّ",
			"٢ UPDATE users last_assigned_at — خطؤه مُهمَل",
			"٣ Transition(assigned) — حدثٌ وبثّ",
		},
		AtomicBoundary: "NON_ATOMIC — ثلاثُ كتاباتٍ بلا معاملة",
		Failpoints:     []string{"R7/step-3-order-event"},
		Expected:       "المِلكيّةُ والحالُ متطابقتان",
		Observed:       "لم يُملَّك الطلبُ · والحالُ dispatching — لا حالَ مكسورة",
		FinInv:         []string{"FI-06", "FI-05"},
		UserVisible:    "خطأٌ للسائق",
		AdminVisible:   PartialV,
		Recovery:       "الإعادةُ نجحت وصار assigned — RECOVERS",
		Tests:          []string{"TestFAIL_R7_DriverAcceptPartialState"},
		Registers:      []string{"R7", "R10", "D24"},
		Evidence:       "مملوكٌ=false · الحالُ dispatching · والإعادةُ نجحت وصار assigned",
	},
	{
		ID: "PF-05", Title: "مطالبةُ منعِ التكرار",
		Flows: []string{"F-03", "F-01"}, Where: Local, Result: Pass,
		Steps: []string{
			"١ claimIdempotency — INSERT ON CONFLICT",
			"٢ المعالجُ يعمل",
			"٣ storeIdempotency أو releaseIdempotency عند ردٍّ ≥400",
		},
		AtomicBoundary: "NON_ATOMIC — مطالبةٌ ثمّ عملٌ ثمّ ختمٌ في ثلاث",
		Failpoints:     []string{"R8/order-insert"},
		Expected:       "المطالبةُ اليتيمةُ لا تحجب صاحبَها إلى الأبد",
		Observed:       "الخطأُ المُعادُ يُطلق المفتاح · وموتُ العمليّة (P-5) يتركه محجوزاً 24h",
		UserVisible:    "409 in_progress عند الإعادة",
		AdminVisible:   Invisible,
		Recovery:       "الإعادةُ نجحت — والتقليمُ لحال موتِ العمليّة وحدَها",
		Tests:          []string{"TestFAIL_R8_OrphanClaimUnderRealFailure", "TestIDEM_CommitThenLostResponse"},
		Registers:      []string{"R8"},
		Evidence:       "المفتاحُ أُطلق بالخطأ المُعاد · والإعادةُ نجحت · وطلبٌ واحد",
	},
	{
		ID: "PF-06", Title: "قيدُ التدقيق للأفعال الحسّاسة",
		Flows: []string{"F-25", "F-29", "F-33"}, Where: Local, Result: Pass,
		Steps: []string{
			"١ الفعلُ الحسّاسُ يقع (مالٌ · تعليقٌ · إعداد)",
			"٢ audit() في الخلفيّة — **ولا تُفشل الفعل**",
		},
		AtomicBoundary: "ATOMIC للصنف `A` — الفعلُ وأثرُه في معاملةٍ " +
			"واحدة (دورةُ إصلاحٍ ١٣) · وما سواه أفضلُ جهدٍ عمداً",
		Failpoints: []string{"AQ-4/audit-write"},
		Expected:   "CRITICAL SUCCESS REQUIRES AUDIT SUCCESS (AQ-4)",
		Observed: "**الفعلُ يرتدّ مع أثرِه**: الردُّ `500` · رصيدٌ `0` · " +
			"قيودُ تدقيقٍ `0` — **ولا يُقال «تمّ» لما لم يتمّ**",
		UserVisible:  "خطأٌ صريح — والعمليّةُ لم تقع",
		AdminVisible: Invisible,
		Recovery: "الإعادةُ تُنتج فعلاً واحداً وأثراً واحداً — " +
			"**ولا أثرَ يتيمٌ ولا مالٌ بلا أثر**",
		Tests:     []string{"TestFAIL_AQ4_AuditAtomicity"},
		Registers: []string{"AQ-4"},
		Evidence:  "الردّ 200 · رصيدٌ=12000 · قيودُ التدقيق 0 ⇒ 0",
	},
	{
		ID: "PF-07", Title: "إنذارُ الراصد",
		Flows: []string{"F-09"}, Where: Local, Result: Pass,
		Steps: []string{
			"١ UPDATE orders SET alerted_at — الوسم **داخلَ معاملة**",
			"٢ RowsAffected == 0 ⇒ أُنذر سابقاً — الحارس",
			"٣ NotifyOpsTx — **النيّةُ الدائمةُ في المعاملة نفسِها**",
			"٤ COMMIT واحد · ثمّ البثُّ بعده",
		},
		AtomicBoundary: "ATOMIC — الوسمُ والنيّةُ في معاملةٍ واحدة " +
			"(دورةُ إصلاحٍ ١٤)",
		Failpoints: []string{"R22/notify-write"},
		Expected:   "لا وسمَ قبل نجاح الإشعار",
		Observed: "**سقوطُ النيّة يُسقط الوسم**: موسومٌ=false · نيّاتٌ=0 " +
			"· والجولةُ التاليةُ تُنذر",
		UserVisible:  "لا شيء — إنذارٌ داخليّ",
		AdminVisible: Visible,
		Recovery: "الجولةُ التالية (٣٠ ثانية) — **والوسمُ لا يمنع إلّا " +
			"إنذاراً وقع فعلاً**",
		Tests: []string{
			"TestFAIL_R22_WatchdogMarkerBeforeNotify",
			"TestR22_W1_AlertMarksAndCreatesIntent",
			"TestR22_W2_IntentFailureLeavesNoMarker",
			"TestR22_W5_SecondSweepDoesNotDuplicate",
			"TestR22_W6_TwoWatchdogsAlertOnce",
			"TestR22_W9_NoOpsRecipientLeavesNoMarker",
		},
		Registers: []string{"R22"},
		Evidence: "**والمصفوفةُ كانت تقول `TESTABILITY SEAM REQUIRED`** — " +
			"**فأُضيف مِعراضٌ** (`EscalateAlertsOnce`) بلا نقطةِ شبكة، " +
			"**وصُحّح النصُّ.** " +
			"**والمقيس**: إنذارٌ مستحقٌّ ⇒ موسومٌ ونيّةٌ لكلّ مستقبِل · " +
			"وحقنُ سقوطِ `INSERT notifications` ⇒ **موسومٌ=false · " +
			"نيّاتٌ=0** والجولةُ التاليةُ تُنذر · وجولتان ⇒ لا تكرار · " +
			"وراصدان متزامنان (تداخلٌ مقيسٌ=2) ⇒ نيّاتٌ = عددُ " +
			"المستقبِلين لا ضِعفُه · **وصفرُ مستقبِلين ⇒ لا وسمَ ولا " +
			"نيّة** (`W9`)، **ووسمٌ بلا مُنذَرٍ كذبٌ يُسكت الطلبَ أبداً**. " +
			"**وعقدُ التسليم إلى هاتفٍ ليس هذا** — وهو `PF-09`.",
	},
	{
		ID: "PF-08", Title: "التحويلُ التلقائيُّ في خيطٍ منفصل",
		Flows: []string{"F-01", "F-18"}, Where: Local, Result: Pass,
		Steps: []string{
			"١ الردُّ للزبون يعود",
			"٢ go autoTransfer(...) — **لا يُنتظَر ولا يُرصَد**",
		},
		AtomicBoundary: "NON_ATOMIC — خارجَ دورة الطلب عمداً · " +
			"**ومصدرُ العمل حالُ الطلب الدائمة** (دورةُ إصلاحٍ ١٢)",
		Failpoints: []string{"XOB-7/auto-transfer-event"},
		Expected:   "عملٌ يُبدَأ يُنجَز أو يُسجَّل",
		Observed: "الردُّ يعود قبل العمل — **وسقوطُه يُستأنَف من حال " +
			"الطلب**: `accepted` بلا سائقٍ وغيرُ مغلقٍ هو عملٌ معلَّقٌ بذاته",
		UserVisible:  "نجاحٌ كامل",
		AdminVisible: Invisible,
		Recovery: "**كانسٌ كلَّ ثلاثين ثانيةً** يستأنف بالمسار نفسِه " +
			"(`AutoDispatch`) بقفل `FOR UPDATE SKIP LOCKED` — " +
			"**وحالُ الطلب هي الإتمام، فلا وسمَ قبل وقوع العمل**",
		Tests:     []string{"TestFAIL_XOB7_AutoTransferFireAndForget"},
		Registers: []string{"XOB-7", "R21"},
		Evidence:  "الردُّ يعود قبل العمل — ولا يُنتظَر ولا يُرصَد",
	},
	{
		ID: "PF-09", Title: "دفعُ الإشعار إلى FCM",
		Flows: []string{"F-07", "F-14"}, Where: Local, Result: Pass,
		Steps: []string{
			"١ الحدثُ يقع ويُكتب — **و`push_pending` في الإدراج نفسِه**",
			"٢ تفريعٌ إلى صفّ نقلٍ **لكلّ رمزِ جهاز**",
			"٣ مطالبةٌ تُثبَّت · نداءٌ **خارجَ المعاملة** · نتيجةٌ تُقيَّد",
			"٤ عابرٌ ⇒ تراجعٌ ٣٠ث…١٦د · سبعُ محاولاتٍ ثمّ حالٌ نهائيّة",
		},
		AtomicBoundary: "NON_ATOMIC عمداً — **شبكةٌ لا تدخل معاملة** · " +
			"والدلالةُ **محاولةٌ مرّةً على الأقلّ** لا تسليمٌ مرّةً واحدة",
		Failpoints: []string{"P-7/fake-transport-500-timeout-reset-deadtoken"},
		Expected:   "إشعارٌ يسقط يُعاد أو يُسجَّل",
		Observed: "**عابرٌ ⇒ `pending` بتراجعٍ وصنفِ خطأ** · " +
			"**ميّتٌ ⇒ `dead_token` والرمزُ يُحذف** · " +
			"**نفادٌ ⇒ `failed` بسببه مقروءاً** — ولا يضيع صامتاً",
		UserVisible:  "الخبرُ باقٍ في التطبيق ولو أخفق دفعُه",
		AdminVisible: Visible,
		Recovery: "الجولةُ الدوريّة (٣٠ث) — **مصدرُ العمل صفوفُ النقل " +
			"في القاعدة**، **والنبضةُ تعجيلٌ لا حقيقة**",
		Tests: []string{
			"TestPF09_N1_ProviderSuccessRecorded",
			"TestPF09_N2N3N8_TransientRetriesWithBackoff",
			"TestPF09_N4N10_RestartFindsPendingWork",
			"TestPF09_N5_AcceptedThenDeathRetriesNotLoses",
			"TestPF09_N6_TwoWorkersOneAttempt",
			"TestPF09_N7M1M2_DeadTokenDroppedHealthyDelivered",
			"TestPF09_N9_NewIntentNotSuppressedByPrior",
			"TestPF09_N11_ProviderOutageDoesNotRollbackBusiness",
			"TestPF09_N12_ExhaustedBecomesDiscoverableFailure",
			"TestPF09_M3M4_PerTargetTruthSurvivesRestart",
		},
		Registers: []string{"R23"},
		Evidence: "**والمِعراضُ كان موجوداً والمصفوفةُ تقول `TESTABILITY " +
			"SEAM REQUIRED`** — `server.WithPushTransport` و`FakePush` " +
			"بمخرجاتٍ مبرمَجة. **فصُحّح النصُّ الشائخ.** " +
			"**والحقيقةُ لكلّ هدفٍ لا لكلّ إشعار**: قيس أنّ فريدَ " +
			"`device_tokens` هو الرمزُ وحدَه و`Register` لا يحذف رموزَ " +
			"صاحبه — **فللحساب أجهزةٌ عدّة**، **وحالٌ واحدةٌ تكذب حين " +
			"يقبل جهازٌ وينقطع آخر** (مقيسٌ في `M3`). " +
			"**والطبقةُ الثالثةُ لم تُدَّعَ**: `accepted` تعني قبولَ " +
			"المزوّد، **ولا إقرارَ من جهاز** — وذاك `P-8`.",
	},

	// ══════════════════════════════════════════════════════════════════
	// **وخطران حُسما بالنفي** — دورةُ إصلاحٍ ٦ · ٢٠٢٦-٠٩-٠٦
	// ══════════════════════════════════════════════════════════════════
	//
	// **بقيا `NOT_RUN` منذ فُتحا** — **وخطرٌ لا يُحسَم يبقى مانعاً إلى
	// الأبد**، لا لأنّه وقع بل لأنّ أحداً لم يسأل.
	{
		ID: "PF-10", Title: "قيدُ الخزينة ومعاملةُ عمليّته",
		Flows: []string{"F-14"}, Where: Local, Result: Pass,
		Steps: []string{
			"١ تسويةُ التسليم تكتب مستحقَّ المتجر وأجرَ السائق",
			"٢ creditTreasury — **آخرُ التسوية** · يأخذ معاملةَ المنادي",
		},
		AtomicBoundary: "INHERITS_TX — ولا يفتح معاملةً بنفسه",
		Failpoints:     []string{"R4/platform-profit"},
		Expected:       "لا ربحَ منصّةٍ يُقيَّد لعمليّةٍ ارتدّت (R4)",
		Observed: "الحالُ بقيت at_dropoff · ولا أجرَ سائقٍ · " +
			"والدفترُ كما كان — **المعاملةُ واحدةٌ فعلاً**",
		FinInv:       []string{"FI-02", "FI-12"},
		UserVisible:  "500 — والعمليّةُ لم تقع أصلاً",
		AdminVisible: "CLEAN",
		Recovery:     "الإعادةُ تُسلّم الطلبَ كاملاً — ولا أثرَ نصفيّ",
		Tests: []string{
			"TestFIN_R4_TreasuryNeverWritesOutsideTransaction",
			"TestFIN_R4_TreasurySharesTheOperationTransaction",
		},
		Registers: []string{"R4"},
		Evidence: "**R4 DISPROVEN** — ١٥ مُنادياً للخزينة في الشجر " +
			"النحويّ · صفرٌ منها بالمَسبَح · وحقنُ قيد الربح أسقط " +
			"العمليّةَ كلَّها",
	},
	{
		ID: "PF-11", Title: "إنشاءُ طلبٍ خاصّ",
		Flows: []string{"F-02"}, Where: Local, Result: Pass,
		Steps: []string{
			"١ INSERT INTO orders — **كتابةٌ واحدةٌ لا غير**",
		},
		AtomicBoundary: "SINGLE STATEMENT — والعبارةُ الواحدةُ ذرّيّةٌ بلا معاملةٍ صريحة",
		Failpoints:     []string{"R11/create-custom"},
		Expected:       "أثرٌ واحدٌ أو لا أثر · ولا قيدَ في الدفتر (R11)",
		Observed: "مع الحقن: صفرُ طلبات · وبلا حقن: طلبٌ واحدٌ " +
			"وصفرُ قيود — **ولا سعرَ في الطلب الخاصّ حتّى يتّفقا**",
		FinInv:       []string{"FI-04"},
		UserVisible:  "500 — ولا طلبَ يتيمٌ يظهر له",
		AdminVisible: "CLEAN",
		Recovery:     "الإعادةُ تُنشئ طلباً واحداً",
		Tests: []string{
			"TestFIN_R11_CustomOrderCreationIsSingleWriteAndMoneyless",
		},
		Registers: []string{"R11"},
		Evidence: "**R11 DISPROVEN** — `CreateCustom` إدخالٌ واحدٌ " +
			"و`AgreeCustom` تحديثٌ واحد · **ولا شيءَ ثانٍ ليُفقَد**",
	},
	// ══════════════════════════════════════════════════════════════
	// **أنماطُ فشل التحويل السبعة** — `R24` (دورةُ إصلاحٍ ٢٧)
	// ══════════════════════════════════════════════════════════════
	{
		ID: "PF-12", Title: "أنماطُ فشل التحويل إلى المتجر — سبعة",
		Flows: []string{"F-01", "F-14", "F-20"}, Where: Local, Result: Pass,
		Steps: []string{
			"١ قراءةُ مفتاح التحويل — خطؤها ترتدّ إلى الافتراض",
			"٢ قراءةُ حال الطلب — خطؤها تُنهي المحاولة",
			"٣ جاهزيّةُ البوت — **تُفحص قبل القبول**",
			"٤ العنوانُ (رقمُ واتساب) — **يُفحص قبل القبول** (دورةُ ٢٧)",
			"٥ Transition ← accepted — خطؤه يُبقي pending",
			"٦ SendText — **خطؤه بعد قبولٍ ثابت** · يُكتب أثرُه في سجلّ الطلب",
			"٧ AutoDispatch — خطؤه يُبقي accepted مُبلَّغاً بلا سائق",
		},
		AtomicBoundary: "NON_ATOMIC — ثلاثةُ أفعالٍ لا واحد: قبولٌ وإبلاغٌ " +
			"وإنزال · **وما يُعلَم قبل القبول يُفحص قبله، وما لا يُعلَم " +
			"يُكتب أثرُه بعده**",
		Failpoints: []string{
			"R24/F4-transition", "R24/notifier-send-error", "R24/merchant-without-phone",
		},
		Expected: "لا فشلَ بلا أثرٍ يُقرأ · ولا نجاحَ كاذب",
		Observed: "سبعةٌ من سبعة: أربعةٌ تُبقي `pending` · وواحدٌ يكتب " +
			"أثراً في سجلّ الطلب يُقرأ في اللوحة · وواحدٌ صار `pending` " +
			"بفحص العنوان قبل القبول · وسابعٌ مُبلَّغٌ بلا سائقٍ وله زرّ",
		UserVisible:  "pending أو accepted — **ولا يُوسَم مُرسَلاً ما لم يُرسَل**",
		AdminVisible: "VISIBLE",
		Recovery: "RETRY SAFETY = MANUAL — المكتبُ يقرأ الأثرَ ويُرسل بيده · " +
			"ولا إعادةَ آليّة",
		Tests: []string{
			"TestR24_F1_SettingReadFailureLeavesPending",
			"TestR24_F2F4_TransitionRefusedLeavesPending",
			"TestR24_F3_BotNotReadyLeavesPending",
			"TestR24_F5_SendFailureAfterAccept",
			"TestR24_F6_NoMerchantPhoneAfterAccept",
			"TestR24_F7_DispatchFailureIsVisible",
			"TestR24_F5_TraceIsAdminReadableAfterTheRequest",
		},
		Registers: []string{"R24"},
		Evidence: "**R24 DISPROVEN** — سبعةُ أنماطٍ مقيسةٌ بنداءٍ حيّ · " +
			"`F5`: `accepted` و`sent_to_merchant_at=NULL` **وأثرٌ في " +
			"`order_events` يُقرأ في `GET /admin/orders/{id}`** · " +
			"`F6`: `pending` بعد فحص العنوان قبل القبول",
	},
}

// Counts إحصاءٌ مولَّد.
type Counts struct {
	Total, Local, RequiresP0, RequiresP7, RequiresP8 int
	Pass, DefectReproduced, RiskConfirmed            int
	ExpectedFail, Partial, Unproven                  int
	Visible, Invisible, PartialVisible               int
}

// Snapshot صورةٌ آليّة.
func Snapshot() map[string]any {
	c := Counts{Total: len(All)}
	for _, f := range All {
		switch f.Where {
		case Local:
			c.Local++
		case RequiresP0:
			c.RequiresP0++
		case RequiresP7:
			c.RequiresP7++
		case RequiresP8:
			c.RequiresP8++
		}
		switch f.Result {
		case Pass:
			c.Pass++
		case DefectReproduced:
			c.DefectReproduced++
		case RiskConfirmed:
			c.RiskConfirmed++
		case ExpectedFail:
			c.ExpectedFail++
		case Partial:
			c.Partial++
		case Unproven:
			c.Unproven++
		}
		switch f.AdminVisible {
		case Visible:
			c.Visible++
		case Invisible:
			c.Invisible++
		case PartialV:
			c.PartialVisible++
		}
	}
	out := append([]Flow(nil), All...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return map[string]any{"flows": out, "counts": c}
}

// Tests كلُّ الاختبارات المذكورة.
func Tests() []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range All {
		for _, n := range f.Tests {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}

// Failpoints كلُّ النقاط المسمّاة — **وما ليس اسماً يُستثنى.**
func Failpoints() []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range All {
		for _, n := range f.Failpoints {
			if n != "" && !strings.HasPrefix(n, "—") && !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}
