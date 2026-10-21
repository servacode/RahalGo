package qa

import (
	"strings"
	"testing"
)

// mcDeliveredOrder **زبونٌ بعنوانٍ محفوظ، وطلبٌ إليه سلّمه سائق.**
func mcDeliveredOrder(t *testing.T, h *Harness, lat, lng float64) (cust *User, drv *User, orderID string) {
	t.Helper()
	cust = h.Customer()
	if res := h.POST("/api/v1/my/addresses", cust.Token, map[string]any{
		"area_building": "الرقة — بناية الاختبار", "kind": "home", "lat": lat, "lng": lng,
	}); res.Code >= 300 {
		t.Fatalf("العنوان: %s", res)
	}
	created := h.POST("/api/v1/orders/custom", cust.Token, map[string]any{
		"request": "طلب اختبار الباب", "address_text": "الرقة — بناية الاختبار", "lat": lat, "lng": lng,
	})
	if created.Code != 201 {
		t.Fatalf("الطلب: %s", created)
	}
	orderID, _ = created.JSON()["id"].(string)
	drv = h.NewUser("driver")
	assignDriver(t, h, orderID, drv.ID)
	return
}

// TestEntrance_SavedOnAddressAndSuggestedToCustomer **بابُ البناية** (قرارُ المالك ٢٠٢٦-١٠-٢١):
// يُحفظ مع عنوان الزبون فيصل السائقَ في الطلب التالي، **وإن بعُد عن النقطة اقتُرح على
// الزبون تعديلُها** — ولا تتغيّر إلّا بموافقته.
func TestEntrance_SavedOnAddressAndSuggestedToCustomer(t *testing.T) {
	h := New(t)
	const lat, lng = 35.9447, 39.0031
	cust, drv, id := mcDeliveredOrder(t, h, lat, lng)
	door := map[string]any{"lat": lat + 0.0011, "lng": lng} // ~١٢٠ م
	if res := h.POST("/api/v1/driver/orders/"+id+"/entrance", drv.Token, door); res.Code != 409 {
		t.Fatalf("قبل التسليم يجب أن يُردّ: %s", res)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET status = 'delivered' WHERE id = $1::uuid`, id); err != nil {
		t.Fatal(err)
	}
	if res := h.POST("/api/v1/driver/orders/"+id+"/entrance", drv.Token, map[string]any{"lat": lat + 0.002, "lng": lng}); res.Code != 422 {
		t.Fatalf("بابٌ على ٢٢٠ م يجب أن يُردّ: %s", res)
	}
	res := h.POST("/api/v1/driver/orders/"+id+"/entrance", drv.Token, door)
	if res.Code != 200 || res.JSON()["saved"] != true || res.JSON()["suggested"] != true {
		t.Fatalf("حفظُ الباب: %s", res)
	}
	mine := h.GET("/api/v1/my/orders/"+id, cust.Token)
	if v, _ := mine.JSON()["dropoff_suggest_lat"].(float64); v == 0 {
		t.Fatalf("الزبونُ لا يرى الاقتراح: %s", mine)
	}
	if strings.Contains(string(mine.Body), "entrance_lat") {
		t.Fatalf("**الزبونُ يرى بابَ السائقين**: %s", mine)
	}

	// **والطلبُ التالي إلى العنوان نفسه يحمل البابَ للسائق.**
	next := h.POST("/api/v1/orders/custom", cust.Token, map[string]any{
		"request": "طلبٌ ثانٍ", "address_text": "الرقة — بناية الاختبار", "lat": lat, "lng": lng,
	})
	nextID, _ := next.JSON()["id"].(string)
	drv2 := h.NewUser("driver")
	assignDriver(t, h, nextID, drv2.ID)
	if list := h.GET("/api/v1/driver/orders", drv2.Token); !strings.Contains(string(list.Body), `"entrance_lat":35.945`) {
		t.Fatalf("السائقُ التالي لا يرى الباب: %s", list)
	}
	// **والطريقُ يُطلب إلى الباب** — والقراءةُ لا تسقط بالضمّ الجديد.
	if rt := h.GET("/api/v1/driver/orders/"+nextID+"/route", drv2.Token); rt.Code != 200 {
		t.Fatalf("طريقُ الطلب التالي: %s", rt)
	}

	// **والموافقةُ تنقل نقطةَ العنوان إلى الباب.**
	if ok := h.POST("/api/v1/orders/"+id+"/accept-dropoff-fix", cust.Token, map[string]any{}); ok.Code != 200 {
		t.Fatalf("الموافقة: %s", ok)
	}
	var moved float64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT ST_Distance(location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography)
		FROM user_addresses WHERE user_id = $1::uuid`, cust.ID, lng, lat+0.0011).Scan(&moved); err != nil {
		t.Fatal(err)
	}
	if moved > 1 {
		t.Fatalf("نقطةُ العنوان لم تنتقل إلى الباب (بُعد %.0f م)", moved)
	}
}

// TestRoadClosure_ReportConfirmAndClear **طريقٌ مسكّر** — يُبلَّغ، ويؤكّده ثانٍ، وتزيله الإدارة.
func TestRoadClosure_ReportConfirmAndClear(t *testing.T) {
	h := New(t)
	a, b := h.NewUser("driver"), h.NewUser("driver")
	p := map[string]any{"lat": 35.9601, "lng": 39.0102, "reason": "works"}
	if bad := h.POST("/api/v1/driver/road-closures", a.Token, map[string]any{"lat": 35.96, "lng": 39.01, "reason": "x"}); bad.Code != 422 {
		t.Fatalf("سببٌ غيرُ معروف: %s", bad)
	}
	first := h.POST("/api/v1/driver/road-closures", a.Token, p)
	if first.Code != 200 || first.JSON()["confirmed"] != false {
		t.Fatalf("البلاغُ الأوّل: %s", first)
	}
	second := h.POST("/api/v1/driver/road-closures", b.Token, map[string]any{"lat": 35.96012, "lng": 39.01021, "reason": "barrier"})
	if second.Code != 200 || second.JSON()["confirmed"] != true {
		t.Fatalf("التأكيد: %s", second)
	}
	admin := h.NewUser("admin")
	list := h.GET("/api/v1/admin/ops-map/road-closures", admin.Token)
	if list.Code != 200 || !strings.Contains(string(list.Body), `"reports":2`) {
		t.Fatalf("قائمةُ المسكَّر: %s", list)
	}
	id, _ := first.JSON()["id"].(float64)
	if del := h.Call("DELETE", "/api/v1/admin/ops-map/road-closures/"+itoa(int(id)), admin.Token, nil, nil); del.Code != 200 {
		t.Fatalf("الإزالة: %s", del)
	}
}

// TestLandmark_SuggestApproveEntersNearLandmark **«أضف معلم»** — بانتظار الإدارة، ثمّ يدخل المعالم.
func TestLandmark_SuggestApproveEntersNearLandmark(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	name := uniq("صيدلية الاختبار ")
	res := h.POST("/api/v1/driver/landmarks", drv.Token, map[string]any{"name": name, "kind": "pharmacy", "lat": 35.9555, "lng": 39.0066})
	if res.Code != 200 {
		t.Fatalf("الاقتراح: %s", res)
	}
	id, _ := res.JSON()["id"].(float64)
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_landmarks WHERE name = $1`, name) })
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*)::int FROM map_landmarks WHERE name = $1`, name).Scan(&n)
	if n != 0 {
		t.Fatal("**المعلمُ دخل قبل موافقة الإدارة**")
	}
	admin := h.NewUser("admin")
	if list := h.GET("/api/v1/admin/ops-map/landmark-suggestions", admin.Token); !strings.Contains(string(list.Body), name) {
		t.Fatalf("الإدارةُ لا ترى الاقتراح: %s", list)
	}
	if d := h.POST("/api/v1/admin/ops-map/landmark-suggestions/"+itoa(int(id))+"/decide", admin.Token, map[string]any{"approve": true}); d.Code != 200 {
		t.Fatalf("الموافقة: %s", d)
	}
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*)::int FROM map_landmarks WHERE name = $1 AND kind = 'amenity=pharmacy'`, name).Scan(&n)
	if n != 1 {
		t.Fatalf("المعلمُ الموافَقُ لم يدخل المعالم (%d)", n)
	}
	if again := h.POST("/api/v1/admin/ops-map/landmark-suggestions/"+itoa(int(id))+"/decide", admin.Token, map[string]any{"approve": false}); again.Code != 409 {
		t.Fatalf("قرارٌ ثانٍ يجب أن يُردّ: %s", again)
	}
}

// TestMapStoresAndClientReport **متاجرُنا على الخريطة، والتطبيقُ يُبلغ قبل الخروج.**
func TestMapStoresAndClientReport(t *testing.T) {
	h := New(t)
	m := h.Factory().Merchant()
	h.NewItemFor(m, 100)
	if res := h.GET("/api/v1/public/map/stores", ""); res.Code != 200 || !strings.Contains(string(res.Body), m.ID) {
		t.Fatalf("المتجرُ لا يظهر على الخريطة: %s", res)
	}
	if gj := h.GET("/api/v1/public/map/stores?format=geojson", ""); gj.Code != 200 ||
		!strings.Contains(string(gj.Body), `"FeatureCollection"`) || !strings.Contains(string(gj.Body), m.ID) {
		t.Fatalf("صيغةُ الخريطة: %s", gj)
	}
	code := uniq("qa_client_")
	if res := h.Call("POST", "/api/v1/auth/client-report", "", map[string]any{"code": code, "path": "/api/v1/x"},
		map[string]string{"X-RahalGo-Client": "android-customer", "X-RahalGo-Version": "30"}); res.Code != 200 {
		t.Fatalf("البلاغ: %s", res)
	}
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*)::int FROM auth_failures WHERE kind = 'client' AND code = $1
		AND client = 'android-customer' AND app_version = '30'`, code).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("بلاغُ التطبيق لم يُسجَّل (%d)", n)
	}
}

// TestDelivery_FarFromPinAsksCustomer **«هون عنوانك الصح؟» من مكان التسليم** (قرارُ المالك
// ٢٠٢٦-١٠-٢١) — شُهد طلبٌ سُلّم على ٦٣٣ م من دبّوسه فلم يُسأل أحد. **ومكانُ التسليم موقعُ
// السائق لحظتَه**: هناك اجتمعا.
func TestDelivery_FarFromPinAsksCustomer(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	var lat, lng float64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT ST_Y(dropoff::geometry), ST_X(dropoff::geometry)
		FROM orders WHERE id = $1::uuid`, oid).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if res := h.POST("/api/v1/my/addresses", cust.Token, map[string]any{
		"area_building": "الرقة — بيت الاختبار", "kind": "home", "lat": lat, "lng": lng,
	}); res.Code >= 300 {
		t.Fatalf("العنوان: %s", res)
	}
	drv := h.driverOf(oid)
	// **السائقُ سلّم على بُعد ~٣٠٠ م من الدبّوس.**
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO driver_track (driver_id, at, recorded_at, accuracy_m)
		VALUES ($1::uuid, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, now(), 10)`,
		drv.ID, lng, lat+0.0027); err != nil {
		t.Fatal(err)
	}
	deliverOrder(t, h, oid, drv)

	mine := h.GET("/api/v1/my/orders/"+oid, cust.Token)
	got, _ := mine.JSON()["dropoff_suggest_lat"].(float64)
	if got < lat+0.0026 || got > lat+0.0028 {
		t.Fatalf("الزبونُ لم يُسأل عن مكان التسليم (%.5f): %s", got, mine)
	}
}
