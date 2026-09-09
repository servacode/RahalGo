package orders

import (
	"sync"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **من يصله الحدثُ لم يتبدّل** — حارسُ الغرف
// ══════════════════════════════════════════════════════════════════════
//
// # لماذا وُجد
//
// **دورةُ ٤٤ بدّلت ما يصل الغرفَ لا من يصله** — **والغرفُ عهدٌ قائم.**
// **وأُعيدت كتابةُ `publishOrder` كلِّها** لتبني الحمولةَ بالسماح،
// **وسقطت في أثناء ذلك إشارةُ الطابور** (`drivers:queue`) **ودفعُ
// العرض** — **ولم يسقط اختبارٌ واحد**: `TestQueueAffecting` يفحص
// الدالّةَ وحدَها، **والدالّةُ سليمةٌ ولا يناديها أحد.**
//
// **وشيفرةٌ ميّتةٌ سليمةٌ لا يُمسكها فحصُ سلامتِها.**
//
// # فما يُقاس هنا
//
// **الغرفُ التي يبثّ إليها `publishOrder` فعلاً** — بمِسنَدٍ يعدّها،
// **لا بقراءةِ الشيفرة.**

// spyPub بائثٌ يعدّ الغرف.
type spyPub struct {
	mu     sync.Mutex
	topics []string
}

func (p *spyPub) Publish(topic string, _ any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.topics = append(p.topics, topic)
}

func (p *spyPub) has(topic string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, t := range p.topics {
		if t == topic {
			return true
		}
	}
	return false
}

func TestPublishOrderReachesEveryRoom(t *testing.T) {
	drv := "drv-1"
	o := &Order{
		ID: "ord-1", CustomerID: "cus-1", MerchantID: "mer-1",
		DriverID: &drv, Status: StAssigned,
	}
	p := &spyPub{}
	s := &Service{pub: p}
	s.publishOrder(o)

	// **والطابورُ منها** — `StAssigned` تُخرج الطلبَ منه، **ومن لم
	// يُبلَّغ رأى طلباً أخذه غيرُه.**
	for _, room := range []string{
		"ops", "merchant:mer-1", "customer:cus-1", "driver:drv-1", topicDriverQueue,
	} {
		if !p.has(room) {
			t.Errorf("**لم يصل الغرفةَ %q شيء** — **ومن لا يصله الحدثُ "+
				"يقرأ قديماً حتّى يُحدّث بيده.**", room)
		}
	}
}

// **وما لا شأنَ للطابور به لا يوقظه.**
func TestPublishOrderKeepsQueueQuiet(t *testing.T) {
	o := &Order{ID: "ord-1", CustomerID: "cus-1", MerchantID: "mer-1", Status: StPreparing}
	p := &spyPub{}
	s := &Service{pub: p}
	s.publishOrder(o)

	if p.has(topicDriverQueue) {
		t.Errorf("**أُيقظ الطابورُ على %q** — **وإيقاظُ كلِّ سائقٍ بلا "+
			"سببٍ يُطفئ إشعاراتِه.**", StPreparing)
	}
	if !p.has("customer:cus-1") {
		t.Error("ولم يصل الزبونَ شيء")
	}
}
