package orders

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/realtime"
)

// TestDriverQueueTopic الاسمُ المنشور هو الاسمُ المُشترَك فيه.
//
// **هذا اختبارُ عقدٍ بين طرفين لا يعرف أحدُهما الآخر**: `orders` تنشر عبر
// واجهة `Publisher` ولا تستورد `realtime`، و`server/ws.go` يشترك بالثابت.
// فلو غُيّر أحدهما وحده **لما انكسر بناءٌ ولا اشتكى مُدقّق** — يسكت كلُّ شيء
// ويصمت الطابور، ولا يُكتشف إلّا بسائقٍ يشتكي أنه لا يرى الطلبات.
func TestDriverQueueTopic(t *testing.T) {
	if topicDriverQueue != realtime.TopicDriverQueue {
		t.Fatalf("موضوع الطابور انحرف: تنشر %q ويُشترَك في %q",
			topicDriverQueue, realtime.TopicDriverQueue)
	}
}

// TestQueueAffecting ما يدخل الطابور وما يخرج منه.
//
// **الخروجُ لا يقلّ أهميةً عن الدخول**: طلبٌ أخذه سائقٌ ولم يختفِ عن شاشات
// الباقين يبقى إغراءً كاذباً — يضغطون عليه فيُردّون بـ«سبقك غيرُك»، وهو ردٌّ
// صحيح يُغني عنه عرضٌ صحيح.
func TestQueueAffecting(t *testing.T) {
	for _, st := range []string{StDispatching, StAssigned, StCancelled, StRejected, StFailed} {
		if !queueAffecting(st) {
			t.Errorf("queueAffecting(%s) = false، والطابور يتغيّر بها", st)
		}
	}
	// أوضاعٌ لا شأن للطابور بها — بثُّها يوقظ كلَّ سائقٍ بلا سبب.
	for _, st := range []string{StPending, StAccepted, StPreparing, StAtPickup, StOnTheWay, StDelivered} {
		if queueAffecting(st) {
			t.Errorf("queueAffecting(%s) = true، والطابور لا يتغيّر بها", st)
		}
	}
}
