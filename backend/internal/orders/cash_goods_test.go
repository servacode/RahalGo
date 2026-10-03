package orders_test

// **متجرٌ «نقداً» رُدّت إليه بضاعتُه** — فحصُ المتجر ٢٠٢٦-١٠-٠١، بقرار المالك.
//
// **قِيس على التجهيز**: بقي مستحقُّه كاملاً (٢٧٬٠٠٠) — `clawBackGoods` يقرأ المحفظةَ،
// ومستحقُّ النقديّ في الاحتباس. **ولا دعمَ تلقائيّاً بعد قرار المالك ٢٠٢٦-١٠-٠٣** — والتعويضُ
// مبلغٌ تكتبه الإدارة.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

func cashGoodsCase(t *testing.T, paid bool, compensation int64) (*cashFixture, string, string, string) {
	t.Helper()
	f := newCashFixture(t)
	ctx := context.Background()
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
	// **وعند الباب المكتبُ يُنهي** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢) — السائقُ لا يُغلق.
	endAtDoor(t, f.svc, f.pool, oid, orders.FaultCustomer, "customer_refused")
	if err := f.svc.SettleGoods(ctx, oid, orders.GoodsToMerchant, f.treasury, compensation); err != nil {
		t.Fatalf("الحسم: %v", err)
	}
	return f, m, owner, oid
}

// TestSETG1_CashDueReturnedGoodsIsReversed **لم يُدفع بعد ⇒ يُعكس، ولا دعمَ تلقائيّاً.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٣: «لازم المصاري ترجع ع حالها والإدارة تقرر تعوض المتجر او لا».)
func TestSETG1_CashDueReturnedGoodsIsReversed(t *testing.T) {
	f, m, owner, oid := cashGoodsCase(t, false, 0)
	s := f.settlement(t, oid, m)
	if s.state != "cash_reversed" || s.reversed != 90_000 {
		t.Fatalf("**المستحقُّ النقديّ لم يُعكس** — بقي للمتجر ثمنُ بضاعةٍ أخذها: %+v", s)
	}
	if got := f.sumKind(t, oid, "merchant_cash_accrued"); got != 0 {
		t.Fatalf("الاحتباسُ لهذا الطلب %d لا صفر", got)
	}
	if got := f.balance(t, owner); got != 0 {
		t.Fatalf("**دُفع للمتجر %d بلا قرار** — والمالُ يعود كما كان", got)
	}
	f.assertFIGreen(t)
}

// TestSETG2_CashPaidReturnedGoodsBecomesDebt **دُفع نقداً ⇒ التزامٌ عليه، ولا دعمَ تلقائيّاً.**
func TestSETG2_CashPaidReturnedGoodsBecomesDebt(t *testing.T) {
	f, m, owner, _ := cashGoodsCase(t, true, 0)
	if got := f.obligation(t, m); got != 90_000 {
		t.Fatalf("**قبض ثمنَ بضاعةٍ رُدّت إليه ولا التزامَ عليه**: %d", got)
	}
	if got := f.balance(t, owner); got != 0 {
		t.Fatalf("دُفع للمتجر %d بلا قرار", got)
	}
	f.assertFIGreen(t)
}

// TestSETG3_CashAdminCompensation **وتعويضُ الإدارة للمتجر النقديّ** — في محفظته من الخزينة،
// **والعكسُ كما هو.**
func TestSETG3_CashAdminCompensation(t *testing.T) {
	f, m, owner, oid := cashGoodsCase(t, false, 6_000)
	if s := f.settlement(t, oid, m); s.state != "cash_reversed" || s.reversed != 90_000 {
		t.Fatalf("المستحقُّ النقديّ لم يُعكس: %+v", s)
	}
	if got := f.balance(t, owner); got != 6_000 {
		t.Fatalf("تعويضُ الإدارة %d لا ٦٬٠٠٠", got)
	}
	f.assertFIGreen(t)
}
