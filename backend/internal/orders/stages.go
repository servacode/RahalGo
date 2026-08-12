package orders

import "time"

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

// ══════════════════════════════════════════════════════════════════════
// **ومتى دخل كلَّ مرحلة — لا أين هو فقط**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «يجب أن نضع الوقت في منتصف الخطّ… الساعة
//  والدقيقة… لنعرف أين الوقتُ الضائع».)
//
// # ورقمُ المرحلة وحدَه لا يقول شيئا
//
// **`ops_stage_at = 3` يقول «هو في الطريق»** — ولا يقول أوقف ساعةً في
// المطبخ أم دقيقتين، **ولا كم بقي بلا سائق.** والمكتبُ يُسأل عن
// التأخير لا عن الموضع.
//
// # والأوقاتُ في الأحداث لا في الأعمدة
//
// **جدولُ `orders` يحمل خمسةً من السبعة** (`accepted_at`
// و`dispatched_at` و`picked_up_at` و`delivered_at` و`created_at`) —
// **ويغيب عمودان**: متى أُسنِد السائقُ ومتى وصل الباب.
//
// **ولا يُضاف عمودان** — `order_events` يسجّل كلَّ انتقالٍ أصلاً،
// **وعمودٌ يُضاف اليوم لمرحلةٍ يُنسى غداً لمرحلةٍ ثانية.** والطيُّ من
// الأحداث يُعطي أيَّ مرحلةٍ تُضاف وقتَها بلا هجرة.
//
// # وأوّلُ دخولٍ لا آخره
//
// **الطلبُ يُعاد إلى الطابور فيدخل «بانتظار سائق» مرّتين.** وسؤالُ
// المكتب «متى بدأ الانتظار» لا «متى انتهت آخرُ محاولة» — **فيُثبَّت
// الأوّلُ ولا يُدهَس بما بعده.**

// lastEntryWins المراحلُ التي يُؤخذ فيها آخرُ دخولٍ لا أوّله.
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا تفترق هذه عن سائرها**
// ══════════════════════════════════════════════════════════════════════
//
// (كشفه جردُ الحالات ٢٠٢٦-٠٨-١٣، وأقرّ المالكُ إصلاحَه.)
//
// **الأصلُ أوّلُ دخول** — لأنّ سؤال المكتب «متى بدأ الانتظار» لا «متى
// انتهت آخرُ محاولة». **وطلبٌ رُدّ إلى الطابور لو أخذ وقتَ عودته لَبدا
// كأنّه نزل لتوّه** وهو واقفٌ فيه منذ نصف ساعة.
//
// **إلّا مرحلةَ الإسناد.** فالسطرُ عليها يحمل **اسمَ من يحمل الطلب
// الآن** — وهو الأخير. **فلو حمل وقتَ الأوّل لَجمع السطرُ رجلين**:
// اسمُ من يحمله ووقتُ من تركه.
//
// **وفي شكوى، هذا بعينه ما يقلب الحقيقة**: يُسأل سائقٌ عن ساعةٍ لم
// يكن فيها صاحبَ الطلب.
//
// **ولا يضيع قياسُ الانتظار**: «بانتظار سائق» تبقى على أوّل نزولٍ
// للطابور، **فالمدّةُ على الخطّ تقول كم انتظر حتّى أخذه من أبقاه** —
// وهي الحقيقةُ كاملةً لا نصفَها.
func lastEntryWins(st Stage) bool {
	// **وفي المسار الخاصّ `agreeing` هي مرحلةُ الإسناد** — يدخلها
	// بأخذه الطلبَ ويخرج منها بتوثيقه.
	return st == StageToStore || st == StageAgreeing
}

// **و«آخرُ دخول» يعني آخرَ عبورٍ إليها من خارجها — لا آخرَ حدثٍ فيها.**
//
// **وأمسكه الاختبارُ قبل أن يُدفَع**: `at_pickup` يقع في «السائق إلى
// المتجر» نفسِها (وصل ولم يستلم بعد). **فلو أُخذ آخرُ حدثٍ لَصار وقتُ
// المرحلة وقتَ وصوله المتجرَ لا وقتَ إسناده** — فتُقرأ «أُسنِد ٩:٣١»
// وقد أُسنِد ٩:٢٥، **وتختفي ستُّ دقائقَ من طريقه.**
//
// **فما يُعدّ دخولاً هو ما سبقته مرحلةٌ أخرى.**

// StageTimes أوقاتُ دخول كلِّ مرحلةٍ على مسارٍ ما.
//
// **والفارغُ يعني مرحلةً لم تُبلَغ** — لا مرحلةً بلا وقت.
type StageTimes []*time.Time

// OpsStageTimes يطوي انتقالاتِ الطلب إلى أوقاتِ مراحل المكتب.
//
// **والانتقالاتُ مرتّبةٌ زمنيّا** — كما تخرج من `order_events` بترتيب
// المعرّف. **ومن مرّرها غيرَ مرتّبةٍ يقرأ أوّلَ ما ورد لا أوّلَ ما وقع.**
func OpsStageTimes(evs []Event) StageTimes {
	out := make(StageTimes, len(OpsStages()))
	list := OpsStages()
	prev := -1
	for i := range evs {
		at := OpsStageIndex(evs[i].ToStatus)
		// **وما لا موضعَ له على المسار يُتخطّى** — الإلغاءُ والرفضُ
		// **ليسا مرحلةً بلغها الطلب** بل نهايةً وقعت به. **ولا يقطع
		// التسلسلَ أيضا**: حالٌ لا مرحلةَ له لا يجعل ما بعده «دخولا».
		if at < 0 {
			continue
		}
		entry := at != prev
		prev = at
		if !entry {
			continue
		}
		if out[at] != nil && !lastEntryWins(list[at]) {
			continue
		}
		t := evs[i].CreatedAt
		out[at] = &t
	}
	return out
}

// ══════════════════════════════════════════════════════════════════════
// **وهل تجاوز كلُّ خطٍّ مهلتَه — وأين وقع التأخير**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «نكتب الوقت المستهلك وإشارة صحّ أنّ الزمن
//  طبيعيّ، وإذا كان هناك تأخير نضع علامة إكس حمراء لنعرف أين حصل
//  التأخير… وخصوصاً إذا أتت شكوى على الطلب، نظرةٌ واحدة على السجلّ تحلّ
//  كلّ الخلاف».)
//
// # والمدّةُ حقيقةٌ والحكمُ تقدير
//
// **المدّةُ بين مرحلتين لا تُناقَش**: طابعا وقتٍ كتبهما المحرّك. **وأمّا
// «أهذا تأخير» فيحتاج مهلةً يضبطها إنسان** — ومن قرأ العلامةَ حكماً
// نهائيّاً **خصم من بريءٍ يومَ تكون المهلةُ ضيّقةً على متجرٍ بعينه.**
//
// **فالعلامةُ تدلّ العينَ على موضع الوقت الضائع، والرقمُ تحتها يقرّر.**
//
// # ولا حكمَ حيث لا مهلة
//
// **خطٌّ بلا مهلةٍ يبقى بلا علامة** — لا «✓» ولا «✗». **وصحٌّ يُكتب بلا
// معيار** يقول «كلُّ شيءٍ سليم» وهو لا يعرف، **وهو أسوأُ من صمت.**
//
// **وخطّا الطريق بلا مهلةٍ اليوم عمدا**: المسافةُ المحفوظة خطٌّ مستقيمٌ
// بين نقطتين (`ST_Distance`) **لا طريقُ شوارع** — والشارعُ أطولُ منه
// بالثلث عادةً. **فمهلةٌ محسوبةٌ منه تتّهم السائقَ بما لم يفعل.**

// StageLimits المهلُ المسموحة — **بالدقائق، والصفرُ «لا مهلةَ لهذا الخطّ».**
type StageLimits struct {
	// Accept **من وصول الطلب إلى قبول المتجر.**
	Accept int
	// Prep **من القبول إلى خروجه للطابور** — مهلةُ المطبخ.
	Prep int
	// Driver **من نزوله الطابورَ إلى أن يأخذه سائق.**
	Driver int
	// Handover **من وقوف السائق عند الباب إلى التسليم.**
	Handover int
	// ══════════════════════════════════════════════════════════════════
	// **ومهلتا الطريق مُدَّتان لا دقائق — لكلّ طلبٍ مهلتُه**
	// ══════════════════════════════════════════════════════════════════
	//
	// **رقمٌ واحدٌ لكلّ الطلبات لا يصلح هنا**: طريقٌ من ثلاثمئة متر
	// وآخرُ من أربعة كيلومترات **تحت مهلةٍ واحدة** يسامح الأوّلَ ويظلم
	// الثاني.
	//
	// **فمهلةُ كلّ طلبٍ من زمن خريطته** — يُلتقط عند الإسناد وعند
	// الاستلام — **مضروباً بهامشٍ يضبطه المالك.**
	//
	// **وصفرُهما «لا حكم»**: خريطةٌ لم تُسأل أو نقطةٌ بلا إحداثيّ —
	// **ولا يُوسَم أحدٌ بتأخيرٍ لم يُقَس.**
	ToStore time.Duration
	ToDoor  time.Duration
	// RouteMarginPct **كم يُسمح للطريق أن يزيد على تقدير الخريطة** — ٪.
	//
	// **والخريطةُ لا تعرف ازدحاماً ولا حاجزاً ولا شارعاً مقطوعا**، وتحسب
	// على سرعاتٍ مفترضةٍ للشوارع. **فتقديرُها أرضيّةٌ لا سقف.**
	RouteMarginPct int
}

// limitFor مهلةُ الدخول إلى هذه المرحلة — **وصفرٌ يعني لا حكم.**
//
// **وبالمفتاح لا بالرقم**: من أعاد ترتيبَ المراحل يوماً **لا ينقل مهلةَ
// المطبخ إلى الطابور** بلا أن ينتبه.
func limitFor(st Stage, lim StageLimits) time.Duration {
	min := func(n int) time.Duration { return time.Duration(n) * time.Minute }
	switch st {
	case StagePreparing:
		return min(lim.Accept)
	case StageSeekingDriver:
		return min(lim.Prep)
	case StageToStore:
		return min(lim.Driver)
	// **والخطُّ إلى «في الطريق» هو مشوارُ السائق إلى المتجر** — يبدأ
	// بإسناده وينتهي باستلامه، **وفيه وقوفُه عند الباب أيضاً.**
	case StageOnTheWay:
		return lim.ToStore
	// **والخطُّ إلى «وصل» هو مشوارُ التوصيل نفسُه.**
	case StageArrived:
		return lim.ToDoor
	case StageDelivered:
		return min(lim.Handover)
	// ══════════════════════════════════════════════════════════════════
	// **وللطلب الخاصّ مهلتان فقط**
	// ══════════════════════════════════════════════════════════════════
	//
	// **«بانتظار سائق» تأخذ مهلةَ السائقين نفسَها** — الطابورُ واحدٌ
	// للنوعين، وهي مذكورةٌ فوق.
	//
	// **وأمّا «التوثيق» و«الشراء» فبلا مهلة**: الاتّفاقُ حديثٌ بين
	// اثنين قد يطول بسؤالٍ وجواب، **والشراءُ يمرّ بسوقٍ لا نعرف
	// طريقَه** — والمنصّةُ لا تعرف من أين يشتري. **ومهلةٌ نضعها
	// تخميناً تُحمّر بطاقاتٍ بلا معنى.**
	default:
		return 0
	}
}

// routeLimit مهلةُ طريقٍ من زمن الخريطة وهامشِ المالك.
//
// **والهامشُ نسبةٌ لا دقائق**: طريقٌ من دقيقتين وآخرُ من عشرين —
// **وخمسُ دقائقَ زيادةً تسامح الأوّلَ أضعافاً وتضيّق على الثاني.**
// والنسبةُ تكبر بكبر الطريق كما يكبر احتمالُ ما يعطّله.
//
// **وصفرٌ يعني «لم تُقَس»** — فلا حكم.
func routeLimit(etaSec *int, marginPct int) time.Duration {
	if etaSec == nil || *etaSec <= 0 {
		return 0
	}
	return time.Duration(*etaSec) * time.Second *
		time.Duration(100+marginPct) / 100
}

// OpsCustomStageTimes أوقاتُ مراحل الطلب الخاصّ في لوحة العمليات.
//
// **ولا تُطوى من الأحداث وحدَها**: «توثيق» و«شراء» كلتاهما تحت الحال
// `assigned` — **والذي يفصلهما طابعُ `custom_agreed_at` لا حدثٌ في
// السجلّ.**
//
// **فوقتُ «الشراء» هو لحظةُ التوثيق** — لأنّها اللحظةُ التي صار فيها
// الشراءُ مسموحاً وبدأ.
func OpsCustomStageTimes(evs []Event, agreedAt *time.Time) StageTimes {
	list := OpsCustomStages()
	out := make(StageTimes, len(list))
	// **وموضعُ كلّ مرحلةٍ يُبحث عنه بالمفتاح** — فإن أُعيد ترتيبُ المسار
	// **لم ينتقل وقتٌ إلى غير صاحبه.**
	idx := map[Stage]int{}
	for i, st := range list {
		idx[st] = i
	}
	prev := Stage("")
	put := func(st Stage, t time.Time) {
		i, ok := idx[st]
		if !ok {
			return
		}
		entry := st != prev
		prev = st
		if !entry {
			return
		}
		if out[i] != nil && !lastEntryWins(st) {
			return
		}
		v := t
		out[i] = &v
	}
	for i := range evs {
		switch evs[i].ToStatus {
		case StPending:
			put(StageWaitingPlatform, evs[i].CreatedAt)
		case StDispatching:
			put(StageSeekingDriver, evs[i].CreatedAt)
		case StAssigned:
			put(StageAgreeing, evs[i].CreatedAt)
		case StPickedUp:
			put(StageOnTheWay, evs[i].CreatedAt)
		case StAtDropoff:
			put(StageArrived, evs[i].CreatedAt)
		case StDelivered:
			put(StageDelivered, evs[i].CreatedAt)
		}
	}
	if agreedAt != nil {
		put(StageBuying, *agreedAt)
	}
	return out
}

// OpsStageLate أتجاوز الانتقالُ إلى كلّ مرحلةٍ مهلتَه.
//
// **والفارغُ «لا حكم»** — إمّا لأنّ المرحلةَ لم تُبلَغ، أو لأنّ خطَّها بلا
// مهلة. **وهو غيرُ `false`**: تلك تقول «قُيس فكان سليما».
func OpsStageLate(times StageTimes, lim StageLimits) []*bool {
	return opsStageLate(OpsStages(), times, lim)
}

// opsStageLate الحكمُ على مسارٍ بعينه — **ولا يفترض مساراً واحدا.**
func opsStageLate(list []Stage, times StageTimes, lim StageLimits) []*bool {
	out := make([]*bool, len(list))
	for i, st := range list {
		max := limitFor(st, lim)
		// **والأوّلُ بلا خطٍّ فوقه** — لا انتقالَ إليه يُقاس.
		if i == 0 || max <= 0 || i >= len(times) {
			continue
		}
		to, from := times[i], times[i-1]
		if to == nil || from == nil {
			continue
		}
		v := to.Sub(*from) > max
		out[i] = &v
	}
	return out
}

// ══════════════════════════════════════════════════════════════════════
// **وللطلب الخاصّ مسارُ مكتبٍ آخرُ — لا متجرَ فيه أصلا**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٣، بعد أن جرّب طلباً خاصّاً ورأى بطاقتَه:
//  «شوف في قيد التجهيز بالمتجر غلط — لأنّ أوّل شي لازم الطلب ياخذه
//  سائق، بعدين يوثّق السعر وأجرة التوصيل، بعدين السائق يجيب الطلب،
//  بعدها بالطريق إلى الزبون، بعدها وصل الزبون، بعدها يسلّم الطلب».)
//
// # وثلاثُ مراحلَ كانت تتحدّث عن متجرٍ لا وجود له
//
// **`OpsStages()` كان مساراً واحداً للجميع**، فبطاقةُ الطلب الخاصّ تقول
// «بانتظار قبول المتجر» و«قيد التجهيز في المتجر» و«السائق إلى المتجر»
// — **ولا متجرَ في الطلب الخاصّ**: السائقُ يشتريه بنفسه من حيث وجده.
//
// **وخبرٌ كاذبٌ في سجلٍّ يُحتجّ به أخطرُ من سجلٍّ ناقص.**
//
// # ولماذا «التوثيق» و«الشراء» نقطتان
//
// **بينهما ضغطةٌ حقيقيّة**: السائقُ يتّفق مع الزبون ويوثّق المبلغ
// والأجرة، **والمحرّكُ يمنع الشراء قبلها** (`ErrCustomNotAgreed`).
//
// **ولو جُمعتا لَما عُرف أين هو الآن**: أيتّفق أم يشتري بماله. **وهما
// سؤالان مختلفان حين يتأخّر طلب.**
//
// # والتوثيقُ طابعُ وقتٍ لا حال
//
// **`custom_agreed_at` ليس حالاً في المحرّك** — فطيُّ هذا المسار يقرأ
// الحالَ والطابعَ معاً. **وهو الموضعُ الوحيدُ الذي لا يكفي فيه الحال.**

const (
	// StageWaitingPlatform **بانتظار موافقة المنصّة** — لم يُرسَل للسائقين.
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٣: «يجب أن توافق المنصّة بالطبع».)
	//
	// **وهي مرحلةٌ حقيقيّةٌ لا فراغ**: الطلبُ يُنشأ `pending` ولا ينزل
	// الطابورَ وحدَه. **وطلبٌ واقفٌ فيها ساعةً يُسأل عنه المكتب لا
	// السائق.**
	StageWaitingPlatform Stage = "waiting_platform"
	// StageAgreeing **توثيق السعر والأجرة** — أخذه سائقٌ ويتّفق الآن.
	StageAgreeing Stage = "agreeing"
	// StageBuying **شراء الطلب** — وثّق ويشتري بماله.
	StageBuying Stage = "buying"
)

// OpsCustomStages مسارُ الطلب الخاصّ كما يراه المكتب.
func OpsCustomStages() []Stage {
	return []Stage{
		StageWaitingPlatform, StageSeekingDriver, StageAgreeing,
		StageBuying, StageOnTheWay, StageArrived, StageDelivered,
	}
}

// OpsCustomStageOf مرحلةُ الطلب الخاصّ في لوحة العمليات.
//
// **و`agreed` لا يُشتقّ من الحال** — هو طابعُ وقتٍ يضعه السائق، **والحالُ
// يبقى `assigned` قبله وبعده.**
func OpsCustomStageOf(status string, agreed bool) Stage {
	switch status {
	case StPending:
		return StageWaitingPlatform
	case StDispatching:
		return StageSeekingDriver
	// **وأخذه سائقٌ فهو يتّفق** — حتّى يوثّق، فيصير يشتري.
	case StAccepted, StPreparing, StAssigned, StAtPickup:
		if agreed {
			return StageBuying
		}
		return StageAgreeing
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

// OpsStagesFor مسارُ المكتب ومفتاحُ الموضع — **بحسب نوع الطلب.**
func OpsStagesFor(kind, status string, agreed bool) ([]Stage, int) {
	list, at := OpsStages(), OpsStageOf(status)
	if kind == KindCustom {
		list, at = OpsCustomStages(), OpsCustomStageOf(status, agreed)
	}
	if at == StageEnded {
		return list, -1
	}
	for i, st := range list {
		if st == at {
			return list, i
		}
	}
	return list, -1
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
