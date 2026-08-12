package orders

// ══════════════════════════════════════════════════════════════════════
// **مراحلُ الرحلة — تعريفٌ واحدٌ لثلاث شاشات**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «لازم نعمل تزامن بين حساب الزبون على ويب
//  والأدمن والسائق على تطبيق… والأفضل يكون بشكلٍ مركزيّ، مو كلّ صفحةٍ
//  تاخذ من مكانٍ مختلف».)
//
// # ما المشكلة التي يحلّها هذا الملفّ
//
// **حالاتُ المحرّك أربعَ عشرة، ومراحلُ الرحلة التي يفهمها الناسُ ستّ.**
// وكلُّ شاشةٍ كانت تطوي الأربعَ عشرة إلى ستٍّ **بجدولٍ عندها**:
//
//	شاشةُ الزبون   →  `trackIndex()` في صفحة الطلبات
//	شاشةُ السائق   →  `legOf()` في تطبيق أندرويد
//	لوحةُ الإدارة  →  لا شيء — لم تكن تعرضها أصلا
//
// **وثلاثةُ جداولَ لشيءٍ واحدٍ تفترق يوما**: يُضاف حالٌ جديدٌ في المحرّك
// فيُدرَج في اثنين ويُنسى في الثالث، **فيقرأ الزبونُ «في الطريق» وتقرأ
// الإدارةُ «وصل»** — ولا أحدَ يعرف أيُّهما الصواب.
//
// # ولماذا مفتاحٌ لا نصّ
//
// **الاسمُ المكتوب يبقى في المعجم** (`packages/i18n` و`strings.xml`) —
// فالمحرّكُ لا يعرف لغةَ من يقرأ، **ولا يُنقَل نصٌّ عربيٌّ في JSON ليُعرض
// كما هو.**
//
// **والذي يُوحَّد هو الطيّ**: أيُّ حالٍ يقع في أيّ مرحلة. **وهو ما يفترق،
// لا الكلمات.**
//
// # والزبونُ لا يرى المتجرَ ولا السائق
//
// (قرارُ المالك: «الزبون ما يهمّه أنّ السائق راح على متجر أو لا، يهمّه
//  يعرف حالة طلبه وبس».)
//
// **فأربعُ حالاتٍ تنطوي عنده في واحدة**: `preparing` و`dispatching`
// و`assigned` و`at_pickup` كلُّها **«قيد التجهيز»** — وهي بتعريف المالك:
// **من لحظة وصول الطلب إلى المتجر حتّى يستلمه السائق.**
//
// **والإدارةُ تراها مفصّلةً** — هي التي تعرف أين ضاع الوقت: **في المطبخ
// أم في انتظار سائق.**

// Stage مرحلةٌ في رحلة الطلب — **مفتاحٌ يُترجَم في الواجهة.**
type Stage string

const (
	// StageWaiting **بانتظار قبول المتجر** — لم يبدأ شيء بعد.
	StageWaiting Stage = "waiting"
	// StageAccepted **قَبِله المتجر** — لحظةٌ لا مدّة.
	StageAccepted Stage = "accepted"
	// StagePreparing **قيد التجهيز** — من دخوله المتجرَ حتّى يستلمه السائق.
	StagePreparing Stage = "preparing"
	// StageOnTheWay **في الطريق** — البضاعةُ مع السائق.
	StageOnTheWay Stage = "on_the_way"
	// StageArrived **وصل** — السائقُ عند الباب ولم يُسلّم بعد.
	StageArrived Stage = "arrived"
	// StageDelivered **تمّ التسليم.**
	StageDelivered Stage = "delivered"
	// StageEnded **انتهى قبل أن يصل** — ملغًى أو مرفوضٌ أو متعذّر.
	//
	// **ولا موضعَ له على المسار**: شريطٌ يقف في منتصفه يُقرأ «عالق» لا
	// «انتهى» — **فيُقال بالحرف أين توقّف ولماذا.**
	StageEnded Stage = "ended"
)

// CustomerStages المراحلُ الستُّ بترتيبها — **لِما يُرسم شريطا.**
func CustomerStages() []Stage {
	return []Stage{
		StageWaiting, StageAccepted, StagePreparing,
		StageOnTheWay, StageArrived, StageDelivered,
	}
}

// StageOf أيُّ مرحلةٍ يقع فيها هذا الحال.
//
// **وحالٌ مجهولٌ يُعدّ منتهيا** لا «بانتظار»: **من عرض حالاً لا يعرفه في
// أوّل المسار** أوهم صاحبَه أنّ طلبَه لم يبدأ وقد انتهى.
func StageOf(status string) Stage {
	switch status {
	case StPending:
		return StageWaiting
	case StAccepted:
		return StageAccepted
	// **الأربعةُ مرحلةٌ واحدةٌ عند الزبون** — بتعريف المالك.
	case StPreparing, StDispatching, StAssigned, StAtPickup:
		return StagePreparing
	case StPickedUp, StOnTheWay:
		return StageOnTheWay
	case StAtDropoff:
		return StageArrived
	case StDelivered:
		return StageDelivered
	default:
		return StageEnded
	}
}

// StageIndex موضعُ الحال على الشريط — **و`-1` لما لا مسارَ له.**
func StageIndex(status string) int {
	at := StageOf(status)
	if at == StageEnded {
		return -1
	}
	for i, s := range CustomerStages() {
		if s == at {
			return i
		}
	}
	return -1
}

// ══════════════════════════════════════════════════════════════════════
// **وللطلب الخاصّ مسارٌ آخرُ — لا متجرَ فيه**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «بالطلب الخاصّ ما في شيءٌ اسمه قيد التحضير…
//  تمّ الشراء برأيي».)
//
// **«قيد التجهيز» تعني أنّ أحداً يطبخ** — **ولا مطبخَ هنا**: السائقُ
// يشتريه بنفسه. **وخبرٌ كاذبٌ أسوأُ من لا خبر**: من اطمأنّ كاذباً ينتظر.
//
// **و«اشتُري» موضعُها بعد ضغطة السائق** — هي إقرارٌ بأنّ مالَه خرج من
// جيبه، **وعلامةٌ تسبقه تَعِد الزبونَ بما لم يقع.**

const (
	// StageSeeking **يُبحث له عن سائق.**
	StageSeeking Stage = "seeking"
	// StageBought **اشترى السائقُ الطلب** — بماله.
	StageBought Stage = "bought"
)

// CustomStages مسارُ الطلب الخاصّ.
func CustomStages() []Stage {
	return []Stage{
		StageWaiting, StageSeeking, StageBought,
		StageOnTheWay, StageArrived, StageDelivered,
	}
}

// CustomStageOf مرحلةُ الطلب الخاصّ.
func CustomStageOf(status string) Stage {
	switch status {
	case StPending:
		return StageWaiting
	case StAccepted, StPreparing, StDispatching, StAssigned, StAtPickup:
		return StageSeeking
	case StPickedUp:
		return StageBought
	case StOnTheWay:
		return StageOnTheWay
	case StAtDropoff:
		return StageArrived
	case StDelivered:
		return StageDelivered
	default:
		return StageEnded
	}
}

// StagesFor المسارُ ومفتاحُ الموضع — **بحسب نوع الطلب.**
func StagesFor(kind, status string) ([]Stage, int) {
	list := CustomerStages()
	at := StageOf(status)
	if kind == "custom" {
		list = CustomStages()
		at = CustomStageOf(status)
	}
	if at == StageEnded {
		return list, -1
	}
	for i, s := range list {
		if s == at {
			return list, i
		}
	}
	return list, -1
}

// ══════════════════════════════════════════════════════════════════════
// **ومسارُ المكتب أطولُ — هو الذي يعرف أين ضاع الوقت**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «الكلّ معنيّ بالرحلة، مو طرفٌ واحد».)
//
// **والزبونُ يطوي أربعَ حالاتٍ في «قيد التجهيز»** لأنّه لا يفرّق بينها
// ولا يملك حيالها شيئا. **والمكتبُ يفرّق**: طلبٌ واقفٌ في المطبخ يُتّصل
// فيه بالمتجر، **وطلبٌ واقفٌ بلا سائقٍ يُسنَد بيد** — **وهما تحت اسمٍ
// واحدٍ عند الزبون.**
//
// **و«بانتظار سائق» أهمُّ حالٍ في لوحة العمليات**: هي وحدَها التي تُوجب
// فعلاً منهم الآن.

const (
	// StageSeekingDriver **بانتظار سائق** — لا أحدَ أخذه بعد.
	StageSeekingDriver Stage = "seeking_driver"
	// StageToStore **السائقُ في طريقه إلى المتجر** أو عنده.
	StageToStore Stage = "to_store"
)

// OpsStages مسارُ الطلب كما يراه المكتب.
func OpsStages() []Stage {
	return []Stage{
		StageWaiting, StagePreparing, StageSeekingDriver,
		StageToStore, StageOnTheWay, StageArrived, StageDelivered,
	}
}

// OpsStageOf مرحلةُ الطلب في لوحة العمليات.
func OpsStageOf(status string) Stage {
	switch status {
	case StPending:
		return StageWaiting
	// **والقبولُ والتحضيرُ واحدٌ عند المكتب**: كلاهما «عند المتجر
	// الآن» — **والفرقُ بينهما ضغطةُ زرٍّ من المتجر لا حالُ الطلب.**
	case StAccepted, StPreparing:
		return StagePreparing
	case StDispatching:
		return StageSeekingDriver
	case StAssigned, StAtPickup:
		return StageToStore
	case StPickedUp, StOnTheWay:
		return StageOnTheWay
	case StAtDropoff:
		return StageArrived
	case StDelivered:
		return StageDelivered
	default:
		return StageEnded
	}
}

// OpsStageIndex موضعُه على مسار المكتب — **و`-1` لما انتهى قبل أن يصل.**
func OpsStageIndex(status string) int {
	at := OpsStageOf(status)
	if at == StageEnded {
		return -1
	}
	for i, s := range OpsStages() {
		if s == at {
			return i
		}
	}
	return -1
}
