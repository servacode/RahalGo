package orders_test

// **توصيلةُ المتجر المسلَّمة ليست «طلباً بلا مستحقِّ متجر»** (فحصُ الدورة ٢٠٢٦-١٠-٠٥)
//
// التوصيلةُ لا بضاعةَ فيها — المتجرُ هو المُرسِل لا البائع، **فلا مستحقَّ له
// عنها.** وكان الثابتُ FI-04.a يستثني المخصَّصَ وحدَه، **فكلُّ توصيلةٍ سُلّمت
// تُقرأ خرقاً** ويصير `moneycheck` أحمرَ على دورةٍ سليمة.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

func TestMerchantDelivery_DeliveredHasNoMerchantDueViolation(t *testing.T) {
	f := setup(t, "at_dropoff", 0, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)
	if _, err := f.pool.Exec(ctx, `DELETE FROM order_items WHERE order_id = $1`, f.orderID); err != nil {
		t.Fatalf("تعذّر حذفُ البنود: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `
		UPDATE orders SET kind = 'merchant_delivery', recipient_name = 'مستلم',
		       recipient_phone = '0999000000', fee_payer = 'recipient'
		WHERE id = $1`, f.orderID); err != nil {
		t.Fatalf("تعذّر جعلُه توصيلة: %v", err)
	}
	base, err := fininv.Capture(ctx, f.pool, "FI-04.a")
	if err != nil {
		t.Fatalf("تعذّرت صورةُ الأساس: %v", err)
	}
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليمُ فشل: %v", err)
	}
	after, err := fininv.Run(ctx, f.pool, "FI-04.a")
	if err != nil {
		t.Fatalf("تعذّر الفحص: %v", err)
	}
	if fresh := base.New(after); len(fresh) > 0 {
		t.Fatalf("توصيلةٌ مسلَّمةٌ قُرئت «بلا مستحقِّ متجر»: %v", fresh)
	}
}
