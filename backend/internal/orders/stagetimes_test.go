package orders

import (
	"testing"
	"time"
)

func at(h, m int) time.Time {
	return time.Date(2026, 8, 12, h, m, 0, 0, time.UTC)
}

// TestOpsStageTimesFoldsEachTransition **كلُّ مرحلةٍ تأخذ وقتَ دخولها.**
//
// **وهذا حارسُ الطيّ**: من بدّل جدولَ `OpsStageOf` فنقل حالاً من مرحلةٍ
// إلى أخرى **يرى الوقتَ ينتقل معه هنا** — لا في بطاقةٍ على شاشة.
func TestOpsStageTimesFoldsEachTransition(t *testing.T) {
	got := OpsStageTimes([]Event{
		{ToStatus: StPending, CreatedAt: at(9, 0)},
		{ToStatus: StAccepted, CreatedAt: at(9, 5)},
		{ToStatus: StDispatching, CreatedAt: at(9, 20)},
		{ToStatus: StAssigned, CreatedAt: at(9, 25)},
		{ToStatus: StAtPickup, CreatedAt: at(9, 31)},
		{ToStatus: StPickedUp, CreatedAt: at(9, 35)},
		{ToStatus: StOnTheWay, CreatedAt: at(9, 35)},
		{ToStatus: StAtDropoff, CreatedAt: at(9, 48)},
		{ToStatus: StDelivered, CreatedAt: at(9, 50)},
	})

	if len(got) != len(OpsStages()) {
		t.Fatalf("طولُ الأوقات %d ومراحلُ المكتب %d", len(got), len(OpsStages()))
	}
	want := []time.Time{
		at(9, 0),  // بانتظار قبول المتجر
		at(9, 5),  // قيد التجهيز في المتجر
		at(9, 20), // بانتظار سائق
		at(9, 25), // السائق إلى المتجر — **الإسنادُ لا الوصول**
		at(9, 35), // في الطريق إلى الزبون — **لحظةُ استلام السائق**
		at(9, 48), // وصل إلى الزبون
		at(9, 50), // تمّ التسليم
	}
	for i := range want {
		if got[i] == nil {
			t.Fatalf("المرحلة %d (%s) بلا وقت", i, OpsStages()[i])
		}
		if !got[i].Equal(want[i]) {
			t.Fatalf("المرحلة %d (%s): وقتُها %s والمنتظر %s",
				i, OpsStages()[i], got[i].Format("15:04"), want[i].Format("15:04"))
		}
	}
}

// TestOpsStageTimesKeepsTheFirstEntry **وأوّلُ دخولٍ لا آخره.**
//
// **الطلبُ يُعاد إلى الطابور فيدخل «بانتظار سائق» مرّتين.** وسؤالُ المكتب
// «متى بدأ الانتظار» — **ولو دُهس الأوّلُ بالثاني لَبدا الطلبُ كأنّه نزل
// الطابورَ لتوّه**، وهو واقفٌ فيه منذ نصف ساعة.
func TestOpsStageTimesKeepsTheFirstEntry(t *testing.T) {
	got := OpsStageTimes([]Event{
		{ToStatus: StDispatching, CreatedAt: at(9, 20)},
		{ToStatus: StAssigned, CreatedAt: at(9, 25)},
		{ToStatus: StDispatching, CreatedAt: at(9, 40)}, // رُدّ إلى الطابور
	})
	i := OpsStageIndex(StDispatching)
	if got[i] == nil || !got[i].Equal(at(9, 20)) {
		t.Fatalf("وقتُ «بانتظار سائق» %v والمنتظر ٩:٢٠", got[i])
	}
}

// TestOpsStageTimesSkipsWhatEndedTheOrder **وما أنهى الطلبَ ليس مرحلةً بلغها.**
//
// **الإلغاءُ والرفضُ يقعان به لا يقع هو فيهما** — و`OpsStageIndex` تردّ
// `-1`. **ولو حُشر في المصفوفة لَخرج عن حدّها** أو دهس مرحلةً أخرى.
func TestOpsStageTimesSkipsWhatEndedTheOrder(t *testing.T) {
	got := OpsStageTimes([]Event{
		{ToStatus: StPending, CreatedAt: at(9, 0)},
		{ToStatus: StCancelled, CreatedAt: at(9, 3)},
	})
	for i := 1; i < len(got); i++ {
		if got[i] != nil {
			t.Fatalf("المرحلة %d أخذت وقتَ الإلغاء", i)
		}
	}
}
