package orders_test

// نصيبُ المنصّة من الطلب الخاصّ — **قرارُ المالك النهائيُّ ٢٠٢٦-٠٩-٢٩.**
//
// # ما تبدّل ولماذا
//
// **كان الخاصُّ ممرّاً محضاً**: يُخصم من الزبون ما اتُّفق عليه ويُودَع
// للسائق كلُّه، **والمنصّةُ تعبر بلا أن تأخذ** (قرارُ المالك ٢٠٢٦-٠٨-٠٩).
//
// **وشهده المالكُ على شاشته ٢٠٢٦-٠٩-٢٩**: «خزينة المنصة فارغة مازالت صفر»
// — **فنقض قرارَه الأوّلَ بقرارٍ نهائيٍّ في اليوم نفسِه.**
//
// # والقاعدةُ الفارقةُ: البضاعةُ ردٌّ لا كسب
//
//	goods_amount   ⇐ **ردُّ مالٍ للسائق بالكامل** — لا تأخذ المنصّةُ منه شيئاً
//	delivery_fee   ⇐ **وحدَها تخضع لنصيب المنصّة**
//
// **والسائقُ دفع ثمنَ البضاعة من جيبه** — **فأخذُ نسبةٍ منه اقتطاعٌ من رأس
// ماله لا من ربحه**، ويجعله يخسر بكلّ طلب.
//
// # ومثالُ المالك نصّاً
//
//	goods 12,000 · fee 5,000 · total 17,000
//	platform 500 · driver delivery 4,500 · driver goods 12,000
//	⇒ سائقٌ 16,500 · خزينةٌ 500 · زبونٌ −17,000
//	⇒ −17,000 + 16,500 + 500 = 0

import (
	"context"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"testing"
)

// shareOf نصيبُ المنصّة من الأجرة وحدَها — **لا من البضاعة.**
func shareOf(fee, pct int64) (platform, driverFee int64) {
	platform = fee * pct / 100
	return platform, fee - platform
}

// ── أ · مثالُ المالك حرفاً — والمعادلةُ تُغلق على صفر ───────────────────
func TestCustomShare_OwnerExampleClosesToZero(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)

	const goods, fee = 12_000, 5_000
	platform, driverFee := shareOf(fee, 10)

	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", goods+fee, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}

	// **والسائقُ يأخذ البضاعةَ كاملةً وأجرَه بعد النصيب.**
	wantDriver := int64(goods + driverFee) // 16,500
	if b, _ := w.Balance(ctx, driver); b != wantDriver {
		t.Fatalf("رصيدُ السائق=%d — المنتظَر %d (بضاعةٌ %d + أجرٌ %d)",
			b, wantDriver, goods, driverFee)
	}
	// **والزبونُ يدفع المجموعَ لا أقلَّ ولا أكثر.**
	if b, _ := w.Balance(ctx, customer); b != 100_000-(goods+fee) {
		t.Fatalf("رصيدُ الزبون=%d — المنتظَر %d", b, 100_000-(goods+fee))
	}
	// **والخزينةُ تأخذ نصيبَها** — وهو الفرقُ بعينه.
	if got := cqTreasury(t); got != platform {
		t.Fatalf("الخزينة=%d — المنتظَر %d (١٠٪ من أجرةِ %d وحدَها)", got, platform, fee)
	}
	// ══════════════════════════════════════════════════════════════════
	// **والمعادلةُ تُغلق على صفرٍ أو لا تُغلق** — نصُّ المالك
	// ══════════════════════════════════════════════════════════════════
	//
	// **ومجموعُ قيودِ الطلب كلِّها صفرٌ** — **ولا يكفي أن يبدو كلُّ طرفٍ
	// صحيحاً**: ليرةٌ تُخصم من الزبون ولا تصل أحداً لا تظهر في أيّ طرف،
	// **وتظهر في المجموع وحدَه.**
	if sum := cqLedgerSum(t, orderID); sum != 0 {
		t.Fatalf("مجموعُ قيود الطلب=%d — **والمعادلةُ يجب أن تُغلق على صفر**", sum)
	}
}

// ── ب · ولا تأخذ المنصّةُ من البضاعة شيئاً ─────────────────────────────
//
// **وهو الشرطُ الذي شدّد عليه المالك**: «لا تأخذ نسبة المنصة من
// goods_amount».
//
// **وبضاعةٌ كبيرةٌ وأجرةٌ صغيرةٌ تكشف الخلط**: لو أُخذت النسبةُ من المجموع
// لبلغ نصيبُ المنصّة ٢٬٠٠٠ بدل ١٠٠ — **عشرون ضعفاً، ومن جيب السائق.**
func TestCustomShare_GoodsNeverTaxed(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)

	const goods, fee = 19_000, 1_000
	platform, driverFee := shareOf(fee, 10) // 100 · 900

	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", goods+fee, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
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
		t.Fatalf("رصيدُ السائق=%d — المنتظَر %d", b, goods+driverFee)
	}
}

// ── ج · والنسبةُ تُلقَط على الطلب فلا يمسّ تغييرُها ما مضى ──────────────
//
// **نصُّ المالك**: «تغيير النسبة لاحقاً لا يؤثر على أي طلب قديم».
//
// **واللقطةُ تُقرأ من الصفّ لا من الإعداد** — **ولو قُرئ الإعدادُ وقتَ
// التسوية لتبدّل مالُ طلبٍ اتُّفق عليه أمس.**
func TestCustomShare_SnapshotSurvivesSettingChange(t *testing.T) {
	svc, w, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)

	const goods, fee = 10_000, 2_000
	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", goods+fee, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}

	// **ثمّ تُرفع النسبةُ إلى النصف بعد الاتّفاق** — ولا يجوز أن تمسّه.
	//
	// **وتُعاد بعده** — **والقاعدةُ مشتركةٌ بين الاختبارات**: إعدادٌ يُرفع
	// ولا يُردّ يجعل اختباراً آخرَ يقرأ نسبةً لم يضعها، **فيسقط بسببٍ لا
	// يخصّه ويُبحَث عن العلّة في الموضع الخطأ.** (وقع فعلاً في هذه الدفعة.)
	exec(ctx, `INSERT INTO app_settings (key, value) VALUES
	           ('delivery.merchant_delivery_platform_percent', '50'::jsonb)
	           ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`)
	t.Cleanup(func() {
		exec(context.Background(),
			`DELETE FROM app_settings WHERE key = 'delivery.merchant_delivery_platform_percent'`)
	})

	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
	platform, driverFee := shareOf(fee, 10) // **بالنسبة القديمة**
	if got := cqTreasury(t); got != platform {
		t.Fatalf("الخزينة=%d — المنتظَر %d باللقطة القديمة. **النسبةُ الجديدةُ مسّت طلباً مضى**",
			got, platform)
	}
	if b, _ := w.Balance(ctx, driver); b != goods+driverFee {
		t.Fatalf("رصيدُ السائق=%d — المنتظَر %d", b, goods+driverFee)
	}
}

// ── د · وإلغاءٌ بعد الاتّفاق لا يترك أثراً ماليّاً ──────────────────────
//
// **نصُّ المالك**: «cancellation after quote where applicable».
//
// **والحجزُ يُفكّ ولا خزينةَ تُقيَّد**: نصيبُ المنصّة من تسليمٍ وقع، **ولا
// شيءَ وقع.**
func TestCustomShare_CancelAfterQuoteLeavesNoMoney(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)

	if err := svc.AgreeCustom(ctx, orderID, driver, 8_000, 2_000); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 10_000, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	// **والإلغاءُ بعد انطلاق السائق للمكتب** (قرارُ المالك ٢٠٢٦-١٠-٠٣) — الزبونُ كتب في الدردشة.
	ops := testdb.NewUser(t, testdb.Pool(t), "admin")
	if _, err := svc.Transition(ctx, ops, []string{"admin"}, orderID, "cancelled", "الزبونُ عدل"); err != nil {
		t.Fatalf("الإلغاء: %v", err)
	}
	if got := cqTreasury(t); got != 0 {
		t.Fatalf("الخزينة=%d بعد إلغاء — **ونصيبٌ من تسليمٍ لم يقع**", got)
	}
	if b, _ := w.Balance(ctx, customer); b != 100_000 {
		t.Fatalf("رصيدُ الزبون=%d — المنتظَر أن يعود كاملاً", b)
	}
	if r := cqReserved(t, w, customer); r != 0 {
		t.Fatalf("محجوزٌ باقٍ=%d بعد الإلغاء", r)
	}
}

// ── هـ · والتسليمُ مرّتين لا يُقيَّد مرّتين ──────────────────────────────
//
// **نصُّ المالك**: «idempotency».
//
// **والتسليمُ قد يُنادى مرّتين** — شبكةٌ تتعثّر فيُعاد النداء. **وقيدٌ ثانٍ
// يعني خصماً ثانياً من زبونٍ دفع مرّة.**
func TestCustomShare_SettlementIsIdempotent(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)

	const goods, fee = 6_000, 4_000
	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", goods+fee, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
	before, _ := w.Balance(ctx, driver)
	treasuryBefore := cqTreasury(t)

	// **ونداءٌ ثانٍ للتسليم** — يُردّ بانتقالٍ غيرِ صالحٍ أو يمرّ بلا أثر،
	// **وكلاهما مقبول. والمرفوضُ أثرٌ ماليٌّ ثانٍ.**
	_, _ = svc.Transition(ctx, driver, []string{"driver"}, orderID, "delivered", "")

	if after, _ := w.Balance(ctx, driver); after != before {
		t.Fatalf("رصيدُ السائق تبدّل بنداءٍ ثانٍ: %d ⇐ %d", before, after)
	}
	if got := cqTreasury(t); got != treasuryBefore {
		t.Fatalf("الخزينةُ تبدّلت بنداءٍ ثانٍ: %d ⇐ %d", treasuryBefore, got)
	}
	if sum := cqLedgerSum(t, orderID); sum != 0 {
		t.Fatalf("المجموعُ=%d بعد نداءٍ ثانٍ — **قيدٌ مكرَّر**", sum)
	}
}
