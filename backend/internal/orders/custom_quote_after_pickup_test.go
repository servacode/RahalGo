package orders_test

// ══════════════════════════════════════════════════════════════════════
// **تعديلُ سعر الخاصّ بعد الاستلام** — قرارُ المالك ٢٠٢٦-١٠-١٥
// ══════════════════════════════════════════════════════════════════════
//
// **الحادثة**: وثّق السائقُ ثمنَ البضاعة ٥٠ بدل ٥٠٠ ثمّ استلم، **فلم يملك المكتبُ
// تصحيحَه** — الزيادةُ بعد الاستلام كانت تُردّ `custom_locked`.
//
// **والقاعدةُ الآن**: النقصُ يقع فوراً، **والزيادةُ تنتظر تأكيدَ الزبون والتسليمُ
// موقوفٌ حتّى يؤكّد** — ثمّ يُسوّى بالمبلغ الجديد.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// apDrive يسوق الطلبَ بيد السائق عبر الحالات المعطاة.
func apDrive(t *testing.T, svc *orders.Service, driver, orderID string, steps ...string) {
	t.Helper()
	for _, to := range steps {
		if _, err := svc.Transition(context.Background(), driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
}

// apPickedUp **طلبٌ خاصٌّ وُثّق ثمنُه وأُكّد ثمّ استُلم ووصل باب الزبون.**
func apPickedUp(t *testing.T, svc *orders.Service, w *wallet.Service, customer, driver,
	method string, goods, fee int64) string {
	t.Helper()
	ctx := context.Background()
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, orderID, driver, goods, fee); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, method, goods+fee, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	apDrive(t, svc, driver, orderID, "picked_up", "on_the_way", "at_dropoff")
	return orderID
}

// ── الحادثةُ نفسُها نقداً: ٥٠ بدل ٥٠٠ بعد الاستلام ─────────────────────
func TestCustomQuoteAfterPickup_IncidentCash(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	admin := testdb.NewUser(t, pool, "admin")
	orderID := apPickedUp(t, svc, w, customer, driver, "cash", 50, 1_500)

	// **المكتبُ يرفعه إلى ٥٠٠** — كان يُردّ `custom_locked`.
	if _, err := svc.AdminOverrideCustomQuote(ctx, orderID, admin, 500, 1_500, "ثمنٌ وُثّق خطأً"); err != nil {
		t.Fatalf("تعديلُ المكتب بعد الاستلام: %v — المنتظَر أن يُقبل", err)
	}
	st := cqRead(t, orderID)
	if st.quoteVersion != 2 || st.confirmedVersion == nil || *st.confirmedVersion != 1 ||
		st.total == nil || *st.total != 2_000 {
		t.Fatalf("بعد التعديل: version=%d confirmed=%v total=%v — المنتظَر 2 و1 و2000",
			st.quoteVersion, st.confirmedVersion, st.total)
	}
	var o *orders.Order
	o, err := svc.GetByID(ctx, orderID)
	if err != nil {
		t.Fatal(err)
	}
	if !o.QuotePending() {
		t.Fatal("السعرُ الجديد لا يظهر بانتظار التأكيد")
	}

	// **والتسليمُ موقوفٌ حتّى يؤكّد الزبون.**
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "delivered", ""); !errors.Is(err, orders.ErrQuoteReconfirmPending) {
		t.Fatalf("التسليمُ قبل إعادة التأكيد: %v — المنتظَر quote_reconfirm_pending", err)
	}

	// **والزبونُ يؤكّد الجديدَ بعد الاستلام** — كان يُردّ `custom_locked`.
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 2_000, 2); err != nil {
		t.Fatalf("إعادةُ التأكيد: %v", err)
	}
	var cashDue int64
	if err := pool.QueryRow(ctx, `SELECT cash_due FROM orders WHERE id::text = $1`, orderID).Scan(&cashDue); err != nil {
		t.Fatal(err)
	}
	if cashDue != 2_000 {
		t.Fatalf("المستحقُّ النقديّ=%d — المنتظَر 2000", cashDue)
	}

	apDrive(t, svc, driver, orderID, "delivered")
	held, err := cashbox.NewService(pool, settings.NewStore(pool)).Held(ctx, driver)
	if err != nil {
		t.Fatal(err)
	}
	// **والصندوقُ يقبض ما يدين به للمكتب** (قرارُ المالك ٢٠٢٦-١٠-١٥) — المبلغُ الجديد ناقصَ
	// ثمن البضاعة الذي دفعه للمتجر: ٢٠٠٠ − ٥٠٠.
	if held != 1_500 {
		t.Fatalf("صندوقُ السائق=%d — المنتظَر 1500 (الأجرةُ الجديدةُ وحدَها)", held)
	}
	// **وأجرتُه بعد حصّة المنصّة ١٠٪** — ١٣٥٠.
	if b, _ := w.Balance(ctx, driver); b != 1_350 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 1350", b)
	}
}

// ── الحادثةُ بالمحفظة: يُحجَز الفرقُ عند إعادة التأكيد ───────────────────
func TestCustomQuoteAfterPickup_IncidentWallet(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	admin := testdb.NewUser(t, pool, "admin")
	cqDeposit(t, customer, 10_000)
	orderID := apPickedUp(t, svc, w, customer, driver, "wallet", 50, 1_500)

	if _, err := svc.AdminOverrideCustomQuote(ctx, orderID, admin, 500, 1_500, "ثمنٌ وُثّق خطأً"); err != nil {
		t.Fatalf("تعديلُ المكتب: %v", err)
	}
	// **والمؤكَّدُ القديمُ قائمٌ بمحجوزه** — لا يُفكّ والبضاعةُ في الطريق.
	if st := cqRead(t, orderID); st.reserved != 1_550 {
		t.Fatalf("محجوزُ الطلب بعد الزيادة=%d — المنتظَر 1550 (القديمُ باقٍ)", st.reserved)
	}
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "delivered", ""); !errors.Is(err, orders.ErrQuoteReconfirmPending) {
		t.Fatalf("التسليمُ قبل إعادة التأكيد: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 2_000, 2); err != nil {
		t.Fatalf("إعادةُ التأكيد: %v", err)
	}
	if st := cqRead(t, orderID); st.reserved != 2_000 {
		t.Fatalf("محجوزُ الطلب بعد إعادة التأكيد=%d — المنتظَر 2000", st.reserved)
	}
	apDrive(t, svc, driver, orderID, "delivered")
	if b, _ := w.Balance(ctx, customer); b != 10_000-2_000 {
		t.Fatalf("رصيدُ الزبون=%d — المنتظَر 8000", b)
	}
	if r := cqReserved(t, w, customer); r != 0 {
		t.Fatalf("محجوزُ المحفظة بعد التسوية=%d — المنتظَر 0", r)
	}
	if b, _ := w.Balance(ctx, driver); b != 1_350 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 1350", b)
	}
}

// ── رصيدٌ لا يكفي الجديد: يُردّ ويبقى القديمُ محجوزاً ──────────────────
func TestCustomQuoteAfterPickup_WalletShort(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	admin := testdb.NewUser(t, testdb.Pool(t), "admin")
	cqDeposit(t, customer, 1_600)
	orderID := apPickedUp(t, svc, w, customer, driver, "wallet", 50, 1_500)

	if _, err := svc.AdminOverrideCustomQuote(ctx, orderID, admin, 500, 1_500, "ثمنٌ وُثّق خطأً"); err != nil {
		t.Fatalf("تعديلُ المكتب: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 2_000, 2); !errors.Is(err, wallet.ErrInsufficient) {
		t.Fatalf("تأكيدٌ برصيدٍ لا يكفي: %v — المنتظَر رصيدٌ غيرُ كافٍ", err)
	}
	if st := cqRead(t, orderID); st.reserved != 1_550 || st.confirmedVersion == nil || *st.confirmedVersion != 1 {
		t.Fatalf("بعد الردّ: reserved=%d confirmed=%v — المنتظَر 1550 و1", st.reserved, st.confirmedVersion)
	}
}

// ── النقصُ بعد الاستلام يقع فوراً ────────────────────────────────────
func TestCustomQuoteAfterPickup_DecreaseImmediate(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	pool := testdb.Pool(t)
	admin := testdb.NewUser(t, pool, "admin")
	orderID := apPickedUp(t, svc, w, customer, driver, "cash", 500, 1_500)

	if _, err := svc.AdminOverrideCustomQuote(ctx, orderID, admin, 50, 1_500, "ثمنٌ أقلّ في الفاتورة"); err != nil {
		t.Fatalf("النقصُ بعد الاستلام: %v", err)
	}
	st := cqRead(t, orderID)
	if st.confirmedVersion == nil || *st.confirmedVersion != st.quoteVersion ||
		st.total == nil || *st.total != 1_550 {
		t.Fatalf("بعد النقص: version=%d confirmed=%v total=%v — المنتظَر تأكيداً حاليّاً و1550",
			st.quoteVersion, st.confirmedVersion, st.total)
	}
	var cashDue int64
	if err := pool.QueryRow(ctx, `SELECT cash_due FROM orders WHERE id::text = $1`, orderID).Scan(&cashDue); err != nil {
		t.Fatal(err)
	}
	if cashDue != 1_550 {
		t.Fatalf("المستحقُّ النقديّ=%d — المنتظَر 1550 (يتبع النقص)", cashDue)
	}
	// **بلا إعادة تأكيد** — يُسلَّم في الحال ويُقبض الجديد.
	apDrive(t, svc, driver, orderID, "delivered")
	held, err := cashbox.NewService(pool, settings.NewStore(pool)).Held(ctx, driver)
	if err != nil {
		t.Fatal(err)
	}
	// **ناقصَ ما دفعه للمتجر** (قرارُ المالك ٢٠٢٦-١٠-١٥): ١٥٥٠ − ٥٠.
	if held != 1_500 {
		t.Fatalf("صندوقُ السائق=%d — المنتظَر 1500", held)
	}
}

// ── الرفضُ علامةٌ للمكتب، والسائقُ لا يعدّل بعد الاستلام ─────────────────
func TestCustomQuoteAfterPickup_RejectFlagAndDriverLocked(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	admin := testdb.NewUser(t, testdb.Pool(t), "admin")
	orderID := apPickedUp(t, svc, w, customer, driver, "cash", 50, 1_500)

	// **والسائقُ ما زال لا يعدّل بعد الاستلام.**
	if err := svc.AgreeCustom(ctx, orderID, driver, 500, 1_500); !errors.Is(err, orders.ErrCustomLocked) {
		t.Fatalf("تعديلُ السائق بعد الاستلام: %v — المنتظَر custom_locked", err)
	}
	// **ولا رفضَ ولا شيءَ ينتظر.**
	if _, err := svc.RejectQuote(ctx, orderID, customer); !errors.Is(err, orders.ErrQuoteNothingPending) {
		t.Fatalf("رفضٌ بلا سعرٍ جديد: %v", err)
	}
	if _, err := svc.AdminOverrideCustomQuote(ctx, orderID, admin, 500, 1_500, "ثمنٌ وُثّق خطأً"); err != nil {
		t.Fatalf("تعديلُ المكتب: %v", err)
	}
	o, err := svc.RejectQuote(ctx, orderID, customer)
	if err != nil {
		t.Fatalf("رفضُ الزبون: %v", err)
	}
	if o.QuoteRejectedAt == nil || o.Status != "at_dropoff" {
		t.Fatalf("بعد الرفض: rejected=%v status=%s — المنتظَر علامةً والطلبُ في مكانه",
			o.QuoteRejectedAt, o.Status)
	}
	// **وتعديلٌ جديدٌ من المكتب يمحو العلامة.**
	o, err = svc.AdminOverrideCustomQuote(ctx, orderID, admin, 300, 1_500, "اتّفاقٌ هاتفيّ")
	if err != nil {
		t.Fatalf("التعديلُ الثاني: %v", err)
	}
	if o.QuoteRejectedAt != nil {
		t.Fatal("علامةُ الرفض باقيةٌ بعد تعديلٍ جديد")
	}
}
