package orders_test

// ══════════════════════════════════════════════════════════════════════
// **الكابتن يطلب تعديل ثمن البضاعة — والزبون يوافق أو يرفض** (قرار المالك ٢٠٢٦-١١-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// الموافقة تمرّ بمسار التأكيد القائم، والرفض يرجّع السعر القديم، والمهلة تنذر المكتب،
// **والثوابت المالية خضرا بعد كل خطوة وبعد التسليم** (`fininv`).

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// prFI **ما استجدّ من خروق الثوابت** بعد الصورة — يُسقط الاختبار بأوّلها.
func prFI(t *testing.T, pool *pgxpool.Pool, base fininv.Baseline, step string) {
	t.Helper()
	vs, err := fininv.Run(context.Background(), pool)
	if err != nil {
		t.Fatalf("%s: فحص الثوابت: %v", step, err)
	}
	for _, v := range base.New(vs) {
		t.Errorf("%s: خرق جديد %s", step, v.String())
	}
}

func prBase(t *testing.T, pool *pgxpool.Pool) fininv.Baseline {
	t.Helper()
	b, err := fininv.Capture(context.Background(), pool)
	if err != nil {
		t.Fatalf("صورة الثوابت: %v", err)
	}
	return b
}

// prDeposit **شحنٌ بحركة دفتر** — لا كتابةُ رصيدٍ عارية (`cqDeposit`) فيخرق `FI-02.a` من نفسه.
func prDeposit(t *testing.T, w *wallet.Service, user string, amount int64) {
	t.Helper()
	if _, err := w.Apply(context.Background(), user, amount, "topup", "", "شحن اختبار", nil); err != nil {
		t.Fatalf("الشحن: %v", err)
	}
}

func prCode(err error) string {
	var ae *httpx.AppError
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func prCashDue(t *testing.T, pool *pgxpool.Pool, orderID string) int64 {
	t.Helper()
	var due int64
	if err := pool.QueryRow(context.Background(),
		`SELECT cash_due FROM orders WHERE id::text = $1`, orderID).Scan(&due); err != nil {
		t.Fatal(err)
	}
	return due
}

func prStatus(t *testing.T, pool *pgxpool.Pool, orderID string) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT status FROM order_price_requests WHERE order_id::text = $1 ORDER BY created_at`, orderID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

// ── نقد: زيادة ← التسليم موقوف ← موافقة ← تسليم بالجديد ─────────────────
func TestDriverPriceRequest_CashRaiseApprove(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	base := prBase(t, pool)
	orderID := apPickedUp(t, svc, w, customer, driver, "cash", 1_000, 1_500)

	o, err := svc.RequestDriverPrice(ctx, orderID, driver, 1_400, "السعر بالمحل أغلى")
	if err != nil {
		t.Fatalf("طلب الكابتن: %v", err)
	}
	if !o.QuotePending() || o.Total != 2_900 {
		t.Fatalf("بعد الطلب: pending=%v total=%d — المنتظَر بانتظار و2900", o.QuotePending(), o.Total)
	}
	// **القديم ما بيتحرك قبل الموافقة.**
	if due := prCashDue(t, pool, orderID); due != 2_500 {
		t.Fatalf("المستحق قبل الموافقة=%d — المنتظَر 2500", due)
	}
	prFI(t, pool, base, "بعد الطلب")
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "delivered", ""); !errors.Is(err, orders.ErrQuoteReconfirmPending) {
		t.Fatalf("تسليم قبل الموافقة: %v — المنتظَر quote_reconfirm_pending", err)
	}
	// **طلب تاني والأول معلّق ممنوع.**
	if _, err := svc.RequestDriverPrice(ctx, orderID, driver, 1_300, "تاني"); !errors.Is(err, orders.ErrPriceRequestPending) {
		t.Fatalf("طلب تاني: %v — المنتظَر price_request_pending", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 2_900, o.QuoteVersion); err != nil {
		t.Fatalf("موافقة الزبون: %v", err)
	}
	if due := prCashDue(t, pool, orderID); due != 2_900 {
		t.Fatalf("المستحق بعد الموافقة=%d — المنتظَر 2900", due)
	}
	if st := prStatus(t, pool, orderID); len(st) != 1 || st[0] != "approved" {
		t.Fatalf("حالة الطلب=%v — المنتظَر approved", st)
	}
	prFI(t, pool, base, "بعد الموافقة")
	apDrive(t, svc, driver, orderID, "delivered")
	held, err := cashbox.NewService(pool, settings.NewStore(pool)).Held(ctx, driver)
	if err != nil {
		t.Fatal(err)
	}
	// **الصندوق = المجموع الجديد ناقص ثمن البضاعة الجديد** (٢٩٠٠ − ١٤٠٠).
	if held != 1_500 {
		t.Fatalf("صندوق الكابتن=%d — المنتظَر 1500", held)
	}
	prFI(t, pool, base, "بعد التسليم")
}

// ── محفظة: نقص ← ينتظر هو كمان ← موافقة ← الحجز بالجديد ← التسوية ──────
func TestDriverPriceRequest_WalletLowerApprove(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	base := prBase(t, pool)
	prDeposit(t, w, customer, 10_000)
	orderID := apPickedUp(t, svc, w, customer, driver, "wallet", 1_000, 1_500)

	o, err := svc.RequestDriverPrice(ctx, orderID, driver, 800, "خصم بالمحل")
	if err != nil {
		t.Fatalf("طلب الكابتن: %v", err)
	}
	// **النقص كمان بانتظار الزبون** — والحجز القديم باقٍ.
	if !o.QuotePending() {
		t.Fatal("النقص طلع بلا موافقة")
	}
	if st := cqRead(t, orderID); st.reserved != 2_500 {
		t.Fatalf("الحجز قبل الموافقة=%d — المنتظَر 2500", st.reserved)
	}
	prFI(t, pool, base, "بعد الطلب")
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 2_300, o.QuoteVersion); err != nil {
		t.Fatalf("موافقة الزبون: %v", err)
	}
	if st := cqRead(t, orderID); st.reserved != 2_300 {
		t.Fatalf("الحجز بعد الموافقة=%d — المنتظَر 2300", st.reserved)
	}
	prFI(t, pool, base, "بعد الموافقة")
	apDrive(t, svc, driver, orderID, "delivered")
	if b, _ := w.Balance(ctx, customer); b != 10_000-2_300 {
		t.Fatalf("رصيد الزبون=%d — المنتظَر 7700", b)
	}
	if r := cqReserved(t, w, customer); r != 0 {
		t.Fatalf("محجوز بعد التسوية=%d", r)
	}
	prFI(t, pool, base, "بعد التسليم")
}

// ── رفض: السعر القديم برجع، والتسليم بالقديم مسموح ─────────────────────
func TestDriverPriceRequest_RejectRevertsCash(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	base := prBase(t, pool)
	orderID := apPickedUp(t, svc, w, customer, driver, "cash", 1_000, 1_500)

	if _, err := svc.RequestDriverPrice(ctx, orderID, driver, 1_300, "السعر تغيّر"); err != nil {
		t.Fatalf("طلب الكابتن: %v", err)
	}
	o, err := svc.RejectQuote(ctx, orderID, customer)
	if err != nil {
		t.Fatalf("رفض الزبون: %v", err)
	}
	if o.QuotePending() || o.Total != 2_500 || o.CustomGoodsAmount == nil || *o.CustomGoodsAmount != 1_000 {
		t.Fatalf("بعد الرفض: pending=%v total=%d — المنتظَر القديم 2500 مؤكَّد", o.QuotePending(), o.Total)
	}
	if o.QuoteRejectedAt == nil {
		t.Fatal("علامة الرفض غابت — المكتب ما بيشوف السطر الأحمر")
	}
	if due := prCashDue(t, pool, orderID); due != 2_500 {
		t.Fatalf("المستحق بعد الرفض=%d — المنتظَر 2500", due)
	}
	if st := prStatus(t, pool, orderID); len(st) != 1 || st[0] != "rejected" {
		t.Fatalf("حالة الطلب=%v — المنتظَر rejected", st)
	}
	prFI(t, pool, base, "بعد الرفض")
	apDrive(t, svc, driver, orderID, "delivered")
	held, err := cashbox.NewService(pool, settings.NewStore(pool)).Held(ctx, driver)
	if err != nil {
		t.Fatal(err)
	}
	if held != 1_500 {
		t.Fatalf("صندوق الكابتن=%d — المنتظَر 1500 (القديم)", held)
	}
	prFI(t, pool, base, "بعد التسليم")
}

// ── رفض بالمحفظة: الحجز القديم كامل ────────────────────────────────────
func TestDriverPriceRequest_RejectRevertsWallet(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	base := prBase(t, pool)
	prDeposit(t, w, customer, 10_000)
	orderID := apPickedUp(t, svc, w, customer, driver, "wallet", 1_000, 1_500)

	if _, err := svc.RequestDriverPrice(ctx, orderID, driver, 1_200, "السعر تغيّر"); err != nil {
		t.Fatalf("طلب الكابتن: %v", err)
	}
	if _, err := svc.RejectQuote(ctx, orderID, customer); err != nil {
		t.Fatalf("رفض الزبون: %v", err)
	}
	if st := cqRead(t, orderID); st.reserved != 2_500 || st.total == nil || *st.total != 2_500 {
		t.Fatalf("بعد الرفض: reserved=%d total=%v — المنتظَر 2500", st.reserved, st.total)
	}
	prFI(t, pool, base, "بعد الرفض")
	apDrive(t, svc, driver, orderID, "delivered")
	if b, _ := w.Balance(ctx, customer); b != 10_000-2_500 {
		t.Fatalf("رصيد الزبون=%d — المنتظَر 7500", b)
	}
	prFI(t, pool, base, "بعد التسليم")
}

// ── الحدود والصلاحيات ─────────────────────────────────────────────────
func TestDriverPriceRequest_Limits(t *testing.T) {
	svc, w, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	stranger := testdb.NewUser(t, pool, "driver")

	// **قبل الاستلام ممنوع** — بابه «وثّق ثمن البضاعة».
	early := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, early, driver, 1_000, 1_500); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmQuote(ctx, early, customer, "cash", 2_500, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RequestDriverPrice(ctx, early, driver, 1_100, "قبل"); !errors.Is(err, orders.ErrPriceRequestStage) {
		t.Fatalf("قبل الاستلام: %v — المنتظَر price_request_stage", err)
	}

	orderID := apPickedUp(t, svc, w, customer, driver, "cash", 1_000, 1_500)
	if _, err := svc.RequestDriverPrice(ctx, orderID, stranger, 1_100, "غريب"); prCode(err) != "not_your_order" {
		t.Fatalf("كابتن غريب: %v — المنتظَر not_your_order", err)
	}
	if _, err := svc.RequestDriverPrice(ctx, orderID, driver, 1_100, "  "); !errors.Is(err, orders.ErrPriceRequestReason) {
		t.Fatalf("بلا سبب: %v", err)
	}
	if _, err := svc.RequestDriverPrice(ctx, orderID, driver, 1_000, "نفسه"); !errors.Is(err, orders.ErrPriceRequestSame) {
		t.Fatalf("نفس الثمن: %v", err)
	}
	// **السقف ٥٠٪ افتراضاً** — ١٦٠٠ من ١٠٠٠ فوقه.
	if _, err := svc.RequestDriverPrice(ctx, orderID, driver, 1_600, "كتير"); prCode(err) != "price_request_too_far" {
		t.Fatalf("فوق السقف: %v — المنتظَر price_request_too_far", err)
	}
	exec(ctx, `INSERT INTO app_settings (key, value) VALUES ('delivery.driver_price_edit_max_percent', '0'::jsonb)
	           ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM app_settings WHERE key = 'delivery.driver_price_edit_max_percent'`)
	})
	if _, err := svc.RequestDriverPrice(ctx, orderID, driver, 1_100, "مقفول"); !errors.Is(err, orders.ErrPriceRequestOff) {
		t.Fatalf("الإعداد صفر: %v — المنتظَر price_request_off", err)
	}
	exec(ctx, `UPDATE app_settings SET value = '50'::jsonb WHERE key = 'delivery.driver_price_edit_max_percent'`)
	if _, err := svc.RequestDriverPrice(ctx, orderID, driver, 1_500, "ع الحد"); err != nil {
		t.Fatalf("ع حد السقف بالضبط: %v — المنتظَر مقبول", err)
	}
}

// ── الأمانة ما إلها ثمن بضاعة ─────────────────────────────────────────
func TestDriverPriceRequest_AmanahRefused(t *testing.T) {
	svc, w, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	exec(ctx, `UPDATE orders SET custom_mode = 'amanah' WHERE id::text = $1`, orderID)
	if err := svc.AgreeCustom(ctx, orderID, driver, 0, 1_500); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 1_500, 1); err != nil {
		t.Fatal(err)
	}
	apDrive(t, svc, driver, orderID, "picked_up")
	if _, err := svc.RequestDriverPrice(ctx, orderID, driver, 100, "أمانة"); !errors.Is(err, orders.ErrPriceRequestStage) {
		t.Fatalf("أمانة: %v — المنتظَر price_request_stage", err)
	}
}

// ── المهلة: إنذار مرّة وحدة؛ والمكتب يحلّ محلّ الطلب ─────────────────────
func TestDriverPriceRequest_TimeoutAndOfficeSupersedes(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	admin := testdb.NewUser(t, pool, "admin")
	orderID := apPickedUp(t, svc, w, customer, driver, "cash", 1_000, 1_500)

	if _, err := svc.RequestDriverPrice(ctx, orderID, driver, 1_200, "السعر تغيّر"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE order_price_requests SET created_at = now() - interval '11 minutes'
		WHERE order_id::text = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	svc.SweepPriceRequestsOnce(ctx)
	var alerted int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM order_price_requests
		WHERE order_id::text = $1 AND alerted_at IS NOT NULL`, orderID).Scan(&alerted); err != nil {
		t.Fatal(err)
	}
	if alerted != 1 {
		t.Fatalf("بعد المهلة: منذَر=%d — المنتظَر 1", alerted)
	}
	if n := svc.SweepPriceRequestsOnce(ctx); n != 0 {
		t.Fatalf("الإنذار تكرّر: %d", n)
	}
	// **المكتب بيضل يعدّل** — وتعديله بيحلّ محلّ طلب الكابتن.
	if _, err := svc.AdminOverrideCustomQuote(ctx, orderID, admin, 1_100, 1_500, "اتصلنا بالزبون"); err != nil {
		t.Fatalf("تعديل المكتب: %v", err)
	}
	if st := prStatus(t, pool, orderID); len(st) != 1 || st[0] != "superseded" {
		t.Fatalf("حالة الطلب=%v — المنتظَر superseded", st)
	}
}

// ── الطلب انتهى والطلب معلّق ⇒ يتسكّر ولا إنذار ───────────────────────
func TestDriverPriceRequest_ClosedWithOrder(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	orderID := apPickedUp(t, svc, w, customer, driver, "cash", 1_000, 1_500)
	if _, err := svc.RequestDriverPrice(ctx, orderID, driver, 1_200, "السعر تغيّر"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE orders SET status = 'cancelled', closed_at = now() WHERE id::text = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE order_price_requests SET created_at = now() - interval '1 hour'
		WHERE order_id::text = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	if n := svc.SweepPriceRequestsOnce(ctx); n != 0 {
		t.Fatalf("إنذار لطلب منتهي: %d", n)
	}
	if st := prStatus(t, pool, orderID); len(st) != 1 || st[0] != "closed" {
		t.Fatalf("حالة الطلب=%v — المنتظَر closed", st)
	}
}
