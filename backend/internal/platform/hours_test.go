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

// ══════════════════════════════════════════════════════════════════════
// **حدُّ الإغلاق — NextCloseAt** (Batch 5)
// ══════════════════════════════════════════════════════════════════════
//
// **ونظيرُ NextOpenAt**: متى يُغلَق المفتوحُ الآن. **يُقاس بنفسِ منطقِ
// OpenAt** (صدرُ اليوم، وذيلُ أمسِ العابر) فلا يفترقان.

func mustTime(t *testing.T, p *time.Time) time.Time {
	t.Helper()
	if p == nil {
		t.Fatal("**NextCloseAt = nil ولا يُنتظَر**")
	}
	return *p
}

// NC-01 · إغلاقُ اليومِ العاديّ — نهايةُ الفترةِ اليوم.
func TestNC01_SameDayClose(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "17:00")}
	got := mustTime(t, s.NextCloseAt(at(0, 12, 0, 0)))
	if !got.Equal(at(0, 17, 0, 0)) {
		t.Fatalf("**إغلاقُ اليوم**: %s ≠ %s", got, at(0, 17, 0, 0))
	}
}

// NC-02 · العابرةُ منتصفَ الليل — صدرُها يُغلَق غداً (مثالُ المالك).
func TestNC02_CrossMidnightHeadClosesTomorrow(t *testing.T) {
	s := Schedule{win(t, 0, "22:00", "03:00")} // الأحد ٢٢:٠٠ ← الاثنين ٠٣:٠٠
	got := mustTime(t, s.NextCloseAt(at(0, 23, 0, 0)))
	want := at(1, 3, 0, 0) // الاثنين ٠٣:٠٠ — لا اليوم
	if !got.Equal(want) {
		t.Fatalf("**صدرُ العابرة يُغلَق غداً**: %s ≠ %s", got, want)
	}
}

// NC-03 · ذيلُ عابرةِ أمسِ — يُغلَق اليوم (حدُّ أسبوعٍ: سبتٌ ← أحد).
func TestNC03_CrossMidnightTailWeekWrap(t *testing.T) {
	s := Schedule{win(t, 6, "22:00", "03:00")} // السبت ٢٢:٠٠ ← الأحد ٠٣:٠٠
	// الأحدُ الواحدةُ ليلاً داخلَ ذيلِ فترةِ السبت.
	got := mustTime(t, s.NextCloseAt(at(0, 1, 0, 0)))
	want := at(0, 3, 0, 0) // الأحد ٠٣:٠٠ اليوم
	if !got.Equal(want) {
		t.Fatalf("**ذيلُ عابرةِ أمس يُغلَق اليوم**: %s ≠ %s", got, want)
	}
}

// NC-04 · مغلقٌ الآن — لا حدَّ إغلاقٍ لحاله (nil).
func TestNC04_ClosedGivesNil(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "17:00")}
	if got := s.NextCloseAt(at(0, 8, 0, 0)); got != nil {
		t.Fatalf("**مغلقٌ ولا حدَّ إغلاقٍ له**، وجد: %s", got)
	}
	if got := s.NextCloseAt(at(0, 17, 0, 0)); got != nil { // الحدُّ خارجٌ
		t.Fatalf("**١٧:٠٠ خارجُ الفترة (الانتهاءُ خارج)**، وجد: %s", got)
	}
}

// NC-05 · متتالية مفتوح→مغلق→مفتوح عبر فترتين.
func TestNC05_OpenCloseReopenSequence(t *testing.T) {
	s := Schedule{win(t, 0, "09:00", "12:00"), win(t, 0, "14:00", "18:00")}
	// مفتوحٌ أولاً — يُغلَق ١٢:٠٠.
	if got := mustTime(t, s.NextCloseAt(at(0, 10, 0, 0))); !got.Equal(at(0, 12, 0, 0)) {
		t.Fatalf("**الفترةُ الأولى تُغلَق ١٢:٠٠**: %s", got)
	}
	// بين الفترتين — مغلقٌ، لا حدَّ إغلاق (nil)، والفتحُ ١٤:٠٠.
	if got := s.NextCloseAt(at(0, 12, 30, 0)); got != nil {
		t.Fatalf("**بين الفترتين مغلقٌ بلا حدِّ إغلاق**، وجد: %s", got)
	}
	if got := mustTime(t, s.NextOpenAt(at(0, 12, 30, 0))); !got.Equal(at(0, 14, 0, 0)) {
		t.Fatalf("**يُفتح ثانيةً ١٤:٠٠**: %s", got)
	}
	// الفترةُ الثانية — تُغلَق ١٨:٠٠.
	if got := mustTime(t, s.NextCloseAt(at(0, 15, 0, 0))); !got.Equal(at(0, 18, 0, 0)) {
		t.Fatalf("**الفترةُ الثانية تُغلَق ١٨:٠٠**: %s", got)
	}
}

// NC-06 · غيرُ سارٍ ⇒ لا حدَّ إغلاقٍ (يُحذَف): المنصّةُ والمنطقة.
func TestNC06_NotEnforcedOmitsNextClose(t *testing.T) {
	sch := Schedule{win(t, 0, "09:00", "17:00")}
	// المنصّة: غيرُ سارٍ ⇒ مفتوحةٌ بلا حدّ.
	if st := Decide(at(0, 12, 0, 0), false, sch, Closure{}); st.NextCloseAt != nil {
		t.Fatalf("**منصّةٌ غيرُ ساريةٍ لا حدَّ إغلاقٍ لها**، وجد: %s", st.NextCloseAt)
	}
	// المنصّة: سارٍ ومفتوحٌ ⇒ حدٌّ موجود.
	if st := Decide(at(0, 12, 0, 0), true, sch, Closure{}); st.NextCloseAt == nil {
		t.Fatal("**منصّةٌ ساريةٌ مفتوحةٌ يجب أن يُعرَف إغلاقُها**")
	}
	// المنطقة: غيرُ ساريةٍ ⇒ لا حدّ.
	if zs := DecideZone(at(0, 12, 0, 0), false, sch); zs.NextCloseAt != nil {
		t.Fatalf("**منطقةٌ غيرُ ساريةٍ لا حدَّ إغلاقٍ لها**، وجد: %s", zs.NextCloseAt)
	}
	// المنطقة: ساريةٌ ومفتوحةٌ ⇒ حدٌّ موجودٌ، ولا NextOpenAt.
	zs := DecideZone(at(0, 12, 0, 0), true, sch)
	if zs.NextCloseAt == nil {
		t.Fatal("**منطقةٌ ساريةٌ مفتوحةٌ يجب أن يُعرَف إغلاقُها**")
	}
	if zs.NextOpenAt != nil {
		t.Fatalf("**ومفتوحةٌ لا موعدَ فتحٍ لها**، وجد: %s", zs.NextOpenAt)
	}
}
