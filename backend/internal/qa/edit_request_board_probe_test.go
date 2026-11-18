package qa

import (
	"net/http"
	"strings"
	"testing"
)

func TestEDITREQ_BoardShowsPending(t *testing.T) {
	h := New(t)
	oid, drv := h.assignedOrder(t)
	ops := h.NewUser("operations")
	r := h.POSTKey("/api/v1/driver/orders/"+oid+"/edit-request", drv.Token, uniq("er"),
		map[string]any{"adds": []map[string]any{{"name": "خبز", "qty": 1, "price": 350}}})
	if r.Code != http.StatusOK {
		t.Fatalf("طلب: %s", r)
	}
	l := h.GET("/api/v1/admin/orders?per_page=100", ops.Token)
	body := string(l.Body)
	i := strings.Index(body, oid)
	if i < 0 {
		t.Fatalf("الطلب مو باللوحة: %s", l)
	}
	if !strings.Contains(body, "edit_request_at") {
		t.Fatalf("اللوحة بلا edit_request_at: %s", body[i:min(len(body), i+4000)])
	}
}
