package qa

// **«الأكتر مبيعاً هالأسبوع»** (قرارُ المالك ٢٠٢٦-١١-١١): أعلى ٥ أصناف بالكمية **المسلَّمة**
// في آخر ٧ أيام لمتجره وحده — لا الملغى، ولا الأقدم، ولا متجرٌ غيره.

import (
	"encoding/json"
	"testing"
	"time"
)

func tiOrder(t *testing.T, h *Harness, merchantID, itemID, name string, qty int, status string, delivered time.Time) {
	t.Helper()
	cust := h.Customer()
	var oid string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO orders (customer_id, merchant_id, status, address_text, dropoff,
		                    payment_method, subtotal, delivery_fee, total, delivered_at)
		VALUES ($1::uuid, $2::uuid, $3, 'الرقة',
		        ST_SetSRID(ST_MakePoint(38.5110,35.5000),4326)::geography, 'cash', 10, 5, 15, $4)
		RETURNING id::text`, cust.ID, merchantID, status, delivered).Scan(&oid); err != nil {
		t.Fatalf("إنشاء الطلب: %v", err)
	}
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM orders WHERE id = $1::uuid`, oid) })
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_items (order_id, merchant_id, menu_item_id, name, unit_price, merchant_price, qty)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, 10, 10, $5)`, oid, merchantID, itemID, name, qty); err != nil {
		t.Fatalf("صنف الطلب: %v", err)
	}
}

func TestTI01_TopItemsDeliveredLastWeekOwnStoreOnly(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	other := f.Merchant()
	a, b := h.NewItemPriced(m, 10, 10), h.NewItemPriced(m, 10, 10)
	c := h.NewItemPriced(other, 10, 10)
	now := time.Now()
	tiOrder(t, h, m.ID, a.ID, a.Name, 2, "delivered", now)
	tiOrder(t, h, m.ID, a.ID, a.Name, 3, "delivered", now.Add(-24*time.Hour))
	tiOrder(t, h, m.ID, b.ID, b.Name, 4, "delivered", now)
	tiOrder(t, h, m.ID, b.ID, b.Name, 50, "cancelled", now)                      // not delivered
	tiOrder(t, h, m.ID, b.ID, b.Name, 50, "delivered", now.Add(-9*24*time.Hour)) // too old
	tiOrder(t, h, other.ID, c.ID, c.Name, 40, "delivered", now)                  // another store

	res := h.GET("/api/v1/merchant/stores/"+m.ID+"/top-items", m.Owner.Token)
	if res.Code != 200 {
		t.Fatalf("top-items: %d %s", res.Code, res.Body)
	}
	var out struct {
		Data struct {
			Items []struct {
				Name string `json:"name"`
				Qty  int    `json:"qty"`
			} `json:"items"`
			ShareURL string `json:"share_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &out); err != nil {
		t.Fatal(err)
	}
	it := out.Data.Items
	if len(it) != 2 || it[0].Name != a.Name || it[0].Qty != 5 || it[1].Name != b.Name || it[1].Qty != 4 {
		t.Fatalf("**الترتيبُ أو الكمياتُ خطأ**: %+v", it)
	}
	if want := "/stores/" + m.ID; len(out.Data.ShareURL) < len(want) || out.Data.ShareURL[len(out.Data.ShareURL)-len(want):] != want {
		t.Fatalf("share_url = %q", out.Data.ShareURL)
	}
	if r := h.GET("/api/v1/merchant/stores/"+other.ID+"/top-items", m.Owner.Token); r.Code != 403 {
		t.Fatalf("**متجرُ غيره يجب 403**: %d", r.Code)
	}
	empty := f.Merchant()
	if r := h.GET("/api/v1/merchant/stores/"+empty.ID+"/top-items", empty.Owner.Token); r.Code != 200 {
		t.Fatalf("متجرٌ بلا مبيعات: %d", r.Code)
	}
}
