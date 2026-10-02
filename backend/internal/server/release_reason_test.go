package server

// ══════════════════════════════════════════════════════════════════════
// **لا «إعادة إلى الطابور» — تركٌ قبل الاستلام بسببٍ وكلمة، ولا يعود الطلبُ إلى
// من تركه أبداً، ودوامُه يُغلَق** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢، البند ٢ و٣)
// ══════════════════════════════════════════════════════════════════════

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func (f *driverFixture) release(driverID, orderID, body string) *httptest.ResponseRecorder {
	return f.call(f.srv.handleDriverRelease, http.MethodPost,
		"/driver/orders/"+orderID+"/release", orderID, driverID, []string{"driver"}, body)
}

func (f *driverFixture) isOnShift(t *testing.T, driverID string) bool {
	t.Helper()
	var on bool
	if err := f.pool.QueryRow(context.Background(),
		`SELECT on_shift FROM users WHERE id = $1`, driverID).Scan(&on); err != nil {
		t.Fatal(err)
	}
	return on
}

// TestRelease_NeedsReasonAndNote_BeforePickupOnly **لا تركَ بلا سببٍ من الثلاثة
// وكلمة — ولا بعد الاستلام.**
func TestRelease_NeedsReasonAndNote_BeforePickupOnly(t *testing.T) {
	f := newDriverFixture(t, 1)
	f.armOps(t)
	d := f.drivers[0]
	orderID := f.problemOrderAt(t, "assigned", d)

	for _, body := range []string{``, `{}`, `{"note":"تعبت"}`,
		`{"reason":"bike_broken"}`, `{"reason":"tired","note":"x"}`} {
		w := f.release(d, orderID, body)
		if w.Code != http.StatusBadRequest || errCode(t, w) != "release_reason_required" {
			t.Fatalf("تركٌ بجسم %q ردّ %d %s — **ولا يُترك الطلبُ إلّا بسببٍ وكلمة**",
				body, w.Code, w.Body.String())
		}
	}
	if st, drv := f.orderRow(t, orderID); st != "assigned" || drv == nil || *drv != d {
		t.Fatalf("(%s · %v) — تحرّك الطلبُ مع الرفض", st, drv)
	}

	after := f.problemOrderAt(t, "picked_up", d)
	w := f.release(d, after, `{"reason":"accident","note":"وقعتُ"}`)
	if w.Code != http.StatusConflict || errCode(t, w) != "release_wrong_stage" {
		t.Fatalf("تركٌ بعد الاستلام ردّ %d %s — **وبابُه الطارئ**", w.Code, w.Body.String())
	}
}

// TestRelease_NeverReturnsToLeaver_ShiftEnds_NoApology **الطلبُ لا يعود إلى من تركه
// ولو لم يوجد غيرُه، ودوامُه يُغلَق، ولا سطرَ اعتذارٍ للزبون.**
func TestRelease_NeverReturnsToLeaver_ShiftEnds_NoApology(t *testing.T) {
	f := newDriverFixture(t, 1)
	f.armOps(t)
	armRotation(t, f, 60)
	ctx := context.Background()
	d := f.drivers[0]
	f.onShift(t, d, true)
	orderID := f.problemOrderAt(t, "assigned", d)

	w := f.release(d, orderID, `{"reason":"bike_broken","note":"انكسر الجنزير"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("التركُ ردّ %d: %s", w.Code, w.Body.String())
	}
	if f.isOnShift(t, d) {
		t.Error("دوامُ التارك لم يُغلَق — **ومن تعطّلت درّاجتُه لا يُعرض عليه**")
	}
	var apologies int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM order_messages WHERE order_id = $1 AND body LIKE '%تعذّر عليّ%'`,
		orderID).Scan(&apologies); err != nil {
		t.Fatal(err)
	}
	if apologies != 0 {
		t.Error("سطرُ «تعذّر عليّ إكمالُ طلبك» كُتب للزبون — **وقد حذفه المالك**")
	}

	// **ولو عاد إلى دوامه وهو الوحيدُ المؤهَّل** — لا يُعرض عليه.
	f.onShift(t, d, true)
	if err := f.srv.orders.OfferNext(ctx, orderID, nil); err != nil {
		t.Fatalf("العرض: %v", err)
	}
	f.srv.orders.SweepWaitingOffers(ctx)
	if got := f.offeredDriver(t, orderID); got != nil && *got == d {
		t.Fatal("عُرض الطلبُ على من تركه — **«لا يعود الطلبُ إلى من تركه أبداً»**")
	}
	if st, drv := f.orderRow(t, orderID); st != "dispatching" || drv != nil {
		t.Fatalf("(%s · %v) — والمتوقّعُ في الطابور بلا سائق", st, drv)
	}

	// **وفي «للجميع» لا يراه في طابوره ولا يقبله بنداءٍ مباشر.**
	f.setSetting(t, "drivers.assignment_mode", "queue")
	for _, id := range f.queueIDs(t, d) {
		if id == orderID {
			t.Fatal("الطلبُ في طابور من تركه")
		}
	}
	if a := f.accept(d, orderID); a.Code != http.StatusConflict || errCode(t, a) != "driver_excluded" {
		t.Fatalf("قبولُ التارك ردّ %d %s", a.Code, a.Body.String())
	}
}
