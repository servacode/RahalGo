package orders_test

// ══════════════════════════════════════════════════════════════════════
// **استرجاعُ طلبٍ خاصٍّ بعد التسليم = الأجرةُ المدفوعةُ وحدَها** — قرارُ المالك ٢٠٢٦-١١-١٠ (القرار ٦)
// ══════════════════════════════════════════════════════════════════════
//
// **كان لا يفعل شيئاً** (جردُ المال ٢٠٢٦-١١-١٠، العطب ١): فرعُ الخاصّ في `settle` لا يعرف
// `delivered → refunded` فيخرج صامتاً — الحالةُ «مُسترَدّ» والزبونُ لم يُردّ له قرش.
//
// وما يقع الآن:
//
//	محفظةُ الزبون  ← **ما دفعه للتوصيل فعلاً** (الأجرةُ بعد إعفاء العرض) — وصفرٌ لا يُقيَّد
//	ثمنُ البضاعة   ← **لا يُردّ من التطبيق** — يُسوّى نقداً خارجَه بين الزبون والمتجر
//	السائق         ← **أجرُه باقٍ** — أدّى الرحلة، وصندوقُه كما هو
//	الخزينة        ← **تتحمّل المردود** — تنزل بمقداره

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// crdFamilies **ثوابتُ يمسّها الاسترجاع** — ومعها «لا استردادَ مكرَّر».
var crdFamilies = []string{"FI-02.a", "FI-05.g", "FI-06", "FI-10", "FI-11", "FI-15"}

// crdGuard **صورةُ الخروق قبل السيناريو وفاحصُ ما استجدّ بعده.**
func crdGuard(t *testing.T) func() {
	t.Helper()
	ctx := context.Background()
	pool := testdb.Pool(t)
	base, err := fininv.Capture(ctx, pool, crdFamilies...)
	if err != nil {
		t.Fatalf("صورةُ الثوابت: %v", err)
	}
	return func() {
		t.Helper()
		after, err := fininv.Run(ctx, pool, crdFamilies...)
		if err != nil {
			t.Fatalf("فحصُ الثوابت: %v", err)
		}
		if fresh := base.New(after); len(fresh) > 0 {
			for _, v := range fresh {
				t.Errorf("خرقٌ جديد %s (%s): %v", v.Check.ID, v.Check.Name, v.Rows)
			}
			t.FailNow()
		}
	}
}

// crdFundTreasury **رصيدٌ للخزينة بقيدٍ في الدفتر** — لا بتعديلٍ خامٍ للرصيد (`fundTestTreasury`)،
// **فذاك رصيدٌ بلا قيدٍ يُقرأ خرقَ `FI-02.a` يتبدّل مع كلّ حركة.**
func crdFundTreasury(t *testing.T, w *wallet.Service) {
	t.Helper()
	var tid string
	if err := testdb.Pool(t).QueryRow(context.Background(),
		`SELECT user_id::text FROM wallets WHERE is_treasury LIMIT 1`).Scan(&tid); err != nil {
		t.Fatalf("الخزينة: %v", err)
	}
	mcsFund(t, w, tid, 100_000)
}

// crdRefunds **قيودُ الاسترجاع على الطلب** — عددُها ومجموعُها.
func crdRefunds(t *testing.T, orderID string) (count int, sum int64) {
	t.Helper()
	if err := testdb.Pool(t).QueryRow(context.Background(), `
		SELECT count(*), COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'refund'`, orderID).Scan(&count, &sum); err != nil {
		t.Fatalf("قيودُ الاسترجاع: %v", err)
	}
	return
}

// crdSnapshot **أرصدةُ الأطراف لحظةً واحدة.**
type crdSnapshot struct{ customer, driver, held, treasury int64 }

func crdTake(t *testing.T, balance func(string) int64, customer, driver string) crdSnapshot {
	t.Helper()
	return crdSnapshot{
		customer: balance(customer), driver: balance(driver),
		held: cqHeld(t, driver), treasury: cqTreasury(t),
	}
}

// crdRefundTwice **يسترجع الطلبَ بيد الأدمن، ثمّ يحاول ثانيةً** — والثانيةُ تُرَدّ ولا تُقيِّد.
func crdRefundTwice(t *testing.T, svc *orders.Service, admin, orderID string) {
	t.Helper()
	ctx := context.Background()
	if _, err := svc.Transition(ctx, admin, []string{"admin"}, orderID, "refunded", "اختبارُ الاسترجاع"); err != nil {
		t.Fatalf("الاسترجاع: %v", err)
	}
	_, _ = svc.Transition(ctx, admin, []string{"admin"}, orderID, "refunded", "اختبارُ الاسترجاع ثانيةً")
}

// ── أ · مشترياتٌ نقداً: تُردّ الأجرةُ وحدَها، والخزينةُ تتحمّلها ──────────────
//
//	بضاعةٌ ١٢٬٠٠٠ · أجرةٌ ٥٬٠٠٠ · حصّةُ المنصّة ١٠٪
//	⇒ الزبونُ +٥٬٠٠٠ · الخزينةُ −٥٬٠٠٠ · محفظةُ السائق وصندوقُه كما هما
func TestCustomRefundAfterDelivery_CashRefundsFeeOnly(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	admin := testdb.NewUser(t, pool, "admin")
	cqArmTreasury(t)
	crdFundTreasury(t, w)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	check := crdGuard(t)
	balance := func(u string) int64 { b, _ := w.Balance(ctx, u); return b }

	const goods, fee = 12_000, 5_000
	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", goods+fee, 1); err != nil {
		t.Fatalf("التأكيد نقداً: %v", err)
	}
	ptDeliver(t, svc, driver, orderID)
	before := crdTake(t, balance, customer, driver)

	crdRefundTwice(t, svc, admin, orderID)
	after := crdTake(t, balance, customer, driver)

	if got := after.customer - before.customer; got != fee {
		t.Fatalf("محفظةُ الزبون +%d — المنتظَر +%d (الأجرةُ وحدَها، والبضاعةُ خارجَ التطبيق)", got, fee)
	}
	if n, sum := crdRefunds(t, orderID); n != 1 || sum != fee {
		t.Fatalf("قيودُ الاسترجاع %d بمجموع %d — المنتظَر قيدٌ واحدٌ بـ %d", n, sum, fee)
	}
	if got := after.treasury - before.treasury; got != -fee {
		t.Fatalf("الخزينة تحرّكت %d — المنتظَر %d (تتحمّل المردود)", got, -fee)
	}
	if after.driver != before.driver {
		t.Fatalf("محفظةُ السائق %d ⇐ %d — وأجرُه باقٍ، أدّى الرحلة", before.driver, after.driver)
	}
	if after.held != before.held {
		t.Fatalf("صندوقُ السائق %d ⇐ %d — ولا يُمسّ", before.held, after.held)
	}
	check()
}

// ── ب · مشترياتٌ بالمحفظة: تُردّ الأجرةُ إلى المحفظة لا المجموعُ ─────────────
func TestCustomRefundAfterDelivery_WalletRefundsFeeOnly(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	admin := testdb.NewUser(t, pool, "admin")
	cqArmTreasury(t)
	crdFundTreasury(t, w)
	mcsFund(t, w, customer, 50_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	check := crdGuard(t)
	balance := func(u string) int64 { b, _ := w.Balance(ctx, u); return b }

	const goods, fee = 3_000, 2_000
	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", goods+fee, 1); err != nil {
		t.Fatalf("التأكيد بالمحفظة: %v", err)
	}
	ptDeliver(t, svc, driver, orderID)
	before := crdTake(t, balance, customer, driver)
	if before.customer != 50_000-goods-fee {
		t.Fatalf("محفظةُ الزبون بعد التسليم %d — المنتظَر %d", before.customer, 50_000-goods-fee)
	}

	crdRefundTwice(t, svc, admin, orderID)
	after := crdTake(t, balance, customer, driver)

	if got := after.customer - before.customer; got != fee {
		t.Fatalf("محفظةُ الزبون +%d — المنتظَر +%d", got, fee)
	}
	if n, sum := crdRefunds(t, orderID); n != 1 || sum != fee {
		t.Fatalf("قيودُ الاسترجاع %d بمجموع %d — المنتظَر قيدٌ واحدٌ بـ %d", n, sum, fee)
	}
	if got := after.treasury - before.treasury; got != -fee {
		t.Fatalf("الخزينة تحرّكت %d — المنتظَر %d", got, -fee)
	}
	if after.driver != before.driver || after.held != before.held {
		t.Fatalf("السائق تحرّك: محفظة %d ⇐ %d · صندوق %d ⇐ %d",
			before.driver, after.driver, before.held, after.held)
	}
	check()
}

// ── ج · أمانةٌ أجرتُها معفاةٌ كلُّها: لا شيءَ يُردّ ولا يُقيَّد ───────────────
func TestCustomRefundAfterDelivery_FreeDeliveryPostsNothing(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	admin := testdb.NewUser(t, pool, "admin")
	cqArmTreasury(t)
	crdFundTreasury(t, w)
	armFreeCampaign(t, []string{orders.FreeKindAmanah}, nil)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	mcsAmanah(t, orderID)
	check := crdGuard(t)
	balance := func(u string) int64 { b, _ := w.Balance(ctx, u); return b }

	if err := svc.AgreeCustom(ctx, orderID, driver, 0, 5_000); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 0, 1); err != nil {
		t.Fatalf("التأكيد بصفر: %v", err)
	}
	ptDeliver(t, svc, driver, orderID)
	before := crdTake(t, balance, customer, driver)

	crdRefundTwice(t, svc, admin, orderID)
	after := crdTake(t, balance, customer, driver)

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "refunded" {
		t.Fatalf("الحالة %q — المنتظَر refunded", status)
	}
	if n, _ := crdRefunds(t, orderID); n != 0 {
		t.Fatalf("قيودُ استرجاعٍ %d — والزبونُ لم يدفع شيئاً", n)
	}
	if after != before {
		t.Fatalf("تحرّك مالٌ في استرجاع طلبٍ مجّانيّ: %+v ⇐ %+v", before, after)
	}
	check()
}
