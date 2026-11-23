package server

import (
	"context"
	"net/http"
	"testing"
)

// **«قبول» على طلبٍ صار للكابتن نفسِه لا يقول «سبقك»** (بلاغُ المالك ٢٠٢٦-١١-٢٣:
// «في اشعار سبقك شخص اخر الى هذا الطلب مع انه الطلب عند نفس السائق»).
func TestAccept_OwnOrderIsNotTaken(t *testing.T) {
	f := newDriverFixture(t, 2)
	for _, d := range f.drivers {
		f.onShift(t, d, true)
	}

	t.Run("ضغط مرتين", func(t *testing.T) {
		oid := f.dispatchingOrder(t, 3000, 1000)
		if w := f.accept(f.drivers[0], oid); w.Code != http.StatusOK {
			t.Fatalf("القبول الأوّل: %d — %s", w.Code, w.Body.String())
		}
		if w := f.accept(f.drivers[0], oid); w.Code != http.StatusOK {
			t.Fatalf("**الضغطة الثانية على طلبه ردّت %d**: %s", w.Code, w.Body.String())
		}
		f.closeOrder(t, oid)
	})

	t.Run("أسندته العمليات ثم ضغط قبول", func(t *testing.T) {
		oid := f.dispatchingOrder(t, 3000, 1000)
		if _, err := f.pool.Exec(context.Background(),
			`UPDATE orders SET driver_id = $2, status = 'assigned' WHERE id = $1`, oid, f.drivers[0]); err != nil {
			t.Fatal(err)
		}
		if w := f.accept(f.drivers[0], oid); w.Code != http.StatusOK {
			t.Fatalf("**قبولُ طلبٍ أُسند إليه ردّ %d**: %s", w.Code, w.Body.String())
		}
		// **وغيرُه ما زال يُردّ «سبقك»**
		if code := errCode(t, f.accept(f.drivers[1], oid)); code != "order_taken" {
			t.Fatalf("كابتنٌ ثانٍ على طلبٍ مأخوذ: توقّعنا order_taken، والنتيجة %q", code)
		}
		if d := f.assignedDriver(t, oid); d == nil || *d != f.drivers[0] {
			t.Fatalf("الطلب انتقل من صاحبه: %v", d)
		}
		f.closeOrder(t, oid)
	})
}

// closeOrder يغلق الطلبَ كي لا يُحسب على سقف الكابتن في الحالة التالية.
func (f *driverFixture) closeOrder(t *testing.T, oid string) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET status = 'delivered', closed_at = now() WHERE id = $1`, oid); err != nil {
		t.Fatal(err)
	}
}
