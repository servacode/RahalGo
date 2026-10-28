package qa

// **حمايةُ «لدي توصيلة»** — دوامُ المنصّة وحدَه (قرارُ المالك ٢٠٢٦-١٠-٢٩).
//
// (نصُّه: «مادام المنصة بالدوام معناها في سائقين، إذا كانت خارج الدوام ما
//  يقدر يرسل».) **وأُلغي شرطُ «سائقٍ بالدوام»** الذي قرّره ٢٠٢٦-١٠-٠١.

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func mdgExpect(t *testing.T, h *Harness, m mdFx, code string) {
	t.Helper()
	q := h.GET(m.url("/delivery-quote"), m.fx.Tok)
	if q.Code != http.StatusServiceUnavailable || q.Err() != code {
		t.Fatalf("عرضُ السعر: %d / %s — والمنتظَرُ 503 / %s", q.Code, q.Err(), code)
	}
	c := h.POSTKey(m.url("/deliveries"), m.fx.Tok, uniq("mdg"), m.body("recipient"))
	if c.Code != http.StatusServiceUnavailable || c.Err() != code {
		t.Fatalf("الإنشاء: %d / %s — والمنتظَرُ 503 / %s", c.Code, c.Err(), code)
	}
}

// TestMDG01_NoDriverOnShiftStillSends **لا سائقَ بالدوام والمنصّةُ بدوامها ⇒ تمرّ**
// (قرارُ المالك ٢٠٢٦-١٠-٢٩ — كانت تُردّ بـ`no_drivers_on_shift`). والتوزيعُ ينتظر.
func TestMDG01_NoDriverOnShiftStillSends(t *testing.T) {
	h := New(t)
	m := newMDFxBare(t, h, "منطقةُ MDG-01")
	// **وسائقو الفحوص السابقة في القاعدة نفسِها خارجَ الدوام** — يُقاس هذا وحدَه.
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET on_shift = false WHERE on_shift`); err != nil {
		t.Fatal(err)
	}
	m.f.Driver(LocationAt(m.z.Lat, m.z.Lng, time.Now()))
	mdgAllowed(t, h, m, "mdg01")
}

// TestMDG06_PlatformClosedHours **المنصّةُ خارجَ دوامها ⇒ لا عرضَ سعرٍ ولا إنشاء**
// ولو كان سائقٌ بالدوام — **ومعه موعدُ الفتح.**
func TestMDG06_PlatformClosedHours(t *testing.T) {
	h := New(t)
	m := newMDFxBare(t, h, "منطقةُ MDG-06")
	mdDriverAtStore(t, h, m.f, m.fx.M.ID)
	closedNow(t, h)
	t.Cleanup(func() { ordersOpen(t, h) })
	mdgExpect(t, h, m, "platform_closed_now")
	q := h.GET(m.url("/delivery-quote"), m.fx.Tok)
	if !strings.Contains(string(q.Body), "next_available_at") {
		t.Fatalf("خارجَ الدوام بلا موعد فتح: %s", q.Body)
	}
}

// mdgAllowed **تمرّ** — عرضُ السعر والإنشاء كلاهما.
func mdgAllowed(t *testing.T, h *Harness, m mdFx, tag string) {
	t.Helper()
	if q := h.GET(m.url("/delivery-quote"), m.fx.Tok); q.Code != http.StatusOK {
		t.Fatalf("عرضُ السعر: %d / %s", q.Code, q.Err())
	}
	m.create(t, h, "recipient", uniq(tag))
}

// TestMDG02_DriverFarStillCounts **بعيدٌ لكن بالدوام ⇒ تمرّ** — قرارُ المالك:
// «مو ضروري يكونون قريبين، المهمّ في سائقين بالدوام».
func TestMDG02_DriverFarStillCounts(t *testing.T) {
	h := New(t)
	m := newMDFxBare(t, h, "منطقةُ MDG-02")
	m.f.Driver(OnShift(), LocationAt(m.z.Lat+1.0, m.z.Lng, time.Now())) // ~١١١ كم
	mdgAllowed(t, h, m, "mdg02")
}

// TestMDG03_BusyOrStaleStillCounts **بالدوام وموقعُه شائخ ⇒ تمرّ** — «حتّى لو كانوا مشغولين».
func TestMDG03_BusyOrStaleStillCounts(t *testing.T) {
	h := New(t)
	m := newMDFxBare(t, h, "منطقةُ MDG-03")
	m.f.Driver(OnShift(), LocationAt(m.z.Lat, m.z.Lng, time.Now().Add(-3*time.Hour)))
	mdgAllowed(t, h, m, "mdg03")
}

// TestMDG04_DriverNearbyPasses **بالدوام عند المتجر** ⇒ تمرّ.
func TestMDG04_DriverNearbyPasses(t *testing.T) {
	h := New(t)
	m := newMDFxBare(t, h, "منطقةُ MDG-04")
	mdDriverAtStore(t, h, m.f, m.fx.M.ID)
	mdgAllowed(t, h, m, "mdg04")
}

// TestMDG05_PlatformPaused **المنصّةُ موقوفةٌ مؤقّتاً** ⇒ رمزُ طلب الزبون نفسُه.
func TestMDG05_PlatformPaused(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDG-05")
	closure(t, h, true, "صيانةٌ مؤقّتة", nil)
	t.Cleanup(func() { closure(t, h, false, "", nil) })
	mdgExpect(t, h, m, "temporarily_unavailable")
}

// TestOFU01_NoOfferOnUnavailableItem **لا عرضَ على صنفٍ «غير متوفر»**
// (نصُّ المالك: «مو معقول ينزل عرض لصنف مو موجود عنده أصلاً»).
func TestOFU01_NoOfferOnUnavailableItem(t *testing.T) {
	h := New(t)
	fx := newOfferFx(t, h, h.Factory(), 1000)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE menu_items SET available = false WHERE id = $1`, fx.Item.ID); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{"title": "x", "menu_item_id": fx.Item.ID, "discount_percent": 10}
	for _, who := range []struct{ path, tok string }{
		{"/api/v1/merchant/stores/" + fx.M.ID + "/offers", fx.Tok},
		{"/api/v1/rep/stores/" + fx.M.ID + "/offers", fx.RepTok},
	} {
		r := h.POST(who.path, who.tok, body)
		if r.Code != http.StatusConflict || r.Err() != "offer_item_unavailable" {
			t.Fatalf("%s: %d / %s — والمنتظَرُ 409 / offer_item_unavailable", who.path, r.Code, r.Err())
		}
	}
}
