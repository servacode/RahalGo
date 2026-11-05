package qa

// ══════════════════════════════════════════════════════════════════════
// **طلباتُ الواتساب — ملحقُ المالك** (٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
//	أ · «اطلب الموقع» من النموذج — الصفُّ نفسُه بنوع `whatsapp`، والردُّ والرابطُ يملآنه
//	ب · الطلبُ لا ينتظر الموقع — يُنشأ بالعنوان، والموقعُ يُكتب نقطةَ تسليمٍ حين يصل
//	ج · رسالةُ طلب الموقع واحدةٌ لكلّ تدفّق — نصُّ المالك حرفاً
//	د · «اكتبلنا عنوانك» مرّةً بعد دقائق، والمكتبُ يصحّح العنوان من اللوحة
//	هـ · مفاتيحُ أجرة «طلب واتساب» وحصّتِه تُنسخ من «لدي توصيلة»
//	و · السائقُ لا يرى رقمَ مستلِم طلب الواتساب أبداً (البند ٧)
//
// **وكلُّها بالأبواب لا بدوالّ المحرّك** — فتُبنى على شيفرة المراحل ١–٥ وتسقط عليها.

import (
	"net/http"
	"strings"
	"testing"
)

// waLocReq **«اطلب الموقع» من نموذج الواتساب** — يردّ رقمَ الطلب ورمزَ الرابط وكاملَ الردّ.
func waLocReq(t *testing.T, h *Harness, tok, phone, store string) (string, string, map[string]any) {
	t.Helper()
	r := h.POST(waURL+"/location-requests", tok, map[string]any{"phone": phone, "store_name": store})
	if r.Code != http.StatusCreated {
		t.Fatalf("**«اطلب الموقع» من النموذج رُدّ**: %d / %s", r.Code, r.Err())
	}
	j := r.JSON()
	id, _ := j["id"].(string)
	link, _ := j["link"].(string)
	i := strings.LastIndex(link, "/loc/")
	if id == "" || i < 0 {
		t.Fatalf("ردُّ طلب الموقع بلا رقمٍ أو رابط: %s", r)
	}
	return id, link[i+len("/loc/"):], j
}

func waLocPoll(t *testing.T, h *Harness, tok, id string) map[string]any {
	t.Helper()
	r := h.GET(waURL+"/location-requests/"+id, tok)
	if r.Code != http.StatusOK {
		t.Fatalf("حالُ طلب الموقع: %d / %s", r.Code, r.Err())
	}
	return r.JSON()
}

// waOrderPoint **نقطةُ التسليم وأمعروفةٌ هي، والأجرة.**
func waOrderPoint(t *testing.T, h *Harness, id string) (lat, lng float64, known bool, fee int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT ST_Y(dropoff::geometry), ST_X(dropoff::geometry), dropoff_known, delivery_fee
		  FROM orders WHERE id = $1`, id).Scan(&lat, &lng, &known, &fee); err != nil {
		t.Fatal(err)
	}
	return
}

func waNotes(t *testing.T, h *Harness, userID, orderID, title string) []string {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT body FROM notifications WHERE user_id = $1::uuid AND entity_id = $2 AND title = $3`,
		userID, orderID, title)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var b string
		if err := rows.Scan(&b); err != nil {
			t.Fatal(err)
		}
		out = append(out, b)
	}
	return out
}

// TestWAORD023_LocationRequestFromForm **أ · «اطلب الموقع» من النموذج** — رسالةُ المالك تصل الرقم،
// والنموذجُ يسأل فيرى «بانتظار» ثمّ النقطةَ حين يشاركها المستلِم بالرابط. **وبقدرة الواتساب وحدَها.**
//
// **وعلى الشيفرة القديمة لا باب** (٤٠٤/٤٠٥).
func TestWAORD023_LocationRequestFromForm(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-023")
	n := armFeedback(t, h, true)
	ops := h.NewUser("operations")
	phone := uniqPhone()

	fin := h.NewUser("finance")
	if r := h.POST(waURL+"/location-requests", fin.Token, map[string]any{"phone": phone}); r.Code != http.StatusForbidden {
		t.Fatalf("**الماليّةُ طلبت موقعاً**: %d", r.Code)
	}

	id, tok, j := waLocReq(t, h, ops.Token, phone, "")
	if j["whatsapp_sent"] != true {
		t.Fatalf("**الرسالةُ لم تُعلَّم مُرسَلة**: %v", j)
	}
	if msgs := n.to(phone); len(msgs) != 1 || msgs[0].text != waLocText(j["link"].(string)) {
		t.Fatalf("**رسالةُ المالك لم تصل الرقم**: %+v", msgs)
	}
	// **ضغطتان للرقم نفسِه في دقيقةٍ تُردّ.**
	if r := h.POST(waURL+"/location-requests", ops.Token, map[string]any{"phone": phone}); r.Code != http.StatusTooManyRequests {
		t.Fatalf("**الحدُّ لم يعمل**: %d", r.Code)
	}
	if st := waLocPoll(t, h, ops.Token, id); st["status"] != "pending" || st["lat"] != nil {
		t.Fatalf("**قبل المشاركة**: %v", st)
	}
	// **والصفحةُ العامّة بلا متجرٍ مكتوبٍ باسم المنصّة** — لا عنوانٌ فارغ.
	pub := h.GET("/api/v1/public/location-requests/"+tok, "")
	if pub.Code != http.StatusOK || pub.JSON()["mode"] != "whatsapp" || pub.JSON()["store_name"] == "" {
		t.Fatalf("**الصفحةُ العامّة**: %s", pub)
	}
	if r := h.POST("/api/v1/public/location-requests/"+tok, "", map[string]any{
		"lat": w.md.dropLat, "lng": w.md.dropLng, "accuracy_m": 9,
	}); r.Code != http.StatusOK {
		t.Fatalf("إرسالُ الموقع: %s", r)
	}
	st := waLocPoll(t, h, ops.Token, id)
	if st["status"] != "received" || !near(st["lat"], w.md.dropLat) || !near(st["lng"], w.md.dropLng) {
		t.Fatalf("**النموذجُ لا يرى الموقع الواصل**: %v", st)
	}
	// **وطلبُ موقعٍ لمتجرٍ لا يُقرأ من هذا الباب.**
	mr := h.POST(w.md.url("/location-requests"), w.md.fx.Tok, map[string]any{"phone": uniqPhone()})
	mid, _ := mr.JSON()["id"].(string)
	if r := h.GET(waURL+"/location-requests/"+mid, ops.Token); r.Code != http.StatusNotFound {
		t.Fatalf("**طلبُ متجرٍ قُرئ من باب المكتب**: %d", r.Code)
	}
}

// TestWAORD024_NoPointThenLocationArrives **ب · الطلبُ لا ينتظر الموقع** — يُنشأ بالعنوان وبأجرةٍ
// يكتبها الموظّف (التغطيةُ من نقطة المتجر)، وطلبُ الموقع مربوطٌ به. **فإن ردّ المستلِمُ بموقعه على
// الواتساب كُتب نقطةَ تسليمٍ ووصل السائقَ المُسنَدَ في لحظته، والأجرةُ كما هي.** ونقطةٌ خارجَ
// التغطية تُكتب ويُخبَر المكتب. **ووصولُه قبل الإنشاء يملأ النقطة.**
//
// **وعلى الشيفرة القديمة**: طلبٌ بلا نقطةٍ يُردّ (`bad_point`).
func TestWAORD024_NoPointThenLocationArrives(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-024")
	armFeedback(t, h, true)
	h.Setting("delivery.custom_fee_min", "10")
	h.Setting("delivery.custom_fee_max", "1000")
	ops := h.NewUser("operations")
	phone := uniqPhone()

	// **عرضُ السعر بلا نقطة**: الأساسُ اقتراحاً والأجرةُ مطلوبة.
	q := h.GET(waURL+"/quote?pickup_lat="+ftoa(w.storeLat)+"&pickup_lng="+ftoa(w.storeLng), ops.Token)
	if q.Code != http.StatusOK || q.JSON()["fee_required"] != true || q.JSON()["has_point"] != false {
		t.Fatalf("**عرضُ السعر بلا موقعٍ للمستلِم**: %d / %s", q.Code, q.Body)
	}

	reqID, _, _ := waLocReq(t, h, ops.Token, phone, "فلافل الأمير")
	b := w.body()
	delete(b, "lat")
	delete(b, "lng")
	b["recipient_phone"] = phone
	b["location_request_id"] = reqID
	if r := waPost(t, h, ops.Token, b); r.Err() != "wa_fee_required" {
		t.Fatalf("**بلا موقعٍ ولا أجرة**: %d / %s", r.Code, r.Err())
	}
	b["fee"] = 0
	if r := waPost(t, h, ops.Token, b); r.Err() != "wa_fee_out_of_range" {
		t.Fatalf("**أجرةٌ صفرٌ بلا موقع**: %d / %s", r.Code, r.Err())
	}
	b["fee"] = 650
	id := waCreated(t, waPost(t, h, ops.Token, b))
	lat, lng, known, fee := waOrderPoint(t, h, id)
	if known || fee != 650 || !nearF(lat, w.storeLat) || !nearF(lng, w.storeLng) {
		t.Fatalf("**الطلبُ بلا موقع**: معروفة %v · أجرة %d · نقطة %v,%v (المنتظَر نقطةُ المتجر غيرَ معروفة)",
			known, fee, lat, lng)
	}
	var linked string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(order_id::text, '') FROM recipient_location_requests WHERE id = $1`,
		reqID).Scan(&linked)
	if linked != id {
		t.Fatalf("**طلبُ الموقع لم يُربط بالطلب**: %q", linked)
	}

	// **السائقُ أخذه ثمّ ردّ المستلِمُ بموقعه على الواتساب.**
	drv := h.driverOf(id)
	if reply := h.API.HandleWALocation(ctxBG(), phone, w.md.dropLat, w.md.dropLng, 7); reply == "" {
		t.Fatal("**لا ردَّ على موقعٍ لطلبٍ مفتوح**")
	}
	lat, lng, known, fee = waOrderPoint(t, h, id)
	if !known || fee != 650 || !nearF(lat, w.md.dropLat) || !nearF(lng, w.md.dropLng) {
		t.Fatalf("**الموقعُ الواصلُ لم يُكتب نقطةَ تسليم** (أو تبدّلت الأجرة): %v · %d · %v,%v", known, fee, lat, lng)
	}
	got := driverOrderOf(t, h, drv.Token, id)
	if got["dropoff_known"] != true || !near(got["lat"], w.md.dropLat) {
		t.Fatalf("**السائقُ لا يرى الموقعَ الواصل**: %v / %v", got["dropoff_known"], got["lat"])
	}
	if bodies := waNotes(t, h, drv.ID, id, "وصل موقع المستلم — صار على الخريطة"); len(bodies) != 1 ||
		strings.Contains(bodies[0], strings.TrimPrefix(phone, "+963")) {
		t.Fatalf("**السائقُ لم يُخبَر بالموقع (أو وصله الرقم)**: %v", bodies)
	}

	// **ونقطةٌ خارجَ التغطية تُكتب ويُخبَر المكتب.**
	phone2 := uniqPhone()
	req2, tok2, _ := waLocReq(t, h, ops.Token, phone2, "")
	b2 := w.body()
	delete(b2, "lat")
	delete(b2, "lng")
	b2["recipient_phone"] = phone2
	b2["location_request_id"] = req2
	b2["fee"] = 500
	id2 := waCreated(t, waPost(t, h, ops.Token, b2))
	farLat := w.md.z.Lat + 0.08
	if r := h.POST("/api/v1/public/location-requests/"+tok2, "", map[string]any{
		"lat": farLat, "lng": w.md.z.Lng,
	}); r.Code != http.StatusOK {
		t.Fatalf("إرسالُ موقعٍ خارجَ التغطية: %s", r)
	}
	lat, _, known, _ = waOrderPoint(t, h, id2)
	if !known || !nearF(lat, farLat) {
		t.Fatalf("**موقعٌ خارجَ التغطية لم يُكتب**: %v · %v", known, lat)
	}
	var told int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE entity_id = $1 AND title = 'موقع مستلم طلب واتساب برّا التغطية'`, id2).Scan(&told)
	if told == 0 {
		t.Fatalf("**المكتبُ لم يُخبَر بموقعٍ خارجَ التغطية**")
	}

	// **ووصولُه قبل الإنشاء يملأ النقطة** — والأجرةُ محسوبةٌ منها.
	phone3 := uniqPhone()
	req3, _, _ := waLocReq(t, h, ops.Token, phone3, "")
	if reply := h.API.HandleWALocation(ctxBG(), phone3, w.md.dropLat, w.md.dropLng, 5); reply == "" {
		t.Fatal("لا ردّ")
	}
	b3 := w.body()
	delete(b3, "lat")
	delete(b3, "lng")
	b3["recipient_phone"] = phone3
	b3["location_request_id"] = req3
	id3 := waCreated(t, waPost(t, h, ops.Token, b3))
	lat, _, known, fee = waOrderPoint(t, h, id3)
	if !known || !nearF(lat, w.md.dropLat) || fee != 500 {
		t.Fatalf("**الموقعُ الواصلُ قبل الإنشاء لم يملأ النقطة**: %v · %v · %d", known, lat, fee)
	}
	// **وطلبُ موقعٍ مربوطٌ لا يُربط ثانيةً.**
	b3["recipient_phone"] = uniqPhone()
	if r := waPost(t, h, ops.Token, b3); r.Err() != "location_request_invalid" {
		t.Fatalf("**طلبُ موقعٍ رُبط بطلبين**: %d / %s", r.Code, r.Err())
	}
}

func nearF(a, b float64) bool { return a > b-1e-6 && a < b+1e-6 }

// TestWAORD025_FollowUpOnceForOfficeRequests **د · «اكتبلنا عنوانك» مرّةً واحدة** — بعد الدقائق
// المضبوطة بلا موقع، **لطلبات المكتب وحدَها**؛ ولا لمن وصل موقعُه، ولا لطلب المتجر، ولا بالصفر.
//
// **وعلى الشيفرة القديمة لا جولةَ متابعة** (`SweepWALocationFollowUp` غيرُ موجودة — لا يُبنى الفحص).
func TestWAORD025_FollowUpOnceForOfficeRequests(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-025")
	n := armFeedback(t, h, true)
	ops := h.NewUser("operations")
	h.Setting("orders.whatsapp_location_followup_min", "3")
	// **وكلُّ طلب موقعٍ آخرَ في القاعدة خارجَ الجولة** — القاعدةُ مشتركةٌ بين الفحوص.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE recipient_location_requests SET followup_sent_at = now() WHERE followup_sent_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	backdate := func(id string, minutes int) {
		if _, err := h.Pool.Exec(ctxBG(), `
			UPDATE recipient_location_requests SET created_at = now() - make_interval(mins => $2) WHERE id = $1`,
			id, minutes); err != nil {
			t.Fatal(err)
		}
	}

	waiting := uniqPhone()
	waitID, _, _ := waLocReq(t, h, ops.Token, waiting, "")
	backdate(waitID, 4)
	fresh := uniqPhone()
	freshID, _, _ := waLocReq(t, h, ops.Token, fresh, "")
	backdate(freshID, 1)
	answered := uniqPhone()
	ansID, _, _ := waLocReq(t, h, ops.Token, answered, "")
	h.API.HandleWALocation(ctxBG(), answered, w.md.dropLat, w.md.dropLng, 5)
	backdate(ansID, 4)
	store := uniqPhone()
	sr := h.POST(w.md.url("/location-requests"), w.md.fx.Tok, map[string]any{"phone": store})
	storeID, _ := sr.JSON()["id"].(string)
	backdate(storeID, 4)

	if got := h.API.SweepWALocationFollowUp(ctxBG()); got != 1 {
		t.Fatalf("**الجولةُ بعثت %d — والمنتظَر 1** (المنتظِرُ بعد ٣ دقائق وحدَه)", got)
	}
	want := "إذا ما قدرت تشارك الموقع، اكتبلنا عنوانك بالتفصيل برسالة وحدة: الحي، الشارع، وأقرب معلم 🌷"
	if msgs := n.to(waiting); len(msgs) != 2 || msgs[1].text != want {
		t.Fatalf("**رسالةُ المتابعة**: %+v", msgs)
	}
	for name, p := range map[string]string{"الحديث": fresh, "من ردّ": answered, "طلب المتجر": store} {
		if msgs := n.to(p); len(msgs) != 1 {
			t.Errorf("**وصلت المتابعةُ %s**: %d رسائل", name, len(msgs))
		}
	}
	if got := h.API.SweepWALocationFollowUp(ctxBG()); got != 0 {
		t.Fatalf("**المتابعةُ بُعثت مرّتين**: %d", got)
	}
	if st := waLocPoll(t, h, ops.Token, waitID); st["followup_sent"] != true {
		t.Fatalf("**النموذجُ لا يعرف أنّها بُعثت**: %v", st)
	}

	// **والصفرُ «لا».**
	h.Setting("orders.whatsapp_location_followup_min", "0")
	off := uniqPhone()
	offID, _, _ := waLocReq(t, h, ops.Token, off, "")
	backdate(offID, 10)
	if got := h.API.SweepWALocationFollowUp(ctxBG()); got != 0 {
		t.Fatalf("**بُعثت والإعدادُ صفر**: %d", got)
	}
}

// TestWAORD026_OfficeEditsAddress **د · المكتبُ يصحّح العنوانَ المكتوب** — قبل التسليم، بتدقيقٍ فيه
// القديمُ والجديد، ويصل السائقَ المُسنَدَ بلا رقم المستلِم. **وبعد التسليم لا، ولغير الواتساب لا.**
//
// **وعلى الشيفرة القديمة لا باب** (٤٠٤/٤٠٥).
func TestWAORD026_OfficeEditsAddress(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-026")
	ops := h.NewUser("operations")
	c := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	drv := h.driverOf(c.ID)

	fin := h.NewUser("finance")
	if r := h.POST(waURL+"/"+c.ID+"/address", fin.Token, map[string]any{"address_text": "حي الثكنة"}); r.Code != http.StatusForbidden {
		t.Fatalf("**الماليّةُ عدّلت العنوان**: %d", r.Code)
	}
	if r := h.POST(waURL+"/"+c.ID+"/address", ops.Token, map[string]any{"address_text": "  "}); r.Code != http.StatusBadRequest {
		t.Fatalf("**عنوانٌ فارغ**: %d", r.Code)
	}
	addr := "حي الثكنة، شارع المنصور، جانب صيدلية الشفاء"
	r := h.POST(waURL+"/"+c.ID+"/address", ops.Token, map[string]any{"address_text": addr})
	if r.Code != http.StatusOK {
		t.Fatalf("**تعديلُ العنوان رُدّ**: %d / %s", r.Code, r.Err())
	}
	var now string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT address_text FROM orders WHERE id = $1`, c.ID).Scan(&now)
	if now != addr {
		t.Fatalf("**العنوانُ لم يُكتب**: %q", now)
	}
	var details string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT details::text FROM audit_log
		WHERE action = 'ops.whatsapp_order_address' AND entity_id = $1 AND actor_user_id = $2::uuid`,
		c.ID, ops.ID).Scan(&details)
	if !strings.Contains(details, "حي الرميلة جانب الجامع") || !strings.Contains(details, addr) {
		t.Fatalf("**السجلُّ لا يحمل القديمَ والجديد**: %s", details)
	}
	if got := driverOrderOf(t, h, drv.Token, c.ID); !strings.HasPrefix(got["address_text"].(string), addr) {
		t.Fatalf("**السائقُ لا يقرأ العنوانَ الجديد**: %v", got["address_text"])
	}
	bodies := waNotes(t, h, drv.ID, c.ID, "تغيّر عنوان المستلم")
	if len(bodies) != 1 || !strings.Contains(bodies[0], addr) || strings.Contains(bodies[0], "935111222") {
		t.Fatalf("**السائقُ لم يُخبَر (أو وصله الرقم)**: %v", bodies)
	}

	// **وبعد التسليم لا.**
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET status = 'delivered', delivered_at = now() WHERE id = $1`, c.ID); err != nil {
		t.Fatal(err)
	}
	if r := h.POST(waURL+"/"+c.ID+"/address", ops.Token, map[string]any{"address_text": "غيره"}); r.Err() != "wa_address_closed" {
		t.Fatalf("**عُدّل عنوانُ طلبٍ مُسلَّم**: %d / %s", r.Code, r.Err())
	}
	// **ولغير طلب الواتساب لا.**
	md := w.md.create(t, h, "recipient", uniq("wa26"))
	if r := h.POST(waURL+"/"+md+"/address", ops.Token, map[string]any{"address_text": "غيره"}); r.Err() != "wa_not_whatsapp" {
		t.Fatalf("**عُدّل عنوانُ «لدي توصيلة» من باب الواتساب**: %d / %s", r.Code, r.Err())
	}
}
