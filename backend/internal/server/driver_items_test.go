package server

// ══════════════════════════════════════════════════════════════════════
// **السائقُ يعرف ما يستلمه** (ملاحظةُ المالك ٢٠٢٦-١٠-١٢)
// ══════════════════════════════════════════════════════════════════════
//
// «بالبطاقة لازم يكون واضح ع السائق شو هو الطلب… هيك السائق اصلا مايعرف شو الطلب».
// **كان `items_count` وحدَه يصل** — والأسماءُ في القاعدة لا تخرج إليه.

import (
	"context"
	"testing"
)

func TestDriverOrders_ItemsByName(t *testing.T) {
	f := newDriverFixture(t, 1)
	driver := f.drivers[0]
	orderID := f.atDropoffOrder(t, driver)
	if _, err := f.pool.Exec(context.Background(), `
		INSERT INTO order_items (order_id, name, unit_price, merchant_price, qty, options)
		VALUES ($1, 'وجبة شاورما عربي', 225, 225, 2, '[{"group":"الحجم","name":"دبل","price_delta":50},{"group":"إضافات","name":"جبنة","price_delta":25}]'),
		       ($1, 'كولا', 50, 50, 1, '[]')`, orderID); err != nil {
		t.Fatal(err)
	}

	list := f.driverOrders(t, driver)
	if len(list) != 1 {
		t.Fatalf("قائمتُه %d", len(list))
	}
	items, _ := list[0]["items"].([]any)
	if len(items) < 2 {
		t.Fatalf("الأصنافُ لم تصل: %v", list[0]["items"])
	}
	found := map[string]map[string]any{}
	for _, raw := range items {
		it, _ := raw.(map[string]any)
		name, _ := it["name"].(string)
		found[name] = it
	}
	sh := found["وجبة شاورما عربي"]
	if sh == nil || sh["qty"] != float64(2) || sh["options"] != "دبل، جبنة" {
		t.Fatalf("الصنفُ بكميّته وخياراته: %v", sh)
	}
	if c := found["كولا"]; c == nil || c["qty"] != float64(1) || c["options"] != "" {
		t.Fatalf("الصنفُ بلا خيارات: %v", c)
	}
	if _, leaked := sh["unit_price"]; leaked {
		t.Fatalf("سعرُ الصنف وصل السائق: %v", sh)
	}
}
