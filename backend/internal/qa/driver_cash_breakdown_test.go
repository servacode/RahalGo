package qa

// تفصيل التحصيل بصندوق السائق نفسه (٢٠٢٦-١٠-٢٧)
//
// تطبيق السائق يقرأ merchant_name وfee_part وcommission_part من /driver/cash،
// وكان الخادم يرسلها بكشف الإدارة وحده — فلا يظهر للسائق شيء.

import "testing"

func TestDriverCash_CollectionShowsBreakdown(t *testing.T) {
	h := New(t)
	h.oldScaleMoney()
	treasury(t, h)
	waiverSettings(h)

	oid, drv, _, m := waiverOrder(t, h, "cash")
	var number int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT number FROM orders WHERE id = $1`, oid).Scan(&number); err != nil {
		t.Fatal(err)
	}

	r := h.GET("/api/v1/driver/cash", drv.Token)
	if r.Code != 200 {
		t.Fatalf("صندوق السائق: %s", r)
	}
	entries, _ := r.JSON()["entries"].([]any)
	var row map[string]any
	for _, e := range entries {
		x, _ := e.(map[string]any)
		if x["kind"] == "order_collection" && num(x["order_number"]) == number {
			row = x
		}
	}
	if row == nil {
		t.Fatalf("لا سطر تحصيل للطلب: %s", r)
	}
	if row["merchant_name"] != m.Name || num(row["fee_part"]) != 500 || num(row["commission_part"]) != 100 ||
		num(row["fee_part"])+num(row["commission_part"]) != num(row["amount"]) {
		t.Errorf("تفصيل التحصيل ناقص بصندوق السائق: %v", row)
	}
}
