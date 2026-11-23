package server

import (
	"net/http"
	"testing"
)

// TestLive_BeforeAssignSaysNotYet **المشاركةُ تبدأ لحظةَ تأكيد الطلب** (قرارُ المالك
// ٢٠٢٦-١١-٢٣: «اذا فاتح او مسكر يظل الموقع المباشر شغال») — فقبل الإسناد يُردّ «لم يُسنَد
// بعد» (`live_location_not_yet`) **فيُبقي الهاتفُ مشاركتَه**، **ولا تُكتب نقطة**، **وبعد
// النهاية يبقى «انتهى»** فيقف.
func TestLive_BeforeAssignSaysNotYet(t *testing.T) {
	f := newDriverFixture(t, 1)
	for _, st := range []string{"pending", "accepted", "preparing", "dispatching"} {
		orderID, owner := f.liveOrder(t, st, nil)
		w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", farFromPin)
		if code := errCode(t, w); code != "live_location_not_yet" {
			t.Fatalf("%s: توقّعنا live_location_not_yet، والنتيجة %d %q", st, w.Code, code)
		}
		if n := f.liveRows(t, orderID); n != 0 {
			t.Fatalf("%s: كُتبت نقطةٌ قبل الإسناد", st)
		}
	}
	for _, st := range []string{"delivered", "cancelled"} {
		orderID, owner := f.liveOrder(t, st, nil)
		w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", farFromPin)
		if code := errCode(t, w); code != "live_location_inactive" {
			t.Fatalf("%s: توقّعنا live_location_inactive، والنتيجة %d %q", st, w.Code, code)
		}
	}
}
