package orders

import "testing"

// TestRequeuedOrderShowsTheDriverWhoKeptIt **الاسمُ والوقتُ لرجلٍ واحد.**
//
// ══════════════════════════════════════════════════════════════════════
// **العطبُ الذي يُصلحه**
// ══════════════════════════════════════════════════════════════════════
//
// (كشفه جردُ الحالات ٢٠٢٦-٠٨-١٣، وأقرّ المالكُ إصلاحَه.)
//
// طلبٌ أخذه سائقٌ ثمّ ردّه، فأخذه ثانٍ. **والسطرُ على خطّ الإسناد يحمل
// اسمَ من يحمله الآن** (`driver_name`) — **فلو حمل وقتَ الأوّل لَجمع
// رجلين**: اسمُ من يحمله ووقتُ من تركه.
//
// **وفي شكوى، هذا بعينه ما يقلب الحقيقة**: يُسأل سائقٌ عن ساعةٍ لم يكن
// فيها صاحبَ الطلب.
func TestRequeuedOrderShowsTheDriverWhoKeptIt(t *testing.T) {
	times := OpsStageTimes([]Event{
		{ToStatus: StPending, CreatedAt: at(9, 0)},
		{ToStatus: StAccepted, CreatedAt: at(9, 10)},
		{ToStatus: StDispatching, CreatedAt: at(9, 20)}, // نزل الطابور
		{ToStatus: StAssigned, CreatedAt: at(9, 25)},    // أخذه الأوّل
		{ToStatus: StDispatching, CreatedAt: at(9, 40)}, // ردّه
		{ToStatus: StAssigned, CreatedAt: at(10, 40)},   // أخذه الثاني
	})

	// **وقتُ الإسناد وقتُ من يحمله الآن.**
	i := OpsStageIndex(StAssigned)
	if times[i] == nil || !times[i].Equal(at(10, 40)) {
		t.Fatalf("وقتُ الإسناد %v والمنتظر ١٠:٤٠ — وقتَ من أبقى الطلب", times[i])
	}

	// **ولا يضيع قياسُ الانتظار**: «بانتظار سائق» على أوّل نزولٍ للطابور،
	// **فالمدّةُ على الخطّ تقول كم انتظر حتّى أخذه من أبقاه** — ساعةً
	// وعشرين، وهي الحقيقةُ كاملةً لا نصفَها.
	j := OpsStageIndex(StDispatching)
	if times[j] == nil || !times[j].Equal(at(9, 20)) {
		t.Fatalf("بدءُ الانتظار %v والمنتظر ٩:٢٠ — أوّلَ نزولٍ للطابور", times[j])
	}
}

// TestArrivingAtStoreDoesNotEraseTheAssignment **والوصولُ ليس إسنادا.**
//
// **`at_pickup` يقع في «السائق إلى المتجر» نفسِها** — وصل ولم يستلم
// بعد. **فلو عُدّ دخولاً لَصار وقتُ المرحلة وقتَ وصوله لا وقتَ إسناده**،
// **فتختفي دقائقُ طريقه إلى المتجر من السجلّ كلِّه.**
//
// **وأمسكه الاختبارُ قبل أن يُدفَع** — وهو ما جعل قاعدةَ «آخر دخول»
// تُقيَّد بالعبور من مرحلةٍ أخرى لا بأيّ حدث.
func TestArrivingAtStoreDoesNotEraseTheAssignment(t *testing.T) {
	times := OpsStageTimes([]Event{
		{ToStatus: StDispatching, CreatedAt: at(9, 20)},
		{ToStatus: StAssigned, CreatedAt: at(9, 25)},
		{ToStatus: StAtPickup, CreatedAt: at(9, 31)}, // وصل المتجر
	})
	i := OpsStageIndex(StAssigned)
	if times[i] == nil || !times[i].Equal(at(9, 25)) {
		t.Fatalf("وقتُ الإسناد %v والمنتظر ٩:٢٥ — لا وقتَ وصوله المتجر", times[i])
	}
}

// TestCustomRequeueKeepsTheSameRule **والمسارُ الخاصّ بالقاعدة نفسِها.**
//
// **`agreeing` هي مرحلةُ الإسناد فيه** — يدخلها بأخذه الطلبَ ويخرج منها
// بتوثيقه. **وقاعدةٌ تُطبَّق في مسارٍ وتُنسى في آخرَ ليست قاعدة.**
func TestCustomRequeueKeepsTheSameRule(t *testing.T) {
	times := OpsCustomStageTimes([]Event{
		{ToStatus: StPending, CreatedAt: at(9, 0)},
		{ToStatus: StDispatching, CreatedAt: at(9, 5)},
		{ToStatus: StAssigned, CreatedAt: at(9, 10)}, // أخذه الأوّل
		{ToStatus: StDispatching, CreatedAt: at(9, 30)},
		{ToStatus: StAssigned, CreatedAt: at(9, 50)}, // أخذه الثاني
	}, nil)

	idx := map[Stage]int{}
	for i, st := range OpsCustomStages() {
		idx[st] = i
	}
	if got := times[idx[StageAgreeing]]; got == nil || !got.Equal(at(9, 50)) {
		t.Fatalf("إسنادُ الخاصّ %v والمنتظر ٩:٥٠", got)
	}
	if got := times[idx[StageSeekingDriver]]; got == nil || !got.Equal(at(9, 5)) {
		t.Fatalf("بدءُ انتظار الخاصّ %v والمنتظر ٩:٠٥", got)
	}
}
