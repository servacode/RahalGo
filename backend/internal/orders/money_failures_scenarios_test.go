package orders_test

// ══════════════════════════════════════════════════════════════════════
// **عائلةُ الطلبات الفاشلة — من يربح ومن يخسر، بالدفتر** (جردُ ٢٠٢٦-١٠-٢٧)
// ══════════════════════════════════════════════════════════════════════
//
// ما لم يكن له اختبارٌ في هذه العائلة:
//
//	متجرٌ «نقداً» فشل بعد الاستلام   ←  السلفةُ ترجع من الدرج، والخسارةُ صافي المتجر
//	ومثلُه بذنب السائق بنسبة ٣٠٪      ←  دينٌ على السائق بنسبة الخسارة، ولا تعويض
//	زبونٌ دفع من محفظته وفشل بعد الاستلام ←  يُردّ له كلُّ ما دفع، والخزينةُ تخسر صافي المتجر
//	موافقةُ تعويض السائق بذنب المتجر  ←  محفظتُه + والخزينةُ − ونزاعٌ على المتجر، ودينُه يُسدّ أوّلاً
//
// **وكلُّ رقمٍ من `wallet_transactions` و`office_cash_entries` و`financial_obligations`**،
// **وثوابتُ FI تُصوَّر قبلُ وتُقاس بعدُ** — فما ظهر هو ذنبُ السيناريو وحدَه.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// mfsFamilies **ثوابتُ الدفتر التي يمسّها الفشل** — حفظُ الطلب والاحتباسُ والصندوق.
var mfsFamilies = []string{
	"FI-06.a", "FI-06.d",
	"FI-14.a", "FI-14.b", "FI-14.c", "FI-14.d", "FI-14.e", "FI-14.f",
	"FI-10.b", "FI-10.c",
}

func mfsCapture(t *testing.T, q fininv.Querier) fininv.Baseline {
	t.Helper()
	b, err := fininv.Capture(context.Background(), q, mfsFamilies...)
	if err != nil {
		t.Fatalf("تصويرُ الثوابت: %v", err)
	}
	return b
}

func mfsAssertGreen(t *testing.T, q fininv.Querier, base fininv.Baseline) {
	t.Helper()
	after, err := fininv.Run(context.Background(), q, mfsFamilies...)
	if err != nil {
		t.Fatalf("تشغيلُ الثوابت: %v", err)
	}
	if fresh := base.New(after); len(fresh) > 0 {
		t.Fatalf("خُرقت ثوابتُ الدفتر بهذا السيناريو: %+v", fresh)
	}
}

// mfsCashStoreFailedAtDoor **متجرٌ «نقداً» ١٠٪ · بضاعةٌ ١٠٠٬٠٠٠ · توصيلٌ ١٠٬٠٠٠ نقداً** —
// يُستلم (السائقُ يدفع للمتجر ٩٠٬٠٠٠ من السلفة) ثمّ يُنهيه المكتبُ عند الباب.
func mfsCashStoreFailedAtDoor(t *testing.T, in orders.DoorResolution) (f *cashFixture, m, owner, oid string, heldBefore int64) {
	t.Helper()
	ctx := context.Background()
	f = newCashFixture(t)
	m, owner = f.merchant(t, "cash", 10)
	heldBefore = f.held(t)
	oid = f.order(t, m, 100_000, 10_000, "cash")
	f.pickup(t, oid)
	if s := f.settlement(t, oid, m); s.state != "cash_paid" || s.amount != 90_000 {
		t.Fatalf("بعد الاستلام: %+v — يُنتظر cash_paid بـ90000 دفعه السائق", s)
	}
	for _, to := range []string{"on_the_way", "at_dropoff"} {
		f.transition(t, oid, to)
	}
	ops := testdb.NewUser(t, f.pool, "operations")
	in.Action = orders.DoorReturnToOffice
	if in.Note == "" {
		in.Note = "قرارُ المكتب"
	}
	if _, err := f.svc.ResolveDoor(ctx, ops, []string{"ops"}, oid, in, nil); err != nil {
		t.Fatalf("إنهاءُ المكتب رُدّ: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM driver_compensation_requests WHERE order_id = $1`, oid)
		_, _ = f.pool.Exec(c,
			`DELETE FROM financial_obligations WHERE party_kind = 'driver' AND party_id = $1`, f.driver)
	})
	return f, m, owner, oid, heldBefore
}

// TestMFS_CashStoreFailedAfterPickup_AdvanceReturned **ذنبُ الزبون، متجرٌ نقديٌّ قبض من السائق.**
//
//	الزبون   لا شيء (لم يدفع نقداً)          السائق  صندوقُه لا يتحرّك · تعويضٌ معلَّق
//	المتجر   يبقى معه ٩٠٬٠٠٠ (cash_paid)      الدرج   −٩٠٬٠٠٠ «سلفة رجعت للمكتب»
//	الخزينة  −٩٠٬٠٠٠ خسارةُ الطلب             والحسمُ «إلى المكتب» لا يحرّك شيئاً
func TestMFS_CashStoreFailedAfterPickup_AdvanceReturned(t *testing.T) {
	ctx := context.Background()
	pool := testdb.Pool(t)
	base := mfsCapture(t, pool)

	f, m, owner, oid, heldBefore := mfsCashStoreFailedAtDoor(t, orders.DoorResolution{
		Fault: orders.FaultCustomer, Reason: "customer_refused", CustomerAction: orders.CustomerNone,
	})

	if n, sum := f.drawer(t, "advance_return", oid); n != 1 || sum != 90_000 {
		t.Fatalf("ردُّ السلفة من الدرج: %d سطراً بمجموع %d — يُنتظر سطرٌ بـ90000", n, sum)
	}
	if got := f.held(t) - heldBefore; got != 0 {
		t.Fatalf("صندوقُ السائق تحرّك %d — ولم يقبض من الزبون شيئاً", got)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != -90_000 {
		t.Fatalf("ربحُ الخزينة على الطلب %d — يُنتظر −90000 (ما قبضه المتجر)", got)
	}
	if s := f.settlement(t, oid, m); s.state != "cash_paid" || s.reversed != 0 {
		t.Fatalf("المتجرُ النقديّ: %+v — مالُه ثبت عند الاستلام", s)
	}
	if got := f.obligation(t, m); got != 0 {
		t.Fatalf("دينٌ على المتجر %d ولم تُردّ إليه بضاعة", got)
	}
	if got := f.balance(t, owner); got != 0 {
		t.Fatalf("قُيّد في محفظة المتجر النقديّ %d", got)
	}
	if got := f.sumKind(t, oid, "refund"); got != 0 {
		t.Fatalf("رُدّ للزبون %d — ولم يدفع شيئاً", got)
	}
	if got := f.sumKind(t, oid, "driver_earning") + f.sumKind(t, oid, "commission"); got != 0 {
		t.Fatalf("أجرٌ أو عمولةٌ على طلبٍ لم يُسلَّم: %d", got)
	}
	var pending int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM driver_compensation_requests
		 WHERE order_id = $1 AND driver_id = $2 AND status = 'pending' AND fault = 'customer'`,
		oid, f.driver).Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if pending != 1 {
		t.Fatalf("طلباتُ التعويض المعلَّقة %d — يُنتظر واحدٌ بذنب الزبون", pending)
	}

	// **والحسمُ «إلى المكتب» لا يحرّك مالاً** — الخسارةُ مقيَّدةٌ منذ الفشل.
	dropReturnTrip(t, f.pool, oid)
	if err := f.svc.SettleGoods(ctx, oid, orders.GoodsToOffice, f.treasury, 0); err != nil {
		t.Fatalf("الحسم: %v", err)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != -90_000 {
		t.Fatalf("الحسمُ إلى المكتب حرّك الخزينة: %d", got)
	}
	if n, sum := f.drawer(t, "advance_return", oid); n != 1 || sum != 90_000 {
		t.Fatalf("الحسمُ كرّر ردَّ السلفة: %d · %d", n, sum)
	}
	mfsAssertGreen(t, pool, base)
}

// TestMFS_CashStoreFailedByDriver_PercentDebt **ذنبُ السائق بنسبة ٣٠٪ — والمتجرُ نقديّ.**
//
// الخسارةُ ٩٠٬٠٠٠ (ما قبضه المتجرُ من السلفة) ⇒ دينٌ على السائق ٢٧٬٠٠٠ بسبب
// `failed_order_driver`، **ولا طلبَ تعويضٍ له**، والخزينةُ تبقى −٩٠٬٠٠٠ (الدينُ يُستردّ لاحقاً).
func TestMFS_CashStoreFailedByDriver_PercentDebt(t *testing.T) {
	ctx := context.Background()
	pool := testdb.Pool(t)
	base := mfsCapture(t, pool)

	f, _, _, oid, heldBefore := mfsCashStoreFailedAtDoor(t, orders.DoorResolution{
		Fault: orders.FaultDriver, Reason: "driver_late",
		DriverShare: orders.DriverSharePercent, DriverPercent: 30,
	})

	var debt int64
	if err := f.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0) FROM financial_obligations
		 WHERE party_kind = 'driver' AND party_id = $1 AND cause = $2 AND order_id = $3::uuid`,
		f.driver, obligations.CauseFailedOrderDriver, oid).Scan(&debt); err != nil {
		t.Fatal(err)
	}
	if debt != 27_000 {
		t.Fatalf("دينُ السائق %d — يُنتظر 27000 (٣٠٪ من 90000)", debt)
	}
	var reqs int
	if err := f.pool.QueryRow(ctx,
		`SELECT count(*) FROM driver_compensation_requests WHERE order_id = $1`, oid).Scan(&reqs); err != nil {
		t.Fatal(err)
	}
	if reqs != 0 {
		t.Fatalf("كُتب للسائق %d طلبَ تعويض — وذنبُه لا تعويضَ فيه", reqs)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != -90_000 {
		t.Fatalf("الخزينة %d — يُنتظر −90000", got)
	}
	if n, sum := f.drawer(t, "advance_return", oid); n != 1 || sum != 90_000 {
		t.Fatalf("ردُّ السلفة %d · %d", n, sum)
	}
	if got := f.held(t) - heldBefore; got != 0 {
		t.Fatalf("صندوقُ السائق تحرّك %d — والدينُ التزامٌ لا قيدُ صندوق", got)
	}
	mfsAssertGreen(t, pool, base)
}

// TestMFS_WalletPaidFailedAfterPickup_CustomerRefunded **زبونٌ دفع ١١٠٬٠٠٠ من محفظته،
// ومتجرٌ «محفظة» ١٠٪ على شراءٍ ٩٠٬٠٠٠** — فشل عند الباب بذنب الزبون.
//
//	الزبون   +١١٠٬٠٠٠ «refund» (البضاعةُ والتوصيلُ كاملاً)
//	المتجر   +٨١٬٠٠٠ في محفظته عند الاستلام، تبقى له
//	السائق   لا أجر ولا صندوق — تعويضٌ معلَّقٌ مقترَحُه ٥٠٪ من الأجرة
//	المندوب  لا شيء (عمولتُه عند التسليم وحدَه)
//	الخزينة  −٨١٬٠٠٠
func TestMFS_WalletPaidFailedAfterPickup_CustomerRefunded(t *testing.T) {
	ctx := context.Background()
	f := setup(t, "at_pickup", 100_000, 10_000, 110_000)
	owner, _ := f.armTreasury(t)
	base := mfsCapture(t, f.pool)
	if _, err := f.wallet.Apply(ctx, f.customer, 110_000, "topup", "seed-"+f.orderID, "رصيدُ اختبار", nil); err != nil {
		t.Fatalf("الشحن: %v", err)
	}
	if _, err := f.wallet.Apply(ctx, f.customer, -110_000, "order_payment", f.orderID, "", nil); err != nil {
		t.Fatalf("خصمُ الطلب: %v", err)
	}
	heldBefore := f.held(t)

	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, to, ""); err != nil {
			t.Fatalf("الانتقالُ إلى %s: %v", to, err)
		}
	}
	f.failAtDoor(t, orders.FaultCustomer, "customer_absent")

	sum := func(kind string) int64 {
		var v int64
		if err := f.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
			WHERE ref = $1 AND kind = $2`, f.orderID, kind).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if got := f.balance(t, f.customer); got != 110_000 {
		t.Fatalf("رصيدُ الزبون %d — يُنتظر 110000 رجع كاملاً", got)
	}
	if got := sum("refund"); got != 110_000 {
		t.Fatalf("الاسترجاع %d لا 110000", got)
	}
	if got := f.balance(t, owner); got != 81_000 {
		t.Fatalf("محفظةُ المتجر %d — يُنتظر 81000 (90000 ناقصَ ١٠٪)", got)
	}
	if got := sum("platform_profit"); got != -81_000 {
		t.Fatalf("الخزينة %d — يُنتظر −81000", got)
	}
	if got := sum("driver_earning"); got != 0 {
		t.Fatalf("أجرُ سائقٍ على طلبٍ فشل: %d", got)
	}
	if got := f.held(t) - heldBefore; got != 0 {
		t.Fatalf("صندوقُ السائق تحرّك %d", got)
	}
	if got := f.balance(t, f.rep); got != 0 {
		t.Fatalf("قُيّد للمندوب %d على طلبٍ لم يُسلَّم", got)
	}
	if fault, suggested, found := f.pendingRequest(t); !found || fault != orders.FaultCustomer || suggested != 5_000 {
		t.Fatalf("التعويضُ المعلَّق (%v · %s · %d) — يُنتظر (true · customer · 5000)", found, fault, suggested)
	}
	mfsAssertGreen(t, f.pool, base)
}

// TestMFS_DriverCompensationApproved_MerchantFault **موافقةُ تعويض السائق بذنب المتجر.**
//
// الطلبُ فشل عند الباب بذنب المتجر (بضاعةٌ خطأ) ⇒ معلَّقٌ ٥٬٠٠٠. وعلى السائق دينٌ قديمٌ ٢٬٠٠٠:
//
//	السائق   +٥٬٠٠٠ «compensation» ثمّ −٢٬٠٠٠ سدادُ دينه ⇒ رصيدُه ٣٬٠٠٠ ودينُه صفر
//	الخزينة  −٥٬٠٠٠ «platform_expense» بمرجع الطلب · +٢٬٠٠٠ «platform_profit» بمرجع طلب التعويض
//	المتجر   نزاعٌ مفتوحٌ عليه بـ٥٬٠٠٠ — والخصمُ قرارُ إنسانٍ لاحقاً
//	ونصيبُ الخزينة على الطلب نفسِه (−٨١٬٠٠٠) لا يتغيّر بالموافقة
func TestMFS_DriverCompensationApproved_MerchantFault(t *testing.T) {
	ctx := context.Background()
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	_, treasury := f.armTreasury(t)
	base := mfsCapture(t, f.pool)
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM disputes WHERE order_id = $1`, f.orderID)
		_, _ = f.pool.Exec(c, `DELETE FROM warnings WHERE order_id = $1`, f.orderID)
		_, _ = f.pool.Exec(c, `DELETE FROM driver_compensation_requests WHERE order_id = $1`, f.orderID)
		_, _ = f.pool.Exec(c,
			`DELETE FROM financial_obligations WHERE party_kind = 'driver' AND party_id = $1`, f.driver)
	})
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, to, ""); err != nil {
			t.Fatalf("الانتقالُ إلى %s: %v", to, err)
		}
	}
	f.failAtDoor(t, orders.FaultMerchant, "customer_refused")
	if _, err := obligations.Create(ctx, f.pool, obligations.PartyDriver, f.driver, 2_000,
		obligations.CauseLegacy, "", nil); err != nil {
		t.Fatalf("زرعُ دين السائق: %v", err)
	}

	sum := func(ref, kind string) int64 {
		var v int64
		if err := f.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
			WHERE ref = $1 AND kind = $2`, ref, kind).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	profitBefore := sum(f.orderID, "platform_profit")
	if profitBefore != -81_000 {
		t.Fatalf("الخزينة قبل الموافقة %d — يُنتظر −81000", profitBefore)
	}

	var reqID string
	if err := f.pool.QueryRow(ctx, `
		SELECT id::text FROM driver_compensation_requests
		 WHERE order_id = $1 AND driver_id = $2 AND status = 'pending'`, f.orderID, f.driver).
		Scan(&reqID); err != nil {
		t.Fatalf("لا تعويضَ معلَّق: %v", err)
	}
	fin := testdb.NewUser(t, f.pool, "finance")
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	c, err := f.svc.CompensationByIDTx(ctx, tx, reqID)
	if err != nil || c == nil {
		t.Fatalf("قراءةُ التعويض: %v", err)
	}
	if c.Fault != orders.FaultMerchant || c.SuggestedAmount != 5_000 {
		t.Fatalf("التعويضُ (%s · %d) — يُنتظر (merchant · 5000)", c.Fault, c.SuggestedAmount)
	}
	if err := f.svc.ApproveCompensationTx(ctx, tx, c, 5_000, fin, "موافقة", false); err != nil {
		t.Fatalf("الموافقة: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	if got := sum(f.orderID, "compensation"); got != 5_000 {
		t.Fatalf("تعويضُ السائق %d لا 5000", got)
	}
	if got := f.balance(t, f.driver); got != 3_000 {
		t.Fatalf("رصيدُ السائق %d — يُنتظر 3000 (5000 ناقصَ دينه 2000)", got)
	}
	debt, err := obligations.Balance(ctx, f.pool, obligations.PartyDriver, f.driver)
	if err != nil {
		t.Fatal(err)
	}
	if debt != 0 {
		t.Fatalf("بقي على السائق %d — والتعويضُ يسدّ دينَه أوّلاً", debt)
	}
	if got := sum(f.orderID, "platform_expense"); got != -5_000 {
		t.Fatalf("نفقةُ الخزينة %d لا −5000", got)
	}
	if got := sum(reqID, "platform_profit"); got != 2_000 {
		t.Fatalf("ما عاد للخزينة من الدين %d لا 2000", got)
	}
	if got := sum(f.orderID, "platform_profit"); got != profitBefore {
		t.Fatalf("الموافقةُ حرّكت نصيبَ الخزينة على الطلب: %d → %d", profitBefore, got)
	}
	var claim int64
	if err := f.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0) FROM disputes
		 WHERE order_id = $1 AND party_role = 'merchant' AND merchant_id = $2`,
		f.orderID, f.merchantID).Scan(&claim); err != nil {
		t.Fatal(err)
	}
	if claim != 5_000 {
		t.Fatalf("النزاعُ على المتجر %d — يُنتظر 5000 بما دُفع", claim)
	}
	var status string
	var amount *int64
	if err := f.pool.QueryRow(ctx,
		`SELECT status, amount FROM driver_compensation_requests WHERE id = $1`, reqID).
		Scan(&status, &amount); err != nil {
		t.Fatal(err)
	}
	if status != orders.CompensationApproved || amount == nil || *amount != 5_000 {
		t.Fatalf("طلبُ التعويض (%s · %v) — يُنتظر (approved · 5000)", status, amount)
	}
	_ = treasury
	mfsAssertGreen(t, f.pool, base)
}
