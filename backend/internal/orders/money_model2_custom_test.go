package orders_test

// ══════════════════════════════════════════════════════════════════════
// **النموذج ٢: الطلبُ الخاصُّ والدولار** (خطّةُ المال ٢٠٢٦-١١-١٠، الدفعة B7)
// ══════════════════════════════════════════════════════════════════════
//
// عطبا جرد المال في النموذج ٢: عمولةُ ثمنٍ بالدولار تُقبض ولو صفُر المستحقُّ بالليرة (العطب ٣)،
// وبضاعةٌ بالدولار ضاعت بعد الشراء لا تُردّ من الدرج بالليرة ولا تُقيَّد خسارة (العطب ٢).

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func m2Stamp(t *testing.T, orderID string) {
	t.Helper()
	if _, err := testdb.Pool(t).Exec(context.Background(),
		`UPDATE orders SET money_model = 2 WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
}

// ── العطب ٣: ثمنٌ بالدولار ومستحقٌّ بالليرة صفر ⇒ العمولةُ تُقبض في الصندوق ──
func TestModel2Custom_USDCommissionWhenCashDueZero(t *testing.T) {
	svc, _, customer, driver, _, orderID := ptFixture(t, "cash")
	ctx := context.Background()
	m2Stamp(t, orderID)
	usdSetRef(t, "100")
	merchant := ptMerchant(t, 10, "")
	if _, err := testdb.Pool(t).Exec(ctx, `UPDATE merchants SET exchange_rate = 100 WHERE id = $1`, merchant); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.LinkPartner(ctx, customer, orderID, merchant, nil); err != nil {
		t.Fatalf("الربط: %v", err)
	}
	if err := svc.AgreeCustomStepUSD(ctx, orderID, driver, "", 0, 100, usdPtr(1250)); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 100, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	// **الأجرةُ أعفاها عرضٌ كلَّها** — فالمستحقُّ بالليرة صفرٌ والدولارُ نقدٌ في يد الكابتن.
	if _, err := testdb.Pool(t).Exec(ctx, `UPDATE orders SET cash_due = 0 WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	heldBefore := cqHeld(t, driver)
	ptDeliver(t, svc, driver, orderID)
	r := usdRead(t, orderID)
	if r.commission != 125 {
		t.Fatalf("**العمولة %d** — المنتظَر 125 (١٠٪ من معادل ١٢٫٥٠$)", r.commission)
	}
	if got := cqHeld(t, driver) - heldBefore; got != 125 {
		t.Fatalf("**صندوقُ الكابتن %d** — المنتظَر العمولة 125", got)
	}
	if adv := ptAdvanceReturn(t, orderID); adv != 0 {
		t.Fatalf("سلفةُ درجٍ %d في النموذج ٢", adv)
	}
}

// ── العطب ٢: بضاعةٌ بالدولار ضاعت بعد الشراء ⇒ لا خسارةَ ولا سلفةَ درجٍ بالليرة ──
func TestModel2Custom_LostUSDGoodsNoLoss(t *testing.T) {
	svc, _, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	ptFeeBounds(t, exec)
	clearTreasury(t, pool)
	cqArmTreasury(t)
	t.Cleanup(func() { clearTreasury(t, pool) })
	usdSetRef(t, "100")
	orderID := cqSeedOrder(t, nil, customer, driver, "driver_defined", nil, true)
	m2Stamp(t, orderID)
	if err := svc.AgreeCustomStepUSD(ctx, orderID, driver, "", 0, 100, usdPtr(500)); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 100, 1); err != nil {
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
		Note: "دولار", DriverShare: orders.DriverShareFull,
	}, nil); err != nil {
		t.Fatalf("الإنهاءُ عند الباب: %v", err)
	}
	dropReturnTrip(t, pool, orderID)
	if got := clgLoss(t, orderID); got != 0 {
		t.Fatalf("**خسارةُ الخزينة %d** — النموذج ٢ لا خسارةَ بضاعةٍ فيه", got)
	}
	if got := ptAdvanceReturn(t, orderID); got != 0 {
		t.Fatalf("**سلفةٌ رجعت %d** — الكابتنُ دفع للمتجر من دولار جيبه لا من الدرج", got)
	}
	if got := cqLedgerSum(t, orderID); got != 0 {
		t.Fatalf("صافي الدفتر %d — المنتظَر صفر", got)
	}
	var debt int64
	_ = pool.QueryRow(ctx, `SELECT COALESCE(sum(amount), 0) FROM financial_obligations WHERE order_id = $1`, orderID).Scan(&debt)
	if debt != 0 {
		t.Fatalf("**دينُ بضاعةٍ على الكابتن %d** — النموذج ٢ بلا دين بضاعة", debt)
	}
}
