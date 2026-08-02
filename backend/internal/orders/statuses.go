package orders

// آلة حالات الطلب (PLAN.md §6.1) — كل انتقال مسموح به لأدوار محددة فقط،
// وكل ما عداه مرفوض. الانتقالات النهائية تُغلق الطلب وتُطلق التسويات المالية.

const (
	StPending     = "pending"
	StAccepted    = "accepted"
	StPreparing   = "preparing"
	StDispatching = "dispatching"
	StAssigned    = "assigned"
	StAtPickup    = "at_pickup"
	StPickedUp    = "picked_up"
	StOnTheWay    = "on_the_way"
	StAtDropoff   = "at_dropoff"
	StDelivered   = "delivered"
	StRejected    = "rejected"
	StCancelled   = "cancelled"
	StFailed      = "failed"
	StRefunded    = "refunded"
)

// transition يعرّف انتقالاً مسموحاً: إلى أي حالة، ومن أي الأدوار.
type transition struct {
	To    string
	Roles []string // الأدوار المخولة (admin دائماً مخول ضمنياً)
}

var opsRoles = []string{"ops"}
var merchantOps = []string{"merchant", "ops"}
var driverOps = []string{"driver", "ops"}

// allowedTransitions خارطة الآلة الكاملة.
var allowedTransitions = map[string][]transition{
	StPending: {
		{StAccepted, merchantOps},
		{StRejected, merchantOps},
		{StCancelled, []string{"customer", "ops"}},
	},
	StAccepted: {
		{StPreparing, merchantOps},
		// **إلى الطابور رأساً** — دون المرور بـ«التحضير».
		//
		// حين تدير المنصةُ الطلبات، المتجرُ خارج النظام: يُبلَّغ برسالةٍ نصّية
		// ولا يضغط شيئاً. **فإعلانُ «بدء التحضير» عنه ادّعاءُ ما لا نعلمه**،
		// وانتظارُه منه انتظارُ ما لا يأتي.
		{StDispatching, opsRoles},
		// نافذة تدارُك للزبون بعد القبول — مدّتها في الإعدادات وتُفرض في المحرّك
		{StCancelled, []string{"customer"}},
		// المتجر يلغي بعد قبوله — نفد صنف أو تعطّل مطبخه. والواقع اليومي يفرضه:
		// بلا هذا يتّصل بالمنصة ليُلغى بالنيابة عنه، فيضيع الوقت ويضيع السبب.
		// والسبب **إلزامي** في هذه الحالة (يُفرض في المعالِج لا في الخارطة).
		{StCancelled, merchantOps},
	},
	StPreparing: {
		{StDispatching, opsRoles}, // طلب سائق
		{StCancelled, merchantOps},
	},
	StDispatching: {
		{StAssigned, driverOps}, // قبول سائق أو إسناد يدوي
		{StCancelled, opsRoles},
	},
	StAssigned: {
		{StAtPickup, driverOps},
		{StDispatching, driverOps}, // فك الإسناد وإعادة الطلب
		{StCancelled, opsRoles},
	},
	StAtPickup: {
		{StPickedUp, driverOps},
		// **المطعمُ مغلقٌ أو رافض — والسائقُ عند بابه.**
		//
		// كانت هذه الحالةُ بلا مخرجٍ غير الاستلام: فلو وصل السائقُ ووجد
		// المحلَّ مغلقاً، أو لم تصله الرسالةُ أصلاً، أو رفض التحضير — **بقي
		// الطلبُ معلّقاً**: مالُ الزبون محجوز، والسائقُ مربوطٌ بطلبٍ لا يُقفل،
		// وسقفُه النقديّ مشغولٌ به.
		//
		// **وهو من هناك، فهو من يقول.** والسببُ إلزاميّ (يُفرض في المعالِج).
		{StFailed, driverOps},
		// **والتحريرُ ممتدٌّ حتى الاستلام لا حتى الوصول.**
		//
		// كان فكُّ الإسناد متاحاً في `assigned` وحدَها — **فمن وصل البابَ ثمّ
		// عرض له عارضٌ لم يملك إلّا أن يُفشل الطلب**: يُقفل بضاعةً لم تخرج،
		// ويُحسب على أحدٍ ذنبٌ لم يقع، **ويُحرم زبونٌ من طلبٍ كان سائقٌ آخر
		// يوصله في دقائق.**
		//
		// **والفرقُ بين التحرير والإفشال ليس في السائق بل في الطلب**: ذاك يعود
		// إلى الطابور حيّاً، وهذا ينتهي. ولا يجوز أن يُختار الثاني لأن الأوّل
		// لا بابَ له.
		//
		// **وحدُّه الاستلام**: بعد أن تصير البضاعةُ في يده لا يُترك الطلبُ
		// لغيره — **الطعامُ معه لا في المتجر.**
		{StDispatching, driverOps},
		{StCancelled, opsRoles},
	},
	StPickedUp: {
		{StOnTheWay, driverOps},
	},
	StOnTheWay: {
		{StAtDropoff, driverOps},
	},
	StAtDropoff: {
		{StDelivered, driverOps},
		{StFailed, driverOps}, // زبون لا يرد / يرفض الاستلام
	},
	StDelivered: {
		{StRefunded, []string{}}, // أدمن فقط
	},
}

// authorizingRole أيُّ أدوارِ الفاعل خوّله هذا الانتقال.
//
// **يُسجَّل لحظةَ وقوعه لا يُخمَّن لاحقاً.** عدُّ المخالفات يسأل «من ألغى؟»،
// و`cancelled` يصل إليها الزبونُ والمتجرُ والعملياتُ والأدمن — **فحسبانُها
// كلَّها على المتجر يحظر بريئاً**. واستنتاجُه لاحقاً من جدول الأدوار يكذب:
// **الأدوارُ تتغيّر والماضي لا يتغيّر.**
//
// والترتيبُ يتبع ترتيبَ الفاعل نفسه: أوّلُ دورٍ يخوّله هو الذي عمل به.
func authorizingRole(from, to string, roles []string) string {
	for _, r := range roles {
		if canTransition(from, to, []string{r}) {
			return r
		}
	}
	return ""
}

// terminal الحالات النهائية — تُغلق الطلب.
func terminal(status string) bool {
	switch status {
	case StDelivered, StRejected, StCancelled, StFailed, StRefunded:
		return true
	}
	return false
}

// refundable الحالات النهائية التي تعيد المدفوع من المحفظة تلقائياً.
func refundOnEnter(status string) bool {
	switch status {
	case StRejected, StCancelled, StFailed, StRefunded:
		return true
	}
	return false
}

// canTransition يتحقق من شرعية الانتقال لهذه الأدوار.
func canTransition(from, to string, roles []string) bool {
	for _, r := range roles {
		if r == "admin" {
			// الأدمن مخول بكل الانتقالات المعرفة في الخارطة
			for _, t := range allowedTransitions[from] {
				if t.To == to {
					return true
				}
			}
			return false
		}
	}
	for _, t := range allowedTransitions[from] {
		if t.To != to {
			continue
		}
		for _, allowed := range t.Roles {
			for _, r := range roles {
				if r == allowed {
					return true
				}
			}
		}
	}
	return false
}
