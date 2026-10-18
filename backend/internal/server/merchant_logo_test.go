package server

import (
	"context"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestMerchantSettings_Logo **صاحبُ المتجر يضع شعارَه** (طلبُ المالك ٢٠٢٦-١٠-١٨) —
// صورتُه هو بنوع الشعار تُقبل، **وصورةُ غيره أو صورةُ صنفٍ تُردّ ولا تمسّ الشعار.**
func TestMerchantSettings_Logo(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	other := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET owner_user_id = $2 WHERE id = $1`,
		f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	media := func(kind, by string) string {
		var id string
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO media (kind, path, thumb_path, width, height, bytes, created_by)
			VALUES ($1, 'x.jpg', 'x_t.jpg', 10, 10, 100, $2) RETURNING id`, kind, by).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	logo := func() string {
		var id *string
		if err := f.pool.QueryRow(ctx, `SELECT logo_media_id::text FROM merchants WHERE id = $1`,
			f.merchantID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if id == nil {
			return ""
		}
		return *id
	}
	set := func(id string) int {
		w := f.call(f.srv.handleMerchantSettings, http.MethodPatch, "/merchant/stores/"+f.merchantID+"/settings",
			f.merchantID, owner, []string{"merchant"}, `{"logo_media_id":"`+id+`"}`)
		return w.Code
	}

	mine := media("merchant_logo", owner)
	if code := set(mine); code != http.StatusOK || logo() != mine {
		t.Fatalf("شعارُه رُدّ: %d، الشعار %q", code, logo())
	}
	for name, id := range map[string]string{
		"صورةُ غيره":  media("merchant_logo", other),
		"صورةُ صنف":   media("menu_item", owner),
		"معرّفٌ غريب": "not-a-uuid",
	} {
		if code := set(id); code != http.StatusBadRequest {
			t.Fatalf("%s قُبلت: %d", name, code)
		}
		if logo() != mine {
			t.Fatalf("%s بدّلت الشعار إلى %q", name, logo())
		}
	}
	// **وحفظُ حقلٍ آخر لا يمحو الشعار.**
	w := f.call(f.srv.handleMerchantSettings, http.MethodPatch, "/merchant/stores/"+f.merchantID+"/settings",
		f.merchantID, owner, []string{"merchant"}, `{"default_prep_minutes":20}`)
	if w.Code != http.StatusOK || logo() != mine {
		t.Fatalf("حفظُ مدّة التحضير: %d، الشعار %q", w.Code, logo())
	}
	// **وغيرُ صاحب المتجر لا يضع شعاراً عليه.**
	w = f.call(f.srv.handleMerchantSettings, http.MethodPatch, "/merchant/stores/"+f.merchantID+"/settings",
		f.merchantID, other, []string{"merchant"}, `{"logo_media_id":"`+media("merchant_logo", other)+`"}`)
	if w.Code == http.StatusOK || logo() != mine {
		t.Fatalf("غيرُ صاحبه بدّل الشعار: %d", w.Code)
	}
}
