package catalog

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestUpdateMerchant_Featured **«متجر مميّز» وترتيبُه يُحفظان من تعديل الأدمن ويُقيَّدان.**
//
// (قرارُ المالك ٢٠٢٦-١٠-١٨.) **وتعديلٌ لا يرسلهما لا يمسّهما**، **وتعديلٌ لا
// يغيّرهما لا يكتب حدثاً.**
func TestUpdateMerchant_Featured(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := NewService(pool, identity.NewService(identity.NewRepo(pool), nil, nil, nil, "test-secret", slog.New(slog.NewTextHandler(io.Discard, nil))))
	actor := testdb.NewUser(t, pool, "admin")

	var catID, id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO categories (name) VALUES ('تصنيف المميز') RETURNING id`).Scan(&catID); err != nil {
		t.Fatalf("تعذّر التصنيف: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, status)
		VALUES ('متجر مميز للتعديل', $1, 10, 'active') RETURNING id`, catID).Scan(&id); err != nil {
		t.Fatalf("تعذّر المتجر: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, id) })

	events := func() int {
		var n int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM audit_log
			WHERE action = 'admin.merchant_featured' AND entity = 'merchant' AND entity_id = $1`, id).Scan(&n); err != nil {
			t.Fatalf("تعذّرت قراءة التدقيق: %v", err)
		}
		return n
	}

	on, order := true, 3
	m, err := svc.UpdateMerchant(ctx, actor, id, MerchantInput{Featured: &on, FeaturedOrder: &order}, "127.0.0.1")
	if err != nil {
		t.Fatalf("التعديل: %v", err)
	}
	if !m.Featured || m.FeaturedOrder != 3 {
		t.Fatalf("بعد التعديل featured=%v order=%d، والمتوقّع true و٣", m.Featured, m.FeaturedOrder)
	}
	if n := events(); n != 1 {
		t.Fatalf("أحداثُ «المميّز» = %d، والمتوقّع ١", n)
	}

	// **تعديلُ الاسم وحدَه لا يمسّهما ولا يكتب حدثاً.**
	name := "متجر مميز للتعديل ٢"
	m, err = svc.UpdateMerchant(ctx, actor, id, MerchantInput{Name: &name}, "127.0.0.1")
	if err != nil {
		t.Fatalf("تعديلُ الاسم: %v", err)
	}
	if !m.Featured || m.FeaturedOrder != 3 {
		t.Fatalf("تعديلُ الاسم مسّ المميّز: featured=%v order=%d", m.Featured, m.FeaturedOrder)
	}
	// **وإرسالُ القيمة نفسِها لا يكتب حدثاً.**
	if _, err := svc.UpdateMerchant(ctx, actor, id, MerchantInput{Featured: &on}, "127.0.0.1"); err != nil {
		t.Fatalf("إعادةُ القيمة: %v", err)
	}
	if n := events(); n != 1 {
		t.Fatalf("أحداثُ «المميّز» بعد تعديلٍ بلا تغيير = %d، والمتوقّع ١", n)
	}

	off := false
	m, err = svc.UpdateMerchant(ctx, actor, id, MerchantInput{Featured: &off}, "127.0.0.1")
	if err != nil {
		t.Fatalf("الإطفاء: %v", err)
	}
	if m.Featured {
		t.Fatal("بقي مميّزاً بعد الإطفاء")
	}
	if n := events(); n != 2 {
		t.Fatalf("أحداثُ «المميّز» بعد الإطفاء = %d، والمتوقّع ٢", n)
	}
}
