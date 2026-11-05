package qa

// مواضعُ السائقين الحيّة في البثّ — **لمن يملك `drivers.read` وحدَه** (L1a، ٢٠٢٦-١١-٠٥).
//
// # ما كان يقع
//
// **نبضةُ السائق تُبثّ بحمولتها** (`driver_live.go`) — **وكانت على `ops`**، و`ops` لكلّ من
// يملك `orders.read` (`ws.go`). **فخدمةُ الزبائن والأمانُ كانت تصلهم في الخلفية مواضعُ كلِّ
// سائقٍ على الدوام كلَّ أربع ثوانٍ** — ولا صلاحيّةَ لهم أن يروا السائقين: **`GET
// /ops-map/drivers` محروسٌ بـ`drivers.read`** (`opsmap/perm.go`).
//
// # وما يُقاس
//
// **مصافحةٌ حقيقيّةٌ ونبضةٌ حقيقيّة** — لا نداءُ دالّة: موظّفُ خدمة الزبائن تصله أحداثُ
// الطلبات ولا يصله موضعُ السائق، **وموظّفُ العمليّات يصله الاثنان.**

import (
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// wsWaitType **ينتظر حدثاً من نوعٍ بعينه** — ويتخطّى ما سواه (ما يبثّه `wsExpect` من طلبات).
func wsWaitType(in <-chan map[string]any, typ string, within time.Duration) map[string]any {
	deadline := time.After(within)
	for {
		select {
		case m, ok := <-in:
			if !ok {
				return nil
			}
			if m["type"] == typ {
				return m
			}
		case <-deadline:
			return nil
		}
	}
}

func TestWS_DriverLocationNeedsDriversRead(t *testing.T) {
	h := New(t)

	// ── المقدّمة تُقرأ لا تُفترض: خدمةُ الزبائن تقرأ الطلبات ولا ترى السائقين، والعمليّاتُ الاثنين ──
	if !roleHasCap(t, h, "customer_support", authz.OrdersRead) || roleHasCap(t, h, "customer_support", authz.DriversRead) {
		t.Fatal("**تبدّلت قدراتُ خدمة الزبائن** — الفحصُ يفترض orders.read بلا drivers.read")
	}
	if !roleHasCap(t, h, "operations", authz.OrdersRead) || !roleHasCap(t, h, "operations", authz.DriversRead) {
		t.Fatal("**تبدّلت قدراتُ العمليّات** — الفحصُ يفترض orders.read وdrivers.read معاً")
	}

	drv := h.NewUser("driver")
	support := h.NewUser("customer_support")
	ops := h.NewUser("operations")

	cs, code := wsDial(t, h, support.Token)
	if cs == nil {
		t.Fatalf("مصافحةُ خدمة الزبائن ردّت %d", code)
	}
	defer cs.CloseNow()
	co, code := wsDial(t, h, ops.Token)
	if co == nil {
		t.Fatalf("مصافحةُ العمليّات ردّت %d", code)
	}
	defer co.CloseNow()
	inS, inO := wsReader(cs), wsReader(co)

	// ── أحداثُ الطلبات تصل الاثنين (وبها يُعرف أنّ الاشتراكَ قام) ──
	if wsExpect(t, h, inS, "ops", 3*time.Second) == nil {
		t.Fatal("**خدمةُ الزبائن لم تعد تصلها أحداثُ الطلبات** — والمطلوبُ حجبُ المواضع وحدَها")
	}
	if wsExpect(t, h, inO, "ops", 3*time.Second) == nil {
		t.Fatal("العمليّاتُ لم تصلها أحداثُ الطلبات")
	}

	// ── ونبضةٌ حقيقيّةٌ من السائق ──
	res := h.POST("/api/v1/driver/location", drv.Token, map[string]any{
		"lat": 35.9506, "lng": 39.0094,
		"recorded_at": time.Now().Add(-time.Second).UTC().Format(time.RFC3339),
	})
	if res.Code != 200 {
		t.Fatalf("النبضة: %s", res)
	}

	if got := wsWaitType(inO, "driver_location", 3*time.Second); got == nil || got["driver_id"] != drv.ID {
		t.Fatalf("**موضعُ السائق لم يصل العمليّات** (%v) — فتبقى خريطتُهم تسأل وتقفز", got)
	}
	if got := wsWaitType(inS, "driver_location", 1500*time.Millisecond); got != nil {
		t.Fatalf("**موضعُ السائق وصل خدمةَ الزبائن** (%v) — ولا صلاحيّةَ لها أن ترى السائقين", got)
	}
}
