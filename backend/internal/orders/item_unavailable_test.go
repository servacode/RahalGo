package orders_test

// ══════════════════════════════════════════════════════════════════════
// **صنفٌ غيرُ متوفّر — يُحذف ويُعاد حسابُ الطلب** — قرارُ المالك ٢٠٢٦-١٠-١٥
// ══════════════════════════════════════════════════════════════════════
//
// **يُقاس ما يقع**: البنودُ والمجموعُ والمستحقُّ النقديّ وردُّ المحفظة وسقوطُ
// الكود ورسمُ المصدر والحدثُ والإشعار، **ومستحقُّ المتجر الذي يتبع البنود.**

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// iuFixture طلبٌ بسطرين: «أ» ×٢ (١٠٠٠ بيعاً / ٩٠٠ شراءً) و«ب» ×١ (٥٠٠ / ٤٥٠)،
// وتوصيلٌ بمئة — **مجموعُه ٢٦٠٠.**
type iuFixture struct {
	*fixture
	lineA, lineB string
}

func iuSetup(t *testing.T, status, method string) *iuFixture {
	t.Helper()
	walletPaid := int64(0)
	if method == "wallet" {
		walletPaid = 2_600
	}
	f := &iuFixture{fixture: setup(t, status, 2_500, 100, walletPaid)}
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `DELETE FROM order_items WHERE order_id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO order_items (order_id, name, unit_price, merchant_price, qty, options)
		VALUES ($1, 'أ', 1000, 900, 2, '[]'::jsonb) RETURNING id::text`, f.orderID).Scan(&f.lineA); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO order_items (order_id, name, unit_price, merchant_price, qty, options)
		VALUES ($1, 'ب', 500, 450, 1, '[]'::jsonb) RETURNING id::text`, f.orderID).Scan(&f.lineB); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET payment_method = $2 WHERE id = $1`,
		f.orderID, method); err != nil {
		t.Fatal(err)
	}
	return f
}

type iuState struct {
	subtotal, deliveryFee, discount, total, walletPaid, cashDue, driverFee, sourcesFee int64
	promo                                                                              *string
	merchantID                                                                         string
}

func (f *iuFixture) read(t *testing.T) iuState {
	t.Helper()
	var s iuState
	if err := f.pool.QueryRow(context.Background(), `
		SELECT subtotal, delivery_fee, discount, total, wallet_paid, cash_due, driver_fee,
		       sources_fee, promo_code, merchant_id::text
		FROM orders WHERE id = $1`, f.orderID).
		Scan(&s.subtotal, &s.deliveryFee, &s.discount, &s.total, &s.walletPaid, &s.cashDue,
			&s.driverFee, &s.sourcesFee, &s.promo, &s.merchantID); err != nil {
		t.Fatal(err)
	}
	return s
}

// ── نقداً: بعضُ الكمّيّة، ثمّ الحدثُ والإشعار ─────────────────────────
func TestItemUnavailable_CashPartialQty(t *testing.T) {
	f := iuSetup(t, "accepted", "cash")
	ctx := context.Background()
	o, res, err := f.svc.MarkItemUnavailable(ctx, f.customer, f.orderID, f.lineA, 1, nil)
	if err != nil {
		t.Fatalf("الحذف: %v", err)
	}
	if res.RemovedQty != 1 || res.OldTotal != 2_600 || res.NewTotal != 1_600 {
		t.Fatalf("النتيجة: %+v — المنتظَر حذفَ واحدٍ و2600←1600", res)
	}
	st := f.read(t)
	if st.subtotal != 1_500 || st.total != 1_600 || st.cashDue != 1_600 || st.deliveryFee != 100 {
		t.Fatalf("بعد الحذف: %+v — المنتظَر بضاعة 1500 ومجموعاً ونقداً 1600 وتوصيلاً 100", st)
	}
	if o.Total != 1_600 {
		t.Fatalf("الطلبُ المردود مجموعُه %d", o.Total)
	}
	var qty int
	if err := f.pool.QueryRow(ctx, `SELECT qty FROM order_items WHERE id = $1`, f.lineA).Scan(&qty); err != nil || qty != 1 {
		t.Fatalf("كمّيّةُ السطر=%d (%v) — المنتظَر 1", qty, err)
	}
	var events, notes int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM order_events WHERE order_id = $1
		   AND from_status = 'accepted' AND to_status = 'accepted' AND note LIKE '%غير متوفر%'`,
		f.orderID).Scan(&events); err != nil || events != 1 {
		t.Fatalf("حدثُ الطلب=%d (%v) — المنتظَر 1", events, err)
	}
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM notifications WHERE user_id = $1 AND entity_id = $2
		   AND body LIKE '%أ غير متوفر%' AND body LIKE '%1,600%'`,
		f.customer, f.orderID).Scan(&notes); err != nil {
		t.Fatal(err)
	}
	if notes != 1 {
		var body string
		_ = f.pool.QueryRow(ctx, `SELECT COALESCE(max(body),'') FROM notifications WHERE entity_id = $1`, f.orderID).Scan(&body)
		t.Fatalf("إشعارُ الزبون=%d — نصُّه: %q", notes, body)
	}
}

// ── بالمحفظة: يُردّ الفرقُ فوراً بقيد استرجاعٍ مرجعُه الطلب ───────────────
func TestItemUnavailable_WalletRefund(t *testing.T) {
	f := iuSetup(t, "preparing", "wallet")
	ctx := context.Background()
	if _, _, err := f.svc.MarkItemUnavailable(ctx, f.customer, f.orderID, f.lineB, 0, nil); err != nil {
		t.Fatalf("الحذف: %v", err)
	}
	st := f.read(t)
	if st.total != 2_100 || st.walletPaid != 2_100 || st.cashDue != 0 {
		t.Fatalf("بعد الحذف: %+v — المنتظَر 2100 من المحفظة ولا نقد", st)
	}
	var refund int64
	if err := f.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE user_id = $1 AND ref = $2 AND kind = 'refund'`, f.customer, f.orderID).Scan(&refund); err != nil {
		t.Fatal(err)
	}
	if refund != 500 {
		t.Fatalf("الاسترجاع=%d — المنتظَر 500", refund)
	}
	if b := f.balance(t, f.customer); b != 500 {
		t.Fatalf("رصيدُ الزبون=%d — المنتظَر 500", b)
	}
}

// ── الحدود: الصنفُ الأخير، وبعد الاستلام ─────────────────────────────
func TestItemUnavailable_Refusals(t *testing.T) {
	f := iuSetup(t, "at_pickup", "cash")
	ctx := context.Background()
	if _, _, err := f.svc.MarkItemUnavailable(ctx, f.customer, f.orderID, f.lineB, 0, nil); err != nil {
		t.Fatalf("حذفُ «ب»: %v", err)
	}
	if _, _, err := f.svc.MarkItemUnavailable(ctx, f.customer, f.orderID, f.lineA, 0, nil); !errors.Is(err, orders.ErrItemUnavailableLast) {
		t.Fatalf("حذفُ الأخير: %v — المنتظَر item_unavailable_last_item", err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'picked_up' WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.svc.MarkItemUnavailable(ctx, f.customer, f.orderID, f.lineA, 1, nil); !errors.Is(err, orders.ErrItemUnavailableTooLate) {
		t.Fatalf("بعد الاستلام: %v — المنتظَر item_unavailable_too_late", err)
	}
}

// iuPromo يزرع كوداً مستعمَلاً على الطلب — **استعمالٌ واحدٌ وعدّادٌ بواحد.**
func iuPromo(t *testing.T, f *iuFixture, kind string, value, minOrder int64, maxDiscount *int64,
	discount int64) string {
	t.Helper()
	ctx := context.Background()
	code := "IU" + f.orderID[:8]
	var promoID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO promo_codes (code, kind, value, min_order, max_discount, active, used_count)
		VALUES ($1, $2, $3, $4, $5, true, 1) RETURNING id::text`,
		code, kind, value, minOrder, maxDiscount).Scan(&promoID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM promo_redemptions WHERE promo_id = $1`, promoID)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM promo_codes WHERE id = $1`, promoID)
	})
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO promo_redemptions (promo_id, order_id, user_id) VALUES ($1, $2, $3)`,
		promoID, f.orderID, f.customer); err != nil {
		t.Fatal(err)
	}
	total := 2_600 - discount
	if _, err := f.pool.Exec(ctx, `
		UPDATE orders SET promo_code = $2, discount = $3, total = $4::bigint,
		       cash_due = CASE WHEN payment_method = 'cash' THEN $4::bigint ELSE 0 END,
		       wallet_paid = CASE WHEN payment_method = 'wallet' THEN $4::bigint ELSE 0 END
		WHERE id = $1`, f.orderID, code, discount, total); err != nil {
		t.Fatal(err)
	}
	return promoID
}

// iuPromoUntouched **الاستعمالُ وعدّادُه كما هما.**
func iuPromoUntouched(t *testing.T, f *iuFixture, promoID string) {
	t.Helper()
	var used, redemptions int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT used_count FROM promo_codes WHERE id = $1`, promoID).Scan(&used); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM promo_redemptions WHERE order_id = $1`, f.orderID).Scan(&redemptions); err != nil {
		t.Fatal(err)
	}
	if used != 1 || redemptions != 1 {
		t.Fatalf("مُسّ استعمالُ الكود: used=%d redemptions=%d — المنتظَر 1 و1", used, redemptions)
	}
}

// ── الكودُ لا يسقط ولو صار الطلبُ دون حدّه — والمجموعُ لا يرتفع أبداً ──────
//
// (قرارُ المالك ٢٠٢٦-١٠-١٥: «العيبُ عيبُ المتجر — الزبونُ لا يدفع أكثرَ بحذف صنف».)
func TestItemUnavailable_PromoKept(t *testing.T) {
	f := iuSetup(t, "accepted", "cash")
	ctx := context.Background()
	// **كودٌ ثابتٌ بثلاثمئة، حدُّه الأدنى ألفان** — المجموعُ ٢٣٠٠.
	promoID := iuPromo(t, f, "fixed", 300, 2_000, nil, 300)
	prev := int64(2_300)
	steps := []struct {
		name               string
		line               string
		qty                int
		wantDisc, wantTotl int64
	}{
		// **«ب» يُحذف والبضاعةُ ٢٠٠٠** — الخصمُ كما هو.
		{"«ب»", f.lineB, 0, 300, 1_800},
		// **ثمّ واحدٌ من «أ» — البضاعةُ ١٠٠٠ دون الحدّ: الكودُ باقٍ.**
		{"واحدٌ من «أ»", f.lineA, 1, 300, 800},
	}
	for _, sp := range steps {
		if _, _, err := f.svc.MarkItemUnavailable(ctx, f.customer, f.orderID, sp.line, sp.qty, nil); err != nil {
			t.Fatalf("حذفُ %s: %v", sp.name, err)
		}
		st := f.read(t)
		if st.discount != sp.wantDisc || st.total != sp.wantTotl || st.promo == nil {
			t.Fatalf("بعد %s: %+v — المنتظَر خصماً %d ومجموعاً %d والكودُ باقٍ",
				sp.name, st, sp.wantDisc, sp.wantTotl)
		}
		if st.total > prev {
			t.Fatalf("بعد %s ارتفع المجموع %d ← %d", sp.name, prev, st.total)
		}
		prev = st.total
	}
	iuPromoUntouched(t, f, promoID)
}

// ── كودٌ أكبرُ من البضاعة الباقية يُقصَر عليها · ونسبةٌ تُعاد بسقفها ──────────
func TestItemUnavailable_PromoCapped(t *testing.T) {
	t.Run("ثابتٌ فوق البضاعة", func(t *testing.T) {
		f := iuSetup(t, "accepted", "wallet")
		ctx := context.Background()
		// **ألفٌ وثمانمئة خصماً على ٢٥٠٠** — المجموعُ ٨٠٠ من المحفظة.
		promoID := iuPromo(t, f, "fixed", 1_800, 0, nil, 1_800)
		if _, _, err := f.svc.MarkItemUnavailable(ctx, f.customer, f.orderID, f.lineA, 0, nil); err != nil {
			t.Fatalf("الحذف: %v", err)
		}
		// **البضاعةُ ٥٠٠ والخصمُ يُقصَر عليها** — المجموعُ التوصيلُ وحدَه، ١٠٠.
		st := f.read(t)
		if st.discount != 500 || st.total != 100 || st.walletPaid != 100 || st.promo == nil {
			t.Fatalf("بعد الحذف: %+v — المنتظَر خصماً 500 ومجموعاً 100 والكودُ باقٍ", st)
		}
		// **ويُردّ الفرقُ للمحفظة — لا يُخصم منها شيء.**
		if b := f.balance(t, f.customer); b != 700 {
			t.Fatalf("رصيدُ الزبون=%d — المنتظَر 700 (800 − 100)", b)
		}
		iuPromoUntouched(t, f, promoID)
	})
	t.Run("نسبةٌ بسقف", func(t *testing.T) {
		f := iuSetup(t, "accepted", "cash")
		ctx := context.Background()
		cap := int64(200)
		// **١٠٪ بسقف ٢٠٠، وحدٌّ أدنى ٢٤٠٠** — على ٢٥٠٠ = ٢٠٠ (السقف).
		promoID := iuPromo(t, f, "percent", 10, 2_400, &cap, 200)
		if _, _, err := f.svc.MarkItemUnavailable(ctx, f.customer, f.orderID, f.lineA, 1, nil); err != nil {
			t.Fatalf("الحذف: %v", err)
		}
		// **١٥٠٠ ⇒ ١٠٪ = ١٥٠** تحت السقف — والحدُّ الأدنى لا يُسأل عنه.
		if st := f.read(t); st.discount != 150 || st.total != 1_450 || st.promo == nil {
			t.Fatalf("بعد الحذف: %+v — المنتظَر خصماً 150 ومجموعاً 1450", st)
		}
		iuPromoUntouched(t, f, promoID)
	})
}

// ── مستحقُّ المتجر وما يدفعه السائق يتبعان البنودَ عند الاستلام ──────────
func TestItemUnavailable_MerchantShareFollows(t *testing.T) {
	f := iuSetup(t, "at_pickup", "cash")
	ctx := context.Background()
	owner := f.rep // **أيُّ حسابٍ مالكاً** — يكفي أن يكون للمتجر مالك.
	if _, err := f.pool.Exec(ctx, `
		UPDATE merchants SET owner_user_id = $2, settlement_method = 'cash' WHERE id = $1`,
		f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `
		UPDATE order_items SET merchant_settlement_method = 'cash' WHERE order_id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	o, _, err := f.svc.MarkItemUnavailable(ctx, f.customer, f.orderID, f.lineA, 1, nil)
	if err != nil {
		t.Fatalf("الحذف: %v", err)
	}
	// **شراءٌ ٩٠٠ + ٤٥٠ = ١٣٥٠، وعمولةٌ ١٠٪ = ١٣٥ ⇒ يدفع السائقُ ١٢١٥.**
	if o.PayMerchant != 1_215 {
		t.Fatalf("ما يدفعه السائقُ للمتجر=%d — المنتظَر 1215", o.PayMerchant)
	}
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "picked_up", ""); err != nil {
		t.Fatalf("الاستلام: %v", err)
	}
	if got := f.platformCommission(t); got != 135 {
		t.Fatalf("عمولةُ المنصّة=%d — المنتظَر 135", got)
	}
	var amount int64
	if err := f.pool.QueryRow(ctx,
		`SELECT amount FROM merchant_settlements WHERE order_id = $1`, f.orderID).Scan(&amount); err != nil {
		t.Fatal(err)
	}
	if amount != 1_215 {
		t.Fatalf("مستحقُّ المتجر=%d — المنتظَر 1215", amount)
	}
}

// ── متجرٌ كاملٌ يخرج: رسمُ المصدر يسقط، ومتجرُ الطلب يخلفه الباقي ─────────
func TestItemUnavailable_SourceGone(t *testing.T) {
	f := iuSetup(t, "accepted", "cash")
	ctx := context.Background()
	var other string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, settlement_method)
		SELECT 'متجرٌ ثانٍ للحذف', category_id, 10, 'wallet' FROM merchants WHERE id = $1
		RETURNING id::text`, f.merchantID).Scan(&other); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, other)
	})
	// **«ب» من المتجر الثاني، والتوصيلُ ٣٠٠ منها ٢٠٠ رسمُ مصدرٍ إضافيّ.**
	if _, err := f.pool.Exec(ctx, `UPDATE order_items SET merchant_id = $2 WHERE id = $1`, f.lineB, other); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `
		UPDATE orders SET delivery_fee = 300, driver_fee = 300, sources_fee = 200,
		       total = 2800, cash_due = 2800 WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	// **ومتجرُ الطلب نفسُه يخرج** — «أ» كلُّه.
	_, res, err := f.svc.MarkItemUnavailable(ctx, f.customer, f.orderID, f.lineA, 0, nil)
	if err != nil || !res.SourceGone {
		t.Fatalf("الحذف: %v gone=%v", err, res.SourceGone)
	}
	st := f.read(t)
	if st.deliveryFee != 100 || st.sourcesFee != 0 || st.driverFee != 100 || st.total != 600 {
		t.Fatalf("بعد خروج المتجر: %+v — المنتظَر توصيلاً 100 بلا رسم مصدر ومجموعاً 600", st)
	}
	if st.merchantID != other {
		t.Fatalf("متجرُ الطلب=%s — المنتظَر الباقي %s", st.merchantID, other)
	}
}
