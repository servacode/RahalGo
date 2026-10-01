package qa

// **الإيقافُ المؤقّتُ يُكتب ولو غاب صفُّه** — فحصُ المتجر ٢٠٢٦-١٠-٠١.
//
// **قِيس على التجهيز**: «أوقف المنصّة» يردّ ٢٠٠ و`active: true` **والطلباتُ تُقبل**
// — صفُّ `service_closure` الوحيدُ غائب (إعادةُ ضبط التجهيز تُفرغ الجداول)، و`UPDATE`
// بلا صفٍّ يمرّ صامتاً.

import (
	"net/http"
	"testing"
)

func TestPH_ClosureWritesEvenWithoutRow(t *testing.T) {
	h := New(t)
	adm := h.NewUser("admin")
	if _, err := h.Pool.Exec(ctxBG(), `DELETE FROM service_closure`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `
			INSERT INTO service_closure (id, active, message) VALUES (true, false, '')
			ON CONFLICT (id) DO UPDATE SET active = false, message = '', ends_at = NULL`)
	})
	closureVia(t, h, adm.Token, true, "صيانة")

	var active bool
	if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE((SELECT active FROM service_closure WHERE id), false)`).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatalf("**«أوقف المنصّة» ردّ نجاحاً ولم يُكتب شيء** — الصفُّ غائبٌ والطلباتُ ستُقبل")
	}
	item := h.NewItem(5000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("ph"), orderBody(item, 1))
	if made.Code != http.StatusServiceUnavailable || made.Err() != "temporarily_unavailable" {
		t.Fatalf("**المنصّةُ موقوفةٌ والطلبُ قُبل**: %d / %s", made.Code, made.Err())
	}
}
