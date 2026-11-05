package qa

// ══════════════════════════════════════════════════════════════════════
// **طلباتُ الواتساب — مراجعةُ ملحق المالك** (٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
//	طلبُ الموقع لرقم المستلِم نفسِه وحدَه — رقمٌ غلطٌ صُحّح لا يُربط طلبُه
//	موقعٌ وصل قبل الإنشاء خارجَ التغطية لا يردّ الطلب — يُنشأ بالعنوان ثمّ يُكتب
//	الموقعُ الواصلُ ينقل الطلبَ إلى منطقته
//	موقعٌ وصل وسقطت كتابتُه تكتبه الجولة
//	القائمةُ: الجاريةُ أوّلاً — طلبٌ مفتوحٌ لا يغيب خلف المنتهية

import (
	"context"
	"testing"
)

// waNoPointBody **حمولةُ النموذج بلا موقعٍ للمستلِم** — برقمه وطلب موقعه وأجرةٍ مكتوبة.
func waNoPointBody(w waFx, phone, reqID string, fee int) map[string]any {
	b := w.body()
	delete(b, "lat")
	delete(b, "lng")
	b["recipient_phone"] = phone
	if reqID != "" {
		b["location_request_id"] = reqID
	}
	if fee > 0 {
		b["fee"] = fee
	}
	return b
}

func waZoneOf(t *testing.T, h *Harness, id string) string {
	t.Helper()
	var z string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(zone_id::text, '') FROM orders WHERE id = $1`, id).Scan(&z); err != nil {
		t.Fatal(err)
	}
	return z
}

// TestWAORD027_LocationRequestBoundToRecipientPhone **طلبُ الموقع لرقم المستلِم نفسِه وحدَه** —
// الموظّفُ طلب الموقعَ من رقمٍ غلطٍ وبعثه صاحبُه، ثمّ صحّح الرقمَ وأرسل الطلب: **يُردّ
// (`location_request_invalid`)** — لا يُربط فتذهب المتابعةُ إلى الغلط، ولا تصير نقطةُ الغريب
// نقطةَ التسليم. **وطلبُ الرقم الصحيح يُربط.**
//
// **وعلى الشيفرة السابقة** (d419e42e) يُنشأ الطلبُ وتُكتب فيه نقطةُ صاحب الرقم الغلط.
func TestWAORD027_LocationRequestBoundToRecipientPhone(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-027")
	armFeedback(t, h, true)
	h.Setting("delivery.custom_fee_min", "10")
	h.Setting("delivery.custom_fee_max", "1000")
	ops := h.NewUser("operations")
	wrong, right := uniqPhone(), uniqPhone()

	wrongReq, _, _ := waLocReq(t, h, ops.Token, wrong, "")
	if reply := h.API.HandleWALocation(ctxBG(), wrong, w.md.dropLat, w.md.dropLng, 5); reply == "" {
		t.Fatal("لا ردّ على موقع صاحب الرقم الغلط")
	}
	if r := waPost(t, h, ops.Token, waNoPointBody(w, right, wrongReq, 600)); r.Err() != "location_request_invalid" {
		t.Fatalf("**طلبُ موقعٍ لرقمٍ آخرَ رُبط بالطلب**: %d / %s", r.Code, r.Body)
	}
	var made int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM orders WHERE origin = 'whatsapp' AND recipient_phone = $1`,
		right).Scan(&made)
	if made != 0 {
		t.Fatalf("**أُنشئ طلبٌ بطلب موقع رقمٍ آخر**: %d", made)
	}

	rightReq, _, _ := waLocReq(t, h, ops.Token, right, "")
	id := waCreated(t, waPost(t, h, ops.Token, waNoPointBody(w, right, rightReq, 600)))
	var linked string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(order_id::text, '') FROM recipient_location_requests WHERE id = $1`,
		rightReq).Scan(&linked)
	if linked != id {
		t.Fatalf("**طلبُ موقع الرقم الصحيح لم يُربط**: %q", linked)
	}
	if _, _, known, _ := waOrderPoint(t, h, id); known {
		t.Fatal("**الطلبُ أخذ نقطةً لم يبعثها مستلِمُه**")
	}
}

// TestWAORD028_EarlyPointOutsideCoverage **موقعٌ وصل قبل الإنشاء خارجَ التغطية لا يردّ الطلب** —
// يُنشأ بالعنوان المكتوب وبأجرةٍ يكتبها الموظّف (بلاها `wa_fee_required` كأيّ طلبٍ بلا موقع)، **ثمّ
// تُكتب النقطةُ ويُخبَر المكتب** (البند ب: «النقطة برّا التغطية بتنكتب وبيتخبّر المكتب»).
//
// **وعلى الشيفرة السابقة** تُحشر النقطةُ في الإنشاء فيردّه حارسُ التغطية — ولا سبيلَ للموظّف.
func TestWAORD028_EarlyPointOutsideCoverage(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-028")
	armFeedback(t, h, true)
	h.Setting("delivery.custom_fee_min", "10")
	h.Setting("delivery.custom_fee_max", "1000")
	ops := h.NewUser("operations")
	phone := uniqPhone()

	reqID, tok, _ := waLocReq(t, h, ops.Token, phone, "")
	farLat := w.md.z.Lat + 0.08
	if r := h.POST("/api/v1/public/location-requests/"+tok, "", map[string]any{
		"lat": farLat, "lng": w.md.z.Lng,
	}); r.Code != 200 {
		t.Fatalf("إرسالُ موقعٍ خارجَ التغطية: %s", r)
	}
	if r := waPost(t, h, ops.Token, waNoPointBody(w, phone, reqID, 0)); r.Err() != "wa_fee_required" {
		t.Fatalf("**الموقعُ الواصلُ خارجَ التغطية حُشر في الإنشاء** (المنتظَر wa_fee_required): %d / %s",
			r.Code, r.Body)
	}
	r := waPost(t, h, ops.Token, waNoPointBody(w, phone, reqID, 600))
	id := waCreated(t, r)
	if r.JSON()["has_point"] != true {
		t.Fatalf("**الردُّ لا يقول إنّ الموقعَ الواصلَ على الطلب**: %s", r)
	}
	lat, _, known, fee := waOrderPoint(t, h, id)
	if !known || !nearF(lat, farLat) || fee != 600 {
		t.Fatalf("**الموقعُ الواصلُ قبل الإنشاء لم يُكتب بعده** (أو تبدّلت الأجرة): %v · %v · %d", known, lat, fee)
	}
	var told int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE entity_id = $1 AND title = 'موقع مستلم طلب واتساب برّا التغطية'`, id).Scan(&told)
	if told == 0 {
		t.Fatal("**المكتبُ لم يُخبَر بموقعٍ خارجَ التغطية**")
	}
	// **ومنطقتُه منطقةُ المتجر** — النقطةُ خارجَ كلّ منطقة.
	if z := waZoneOf(t, h, id); z != w.md.z.ID {
		t.Fatalf("**منطقةُ الطلب تبدّلت إلى غير منطقة**: %q", z)
	}
}

// TestWAORD029_ArrivedPointMovesOrderZone **الموقعُ الواصلُ ينقل الطلبَ إلى منطقته** — أُنشئ بلا
// موقعٍ على منطقة المتجر، **فلمّا وصل موقعُ المستلِم في منطقةٍ أخرى صارت منطقتَه** (كما لو أُنشئ
// به)، **والأجرةُ كما هي.**
//
// **وعلى الشيفرة السابقة** تبقى منطقةُ المتجر.
func TestWAORD029_ArrivedPointMovesOrderZone(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-029")
	armFeedback(t, h, true)
	h.Setting("delivery.custom_fee_min", "10")
	h.Setting("delivery.custom_fee_max", "1000")
	zb := newZone(t, h, "منطقةُ WA-029 ب", w.md.z.Lat+0.03, w.md.z.Lng)
	zoneHours(t, h, zb.ID, false)
	bLat, bLng := zb.Lat+0.002, zb.Lng
	if ok, err := h.Orders.CoverableAt(context.Background(), h.Pool, bLat, bLng); err != nil || !ok {
		t.Fatalf("نقطةُ المنطقة الثانية ليست مغطّاة: %v / %v", ok, err)
	}
	ops := h.NewUser("operations")
	phone := uniqPhone()

	reqID, _, _ := waLocReq(t, h, ops.Token, phone, "")
	id := waCreated(t, waPost(t, h, ops.Token, waNoPointBody(w, phone, reqID, 600)))
	if z := waZoneOf(t, h, id); z != w.md.z.ID {
		t.Fatalf("الطلبُ بلا موقعٍ ليس على منطقة المتجر: %q", z)
	}
	if reply := h.API.HandleWALocation(ctxBG(), phone, bLat, bLng, 6); reply == "" {
		t.Fatal("لا ردّ")
	}
	if z := waZoneOf(t, h, id); z != zb.ID {
		t.Fatalf("**الموقعُ الواصلُ لم ينقل الطلبَ إلى منطقته**: %q (المنتظَر %q)", z, zb.ID)
	}
	if _, _, known, fee := waOrderPoint(t, h, id); !known || fee != 600 {
		t.Fatalf("**النقطةُ لم تُكتب أو تبدّلت الأجرة**: %v · %d", known, fee)
	}
}

// TestWAORD030_SweepWritesALostLocation **موقعٌ وصل وسقطت كتابتُه في الطلب تكتبه الجولة** — طلبُ
// الموقع يقول «وصل» والطلبُ بلا نقطة (عطلٌ بين الخطوتين): **الجولةُ تكتبها وتُخبر السائقَ مرّةً**،
// والأجرةُ كما هي. **ولو أُطفئت المتابعة.**
//
// **وعلى الشيفرة السابقة** تبقى النقطةُ ضائعةً بلا محاولةٍ ثانية.
func TestWAORD030_SweepWritesALostLocation(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-030")
	armFeedback(t, h, true)
	h.Setting("delivery.custom_fee_min", "10")
	h.Setting("delivery.custom_fee_max", "1000")
	h.Setting("orders.whatsapp_location_followup_min", "0")
	ops := h.NewUser("operations")
	phone := uniqPhone()

	reqID, _, _ := waLocReq(t, h, ops.Token, phone, "")
	id := waCreated(t, waPost(t, h, ops.Token, waNoPointBody(w, phone, reqID, 600)))
	drv := h.driverOf(id)
	// **«وصل» ثُبّت والكتابةُ في الطلب سقطت** — قبل دقيقتين.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE recipient_location_requests
		   SET lat = $2, lng = $3, received_at = now() - interval '2 minutes'
		 WHERE id = $1`, reqID, w.md.dropLat, w.md.dropLng); err != nil {
		t.Fatal(err)
	}
	h.API.SweepWALocationFollowUp(ctxBG())
	lat, lng, known, fee := waOrderPoint(t, h, id)
	if !known || !nearF(lat, w.md.dropLat) || !nearF(lng, w.md.dropLng) || fee != 600 {
		t.Fatalf("**الجولةُ لم تكتب الموقعَ الواصل** (أو تبدّلت الأجرة): %v · %v,%v · %d", known, lat, lng, fee)
	}
	h.API.SweepWALocationFollowUp(ctxBG())
	if bodies := waNotes(t, h, drv.ID, id, "وصل موقع المستلم — صار على الخريطة"); len(bodies) != 1 {
		t.Fatalf("**السائقُ أُخبر %d مرّة — والمنتظَر مرّة**", len(bodies))
	}
}

// TestWAORD031_ListKeepsOpenOrdersFirst **«آخر طلبات الواتساب»: الجاريةُ أوّلاً** — طلبٌ مفتوحٌ خلفه
// خمسون منتهياً أحدثَ منه يبقى في القائمة، **فيُصحَّح عنوانُه منها** (لا تعديلَ في بطاقة الطلب).
//
// **وعلى الشيفرة السابقة** آخرُ عشرين بلا تفريق — فيغيب.
func TestWAORD031_ListKeepsOpenOrdersFirst(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-031")
	ops := h.NewUser("operations")
	open := waEngineCreate(t, h, w.md.admin.ID, w.typed())

	ctx := context.Background()
	for i := 0; i < 50; i++ {
		tx, err := h.Pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		c, err := h.Orders.CreateWhatsAppOrderIn(ctx, tx, w.md.admin.ID, w.typed())
		if err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("إنشاءُ طلبٍ %d: %v", i, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := h.Orders.CancelWhatsAppOrder(ctx, w.md.admin.ID, c.ID, "فحص القائمة", nil); err != nil {
			t.Fatalf("إلغاءُ طلبٍ %d: %v", i, err)
		}
	}
	r := h.GET(waURL, ops.Token)
	if r.Code != 200 {
		t.Fatalf("القائمة: %d / %s", r.Code, r.Err())
	}
	list, _ := r.JSON()["orders"].([]any)
	found := false
	for _, o := range list {
		if m, _ := o.(map[string]any); m != nil && m["id"] == open.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("**الطلبُ المفتوحُ غاب عن القائمة خلف المنتهية** (%d طلباً)", len(list))
	}
}
