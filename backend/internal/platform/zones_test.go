package platform

// ══════════════════════════════════════════════════════════════════════
// **وقتُ المنطقة وتقاطعُه مع وقتِ المنصّة** (`ZH`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **وآلةُ الجدول مقيسةٌ في `hours_test.go`** — الحدُّ وعبورُ منتصف الليل
// وحدُّ الأسبوع والتداخل — **ولا تُعاد هنا**: **المنطقةُ تستعمل
// `Schedule` عينَها، فما ثبت لها ثبت للمنصّة.**
//
// **وهذه تقيس ما يخصّ المنطقةَ وحدَها**: **رايةُ السريان لكلّ منطقة**،
// **والتقاطعُ مع المنصّة.**

import (
	"testing"
	"time"
)

// ZH-01 · **جدولٌ غيرُ سارٍ ⇒ مفتوحةٌ على مدار الساعة.**
//
// **وهو حالُ كلّ منطقةٍ قائمةٍ اليوم** — **فنشرُ الهجرة لا يُغلق واحدة.**
func TestZH01_DisabledScheduleIsAlwaysOpen(t *testing.T) {
	// **ولو كان الجدولُ مكتوباً ومغلقاً الآن** — الرايةُ هي الحاكمة.
	sch := Schedule{win(t, 0, "02:00", "03:00")}
	for _, moment := range []time.Time{at(0, 12, 0, 0), at(3, 23, 0, 0)} {
		st := DecideZone(moment, false, sch)
		if !st.Open {
			t.Fatalf("**منطقةٌ بلا جدولٍ سارٍ أُغلقت** في %v", moment)
		}
		if st.NextOpenAt != nil {
			t.Fatal("**موعدُ عودةٍ لمن لم يغب**")
		}
		if st.Enforced {
			t.Fatal("**رايةٌ مرفوعةٌ بلا سبب**")
		}
	}
}

// ZH-02 · وسارياً يُفتح داخلَ الفترة.
func TestZH02_EnforcedOpenInsideWindow(t *testing.T) {
	sch := Schedule{win(t, 0, "08:00", "18:00")}
	if !DecideZone(at(0, 12, 0, 0), true, sch).Open {
		t.Fatal("**الظهرُ داخلَ ٠٨:٠٠←١٨:٠٠ وأُغلق**")
	}
}

// ZH-03 · حدُّ البدء داخلٌ — و ZH-04 حدُّ الانتهاء خارج.
func TestZH03_ZH04_Boundaries(t *testing.T) {
	sch := Schedule{win(t, 0, "08:00", "18:00")}
	if !DecideZone(at(0, 8, 0, 0), true, sch).Open {
		t.Fatal("**٠٨:٠٠:٠٠ يجب أن يكون مفتوحاً — البدءُ داخل**")
	}
	if !DecideZone(at(0, 17, 59, 59), true, sch).Open {
		t.Fatal("**١٧:٥٩:٥٩ يجب أن يكون مفتوحاً**")
	}
	if DecideZone(at(0, 18, 0, 0), true, sch).Open {
		t.Fatal("**١٨:٠٠:٠٠ يجب أن يكون مغلقاً — الانتهاءُ خارج**")
	}
}

// ZH-05 · يومٌ بلا فتراتٍ مغلق — و ZH-07 ما بين فترتين مغلق.
func TestZH05_ZH07_ClosedDayAndGap(t *testing.T) {
	sch := Schedule{win(t, 0, "08:00", "12:00"), win(t, 0, "16:00", "22:00")}
	if DecideZone(at(1, 12, 0, 0), true, sch).Open {
		t.Fatal("**الاثنينُ بلا فتراتٍ وفُتح**")
	}
	if DecideZone(at(0, 14, 0, 0), true, sch).Open {
		t.Fatal("**الثانيةُ بين فترتين وفُتحت**")
	}
}

// ZH-06 · فترتان في اليوم — كلتاهما تعملان.
func TestZH06_MultipleWindows(t *testing.T) {
	sch := Schedule{win(t, 0, "08:00", "12:00"), win(t, 0, "16:00", "22:00")}
	if !DecideZone(at(0, 9, 0, 0), true, sch).Open || !DecideZone(at(0, 17, 0, 0), true, sch).Open {
		t.Fatal("**فترتان في اليوم ولم تُفتح إحداهما**")
	}
}

// ZH-08 · عبورُ منتصف الليل — و ZH-09 ذيلُه في اليوم التالي.
func TestZH08_ZH09_CrossMidnight(t *testing.T) {
	sch := Schedule{win(t, 0, "20:00", "02:00")}
	if !DecideZone(at(0, 23, 0, 0), true, sch).Open {
		t.Fatal("**الحاديةَ عشرةَ داخلَ ٢٠:٠٠←٠٢:٠٠ وأُغلقت**")
	}
	if !DecideZone(at(1, 1, 0, 0), true, sch).Open {
		t.Fatal("**ذيلُ أمس لم يصل الواحدةَ ليلاً**")
	}
	if DecideZone(at(1, 2, 0, 0), true, sch).Open {
		t.Fatal("**٠٢:٠٠:٠٠ حدُّ انتهاءٍ وهو خارج**")
	}
}

// ZH-10 · حدُّ الأسبوع — سبتٌ يعبر إلى أحد.
func TestZH10_WeekBoundary(t *testing.T) {
	sch := Schedule{win(t, 6, "21:00", "04:00")}
	if !DecideZone(at(6, 23, 0, 0), true, sch).Open {
		t.Fatal("**ليلةُ السبت لم تُفتح**")
	}
	if !DecideZone(at(0, 3, 0, 0).AddDate(0, 0, 7), true, sch).Open {
		t.Fatal("**ذيلُ السبت لم يصل فجرَ الأحد**")
	}
}

// ZH-26 · موعدُ الفترة القادمة في اليوم نفسِه.
func TestZH26_NextWindowSameDay(t *testing.T) {
	sch := Schedule{win(t, 0, "08:00", "12:00"), win(t, 0, "16:00", "22:00")}
	st := DecideZone(at(0, 14, 0, 0), true, sch)
	if st.Open {
		t.Fatal("مقدّمةٌ مكسورة")
	}
	if st.NextOpenAt == nil || !st.NextOpenAt.Equal(at(0, 16, 0, 0)) {
		t.Fatalf("**الموعدُ ١٦:٠٠ اليوم**، وجاء %v", st.NextOpenAt)
	}
}

// ZH-27 · وفي يومٍ تالٍ حين لا يبقى شيءٌ اليوم — ولا يُعاد موعدٌ مضى.
func TestZH27_NextWindowLaterDay(t *testing.T) {
	sch := Schedule{win(t, 0, "08:00", "12:00"), win(t, 2, "09:00", "11:00")}
	now := at(0, 20, 0, 0)
	st := DecideZone(now, true, sch)
	if st.NextOpenAt == nil || !st.NextOpenAt.Equal(at(2, 9, 0, 0)) {
		t.Fatalf("**الموعدُ الثلاثاءَ ٠٩:٠٠**، وجاء %v", st.NextOpenAt)
	}
	if !st.NextOpenAt.After(now) {
		t.Fatal("**موعدٌ في الماضي**")
	}
}

// ZH-13 · **ومنطقتان بجدولين مختلفين** — **ولا جدولَ عامٌّ واحد.**
//
// **والقرارُ دالّةٌ تأخذ جدولاً** — **فمنطقتان تُقاسان بنداءين لا
// بحالٍ مشتركةٍ تُخزَّن.**
func TestZH13_ZonesAreIndependent(t *testing.T) {
	a := Schedule{win(t, 0, "08:00", "18:00")} // نهاريّة
	b := Schedule{win(t, 0, "20:00", "23:00")} // ليليّة
	moment := at(0, 12, 0, 0)
	if !DecideZone(moment, true, a).Open {
		t.Fatal("**النهاريّةُ أُغلقت ظهراً**")
	}
	if DecideZone(moment, true, b).Open {
		t.Fatal("**الليليّةُ فُتحت ظهراً** — فالجدولان واحد")
	}
	// **وثالثةٌ بلا سريانٍ مفتوحةٌ في الحالين.**
	if !DecideZone(moment, false, b).Open {
		t.Fatal("**غيرُ الساريةِ تبعت جدولَ غيرِها**")
	}
}

// ZH-29 · **وساعةُ الجهاز لا تحكم.**
func TestZH29_DeviceClockIrrelevant(t *testing.T) {
	sch := Schedule{win(t, 0, "08:00", "18:00")}
	moment := at(0, 12, 0, 0)
	for _, name := range []string{"UTC", "America/New_York", "Asia/Tokyo"} {
		loc, err := time.LoadLocation(name)
		if err != nil {
			continue
		}
		if !DecideZone(moment.In(loc), true, sch).Open {
			t.Fatalf("**اللحظةُ عينُها أُغلقت حين قُرئت بـ%s**", name)
		}
	}
}

// ═════════════════ ZH-28 · تقاطعُ المنصّة والمنطقة ═════════════════

// TestZH28_CombinedNextOrderingTime **ومن قال «يعود التوصيل الثالثة»
// والمنصّةُ لا تستقبل حتّى الخامسة كذب.**
func TestZH28_CombinedNextOrderingTime(t *testing.T) {
	// **المنطقةُ تفتح ١٥:٠٠، والمنصّةُ لا تستقبل قبل ١٧:٠٠.**
	zone := Schedule{win(t, 0, "15:00", "23:00")}
	plat := Schedule{win(t, 0, "17:00", "23:00")}
	got := NextBothOpen(at(0, 10, 0, 0), true, plat, Closure{}, true, zone)
	if got == nil || !got.Equal(at(0, 17, 0, 0)) {
		t.Fatalf("**الموعدُ ١٧:٠٠ لا ١٥:٠٠** — وجاء %v", got)
	}
}

// **والعكسُ كذلك**: المنصّةُ تسبق والمنطقةُ تتأخّر.
func TestZH28b_ZoneIsTheLaterOne(t *testing.T) {
	zone := Schedule{win(t, 0, "18:00", "23:00")}
	plat := Schedule{win(t, 0, "09:00", "23:00")}
	got := NextBothOpen(at(0, 10, 0, 0), true, plat, Closure{}, true, zone)
	if got == nil || !got.Equal(at(0, 18, 0, 0)) {
		t.Fatalf("**الموعدُ ١٨:٠٠** — وجاء %v", got)
	}
}

// **والإيقافُ المؤقّتُ يدخل الحسبة** — **ولا يُقال «نعود» وهو سارٍ.**
func TestZH28c_ClosureEntersTheIntersection(t *testing.T) {
	zone := Schedule{win(t, 0, "08:00", "23:00")}
	plat := Schedule{win(t, 0, "08:00", "23:00")}
	ends := at(0, 14, 0, 0)
	got := NextBothOpen(at(0, 10, 0, 0), true, plat,
		Closure{Active: true, EndsAt: &ends}, true, zone)
	if got == nil || !got.Equal(ends) {
		t.Fatalf("**الموعدُ منتهى الإيقاف ١٤:٠٠** — وجاء %v", got)
	}
}

// **وإيقافٌ بلا موعدٍ لا موعدَ بعده** — **ولا يُخترَع.**
func TestZH28d_IndefiniteClosureHasNoNext(t *testing.T) {
	sch := Schedule{win(t, 0, "08:00", "23:00")}
	got := NextBothOpen(at(0, 10, 0, 0), true, sch,
		Closure{Active: true}, true, sch)
	if got != nil {
		t.Fatalf("**اخترعَ موعداً والإيقافُ بلا نهاية**: %v", got)
	}
}

// **ومفتوحان الآن ⇒ الآن.**
func TestZH28e_BothOpenNow(t *testing.T) {
	sch := Schedule{win(t, 0, "08:00", "23:00")}
	now := at(0, 10, 0, 0)
	got := NextBothOpen(now, true, sch, Closure{}, true, sch)
	if got == nil || !got.Equal(now) {
		t.Fatalf("**مفتوحان الآن فالموعدُ الآن** — وجاء %v", got)
	}
}

// **وغيرُ الساريَين لا يمنعان شيئاً** — الموعدُ الآن.
func TestZH28f_NeitherEnforced(t *testing.T) {
	now := at(0, 3, 0, 0)
	got := NextBothOpen(now, false, Schedule{}, Closure{}, false, Schedule{})
	if got == nil || !got.Equal(now) {
		t.Fatalf("**لا جدولَ ساريَ ومع ذلك أُجِّل**: %v", got)
	}
}
