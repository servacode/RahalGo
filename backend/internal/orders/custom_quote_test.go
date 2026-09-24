package orders_test

// ══════════════════════════════════════════════════════════════════════
// **عقدُ عرضِ السعر المخصَّص — سلوكاً حيّاً على قاعدةٍ حقيقيّة** — Batch 2a
// ══════════════════════════════════════════════════════════════════════
//
// **يُقاس ما يقع لا أنّه لم يسقط**: الحجزُ يُوضع ويُفكّ ويُسوّى، والنسخةُ
// تتزايد عند تغييرٍ حقيقيٍّ وحدَه، والتأكيدُ يُقفَل ويُبطَل، والمحفظةُ لا
// تُقبَل بلا رصيد، والاستلامُ لا يبدأ بلا تأكيد.

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// cqSetup يبني محرّكاً بمحفظةٍ وصندوقٍ وإعدادات، وزبوناً وسائقاً.
func cqSetup(t *testing.T) (*orders.Service, *wallet.Service, string, string, func(ctx context.Context, sql string, args ...any)) {
	t.Helper()
	pool := testdb.Pool(t)
	w := wallet.NewService(pool)
	cb := cashbox.NewService(pool, settings.NewStore(pool))
	svc := orders.NewService(pool, nil, w, cb, nil,
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	svc.SetSettings(settings.NewStore(pool))
	customer := testdb.NewUser(t, pool, "customer")
	driver := testdb.NewUser(t, pool, "driver")
	exec := func(ctx context.Context, sql string, args ...any) {
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("SQL: %v — %s", err, sql)
		}
	}
	return svc, w, customer, driver, exec
}

// cqSeedOrder يزرع طلباً مخصَّصاً مُسنَداً بلقطة سياسة أجرةٍ معطاة.
func cqSeedOrder(t *testing.T, w *wallet.Service, customer, driver, feeSource string, feeSnapshot *int64, mayChange bool) string {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, driver_id, kind, status, address_text, dropoff,
		                    payment_method, subtotal, delivery_fee, total,
		                    custom_fee_source, custom_fee_snapshot, custom_driver_may_change_fee,
		                    snap_merchant_commission_percent, snap_rep_commission_percent,
		                    snap_commission_source, snap_activation_orders)
		VALUES ($1, $2, 'custom', 'assigned', 'الرقة',
		        ST_SetSRID(ST_MakePoint(39.0094, 35.9506),4326)::geography, 'cash',
		        0, 0, 0, $3, $4, $5, `+qaSnapSQLX()+`)
		RETURNING id::text`, customer, driver, feeSource, feeSnapshot, mayChange).Scan(&id); err != nil {
		t.Fatalf("تعذّر زرعُ الطلب: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id::text = $1`, id)
	})
	return id
}

// cqDeposit يضع رصيداً في محفظة الزبون مباشرةً.
func cqDeposit(t *testing.T, customer string, amount int64) {
	t.Helper()
	pool := testdb.Pool(t)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO wallets (user_id, balance) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET balance = EXCLUDED.balance`, customer, amount); err != nil {
		t.Fatalf("تعذّر إيداعُ الرصيد: %v", err)
	}
}

type cqState struct {
	goods, fee, total     *int64
	quoteVersion          int64
	confirmedAt           *string
	confirmedTotal        *int64
	confirmedVersion      *int64
	reserved              int64
	paidAt                *string
	paymentMethod, status string
}

func cqRead(t *testing.T, orderID string) cqState {
	t.Helper()
	pool := testdb.Pool(t)
	var s cqState
	if err := pool.QueryRow(context.Background(), `
		SELECT custom_goods_amount, custom_fee, total, quote_version,
		       quote_confirmed_at::text, quote_confirmed_total, quote_confirmed_version,
		       custom_reserved_amount, custom_paid_at::text, payment_method, status
		FROM orders WHERE id::text = $1`, orderID).
		Scan(&s.goods, &s.fee, &s.total, &s.quoteVersion, &s.confirmedAt, &s.confirmedTotal,
			&s.confirmedVersion, &s.reserved, &s.paidAt, &s.paymentMethod, &s.status); err != nil {
		t.Fatalf("قراءةُ الطلب: %v", err)
	}
	return s
}

func cqReserved(t *testing.T, w *wallet.Service, user string) int64 {
	t.Helper()
	l, err := w.LayersOf(context.Background(), user)
	if err != nil {
		t.Fatalf("طبقاتُ المحفظة: %v", err)
	}
	return l.Reserved
}

// ── ١ ── المسارُ الكامل بالمحفظة: عرضٌ ← تأكيدٌ ← حجزٌ ← تسليمٌ ← تسوية ──
func TestCustomQuote_WalletHappyPath(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)

	if err := svc.AgreeCustom(ctx, orderID, driver, 6_000, 1_500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	st := cqRead(t, orderID)
	if st.quoteVersion != 1 || st.total == nil || *st.total != 7_500 {
		t.Fatalf("بعد الاتّفاق: version=%d total=%v — المنتظَر 1 و7500", st.quoteVersion, st.total)
	}

	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 7_500, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	st = cqRead(t, orderID)
	if st.confirmedAt == nil || st.reserved != 7_500 || st.paymentMethod != "wallet" {
		t.Fatalf("بعد التأكيد: confirmed=%v reserved=%d method=%s", st.confirmedAt, st.reserved, st.paymentMethod)
	}
	if r := cqReserved(t, w, customer); r != 7_500 {
		t.Fatalf("محجوزُ المحفظة=%d — المنتظَر 7500", r)
	}

	// **السَّوقُ إلى التسليم** — الاستلامُ مسموحٌ لأنّ التأكيدَ حاليٌّ.
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
	st = cqRead(t, orderID)
	if st.reserved != 0 || st.paidAt == nil {
		t.Fatalf("بعد التسليم: reserved=%d paid=%v — المنتظَر 0 ومدفوع", st.reserved, st.paidAt)
	}
	if b, _ := w.Balance(ctx, customer); b != 100_000-7_500 {
		t.Fatalf("رصيدُ الزبون=%d — المنتظَر %d", b, 100_000-7_500)
	}
	if b, _ := w.Balance(ctx, driver); b != 7_500 {
		t.Fatalf("رصيدُ السائق=%d — المنتظَر 7500", b)
	}
	if r := cqReserved(t, w, customer); r != 0 {
		t.Fatalf("محجوزُ المحفظة بعد التسوية=%d — المنتظَر 0", r)
	}
}

// ── ٢ ── الأجرةُ من المنصة تُفرَض ولا يغيّرها السائق ────────────────────
func TestCustomQuote_AdminDefinedFeeForced(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	snap := int64(2_000)
	orderID := cqSeedOrder(t, w, customer, driver, "admin_defined", &snap, false)

	// **السائقُ يرسل 9999 أجرةً — تُهمَل، وتُفرض 2000.**
	if err := svc.AgreeCustom(ctx, orderID, driver, 6_000, 9_999); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	st := cqRead(t, orderID)
	if st.fee == nil || *st.fee != 2_000 || st.total == nil || *st.total != 8_000 {
		t.Fatalf("الأجرة=%v الإجمالي=%v — المنتظَر 2000 و8000 (فُرضت لقطةُ المنصة)", st.fee, st.total)
	}
}

// ── ٣ ── تأكيدٌ لعرضٍ تبدّل يُردّ quote_changed ─────────────────────────
func TestCustomQuote_StaleRejected(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, orderID, driver, 6_000, 1_500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	// نسخةٌ خطأ
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 7_500, 99); !errors.Is(err, orders.ErrQuoteChanged) {
		t.Fatalf("نسخةٌ خطأ: المنتظَر ErrQuoteChanged، وجاء %v", err)
	}
	// مبلغٌ خطأ
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 9_999, 1); !errors.Is(err, orders.ErrQuoteChanged) {
		t.Fatalf("مبلغٌ خطأ: المنتظَر ErrQuoteChanged، وجاء %v", err)
	}
	if r := cqReserved(t, w, customer); r != 0 {
		t.Fatalf("لا حجزَ لعرضٍ لم يُؤكَّد — reserved=%d", r)
	}
}

// ── ٤ ── الزيادةُ بعد التأكيد تُبطله وتفكّ الحجز وتمنع الاستلام ──────────
func TestCustomQuote_IncreaseInvalidatesAndReleases(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, orderID, driver, 6_000, 1_500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 7_500, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	// **زيادة** — الإجماليُّ 9500 > 7500 المؤكَّد.
	if err := svc.AgreeCustom(ctx, orderID, driver, 8_000, 1_500); err != nil {
		t.Fatalf("الزيادة: %v", err)
	}
	st := cqRead(t, orderID)
	if st.confirmedAt != nil || st.reserved != 0 || st.quoteVersion != 2 {
		t.Fatalf("بعد الزيادة: confirmed=%v reserved=%d version=%d — المنتظَر مُبطَلٌ وصفرٌ ونسخة 2",
			st.confirmedAt, st.reserved, st.quoteVersion)
	}
	if r := cqReserved(t, w, customer); r != 0 {
		t.Fatalf("محجوزُ المحفظة بعد الزيادة=%d — المنتظَر 0", r)
	}
	// **والاستلامُ ممنوعٌ حتّى يُعاد التأكيد.**
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); !errors.Is(err, orders.ErrQuoteNotConfirmed) {
		t.Fatalf("الاستلامُ بلا تأكيد: المنتظَر ErrQuoteNotConfirmed، وجاء %v", err)
	}
}

// ── ٥ ── النقصُ بعد التأكيد يُبقيه ويُخفّض الحجز ────────────────────────
func TestCustomQuote_DecreaseKeepsConfirmation(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, orderID, driver, 6_000, 1_500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 7_500, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	// **نقص** — الإجماليُّ 6500 < 7500.
	if err := svc.AgreeCustom(ctx, orderID, driver, 5_000, 1_500); err != nil {
		t.Fatalf("النقص: %v", err)
	}
	st := cqRead(t, orderID)
	if st.confirmedAt == nil || st.reserved != 6_500 || st.confirmedTotal == nil || *st.confirmedTotal != 6_500 {
		t.Fatalf("بعد النقص: confirmed=%v reserved=%d confirmedTotal=%v — المنتظَر باقٍ و6500",
			st.confirmedAt, st.reserved, st.confirmedTotal)
	}
	if st.confirmedVersion == nil || *st.confirmedVersion != st.quoteVersion {
		t.Fatalf("نسخةُ التأكيد=%v والنسخة=%d — يجب أن تتطابقا بعد النقص", st.confirmedVersion, st.quoteVersion)
	}
	if r := cqReserved(t, w, customer); r != 6_500 {
		t.Fatalf("محجوزُ المحفظة بعد النقص=%d — المنتظَر 6500", r)
	}
	// **والاستلامُ مسموحٌ — التأكيدُ حاليٌّ (نسخته تطابق).**
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); err != nil {
		t.Fatalf("الاستلامُ بعد النقص: %v", err)
	}
}

// ── ٦ ── المحفظةُ لا تُقبَل بلا رصيدٍ كافٍ، ولا حجزَ عندها ───────────────
func TestCustomQuote_WalletInsufficient(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqDeposit(t, customer, 1_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, orderID, driver, 6_000, 1_500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 7_500, 1); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("رصيدٌ لا يكفي: المنتظَر ErrInsufficient، وجاء %v", err)
	}
	st := cqRead(t, orderID)
	if st.confirmedAt != nil || st.reserved != 0 {
		t.Fatalf("رُفض التأكيد فلا حجزَ ولا قفل — confirmed=%v reserved=%d", st.confirmedAt, st.reserved)
	}
	if r := cqReserved(t, w, customer); r != 0 {
		t.Fatalf("محجوزُ المحفظة=%d — المنتظَر 0", r)
	}
}

// ── ٧ ── الإلغاءُ يفكّ الحجز ────────────────────────────────────────────
func TestCustomQuote_CancelReleasesReservation(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, orderID, driver, 6_000, 1_500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 7_500, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	if r := cqReserved(t, w, customer); r != 7_500 {
		t.Fatalf("قبل الإلغاء المحجوز=%d", r)
	}
	if _, err := svc.Transition(ctx, driver, []string{"admin"}, orderID, "cancelled", "اختبار"); err != nil {
		t.Fatalf("الإلغاء: %v", err)
	}
	st := cqRead(t, orderID)
	if st.reserved != 0 {
		t.Fatalf("بعد الإلغاء حجزُ الطلب=%d — المنتظَر 0", st.reserved)
	}
	if r := cqReserved(t, w, customer); r != 0 {
		t.Fatalf("بعد الإلغاء محجوزُ المحفظة=%d — المنتظَر 0", r)
	}
}

// ── ٨ ── إعادةُ الاتّفاق بالقيم نفسِها لا تُزيد النسخة ───────────────────
func TestCustomQuote_NoOpAgreeNoVersionBump(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, orderID, driver, 6_000, 1_500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if svc.AgreeCustom(ctx, orderID, driver, 6_000, 1_500) != nil {
		t.Fatal("إعادةُ الاتّفاق بالقيم نفسِها يجب أن تنجح بلا أثر")
	}
	if st := cqRead(t, orderID); st.quoteVersion != 1 {
		t.Fatalf("النسخة=%d — المنتظَر 1 (لا تغييرَ حقيقيّ)", st.quoteVersion)
	}
}

// ── ٩ ── الاستلامُ ممنوعٌ بلا تأكيدٍ، مسموحٌ بعده ───────────────────────
func TestCustomQuote_PickupGate(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, orderID, driver, 6_000, 1_500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); !errors.Is(err, orders.ErrQuoteNotConfirmed) {
		t.Fatalf("الاستلامُ بلا تأكيد: المنتظَر ErrQuoteNotConfirmed، وجاء %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 7_500, 1); err != nil {
		t.Fatalf("التأكيد نقداً: %v", err)
	}
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); err != nil {
		t.Fatalf("الاستلامُ بعد التأكيد: %v", err)
	}
}

// ── ١٠ ── تأكيدان متزامنان: واحدٌ يحجز، ولا مضاعفة ──────────────────────
func TestCustomQuote_ConcurrentConfirmReservesOnce(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqDeposit(t, customer, 100_000)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, orderID, driver, 6_000, 1_500); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			_, errs[k] = svc.ConfirmQuote(ctx, orderID, customer, "wallet", 7_500, 1)
		}(i)
	}
	wg.Wait()
	for _, e := range errs {
		if e != nil {
			t.Fatalf("تأكيدٌ متزامنٌ سقط: %v", e)
		}
	}
	// **الحجزُ مرّةً واحدة** — القفلُ ثمّ التكرارُ اللطيف يمنعان المضاعفة.
	if r := cqReserved(t, w, customer); r != 7_500 {
		t.Fatalf("محجوزُ المحفظة=%d — المنتظَر 7500 مرّةً واحدة", r)
	}
	if st := cqRead(t, orderID); st.reserved != 7_500 {
		t.Fatalf("حجزُ الطلب=%d — المنتظَر 7500", st.reserved)
	}
}
