package qa

// ══════════════════════════════════════════════════════════════════════
// **«بدّي أبعت أمانة»** (قرارُ المالك ٢٠٢٦-١١-١٩)
// ══════════════════════════════════════════════════════════════════════
//
// الاستلامُ من عنوان الزبون، والتسليمُ لموقع المستلِم (يصل بالواتساب — قبل الإرسال أو بعده). ولحدّ ما
// يصل `dropoff_known = false` فلا يُوجَّه الكابتنُ لبيت المرسل. والكودُ للمستلِم بالواتساب.
// **و«أستقبل» (الافتراض والتطبيقُ القديم) كما كانت.**

import (
	"net/http"
	"strings"
	"testing"
)

func amanahSendBody(req string) map[string]any {
	b := amanahBody(req)
	b["amanah_direction"] = "send"
	b["holder_name"] = "أم خالد"
	return b
}

// TestAmanahSend_SharedAfterSend **أبعت، والمستلِمُ شارك بعد الإرسال.**
func TestAmanahSend_SharedAfterSend(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	reqID, tok := requestHolderLocation(t, h, cust.Token, "0944777888")
	b := amanahSendBody("أمانة: ظرف أوراق")
	b["holder_phone"] = "0944777888"
	b["location_request_id"] = reqID
	created := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("ams-a"), b)
	if created.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %s", created)
	}
	c := created.JSON()
	id, _ := c["id"].(string)
	if c["amanah_direction"] != "send" || !near(c["amanah_pickup_lat"], amLat) || dropoffKnownOf(t, h, id) {
		t.Fatalf("ردُّ الإنشاء: اتّجاه %v · استلام %v · معروف %v", c["amanah_direction"], c["amanah_pickup_lat"], dropoffKnownOf(t, h, id))
	}

	drv := h.NewUser("driver")
	assignDriver(t, h, id, drv.ID)
	got := driverOrderOf(t, h, drv.Token, id)
	if got["amanah_direction"] != "send" || !near(got["pickup_lat"], amLat) || got["dropoff_known"] != false {
		t.Fatalf("الكابتن: اتّجاه %v · استلام %v · معروف %v", got["amanah_direction"], got["pickup_lat"], got["dropoff_known"])
	}
	if got["pickup_address"] != "عنوان المرسل" || got["holder_name"] != "أم خالد" {
		t.Fatalf("الكابتن: عنوان %v · اسم %v", got["pickup_address"], got["holder_name"])
	}

	// **والمستلِمُ يشارك بعد الإرسال ⇒ نقطةُ التسليم.**
	shareLocation(t, h, tok)
	got = driverOrderOf(t, h, drv.Token, id)
	if got["dropoff_known"] != true || !near(got["lat"], amPLat) || !near(got["pickup_lat"], amLat) {
		t.Fatalf("بعد المشاركة: معروف %v · تسليم %v · استلام %v", got["dropoff_known"], got["lat"], got["pickup_lat"])
	}
	var pickupNote, dir string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT pickup_override_note, amanah_direction FROM orders WHERE id = $1`, id).Scan(&pickupNote, &dir)
	if pickupNote != "عنوان المرسل" || dir != "send" {
		t.Fatalf("القاعدة: %q %q", pickupNote, dir)
	}
}

// TestAmanahSend_SharedBeforeSendAndCodeToRecipient **المستلِمُ شارك قبل الإرسال، والكودُ له بالواتساب.**
func TestAmanahSend_SharedBeforeSendAndCodeToRecipient(t *testing.T) {
	h := New(t)
	bot := &liveNudgeBot{}
	h.API.SetMerchantNotifier(bot)
	h.Setting("delivery.code_required", "true")
	cust := h.Customer()
	reqID, tok := requestHolderLocation(t, h, cust.Token, "0955111222")
	shareLocation(t, h, tok)
	b := amanahSendBody("أمانة: مفتاح")
	b["holder_phone"] = "0955111222"
	b["location_request_id"] = reqID
	created := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("ams-b"), b)
	if created.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %s", created)
	}
	c := created.JSON()
	id, _ := c["id"].(string)
	if !dropoffKnownOf(t, h, id) || !near(c["lat"], amPLat) {
		t.Fatalf("ردُّ الإنشاء: معروف %v · تسليم %v", dropoffKnownOf(t, h, id), c["lat"])
	}
	var phone string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT amanah_holder_phone FROM orders WHERE id = $1`, id).Scan(&phone); err != nil {
		t.Fatal(err)
	}
	// **وتوثيقُ الأجرة ثمّ الاستلام ⇒ الكودُ للمستلِم.**
	drv := h.NewUser("driver")
	assignDriver(t, h, id, drv.ID)
	if r := h.POST("/api/v1/driver/orders/"+id+"/agree", drv.Token, map[string]any{"fee": 300}); r.Code >= 400 {
		t.Fatalf("الأجرة: %s", r)
	}
	var ver int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT quote_version FROM orders WHERE id = $1`, id).Scan(&ver)
	if r := h.POST("/api/v1/orders/"+id+"/confirm-quote", cust.Token,
		map[string]any{"payment_method": "cash", "expected_total": 300, "expected_quote_version": ver}); r.Code >= 400 {
		t.Fatalf("تأكيد السعر: %s", r)
	}
	for _, to := range []string{"picked_up"} {
		if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("الانتقال %s: %s", to, r)
		}
	}
	var code *string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT delivery_code FROM orders WHERE id = $1`, id).Scan(&code)
	if code == nil {
		t.Fatalf("ما انولد كود")
	}
	waitFor(t, func() bool { return bot.count() > 0 })
	toRecipient := false
	for _, m := range bot.sent {
		if strings.HasPrefix(m, phone+"|") && strings.Contains(m, *code) {
			toRecipient = true
		}
		if strings.HasPrefix(m, cust.Phone+"|") && strings.Contains(m, *code) {
			t.Fatalf("**الكود راح للمرسل بالواتساب بدل المستلم**: %q", m)
		}
	}
	if !toRecipient {
		t.Fatalf("الكود ما وصل للمستلم: %v", bot.sent)
	}
}

// TestAmanahSend_ReceiveUnchanged **«أستقبل» والتطبيقُ القديم كما كانا.**
func TestAmanahSend_ReceiveUnchanged(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	created := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("ams-r"), amanahBody("أمانة قديمة"))
	if created.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %s", created)
	}
	c := created.JSON()
	rid, _ := c["id"].(string)
	if c["amanah_direction"] != "receive" || !dropoffKnownOf(t, h, rid) || !near(c["lat"], amLat) {
		t.Fatalf("«أستقبل»: اتّجاه %v · تسليم %v", c["amanah_direction"], c["lat"])
	}
}

func dropoffKnownOf(t *testing.T, h *Harness, id string) bool {
	t.Helper()
	var k bool
	if err := h.Pool.QueryRow(ctxBG(), `SELECT dropoff_known FROM orders WHERE id = $1`, id).Scan(&k); err != nil {
		t.Fatal(err)
	}
	return k
}
