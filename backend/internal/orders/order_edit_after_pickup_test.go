package orders_test

// ══════════════════════════════════════════════════════════════════════
// **تعديلُ الطلب العاديّ بعد الاستلام** (قرارُ المالك ٢٠٢٦-١١-٠٣، المرحلة ٢)
// ══════════════════════════════════════════════════════════════════════
//
// عند الاستلام دفع السائقُ للمتجر «نقداً» حقَّه (أو قُيّد في محفظة متجر «محفظة»).
// **فكلُّ تعديلٍ بعده يُقاس**: فرقُ حقّ المتجر بقيدٍ جديد، ومن دفعه، والمجموعُ والنقد،
// **ثمّ يُسلَّم الطلب وثوابتُ الدفتر كلُّها خضراء** — ثمّ يُسترجع أو يُصحَّح «انلغى»
// ويبقى الدفترُ أخضر.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// apChecks **ثوابتُ الدفتر التي يمسّها التعديل** — بعد كلّ خطوة.
var apChecks = []string{
	"FI-03.b", "FI-03.c", "FI-05.c", "FI-06.a", "FI-06.b", "FI-10.b", "FI-10.c",
	"FI-13.a", "FI-13.c",
	"FI-14.a", "FI-14.b", "FI-14.c", "FI-14.d", "FI-14.e", "FI-14.f", "FI-14.h",
}

func (f *cashFixture) green(t *testing.T, when string) {
	t.Helper()
	v, err := fininv.Run(context.Background(), f.pool, apChecks...)
	if err != nil {
		t.Fatalf("%s: تعذّر تشغيلُ الثوابت: %v", when, err)
	}
	if len(v) != 0 {
		t.Fatalf("%s: خُرقت ثوابتُ FI: %+v", when, v)
	}
}

func (f *cashFixture) line(t *testing.T, orderID string) string {
	t.Helper()
	return f.exec(t, `SELECT id::text FROM order_items WHERE order_id = $1 AND NOT returned ORDER BY name LIMIT 1`, orderID)
}

func (f *cashFixture) edit(t *testing.T, orderID string, req orders.EditRequest) orders.EditResult {
	t.Helper()
	if req.Reason == "" {
		req.Reason = "اختبار"
	}
	_, res, err := f.svc.EditOrder(context.Background(), f.treasury, orderID, req, true, nil)
	if err != nil {
		t.Fatalf("التعديل: %v", err)
	}
	return res
}

func (f *cashFixture) orderRow(t *testing.T, orderID string) (total, cashDue, walletPaid int64) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(),
		`SELECT total, cash_due, wallet_paid FROM orders WHERE id = $1`, orderID).
		Scan(&total, &cashDue, &walletPaid); err != nil {
		t.Fatal(err)
	}
	return
}

func (f *cashFixture) adjusted(t *testing.T, orderID, merchantID string) int64 {
	t.Helper()
	var v int64
	if err := f.pool.QueryRow(context.Background(), `
		SELECT adjusted_amount FROM merchant_settlements WHERE order_id = $1 AND merchant_id = $2`,
		orderID, merchantID).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// ── المكتبُ وحدَه بقدرته — وبلا قدرةٍ يُردّ ولا يتغيّر شيء ───────────────────
func TestOrderEditAP_NeedsCapability(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	f.pickup(t, oid)
	_, _, err := f.svc.EditOrder(context.Background(), f.driver, oid, orders.EditRequest{
		Reason: "بلا قدرة", Lines: []orders.EditLine{{ItemID: f.line(t, oid), Qty: 2}},
	}, false, nil)
	if !errors.Is(err, orders.ErrOrderEditAfterPickup) {
		t.Fatalf("بلا قدرة: %v — يُنتظر order_edit_after_pickup", err)
	}
	if total, _, _ := f.orderRow(t, oid); total != 60_000 {
		t.Fatalf("تغيّر المجموع %d", total)
	}
}

// ── متجرٌ «نقداً»: صنفٌ يُضاف بعد الاستلام ⇒ السائقُ يدفع الفرق ويقبض الصحيح ──
func TestOrderEditAP_CashStoreAddItem(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	add := editMenuItem(t, f, m, "إضافة", 5_000)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	heldBefore := f.held(t)
	f.pickup(t, oid) // دفع السائقُ ٤٥٬٠٠٠

	res := f.edit(t, oid, orders.EditRequest{Add: []orders.EditAdd{{MenuItemID: add, Qty: 1}}})
	if len(res.Stores) != 1 || res.Stores[0].DriverPays != 4_500 || res.Stores[0].NewDue != 49_500 {
		t.Fatalf("فرقُ المتجر: %+v — يُنتظر 4500 يدفعها السائق", res.Stores)
	}
	if total, cash, _ := f.orderRow(t, oid); total != 65_000 || cash != 65_000 {
		t.Fatalf("المجموع %d والنقد %d — يُنتظر 65000", total, cash)
	}
	if got := f.adjusted(t, oid, m); got != 4_500 {
		t.Fatalf("adjusted_amount=%d", got)
	}
	if got := f.sumKind(t, oid, "merchant_cash_paid"); got != -49_500 {
		t.Fatalf("ما دفعه السائقُ في الدفتر %d لا -49500", got)
	}
	f.green(t, "بعد التعديل")
	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	if got := f.held(t) - heldBefore; got != 65_000-49_500 {
		t.Fatalf("صندوقُ السائق زاد %d لا %d", got, 65_000-49_500)
	}
	// **العمولةُ على البضاعة الجديدة**: ١٠٪ من ٥٥٬٠٠٠.
	var commission int64
	_ = f.pool.QueryRow(context.Background(), `SELECT platform_commission FROM orders WHERE id = $1`, oid).Scan(&commission)
	if commission != 5_500 {
		t.Fatalf("عمولةُ المنصّة %d لا 5500", commission)
	}
	f.green(t, "بعد التسليم")
}

// ── «المتجر رجّع المصاري»: نقصٌ يُردّ للسائق، ثمّ استرجاعٌ بعد التسليم ─────────
func TestOrderEditAP_CashStoreRefundedThenRefund(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	ctx := context.Background()
	line := f.line(t, oid)
	// قبل الاستلام: ×٢ ⇒ ١٠٠٬٠٠٠.
	if _, _, err := f.svc.EditOrder(ctx, f.treasury, oid, orders.EditRequest{
		Reason: "قبل", Lines: []orders.EditLine{{ItemID: line, Qty: 2}}}, false, nil); err != nil {
		t.Fatal(err)
	}
	heldBefore := f.held(t)
	f.pickup(t, oid) // دفع ٩٠٬٠٠٠

	res := f.edit(t, oid, orders.EditRequest{StoreRefunded: true,
		Lines: []orders.EditLine{{ItemID: line, Qty: 1}}})
	if res.Stores[0].DriverPays != -45_000 || res.NewTotal != 60_000 {
		t.Fatalf("النقص: %+v", res)
	}
	f.green(t, "بعد النقص")
	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	if got := f.held(t) - heldBefore; got != 60_000-45_000 {
		t.Fatalf("صندوقُ السائق زاد %d لا 15000", got)
	}
	f.green(t, "بعد التسليم")
	// **الاسترجاعُ يعكس الحقَّ بتصحيحه**: المدفوعُ ٤٥٬٠٠٠ يصير ديناً على المتجر.
	f.transition(t, oid, "refunded")
	if got := f.obligation(t, m); got != 45_000 {
		t.Fatalf("دينُ المتجر بعد الاسترجاع %d لا 45000", got)
	}
	f.green(t, "بعد الاسترجاع")
}

// ── الزبونُ رجّع صنفاً والمنصّةُ تتحمّل: حقُّ المتجر كما هو ─────────────────────
func TestOrderEditAP_CustomerReturnedPlatformBears(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	ctx := context.Background()
	line := f.line(t, oid)
	if _, _, err := f.svc.EditOrder(ctx, f.treasury, oid, orders.EditRequest{
		Reason: "قبل", Lines: []orders.EditLine{{ItemID: line, Qty: 2}}}, false, nil); err != nil {
		t.Fatal(err)
	}
	heldBefore := f.held(t)
	f.pickup(t, oid)
	res := f.edit(t, oid, orders.EditRequest{Lines: []orders.EditLine{{ItemID: line, Qty: 1}}})
	if len(res.Stores) != 0 || res.NewTotal != 60_000 || res.Changes[0].Returned != 1 {
		t.Fatalf("المرتجع: %+v — يُنتظر بلا فرقٍ للمتجر ومجموعاً 60000", res)
	}
	var returned int
	_ = f.pool.QueryRow(ctx, `SELECT COALESCE(sum(qty),0) FROM order_items WHERE order_id = $1 AND returned AND unit_price = 0`, oid).Scan(&returned)
	if returned != 1 {
		t.Fatalf("سطرُ المرتجع %d", returned)
	}
	if s := f.settlement(t, oid, m); s.amount != 90_000 || f.adjusted(t, oid, m) != 0 {
		t.Fatalf("حقُّ المتجر تغيّر: %+v", s)
	}
	f.green(t, "بعد المرتجع")
	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	// السائقُ دفع ٩٠٬٠٠٠ وقبض ٦٠٬٠٠٠: **صندوقُه لا يهبط، والسلفةُ ٣٠٬٠٠٠ تُردّ من الدرج.**
	if got := f.held(t) - heldBefore; got != 0 {
		t.Fatalf("صندوقُ السائق تحرّك %d", got)
	}
	if n, sum := f.drawer(t, "advance_return", oid); n != 1 || sum != 30_000 {
		t.Fatalf("سلفةُ الدرج %d/%d — يُنتظر 30000", n, sum)
	}
	f.green(t, "بعد التسليم")
}

// ── متجرٌ «محفظة»: زيادةٌ ثمّ نقصٌ أكبرُ من رصيده ⇒ الباقي دين ───────────────────
func TestOrderEditAP_WalletStoreDebt(t *testing.T) {
	f := newCashFixture(t)
	m, owner := f.merchant(t, "wallet", 10)
	ctx := context.Background()
	oid := f.order(t, m, 50_000, 10_000, "wallet")
	line := f.line(t, oid)
	f.pickup(t, oid) // ٤٥٬٠٠٠ في محفظته

	res := f.edit(t, oid, orders.EditRequest{Lines: []orders.EditLine{{ItemID: line, Qty: 3}}})
	if res.Stores[0].NewDue != 135_000 {
		t.Fatalf("الزيادة: %+v", res.Stores)
	}
	if got := f.balance(t, owner); got != 135_000 {
		t.Fatalf("محفظةُ المتجر %d لا 135000", got)
	}
	f.green(t, "بعد الزيادة")
	// سحب المتجرُ أكثرَ رصيده فبقي ٥٬٠٠٠.
	if _, err := f.wallet.Apply(ctx, owner, -130_000, "payout", "test-payout-"+oid, "سحب", nil); err != nil {
		t.Fatal(err)
	}
	res = f.edit(t, oid, orders.EditRequest{StoreRefunded: true, Lines: []orders.EditLine{{ItemID: line, Qty: 1}}})
	if res.Stores[0].Debt != 85_000 || res.Stores[0].NewDue != 45_000 {
		t.Fatalf("النقص: %+v — يُنتظر دين 85000", res.Stores)
	}
	if got := f.balance(t, owner); got != 0 {
		t.Fatalf("محفظةُ المتجر %d لا 0", got)
	}
	if got := f.obligation(t, m); got != 85_000 {
		t.Fatalf("الدين %d لا 85000", got)
	}
	f.green(t, "بعد النقص")
	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	f.green(t, "بعد التسليم")
	// **والاسترجاعُ لا يأخذ الدينَ مرّتين**: يُستردّ ما بقي من المقيَّد ناقصَ الدين.
	f.transition(t, oid, "refunded")
	if got := f.obligation(t, m); got != 85_000+45_000 {
		t.Fatalf("الدين بعد الاسترجاع %d لا 130000 (85000 + حقُّه 45000)", got)
	}
	f.green(t, "بعد الاسترجاع")
}

// ── زبونٌ دفع من المحفظة وزاد طلبُه ورصيدُه لا يكفي ⇒ الباقي نقد ─────────────────
func TestOrderEditAP_WalletCustomerShortfallBecomesCash(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	add := editMenuItem(t, f, m, "إضافة", 20_000)
	oid := f.orderWalletPaid(t, m, 50_000, 10_000)
	ctx := context.Background()
	if _, err := f.wallet.Apply(ctx, f.customer, 5_000, "topup", "seed2-"+oid, "رصيد", nil); err != nil {
		t.Fatal(err)
	}
	heldBefore := f.held(t)
	f.pickup(t, oid)
	res := f.edit(t, oid, orders.EditRequest{Add: []orders.EditAdd{{MenuItemID: add, Qty: 1}}})
	if res.Charged != 5_000 || res.CashDue != 15_000 || res.NewTotal != 80_000 {
		t.Fatalf("الزيادة: %+v — يُنتظر خصم 5000 ونقد 15000", res)
	}
	if total, cash, wp := f.orderRow(t, oid); total != 80_000 || cash != 15_000 || wp != 65_000 {
		t.Fatalf("الطلب %d/%d/%d", total, cash, wp)
	}
	f.green(t, "بعد التعديل")
	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	// دفع السائقُ ٤٥٬٠٠٠ + ١٨٬٠٠٠ وقبض ١٥٬٠٠٠ ⇒ صندوقُه كما هو والسلفةُ ٤٨٬٠٠٠ من الدرج.
	if got := f.held(t) - heldBefore; got != 0 {
		t.Fatalf("صندوقُ السائق تحرّك %d", got)
	}
	f.green(t, "بعد التسليم")
}

// ── «تصحيح: الطلب انلغى» بعد طلبٍ عُدّل ⇒ الدفترُ أخضر ─────────────────────────
func TestOrderEditAP_ThenUndeliver(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	add := editMenuItem(t, f, m, "إضافة", 5_000)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	f.pickup(t, oid)
	f.edit(t, oid, orders.EditRequest{Add: []orders.EditAdd{{MenuItemID: add, Qty: 2}}})
	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	if _, _, err := f.svc.Undeliver(context.Background(), f.treasury, []string{"admin"}, oid,
		"سُجّل خطأً", nil); err != nil {
		t.Fatalf("التصحيح: %v", err)
	}
	f.green(t, "بعد «انلغى»")
}

// ── بعد الاستلام: لا متجرَ جديد، ولا تعديلَ بعد التسليم ───────────────────────
func TestOrderEditAP_Guards(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	other, _ := f.merchant(t, "cash", 10)
	foreign := editMenuItem(t, f, other, "غريب", 1_000)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	ctx := context.Background()
	f.pickup(t, oid)
	_, _, err := f.svc.EditOrder(ctx, f.treasury, oid, orders.EditRequest{Reason: "غريب",
		Add: []orders.EditAdd{{MenuItemID: foreign, Qty: 1}}}, true, nil)
	if !errors.Is(err, orders.ErrOrderEditStore) {
		t.Fatalf("متجرٌ جديد بعد الاستلام: %v", err)
	}
	_, _, err = f.svc.EditOrder(ctx, f.treasury, oid, orders.EditRequest{Reason: "آخر",
		StoreRefunded: true, Lines: []orders.EditLine{{ItemID: f.line(t, oid), Qty: 0}}}, true, nil)
	if !errors.Is(err, orders.ErrOrderEditLast) {
		t.Fatalf("آخرُ صنف: %v", err)
	}
	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	_, _, err = f.svc.EditOrder(ctx, f.treasury, oid, orders.EditRequest{Reason: "بعد",
		Lines: []orders.EditLine{{ItemID: f.line(t, oid), Qty: 2}}}, true, nil)
	if !errors.Is(err, orders.ErrOrderEditClosed) {
		t.Fatalf("بعد التسليم: %v", err)
	}
	f.green(t, "بعد الردود")
}
