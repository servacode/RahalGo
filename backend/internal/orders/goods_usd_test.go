package orders_test

// ══════════════════════════════════════════════════════════════════════
// **ثمنُ البضاعة بالليرة أو بالدولار** — قرارُ المالك ٢٠٢٦-١١-٠٩
// ══════════════════════════════════════════════════════════════════════
//
//	المجموعُ والمستحقُّ النقديُّ وحجزُ المحفظة ⇐ الأجرةُ وحدَها
//	عمولةُ المتجر التابع ⇐ بسعر المتجر، وإلّا المرجعيّ — ملقوطاً لحظةَ التأكيد
//	سقفُ نقد الزبون وتعرّضُ السائق وخسارةُ البضاعة ⇐ بالمعادل
//	تطبيقٌ قديمٌ بلا عملة ⇐ ليرة

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func usdPtr(c int64) *int64 { return &c }

// usdSetRef **سعرُ الدولار المرجعيّ** — ويُمحى بعد الاختبار.
func usdSetRef(t *testing.T, rate string) {
	t.Helper()
	pool := testdb.Pool(t)
	if _, err := pool.Exec(context.Background(), `INSERT INTO app_settings (key, value)
		VALUES ('merchants.usd_reference_rate', $1::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, rate); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key = 'merchants.usd_reference_rate'`)
	})
}

type usdRow struct {
	goods, total, cashDue, reserved, storePaid, commission, equiv int64
	usd                                                           *int64
	rate                                                          *float64
}

func usdRead(t *testing.T, orderID string) usdRow {
	t.Helper()
	var r usdRow
	if err := testdb.Pool(t).QueryRow(context.Background(), `
		SELECT COALESCE(custom_goods_amount, 0), total, cash_due, custom_reserved_amount,
		       COALESCE(custom_store_paid, 0), platform_commission, goods_usd_syp,
		       custom_goods_usd, goods_fx_rate::float8
		  FROM orders WHERE id = $1`, orderID).
		Scan(&r.goods, &r.total, &r.cashDue, &r.reserved, &r.storePaid, &r.commission, &r.equiv,
			&r.usd, &r.rate); err != nil {
		t.Fatal(err)
	}
	return r
}

// TestGoodsUSD_PartnerCash_StoreRateBeatsReference **مثالٌ كاملٌ نقداً مع متجرٍ تابع**:
// ١٢٫٥٠$ وسعرُ المتجر ١٠٠ (والمرجعيُّ ١٢٠ لا يُقرأ) ⇒ معادلٌ ١٢٥٠، وعمولةٌ ١٠٪ = ١٢٥.
// المجموعُ والمستحقُّ الأجرةُ ١٠٠ وحدَها، **وصندوقُ السائق ٢٢٥** (أجرةٌ + عمولة)، والخزينةُ ١٢٥،
// ويدفع للمتجر ١١٫٢٥$.
func TestGoodsUSD_PartnerCash_StoreRateBeatsReference(t *testing.T) {
	svc, balance, customer, driver, _, orderID := ptFixture(t, "cash")
	ctx := context.Background()
	usdSetRef(t, "120")
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
	r := usdRead(t, orderID)
	if r.goods != 0 || r.usd == nil || *r.usd != 1250 || r.total != 100 {
		t.Fatalf("**بعد الاتّفاق**: goods=%d usd=%v total=%d — المنتظَر 0 و1250 و100", r.goods, r.usd, r.total)
	}
	o, err := svc.GetByID(ctx, orderID)
	if err != nil {
		t.Fatal(err)
	}
	if o.CustomGoodsUSD == nil || *o.CustomGoodsUSD != 1250 {
		t.Fatalf("**الطلبُ لا يحمل ثمنَ الدولار**: %v", o.CustomGoodsUSD)
	}
	if v := orders.ViewFor(orders.AudienceCustomer, o); v["custom_goods_usd"] == nil {
		t.Fatal("**الزبونُ لا يرى ثمنَ الدولار**")
	}
	if v := orders.ViewFor(orders.AudienceCustomer, o); v["goods_fx_rate"] != nil || v["goods_usd_syp"] != nil {
		t.Fatal("**سعرُ الصرف ومعادلُه للمكتب وحدَه**")
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 100, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	r = usdRead(t, orderID)
	if r.rate == nil || *r.rate != 100 || r.equiv != 1250 || r.cashDue != 100 {
		t.Fatalf("**اللقطة**: rate=%v equiv=%d cash_due=%d — المنتظَر 100 و1250 و100", r.rate, r.equiv, r.cashDue)
	}
	// **والسائقُ يرى ما يدفعه للمتجر بالدولار** — ولا يُجمع مع الليرة.
	o, _ = svc.GetByID(ctx, orderID)
	if o.PayMerchant != 0 || len(o.PayMerchants) != 1 || o.PayMerchants[0].Currency != orders.CurrencyUSD ||
		o.PayMerchants[0].Amount != 1125 {
		t.Fatalf("**ادفع للمتجر**: %d / %+v — المنتظَر 1125 سنتاً بالدولار", o.PayMerchant, o.PayMerchants)
	}
	// **وتعرّضُ السائق مالُ المنصّة وحدَه** (قرارُ المالك ٢٠٢٦-١١-١٠، القرار ٧) — الأجرةُ ١٠٠؛
	// وثمنُ البضاعة بالدولار لا يُعدّ (كان: المحصَّل + 100 + 1250).
	cb := cashbox.NewService(testdb.Pool(t), settings.NewStore(testdb.Pool(t)))
	heldBefore := cqHeld(t, driver)
	if exp, err := cb.Exposure(ctx, testdb.Pool(t), driver); err != nil || exp != heldBefore+100 {
		t.Fatalf("**التعرّض** %d (%v) — المنتظَر المحصَّل + 100", exp, err)
	}
	// **وتبديلُ المرجعيّ بعد التأكيد لا يمسّ الطلب.**
	usdSetRef(t, "500")
	ptDeliver(t, svc, driver, orderID)

	r = usdRead(t, orderID)
	if r.commission != 125 || r.storePaid != 1125 {
		t.Fatalf("**العمولة %d والمدفوعُ للمتجر %d** — المنتظَر 125 و1125", r.commission, r.storePaid)
	}
	if got := cqHeld(t, driver) - heldBefore; got != 225 {
		t.Fatalf("**صندوقُ السائق %d** — المنتظَر 225 (أجرة 100 + عمولة 125)", got)
	}
	if got := balance(driver); got != 100 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 100", got)
	}
	if got := cqTreasury(t); got != 125 {
		t.Fatalf("**الخزينة %d** — المنتظَر 125 (العمولة)", got)
	}
	if adv := ptAdvanceReturn(t, orderID); adv != 0 {
		t.Fatalf("سلفةٌ رجعت %d — والسائقُ دفع للمتجر من دولار الزبون", adv)
	}
	// **FI-06.a: صافي دفتر الطلب = نقدُ السائق منه.**
	if got := cqLedgerSum(t, orderID); got != 225 {
		t.Fatalf("**صافي الدفتر %d** — المنتظَر 225", got)
	}
}

// TestGoodsUSD_WalletPaysFeeOnly_ReferenceRate **المحفظةُ تدفع الأجرةَ وحدَها** والسعرُ المرجعيّ
// يُلقط بلا متجرٍ تابع: ١٠$ × ١٢٠ = ١٢٠٠. **ولا عمولةَ ولا سلفة**، والخزينةُ نصيبُها من الأجرة وحدَه.
func TestGoodsUSD_WalletPaysFeeOnly_ReferenceRate(t *testing.T) {
	svc, balance, customer, driver, _, orderID := ptFixture(t, "wallet")
	ctx := context.Background()
	usdSetRef(t, "120")
	if err := svc.AgreeCustomStepUSD(ctx, orderID, driver, "", 0, 100, usdPtr(1000)); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 100, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	r := usdRead(t, orderID)
	if r.reserved != 100 || r.cashDue != 0 || r.equiv != 1200 || r.rate == nil || *r.rate != 120 {
		t.Fatalf("**الحجز %d والمستحقّ %d والمعادل %d والسعر %v** — المنتظَر 100 و0 و1200 و120",
			r.reserved, r.cashDue, r.equiv, r.rate)
	}
	heldBefore := cqHeld(t, driver)
	ptDeliver(t, svc, driver, orderID)
	r = usdRead(t, orderID)
	if r.commission != 0 || r.storePaid != 1200 {
		t.Fatalf("العمولة %d والمدفوعُ للمتجر %d — المنتظَر 0 و1200", r.commission, r.storePaid)
	}
	if got := cqHeld(t, driver) - heldBefore; got != 0 {
		t.Fatalf("صندوقُ السائق %d — الدولارُ دفعه كلَّه للمتجر", got)
	}
	if got := balance(customer); got != 10_000-100 {
		t.Fatalf("**محفظةُ الزبون %d** — خُصمت الأجرةُ وحدَها", got)
	}
	if got := balance(driver); got != 100 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 100", got)
	}
	if adv := ptAdvanceReturn(t, orderID); adv != 0 {
		t.Fatalf("سلفةٌ رجعت %d", adv)
	}
	if got := cqLedgerSum(t, orderID); got != 0 {
		t.Fatalf("صافي الدفتر %d — المنتظَر 0", got)
	}
}

// TestGoodsUSD_CODLimitCountsEquivalent **سقفُ نقد الزبون يحسب معادلَ الدولار** — بالنقد والمحفظة معاً
// (الدولارُ نقدٌ للسائق في الحالين).
func TestGoodsUSD_CODLimitCountsEquivalent(t *testing.T) {
	for _, pay := range []string{"cash", "wallet"} {
		t.Run(pay, func(t *testing.T) {
			svc, _, customer, driver, _, orderID := ptFixture(t, pay)
			ctx := context.Background()
			usdSetRef(t, "100")
			pool := testdb.Pool(t)
			if _, err := pool.Exec(ctx, `INSERT INTO app_settings (key, value) VALUES ('customers.cod_limit', '1000'::jsonb)
				ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_, _ = pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key = 'customers.cod_limit'`)
			})
			if err := svc.AgreeCustomStepUSD(ctx, orderID, driver, "", 0, 100, usdPtr(2000)); err != nil {
				t.Fatalf("الاتّفاق: %v", err)
			}
			if _, err := svc.ConfirmQuote(ctx, orderID, customer, pay, 100, 1); !errors.Is(err, orders.ErrCODLimitExceeded) {
				t.Fatalf("**٢٠$ = ٢٠٠٠ ل.س والسقفُ ١٠٠٠**: %v — المنتظَر cod_limit", err)
			}
			if r := usdRead(t, orderID); r.equiv != 0 || r.rate != nil {
				t.Fatalf("**تأكيدٌ مردودٌ ترك لقطة**: %d / %v", r.equiv, r.rate)
			}
		})
	}
}

// TestGoodsUSD_LostGoodsPostsEquivalent **مشترياتٌ بالدولار فشلت بعد الشراء** ⇒ الخسارةُ بالمعادل
// (٥$ × ١٠٠ = ٥٠٠) — كالليرة تماماً.
func TestGoodsUSD_LostGoodsPostsEquivalent(t *testing.T) {
	svc, _, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	ptFeeBounds(t, exec)
	clearTreasury(t, pool)
	cqArmTreasury(t)
	t.Cleanup(func() { clearTreasury(t, pool) })
	usdSetRef(t, "100")
	orderID := cqSeedOrder(t, nil, customer, driver, "driver_defined", nil, true)
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
		Note: "دولار", DriverShare: orders.DriverSharePlatform,
	}, nil); err != nil {
		t.Fatalf("الإنهاءُ عند الباب: %v", err)
	}
	dropReturnTrip(t, pool, orderID)
	if got := clgLoss(t, orderID); got != 500 {
		t.Fatalf("**خسارةُ الخزينة %d** — المنتظَر 500 (معادلُ ٥$)", got)
	}
	if got := ptAdvanceReturn(t, orderID); got != 500 {
		t.Fatalf("سلفةٌ رجعت %d — المنتظَر 500", got)
	}
	if got := cqLedgerSum(t, orderID); got != -500 {
		t.Fatalf("صافي الدفتر %d — المنتظَر −500", got)
	}
}

// TestGoodsUSD_CurrencySwitchReconfirms **تبديلُ العملة يُعيد التأكيد ولو نقص المجموع** — ٥٠٠ ل.س
// مؤكَّدةٌ ثمّ ٥$: المجموعُ ١٠٠ والتأكيدُ يسقط والنسخةُ تزيد. **ثمّ رجوعٌ لليرة يمحو الدولارَ ولقطتَه.**
func TestGoodsUSD_CurrencySwitchReconfirms(t *testing.T) {
	svc, _, customer, driver, _, orderID := ptFixture(t, "cash")
	ctx := context.Background()
	usdSetRef(t, "100")
	if err := svc.AgreeCustom(ctx, orderID, driver, 500, 100); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 600, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	if err := svc.AgreeCustomStepUSD(ctx, orderID, driver, "", 0, 100, usdPtr(500)); err != nil {
		t.Fatalf("إلى الدولار: %v", err)
	}
	s := cqRead(t, orderID)
	if s.confirmedAt != nil || s.quoteVersion != 2 || *s.total != 100 {
		t.Fatalf("**التأكيد %v والنسخة %d والمجموع %d** — المنتظَر سقوطُه و2 و100", s.confirmedAt, s.quoteVersion, *s.total)
	}
	// **وإعادةُ الدولار نفسِه لا تغيّر شيئاً.**
	if err := svc.AgreeCustomStepUSD(ctx, orderID, driver, "", 0, 100, usdPtr(500)); err != nil {
		t.Fatal(err)
	}
	if s := cqRead(t, orderID); s.quoteVersion != 2 {
		t.Fatalf("إعادةُ القيمة نفسِها زادت النسخة: %d", s.quoteVersion)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 100, 2); err != nil {
		t.Fatalf("تأكيدُ الدولار: %v", err)
	}
	// **ورجوعٌ لليرة (تطبيقٌ قديمٌ بلا عملة) يمحو الدولار ولقطتَه.**
	if err := svc.AgreeCustom(ctx, orderID, driver, 700, 100); err != nil {
		t.Fatalf("رجوعٌ لليرة: %v", err)
	}
	r := usdRead(t, orderID)
	if r.usd != nil || r.rate != nil || r.equiv != 0 || r.goods != 700 || r.total != 800 {
		t.Fatalf("**بعد الرجوع**: %+v", r)
	}
}

// TestGoodsUSD_AmanahRefused **الأمانةُ بلا ثمن** — دولارٌ عليها يُردّ.
func TestGoodsUSD_AmanahRefused(t *testing.T) {
	svc, _, _, driver, _, orderID := ptFixture(t, "cash")
	ctx := context.Background()
	if _, err := testdb.Pool(t).Exec(ctx, `UPDATE orders SET custom_mode = 'amanah' WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	if err := svc.AgreeCustomStepUSD(ctx, orderID, driver, "", 0, 100, usdPtr(500)); !errors.Is(err, orders.ErrGoodsUSDNotPurchase) {
		t.Fatalf("دولارٌ على أمانة: %v", err)
	}
}

// TestGoodsUSD_PriceRequestSameCurrency **طلبُ الكابتن بعد الاستلام بعملة الثمن المؤكَّد** — وبغيرها يُردّ،
// **ورفضُ الزبون يُرجع الدولارَ القديم.**
func TestGoodsUSD_PriceRequestSameCurrency(t *testing.T) {
	svc, _, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	ptFeeBounds(t, exec)
	usdSetRef(t, "100")
	exec(ctx, `INSERT INTO app_settings (key, value) VALUES ('delivery.driver_price_edit_max_percent', '50'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`)
	t.Cleanup(func() {
		_, _ = testdb.Pool(t).Exec(context.Background(),
			`DELETE FROM app_settings WHERE key = 'delivery.driver_price_edit_max_percent'`)
	})
	orderID := cqSeedOrder(t, nil, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustomStepUSD(ctx, orderID, driver, "", 0, 100, usdPtr(1000)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 100, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RequestDriverPriceCurrency(ctx, orderID, driver, 1200, "", "أغلى"); !errors.Is(err, orders.ErrPriceRequestCurrency) {
		t.Fatalf("ليرةٌ على ثمنٍ بالدولار: %v", err)
	}
	if _, err := svc.RequestDriverPriceCurrency(ctx, orderID, driver, 1200, orders.CurrencyUSD, "أغلى"); err != nil {
		t.Fatalf("طلبُ الدولار: %v", err)
	}
	if r := usdRead(t, orderID); r.usd == nil || *r.usd != 1200 || r.total != 100 {
		t.Fatalf("بعد الطلب: %+v", r)
	}
	if _, err := svc.RejectQuote(ctx, orderID, customer); err != nil {
		t.Fatalf("الرفض: %v", err)
	}
	if r := usdRead(t, orderID); r.usd == nil || *r.usd != 1000 {
		t.Fatalf("**الرفضُ لم يُرجع الدولارَ القديم**: %+v", r)
	}
}
