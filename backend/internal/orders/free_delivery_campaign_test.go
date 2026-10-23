package orders_test

// ══════════════════════════════════════════════════════════════════════
// **عرضُ «التوصيل المجاني»** — قرارُ المالك ٢٠٢٦-١٠-٢٣
// ══════════════════════════════════════════════════════════════════════
//
// مرّةٌ لكلّ زبونٍ بكلّ عرض، **والسائقُ يأخذ أجرتَه كاملةً**، والخزينةُ تتحمّل
// المعفى — **وثوابتُ الحفظ (`FI-06`) لا تُخرَق.** وطلبٌ أُلغي يُحرّر الحقّ.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// armFreeCampaign **عرضٌ شغّالٌ الآن** — ويُمحى بعد الاختبار (فلا يمسّ غيرَه).
func armFreeCampaign(t *testing.T, kinds []string, maxFee *int64) string {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	// **ولا عرضَ آخرُ شغّالٌ يسبقه** — عرضٌ تُرك من اختبارٍ سقط يُطفأ.
	if _, err := pool.Exec(ctx, `UPDATE free_delivery_campaigns SET active = false`); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO free_delivery_campaigns (active, starts_at, ends_at, kinds, max_fee)
		VALUES (true, now() - interval '1 hour', now() + interval '1 day', $1, $2)
		RETURNING id::text`, kinds, maxFee).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ العرض: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM free_delivery_campaigns WHERE id = $1`, id)
	})
	return id
}

// fundTestTreasury **رصيدٌ للخزينة يكفي ما تتحمّله** — خزينةٌ سالبةٌ لا يُرفع علَمُها
// في تنظيف الاختبار التالي. والقياسُ بالقيود لا بالرصيد، فلا يتبدّل.
func fundTestTreasury(t *testing.T) {
	t.Helper()
	if _, err := testdb.Pool(t).Exec(context.Background(),
		`UPDATE wallets SET balance = balance + 100000 WHERE is_treasury`); err != nil {
		t.Fatal(err)
	}
}

func setDeliveryFee(t *testing.T, f *fixture, fee string) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), `
		INSERT INTO app_settings (key, value) VALUES ('delivery.fee', $1::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, fee); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) createStd(t *testing.T, itemID string) *orders.Order {
	t.Helper()
	o, err := f.svc.Create(context.Background(), f.customer, []string{"customer"}, orders.CreateInput{
		CustomerID: f.customer, MerchantID: f.merchantID,
		Items:       []orders.ItemInput{{MenuItemID: itemID, Qty: 1}},
		AddressText: "توصيلٌ مجّانيّ بعرض", Lat: 35.9528, Lng: 39.0079, PaymentMethod: "cash",
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("تعذّر إنشاءُ الطلب: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, o.ID)
	})
	return o
}

func (f *fixture) fees(t *testing.T, id string) (delivery, driver, waived, total int64) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(),
		`SELECT delivery_fee, driver_fee, promo_delivery_waived, total FROM orders WHERE id = $1`, id).
		Scan(&delivery, &driver, &waived, &total); err != nil {
		t.Fatal(err)
	}
	return
}

// TestFreeCampaign_StandardOncePerCustomer **الأوّلُ مجّانيٌّ والثاني بأجرته — والسائقُ يقبض كاملاً.**
func TestFreeCampaign_StandardOncePerCustomer(t *testing.T) {
	f := setup(t, "pending", 100_000, 0, 0)
	ctx := context.Background()
	f.verifyWhatsApp(t)
	f.armTreasury(t)
	itemID := f.armItem(t)
	setDeliveryFee(t, f, "5000")
	armFreeCampaign(t, []string{orders.FreeKindStandard}, nil)

	base, err := fininv.Capture(ctx, f.pool, "FI-06")
	if err != nil {
		t.Fatal(err)
	}
	first := f.createStd(t, itemID)
	if d, drv, w, _ := f.fees(t, first.ID); d != 0 || drv != 5000 || w != 5000 {
		t.Fatalf("الأوّل: يدفع %d · السائقُ %d · المعفى %d — والمنتظَر 0 · 5000 · 5000", d, drv, w)
	}
	second := f.createStd(t, itemID)
	if d, _, w, _ := f.fees(t, second.ID); d != 5000 || w != 0 {
		t.Fatalf("**الثاني مجّانيٌّ أيضاً**: يدفع %d · المعفى %d — والمرّةُ واحدةٌ بالعرض", d, w)
	}

	// **يُسلَّم الأوّلُ: السائقُ يقبض أجرَه كاملاً والحفظُ لا يُخرَق.**
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET driver_id = $2, status = 'at_dropoff' WHERE id = $1`, first.ID, f.driver); err != nil {
		t.Fatal(err)
	}
	before := f.balance(t, f.driver)
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, first.ID, "delivered", ""); err != nil {
		t.Fatalf("التسليم: %v", err)
	}
	if got := f.balance(t, f.driver) - before; got != 5000 {
		t.Fatalf("أجرُ السائق %d — والمنتظَر 5000 كاملاً", got)
	}
	after, err := fininv.Run(ctx, f.pool, "FI-06")
	if err != nil {
		t.Fatal(err)
	}
	if fresh := base.New(after); len(fresh) > 0 {
		t.Fatalf("الحفظُ خُرق بالتوصيل المجانيّ: %v", fresh)
	}
}

// TestFreeCampaign_CancelFreesClaim **طلبٌ أُلغي يُرجع لصاحبه توصيلتَه.**
func TestFreeCampaign_CancelFreesClaim(t *testing.T) {
	f := setup(t, "pending", 100_000, 0, 0)
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)
	setDeliveryFee(t, f, "5000")
	armFreeCampaign(t, []string{orders.FreeKindStandard}, nil)

	first := f.createStd(t, itemID)
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET status = 'cancelled' WHERE id = $1`, first.ID); err != nil {
		t.Fatal(err)
	}
	again := f.createStd(t, itemID)
	if d, _, w, _ := f.fees(t, again.ID); d != 0 || w != 5000 {
		t.Fatalf("بعد الإلغاء: يدفع %d · المعفى %d — **ولم يرجع حقُّه**", d, w)
	}
}

// TestFreeCampaign_MaxFeeAndKinds **السقفُ يُعفي جزءاً، والنوعُ غيرُ المشمول بأجرته.**
func TestFreeCampaign_MaxFeeAndKinds(t *testing.T) {
	f := setup(t, "pending", 100_000, 0, 0)
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)
	setDeliveryFee(t, f, "5000")

	// **عرضٌ للمشتريات وحدَها** ⇒ العاديُّ بأجرته.
	armFreeCampaign(t, []string{orders.FreeKindPurchase}, nil)
	o := f.createStd(t, itemID)
	if d, _, w, _ := f.fees(t, o.ID); d != 5000 || w != 0 {
		t.Fatalf("عرضُ المشتريات أعفى العاديّ: يدفع %d · المعفى %d", d, w)
	}

	// **وسقفٌ ٢٠٠٠** ⇒ يدفع ٣٠٠٠.
	cap := int64(2000)
	armFreeCampaign(t, []string{orders.FreeKindStandard}, &cap)
	o2 := f.createStd(t, itemID)
	if d, drv, w, _ := f.fees(t, o2.ID); d != 3000 || w != 2000 || drv != 5000 {
		t.Fatalf("بالسقف: يدفع %d · المعفى %d · السائق %d — والمنتظَر 3000 · 2000 · 5000", d, w, drv)
	}
}

// TestFreeCampaign_CustomPurchase **المشترياتُ: الزبونُ يؤكّد البضاعةَ وحدَها، والسائقُ بأجرته.**
func TestFreeCampaign_CustomPurchase(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	fundTestTreasury(t)
	armFreeCampaign(t, []string{orders.FreeKindPurchase, orders.FreeKindAmanah}, nil)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)

	const goods, fee = 12_000, 5_000
	platform, driverFee := shareOf(fee, 10)
	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	pool := testdb.Pool(t)
	var delivery, waived, total, customFee int64
	if err := pool.QueryRow(ctx, `SELECT delivery_fee, promo_delivery_waived, total, custom_fee
		FROM orders WHERE id = $1`, orderID).Scan(&delivery, &waived, &total, &customFee); err != nil {
		t.Fatal(err)
	}
	if delivery != 0 || waived != fee || total != goods || customFee != fee {
		t.Fatalf("بعد الاتّفاق: يدفع %d · المعفى %d · المجموع %d · المتّفق %d", delivery, waived, total, customFee)
	}
	// **والزبونُ يرى الأجرةَ صفراً** — نسختُه تعرض `custom_fee`.
	o, err := svc.GetByID(ctx, orderID)
	if err != nil {
		t.Fatal(err)
	}
	if v := orders.ViewFor(orders.AudienceCustomer, o)["custom_fee"]; v != float64(0) && v != int64(0) {
		t.Fatalf("الزبونُ يرى الأجرة %v — والمنتظَر 0", v)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", goods, 1); err != nil {
		t.Fatalf("التأكيد بالبضاعة وحدَها: %v", err)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
	if b, _ := w.Balance(ctx, driver); b != driverFee {
		t.Fatalf("محفظةُ السائق %d — والمنتظَر %d (أجرُه كاملاً)", b, driverFee)
	}
	// **والخزينةُ تتحمّل الأجرةَ ناقصَ نصيبها.**
	if got := cqTreasury(t); got != platform-fee {
		t.Fatalf("الخزينة %d — والمنتظَر %d", got, platform-fee)
	}
}

// TestFreeCampaign_AmanahFullyFree **أمانةٌ أجرتُها كلُّها معفاة — نقدٌ صفر، والسائقُ يقبض.**
func TestFreeCampaign_AmanahFullyFree(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	fundTestTreasury(t)
	armFreeCampaign(t, []string{orders.FreeKindAmanah}, nil)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if _, err := testdb.Pool(t).Exec(ctx, `UPDATE orders SET custom_mode = 'amanah' WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	const fee = 5_000
	if err := svc.AgreeCustom(ctx, orderID, driver, 0, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	// **أجرُه المعتمَد من الصفّ** — حصّةُ الأمانة غيرُ حصّة المشتريات.
	var driverFee int64
	if err := testdb.Pool(t).QueryRow(ctx, `SELECT driver_fee FROM orders WHERE id = $1`, orderID).Scan(&driverFee); err != nil || driverFee <= 0 {
		t.Fatalf("أجرُ السائق المعتمَد %d: %v", driverFee, err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 0, 1); err != nil {
		t.Fatalf("التأكيد بصفر: %v", err)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
	if b, _ := w.Balance(ctx, driver); b != driverFee {
		t.Fatalf("محفظةُ السائق %d — والمنتظَر %d: **قاد أمانةً مجانيّةً بلا أجر**", b, driverFee)
	}
}
