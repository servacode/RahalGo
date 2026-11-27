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

// **الختمُ يصل حيّاً** (قرارُ المالك ٢٠٢٦-١١-٢٧): تسليمُ الطلب يبثّ إشارةَ `loyalty`
// للزبون وللمتجر، **وغيرُ التسليم لا يبثّها.**
type typedPub struct {
	mu  sync.Mutex
	got map[string][]string
}

func (p *typedPub) Publish(topic string, ev any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.got == nil {
		p.got = map[string][]string{}
	}
	if m, ok := ev.(map[string]any); ok {
		if ty, _ := m["type"].(string); ty != "" {
			p.got[topic] = append(p.got[topic], ty)
		}
	}
}

func (p *typedPub) hasType(topic, ty string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, t := range p.got[topic] {
		if t == ty {
			return true
		}
	}
	return false
}

func TestPublishOrderDeliveredEmitsLoyaltyTouch(t *testing.T) {
	p := &typedPub{}
	s := &Service{pub: p}
	s.publishOrder(&Order{ID: "o1", CustomerID: "cus-1", MerchantID: "mer-1", Status: StDelivered})
	for _, room := range []string{"customer:cus-1", "merchant:mer-1"} {
		if !p.hasType(room, "loyalty") {
			t.Errorf("لم تصل إشارةُ loyalty الغرفةَ %q", room)
		}
	}
	q := &typedPub{}
	(&Service{pub: q}).publishOrder(&Order{ID: "o2", CustomerID: "cus-1", MerchantID: "mer-1", Status: StAssigned})
	if q.hasType("customer:cus-1", "loyalty") || q.hasType("merchant:mer-1", "loyalty") {
		t.Error("إشارةُ loyalty خرجت قبل التسليم")
	}
}
