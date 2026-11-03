package qa

import (
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **«ردّ بموقعك»** (قرارُ المالك ٢٠٢٦-١١-٠٣) — المستلِمُ يردّ على رسالة البوت بموقعه
// من مشبك واتساب، فيمتلئ الطلبُ **كما يمتلئ بالرابط تماماً**.
// ══════════════════════════════════════════════════════════════════════

// waLocStatus يقرأ حالَ الطلب من نداء الزبون نفسِه الذي يسأله التطبيق.
func waLocStatus(t *testing.T, h *Harness, token, id string) map[string]any {
	t.Helper()
	r := h.GET("/api/v1/my/location-requests/"+id, token)
	if r.Code != http.StatusOK {
		t.Fatalf("حالُ طلب الموقع: %s", r)
	}
	return r.JSON()
}

// الردُّ بالموقع يملأ الطلبَ كالرابط: الحالُ والنقطةُ والدقّةُ، وينعكس في طلب الأمانة
// نقطةَ استلامٍ يراها السائق. ومرّتان: الثانيةُ لا طلبَ مفتوحاً فلا ردّ.
func TestWALocationReplyFillsLikeToken(t *testing.T) {
	h := New(t)
	cust := h.Customer()

	// الطريقُ المرجعيّ: الرابط (دقّته 8).
	refID, tok := requestHolderLocation(t, h, cust.Token, "0944555601")
	shareLocation(t, h, tok)
	ref := waLocStatus(t, h, cust.Token, refID)

	// طريقُ الواتساب: رقمٌ آخر، والطلبُ مرتبطٌ بأمانةٍ أُرسلت قبل وصول الموقع.
	waID, _ := requestHolderLocation(t, h, cust.Token, "0944555602")
	b := amanahBody("أمانة: حقيبة")
	b["holder_phone"] = "0944555602"
	b["location_request_id"] = waID
	created := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("wal"), b)
	if created.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الأمانة: %s", created)
	}
	orderID, _ := created.JSON()["id"].(string)
	drv := h.NewUser("driver")
	assignDriver(t, h, orderID, drv.ID)

	reply := h.API.HandleWALocation(ctxBG(), "+963944555602", amPLat, amPLng, 8.4)
	if reply == "" {
		t.Fatal("لا ردَّ على موقعٍ لطلبٍ مفتوح")
	}
	got := waLocStatus(t, h, cust.Token, waID)
	if got["status"] != "received" || got["status"] != ref["status"] {
		t.Fatalf("الحال %v — والمرجعُ %v", got["status"], ref["status"])
	}
	if !near(got["lat"], amPLat) || !near(got["lng"], amPLng) || got["accuracy_m"] != ref["accuracy_m"] {
		t.Fatalf("النقطةُ/الدقّةُ تخالف الرابط: %v / %v / %v مقابل %v", got["lat"], got["lng"], got["accuracy_m"], ref["accuracy_m"])
	}
	if got["received_at"] == nil {
		t.Fatal("received_at فارغ")
	}
	pick := driverOrderOf(t, h, drv.Token, orderID)
	if !near(pick["pickup_lat"], amPLat) || pick["pickup_address"] != "موقع الأمانة" {
		t.Fatalf("لم تصر نقطةَ استلامٍ للسائق: %v / %v", pick["pickup_lat"], pick["pickup_address"])
	}

	// الثانية: لا طلبَ مفتوحاً ⇒ لا ردّ، ولا تغيير.
	if r := h.API.HandleWALocation(ctxBG(), "+963944555602", 35.0, 39.0, 0); r != "" {
		t.Fatalf("ردٌّ بلا طلبٍ مفتوح: %q", r)
	}
	if again := waLocStatus(t, h, cust.Token, waID); !near(again["lat"], amPLat) {
		t.Fatalf("الموقعُ تبدّل بعد الإغلاق: %v", again["lat"])
	}
}

// رقمٌ بلا طلب، ومنتهي الصلاحية، ورقمٌ غير صالح ⇒ لا ردّ ولا كتابة.
func TestWALocationReplyIgnoredWhenNothingOpen(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	if r := h.API.HandleWALocation(ctxBG(), "+963944555610", amPLat, amPLng, 5); r != "" {
		t.Fatalf("رقمٌ بلا أيّ طلب ⇒ %q", r)
	}
	id, _ := requestHolderLocation(t, h, cust.Token, "0944555611")
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE recipient_location_requests SET expires_at = now() - interval '1 minute' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if r := h.API.HandleWALocation(ctxBG(), "+963944555611", amPLat, amPLng, 5); r != "" {
		t.Fatalf("طلبٌ منتهٍ ⇒ %q", r)
	}
	if s := waLocStatus(t, h, cust.Token, id); s["status"] != "expired" || s["lat"] != nil {
		t.Fatalf("الطلبُ المنتهي تغيّر: %v", s)
	}
	if r := h.API.HandleWALocation(ctxBG(), "ليس رقماً", amPLat, amPLng, 5); r != "" {
		t.Fatalf("رقمٌ غيرُ صالح ⇒ %q", r)
	}
	if r := h.API.HandleWALocation(ctxBG(), "+963944555611", 200, 500, 5); r != "" {
		t.Fatalf("إحداثيّةٌ خارجَ المدى ⇒ %q", r)
	}
}

// طلبان مفتوحان للرقم نفسِه: يمتلئ **الأحدثُ** وحدَه.
func TestWALocationReplyFillsLatestOfTwo(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	older, _ := requestHolderLocation(t, h, cust.Token, "0944555620")
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE recipient_location_requests SET created_at = now() - interval '10 minutes' WHERE id = $1`, older); err != nil {
		t.Fatal(err)
	}
	newer, _ := requestHolderLocation(t, h, cust.Token, "0944555620")
	if r := h.API.HandleWALocation(ctxBG(), "+963944555620", amPLat, amPLng, 5); r == "" {
		t.Fatal("لا ردَّ")
	}
	if s := waLocStatus(t, h, cust.Token, newer); s["status"] != "received" {
		t.Fatalf("الأحدثُ لم يمتلئ: %v", s["status"])
	}
	if s := waLocStatus(t, h, cust.Token, older); s["status"] != "pending" {
		t.Fatalf("الأقدمُ امتلأ: %v", s["status"])
	}
}
