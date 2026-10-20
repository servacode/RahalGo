package qa

import (
	"net/http"
	"testing"
)

// TestFETCHAMT_USD **مبلغُ الإحضار بالدولار** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — يُحفظ بالسنتات
// ويُردّ بعملته، **والسائقُ يكتب بعملة الطلب**، والإشعارُ يقول «١٣٫٢٥$» لا «١٣٢٥».
// ودولارٌ بثلاث منازل يُردّ.
func TestFETCHAMT_USD(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ FETCHAMT-USD")
	owner := m.fx.M.Owner.ID

	body := m.body("recipient")
	body["delivery_mode"] = "fetch"
	body["collect_amount"] = 12.345
	body["amount_currency"] = "USD"
	if r := h.POSTKey(m.url("/deliveries"), m.fx.Tok, uniq("fetchusd-bad"), body); r.Code != http.StatusBadRequest {
		t.Fatalf("دولارٌ بثلاث منازل قُبل: %s", r)
	}
	body["collect_amount"] = 12.5
	r := h.POSTKey(m.url("/deliveries"), m.fx.Tok, uniq("fetchusd"), body)
	if r.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %d / %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	m.accept(t, h, id)
	drv := h.driverOf(id)

	one := h.GET("/api/v1/merchant/deliveries/"+id, m.fx.Tok).JSON()
	if one["collect_amount"] != float64(1250) || one["amount_currency"] != "USD" {
		t.Fatalf("التقدير لم يُحفظ ١٢٥٠ سنتاً بالدولار: %v %v", one["collect_amount"], one["amount_currency"])
	}
	if r := fetchStep(h, drv, id, "at_pickup"); r.Code >= 400 {
		t.Fatalf("الوصول: %s", r)
	}
	if r := h.POST("/api/v1/driver/orders/"+id+"/fetch-amount", drv.Token, map[string]any{"amount": 13.25}); r.Code != http.StatusOK {
		t.Fatalf("كتابةُ المبلغ بالدولار: %s", r)
	}
	one = h.GET("/api/v1/merchant/deliveries/"+id, m.fx.Tok).JSON()
	if one["fetch_amount"] != float64(1325) {
		t.Fatalf("مبلغُ السائق لم يُحفظ ١٣٢٥ سنتاً: %v", one["fetch_amount"])
	}
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE user_id = $1::uuid AND entity_id = $2 AND body LIKE '%13.25$%'`, owner, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("إشعارُ المتجر لا يقول «13.25$» (= %d)", n)
	}
}
