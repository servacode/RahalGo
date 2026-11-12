package qa

// **تطبيقُ المتجر يرنّ حين يصله طلب — في وضع المنصّة أيضاً** (تجربةُ المالك ٢٠٢٦-١١-١٢).
//
// في وضع المنصّة المكتبُ يقبل ويحوّل والمتجرُ يتفرّج، **وكان تطبيقُه صامتاً**. فصار:
// إشعارٌ عند إنشاء الطلب («طلب جديد قادم إليك»)، **وإشعارٌ ثانٍ لحظةَ يحوّله المكتبُ إليه**
// («عندك طلب جديد — جهّزه») — مرّةً واحدة، لا مع «أعد الإرسال».

import (
	"net/http"
	"strconv"
	"testing"
)

func hoCount(t *testing.T, h *Harness, userID, title string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM notifications WHERE user_id = $1::uuid AND title = $2`, userID, title).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestHO01_PlatformModeMerchantIsNotifiedOnCreateAndOnHandoff(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("platform.orders_mode", `"platform"`)
	h.Setting("orders.notify_merchant", "true")
	h.Setting("orders.auto_transfer", "false")
	h.Setting("orders.auto_accept_min", "0")
	h.Setting("drivers.assignment_mode", `"queue"`)
	treasury(t, h)
	f := h.Factory()
	m := f.Merchant()
	item := h.NewItemPriced(m, 1000, 1000)
	admin := h.NewUser("admin")
	cust := h.Customer()

	created := h.POSTKey("/api/v1/orders", cust.Token, uniq("ho"), orderBody(item, 2))
	if created.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الطلب: %s", created)
	}
	oid, _ := created.JSON()["id"].(string)

	if n := hoCount(t, h, m.Owner.ID, "طلب جديد قادم إليك"); n != 1 {
		t.Fatalf("**إشعارُ الإنشاء للمتجر**: %d (يُنتظر 1)", n)
	}
	if n := hoCount(t, h, m.Owner.ID, "عندك طلب جديد — جهّزه"); n != 0 {
		t.Fatalf("**لا إشعارَ تحويلٍ قبل التحويل**: %d", n)
	}

	if r := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "accepted"}); r.Code >= 400 {
		t.Fatalf("قبولُ المكتب: %s", r)
	}
	send := func() {
		if r := h.POST("/api/v1/admin/orders/"+oid+"/whatsapp", admin.Token,
			map[string]any{"channel": "manual"}); r.Code != http.StatusOK {
			t.Fatalf("التحويلُ للمتجر: %s", r)
		}
	}
	send()
	if n := hoCount(t, h, m.Owner.ID, "عندك طلب جديد — جهّزه"); n != 1 {
		t.Fatalf("**التحويلُ يُرنّ تطبيقَ المتجر**: %d (يُنتظر 1)", n)
	}
	// «أعد الإرسال» لا يرنّ ثانيةً.
	send()
	if n := hoCount(t, h, m.Owner.ID, "عندك طلب جديد — جهّزه"); n != 1 {
		t.Fatalf("**إعادةُ الإرسال لا تكرّر الإشعار**: %d", n)
	}
	// والإشعارُ يحمل رقمَ الطلب.
	var body string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT body FROM notifications WHERE user_id = $1::uuid AND title = $2`,
		m.Owner.ID, "عندك طلب جديد — جهّزه").Scan(&body); err != nil {
		t.Fatal(err)
	}
	var number int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT number FROM orders WHERE id = $1::uuid`, oid).Scan(&number); err != nil {
		t.Fatal(err)
	}
	if want := "#" + strconv.FormatInt(number, 10); len(body) < len(want) || body[:len(want)] != want {
		t.Fatalf("**نصُّ الإشعار يبدأ برقم الطلب**: %q", body)
	}
}
