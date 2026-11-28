package qa

// ══════════════════════════════════════════════════════════════════════
// **الطرفُ الآخر يشارك موقعَه المباشر من تطبيقنا** (قرارُ المالك ٢٠٢٦-١١-٢٨)
// ══════════════════════════════════════════════════════════════════════
//
// الإشعارُ لصاحب الحساب وحدَه، والبطاقةُ لصاحب الرقم وحدَه (وغيرُه ٤٠٤)، و«نعم» قبل أيّ نقطة، وأوّلُ
// نقطةٍ تملأ طلبَ الموقع والطلبَ كردّ الواتساب، والتذكيرُ مرّةً واحدة، و«لا» سطرٌ بسجلّ الطلب.

import (
	"net/http"
	"strings"
	"testing"
)

// partyFx **أمانةٌ «أبعت» مستلمُها صاحبُ حساب.**
type partyFx struct {
	cust, party *User
	orderID     string
	reqID       string
}

func newPartyFx(t *testing.T, h *Harness, direction string) partyFx {
	t.Helper()
	cust := h.Customer()
	party := h.Customer()
	reqID, _ := requestHolderLocation(t, h, cust.Token, party.Phone)
	b := amanahBody("أمانة: ظرف")
	if direction == "send" {
		b = amanahSendBody("أمانة: ظرف")
	}
	b["holder_phone"] = party.Phone
	b["location_request_id"] = reqID
	created := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("party"), b)
	if created.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %s", created)
	}
	id, _ := created.JSON()["id"].(string)
	return partyFx{cust: cust, party: party, orderID: id, reqID: reqID}
}

func partyPushes(t *testing.T, h *Harness, userID, orderID string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM notifications
		 WHERE user_id = $1::uuid AND entity = 'party_order' AND entity_id::text = $2`,
		userID, orderID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestPartyLive_PushOnlyWhenAccountExists **الإشعارُ لرقمٍ عليه حساب، ولا شيءَ لرقمٍ بلا حساب.**
func TestPartyLive_PushOnlyWhenAccountExists(t *testing.T) {
	h := New(t)
	fx := newPartyFx(t, h, "send")
	waitFor(t, func() bool { return partyPushes(t, h, fx.party.ID, fx.orderID) == 1 })
	var title string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT title FROM notifications WHERE user_id = $1::uuid AND entity = 'party_order'`,
		fx.party.ID).Scan(&title)
	if !strings.Contains(title, "لديك أمانة من صاحب الرقم 0"+fx.cust.Phone[4:]) {
		t.Fatalf("عنوانُ الإشعار: %q", title)
	}
	// **والجولةُ لا تكرّره.**
	if inv, _ := h.API.SweepPartyInvites(ctxBG()); inv != 0 || partyPushes(t, h, fx.party.ID, fx.orderID) != 1 {
		t.Fatalf("الإشعارُ تكرّر: %d", partyPushes(t, h, fx.party.ID, fx.orderID))
	}

	// **رقمٌ بلا حساب ⇒ لا إشعار** (والواتساب كما هو).
	cust := h.Customer()
	stranger := uniqPhone()
	reqID, _ := requestHolderLocation(t, h, cust.Token, stranger)
	b := amanahSendBody("أمانة: مفتاح")
	b["holder_phone"] = stranger
	b["location_request_id"] = reqID
	created := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("party-n"), b)
	if created.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %s", created)
	}
	id, _ := created.JSON()["id"].(string)
	h.API.SweepPartyInvites(ctxBG())
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications WHERE entity = 'party_order' AND entity_id::text = $1`, id).Scan(&n)
	if n != 0 {
		t.Fatalf("**إشعارٌ لرقمٍ بلا حساب**: %d", n)
	}
}

// TestPartyLive_OnlyPhoneOwnerSeesIt **البطاقةُ لصاحب الرقم وحدَه — وغيرُه ومعرّفٌ مجهول ٤٠٤.**
func TestPartyLive_OnlyPhoneOwnerSeesIt(t *testing.T) {
	h := New(t)
	fx := newPartyFx(t, h, "send")
	drv := h.NewUser("driver")
	assignDriver(t, h, fx.orderID, drv.ID)

	got := h.GET("/api/v1/my/party-orders/"+fx.orderID, fx.party.Token)
	if got.Code != http.StatusOK {
		t.Fatalf("صاحبُ الرقم: %s", got)
	}
	v := got.JSON()
	if v["kind"] != "amanah" || v["role"] != "dropoff" || v["captain_assigned"] != true ||
		v["sender_phone"] != "0"+fx.cust.Phone[4:] || v["answer"] != "" {
		t.Fatalf("البطاقة: %v", v)
	}
	for k := range v {
		if _, ok := PartyViewFields[k]; !ok {
			t.Fatalf("**حقلٌ خارجَ عقد البطاقة**: %q", k)
		}
	}
	list := h.GET("/api/v1/my/party-orders", fx.party.Token)
	if list.Code != http.StatusOK || !strings.Contains(string(list.Body), fx.orderID) {
		t.Fatalf("القائمة: %s", list)
	}

	other := h.Customer()
	for _, path := range []string{"/api/v1/my/party-orders/" + fx.orderID,
		"/api/v1/my/party-orders/00000000-0000-0000-0000-000000000000", "/api/v1/my/party-orders/x"} {
		if r := h.GET(path, other.Token); r.Code != http.StatusNotFound {
			t.Fatalf("**غيرُه قرأ البطاقة** %s: %s", path, r)
		}
	}
	if r := h.POST("/api/v1/my/party-orders/"+fx.orderID+"/answer", other.Token, map[string]any{"answer": "yes"}); r.Code != http.StatusNotFound {
		t.Fatalf("**غيرُه جاوب**: %s", r)
	}
	if r := h.POST("/api/v1/my/party-orders/"+fx.orderID+"/live-location", other.Token,
		map[string]any{"lat": amPLat, "lng": amPLng}); r.Code != http.StatusNotFound {
		t.Fatalf("**غيرُه كتب موقعاً**: %s", r)
	}
	// **وصاحبُ الطلب نفسُه مو «طرف»** — وقائمتُه فاضية.
	if r := h.GET("/api/v1/my/party-orders/"+fx.orderID, fx.cust.Token); r.Code != http.StatusNotFound {
		t.Fatalf("صاحبُ الطلب: %s", r)
	}
	if list := h.GET("/api/v1/my/party-orders", other.Token); strings.Contains(string(list.Body), fx.orderID) {
		t.Fatalf("**قائمةُ غيره فيها الطلب**: %s", list)
	}
}

// TestPartyLive_YesThenFirstPointFillsRequestAndOrder **لا نقطةَ قبل «نعم»، وأوّلُها كردّ الواتساب.**
func TestPartyLive_YesThenFirstPointFillsRequestAndOrder(t *testing.T) {
	h := New(t)
	fx := newPartyFx(t, h, "send")
	drv := h.NewUser("driver")
	assignDriver(t, h, fx.orderID, drv.ID)
	pt := map[string]any{"lat": amPLat, "lng": amPLng, "accuracy_m": 6}
	live := "/api/v1/my/party-orders/" + fx.orderID + "/live-location"
	if r := h.POST(live, fx.party.Token, pt); r.Code != http.StatusConflict || !strings.Contains(string(r.Body), "party_not_consented") {
		t.Fatalf("**نقطةٌ قبل «نعم»**: %s", r)
	}
	if r := h.POST("/api/v1/my/party-orders/"+fx.orderID+"/answer", fx.party.Token, map[string]any{"answer": "yes"}); r.Code != http.StatusOK || r.JSON()["answer"] != "yes" {
		t.Fatalf("«نعم»: %s", r)
	}
	if r := h.POST(live, fx.party.Token, pt); r.Code != http.StatusOK {
		t.Fatalf("أوّلُ نقطة: %s", r)
	}
	var received bool
	_ = h.Pool.QueryRow(ctxBG(), `SELECT received_at IS NOT NULL FROM recipient_location_requests WHERE id = $1`, fx.reqID).Scan(&received)
	if !received || !dropoffKnownOf(t, h, fx.orderID) {
		t.Fatalf("أوّلُ نقطة ما عبّت: وصل %v · معروف %v", received, dropoffKnownOf(t, h, fx.orderID))
	}
	got := driverOrderOf(t, h, drv.Token, fx.orderID)
	if !near(got["lat"], amPLat) {
		t.Fatalf("الكابتن ما شاف نقطة التسليم: %v", got["lat"])
	}
	// **والنقاطُ اللاحقة موقعٌ مباشرٌ يقرؤه الكابتن** — بنفس الجدول.
	pt2 := map[string]any{"lat": amPLat + 0.001, "lng": amPLng, "accuracy_m": 5}
	if r := h.POST(live, fx.party.Token, pt2); r.Code != http.StatusOK {
		t.Fatalf("نقطةٌ ثانية: %s", r)
	}
	var lat float64
	var who string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT ST_Y(point::geometry), customer_id::text FROM order_live_locations WHERE order_id = $1`,
		fx.orderID).Scan(&lat, &who); err != nil {
		t.Fatalf("الموقعُ المباشر ما انكتب: %v", err)
	}
	if !near(lat, amPLat+0.001) || who != fx.party.ID {
		t.Fatalf("الموقعُ المباشر: %v %v", lat, who)
	}
	// **وبعد التسليم يقف** — ٤٠٩ لا ٤٠٤.
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET status = 'delivered', delivered_at = now() WHERE id = $1`, fx.orderID); err != nil {
		t.Fatal(err)
	}
	if r := h.POST(live, fx.party.Token, pt2); r.Code != http.StatusConflict || !strings.Contains(string(r.Body), "live_location_inactive") {
		t.Fatalf("بعد التسليم: %s", r)
	}
}

// TestPartyLive_ReminderOnce **التذكيرُ بعد خمس دقائق بلا جواب — مرّةً واحدة، ولا تذكيرَ لمن جاوب.**
func TestPartyLive_ReminderOnce(t *testing.T) {
	h := New(t)
	fx := newPartyFx(t, h, "send")
	waitFor(t, func() bool { return partyPushes(t, h, fx.party.ID, fx.orderID) == 1 })
	if _, rem := h.API.SweepPartyInvites(ctxBG()); rem != 0 {
		t.Fatalf("تذكيرٌ قبل وقته: %d", rem)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE recipient_location_requests SET party_push_at = now() - interval '6 minutes' WHERE id = $1`, fx.reqID); err != nil {
		t.Fatal(err)
	}
	if _, rem := h.API.SweepPartyInvites(ctxBG()); rem != 1 || partyPushes(t, h, fx.party.ID, fx.orderID) != 2 {
		t.Fatalf("التذكير: %d · الإشعارات %d", rem, partyPushes(t, h, fx.party.ID, fx.orderID))
	}
	if _, rem := h.API.SweepPartyInvites(ctxBG()); rem != 0 || partyPushes(t, h, fx.party.ID, fx.orderID) != 2 {
		t.Fatalf("**التذكيرُ تكرّر**: %d", partyPushes(t, h, fx.party.ID, fx.orderID))
	}

	// **ومن جاوب «لا» لا يُذكَّر — وتنكتب بسجلّ الطلب للمكتب.**
	fx2 := newPartyFx(t, h, "send")
	if r := h.POST("/api/v1/my/party-orders/"+fx2.orderID+"/answer", fx2.party.Token, map[string]any{"answer": "no"}); r.Code != http.StatusOK {
		t.Fatalf("«لا»: %s", r)
	}
	_, _ = h.Pool.Exec(ctxBG(), `UPDATE recipient_location_requests SET party_push_at = now() - interval '6 minutes' WHERE id = $1`, fx2.reqID)
	h.API.SweepPartyInvites(ctxBG())
	if n := partyPushes(t, h, fx2.party.ID, fx2.orderID); n > 1 {
		t.Fatalf("**تذكيرٌ بعد «لا»**: %d", n)
	}
	var notes int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_events WHERE order_id = $1 AND from_status = to_status AND note LIKE '%«لا»%'`,
		fx2.orderID).Scan(&notes)
	if notes != 1 {
		t.Fatalf("سطرُ «لا» بالسجلّ: %d", notes)
	}
}

// TestPartyLive_PickupHolderMovesPickup **صاحبُ «أستقبل»: أوّلُ نقطةٍ نقطةُ الاستلام، وحركتُه تحدّثها.**
func TestPartyLive_PickupHolderMovesPickup(t *testing.T) {
	h := New(t)
	fx := newPartyFx(t, h, "receive")
	if r := h.POST("/api/v1/my/party-orders/"+fx.orderID+"/answer", fx.party.Token, map[string]any{"answer": "yes"}); r.Code != http.StatusOK || r.JSON()["role"] != "pickup" {
		t.Fatalf("«نعم»: %s", r)
	}
	live := "/api/v1/my/party-orders/" + fx.orderID + "/live-location"
	if r := h.POST(live, fx.party.Token, map[string]any{"lat": amPLat, "lng": amPLng, "accuracy_m": 6}); r.Code != http.StatusOK {
		t.Fatalf("أوّلُ نقطة: %s", r)
	}
	pickLat := func() float64 {
		var v float64
		if err := h.Pool.QueryRow(ctxBG(), `SELECT ST_Y(pickup_override::geometry) FROM orders WHERE id = $1`, fx.orderID).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if !near(pickLat(), amPLat) {
		t.Fatalf("نقطةُ الاستلام: %v", pickLat())
	}
	_, _ = h.Pool.Exec(ctxBG(), `UPDATE recipient_location_requests SET party_live_at = now() - interval '1 minute' WHERE id = $1`, fx.reqID)
	if r := h.POST(live, fx.party.Token, map[string]any{"lat": amPLat + 0.001, "lng": amPLng, "accuracy_m": 6}); r.Code != http.StatusOK {
		t.Fatalf("نقطةٌ ثانية: %s", r)
	}
	if !near(pickLat(), amPLat+0.001) {
		t.Fatalf("نقطةُ الاستلام ما تحرّكت: %v", pickLat())
	}
}

// TestPartyLive_AmanahSenderOwnLiveRefused **مرسلُ «أبعت» ما بيكتب موقعاً مباشراً** — موقعُه مو الباب.
func TestPartyLive_AmanahSenderOwnLiveRefused(t *testing.T) {
	h := New(t)
	fx := newPartyFx(t, h, "send")
	drv := h.NewUser("driver")
	assignDriver(t, h, fx.orderID, drv.ID)
	r := h.POST("/api/v1/orders/"+fx.orderID+"/live-location", fx.cust.Token, map[string]any{"lat": amLat, "lng": amLng, "accuracy_m": 5})
	if r.Code != http.StatusConflict || !strings.Contains(string(r.Body), "live_location_inactive") {
		t.Fatalf("موقعُ المرسل المباشر: %s", r)
	}
}
