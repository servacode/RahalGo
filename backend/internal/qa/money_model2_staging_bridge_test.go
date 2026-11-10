package qa

// ══════════════════════════════════════════════════════════════════════
// **جسرُ شهادةِ النموذج ٢ على التجهيز** — يُشغَّل كما يشغّله السكربت (`scripts/qa/money_model2_staging.py`)
// ══════════════════════════════════════════════════════════════════════
//
// (موافقةُ المالك ٢٠٢٦-١١-١٠: «جرّب النموذج ٢ من أوّله لآخره على التجهيز».)
//
// **مِسنَدٌ بهويّة التجهيز** (`NewStaging`) فتُسجَّل جسورُ QA، **ثمّ السيناريوهاتُ نفسُها عبر الجسر
// وحده** — المفتاح، متجرُ QA، جلسةُ زبون QA، السُّلّم، أفعالُ المكتب، وقراءةُ المال — **وكلُّ سيناريو
// يُحكم عليه من قراءة `/qa/money` لا من القاعدة.** **وفي غير التجهيز كلُّه ٤٠٤.**

import (
	"net/http"
	"testing"
)

const m2bAPI = "/api/v1"

// m2bNum **رقمٌ من خريطة JSON** — والغائبُ صفر.
func m2bNum(m map[string]any, k string) int64 {
	if v, ok := m[k].(float64); ok {
		return int64(v)
	}
	return 0
}

func m2bMap(m map[string]any, k string) map[string]any {
	v, _ := m[k].(map[string]any)
	if v == nil {
		return map[string]any{}
	}
	return v
}

type m2bFx struct {
	h        *Harness
	itemID   string
	storeID  string
	custTok  string
	drvTok   string
	previous bool
}

// m2bSetup **ما يفعله السكربت أوّلاً** — المفتاحُ يُشغَّل (ويُحفظ السابق)، ومتجرُ QA، والجلستان.
func m2bSetup(t *testing.T) *m2bFx {
	t.Helper()
	h := NewStaging(t)
	h.Factory().Treasury()
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("orders.max_open_per_customer", "0")
	h.Setting("delivery.fee", "300")
	h.Setting("delivery.by_distance", "false")
	fx := &m2bFx{h: h}

	sw := h.POST(m2bAPI+"/qa/setting", "", map[string]any{"key": "finance.goods_outside", "value": true})
	if sw.Code != http.StatusOK {
		t.Fatalf("قلبُ المفتاح: %s", sw)
	}
	fx.previous, _ = sw.JSON()["previous"].(bool)
	t.Cleanup(func() {
		h.POST(m2bAPI+"/qa/setting", "", map[string]any{"key": "finance.goods_outside", "value": fx.previous})
	})

	st := h.POST(m2bAPI+"/qa/seed", "", map[string]any{"kind": "m2_store_setup"})
	if st.Code != http.StatusOK {
		t.Fatalf("متجرُ النموذج ٢: %s", st)
	}
	fx.itemID, _ = st.JSON()["item_id"].(string)
	fx.storeID, _ = st.JSON()["merchant_id"].(string)
	// **وإعادةُ النداء لا تُضاعف** — نفسُ المتجر ونفسُ الصنف.
	again := h.POST(m2bAPI+"/qa/seed", "", map[string]any{"kind": "m2_store_setup"})
	if id, _ := again.JSON()["item_id"].(string); again.Code != 200 || id != fx.itemID || again.JSON()["reused"] != true {
		t.Fatalf("إعادةُ متجر النموذج ٢ ضاعفت: %s", again)
	}

	// **ودوامُ سائق QA** — الإسنادُ لسائقٍ في دوامه (والسكربتُ يفعل مثله ويطفئه في الخاتمة).
	sh := h.POST(m2bAPI+"/qa/seed", "", map[string]any{"kind": "driver_shift", "value_bool": true})
	if sh.Code != http.StatusOK {
		t.Fatalf("دوامُ سائق QA: %s", sh)
	}
	prevMode, _ := sh.JSON()["previous_mode"].(string)
	t.Cleanup(func() {
		h.POST(m2bAPI+"/qa/seed", "", map[string]any{"kind": "driver_shift", "value_bool": false, "value_str": prevMode})
	})
	cs := h.POST(m2bAPI+"/qa/session", "", map[string]any{})
	fx.custTok = m2bToken(t, cs)
	ds := h.POST(m2bAPI+"/qa/driver-session", "", map[string]any{})
	fx.drvTok = m2bToken(t, ds)
	return fx
}

func m2bToken(t *testing.T, r Res) string {
	t.Helper()
	if r.Code != http.StatusOK {
		t.Fatalf("الجلسة: %s", r)
	}
	tok, _ := m2bMap(r.JSON(), "tokens")["access_token"].(string)
	if tok == "" {
		t.Fatalf("الجلسةُ بلا توكن: %s", r)
	}
	return tok
}

func (fx *m2bFx) order(t *testing.T, qty int, method, scope string) string {
	t.Helper()
	body := map[string]any{
		"items":        []map[string]any{{"menu_item_id": fx.itemID, "qty": qty}},
		"address_text": "الرقة — شارع الاختبار", "lat": 35.9506, "lng": 39.0094,
		"payment_method": method,
	}
	if scope != "" {
		body["wallet_scope"] = scope
	}
	r := fx.h.POSTKey(m2bAPI+"/orders", fx.custTok, uniq("m2b"), body)
	if r.Code >= 400 {
		t.Fatalf("إنشاءُ طلب زبون QA: %s", r)
	}
	id, _ := r.JSON()["id"].(string)
	return id
}

func (fx *m2bFx) advance(t *testing.T, oid, target string) {
	t.Helper()
	r := fx.h.POST(m2bAPI+"/qa/seed", "", map[string]any{"kind": "order_advance", "order_id": oid, "target": target})
	if r.Code != http.StatusOK || r.JSON()["status"] != target {
		t.Fatalf("السوقُ إلى %s: %s", target, r)
	}
}

func (fx *m2bFx) money(t *testing.T, oid string) map[string]any {
	t.Helper()
	r := fx.h.GET(m2bAPI+"/qa/money?order_id="+oid, "")
	if r.Code != http.StatusOK {
		t.Fatalf("قراءةُ المال: %s", r)
	}
	return r.JSON()
}

// m2bClean **لا خرقَ ثوابت يذكر الطلب** — ولا قيدَ بضاعة.
func m2bClean(t *testing.T, name string, m map[string]any) {
	t.Helper()
	inv := m2bMap(m, "invariants")
	if v, _ := inv["violations"].([]any); len(v) != 0 || m2bNum(inv, "checked") == 0 {
		t.Errorf("%s: خروقُ ثوابت تذكر الطلب %v (فُحص %d)", name, v, m2bNum(inv, "checked"))
	}
	if g := m2bNum(m2bMap(m, "ledger"), "goods_rows"); g != 0 {
		t.Errorf("%s: قيودُ بضاعة %d — النموذج ٢ لا يكتبها", name, g)
	}
	if adv := m2bNum(m2bMap(m, "office_cash"), "advance_return"); adv != 0 {
		t.Errorf("%s: سلفةُ درجٍ %d", name, adv)
	}
}

// ── السُّلّمُ كلُّه، والدفعُ للمتجر عند الاستلام، والقراءة ──────────────────
func TestM2Bridge_StandardCashAndWalletLadder(t *testing.T) {
	fx := m2bSetup(t)
	for _, c := range []struct{ name, method, scope string }{
		{"cash", "cash", ""},
		{"fee-from-wallet", "", "delivery"},
	} {
		if c.scope != "" {
			if r := fx.h.POST(m2bAPI+"/qa/seed", "", map[string]any{"kind": "wallet_fund", "value_int": 5000}); r.Code != 200 {
				t.Fatalf("شحنُ محفظة QA: %s", r)
			}
		}
		oid := fx.order(t, 1, c.method, c.scope)
		fx.advance(t, oid, "assigned")
		m := fx.money(t, oid)
		o := m2bMap(m, "order")
		if m2bNum(o, "money_model") != 2 {
			t.Fatalf("%s: نموذجُ الطلب %v", c.name, o["money_model"])
		}
		// **سقفُ النقد يعدّ مالَ المنصّة وحدَه** (B9): النقدُ ناقصَ حقّ المتجر ٩٠٠٠.
		if pc := m2bNum(m, "platform_cash"); pc != m2bNum(o, "cash_due")-9000 {
			t.Errorf("%s: مالُ المنصّة %d — يُنتظر النقد %d − 9000", c.name, pc, m2bNum(o, "cash_due"))
		}
		fx.advance(t, oid, "picked_up")
		m = fx.money(t, oid)
		rows, _ := m2bMap(m, "settlements")["rows"].([]any)
		if len(rows) != 1 {
			t.Fatalf("%s: صفوفُ التسوية %v", c.name, rows)
		}
		row, _ := rows[0].(map[string]any)
		if row["off_ledger"] != true || row["state"] != "cash_paid" || row["paid_via"] != "driver" ||
			m2bNum(row, "amount") != 9000 || row["has_ledger_tx"] != false {
			t.Errorf("%s: صفُّ التسوية %v — يُنتظر سجلٌّ بلا دفتر دفعه الكابتن 9000", c.name, row)
		}
		if te := m2bNum(m2bMap(m, "ledger"), "treasury_effect"); te != 0 && c.scope == "" {
			t.Errorf("%s: الخزينةُ تحرّكت عند الاستلام %d", c.name, te)
		}
		fx.advance(t, oid, "delivered")
		m = fx.money(t, oid)
		o, led := m2bMap(m, "order"), m2bMap(m, "ledger")
		box := m2bNum(m2bMap(m, "driver_box"), "collection")
		if box != m2bNum(o, "cash_due")-9000 {
			t.Errorf("%s: الصندوق %d — يُنتظر النقد %d − 9000", c.name, box, m2bNum(o, "cash_due"))
		}
		if want := m2bNum(o, "wallet_paid") + box - m2bNum(o, "driver_fee"); m2bNum(led, "platform_profit") != want {
			t.Errorf("%s: ربحُ المنصّة %d — يُنتظر %d", c.name, m2bNum(led, "platform_profit"), want)
		}
		if c.scope != "" && m2bNum(o, "wallet_paid") != m2bNum(o, "fee_paid") {
			t.Errorf("%s: المحفظةُ دفعت %d — يُنتظر الأجرةُ وحدَها %d", c.name, m2bNum(o, "wallet_paid"), m2bNum(o, "fee_paid"))
		}
		m2bClean(t, c.name, m)
	}
	fx.h.POST(m2bAPI+"/qa/seed", "", map[string]any{"kind": "wallet_drain"})
}

// ── الاسترجاعُ بعد التسليم: الأجرةُ وحدَها ───────────────────────────────
func TestM2Bridge_RefundAfterDeliveryFeeOnly(t *testing.T) {
	fx := m2bSetup(t)
	oid := fx.order(t, 1, "cash", "")
	fx.advance(t, oid, "delivered")
	r := fx.h.POST(m2bAPI+"/qa/orders/"+oid+"/transition", "", map[string]any{"to": "refunded", "note": "QA model2 refund"})
	if r.Code != 200 {
		t.Fatalf("الاسترجاع: %s", r)
	}
	m := fx.money(t, oid)
	led, o := m2bMap(m, "ledger"), m2bMap(m, "order")
	if got := m2bNum(m2bMap(led, "by_kind"), "refund"); got != m2bNum(o, "fee_paid") || got != 300 {
		t.Errorf("المردود %d — يُنتظر الأجرةُ المدفوعة %d", got, m2bNum(o, "fee_paid"))
	}
	if obl, _ := m["obligations"].([]any); len(obl) != 0 {
		t.Errorf("دينٌ بالاسترجاع: %v", obl)
	}
	m2bClean(t, "refund", m)
	fx.h.POST(m2bAPI+"/qa/seed", "", map[string]any{"kind": "wallet_drain"})
}

// ── الفشلُ بعد الاستلام: البضاعةُ ترجع للمتجر، ولا سلفةَ ولا دينَ ولا خسارة ──
func TestM2Bridge_FailureAfterPickupReturnsToStore(t *testing.T) {
	fx := m2bSetup(t)
	oid := fx.order(t, 1, "cash", "")
	fx.advance(t, oid, "at_dropoff")
	code := firstDoorReport(t, fx.h, &User{Token: fx.drvTok})
	if r := fx.h.POST(m2bAPI+"/driver/orders/"+oid+"/report", fx.drvTok, map[string]any{"code": code, "note": "QA"}); r.Code >= 400 {
		t.Fatalf("بلاغُ الباب: %s", r)
	}
	if r := fx.h.POST(m2bAPI+"/qa/orders/"+oid+"/door-resolution", "", map[string]any{
		"action": "return_to_office", "fault": "customer", "customer_action": "none", "return_to": "store", "note": "QA",
	}); r.Code >= 400 {
		t.Fatalf("قرارُ المكتب: %s", r)
	}
	if r := fx.h.POST(m2bAPI+"/driver/orders/"+oid+"/goods-handed", fx.drvTok, map[string]any{}); r.Code >= 400 {
		t.Fatalf("سلّمت البضاعة: %s", r)
	}
	if r := fx.h.POST(m2bAPI+"/qa/orders/"+oid+"/goods", "", map[string]any{"to": "merchant"}); r.Code >= 400 {
		t.Fatalf("حسمُ البضاعة: %s", r)
	}
	m := fx.money(t, oid)
	rows, _ := m2bMap(m, "settlements")["rows"].([]any)
	row, _ := rows[0].(map[string]any)
	if row["state"] != "goods_returned" || m2bNum(m2bMap(m, "ledger"), "platform_profit") < 0 {
		t.Errorf("رجعت للمتجر: %v · ربح %d", row, m2bNum(m2bMap(m, "ledger"), "platform_profit"))
	}
	if obl, _ := m["obligations"].([]any); len(obl) != 0 {
		t.Errorf("دينٌ عن بضاعةٍ رجعت: %v", obl)
	}
	m2bClean(t, "failure", m)

	// ── والمتجرُ رفض: «مصاري الكابتن عالقة» والمكتبُ يقرّر «على الزبون» — لا مالَ يتحرّك ──
	oid = fx.order(t, 1, "cash", "")
	fx.advance(t, oid, "at_dropoff")
	if r := fx.h.POST(m2bAPI+"/qa/orders/"+oid+"/door-resolution", "", map[string]any{
		"action": "return_to_office", "fault": "customer", "customer_action": "none", "return_to": "office", "note": "QA",
	}); r.Code >= 400 {
		t.Fatalf("قرارُ المكتب (للمكتب): %s", r)
	}
	if r := fx.h.POST(m2bAPI+"/driver/orders/"+oid+"/goods-handed", fx.drvTok, map[string]any{}); r.Code >= 400 {
		t.Fatalf("سلّمت البضاعة للمكتب: %s", r)
	}
	if r := fx.h.POST(m2bAPI+"/qa/orders/"+oid+"/goods", "", map[string]any{"to": "platform"}); r.Code >= 400 {
		t.Fatalf("حسمُ البضاعة للمكتب: %s", r)
	}
	if o := m2bMap(fx.money(t, oid), "order"); m2bNum(o, "stuck_goods_amount") != 9000 {
		t.Errorf("المصاري العالقة %d — يُنتظر 9000", m2bNum(o, "stuck_goods_amount"))
	}
	before := m2bNum(m2bMap(fx.money(t, oid), "ledger"), "treasury_effect")
	if r := fx.h.POST(m2bAPI+"/qa/orders/"+oid+"/stuck-goods", "", map[string]any{"decision": "customer"}); r.Code != 200 {
		t.Fatalf("قرارُ العالق: %s", r)
	}
	m = fx.money(t, oid)
	if after := m2bNum(m2bMap(m, "ledger"), "treasury_effect"); after != before {
		t.Errorf("«على الزبون» حرّك الخزينة %d ← %d", before, after)
	}
	m2bClean(t, "stuck-goods", m)
}

// ── طلبُ تعديل الكابتن بعد الاستلام وموافقةُ المكتب ───────────────────────
func TestM2Bridge_DriverEditRequestApproved(t *testing.T) {
	fx := m2bSetup(t)
	oid := fx.order(t, 2, "cash", "")
	fx.advance(t, oid, "picked_up")
	items, _ := fx.money(t, oid)["items"].([]any)
	line, _ := items[0].(map[string]any)
	req := fx.h.POSTKey(m2bAPI+"/driver/orders/"+oid+"/edit-request", fx.drvTok, uniq("m2er"), map[string]any{
		"lines": []map[string]any{{"item_id": line["id"], "action": "missing", "qty": 1}}, "note": "QA",
	})
	if req.Code != 200 {
		t.Fatalf("طلبُ التعديل: %s", req)
	}
	rid, _ := m2bMap(req.JSON(), "request")["id"].(string)
	// **ولا تسليمَ وطلبُ التعديل بانتظار المكتب** — حارسُ المعالِج نفسُه في الجسر.
	if r := fx.h.POST(m2bAPI+"/qa/seed", "", map[string]any{"kind": "order_advance", "order_id": oid, "target": "delivered"}); r.Code != http.StatusConflict {
		t.Errorf("سُلّم وطلبُ التعديل معلّق: %s", r)
	}
	if r := fx.h.POST(m2bAPI+"/qa/orders/"+oid+"/edit-request/approve", "", map[string]any{"request_id": rid}); r.Code != 200 {
		t.Fatalf("الموافقة: %s", r)
	}
	fx.advance(t, oid, "delivered")
	m := fx.money(t, oid)
	set := m2bMap(m, "settlements")
	if m2bNum(set, "adjustments") != 1 || m2bNum(set, "adjustments_with_ledger_tx") != 0 {
		t.Errorf("سطرُ التصحيح %v — يُنتظر واحدٌ بلا قيد", set)
	}
	if box := m2bNum(m2bMap(m, "driver_box"), "collection"); box != m2bNum(m2bMap(m, "order"), "cash_due")-9000 {
		t.Errorf("الصندوق %d — يُنتظر النقد بعد التعديل − 9000", box)
	}
	m2bClean(t, "edit", m)
}

// ── طلبُ الواتساب: العمولةُ عبر الصندوق ولا خصمَ من المتجر ────────────────
func TestM2Bridge_WhatsAppOrder(t *testing.T) {
	fx := m2bSetup(t)
	r := fx.h.POSTKey(m2bAPI+"/qa/whatsapp-orders", "", uniq("m2wa"), map[string]any{
		"store_id": fx.storeID, "recipient_phone": "+963900555001", "recipient_name": "QA",
		"address_text": "الرقة — شارع الاختبار", "lat": 35.9506, "lng": 39.0094, "fee": 600,
	})
	if r.Code >= 400 {
		t.Fatalf("طلبُ الواتساب: %s", r)
	}
	oid, _ := r.JSON()["id"].(string)
	if oid == "" {
		oid, _ = m2bMap(r.JSON(), "order")["id"].(string)
	}
	fx.advance(t, oid, "at_pickup")
	if f := fx.h.POST(m2bAPI+"/driver/orders/"+oid+"/fetch-amount", fx.drvTok, map[string]any{"amount": 2400}); f.Code != 200 {
		t.Fatalf("كتابةُ الثمن: %s", f)
	}
	fx.advance(t, oid, "delivered")
	m := fx.money(t, oid)
	o := m2bMap(m, "order")
	if m2bNum(o, "platform_commission") != 240 || m2bNum(m2bMap(m, "driver_box"), "collection") != 840 {
		t.Errorf("الواتساب: عمولة %d صندوق %d — يُنتظر 240 و840", m2bNum(o, "platform_commission"),
			m2bNum(m2bMap(m, "driver_box"), "collection"))
	}
	if got := m2bNum(m2bMap(m2bMap(m, "ledger"), "by_party"), "merchant"); got != 0 {
		t.Errorf("محفظةُ المتجر تحرّكت %d", got)
	}
	m2bClean(t, "whatsapp", m)

	// **ورقمٌ غيرُ رقم QA يُردّ** — لا رسالةَ لغير QA.
	bad := fx.h.POSTKey(m2bAPI+"/qa/whatsapp-orders", "", uniq("m2wa"), map[string]any{
		"store_id": fx.storeID, "recipient_phone": "+963933000111", "lat": 35.9506, "lng": 39.0094, "fee": 600,
	})
	if bad.Code != http.StatusForbidden {
		t.Errorf("طلبُ واتساب لرقمٍ غيرِ QA مرّ: %s", bad)
	}
}

// ── وطلبُ غيرِ QA لا يُمَسّ ولا يُقرأ ────────────────────────────────────
func TestM2Bridge_RefusesNonQAOrder(t *testing.T) {
	fx := m2bSetup(t)
	other := fx.h.Customer()
	item := fx.h.NewItem(1000)
	made := fx.h.POSTKey(m2bAPI+"/orders", other.Token, uniq("m2x"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("طلبٌ لغير QA: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	for _, r := range []Res{
		fx.h.GET(m2bAPI+"/qa/money?order_id="+oid, ""),
		fx.h.POST(m2bAPI+"/qa/orders/"+oid+"/transition", "", map[string]any{"to": "cancelled", "note": "x"}),
		fx.h.POST(m2bAPI+"/qa/seed", "", map[string]any{"kind": "order_advance", "order_id": oid, "target": "accepted"}),
	} {
		if r.Code != http.StatusForbidden {
			t.Errorf("طلبُ غيرِ QA قُبل: %s", r)
		}
	}
	if st := fx.h.statusOf(oid); st != "pending" {
		t.Errorf("طلبُ غيرِ QA تحرّك إلى %s", st)
	}
}

// ── وفي غير التجهيز: كلُّ مسارٍ جديدٍ ٤٠٤ ─────────────────────────────────
func TestM2Bridge_NotRegisteredOutsideStaging(t *testing.T) {
	h := New(t)
	for _, c := range []struct{ method, path string }{
		{"GET", "/qa/money?order_id=00000000-0000-0000-0000-000000000000"},
		{"POST", "/qa/orders/00000000-0000-0000-0000-000000000000/transition"},
		{"POST", "/qa/orders/00000000-0000-0000-0000-000000000000/door-resolution"},
		{"POST", "/qa/orders/00000000-0000-0000-0000-000000000000/goods"},
		{"POST", "/qa/orders/00000000-0000-0000-0000-000000000000/stuck-goods"},
		{"POST", "/qa/orders/00000000-0000-0000-0000-000000000000/edit"},
		{"POST", "/qa/orders/00000000-0000-0000-0000-000000000000/edit-request/approve"},
		{"POST", "/qa/orders/00000000-0000-0000-0000-000000000000/proof-exception"},
		{"POST", "/qa/orders/00000000-0000-0000-0000-000000000000/partner"},
		{"POST", "/qa/whatsapp-orders"},
		{"POST", "/qa/drivers/00000000-0000-0000-0000-000000000000/settle"},
		{"POST", "/qa/setting"},
		{"POST", "/qa/seed"},
	} {
		if r := h.Call(c.method, m2bAPI+c.path, "", map[string]any{"key": "finance.goods_outside", "value": true, "kind": "m2_store_setup"}, nil); r.Code != http.StatusNotFound {
			t.Errorf("%s %s خارجَ التجهيز ردّ %d — يُنتظر 404", c.method, c.path, r.Code)
		}
	}
	if h.Settings.GetBool(t.Context(), "finance.goods_outside") {
		t.Error("المفتاحُ انقلب خارجَ التجهيز")
	}
}

// m2bCustom **طلبُ مشترياتٍ لزبون QA حتّى الإسناد** — وسياسةُ الأجرة «يحدّدها الكابتن» تُستعاد بعده.
func (fx *m2bFx) custom(t *testing.T) string {
	t.Helper()
	pol := fx.h.POST(m2bAPI+"/qa/custom-fee-policy", "", map[string]any{"source": "driver_defined", "fee": 0, "driver_may_change": false})
	if pol.Code != 200 {
		t.Fatalf("سياسةُ الأجرة: %s", pol)
	}
	r := fx.h.POSTKey(m2bAPI+"/orders/custom", fx.custTok, uniq("m2c"), customBody("QA model2 purchase"))
	if r.Code >= 400 {
		t.Fatalf("طلبُ المشتريات: %s", r)
	}
	oid, _ := r.JSON()["id"].(string)
	fx.advance(t, oid, "assigned")
	return oid
}

// m2bConfirm **الزبونُ يؤكّد العرضَ من بابه الحقيقيّ** — بمبلغ العرض ونسخته من قراءة المال.
func (fx *m2bFx) confirm(t *testing.T, oid string) {
	t.Helper()
	o := m2bMap(fx.money(t, oid), "order")
	r := fx.h.POST(m2bAPI+"/orders/"+oid+"/confirm-quote", fx.custTok, map[string]any{
		"payment_method": "cash", "expected_total": m2bNum(o, "quote_total"), "expected_quote_version": m2bNum(o, "quote_version"),
	})
	if r.Code != 200 {
		t.Fatalf("تأكيدُ العرض: %s", r)
	}
}

// ── المشترياتُ بالنموذج ٢: لا قيدَ بضاعةٍ ولا سلفة، والعرضُ المؤكَّدُ لا يُكتب فوقه ──
func TestM2Bridge_CustomAndUSD(t *testing.T) {
	fx := m2bSetup(t)
	oid := fx.custom(t)
	if r := fx.h.POST(m2bAPI+"/qa/custom-agree", "", map[string]any{"goods": 300, "fee": 150}); r.Code != 200 {
		t.Fatalf("الاتّفاق: %s", r)
	}
	fx.confirm(t, oid)
	fx.advance(t, oid, "delivered")
	m := fx.money(t, oid)
	o, led := m2bMap(m, "order"), m2bMap(m, "ledger")
	if m2bNum(o, "custom_goods_amount") != 300 || m2bNum(o, "custom_fee") != 150 {
		t.Errorf("العرضُ المؤكَّد كُتب فوقه: %v", o)
	}
	box := m2bNum(m2bMap(m, "driver_box"), "collection")
	if want := m2bNum(o, "wallet_paid") + box - m2bNum(o, "driver_fee"); m2bNum(led, "platform_profit") != want {
		t.Errorf("مشتريات: ربح %d — يُنتظر %d", m2bNum(led, "platform_profit"), want)
	}
	m2bClean(t, "custom", m)

	// ── بالدولار ومربوطٌ بمتجر QA: العمولةُ من معادل الدولار، والسقفُ لا يعدّه ──
	oid = fx.custom(t)
	if r := fx.h.POST(m2bAPI+"/qa/orders/"+oid+"/partner", "", map[string]any{"merchant_id": fx.storeID}); r.Code != 200 {
		t.Fatalf("الربط: %s", r)
	}
	if r := fx.h.POST(m2bAPI+"/qa/custom-agree", "", map[string]any{"goods": 0, "fee": 150, "goods_usd_cents": 1250}); r.Code != 200 {
		t.Fatalf("الاتّفاقُ بالدولار: %s", r)
	}
	fx.confirm(t, oid)
	m = fx.money(t, oid)
	if pc, cd := m2bNum(m, "platform_cash"), m2bNum(m2bMap(m, "order"), "cash_due"); pc != cd {
		t.Errorf("الدولار: مالُ المنصّة %d — يُنتظر النقدُ بالليرة %d (الدولارُ لا يُعدّ)", pc, cd)
	}
	fx.advance(t, oid, "delivered")
	m = fx.money(t, oid)
	o = m2bMap(m, "order")
	if m2bNum(o, "custom_goods_usd") != 1250 || m2bNum(o, "goods_usd_syp") != 1250 || m2bNum(o, "platform_commission") != 125 {
		t.Errorf("الدولار: %v — يُنتظر 1250 سنتاً ومعادلُ 1250 وعمولةُ 125", o)
	}
	m2bClean(t, "usd", m)
}
