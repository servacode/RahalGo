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
		Flows: []string{"F-25", "F-29", "F-33"}, Where: Local, Result: ExpectedFail,
		Steps: []string{
			"١ الفعلُ الحسّاسُ يقع (مالٌ · تعليقٌ · إعداد)",
			"٢ audit() في الخلفيّة — **ولا تُفشل الفعل**",
		},
		AtomicBoundary: "NON_ATOMIC — التدقيقُ أفضلُ جهد",
		Failpoints:     []string{"AQ-4/audit-write"},
		Expected:       "CRITICAL SUCCESS REQUIRES AUDIT SUCCESS (AQ-4)",
		Observed:       "الفعلُ ينجح والقيدُ يسقط صامتاً",
		UserVisible:    "نجاحٌ كامل",
		AdminVisible:   Invisible,
		Recovery:       "لا مسارَ — الأثرُ ضاع",
		Tests:          []string{"TestFAIL_AQ4_AuditAtomicity"},
		Registers:      []string{"AQ-4"},
		Evidence:       "الردّ 200 · رصيدٌ=12000 · قيودُ التدقيق 0 ⇒ 0",
	},
	{
		ID: "PF-07", Title: "إنذارُ الراصد",
		Flows: []string{"F-09"}, Where: Local, Result: Unproven,
		Steps: []string{
			"١ UPDATE orders SET alerted_at — الوسم",
			"٢ RowsAffected == 0 ⇒ لا إعادة — الحارس",
			"٣ NotifyOps — الإشعار",
		},
		AtomicBoundary: "NON_ATOMIC — وسمٌ ثمّ إشعار",
		Failpoints:     []string{"R22/notify-write"},
		Expected:       "لا وسمَ قبل نجاح الإشعار",
		Observed:       "لم يُقَس — الترتيبُ مقروءٌ من الشيفرة والتشغيلُ يحتاج مِعراضاً",
		UserVisible:    "لا شيء — إنذارٌ داخليّ",
		AdminVisible:   Invisible,
		Recovery:       "لا مسارَ — الوسمُ يمنع إعادةَ المحاولة",
		Tests:          []string{"TestFAIL_R22_WatchdogMarkerBeforeNotify"},
		Registers:      []string{"R22"},
		Evidence:       "لم يُشغَّل — لا بابَ إداريٌّ للراصد · TESTABILITY SEAM REQUIRED",
	},
	{
		ID: "PF-08", Title: "التحويلُ التلقائيُّ في خيطٍ منفصل",
		Flows: []string{"F-01", "F-18"}, Where: Local, Result: Partial,
		Steps: []string{
			"١ الردُّ للزبون يعود",
			"٢ go autoTransfer(...) — **لا يُنتظَر ولا يُرصَد**",
		},
		AtomicBoundary: "NON_ATOMIC — خارجَ دورة الطلب أصلاً",
		Failpoints:     []string{"XOB-7/auto-transfer-event"},
		Expected:       "عملٌ يُبدَأ يُنجَز أو يُسجَّل",
		Observed:       "الردُّ يعود قبل العمل · وسقوطُه لا يظهر لأحد",
		UserVisible:    "نجاحٌ كامل",
		AdminVisible:   Invisible,
		Recovery:       "لا صفَّ دائمٌ ولا إعادةَ محاولة",
		Tests:          []string{"TestFAIL_XOB7_AutoTransferFireAndForget"},
		Registers:      []string{"XOB-7", "R21"},
		Evidence:       "الردُّ يعود قبل العمل — ولا يُنتظَر ولا يُرصَد",
	},
	{
		ID: "PF-09", Title: "دفعُ الإشعار إلى FCM",
		Flows: []string{"F-07", "F-14"}, Where: Local, Result: RiskConfirmed,
		Steps: []string{
			"١ الحدثُ يقع ويُكتب",
			"٢ push.Send إلى المنصّة — **مرّةً واحدةً بلا إعادة**",
		},
		AtomicBoundary: "NON_ATOMIC — دفعٌ خارج المعاملة",
		Failpoints:     []string{"P-7/fake-transport-500-timeout-reset"},
		Expected:       "إشعارٌ يسقط يُعاد أو يُسجَّل",
		Observed:       "ثلاثةُ إخفاقاتٍ · ثلاثةُ نداءاتٍ · ولا رابع — يضيع بلا أثر",
		UserVisible:    "لا شيء",
		AdminVisible:   Invisible,
		Recovery:       "لا إعادةَ ولا صفَّ انتظارٍ ولا حالٌ معلَّقة",
		Tests:          []string{"TestEV_R23PushFailureIsLost", "TestFAIL_P7SeamRequired_FCMTransport"},
		Registers:      []string{"R23"},
		Evidence:       "لا مِعراضَ لحقن push.Transport — TESTABILITY SEAM REQUIRED",
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
