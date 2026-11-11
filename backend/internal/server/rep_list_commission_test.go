package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestRepStoreList_CommissionPercent نسبةُ عمولة المنصّة تصل قائمةَ «عملائي» (قراءةٌ فقط،
// طلبُ المالك ٢٠٢٦-١١-١١) — وهي نفسُها التي في التفصيل: خاصّةُ المتجر إن وُضعت وإلّا العامّة.
func TestRepStoreList_CommissionPercent(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	rep := testdb.NewUser(t, f.pool, "sales")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET sales_rep_user_id = $2 WHERE id = $1`, f.merchantID, rep); err != nil {
		t.Fatal(err)
	}

	pct := func(body []byte, list bool) float64 {
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(body, &env); err != nil {
			t.Fatal(err)
		}
		type row struct {
			ID      string  `json:"id"`
			Percent float64 `json:"store_commission_percent"`
		}
		if list {
			var rows []row
			if err := json.Unmarshal(env.Data, &rows); err != nil {
				t.Fatal(err)
			}
			for _, r := range rows {
				if r.ID == f.merchantID {
					return r.Percent
				}
			}
			t.Fatal("المتجر غير موجود في القائمة")
		}
		var d struct {
			Merchant row `json:"merchant"`
		}
		if err := json.Unmarshal(env.Data, &d); err != nil {
			t.Fatal(err)
		}
		return d.Merchant.Percent
	}
	same := func(label string, want float64) {
		lw := callWithID(f, f.srv.handleRepMerchants, "", rep, []string{"sales"}, "{}")
		if lw.Code != http.StatusOK {
			t.Fatalf("القائمة %d: %s", lw.Code, lw.Body.String())
		}
		dw := callWithID(f, f.srv.handleRepMerchantDetail, f.merchantID, rep, []string{"sales"}, "{}")
		if dw.Code != http.StatusOK {
			t.Fatalf("التفصيل %d: %s", dw.Code, dw.Body.String())
		}
		l, d := pct(lw.Body.Bytes(), true), pct(dw.Body.Bytes(), false)
		if l != d {
			t.Fatalf("%s: القائمة %v والتفصيل %v — يلزم أن يتطابقا", label, l, d)
		}
		if want >= 0 && l != want {
			t.Fatalf("%s: المتوقّع %v والوارد %v", label, want, l)
		}
	}
	// **بلا نسبةٍ خاصّة** — العامّةُ كما يحسبها التفصيل.
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET commission_percent = NULL WHERE id = $1`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	same("العامّة", -1)
	// **وبنسبةٍ خاصّة** — هي نفسُها.
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET commission_percent = 7.5 WHERE id = $1`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	same("الخاصّة", 7.5)
}
