package orders_test

import (
	"context"
	"testing"
)

// TestReleaseAtPickup السائقُ يحرّر الطلبَ وهو عند باب المتجر.
//
// # المسألة
//
// كان فكُّ الإسناد متاحاً في `assigned` وحدَها. **فمن قال «وصلتُ المتجر» ثمّ
// عرض له عارضٌ — عطلٌ، أو نداءٌ عاجل، أو انتظارٌ طال — لم يبقَ له إلّا زرُّ
// «تعذّر التسليم»**: يُقفل طلباً بضاعتُه لم تخرج بعد، ويُحسب على أحدٍ ذنبٌ لم
// يقع، **ويُحرم زبونٌ من طلبٍ كان سائقٌ آخر يوصله في دقائق.**
//
// # وثلاثةُ أشياءَ يفحصها هذا الاختبار
//
//  1. أن الانتقالَ **مسموح** — وهو التغيير الظاهر
//  2. أن **`driver_id` صار فارغاً** — وهو التغيير الذي لولاه لَعاد الطلبُ إلى
//     الطابور وسائقُه ملتصقٌ به: **يُرى مأخوذاً فلا يلتقطه أحد**
//  3. أن **لا مالَ تحرّك** — التحريرُ ترتيبٌ لا محاسبة، وبضاعةٌ لم تخرج لا
//     تُوجب على أحدٍ شيئاً
func TestReleaseAtPickup(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	owner, treasury := f.armTreasury(t)

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
		f.orderID, "dispatching", ""); err != nil {
		t.Fatalf("التحريرُ من باب المتجر رُفض: %v", err)
	}

	var driverID *string
	var status string
	if err := f.pool.QueryRow(ctx,
		`SELECT status, driver_id::text FROM orders WHERE id = $1`, f.orderID).
		Scan(&status, &driverID); err != nil {
		t.Fatalf("تعذّرت قراءة الطلب: %v", err)
	}
	if status != "dispatching" {
		t.Errorf("الحالة = %q، والمتوقّع dispatching", status)
	}
	// **الشرطُ الذي لولاه لَبقي الطلبُ معلّقاً في الطابور.**
	if driverID != nil {
		t.Errorf("عاد إلى الطابور وسائقُه ملتصقٌ به: %s", *driverID)
	}

	// **ولا مالَ تحرّك**: بضاعةٌ لم تخرج لا تُوجب على أحدٍ شيئاً.
	if got := f.balance(t, owner); got != 0 {
		t.Errorf("قُيّد للمتجر عند التحرير: %d", got)
	}
	if got := f.balance(t, f.driver); got != 0 {
		t.Errorf("قُيّد للسائق عند التحرير: %d", got)
	}
	if got := f.balance(t, treasury); got != 0 {
		t.Errorf("تحرّكت الخزينةُ عند التحرير: %d", got)
	}
}
