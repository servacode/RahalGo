package orders_test

// ══════════════════════════════════════════════════════════════════════
// **السائقُ يدفع للمتجر عند الاستلام** — قرارُ المالك ٢٠٢٦-١٠-١٥
// ══════════════════════════════════════════════════════════════════════
//
// المثالُ المُقرّ: بضاعةٌ ٥٠٠ · عمولةُ المتجر ١٠٪ ⇒ صافيه ٤٥٠ · توصيلٌ ١٠٠ للسائق
// كاملاً. **الزبونُ يدفع ٦٠٠ نقداً، والسائقُ دفع للمتجر ٤٥٠ عند الاستلام** (من
// سلفةِ مدير المكتب، خارجَ النظام) — **فما يدين به للمكتب ١٥٠** (عمولةٌ ٥٠ +
// توصيلٌ ١٠٠). **ولا يبقى للمتجر شيء**، وربحُ الخزينة كما كان.
//
// (الأرقامُ هنا مضروبةٌ في مئة: ٥٠٬٠٠٠ و١٠٬٠٠٠.)

import (
	"context"
	"fmt"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func (f *cashFixture) held(t *testing.T) int64 {
	t.Helper()
	v, err := cashbox.NewService(f.pool, settings.NewStore(f.pool)).Held(context.Background(), f.driver)
	if err != nil {
		t.Fatalf("الصندوق: %v", err)
	}
	return v
}

// paidVia **من أين دُفع المستحقّ** — وفارغٌ إن لم يُدفع.
func (f *cashFixture) paidVia(t *testing.T, orderID, merchantID string) string {
	t.Helper()
	var v *string
	if err := f.pool.QueryRow(context.Background(), `
		SELECT paid_via FROM merchant_settlements WHERE order_id = $1 AND merchant_id = $2`,
		orderID, merchantID).Scan(&v); err != nil {
		t.Fatalf("paid_via: %v", err)
	}
	if v == nil {
		return ""
	}
	return *v
}

// drawer **سطورُ درج المكتب لمصدرٍ ومرجع** — العددُ والمجموع.
func (f *cashFixture) drawer(t *testing.T, source, ref string) (int, int64) {
	t.Helper()
	var n int
	var sum int64
	if err := f.pool.QueryRow(context.Background(), `
		SELECT count(*), COALESCE(sum(amount), 0) FROM office_cash_entries
		 WHERE source = $1 AND ref = $2 AND direction = 'out'`, source, ref).Scan(&n, &sum); err != nil {
		t.Fatalf("الدرج: %v", err)
	}
	return n, sum
}

// orderWalletPaid **طلبٌ دفعه الزبونُ كلَّه من محفظته** — والمتجرُ «نقداً».
func (f *cashFixture) orderWalletPaid(t *testing.T, merchantID string, subtotal, deliveryFee int64) string {
	t.Helper()
	ctx := context.Background()
	oid := f.order(t, merchantID, subtotal, deliveryFee, "cash")
	total := subtotal + deliveryFee
	if _, err := f.pool.Exec(ctx, `
		UPDATE orders SET payment_method = 'wallet', wallet_paid = total, cash_due = 0 WHERE id = $1`,
		oid); err != nil {
		t.Fatalf("تعذّر جعلُ الطلب محفظيّاً: %v", err)
	}
	if _, err := f.wallet.Apply(ctx, f.customer, total, "topup", "seed-"+oid, "رصيدُ اختبار", nil); err != nil {
		t.Fatalf("الشحن: %v", err)
	}
	if _, err := f.wallet.Apply(ctx, f.customer, -total, "order_payment", oid, "", nil); err != nil {
		t.Fatalf("خصمُ الطلب: %v", err)
	}
	return oid
}

func (f *cashFixture) assertDriverPayGreen(t *testing.T) {
	t.Helper()
	v, err := fininv.Run(context.Background(), f.pool,
		"FI-14.a", "FI-14.b", "FI-14.c", "FI-14.d", "FI-14.e", "FI-14.f",
		"FI-06.a", "FI-10.b", "FI-10.c", "FI-15.b", "FI-15.e")
	if err != nil {
		t.Fatalf("تعذّر تشغيلُ الثوابت: %v", err)
	}
	if len(v) != 0 {
		t.Fatalf("خُرقت ثوابتُ FI: %+v", v)
	}
}

// TestDRVPAY_CashOrderCashStore **الطلبُ النقديُّ والمتجرُ «نقداً»**: عند الاستلام
// المستحقُّ مدفوعٌ من السائق ولا قائمَ له · وعند التسليم صندوقُ السائق = نقدُ الطلب
// ناقصَ ما دفعه · وأجرُ السائق وربحُ الخزينة كما كانا · ولا سطرَ في الدرج.
func TestDRVPAY_CashOrderCashStore(t *testing.T) {
	f := newCashFixture(t)
	m, owner := f.merchant(t, "cash", 10)
	holdBefore := f.balance(t, f.holding)
	heldBefore := f.held(t)
	oid := f.order(t, m, 50_000, 10_000, "cash") // cash_due = 60_000
	net := int64(45_000)

	f.pickup(t, oid)

	s := f.settlement(t, oid, m)
	if s.state != "cash_paid" || s.amount != net {
		t.Fatalf("بعد الاستلام: %+v — يُنتظر cash_paid بـ%d", s, net)
	}
	if via := f.paidVia(t, oid, m); via != "driver" {
		t.Fatalf("دُفع من %q لا من السائق", via)
	}
	if got := f.balance(t, f.holding) - holdBefore; got != 0 {
		t.Fatalf("**بقي في الاحتباس %d** — والمتجرُ قبض من السائق", got)
	}
	if got := f.sumKind(t, oid, "merchant_cash_paid"); got != -net {
		t.Fatalf("قيدُ الدفع %d لا %d", got, -net)
	}
	if got := f.balance(t, owner); got != 0 {
		t.Fatalf("قُيّد للمتجر في محفظته %d", got)
	}

	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}

	if got := f.held(t) - heldBefore; got != 60_000-net {
		t.Fatalf("**صندوقُ السائق زاد %d لا %d** — العمولةُ والتوصيلُ وحدَهما", got, 60_000-net)
	}
	if got := f.sumKind(t, oid, "driver_earning"); got != 10_000 {
		t.Fatalf("أجرُ السائق %d لا 10000", got)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != 5_000 {
		t.Fatalf("ربحُ الخزينة %d لا 5000 (العمولة)", got)
	}
	sid := f.settlementID(t, oid, m)
	if n, _ := f.drawer(t, "merchant_cash_paid", sid); n != 0 {
		t.Fatalf("**سطرُ دفعٍ في الدرج** والنقدُ لم يمرّ به: %d", n)
	}
	if n, _ := f.drawer(t, "advance_return", oid); n != 0 {
		t.Fatalf("سلفةٌ رجعت والزبونُ دفع نقداً يغطّيها: %d", n)
	}
	f.assertDriverPayGreen(t)
}

// TestDRVPAY_WalletCustomerCashStore **الزبونُ دفع من محفظته والمتجرُ «نقداً»**:
// السائقُ دفع للمتجر من السلفة ولم يقبض شيئاً — **فالمكتبُ يردّ السلفةَ من الدرج**
// (`advance_return`) بمقدار الصافي، وصندوقُ السائق لا يتحرّك، والربحُ كما كان.
func TestDRVPAY_WalletCustomerCashStore(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	heldBefore := f.held(t)
	oid := f.orderWalletPaid(t, m, 50_000, 10_000)
	net := int64(45_000)

	f.deliver(t, oid)

	if s := f.settlement(t, oid, m); s.state != "cash_paid" {
		t.Fatalf("الحالة %q لا cash_paid", s.state)
	}
	if got := f.held(t) - heldBefore; got != 0 {
		t.Fatalf("صندوقُ السائق تحرّك %d — ولم يقبض شيئاً", got)
	}
	n, sum := f.drawer(t, "advance_return", oid)
	if n != 1 || sum != net {
		t.Fatalf("**ردُّ السلفة**: %d سطراً بمجموع %d — يُنتظر سطرٌ بـ%d", n, sum, net)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != 5_000 {
		t.Fatalf("ربحُ الخزينة %d لا 5000 — السلفةُ ليست ربحاً ولا خسارة", got)
	}
	f.assertDriverPayGreen(t)
}

// TestDRVPAY_WalletMerchantUnchanged **المتجرُ «محفظة» كما كان**: مستحقُّه في
// محفظته، والسائقُ لا يدفع له شيئاً، وصندوقُه يقبض نقدَ الطلب كلَّه.
func TestDRVPAY_WalletMerchantUnchanged(t *testing.T) {
	f := newCashFixture(t)
	m, owner := f.merchant(t, "wallet", 10)
	heldBefore := f.held(t)
	oid := f.order(t, m, 50_000, 10_000, "wallet")

	f.deliver(t, oid)

	if s := f.settlement(t, oid, m); s.method != "wallet" || s.state != "wallet_credited" {
		t.Fatalf("المحفظيُّ: %+v", s)
	}
	if got := f.balance(t, owner); got != 45_000 {
		t.Fatalf("محفظةُ المتجر %d لا 45000", got)
	}
	if got := f.held(t) - heldBefore; got != 60_000 {
		t.Fatalf("صندوقُ السائق %d لا 60000", got)
	}
	if got := f.sumKind(t, oid, "merchant_cash_paid"); got != 0 {
		t.Fatalf("دفعٌ نقديٌّ لمتجرٍ محفظيّ: %d", got)
	}
	if n, _ := f.drawer(t, "advance_return", oid); n != 0 {
		t.Fatal("سلفةٌ لمتجرٍ محفظيّ")
	}
	f.assertDriverPayGreen(t)
}

// ── FI-06.d · طلبٌ حُوسب بلا ربحٍ ليس خرقاً ─────────────────────────────

func hitsOrder(vs []fininv.Violation, number int64) bool {
	want := fmt.Sprint(number)
	for _, v := range vs {
		for _, r := range v.Rows {
			if len(r) > 0 && fmt.Sprint(r[0]) == want {
				return true
			}
		}
	}
	return false
}

// TestFI06d_ZeroProfitDeliveredCustomIsAccounted **مشترياتٌ سُلّمت بلا ربحٍ للمنصّة**
// لا قيدَ `platform_profit` لها (`creditTreasury` لا تكتب صفراً) — **وهي محاسَبةٌ لا
// «قبل المحاسبة».** وطلبٌ قبل الاستلام بقيدٍ غريبٍ يبقى خرقاً.
func TestFI06d_ZeroProfitDeliveredCustomIsAccounted(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	driver := testdb.NewUser(t, f.pool, "driver")

	var custom string
	var customNo int64
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, driver_id, kind, custom_mode, custom_request, status,
			address_text, dropoff, payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due,
			custom_goods_amount, custom_fee)
		VALUES ($1, $2, 'custom', 'purchase', 'خبز', 'delivered', 'عنوان',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'wallet', 0, 0, 0, 1000, 0, 1000, 0)
		RETURNING id::text, number`, f.customer, driver).Scan(&custom, &customNo); err != nil {
		t.Fatalf("طلبٌ خاصّ: %v", err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, custom) })
	if _, err := f.wallet.Apply(ctx, f.customer, 1000, "topup", "seed-"+custom, "رصيد", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := f.wallet.Apply(ctx, f.customer, -1000, "order_payment", custom, "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := f.wallet.Apply(ctx, driver, 1000, "custom_goods_return", custom, "", nil); err != nil {
		t.Fatal(err)
	}

	m, _ := f.merchant(t, "cash", 10)
	early := f.order(t, m, 10_000, 1_000, "cash")
	var earlyNo int64
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'accepted' WHERE id = $1`, early); err != nil {
		t.Fatal(err)
	}
	_ = f.pool.QueryRow(ctx, `SELECT number FROM orders WHERE id = $1`, early).Scan(&earlyNo)
	if _, err := f.wallet.Apply(ctx, driver, 500, "driver_earning", early, "قيدٌ قبل أوانه", nil); err != nil {
		t.Fatal(err)
	}

	vs, err := fininv.Run(ctx, f.pool, "FI-06.d")
	if err != nil {
		t.Fatal(err)
	}
	if hitsOrder(vs, customNo) {
		t.Fatalf("**مشترياتٌ سُلّمت بلا ربحٍ قُرئت خرقاً**: %v", vs)
	}
	if !hitsOrder(vs, earlyNo) {
		t.Fatalf("**طلبٌ قبل الاستلام بقيد أجرٍ لم يُكشف**: %v", vs)
	}
}
