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
// التعويضُ **تلقائيٌّ بلا يد** (قرار المالك). فلو تُرك الذنبُ لحكمٍ لاحق لَصار
// التعويضُ يدوياً من بابٍ آخر — **وقاعدةٌ تُنفَّذ بيدٍ ليست قاعدة، هي عادة.**
//
// # وثلاثةٌ لا تُعوَّض
//
// ذنبُ السائق لا تعويضَ فيه، **وذنبُ المتجر كذلك**: المنصةُ تتحمّل بضاعتَه
// وتعوّض سائقَها، **ولا تجمع عليها الاثنين بلا سبب**. وتفصيلُ ما وقع يبقى في
// نصٍّ اختياريّ بجانب السبب — **القائمةُ تُصنّف والنصُّ يشرح.**

// FailReason سببٌ معرَّف، وذنبُه، وأينَ يظهر.
type FailReason struct {
	Code string
	// Fault من تسبّب — يقرّر التعويض
	Fault string
	// At الحالةُ التي يظهر فيها للسائق
	At string
}

const (
	FaultCustomer = "customer"
	FaultDriver   = "driver"
	FaultMerchant = "merchant"
	FaultPlatform = "platform"
)

// FailReasons ما يملك السائقُ اختيارَه. **والرموزُ لا تُترجَم هنا** — نصُّها
// في `packages/i18n` تحت `driver.failReasons`.
var FailReasons = []FailReason{
	// ── عند باب الزبون ──────────────────────────────────────────────────
	{Code: "customer_absent", Fault: FaultCustomer, At: StAtDropoff},
	{Code: "customer_refused", Fault: FaultCustomer, At: StAtDropoff},
	{Code: "customer_unreachable", Fault: FaultCustomer, At: StAtDropoff},
	{Code: "address_wrong", Fault: FaultCustomer, At: StAtDropoff},
	// **وذنبُ السائق يُقرّ به السائقُ نفسُه.**
	//
	// وقد يُظنّ أن أحداً لن يختاره — **لكنّ وجودَه يجعل غيابَه اختياراً**:
	// من تأخّر فبرد الطعامُ يجد لفظاً يقوله بدل أن يكتب «الزبون رفض» ويحمّل
	// زبوناً ذنبَه. **ومن لم يجد لفظاً لصدقه قال أقربَ الألفاظ إليه.**
	{Code: "driver_late", Fault: FaultDriver, At: StAtDropoff},

	// ── عند باب المتجر ──────────────────────────────────────────────────
	{Code: "merchant_closed", Fault: FaultMerchant, At: StAtPickup},
	{Code: "merchant_refused", Fault: FaultMerchant, At: StAtPickup},
	{Code: "merchant_not_ready", Fault: FaultMerchant, At: StAtPickup},
	{Code: "order_unknown", Fault: FaultMerchant, At: StAtPickup},
}

// FaultOf ذنبُ السببِ المذكور — وفراغٌ إن كان الرمزُ مجهولاً.
//
// **ومجهولٌ لا يُنسب إلى أحد**: نسبتُه إلى الزبون تُعوّض بلا وجه، ونسبتُه إلى
// السائق تحرمه بلا وجه. **والسكوتُ أعدلُ من حكمٍ على غير بيّنة.**
func FaultOf(code string) string {
	for _, r := range FailReasons {
		if r.Code == code {
			return r.Fault
		}
	}
	return ""
}

// FailReasonsAt طريقةٌ على الخدمة — لتُنادى من الخادم بلا استيراد الحزمة كلِّها.
func (s *Service) FailReasonsAt(status string) []FailReason { return FailReasonsAt(status) }

// FailReasonsAt ما يُعرض في هذه الحالة.
func FailReasonsAt(status string) []FailReason {
	out := []FailReason{}
	for _, r := range FailReasons {
		if r.At == status {
			out = append(out, r)
		}
	}
	return out
}
