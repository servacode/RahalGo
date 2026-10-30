package server

// ══════════════════════════════════════════════════════════════════════
// **السائقُ يعرف ما يستلمه** (ملاحظةُ المالك ٢٠٢٦-١٠-١٢)
// ══════════════════════════════════════════════════════════════════════
//
// «بالبطاقة لازم يكون واضح ع السائق شو هو الطلب… هيك السائق اصلا مايعرف شو الطلب».
// **كان `items_count` وحدَه يصل** — والأسماءُ في القاعدة لا تخرج إليه.
// **ومنذ ٢٠٢٦-١٠-٣٠ الفاتورةُ كاملة** — سعرُ البيع والمشروبُ بعد صنفه والخياراتُ بمجموعاتها.

import (
	"context"
	"testing"
)

func TestDriverOrders_ItemsByName(t *testing.T) {
	f := newDriverFixture(t, 1)
	driver := f.drivers[0]
	orderID := f.atDropoffOrder(t, driver)
	ctx := context.Background()
	var shawarma string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO order_items (order_id, name, unit_price, merchant_price, qty, options)
		VALUES ($1, 'وجبة شاورما عربي', 300, 225, 2, '[{"group":"الحجم","name":"دبل","price_delta":50},{"group":"إضافات","name":"جبنة","price_delta":25}]')
		RETURNING id::text`, orderID).Scan(&shawarma); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO order_items (order_id, name, unit_price, merchant_price, qty, options, drink_for)
		VALUES ($1, 'عيران', 25, 20, 2, '[]', $2::uuid),
		       ($1, 'كولا', 50, 40, 1, '[]', NULL),
		       ($1, 'بوط', 9100, 9000, 1, '[{"group":"النمرة","name":"40","price_delta":0}]', NULL)`,
		orderID, shawarma); err != nil {
		t.Fatal(err)
	}

	list := f.driverOrders(t, driver)
	if len(list) != 1 {
		t.Fatalf("قائمتُه %d", len(list))
	}
	items, _ := list[0]["items"].([]any)
	if len(items) != 4 {
		t.Fatalf("الأصنافُ لم تصل كلُّها: %v", list[0]["items"])
	}
	found := map[string]map[string]any{}
	at := map[string]int{}
	for i, raw := range items {
		it, _ := raw.(map[string]any)
		name, _ := it["name"].(string)
		found[name] = it
		at[name] = i
	}
	sh := found["وجبة شاورما عربي"]
	// **والنصُّ القديمُ باقٍ** لنسخ السائق القديمة.
	if sh == nil || sh["qty"] != float64(2) || sh["options"] != "دبل، جبنة" {
		t.Fatalf("الصنفُ بكميّته وخياراته: %v", sh)
	}
	// **وسعرُ البيع لا سعرُ الشراء** (قرارُ المالك ٢٠٢٦-١٠-٣٠).
	if sh["unit_price"] != float64(300) {
		t.Fatalf("سعرُ البيع لم يصل السائق: %v", sh)
	}
	if _, leaked := sh["merchant_price"]; leaked {
		t.Fatalf("سعرُ الشراء وصل السائق: %v", sh)
	}
	opts, _ := sh["option_list"].([]any)
	if len(opts) != 2 {
		t.Fatalf("الخياراتُ بمجموعاتها: %v", sh["option_list"])
	}
	if o, _ := opts[0].(map[string]any); o["group"] != "الحجم" || o["name"] != "دبل" {
		t.Fatalf("الخيارُ الأوّل: %v", opts[0])
	}
	if b := found["بوط"]; b == nil || b["unit_price"] != float64(9100) {
		t.Fatalf("البوطُ بسعره: %v", b)
	}
	c := found["كولا"]
	if c == nil || c["qty"] != float64(1) || c["options"] != "" || c["drink_for"] != nil {
		t.Fatalf("الصنفُ بلا خيارات: %v", c)
	}
	if o, _ := c["option_list"].([]any); o == nil || len(o) != 0 {
		t.Fatalf("خياراتٌ فارغةٌ تصل قائمةً فارغة: %v", c["option_list"])
	}
	// **والمشروبُ بعد صنفه مباشرةً** ومعه `drink_for`.
	d := found["عيران"]
	if d == nil || d["drink_for"] != shawarma || d["unit_price"] != float64(25) {
		t.Fatalf("المشروبُ وصنفُه: %v", d)
	}
	if at["عيران"] != at["وجبة شاورما عربي"]+1 {
		t.Fatalf("المشروبُ ليس بعد صنفه: %v", items)
	}
	if sh["id"] != shawarma {
		t.Fatalf("معرّفُ الصنف: %v", sh["id"])
	}
	// **ومجموعُ الأصناف والخصم** لكتلة الفاتورة.
	if _, ok := list[0]["subtotal"]; !ok {
		t.Fatalf("المجموعُ لم يصل: %v", list[0])
	}
	if _, ok := list[0]["discount"]; !ok {
		t.Fatalf("الخصمُ لم يصل: %v", list[0])
	}
}
