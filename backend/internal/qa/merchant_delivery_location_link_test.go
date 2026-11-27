package qa

// ══════════════════════════════════════════════════════════════════════
// **«لدي توصيلة» أُرسلت قبل أن يصل موقعُ المستلِم** (بلاغُ المالك ٢٠٢٦-١١-٢٧)
// ══════════════════════════════════════════════════════════════════════
//
// «المتجر بعت توصيلة، والمستلم بعت موقعه، بس ما وصل للكابتن.» — المتجرُ طلب الموقعَ ثمّ أرسل
// التوصيلةَ قبل الردّ، **ولا شيءَ كان يربط طلبَ الموقع بالتوصيلة.** والآن يُربط عند الإنشاء،
// **وحين يصل الموقعُ (ردُّ الواتساب أو صفحةُ الرابط) يُكتب ويراه الكابتن، والأجرةُ كما هي.**

import (
	"net/http"
	"strings"
	"testing"
)

// mdLocReq **«اطلب الموقع من المستلم» من نموذج المتجر** — يردّ رقمَ الطلب ورمزَ رابطه.
func mdLocReq(t *testing.T, h *Harness, m mdFx, phone, mode string) (string, string) {
	t.Helper()
	r := h.POST(m.url("/location-requests"), m.fx.Tok, map[string]any{"phone": phone, "mode": mode})
	if r.Code != http.StatusCreated {
		t.Fatalf("«اطلب الموقع» من المتجر: %d / %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	link, _ := r.JSON()["link"].(string)
	i := strings.LastIndex(link, "/loc/")
	if id == "" || i < 0 {
		t.Fatalf("ردُّ طلب الموقع بلا رقمٍ أو رابط: %s", r)
	}
	return id, link[i+len("/loc/"):]
}

// mdCreateNoPoint **توصيلةٌ بلا نقطة لرقمٍ معيّن** — كما يرسلها النموذجُ قبل وصول الموقع.
func mdCreateNoPoint(t *testing.T, h *Harness, m mdFx, phone, mode string, extra map[string]any) string {
	t.Helper()
	b := m.body("recipient")
	delete(b, "lat")
	delete(b, "lng")
	b["recipient_phone"] = phone
	if mode != "" {
		b["delivery_mode"] = mode
	}
	for k, v := range extra {
		b[k] = v
	}
	r := h.POSTKey(m.url("/deliveries"), m.fx.Tok, uniq("mdloc"), b)
	if r.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %d / %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	return id
}

// mdPoint **نقطةُ التوصيلة من الصفّ** — معروفة؟ وأين؟ وبكم أجرتُها.
func mdPoint(t *testing.T, h *Harness, id string) (known bool, lat, lng float64, fee int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT dropoff_known, ST_Y(dropoff::geometry), ST_X(dropoff::geometry), delivery_fee
		  FROM orders WHERE id = $1::uuid`, id).Scan(&known, &lat, &lng, &fee); err != nil {
		t.Fatal(err)
	}
	return
}

// TestMDLOC01_SendLocationArrivesByWhatsAppAfterSubmit **«توصيل»: المستلِمُ ردّ بالواتساب بعد الإرسال**
// — والكابتنُ المُسنَد يراه على الخريطة ويُشعَر، والأجرةُ لم تتغيّر.
func TestMDLOC01_SendLocationArrivesByWhatsAppAfterSubmit(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDLOC-01")
	phone := uniqPhone()
	mdLocReq(t, h, m, phone, "send")
	id := mdCreateNoPoint(t, h, m, phone, "", nil)
	known, _, _, fee := mdPoint(t, h, id)
	if known {
		t.Fatal("توصيلةٌ بلا نقطةٍ وُلدت بنقطةٍ معروفة")
	}
	m.accept(t, h, id)
	drv := h.driverOf(id)
	if got := driverOrderOf(t, h, drv.Token, id); got["dropoff_known"] != false {
		t.Fatalf("**قبل الموقع النقطةُ معروفة عند الكابتن**: %v", got["dropoff_known"])
	}
	room := h.Listen("driver:" + drv.ID)
	if r := h.API.HandleWALocation(ctxBG(), phone, m.dropLat, m.dropLng, 8); r == "" {
		t.Fatal("**لا ردَّ على موقع المستلِم**")
	}
	if !waDriverSignalled(room) {
		t.Fatal("**غرفةُ الكابتن لم تصلها إشارةٌ بعد وصول الموقع**")
	}
	known2, lat, lng, fee2 := mdPoint(t, h, id)
	if !known2 || !nearF(lat, m.dropLat) || !nearF(lng, m.dropLng) {
		t.Fatalf("**الموقعُ الواصلُ لم يُكتب في التوصيلة** (بلاغُ المالك): %v %v,%v", known2, lat, lng)
	}
	if fee2 != fee {
		t.Fatalf("**الأجرةُ تغيّرت بصمت**: %d ثمّ %d", fee, fee2)
	}
	got := driverOrderOf(t, h, drv.Token, id)
	if got["dropoff_known"] != true || !near(got["lat"], m.dropLat) || !near(got["lng"], m.dropLng) {
		t.Fatalf("**الكابتنُ لا يرى موقعَ المستلِم**: %v · %v,%v", got["dropoff_known"], got["lat"], got["lng"])
	}
	if n := waNotes(t, h, drv.ID, id, "وصل موقع المستلم — صار على الخريطة"); len(n) != 1 {
		t.Fatalf("**إشعارُ الكابتن**: %v", n)
	}
}

// TestMDLOC02_FetchLocationArrivesByLinkAfterSubmit **«إحضار»: الموقعُ من صفحة الرابط بعد الإرسال**
// — ومعرّفُ الطلب صريحٌ في الإنشاء. يصير مكانَ الاستلام عند الكابتن، والأجرةُ كما هي.
func TestMDLOC02_FetchLocationArrivesByLinkAfterSubmit(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDLOC-02")
	phone := uniqPhone()
	reqID, tok := mdLocReq(t, h, m, phone, "fetch")
	id := mdCreateNoPoint(t, h, m, phone, "fetch", map[string]any{"location_request_id": reqID})
	_, _, _, fee := mdPoint(t, h, id)
	m.accept(t, h, id)
	drv := h.driverOf(id)
	if r := h.POST("/api/v1/public/location-requests/"+tok, "", map[string]any{
		"lat": m.dropLat, "lng": m.dropLng, "accuracy_m": 9,
	}); r.Code != http.StatusOK {
		t.Fatalf("إرسالُ الموقع من الصفحة: %s", r)
	}
	known, lat, _, fee2 := mdPoint(t, h, id)
	if !known || !nearF(lat, m.dropLat) || fee2 != fee {
		t.Fatalf("**الإحضارُ لم يأخذ الموقعَ الواصل أو تغيّرت الأجرة**: %v %v · %d ثمّ %d", known, lat, fee, fee2)
	}
	got := driverOrderOf(t, h, drv.Token, id)
	if !near(got["nav_lat"], m.dropLat) || !near(got["nav_lng"], m.dropLng) {
		t.Fatalf("**الاستلامُ عند الكابتن ليس من الموقع الواصل**: %v,%v", got["nav_lat"], got["nav_lng"])
	}
	if n := waNotes(t, h, drv.ID, id, "وصل موقع الاستلام — صار على الخريطة"); len(n) != 1 {
		t.Fatalf("**إشعارُ الكابتن في الإحضار**: %v", n)
	}
}

// TestMDLOC03_LocationArrivedBeforeSubmit **وصل الموقعُ قبل الإرسال ولم يملأ النموذج** — يُكتب عند
// الإنشاء، والأجرةُ أجرةُ ما رآه المتجر. **وطلبُ موقعٍ رُبط بتوصيلةٍ لا يُعاد لغيرها.**
func TestMDLOC03_LocationArrivedBeforeSubmit(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDLOC-03")
	phone := uniqPhone()
	_, tok := mdLocReq(t, h, m, phone, "send")
	if r := h.POST("/api/v1/public/location-requests/"+tok, "", map[string]any{
		"lat": m.dropLat, "lng": m.dropLng,
	}); r.Code != http.StatusOK {
		t.Fatalf("إرسالُ الموقع: %s", r)
	}
	id := mdCreateNoPoint(t, h, m, phone, "", nil)
	known, lat, lng, fee := mdPoint(t, h, id)
	if !known || !nearF(lat, m.dropLat) || !nearF(lng, m.dropLng) {
		t.Fatalf("**الموقعُ الواصلُ قبل الإرسال لم يُكتب**: %v %v,%v", known, lat, lng)
	}
	if fee != 500 {
		t.Fatalf("الأجرة %d والمنتظَر 500 (أجرةُ النموذج)", fee)
	}
	var linked string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(order_id::text, '') FROM recipient_location_requests
		 WHERE merchant_id = $1::uuid AND phone = $2`, m.fx.M.ID, phone).Scan(&linked); err != nil {
		t.Fatal(err)
	}
	if linked != id {
		t.Fatalf("طلبُ الموقع لم يُربط بالتوصيلة: %q", linked)
	}
	// **توصيلةٌ ثانيةٌ للرقم نفسِه لا تأخذ موقعاً رُبط بالأولى.**
	id2 := mdCreateNoPoint(t, h, m, phone, "", nil)
	if k, _, _, _ := mdPoint(t, h, id2); k {
		t.Fatal("**طلبُ موقعٍ مربوطٌ أُعيد لتوصيلةٍ ثانية**")
	}
}
