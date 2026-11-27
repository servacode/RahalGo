package orders_test

// ══════════════════════════════════════════════════════════════════════
// **«تعديل الأجرة» — العاديّ والخاصّ** (قرارُ المالك ٢٠٢٦-١١-١٨)
// ══════════════════════════════════════════════════════════════════════
//
// **يُقاس ما يقع ثمّ يُسلَّم الطلب**: أجرُ السائق عند التسليم = `DriverFeeWithShare` على الأجرة
// الجديدة بلقطة الطلب، والدفترُ أخضر. والتوصيلةُ (`merchant_delivery`) تُقاس بالمسار الحيّ في
// `qa/fee_edit_merchant_delivery_test.go`.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

func (f *cashFixture) feeEdit(t *testing.T, orderID string, fee int64, dry bool) (orders.EditFeeResult, error) {
	t.Helper()
	_, res, err := f.svc.EditDeliveryFee(context.Background(), f.treasury, orderID,
		orders.EditFeeRequest{Fee: fee, Reason: "طلب الزبون مشوار أبعد", DryRun: dry}, nil)
	return res, err
}

// share **حصّةُ المنصّة ١٠٪ على لقطة الطلب** — فيُقاس أنّ القسمةَ تُعاد على الأجرة الجديدة.
func (f *cashFixture) share(t *testing.T, orderID string, pct, fixed int64) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), `
		UPDATE orders SET snap_platform_delivery_percent = $2::bigint, snap_platform_delivery_fixed = $3::bigint,
		       driver_fee = delivery_fee - CASE WHEN $3::bigint > 0 THEN LEAST($3::bigint, delivery_fee)
		                                        ELSE delivery_fee * $2::bigint / 100 END
		 WHERE id = $1`, orderID, pct, fixed); err != nil {
		t.Fatal(err)
	}
}

func (f *cashFixture) feeRow(t *testing.T, orderID string) (fee, waived, driverFee, total, cashDue, walletPaid int64) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(), `
		SELECT delivery_fee, promo_delivery_waived, driver_fee, total, cash_due, wallet_paid
		  FROM orders WHERE id = $1`, orderID).
		Scan(&fee, &waived, &driverFee, &total, &cashDue, &walletPaid); err != nil {
		t.Fatal(err)
	}
	return
}

// ── نقداً: المعاينةُ لا تترك أثراً، ثمّ زيادةٌ يقبضها السائق ويأخذ أجرَه منها ──────────
func TestFeeEdit_StandardCash(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash") // at_pickup، نقداً ٦٠٬٠٠٠
	f.share(t, oid, 10, 0)
	ctx := context.Background()

	res, err := f.feeEdit(t, oid, 12_000, true)
	if err != nil {
		t.Fatalf("المعاينة: %v", err)
	}
	if res.OldFee != 10_000 || res.NewFee != 12_000 || res.DriverFee != 10_800 || res.PlatformShare != 1_200 ||
		res.OldDriverFee != 9_000 || res.NewTotal != 62_000 || res.CashDue != 62_000 || !res.DryRun {
		t.Fatalf("المعاينة: %+v", res)
	}
	if fee, _, df, total, cash, _ := f.feeRow(t, oid); fee != 10_000 || df != 9_000 || total != 60_000 || cash != 60_000 {
		t.Fatalf("**المعاينةُ غيّرت الطلب**: %d/%d/%d/%d", fee, df, total, cash)
	}
	var n int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE entity_id = $1`, oid).Scan(&n)
	if n != 0 {
		t.Fatalf("المعاينةُ أرسلت %d إشعاراً", n)
	}

	if _, err := f.feeEdit(t, oid, 12_000, false); err != nil {
		t.Fatalf("التعديل: %v", err)
	}
	if fee, waived, df, total, cash, wp := f.feeRow(t, oid); fee != 12_000 || waived != 0 || df != 10_800 ||
		total != 62_000 || cash != 62_000 || wp != 0 {
		t.Fatalf("بعد التعديل: %d/%d/%d/%d/%d/%d", fee, waived, df, total, cash, wp)
	}
	var events, custNotes, drvNotes int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM order_events WHERE order_id = $1 AND note LIKE '%تعديل الأجرة%'
		AND note LIKE '%مشوار أبعد%'`, oid).Scan(&events)
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE entity_id = $1 AND user_id = $2
		AND body LIKE '%62,000%'`, oid, f.customer).Scan(&custNotes)
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE entity_id = $1 AND user_id = $2
		AND body LIKE '%10,800%'`, oid, f.driver).Scan(&drvNotes)
	if events != 1 || custNotes != 1 || drvNotes != 1 {
		t.Fatalf("الأثر: حدث=%d زبون=%d سائق=%d", events, custNotes, drvNotes)
	}

	f.deliver(t, oid)
	if got := f.sumKind(t, oid, "driver_earning"); got != orders.DriverFeeWithShare(12_000, 10, 0) {
		t.Fatalf("أجرُ السائق %d — والمنتظَر %d", got, orders.DriverFeeWithShare(12_000, 10, 0))
	}
	f.green(t, "بعد التسليم")
}

// ── بالمحفظة: النقصُ يُردّ، والزيادةُ تُخصم بما يكفي والباقي نقد ────────────────────
func TestFeeEdit_StandardWallet(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.orderWalletPaid(t, m, 50_000, 10_000) // دُفع ٦٠٬٠٠٠ من المحفظة، والرصيدُ صفر
	f.share(t, oid, 10, 0)

	res, err := f.feeEdit(t, oid, 8_000, false)
	if err != nil || res.Refunded != 2_000 || res.Charged != 0 {
		t.Fatalf("النقص: %v %+v", err, res)
	}
	if b := f.balance(t, f.customer); b != 2_000 {
		t.Fatalf("الرصيدُ بعد النقص %d — والمنتظَر 2000", b)
	}
	if _, _, _, total, cash, wp := f.feeRow(t, oid); total != 58_000 || wp != 58_000 || cash != 0 {
		t.Fatalf("بعد النقص: %d/%d/%d", total, cash, wp)
	}

	// زيادةٌ إلى ١٣٬٠٠٠ ⇒ المطلوب ٦٣٬٠٠٠: يُخصم الرصيدُ ٢٬٠٠٠ والباقي ٣٬٠٠٠ نقد.
	res, err = f.feeEdit(t, oid, 13_000, false)
	if err != nil || res.Charged != 2_000 || res.CashDue != 3_000 || res.NewTotal != 63_000 {
		t.Fatalf("الزيادة: %v %+v", err, res)
	}
	if b := f.balance(t, f.customer); b != 0 {
		t.Fatalf("الرصيدُ بعد الزيادة %d", b)
	}
	f.green(t, "قبل التسليم")
	f.deliver(t, oid)
	if got := f.sumKind(t, oid, "driver_earning"); got != orders.DriverFeeWithShare(13_000, 10, 0) {
		t.Fatalf("أجرُ السائق %d — والمنتظَر %d", got, orders.DriverFeeWithShare(13_000, 10, 0))
	}
	f.green(t, "بعد التسليم")
}

// ── توصيلٌ مجانيٌّ بسقف: يُعاد على الأجرة الجديدة، والسائقُ بأجره كاملاً ─────────────────
func TestFeeEdit_StandardFreeDeliveryCap(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	f.share(t, oid, 10, 0)
	ctx := context.Background()
	var camp string
	now := time.Now()
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO free_delivery_campaigns (active, starts_at, ends_at, kinds, max_fee)
		VALUES (false, $1, $2, ARRAY['standard'], 6000) RETURNING id::text`,
		now.Add(-time.Hour), now.Add(time.Hour)).Scan(&camp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM free_delivery_campaigns WHERE id = $1`, camp)
	})
	// **الطلبُ نال العرضَ عند إنشائه**: أُعفي من ٦٬٠٠٠ ودفع ٤٬٠٠٠.
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO free_delivery_claims (campaign_id, claimant, order_id, kind, waived)
		VALUES ($1, 'fee-edit-'||$2::text, $2::uuid, 'standard', 6000)`, camp, oid); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `
		UPDATE orders SET delivery_fee = 4000, promo_delivery_waived = 6000, total = 54000, cash_due = 54000
		 WHERE id = $1`, oid); err != nil {
		t.Fatal(err)
	}

	res, err := f.feeEdit(t, oid, 15_000, false)
	if err != nil || res.OldFee != 10_000 || res.Waived != 6_000 || res.NewTotal != 59_000 ||
		res.DriverFee != 13_500 {
		t.Fatalf("الزيادة مع العرض: %v %+v", err, res)
	}
	// ونقصٌ تحت السقف ⇒ يُعفى من كلّها.
	res, err = f.feeEdit(t, oid, 5_000, false)
	if err != nil || res.Waived != 5_000 || res.NewTotal != 50_000 || res.DriverFee != 4_500 {
		t.Fatalf("النقص تحت السقف: %v %+v", err, res)
	}
	var claimed int64
	_ = f.pool.QueryRow(ctx, `SELECT waived FROM free_delivery_claims WHERE order_id = $1`, oid).Scan(&claimed)
	if claimed != 5_000 {
		t.Fatalf("صفُّ العرض %d — والمنتظَر 5000", claimed)
	}
	f.deliver(t, oid)
	if got := f.sumKind(t, oid, "driver_earning"); got != 4_500 {
		t.Fatalf("أجرُ السائق %d — والمنتظَر 4500 (الأجرةُ كاملةً ناقصَ الحصّة)", got)
	}
	f.green(t, "بعد التسليم")
}

// ── ما يُردّ: بلا سبب، بلا تغيير، بعد التسليم، وأجرةٌ تأكلها الحصّةُ الثابتة ─────────────
func TestFeeEdit_Guards(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	ctx := context.Background()
	if _, _, err := f.svc.EditDeliveryFee(ctx, f.treasury, oid,
		orders.EditFeeRequest{Fee: 12_000, Reason: "  "}, nil); !errors.Is(err, orders.ErrOrderEditReason) {
		t.Fatalf("بلا سبب: %v", err)
	}
	if _, err := f.feeEdit(t, oid, 10_000, false); !errors.Is(err, orders.ErrOrderEditNoChange) {
		t.Fatalf("بلا تغيير: %v", err)
	}
	if _, err := f.feeEdit(t, oid, 0, false); !errors.Is(err, orders.ErrFeeEditRange) {
		t.Fatalf("صفر: %v", err)
	}
	f.share(t, oid, 0, 9_000)
	if _, err := f.feeEdit(t, oid, 8_000, false); !errors.Is(err, orders.ErrFeeEditRange) {
		t.Fatalf("أجرةٌ تحت الحصّة الثابتة: %v", err)
	}
	if fee, _, _, total, _, _ := f.feeRow(t, oid); fee != 10_000 || total != 60_000 {
		t.Fatalf("تغيّر الطلبُ بعد ردود: %d/%d", fee, total)
	}
	f.deliver(t, oid)
	if _, err := f.feeEdit(t, oid, 12_000, false); !errors.Is(err, orders.ErrOrderEditClosed) {
		t.Fatalf("بعد التسليم: %v", err)
	}
}

// customFeeOrder **مشترياتٌ وُثّق ثمنُها وأجرتُها وأكّدها الزبونُ نقداً** — على الخزينة المسلّحة.
func (f *cashFixture) customFeeOrder(t *testing.T, goods, fee int64) string {
	t.Helper()
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `INSERT INTO app_settings (key, value) VALUES
		('delivery.share_purchase', '10'::jsonb) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key = 'delivery.share_purchase'`)
	})
	oid := cqSeedOrder(t, f.wallet, f.customer, f.driver, "driver_defined", nil, true)
	if err := f.svc.AgreeCustom(ctx, oid, f.driver, goods, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := f.svc.ConfirmQuote(ctx, oid, f.customer, "cash", goods+fee, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	return oid
}

// ── الخاصّ نقصاً: يبقى التأكيد، ويُسلَّم بالأجرة الجديدة ──────────────────────────────
func TestFeeEdit_CustomDecrease(t *testing.T) {
	f := newCashFixture(t)
	oid := f.customFeeOrder(t, 5_000, 500)
	res, err := f.feeEdit(t, oid, 400, true)
	if err != nil || res.OldFee != 500 || res.NewTotal != 5_400 || res.DriverFee != 360 || res.Reconfirm {
		t.Fatalf("المعاينة: %v %+v", err, res)
	}
	if st := cqRead(t, oid); *st.fee != 500 || st.quoteVersion != 1 {
		t.Fatalf("**المعاينةُ غيّرت العرض**: %+v", st)
	}
	if _, err := f.feeEdit(t, oid, 400, false); err != nil {
		t.Fatalf("التعديل: %v", err)
	}
	if st := cqRead(t, oid); *st.fee != 400 || *st.total != 5_400 || st.confirmedVersion == nil ||
		*st.confirmedVersion != 2 {
		t.Fatalf("بعد النقص: fee=%v total=%v confirmed=%v", *st.fee, *st.total, st.confirmedVersion)
	}
	apDrive(t, f.svc, f.driver, oid, "picked_up", "on_the_way", "at_dropoff", "delivered")
	if got := f.sumKind(t, oid, "driver_earning"); got != orders.DriverFeeWithShare(400, 10, 0) {
		t.Fatalf("أجرُ السائق %d — والمنتظَر %d", got, orders.DriverFeeWithShare(400, 10, 0))
	}
	f.green(t, "الخاصّ بعد التسليم")
}

// ── الخاصّ زيادةً: تقع فوراً (قرارُ المالك ٢٠٢٦-١١-٢٧) — النقدُ وإشعارُ الكابتن بالمبلغ الجديد ──
func TestFeeEdit_CustomIncreaseAppliesNow(t *testing.T) {
	f := newCashFixture(t)
	oid := f.customFeeOrder(t, 5_000, 500)
	res, err := f.feeEdit(t, oid, 700, false)
	if err != nil || res.Reconfirm || res.NewTotal != 5_700 || res.DriverFee != 630 || res.CashDue != 5_700 {
		t.Fatalf("الزيادة: %v %+v — المنتظَر نافذةً ونقداً 5700", err, res)
	}
	if st := cqRead(t, oid); st.confirmedVersion == nil || *st.confirmedVersion != 2 ||
		st.confirmedTotal == nil || *st.confirmedTotal != 5_700 {
		t.Fatalf("المؤكَّد: %+v", st)
	}
	if _, _, _, _, cash, _ := f.feeRow(t, oid); cash != 5_700 {
		t.Fatalf("النقدُ %d — المنتظَر 5700", cash)
	}
	var drvNotes int
	_ = f.pool.QueryRow(context.Background(), `SELECT count(*) FROM notifications WHERE entity_id = $1
		AND user_id = $2 AND body LIKE '%5,700%' AND body LIKE '%630%'`, oid, f.driver).Scan(&drvNotes)
	if drvNotes != 1 {
		t.Fatalf("إشعارُ الكابتن بالمبلغ الجديد=%d", drvNotes)
	}
	apDrive(t, f.svc, f.driver, oid, "picked_up", "on_the_way", "at_dropoff", "delivered")
	if got := f.sumKind(t, oid, "driver_earning"); got != 630 {
		t.Fatalf("أجرُ السائق %d — والمنتظَر 630", got)
	}
	f.green(t, "الخاصّ بعد التسليم")
}

// ── الخاصّ قبل توثيق الأجرة يُردّ ───────────────────────────────────────────────────
func TestFeeEdit_CustomNotAgreed(t *testing.T) {
	f := newCashFixture(t)
	oid := cqSeedOrder(t, f.wallet, f.customer, f.driver, "driver_defined", nil, true)
	if _, err := f.feeEdit(t, oid, 700, false); !errors.Is(err, orders.ErrFeeEditNotAgreed) {
		t.Fatalf("قبل التوثيق: %v", err)
	}
}
