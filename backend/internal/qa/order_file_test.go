package qa

// ══════════════════════════════════════════════════════════════════════
// **ملفُّ الطلب في سجلّ اللوحة** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
// ══════════════════════════════════════════════════════════════════════
//
// «لمّا افتح السجلّ لازم افهم لكلّ طلب: شو صار، كيف مشي، وحركة المال.» — ردُّ
// `GET /admin/orders/{id}` يحمل خصمَ العرض ومن تحمّله، ومحوَ العمولة بمن ولماذا، والشكاوى
// المربوطة، ومجاميعَ الدفتر، وصفةَ فاعل كلّ خطوة، وخطوةَ «جهّز المتجر». والقائمةُ تبحث باسم
// السائق وهاتفه والمتجر، وتفلتر بطريقة الدفع والمتأخّر والعمولة الممحيّة.

import (
	"net/url"
	"testing"
)

func TestORDERFILE_AdminDetailAndHistoryFilters(t *testing.T) {
	h := New(t)
	h.oldScaleMoney()
	treasury(t, h)
	waiverSettings(h)
	fin := h.NewUser("finance")
	admin := h.NewUser("admin")

	oid, drv, _, m := waiverOrder(t, h, "cash")

	// متى جهّز المتجر · خصمُ عرضٍ على الصنف · شكوى مربوطة · وتأخيرٌ في التسليم.
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET ready_at = created_at + interval '1 second',
		delivered_at = created_at + interval '5 hours' WHERE id = $1`, oid); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE order_items SET offer_cut = 50, offer_borne_by = 'merchant'
		WHERE order_id = $1`, oid); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO tickets (customer_id, order_id, subject)
		SELECT customer_id, id, 'لم يصلني كامل' FROM orders WHERE id = $1`, oid); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM tickets WHERE order_id = $1`, oid) })

	if r := h.POST("/api/v1/admin/orders/"+oid+"/waive-commission", fin.Token,
		map[string]any{"reason": "المتجر أخذ كامل"}); r.Code != 200 {
		t.Fatalf("المحو: %s", r)
	}

	r := h.GET("/api/v1/admin/orders/"+oid, admin.Token)
	if r.Code != 200 {
		t.Fatalf("ملفُّ الطلب: %s", r)
	}
	body := r.JSON()
	file, _ := body["file"].(map[string]any)
	if file == nil {
		t.Fatalf("**لا ملحقَ `file` في ردّ الإدارة**: %v", body)
	}

	wv, _ := file["commission_waiver"].(map[string]any)
	if wv == nil || num(wv["amount"]) != 100 || wv["reason"] != "المتجر أخذ كامل" || wv["by_name"] != fin.Name || wv["at"] == nil {
		t.Errorf("**محوُ العمولة في الملفّ**: %v", wv)
	}
	if num(body["commission_waived"]) != 100 {
		t.Errorf("العمولةُ الممحيّةُ على الطلب: %v", body["commission_waived"])
	}

	offers, _ := file["item_offers"].([]any)
	if len(offers) != 1 {
		t.Fatalf("**خصمُ العرض على الأصناف**: %v", offers)
	}
	if of, _ := offers[0].(map[string]any); num(of["offer_cut"]) != 50 || of["offer_borne_by"] != "merchant" || of["item_id"] == "" {
		t.Errorf("خصمُ العرض: %v", of)
	}

	cases, _ := file["cases"].([]any)
	if len(cases) != 1 {
		t.Fatalf("**الشكاوى المربوطة**: %v", cases)
	}
	if c, _ := cases[0].(map[string]any); c["kind"] != "ticket" || c["status"] != "open" || c["title"] != "لم يصلني كامل" || num(c["number"]) == 0 {
		t.Errorf("الشكوى: %v", c)
	}

	ledger, _ := file["ledger"].(map[string]any)
	if ledger == nil || num(ledger["lines"]) == 0 || num(ledger["driver"]) != 500 {
		t.Errorf("**مجاميعُ الدفتر**: %v", ledger)
	}
	if file["late"] != true || num(file["late_after_min"]) <= 0 {
		t.Errorf("**حكمُ التأخير**: late=%v بعد %v", file["late"], file["late_after_min"])
	}

	steps, _ := body["timeline"].([]any)
	var ready, byDriver bool
	for _, s := range steps {
		st, _ := s.(map[string]any)
		if st["actor_role"] == nil || st["actor_role"] == "" {
			t.Errorf("**خطوةٌ بلا صفة فاعل**: %v", st)
		}
		if st["status"] == "ready" && st["actor_role"] == "merchant" && st["actor"] == m.Name {
			ready = true
		}
		if st["status"] == "picked_up" && st["actor_role"] == "driver" && st["actor"] == drv.Name {
			byDriver = true
		}
	}
	if !ready || !byDriver {
		t.Errorf("**المسار: «جهّز المتجر» %v · «استلم السائق» بصفته %v**: %v", ready, byDriver, steps)
	}

	board, _ := body["board"].(map[string]any)
	if board == nil || num(board["cases"]) != 1 || board["late"] != true || board["transferred"] != false {
		t.Errorf("**علاماتُ الكرت**: %v", board)
	}

	// ── القائمة: البحثُ الأعرض والفلاترُ الثلاثة ──
	has := func(q url.Values) bool {
		t.Helper()
		q.Set("closed", "1")
		q.Set("per_page", "100")
		res := h.GET("/api/v1/admin/orders?"+q.Encode(), admin.Token)
		if res.Code != 200 {
			t.Fatalf("القائمة %v: %s", q, res)
		}
		list, _ := res.JSON()["orders"].([]any)
		for _, o := range list {
			if row, _ := o.(map[string]any); row["id"] == oid {
				return true
			}
		}
		return false
	}
	if !has(url.Values{"query": {drv.Phone}}) {
		t.Error("**البحثُ بهاتف السائق لا يجد طلبه**")
	}
	if !has(url.Values{"query": {m.Name}, "waived": {"1"}}) {
		t.Error("**البحثُ باسم المتجر مع «العمولة ممحيّة» لا يجد الطلب**")
	}
	if !has(url.Values{"pay": {"cash"}, "late": {"1"}, "waived": {"1"}, "merchant_id": {m.ID}}) {
		t.Error("**نقدٌ ومتأخّرٌ وممحيّ لا يجد الطلب**")
	}
	if has(url.Values{"pay": {"wallet"}, "merchant_id": {m.ID}}) {
		t.Error("**فلترُ المحفظة أظهر طلباً نقديّاً**")
	}
	if res := h.GET("/api/v1/admin/orders?closed=1&pay=card", admin.Token); res.Code != 400 {
		t.Errorf("طريقةُ دفعٍ لا تُعرف: %s", res)
	}

	// ── كرتُ السجلّ يحمل كلَّ شيء (٢٠٢٦-١٠-٣٠): البنودُ والمالُ ومن أنهاه — بلا نداءٍ لكلّ كرت ──
	card := func(path string) map[string]any {
		t.Helper()
		res := h.GET(path, admin.Token)
		if res.Code != 200 {
			t.Fatalf("القائمة: %s", res)
		}
		list, _ := res.JSON()["orders"].([]any)
		for _, o := range list {
			if row, _ := o.(map[string]any); row["id"] == oid {
				return row
			}
		}
		t.Fatalf("الطلبُ ليس في %s", path)
		return nil
	}
	row := card("/api/v1/admin/orders?closed=1&counts=1&per_page=100&merchant_id=" + m.ID)
	items, _ := row["items"].([]any)
	cb, _ := row["board"].(map[string]any)
	money, _ := cb["money"].(map[string]any)
	if len(items) == 0 {
		t.Errorf("**كرتُ السجلّ بلا بنود**: %v", row["items"])
	}
	if money == nil || num(money["lines"]) == 0 || num(money["offer_merchant"]) != 50*num(items[0].(map[string]any)["qty"]) ||
		num(money["offer_platform"]) != 0 || money["rep"] == nil || money["platform"] == nil {
		t.Errorf("**مالُ كرت السجلّ**: %v", money)
	}
	if cb["ended_by_name"] != drv.Name {
		t.Errorf("**من أنهى الطلب**: %v — يُنتظر %s", cb["ended_by_name"], drv.Name)
	}
	// **ولوحةُ العمل لا تحملها** — بلا `counts=1`.
	live := card("/api/v1/admin/orders?closed=1&per_page=100&merchant_id=" + m.ID)
	if lb, _ := live["board"].(map[string]any); lb["money"] != nil || lb["ended_by_name"] != nil {
		t.Errorf("**ملءُ السجلّ وصل لوحةَ العمل**: %v · %v", lb["money"], lb["ended_by_name"])
	}
}
