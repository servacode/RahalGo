package main

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestReset_ReseedsExpenseCategories **إعادةُ الضبط تعيد أبوابَ المصروف الثمانية.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤، المصروفات البند ٦.) **وكان الإفراغُ يمحوها ولا
// يعيدها** — فقائمةُ الباب على التجهيز فارغةٌ والحفظُ لا يعمل.
//
// **والإفراغُ في معاملةٍ تُرجَع** — فلا تُمسّ قاعدةُ الاختبار خارجَه.
func TestReset_ReseedsExpenseCategories(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	if _, err := tx.Exec(ctx, `TRUNCATE TABLE expense_categories CASCADE`); err != nil {
		t.Fatalf("الإفراغ: %v", err)
	}
	n, err := reseedDefaults(ctx, tx)
	if err != nil {
		t.Fatalf("البذر: %v", err)
	}
	var got int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM expense_categories WHERE active`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if n != 8 || got != 8 {
		t.Fatalf("بعد إعادة الضبط %d باباً (أُعلن %d) — يُنتظر ٨", got, n)
	}
	var rent, other int
	if err := tx.QueryRow(ctx, `
		SELECT (SELECT sort_order FROM expense_categories WHERE name = 'إيجار'),
		       (SELECT sort_order FROM expense_categories WHERE name = 'أخرى')`).Scan(&rent, &other); err != nil {
		t.Fatal(err)
	}
	if rent != 1 || other != 99 {
		t.Fatalf("الترتيب: إيجار %d وأخرى %d", rent, other)
	}
}
