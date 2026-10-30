package qa

import (
	"net/http"
	"testing"
)

// TestAmanahPickup_DriverSeesPoint **موقعُ الأمانة على الخريطة** (قرارُ المالك ٢٠٢٦-١٠-٣٠).
//
// الزبونُ يحدّد نقطةَ الاستلام في الأمانة فيراها السائقُ على خريطته (`pickup_lat`/`nav_lat`)
// ويراها الزبونُ في طلبه. وفي المشتريات تُهمَل،
// ونقطةٌ خارجَ التغطية تُردّ ولا يُنشأ طلب.
func TestAmanahPickup_DriverSeesPoint(t *testing.T) {
	h := New(t)
	const lat, lng = 35.9447, 39.0031
	const pLat, pLng = 35.9475, 39.0060
	cust := h.Customer()

	body := map[string]any{
		"request": "أمانة: كيس أوراق", "address_text": "الرقة — بناية الاختبار",
		"lat": lat, "lng": lng, "custom_mode": "amanah",
		"pickup_lat": pLat, "pickup_lng": pLng,
	}
	created := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amp"), body)
	if created.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الأمانة: %s", created)
	}
	id, _ := created.JSON()["id"].(string)
	if v, _ := created.JSON()["amanah_pickup_lat"].(float64); v < pLat-1e-6 || v > pLat+1e-6 {
		t.Fatalf("ردُّ الإنشاء بلا موقع الأمانة: %s", created)
	}
	if v, _ := h.GET("/api/v1/my/orders/"+id, cust.Token).JSON()["amanah_pickup_lng"].(float64); v < pLng-1e-6 || v > pLng+1e-6 {
		t.Fatalf("الزبونُ لا يرى موقع الأمانة في طلبه (%v)", v)
	}

	drv := h.NewUser("driver")
	assignDriver(t, h, id, drv.ID)
	list := h.GET("/api/v1/driver/orders", drv.Token)
	if list.Code != http.StatusOK {
		t.Fatalf("قائمةُ السائق: %s", list)
	}
	var found map[string]any
	for _, x := range mdList(list) {
		if m, _ := x.(map[string]any); m != nil && m["id"] == id {
			found = m
		}
	}
	if found == nil {
		t.Fatalf("الطلبُ ليس في قائمة السائق: %s", list)
	}
	for _, k := range []string{"pickup_lat", "nav_lat"} {
		if v, _ := found[k].(float64); v < pLat-1e-6 || v > pLat+1e-6 {
			t.Fatalf("السائقُ لا يرى موقع الأمانة في %s: %v", k, found[k])
		}
	}
	if found["pickup_address"] != "موقع الأمانة" {
		t.Fatalf("عنوانُ الاستلام للسائق %v — يُنتظر «موقع الأمانة»", found["pickup_address"])
	}

	// **والمشترياتُ تُهمله** — السائقُ يختار من أين يشتري.
	body["custom_mode"] = "purchase"
	body["request"] = "مشتريات: خبز"
	pur := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amp-p"), body)
	if pur.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ المشتريات: %s", pur)
	}
	pid, _ := pur.JSON()["id"].(string)
	var has bool
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT pickup_override IS NOT NULL FROM orders WHERE id = $1::uuid`, pid).Scan(&has); err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("**المشترياتُ كتبت نقطةَ استلام** — يجب أن تُهمَل")
	}

	// **ونقطةٌ خارجَ التغطية تُردّ** ولا يُنشأ شيء.
	body["custom_mode"] = "amanah"
	body["request"] = "أمانة من بعيد"
	body["pickup_lat"], body["pickup_lng"] = 48.85, 2.35
	if far := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amp-f"), body); far.Code < 400 {
		t.Fatalf("موقعُ أمانةٍ خارجَ التغطية قُبل: %s", far)
	}
}
