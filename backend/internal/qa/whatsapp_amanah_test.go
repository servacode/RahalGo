package qa

// ══════════════════════════════════════════════════════════════════════
// **«أمانة من شخص» بطلب الواتساب** — قرارُ المالك ٢٠٢٦-١١-١١
// ══════════════════════════════════════════════════════════════════════
//
// «لازم في خيار امانة كمان بحيث الرسالة توصل لصاحب الامانة والي بدو يستلمها وهيك الاثنين يوصلنا
// موقعهم فنروح ع موقع الامانة نستلمها ونرجع نوصلها لصاحبها».
//
//	١ صاحبُ الأمانة مكانَ المتجر: رقمُه إلزاميّ، واسمُه اختياريّ، ولا متجرَ معه ولا رقمٌ واحدٌ للطرفين
//	٢ البوتُ يطلب الموقعَ من الطرفين — رسالةٌ واحدةٌ بسطر «لديك أمانة»
//	٣ لا ينزل للسائقين حتّى يصل موقعُ صاحب الأمانة — ولا بيد المكتب
//	٤ موقعُ صاحب الأمانة يملأ نقطةَ الاستلام وينزل الطلب، وموقعُ المستلِم يملأ التسليمَ وحدَه
//	٥ نقطةُ صاحب الأمانة المحدَّدةُ يدويّاً تُنزله فوراً، ووضعُ «من متجر» كما كان
//
// **والأرقامُ كلُّها وهميّةٌ** (`uniqPhone`).

import (
	"strings"
	"testing"
	"time"
)

// waAmanahBody **ما يرسله النموذجُ في وضع «أمانة من شخص».**
func waAmanahBody(sender, recipient string, fee int) map[string]any {
	return map[string]any{
		"amanah": true, "sender_name": "أبو خالد", "sender_phone": sender,
		"recipient_phone": recipient, "details": "كيس أوراق", "fee": fee,
	}
}

// waLocReqCount **طلباتُ الموقع المربوطة بالطلب لهدفٍ ورقم** — `dropoff` للمستلِم و`pickup` لصاحب الأمانة،
// **وكلاهما طلبُ المكتب** (`mode='whatsapp'`).
func waLocReqCount(t *testing.T, h *Harness, orderID, target, phone string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM recipient_location_requests
		WHERE order_id = $1 AND mode = 'whatsapp' AND target = $2 AND phone = $3 AND whatsapp_sent`,
		orderID, target, phone).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestWAAMN001_Validations **حرّاسُ الإنشاء** — رقمُ صاحب الأمانة، ورقمان مختلفان، ولا متجرَ ولا ثمن.
//
// **وعلى الشيفرة القديمة**: `amanah` يُهمَل فيُردّ الطلبُ بـ`wa_store_required` في كلّ حال.
func TestWAAMN001_Validations(t *testing.T) {
	h := New(t)
	_ = waSimpleFx(t, h, "منطقةُ WA-AMN-001")
	ops := h.NewUser("operations")
	sender, recipient := uniqPhone(), uniqPhone()

	b := waAmanahBody("", recipient, 600)
	if r := waPost(t, h, ops.Token, b); r.Err() != "wa_amanah_sender_required" {
		t.Fatalf("**بلا رقم صاحب الأمانة**: %d / %s", r.Code, r.Err())
	}
	if r := waPost(t, h, ops.Token, waAmanahBody(recipient, recipient, 600)); r.Err() != "wa_same_phone" {
		t.Fatalf("**رقمٌ واحدٌ للطرفين**: %d / %s", r.Code, r.Err())
	}
	b = waAmanahBody(sender, recipient, 600)
	b["store_name"] = "فلافل الأمير"
	if r := waPost(t, h, ops.Token, b); r.Err() != "wa_amanah_store" {
		t.Fatalf("**متجرٌ مع الأمانة**: %d / %s", r.Code, r.Err())
	}
	b = waAmanahBody(sender, recipient, 600)
	b["goods_amount"] = 2400
	if r := waPost(t, h, ops.Token, b); r.Err() != "validation" {
		t.Fatalf("**ثمنُ بضاعةٍ في أمانة**: %d / %s", r.Code, r.Err())
	}
	b = waAmanahBody(sender, recipient, 600)
	delete(b, "fee")
	if r := waPost(t, h, ops.Token, b); r.Err() != "wa_fee_required" {
		t.Fatalf("**أمانةٌ بلا نقطتين بلا أجرة**: %d / %s", r.Code, r.Err())
	}
}

// TestWAAMN002_WaitsForSenderThenDispatches **الأمانةُ تنتظر موقعَ صاحبها ثمّ تنزل.**
//
// **وعلى الشيفرة القديمة**: لا وضعَ أمانة (الإنشاءُ يُردّ)، ولو وُجد لنزل للسائقين بلا نقطة استلام.
func TestWAAMN002_WaitsForSenderThenDispatches(t *testing.T) {
	h := New(t)
	w := waSimpleFx(t, h, "منطقةُ WA-AMN-002")
	n := armFeedback(t, h, true)
	ops := h.NewUser("operations")
	drv := w.md.f.Driver(OnShift(), LocationAt(w.storeLat, w.storeLng, time.Now()))
	sender, recipient := uniqPhone(), uniqPhone()

	r := waPost(t, h, ops.Token, waAmanahBody(sender, recipient, 600))
	id := waCreated(t, r)
	if r.JSON()["pickup_pending"] != true || r.JSON()["amanah"] != true {
		t.Fatalf("**الردُّ لا يقول إنّها تنتظر موقعَ صاحبها**: %s", r.Body)
	}

	// ── ١ · الصفّ: مقبولٌ لا في الطابور، وصاحبُ الأمانة مكانَ المتجر، ولا ثمن ─────────
	var status, pickName, pickPhone, fetchStatus string
	var amanah, pickKnown bool
	var collect, fetchAmount *int64
	var fee, cash int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT status, wa_amanah, pickup_name, COALESCE(pickup_phone, ''), pickup_override IS NOT NULL,
		       collect_amount, fetch_amount, COALESCE(fetch_amount_status, ''), delivery_fee, cash_due
		  FROM orders WHERE id = $1`, id).
		Scan(&status, &amanah, &pickName, &pickPhone, &pickKnown, &collect, &fetchAmount, &fetchStatus,
			&fee, &cash); err != nil {
		t.Fatal(err)
	}
	if status != "accepted" || !amanah || pickName != "أبو خالد" || pickPhone != sender || pickKnown ||
		collect != nil || fetchAmount == nil || *fetchAmount != 0 || fetchStatus != "approved" ||
		fee != 600 || cash != 600 {
		t.Fatalf("**الصفّ**: %s · أمانة %v · %q/%q · نقطة %v · بضاعة %v · ثمن %v/%q · %d/%d",
			status, amanah, pickName, pickPhone, pickKnown, collect, fetchAmount, fetchStatus, fee, cash)
	}

	// ── ٢ · رسالتان لرقمين: «لديك أمانة» ─────────────────────────────────
	if waLocReqCount(t, h, id, "dropoff", recipient) != 1 || waLocReqCount(t, h, id, "pickup", sender) != 1 {
		t.Fatalf("**طلبا الموقع لم يُكتبا للطرفين**")
	}
	for name, p := range map[string]string{"صاحب الأمانة": sender, "المستلم": recipient} {
		if msgs := n.to(p); len(msgs) != 1 || msgs[0].text != waLocText(true) {
			t.Fatalf("**رسالةُ %s**: %+v", name, msgs)
		}
	}

	// ── ٣ · لا يراه سائقٌ، ولا ينزله المكتبُ بيده ───────────────────────────
	if containsID(h.GET("/api/v1/driver/queue", drv.Token), id) ||
		containsID(h.GET("/api/v1/driver/orders", drv.Token), id) {
		t.Fatalf("**السائقُ يرى أمانةً بلا نقطة استلام**")
	}
	if tr := h.POST("/api/v1/admin/orders/"+id+"/transition", ops.Token,
		map[string]any{"to": "dispatching"}); tr.Err() != "wa_pickup_pending" {
		t.Fatalf("**المكتبُ أنزلها قبل الموقع**: %d / %s", tr.Code, tr.Err())
	}

	// ── ٤ · موقعُ المستلِم يملأ التسليمَ وحدَه ────────────────────────────
	if reply := h.API.HandleWALocation(ctxBG(), recipient, w.md.dropLat, w.md.dropLng, 6); reply == "" {
		t.Fatal("**لا ردَّ على موقع المستلِم**")
	}
	lat, lng, known, _ := waOrderPoint(t, h, id)
	if !known || !nearF(lat, w.md.dropLat) || !nearF(lng, w.md.dropLng) {
		t.Fatalf("**موقعُ المستلِم لم يُكتب تسليماً**: %v · %v,%v", known, lat, lng)
	}
	if st := h.statusOf(id); st != "accepted" {
		t.Fatalf("**نزل بموقع المستلِم وحدَه**: %s", st)
	}

	// ── ٥ · موقعُ صاحب الأمانة يملأ الاستلامَ وينزل الطلب ─────────────────
	if reply := h.API.HandleWALocation(ctxBG(), sender, w.storeLat, w.storeLng, 6); reply == "" {
		t.Fatal("**لا ردَّ على موقع صاحب الأمانة**")
	}
	var pLat, pLng float64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT ST_Y(pickup_override::geometry), ST_X(pickup_override::geometry) FROM orders WHERE id = $1`, id).
		Scan(&pLat, &pLng); err != nil {
		t.Fatalf("**نقطةُ الاستلام لم تُكتب**: %v", err)
	}
	if !nearF(pLat, w.storeLat) || !nearF(pLng, w.storeLng) {
		t.Fatalf("**نقطةُ الاستلام**: %v,%v والمنتظَر %v,%v", pLat, pLng, w.storeLat, w.storeLng)
	}
	// **ونقطةُ التسليم لم تُمسّ.**
	if lat2, lng2, known2, _ := waOrderPoint(t, h, id); !known2 || !nearF(lat2, lat) || !nearF(lng2, lng) {
		t.Fatalf("**موقعُ صاحب الأمانة كُتب فوق التسليم**: %v,%v", lat2, lng2)
	}
	if st := h.statusOf(id); st != "dispatching" && st != "assigned" {
		t.Fatalf("**لم ينزل للسائقين بعد موقع صاحب الأمانة**: %s", st)
	}
	if !containsID(h.GET("/api/v1/driver/queue", drv.Token), id) &&
		!containsID(h.GET("/api/v1/driver/orders", drv.Token), id) {
		t.Fatalf("**السائقُ القريبُ لا يرى الأمانةَ بعد نزولها**")
	}

	// ── ٦ · السائقُ يقرأ «استلام من» صاحب الأمانة ورقمَه — بلا تحديث تطبيقه ────────
	d2 := h.driverOf(id)
	got := driverOrderOf(t, h, d2.Token, id)
	if got["merchant_name"] != "أبو خالد" || got["merchant_phone"] != sender {
		t.Fatalf("**السائقُ لا يرى صاحبَ الأمانة**: %v / %v", got["merchant_name"], got["merchant_phone"])
	}
	// **ولا ثمنَ يُكتب فوق الصفر في أمانة.**
	if tr := h.POST("/api/v1/driver/orders/"+id+"/transition", d2.Token, map[string]any{"to": "at_pickup"}); tr.Code >= 400 {
		t.Fatalf("وصل مكان الاستلام: %s", tr)
	}
	if fr := h.POST("/api/v1/driver/orders/"+id+"/fetch-amount", d2.Token, map[string]any{"amount": 2400}); fr.Err() != "fetch_amount_stage" {
		t.Fatalf("**ثمنُ بضاعةٍ في أمانة**: %d / %s", fr.Code, fr.Err())
	}

	// ── ٧ · وقائمةُ اللوحة تعرف الأمانة ──────────────────────────────────
	list := h.GET(waURL, ops.Token)
	rows, _ := list.JSON()["orders"].([]any)
	seen := false
	for _, x := range rows {
		if m, _ := x.(map[string]any); m != nil && m["id"] == id {
			seen = m["wa_amanah"] == true && m["pickup_pending"] == false
		}
	}
	if !seen {
		t.Fatalf("**القائمةُ لا تقول «أمانة»**: %s", list.Body)
	}
}

// TestWAAMN003_PinnedSenderGoesStraightToDrivers **نقطةُ صاحب الأمانة المحدَّدةُ يدويّاً تُنزله فوراً** —
// ولا يُطلب منه موقع.
func TestWAAMN003_PinnedSenderGoesStraightToDrivers(t *testing.T) {
	h := New(t)
	w := waSimpleFx(t, h, "منطقةُ WA-AMN-003")
	n := armFeedback(t, h, true)
	ops := h.NewUser("operations")
	sender, recipient := uniqPhone(), uniqPhone()

	b := waAmanahBody(sender, recipient, 600)
	b["sender_lat"], b["sender_lng"] = w.storeLat, w.storeLng
	r := waPost(t, h, ops.Token, b)
	id := waCreated(t, r)
	if r.JSON()["pickup_pending"] != false {
		t.Fatalf("**نقطةٌ محدَّدةٌ والردُّ يقول «تنتظر»**: %s", r.Body)
	}
	if st := h.statusOf(id); st != "dispatching" && st != "assigned" {
		t.Fatalf("**بنقطة صاحب الأمانة لم ينزل**: %s", st)
	}
	if waLocReqCount(t, h, id, "pickup", sender) != 0 || len(n.to(sender)) != 0 {
		t.Fatalf("**طُلب الموقعُ من صاحب أمانةٍ نقطتُه معروفة**")
	}
	if msgs := n.to(recipient); len(msgs) != 1 || msgs[0].text != waLocText(true) {
		t.Fatalf("**رسالةُ المستلِم**: %+v", msgs)
	}
}

// TestWAAMN004_StoreModeUnchanged **وضعُ «من متجر» كما كان** — في الطابور، بلا علامة أمانة، وطلبُ
// موقعٍ واحدٌ للزبون برسالة «لديك طلب».
func TestWAAMN004_StoreModeUnchanged(t *testing.T) {
	h := New(t)
	_ = waSimpleFx(t, h, "منطقةُ WA-AMN-004")
	n := armFeedback(t, h, true)
	ops := h.NewUser("operations")
	phone := uniqPhone()

	r := waPost(t, h, ops.Token, waSimpleBody(phone, 600))
	id := waCreated(t, r)
	var status string
	var amanah bool
	var fetchStatus *string
	var pickups int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT status, wa_amanah, fetch_amount_status,
		       (SELECT count(*) FROM recipient_location_requests WHERE order_id = o.id AND target = 'pickup')
		  FROM orders o WHERE id = $1`, id).Scan(&status, &amanah, &fetchStatus, &pickups); err != nil {
		t.Fatal(err)
	}
	if (status != "dispatching" && status != "assigned") || amanah || fetchStatus != nil || pickups != 0 {
		t.Fatalf("**وضعُ المتجر تغيّر**: %s · أمانة %v · ثمن %v · طلباتُ استلام %d", status, amanah, fetchStatus, pickups)
	}
	if msgs := n.to(phone); len(msgs) != 1 || msgs[0].text != waLocTextWA() {
		t.Fatalf("**رسالةُ الزبون**: %+v", msgs)
	}
	if r.JSON()["pickup_pending"] != false {
		t.Fatalf("**وضعُ المتجر يقول «تنتظر»**: %s", r.Body)
	}
}

// TestWAAMN005_FollowUpCoversSender **«ما وصلنا موقعك» تصل صاحبَ الأمانة** ما دامت نقطةُ الاستلام
// مجهولة — ولا تصله بعد أن يصل موقعُه.
func TestWAAMN005_FollowUpCoversSender(t *testing.T) {
	h := New(t)
	w := waSimpleFx(t, h, "منطقةُ WA-AMN-005")
	n := armFeedback(t, h, true)
	ops := h.NewUser("operations")
	h.Setting("orders.whatsapp_location_followup_min", "3")
	// **وكلُّ طلب موقعٍ آخرَ في القاعدة خارجَ الجولة** — القاعدةُ مشتركةٌ بين الفحوص.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE recipient_location_requests SET followup_sent_at = now() WHERE followup_sent_at IS NULL`); err != nil {
		t.Fatal(err)
	}
	sender, recipient := uniqPhone(), uniqPhone()
	id := waCreated(t, waPost(t, h, ops.Token, waAmanahBody(sender, recipient, 600)))
	// **المستلِمُ ردّ، وصاحبُ الأمانة لم يردّ.**
	h.API.HandleWALocation(ctxBG(), recipient, w.md.dropLat, w.md.dropLng, 6)
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE recipient_location_requests SET created_at = now() - interval '4 minutes' WHERE order_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if got := h.API.SweepWALocationFollowUp(ctxBG()); got != 1 {
		t.Fatalf("**الجولةُ بعثت %d — والمنتظَر 1** (صاحبُ الأمانة وحدَه)", got)
	}
	if msgs := n.to(sender); len(msgs) != 2 || !strings.Contains(msgs[1].text, "ما وصلنا موقعك") {
		t.Fatalf("**تذكيرُ صاحب الأمانة**: %+v", msgs)
	}
	if msgs := n.to(recipient); len(msgs) != 1 {
		t.Fatalf("**ذُكِّر مستلِمٌ وصل موقعُه**: %+v", msgs)
	}
	if st := h.statusOf(id); st != "accepted" {
		t.Fatalf("**الجولةُ أنزلت أمانةً بلا موقع صاحبها**: %s", st)
	}
}
