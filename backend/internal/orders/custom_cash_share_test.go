package orders_test

// الطلبُ الخاصُّ **نقداً** — قرارُ المالك ٢٠٢٦-٠٩-٢٩ («Cash Custom إن وجد»).
//
// # وكان فراغاً ماليّاً كاملاً
//
// **قِيس ٢٠٢٦-٠٩-٢٩**: `ConfirmQuote` لا تضبط `cash_due` أبداً، والتسويةُ
// تخرج مبكّراً لغير المحفظة — **فلا صندوقَ نقدٍ للسائق، ولا قيدَ في الدفتر،
// ولا نصيبَ للمنصّة.** **يقبض المبلغَ ولا أثرَ له في المنصّة إطلاقاً.**
//
// # والفرقُ عن المحفظة
//
// **لا خصمَ من الزبون** — دفع نقداً بيده، **وقيدٌ عليه يخصم مرّتين.**
// **والمالُ كلُّه في صندوق السائق**، ومحفظتُه تأخذ ما يستحقّ، **والبقيّةُ
// ذمّةٌ عليه للمنصّة** — وهو عينُ ما يقع في الطلب العاديِّ النقديّ.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// cqHeld ما في صندوق نقد السائق.
func cqHeld(t *testing.T, driver string) int64 {
	t.Helper()
	pool := testdb.Pool(t)
	var v int64
	if err := pool.QueryRow(context.Background(),
		`SELECT COALESCE(held, 0) FROM driver_cash_boxes WHERE driver_id = $1`,
		driver).Scan(&v); err != nil {
		return 0 // **ولا صندوقَ بعدُ = صفر** — وهي الحالُ قبل أوّل قبض.
	}
	return v
}

// ── أ · النقدُ يُقيَّد كاملاً، ونصيبُ المنصّة ذمّةٌ على السائق ─────────────
func TestCustomCash_CollectedAndShared(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)

	const goods, fee = 12_000, 5_000
	platform, driverFee := shareOf(fee, 10) // 500 · 4,500
	heldBefore := cqHeld(t, driver)

	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", goods+fee, 1); err != nil {
		t.Fatalf("التأكيد نقداً: %v", err)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}

	// **وكلُّ المقبوضِ في صندوقه** — **وصندوقٌ لا يعرف ما قُبض لا يحرس
	// سقفاً ولا يُقاصّ.**
	if got := cqHeld(t, driver) - heldBefore; got != goods+fee {
		t.Fatalf("صندوقُ السائق=%d — المنتظَر %d (كلُّ المقبوض)", got, goods+fee)
	}
	// **ومحفظتُه تأخذ ما يستحقّ** — بضاعتُه وأجرُه بعد النصيب.
	if b, _ := w.Balance(ctx, driver); b != goods+driverFee {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر %d", b, goods+driverFee)
	}
	// **ولا يُخصم الزبونُ** — دفع نقداً بيده.
	if b, _ := w.Balance(ctx, customer); b != 0 {
		t.Fatalf("محفظةُ الزبون=%d — **ولا يُخصم من دفع نقداً**", b)
	}
	// **والخزينةُ تأخذ نصيبَها** — ذمّةً على السائق حتّى يُقاصّ.
	if got := cqTreasury(t); got != platform {
		t.Fatalf("الخزينة=%d — المنتظَر %d", got, platform)
	}
	// **والمعادلةُ تُغلق**: ما وصل الأطرافَ = ما قُبض.
	if sum := cqLedgerSum(t, orderID); sum != goods+fee {
		t.Fatalf("مجموعُ القيود=%d — المنتظَر %d (المقبوضُ نقداً يدخل من خارج الدفتر)",
			sum, goods+fee)
	}
}

// ── ب · والبضاعةُ لا تُؤخذ منها نسبةٌ نقداً كما لا تُؤخذ محفظةً ──────────
func TestCustomCash_GoodsNeverTaxed(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)

	const goods, fee = 19_000, 1_000
	platform, driverFee := shareOf(fee, 10) // 100 · 900

	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", goods+fee, 1); err != nil {
		t.Fatalf("التأكيد نقداً: %v", err)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
	if got := cqTreasury(t); got != platform {
		t.Fatalf("الخزينة=%d — المنتظَر %d. **ولو أُخذت من المجموع لكانت %d**",
			got, platform, (goods+fee)*10/100)
	}
	if b, _ := w.Balance(ctx, driver); b != goods+driverFee {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر %d", b, goods+driverFee)
	}
}

// ── ج · وتسليمٌ مرّتين نقداً لا يُقيَّد مرّتين ───────────────────────────
func TestCustomCash_Idempotent(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)

	if err := svc.AgreeCustom(ctx, orderID, driver, 4_000, 2_000); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 6_000, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
	held, treasury := cqHeld(t, driver), cqTreasury(t)
	bal, _ := w.Balance(ctx, driver)

	_, _ = svc.Transition(ctx, driver, []string{"driver"}, orderID, "delivered", "")

	if got := cqHeld(t, driver); got != held {
		t.Fatalf("الصندوقُ تبدّل بنداءٍ ثانٍ: %d ⇐ %d", held, got)
	}
	if got := cqTreasury(t); got != treasury {
		t.Fatalf("الخزينةُ تبدّلت بنداءٍ ثانٍ: %d ⇐ %d", treasury, got)
	}
	if got, _ := w.Balance(ctx, driver); got != bal {
		t.Fatalf("محفظةُ السائق تبدّلت بنداءٍ ثانٍ: %d ⇐ %d", bal, got)
	}
}

// ── د · وتبديلُ الطريقة من نقدٍ إلى محفظةٍ يُصفّر المستحقَّ النقديّ ────────
//
// **ومستحقٌّ نقديٌّ باقٍ على طلبٍ صار محفظيّاً يجعل السائقَ يقبض مرّةً
// والمحفظةَ تُخصم مرّةً** — **ويُقرأ في الدفتر ضِعفَ ما دُفع.**
func TestCustomCash_SwitchToWalletClearsCashDue(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	cqDeposit(t, customer, 50_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)

	if err := svc.AgreeCustom(ctx, orderID, driver, 3_000, 1_000); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 4_000, 1); err != nil {
		t.Fatalf("التأكيد نقداً: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 4_000, 1); err != nil {
		t.Fatalf("التبديلُ إلى المحفظة: %v", err)
	}

	pool := testdb.Pool(t)
	var cashDue int64
	if err := pool.QueryRow(ctx, `SELECT cash_due FROM orders WHERE id = $1`, orderID).
		Scan(&cashDue); err != nil {
		t.Fatalf("تعذّرت قراءةُ المستحقّ: %v", err)
	}
	if cashDue != 0 {
		t.Fatalf("المستحقُّ النقديُّ=%d بعد التبديل إلى المحفظة — **ويُقبض مرّتين**", cashDue)
	}
}
