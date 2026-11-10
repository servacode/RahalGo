package orders_test

// ══════════════════════════════════════════════════════════════════════
// **«التوصيل من المحفظة والباقي كاش»** — قرارُ المالك ٢٠٢٦-١١-٠٩ (هجرة ٠٥٢٦)
// ══════════════════════════════════════════════════════════════════════
//
// الطلبُ الخاصّ: يُحجز عند التأكيد **الأجرةُ وحدَها** والبضاعةُ نقدٌ للكابتن، **والتسليمُ
// يسوّي الحجزَ ويقبض النقدَ معاً** — الكابتن يأخذ بضاعتَه وأجرَه، والحجزُ يُفكّ، و`wallet_paid`
// = الأجرة، والخزينةُ نصيبَها، **وقيودُ الطلب تُغلق على صفر.** وزيادةُ السعر تُبطل (قرارُ
// المالك ١: أ)، والنقصُ يُبقي، والإلغاءُ يفكّ، والأمانةُ «التوصيل» = «الكل».
//
// والطلبُ العاديّ: التعديلُ وحذفُ الصنف تحت «التوصيل من المحفظة» — **زيادةُ البضاعة نقد**،
// وتغيّرُ الأجرة يُخصم فرقُه أو يُردّ.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

var wsInv = []string{"FI-03", "FI-04", "FI-05", "FI-06", "FI-10", "FI-11"}

func wsBase(t *testing.T) fininv.Baseline {
	t.Helper()
	b, err := fininv.Capture(context.Background(), testdb.Pool(t), wsInv...)
	if err != nil {
		t.Fatalf("تصويرُ الأساس: %v", err)
	}
	return b
}

func wsGreen(t *testing.T, base fininv.Baseline) {
	t.Helper()
	vs, err := fininv.Run(context.Background(), testdb.Pool(t), wsInv...)
	if err != nil {
		t.Fatalf("الثوابت: %v", err)
	}
	for _, v := range base.New(vs) {
		t.Errorf("خرقٌ جديد: %s", v)
	}
}

type wsCustom struct {
	scope         string
	reserved      int64
	cashDue       int64
	walletPaid    int64
	paid          bool
	confirmed     bool
	paymentMethod string
}

func wsReadCustom(t *testing.T, orderID string) wsCustom {
	t.Helper()
	var s wsCustom
	if err := testdb.Pool(t).QueryRow(context.Background(), `
		SELECT wallet_scope, custom_reserved_amount, cash_due, wallet_paid,
		       custom_paid_at IS NOT NULL, quote_confirmed_at IS NOT NULL, payment_method
		FROM orders WHERE id::text = $1`, orderID).
		Scan(&s.scope, &s.reserved, &s.cashDue, &s.walletPaid, &s.paid, &s.confirmed, &s.paymentMethod); err != nil {
		t.Fatalf("قراءةُ الطلب: %v", err)
	}
	return s
}

func TestWalletPortion(t *testing.T) {
	for _, c := range []struct {
		scope            string
		total, fee, want int64
	}{
		{"full", 1_700, 500, 1_700},
		{"delivery", 1_700, 500, 500},
		{"delivery", 300, 500, 300}, // خصمٌ أنزل المجموعَ تحت الأجرة
		{"delivery", 1_700, 0, 0},   // توصيلٌ مجانيّ
		{"none", 1_700, 500, 0},
		{"delivery", 1_700, -5, 0},
	} {
		if got := orders.WalletPortion(c.scope, c.total, c.fee); got != c.want {
			t.Errorf("WalletPortion(%s, %d, %d) = %d — يُنتظر %d", c.scope, c.total, c.fee, got, c.want)
		}
	}
}

// ── التأكيدُ بـ«التوصيل» ← التسليم: الكابتن بضاعتُه وأجرُه، والحجزُ يُفكّ، والدفترُ صفر ──
func TestWScopeCustom_ConfirmDeliveryThenDeliver(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	base := wsBase(t)
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	const goods, fee = 12_000, 5_000
	platform, driverFee := shareOf(fee, 10)

	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuoteScope(ctx, orderID, customer, "wallet", "delivery", goods+fee, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	st := wsReadCustom(t, orderID)
	if st.scope != "delivery" || st.reserved != fee || st.cashDue != goods || st.walletPaid != 0 {
		t.Fatalf("بعد التأكيد: %+v — يُنتظر حجزُ الأجرة %d ونقدُ البضاعة %d ولا مدفوعَ قبل الخصم", st, fee, goods)
	}
	if r := cqReserved(t, w, customer); r != fee {
		t.Fatalf("محجوزُ المحفظة %d — يُنتظر %d", r, fee)
	}
	// **ونفسُ التأكيد ثانيةً لا شيء.**
	if _, err := svc.ConfirmQuoteScope(ctx, orderID, customer, "wallet", "delivery", goods+fee, 1); err != nil {
		t.Fatalf("تكرارُ التأكيد: %v", err)
	}
	if r := cqReserved(t, w, customer); r != fee {
		t.Fatalf("التكرارُ حرّك الحجز: %d", r)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
	st = wsReadCustom(t, orderID)
	if st.reserved != 0 || !st.paid || st.walletPaid != fee || st.cashDue != goods {
		t.Fatalf("بعد التسليم: %+v — يُنتظر حجزٌ صفر ومدفوعٌ = الأجرة", st)
	}
	if r := cqReserved(t, w, customer); r != 0 {
		t.Fatalf("محجوزُ المحفظة بعد التسوية %d", r)
	}
	if b, _ := w.Balance(ctx, customer); b != 100_000-fee {
		t.Fatalf("رصيدُ الزبون %d — يُنتظر %d (الأجرةُ وحدَها)", b, 100_000-fee)
	}
	if b, _ := w.Balance(ctx, driver); b != driverFee {
		t.Fatalf("رصيدُ الكابتن %d — يُنتظر أجرُه %d (لا يخسر أجرتَه)", b, driverFee)
	}
	// **الكابتن قبض البضاعة ودفعها للمتجر** — فلا يدين للمكتب بشيءٍ منها.
	if h := cqHeld(t, driver); h != 0 {
		t.Fatalf("صندوقُ الكابتن %d — يُنتظر صفر (قبض البضاعة ودفعها للمتجر)", h)
	}
	if got := cqTreasury(t); got != platform {
		t.Fatalf("الخزينة %d — يُنتظر %d", got, platform)
	}
	if sum := cqLedgerSum(t, orderID); sum != 0 {
		t.Fatalf("مجموعُ قيود الطلب %d — يُنتظر صفر", sum)
	}
	wsGreen(t, base)
}

// ── زيادةُ السعر بعد «التوصيل» تُبطل وتفكّ الحجز (قرارُ المالك ١: أ)، والنقصُ يُبقي ──
func TestWScopeCustom_IncreaseInvalidatesDecreaseKeeps(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	base := wsBase(t)
	cqDeposit(t, customer, 100_000)

	up := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, up, driver, 6_000, 1_500); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmQuoteScope(ctx, up, customer, "wallet", "delivery", 7_500, 1); err != nil {
		t.Fatal(err)
	}
	if err := svc.AgreeCustom(ctx, up, driver, 8_000, 1_500); err != nil {
		t.Fatalf("الزيادة: %v", err)
	}
	st := wsReadCustom(t, up)
	if st.confirmed || st.reserved != 0 {
		t.Fatalf("بعد الزيادة: %+v — يُنتظر تأكيدٌ مُبطَل وحجزٌ مفكوك", st)
	}
	if r := cqReserved(t, w, customer); r != 0 {
		t.Fatalf("محجوزُ المحفظة %d بعد الزيادة", r)
	}
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, up, "picked_up", ""); !errors.Is(err, orders.ErrQuoteNotConfirmed) {
		t.Fatalf("الاستلامُ بلا تأكيد: %v", err)
	}
	// **ويختار ثانيةً بالسعر الجديد.**
	if _, err := svc.ConfirmQuoteScope(ctx, up, customer, "wallet", "delivery", 9_500, 2); err != nil {
		t.Fatalf("التأكيدُ الجديد: %v", err)
	}
	if st := wsReadCustom(t, up); st.reserved != 1_500 || st.cashDue != 8_000 {
		t.Fatalf("بعد التأكيد الجديد: %+v", st)
	}

	down := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, down, driver, 6_000, 1_500); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmQuoteScope(ctx, down, customer, "wallet", "delivery", 7_500, 1); err != nil {
		t.Fatal(err)
	}
	if err := svc.AgreeCustom(ctx, down, driver, 4_000, 1_500); err != nil {
		t.Fatalf("النقص: %v", err)
	}
	st = wsReadCustom(t, down)
	if !st.confirmed || st.reserved != 1_500 || st.cashDue != 4_000 {
		t.Fatalf("بعد النقص: %+v — يُنتظر تأكيدٌ باقٍ وحجزُ الأجرة ونقدُ البضاعة الجديدة", st)
	}
	// **والإلغاءُ يفكّ الحجز.**
	if _, err := svc.Transition(ctx, customer, []string{"admin"}, down, "cancelled", "ws"); err != nil {
		t.Fatalf("الإلغاء: %v", err)
	}
	if st := wsReadCustom(t, down); st.reserved != 0 {
		t.Fatalf("بعد الإلغاء: %+v", st)
	}
	if r := cqReserved(t, w, customer); r != 1_500 {
		t.Fatalf("محجوزُ المحفظة %d — يُنتظر حجزُ الطلب الأوّل وحدَه 1500", r)
	}
	wsGreen(t, base)
}

// ── الأمانة: «التوصيل» = «الكل» ──────────────────────────────────────────
func TestWScopeCustom_AmanahDeliveryEqualsFull(t *testing.T) {
	svc, w, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	base := wsBase(t)
	cqDeposit(t, customer, 50_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	exec(ctx, `UPDATE orders SET custom_mode = 'amanah' WHERE id::text = $1`, orderID)
	if err := svc.AgreeCustom(ctx, orderID, driver, 0, 2_000); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuoteScope(ctx, orderID, customer, "wallet", "delivery", 2_000, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	if st := wsReadCustom(t, orderID); st.reserved != 2_000 || st.cashDue != 0 {
		t.Fatalf("الأمانة: %+v — يُنتظر حجزُ الأجرة كلّها", st)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
	if st := wsReadCustom(t, orderID); st.reserved != 0 || st.walletPaid != 2_000 || !st.paid {
		t.Fatalf("بعد التسليم: %+v", st)
	}
	if sum := cqLedgerSum(t, orderID); sum != 0 {
		t.Fatalf("مجموعُ قيود الأمانة %d", sum)
	}
	wsGreen(t, base)
}

// ── التبديلُ بعد الاستلام: «الكل» ← «التوصيل» ← تسليمٌ أخضر ─────────────────
func TestWScopeCustom_ChangeAfterPickup(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	base := wsBase(t)
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	const goods, fee = 9_000, 2_000
	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatal(err)
	}
	// **وقبل التأكيد لا تبديل** — يختار من شاشة التأكيد.
	if _, err := svc.ChangeWalletScope(ctx, orderID, customer, "delivery", goods+fee); !errors.Is(err, orders.ErrWalletScopeNotAllowed) {
		t.Fatalf("قبل التأكيد: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", goods+fee, 1); err != nil {
		t.Fatal(err)
	}
	if st := wsReadCustom(t, orderID); st.scope != "full" || st.reserved != goods+fee {
		t.Fatalf("التطبيقُ القديم (wallet بلا نطاق): %+v — يُنتظر «الكل»", st)
	}
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ChangeWalletScope(ctx, orderID, customer, "delivery", goods+fee); err != nil {
		t.Fatalf("التبديل: %v", err)
	}
	if st := wsReadCustom(t, orderID); st.scope != "delivery" || st.reserved != fee || st.cashDue != goods {
		t.Fatalf("بعد التبديل: %+v", st)
	}
	if r := cqReserved(t, w, customer); r != fee {
		t.Fatalf("محجوزُ المحفظة %d — يُنتظر %d", r, fee)
	}
	for _, to := range []string{"on_the_way", "at_dropoff"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.ChangeWalletScope(ctx, orderID, customer, "full", goods+fee); !errors.Is(err, orders.ErrWalletScopeLocked) {
		t.Fatalf("عند الباب: %v — يُنتظر wallet_scope_locked", err)
	}
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "delivered", ""); err != nil {
		t.Fatal(err)
	}
	if st := wsReadCustom(t, orderID); st.reserved != 0 || st.walletPaid != fee {
		t.Fatalf("بعد التسليم: %+v", st)
	}
	if sum := cqLedgerSum(t, orderID); sum != 0 {
		t.Fatalf("مجموعُ قيود الطلب %d", sum)
	}
	wsGreen(t, base)
}

// ── العاديّ تحت «التوصيل»: التعديلُ يجعل زيادةَ البضاعة نقداً، والحذفُ يُنقص النقد ──
func TestWScopeRegular_EditAndRemovalUnderDelivery(t *testing.T) {
	f := iuSetup(t, "accepted", "cash") // أ ×٢ (١٠٠٠) · ب ×١ (٥٠٠) · توصيل ١٠٠ = ٢٦٠٠
	ctx := context.Background()
	base := wsBase(t)
	if _, err := f.wallet.Apply(ctx, f.customer, 5_000, "topup", "ws-seed-"+f.orderID, "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := f.wallet.Apply(ctx, f.customer, -100, "order_payment", f.orderID, "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET payment_method = 'wallet', wallet_scope = 'delivery',
		wallet_paid = 100, cash_due = 2500 WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	// زيادة: «أ» ×٣ ⇒ بضاعة ٣٥٠٠ — **نقداً كلُّها**، والمحفظةُ لا تُمسّ.
	if _, res, err := f.svc.EditOrder(ctx, f.customer, f.orderID, orders.EditRequest{
		Reason: "ws", Lines: []orders.EditLine{{ItemID: f.lineA, Qty: 3}},
	}, false, nil); err != nil || res.Charged != 0 || res.Refunded != 0 {
		t.Fatalf("الزيادة: %v %+v — يُنتظر لا خصمَ ولا ردّ", err, res)
	}
	st := f.read(t)
	if st.walletPaid != 100 || st.cashDue != 3_500 || st.total != 3_600 {
		t.Fatalf("بعد الزيادة: %+v — يُنتظر محفظةٌ ١٠٠ ونقدٌ ٣٥٠٠", st)
	}
	// حذفُ «ب» (غير متوفّر) ⇒ النقدُ ينقص ٥٠٠، والمحفظةُ كما هي.
	if _, res, err := f.svc.MarkItemUnavailable(ctx, f.customer, f.orderID, f.lineB, 0, nil); err != nil || res.Refunded != 0 {
		t.Fatalf("الحذف: %v %+v", err, res)
	}
	st = f.read(t)
	if st.walletPaid != 100 || st.cashDue != 3_000 {
		t.Fatalf("بعد الحذف: %+v — يُنتظر محفظةٌ ١٠٠ ونقدٌ ٣٠٠٠", st)
	}
	wsGreen(t, base)
}
