package catalog

// **حذفُ التصنيف** (قرارُ المالك ٢٠٢٦-١٠-٠٦) — يُحذف ما لا متجرَ عليه، ويُردّ ما عليه متجر.
//
// **ووقع على الإنتاج أوّلَ يوم**: جملةُ تفريغ المرشَّحين سمّت جدولاً لا يوجد (`leads` لا
// `merchant_leads`) — **فسقط كلُّ حذفٍ بخطأٍ داخليّ ولم يكشفه بناءٌ ولا حارس.**

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestDeleteCategory_FreeDeletes_UsedRefused(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := NewService(pool, nil)
	actor := testdb.NewUser(t, pool, "admin")

	newCat := func(name string) string {
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO categories (name, icon, sort_order) VALUES ($1, '', 950) RETURNING id`, name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM categories WHERE id = $1`, id) })
		return id
	}

	// **تصنيفٌ لا متجرَ عليه يُحذف.**
	free := newCat("تصنيفُ حذفٍ حرّ")
	if err := svc.DeleteCategory(ctx, actor, free, "127.0.0.1"); err != nil {
		t.Fatalf("تصنيفٌ بلا متاجر لم يُحذف: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM categories WHERE id = $1`, free).Scan(&n); err != nil || n != 0 {
		t.Fatalf("بقي التصنيفُ بعد حذفه: n=%d err=%v", n, err)
	}

	// **وما عليه متجرٌ يُردّ بعدد متاجره.**
	used := newCat("تصنيفٌ عليه متجر")
	owner := testdb.NewUser(t, pool, "merchant")
	var mid string
	if err := pool.QueryRow(ctx,
		`INSERT INTO merchants (name, category_id, owner_user_id) VALUES ('متجرُ تصنيف', $1, $2) RETURNING id`,
		used, owner).Scan(&mid); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, mid) })
	err := svc.DeleteCategory(ctx, actor, used, "127.0.0.1")
	var app *httpx.AppError
	if !errors.As(err, &app) || app.Code != "category_in_use" {
		t.Fatalf("تصنيفٌ عليه متجرٌ لم يُردّ بـcategory_in_use: %v", err)
	}
}
