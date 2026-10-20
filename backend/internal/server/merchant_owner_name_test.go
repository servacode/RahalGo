package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestUpdateMerchant_OwnerNameSavedAndShown **اسمُ صاحب المتجر يُحفظ ويظهر في التعديل**
// (بلاغُ المالك ٢٠٢٦-١٠-٢٠) — ولو كان حسابُه قائماً.
func TestUpdateMerchant_OwnerNameSavedAndShown(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	admin := testdb.NewUser(t, f.pool, "admin")
	owner := testdb.NewUser(t, f.pool, "merchant")
	var phone string
	if err := f.pool.QueryRow(ctx, `UPDATE users SET full_name = 'قديم' WHERE id = $1 RETURNING phone`,
		owner).Scan(&phone); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET owner_user_id = $2 WHERE id = $1`, f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	w := f.call(f.srv.handleUpdateMerchant, http.MethodPatch, "/admin/merchants/"+f.merchantID,
		f.merchantID, admin, []string{"admin"},
		`{"owner_phone":"`+phone+`","owner_name":"أبو أحمد"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("الحفظ ردّ %d: %s", w.Code, w.Body.String())
	}
	var name string
	_ = f.pool.QueryRow(ctx, `SELECT full_name FROM users WHERE id = $1`, owner).Scan(&name)
	if name != "أبو أحمد" {
		t.Fatalf("اسمُ صاحب المتجر لم يُحفظ: %q", name)
	}
	var out struct {
		Data struct {
			OwnerName *string `json:"owner_name"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out.Data.OwnerName == nil || *out.Data.OwnerName != "أبو أحمد" {
		t.Fatalf("الاسمُ لا يعود مع المتجر — فيُفتح التعديلُ فارغاً: %s", w.Body.String())
	}
}
