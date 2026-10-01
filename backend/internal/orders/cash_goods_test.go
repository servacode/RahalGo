package orders_test

// **متجرٌ «نقداً» رُدّت إليه بضاعتُه** — فحصُ المتجر ٢٠٢٦-١٠-٠١، بقرار المالك.
//
// **قِيس على التجهيز**: بقي مستحقُّه كاملاً (٢٧٬٠٠٠) ولم يُدفع دعم — `clawBackGoods`
// يقرأ المحفظةَ، ومستحقُّ النقديّ في الاحتباس.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

func cashGoodsCase(t *testing.T, paid bool) (*cashFixture, string, string, string) {
	t.Helper()
	f := newCashFixture(t)
	ctx := context.Background()
	st := settings.NewStore(f.pool)
	var before *string
	_ = f.pool.QueryRow(ctx, `SELECT value::text FROM app_settings WHERE key = 'merchants.return_support_percent'`).Scan(&before)
	if err := st.SetInternal(ctx, "merchants.return_support_percent", 20); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if before == nil {
			_, _ = f.pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key = 'merchants.return_support_percent'`)
		} else {
			_, _ = f.pool.Exec(context.Background(), `UPDATE app_settings SET value = $1::jsonb WHERE key = 'merchants.return_support_percent'`, *before)
		}
	})
	m, owner := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 100_000, 0, "cash")
	f.pickup(t, oid)
	if s := f.settlement(t, oid, m); s.state != "cash_due" || s.amount != 90_000 {
		t.Fatalf("المستحقُّ النقديّ: %+v", s)
	}
	if paid {
		if _, err := f.svc.MarkCashSettlementPaid(ctx, f.settlementID(t, oid, m), f.treasury, "دُفع", ""); err != nil {
			t.Fatal(err)
		}
	}
	for _, to := range []string{"on_the_way", "at_dropoff"} {
		f.transition(t, oid, to)
	}
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"}, oid, "failed", "", "customer_absent"); err != nil {
		t.Fatalf("الإفشال: %v", err)
	}
	if err := f.svc.SettleGoods(ctx, oid, orders.GoodsToMerchant, f.treasury); err != nil {
		t.Fatalf("الحسم: %v", err)
	}
	return f, m, owner, oid
}

// TestSETG1_CashDueReturnedGoodsIsReversed **لم يُدفع بعد ⇒ يُعكس، والدعمُ في محفظته.**
func TestSETG1_CashDueReturnedGoodsIsReversed(t *testing.T) {
	f, m, owner, oid := cashGoodsCase(t, false)
	s := f.settlement(t, oid, m)
	if s.state != "cash_reversed" || s.reversed != 90_000 {
		t.Fatalf("**المستحقُّ النقديّ لم يُعكس** — بقي للمتجر ثمنُ بضاعةٍ أخذها: %+v", s)
	}
	if got := f.sumKind(t, oid, "merchant_cash_accrued"); got != 0 {
		t.Fatalf("الاحتباسُ لهذا الطلب %d لا صفر", got)
	}
	if got := f.balance(t, owner); got != 18_000 {
		t.Fatalf("**دعمُ المرتجع %d لا ١٨٬٠٠٠** (٢٠٪ من ٩٠٬٠٠٠)", got)
	}
	f.assertFIGreen(t)
}

// TestSETG2_CashPaidReturnedGoodsBecomesDebt **دُفع نقداً ⇒ التزامٌ عليه، والدعمُ في محفظته.**
func TestSETG2_CashPaidReturnedGoodsBecomesDebt(t *testing.T) {
	f, m, owner, _ := cashGoodsCase(t, true)
	if got := f.obligation(t, m); got != 90_000 {
		t.Fatalf("**قبض ثمنَ بضاعةٍ رُدّت إليه ولا التزامَ عليه**: %d", got)
	}
	if got := f.balance(t, owner); got != 18_000 {
		t.Fatalf("دعمُ المرتجع %d لا ١٨٬٠٠٠", got)
	}
	f.assertFIGreen(t)
}
