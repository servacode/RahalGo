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
	// Kind «فشلٌ» يحرّك الطلب، أو «بلاغٌ» يُنبّه العملياتِ ولا يمسّه، أو «تركٌ»
	// قبل الاستلام بسببٍ يخصّ السائق (`/release`).
	Kind string
	// Retired **لا يُعرض للسائق بعد اليوم** — ويبقى مقروءاً في البلاغات القديمة وسبباً للمكتب.
	Retired bool
}

const (
	FaultCustomer = "customer"
	FaultDriver   = "driver"
	FaultMerchant = "merchant"
	FaultPlatform = "platform"
)

// IsFault أهذا ذنبٌ معروف؟ — الأربعةُ وحدَها.
func IsFault(f string) bool {
	switch f {
	case FaultCustomer, FaultDriver, FaultMerchant, FaultPlatform:
		return true
	}
	return false
}

// أنواعُ السبب.
const (
	// ReasonFail يحرّك الطلب — **عند المتجر وحدَه**: يعود إلى المكتب حيّاً.
	ReasonFail = "fail"
	// ReasonReport **بلاغٌ لا فشل** — العملياتُ تُنبَّه والطلبُ كما هو.
	ReasonReport = "report"
	// ReasonRelease **تركٌ قبل الاستلام بمشكلةٍ تخصّ السائق** — يُرسَل إلى
	// `/release` لا إلى الانتقال (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢، البند ٢).
	ReasonRelease = "release"
)

// FailReasons ما يملك السائقُ اختيارَه فيحرّك الطلب. **والرموزُ لا تُترجَم
// هنا** — نصُّها في `packages/i18n` تحت `driver.failReasons`.
//
// ══════════════════════════════════════════════════════════════════════
// **ولا سببَ عند باب الزبون هنا بعد اليوم** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// «دائماً إذا في مشكلة بين السائق والزبون يكون الردّ: انتظر، الإدارة تقوم
// بالتواصل مع الزبون، **ويبقى الطلبُ مع السائق إلى أن تُحلّ القصّة**… وقتها
// الإدارةُ هي تُنهي الطلبَ من عندها.» **فأسبابُ الباب كلُّها صارت بلاغاتٍ**
// (`StageReports`) — **والإنهاءُ بابُ الإدارة** (`door.go`).
//
// ══════════════════════════════════════════════════════════════════════
// **ولا سببَ عند المتجر هنا أيضاً** (قرارُ المالك ٢٠٢٦-١٠-٠٣: «برأيي ينتظر الإدارة
// تحلّ المشكلة أفضل»)
// ══════════════════════════════════════════════════════════════════════
//
// كانت «المتجرُ مغلق · يرفض · لا يعرف الطلب» تسحب الطلبَ من السائق بكلمته — **«ويطمع
// السائقُ ويصير يخترع مشاكل»**. **فصارت بلاغاتٍ** (`StageReports`)، **والإدارةُ تتّصل
// بالمتجر وتقرّر** من لوحتها (`door.go`): «استلم الطلب» أو «حوّل لمتجرٍ آخر».
var FailReasons = []FailReason{}

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
	// **ومشكلةُ المتجر بلاغٌ والسائقُ ينتظر** (٢٠٢٦-١٠-٠٣) — والإدارةُ تقرّر.
	{Code: "merchant_closed", Fault: FaultMerchant, At: StAtPickup, Kind: ReasonReport},
	{Code: "merchant_refused", Fault: FaultMerchant, At: StAtPickup, Kind: ReasonReport},
	{Code: "order_unknown", Fault: FaultMerchant, At: StAtPickup, Kind: ReasonReport},
	// **«الطلبُ ما بيلتغي بعد ما يصير عند السائق»** — فإلغاءُ الزبون بالهاتف
	// خبرٌ للمكتب لا فعلٌ في الطلب. **ولا «يريد عنواناً آخر»** — الزبونُ لا
	// يغيّر العنوانَ بعد الطلب (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢، البند ٤).
	// **وقبل الشراء في الخاصّ أيضاً** (٢٠٢٦-١٠-٠٣): لا إلغاءَ للزبون بعد انطلاق السائق — يكتب في
	// الدردشة، والسائقُ يبلّغ، والمكتبُ يلغي أو يُكمل.
	{Code: "customer_cancelled_by_phone", Fault: FaultCustomer, At: StAssigned, Kind: ReasonReport},
	{Code: "customer_cancelled_by_phone", Fault: FaultCustomer, At: StPickedUp, Kind: ReasonReport},
	{Code: "customer_cancelled_by_phone", Fault: FaultCustomer, At: StOnTheWay, Kind: ReasonReport},

	// ── عند باب الزبون — **بلاغاتٌ كلُّها، والإدارةُ تُنهي** ──────────────
	//
	// (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢، البند ١.) **والذنبُ هنا ما يقترحه السببُ
	// لا حكمٌ**: الحكمُ يكتبه المكتبُ حين يُنهي (`door.go`) — **فـ«تأخّرتُ أنا»
	// ذنبُ السائق إن أقرّته الإدارة، و«رفض» ذنبُ الزبون إن أقرّته.**
	{Code: "customer_absent", Fault: FaultCustomer, At: StAtDropoff, Kind: ReasonReport},
	{Code: "customer_refused", Fault: FaultCustomer, At: StAtDropoff, Kind: ReasonReport},
	// **«لا يرد على الهاتف» يُشال من القائمة** (قرارُ المالك ٢٠٢٦-١٠-٠٣: «اتصرّف بالمنطقي، الغيه إذا ما
	// يلزم») — معناه معنى «الزبون لا يرد — اطلب من الإدارة الاتصال به»، والسائقُ يحتار بينهما.
	{Code: "customer_unreachable", Fault: FaultCustomer, At: StAtDropoff, Kind: ReasonReport, Retired: true},
	{Code: "address_wrong", Fault: FaultCustomer, At: StAtDropoff, Kind: ReasonReport},
	// **وذنبُ السائق يُقرّ به السائقُ نفسُه** — ووجودُه يجعل غيابَه اختياراً:
	// من تأخّر فبرد الطعامُ يجد لفظاً يقوله بدل أن يكتب «الزبون رفض».
	{Code: "driver_late", Fault: FaultDriver, At: StAtDropoff, Kind: ReasonReport},
	// **لا يُجيب أحد — والإدارةُ تتّصل** (السائقُ لا يملك رقمَه).
	{Code: "customer_no_answer", Fault: FaultCustomer, At: StAtDropoff, Kind: ReasonReport},
}

// ReleaseReasons **أسبابُ ترك الطلب قبل الاستلام** — «لدي مشكلة» في الطريق إلى
// المتجر وعنده (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢، البند ٢).
//
// «مجرّد ما ينطلق السائق ما يصير ينعاد للطابور» — **إلّا بسببٍ من هذه**، وكلمةٍ
// تشرحه. **والطلبُ لا يعود إلى من تركه أبداً، ودوامُه يُغلَق وحدَه**
// (`server/driver_handlers.go` · `handleDriverRelease`).
var ReleaseReasons = []FailReason{
	{Code: "bike_broken", Fault: FaultDriver, At: StAssigned, Kind: ReasonRelease},
	{Code: "accident", Fault: FaultDriver, At: StAssigned, Kind: ReasonRelease},
	{Code: "force_majeure", Fault: FaultDriver, At: StAssigned, Kind: ReasonRelease},
	{Code: "bike_broken", Fault: FaultDriver, At: StAtPickup, Kind: ReasonRelease},
	{Code: "accident", Fault: FaultDriver, At: StAtPickup, Kind: ReasonRelease},
	{Code: "force_majeure", Fault: FaultDriver, At: StAtPickup, Kind: ReasonRelease},
}

// ReleaseReasonAt **سببُ الترك إن كان يخصّ هذه المرحلة.**
func ReleaseReasonAt(code, status string) (FailReason, bool) {
	for _, r := range ReleaseReasons {
		if r.Code == code && r.At == status {
			return r, true
		}
	}
	return FailReason{}, false
}

// IsDoorReport **أهذا بلاغٌ عند باب الزبون؟**
func IsDoorReport(code string) bool {
	_, ok := StageReportAt(code, StAtDropoff)
	return ok
}

// AfterPickup **البضاعةُ في يد السائق** — من الاستلام إلى باب الزبون.
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٣: الزبونُ طلب الإلغاءَ والسائقُ في الطريق — «وين ألاقي
// الموضوع بلوحة الإدارة مشان أحلّه؟». **فلوحةُ الإدارة وأمراها في كلّ هذه المراحل**
// لا عند الباب وحدَه.)
func AfterPickup(status string) bool {
	return status == StPickedUp || status == StOnTheWay || status == StAtDropoff
}

// OfficeReasonAt **سببُ إنهاء المكتب يخصّ ضفّتَه** — بلاغُ المتجر عند المتجر، وبلاغُ
// الطريق والباب بعد الاستلام. **فلا يُنهى طلبٌ عند الباب بـ«المتجرُ مغلق».**
func OfficeReasonAt(code, from string) bool {
	if from == StAtPickup {
		_, ok := StageReportAt(code, StAtPickup)
		return ok
	}
	for _, st := range []string{StPickedUp, StOnTheWay, StAtDropoff} {
		if _, ok := StageReportAt(code, st); ok {
			return true
		}
	}
	return false
}

// OfficeReasonsAt **رموزُ البلاغ التي يقبلها إنهاءُ المكتب في هذه المرحلة** —
// بقاعدة `OfficeReasonAt` نفسِها، **بلا المحذوف** (`Retired`) وبلا تكرار.
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ١٦: قائمةُ النافذة تأتي من المحرّك.)
func OfficeReasonsAt(status string) []string {
	out := []string{}
	if !OfficeDecides(status) {
		return out
	}
	seen := map[string]bool{}
	for _, r := range StageReports {
		if r.Retired || seen[r.Code] || !OfficeReasonAt(r.Code, status) {
			continue
		}
		seen[r.Code] = true
		out = append(out, r.Code)
	}
	return out
}

// OfficeDecides **مراحلُ يقرّر فيها المكتبُ لا السائق** — عند المتجر، وبعد الاستلام.
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٣: مشكلةُ المتجر «ينتظر الإدارة تحلّ المشكلة».)
func OfficeDecides(status string) bool {
	return status == StAtPickup || AfterPickup(status)
}

// OfficeAnswers **يردّ المكتبُ على بلاغٍ في هذا الحال** — ما يقرّر فيه، **وقبل المتجر أيضاً**
// (قرارُ المالك ٢٠٢٦-١٠-٠٣: «الإلغاءُ قبل المتجر الإدارةُ تقرّره، لأنّ الزبونَ لا يبقى عنده
// زرُّ إلغاء»): «ألغِ الطلب» أو «أكمل الطلب». **ولا «عُد إلى المكتب» قبل المتجر** — لا بضاعةَ معه.
func OfficeAnswers(status string) bool {
	return status == StAssigned || OfficeDecides(status)
}

// IsTripReport **بلاغٌ يقرّر فيه المكتب** — عند المتجر أو في الطريق أو عند الباب.
// **يُقبل سبباً في إنهاء الإدارة.**
func IsTripReport(code string) bool {
	for _, st := range []string{StAtPickup, StPickedUp, StOnTheWay, StAtDropoff} {
		if _, ok := StageReportAt(code, st); ok {
			return true
		}
	}
	return false
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

// SuggestedFault **الذنبُ الذي يقترحه السبب** — للفشل وبلاغات الباب. **اقتراحٌ
// لشاشة الإدارة لا حكم**: الحكمُ يُكتب في إنهائها (`door.go`).
func SuggestedFault(code string) string {
	if f := FaultOf(code); f != "" {
		return f
	}
	for _, r := range StageReports {
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

// FailReasonsAt ما يُعرض في هذه الحالة — **الفشلُ ثمّ البلاغاتُ ثمّ الترك.**
func FailReasonsAt(status string) []FailReason {
	out := []FailReason{}
	for _, list := range [][]FailReason{FailReasons, StageReports, ReleaseReasons} {
		for _, r := range list {
			if r.At == status && !r.Retired {
				out = append(out, r)
			}
		}
	}
	return out
}

// Closes **أيُغلق هذا السببُ الطلب؟** — **لا شيءَ بيد السائق يُغلقه بعد اليوم.**
//
// كان الفشلُ عند الزبون يُغلق — **ونُسخ** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢): الإدارةُ
// تُنهي. **وعند المتجر لا** (٢٠٢٦-١٠-٠٢): الطلبُ ينتظر العمليات. **ويبقى الحقلُ
// في الردّ** — الشاشةُ تقرؤه لتقول للسائق ما سيقع قبل أن يضغط.
func (r FailReason) Closes() bool {
	return false
}
