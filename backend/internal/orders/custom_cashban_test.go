package orders_test

// ══════════════════════════════════════════════════════════════════════
//  D6 — **حظرُ النقد يشمل الطلبَ الخاصَّ كما يشمل العاديّ**
// ══════════════════════════════════════════════════════════════════════
//
// (بوّابةُ العوائق · `CUSTOMER-ACCEPTANCE-MASTER.md`.)
//
// # العيب
//
// **حظرُ النقد كان يُفحَص في مسار الطلب العاديّ وحدَه** (`CreateTx` →
// `cashBlocked`). **والطلبُ الخاصُّ (`POST /orders/custom`) بابٌ مفتوح**: من
// رفض الاستلامَ فقُفل عليه النقدُ يُنشئ طلباً خاصّاً نقداً — أو يُغفل حقلَ
// الدفع فيصير نقداً تلقائيّاً — **فيلتفّ على القفل الذي يحمي مالَ المنصّة.**
//
// # العقدُ بعد الإصلاح
//
// **من مُنع النقدَ في العاديّ يُمنعه في الخاصّ** — بالمصدر نفسِه
// (`cashBlocked`) والرسالةِ نفسِها (`ErrCashBlocked`). **والمحفظةُ تبقى
// مفتوحةً له** — الحظرُ على النقد لا على الحساب. **والطلبُ المرفوضُ لا يُكتب**:
// لا صفَّ طلبٍ خاصٍّ ولا قيدَ محفظة.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// banCashForCustomer **يجعل النقدَ محظوراً على زبون العُدّة** — إخفاقٌ واحدٌ
// بذنبه والعتبةُ واحدة، كما في `TestCashBlocked_AfterCustomerFault`.
func banCashForCustomer(t *testing.T, f *fixture) {
	t.Helper()
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES
			('customers.cash_ban_failures', '1'::jsonb),
			('customers.cash_ban_days', '30'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("تعذّر ضبطُ مفاتيح الحظر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`UPDATE app_settings SET value = '0'::jsonb
			  WHERE key = 'customers.cash_ban_failures'`)
	})
	// **الطلبُ الجاهزُ في العُدّة (`at_dropoff`) يفشل بذنب الزبون** — فيصير له
	// إخفاقٌ واحدٌ يُقفل النقدَ عليه.
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "failed",
		"customer_refused"); err != nil {
		t.Fatalf("تعذّر إفشالُ الطلب: %v", err)
	}
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET fault = 'customer' WHERE id = $1`, f.orderID); err != nil {
		t.Fatalf("تعذّر تثبيتُ الذنب: %v", err)
	}
}

// placeCustom **يُنشئ طلباً خاصّاً بالمسار الحقيقيّ** (`CreateCustomTx` في
// معاملة) — هو ما يناديه المعالِجُ `POST /orders/custom` بعينه.
//
// **ويُمرَّر `payment` كما يرسله الجسمُ**: `""` يعني إغفالَ الحقل (الحالةُ التي
// يصير فيها نقداً تلقائيّاً).
func placeCustom(ctx context.Context, f *fixture, payment string) (*orders.Order, error) {
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	o, _, err := f.svc.CreateCustomTx(ctx, tx, f.customer,
		"شاورما دجاج من مطعم الأصيل، بلا ثوم", "عنوانُ اختبار الخاصّ",
		payment, "", 35.9528, 39.0079)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return o, nil
}

// customCount **عددُ الطلبات الخاصّة لهذا الزبون** — يجب أن يبقى صفراً بعد رفضٍ.
func customCount(ctx context.Context, f *fixture) int64 {
	var n int64
	_ = f.pool.QueryRow(ctx,
		`SELECT count(*) FROM orders WHERE customer_id = $1 AND kind = 'custom'`,
		f.customer).Scan(&n)
	return n
}

// walletTxCount **قيودُ محفظة الزبون** — رفضٌ ماليٌّ لا يقيّد شيئاً.
func walletTxCount(ctx context.Context, f *fixture) int64 {
	var n int64
	_ = f.pool.QueryRow(ctx,
		`SELECT count(*) FROM wallet_transactions WHERE user_id = $1`, f.customer).Scan(&n)
	return n
}

// ── A + D · محظورُ النقد: نقداً صريحاً أو بإغفال الحقل — يُرفض ولا يُكتب ──────
//
// **والحالتان بابٌ واحد**: الجسمُ يرسل `"cash"`، أو يُغفل الحقلَ فيصير نقداً
// تلقائيّاً. **كلتاهما يجب أن تُرفض** — وإلّا بقي البابُ مفتوحاً بنصفه.
func TestCustomCashBan_BannedCashRejected(t *testing.T) {
	ctx := context.Background()
	for _, payment := range []struct{ name, val string }{
		{"cash صريحاً", "cash"},
		{"بإغفال الحقل", ""},
	} {
		t.Run(payment.name, func(t *testing.T) {
			f := setup(t, "at_dropoff", 100_000, 10_000, 0)
			banCashForCustomer(t, f)

			before := customCount(ctx, f)
			_, err := placeCustom(ctx, f, payment.val)
			if !errors.Is(err, orders.ErrCashBlocked) {
				t.Fatalf("**طلبٌ خاصٌّ نقديٌّ مُنع صاحبُه النقدَ ولم يُرفض** (%v) — "+
					"والبابُ الخاصُّ يلتفّ على قفلٍ يحمي مالَ المنصّة", err)
			}
			// **ولا أثرَ لرفضٍ**: لا صفَّ طلبٍ خاصٍّ جديد ولا قيدَ محفظة.
			if after := customCount(ctx, f); after != before {
				t.Errorf("**أُنشئ طلبٌ خاصٌّ رغم الرفض**: %d ⇐ %d", before, after)
			}
			if wtx := walletTxCount(ctx, f); wtx != 0 {
				t.Errorf("**رفضٌ نقديٌّ حرّك المحفظة**: %d قيداً", wtx)
			}
		})
	}
}

// ── C · محظورُ النقد يُبقى له بابُ المحفظة ───────────────────────────────
//
// **الحظرُ على النقد لا على الحساب** — من قُفل عليه النقدُ يطلب من محفظته.
func TestCustomCashBan_BannedWalletAccepted(t *testing.T) {
	ctx := context.Background()
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	banCashForCustomer(t, f)

	o, err := placeCustom(ctx, f, "wallet")
	if err != nil {
		t.Fatalf("**مُنع طلبٌ خاصٌّ بالمحفظة عن زبونٍ حُظر عنه النقدُ وحدَه** — %v", err)
	}
	if o.PaymentMethod != "wallet" {
		t.Errorf("طريقةُ الدفع ليست محفظة: %q", o.PaymentMethod)
	}
}

// ── B · من لم يُحظر عنه النقدُ يطلب خاصّاً نقداً كما كان ──────────────────
//
// **وحارسٌ يرفض الجميعَ نصفُ حارس**: لا بدّ أن يمرّ الطلبُ المشروع.
func TestCustomCashBan_AllowedCashAccepted(t *testing.T) {
	ctx := context.Background()
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	// **لا حظرَ** — زبونٌ لم يُخفق، ومفتاحُ العتبة صفرٌ افتراضاً.

	o, err := placeCustom(ctx, f, "cash")
	if err != nil {
		t.Fatalf("**رُفض طلبٌ خاصٌّ نقديٌّ مشروعٌ عن زبونٍ غيرِ محظور** — %v", err)
	}
	if o.PaymentMethod != "cash" {
		t.Errorf("طريقةُ الدفع ليست نقداً: %q", o.PaymentMethod)
	}
}
