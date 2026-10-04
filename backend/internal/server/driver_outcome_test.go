package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestDriverOutcome_SaysWhyTheOrderLeft **الطلبُ يختفي من قائمته — والبابُ يقول لماذا.**
func TestDriverOutcome_SaysWhyTheOrderLeft(t *testing.T) {
	f := newDriverFixture(t, 2)
	d, other := f.drivers[0], f.drivers[1]
	id := f.problemOrderAt(t, "assigned", d)
	ops := testdb.NewUser(t, f.pool, "operations")
	if _, err := f.srv.orders.Transition(context.Background(), ops, []string{"ops"}, id, "dispatching", ""); err != nil {
		t.Fatalf("تعذّرت الإعادة: %v", err)
	}

	w := f.call(f.srv.handleDriverOrderOutcome, http.MethodGet, "/driver/orders/"+id+"/outcome", id, d,
		[]string{"driver"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("المآل ردّ %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data struct {
			Number  int64  `json:"number"`
			Status  string `json:"status"`
			Reason  string `json:"reason"`
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Reason != "requeued_ops" || body.Data.Message == "" || body.Data.Number == 0 {
		t.Errorf("المآل %+v", body.Data)
	}

	// **ولا يُجاب سائقٌ لم يحمله.**
	w = f.call(f.srv.handleDriverOrderOutcome, http.MethodGet, "/driver/orders/"+id+"/outcome", id, other,
		[]string{"driver"}, "")
	if w.Code != http.StatusNotFound {
		t.Errorf("سائقٌ غريبٌ ردّ %d — **لا يُجاب عن طلبٍ لم يكن بيده**", w.Code)
	}
}
