package orders_test

import (
	"context"
	"testing"
)

// TestFaultFailureCountsAsViolation امتناعُ المتجر يُعدّ كما يُعدّ إلغاؤه.
//
// # الخللُ الذي يمسكه
//
// كان العدُّ يشترط `ended_by = 'merchant'` — **ومن أغلق بابَه والسائقُ عنده لم
// يُحسب عليه شيء**: الطلبُ ينتهي `failed` **وينهيه السائقُ لا المتجر**، فتقول
// `ended_by` «السائق» والذنبُ للمتجر.
//
// **وهو أسوأُ من الإلغاء لا أهون**: في الإلغاء يعرف الزبونُ باكراً، وفي
// الامتناع يكون السائقُ قد قاد والزبونُ قد انتظر — **ثمّ لا شيء.**
//
// # ويفحص ثلاثةً
//
//  1. أن العدّاد ارتفع بعد فشلٍ ذنبُه المتجر
//  2. أن **إنذاراً سُجّل** — لا إشعاراً يُقرأ ويُنسى
//  3. أن **الإنذارَ واحدٌ لا اثنان** على الطلب الواحد
func TestFaultFailureCountsAsViolation(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()

	before, err := f.svc.MerchantViolations(ctx, f.pool, f.merchantID)
	if err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}

	// **متجرٌ مغلقٌ والسائقُ عند بابه** — ذنبُه من القائمة لا من تقدير أحد.
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "المحل مغلق", "merchant_closed"); err != nil {
		t.Fatalf("الإفشال فشل: %v", err)
	}

	after, err := f.svc.MerchantViolations(ctx, f.pool, f.merchantID)
	if err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}
	if after != before+1 {
		t.Errorf("العدّاد قبل=%d بعد=%d — امتناعُ المتجر لم يُحسب عليه", before, after)
	}

	// **والإنذارُ سجلٌّ لا إشعار.**
	var n int
	if err := f.pool.QueryRow(ctx,
		`SELECT count(*) FROM warnings WHERE order_id = $1`, f.orderID).
		Scan(&n); err != nil {
		t.Fatalf("تعذّرت قراءة الإنذارات: %v", err)
	}
	if n != 1 {
		t.Errorf("عددُ الإنذارات = %d، والمتوقّع 1", n)
	}

	var reason string
	if err := f.pool.QueryRow(ctx,
		`SELECT reason FROM warnings WHERE order_id = $1`, f.orderID).
		Scan(&reason); err != nil {
		t.Fatalf("تعذّرت قراءة السبب: %v", err)
	}
	if reason != "merchant_closed" {
		t.Errorf("سببُ الإنذار = %q، والمتوقّع merchant_closed", reason)
	}
}

// TestCustomerFaultDoesNotWarnMerchant ذنبُ الزبون لا يُنذَر به المتجر.
//
// **والحدُّ هنا لا في نيّة أحد**: `FaultOf` تقرأ القائمة، **ومن غاب عن بابه
// ليس ذنبَ من طبخ.**
func TestCustomerFaultDoesNotWarnMerchant(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()

	before, err := f.svc.MerchantViolations(ctx, f.pool, f.merchantID)
	if err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "", "customer_absent"); err != nil {
		t.Fatalf("الإفشال فشل: %v", err)
	}
	after, err := f.svc.MerchantViolations(ctx, f.pool, f.merchantID)
	if err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}
	if after != before {
		t.Errorf("حُسب على المتجر ذنبُ غيره: قبل=%d بعد=%d", before, after)
	}
	var n int
	_ = f.pool.QueryRow(ctx,
		`SELECT count(*) FROM warnings WHERE order_id = $1`, f.orderID).Scan(&n)
	if n != 0 {
		t.Errorf("أُنذر المتجرُ بذنب الزبون: %d", n)
	}
}
