package platform

// ══════════════════════════════════════════════════════════════════════
// **الأسبقيّةُ وموعدُ العودة** (`PH-23`، `PH-24`، وتقاطعُ الطبقتين)
// ══════════════════════════════════════════════════════════════════════

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func ptr(t time.Time) *time.Time { return &t }

// **الجدولُ لا يسري حتّى يُفعَّل** — **وهو ما يجعل الهجرةَ لا تُغلق
// شيئاً يومَ تُنشر.**
func TestPH_ScheduleDormantUntilEnforced(t *testing.T) {
	sch := Schedule{win(t, 0, "09:00", "17:00")}
	st := Decide(at(0, 23, 0, 0), false, sch, Closure{})
	if !st.OrderingAvailable {
		t.Fatal("**جدولٌ غيرُ مفعَّلٍ أغلق الاستقبال** — " +
			"**فنشرُ الهجرة يوقف المنصّة**")
	}
	if st.Reason != ReasonOpen {
		t.Fatalf("سببٌ بلا منع: %q", st.Reason)
	}
}

// **ومُفعَّلاً يغلق خارجَ الدوام.**
func TestPH17_EnforcedScheduleCloses(t *testing.T) {
	sch := Schedule{win(t, 0, "09:00", "17:00")}
	st := Decide(at(0, 23, 0, 0), true, sch, Closure{})
	if st.OrderingAvailable {
		t.Fatal("**الحاديةَ عشرةَ ليلاً خارجَ ٠٩:٠٠←١٧:٠٠ ومع ذلك فُتح**")
	}
	if st.Reason != ReasonPlatformClosedNow {
		t.Fatalf("**السببُ يجب أن يكون خارجَ الدوام**، وجاء %q", st.Reason)
	}
	if st.NextAvailableAt == nil {
		t.Fatal("**أُغلق بلا موعدِ عودةٍ وهو معلوم**")
	}
}

// **والإيقافُ المؤقّتُ أخصُّ فيسبق الجدولَ** — **وصيانةٌ تقع وسطَ
// الدوام لا يجوز أن يقول عنها الجدولُ «مفتوح».**
func TestPH15_ClosurePrecedesSchedule(t *testing.T) {
	sch := Schedule{win(t, 0, "09:00", "17:00")}
	st := Decide(at(0, 12, 0, 0), true, sch,
		Closure{Active: true, Message: "صيانةٌ مؤقّتة"})
	if st.OrderingAvailable {
		t.Fatal("**إيقافٌ مؤقّتٌ مفعَّلٌ ومع ذلك قُبل الطلب**")
	}
	if st.Reason != ReasonTemporarilyUnavailable {
		t.Fatalf("**السببُ يجب أن يكون الإيقافَ المؤقّت**، وجاء %q", st.Reason)
	}
	if st.Message != "صيانةٌ مؤقّتة" {
		t.Fatalf("**نصُّ المالك لم يصل**: %q", st.Message)
	}
}

// PH-23 · **موعدُ العودة تقاطعُ الإيقافِ والجدول.**
//
// **وهو بلاغُ المالك نصّاً**: ينتهي الإيقافُ الثالثةَ والدوامُ يبدأ
// الخامسةَ ⇒ **الموعدُ الخامسة**.
func TestPH23_ClosureEndIntersectsSchedule(t *testing.T) {
	sch := Schedule{win(t, 0, "17:00", "23:00")}
	st := Decide(at(0, 12, 0, 0), true, sch, Closure{
		Active: true, Message: "صيانة", EndsAt: ptr(at(0, 15, 0, 0)),
	})
	if st.NextAvailableAt == nil {
		t.Fatal("**موعدٌ معلومٌ ولم يُعَد**")
	}
	if !st.NextAvailableAt.Equal(at(0, 17, 0, 0)) {
		t.Fatalf("**قيل للزبون %v والدوامُ يبدأ ١٧:٠٠** — "+
			"**فعاد فوجد البابَ مغلقاً**", *st.NextAvailableAt)
	}
}

// **وإن انتهى الإيقافُ داخلَ الدوام فالموعدُ نهايتُه هو.**
func TestPH23b_ClosureEndInsideWindow(t *testing.T) {
	sch := Schedule{win(t, 0, "09:00", "23:00")}
	st := Decide(at(0, 12, 0, 0), true, sch, Closure{
		Active: true, EndsAt: ptr(at(0, 15, 0, 0)),
	})
	if st.NextAvailableAt == nil || !st.NextAvailableAt.Equal(at(0, 15, 0, 0)) {
		t.Fatalf("**الموعدُ نهايةُ الإيقاف**، وجاء %v", st.NextAvailableAt)
	}
}

// PH-24 · **وإيقافٌ بلا موعدٍ لا يُخترَع له موعد.**
func TestPH24_IndefiniteClosureInventsNothing(t *testing.T) {
	sch := Schedule{win(t, 0, "09:00", "23:00")}
	st := Decide(at(0, 12, 0, 0), true, sch,
		Closure{Active: true, Message: "توقّفٌ طارئ"})
	if st.NextAvailableAt != nil {
		t.Fatalf("**اخترعَ موعداً لا يعرفه أحد: %v**", *st.NextAvailableAt)
	}
}

// **وموعدُ عودةٍ مضى يُنهي الإيقافَ بنفسه** — **ولا مهمّةَ دوريّةً
// تُطفئه**: مهمّةٌ تتأخّر، والوقتُ لا يتأخّر.
func TestPH_ExpiredClosureSelfLifts(t *testing.T) {
	sch := Schedule{win(t, 0, "09:00", "23:00")}
	st := Decide(at(0, 16, 0, 0), true, sch, Closure{
		Active: true, EndsAt: ptr(at(0, 15, 0, 0)),
	})
	if !st.OrderingAvailable {
		t.Fatalf("**إيقافٌ انقضى وقتُه ما زال يمنع**: %q", st.Reason)
	}
}

// **والحدُّ لحظةَ الانتهاء نفسِها**: يُرفَع.
func TestPH_ClosureEndBoundary(t *testing.T) {
	st := Decide(at(0, 15, 0, 0), false, Schedule{}, Closure{
		Active: true, EndsAt: ptr(at(0, 15, 0, 0)),
	})
	if !st.OrderingAvailable {
		t.Fatal("**لحظةُ الانتهاء نفسُها ما زالت ممنوعة**")
	}
}

// **وحالُ الردّ تحمل منطقةَ المنصّة ولحظةَ الخادم** — **فالشاشةُ
// تنسّق ولا تحكم.**
func TestPH_StateCarriesServerTruth(t *testing.T) {
	now := at(0, 12, 0, 0)
	st := Decide(now, true, Schedule{win(t, 0, "09:00", "17:00")}, Closure{})
	if st.Timezone != TZ {
		t.Fatalf("منطقةٌ غيرُ منطقة المنصّة: %q", st.Timezone)
	}
	if !st.ServerTime.Equal(now) {
		t.Fatal("**لحظةُ الخادم لم تُعَد**")
	}
	if len(st.TodayWindows) != 1 {
		t.Fatalf("**فتراتُ اليوم لم تُعَد**: %d", len(st.TodayWindows))
	}
}

// NC-07 · تسلسلُ next_close_at في الردّ (Batch 5): يظهر حين تُفتح بجدولٍ
// سارٍ، ويُحذَف حين تكون مغلقةً أو غيرَ سارية (omitempty).
func TestNC07_NextCloseAtSerialization(t *testing.T) {
	sch := Schedule{win(t, 0, "09:00", "17:00")}
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, Location()) // الأحد ظهراً — مفتوح

	openEnforced, _ := json.Marshal(Decide(now, true, sch, Closure{}))
	if !strings.Contains(string(openEnforced), `"next_close_at"`) {
		t.Fatalf("**مفتوحةٌ ساريةٌ يجب أن تحمل next_close_at**: %s", openEnforced)
	}
	notEnforced, _ := json.Marshal(Decide(now, false, sch, Closure{}))
	if strings.Contains(string(notEnforced), `"next_close_at"`) {
		t.Fatalf("**غيرُ ساريةٍ لا next_close_at**: %s", notEnforced)
	}
	closedNow := time.Date(2026, 9, 13, 20, 0, 0, 0, Location()) // مغلق
	closed, _ := json.Marshal(Decide(closedNow, true, sch, Closure{}))
	if strings.Contains(string(closed), `"next_close_at"`) {
		t.Fatalf("**مغلقةٌ لا next_close_at**: %s", closed)
	}
	if !strings.Contains(string(closed), `"next_available_at"`) {
		t.Fatalf("**مغلقةٌ يجب أن تحمل next_available_at**: %s", closed)
	}
}
