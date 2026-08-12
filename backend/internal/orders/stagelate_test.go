package orders

import "testing"

// lim مهلٌ معروفةٌ للاختبار — **بالدقائق.**
var lim = StageLimits{Accept: 5, Prep: 20, Driver: 10, Handover: 5}

// TestOpsStageLateMarksTheSlowLeg **العلامةُ تقع على الخطّ البطيء وحدَه.**
//
// **وهذا حارسُ الشكوى**: من فتح السجلَّ ليعرف من يُسأل عن ساعةٍ ضائعة
// **يجب أن يجد العلامةَ حيث ضاعت** — لا على كلّ خطٍّ ولا على غير موضعها.
func TestOpsStageLateMarksTheSlowLeg(t *testing.T) {
	// المطبخُ أخذ ٤٥ دقيقةً ومهلتُه ٢٠ — **وما عداه في مهلته.**
	got := OpsStageLate(OpsStageTimes([]Event{
		{ToStatus: StPending, CreatedAt: at(9, 0)},
		{ToStatus: StAccepted, CreatedAt: at(9, 3)},     // ٣ د ≤ ٥
		{ToStatus: StDispatching, CreatedAt: at(9, 48)}, // ٤٥ د > ٢٠
		{ToStatus: StAssigned, CreatedAt: at(9, 52)},    // ٤ د ≤ ١٠
		{ToStatus: StPickedUp, CreatedAt: at(10, 5)},
		{ToStatus: StAtDropoff, CreatedAt: at(10, 20)},
		{ToStatus: StDelivered, CreatedAt: at(10, 22)}, // دقيقتان ≤ ٥
	}), lim)

	want := map[Stage]*bool{
		StagePreparing:     boolp(false),
		StageSeekingDriver: boolp(true),
		StageToStore:       boolp(false),
		StageDelivered:     boolp(false),
	}
	for i, st := range OpsStages() {
		w, judged := want[st]
		switch {
		case !judged && got[i] != nil:
			t.Fatalf("%s حُكم عليها وهي بلا مهلة", st)
		case judged && got[i] == nil:
			t.Fatalf("%s بلا حكمٍ ولها مهلة", st)
		case judged && *got[i] != *w:
			t.Fatalf("%s: الحكمُ %v والمنتظر %v", st, *got[i], *w)
		}
	}
}

// TestOpsStageLateStaysSilentWithoutALimit **وصفرُ المهلة صمتٌ لا اتّهام.**
//
// **صفرٌ يعني «لا مهلةَ لي على هذا الخطّ»** — ولو قُرئ حدّاً لَصار كلُّ
// انتقالٍ متأخّراً (أيُّ مدّةٍ أكبرُ من صفر)، **فتمتلئ البطاقةُ بحمرةٍ لا
// تدلّ على شيء.**
func TestOpsStageLateStaysSilentWithoutALimit(t *testing.T) {
	times := OpsStageTimes([]Event{
		{ToStatus: StPending, CreatedAt: at(9, 0)},
		{ToStatus: StAccepted, CreatedAt: at(11, 0)}, // ساعتان
	})
	got := OpsStageLate(times, StageLimits{})
	for i, st := range OpsStages() {
		if got[i] != nil {
			t.Fatalf("%s حُكم عليها بلا مهلة", st)
		}
	}
}

// TestOpsStageLateJudgesOnlyWhatWasReached **وما لم يُبلَغ لا يُحكم عليه.**
//
// **طلبٌ ما زال في المطبخ لا يُقال عن طريقه إنّه سليم** — لم يبدأ بعد.
// **و«✓» على خطٍّ لم يُعبَر** يطمئن من لا يجوز أن يطمئنّ.
func TestOpsStageLateJudgesOnlyWhatWasReached(t *testing.T) {
	got := OpsStageLate(OpsStageTimes([]Event{
		{ToStatus: StPending, CreatedAt: at(9, 0)},
		{ToStatus: StAccepted, CreatedAt: at(9, 2)},
	}), lim)
	i := OpsStageIndex(StDispatching) // بانتظار سائق — لم يبلغها
	if got[i] != nil {
		t.Fatalf("حُكم على مرحلةٍ لم تُبلَغ: %v", *got[i])
	}
	j := OpsStageIndex(StAccepted) // قيد التجهيز — بلغها في دقيقتين
	if got[j] == nil || *got[j] {
		t.Fatalf("قبولٌ في دقيقتين قُرئ متأخّرا: %v", got[j])
	}
}

func boolp(v bool) *bool { return &v }
