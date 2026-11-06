package qa

// ══════════════════════════════════════════════════════════════════════
// **نموذجُ الواتساب المبسَّط** — قراراتُ المالك ٢٠٢٦-١١-٠٦
// ══════════════════════════════════════════════════════════════════════
//
//	١ متجرُنا: اسمُه من القائمة وحدَه — لا رقمَ ولا عنوانَ للمتجر
//	٢ متجرٌ غيرُ تابع: اسمُه مكتوباً وحدَه — لا رقمَ ولا موقعَ ولا خريطة
//	٣ لا اسمَ للمستلِم («زبون») — رقمُه وحدَه، **والموقعُ والعنوانُ يطلبهما البوتُ ويملآنهما**
//	٤ ثمنُ البضاعة يكتبه السائقُ بتطبيقه، **والزبونُ يُخبَر برسالة** — بلا موافقة
//
// **والأرقامُ كلُّها وهميّةٌ** (`uniqPhone`) — لا رقمَ حقيقيّاً في فحص.

import (
	"net/http"
	"strings"
	"testing"
)

// waSimpleFx **منطقةُ الفحص أوّلُ المناطق** — فمركزُها مرساةُ طلبٍ لا نقطةَ فيه (`waCityAnchor`).
func waSimpleFx(t *testing.T, h *Harness, name string) waFx {
	t.Helper()
	w := newWAFx(t, h, name)
	h.Setting("delivery.custom_fee_min", "10")
	h.Setting("delivery.custom_fee_max", "5000")
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE delivery_zones SET sort_order = (SELECT COALESCE(min(sort_order), 0) - 1 FROM delivery_zones)
		 WHERE id = $1`, w.md.z.ID); err != nil {
		t.Fatal(err)
	}
	return w
}

// waSimpleBody **ما يرسله النموذجُ الجديد** — اسمُ المتجر (أو معرّفُه) ورقمُ الزبون والأجرة لا غير.
func waSimpleBody(phone string, fee int) map[string]any {
	return map[string]any{"store_name": "فلافل الأمير", "recipient_phone": phone, "fee": fee}
}

// TestWAORD040_StoreNameAndPhoneOnly **متجرٌ باسمه ورقمُ الزبون — ينزل الطلب** والبوتُ يطلب الموقعَ
// والعنوانَ في رسالةٍ واحدة، **واسمُ المستلِم «زبون».** ومتجرُنا بمعرّفه وحدَه كذلك.
//
// **وعلى الشيفرة القديمة**: `wa_store_required` (المتجرُ غيرُ التابع يلزمه موقع) ثمّ `validation`
// (لا اسمَ ولا عنوان)، **ولا رسالةَ تصل الزبون.**
func TestWAORD040_StoreNameAndPhoneOnly(t *testing.T) {
	h := New(t)
	w := waSimpleFx(t, h, "منطقةُ WA-040")
	n := armFeedback(t, h, true)
	ops := h.NewUser("operations")

	// **بلا أجرةٍ تُردّ** — لا مسافةَ تُحسب، والموظّفُ يكتبها.
	phone := uniqPhone()
	b := waSimpleBody(phone, 600)
	delete(b, "fee")
	if r := waPost(t, h, ops.Token, b); r.Err() != "wa_fee_required" {
		t.Fatalf("**بلا أجرة**: %d / %s", r.Code, r.Err())
	}
	// **وعرضُ السعر بلا متجرٍ معروفِ النقطة**: الأساسُ اقتراحاً والأجرةُ مطلوبة.
	if q := h.GET(waURL+"/quote", ops.Token); q.Code != http.StatusOK || q.JSON()["fee_required"] != true {
		t.Fatalf("**عرضُ السعر بلا نقطة**: %d / %s", q.Code, q.Body)
	}

	id := waCreated(t, waPost(t, h, ops.Token, waSimpleBody(phone, 600)))
	var name, addr, pickName string
	var pickPhone, pickAddr *string
	var pickKnown, dropKnown bool
	var fee, cash int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT recipient_name, address_text, pickup_name, pickup_phone, pickup_address,
		       pickup_override IS NOT NULL, dropoff_known, delivery_fee, cash_due
		  FROM orders WHERE id = $1`, id).
		Scan(&name, &addr, &pickName, &pickPhone, &pickAddr, &pickKnown, &dropKnown, &fee, &cash); err != nil {
		t.Fatal(err)
	}
	if name != "زبون" || addr != "" || pickName != "فلافل الأمير" || pickPhone != nil || pickAddr != nil ||
		pickKnown || dropKnown || fee != 600 || cash != 600 {
		t.Fatalf("**الصفّ**: اسم %q · عنوان %q · متجر %q/%v/%v · نقطةُ متجر %v · نقطةُ زبون %v · %d/%d",
			name, addr, pickName, pickPhone, pickAddr, pickKnown, dropKnown, fee, cash)
	}
	// **وطلبُ الموقع والعنوان كُتب مربوطاً ووصلت رسالتُه** — نصُّ المالك ومعه سطرُ العنوان.
	var linked int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM recipient_location_requests
		WHERE order_id = $1 AND mode = 'whatsapp' AND phone = $2 AND whatsapp_sent`, id, phone).Scan(&linked)
	if linked != 1 {
		t.Fatalf("**طلبُ الموقع لم يُكتب تلقائيّاً مربوطاً بالطلب**: %d", linked)
	}
	if msgs := n.to(phone); len(msgs) != 1 || msgs[0].text != waLocTextWA() {
		t.Fatalf("**رسالةُ الموقع والعنوان**: %+v", msgs)
	}
	// **والسائقُ يراه** — بلا نقطة استلام، واسمُ المتجر وحدَه.
	drv := h.driverOf(id)
	got := driverOrderOf(t, h, drv.Token, id)
	if got["merchant_name"] != "فلافل الأمير" {
		t.Fatalf("**السائقُ لا يرى اسمَ المتجر**: %v", got["merchant_name"])
	}

	// ── ومتجرُنا بمعرّفه وحدَه ─────────────────────────────────────────
	p2 := uniqPhone()
	b2 := map[string]any{"store_id": w.md.fx.M.ID, "recipient_phone": p2, "fee": 700}
	id2 := waCreated(t, waPost(t, h, ops.Token, b2))
	var pm *string
	var pk bool
	_ = h.Pool.QueryRow(ctxBG(), `SELECT pickup_merchant_id::text, pickup_override IS NOT NULL
		FROM orders WHERE id = $1`, id2).Scan(&pm, &pk)
	if pm == nil || *pm != w.md.fx.M.ID || !pk {
		t.Fatalf("**متجرُنا بمعرّفه**: %v / %v", pm, pk)
	}
	if msgs := n.to(p2); len(msgs) != 1 {
		t.Fatalf("**رسالةُ متجرنا**: %+v", msgs)
	}
}

// TestWAORD041_TextReplyIsTheAddress **ردُّ الزبون النصّيُّ عنوانُ الطلب** — مرّةً واحدة، ويصل السائق،
// **والعنوانُ الذي كتبه الموظّفُ قبله في سجلّ الأحداث**، والمتابعةُ لا تُبعث بعده.
//
// **وعلى الشيفرة القديمة لا يُبنى**: لا `HandleWAAddressText` (البوتُ لا يلتقط النصّ).
func TestWAORD041_TextReplyIsTheAddress(t *testing.T) {
	h := New(t)
	_ = waSimpleFx(t, h, "منطقةُ WA-041")
	n := armFeedback(t, h, true)
	ops := h.NewUser("operations")
	h.Setting("orders.whatsapp_location_followup_min", "3")

	phone := uniqPhone()
	id := waCreated(t, waPost(t, h, ops.Token, waSimpleBody(phone, 600)))
	drv := h.driverOf(id)
	// **الموظّفُ كتب عنواناً من الواتساب قبل ردّ الزبون.**
	if r := h.POST(waURL+"/"+id+"/address", ops.Token, map[string]any{"address_text": "الرميلة"}); r.Code != http.StatusOK {
		t.Fatalf("تعديلُ العنوان: %s", r)
	}

	// **ونصٌّ لا يشبه عنواناً لا يُلتقط.**
	if reply := h.API.HandleWAAddressText(ctxBG(), phone, "👍"); reply != "" {
		t.Fatalf("**إشارةٌ عُدّت عنواناً**: %q", reply)
	}
	addr := "حي الدرعية، شارع المنصور، جانب فرن الأمل"
	if reply := h.API.HandleWAAddressText(ctxBG(), phone, addr); reply == "" {
		t.Fatal("**لا ردَّ على عنوانٍ لطلبٍ مفتوح**")
	}
	var got string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT address_text FROM orders WHERE id = $1`, id).Scan(&got)
	if got != addr {
		t.Fatalf("**العنوانُ لم يُكتب في الطلب**: %q", got)
	}
	var old string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT details->>'old' FROM audit_log
		WHERE action = 'ops.whatsapp_order_address_bot' AND entity_id = $1`, id).Scan(&old)
	if old != "الرميلة" {
		t.Fatalf("**عنوانُ الموظّف القديم ضاع**: %q", old)
	}
	// **وثانيةٌ لا تُبدّله.**
	if reply := h.API.HandleWAAddressText(ctxBG(), phone, "حي الثكنة"); reply != "" {
		t.Fatalf("**ردٌّ ثانٍ التُقط**: %q", reply)
	}
	_ = h.Pool.QueryRow(ctxBG(), `SELECT address_text FROM orders WHERE id = $1`, id).Scan(&got)
	if got != addr {
		t.Fatalf("**الردُّ الثاني بدّل العنوان**: %q", got)
	}
	// **والسائقُ يراه ويُخبَر بلا رقم الزبون.**
	if d := driverOrderOf(t, h, drv.Token, id); !strings.HasPrefix(fmtAny(d["address_text"]), addr) {
		t.Fatalf("**السائقُ لا يرى العنوان**: %v", d["address_text"])
	}
	for _, b := range waNotes(t, h, drv.ID, id, "تغيّر عنوان المستلم") {
		if strings.Contains(b, strings.TrimPrefix(phone, "+963")) {
			t.Fatalf("**رقمُ الزبون وصل السائق**: %q", b)
		}
	}
	// **والمتابعةُ لا تُبعث بعد العنوان** — ولو مضت الدقائقُ الثلاث بلا موقع.
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE recipient_location_requests
		SET created_at = now() - interval '5 minutes' WHERE order_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	before := len(n.to(phone))
	h.API.SweepWALocationFollowUp(ctxBG())
	if after := len(n.to(phone)); after != before {
		t.Fatalf("**المتابعةُ بُعثت بعد وصول العنوان**: %d ← %d", before, after)
	}
}

// TestWAORD042_LocationReplyStillFillsThePoint **الردُّ بالموقع يملأ نقطةَ الطلب كما كان** — والطلبُ
// بلا نقطة متجرٍ يُرسى بعدها على باب الزبون.
func TestWAORD042_LocationReplyStillFillsThePoint(t *testing.T) {
	h := New(t)
	w := waSimpleFx(t, h, "منطقةُ WA-042")
	armFeedback(t, h, true)
	ops := h.NewUser("operations")
	phone := uniqPhone()
	id := waCreated(t, waPost(t, h, ops.Token, waSimpleBody(phone, 600)))
	if reply := h.API.HandleWALocation(ctxBG(), phone, w.md.dropLat, w.md.dropLng, 6); reply == "" {
		t.Fatal("**لا ردَّ على موقعٍ لطلبٍ مفتوح**")
	}
	lat, lng, known, fee := waOrderPoint(t, h, id)
	if !known || fee != 600 || !nearF(lat, w.md.dropLat) || !nearF(lng, w.md.dropLng) {
		t.Fatalf("**الموقعُ لم يُكتب**: %v · %d · %v,%v", known, fee, lat, lng)
	}
	// **وبعد الموقع لا يُلتقط نصٌّ عنواناً** (قرارُ المالك ٢٠٢٦-١١-٠٦: «إذا بعث الموقع ما بيلزم العنوان») —
	// «شكراً» بعد الموقع ليست عنواناً.
	var before string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT address_text FROM orders WHERE id = $1`, id).Scan(&before)
	if reply := h.API.HandleWAAddressText(ctxBG(), phone, "شكراً كتير"); reply != "" {
		t.Fatalf("**نصٌّ بعد الموقع التُقط عنواناً**: %q", reply)
	}
	var after string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT address_text FROM orders WHERE id = $1`, id).Scan(&after)
	if after != before {
		t.Fatalf("**نصٌّ بعد الموقع بدّل العنوان**: %q ← %q", before, after)
	}
}

// TestWAORD043_DriverTypesGoodsCustomerInformed **السائقُ يكتب ثمنَ الطلب ⇒ يصل الإدارةَ، والزبونَ
// رسالةٌ واحدةٌ بالمبالغ** — بلا موافقة، **والدفترُ يقيّد الأجرةَ وحدَها** وثوابتُه خضراء.
//
// **وعلى الشيفرة القديمة**: `fetch_amount_stage` (البابُ لـ«إحضار» وحدَه) ولا رسالة.
func TestWAORD043_DriverTypesGoodsCustomerInformed(t *testing.T) {
	h := New(t)
	_ = waSimpleFx(t, h, "منطقةُ WA-043")
	n := armFeedback(t, h, true)
	base := mdaBaseline(t, h)
	ops := h.NewUser("operations")
	phone := uniqPhone()
	id := waCreated(t, waPost(t, h, ops.Token, waSimpleBody(phone, 600)))
	drv := h.driverOf(id)
	sentBefore := len(n.to(phone))

	// **قبل مكان الاستلام لا يُكتب.**
	if r := h.POST("/api/v1/driver/orders/"+id+"/fetch-amount", drv.Token, map[string]any{"amount": 2400}); r.Err() != "fetch_amount_stage" {
		t.Fatalf("**ثمنٌ قبل مكان الاستلام**: %d / %s", r.Code, r.Err())
	}
	if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": "at_pickup"}); r.Code >= 400 {
		t.Fatalf("وصلت المتجر: %s", r)
	}
	r := h.POST("/api/v1/driver/orders/"+id+"/fetch-amount", drv.Token, map[string]any{"amount": 2400})
	if r.Code != http.StatusOK {
		t.Fatalf("**السائقُ لم يكتب الثمن**: %d / %s", r.Code, r.Err())
	}
	var amount, collect *int64
	var status string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT fetch_amount, COALESCE(fetch_amount_status, ''), collect_amount
		FROM orders WHERE id = $1`, id).Scan(&amount, &status, &collect)
	if amount == nil || *amount != 2400 || status != "approved" || collect == nil || *collect != 2400 {
		t.Fatalf("**الثمنُ في الطلب**: %v · %q · %v", amount, status, collect)
	}
	// **ويصل الإدارةَ** — في قائمة طلبات الواتساب.
	list := h.GET(waURL, ops.Token)
	found := false
	rows, _ := list.JSON()["orders"].([]any)
	for _, x := range rows {
		if m, _ := x.(map[string]any); m != nil && m["id"] == id {
			found = m["fetch_amount"] == float64(2400) && m["collect_amount"] == float64(2400)
		}
	}
	if !found {
		t.Fatalf("**الإدارةُ لا ترى الثمن**: %s", list.Body)
	}
	// **ورسالةٌ واحدةٌ للزبون بالأرقام الصحيحة.**
	msgs := n.to(phone)
	if len(msgs) != sentBefore+1 {
		t.Fatalf("**رسائلُ الزبون**: %d والمنتظَر %d", len(msgs), sentBefore+1)
	}
	want := "ثمن طلبك 2,400 ل.س + أجرة التوصيل 600 ل.س = المجموع 3,000 ل.س"
	if msgs[len(msgs)-1].text != want {
		t.Fatalf("**نصُّ رسالة المبالغ**: %q — والمنتظَر %q", msgs[len(msgs)-1].text, want)
	}

	// **والتسليمُ يمضي كالعادة — والدفترُ يقيّد الأجرةَ وحدَها.**
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s: %s", to, r)
		}
	}
	h.authorizeProofExempt(id)
	if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": "delivered"}); r.Code >= 400 {
		t.Fatalf("التسليم: %s", r)
	}
	if got := waCashOf(t, h, id); got != 600 {
		t.Fatalf("**نقدُ الصندوق %d والمنتظَر 600 — البضاعةُ لا تدخله**", got)
	}
	led := waLedgerOf(t, h, id)
	if led["platform_profit"] != 60 || led["driver_earning"] != 540 || len(led) != 2 {
		t.Fatalf("**الدفتر**: %v — المنتظَر أجرُ السائق 540 وحصّةُ المنصّة 60 لا غير", led)
	}
	mdaNoNewViolations(t, h, base)
}

func fmtAny(v any) string { s, _ := v.(string); return s }
