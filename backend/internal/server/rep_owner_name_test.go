package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestRepStore_OwnerNameShown اسمُ صاحب المتجر يصل المندوبَ في القائمة والتفصيل (حقلُ قراءةٍ فقط)،
// وفارغُه null لا نصٌّ فارغ، ورقمُه يبقى في الردّ للاتّصال.
func TestRepStore_OwnerNameShown(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	rep := testdb.NewUser(t, f.pool, "sales")
	owner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx, `UPDATE users SET full_name = 'أبو أحمد' WHERE id = $1`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx,
		`UPDATE merchants SET owner_user_id = $2, sales_rep_user_id = $3 WHERE id = $1`,
		f.merchantID, owner, rep); err != nil {
		t.Fatal(err)
	}

	read := func(name string, w interface{ Bytes() []byte }) (*string, *string) {
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(w.Bytes(), &env); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		type row struct {
			ID         string  `json:"id"`
			OwnerName  *string `json:"owner_name"`
			OwnerPhone *string `json:"owner_phone"`
		}
		var one row
		var list []row
		if err := json.Unmarshal(env.Data, &list); err == nil {
			for _, r := range list {
				if r.ID == f.merchantID {
					return r.OwnerName, r.OwnerPhone
				}
			}
			t.Fatalf("%s: المتجر غير موجود في القائمة", name)
		}
		var d struct {
			Merchant row `json:"merchant"`
		}
		if err := json.Unmarshal(env.Data, &d); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		one = d.Merchant
		return one.OwnerName, one.OwnerPhone
	}

	check := func(label string, want string) {
		lw := callWithID(f, f.srv.handleRepMerchants, "", rep, []string{"sales"}, "{}")
		if lw.Code != http.StatusOK {
			t.Fatalf("القائمة %d: %s", lw.Code, lw.Body.String())
		}
		ln, lp := read("list", lw.Body)
		dw := callWithID(f, f.srv.handleRepMerchantDetail, f.merchantID, rep, []string{"sales"}, "{}")
		if dw.Code != http.StatusOK {
			t.Fatalf("التفصيل %d: %s", dw.Code, dw.Body.String())
		}
		dn, dp := read("detail", dw.Body)
		for name, got := range map[string]*string{"القائمة/" + label: ln, "التفصيل/" + label: dn} {
			if want == "" && got != nil {
				t.Fatalf("%s: المتوقّع null والوارد %q", name, *got)
			}
			if want != "" && (got == nil || *got != want) {
				t.Fatalf("%s: المتوقّع %q والوارد %v", name, want, got)
			}
		}
		if lp == nil || dp == nil {
			t.Fatalf("رقمُ صاحب المتجر اختفى من الردّ — يلزم للاتّصال")
		}
	}
	check("بالاسم", "أبو أحمد")

	if _, err := f.pool.Exec(ctx, `UPDATE users SET full_name = '  ' WHERE id = $1`, owner); err != nil {
		t.Fatal(err)
	}
	check("بلا اسم", "")
}
