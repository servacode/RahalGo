package server

// **سائقٌ لا يمسّ طلبَ سائقٍ آخر** — سدُّ IDOR على بوّابة السائق.
//
// الحارسُ `driverOwnsOrder` (`driver_handlers.go`) يقف قبل كلّ فعلٍ على طلبٍ
// بمعرّفه (نقلٌ/تحريرٌ/إثباتٌ/مسارٌ/ردٌّ/طوارئ)، لكن **لم يكن له اختبارٌ يُثبت
// أنّ سائقاً لا ينقل طلبَ غيرِه** (فجوةُ اختبارٍ أمنيّة، جردُ ٢٠٢٦-٠٩-٢٧).
// **ولو انكسر الحارسُ يوماً لَنقل سائقٌ طلبَ آخر** — يقبض أجرَه أو يُفشله.

import (
	"context"
	"net/http"
	"testing"
)

// TestDriverTransition_IDOR_ForeignOrderRejected
// **ب لا ينقل طلبَ أ** — 403 not_your_order، والطلبُ لا يتحرّك.
func TestDriverTransition_IDOR_ForeignOrderRejected(t *testing.T) {
	f := newDriverFixture(t, 2)
	driverA, driverB := f.drivers[0], f.drivers[1]
	orderID := f.atDropoffOrder(t, driverA) // مُسنَدٌ إلى أ

	w := f.fail(driverB, orderID, "customer_absent", "") // ب يحاول
	if w.Code != http.StatusForbidden {
		t.Fatalf("ب نقل طلبَ أ: الرمز %d (المتوقّع 403)\n%s", w.Code, w.Body.String())
	}
	if code := errCode(t, w); code != "not_your_order" {
		t.Fatalf("الرمزُ %q، والمتوقّع not_your_order", code)
	}
	// **والطلبُ لم يتحرّك** — ما زال عند باب الزبون بيد أ.
	var status string
	var assigned *string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT status, driver_id::text FROM orders WHERE id = $1`, orderID).
		Scan(&status, &assigned); err != nil {
		t.Fatalf("قراءةُ الطلب: %v", err)
	}
	if status != "at_dropoff" || assigned == nil || *assigned != driverA {
		t.Fatalf("تغيّر طلبُ أ بفعل ب: status=%s driver=%v", status, assigned)
	}
}

// TestDriverTransition_IDOR_UnassignedOrderRejected
// **وطلبٌ بلا سائقٍ لا يملكه عابر** — لا ينقله من ليس مُسنَداً إليه.
func TestDriverTransition_IDOR_UnassignedOrderRejected(t *testing.T) {
	f := newDriverFixture(t, 1)
	orderID := f.dispatchingOrder(t, 20000, 5000) // بلا سائق
	w := f.fail(f.drivers[0], orderID, "customer_absent", "")
	if w.Code != http.StatusForbidden || errCode(t, w) != "not_your_order" {
		t.Fatalf("سائقٌ نقل طلباً غيرَ مُسنَدٍ إليه: %d %s", w.Code, w.Body.String())
	}
}
