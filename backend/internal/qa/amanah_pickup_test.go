package qa

import (
	"net/http"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **الأمانة: صاحبُها ورقمُه، وموقعُه يطلبه الزبونُ منه** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
// ══════════════════════════════════════════════════════════════════════
//
// **الزبونُ لا يعرف أين الأمانة** — يكتب اسمَ صاحبها ورقمَه، ويضغط «اطلب الموقع منه»
// فيصل صاحبَها رابطُ `/loc/<رمز>` كـ«إحضار» المتجر. **ونقطتُه تصير نقطةَ الاستلام**
// قبل الإنشاء أو بعده، والسائقُ يراها على خريطته ويرى الاسمَ والرقم.

const (
	amLat, amLng = 35.9447, 39.0031
	amPLat       = 35.9475
	amPLng       = 39.0060
)

func amanahBody(req string) map[string]any {
	return map[string]any{
		"request": req, "address_text": "الرقة — بناية الاختبار",
		"lat": amLat, "lng": amLng, "custom_mode": "amanah",
		"holder_name": "أبو علي", "holder_phone": "0933 123 456",
	}
}

func near(v any, want float64) bool {
	f, ok := v.(float64)
	return ok && f > want-1e-6 && f < want+1e-6
}

// driverOrderOf **طلبٌ بعينه من قائمة السائق.**
func driverOrderOf(t *testing.T, h *Harness, token, id string) map[string]any {
	t.Helper()
	list := h.GET("/api/v1/driver/orders", token)
	if list.Code != http.StatusOK {
		t.Fatalf("قائمةُ السائق: %s", list)
	}
	for _, x := range mdList(list) {
		if m, _ := x.(map[string]any); m != nil && m["id"] == id {
			return m
		}
	}
	t.Fatalf("الطلبُ ليس في قائمة السائق: %s", list)
	return nil
}

// TestAmanahHolder_RequiredAndVisible **الاسمُ والرقمُ إلزاميّان في الأمانة**، ويراهما الزبونُ
// والسائقُ المُسنَد. **والمشترياتُ لا تتأثّر.**
func TestAmanahHolder_RequiredAndVisible(t *testing.T) {
	h := New(t)
	cust := h.Customer()

	// **بلا صاحبٍ تُردّ** — «ناقص».
	b := amanahBody("أمانة بلا صاحب")
	delete(b, "holder_name")
	if r := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amh-n"), b); r.Code != http.StatusUnprocessableEntity || r.Err() != "amanah_holder_required" {
		t.Fatalf("أمانةٌ بلا اسم صاحبها قُبلت: %d %s", r.Code, r.Err())
	}
	b = amanahBody("أمانة برقم خاطئ")
	b["holder_phone"] = "12"
	if r := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amh-p"), b); r.Code < 400 {
		t.Fatalf("رقمٌ خاطئٌ قُبل: %s", r)
	}

	// **والمشترياتُ بلا صاحبٍ تُقبل** — لا شأنَ لها به.
	pb := amanahBody("مشتريات: خبز")
	pb["custom_mode"] = "purchase"
	delete(pb, "holder_name")
	delete(pb, "holder_phone")
	pur := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amh-pur"), pb)
	if pur.Code != http.StatusCreated {
		t.Fatalf("المشترياتُ رُدّت: %s", pur)
	}
	if _, has := pur.JSON()["holder_name"]; has {
		t.Fatalf("المشترياتُ تحمل صاحبَ أمانة: %s", pur)
	}

	created := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amh-ok"), amanahBody("أمانة: كيس أوراق"))
	if created.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الأمانة: %s", created)
	}
	id, _ := created.JSON()["id"].(string)
	mine := h.GET("/api/v1/my/orders/"+id, cust.Token).JSON()
	if mine["holder_name"] != "أبو علي" || mine["holder_phone"] != "+963933123456" {
		t.Fatalf("الزبونُ لا يرى صاحبَ الأمانة كما كتبه: %v / %v", mine["holder_name"], mine["holder_phone"])
	}

	drv := h.NewUser("driver")
	assignDriver(t, h, id, drv.ID)
	got := driverOrderOf(t, h, drv.Token, id)
	if got["holder_name"] != "أبو علي" || got["holder_phone"] != "+963933123456" {
		t.Fatalf("السائقُ لا يرى صاحبَ الأمانة: %v / %v", got["holder_name"], got["holder_phone"])
	}
	// **وبلا موقعٍ لا نقطةَ استلام** — يتّصل به.
	if got["pickup_lat"] != nil {
		t.Fatalf("نقطةُ استلامٍ بلا موقع: %v", got["pickup_lat"])
	}
}

// requestHolderLocation **«اطلب الموقع منه»** — يردّ رقمَ الطلب ورمزَ الرابط.
func requestHolderLocation(t *testing.T, h *Harness, token, phone string) (string, string) {
	t.Helper()
	r := h.POST("/api/v1/my/location-requests", token, map[string]any{"phone": phone})
	if r.Code != http.StatusCreated {
		t.Fatalf("طلبُ الموقع: %s", r)
	}
	id, _ := r.JSON()["id"].(string)
	link, _ := r.JSON()["link"].(string)
	i := strings.LastIndex(link, "/loc/")
	if id == "" || i < 0 {
		t.Fatalf("ردُّ طلب الموقع بلا رقمٍ أو رابط: %s", r)
	}
	return id, link[i+len("/loc/"):]
}

func shareLocation(t *testing.T, h *Harness, tok string) {
	t.Helper()
	pub := h.GET("/api/v1/public/location-requests/"+tok, "")
	if pub.Code != http.StatusOK || pub.JSON()["mode"] != "amanah" {
		t.Fatalf("صفحةُ الرابط: %s", pub)
	}
	if r := h.POST("/api/v1/public/location-requests/"+tok, "", map[string]any{
		"lat": amPLat, "lng": amPLng, "accuracy_m": 8,
	}); r.Code != http.StatusOK {
		t.Fatalf("إرسالُ الموقع: %s", r)
	}
}

// TestAmanahHolder_SharedBeforeSend **شارك قبل الإرسال** — النقطةُ نقطةُ الاستلام عند الإنشاء.
func TestAmanahHolder_SharedBeforeSend(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	reqID, tok := requestHolderLocation(t, h, cust.Token, "0933123456")
	// **والضغطةُ الثانيةُ للرقم نفسِه في دقيقةٍ تُردّ** — كحدّ المتجر.
	if r := h.POST("/api/v1/my/location-requests", cust.Token, map[string]any{"phone": "0933123456"}); r.Code != http.StatusTooManyRequests {
		t.Fatalf("الحدُّ لم يعمل: %s", r)
	}
	// **ولا يقرأ غيرُه طلبَه.**
	other := h.Customer()
	if r := h.GET("/api/v1/my/location-requests/"+reqID, other.Token); r.Code != http.StatusNotFound {
		t.Fatalf("زبونٌ آخرُ قرأ طلبَ الموقع: %s", r)
	}
	shareLocation(t, h, tok)
	st := h.GET("/api/v1/my/location-requests/"+reqID, cust.Token).JSON()
	if st["status"] != "received" || !near(st["lat"], amPLat) {
		t.Fatalf("حالُ الطلب بعد المشاركة: %v", st)
	}

	b := amanahBody("أمانة: مفتاح")
	b["location_request_id"] = reqID
	created := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amh-sb"), b)
	if created.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الأمانة: %s", created)
	}
	id, _ := created.JSON()["id"].(string)
	if !near(created.JSON()["amanah_pickup_lat"], amPLat) {
		t.Fatalf("ردُّ الإنشاء بلا موقع الأمانة: %s", created)
	}
	drv := h.NewUser("driver")
	assignDriver(t, h, id, drv.ID)
	got := driverOrderOf(t, h, drv.Token, id)
	for _, k := range []string{"pickup_lat", "nav_lat"} {
		if !near(got[k], amPLat) {
			t.Fatalf("السائقُ لا يرى موقع الأمانة في %s: %v", k, got[k])
		}
	}
	if got["pickup_address"] != "موقع الأمانة" {
		t.Fatalf("عنوانُ الاستلام للسائق %v — يُنتظر «موقع الأمانة»", got["pickup_address"])
	}

	// **وطلبُ الموقع لطلبٍ واحد** — لا يُربط بثانٍ.
	b2 := amanahBody("أمانة ثانية")
	b2["location_request_id"] = reqID
	if r := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amh-sb2"), b2); r.Code != http.StatusUnprocessableEntity || r.Err() != "location_request_invalid" {
		t.Fatalf("طلبُ موقعٍ رُبط بطلبين: %d %s", r.Code, r.Err())
	}
}

// TestAmanahHolder_SharedAfterSend **شارك بعد الإرسال** — النقطةُ تُكتب في الطلب حين تصل.
func TestAmanahHolder_SharedAfterSend(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	reqID, tok := requestHolderLocation(t, h, cust.Token, "0944555666")
	b := amanahBody("أمانة: أوراق")
	b["holder_phone"] = "0944555666"
	b["location_request_id"] = reqID
	created := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amh-sa"), b)
	if created.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الأمانة: %s", created)
	}
	id, _ := created.JSON()["id"].(string)
	drv := h.NewUser("driver")
	assignDriver(t, h, id, drv.ID)
	if got := driverOrderOf(t, h, drv.Token, id); got["pickup_lat"] != nil {
		t.Fatalf("نقطةُ استلامٍ قبل المشاركة: %v", got["pickup_lat"])
	}

	shareLocation(t, h, tok)
	got := driverOrderOf(t, h, drv.Token, id)
	if !near(got["pickup_lat"], amPLat) || !near(got["nav_lng"], amPLng) {
		t.Fatalf("الموقعُ المُشارَكُ بعد الإرسال لم يصر نقطةَ الاستلام: %v / %v", got["pickup_lat"], got["nav_lng"])
	}
	if got["pickup_address"] != "موقع الأمانة" {
		t.Fatalf("عنوانُ الاستلام للسائق %v — يُنتظر «موقع الأمانة»", got["pickup_address"])
	}
}

// TestAmanahPickup_ExplicitPointStillWorks **النقطةُ الصريحةُ باقيةٌ في الخادم** (للتطبيق القديم)،
// والمشترياتُ تُهملها، **ونقطةٌ خارجَ التغطية تُردّ.**
func TestAmanahPickup_ExplicitPointStillWorks(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	body := amanahBody("أمانة: كيس")
	body["pickup_lat"], body["pickup_lng"] = amPLat, amPLng
	created := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amp"), body)
	if created.Code != http.StatusCreated || !near(created.JSON()["amanah_pickup_lat"], amPLat) {
		t.Fatalf("إنشاءُ الأمانة بنقطة: %s", created)
	}

	body["custom_mode"] = "purchase"
	body["request"] = "مشتريات: خبز"
	pur := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amp-p"), body)
	if pur.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ المشتريات: %s", pur)
	}
	pid, _ := pur.JSON()["id"].(string)
	var has bool
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT pickup_override IS NOT NULL OR amanah_holder_name IS NOT NULL FROM orders WHERE id = $1::uuid`, pid).Scan(&has); err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("**المشترياتُ كتبت نقطةَ استلامٍ أو صاحبَ أمانة** — يجب أن تُهمَل")
	}

	body["custom_mode"] = "amanah"
	body["request"] = "أمانة من بعيد"
	body["pickup_lat"], body["pickup_lng"] = 48.85, 2.35
	if far := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amp-f"), body); far.Code < 400 {
		t.Fatalf("موقعُ أمانةٍ خارجَ التغطية قُبل: %s", far)
	}
}

// TestAmanahHolder_OldAppWithoutHolder **التطبيقُ القديمُ لا يرسل صاحبَ الأمانة فيُقبل** — والإلزامُ
// في التطبيق الجديد. ومن أرسل أحدَهما بلا الآخر يُردّ (اختبارُ «المطلوب» أعلاه).
func TestAmanahHolder_OldAppWithoutHolder(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	b := amanahBody("أمانة من تطبيقٍ قديم")
	delete(b, "holder_name")
	delete(b, "holder_phone")
	if r := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("amh-old"), b); r.Code != http.StatusCreated {
		t.Fatalf("**أمانةُ التطبيق القديم رُدّت**: %d %s", r.Code, r.Err())
	}
}
