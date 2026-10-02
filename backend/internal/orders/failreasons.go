package orders

// أسبابُ تعذّر التسليم — **وكلُّ سببٍ يحمل ذنبَه معه.**
//
// # لماذا قائمةٌ لا نصٌّ حرّ
//
// في تجربةٍ حيّة كتب السائقُ: «الزبون لا يقبل الاستلام او رفض الاستلام او لم
// اجد احد في العنوان او العنوان وهمي» — **أربعةُ أحكامٍ في سطرٍ واحد.**
//
// **وهي ليست شيئاً واحداً:**
//
//	لم أجد أحداً   →  ربّما تأخّر · يُعاد الاتّصال
//	رفض الاستلام   →  خلافٌ على الطلب
//	عنوانٌ وهميّ    →  زبونٌ يستحقّ المراجعة، وربّما الحظر
//
// **ونصٌّ حرٌّ لا يُعدّ ولا يُقاس**: لا يُعرف كم مرّةً كان العنوانُ وهمياً هذا
// الشهر، ولا أيُّ زبونٍ يتكرّر معه ذلك.
//
// # والذنبُ في القائمة لا في تقدير موظّف
//
// **الذنبُ يُقرأ من السبب لا يُكتب ثابتاً** (قرارُ المالك ٢٠٢٦-١٠-٠٢): «تأخّرتُ
// أنا» ذنبُ السائق ولا تعويضَ له، و«الزبونُ رفض» ذنبُ الزبون.
//
// **والتعويضُ لم يعد تلقائيّاً** (نُسخ ٢٠٢٦-١٠-٠٢): الذنبُ يقرّر **أيستحقّ**
// السائقُ، **وإنسانٌ في العمليات يوافق** قبل أن يُقبض. (قِيس قبل القرار: ٥٬٠٠٠
// تُدفع فوراً في كلّ ضغطة — فسائقٌ يكسب بضغطة.) انظر `compensation_requests.go`.
//
// وذنبُ السائق لا تعويضَ فيه. وتفصيلُ ما وقع يبقى في نصٍّ اختياريّ بجانب
// السبب — **القائمةُ تُصنّف والنصُّ يشرح.**

// FailReason سببٌ معرَّف، وذنبُه، وأينَ يظهر، **وما يفعله بالطلب.**
type FailReason struct {
	Code string
	// Fault من تسبّب — يقرّر التعويض
	Fault string
	// At الحالةُ التي يظهر فيها للسائق — **ولا يُقبل في غيرها** (٢٠٢٦-١٠-٠٢)
	At string
	// Kind «فشلٌ» يحرّك الطلب، أو «بلاغٌ» يُنبّه العملياتِ ولا يمسّه.
	Kind string
	// DoorWait **لا يُقبل قبل أن يمضي انتظارُ الباب** (`drivers.door_wait_sec`).
	DoorWait bool
}

const (
	FaultCustomer = "customer"
	FaultDriver   = "driver"
	FaultMerchant = "merchant"
	FaultPlatform = "platform"
)

// نوعا السبب.
const (
	// ReasonFail يحرّك الطلب — يُغلقه عند الزبون، ويوقفه للعمليات عند المتجر.
	ReasonFail = "fail"
	// ReasonReport **بلاغٌ لا فشل** — العملياتُ تُنبَّه والطلبُ كما هو.
	ReasonReport = "report"
)

// FailReasons ما يملك السائقُ اختيارَه. **والرموزُ لا تُترجَم هنا** — نصُّها
// في `packages/i18n` تحت `driver.failReasons`.
var FailReasons = []FailReason{
	// ── عند باب الزبون ──────────────────────────────────────────────────
	//
	// **والغيابُ وانقطاعُ الردّ بعد انتظارٍ لا قبله** (قرارُ المالك ٢٠٢٦-١٠-٠٢):
	// قِيس على التجهيز فشلٌ قُبل بعد ٦٨ ثانيةً من الوصول — **وضغطةٌ واحدةٌ
	// تمنع زبوناً من النقد شهراً.** والإدارةُ هي التي تتّصل به في الأثناء،
	// **لأنّ السائقَ لا يملك رقمَه أصلاً** (`customer_no_answer` أدناه).
	{Code: "customer_absent", Fault: FaultCustomer, At: StAtDropoff, Kind: ReasonFail, DoorWait: true},
	{Code: "customer_refused", Fault: FaultCustomer, At: StAtDropoff, Kind: ReasonFail},
	{Code: "customer_unreachable", Fault: FaultCustomer, At: StAtDropoff, Kind: ReasonFail, DoorWait: true},
	{Code: "address_wrong", Fault: FaultCustomer, At: StAtDropoff, Kind: ReasonFail},
	// **وذنبُ السائق يُقرّ به السائقُ نفسُه.**
	//
	// وقد يُظنّ أن أحداً لن يختاره — **لكنّ وجودَه يجعل غيابَه اختياراً**:
	// من تأخّر فبرد الطعامُ يجد لفظاً يقوله بدل أن يكتب «الزبون رفض» ويحمّل
	// زبوناً ذنبَه. **ومن لم يجد لفظاً لصدقه قال أقربَ الألفاظ إليه.**
	{Code: "driver_late", Fault: FaultDriver, At: StAtDropoff, Kind: ReasonFail},

	// ── عند باب المتجر ──────────────────────────────────────────────────
	//
	// **وهذه لا تُغلق الطلب** — ينتظر العمليات (تكلّم المتجرَ أو تبدّله).
	// انظر `merchant_blocked.go`. **و«الطلبُ غيرُ جاهز» خرج منها إلى البلاغات**
	// (قرارُ المالك ٢٠٢٦-١٠-٠٢): السائقُ ينتظر عليه، ولا يُحرَّر منه.
	{Code: "merchant_closed", Fault: FaultMerchant, At: StAtPickup, Kind: ReasonFail},
	{Code: "merchant_refused", Fault: FaultMerchant, At: StAtPickup, Kind: ReasonFail},
	{Code: "order_unknown", Fault: FaultMerchant, At: StAtPickup, Kind: ReasonFail},
}

// StageReports **بلاغاتٌ لا تغيّر الطلب** — زرُّ «لدي مشكلة» في كلّ مرحلة.
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٢: «زرّ لدي مشكلة له عملٌ معيّنٌ بكلّ مرحلة… ما يصير
// يتسكّر الطلبُ بأيّ حالةٍ قبل ما يصل للزبون».)
//
// **والبلاغُ يصل العملياتِ تنبيهاً ويُكتب في سجلّ الطلب** — ولا مالَ يتحرّك
// ولا حالَ تتبدّل. **ومن يقرّر ما بعده إنسانٌ في المكتب لا ضغطةٌ في الشارع.**
var StageReports = []FailReason{
	// المطبخُ لم ينتهِ — **السائقُ باقٍ على الطلب والعملياتُ تعلم.**
	{Code: "merchant_not_ready", Fault: FaultMerchant, At: StAtPickup, Kind: ReasonReport},
	// **«الطلبُ ما بيلتغي بعد ما يصير عند السائق»** — فإلغاءُ الزبون بالهاتف
	// وعنوانُه الجديد خبرٌ للمكتب لا فعلٌ في الطلب.
	{Code: "customer_cancelled_by_phone", Fault: FaultCustomer, At: StPickedUp, Kind: ReasonReport},
	{Code: "customer_cancelled_by_phone", Fault: FaultCustomer, At: StOnTheWay, Kind: ReasonReport},
	{Code: "customer_new_address", Fault: FaultCustomer, At: StPickedUp, Kind: ReasonReport},
	{Code: "customer_new_address", Fault: FaultCustomer, At: StOnTheWay, Kind: ReasonReport},
	// **عند الباب ولا يُجيب أحد — والإدارةُ تتّصل** (السائقُ لا يملك رقمَه).
	{Code: "customer_no_answer", Fault: FaultCustomer, At: StAtDropoff, Kind: ReasonReport},
}

// FaultOf ذنبُ السببِ المذكور — وفراغٌ إن كان الرمزُ مجهولاً.
//
// **ومجهولٌ لا يُنسب إلى أحد**: نسبتُه إلى الزبون تُعوّض بلا وجه، ونسبتُه إلى
// السائق تحرمه بلا وجه. **والسكوتُ أعدلُ من حكمٍ على غير بيّنة.**
//
// **والبلاغاتُ ليست هنا** — لا تُغلق طلباً فلا ذنبَ يُكتب عليه.
func FaultOf(code string) string {
	for _, r := range FailReasons {
		if r.Code == code {
			return r.Fault
		}
	}
	return ""
}

// failReasonAt السببُ إن كان يخصّ هذه المرحلة — **وإلّا فلا.**
//
// (قِيس على التجهيز ٢٠٢٦-١٠-٠٢: «الزبونُ غير موجود» قُبل والسائقُ عند المتجر،
// و«المتجرُ مغلق» قُبل والسائقُ عند باب الزبون — **فدُفع تعويضٌ في كلّ مرّة.**)
func failReasonAt(code, status string) (FailReason, bool) {
	for _, r := range FailReasons {
		if r.Code == code && r.At == status {
			return r, true
		}
	}
	return FailReason{}, false
}

// StageReportAt البلاغُ إن كان يخصّ هذه المرحلة.
func StageReportAt(code, status string) (FailReason, bool) {
	for _, r := range StageReports {
		if r.Code == code && r.At == status {
			return r, true
		}
	}
	return FailReason{}, false
}

// IsStageReport أهذا الرمزُ بلاغٌ في أيّ مرحلة؟
func IsStageReport(code string) bool {
	for _, r := range StageReports {
		if r.Code == code {
			return true
		}
	}
	return false
}

// FailReasonsAt طريقةٌ على الخدمة — لتُنادى من الخادم بلا استيراد الحزمة كلِّها.
func (s *Service) FailReasonsAt(status string) []FailReason { return FailReasonsAt(status) }

// FailReasonsAt ما يُعرض في هذه الحالة — **الفشلُ أوّلاً ثمّ البلاغات.**
func FailReasonsAt(status string) []FailReason {
	out := []FailReason{}
	for _, r := range FailReasons {
		if r.At == status {
			out = append(out, r)
		}
	}
	for _, r := range StageReports {
		if r.At == status {
			out = append(out, r)
		}
	}
	return out
}

// Closes **أيُغلق هذا السببُ الطلب؟** — الفشلُ عند الزبون وحدَه.
//
// **وعند المتجر لا** (قرارُ المالك ٢٠٢٦-١٠-٠٢): الطلبُ ينتظر العمليات،
// **والبلاغُ لا يغيّر شيئاً.** والشاشةُ تقرؤه لتقول للسائق ما سيقع قبل أن يضغط.
func (r FailReason) Closes() bool {
	return r.Kind == ReasonFail && r.At == StAtDropoff
}
