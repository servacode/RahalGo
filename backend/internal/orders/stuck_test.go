package orders_test

import (
	"context"
	"testing"
)

// TestPostPickupHasExit ما بعد الاستلام صار له مخرجٌ للعمليات.
//
// # الثقبُ الذي يمسكه
//
// كانت `picked_up` لا تؤدّي إلّا إلى `on_the_way`، وتلك لا تؤدّي إلّا إلى
// `at_dropoff`. **فسائقٌ اختفى بطلبٍ في يده يترك الطلبَ عالقاً إلى الأبد**:
// لا يُلغى، ولا يُفشل، ولا يُسنَد لغيره — **والزبونُ ينتظر طعاماً لن يأتي ولا
// أحد يملك أن يُنهي انتظارَه.**
//
// وهي العلّةُ نفسُها التي كانت في `at_pickup`: **بابٌ يُدخَل منه ولا يُخرَج.**
//
// # ويفحص أربعةً
//
//  1. أن العملياتِ تملك تحريرَه من كلٍّ من الحالات الثلاث
//  2. أن **`driver_id` صار فارغاً** — وإلّا عاد إلى الطابور وسائقُه ملتصقٌ به
//  3. أن **السائقَ لا يملك ذلك بنفسه** — وإلّا ترك كلُّ من ثقل عليه طلبٌ طلبَه
//  4. أن **مالَ المتجر يبقى** — بضاعتُه خرجت فاستحقّ، ولا يُستردّ بتحريرٍ
//     لا ذنبَ له فيه
func TestPostPickupHasExit(t *testing.T) {
	for _, from := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		t.Run(from, func(t *testing.T) {
			f := setup(t, from, 100_000, 10_000, 0)
			ctx := context.Background()
			owner, _ := f.armTreasury(t)

			// **المتجرُ قبض عند خروج البضاعة** — نُثبّته قبل التحرير كي يُقاس.
			//
			// والحالاتُ هنا تُنشأ إدراجاً مباشراً، فلا تسويةَ جرت: نقيسُ أن
			// التحريرَ **لا يُحدث** قيداً لا أن قيداً سابقاً بقي.
			beforeOwner := f.balance(t, owner)

			// **والسائقُ لا يملكها بنفسه.**
			if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
				f.orderID, "dispatching", ""); err == nil {
				t.Fatal("السائقُ حرّر طلباً في يده — وترْكُ الطلب بعد الاستلام قرارُ منصة")
			}

			if _, err := f.svc.Transition(ctx, f.driver, []string{"ops"},
				f.orderID, "dispatching", "سائقٌ اختفى"); err != nil {
				t.Fatalf("العملياتُ لم تملك تحريرَه من %s: %v", from, err)
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
			if driverID != nil {
				t.Errorf("عاد إلى الطابور وسائقُه الغائبُ ملتصقٌ به: %s", *driverID)
			}
			if got := f.balance(t, owner); got != beforeOwner {
				t.Errorf("تحرّك مالُ المتجر بتحريرٍ لا ذنبَ له فيه: %d ← %d",
					beforeOwner, got)
			}
		})
	}
}
