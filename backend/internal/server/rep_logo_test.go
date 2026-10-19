package server

import (
	"context"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestRepSetLogo **المندوبُ يضع شعارَ عميله** (طلبُ المالك ٢٠٢٦-١٠-١٩) — صورتُه هو بنوع
// الشعار تُقبل، **وصورةُ غيره أو صورةُ صنفٍ تُردّ ولا تمسّ الشعار.** (والعميلُ يحرسه
// `repMenuGuard` على المسار — يفحصه `TestRepMenuGuard_OnlyOwnClients`.)
func TestRepSetLogo(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	rep := testdb.NewUser(t, f.pool, "sales")
	other := testdb.NewUser(t, f.pool, "sales")
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
		return f.call(f.srv.handleRepSetLogo, http.MethodPut, "/rep/stores/"+f.merchantID+"/logo",
			f.merchantID, rep, []string{"sales"}, `{"logo_media_id":"`+id+`"}`).Code
	}

	mine := media("merchant_logo", rep)
	if code := set(mine); code != http.StatusOK || logo() != mine {
		t.Fatalf("شعارُ المندوب رُدّ: %d، الشعار %q", code, logo())
	}
	for name, id := range map[string]string{
		"صورةُ مندوبٍ آخر": media("merchant_logo", other),
		"صورةُ صنف":        media("menu_item", rep),
		"معرّفٌ غريب":      "not-a-uuid",
	} {
		if code := set(id); code != http.StatusBadRequest {
			t.Fatalf("%s قُبلت: %d", name, code)
		}
		if logo() != mine {
			t.Fatalf("%s بدّلت الشعار إلى %q", name, logo())
		}
	}
}
