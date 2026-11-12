package qa

// ══════════════════════════════════════════════════════════════════════
// **طلب تعديل من الكابتن — من الأبواب** (قرارُ المالك ٢٠٢٦-١١-١٠)
// ══════════════════════════════════════════════════════════════════════
//
// الكابتن يبعت (بمفتاح محاولة — الإعادة ما بتبعت طلبين)، والمكتب يشوف بالمعاينة ويوافق أو يرفض،
// والمجموع يتبع. ومال ما بعد الاستلام وقدرته مقيسين باختبارات الحزمة (`orders/order_edit_request_test.go`).
// وطلب الواتساب: الكابتن بيصحّح الثمن بعد الاستلام والعمولة على آخر رقم.

import (
	"net/http"
	"strings"
	"testing"
)

func TestEDITREQ01_DriverAsksOfficeDecides(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)
	ops := h.NewUser("operations")
	var total0 int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT total FROM orders WHERE id = $1`, oid).Scan(&total0); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/driver/orders/" + oid + "/edit-request"
	body := map[string]any{"adds": []map[string]any{{"name": "خبز", "qty": 2, "price": 250}}, "note": "الزبون طلب"}
	key := uniq("er")
	r := h.POSTKey(path, drv.Token, key, body)
	if r.Code != http.StatusOK {
		t.Fatalf("طلب التعديل: %s", r)
	}
	req, _ := r.JSON()["request"].(map[string]any)
	reqID, _ := req["id"].(string)
	if reqID == "" || req["status"] != "pending" || int64(num(req["new_total"])) != total0+500 {
		t.Fatalf("الرد: %s", r)
	}
	// **والإعادة بنفس المفتاح ما بتبعت طلب تاني.**
	if again := h.POSTKey(path, drv.Token, key, body); again.Code != http.StatusOK {
		t.Fatalf("الإعادة: %s", again)
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_edit_requests WHERE order_id = $1`, oid).Scan(&n)
	if n != 1 {
		t.Fatalf("انبعت %d طلب — يُنتظر واحد", n)
	}
	// **وكابتن غيره ما بيبعت.**
	if other := h.POST(path, h.NewUser("driver").Token, body); other.Code < 400 {
		t.Fatalf("كابتن غيره بعت: %s", other)
	}
	// **والكابتن بيشوف حالته.**
	if g := h.GET(path, drv.Token); g.Code != http.StatusOK || !strings.Contains(string(g.Body), `"pending"`) {
		t.Fatalf("الكابتن ما بيشوف طلبه: %s", g)
	}
	// **والمكتب بيشوف المعاينة.**
	admin := "/api/v1/admin/orders/" + oid + "/edit-request"
	g := h.GET(admin, ops.Token)
	pend, _ := g.JSON()["request"].(map[string]any)
	prev, _ := pend["preview"].(map[string]any)
	if g.Code != http.StatusOK || prev == nil || int64(num(prev["new_total"])) != total0+500 {
		t.Fatalf("معاينة المكتب: %s", g)
	}
	// **والزبون ما بيوافق ولا بيرفض** — الباب للمكتب.
	if c := h.POST(admin+"/approve", h.Customer().Token, map[string]any{"request_id": reqID}); c.Code < 400 {
		t.Fatalf("زبون وافق: %s", c)
	}
	if a := h.POST(admin+"/approve", ops.Token, map[string]any{"request_id": reqID}); a.Code != http.StatusOK {
		t.Fatalf("الموافقة: %s", a)
	}
	var total int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT total FROM orders WHERE id = $1`, oid).Scan(&total)
	if total != total0+500 {
		t.Fatalf("المجموع %d — يُنتظر %d", total, total0+500)
	}
	var audits int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM audit_log WHERE entity_id = $1
		AND action IN ('driver.order_edit_request', 'ops.order_edit_request_approved')`, oid).Scan(&audits)
	if audits != 2 {
		t.Fatalf("سطور التدقيق %d — يُنتظر 2", audits)
	}

	// ── طلب تاني بيترفض بسبب ──
	r = h.POSTKey(path, drv.Token, uniq("er"), map[string]any{"adds": []map[string]any{{"name": "ماء", "qty": 1, "price": 100}}})
	req, _ = r.JSON()["request"].(map[string]any)
	reqID, _ = req["id"].(string)
	if x := h.POST(admin+"/reject", ops.Token, map[string]any{"request_id": reqID}); x.Code != http.StatusBadRequest || x.Err() != "edit_request_reject_reason" {
		t.Fatalf("رفض بلا سبب: %s", x)
	}
	if x := h.POST(admin+"/reject", ops.Token, map[string]any{"request_id": reqID, "reason": "مو صحيح"}); x.Code != http.StatusOK {
		t.Fatalf("الرفض: %s", x)
	}
	_ = h.Pool.QueryRow(ctxBG(), `SELECT total FROM orders WHERE id = $1`, oid).Scan(&total)
	if total != total0+500 {
		t.Fatalf("الرفض غيّر المجموع: %d", total)
	}
	if g := h.GET(path, drv.Token); !strings.Contains(string(g.Body), `"rejected"`) || !strings.Contains(string(g.Body), "مو صحيح") {
		t.Fatalf("الكابتن ما شاف الرفض: %s", g)
	}
}

// TestWACOMM_CorrectedAfterPickup **الكابتن بيصحّح ثمن طلب الواتساب بعد الاستلام** (قرارُ المالك
// ٢٠٢٦-١١-١٠) ⇒ الزبون بتوصله رسالة «تعدّل المبلغ»، **والعمولة على آخر رقم** (١٠٪ من ٣٠٠٠ = ٣٠٠).
//
// **وعلى الشيفرة القديمة يسقط**: الكتابة بعد «استلمت» بالطريق بتنرد `fetch_amount_stage`.
func TestWACOMM_CorrectedAfterPickup(t *testing.T) {
	h := New(t)
	c := newWAComm(t, h, "منطقةُ WA-COMM-FIX", 1000)
	n := armFeedback(t, h, true)
	phone := uniqPhone()
	id := c.ourStore(t, h, map[string]any{"recipient_phone": phone})
	drv := h.driverOf(id)
	step := func(to string) {
		t.Helper()
		if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("الانتقال إلى %s: %s", to, r)
		}
	}
	write := func(amount int64) {
		t.Helper()
		if r := h.POST("/api/v1/driver/orders/"+id+"/fetch-amount", drv.Token, map[string]any{"amount": amount}); r.Code != http.StatusOK {
			t.Fatalf("كتابة %d: %d / %s", amount, r.Code, r.Err())
		}
	}
	step("at_pickup")
	write(2400)
	step("picked_up")
	step("on_the_way")
	write(3000)
	msgs := n.to(phone)
	if len(msgs) == 0 || !strings.HasPrefix(msgs[len(msgs)-1].text, "تعدّل المبلغ: ") ||
		!strings.Contains(msgs[len(msgs)-1].text, "3,000") {
		t.Fatalf("رسالة التصحيح للزبون: %+v", msgs)
	}
	step("at_dropoff")
	write(3200)
	h.authorizeProofExempt(id)
	step("delivered")
	if pc := waPlatformCommission(t, h, id); pc != 320 {
		t.Fatalf("العمولة %d — يُنتظر 320 من آخر رقم", pc)
	}
	if got := waBalance(t, h, c.ownerID); got != 680 {
		t.Fatalf("محفظة المتجر %d — يُنتظر 680", got)
	}
	var notes int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_events WHERE order_id = $1 AND note LIKE '%عدّل ثمن الطلب%'`, id).Scan(&notes)
	if notes != 2 {
		t.Fatalf("سطور التصحيح بالسجل %d — يُنتظر 2", notes)
	}
	// **وبعد التسليم ما في كتابة.**
	if r := h.POST("/api/v1/driver/orders/"+id+"/fetch-amount", drv.Token, map[string]any{"amount": 1}); r.Code < 400 {
		t.Fatalf("كتابة بعد التسليم: %s", r)
	}
}

// TestEDITREQ_USDPriceConvertedByStoreRate **صنفٌ مضافٌ بالدولار** (قرارُ المالك ٢٠٢٦-١١-١٢) — يُحوَّل
// بسعر صرف المتجر لأقرب ٥ ليرات، **والحسابُ بالليرة**: ٢٫٥$ × ١٢٠ = ٣٠٠ ل.س للقطعة.
func TestEDITREQ_USDPriceConvertedByStoreRate(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)
	var total0 int64
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE merchants SET exchange_rate = 120 WHERE id = (SELECT merchant_id FROM orders WHERE id = $1)`, oid); err != nil {
		t.Fatal(err)
	}
	if err := h.Pool.QueryRow(ctxBG(), `SELECT total FROM orders WHERE id = $1`, oid).Scan(&total0); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/driver/orders/" + oid + "/edit-request"
	r := h.POSTKey(path, drv.Token, uniq("er"), map[string]any{
		"adds": []map[string]any{{"name": "شاحن", "qty": 2, "price": 1, "price_usd": 250}},
	})
	if r.Code != http.StatusOK {
		t.Fatalf("طلب التعديل: %s", r)
	}
	req, _ := r.JSON()["request"].(map[string]any)
	if int64(num(req["new_total"])) != total0+600 {
		t.Fatalf("**الدولارُ لم يُحوَّل بسعر المتجر**: المجموع %v — يُنتظر %d (٢ × ٣٠٠): %s", req["new_total"], total0+600, r)
	}
	// **وسعرٌ بالدولار سالبٌ يُرفض.**
	if bad := h.POSTKey(path, drv.Token, uniq("er"), map[string]any{
		"adds": []map[string]any{{"name": "x", "qty": 1, "price": 1, "price_usd": -5}},
	}); bad.Code != http.StatusBadRequest {
		t.Fatalf("دولارٌ سالبٌ قُبل: %s", bad)
	}
}
