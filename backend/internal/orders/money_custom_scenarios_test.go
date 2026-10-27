package orders_test

// ══════════════════════════════════════════════════════════════════════
// **سيناريوهات مال الطلب الخاصّ — «اشتري لي» و«أمانة»** (عائلةُ الجرد ٢٠٢٦-١٠-٢٧)
// ══════════════════════════════════════════════════════════════════════
//
// تسدّ ما لم يُقَس في اختبارات الخاصّ القائمة:
//
//	أ · أمانةٌ بالمحفظة بحصّة الأمانة النسبيّة          — الزبون والسائق والخزينة، ولا صندوقَ ولا درج
//	ب · أمانةٌ نقداً بأجرةٍ مفروضةٍ من المنصّة وحصّةٍ ثابتة — السائقُ لا يغيّر الأجرة، والحصّةُ مبلغ
//	ج · مشترياتٌ بالمحفظة بحصّةٍ ثابتة                    — الدرجُ يردّ ثمنَ البضاعة سلفةً
//	د · مشترياتٌ من متجرٍ تابعٍ نقداً مع حصّةٍ من الأجرة   — العمولةُ والمندوبُ وحصّةُ الأجرة معاً
//	هـ · مشترياتٌ من متجرٍ تابعٍ ضاعت بعد الشراء          — الخسارةُ صافي ما دُفع للمتجر لا الثمنُ كلُّه
//	و · أجرةُ الطلب الخاصّ تسدّ دينَ السائق               — والمقتطَعُ يعود للخزينة
//
// **وكلٌّ منها يتحقّق أنّ ثوابت المال (`FI-02` و`FI-06` و`FI-10` و`FI-11` و`FI-15`) لم يُخرق
// منها شيءٌ جديد** — بالفرق عن صورةٍ قبل السيناريو، فالقاعدةُ مشتركة.
//
// **ولا افتراضَ لساعة اليوم**: لا شيء هنا يقرأ «اليوم».

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// mcsFamilies **عوائلُ الثوابت التي يمسّها الطلبُ الخاصّ.**
var mcsFamilies = []string{"FI-02.a", "FI-06", "FI-10", "FI-11", "FI-15"}

// mcsGuard **يصوّر الخروقَ القائمة ويُرجع فاحصاً لما استجدّ بعد السيناريو.**
func mcsGuard(t *testing.T) func() {
	t.Helper()
	ctx := context.Background()
	pool := testdb.Pool(t)
	base, err := fininv.Capture(ctx, pool, mcsFamilies...)
	if err != nil {
		t.Fatalf("صورةُ الثوابت: %v", err)
	}
	return func() {
		t.Helper()
		after, err := fininv.Run(ctx, pool, mcsFamilies...)
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

// mcsSetting **يضبط إعداداً ويمحوه بعد الاختبار.**
func mcsSetting(t *testing.T, key, jsonValue string) {
	t.Helper()
	pool := testdb.Pool(t)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO app_settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, jsonValue); err != nil {
		t.Fatalf("الإعداد %s: %v", key, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key = $1`, key)
	})
}

// mcsFund **يموّل المحفظةَ بقيدٍ في الدفتر لا بتعديلٍ خامٍ للرصيد.**
//
// **و`cqDeposit` يكتب الرصيدَ مباشرةً** فيبقى رصيدٌ بلا قيد — **وهذا عينُ خرق
// `FI-02.a`**؛ وبصورةٍ قبل السيناريو يتغيّر الصفُّ نفسُه (الرصيدُ والمجموع) بعده
// **فيُقرأ «خرقاً جديداً» وهو من الزرع لا من المحرّك.**
func mcsFund(t *testing.T, w *wallet.Service, userID string, amount int64) {
	t.Helper()
	if _, err := w.Apply(context.Background(), userID, amount, "topup", "", "تمويلُ اختبار", nil); err != nil {
		t.Fatalf("تمويلُ المحفظة: %v", err)
	}
}

// mcsAmanah **يحوّل الطلبَ المزروع إلى أمانة** — لا ثمنَ بضاعةٍ فيها.
func mcsAmanah(t *testing.T, orderID string) {
	t.Helper()
	if _, err := testdb.Pool(t).Exec(context.Background(),
		`UPDATE orders SET custom_mode = 'amanah' WHERE id = $1`, orderID); err != nil {
		t.Fatalf("الأمانة: %v", err)
	}
}

// mcsOrderMoney **أعمدةُ مال الطلب بعد التسوية.**
func mcsOrderMoney(t *testing.T, orderID string) (driverFee, commission, storePaid int64) {
	t.Helper()
	if err := testdb.Pool(t).QueryRow(context.Background(), `
		SELECT driver_fee, platform_commission, COALESCE(custom_store_paid, 0)
		FROM orders WHERE id = $1`, orderID).Scan(&driverFee, &commission, &storePaid); err != nil {
		t.Fatalf("أعمدةُ الطلب: %v", err)
	}
	return
}

// ── أ · أمانةٌ بالمحفظة: حصّةُ الأمانة ٢٠٪ من الأجرة ──────────────────────
//
//	أجرةٌ ٥٠٠ · حصّةُ الأمانة ٢٠٪ ⇒ السائقُ ٤٠٠ · الخزينةُ ١٠٠ · الزبونُ −٥٠٠
//	لا بضاعة ⇒ لا صندوقَ ولا سلفةَ من الدرج · مجموعُ الدفتر صفر
func TestMoneyCustom_AmanahWallet_AmanahShare(t *testing.T) {
	svc, w, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	ptFeeBounds(t, exec)
	mcsSetting(t, "delivery.share_amanah", "20")
	cqArmTreasury(t)
	mcsFund(t, w, customer, 2_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	mcsAmanah(t, orderID)
	check := mcsGuard(t)

	if err := svc.AgreeCustom(ctx, orderID, driver, 0, 500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if df, _, _ := mcsOrderMoney(t, orderID); df != 400 {
		t.Fatalf("أجرُ السائق الملقوط=%d — المنتظَر 400 (حصّةُ الأمانة لا المشتريات)", df)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 500, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	heldBefore := cqHeld(t, driver)
	ptDeliver(t, svc, driver, orderID)

	if b, _ := w.Balance(ctx, customer); b != 1_500 {
		t.Fatalf("محفظةُ الزبون=%d — المنتظَر 1500", b)
	}
	if r := cqReserved(t, w, customer); r != 0 {
		t.Fatalf("محجوزٌ باقٍ=%d بعد التسليم", r)
	}
	if b, _ := w.Balance(ctx, driver); b != 400 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 400", b)
	}
	if got := cqTreasury(t); got != 100 {
		t.Fatalf("الخزينة=%d — المنتظَر 100", got)
	}
	if got := cqHeld(t, driver) - heldBefore; got != 0 {
		t.Fatalf("صندوقُ السائق تحرّك %d — ولم يقبض نقداً", got)
	}
	if adv := ptAdvanceReturn(t, orderID); adv != 0 {
		t.Fatalf("سلفةٌ من الدرج=%d — والأمانةُ لا ثمنَ فيها", adv)
	}
	if sum := cqLedgerSum(t, orderID); sum != 0 {
		t.Fatalf("مجموعُ دفتر الطلب=%d — المنتظَر 0", sum)
	}
	check()
}

// ── ب · أمانةٌ نقداً بأجرةٍ مفروضةٍ من المنصّة وحصّةٍ ثابتة ────────────────
//
//	المنصّةُ تفرض ٣٠٠ (والسائقُ أرسل ٩٩٩ فأُهمل) · حصّةٌ ثابتةٌ ٥٠
//	⇒ الصندوقُ +٣٠٠ · محفظةُ السائق ٢٥٠ · الخزينةُ ٥٠ · مجموعُ الدفتر = النقد ٣٠٠
func TestMoneyCustom_AmanahCash_AdminFeeFixedShare(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	mcsSetting(t, "delivery.share_amanah", "50")
	mcsSetting(t, "delivery.share_amanah_mode", `"fixed"`)
	cqArmTreasury(t)
	snap := int64(300)
	orderID := cqSeedOrder(t, w, customer, driver, "admin_defined", &snap, false)
	mcsAmanah(t, orderID)
	check := mcsGuard(t)

	if err := svc.AgreeCustom(ctx, orderID, driver, 0, 999); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	var fee int64
	if err := testdb.Pool(t).QueryRow(ctx, `SELECT custom_fee FROM orders WHERE id = $1`, orderID).Scan(&fee); err != nil {
		t.Fatal(err)
	}
	if fee != 300 {
		t.Fatalf("الأجرة=%d — المنتظَر 300: **السائقُ غيّر أجرةً تفرضها المنصّة**", fee)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 300, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	heldBefore := cqHeld(t, driver)
	ptDeliver(t, svc, driver, orderID)

	if got := cqHeld(t, driver) - heldBefore; got != 300 {
		t.Fatalf("صندوقُ السائق=%d — المنتظَر 300", got)
	}
	if b, _ := w.Balance(ctx, driver); b != 250 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 250 (٣٠٠ ناقصَ حصّةٍ ثابتةٍ ٥٠)", b)
	}
	if b, _ := w.Balance(ctx, customer); b != 0 {
		t.Fatalf("محفظةُ الزبون=%d — ودفع نقداً", b)
	}
	if got := cqTreasury(t); got != 50 {
		t.Fatalf("الخزينة=%d — المنتظَر 50", got)
	}
	if sum := cqLedgerSum(t, orderID); sum != 300 {
		t.Fatalf("مجموعُ دفتر الطلب=%d — المنتظَر 300 (= نقدُ الصندوق)", sum)
	}
	check()
}

// ── ج · مشترياتٌ بالمحفظة بحصّةٍ ثابتة: الدرجُ يردّ ثمنَ البضاعة ─────────────
//
//	بضاعةٌ ١٠٠٠ · أجرةٌ ٥٠٠ · حصّةٌ ثابتةٌ ١٠٠ ⇒ الزبونُ −١٥٠٠ · السائقُ ٤٠٠ · الخزينةُ ١٠٠
//	السائقُ دفع ١٠٠٠ للمتجر من سلفة المكتب ولم يقبض نقداً ⇒ الدرجُ يُخرج ١٠٠٠ «سلفة رجعت»
//	مجموعُ الدفتر −١٠٠٠ (ثمنُ البضاعة خرج من المحفظة ليعوّض المكتب)
func TestMoneyCustom_PurchaseWallet_FixedShare_AdvanceReturn(t *testing.T) {
	svc, w, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	ptFeeBounds(t, exec)
	exec(ctx, `UPDATE app_settings SET value = '100'::jsonb WHERE key = 'delivery.share_purchase'`)
	mcsSetting(t, "delivery.share_purchase_mode", `"fixed"`)
	cqArmTreasury(t)
	mcsFund(t, w, customer, 5_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	check := mcsGuard(t)

	if err := svc.AgreeCustom(ctx, orderID, driver, 1_000, 500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 1_500, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	heldBefore := cqHeld(t, driver)
	ptDeliver(t, svc, driver, orderID)

	if b, _ := w.Balance(ctx, customer); b != 3_500 {
		t.Fatalf("محفظةُ الزبون=%d — المنتظَر 3500", b)
	}
	if b, _ := w.Balance(ctx, driver); b != 400 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 400 (٥٠٠ ناقصَ ١٠٠)", b)
	}
	if got := cqTreasury(t); got != 100 {
		t.Fatalf("الخزينة=%d — المنتظَر 100", got)
	}
	if got := cqHeld(t, driver) - heldBefore; got != 0 {
		t.Fatalf("صندوقُ السائق تحرّك %d — ولم يقبض نقداً", got)
	}
	if adv := ptAdvanceReturn(t, orderID); adv != 1_000 {
		t.Fatalf("سلفةٌ رجعت للمكتب=%d — المنتظَر 1000", adv)
	}
	if _, commission, storePaid := mcsOrderMoney(t, orderID); commission != 0 || storePaid != 1_000 {
		t.Fatalf("العمولة=%d والمدفوعُ للمتجر=%d — المنتظَر 0 و1000", commission, storePaid)
	}
	if sum := cqLedgerSum(t, orderID); sum != -1_000 {
		t.Fatalf("مجموعُ دفتر الطلب=%d — المنتظَر −1000", sum)
	}
	check()
}

// ── د · متجرٌ تابعٌ نقداً مع حصّةٍ من الأجرة ─────────────────────────────
//
//	بضاعةٌ ١٠٠٠ · عمولةُ المتجر ١٠٪ = ١٠٠ · أجرةٌ ٥٠٠ بحصّة ١٠٪ = ٥٠ · مندوبٌ ١٠٪ من العمولة = ١٠
//	الزبونُ يدفع ١٥٠٠ نقداً · السائقُ يدفع للمتجر ٩٠٠ ⇒ صندوقُه ٦٠٠
//	محفظةُ السائق ٤٥٠ · المندوبُ ١٠ · الخزينةُ ١٤٠ (= ٩٠ من العمولة + ٥٠ من الأجرة) · الدرجُ صفر
func TestMoneyCustom_PartnerCash_CommissionPlusFeeShare(t *testing.T) {
	svc, w, customer, driver, exec := cqSetup(t) // حصّةُ المشتريات ١٠٪
	ctx := context.Background()
	ptFeeBounds(t, exec)
	cqArmTreasury(t)
	rep := testdb.NewUser(t, testdb.Pool(t), "sales")
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	exec(ctx, `UPDATE orders SET snap_rep_commission_percent = 10 WHERE id = $1`, orderID)
	merchant := ptMerchant(t, 10, rep)
	check := mcsGuard(t)

	if _, _, err := svc.LinkPartner(ctx, customer, orderID, merchant, nil); err != nil {
		t.Fatalf("الربط: %v", err)
	}
	if err := svc.AgreeCustom(ctx, orderID, driver, 1_000, 500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 1_500, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	if o, err := svc.GetByID(ctx, orderID); err != nil || o.PayMerchant != 900 {
		t.Fatalf("ادفع للمتجر: %v · %v — المنتظَر 900", o, err)
	}
	heldBefore := cqHeld(t, driver)
	ptDeliver(t, svc, driver, orderID)

	if got := cqHeld(t, driver) - heldBefore; got != 600 {
		t.Fatalf("صندوقُ السائق=%d — المنتظَر 600 (١٥٠٠ ناقصَ ٩٠٠ للمتجر)", got)
	}
	if b, _ := w.Balance(ctx, driver); b != 450 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 450", b)
	}
	if b, _ := w.Balance(ctx, rep); b != 10 {
		t.Fatalf("المندوب=%d — المنتظَر 10", b)
	}
	if got := cqTreasury(t); got != 140 {
		t.Fatalf("الخزينة=%d — المنتظَر 140 (عمولةٌ ٩٠ بعد المندوب + حصّةُ الأجرة ٥٠)", got)
	}
	if _, commission, storePaid := mcsOrderMoney(t, orderID); commission != 100 || storePaid != 900 {
		t.Fatalf("العمولة=%d والمدفوعُ للمتجر=%d — المنتظَر 100 و900", commission, storePaid)
	}
	if adv := ptAdvanceReturn(t, orderID); adv != 0 {
		t.Fatalf("سلفةٌ رجعت=%d — والنقدُ غطّاها", adv)
	}
	if sum := cqLedgerSum(t, orderID); sum != 600 {
		t.Fatalf("مجموعُ دفتر الطلب=%d — المنتظَر 600 (= نقدُ الصندوق)", sum)
	}
	check()
}

// ── هـ · متجرٌ تابعٌ ضاعت بضاعتُه بعد الشراء ─────────────────────────────
//
//	بضاعةٌ ١٠٠٠ بعمولة ١٠٪ ⇒ دفع السائقُ للمتجر ٩٠٠ · أُنهي عند الباب والمنصّةُ تتحمّل
//	الخزينةُ −٩٠٠ (لا −١٠٠٠: العمولةُ لم تخرج من سلفة المكتب) · الدرجُ يردّ ٩٠٠
//	لا أجرَ للسائق · لا نصيبَ للمندوب · لا صندوق · لا دين
func TestMoneyCustom_PartnerLostGoods_NetLoss(t *testing.T) {
	svc, w, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	ptFeeBounds(t, exec)
	clearTreasury(t, pool)
	cqArmTreasury(t)
	t.Cleanup(func() { clearTreasury(t, pool) })
	rep := testdb.NewUser(t, pool, "sales")
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	exec(ctx, `UPDATE orders SET snap_rep_commission_percent = 10 WHERE id = $1`, orderID)
	merchant := ptMerchant(t, 10, rep)
	check := mcsGuard(t)

	if _, _, err := svc.LinkPartner(ctx, customer, orderID, merchant, nil); err != nil {
		t.Fatalf("الربط: %v", err)
	}
	if err := svc.AgreeCustom(ctx, orderID, driver, 1_000, 500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 1_500, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
	heldBefore := cqHeld(t, driver)
	ops := testdb.NewUser(t, pool, "operations")
	if _, err := svc.ResolveDoor(ctx, ops, []string{"ops"}, orderID, orders.DoorResolution{
		Action: orders.DoorReturnToOffice, Fault: orders.FaultDriver, Reason: "customer_refused",
		Note: "البضاعةُ ضاعت بعد الشراء", DriverShare: orders.DriverSharePlatform,
	}, nil); err != nil {
		t.Fatalf("الإنهاءُ عند الباب: %v", err)
	}
	dropReturnTrip(t, pool, orderID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM financial_obligations WHERE party_kind = 'driver' AND party_id = $1`, driver)
		_, _ = pool.Exec(context.Background(), `UPDATE users SET driver_debt = 0 WHERE id = $1`, driver)
	})

	if got := clgLoss(t, orderID); got != 900 {
		t.Fatalf("خسارةُ الخزينة=%d — المنتظَر 900 (صافي ما دُفع للمتجر)", got)
	}
	if got := ptAdvanceReturn(t, orderID); got != 900 {
		t.Fatalf("سلفةٌ رجعت للمكتب=%d — المنتظَر 900", got)
	}
	if _, _, storePaid := mcsOrderMoney(t, orderID); storePaid != 900 {
		t.Fatalf("custom_store_paid=%d — المنتظَر 900", storePaid)
	}
	if b, _ := w.Balance(ctx, driver); b != 0 {
		t.Fatalf("محفظةُ السائق=%d — ولا أجرَ لطلبٍ لم يُسلَّم", b)
	}
	if b, _ := w.Balance(ctx, rep); b != 0 {
		t.Fatalf("المندوب=%d — ولا عمولةَ لطلبٍ لم يُسلَّم", b)
	}
	if got := cqHeld(t, driver) - heldBefore; got != 0 {
		t.Fatalf("صندوقُ السائق تحرّك %d — ولم يقبض من الزبون", got)
	}
	if got := clgDebt(t, orderID, driver); got != 0 {
		t.Fatalf("دينٌ على السائق %d — والمنصّةُ تتحمّل", got)
	}
	if sum := cqLedgerSum(t, orderID); sum != -900 {
		t.Fatalf("صافي دفتر الطلب=%d — المنتظَر −900", sum)
	}
	check()
}

// ── و · أجرةُ الطلب الخاصّ تسدّ دينَ السائق ──────────────────────────────
//
//	أمانةٌ بالمحفظة · أجرةٌ ٥٠٠ بحصّة ٢٠٪ ⇒ أجرُه ٤٠٠ · وعليه دينٌ قديمٌ ٣٠٠
//	⇒ يُقيَّد ٤٠٠ ثمّ −٣٠٠ اقتطاعاً: محفظتُه ١٠٠ · دينُه صفر
//	⇒ الخزينةُ ٤٠٠ (١٠٠ حصّةٌ + ٣٠٠ دينٌ عاد) · مجموعُ الدفتر صفر
func TestMoneyCustom_FeeOffsetsDriverDebt(t *testing.T) {
	svc, w, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	ptFeeBounds(t, exec)
	mcsSetting(t, "delivery.share_amanah", "20")
	cqArmTreasury(t)
	mcsFund(t, w, customer, 1_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	mcsAmanah(t, orderID)
	if _, err := obligations.Create(ctx, pool, obligations.PartyDriver, driver, 300,
		obligations.CauseLegacy, "", nil); err != nil {
		t.Fatalf("الدين: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM obligation_settlements
			WHERE obligation_id IN (SELECT id FROM financial_obligations
			                         WHERE party_kind = 'driver' AND party_id = $1)`, driver)
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM financial_obligations WHERE party_kind = 'driver' AND party_id = $1`, driver)
		_, _ = pool.Exec(context.Background(), `UPDATE users SET driver_debt = 0 WHERE id = $1`, driver)
	})
	check := mcsGuard(t)

	if err := svc.AgreeCustom(ctx, orderID, driver, 0, 500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 500, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	ptDeliver(t, svc, driver, orderID)

	if b, _ := w.Balance(ctx, driver); b != 100 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 100 (٤٠٠ ناقصَ دينٍ ٣٠٠)", b)
	}
	if debt, err := obligations.Balance(ctx, pool, obligations.PartyDriver, driver); err != nil || debt != 0 {
		t.Fatalf("دينُ السائق الباقي=%d · %v — المنتظَر 0", debt, err)
	}
	if got := cqTreasury(t); got != 400 {
		t.Fatalf("الخزينة=%d — المنتظَر 400 (حصّةٌ ١٠٠ + دينٌ عاد ٣٠٠)", got)
	}
	if b, _ := w.Balance(ctx, customer); b != 500 {
		t.Fatalf("محفظةُ الزبون=%d — المنتظَر 500", b)
	}
	if sum := cqLedgerSum(t, orderID); sum != 0 {
		t.Fatalf("مجموعُ دفتر الطلب=%d — المنتظَر 0", sum)
	}
	check()
}
