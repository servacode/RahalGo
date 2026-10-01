package qa

// **بطاقةُ الطلب عند المتجر تعرض سعرَه وعمولتَه ومستحقَّه** — فحصُ المتجر ٢٠٢٦-١٠-٠١.
//
// **رُئي على الجهاز**: «خصم المنصة ٠٪ · المستحق لك ٠» على طلبٍ بـ٤٥٬٠٠٠ — عمودُ
// المتجر فارغٌ («اتبع العامّ») فكانت الدالّةُ تسقط وتترك الأصفار.
// (قرارُ المالك: «سعرُه ونسبةُ العمولة ومستحقُّه».)

import "testing"

func TestMCM01_CardShowsMerchantMoneyWithoutOverride(t *testing.T) {
	h := New(t)
	h.Setting("merchants.commission_percent", "10")
	item := h.NewItem(5000)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE merchants SET commission_percent = NULL WHERE id = $1`, item.MerchantID); err != nil {
		t.Fatal(err)
	}
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("mcm"), orderBody(item, 2))
	oid, _ := made.JSON()["id"].(string)
	if oid == "" {
		t.Fatalf("الطلب: %s", made)
	}
	var owner string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT owner_user_id::text FROM merchants WHERE id = $1`, item.MerchantID).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	r := h.GET("/api/v1/merchant/orders/"+oid, h.TokenFor(owner, "merchant"))
	if r.Code != 200 {
		t.Fatalf("الطلبُ عند المتجر: %d / %s", r.Code, r.Err())
	}
	j := r.JSON()
	pct, _ := j["commission_percent"].(float64)
	cut, _ := j["platform_commission"].(float64)
	net, _ := j["merchant_net"].(float64)
	sub, _ := j["subtotal"].(float64)
	if pct != 10 || cut != 1000 || net != 9000 || sub != 10000 {
		t.Fatalf("**البطاقةُ تعرض** نسبة %v · عمولة %v · صافي %v · مجموع %v — **والمنتظَرُ ١٠٪ · ١٬٠٠٠ · ٩٬٠٠٠ · ١٠٬٠٠٠**",
			pct, cut, net, sub)
	}
}
