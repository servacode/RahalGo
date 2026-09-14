package platform

// ══════════════════════════════════════════════════════════════════════
// **تقاطعُ القيود الزمنيّة — لا أصغرُ المواعيد** (`AV-13`…`AV-18`)
// ══════════════════════════════════════════════════════════════════════
//
// **ومن أخذ أصغرَ موعدٍ في كلّ جدولٍ على حدةٍ أجاب بموعدٍ لا يُطلَب فيه
// شيء** — **والزبونُ يعود في ذلك الموعد فيجد البابَ مغلقاً، ولا يعود
// ثالثةً.**

import (
	"testing"
	"time"
)

func gate(t *testing.T, day int, from, to string) Gate {
	t.Helper()
	return Gate{Enforced: true, Sch: Schedule{win(t, day, from, to)}}
}

// AV-15 · **ثلاثةُ قيودٍ تتقاطع في أبعدها.**
//
// **وهو مثالُ المالك نصّاً**: المنصّةُ ٠٩:٠٠ · المنطقةُ ١٠:٠٠ ·
// المتجرُ ١١:٣٠ ⇒ **١١:٣٠.**
func TestAV15_ThreeWayIntersection(t *testing.T) {
	got := NextAllOpen(at(0, 7, 0, 0), Closure{},
		gate(t, 0, "09:00", "23:00"),
		gate(t, 0, "10:00", "23:00"),
		gate(t, 0, "11:30", "23:00"))
	if got == nil || !got.Equal(at(0, 11, 30, 0)) {
		t.Fatalf("**الموعدُ ١١:٣٠ لا أصغرُ الثلاثة** — وجاء %v", got)
	}
}

// **ولا يكفي أن يفتح كلٌّ منها — بل أن يجتمعوا.**
//
// **والمتجرُ يفتح ٠٨:٠٠←١٠:٠٠ والمنطقةُ ١١:٠٠←١٣:٠٠** — **ولا تقاطعَ
// اليومَ**، **فالجوابُ في يومٍ آخرَ أو لا جواب.**
func TestAV15b_NoOverlapTodayMeansAnotherDay(t *testing.T) {
	merchant := Gate{Enforced: true, Sch: Schedule{
		win(t, 0, "08:00", "10:00"), win(t, 2, "09:00", "20:00"),
	}}
	zone := Gate{Enforced: true, Sch: Schedule{
		win(t, 0, "11:00", "13:00"), win(t, 2, "10:00", "18:00"),
	}}
	got := NextAllOpen(at(0, 7, 0, 0), Closure{}, merchant, zone)
	if got == nil {
		t.Fatal("**لا موعدَ وقد وُجد تقاطعٌ يومَ الثلاثاء**")
	}
	if !got.Equal(at(2, 10, 0, 0)) {
		t.Fatalf("**الموعدُ الثلاثاءَ ١٠:٠٠ — حيث يجتمعان** — وجاء %v", *got)
	}
}

// AV-13 · **موعدٌ في اليوم نفسِه.**
func TestAV13_SameDay(t *testing.T) {
	got := NextAllOpen(at(0, 8, 0, 0), Closure{},
		gate(t, 0, "09:00", "17:00"), gate(t, 0, "09:00", "17:00"))
	if got == nil || !got.Equal(at(0, 9, 0, 0)) {
		t.Fatalf("**الموعدُ ٠٩:٠٠ اليوم** — وجاء %v", got)
	}
}

// AV-14 · **وفي يومٍ تالٍ حين لا يبقى شيءٌ اليوم.**
func TestAV14_LaterDay(t *testing.T) {
	now := at(0, 20, 0, 0)
	// **والأوّلُ مفتوحٌ يومين والثاني يومَ الأربعاء وحدَه** — **فأوّلُ
	// اجتماعٍ بعد مساء الأحد هو الأربعاء.**
	both := Gate{Enforced: true, Sch: Schedule{
		win(t, 0, "09:00", "17:00"), win(t, 3, "09:00", "17:00"),
	}}
	got := NextAllOpen(now, Closure{}, both, gate(t, 3, "09:00", "17:00"))
	if got == nil {
		t.Fatal("**لا موعدَ وقد وُجد**")
	}
	if !got.After(now) {
		t.Fatalf("**موعدٌ في الماضي**: %v", *got)
	}
	// **والأربعاءُ هو أوّلُ يومٍ يجتمعان فيه.**
	if !got.Equal(at(3, 9, 0, 0)) {
		t.Fatalf("**الموعدُ الأربعاءَ ٠٩:٠٠** — وجاء %v", *got)
	}
}

// AV-16 · **تقاطعٌ يعبر منتصفَ الليل.**
//
// **والمنصّةُ ٢٠:٠٠←٠٣:٠٠ والمنطقةُ ٢٢:٠٠←٠٢:٠٠** — **فالتقاطعُ يبدأ
// ٢٢:٠٠.**
func TestAV16_CrossMidnightIntersection(t *testing.T) {
	got := NextAllOpen(at(0, 12, 0, 0), Closure{},
		gate(t, 0, "20:00", "03:00"),
		gate(t, 0, "22:00", "02:00"))
	if got == nil || !got.Equal(at(0, 22, 0, 0)) {
		t.Fatalf("**الموعدُ ٢٢:٠٠** — وجاء %v", got)
	}
}

// **وبعد منتصف الليل يبقى التقاطعُ قائماً حتّى ٠٢:٠٠.**
func TestAV16b_SpillIsStillOpen(t *testing.T) {
	p := gate(t, 0, "20:00", "03:00")
	z := gate(t, 0, "22:00", "02:00")
	inside := at(1, 1, 0, 0)
	got := NextAllOpen(inside, Closure{}, p, z)
	if got == nil || !got.Equal(inside) {
		t.Fatalf("**الواحدةُ ليلاً داخلَ التقاطع فالموعدُ الآن** — وجاء %v", got)
	}
	// **و٠٢:٠٠ حدُّ انتهاءِ المنطقة — فخارجُه.**
	after := NextAllOpen(at(1, 2, 0, 0), Closure{}, p, z)
	if after != nil && after.Equal(at(1, 2, 0, 0)) {
		t.Fatal("**٠٢:٠٠ حدُّ انتهاءٍ وهو خارج**")
	}
}

// AV-17 · **تقاطعٌ على حدّ الأسبوع** — سبتٌ يعبر إلى أحد.
func TestAV17_WeekBoundaryIntersection(t *testing.T) {
	got := NextAllOpen(at(6, 12, 0, 0), Closure{},
		gate(t, 6, "21:00", "04:00"),
		gate(t, 6, "23:00", "03:00"))
	if got == nil || !got.Equal(at(6, 23, 0, 0)) {
		t.Fatalf("**الموعدُ ٢٣:٠٠ ليلةَ السبت** — وجاء %v", got)
	}
	// **وفجرُ الأحد داخلَ التقاطع.**
	dawn := at(0, 2, 0, 0).AddDate(0, 0, 7)
	if in := NextAllOpen(dawn, Closure{},
		gate(t, 6, "21:00", "04:00"), gate(t, 6, "23:00", "03:00")); in == nil || !in.Equal(dawn) {
		t.Fatalf("**فجرُ الأحد داخلَ ذيلِ السبت** — وجاء %v", in)
	}
}

// AV-18 · **إيقافٌ بلا نهايةٍ لا موعدَ بعده.**
func TestAV18_IndefiniteClosureInventsNothing(t *testing.T) {
	got := NextAllOpen(at(0, 12, 0, 0), Closure{Active: true},
		gate(t, 0, "09:00", "23:00"))
	if got != nil {
		t.Fatalf("**اخترعَ موعداً والإيقافُ بلا نهاية**: %v", *got)
	}
}

// **وقيدٌ سارٍ بلا فتراتٍ لا يُفتح أبداً** — **وهو الإغلاقُ الطارئ:
// لا نعرف متى يُرفَع، فلا يُقال موعد.**
func TestAV18b_EmptyEnforcedGateHasNoNext(t *testing.T) {
	got := NextAllOpen(at(0, 12, 0, 0), Closure{},
		gate(t, 0, "09:00", "23:00"),
		Gate{Enforced: true, Sch: Schedule{}})
	if got != nil {
		t.Fatalf("**اخترعَ موعداً لقيدٍ لا يُفتح**: %v", *got)
	}
}

// **وقيودٌ غيرُ ساريةٍ لا تقيّد** — الموعدُ الآن.
func TestAV_UnenforcedGatesDoNotConstrain(t *testing.T) {
	now := at(0, 3, 0, 0)
	got := NextAllOpen(now, Closure{},
		Gate{}, Gate{}, Gate{})
	if got == nil || !got.Equal(now) {
		t.Fatalf("**قيودٌ غيرُ ساريةٍ أجّلت**: %v", got)
	}
}

// **ومنتهى الإيقاف مرشَّحٌ في التقاطع** — ويُزاح إلى أوّلِ فتحٍ بعده.
func TestAV_ClosureEndJoinsTheIntersection(t *testing.T) {
	ends := at(0, 14, 0, 0)
	got := NextAllOpen(at(0, 10, 0, 0), Closure{Active: true, EndsAt: &ends},
		gate(t, 0, "09:00", "23:00"),
		gate(t, 0, "16:00", "23:00"))
	if got == nil || !got.Equal(at(0, 16, 0, 0)) {
		t.Fatalf("**الموعدُ ١٦:٠٠ لا ١٤:٠٠** — وجاء %v", got)
	}
}

// **والأفقُ محدود** — **ولا دورانَ بلا نهاية**: جدولان لا يتقاطعان
// أبداً يردّان فارغاً في زمنٍ معقول.
func TestAV_BoundedHorizon(t *testing.T) {
	done := make(chan *time.Time, 1)
	go func() {
		done <- NextAllOpen(at(0, 12, 0, 0), Closure{},
			gate(t, 0, "08:00", "10:00"),
			gate(t, 0, "11:00", "13:00"))
	}()
	select {
	case got := <-done:
		if got != nil {
			t.Fatalf("**تقاطعٌ لا وجودَ له**: %v", *got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("**دارَ بلا نهاية** — والأفقُ يجب أن يكون محدوداً")
	}
}
