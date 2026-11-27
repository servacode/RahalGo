package orders_test

// ══════════════════════════════════════════════════════════════════════
// **أجرةُ المكتب نهائيّة** (قرارُ المالك ٢٠٢٦-١١-٢٧)
// ══════════════════════════════════════════════════════════════════════
//
// «أيّ طلب: لمّا المكتب يعدّل أجرة التوصيل لازم تتغيّر بكلّ مكان وتبقى بالضبط السعر اللي حطّيته».
// **يُقاس أنّ لا حسابَ لاحقاً يُرجعها**: خطوةُ ثمن البضاعة، وإسنادٌ جديد (`autoAgreeFixedFee`)،
// وتعديلُ أصناف الطلب العاديّ وحذفُ متجرٍ منه — **وأنّ النقدَ وإشعارَ الكابتن بالمبلغ الجديد.**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

type feeFinalRow struct {
	customFee, deliveryFee, total, cashDue, driverFee, snapshot, reserved int64
	manual                                                                bool
}

func readFeeFinal(t *testing.T, f *cashFixture, oid string) feeFinalRow {
	t.Helper()
	var r feeFinalRow
	if err := f.pool.QueryRow(context.Background(), `
		SELECT COALESCE(custom_fee, 0), delivery_fee, total, cash_due, driver_fee,
		       COALESCE(custom_fee_snapshot, 0), custom_reserved_amount, fee_manual
		  FROM orders WHERE id = $1`, oid).
		Scan(&r.customFee, &r.deliveryFee, &r.total, &r.cashDue, &r.driverFee, &r.snapshot,
			&r.reserved, &r.manual); err != nil {
		t.Fatal(err)
	}
	return r
}

// ── أجرةُ المنصّة المفروضة: المكتبُ يكتب ٧٠٠ — لا خطوةُ الثمن ولا الإسنادُ يُرجعانها إلى ١٠٠ ──
func TestFeeFinal_AdminDefinedCustom(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	snap := int64(100)
	oid := cqSeedOrder(t, f.wallet, f.customer, f.driver, "admin_defined", &snap, false)

	// الإسنادُ يثبّت أجرةَ المنصّة ويؤكّدها تلقائيّاً.
	f.svc.AutoAgreeFixedFeeForTest(ctx, oid, f.customer, f.driver)
	if st := cqRead(t, oid); st.fee == nil || *st.fee != 100 || st.confirmedVersion == nil {
		t.Fatalf("بعد الإسناد: fee=%v confirmed=%v", st.fee, st.confirmedVersion)
	}

	res, err := f.feeEdit(t, oid, 700, false)
	if err != nil || res.Reconfirm || res.NewTotal != 700 || res.CashDue != 700 {
		t.Fatalf("تعديلُ المكتب: %v %+v — المنتظَر نافذاً بلا تأكيد ونقداً 700", err, res)
	}
	if r := readFeeFinal(t, f, oid); r.customFee != 700 || r.deliveryFee != 700 || r.cashDue != 700 ||
		r.snapshot != 700 || !r.manual {
		t.Fatalf("بعد التعديل: %+v", r)
	}

	// **خطوةُ ثمن البضاعة** — كانت تُرجع الأجرةَ إلى اللقطة ١٠٠.
	if err := f.svc.AgreeCustomStep(ctx, oid, f.driver, orders.AgreeStepGoods, 5_000, 0); err != nil {
		t.Fatalf("خطوةُ الثمن: %v", err)
	}
	r := readFeeFinal(t, f, oid)
	if r.customFee != 700 || r.deliveryFee != 700 || r.total != 5_700 {
		t.Fatalf("**خطوةُ الثمن أرجعت الأجرة**: %+v — المنتظَر 700 ومجموعاً 5700", r)
	}
	st := cqRead(t, oid)
	if _, err := f.svc.ConfirmQuote(ctx, oid, f.customer, "cash", 5_700, st.quoteVersion); err != nil {
		t.Fatalf("تأكيدُ الثمن: %v", err)
	}
	if r := readFeeFinal(t, f, oid); r.cashDue != 5_700 {
		t.Fatalf("النقدُ بعد التأكيد %d — المنتظَر 5700", r.cashDue)
	}

	// **إسنادٌ جديد** (فكٌّ ثمّ إسناد) — كان `autoAgreeFixedFee` يُرجعها إلى اللقطة.
	f.svc.AutoAgreeFixedFeeForTest(ctx, oid, f.customer, f.driver)
	if r := readFeeFinal(t, f, oid); r.customFee != 700 || r.deliveryFee != 700 || r.total != 5_700 ||
		r.cashDue != 5_700 {
		t.Fatalf("**الإسنادُ أرجع الأجرة**: %+v", r)
	}
	// **والعلامةُ وحدَها تكفي** — لقطةٌ قديمةٌ (طلبٌ قبل الإصلاح) لا تُرجعها.
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET custom_fee_snapshot = 100 WHERE id = $1`, oid); err != nil {
		t.Fatal(err)
	}
	f.svc.AutoAgreeFixedFeeForTest(ctx, oid, f.customer, f.driver)
	if err := f.svc.AgreeCustomStep(ctx, oid, f.driver, orders.AgreeStepGoods, 5_000, 0); err != nil {
		t.Fatalf("خطوةُ الثمن ثانيةً: %v", err)
	}
	if r := readFeeFinal(t, f, oid); r.customFee != 700 || r.total != 5_700 || r.cashDue != 5_700 {
		t.Fatalf("**اللقطةُ القديمةُ أرجعت الأجرة**: %+v", r)
	}

	apDrive(t, f.svc, f.driver, oid, "picked_up", "on_the_way", "at_dropoff", "delivered")
	r = readFeeFinal(t, f, oid)
	if got := f.sumKind(t, oid, "driver_earning"); got != r.driverFee {
		t.Fatalf("أجرُ السائق %d — والمنتظَر %d", got, r.driverFee)
	}
	f.green(t, "أجرة المنصّة بعد التسليم")
}

// walletCustomOrder **مشترياتٌ ٥٠٠٠ + ٥٠٠ أكّدها الزبونُ «الكل من المحفظة»** برصيدٍ `balance`.
func walletCustomOrder(t *testing.T, f *cashFixture, balance int64) string {
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
	if err := f.svc.AgreeCustom(ctx, oid, f.driver, 5_000, 500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	cqDeposit(t, f.customer, balance)
	if _, err := f.svc.ConfirmQuote(ctx, oid, f.customer, "wallet", 5_500, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	return oid
}

// ── خاصٌّ زيادةً بالمحفظة والرصيدُ يكفي: يُحجز الفرقُ فوراً ويُسلَّم بالأجرة الجديدة ──────────
func TestFeeFinal_CustomWalletIncrease(t *testing.T) {
	f := newCashFixture(t)
	oid := walletCustomOrder(t, f, 10_000)
	res, err := f.feeEdit(t, oid, 700, false)
	if err != nil || res.Reconfirm || res.NewTotal != 5_700 || res.CashDue != 0 {
		t.Fatalf("الزيادة: %v %+v — المنتظَر نافذةً بلا نقد", err, res)
	}
	if r := readFeeFinal(t, f, oid); r.reserved != 5_700 || r.cashDue != 0 || r.total != 5_700 {
		t.Fatalf("بعد الزيادة: %+v — المنتظَر حجزاً 5700", r)
	}
	if got := cqReserved(t, f.wallet, f.customer); got != 5_700 {
		t.Fatalf("محجوزُ المحفظة %d — المنتظَر 5700", got)
	}
	f.green(t, "قبل التسليم")
	apDrive(t, f.svc, f.driver, oid, "picked_up", "on_the_way", "at_dropoff", "delivered")
	if got := f.sumKind(t, oid, "driver_earning"); got != orders.DriverFeeWithShare(700, 10, 0) {
		t.Fatalf("أجرُ السائق %d — والمنتظَر %d", got, orders.DriverFeeWithShare(700, 10, 0))
	}
	f.green(t, "الخاصّ بالمحفظة بعد التسليم")
}

// ── «الكل من المحفظة» والرصيدُ لا يكفي الزيادة: تقع فوراً والباقي نقدٌ يقبضه الكابتن ──────────
// (توضيحُ المالك ٢٠٢٦-١١-٢٧: «الأجرة إلنا» — لا شيءَ ينتظر الزبون.)
func TestFeeFinal_CustomWalletShortRemainderCash(t *testing.T) {
	f := newCashFixture(t)
	oid := walletCustomOrder(t, f, 5_600) // يكفي المؤكَّدَ و١٠٠ فوقه — والزيادةُ ٢٠٠
	res, err := f.feeEdit(t, oid, 700, false)
	if err != nil || res.Reconfirm || res.NewTotal != 5_700 || res.CashDue != 100 {
		t.Fatalf("الزيادة: %v %+v — المنتظَر نافذةً ونقداً 100", err, res)
	}
	if r := readFeeFinal(t, f, oid); r.customFee != 700 || r.total != 5_700 || r.reserved != 5_600 ||
		r.cashDue != 100 {
		t.Fatalf("بعد الزيادة: %+v", r)
	}
	var drvNotes int
	_ = f.pool.QueryRow(context.Background(), `SELECT count(*) FROM notifications WHERE entity_id = $1
		AND user_id = $2 AND body LIKE '%630%' AND body LIKE '%قبضه صار 100 %'`, oid, f.driver).Scan(&drvNotes)
	if drvNotes != 1 {
		t.Fatalf("إشعارُ الكابتن بالنقد الجديد=%d", drvNotes)
	}
	f.green(t, "قبل التسليم")
	apDrive(t, f.svc, f.driver, oid, "picked_up", "on_the_way", "at_dropoff", "delivered")
	if got := f.sumKind(t, oid, "driver_earning"); got != orders.DriverFeeWithShare(700, 10, 0) {
		t.Fatalf("أجرُ السائق %d — والمنتظَر %d", got, orders.DriverFeeWithShare(700, 10, 0))
	}
	if got := cqReserved(t, f.wallet, f.customer); got != 0 {
		t.Fatalf("محجوزٌ مجمَّدٌ بعد التسليم: %d", got)
	}
	f.green(t, "بعد التسليم")
}

// ── بعد الاستلام: زيادةُ المكتب نافذةٌ فوراً، والتسليمُ لا ينتظر الزبون ──────────────────────
func TestFeeFinal_CustomAfterPickupNoWait(t *testing.T) {
	f := newCashFixture(t)
	oid := f.customFeeOrder(t, 5_000, 500)
	apDrive(t, f.svc, f.driver, oid, "picked_up", "on_the_way", "at_dropoff")
	res, err := f.feeEdit(t, oid, 800, false)
	if err != nil || res.Reconfirm || !res.AfterPickup || res.CashDue != 5_800 || res.DriverFee != 720 {
		t.Fatalf("بعد الاستلام: %v %+v — المنتظَر نافذةً ونقداً 5800", err, res)
	}
	var drvNotes int
	_ = f.pool.QueryRow(context.Background(), `SELECT count(*) FROM notifications WHERE entity_id = $1
		AND user_id = $2 AND body LIKE '%5,800%' AND body LIKE '%720%'`, oid, f.driver).Scan(&drvNotes)
	if drvNotes != 1 {
		t.Fatalf("إشعارُ الكابتن بالمبلغ الجديد=%d", drvNotes)
	}
	// **والتسليمُ يمرّ بلا تأكيدٍ من الزبون.**
	apDrive(t, f.svc, f.driver, oid, "delivered")
	if got := f.sumKind(t, oid, "driver_earning"); got != 720 {
		t.Fatalf("أجرُ السائق %d — والمنتظَر 720", got)
	}
	f.green(t, "بعد التسليم")
}

// ── العاديّ: تعديلُ الأصناف وخروجُ متجرٍ لا يُرجعان أجرةَ المكتب ───────────────────────────
func TestFeeFinal_StandardKeptAfterItemEdits(t *testing.T) {
	f := iuSetup(t, "accepted", "cash") // أ ×٢ (١٠٠٠) · ب ×١ (٥٠٠)
	ctx := context.Background()
	admin := testdb.NewUser(t, f.pool, "admin")
	var other string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, settlement_method)
		SELECT 'متجرٌ ثانٍ للأجرة', category_id, 10, 'wallet' FROM merchants WHERE id = $1
		RETURNING id::text`, f.merchantID).Scan(&other); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, other)
	})
	// «ب» من متجرٍ ثانٍ، والتوصيلُ ٣٠٠ منها ٢٠٠ رسمُ مصدرٍ إضافيّ.
	if _, err := f.pool.Exec(ctx, `UPDATE order_items SET merchant_id = $2 WHERE id = $1`, f.lineB, other); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `
		UPDATE orders SET delivery_fee = 300, driver_fee = 300, sources_fee = 200,
		       total = 2800, cash_due = 2800 WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}

	if _, _, err := f.svc.EditDeliveryFee(ctx, admin, f.orderID,
		orders.EditFeeRequest{Fee: 1_000, Reason: "مشوار أبعد"}, nil); err != nil {
		t.Fatalf("تعديلُ الأجرة: %v", err)
	}
	if st := f.read(t); st.deliveryFee != 1_000 || st.total != 3_500 {
		t.Fatalf("بعد تعديل الأجرة: %+v", st)
	}

	// **تعديلُ الأصناف** — «أ» ×٣.
	if _, _, err := f.svc.EditOrder(ctx, f.customer, f.orderID, orders.EditRequest{
		Reason: "طلب الزبون",
		Lines:  []orders.EditLine{{ItemID: f.lineA, Qty: 3}, {ItemID: f.lineB, Qty: 1}},
	}, false, nil); err != nil {
		t.Fatalf("تعديلُ الأصناف: %v", err)
	}
	if st := f.read(t); st.deliveryFee != 1_000 || st.subtotal != 3_500 || st.total != 4_500 ||
		st.cashDue != 4_500 {
		t.Fatalf("**تعديلُ الأصناف غيّر الأجرة**: %+v", st)
	}

	// **خروجُ المتجر الثاني** — كان يُسقط رسمَ المصدر من أجرة المكتب.
	if _, res, err := f.svc.MarkItemUnavailable(ctx, f.customer, f.orderID, f.lineB, 0, nil); err != nil || !res.SourceGone {
		t.Fatalf("الحذف: %v gone=%v", err, res.SourceGone)
	}
	st := f.read(t)
	if st.deliveryFee != 1_000 || st.driverFee != 1_000 || st.subtotal != 3_000 || st.total != 4_000 ||
		st.cashDue != 4_000 {
		t.Fatalf("**خروجُ المتجر أنزل أجرةَ المكتب**: %+v — المنتظَر أجرة 1000 ومجموعاً 4000", st)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **«توثيق السعر وموافقة الزبون»** (قرارُ المالك ٢٠٢٦-١١-٢٧) — مشتعلاً: الاستلامُ موقوفٌ حتّى يؤكّد
// الزبون. مطفأً: يُؤكَّد التوثيقُ تلقائيّاً ويمضي الطلب، والزبونُ يُخبَر خبراً لا طلبَ موافقة.
// ══════════════════════════════════════════════════════════════════════

// setCustomerConfirm **يضبط المفتاح لهذا الاختبار ويُعيده بعده.**
func setCustomerConfirm(t *testing.T, f *cashFixture, on bool) {
	t.Helper()
	val := "true"
	if !on {
		val = "false"
	}
	if _, err := f.pool.Exec(context.Background(), `INSERT INTO app_settings (key, value)
		VALUES ('delivery.custom_quote_customer_confirm', $1::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, val); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM app_settings WHERE key = 'delivery.custom_quote_customer_confirm'`)
	})
}

func TestCustomerConfirmSwitch_OnBlocksUntilConfirm(t *testing.T) {
	f := newCashFixture(t)
	setCustomerConfirm(t, f, true)
	ctx := context.Background()
	oid := cqSeedOrder(t, f.wallet, f.customer, f.driver, "driver_defined", nil, true)
	if err := f.svc.AgreeCustom(ctx, oid, f.driver, 5_000, 500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if st := cqRead(t, oid); st.confirmedAt != nil {
		t.Fatalf("مشتعلاً وتأكّد وحدَه: %+v", st)
	}
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, oid, "picked_up", ""); err == nil {
		t.Fatal("مشتعلاً: الاستلامُ قبل تأكيد الزبون لم يُردّ")
	}
	if _, err := f.svc.ConfirmQuote(ctx, oid, f.customer, "cash", 5_500, 1); err != nil {
		t.Fatalf("تأكيدُ الزبون: %v", err)
	}
	apDrive(t, f.svc, f.driver, oid, "picked_up", "on_the_way", "at_dropoff", "delivered")
	f.green(t, "ON")
}

func TestCustomerConfirmSwitch_OffAutoConfirms(t *testing.T) {
	f := newCashFixture(t)
	setCustomerConfirm(t, f, false)
	ctx := context.Background()
	oid := cqSeedOrder(t, f.wallet, f.customer, f.driver, "driver_defined", nil, true)
	if err := f.svc.AgreeCustom(ctx, oid, f.driver, 5_000, 500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	st := cqRead(t, oid)
	if st.confirmedVersion == nil || *st.confirmedVersion != st.quoteVersion || st.confirmedTotal == nil ||
		*st.confirmedTotal != 5_500 {
		t.Fatalf("مطفأً ولم يتأكّد تلقائيّاً: %+v", st)
	}
	if _, _, _, _, cash, _ := f.feeRow(t, oid); cash != 5_500 {
		t.Fatalf("مطفأً: النقد %d — المنتظَر 5500", cash)
	}
	// (إشعارُ الخبر يمرّ بـ`s.notify` غيرِ المُعَدّ في هذه العُدّة — فلا يُعدّ هنا.)
	apDrive(t, f.svc, f.driver, oid, "picked_up", "on_the_way", "at_dropoff", "delivered")
	f.green(t, "OFF")
}
