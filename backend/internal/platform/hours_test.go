package platform

// ══════════════════════════════════════════════════════════════════════
// **حدودُ الدوام تُقاس بالثانية** (`PH-01`…`PH-13`، `PH-23`…`PH-26`)
// ══════════════════════════════════════════════════════════════════════
//
// **والقرارُ دالّةٌ خالصةٌ فيُقاس بلا قاعدةٍ ولا خادم** — **وثلاثون
// حالةً أكثرُها لحظاتٌ بعينها**: ٠٩:٠٠:٠٠ و١٦:٥٩:٥٩ و١٧:٠٠:٠٠.
// **وفحصٌ يحتاج قاعدةً لكلّ ثانيةٍ فحصٌ لا يُكتب.**
//
// **والأسبوعُ المرجعُ ٢٠٢٦-٠٩-١٣ أحدٌ** — فيُعَدّ منه إلى السبت.

import (
	"testing"
	"time"
)

// at لحظةٌ في الأسبوع المرجع — `day` ٠=الأحد … ٦=السبت.
func at(day, hh, mm, ss int) time.Time {
	return time.Date(2026, 9, 13+day, hh, mm, ss, 0, Location())
}

func hhmm(t *testing.T, s string) Minutes {
	t.Helper()
	m, err := ParseClock(s)
	if err != nil {
		t.Fatalf("ساعةٌ لا تُقرأ: %s", s)
	}
	return m
}

// win يبني فترةً — ويسقط الفحصُ إن لم تُقرأ ساعتُها.
func win(t *testing.T, day int, from, to string) Window {
	t.Helper()
	return Window{Day: day, Start: hhmm(t, from), End: hhmm(t, to)}
}

// ═════════════════ الجدولُ نفسُه ═════════════════

// PH-01 · فترةُ يومٍ عاديّة.
func TestPH01_WeekdayOpenInterval(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "17:00")}
	if !s.OpenAt(at(0, 12, 0, 0)) {
		t.Fatal("**الظهرُ داخلَ ٠٩:٠٠←١٧:٠٠ ولم يُفتح**")
	}
	if s.OpenAt(at(0, 8, 59, 59)) {
		t.Fatal("**ما قبلَ البدء مفتوحٌ**")
	}
}

// PH-02 · حدُّ البدء داخلٌ.
func TestPH02_OpeningBoundaryInclusive(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "17:00")}
	if !s.OpenAt(at(0, 9, 0, 0)) {
		t.Fatal("**٠٩:٠٠:٠٠ يجب أن يكون مفتوحاً — البدءُ داخل**")
	}
}

// PH-03 · وحدُّ الانتهاء خارجٌ.
func TestPH03_ClosingBoundaryExclusive(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "17:00")}
	if !s.OpenAt(at(0, 16, 59, 59)) {
		t.Fatal("**١٦:٥٩:٥٩ يجب أن يكون مفتوحاً**")
	}
	if s.OpenAt(at(0, 17, 0, 0)) {
		t.Fatal("**١٧:٠٠:٠٠ يجب أن يكون مغلقاً — الانتهاءُ خارج**")
	}
}

// PH-04 · يومٌ بلا فتراتٍ مغلقٌ بذاته.
func TestPH04_ClosedDay(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "17:00")}
	if s.OpenAt(at(1, 12, 0, 0)) {
		t.Fatal("**الاثنينُ بلا فتراتٍ ومع ذلك فُتح**")
	}
}

// PH-05 · فترتان في اليوم — و PH-06 ما بينهما مغلق.
func TestPH05_MultipleWindows(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "14:00"), win(t, 0, "17:00", "23:00")}
	if !s.OpenAt(at(0, 10, 0, 0)) || !s.OpenAt(at(0, 18, 0, 0)) {
		t.Fatal("**فترتان في اليوم ولم تُفتح إحداهما**")
	}
}

// PH-06 · وما بين الفترتين مغلق.
func TestPH06_BetweenWindowsClosed(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "14:00"), win(t, 0, "17:00", "23:00")}
	if s.OpenAt(at(0, 15, 0, 0)) {
		t.Fatal("**الثالثةُ بين فترتين ومع ذلك فُتحت**")
	}
}

// PH-07 · فترةٌ تعبر منتصفَ الليل — صدرُها.
func TestPH07_CrossMidnightHead(t *testing.T) {
	s := Schedule{win(t, 0, "17:00", "01:00")}
	if !s.OpenAt(at(0, 23, 30, 0)) {
		t.Fatal("**١١:٣٠ مساءً داخلَ ١٧:٠٠←٠١:٠٠ ولم تُفتح**")
	}
}

// PH-08 · وذيلُها بعد منتصف الليل — **وهو ما يُنسى.**
func TestPH08_PreviousDaySpill(t *testing.T) {
	s := Schedule{win(t, 0, "17:00", "01:00")}
	if !s.OpenAt(at(1, 0, 30, 0)) {
		t.Fatal("**نصفُ ساعةٍ بعد منتصف الليل من فترةِ أمس ومع ذلك أُغلق**")
	}
	if s.OpenAt(at(1, 1, 0, 0)) {
		t.Fatal("**٠١:٠٠:٠٠ حدُّ انتهاءٍ وهو خارجٌ**")
	}
	// **ولا يُفتح الاثنينُ نهاراً** — الذيلُ ساعةٌ لا يوم.
	if s.OpenAt(at(1, 12, 0, 0)) {
		t.Fatal("**ذيلُ أمس فتح يومَ الاثنين كلَّه**")
	}
}

// PH-09 · حدُّ الأسبوع — سبتٌ يعبر إلى أحد.
func TestPH09_WeekBoundary(t *testing.T) {
	s := Schedule{win(t, 6, "22:00", "03:00")}
	if !s.OpenAt(at(6, 23, 0, 0)) {
		t.Fatal("**ليلةُ السبت لم تُفتح**")
	}
	// **والأحدُ التالي هو `day=0` في الأسبوع الذي يليه.**
	if !s.OpenAt(at(0, 2, 0, 0).AddDate(0, 0, 7)) {
		t.Fatal("**ذيلُ السبت لم يصل إلى فجر الأحد**")
	}
}

// ═════════════════ موعدُ الفتح القادم ═════════════════

// PH-10 · فترةٌ لاحقةٌ في اليوم نفسِه.
func TestPH10_NextOpenSameDay(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "14:00"), win(t, 0, "17:00", "23:00")}
	got := s.NextOpenAt(at(0, 15, 0, 0))
	if got == nil || !got.Equal(at(0, 17, 0, 0)) {
		t.Fatalf("**الموعدُ القادمُ ١٧:٠٠ اليوم**، وجاء %v", got)
	}
}

// PH-11 · وفي يومٍ تالٍ حين لا يبقى شيءٌ اليوم.
func TestPH11_NextOpenLaterDay(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "14:00"), win(t, 3, "10:00", "12:00")}
	got := s.NextOpenAt(at(0, 20, 0, 0))
	if got == nil || !got.Equal(at(3, 10, 0, 0)) {
		t.Fatalf("**الموعدُ الأربعاءَ ١٠:٠٠**، وجاء %v", got)
	}
}

// PH-12 · ولا يُعاد موعدٌ مضى.
func TestPH12_NoPastNextOpen(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "14:00")}
	now := at(0, 20, 0, 0)
	got := s.NextOpenAt(now)
	if got == nil {
		t.Fatal("**لا موعدَ في أسبوعٍ كامل**")
	}
	if !got.After(now) {
		t.Fatalf("**موعدٌ في الماضي: %v والآن %v**", got, now)
	}
	// **ويقع في الأحد القادم لا في الماضي.**
	if !got.Equal(at(0, 9, 0, 0).AddDate(0, 0, 7)) {
		t.Fatalf("**الموعدُ الأحدَ القادم**، وجاء %v", got)
	}
}

// **ومن كان داخلَ دوامه يُعاد له الآن** — السؤالُ «متى أستطيع».
func TestPH10b_NextOpenWhileOpenIsNow(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "17:00")}
	now := at(0, 10, 0, 0)
	got := s.NextOpenAt(now)
	if got == nil || !got.Equal(now) {
		t.Fatalf("**مفتوحٌ الآن فالموعدُ الآن**، وجاء %v", got)
	}
}

// **وجدولٌ فارغٌ لا موعدَ له** — **ولا يُخترَع.**
func TestPH24b_EmptyScheduleHasNoNextOpen(t *testing.T) {
	if got := (Schedule{}).NextOpenAt(at(0, 10, 0, 0)); got != nil {
		t.Fatalf("**جدولٌ فارغٌ أعاد موعداً: %v**", got)
	}
}

// ═════════════════ PH-13 · ساعةُ الجهاز لا تحكم ═════════════════
//
// **ولا سبيلَ إلى ساعة الجهاز من هنا أصلاً** — **والقرارُ يأخذ لحظةً
// تُمرَّر إليه**، **فالبرهانُ أنّ اللحظةَ نفسَها تُقرأ بمنطقة المنصّة
// مهما كانت منطقةُ من يسأل.**
func TestPH13_DeviceTimezoneIrrelevant(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "17:00")}
	moment := at(0, 12, 0, 0) // ظهرُ دمشق

	for _, name := range []string{"UTC", "America/New_York", "Asia/Tokyo"} {
		loc, err := time.LoadLocation(name)
		if err != nil {
			continue // منطقةٌ غيرُ منصَّبةٍ على هذا الجهاز — لا تُسقط الفحص
		}
		if !s.OpenAt(moment.In(loc)) {
			t.Fatalf("**اللحظةُ عينُها أُغلقت حين قُرئت بـ%s** — "+
				"**فمنطقةُ القارئ حكمت**", name)
		}
	}

	// **وساعةٌ تُزاح لا تفتح شيئاً**: الخامسةُ مساءً في دمشق مغلقةٌ ولو
	// سمّاها جهازٌ آخرُ ظهراً.
	if s.OpenAt(at(0, 17, 30, 0).In(time.UTC)) {
		t.Fatal("**لحظةٌ خارجَ الدوام فُتحت بتبديل المنطقة**")
	}
}

// ═════════════════ التحقّقُ من الجدول ═════════════════

// PH-25 · تداخلٌ يُردّ.
func TestPH25_OverlapRejected(t *testing.T) {
	_, err := Validate([]Window{
		win(t, 0, "09:00", "14:00"),
		win(t, 0, "13:00", "18:00"),
	})
	if err != ErrOverlap {
		t.Fatalf("**تداخلٌ في اليوم نفسِه قُبل**: %v", err)
	}
}

// **وتداخلٌ عبر منتصف الليل بين يومين** — **وهو ما لا يراه فحصُ يومٍ.**
func TestPH25b_CrossDayOverlapRejected(t *testing.T) {
	_, err := Validate([]Window{
		win(t, 0, "22:00", "03:00"),
		win(t, 1, "01:00", "05:00"),
	})
	if err != ErrOverlap {
		t.Fatalf("**ذيلُ الأحد يلتقي صدرَ الاثنين ومع ذلك قُبل**: %v", err)
	}
}

// **وتداخلٌ على مدارِ الأسبوع** — سبتٌ يعبر إلى أحد.
func TestPH25c_WeekWrapOverlapRejected(t *testing.T) {
	_, err := Validate([]Window{
		win(t, 6, "22:00", "03:00"),
		win(t, 0, "02:00", "06:00"),
	})
	if err != ErrOverlap {
		t.Fatalf("**ذيلُ السبت يلتقي فجرَ الأحد ومع ذلك قُبل**: %v", err)
	}
}

// PH-26 · وقيمٌ لا تصلح تُردّ.
func TestPH26_InvalidRejected(t *testing.T) {
	cases := []struct {
		name string
		ws   []Window
		want error
	}{
		{"يومٌ خارجَ المدى", []Window{{Day: 7, Start: 60, End: 120}}, ErrBadWindow},
		{"يومٌ سالب", []Window{{Day: -1, Start: 60, End: 120}}, ErrBadWindow},
		{"دقيقةٌ خارجَ اليوم", []Window{{Day: 0, Start: 60, End: 1440}}, ErrBadWindow},
		{"بدءٌ يساوي انتهاءً", []Window{{Day: 0, Start: 600, End: 600}}, ErrEmptyWindow},
	}
	for _, c := range cases {
		if _, err := Validate(c.ws); err != c.want {
			t.Fatalf("%s: **قُبل ما لا يُقبَل** (%v)", c.name, err)
		}
	}
	if _, err := ParseClock("24:00"); err == nil {
		t.Fatal("**٢٤:٠٠ قُبلت — ونهايةُ اليوم تُكتب ٠٠:٠٠**")
	}
}

// **والترتيبُ حتميٌّ** — **وجدولٌ يُعاد بترتيبٍ مختلفٍ كلَّ مرّةٍ يُقرأ
// تبدّلاً.**
func TestPH_DeterministicOrder(t *testing.T) {
	got, err := Validate([]Window{
		win(t, 3, "10:00", "12:00"),
		win(t, 0, "17:00", "23:00"),
		win(t, 0, "09:00", "14:00"),
	})
	if err != nil {
		t.Fatalf("جدولٌ صالحٌ رُدّ: %v", err)
	}
	want := []string{"0 09:00", "0 17:00", "3 10:00"}
	for i, w := range got {
		if got := string(rune('0'+w.Day)) + " " + w.Start.String(); got != want[i] {
			t.Fatalf("الترتيبُ %d: %s ≠ %s", i, got, want[i])
		}
	}
}
