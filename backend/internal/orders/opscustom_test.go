package orders

import "testing"

// TestOpsCustomPathHasNoMerchant **ولا متجرَ في مسار الطلب الخاصّ.**
//
// **وهذا حارسُ الخبر الكاذب** — (شكوى المالك ٢٠٢٦-٠٨-١٣ بعد أن جرّب
// طلباً خاصّاً: «شوف في قيد التجهيز بالمتجر غلط»).
//
// **بطاقتُه كانت تقول «بانتظار قبول المتجر» و«قيد التجهيز في المتجر»
// و«السائق إلى المتجر»** — والسائقُ يشتريه بنفسه من حيث وجده.
func TestOpsCustomPathHasNoMerchant(t *testing.T) {
	merchantOnly := map[Stage]bool{
		StageWaiting: true, StagePreparing: true, StageToStore: true,
	}
	for _, st := range OpsCustomStages() {
		if merchantOnly[st] {
			t.Fatalf("مرحلةُ متجرٍ في مسار الطلب الخاصّ: %s", st)
		}
	}
}

// TestOpsCustomStageOfWalksTheOwnersOrder **وترتيبُ المالك هو المسار.**
//
// «أوّل شي لازم الطلب ياخذه سائق · بعدين يوثّق السعر وأجرة التوصيل ·
//
//	بعدين السائق يجيب الطلب · بعدها بالطريق إلى الزبون · بعدها وصل
//	الزبون · بعدها يسلّم الطلب».
func TestOpsCustomStageOfWalksTheOwnersOrder(t *testing.T) {
	steps := []struct {
		status string
		agreed bool
		want   Stage
	}{
		{StPending, false, StageWaitingPlatform},
		{StDispatching, false, StageSeekingDriver},
		{StAssigned, false, StageAgreeing},
		{StAssigned, true, StageBuying}, // **وثّق فصار يشتري**
		{StPickedUp, true, StageOnTheWay},
		{StAtDropoff, true, StageArrived},
		{StDelivered, true, StageDelivered},
	}
	last := -1
	for _, c := range steps {
		got := OpsCustomStageOf(c.status, c.agreed)
		if got != c.want {
			t.Fatalf("%s (وثّق=%v): %s والمنتظر %s", c.status, c.agreed, got, c.want)
		}
		_, at := OpsStagesFor(KindCustom, c.status, c.agreed)
		// **والمسارُ يتقدّم ولا يرتدّ** — شريطٌ يعود إلى الوراء يُقرأ عطبا.
		if at <= last {
			t.Fatalf("%s موضعُها %d ولم تتقدّم عن %d", c.status, at, last)
		}
		last = at
	}
}

// TestAgreedIsWhatSeparatesAgreeingFromBuying **والتوثيقُ وحدَه يفصلهما.**
//
// **الحالُ يبقى `assigned` قبل التوثيق وبعده** — فمن طوى المسارَ من
// الحال وحدَه **لا يعرف أيتّفق السائقُ أم يشتري**، وهما سؤالان مختلفان
// حين يتأخّر طلب.
func TestAgreedIsWhatSeparatesAgreeingFromBuying(t *testing.T) {
	if OpsCustomStageOf(StAssigned, false) == OpsCustomStageOf(StAssigned, true) {
		t.Fatal("التوثيقُ لا يحرّك المسار — فالحالُ وحدَه هو ما يُقرأ")
	}
}

// TestOpsCustomTimesTakeAgreementForBuying **ووقتُ الشراء لحظةُ التوثيق.**
//
// **لا حدثَ في السجلّ لبدء الشراء** — والذي يُعرف هو اللحظةُ التي صار
// فيها مسموحاً (`ErrCustomNotAgreed` تمنعه قبلها).
func TestOpsCustomTimesTakeAgreementForBuying(t *testing.T) {
	agreed := at(9, 20)
	times := OpsCustomStageTimes([]Event{
		{ToStatus: StPending, CreatedAt: at(9, 0)},
		{ToStatus: StDispatching, CreatedAt: at(9, 5)},
		{ToStatus: StAssigned, CreatedAt: at(9, 10)},
		{ToStatus: StPickedUp, CreatedAt: at(9, 45)},
	}, &agreed)

	idx := map[Stage]int{}
	for i, st := range OpsCustomStages() {
		idx[st] = i
	}
	if got := times[idx[StageBuying]]; got == nil || !got.Equal(agreed) {
		t.Fatalf("وقتُ الشراء %v والمنتظر وقتَ التوثيق", got)
	}
	if got := times[idx[StageAgreeing]]; got == nil || !got.Equal(at(9, 10)) {
		t.Fatalf("وقتُ التوثيق %v والمنتظر لحظةَ الإسناد", got)
	}
	// **وبلا توثيقٍ لا وقتَ للشراء** — لم يبدأ.
	none := OpsCustomStageTimes([]Event{
		{ToStatus: StAssigned, CreatedAt: at(9, 10)},
	}, nil)
	if none[idx[StageBuying]] != nil {
		t.Fatal("شراءٌ له وقتٌ ولم يُوثَّق بعد")
	}
}
