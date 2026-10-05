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
