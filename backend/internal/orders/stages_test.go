package orders

import "testing"

// TestEveryStatusHasAStage **حالٌ لا مرحلةَ له يُعرض فراغاً في ثلاث شاشات.**
//
// **وهذا هو الحارسُ الذي يجعل المركزيّة تعمل**: من أضاف حالاً في المحرّك
// ولم يضعه في `StageOf` **يسقط هنا** — لا في شكوى زبونٍ بعد شهر.
func TestEveryStatusHasAStage(t *testing.T) {
	live := []string{
		StPending, StAccepted, StPreparing, StDispatching,
		StAssigned, StAtPickup, StPickedUp, StOnTheWay,
		StAtDropoff, StDelivered,
	}
	for _, st := range live {
		if StageOf(st) == StageEnded {
			t.Errorf("حالٌ حيٌّ بلا مرحلة: %s", st)
		}
		if StageIndex(st) < 0 {
			t.Errorf("حالٌ حيٌّ بلا موضعٍ على الشريط: %s", st)
		}
		if CustomStageOf(st) == StageEnded {
			t.Errorf("حالٌ حيٌّ بلا مرحلةٍ في الطلب الخاصّ: %s", st)
		}
	}

	// **وما انتهى قبل أن يصل لا مسارَ له** — يُقال بالحرف لا يُرسم.
	for _, st := range []string{StRejected, StCancelled, StFailed, StRefunded} {
		if StageOf(st) != StageEnded {
			t.Errorf("حالٌ منتهٍ وُضع على المسار: %s", st)
		}
		if StageIndex(st) != -1 {
			t.Errorf("حالٌ منتهٍ له موضع: %s", st)
		}
	}

	// **ومجهولٌ يُعدّ منتهيا** — لا «بانتظار»: من عرضه في أوّل المسار
	// أوهم صاحبَه أنّ طلبَه لم يبدأ وقد انتهى.
	if StageOf("something_new") != StageEnded {
		t.Error("حالٌ مجهولٌ عُدّ حيّا")
	}
}

// TestCustomerFoldsTheStoreLeg **الزبونُ لا يرى المتجرَ ولا السائق.**
//
// (قرارُ المالك: «ما يهمّه أنّ السائق راح على متجر أو لا».)
func TestCustomerFoldsTheStoreLeg(t *testing.T) {
	folded := []string{StPreparing, StDispatching, StAssigned, StAtPickup}
	for _, st := range folded {
		if StageOf(st) != StagePreparing {
			t.Errorf("%s لم تُطوَ في «قيد التجهيز» — بل %s", st, StageOf(st))
		}
	}
	// **وما بعد الاستلام يُقرأ «في الطريق»** — لا «قيد التجهيز».
	if StageOf(StPickedUp) != StageOnTheWay {
		t.Error("الاستلامُ لم ينقل الزبونَ إلى «في الطريق»")
	}
	// **و«وصل» مرحلةٌ قائمةٌ بذاتها** — وهي أهمُّ ما ينتظره الزبون،
	// **وكانت مطويّةً في «في الطريق» فلا يعرف أنّ السائق عند بابه.**
	if StageOf(StAtDropoff) != StageArrived {
		t.Error("الوصولُ مطويٌّ في «في الطريق»")
	}
}

// TestStagesForPicksTheRightPath **ولا مطبخَ في الطلب الخاصّ.**
func TestStagesForPicksTheRightPath(t *testing.T) {
	if _, at := StagesFor("custom", StPickedUp); at != 2 {
		t.Errorf("«اشترى» ليست الثالثة في الطلب الخاصّ: %d", at)
	}
	if _, at := StagesFor("", StPickedUp); at != 3 {
		t.Errorf("«في الطريق» ليست الرابعة في الطلب العاديّ: %d", at)
	}
	if list, _ := StagesFor("custom", StPending); list[2] != StageBought {
		t.Error("مسارُ الطلب الخاصّ فيه «قيد التجهيز»")
	}
}
