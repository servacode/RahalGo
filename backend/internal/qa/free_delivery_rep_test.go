package qa

import (
	"testing"
	"time"
)

// TestFreeDelivery_RepKeepsCommission **عرضُ التوصيل المجانيّ على المنصّة وحدها** (قرارُ المالك ٢٠٢٦-١٠-٢٦:
// «المندوب ما له علاقة، ياخذ عمولته طبيعي مثل السائق، المنصّة تتحمّل») — كان الإعفاءُ يُطرح من ربح المنصّة
// قبل نصيب المندوب (كالكوبون) فيأخذ صفراً. **والكوبونُ يبقى كما قُرّر ٢٠٢٦-١٠-١٥.**
func TestFreeDelivery_RepKeepsCommission(t *testing.T) {
	h := New(t)
	treasury(t, h)
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM free_delivery_campaigns`) })
	_, _ = h.Pool.Exec(ctxBG(), `UPDATE free_delivery_campaigns SET active = false`)
	h.Setting("merchants.commission_percent", "10")
	h.Setting("sales.commission_percent", "10")
	h.Setting("pricing.margin_fixed", "0")
	h.Setting("delivery.fee", "500")
	h.Setting("delivery.by_distance", "false")
	now := time.Now().UTC()
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO free_delivery_campaigns (active, starts_at, ends_at, kinds)
		VALUES (true, $1, $2, ARRAY['standard'])`, now.Add(-time.Hour), now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	f := h.Factory()
	r := f.RepAccount()
	m := f.Merchant(OwnedByRep(r.ID))
	item := h.NewItemFor(m, 10_000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("fdrep"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	var waived int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT promo_delivery_waived FROM orders WHERE id = $1`, oid).Scan(&waived); err != nil {
		t.Fatal(err)
	}
	if waived <= 0 {
		t.Fatalf("العرضُ لم يُطبَّق على الطلب: معفى %d", waived)
	}
	deliverOrder(t, h, oid, h.driverOf(oid))

	// عمولةُ المتجر ١٠٪ من ١٠٬٠٠٠ = ١٬٠٠٠ ⇒ نصيبُ المندوب ١٠٪ = ١٠٠ — **ولا يمسّها العرض.**
	if got := repCommissionOf(t, h, oid); got != 100 {
		t.Fatalf("عمولةُ المندوب %d — والمنتظَر 100 (العرضُ على المنصّة لا عليه)", got)
	}
}
