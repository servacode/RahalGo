package qa

// **بلاغُ المتجر لا يُسكت شكوى الزبون** — قرارُ المالك ٢٠٢٦-١٠-٠١ (هجرة ٠١٧١).
//
// **قِيس على التجهيز**: المتجرُ بلّغ عن تأخّر السائق، **فرُدّ الزبونُ حين اشتكى
// من الطعام على الطلب نفسِه بـ«لك شكوى مفتوحة» — ولم يفتح شيئاً.**
// **والتكرارُ من الشخص نفسِه يبقى ممنوعاً.**

import "testing"

func TestTKT01_MerchantReportDoesNotSilenceCustomer(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	drv := h.NewUser("driver")
	item := h.NewItem(5000)

	var ownerID string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT owner_user_id::text FROM merchants WHERE id = $1::uuid`, item.MerchantID).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	if oid == "" {
		t.Fatalf("الطلب: %s", made)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'delivered', delivered_at = now(), closed_at = now(), driver_id = $2::uuid
		WHERE id = $1::uuid`, oid, drv.ID); err != nil {
		t.Fatal(err)
	}

	mtok := h.TokenFor(ownerID, "merchant")
	if r := h.POST("/api/v1/merchant/orders/"+oid+"/report", mtok,
		map[string]any{"reason": "driver_late_pickup", "note": "تأخّر"}); r.Code != 201 {
		t.Fatalf("بلاغُ المتجر: %d %s", r.Code, r.Err())
	}
	c1 := h.POST("/api/v1/my/orders/"+oid+"/complaint", cust.Token, map[string]any{"reason": "quality", "note": "بارد"})
	if c1.Code != 201 {
		t.Fatalf("**بلاغُ المتجر أسكت شكوى الزبون**: %d / %s", c1.Code, c1.Err())
	}
	c2 := h.POST("/api/v1/my/orders/"+oid+"/complaint", cust.Token, map[string]any{"reason": "quality", "note": "مرّة ثانية"})
	if c2.Code != 409 || c2.Err() != "complaint_already_open" {
		t.Fatalf("**الزبونُ فتح شكوى ثانية على الطلب نفسِه**: %d / %s", c2.Code, c2.Err())
	}
	r2 := h.POST("/api/v1/merchant/orders/"+oid+"/report", mtok, map[string]any{"reason": "other", "note": "ثانٍ"})
	if r2.Code != 409 {
		t.Fatalf("**المتجرُ فتح بلاغاً ثانياً على الطلب نفسِه**: %d / %s", r2.Code, r2.Err())
	}
}
