package server

// **لا يُنتزَع طلبٌ معروضٌ على غيره** (فحصُ الهجوم ٢٠٢٦-١٠-٠٥، بموافقة المالك)
//
// **حارسٌ لا إصلاح**: نمطُ «الدور» (الافتراضيّ) يردّ هذا أصلاً بـ`order_taken`. والثغرةُ
// التي قاسها فحصُ الهجوم في نمط «للجميع» مع القرب (قبولُ طلبٍ لا يُرى بمعرّفه) **تبقى
// مفتوحةً هناك** — فلا يُشغَّل «للجميع» قبل سدّها.

import (
	"context"
	"net/http"
	"testing"
)

func TestAccept_OrderOfferedToAnotherDriverIsRefused(t *testing.T) {
	f := newDriverFixture(t, 2)
	owner, thief := f.drivers[0], f.drivers[1]
	f.onShift(t, owner, true)
	f.onShift(t, thief, true)
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	if _, err := f.pool.Exec(context.Background(), `
		UPDATE orders SET status = 'dispatching', driver_id = NULL,
		       offered_driver_id = $2, offer_expires_at = now() + interval '1 minute'
		WHERE id = $1`, ord, owner); err != nil {
		t.Fatal(err)
	}
	if w := f.accept(thief, ord); w.Code == http.StatusOK {
		t.Fatalf("سائقٌ انتزع طلباً معروضاً على غيره: %d %s", w.Code, w.Body.String())
	}
	if w := f.accept(owner, ord); w.Code != http.StatusOK {
		t.Fatalf("صاحبُ العرض لم يستطع القبول: %d %s", w.Code, w.Body.String())
	}
}

// TestProximity_AcceptOutsideRingIsRefused **في «الأسرع» لا يقبل البعيدُ ما لا يراه.**
func TestProximity_AcceptOutsideRingIsRefused(t *testing.T) {
	f := newDriverFixture(t, 2)
	armProximity(t, f)
	f.setSetting(t, "drivers.assignment_mode", "queue")
	near, out := f.drivers[0], f.drivers[1]
	f.onShift(t, near, true)
	f.onShift(t, out, true)
	f.standAt(t, near, nearLat, nearLng)
	f.standAt(t, out, outLat, outLng) // ~١٢٫٦كم — خارجَ الأقصى
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)

	if hasID(f.queueIDs(t, out), ord) {
		t.Fatalf("البعيدُ يرى الطلب — والاختبارُ يفترض أنّه لا يراه")
	}
	if w := f.accept(out, ord); w.Code == http.StatusOK {
		t.Fatalf("سائقٌ خارجَ الحلقة قبِل طلباً لا يراه: %d %s", w.Code, w.Body.String())
	}
	if w := f.accept(near, ord); w.Code != http.StatusOK {
		t.Fatalf("القريبُ لم يستطع القبول: %d %s", w.Code, w.Body.String())
	}
}

// TestProximity_AcceptWithStaleLocationIsRefused **ولا من شاخ موضعُه.**
func TestProximity_AcceptWithStaleLocationIsRefused(t *testing.T) {
	f := newDriverFixture(t, 1)
	armProximity(t, f)
	f.setSetting(t, "drivers.assignment_mode", "queue")
	d := f.drivers[0]
	f.onShift(t, d, true)
	f.standAt(t, d, nearLat, nearLng)
	ord := f.orderAt(t, pmLat, pmLng, pdLat, pdLng)
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE users SET last_location_at = now() - interval '2 hours' WHERE id = $1`, d); err != nil {
		t.Fatal(err)
	}
	if w := f.accept(d, ord); w.Code == http.StatusOK {
		t.Fatalf("سائقٌ بموضعٍ شائخٍ قبِل طلباً: %d %s", w.Code, w.Body.String())
	}
}
