package orders_test

// **طلبُ محفظةٍ أُلغي قبل المتجر واستُردّ ليس خرقاً** (فحصُ الدورة ٢٠٢٦-١٠-٠٥)
//
// الإلغاءُ قبل المحاسبة يُعيد ما خُصم (`refund`) فيصير صافي دفتر الطلب صفراً،
// **و`wallet_paid` يبقى شاهداً على ما دُفع.** وكان FI-06.d ينتظر صافياً يساوي
// `-wallet_paid` دائماً، **فكلُّ إلغاءٍ سليمٍ من المحفظة يُقرأ خرقاً.**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

func TestWalletOrderCancelledBeforeSettlement_NoFI06dViolation(t *testing.T) {
	f := setup(t, "pending", 20_000, 5_000, 25_000)
	ctx := context.Background()
	f.armTreasury(t)
	if _, err := f.wallet.Apply(ctx, f.customer, 25_000, "topup", "seed-"+f.orderID, "رصيدُ اختبار", nil); err != nil {
		t.Fatalf("تعذّر الشحن: %v", err)
	}
	if _, err := f.wallet.Apply(ctx, f.customer, -25_000, "order_payment", f.orderID, "", nil); err != nil {
		t.Fatalf("تعذّر خصمُ الطلب: %v", err)
	}
	base, err := fininv.Capture(ctx, f.pool, "FI-06.d")
	if err != nil {
		t.Fatalf("تعذّرت صورةُ الأساس: %v", err)
	}
	if _, err := f.svc.Transition(ctx, f.customer, []string{"customer"}, f.orderID, "cancelled", "غيّرتُ رأيي"); err != nil {
		t.Fatalf("الإلغاءُ فشل: %v", err)
	}
	if got := f.balance(t, f.customer); got != 25_000 {
		t.Fatalf("رصيدُ الزبون بعد الإلغاء = %d والمتوقّع 25000", got)
	}
	after, err := fininv.Run(ctx, f.pool, "FI-06.d")
	if err != nil {
		t.Fatalf("تعذّر الفحص: %v", err)
	}
	if fresh := base.New(after); len(fresh) > 0 {
		t.Fatalf("إلغاءٌ مستردٌّ من المحفظة قُرئ خرقاً: %v", fresh)
	}
}
