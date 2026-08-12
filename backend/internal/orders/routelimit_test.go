package orders

import (
	"testing"
	"time"
)

func sec(n int) *int { return &n }

// TestRouteLimitAddsTheMargin **المهلةُ تقديرُ الخريطة زائدَ الهامش.**
//
// **والخريطةُ أرضيّةٌ لا سقف**: تحسب على سرعاتٍ مفترضةٍ للشوارع، **ولا
// تعرف ازدحاماً ولا حاجزاً ولا إشارةً طويلة.**
func TestRouteLimitAddsTheMargin(t *testing.T) {
	// ستُّ دقائقَ من الخريطة، وهامشٌ نصف → تسع.
	if got := routeLimit(sec(360), 50); got != 9*time.Minute {
		t.Fatalf("المهلةُ %s والمنتظر ٩ دقائق", got)
	}
	// **وصفرُ الهامش يحاسب حرفيّا** — لا يرفع الحكم.
	if got := routeLimit(sec(360), 0); got != 6*time.Minute {
		t.Fatalf("بلا هامشٍ: %s والمنتظر ٦ دقائق", got)
	}
}

// TestRouteLimitStaysSilentWhenUnmeasured **وما لم يُقَس لا يُحكم عليه.**
//
// **الخريطةُ تتعطّل، والسائقُ قد يُغلق تتبّعَه، والعنوانُ قد يكون بلا
// إحداثيّ.** فيبقى العمودُ فارغاً — **ولا يُوسَم أحدٌ بتأخيرٍ لم يُقَس.**
func TestRouteLimitStaysSilentWhenUnmeasured(t *testing.T) {
	if got := routeLimit(nil, 50); got != 0 {
		t.Fatalf("حُكم بلا قياس: %s", got)
	}
	if got := routeLimit(sec(0), 50); got != 0 {
		t.Fatalf("صفرُ الثواني قُرئ مهلة: %s", got)
	}
}

// TestRouteLegsAreJudgedByTheirOwnMap **ولكلّ طلبٍ مهلةُ طريقه هو.**
//
// **وهذا هو العيبُ الذي مُنع**: مهلةٌ محسوبةٌ من خطٍّ مستقيمٍ تُقصّر
// الطريقَ نحوَ نصفِه (قِيس في الرقّة: الشارعُ أطولُ بـ٤٩٪ وسطيّاً) —
// **فتضع علامةً حمراءَ على سائقٍ سار الطريقَ الصحيح.**
//
// **فالمهلةُ من خريطة هذا الطلب**: مشوارٌ تقول الخريطةُ عشرَ دقائقَ
// وقُطع في اثنتَي عشرةَ **سليمٌ بهامش النصف**، ولو قِيس بخطٍّ مستقيمٍ
// (سبعُ دقائق) **لَعُدّ متأخّرا.**
func TestRouteLegsAreJudgedByTheirOwnMap(t *testing.T) {
	times := OpsStageTimes([]Event{
		{ToStatus: StAssigned, CreatedAt: at(9, 0)},
		{ToStatus: StPickedUp, CreatedAt: at(9, 12)},  // ١٢ د إلى المتجر
		{ToStatus: StAtDropoff, CreatedAt: at(9, 40)}, // ٢٨ د إلى الباب
	})
	got := OpsStageLate(times, StageLimits{
		ToStore: routeLimit(sec(600), 50), // خريطةٌ ١٠ د → مهلة ١٥
		ToDoor:  routeLimit(sec(900), 50), // خريطةٌ ١٥ د → مهلة ٢٢٫٥
	})

	i := OpsStageIndex(StPickedUp) // «في الطريق إلى الزبون»
	if got[i] == nil || *got[i] {
		t.Fatalf("مشوارٌ في مهلته قُرئ متأخّرا: %v", got[i])
	}
	j := OpsStageIndex(StAtDropoff) // «وصل إلى الزبون»
	if got[j] == nil || !*got[j] {
		t.Fatalf("مشوارٌ تجاوز مهلتَه لم يُعلَّم: %v", got[j])
	}
}
