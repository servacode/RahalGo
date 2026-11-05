package qa

// ══════════════════════════════════════════════════════════════════════
// **مسارُ الزبون لا يكرّر خطوةً لم تتبدّل** — خلل PD-5 (٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// صنفٌ غيرُ متوفّر وتحويلُ المتجر وتعديلُ الطلب **تكتب في السجلّ سطراً حالُه قبلُ وبعدُ واحدة**
// (`from = to`) — أثرٌ يقرؤه المكتب لا انتقال. **وكان مسارُ الزبون يعرضه خطوةً ثانية**:
// «قُبل» ثمّ «قُبل» ثمّ «قُبل». **والمكتبُ يبقى يرى السطورَ كلَّها في مساره الكامل.**

import (
	"net/http"
	"testing"
)

// timelineCount **كم خطوةً بهذه الحال في مسار الطلب.**
func timelineCount(t *testing.T, body map[string]any, status string) int {
	t.Helper()
	steps, _ := body["timeline"].([]any)
	n := 0
	for _, s := range steps {
		if st, _ := s.(map[string]any); st["status"] == status {
			n++
		}
	}
	return n
}

// TestCustomerTimelineSkipsSameStatusEvents **T0e — طلبٌ مقبول، ثمّ صنفٌ غيرُ متوفّر وتحويلٌ
// لمتجرٍ آخر وهو «مقبول» ⇒ الزبونُ يرى «قُبل» مرّةً واحدة، والمكتبُ يرى السطورَ الثلاثة.**
func TestCustomerTimelineSkipsSameStatusEvents(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	// **المكتبُ يقبل والطلبُ يبقى «مقبولاً»** حتّى يُبلَّغ المتجر — لا يتقدّم وحدَه.
	h.Setting("platform.orders_mode", `"platform"`)
	h.Setting("orders.notify_merchant", "true")
	h.Setting("orders.auto_transfer", "false")
	treasury(t, h)
	staff := h.NewUser("operations")
	admin := h.NewUser("admin")
	item := h.NewItem(1000)
	alt := h.NewItem(1000)
	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("pd5"), orderBody(item, 2))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	const a = "/api/v1/admin/orders/"
	if r := h.POST(a+oid+"/transition", staff.Token,
		map[string]any{"to": "accepted", "note": "قبولٌ من العمليّات"}); r.Code >= 400 {
		t.Fatalf("القبول: %s", r)
	}
	if st := h.statusOf(oid); st != "accepted" {
		t.Fatalf("الحالُ بعد القبول %q — يُنتظر accepted", st)
	}
	var line string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT id::text FROM order_items WHERE order_id = $1::uuid`, oid).Scan(&line); err != nil {
		t.Fatal(err)
	}
	// **سطرٌ «مقبول ← مقبول»**: واحدٌ من الاثنين غيرُ متوفّر.
	if r := h.POST(a+oid+"/items/"+line+"/unavailable", staff.Token,
		map[string]any{"qty": 1}); r.Code != http.StatusOK {
		t.Fatalf("صنفٌ غيرُ متوفّر: %s", r)
	}
	// **وسطرٌ ثانٍ**: تحويلٌ لمتجرٍ آخر والطلبُ «مقبول».
	if r := h.POST(a+oid+"/transfer", staff.Token, map[string]any{
		"merchant_id": alt.MerchantID, "note": "المتجر اعتذر",
		"items": []map[string]any{{"order_item_id": line, "menu_item_id": alt.ID}},
	}); r.Code != http.StatusOK {
		t.Fatalf("التحويل: %s", r)
	}

	mine := h.GET("/api/v1/my/orders/"+oid, cust.Token)
	if mine.Code != http.StatusOK {
		t.Fatalf("طلبُ الزبون: %s", mine)
	}
	if n := timelineCount(t, mine.JSON(), "accepted"); n != 1 {
		t.Errorf("«قُبل» في مسار الزبون %d مرّات — يُنتظر مرّةً واحدة: %v", n, mine.JSON()["timeline"])
	}

	// **والمكتبُ يرى الأثرَ كلَّه** — القبولُ وسطرا «مقبول ← مقبول».
	office := h.GET(a+oid, admin.Token)
	if office.Code != http.StatusOK {
		t.Fatalf("طلبُ المكتب: %s", office)
	}
	if n := timelineCount(t, office.JSON(), "accepted"); n != 3 {
		t.Errorf("«مقبول» في مسار المكتب %d — يُنتظر 3 (القبولُ والحذفُ والتحويل)", n)
	}
}
