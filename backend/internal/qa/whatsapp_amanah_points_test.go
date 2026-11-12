package qa

// ══════════════════════════════════════════════════════════════════════
// **أمانةُ الواتساب بنقطتين** — قرارُ المالك ٢٠٢٦-١١-١٢
// ══════════════════════════════════════════════════════════════════════
//
// «الاستلام من» و«التسليم لـ»: بطاقتان على اللوحة، وعلامتان «أ» و«ب» على الخريطة،
// و«استلم الأمانة من» / «سلّمها لـ» عند السائق. **عرضٌ وحدَه** — لا مالَ ولا حالَ يتغيّر.

import (
	"testing"
	"time"
)

// TestWAAMN006_TwoPointsInDashboardAndMap **طلبُ اللوحة وطبقةُ الخريطة يحملان علامةَ الأمانة** —
// ونقطةُ الاستلام غائبةٌ ما دام موقعُ صاحبها لم يصل، والتسليمُ «ناطرين» ما دام المستلِمُ لم يردّ.
// والسائقُ يعرف أنّها أمانة.
//
// **وعلى الشيفرة القديمة**: لا `wa_amanah` في ردّ الطلب ولا في الخريطة ولا عند السائق.
func TestWAAMN006_TwoPointsInDashboardAndMap(t *testing.T) {
	h := New(t)
	w := waSimpleFx(t, h, "منطقةُ WA-AMN-006")
	_ = armFeedback(t, h, true)
	ops := h.NewUser("operations")
	admin := h.NewUser("admin")
	_ = w.md.f.Driver(OnShift(), LocationAt(w.storeLat, w.storeLng, time.Now()))
	sender, recipient := uniqPhone(), uniqPhone()
	id := waCreated(t, waPost(t, h, ops.Token, waAmanahBody(sender, recipient, 600)))

	adminOrder := func(oid string) map[string]any {
		t.Helper()
		r := h.GET("/api/v1/admin/orders/"+oid, admin.Token)
		if r.Code != 200 {
			t.Fatalf("طلبُ اللوحة: %s", r)
		}
		m := r.JSON()
		if inner, ok := m["order"].(map[string]any); ok {
			m = inner
		}
		return m
	}
	mapOrderOf := func() map[string]any {
		t.Helper()
		r := h.GET("/api/v1/admin/ops-map/orders", admin.Token)
		if r.Code != 200 {
			t.Fatalf("طبقةُ الطلبات: %s", r)
		}
		rows, _ := r.JSON()["orders"].([]any)
		for _, x := range rows {
			if m, _ := x.(map[string]any); m != nil && m["id"] == id {
				return m
			}
		}
		t.Fatalf("الأمانةُ %s ليست على الخريطة: %s", id, r.Body)
		return nil
	}

	// ── ١ · قبل الموقعين: علامةُ الأمانة، ولا نقطةَ استلام، والتسليمُ ناطر ─────────
	o := adminOrder(id)
	if o["wa_amanah"] != true || o["pickup_lat"] != nil || o["dropoff_known"] != false ||
		o["merchant_name"] != "أبو خالد" || o["merchant_phone"] != sender || o["customer_phone"] != recipient {
		t.Fatalf("**طلبُ اللوحة قبل الموقعين**: أمانة %v · استلام %v · تسليم %v · %v/%v · %v",
			o["wa_amanah"], o["pickup_lat"], o["dropoff_known"], o["merchant_name"], o["merchant_phone"], o["customer_phone"])
	}
	mo := mapOrderOf()
	if mo["wa_amanah"] != true || mo["pick_lat"] != nil || mo["drop_pending"] != true {
		t.Fatalf("**الخريطةُ قبل الموقعين**: %v", mo)
	}

	// ── ٢ · بعد الموقعين: نقطتان، ولا «ناطرين» ──────────────────────────────
	h.API.HandleWALocation(ctxBG(), recipient, w.md.dropLat, w.md.dropLng, 6)
	h.API.HandleWALocation(ctxBG(), sender, w.storeLat, w.storeLng, 6)
	o = adminOrder(id)
	pl, _ := o["pickup_lat"].(float64)
	dl, _ := o["lat"].(float64)
	if o["wa_amanah"] != true || !nearF(pl, w.storeLat) || o["dropoff_known"] != true || !nearF(dl, w.md.dropLat) {
		t.Fatalf("**طلبُ اللوحة بعد الموقعين**: أمانة %v · استلام %v · تسليم %v/%v",
			o["wa_amanah"], o["pickup_lat"], o["dropoff_known"], o["lat"])
	}
	mo = mapOrderOf()
	ml, _ := mo["pick_lat"].(float64)
	if mo["wa_amanah"] != true || !nearF(ml, w.storeLat) || mo["drop_pending"] != nil {
		t.Fatalf("**الخريطةُ بعد الموقعين**: %v", mo)
	}

	// ── ٣ · والسائقُ يعرف أنّها أمانة ──────────────────────────────────────
	d := h.driverOf(id)
	if got := driverOrderOf(t, h, d.Token, id); got["wa_amanah"] != true {
		t.Fatalf("**السائقُ لا يعرف أنّها أمانة**: %v", got)
	}

	// ── ٤ · وطلبُ «من متجر» بلا علامة ─────────────────────────────────────
	plain := waCreated(t, waPost(t, h, ops.Token, waSimpleBody(uniqPhone(), 600)))
	if v, has := adminOrder(plain)["wa_amanah"]; has {
		t.Fatalf("**طلبُ المتجر يحمل علامةَ الأمانة**: %v", v)
	}
}
