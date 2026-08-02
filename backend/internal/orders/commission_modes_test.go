package orders_test

import (
	"context"
	"testing"
)

// TestCommission_IgnoresPaymentMethodAndManagementMode عمولةُ المندوب لا تتبع
// طريقةَ الدفع ولا من يدير الطلبات.
//
// **سؤالُ المالك**: «العمولة يجب أن تصل سواء دفع الزبونُ نقداً أو من المحفظة،
// والمندوبُ لا علاقة له بذلك — وحتى لو كانت إدارةُ الطلبات من داخل المنصة».
//
// وهو محقٌّ في الحكم، وهذا الاختبار يجعله **قاعدةً محروسة** لا اتفاقاً شفهياً:
//
//   - **طريقةُ الدفع** تقرّر أين يُقيَّد المال (صندوقُ السائق أم رصيدُ الزبون)
//     **ولا تقرّر هل يُقيَّد للمندوب شيء**. عمولتُه من قيمة البضاعة لا من
//     مسار النقد.
//   - **ووضعُ الإدارة** (`merchants.self_manage_orders`) شأنُ من يضغط الأزرار
//     — المتجرُ في بوابته أم العملياتُ نيابةً عنه. **والمندوبُ جلب المتجرَ في
//     الحالين، فحقُّه واحد.**
//
// ولو ربطهما أحدٌ يوماً بسطرٍ في التسوية لسقط هنا — **قبل أن يسقط في محفظة
// مندوبٍ يعمل في الميدان**.
func TestCommission_IgnoresPaymentMethodAndManagementMode(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		walletPaid int64
	}{
		{"نقداً عند الاستلام", "cash", 0},
		{"من المحفظة كاملاً", "wallet", 110_000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := setup(t, "at_dropoff", 100_000, 10_000, c.walletPaid)
			ctx := context.Background()

			// **المنصة تدير الطلبات** — المتجر خارج النظام يُبلَّغ برسالة.
			if _, err := f.pool.Exec(ctx, `
				INSERT INTO app_settings (key, value)
				VALUES ('merchants.self_manage_orders', 'false'::jsonb)
				ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
				t.Fatalf("تعذّر ضبط وضع الإدارة: %v", err)
			}
			t.Cleanup(func() {
				_, _ = f.pool.Exec(context.Background(), `
					UPDATE app_settings SET value = 'true'::jsonb
					WHERE key = 'merchants.self_manage_orders'`)
			})
			if _, err := f.pool.Exec(ctx,
				`UPDATE orders SET payment_method = $2 WHERE id = $1`,
				f.orderID, c.method); err != nil {
				t.Fatalf("تعذّر ضبط طريقة الدفع: %v", err)
			}

			if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
				f.orderID, "delivered", ""); err != nil {
				t.Fatalf("التسليم فشل: %v", err)
			}

			// عمولةُ المنصة ١٠٪ من **سعر الشراء** — لا من الإجمالي ولا من سعر
			// البيع. رسمُ التوصيل أجرُ خدمةٍ تؤدّيها المنصة بسائقها لا بيعُ
			// المتجر، **وسعرُ البيع فيه هامشُنا وليس بيعتَه.**
			if got := f.platformCommission(t); got != 9_000 {
				t.Errorf("عمولة المنصة = %d، والمتوقّع 9000", got)
			}
			// ونصيبُ المندوب ١٠٪ من **الهامش** — ١٠٬٠٠٠ فنصيبُه ١٬٠٠٠.
			if got := f.balance(t, f.rep); got != 1_000 {
				t.Errorf("عمولة المندوب = %d، والمتوقّع 1000 — %s ووضعُ الإدارة «المنصة تدير»",
					got, c.name)
			}
		})
	}
}
