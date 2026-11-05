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
	// **والطلبُ إذا انتهى لا يبقى منه عند السائق ما يدلّ على صاحبه** (قرارُ المالك
	// ٢٠٢٦-١١-٠٥، البند ٢): لا اسمَه ولا عنوانَه ولا نقاطَه ولا ما كتبه — **فالسجلُّ
	// لا يصير قائمةَ زبائن.** رقمُ الطلب والمتجرُ والمبالغُ والمسافةُ والتاريخُ تبقى.
	for _, leak := range []string{"customer_phone", "customer_id", "lat", "lng",
		"customer_name", "address_text", "notes", "custom_request", "holder_name", "holder_phone",
		"entrance_lat", "entrance_lng", "customer_live", "proof_url", "zone_name", "items",
		"amanah_pickup_lat", "amanah_pickup_lng"} {
		if v, ok := row[leak]; ok {
			t.Errorf("**السجلُّ يحمل `%s` = %v** — لا يصل إلى السائق", leak, v)
		}
	}
	// **وما تعرضه شاشةُ السجلّ باقٍ.**
	for _, need := range []string{"number", "status", "merchant_name", "total", "created_at", "leg_m", "can_rate_merchant", "merchant_rated"} {
		if _, ok := row[need]; !ok {
			t.Errorf("**غاب `%s`** — وشاشةُ السجلّ تعرضه", need)
		}
	}
}
