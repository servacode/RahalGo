package orders_test

// ══════════════════════════════════════════════════════════════════════
// **صنفٌ غيرُ متوفّرٍ في طلب محفظة — والخزينةُ تقرأ المدفوعَ من الدفتر**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-١٠-٢٧: «اكيد صح الغلطين».) `MarkItemUnavailable` يردّ الفرقَ قيدَ
// `refund` **ويُنزل `wallet_paid` معه** — فكانت `creditTreasury` تطرح الردَّ مرّتين.
// **وصارت تقرأ المدفوعَ من قيود `order_payment`** فيُطرح الردُّ مرّةً واحدة.
//
// والمثالُ نفسُه في `money_wallet_order_scenarios_test.go`: دفع ٣٥٠٠ · يُحذف صنفٌ بـ٦٠٠ ⇒
// ٢٩٠٠ · المتجر ١٨٠٠ · السائق ٤٠٠ · المندوب ٦٠ ⇒ الخزينةُ ٦٤٠.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

// iuOrder **طلبُ المثال بعد حذف الصنف الثاني** — والمدفوعُ صار ٢٩٠٠ والمردودُ ٦٠٠.
func iuOrder(t *testing.T, f *woFixture) string {
	t.Helper()
	oid, lines := f.order(t, "wallet", 500, 400,
		woLine{sale: 1200, cost: 1000, qty: 2}, woLine{sale: 600, cost: 500, qty: 1})
	if _, _, err := f.svc.MarkItemUnavailable(context.Background(), f.customer, oid, lines[1], 0, nil); err != nil {
		t.Fatalf("حذفُ الصنف: %v", err)
	}
	if got := f.balance(t, f.customer); got != 600 {
		t.Fatalf("ردُّ المحفظة الجزئيّ %d لا 600", got)
	}
	return oid
}

// TestItemUnavailable_DeliveredThenRefunded_TreasuryConserves **صنفٌ غيرُ متوفّر ⇒ تسليم ⇒
// استرجاعٌ كامل** — الزبونُ يستردّ ٢٩٠٠ فوق الـ٦٠٠ (٣٥٠٠ كلَّها)، ومستحقُّ المتجر يُعكَس،
// **والسائقُ والمندوبُ يحتفظان** (قرارُ المالك ٢٠٢٦-١٠-١٦) ⇒ الخزينةُ −٤٦٠ (٤٠٠ + ٦٠) لا غير.
//
// وعلى الحسبة القديمة: المدفوعُ ٢٩٠٠ والمردودُ ٣٥٠٠ ⇒ **−١٠٦٠** — الـ٦٠٠ مرّتين.
func TestItemUnavailable_DeliveredThenRefunded_TreasuryConserves(t *testing.T) {
	f := newWO(t, "wallet")
	base := f.baseline(t)
	oid := iuOrder(t, f)
	f.deliverFromPickup(t, oid)
	if got := f.sumKind(t, oid, "platform_profit"); got != 640 {
		t.Fatalf("ربحُ الخزينة بعد التسليم %d لا 640", got)
	}
	f.transition(t, oid, "refunded")

	if got := f.balance(t, f.customer); got != 3500 {
		t.Errorf("رصيدُ الزبون %d — والمنتظَرُ 3500 (600 عند الحذف + 2900 بالاسترجاع)", got)
	}
	if got := f.balance(t, f.owner); got != 0 {
		t.Errorf("بقي للمتجر %d عن طلبٍ مُسترجَع", got)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != -460 {
		t.Errorf("ربحُ الخزينة على طلبٍ رُدّ كاملاً %d — والمنتظَرُ −460 (أجرُ السائق 400 + المندوب 60)", got)
	}
	f.assertGreen(t, base)
}

// TestItemUnavailable_ThenCancelled_RefundsRemaining **صنفٌ غيرُ متوفّر ⇒ إلغاءٌ قبل الاستلام**
// — يُردّ **الباقي** (٢٩٠٠ من العمود) لا الأصلُ (٣٥٠٠): **الزبونُ يستردّ ما دفع لا أكثر.**
func TestItemUnavailable_ThenCancelled_RefundsRemaining(t *testing.T) {
	f := newWO(t, "wallet")
	base := f.baseline(t)
	oid := iuOrder(t, f)
	if _, err := f.svc.Transition(context.Background(), f.treasury, []string{"admin"},
		oid, "cancelled", "اختبار"); err != nil {
		t.Fatalf("الإلغاء: %v", err)
	}
	if got := f.balance(t, f.customer); got != 3500 {
		t.Errorf("رصيدُ الزبون %d — والمنتظَرُ 3500 بالضبط (لا ردَّ مضاعف)", got)
	}
	if got := f.sumKind(t, oid, "refund"); got != 3500 {
		t.Errorf("مجموعُ الردود %d لا 3500 (600 + 2900)", got)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != 0 {
		t.Errorf("ربحٌ %d على طلبٍ أُلغي قبل الاستلام", got)
	}
	f.assertGreen(t, base)
}

// TestItemUnavailable_ThenFailedAfterPickup_TreasuryConserves **صنفٌ غيرُ متوفّر ⇒ فشلٌ بعد
// الاستلام بذنب المنصّة** — يُردّ ٢٩٠٠ والمتجرُ قبض ١٨٠٠ ⇒ الخزينةُ −١٨٠٠ (خسارتُها البضاعة)،
// وثابتُ الحفظ أخضر. وعلى الحسبة القديمة كانت −٢٤٠٠.
func TestItemUnavailable_ThenFailedAfterPickup_TreasuryConserves(t *testing.T) {
	f := newWO(t, "wallet")
	base := f.baseline(t)
	oid := iuOrder(t, f)
	f.failOnTheWay(t, oid)
	if got := f.balance(t, f.customer); got != 3500 {
		t.Errorf("رصيدُ الزبون %d لا 3500", got)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != -1800 {
		t.Errorf("ربحُ الخزينة %d — والمنتظَرُ −1800 (ما قبضه المتجر)", got)
	}
	f.assertGreen(t, base)
}

// TestItemUnavailable_OldShape_RecomputeCorrects **طلبٌ قديمٌ حُسب بالعطب ⇒ «إعادة الحساب» تصحّحه.**
//
// يُبنى الطلبُ ويُسلَّم، **ثمّ يُزرع قيدُ العطب القديم بيده** (−٦٠٠ `platform_profit` — ما كانت
// تكتبه الحسبةُ القديمة فيصير المجموعُ ٤٠). ثمّ `CreditTreasuryTx` — **وهو ما ينادي به
// `POST /admin/orders/{id}/recompute` بعينه** — فيُكتب قيدُ تصحيحٍ بالفرق (+٦٠٠).
func TestItemUnavailable_OldShape_RecomputeCorrects(t *testing.T) {
	f := newWO(t, "wallet")
	ctx := context.Background()
	base := f.baseline(t)
	oid := iuOrder(t, f)
	f.deliverFromPickup(t, oid)

	// ── الشكلُ القديم: ربحٌ ٤٠ لا ٦٤٠ ──
	if _, err := f.wallet.Apply(ctx, f.treasury, -600, "platform_profit", oid,
		"تسويةُ نصيب المنصة", nil); err != nil {
		t.Fatalf("زرعُ الشكل القديم: %v", err)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != 40 {
		t.Fatalf("الشكلُ القديم %d لا 40", got)
	}
	after, err := fininv.Run(ctx, f.pool, "FI-06", "FI-10", "FI-14")
	if err != nil {
		t.Fatal(err)
	}
	if len(base.New(after)) == 0 {
		t.Fatal("الشكلُ القديمُ لم يُقرأ خرقاً في FI-06 — الحارسُ أعمى")
	}

	// ── إعادةُ الحساب ──
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.CreditTreasuryTx(ctx, tx, oid, f.treasury); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("إعادةُ الحساب: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if got := f.sumKind(t, oid, "platform_profit"); got != 640 {
		t.Errorf("بعد إعادة الحساب %d لا 640", got)
	}
	var corrections int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'platform_profit' AND amount = 600`, oid).Scan(&corrections); err != nil {
		t.Fatal(err)
	}
	if corrections != 1 {
		t.Errorf("قيودُ التصحيح (+600) %d — والمنتظَرُ واحد", corrections)
	}
	f.assertGreen(t, base)
}
