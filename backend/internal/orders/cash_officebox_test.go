package orders_test

import (
	"context"
	"testing"
)

// TestTREASURY_MerchantCashPaidLeavesOfficeCashbox **نقدٌ دُفع لمتجرٍ خرج من صندوق
// المكتب** (قرارُ المالك ٢٠٢٦-١٠-٠٤ — الخزينة): سطرٌ خارجٌ بمرجع التسوية وبمقدار
// ما دُفع، في معاملة القيد نفسِها — ومرّةً واحدةً ولو أُعيد التأكيد.
func TestTREASURY_MerchantCashPaidLeavesOfficeCashbox(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 100_000, 0, "cash")
	f.pickup(t, oid)
	sid := f.settlementID(t, oid, m)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := f.svc.MarkCashSettlementPaid(ctx, sid, f.treasury, "دُفع", ""); err != nil {
			t.Fatalf("تأكيدُ الدفع فشل: %v", err)
		}
	}
	var n int
	var sum int64
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*), COALESCE(sum(amount), 0) FROM office_cash_entries
		 WHERE source = 'merchant_cash_paid' AND direction = 'out' AND ref = $1`, sid).
		Scan(&n, &sum); err != nil {
		t.Fatal(err)
	}
	if n != 1 || sum != 90_000 {
		t.Fatalf("سطورُ الصندوق %d بمجموع %d — يُنتظر سطرٌ واحدٌ بـ90000", n, sum)
	}
}
