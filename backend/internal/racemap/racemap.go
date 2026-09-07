// Package racemap خريطةُ التدفّقات الحسّاسة للتزامن — **`P-5`.**
//
// **ولا Markdown مصدراً وحيداً** (البند ٣٣): `P-9` سيسأل «أيُّ سباقٍ يمسّه
// هذا الملفّ؟» و`P-10` سيسأل «أكلُّ سباقٍ حرِجٍ له اختبارٌ شُغّل؟» —
// **وسؤالان كهذان لا يُجابان من نصٍّ منسَّق.**
package racemap

import "sort"

// Where أين يُثبَت.
type Where string

const (
	// Local يُثبَت على القاعدة المحلّيّة المعتمدة في `P-3V`.
	Local Where = "LOCAL"
	// RequiresP0 يحتاج بيئةَ تكاملٍ متعدّدةَ الخدمات.
	RequiresP0 Where = "REQUIRES_P0"
	// DeferredP8 طبقةُ أندرويد.
	DeferredP8 Where = "DEFERRED_TO_P8"
)

// Result نتيجةُ السباق كما قِيست — **ولا `PASS` زور.**
type Result string

const (
	Pass           Result = "PASS"
	RiskConfirmed  Result = "RISK_CONFIRMED"
	Proven         Result = "PROVEN"
	ExpectedFail   Result = "EXPECTED_FAIL"
	Partial        Result = "PARTIAL"
	NotImplemented Result = "NOT_IMPLEMENTED"
)

// Race تدفّقٌ حسّاسٌ للتزامن.
type Race struct {
	// ID معرّفٌ مستقرّ — `C-01`.
	ID string `json:"id"`
	// Flows تدفّقاتُ المنظومة التي يمسّها.
	Flows []string `json:"flows"`
	// Title ما هو.
	Title string `json:"title"`
	// Actors من يتنازع.
	Actors []string `json:"actors"`
	// Shared المورِدُ المشترَك.
	Shared string `json:"shared_resource"`
	// Window نافذةُ السباق — **مقيسةٌ من الشيفرة لا مظنونة.**
	Window string `json:"race_window"`
	// Invariant الثابتُ الذي يجب أن يصمد.
	Invariant string `json:"invariant"`
	// Tests الاختباراتُ التي تحرسه.
	Tests []string `json:"tests"`
	// Result ما قِيس.
	Result Result `json:"result"`
	// Registers ما يمسّه من السجلّات المجمَّدة.
	Registers []string `json:"registers,omitempty"`
	// FinInv ثوابتُ `P-4` التي تُشغَّل بعده.
	FinInv []string `json:"financial_invariants,omitempty"`
	// Where أين يُثبَت.
	Where Where `json:"where"`
	// TxClass حدُّ المعاملة المقيسُ في `P-4`.
	TxClass string `json:"transaction_class,omitempty"`
	// Evidence سطرُ الدليل كما طُبع.
	Evidence string `json:"evidence,omitempty"`
}

// All التدفّقاتُ العشرة — **`CONCURRENCY-SENSITIVE FLOWS = 10`.**
var All = []Race{
	{
		ID: "C-01", Title: "سائقان يقبلان الطلبَ نفسَه",
		Flows: []string{"F-07", "F-08"}, Actors: []string{"driver", "driver"},
		Shared: "orders.driver_id", Where: Local, Result: Pass,
		Window:    "UPDATE … WHERE driver_id IS NULL AND status='dispatching' — **شرطيٌّ ذرّيّ**",
		Invariant: "AT MOST ONE DRIVER OWNS THE ORDER",
		Tests:     []string{"TestRACE_TwoDriversSameOrder"},
		Registers: []string{"R10"}, FinInv: []string{"FI-05", "FI-06", "FI-02"},
		TxClass:  "ATOMIC (شرطٌ في جملةِ التحديث نفسِها)",
		Evidence: "5 جولات · فائزٌ واحدٌ في كلٍّ · صاحبُ الطلبِ واحدٌ في القاعدة",
	},
	{
		ID: "C-02", Title: "سقفُ الطلبات النشطة",
		Flows: []string{"F-08", "F-10"}, Actors: []string{"driver", "driver"},
		Shared: "عدُّ طلبات السائق المفتوحة", Where: Local, Result: RiskConfirmed,
		Window:    "driver_handlers.go:519 يقرأ العددَ · ثمّ يقارن في Go · ثمّ يكتب — **بلا قفلٍ ولا معاملة**",
		Invariant: "active orders <= drivers.max_active_orders",
		Tests:     []string{"TestRACE_MaxActiveOrders"},
		Registers: []string{"R10"},
		TxClass:   "NON_ATOMIC (قراءةٌ ثمّ كتابةٌ في جملتين)",
		Evidence:  "الأساسُ المتتابعُ 409 · والتزامنُ تجاوزَ السقفَ في 3 من 5",
	},
	{
		ID: "C-03", Title: "تحويلُ مرشَّحٍ واحدٍ مرّتين",
		Flows: []string{"F-21", "F-22"}, Actors: []string{"admin", "admin"},
		Shared: "merchant_leads.merchant_id · merchants.lead_id", Where: Local, Result: Pass,
		Window: "**أُغلقت في دورةِ إصلاحٍ ٥**: كان `convertLead` يقرأ " +
			"`merchant_id` خارجَ المعاملة ثمّ يُنشئ المتجر — قراءةٌ ثمّ " +
			"كتابةٌ بلا قفل. **وصار يقفل صفَّ المرشَّح `FOR UPDATE` " +
			"داخلَ المعاملة ويعيد الفحصَ تحته**، **وللمتجر `lead_id` " +
			"بفهرسٍ فريدٍ يرفض الثانيَ في القاعدة.**",
		Invariant: "ONE REAL MERCHANT IDENTITY MUST NOT BECOME TWO PAYABLE MERCHANT/REP RELATIONSHIPS",
		Tests: []string{"TestRACE_DuplicateLeadConversion",
			"TestUNIQ_ConcurrentLeadConversionMakesOneMerchant",
			"TestUNIQ_ConcurrentConversionWithExistingOwner",
			"TestUNIQ_DatabaseRefusesSecondMerchantForSameLead"},
		Registers: []string{"XG-18", "D2", "D25"}, FinInv: []string{"FI-02", "FI-03", "FI-05"},
		TxClass: "ATOMIC (دورةُ إصلاحٍ ٤) + ROW LOCK + UNIQUE INDEX (دورةُ إصلاحٍ ٥)",
		Evidence: "**قبل**: ستُّ جولاتٍ من ست ⇒ متجران وإسنادان للمندوب " +
			"(وصاحبُ المتجر له حسابٌ من قبل — والحمايةُ كانت تفرّدَ " +
			"`users.phone` لا تفرّدَ المرشَّح). " +
			"**بعد**: ستٌّ من ست ⇒ متجرٌ واحدٌ وإسنادٌ واحد، بتداخلٍ " +
			"مقيسٍ 21–83 مِلّي ثانية.",
	},
	{
		ID: "C-04", Title: "تدخّلُ الإدارة مقابلَ فعلِ التطبيق",
		Flows: []string{"F-19", "F-08"}, Actors: []string{"admin", "driver"},
		Shared: "orders.driver_id · orders.status", Where: Local, Result: Pass,
		Window:    "إسنادٌ إداريٌّ مقابلَ قبولِ سائقٍ على الطلب نفسِه",
		Invariant: "لا حالَ مكسورةٌ ولا انتقالٌ مستحيلٌ ولا أثرٌ ماليٌّ مزدوج",
		Tests:     []string{"TestRACE_AdminVsAppTransition"},
		Registers: []string{"R7"}, FinInv: []string{"FI-05", "FI-06", "FI-02", "FI-10"},
		TxClass:  "PARTIALLY_ATOMIC (settle داخل معاملةِ الانتقال)",
		Evidence: "5 جولات · نجح واحدٌ في كلٍّ · وحالٌ مشروعةٌ دائماً",
	},
	{
		ID: "C-05", Title: "مفتاحُ تكرارٍ واحدٌ بحمولةٍ واحدة",
		Flows: []string{"F-01", "F-03"}, Actors: []string{"customer", "customer"},
		Shared: "idempotency_keys", Where: Local, Result: Pass,
		Window:    "INSERT … ON CONFLICT DO NOTHING — **مطالبةٌ ذرّيّة**",
		Invariant: "ONE ECONOMIC EFFECT PER KEY",
		Tests: []string{"TestIDEM_SameKeySamePayloadSequential",
			"TestIDEM_SameKeySamePayloadConcurrent", "TestIDEM_InProgressOverlap"},
		Registers: []string{"R8"},
		TxClass:   "ATOMIC (قيدُ التفرّد يحسم المطالبة)",
		Evidence:  "5 جولاتٍ متزامنة · طلبٌ واحدٌ في كلٍّ · والثاني in_progress أثناء نافذةِ الأوّل",
	},
	{
		ID: "C-06", Title: "مطالبةٌ يتيمةٌ بعد فقدِ الردّ",
		Flows: []string{"F-03"}, Actors: []string{"customer"},
		Shared: "idempotency_keys.done", Where: Local, Result: Pass,
		Window: "المطالبةُ تُكتب في claimIdempotency والختمُ في storeIdempotency — " +
			"**وسقوطُ العمليّة بينهما يترك done=false**",
		Invariant: "المطالبةُ اليتيمةُ لا تحجب صاحبَها إلى الأبد",
		Tests: []string{"TestIDEM_CommitThenLostResponse", "TestIDEM_CleanupAfterTTL",
			"TestFAIL_C06_OrphanBeforeCommitBlocksOwner",
			"TestFAIL_C06_TwoReclaimersExecuteNothing",
			"TestIDEM_CommittedBeforeResultDoesNotDuplicate",
			"TestIDEM_LostResponseReplays", "TestIDEM_CleanupSparesLiveClaim"},
		Registers: []string{"R8"},
		TxClass:   "ATOMIC — العملُ وعلامةُ التثبيت والنتيجةُ في معاملةٍ واحدة",
		Evidence: "**بعد الإصلاح**: يتيمةٌ شائخةٌ ⇒ `201` وتنفيذٌ واحدٌ " +
			"وعلامةٌ مثبَّتة · ومستردّان متزامنان ⇒ تنفيذٌ واحد · ومالكٌ " +
			"استُرِدّت منه ⇒ صفرُ كتابات · ومهلةٌ انتهت ومعاملةٌ حيّةٌ ⇒ " +
			"المستردُّ محبوسٌ على القفل ثمّ يقرأ ما ثبت · وعملٌ ثبت ثمّ " +
			"ماتت نتيجتُه ⇒ يُعاد الردُّ بلا تكرار · والتقليمُ لا يمسّ " +
			"حيّاً · ومالكٌ قديمٌ لا يمحو مطالبةَ غيرِه.",
	},
	{
		ID: "C-07", Title: "قرارا سحبٍ متزامنان",
		Flows: []string{"F-24"}, Actors: []string{"admin", "admin"},
		Shared: "payout_requests.status · محفظةُ الطالب", Where: Local, Result: Pass,
		Window:    "SELECT … FOR UPDATE ثمّ شرطُ status='pending' داخلَ معاملة",
		Invariant: "لا تسويةٌ مزدوجةٌ ولا خصمٌ مزدوج",
		Tests:     []string{"TestRACE_DuplicatePayout"},
		FinInv:    []string{"FI-11", "FI-05", "FI-04", "FI-02"},
		TxClass:   "ATOMIC (Begin + FOR UPDATE + شرطُ الحال)",
		Evidence:  "قيدُ سحبٍ واحدٌ · مجموعُه −100000 · ونجح قرارٌ واحدٌ من اثنين",
	},
	{
		ID: "C-08", Title: "استردادٌ مقابلَ سحبٍ على المستحقِّ نفسِه",
		Flows: []string{"F-15", "F-24"}, Actors: []string{"admin", "admin"},
		Shared: "محفظةُ صاحب المتجر", Where: Local, Result: ExpectedFail,
		Window: "reverseCommissions يعكس المستحقَّ · وقرارُ السحب يخصمه — " +
			"**والقيدُ CHECK (balance >= 0) يُسقط الخاسر**",
		Invariant: "CUSTOMER REFUND ENTITLEMENT MUST NOT DEPEND ON CURRENT DOWNSTREAM ACTOR BALANCES",
		Tests:     []string{"TestRACE_RefundVsPayout"},
		Registers: []string{"XG-10", "XG-11"},
		FinInv:    []string{"FI-11", "FI-05", "FI-04", "FI-06", "FI-10"},
		TxClass:   "PARTIALLY_ATOMIC (كلٌّ ذرّيٌّ في نفسِه · ولا تنسيقَ بينهما)",
		Evidence:  "السحبُ فاز بـ40000 · والاستردادُ سقط · والمستردُّ للزبون = 0",
	},
	{
		ID: "C-09", Title: "إبطالُ جلسةٍ أثناء طلبٍ جارٍ",
		Flows: []string{"F-30"}, Actors: []string{"admin", "customer"},
		Shared: "refresh_tokens · رمزُ الوصول", Where: Local, Result: Pass,
		Window:    "الإبطالُ يكتب القاعدةَ ثمّ Redis · والوسيطُ يسأل الحقيقةَ الموثوقة",
		Invariant: "إبطالُ الجلسة يُنهي القدرةَ على العمل",
		Tests: []string{"TestRACE_SessionRevokedDuringRequest",
			"TestR16_R2_RedisHitDenies",
			"TestR16_R3_RedisHealthyButKeyMissingDBCatches",
			"TestR16_STG_RedisOutageAuthorityHolds"},
		// **و`R15` نُزعت من هنا** — **لم يقس هذا الصفُّ سريانَ سحبِ
		// دورٍ قطّ، وإنّما إبطالَ جلسة.** **ودليلٌ يُنسَب إلى سجلٍّ لا
		// يقيسه يُغلقه بالكلام.**
		Registers: []string{"R16"},
		TxClass:   "—",
		Evidence: "**والقياسُ السابقُ كان على توكنٍ بلا معرّفِ جلسة** — " +
			"**فلم يُسأل عنه أصلاً**، وقُرئ «رمزُ الوصول ينجو» وهو " +
			"«صنفٌ لا يُبطَل». **وصُحّح إلى توكنٍ له صفٌّ دائم**: " +
			"نداءٌ بعد الإبطال ⇒ 401. " +
			"**وبذاكرةٍ حقيقيّةٍ تُوقَف**: حيّةٌ ⇒ 200 · مُبطَلةٌ ⇒ 401 · " +
			"**وإبطالٌ وقع والذاكرةُ ساقطةٌ يبقى نافذاً بعد عودتها** " +
			"(المفتاحُ غائبٌ والقاعدةُ تحكم).",
	},
	{
		ID: "C-10", Title: "طابورُ موقع السائق",
		Flows: []string{"F-11", "F-13"}, Actors: []string{"driver-app"},
		Shared: "طابورُ المواقع في الجهاز", Where: DeferredP8, Result: Partial,
		Window:    "all → شبكة → clear بلا قفلٍ — **في العميل لا في الخادم**",
		Invariant: "لا نقطةَ تُمسَح قبل أن تصل · ولا نقطةَ تُرسَل مرّتين",
		Tests:     []string{"TestRACE_LocationQueueIsClientSide"},
		Registers: []string{"R19", "R20"},
		TxClass:   "—",
		Evidence:  "بابُ الخادم يتحمّل 4 دفعاتٍ متزامنةً بلا ازدواج · وفقدُ الدفعة في العميل → P-8",
	},
}

// Counts إحصاءٌ مولَّد.
type Counts struct {
	Total          int `json:"total"`
	Local          int `json:"local"`
	RequiresP0     int `json:"requires_p0"`
	DeferredP8     int `json:"deferred_p8"`
	Pass           int `json:"pass"`
	RiskConfirmed  int `json:"risk_confirmed"`
	Proven         int `json:"proven"`
	ExpectedFail   int `json:"expected_fail"`
	Partial        int `json:"partial"`
	NotImplemented int `json:"not_implemented"`
}

// Snapshot صورةٌ آليّةٌ للخريطة.
func Snapshot() map[string]any {
	c := Counts{Total: len(All)}
	for _, r := range All {
		switch r.Where {
		case Local:
			c.Local++
		case RequiresP0:
			c.RequiresP0++
		case DeferredP8:
			c.DeferredP8++
		}
		switch r.Result {
		case Pass:
			c.Pass++
		case RiskConfirmed:
			c.RiskConfirmed++
		case Proven:
			c.Proven++
		case ExpectedFail:
			c.ExpectedFail++
		case Partial:
			c.Partial++
		case NotImplemented:
			c.NotImplemented++
		}
	}
	races := append([]Race(nil), All...)
	sort.Slice(races, func(i, j int) bool { return races[i].ID < races[j].ID })
	return map[string]any{"races": races, "counts": c}
}

// Tests كلُّ الاختبارات المذكورة في الخريطة.
func Tests() []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range All {
		for _, n := range r.Tests {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}
