package server

// **طريقُ طلبِ غيره لا يُرسم ولا يسقط** (تجربةُ القبول ٢٠٢٦-١٠-٠٣) — كان
// `GET /driver/orders/{id}/route` لطلبِ سائقٍ آخر يردّ **٥٠٠**: المسحُ لا يجد صفّاً
// (`driver_id = $2`) **فيُردّ «لا صفّ» خطأَ خادم.** **والجوابُ كبقيّة أبواب السائق:
// `not_your_order`.**

import (
	"net/http"
	"strings"
	"testing"
)

func TestDriverRoute_OtherDriversOrderIsNotYours(t *testing.T) {
	f := newDriverFixture(t, 2)
	owner, stranger := f.drivers[0], f.drivers[1]
	id := f.problemOrderAt(t, "at_pickup", owner)

	w := f.call(f.srv.handleDriverOrderRoute, http.MethodGet, "/driver/orders/"+id+"/route", id, stranger,
		[]string{"driver"}, "")
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "not_your_order") {
		t.Fatalf("طريقُ طلبِ غيره ردّ %d: %s — والمتوقّعُ 403 not_your_order", w.Code, w.Body.String())
	}
	// **ومعرّفٌ ليس معرّفاً** لا يصير خطأَ خادمٍ أيضاً.
	w = f.call(f.srv.handleDriverOrderRoute, http.MethodGet, "/driver/orders/x/route", "not-a-uuid", stranger,
		[]string{"driver"}, "")
	if w.Code >= 500 {
		t.Fatalf("معرّفٌ فاسدٌ ردّ %d: %s", w.Code, w.Body.String())
	}
}
