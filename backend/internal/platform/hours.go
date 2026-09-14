package platform

// ══════════════════════════════════════════════════════════════════════
// **دوامُ المنصّة — جدولٌ أسبوعيٌّ يحكمه الخادمُ وحدَه** (`PH`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// # وما الذي لم يكن موجوداً
//
// **ووضعُ الإطلاق رايةٌ لا جدول** (`launch.*`): **يقول أيُّ بابٍ مفتوحٌ
// أصلاً**، **ولا يعرف يومَ أسبوعٍ ولا ساعة.** **ودوامُ المتاجر جدولٌ**
// (`merchant_hours`) **لكنّه لمتجرٍ بعينه** — **والمنصّةُ نفسُها لا دوامَ
// لها.**
//
// **فمن أراد أن يغلق الاستقبالَ ليلاً لم يجد إلّا أن يطفئ
// `launch.customer_orders` بيده كلَّ ليلةٍ ويشعلَه كلَّ صباح** — **وذاك
// عملٌ يُنسى مرّةً فيبيت البابُ مفتوحاً بلا أحدٍ يحضّر.**
//
// # ولمَ الحسابُ هنا بلا قاعدةِ بيانات
//
// **وحدودُ الدقيقة تُقاس بالحساب لا بالنداء**: ثلاثون حالةً في قائمة
// المالك (`PH-01`…`PH-30`) **أكثرُها لحظاتٌ بعينها** — ٠٩:٠٠:٠٠ و
// ١٦:٥٩:٥٩ و١٧:٠٠:٠٠. **وفحصٌ يحتاج قاعدةً لكلّ لحظةٍ فحصٌ لا يُكتب.**
//
// **فالقرارُ دالّةٌ خالصةٌ تأخذ اللحظةَ والجدول** — **والقاعدةُ تحفظ
// الجدولَ ولا تحكم به.**
//
// # والحدودُ معرَّفةٌ مرّةً ولا تُترك للحدس
//
//	البدءُ داخلٌ   ·  ٠٩:٠٠:٠٠ مفتوحٌ
//	والانتهاءُ خارجٌ ·  ١٧:٠٠:٠٠ مغلقٌ · ١٦:٥٩:٥٩ مفتوحٌ
//
// **وتساوٍ غامضٌ على الحدّ يصنع طلباً يُقبَل في ثانيةٍ ويُردّ في التي
// تليها بلا سببٍ يُفهَم** — **فيُكتب الحدُّ ويُقاس.**

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// TZ **المنطقةُ الزمنيّةُ الحاكمة — باسمها لا بفارقِ ساعات.**
//
// **وفارقٌ ثابتٌ يكذب مرّتين في السنة** — **والاسمُ يحمل تحوّلَ التوقيت
// معه.** **ولا تُقرأ من جهازٍ**: ساعةُ هاتفٍ تُبدَّل بإصبعٍ، **والأهليّةُ
// لا تُبنى على ما يملك صاحبُ المصلحة تبديلَه.**
const TZ = "Asia/Damascus"

// Location **منطقةُ الخادم الزمنيّة** — و`UTC` إن تعذّر تحميلُ الاسم.
//
// **وتعذُّرُ التحميل لا يُسقط المحرّك** — **ولكنّه يُبلَّغ**: انظر
// `LoadLocation`.
func Location() *time.Location {
	if loc, err := time.LoadLocation(TZ); err == nil {
		return loc
	}
	return time.UTC
}

// Minutes **دقائقُ من منتصف الليل** — `[0, 1440]`.
type Minutes int

// dayMinutes دقائقُ اليوم الكامل.
const dayMinutes Minutes = 24 * 60

// weekMinutes دقائقُ الأسبوع — **مدارُ الجدول.**
const weekMinutes = 7 * dayMinutes

// String يكتبها `HH:MM` — **للرسائل وللوحة.**
func (m Minutes) String() string { return fmt.Sprintf("%02d:%02d", int(m)/60, int(m)%60) }

// ParseClock يقرأ `HH:MM` — **ولا يقبل `24:00` ولا ثوانيَ.**
//
// **ونهايةُ اليوم تُكتب `00:00`** — **وهي عبورٌ لمنتصف الليل بالتعريف
// أدناه، فلا تحتاج رمزاً ثانياً.**
func ParseClock(s string) (Minutes, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, ErrBadWindow
	}
	return Minutes(t.Hour()*60 + t.Minute()), nil
}

// Window **فترةُ عملٍ في يومٍ من الأسبوع.**
//
// **و`End <= Start` تعني عبورَ منتصف الليل** — **١٧:٠٠ ← ٠١:٠٠ فترةٌ
// تبدأ اليومَ وتنتهي غداً**، **وهي أكثرُ ما تعمل به الرقّة.**
//
// **ولا رايةَ «مغلق»** — **واليومُ بلا فتراتٍ مغلقٌ بذاته.** **ورايةٌ
// زائدةٌ تصنع حالين يقولان الشيءَ نفسَه فيفترقان يوماً.**
type Window struct {
	// Day **٠=الأحد … ٦=السبت** — **كما يعدّها `time.Weekday` وبوستغرس.**
	Day int `json:"day_of_week"`
	// Start **البدءُ داخلٌ.**
	Start Minutes `json:"-"`
	// End **الانتهاءُ خارجٌ.**
	End Minutes `json:"-"`
}

// StartText و EndText **نصّاً للوحة وللردّ العامّ** — `HH:MM`.
func (w Window) StartText() string { return w.Start.String() }

// EndText انظر StartText.
func (w Window) EndText() string { return w.End.String() }

// MarshalJSON **تُكتب بالنصّ لا بالدقائق** — **ورقمٌ خامٌّ في ردٍّ عامٍّ
// يُقرأ خطأً**، والشاشةُ تعرض `HH:MM` كما ضبطها المالك.
func (w Window) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"day_of_week":%d,"start":%q,"end":%q}`,
		w.Day, w.Start.String(), w.End.String())), nil
}

// CrossesMidnight **أتعبر الفترةُ منتصفَ الليل؟**
func (w Window) CrossesMidnight() bool { return w.End <= w.Start }

// span مدى الفترة بالدقائق — **والعابرةُ تُمدّ إلى ما بعد اليوم.**
func (w Window) span() (from, to Minutes) {
	from = Minutes(w.Day)*dayMinutes + w.Start
	to = Minutes(w.Day)*dayMinutes + w.End
	if w.CrossesMidnight() {
		to += dayMinutes
	}
	return from, to
}

var (
	// ErrBadWindow **فترةٌ لا تُقرأ** — يومٌ خارج المدى أو ساعةٌ لا تُحلَّل.
	ErrBadWindow = errors.New("فترةُ دوامٍ غيرُ صالحة")
	// ErrEmptyWindow **بدءٌ يساوي انتهاءً** — **ولا يُعرَف أصفرٌ هو أم يومٌ
	// كامل**، **والغموضُ يُردّ ولا يُخمَّن.**
	ErrEmptyWindow = errors.New("فترةٌ بلا مدّة — البدءُ يساوي الانتهاء")
	// ErrOverlap **فترتان تتداخلان** — **ومن كتب ٠٩:٠٠←١٤:٠٠ و١٣:٠٠←١٨:٠٠
	// لم يقصد ساعةً مكرّرة**، **وجدولٌ متداخلٌ يُجيب إجابتين.**
	ErrOverlap = errors.New("فترتا دوامٍ متداخلتان")
)

// Validate **يفحص الجدولَ كلَّه — ويُرتّبه.**
//
// **والتحقّقُ في المحرّك لا في الشاشة** — **ونداءٌ مباشرٌ يتجاوز كلَّ
// تحقّقٍ في متصفّح.**
//
// **والتداخلُ يُقاس على مدارِ الأسبوع لا داخلَ اليوم**: **فترةُ السبت
// ٢٢:٠٠←٠٣:٠٠ تلتقي بفترة الأحد ٠١:٠٠←٠٥:٠٠** — **ويومان مختلفان في
// الجدول وساعةٌ واحدةٌ في الواقع.**
func Validate(ws []Window) ([]Window, error) {
	for _, w := range ws {
		if w.Day < 0 || w.Day > 6 {
			return nil, ErrBadWindow
		}
		if w.Start < 0 || w.Start >= dayMinutes || w.End < 0 || w.End >= dayMinutes {
			return nil, ErrBadWindow
		}
		if w.Start == w.End {
			return nil, ErrEmptyWindow
		}
	}

	out := append([]Window(nil), ws...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Day != out[j].Day {
			return out[i].Day < out[j].Day
		}
		return out[i].Start < out[j].Start
	})

	// **والتداخلُ يُقاس بالمدى المطلق** — **ومدارُ الأسبوع يُطوى بالباقي.**
	for i := range out {
		fi, ti := out[i].span()
		for j := i + 1; j < len(out); j++ {
			fj, tj := out[j].span()
			if overlapOnWeek(fi, ti, fj, tj) {
				return nil, ErrOverlap
			}
		}
	}
	return out, nil
}

// overlapOnWeek **أيتقاطع مديان على مدارِ أسبوع؟**
//
// **والمدى قد يتجاوز نهايةَ الأسبوع** (سبتٌ يعبر إلى أحد) — **فيُطوى
// ويُقاس شقّاه.**
func overlapOnWeek(a1, a2, b1, b2 Minutes) bool {
	for _, a := range foldWeek(a1, a2) {
		for _, b := range foldWeek(b1, b2) {
			if a[0] < b[1] && b[0] < a[1] {
				return true
			}
		}
	}
	return false
}

// foldWeek يطوي مدىً على مدارِ الأسبوع إلى شقٍّ أو شقّين.
func foldWeek(from, to Minutes) [][2]Minutes {
	from %= weekMinutes
	if to <= weekMinutes {
		return [][2]Minutes{{from, to}}
	}
	return [][2]Minutes{{from, weekMinutes}, {0, to - weekMinutes}}
}

// Schedule **جدولُ الأسبوع** — مرتَّبٌ ومُتحقَّقٌ منه.
type Schedule []Window

// OpenAt **أالمنصّةُ داخلَ دوامها في هذه اللحظة؟**
//
// **والبدءُ داخلٌ والانتهاءُ خارجٌ** — **وهو ما يُقاس في `PH-02` و`PH-03`.**
//
// **وذيلُ أمس يُسأل عنه**: **الساعةُ الواحدةُ ليلاً تقع في فترةٍ بدأت
// أمسِ السادسةَ مساءً** — **ومن نظر إلى صفوف اليوم وحدَها أغلق المنصّةَ
// في أوّل ساعاتِ ليلها.** (**وهو العطبُ عينُه الذي أُصلح في دوام
// المتاجر.**)
func (s Schedule) OpenAt(now time.Time) bool {
	t := now.In(Location())
	day := int(t.Weekday())
	m := Minutes(t.Hour()*60 + t.Minute())
	yesterday := (day + 6) % 7

	for _, w := range s {
		switch {
		case w.Day == day && !w.CrossesMidnight():
			if m >= w.Start && m < w.End {
				return true
			}
		case w.Day == day && w.CrossesMidnight():
			// **صدرُ الفترة العابرة** — من بدئها إلى منتصف الليل.
			if m >= w.Start {
				return true
			}
		case w.Day == yesterday && w.CrossesMidnight():
			// **وذيلُها في اليوم التالي.**
			if m < w.End {
				return true
			}
		}
	}
	return false
}

// NextOpenAt **أوّلُ لحظةٍ يُفتح فيها الاستقبالُ بعد هذه اللحظة.**
//
// **ولا يُعاد موعدٌ مضى** — **وموعدٌ في الماضي أسوأُ من لا موعد** (وهي
// القاعدةُ عينُها في `NextOpenSQL` لدوام المتاجر).
//
// **ومن كان داخلَ دوامه الآن يُعاد له الآن** — **فالسؤالُ «متى أستطيع»
// لا «متى يبدأ ما بعد».**
//
// **وفارغٌ يعني «لا موعدَ في أسبوع»** — **جدولٌ فارغٌ أو كلُّ أيّامه
// مغلقة.** **ولا يُخترَع موعدٌ لا يعرفه أحد.**
func (s Schedule) NextOpenAt(now time.Time) *time.Time {
	if len(s) == 0 {
		return nil
	}
	if s.OpenAt(now) {
		t := now
		return &t
	}
	loc := Location()
	t := now.In(loc)
	// **ثمانيةُ أيّامٍ لا سبعة** — **فمن سُئل يومَ الأحد مساءً عن فترةٍ
	// تبدأ الأحدَ صباحاً وجدها في الأسبوع القادم**، **ولولا اليوم
	// الثامن لَعاد فارغاً وله موعدٌ بعد ستّ ساعات.**
	var best *time.Time
	for d := 0; d <= 8; d++ {
		day := t.AddDate(0, 0, d)
		dow := int(day.Weekday())
		for _, w := range s {
			if w.Day != dow {
				continue
			}
			start := time.Date(day.Year(), day.Month(), day.Day(),
				int(w.Start)/60, int(w.Start)%60, 0, 0, loc)
			if !start.After(now) {
				continue
			}
			if best == nil || start.Before(*best) {
				c := start
				best = &c
			}
		}
		if best != nil {
			// **ويومٌ وُجد فيه موعدٌ يُنهي البحث** — **وما بعده أبعد.**
			break
		}
	}
	return best
}

// Today **فتراتُ اليوم الجاري** — **للعرض لا للحكم.**
//
// **ويقرؤها الزبونُ ليعرف «أوقاتُ العمل اليوم»** — **ولا تُحسَب في
// الشاشة**: **حسبةٌ ثانيةٌ تفترق عن الأولى يوماً.**
func (s Schedule) Today(now time.Time) []Window {
	day := int(now.In(Location()).Weekday())
	out := []Window{}
	for _, w := range s {
		if w.Day == day {
			out = append(out, w)
		}
	}
	return out
}
