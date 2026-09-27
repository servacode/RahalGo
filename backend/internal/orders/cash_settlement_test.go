package orders_test

// ══════════════════════════════════════════════════════════════════════
// **تسويةُ المتجر نقداً/محفظةً — البراهينُ SET-01..30** — دورةُ ٢٠٢٦-٠٩-٢٧
// ══════════════════════════════════════════════════════════════════════
//
// **بالحساب لا بالانطباع**: كلُّ اختبارٍ يمشي المسارَ الحقيقيَّ (settleMerchant
// عبر الانتقال) ويقيس الدفترَ وصفَّ التسوية والاحتباسَ والالتزامَ، ويشغّل
// ثوابتَ FI-14 حيّةً.

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

type cashFixture struct {
	pool       *pgxpool.Pool
	svc        *orders.Service
	wallet     *wallet.Service
	customer   string
	driver     string
	treasury   string
	holding    string
	categoryID string
}

func newCashFixture(t *testing.T) *cashFixture {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	store := settings.NewStore(pool)

	f := &cashFixture{
		pool:     pool,
		wallet:   wallet.NewService(pool),
		customer: testdb.NewUser(t, pool, "customer"),
		driver:   testdb.NewUser(t, pool, "driver"),
		treasury: testdb.NewUser(t, pool, "admin"),
	}
	f.svc = orders.NewService(pool, nil, f.wallet, cashbox.NewService(pool, store), nil, quiet)
	f.svc.SetSettings(store)

	clearTreasury(t, pool)
	if _, err := pool.Exec(ctx, `UPDATE wallets SET is_treasury = true WHERE user_id = $1`, f.treasury); err != nil {
		t.Fatalf("تعذّر وسمُ الخزينة: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM wallet_transactions WHERE user_id = $1`, f.treasury)
		_, _ = pool.Exec(c, `DELETE FROM wallets WHERE user_id = $1`, f.treasury)
		_, _ = pool.Exec(c, `UPDATE wallets SET is_treasury = false WHERE user_id = $1`, f.treasury)
	})

	hid, err := wallet.EnsureCashHolding(ctx, pool)
	if err != nil {
		t.Fatalf("تعذّرت محفظةُ الاحتباس: %v", err)
	}
	f.holding = hid

	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&f.categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	return f
}

// merchant ينشئ متجراً بطريقةِ تسويةٍ ونسبةِ عمولةٍ ومالكٍ مخصَّص.
func (f *cashFixture) merchant(t *testing.T, method string, commissionPct int64) (merchantID, ownerID string) {
	t.Helper()
	ctx := context.Background()
	ownerID = testdb.NewUser(t, f.pool, "merchant")
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, owner_user_id, commission_percent, settlement_method, status)
		VALUES ('متجرُ اختبارِ النقد', $1, $2, $3, $4, 'active') RETURNING id::text`,
		f.categoryID, ownerID, commissionPct, method).Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر إنشاء متجر: %v", err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchantID) })
	return merchantID, ownerID
}

// order ينشئ طلباً نقديّاً عند «at_pickup» ببندٍ واحدٍ يحمل لقطةَ الطريقة.
// snapMethod لقطةُ order_items؛ فارغٌ ⇒ تُترَك NULL فتُشتقّ من إعداد المتجر.
func (f *cashFixture) order(t *testing.T, merchantID string, subtotal, deliveryFee int64, snapMethod string) string {
	t.Helper()
	ctx := context.Background()
	total := subtotal + deliveryFee
	var orderID string
	// **و`driver_fee = delivery_fee`** (هجرة 0164): طلبٌ بلا عرضٍ، أجرُ السائقِ
	// = أجرةُ التوصيل. ومن دونه لَقُرئ صفراً — يُدفَع اليوم من `driver_fee`.
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, driver_fee, total, wallet_paid, cash_due,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ($1, $2, $3, 'at_pickup', 'عنوان', ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', $4, $5, $5, $6, 0, $6, `+qaSnapSQLX()+`)
		RETURNING id::text`,
		f.customer, merchantID, f.driver, subtotal, deliveryFee, total).Scan(&orderID); err != nil {
		t.Fatalf("تعذّر إنشاء طلب: %v", err)
	}
	var snap any
	if snapMethod != "" {
		snap = snapMethod
	}
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO order_items (order_id, name, unit_price, merchant_price, merchant_id, qty, options, merchant_settlement_method)
		VALUES ($1, 'صنف', $2, $2, $3, 1, '[]'::jsonb, $4)`,
		orderID, subtotal, merchantID, snap); err != nil {
		t.Fatalf("تعذّر بندُ الطلب: %v", err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID) })
	return orderID
}

func (f *cashFixture) transition(t *testing.T, orderID, to string) {
	t.Helper()
	role, actor := "driver", f.driver
	if to == "refunded" {
		role, actor = "admin", f.treasury
	}
	if _, err := f.svc.Transition(context.Background(), actor, []string{role}, orderID, to, "اختبار"); err != nil {
		t.Fatalf("الانتقال إلى %s فشل: %v", to, err)
	}
}

func (f *cashFixture) pickup(t *testing.T, orderID string) { f.transition(t, orderID, "picked_up") }
func (f *cashFixture) deliver(t *testing.T, orderID string) {
	for _, st := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, orderID, st)
	}
}

func (f *cashFixture) balance(t *testing.T, userID string) int64 {
	t.Helper()
	var v int64
	if err := f.pool.QueryRow(context.Background(),
		`SELECT COALESCE((SELECT balance FROM wallets WHERE user_id = $1), 0)`, userID).Scan(&v); err != nil {
		t.Fatalf("رصيد: %v", err)
	}
	return v
}

func (f *cashFixture) sumKind(t *testing.T, orderID, kind string) int64 {
	t.Helper()
	var v int64
	if err := f.pool.QueryRow(context.Background(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions WHERE ref = $1 AND kind = $2`,
		orderID, kind).Scan(&v); err != nil {
		t.Fatalf("قيود %s: %v", kind, err)
	}
	return v
}

type settleRow struct {
	method   string
	amount   int64
	reversed int64
	state    string
	found    bool
}

func (f *cashFixture) settlement(t *testing.T, orderID, merchantID string) settleRow {
	t.Helper()
	var r settleRow
	err := f.pool.QueryRow(context.Background(), `
		SELECT method, amount, reversed_amount, state FROM merchant_settlements
		WHERE order_id = $1 AND merchant_id = $2`, orderID, merchantID).
		Scan(&r.method, &r.amount, &r.reversed, &r.state)
	if err == nil {
		r.found = true
	}
	return r
}

func (f *cashFixture) obligation(t *testing.T, merchantID string) int64 {
	t.Helper()
	v, err := obligations.Balance(context.Background(), f.pool, obligations.PartyMerchant, merchantID)
	if err != nil {
		t.Fatalf("التزام: %v", err)
	}
	return v
}

// mkDebt يزرع التزامَ متجرٍ سابقاً (بضاعةٌ رُدّت) — لبراهين الاقتطاع.
func (f *cashFixture) mkDebt(t *testing.T, merchantID string, amount int64) {
	t.Helper()
	if _, err := obligations.Create(context.Background(), f.pool, obligations.PartyMerchant,
		merchantID, amount, obligations.CauseReturnedGoods, "", &f.treasury); err != nil {
		t.Fatalf("تعذّر زرعُ الالتزام: %v", err)
	}
}

func (f *cashFixture) assertFIGreen(t *testing.T) {
	t.Helper()
	v, err := fininv.Run(context.Background(), f.pool,
		"FI-14.a", "FI-14.b", "FI-14.c", "FI-14.d", "FI-14.e", "FI-14.f", "FI-06.a")
	if err != nil {
		t.Fatalf("تعذّر تشغيلُ الثوابت: %v", err)
	}
	if len(v) != 0 {
		t.Fatalf("خُرقت ثوابتُ FI: %+v", v)
	}
}

// ── SET-12 · التسويةُ النقديّة: احتباسٌ لا محفظة ────────────────────────
func TestSET12_CashAccrual(t *testing.T) {
	f := newCashFixture(t)
	m, owner := f.merchant(t, "cash", 10)
	before := f.balance(t, f.holding)
	oid := f.order(t, m, 100_000, 10_000, "cash")
	f.pickup(t, oid)

	net := int64(90_000) // 100k − 10% عمولة
	if got := f.balance(t, owner); got != 0 {
		t.Fatalf("متجرٌ نقديٌّ قُيّد في محفظته %d — والنقديُّ لا يُقيَّد", got)
	}
	if got := f.sumKind(t, oid, "merchant_earning"); got != 0 {
		t.Fatalf("قيدُ مستحقٍّ محفظيّ %d لطلبٍ نقديّ", got)
	}
	if got := f.balance(t, f.holding) - before; got != net {
		t.Fatalf("الاحتباس زاد %d لا %d", got, net)
	}
	s := f.settlement(t, oid, m)
	if !s.found || s.method != "cash" || s.amount != net || s.reversed != 0 || s.state != "cash_due" {
		t.Fatalf("صفُّ التسوية: %+v — يُنتظر cash/%d/0/cash_due", s, net)
	}
	f.assertFIGreen(t)
}

// ── SET-14 · XOR بنيويّ: طريقةٌ واحدةٌ لكلّ مصدر ────────────────────────
func TestSET14_XORPerMethod(t *testing.T) {
	f := newCashFixture(t)
	mc, _ := f.merchant(t, "cash", 10)
	mw, ownerW := f.merchant(t, "wallet", 10)

	oc := f.order(t, mc, 50_000, 5_000, "cash")
	f.pickup(t, oc)
	ow := f.order(t, mw, 50_000, 5_000, "wallet")
	f.pickup(t, ow)

	if s := f.settlement(t, oc, mc); s.method != "cash" {
		t.Fatalf("النقديُّ طريقتُه %q", s.method)
	}
	if f.sumKind(t, oc, "merchant_earning") != 0 {
		t.Fatal("النقديُّ له قيدُ محفظة")
	}
	if s := f.settlement(t, ow, mw); s.method != "wallet" {
		t.Fatalf("المحفظيُّ طريقتُه %q", s.method)
	}
	if f.balance(t, ownerW) != 45_000 {
		t.Fatalf("المحفظيُّ لم يُقيَّد له %d", f.balance(t, ownerW))
	}
	if f.sumKind(t, ow, "merchant_cash_accrued") != 0 {
		t.Fatal("المحفظيُّ له قيدُ احتباس")
	}
	f.assertFIGreen(t)
}

// ── SET-15/16 · تأكيدُ الدفعِ نقداً — مرّةً ولا يتكرّر ─────────────────
func TestSET15_16_AdminPayIdempotent(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 100_000, 0, "cash")
	f.pickup(t, oid)
	sid := f.settlementID(t, oid, m)
	holdBefore := f.balance(t, f.holding)

	res, err := f.svc.MarkCashSettlementPaid(context.Background(), sid, f.treasury, "دُفع", "")
	if err != nil {
		t.Fatalf("تأكيدُ الدفع فشل: %v", err)
	}
	if res.AlreadyPaid || res.Amount != 90_000 {
		t.Fatalf("نتيجةُ الدفع: %+v", res)
	}
	if got := holdBefore - f.balance(t, f.holding); got != 90_000 {
		t.Fatalf("الاحتباس نقص %d لا 90000", got)
	}
	if s := f.settlement(t, oid, m); s.state != "cash_paid" {
		t.Fatalf("الحالة %q لا cash_paid", s.state)
	}
	// **إعادةُ التأكيد: لا قيدَ ولا نقصَ ثانٍ.**
	holdAfter := f.balance(t, f.holding)
	res2, err := f.svc.MarkCashSettlementPaid(context.Background(), sid, f.treasury, "ثانية", "")
	if err != nil {
		t.Fatalf("إعادةُ الدفع أخطأت: %v", err)
	}
	if !res2.AlreadyPaid {
		t.Fatal("إعادةُ التأكيد لم تُعَدّ مدفوعةً سلفاً")
	}
	if f.balance(t, f.holding) != holdAfter {
		t.Fatal("الاحتباس تحرّك في إعادةِ التأكيد")
	}
	f.assertFIGreen(t)
}

func (f *cashFixture) settlementID(t *testing.T, orderID, merchantID string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT id::text FROM merchant_settlements WHERE order_id = $1 AND merchant_id = $2`,
		orderID, merchantID).Scan(&id); err != nil {
		t.Fatalf("مُعرّفُ التسوية: %v", err)
	}
	return id
}

// ── SET-17 · استردادٌ قبل الدفع: عكسٌ لا التزام ────────────────────────
func TestSET17_CashRefundPrePay(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	holdStart := f.balance(t, f.holding)
	oid := f.order(t, m, 100_000, 10_000, "cash")
	f.deliver(t, oid)
	f.transition(t, oid, "refunded")

	if s := f.settlement(t, oid, m); s.state != "cash_reversed" || s.reversed != s.amount {
		t.Fatalf("بعد الاسترداد قبل الدفع: %+v — يُنتظر cash_reversed كاملاً", s)
	}
	if f.balance(t, f.holding) != holdStart {
		t.Fatalf("الاحتباس لم يعُد إلى أصله: %d ≠ %d", f.balance(t, f.holding), holdStart)
	}
	if f.obligation(t, m) != 0 {
		t.Fatalf("نشأ التزامٌ لاستردادٍ قبل الدفع: %d", f.obligation(t, m))
	}
	f.assertFIGreen(t)
}

// ── SET-18 · استردادٌ بعد الدفع: التزامٌ لا يُعاد كتابةُ الدفع ─────────
func TestSET18_CashRefundPostPay(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 100_000, 10_000, "cash")
	f.deliver(t, oid)
	sid := f.settlementID(t, oid, m)
	if _, err := f.svc.MarkCashSettlementPaid(context.Background(), sid, f.treasury, "دُفع", ""); err != nil {
		t.Fatalf("الدفع فشل: %v", err)
	}
	f.transition(t, oid, "refunded")

	if s := f.settlement(t, oid, m); s.state != "cash_paid" {
		t.Fatalf("الدفعُ أُعيدت كتابتُه — الحالة %q لا cash_paid", s.state)
	}
	if got := f.obligation(t, m); got != 90_000 {
		t.Fatalf("الالتزامُ بعد استردادِ مدفوعٍ = %d لا 90000", got)
	}
	f.assertFIGreen(t)
}

// ── SET-30 · اقتطاعُ الدَّينِ من مستحقٍّ نقديّ (كامل/جزئي/يفوق) ────────
func TestSET30_CashDebtOffset(t *testing.T) {
	cases := []struct {
		name              string
		debt, net         int64
		wantOutstanding   int64
		wantReversed      int64
		wantState         string
		wantRemainingDebt int64
	}{
		{"جزئي", 30_000, 90_000, 60_000, 30_000, "cash_due", 0},
		{"يفوق", 150_000, 90_000, 0, 90_000, "cash_reversed", 60_000},
		{"مساوٍ", 90_000, 90_000, 0, 90_000, "cash_reversed", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newCashFixture(t)
			m, _ := f.merchant(t, "cash", 10)
			f.mkDebt(t, m, c.debt)
			holdBefore := f.balance(t, f.holding)
			oid := f.order(t, m, 100_000, 0, "cash") // net = 90k
			f.pickup(t, oid)

			s := f.settlement(t, oid, m)
			if s.amount != c.net {
				t.Fatalf("المبلغُ المستحقُّ %d لا %d", s.amount, c.net)
			}
			if s.amount-s.reversed != c.wantOutstanding {
				t.Fatalf("القائمُ %d لا %d", s.amount-s.reversed, c.wantOutstanding)
			}
			if s.reversed != c.wantReversed || s.state != c.wantState {
				t.Fatalf("عكس=%d حالة=%s — يُنتظر %d/%s", s.reversed, s.state, c.wantReversed, c.wantState)
			}
			if got := f.balance(t, f.holding) - holdBefore; got != c.wantOutstanding {
				t.Fatalf("الاحتباس زاد %d لا %d (القائم)", got, c.wantOutstanding)
			}
			if got := f.obligation(t, m); got != c.wantRemainingDebt {
				t.Fatalf("الالتزامُ الباقي %d لا %d", got, c.wantRemainingDebt)
			}
			// **ولا مالٌ اختُلق**: لا قيدَ محفظةٍ للمتجر، الاقتطاعُ نصيبٌ أُنقص.
			if f.sumKind(t, oid, "merchant_earning") != 0 {
				t.Fatal("قيدُ محفظةٍ في مسارٍ نقديّ")
			}
			f.assertFIGreen(t)
		})
	}
}

// ── SET-28 · صافٍ صفريّ (عمولة ١٠٠٪): لا صفَّ ولا قيد ─────────────────
func TestSET28_ZeroNet(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 100) // 100% ⇒ due = 0
	holdBefore := f.balance(t, f.holding)
	oid := f.order(t, m, 100_000, 0, "cash")
	f.pickup(t, oid)

	if s := f.settlement(t, oid, m); s.found {
		t.Fatalf("صافٍ صفريٌّ أنشأ صفَّ تسوية: %+v", s)
	}
	if f.balance(t, f.holding) != holdBefore {
		t.Fatal("صافٍ صفريٌّ حرّك الاحتباس")
	}
	if f.sumKind(t, oid, "merchant_cash_accrued") != 0 {
		t.Fatal("صافٍ صفريٌّ قيّد احتباساً")
	}
	f.assertFIGreen(t)
}

// ── SET-26 · حارسُ التراث: طلبٌ سُوّي محفظةً قبل الترحيل ───────────────
func TestSET26_LegacyCutover(t *testing.T) {
	f := newCashFixture(t)
	m, owner := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 100_000, 0, "cash")
	// **نزرع قيدَ مستحقٍّ محفظيٍّ تاريخيّاً بلا صفِّ تسوية** (كالطلب قبل الترحيل).
	if _, err := f.wallet.ApplyTx(context.Background(), f.pool, owner, 90_000, "merchant_earning",
		oid, "تراثٌ قبل الترحيل", &f.treasury); err != nil {
		t.Fatalf("تعذّر زرعُ التراث: %v", err)
	}
	ownerBefore := f.balance(t, owner)
	holdBefore := f.balance(t, f.holding)

	f.deliver(t, oid) // settleMerchant يُنادى — يجب أن يراه مُسوّىً سلفاً

	if f.balance(t, owner) != ownerBefore {
		t.Fatalf("قُيّد للمتجر ثانيةً: %d ≠ %d", f.balance(t, owner), ownerBefore)
	}
	if f.balance(t, f.holding) != holdBefore {
		t.Fatal("نشأ احتباسٌ لطلبٍ تراثيّ")
	}
	if s := f.settlement(t, oid, m); s.found {
		t.Fatalf("نشأ صفُّ تسويةٍ لطلبٍ تراثيّ: %+v", s)
	}
}

// ── SET-04/05/06 · اللقطةُ للمستقبلِ فقط ──────────────────────────────
//
// **طلبان لمتجرٍ واحد**: الأوّلُ بلقطةِ محفظة، ثمّ يُبدَّل الإعدادُ نقداً،
// والثاني بلقطةِ نقد. **كلٌّ يُسوّى بلقطته لا بإعداد اليوم.**
func TestSET06_SnapshotIsFutureOnly(t *testing.T) {
	f := newCashFixture(t)
	m, ownerW := f.merchant(t, "wallet", 10)

	oldOrder := f.order(t, m, 100_000, 0, "wallet") // لقطةٌ محفظيّة
	// **يُبدَّل الإعدادُ إلى نقد** (كتغييرِ الأدمن).
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE merchants SET settlement_method = 'cash' WHERE id = $1`, m); err != nil {
		t.Fatalf("تعذّر تبديلُ الإعداد: %v", err)
	}
	newOrder := f.order(t, m, 100_000, 0, "cash") // لقطةٌ نقديّة

	f.pickup(t, oldOrder)
	f.pickup(t, newOrder)

	// **القديمُ محفظةً** — قُيّد للمالك.
	if f.balance(t, ownerW) != 90_000 {
		t.Fatalf("الطلبُ القديمُ لم يُسوَّ محفظةً: %d", f.balance(t, ownerW))
	}
	if s := f.settlement(t, oldOrder, m); s.method != "wallet" {
		t.Fatalf("القديمُ طريقتُه %q لا wallet", s.method)
	}
	// **الجديدُ نقداً** — احتباسٌ لا محفظة.
	if s := f.settlement(t, newOrder, m); s.method != "cash" {
		t.Fatalf("الجديدُ طريقتُه %q لا cash", s.method)
	}
	f.assertFIGreen(t)
}

// ── SET-24/party · الدفعُ للمالكِ الحاليّ لا القديم ───────────────────
func TestSET24_PayCurrentOwner(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 100_000, 0, "cash")
	f.pickup(t, oid)
	// **تتبدّل الملكيّةُ بعد النشأة قبل الدفع.**
	newOwner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE merchants SET owner_user_id = $2 WHERE id = $1`, m, newOwner); err != nil {
		t.Fatalf("تعذّر تبديلُ المالك: %v", err)
	}
	sid := f.settlementID(t, oid, m)
	res, err := f.svc.MarkCashSettlementPaid(context.Background(), sid, f.treasury, "دُفع", "")
	if err != nil {
		t.Fatalf("الدفع فشل: %v", err)
	}
	if res.OwnerUserID != newOwner {
		t.Fatalf("دُفع/أُشعر مالكٌ خطأ: %s ≠ %s (الحاليّ)", res.OwnerUserID, newOwner)
	}
	var paidOwner string
	_ = f.pool.QueryRow(context.Background(),
		`SELECT paid_owner_user_id::text FROM merchant_settlements WHERE id = $1`, sid).Scan(&paidOwner)
	if paidOwner != newOwner {
		t.Fatalf("سُجّل مالكُ الدفع %s لا الحاليّ %s", paidOwner, newOwner)
	}
	_ = time.Now
}
