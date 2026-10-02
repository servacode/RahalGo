package qa

// ══════════════════════════════════════════════════════════════════════
// **سجلُّ السائق لا يحمل رقمَ الزبون ولا موضعَه** (فحصُ دورة السائق ٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس على التجهيز**: `GET /driver/orders/history` يردّ صفَّ الطلب كاملاً كما يراه
// المدير — **وفيه `customer_phone` ومعرّفُ الزبون وإحداثيّاتُ بابه.** وقرارُ المالك
// (٢٠٢٦-٠٨-٠٩): «ولا رقمَ زبونٍ يصل إلى سائق». **والطلبُ انتهى فلا حاجةَ لموضعه أصلاً.**

import (
	"testing"
)

func TestDRV_HISTORY_NoCustomerPhoneOrLocation(t *testing.T) {
	h := New(t)
	fx := newTenureFx(t, h)
	acceptOrder(t, h, fx.DrvA, fx.Order)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET status = 'delivered', delivered_at = now(), closed_at = now() WHERE id = $1::uuid`,
		fx.Order); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}

	r := h.GET("/api/v1/driver/orders/history", fx.DrvA.Token)
	if r.Code != 200 {
		t.Fatalf("**السجلُّ رُدّ**: %s", r)
	}
	rows, _ := r.JSON()["orders"].([]any)
	var row map[string]any
	for _, x := range rows {
		m, _ := x.(map[string]any)
		if m["id"] == fx.Order {
			row = m
		}
	}
	if row == nil {
		t.Fatalf("**الطلبُ ليس في سجلّه**: %s", r)
	}
	for _, leak := range []string{"customer_phone", "customer_id", "lat", "lng"} {
		if v, ok := row[leak]; ok {
			t.Errorf("**السجلُّ يحمل `%s` = %v** — لا يصل إلى السائق", leak, v)
		}
	}
	// **وما تعرضه شاشةُ السجلّ باقٍ.**
	for _, need := range []string{"number", "status", "merchant_name", "customer_name", "address_text", "total", "can_rate_merchant"} {
		if _, ok := row[need]; !ok {
			t.Errorf("**غاب `%s`** — وشاشةُ السجلّ تعرضه", need)
		}
	}
}
