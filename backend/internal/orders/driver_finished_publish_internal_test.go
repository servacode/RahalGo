package orders

import (
	"sync"
	"testing"
	"time"
)

// capPub **بائثٌ يحفظ الحمولةَ لكلّ غرفة** — آخرُها.
type capPub struct {
	mu   sync.Mutex
	last map[string]any
}

func (p *capPub) Publish(topic string, payload any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.last == nil {
		p.last = map[string]any{}
	}
	p.last[topic] = payload
}

func (p *capPub) orderIn(t *testing.T, topic string) map[string]any {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	m, _ := p.last[topic].(map[string]any)
	if m == nil {
		t.Fatalf("لم يصل الغرفةَ %q شيء", topic)
	}
	o, _ := m["order"].(map[string]any)
	if o == nil {
		t.Fatalf("حمولةُ %q بلا طلب: %v", topic, m)
	}
	return o
}

// TestPublishOrder_FinishedReachesDriverWithoutCustomer **طلبٌ انتهى يصل سائقَه بلا زبون**
// (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٢ — مراجعةُ الفرع). **كان كلُّ تحديثٍ بعد التسليم**
// (استردادٌ أو تعويضٌ أو تعديلُ إدارة) **يُبثّ لغرفة السائق بقائمته كاملةً** — العنوانُ
// والموضعُ وبابُ البناية — فيصل عنوانُ طلبٍ انتهى شاشتَه بعد أيّام. **والإشارةُ نفسُها
// تبقى** (يُعيد بها قائمتَه فيختفي الطلب)، **ومشوارُ الإرجاع المفتوح يبقى كاملاً.**
func TestPublishOrder_FinishedReachesDriverWithoutCustomer(t *testing.T) {
	drv := "drv-1"
	now := time.Now()
	ent := 35.95
	base := func() *Order {
		return &Order{
			ID: "ord-1", Number: 77, CustomerID: "cus-1", MerchantID: "mer-1", DriverID: &drv,
			CustomerName: "أحمد محمد", AddressText: "شارع النور بناية 3",
			Lat: 35.95, Lng: 39.01, EntranceLat: &ent, EntranceLng: &ent,
		}
	}
	for _, st := range []string{StDelivered, StCancelled, StRejected, StFailed, StRefunded} {
		o := base()
		o.Status, o.ClosedAt = st, &now
		p := &capPub{}
		(&Service{pub: p}).publishOrder(o)
		got := p.orderIn(t, "driver:drv-1")
		for _, k := range []string{"customer_name", "address_text", "lat", "lng", "entrance_lat", "customer_id"} {
			if _, ok := got[k]; ok {
				t.Errorf("%s: غرفةُ السائق فيها %q — **طلبٌ انتهى**", st, k)
			}
		}
		if got["id"] != "ord-1" || got["status"] != st {
			t.Errorf("%s: الإشارةُ بلا رقمٍ أو حالة: %v", st, got)
		}
		// **والمكتبُ لا يُمسّ** — يرى الطلبَ كاملاً.
		if ops := p.orderIn(t, "ops"); ops["address_text"] != "شارع النور بناية 3" {
			t.Errorf("%s: غرفةُ المكتب فقدت العنوان: %v", st, ops["address_text"])
		}
	}

	// **ومشوارُ الإرجاع المفتوح يبقى كاملاً** — البضاعةُ معه والطلبُ في قائمته.
	o := base()
	office := "office"
	o.Status, o.ClosedAt, o.ReturnTo = StFailed, &now, &office
	p := &capPub{}
	(&Service{pub: p}).publishOrder(o)
	if got := p.orderIn(t, "driver:drv-1"); got["address_text"] == nil || got["return_to"] != "office" {
		t.Errorf("مشوارُ الإرجاع المفتوح فقد حقولَه: %v", got)
	}
	// **والجاري كما كان.**
	o = base()
	o.Status = StOnTheWay
	p = &capPub{}
	(&Service{pub: p}).publishOrder(o)
	if got := p.orderIn(t, "driver:drv-1"); got["address_text"] != "شارع النور بناية 3" {
		t.Errorf("الطلبُ الجاري فقد عنوانَه عند سائقه: %v", got)
	}
}
