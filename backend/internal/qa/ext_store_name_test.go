package qa

// ══════════════════════════════════════════════════════════════════════
// **اسمُ المتجر في «من متجر غير تابع لنا»** — قرارُ المالك ٢٠٢٦-١١-١٢
// ══════════════════════════════════════════════════════════════════════
//
// «طلب المشتريات لازم ينذكر اسم المتجر ليعرف مين المتجر الي لازم نشتري منه.» الاسمُ إلزاميّ عند
// القلب، وهاتفُ المتجر وعنوانُه ونقطتُه اختياريّة؛ **وتصل السائقَ في حقول الاستلام نفسِها** التي
// يقرؤها في طلب الواتساب (`merchant_name` · `merchant_phone` · `pickup_address` · `pickup_lat/lng`).
// وطلبُ الواتساب من متجرٍ غير تابعٍ يحمل الأربعةَ للسائق كذلك.

import (
	"net/http"
	"testing"
)

// TestEXTNAME_001_StoreNameRequired **بلا اسمٍ يُردّ 400 ولا يُقلب الطلب.**
func TestEXTNAME_001_StoreNameRequired(t *testing.T) {
	h := New(t)
	extStoreSetup(h)
	ops := h.NewUser("operations")
	m := h.Factory().Merchant(Commission(10))
	oid, _ := extStoreOrder(t, h, "accepted", orderBody(h.NewItemFor(m, 1000), 1), m)

	for name, body := range map[string]map[string]any{
		"بلا اسم":     {"reason": "سبب"},
		"اسمٌ فارغ":   {"reason": "سبب", "store_name": "   "},
		"نقطةٌ ناقصة": {"reason": "سبب", "store_name": "متجر", "lat": 35.95},
		"نقطةُ الصفر": {"reason": "سبب", "store_name": "متجر", "lat": 0.0, "lng": 0.0},
	} {
		if res := extStorePostBody(h, ops.Token, oid, body); res.Code != http.StatusBadRequest {
			t.Errorf("%s: المنتظَر 400 — جاء %s", name, res)
		}
	}
	var kind string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT kind FROM orders WHERE id = $1::uuid`, oid).Scan(&kind)
	if kind == "custom" {
		t.Errorf("قُلب الطلبُ رغم الرفض: %s", kind)
	}
}

// TestEXTNAME_002_DriverSeesStoreName **الاسمُ وحدَه يصل السائقَ بلا نقطة.**
func TestEXTNAME_002_DriverSeesStoreName(t *testing.T) {
	h := New(t)
	extStoreSetup(h)
	ops := h.NewUser("operations")
	m := h.Factory().Merchant(Commission(10))
	oid, drv := extStoreOrder(t, h, "assigned", orderBody(h.NewItemFor(m, 1000), 1), m)

	res := extStorePostBody(h, ops.Token, oid, map[string]any{"reason": "سبب", "store_name": "  فلافل الأمير  "})
	if res.Code != http.StatusOK {
		t.Fatalf("القلب: %s", res)
	}
	got := driverOrderOf(t, h, drv.Token, oid)
	if got["merchant_name"] != "فلافل الأمير" || got["kind"] != "custom" {
		t.Fatalf("السائقُ لا يرى اسمَ المتجر: %v / %v", got["merchant_name"], got["kind"])
	}
	if got["pickup_lat"] != nil {
		t.Errorf("نقطةُ استلامٍ لم تُحدَّد: %v", got["pickup_lat"])
	}
}

// TestEXTNAME_003_DriverSeesStorePoint **الهاتفُ والعنوانُ والنقطةُ تصل السائقَ.**
func TestEXTNAME_003_DriverSeesStorePoint(t *testing.T) {
	h := New(t)
	extStoreSetup(h)
	ops := h.NewUser("operations")
	m := h.Factory().Merchant(Commission(10))
	oid, drv := extStoreOrder(t, h, "assigned", orderBody(h.NewItemFor(m, 1000), 1), m)

	res := extStorePostBody(h, ops.Token, oid, map[string]any{
		"reason": "سبب", "store_name": "فلافل الأمير", "store_phone": "0933444555",
		"store_address": "شارع تل أبيض", "lat": 35.9520, "lng": 39.0100,
	})
	if res.Code != http.StatusOK {
		t.Fatalf("القلب: %s", res)
	}
	got := driverOrderOf(t, h, drv.Token, oid)
	if got["merchant_name"] != "فلافل الأمير" || got["merchant_phone"] != "0933444555" ||
		got["pickup_address"] != "شارع تل أبيض" {
		t.Fatalf("اسمٌ/هاتفٌ/عنوانٌ: %v / %v / %v", got["merchant_name"], got["merchant_phone"], got["pickup_address"])
	}
	if !near(got["pickup_lat"], 35.9520) || !near(got["pickup_lng"], 39.0100) ||
		!near(got["nav_lat"], 35.9520) || !near(got["nav_lng"], 39.0100) {
		t.Fatalf("نقطةُ الاستلام ليست نقطةَ المتجر: %v,%v · nav %v,%v",
			got["pickup_lat"], got["pickup_lng"], got["nav_lat"], got["nav_lng"])
	}
}

// TestWAORD_DriverSeesNonPartnerShop **طلبُ الواتساب من متجرٍ غير تابع: الاسمُ والهاتفُ والعنوانُ
// والنقطةُ كلُّها في حمولة السائق** (قرارُ المالك ٢٠٢٦-١١-١٢).
func TestWAORD_DriverSeesNonPartnerShop(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-ظاهر")
	ops := h.NewUser("operations")
	id := waCreated(t, waPost(t, h, ops.Token, w.body()))

	drv := h.NewUser("driver")
	assignDriver(t, h, id, drv.ID)
	got := driverOrderOf(t, h, drv.Token, id)
	if got["merchant_name"] != "فلافل الأمير" || got["merchant_phone"] != "+963933444555" ||
		got["pickup_address"] != "شارع تل أبيض" {
		t.Fatalf("اسمٌ/هاتفٌ/عنوانٌ: %v / %v / %v", got["merchant_name"], got["merchant_phone"], got["pickup_address"])
	}
	if !near(got["pickup_lat"], w.storeLat) || !near(got["pickup_lng"], w.storeLng) ||
		!near(got["nav_lat"], w.storeLat) || !near(got["nav_lng"], w.storeLng) {
		t.Fatalf("نقطةُ المتجر: %v,%v · nav %v,%v", got["pickup_lat"], got["pickup_lng"], got["nav_lat"], got["nav_lng"])
	}
}
