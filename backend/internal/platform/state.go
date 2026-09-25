package platform

// ══════════════════════════════════════════════════════════════════════
// **حالُ الاستقبال — وترتيبُ الأسبقيّة مكتوبٌ لا مُستنتَج** (`PH`)
// ══════════════════════════════════════════════════════════════════════
//
// # ثلاثُ طبقاتٍ لا واحدة
//
//	١ · وضعُ الإطلاق      `launch.customer_orders`   ⇒ `launch_closed`
//	٢ · إيقافٌ مؤقّت       صفُّ `service_closure`      ⇒ `temporarily_unavailable`
//	٣ · جدولُ الدوام       `platform_hours`           ⇒ `platform_closed_now`
//
// **والأولى تبقى حيث هي** — في `requireLaunch` قبل كلّ شيء — **ولا
// تُستبدَل**: **بابٌ لم يُفتح بعدُ ليس بابَ مغلقٍ مؤقّتاً**، **ومن خلط
// بينهما قال للناس «نعود الساعةَ الرابعة» عن بابٍ لا موعدَ لفتحه.**
//
// **وهذه الوحدةُ تحكم الثانيةَ والثالثةَ وحدَهما.**
//
// # ولمَ الإيقافُ المؤقّتُ قبل الجدول
//
// **والصيانةُ تقع في وسط الدوام** — **فلو سُئل الجدولُ أوّلاً لقال
// «مفتوح» ومضى الطلبُ إلى محرّكٍ يُصلَح.** **والأخصُّ يسبق الأعمّ.**

import "time"

// Reason **لِمَ لا يُستقبَل الطلبُ الآن** — وفارغٌ يعني يُستقبَل.
//
// **والرمزُ هو رمزُ الخطأ نفسُه** — **فتترجمه الشاشةُ بخريطتها القائمة،
// ولا نصَّ ثانٍ يُكتب** (القاعدةُ عينُها في `serviceable_reason`).
type Reason string

const (
	// ReasonOpen **يُستقبَل.**
	ReasonOpen Reason = ""
	// ReasonTemporarilyUnavailable **إيقافٌ تشغيليٌّ مؤقّت** — صيانةٌ أو
	// طارئ. **وهو حالٌ يضبطها المالكُ من اللوحة بلا إصدارِ تطبيق.**
	ReasonTemporarilyUnavailable Reason = "temporarily_unavailable"
	// ReasonPlatformClosedNow **خارجَ دوام المنصّة.**
	ReasonPlatformClosedNow Reason = "platform_closed_now"
)

// Closure **إيقافٌ تشغيليٌّ مؤقّت — صفٌّ واحدٌ لا أكثر.**
//
// **وليس وضعَ إطلاق**: `launch.*` **تقول أيُّ عملٍ فُتح للناس أصلاً**،
// **وهذا يقول «مفتوحٌ ولكن ليس الآن».**
type Closure struct {
	// Active **مُفعَّلٌ بيد المالك.**
	Active bool `json:"active"`
	// Message **نصُّ المالك كما يقرؤه الزبون** — عربيٌّ، ويضبطه من اللوحة.
	Message string `json:"message,omitempty"`
	// EndsAt **موعدُ العودة إن كان معلوماً** — و`nil` إن لم يكن.
	//
	// **ولا يُخترَع موعد**: **«نعود قريباً» أصدقُ من ساعةٍ لا يفي بها
	// أحد** — **ومن قال موعداً ولم يفِ فقد ما هو أغلى من ساعة.**
	EndsAt *time.Time `json:"ends_at,omitempty"`
}

// ActiveAt **أهو سارٍ في هذه اللحظة؟**
//
// **وموعدٌ مضى يُنهيه بنفسه** — **ولا مهمّةً دوريّةً تُطفئه**: **مهمّةٌ
// قد تتأخّر أو تسقط، والوقتُ لا يتأخّر.**
func (c Closure) ActiveAt(now time.Time) bool {
	if !c.Active {
		return false
	}
	if c.EndsAt != nil && !now.Before(*c.EndsAt) {
		return false
	}
	return true
}

// State **ما تقوله المنصّةُ عن استقبالها الآن.**
type State struct {
	// OrderingAvailable **أيُقبَل طلبٌ جديدٌ الآن؟**
	//
	// **ولا يشمل وضعَ الإطلاق** — **وهو طبقةٌ فوقَ هذه تُقرأ قبلها.**
	OrderingAvailable bool `json:"ordering_available"`
	// Reason انظر Reason.
	Reason Reason `json:"reason,omitempty"`
	// Message **نصُّ المالك** — للإيقاف المؤقّت وحدَه اليوم.
	Message string `json:"message,omitempty"`
	// NextAvailableAt **أوّلُ لحظةٍ يُقبَل فيها طلبٌ** — و`nil` إن لم
	// تُعرَف.
	NextAvailableAt *time.Time `json:"next_available_at,omitempty"`
	// NextCloseAt **متى يُغلَق الاستقبالُ المفتوحُ الآن** — نهايةُ الفترة
	// الجارية. **يُملأ حين تكون مفتوحةً بجدولٍ سارٍ**، و`nil` حين لا جدولَ
	// سارٍ (مفتوحةٌ بلا حدّ) أو حين تكون مغلقةً. **وبه يُجدّد التطبيقُ
	// نفسَه عند الحدّ دون حدثٍ من الخادم.**
	NextCloseAt *time.Time `json:"next_close_at,omitempty"`
	// ServerTime **لحظةُ الخادم** — **والشاشةُ تنسّقها ولا تحكم بها.**
	ServerTime time.Time `json:"server_time"`
	// Timezone **باسمها** — انظر TZ.
	Timezone string `json:"timezone"`
	// HoursEnforced **أالجدولُ ساري المفعول؟**
	HoursEnforced bool `json:"hours_enforced"`
	// TodayWindows **فتراتُ اليوم** — للعرض.
	TodayWindows []Window `json:"today_windows"`
}

// Decide **القرارُ كلُّه في دالّةٍ خالصة.**
//
// **ولا قاعدةَ ولا ساعةَ نظامٍ ولا إعداد** — **تأخذ ما تحتاجه وتردّ.**
// **وثلاثون حالةً في قائمة المالك تُقاس بها بلا خادمٍ يُشغَّل.**
func Decide(now time.Time, enforced bool, sch Schedule, c Closure) State {
	st := State{
		ServerTime:    now,
		Timezone:      TZ,
		HoursEnforced: enforced,
		TodayWindows:  sch.Today(now),
	}

	// **١ · الإيقافُ المؤقّتُ أخصُّ فيسبق.**
	if c.ActiveAt(now) {
		st.Reason = ReasonTemporarilyUnavailable
		st.Message = c.Message
		// **وموعدُ العودة تقاطعُ الإيقافِ والجدول** — **ومن قال «نعود
		// الثالثة» والدوامُ يبدأ الخامسةَ كذب مرّتين.**
		if c.EndsAt != nil {
			st.NextAvailableAt = availableFrom(*c.EndsAt, enforced, sch)
		}
		return st
	}

	// **٢ · ثمّ الجدول.**
	if enforced && !sch.OpenAt(now) {
		st.Reason = ReasonPlatformClosedNow
		st.NextAvailableAt = sch.NextOpenAt(now)
		return st
	}

	st.OrderingAvailable = true
	// **وحين تُفتح بجدولٍ سارٍ يُعرَف متى تُغلَق** — نهايةُ الفترة الجارية،
	// وبها يُجدّد التطبيقُ نفسَه عند الحدّ. **وغيرُ الساريةِ مفتوحةٌ بلا
	// حدٍّ فلا موعدَ إغلاق.**
	if enforced {
		st.NextCloseAt = sch.NextCloseAt(now)
	}
	return st
}

// availableFrom **أوّلُ لحظةٍ صالحةٍ ابتداءً من `t`** — بحسب الجدول.
//
// **ولا جدولَ سارٍ يعني أنّ `t` نفسَها هي الموعد.**
func availableFrom(t time.Time, enforced bool, sch Schedule) *time.Time {
	if !enforced {
		out := t
		return &out
	}
	if sch.OpenAt(t) {
		out := t
		return &out
	}
	return sch.NextOpenAt(t)
}
