package orders_test

// ══════════════════════════════════════════════════════════════════════
// **تصحيحُ 0434: ثمنُ بضاعة المشتريات القديمة يعود للمنصّة** (قرارُ المالك ٢٠٢٦-١٠-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// قِيس في الإنتاج: #1002 بضاعة ١١٠٠ وأجرة ١٠٠ ودخل محفظةَ السائق ١٢٠٠، والمكتبُ دفع
// ثمنَ البضاعة. **فالتصحيحُ ينقل ١١٠٠ من محفظته إلى الخزينة، مرّةً واحدة.**

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestFixOldPurchaseGoods_MovesGoodsOnce(t *testing.T) {
	_, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	pool := testdb.Pool(t)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	setMode(t, orderID, "purchase")
	if _, err := pool.Exec(ctx, `UPDATE orders SET status = 'delivered', custom_goods_amount = 1100,
		custom_fee = 100 WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	// **كما كانت التسويةُ القديمة**: بضاعةٌ وأجرٌ في محفظة السائق.
	if _, err := w.Apply(ctx, driver, 1200, "driver_earning", orderID, "قديم — بضاعةٌ وأجرٌ", nil); err != nil {
		t.Fatal(err)
	}
	before := cqTreasuryBalance(t)

	_, self, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(self), "..", "migrate", "migrations")
	for _, f := range []string{"0434_fix_old_purchase_goods.sql", "0435_goods_out_of_ledger.sql"} {
		sql, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 2; i++ { // **والثانيةُ لا تفعل شيئاً**
			if _, err := pool.Exec(ctx, string(sql)); err != nil {
				t.Fatalf("التصحيح %s: %v", f, err)
			}
		}
	}
	if b, _ := w.Balance(ctx, driver); b != 100 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 100 (أجرتُه وحدَها)", b)
	}
	// **و0435 يعكس رجعةَ الخزينة** — ثمنُ البضاعة مستردٌّ لا ربح، فلا يدخل الحسابات.
	if g := cqGoodsReturned(t, orderID); g != 0 {
		t.Fatalf("صافي رجعة البضاعة=%d — المنتظَر 0", g)
	}
	if d := cqTreasuryBalance(t) - before; d != 0 {
		t.Fatalf("الخزينةُ تحرّكت %d — **والمستردُّ لا يُحسب لها**", d)
	}
}

func cqTreasuryBalance(t *testing.T) int64 {
	t.Helper()
	var v int64
	if err := testdb.Pool(t).QueryRow(context.Background(),
		`SELECT COALESCE(sum(balance), 0) FROM wallets WHERE is_treasury`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}
