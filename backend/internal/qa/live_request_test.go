package qa

import "testing"

// TestLiveRequest_DriverAsksCustomerAgain **«اطلب موقع الزبون المباشر»** (قرارُ المالك
// ٢٠٢٦-١٠-٢١): يصل الزبونَ إشعارٌ ويتبدّل `live_requested_at` فيُسأل ثانيةً — **وثلاثُ
// مرّاتٍ لا أكثر، ودقيقتان بينها، ولغير سائقه لا.**
func TestLiveRequest_DriverAsksCustomerAgain(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	id := newCustomOrder(t, h, cust)
	drv := h.NewUser("driver")
	assignDriver(t, h, id, drv.ID)
	path := "/api/v1/driver/orders/" + id + "/request-live-location"

	res := h.POST(path, drv.Token, map[string]any{})
	if res.Code != 200 {
		t.Fatalf("الطلبُ الأوّل: %s", res)
	}
	if left, _ := res.JSON()["remaining"].(float64); left != 2 {
		t.Fatalf("المتبقّي = %v — والمتوقَّع ٢", res.JSON()["remaining"])
	}
	mine := h.GET("/api/v1/my/orders/"+id, cust.Token)
	if v, _ := mine.JSON()["live_requested_at"].(string); v == "" {
		t.Fatalf("**الزبونُ لا يرى أنّ السائقَ طلب** — فلا يُسأل ثانيةً: %s", mine)
	}
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*)::int FROM notifications WHERE user_id = $1::uuid
		AND title LIKE 'الكابتن طالب موقعك المباشر%'`, cust.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("إشعارُ الزبون = %d — والمتوقَّع ١", n)
	}

	if again := h.POST(path, drv.Token, map[string]any{}); again.Code != 429 || again.Err() != "live_request_too_soon" {
		t.Fatalf("طلبٌ ثانٍ فوراً يجب أن يُردّ بالانتظار: %s", again)
	}
	back := func() {
		if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET live_requested_at = now() - interval '3 minutes'
			WHERE id = $1::uuid`, id); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		back()
		if r := h.POST(path, drv.Token, map[string]any{}); r.Code != 200 {
			t.Fatalf("الطلبُ %d: %s", i+2, r)
		}
	}
	back()
	if over := h.POST(path, drv.Token, map[string]any{}); over.Code != 429 || over.Err() != "live_request_limit" {
		t.Fatalf("الرابعُ يجب أن يُردّ بالحدّ: %s", over)
	}
	other := h.NewUser("driver")
	if r := h.POST(path, other.Token, map[string]any{}); r.Code != 403 {
		t.Fatalf("سائقٌ غيرُه يجب أن يُردّ: %s", r)
	}
}
