package orders_test

// ══════════════════════════════════════════════════════════════════════
// **نصيبُ المندوب بعد الكوبون** — قرارُ المالك ٢٠٢٦-١٠-١٥
// ══════════════════════════════════════════════════════════════════════
//
// «الكوبونات من حصّة المنصّة» — **فقاعدةُ المندوب = الهامش + العمولة − خصمُ الكود −
// إعفاءُ التوصيل، وأرضُها صفر.** **وكان يأخذ نسبتَه من ربحٍ أكله الكوبون**:
// عمولةٌ خمسون وكوبونٌ خمسون ⇒ ربحُ المنصّة صفر، والمندوبُ يقبض خمسة.

import (
	"context"
	"testing"
)

func TestRepShare_AfterCoupon(t *testing.T) {
	cases := []struct {
		name             string
		discount, waived int64
		wantRep          int64
	}{
		// **عمولةٌ ٥٠ وكوبونٌ ٥٠ ⇒ لا شيء للمندوب.**
		{"كوبونٌ يأكل الربحَ كلَّه", 50, 0, 0},
		// **كوبونٌ ٢٠ ⇒ ١٠٪ من ٣٠ = ٣.**
		{"كوبونٌ بعشرين", 20, 0, 3},
		// **وإعفاءُ التوصيل كلفةُ كودٍ كذلك.**
		{"توصيلٌ مجّانيٌّ بعشرين", 0, 20, 3},
		// **وكوبونٌ أكبرُ من الربح لا يجعل المندوبَ مديناً.**
		{"كوبونٌ فوق الربح", 80, 0, 0},
		// **وبلا كوبون كما كان**: ١٠٪ من ٥٠ = ٥.
		{"بلا كوبون", 0, 0, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// **بضاعةٌ بخمسمئة شراءً وبيعاً** — هامشٌ صفر، وعمولةُ المتجر ١٠٪ = ٥٠.
			f := setup(t, "at_dropoff", 500, 100, 0)
			ctx := context.Background()
			total := 500 + 100 - c.discount
			if _, err := f.pool.Exec(ctx, `
				UPDATE order_items SET merchant_price = unit_price WHERE order_id = $1`,
				f.orderID); err != nil {
				t.Fatalf("تصفيرُ الهامش: %v", err)
			}
			if _, err := f.pool.Exec(ctx, `
				UPDATE orders SET discount = $2, promo_delivery_waived = $3,
				       total = $4, cash_due = $4 WHERE id = $1`,
				f.orderID, c.discount, c.waived, total); err != nil {
				t.Fatalf("ضبطُ الكوبون: %v", err)
			}
			if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
				f.orderID, "delivered", ""); err != nil {
				t.Fatalf("التسليم: %v", err)
			}
			if got := f.platformCommission(t); got != 50 {
				t.Fatalf("عمولةُ المنصّة = %d — المنتظَر 50", got)
			}
			if got := f.balance(t, f.rep); got != c.wantRep {
				t.Errorf("نصيبُ المندوب = %d — المنتظَر %d (١٠٪ من العمولة ناقصَ الكوبون)",
					got, c.wantRep)
			}
		})
	}
}
