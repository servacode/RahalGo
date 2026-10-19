package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/opsmap"
)

// TestCustomerLive_ReachesOpsMap **موضعُ الزبون الحيُّ يصل خريطةَ العمليات** (طلبُ المالك
// ٢٠٢٦-١٠-١٩) — يُبثّ على `ops` بحمولته، **ويحمله الطلبُ في قائمة الخريطة** مع بعده عن
// الدبّوس، **وإيقافُ المشاركة يُبثّ ويُمحى.**
func TestCustomerLive_ReachesOpsMap(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	orderID, owner := f.liveOrder(t, "on_the_way", &d)
	ch, cancel := f.srv.hub.Subscribe([]string{"ops"})
	defer cancel()
	next := func() map[string]any {
		t.Helper()
		deadline := time.After(2 * time.Second)
		for {
			select {
			case raw := <-ch:
				var ev map[string]any
				_ = json.Unmarshal(raw, &ev)
				if ev["type"] == "customer_location" {
					return ev
				}
			case <-deadline:
				return nil
			}
		}
	}

	w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", preciseAt100)
	if w.Code != http.StatusOK {
		t.Fatalf("الكتابة: %d %s", w.Code, w.Body.String())
	}
	ev := next()
	if ev == nil || ev["order_id"] != orderID || ev["lat"] == nil || ev["stopped"] != nil {
		t.Fatalf("لم يُبثّ موضعُ الزبون على ops: %v", ev)
	}

	list, err := opsmap.ActiveOrders(context.Background(), f.pool, nil, opsmap.OrderFilter{}, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	var live *opsmap.CustomerLive
	for _, o := range list {
		if o.ID == orderID {
			live = o.CustomerLive
		}
	}
	if live == nil || live.PinGapM < 50 {
		t.Fatalf("الطلبُ على الخريطة بلا موضع الزبون أو بلا بعده عن الدبّوس: %+v", live)
	}

	w = liveCall(f.srv.handleStopCustomerLiveLocation, http.MethodDelete, orderID, owner, "customer", "")
	if w.Code != http.StatusOK {
		t.Fatalf("الإيقاف: %d %s", w.Code, w.Body.String())
	}
	if ev := next(); ev == nil || ev["stopped"] != true {
		t.Fatalf("الإيقافُ لم يُبثّ: %v", ev)
	}
}
