package orders_test

// ══════════════════════════════════════════════════════════════════════
// **مشترياتٌ فشلت بعد الشراء — خسارةٌ على المنصّة والمكتبُ يقرّر من يدفع**
// (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
//	ثمنٌ ٥٠٠ وأجرةٌ ١٠٠ نقداً · السائقُ اشترى ثمّ أنهاه المكتبُ عند الباب بذنب السائق
//	الخزينة −٥٠٠ · الدرج ٥٠٠ سلفةً رجعت · «على السائق كاملاً» ⇒ دينٌ ٥٠٠
//
// **وكان لا يُقيَّد شيء**: البضاعةُ «مالُ مكتبٍ» خارجَ الدفتر، فلا خسارةَ ولا دين.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// clgFailedPurchase **مشترياتٌ اشتراها السائقُ ثمّ أُنهيت عند الباب بذنبه.**
func clgFailedPurchase(t *testing.T, share string) (orderID, driver string) {
	t.Helper()
	svc, _, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	ptFeeBounds(t, exec)
	// **والخزينةُ تخسر هنا فتهبط تحت الصفر** — تُصفَّر قبل نزع وسمها (`clearTreasury`).
	clearTreasury(t, pool)
	cqArmTreasury(t)
	t.Cleanup(func() { clearTreasury(t, pool) })
	orderID = cqSeedOrder(t, nil, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, orderID, driver, 500, 100); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 600, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
	ops := testdb.NewUser(t, pool, "operations")
	if _, err := svc.ResolveDoor(ctx, ops, []string{"ops"}, orderID, orders.DoorResolution{
		Action: orders.DoorReturnToOffice, Fault: orders.FaultDriver, Reason: "customer_refused",
		Note: "السائقُ أضاع العنوان", DriverShare: share,
	}, nil); err != nil {
		t.Fatalf("الإنهاءُ عند الباب رُدّ: %v", err)
	}
	dropReturnTrip(t, pool, orderID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM financial_obligations WHERE party_kind = 'driver' AND party_id = $1`, driver)
		_, _ = pool.Exec(context.Background(), `UPDATE users SET driver_debt = 0 WHERE id = $1`, driver)
	})
	return orderID, driver
}

// clgLoss **ما خسرته الخزينةُ على الطلب** — سالبُ مجموع `platform_profit` بمرجعه.
func clgLoss(t *testing.T, orderID string) int64 {
	t.Helper()
	var v int64
	if err := testdb.Pool(t).QueryRow(context.Background(), `
		SELECT -COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'platform_profit'`, orderID).Scan(&v); err != nil {
		t.Fatalf("الخسارة: %v", err)
	}
	return v
}

// clgDebt **دينُ السائق بخسارة هذا الطلب.**
func clgDebt(t *testing.T, orderID, driver string) int64 {
	t.Helper()
	var v int64
	if err := testdb.Pool(t).QueryRow(context.Background(), `
		SELECT COALESCE(sum(amount), 0) FROM financial_obligations
		 WHERE party_kind = 'driver' AND party_id = $1 AND cause = 'failed_order_driver'
		   AND order_id = $2::uuid`, driver, orderID).Scan(&v); err != nil {
		t.Fatalf("الدين: %v", err)
	}
	return v
}

// TestCustomPurchase_FailedAfterPickup_PlatformLoss **المنصّةُ تتحمّل افتراضاً — والخسارةُ تُقيَّد.**
func TestCustomPurchase_FailedAfterPickup_PlatformLoss(t *testing.T) {
	orderID, driver := clgFailedPurchase(t, orders.DriverSharePlatform)
	if got := clgLoss(t, orderID); got != 500 {
		t.Fatalf("خسارةُ الخزينة=%d — المنتظَر 500 (ما دفعه السائقُ للمتجر)", got)
	}
	if got := ptAdvanceReturn(t, orderID); got != 500 {
		t.Fatalf("سلفةٌ رجعت للمكتب=%d — المنتظَر 500", got)
	}
	var storePaid int64
	if err := testdb.Pool(t).QueryRow(context.Background(),
		`SELECT COALESCE(custom_store_paid, 0) FROM orders WHERE id = $1`, orderID).Scan(&storePaid); err != nil {
		t.Fatal(err)
	}
	if storePaid != 500 {
		t.Fatalf("custom_store_paid=%d — المنتظَر 500", storePaid)
	}
	if got := clgDebt(t, orderID, driver); got != 0 {
		t.Fatalf("دينٌ على السائق %d — والمنصّةُ هي التي تتحمّل", got)
	}
	// **وصافي دفتر الطلب = النقدُ صفرٌ − السلفة** (FI-06.a).
	if got := cqLedgerSum(t, orderID); got != -500 {
		t.Fatalf("صافي دفتر الطلب=%d — المنتظَر −500", got)
	}
}

// TestCustomPurchase_FailedAfterPickup_DriverFull **«على السائق كاملاً» ⇒ دينٌ بالخسارة كلِّها.**
func TestCustomPurchase_FailedAfterPickup_DriverFull(t *testing.T) {
	orderID, driver := clgFailedPurchase(t, orders.DriverShareFull)
	if got := clgLoss(t, orderID); got != 500 {
		t.Fatalf("خسارةُ الخزينة=%d — المنتظَر 500", got)
	}
	if got := clgDebt(t, orderID, driver); got != 500 {
		t.Fatalf("دينُ السائق=%d — المنتظَر 500", got)
	}
}
