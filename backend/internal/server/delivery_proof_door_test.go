package server

import (
	"net/http"
	"testing"
)

// TestDeliveryProof_OnlyAtTheDoor **صورةُ التسليم عند الباب وحدَه** (٢٠٢٦-١٠-٠٢) —
// كانت تُقبل في الطريق فتُحفظ «إثباتاً» لتسليمٍ لم يقع.
func TestDeliveryProof_OnlyAtTheDoor(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	for _, st := range []string{"assigned", "at_pickup", "picked_up", "on_the_way"} {
		id := f.problemOrderAt(t, st, d)
		w := f.call(f.srv.handleDeliveryProof, http.MethodPost, "/driver/orders/"+id+"/proof", id, d,
			[]string{"driver"}, "")
		if code := errCode(t, w); code != "proof_not_at_door" {
			t.Errorf("%s: %d %q — **صورةٌ قبل الباب تُقبل إثباتاً**", st, w.Code, code)
		}
	}
}
