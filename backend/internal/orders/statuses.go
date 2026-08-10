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

// statusAr اسمُ الحالة بالعربيّة — **لِما يُكتب لصاحب الحساب لا للسجلّ.**
//
// (شهده المالك ٢٠٢٦-٠٨-٠٧ في محفظته: «استرجاع طلب (cancelled) — فيه كلمةٌ
// إنكليزيّةٌ غيرُ مفهومة».)
//
// **وملاحظةُ الحركة تُقرأ في المحفظة** — لا في سجلّ مطوّر. وكانت تُبنى
// بـ%s من ثابت الحالة، **وثوابتُ الحالات إنكليزيّةٌ لأنّها مفاتيحُ قاعدةِ
// بيانات** لا نصوصاً.
//
// **ولا يُترجَم هنا شيءٌ آخر**: هذه هي النصوصُ الوحيدةُ التي يكتبها المحرّك
// في المال، **وما عداها يأتي من المعجم في الواجهة.**
func statusAr(status string) string {
	switch status {
	case StPending:
		return "بانتظار القبول"
	case StAccepted:
		return "مقبول"
	case StPreparing:
		return "قيد التحضير"
	case StDispatching:
		return "بحثٌ عن سائق"
	case StAssigned:
		return "مُسنَد"
	case StAtPickup:
		return "عند المتجر"
	case StPickedUp:
		return "استُلم من المتجر"
	case StOnTheWay:
		return "في الطريق"
	case StAtDropoff:
		return "عند العنوان"
	case StDelivered:
		return "مُسلَّم"
	case StRejected:
		return "مرفوض"
	case StCancelled:
		return "ملغى"
	case StFailed:
		return "فاشل"
	case StRefunded:
		return "مُسترَجع"
	}
	// **وحالةٌ لا اسمَ لها تُكتب كما هي** — نصٌّ غريبٌ أهونُ من نصٍّ ناقص،
	// **وحارسٌ في الاختبار يمنع أن تمرّ حالةٌ جديدةٌ بلا اسم.**
	return status
}

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
		// **صفٌّ واحدٌ لثلاثة أدوار — لا ثلاثةُ صفوف.**
		//
		// كانا سطرين: `{cancelled, [customer]}` و`{cancelled, merchantOps}`.
		// **والخارطةُ تقرأ الأوّلَ الذي يطابق فيعمل الاثنان**، لكنّ الجدولَ
		// المولَّد من الخارطة يعرضهما صفّين متطابقين — **فيبدو أنّ للانتقال
		// معنيين وهو واحد.**
		//
		// ومعانيها الثلاثة تبقى كما هي:
		//   - **الزبون**: نافذةُ تدارُكٍ مدّتها في الإعدادات، تُفرض في المحرّك
		//   - **المتجر**: نفد صنفٌ أو تعطّل مطبخُه — **والواقعُ اليوميّ يفرضه**،
		//     وبلا هذا يتّصل بالمنصة ليُلغى بالنيابة عنه فيضيع الوقتُ والسبب
		//   - **العمليات**: قبل التحويل، وطلبٌ لم يعلم به مطبخ
		//
		// **والسببُ إلزاميٌّ على المتجر** — يُفرض في المعالِج لا في الخارطة.
		{StCancelled, []string{"customer", "merchant", "ops"}},
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
	// **وما بعد الاستلام كان بلا مخرجٍ ألبتّة.**
	//
	// `picked_up` لا تؤدّي إلّا إلى `on_the_way`، وتلك لا تؤدّي إلّا إلى
	// `at_dropoff`. **فسائقٌ اختفى بطلبٍ في يده يترك الطلبَ عالقاً إلى الأبد**:
	// لا يُلغى، ولا يُفشل، ولا يُسنَد لغيره. **والزبونُ ينتظر طعاماً لن يأتي
	// ولا أحد يملك أن يُنهي انتظارَه.**
	//
	// وهي العلّةُ نفسُها التي كانت في `at_pickup` — **بابٌ يُدخَل منه ولا
	// يُخرَج**، والفرقُ أن هذه البضاعةُ فيها خرجت فعلاً.
	//
	// **والمخارجُ للعمليات وحدَها**: التحريرُ بعد الاستلام قرارُ منصةٍ لا
	// قرارُ سائق — **وإلّا لَترك كلُّ من ثقل عليه طلبٌ طلبَه.**
	StPickedUp: {
		{StOnTheWay, driverOps},
		{StDispatching, opsRoles},
		{StFailed, opsRoles},
		{StCancelled, opsRoles},
	},
	StOnTheWay: {
		{StAtDropoff, driverOps},
		{StDispatching, opsRoles},
		{StFailed, opsRoles},
		{StCancelled, opsRoles},
	},
	StAtDropoff: {
		{StDelivered, driverOps},
		{StFailed, driverOps}, // زبون لا يرد / يرفض الاستلام
		{StDispatching, opsRoles},
		{StCancelled, opsRoles},
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
func authorizingRole(kind, from, to string, roles []string) string {
	for _, r := range roles {
		if canTransition(kind, from, to, []string{r}) {
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
// customTransitions **خارطةُ الطلب الخاصّ — بلا «مقبول» ولا «تحضير».**
//
// (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «لا يوجد تحويل للمتجر، هذا خطأ — لأنّه بالأساس
//
//	الطلبُ ليس من متجر».)
//
// # ولماذا خارطةٌ ثانية
//
// **«مقبول» تعني قَبِله المتجر، و«تحضير» تعني يطبخه** — **ولا متجرَ في الطلب
// الخاصّ.** فحالتان لا معنى لهما، **وضغطتان على الإدارة بلا سبب**: تقبل نيابةً
// عن لا أحد، ثمّ تحوّل.
//
// **وحالةٌ بلا معنًى ليست زائدةً فقط** — تُقرأ خبراً كاذباً: من رأى «قيد
// التحضير» ظنّ أنّ مطبخاً يعمل، **ولا مطبخَ ولا سائقَ بعد.**
//
// **فيمضي من «معلَّق» إلى الطابور رأساً** بموافقة الإدارة وحدَها.
var customTransitions = map[string][]transition{
	StPending: {
		// **موافقةُ الإدارة تُنزله الطابور** — خطوةٌ واحدةٌ لا اثنتان.
		{StDispatching, opsRoles},
		{StRejected, opsRoles},
		{StCancelled, []string{"customer", "ops"}},
	},
	// ══════════════════════════════════════════════════════════════════
	// **ويُلغيه صاحبُه ما لم يُشترَ بعد**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شهده المالك ٢٠٢٦-٠٨-١٠: «الإلغاءُ لم يُطبَّق على الطلبات الخاصّة
	//  بنفس الأسلوب — لا يوجد إلغاءُ طلب».)
	//
	// **كان يملك الإلغاءَ ما دامت الإدارةُ لم توافق، ثمّ يُقفل البابُ إلى
	// الأبد**: الخارطةُ لا تذكر «معلَّق ← الطابور»، فتسقط على العامّة —
	// **وفيها الإلغاءُ للعمليات وحدَها.**
	//
	// # ولماذا حتّى الشراء لا أقلّ
	//
	// **لا شيءَ اشتُري قبل ضغطة «اشتريتُ الطلب»** — لا مالَ خرج من جيب
	// أحد. **والإلغاءُ ساعتَئذٍ يكلّف السائقَ وقتاً، والمنعُ يكلّفه بضاعةً.**
	//
	// **ومن مُنع من الإلغاء لا يقبل الطلبَ عند الباب** — يرفضه، **والسائقُ
	// قد اشترى بماله.** فالمنعُ لا يحمي السائقَ، **إنّما يؤخّر الرفضَ إلى
	// ما بعد الخسارة.**
	//
	// # ولا مهلةَ زمنيّة
	//
	// **مهلةُ العاديّ تحرس مطبخاً بدأ يطبخ** — ولا مطبخَ هنا. **والحدُّ
	// الطبيعيُّ حدثٌ لا ساعة**: خروجُ المال من جيب السائق.
	StDispatching: {
		{StAssigned, driverOps},
		{StCancelled, []string{"customer", "ops"}},
	},
	// **ولا «وصلتُ إلى المتجر»** — (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «ما في شيء
	// اسمه وصلتُ للمتجر»).
	//
	// **لا متجرَ يقف عنده.** والمرحلةُ بعد الإسناد محادثةٌ واتّفاق، **ثمّ
	// يشتري** — فيمضي من «أُسند» إلى «اشتريتُ» رأساً.
	StAssigned: {
		{StPickedUp, []string{"driver"}},
		{StDispatching, []string{"driver", "ops"}},
		{StCancelled, []string{"customer", "ops"}},
	},
	// **وبعد الشراء لا إلغاءَ لصاحبه** — (ولا يُذكر هنا فيسقط على العامّة:
	// `StPickedUp` فيها الإلغاءُ للعمليات وحدَها). **مالُ السائق خرج**،
	// **ومن ألغى بعده ترك بضاعةً في يدٍ دفعت ثمنَها.**
}

// merchantOnlyStates **حالاتٌ لا وجودَ لها في الطلب الخاصّ.**
//
// **«مقبول» فعلُ متجرٍ و«تحضير» فعلُ مطبخ** — ولا واحدَ منهما هنا.
//
// **ولا يبلغهما الخاصُّ أصلاً** (خارطتُه تمضي من «معلَّق» إلى الطابور)،
// **لكنّ خارطةً تقول «نعم» عن حالةٍ لا تُبلَغ فخٌّ**: يُضاف بابٌ يوماً فيمرّ
// منه ما لا يجب، **ولا يُنتبَه لأنّ الشرطَ كان مكتوباً «نعم» منذ البداية.**
var merchantOnlyStates = map[string]bool{StAccepted: true, StPreparing: true}

// transitionsFor **خارطةُ هذا الطلب** — بحسب نوعه.
//
// **وما لم تُعرَّف له حالةٌ خاصّةٌ يرث العامّة**: مراحلُ الطريق واحدةٌ في
// النوعين — يستلم ويمضي ويسلّم. **والفرقُ في أوّله لا في طريقه.**
func transitionsFor(kind, from string) []transition {
	if kind == KindCustom {
		if ts, ok := customTransitions[from]; ok {
			return ts
		}
		// **ولا يُسمح بما لا معنى له فيه.**
		if merchantOnlyStates[from] {
			return nil
		}
		var out []transition
		for _, t := range allowedTransitions[from] {
			if !merchantOnlyStates[t.To] {
				out = append(out, t)
			}
		}
		return out
	}
	return allowedTransitions[from]
}

func canTransition(kind, from, to string, roles []string) bool {
	for _, r := range roles {
		if r == "admin" {
			// الأدمن مخول بكل الانتقالات المعرفة في الخارطة
			for _, t := range transitionsFor(kind, from) {
				if t.To == to {
					return true
				}
			}
			return false
		}
	}
	for _, t := range transitionsFor(kind, from) {
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

// أنواعُ الطلب.
//
// **والخاصُّ خدمةٌ للسائق لا بيعٌ للمنصّة** (قرارُ المالك ٢٠٢٦-٠٨-٠٩): يدفع
// من جيبه ويستردّ عند التسليم، **والمنصّةُ توثّق ولا تحاسب.**
const (
	// KindStandard طلبٌ من متجرٍ في المنصّة — له أصنافٌ وأسعارٌ وعمولة.
	KindStandard = "standard"
	// KindCustom طلبٌ خاصّ — يصفه الزبونُ بلفظه، ولا متجرَ له ولا سعرَ عند
	// إنشائه. **ولا أثرَ ماليَّ له في دفتر المنصّة.**
	KindCustom = "custom"
)
