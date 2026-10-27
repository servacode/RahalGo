package orders_test

// ══════════════════════════════════════════════════════════════════════
// **طلبُ متجرٍ مدفوعٌ من محفظة الزبون — من يربح ومن يخسر في كلّ نهاية**
// ══════════════════════════════════════════════════════════════════════
//
// المثالُ الواحدُ لكلّ الحالات (ليرةٌ جديدة):
//
//	صنفٌ: شراؤه ١٠٠٠ وبيعُه ١٢٠٠ (هامشُ المنصّة ٢٠٠)
//	عمولةُ المتجر ١٠٪ من الشراء = ١٠٠ ⇒ صافي المتجر ٩٠٠
//	أجرةُ التوصيل ٥٠٠ · حصّةُ المنصّة منها ١٠٠ ⇒ أجرُ السائق ٤٠٠
//	المندوبُ ١٠٪ من (العمولة ١٠٠ + الهامش ٢٠٠) = ٣٠
//	الزبونُ يدفع ١٧٠٠ من محفظته عند الإنشاء
//
// **وكلُّ رقمٍ يُقاس من الدفتر بعد الانتقال الحقيقيّ** (`Transition`/`ResolveDoor`)،
// **وثوابتُ الحفظ `FI-06` و`FI-10` و`FI-14` تُصوَّر قبل وتُفحَص بعد** — فلا يُحسب
// على السيناريو خرقٌ خلّفه غيرُه في القاعدة المشتركة.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/pricing"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// woLine بندٌ في طلب المحفظة: سعرُ البيع وسعرُ الشراء والكمّيّة.
type woLine struct {
	sale, cost int64
	qty        int
}

// woFixture **عُدّةُ النقد نفسُها** (خزينةٌ واحتباسٌ وسائقٌ وزبون) **ومعها مندوبٌ ومتجر.**
type woFixture struct {
	*cashFixture
	rep      string
	merchant string
	owner    string
}

// newWO متجرٌ بطريقة تسويةٍ (`wallet` أو `cash`) وعمولةِ ١٠٪ ومنسوبٌ لمندوب.
func newWO(t *testing.T, method string) *woFixture {
	t.Helper()
	f := &woFixture{cashFixture: newCashFixture(t)}
	f.merchant, f.owner = f.cashFixture.merchant(t, method, 10)
	f.rep = testdb.NewUser(t, f.pool, "sales")
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE merchants SET sales_rep_user_id = $2 WHERE id = $1`, f.merchant, f.rep); err != nil {
		t.Fatalf("تعذّر ربطُ المندوب: %v", err)
	}
	return f
}

// order **طلبٌ مدفوعٌ كلُّه من المحفظة** عند «at_pickup» — والخصمُ قيدٌ حقيقيّ
// (`order_payment`) بعد شحنٍ بالمقدار نفسِه، كما يكتبه الإنشاء.
//
// **واللقطةُ مكتوبةٌ بيدها لا من إعدادات اللحظة**: نسبةُ المندوب ١٠ ومصدرُها
// الثابتُ والعتبةُ ١ — فلا يتبدّل الرقمُ بإعدادٍ تركه اختبارٌ آخر.
func (f *woFixture) order(t *testing.T, method string, deliveryFee, driverFee int64, lines ...woLine) (string, []string) {
	t.Helper()
	ctx := context.Background()
	var subtotal int64
	for _, l := range lines {
		subtotal += l.sale * int64(l.qty)
	}
	total := subtotal + deliveryFee
	var oid string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, driver_fee, total, wallet_paid, cash_due,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ($1, $2, $3, 'at_pickup', 'عنوانُ اختبار المحفظة',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'wallet', $4, $5, $6, $7, $7, 0, 10, 10, $8, 1)
		RETURNING id::text`,
		f.customer, f.merchant, f.driver, subtotal, deliveryFee, driverFee, total,
		string(pricing.RepCommissionSourceFixed)).Scan(&oid); err != nil {
		t.Fatalf("تعذّر إنشاءُ الطلب: %v", err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, oid) })
	ids := make([]string, 0, len(lines))
	for _, l := range lines {
		var id string
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO order_items (order_id, name, unit_price, merchant_price, merchant_id, qty,
			                         options, merchant_settlement_method)
			VALUES ($1, 'صنفُ محفظة', $2, $3, $4, $5, '[]'::jsonb, $6) RETURNING id::text`,
			oid, l.sale, l.cost, f.merchant, l.qty, method).Scan(&id); err != nil {
			t.Fatalf("تعذّر بندُ الطلب: %v", err)
		}
		ids = append(ids, id)
	}
	if _, err := f.wallet.Apply(ctx, f.customer, total, "topup", "seed-"+oid, "رصيدُ اختبار", nil); err != nil {
		t.Fatalf("الشحن: %v", err)
	}
	if _, err := f.wallet.Apply(ctx, f.customer, -total, "order_payment", oid, "", nil); err != nil {
		t.Fatalf("خصمُ الطلب: %v", err)
	}
	return oid, ids
}

// baseline صورةُ ما هو مخروقٌ قبل السيناريو.
func (f *woFixture) baseline(t *testing.T) fininv.Baseline {
	t.Helper()
	b, err := fininv.Capture(context.Background(), f.pool, "FI-06", "FI-10", "FI-14")
	if err != nil {
		t.Fatalf("تصويرُ الأساس: %v", err)
	}
	return b
}

// assertGreen **ما استجدّ من خرقٍ بعد السيناريو ذنبُه.**
func (f *woFixture) assertGreen(t *testing.T, base fininv.Baseline) {
	t.Helper()
	after, err := fininv.Run(context.Background(), f.pool, "FI-06", "FI-10", "FI-14")
	if err != nil {
		t.Fatalf("تشغيلُ الثوابت: %v", err)
	}
	for _, v := range base.New(after) {
		t.Errorf("خرقُ ثابتٍ ماليّ: %s", v)
	}
}

func (f *woFixture) platformCommission(t *testing.T, oid string) int64 {
	t.Helper()
	var v int64
	if err := f.pool.QueryRow(context.Background(),
		`SELECT platform_commission FROM orders WHERE id = $1`, oid).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func (f *woFixture) pendingCompensation(t *testing.T, oid string) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM driver_compensation_requests
		WHERE order_id = $1 AND status = 'pending'`, oid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// deliverFromPickup الاستلامُ ثمّ الطريقُ ثمّ الباب ثمّ التسليم — بيد السائق.
func (f *woFixture) deliverFromPickup(t *testing.T, oid string) {
	t.Helper()
	for _, st := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
}

// failOnTheWay **المكتبُ يُنهي الطلبَ بعد الاستلام** — بذنب المنصّة (فيُكتب طلبُ تعويضٍ معلَّق).
func (f *woFixture) failOnTheWay(t *testing.T, oid string) {
	t.Helper()
	f.transition(t, oid, "picked_up")
	f.transition(t, oid, "on_the_way")
	endAtDoor(t, f.svc, f.pool, oid, orders.FaultPlatform, "")
}

// ── ١ · التسليمُ: المتجرُ «محفظة» ──────────────────────────────────────

// TestWalletOrder_Delivered_WalletStore **كلُّ طرفٍ يقبض نصيبَه، وصندوقُ السائق لا يتحرّك.**
//
//	الزبون −١٧٠٠ · المتجر +٩٠٠ (عند الاستلام) · السائق +٤٠٠ محفظةً و٠ صندوقاً
//	المندوب +٣٠ · الخزينة +٣٧٠ (= عمولة ١٠٠ + هامش ٢٠٠ + حصّةُ توصيل ١٠٠ − مندوب ٣٠)
//	ولا سطرَ في درج المكتب.
func TestWalletOrder_Delivered_WalletStore(t *testing.T) {
	f := newWO(t, "wallet")
	base := f.baseline(t)
	heldBefore := f.held(t)
	oid, _ := f.order(t, "wallet", 500, 400, woLine{sale: 1200, cost: 1000, qty: 1})

	f.transition(t, oid, "picked_up")
	if got := f.balance(t, f.owner); got != 900 {
		t.Fatalf("المتجرُ قبض %d عند الاستلام — والمنتظَر 900", got)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != 800 {
		t.Fatalf("الخزينةُ بعد الاستلام %d — والمنتظَر 800 (1700 − 900)", got)
	}
	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}

	if got := f.balance(t, f.customer); got != 0 {
		t.Errorf("رصيدُ الزبون %d — دفع 1700 من محفظته", got)
	}
	if got := f.balance(t, f.owner); got != 900 {
		t.Errorf("محفظةُ المتجر %d لا 900", got)
	}
	if got := f.balance(t, f.driver); got != 400 {
		t.Errorf("أجرُ السائق %d لا 400", got)
	}
	if got := f.held(t) - heldBefore; got != 0 {
		t.Errorf("صندوقُ السائق تحرّك %d — والزبونُ لم يدفع نقداً", got)
	}
	if got := f.balance(t, f.rep); got != 30 {
		t.Errorf("عمولةُ المندوب %d لا 30", got)
	}
	if got := f.platformCommission(t, oid); got != 100 {
		t.Errorf("عمولةُ المنصّة %d لا 100", got)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != 370 {
		t.Errorf("ربحُ الخزينة %d لا 370", got)
	}
	if n, _ := f.drawer(t, "advance_return", oid); n != 0 {
		t.Errorf("سطرُ سلفةٍ في الدرج لمتجرٍ محفظيّ: %d", n)
	}
	f.assertGreen(t, base)
}

// ── ٢ · الاسترجاعُ بعد التسليم: المتجرُ «محفظة» ─────────────────────────

// TestWalletOrder_RefundAfterDelivery_WalletStore **الزبونُ يستعيد ١٧٠٠ إلى محفظته،
// ومستحقُّ المتجر يُعكَس، والسائقُ والمندوبُ يحتفظان** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// **والخزينةُ تخسر ٤٣٠ = أجرُ السائق ٤٠٠ + المندوب ٣٠.**
func TestWalletOrder_RefundAfterDelivery_WalletStore(t *testing.T) {
	f := newWO(t, "wallet")
	base := f.baseline(t)
	oid, _ := f.order(t, "wallet", 500, 400, woLine{sale: 1200, cost: 1000, qty: 1})
	f.deliverFromPickup(t, oid)
	f.transition(t, oid, "refunded")

	if got := f.balance(t, f.customer); got != 1700 {
		t.Errorf("رصيدُ الزبون بعد الاسترجاع %d — والمنتظَر 1700 كاملاً", got)
	}
	if got := f.sumKind(t, oid, "refund"); got != 1700 {
		t.Errorf("قيدُ الاسترجاع %d لا 1700", got)
	}
	if got := f.balance(t, f.owner); got != 0 {
		t.Errorf("بقي للمتجر %d عن طلبٍ مُسترجَع", got)
	}
	if got := f.sumKind(t, oid, "merchant_earning"); got != 0 {
		t.Errorf("صافي مستحقّ المتجر %d — والعكسُ في حسابه هو", got)
	}
	if got := f.balance(t, f.driver); got != 400 {
		t.Errorf("أجرُ السائق %d — والرحلةُ أُدّيت فيبقى 400", got)
	}
	if got := f.balance(t, f.rep); got != 30 {
		t.Errorf("عمولةُ المندوب %d — وتبقى له 30 (قرارُ المالك ٢٠٢٦-١٠-١٦)", got)
	}
	if got := f.platformCommission(t, oid); got != 0 {
		t.Errorf("عمولةُ المنصّة %d بعد الاسترجاع — تُصفَّر", got)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != -430 {
		t.Errorf("نصيبُ الخزينة %d — والمنتظَر −430 (أجرُ السائق + المندوب)", got)
	}
	if got := f.obligation(t, f.merchant); got != 0 {
		t.Errorf("نشأ دَينٌ على متجرٍ عُكس مستحقُّه من محفظته: %d", got)
	}
	f.assertGreen(t, base)
}

// ── ٣ · الاسترجاعُ بعد التسليم: المتجرُ «نقداً» ─────────────────────────

// TestWalletOrder_RefundAfterDelivery_CashStore **السائقُ دفع للمتجر ٩٠٠ من السلفة
// عند الاستلام، والمكتبُ ردّها من الدرج عند التسليم** (الزبونُ لم يدفع نقداً).
// **وبعد الاسترجاع يصير على المتجر دَينٌ بـ٩٠٠** — والخزينةُ تقرأ −١٣٣٠ على هذا
// الطلب (٩٠٠ منها دَينٌ يُستردّ من مستحقّاته القادمة).
func TestWalletOrder_RefundAfterDelivery_CashStore(t *testing.T) {
	f := newWO(t, "cash")
	base := f.baseline(t)
	heldBefore := f.held(t)
	oid, _ := f.order(t, "cash", 500, 400, woLine{sale: 1200, cost: 1000, qty: 1})
	f.deliverFromPickup(t, oid)

	if s := f.settlement(t, oid, f.merchant); s.state != "cash_paid" || s.amount != 900 {
		t.Fatalf("بعد التسليم: %+v — يُنتظر cash_paid بـ900", s)
	}
	if n, sum := f.drawer(t, "advance_return", oid); n != 1 || sum != 900 {
		t.Fatalf("ردُّ السلفة من الدرج: %d سطراً بـ%d — يُنتظر سطرٌ بـ900", n, sum)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != 370 {
		t.Fatalf("ربحُ الخزينة عند التسليم %d لا 370", got)
	}

	f.transition(t, oid, "refunded")

	if got := f.balance(t, f.customer); got != 1700 {
		t.Errorf("رصيدُ الزبون %d لا 1700", got)
	}
	if got := f.obligation(t, f.merchant); got != 900 {
		t.Errorf("دَينُ المتجر بعد استرجاع مدفوعٍ %d لا 900", got)
	}
	if s := f.settlement(t, oid, f.merchant); s.state != "cash_paid" {
		t.Errorf("الدفعُ أُعيدت كتابتُه: %q", s.state)
	}
	if got := f.balance(t, f.owner); got != 0 {
		t.Errorf("قُيّد في محفظة متجرٍ نقديّ %d", got)
	}
	if got := f.balance(t, f.driver); got != 400 {
		t.Errorf("أجرُ السائق %d لا 400", got)
	}
	if got := f.held(t) - heldBefore; got != 0 {
		t.Errorf("صندوقُ السائق تحرّك %d", got)
	}
	if got := f.balance(t, f.rep); got != 30 {
		t.Errorf("عمولةُ المندوب %d لا 30", got)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != -1330 {
		t.Errorf("نصيبُ الخزينة %d — والمنتظَر −1330 (900 دَينٌ على المتجر + 400 + 30)", got)
	}
	if n, sum := f.drawer(t, "advance_return", oid); n != 1 || sum != 900 {
		t.Errorf("الدرجُ بعد الاسترجاع: %d سطراً بـ%d — لا يتغيّر", n, sum)
	}
	f.assertGreen(t, base)
}

// ── ٤ · تعذّرٌ بعد الاستلام: المتجرُ «محفظة» ───────────────────────────

// TestWalletOrder_FailedAfterPickup_WalletStore **الزبونُ يستعيد ١٧٠٠، والمتجرُ يبقى
// له ٩٠٠ (باع وسلّم)، والخزينةُ تخسر ٩٠٠** — ولا أجرَ للسائق ولا عمولةَ للمندوب،
// **وطلبُ تعويضٍ معلَّقٌ للسائق** بذنب المنصّة (لا قيدَ حتّى توافق العمليّات).
func TestWalletOrder_FailedAfterPickup_WalletStore(t *testing.T) {
	f := newWO(t, "wallet")
	base := f.baseline(t)
	oid, _ := f.order(t, "wallet", 500, 400, woLine{sale: 1200, cost: 1000, qty: 1})
	f.failOnTheWay(t, oid)

	if got := f.balance(t, f.customer); got != 1700 {
		t.Errorf("رصيدُ الزبون بعد التعذّر %d لا 1700", got)
	}
	if got := f.balance(t, f.owner); got != 900 {
		t.Errorf("المتجرُ %d — ومستحقُّه ثبت عند الاستلام 900", got)
	}
	if got := f.balance(t, f.driver); got != 0 {
		t.Errorf("قُيّد للسائق %d قبل موافقة العمليّات", got)
	}
	if got := f.pendingCompensation(t, oid); got != 1 {
		t.Errorf("طلباتُ التعويض المعلَّقة %d لا 1", got)
	}
	if got := f.balance(t, f.rep); got != 0 {
		t.Errorf("عمولةُ مندوبٍ %d على طلبٍ لم يصل", got)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != -900 {
		t.Errorf("نصيبُ الخزينة %d — والمنتظَر −900 (البضاعةُ التي دفعتها)", got)
	}
	if n, _ := f.drawer(t, "advance_return", oid); n != 0 {
		t.Errorf("سطرُ سلفةٍ لمتجرٍ محفظيّ: %d", n)
	}
	f.assertGreen(t, base)
}

// ── ٥ · تعذّرٌ بعد الاستلام: المتجرُ «نقداً» ───────────────────────────

// TestWalletOrder_FailedAfterPickup_CashStore **السائقُ دفع ٩٠٠ من السلفة ولم يقبض شيئاً
// ⇒ المكتبُ يردّها من الدرج** — والزبونُ يستعيد ١٧٠٠ والخزينةُ تخسر ٩٠٠.
func TestWalletOrder_FailedAfterPickup_CashStore(t *testing.T) {
	f := newWO(t, "cash")
	base := f.baseline(t)
	heldBefore := f.held(t)
	oid, _ := f.order(t, "cash", 500, 400, woLine{sale: 1200, cost: 1000, qty: 1})
	f.failOnTheWay(t, oid)

	if got := f.balance(t, f.customer); got != 1700 {
		t.Errorf("رصيدُ الزبون %d لا 1700", got)
	}
	if s := f.settlement(t, oid, f.merchant); s.state != "cash_paid" || s.amount != 900 {
		t.Errorf("تسويةُ المتجر: %+v — يُنتظر cash_paid بـ900", s)
	}
	if n, sum := f.drawer(t, "advance_return", oid); n != 1 || sum != 900 {
		t.Errorf("ردُّ السلفة: %d سطراً بـ%d — يُنتظر سطرٌ بـ900", n, sum)
	}
	if got := f.held(t) - heldBefore; got != 0 {
		t.Errorf("صندوقُ السائق تحرّك %d", got)
	}
	if got := f.obligation(t, f.merchant); got != 0 {
		t.Errorf("دَينٌ على المتجر %d — وبضاعتُه خرجت فحقُّه ثبت", got)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != -900 {
		t.Errorf("نصيبُ الخزينة %d لا −900", got)
	}
	f.assertGreen(t, base)
}

// ── ٦ · صنفٌ غيرُ متوفّرٍ (ردٌّ جزئيٌّ للمحفظة) ثمّ تسليم ──────────────

// TestWalletOrder_ItemUnavailableThenDelivered_TreasuryConserves **ردٌّ جزئيٌّ قبل الاستلام
// لا يُحسب على الخزينة مرّتين.**
//
//	بندان: ٢× (١٢٠٠/١٠٠٠) و١× (٦٠٠/٥٠٠) · توصيلٌ ٥٠٠ (للسائق ٤٠٠) ⇒ دفع ٣٥٠٠
//	يُحذف الثاني ⇒ المجموعُ ٢٩٠٠ ويُردّ ٦٠٠ للمحفظة (wallet_paid يصير ٢٩٠٠)
//	المتجر ٢٠٠٠ − ٢٠٠ = ١٨٠٠ · السائق ٤٠٠ · المندوب ١٠٪ × (٢٠٠ + ٤٠٠) = ٦٠
//	الخزينةُ الصحيحة = ٢٩٠٠ − ١٨٠٠ − ٤٠٠ − ٦٠ = ٦٤٠
//
// **و`creditTreasury` تأخذ `wallet_paid` (بعد التخفيض) ثمّ تطرح قيدَ الردّ الجزئيّ
// مرّةً ثانية** — فتُقيّد ٤٠ لا ٦٤٠ ويختلّ `FI-06.a` بـ٦٠٠. **هذا الاختبارُ يكشف ذلك.**
func TestWalletOrder_ItemUnavailableThenDelivered_TreasuryConserves(t *testing.T) {
	// **كان عطباً حقيقيّاً في المحرّك** (٢٠٢٦-١٠-٢٧): `MarkItemUnavailable` يكتب قيدَ
	// `refund` بالفرق **ويُنزل `wallet_paid` معه**، و`creditTreasury` كانت تأخذ
	// `wallet_paid` ثمّ تطرح كلَّ `refund` — **فيُطرح الردُّ مرّتين.** وصارت تقرأ المدفوعَ
	// من قيود `order_payment` (قرارُ المالك ٢٠٢٦-١٠-٢٧).
	f := newWO(t, "wallet")
	ctx := context.Background()
	base := f.baseline(t)
	oid, lines := f.order(t, "wallet", 500, 400,
		woLine{sale: 1200, cost: 1000, qty: 2}, woLine{sale: 600, cost: 500, qty: 1})

	if _, _, err := f.svc.MarkItemUnavailable(ctx, f.customer, oid, lines[1], 0, nil); err != nil {
		t.Fatalf("حذفُ الصنف: %v", err)
	}
	var walletPaid, driverFee int64
	if err := f.pool.QueryRow(ctx, `SELECT wallet_paid, driver_fee FROM orders WHERE id = $1`, oid).
		Scan(&walletPaid, &driverFee); err != nil {
		t.Fatal(err)
	}
	if walletPaid != 2900 || driverFee != 400 {
		t.Fatalf("بعد الحذف: المدفوع %d وأجرُ السائق %d — يُنتظر 2900 و400", walletPaid, driverFee)
	}
	if got := f.balance(t, f.customer); got != 600 {
		t.Fatalf("ردُّ المحفظة الجزئيّ %d لا 600", got)
	}

	f.deliverFromPickup(t, oid)

	if got := f.balance(t, f.owner); got != 1800 {
		t.Errorf("مستحقُّ المتجر %d لا 1800", got)
	}
	if got := f.balance(t, f.driver); got != 400 {
		t.Errorf("أجرُ السائق %d لا 400", got)
	}
	if got := f.balance(t, f.rep); got != 60 {
		t.Errorf("عمولةُ المندوب %d لا 60", got)
	}
	// **ما دفعه الزبونُ فعلاً = خصمُه ناقصَ ما رُدّ له** — من الدفتر لا من العمود.
	var netPaid int64
	if err := f.pool.QueryRow(ctx, `
		SELECT -COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND user_id = $2 AND kind IN ('order_payment', 'refund')`,
		oid, f.customer).Scan(&netPaid); err != nil {
		t.Fatal(err)
	}
	if netPaid != 2900 {
		t.Fatalf("صافي ما دفعه الزبون %d لا 2900", netPaid)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != 640 {
		t.Errorf("ربحُ الخزينة %d — والصحيحُ 640 (2900 − 1800 − 400 − 60)؛ "+
			"والفرقُ %d هو الردُّ الجزئيّ محسوباً مرّتين", got, 640-got)
	}
	f.assertGreen(t, base)
}
