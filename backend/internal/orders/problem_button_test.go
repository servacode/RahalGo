package orders_test

// ══════════════════════════════════════════════════════════════════════
// **زرُّ «لدي مشكلة» — لكلّ مرحلةٍ عملُها** (قرارُ المالك ٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس على التجهيز قبل القرار**: الخادمُ يقبل «الزبونُ غير موجود» والسائقُ
// عند المتجر، و«المتجرُ مغلق» وهو عند باب الزبون، **ويدفع ٥٬٠٠٠ فوراً في كلّ
// مرّة — حتّى لـ«تأخّرتُ أنا»**، ويقبل الفشلَ بعد ٦٨ ثانيةً من الوصول، **والتوصيلةُ
// تعلق إلى الأبد إن تعذّر متجرُها.** وكلُّ اختبارٍ هنا يسقط على واحدةٍ منها.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// pendingRequest طلبُ التعويض المعلَّق لهذا الطلب — وفراغٌ إن لم يُكتب.
func (f *fixture) pendingRequest(t *testing.T) (fault string, suggested int64, found bool) {
	t.Helper()
	err := f.pool.QueryRow(context.Background(), `
		SELECT fault, suggested_amount FROM driver_compensation_requests
		WHERE order_id = $1 AND driver_id = $2 AND status = 'pending'`,
		f.orderID, f.driver).Scan(&fault, &suggested)
	if err != nil {
		return "", 0, false
	}
	return fault, suggested, true
}

func (f *fixture) statusOf(t *testing.T) string {
	t.Helper()
	var st string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT status FROM orders WHERE id = $1`, f.orderID).Scan(&st); err != nil {
		t.Fatalf("تعذّرت قراءةُ الحال: %v", err)
	}
	return st
}

// TestFailReason_RefusedOutsideItsStage **السببُ يخصّ مرحلتَه — وإلّا رُدّ.**
func TestFailReason_RefusedOutsideItsStage(t *testing.T) {
	cases := []struct{ at, reason string }{
		{"at_pickup", "customer_absent"},
		{"at_pickup", "driver_late"},
		{"at_pickup", "address_wrong"},
		{"at_dropoff", "merchant_closed"},
		{"at_dropoff", "order_unknown"},
		// **والبلاغُ ليس فشلاً** — «الطلبُ غيرُ جاهز» لا يُفشل شيئاً.
		{"at_pickup", "merchant_not_ready"},
		{"at_dropoff", "سببٌ لا وجودَ له"},
	}
	for _, c := range cases {
		t.Run(c.at+"/"+c.reason, func(t *testing.T) {
			f := setup(t, c.at, 100_000, 10_000, 0)
			ctx := context.Background()
			f.armTreasury(t)
			before := f.balance(t, f.driver)
			_, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
				f.orderID, "failed", "", c.reason)
			// **وعند الباب لا فشلَ للسائق أصلاً** (مساءَ ٢٠٢٦-١٠-٠٢) — يُردّ
			// بالخارطة قبل أن يُسأل عن السبب.
			if c.at == "at_dropoff" && errors.Is(err, orders.ErrBadTransition) {
				err = orders.ErrFailReasonStage
			}
			if !errors.Is(err, orders.ErrFailReasonStage) {
				t.Fatalf("قُبل «%s» في %s (الخطأ: %v) — **وسببٌ لا يخصّ المرحلة دفع تعويضاً على التجهيز**",
					c.reason, c.at, err)
			}
			if st := f.statusOf(t); st != c.at {
				t.Errorf("تبدّلت الحالُ إلى %q مع الرفض", st)
			}
			if got := f.balance(t, f.driver) - before; got != 0 {
				t.Errorf("تحرّك مالُ السائق %d مع الرفض", got)
			}
		})
	}
}

// TestFailAtDropoff_NoAutoCompensation_PendingByFault **لا قيدَ تلقائيّ —
// وطلبٌ معلَّقٌ بحسب الذنب.** والذنبُ صار يكتبه المكتبُ حين يُنهي عند الباب
// (مساءَ ٢٠٢٦-١٠-٠٢) — انظر `door_resolution_test.go`.
func TestFailAtDropoff_NoAutoCompensation_PendingByFault(t *testing.T) {
	cases := []struct {
		reason    string
		fault     string
		suggested int64 // ٥٠٪ من أجرٍ قدرُه ١٠٬٠٠٠
		pending   bool
	}{
		{"customer_refused", "customer", 5_000, true},
		{"address_wrong", "customer", 5_000, true},
		// **«تأخّرتُ أنا» ذنبُ السائق إن أقرّه المكتب** — لا يُطلَب له شيء.
		{"driver_late", "driver", 0, false},
	}
	for _, c := range cases {
		t.Run(c.reason, func(t *testing.T) {
			f := setup(t, "at_dropoff", 100_000, 10_000, 0)
			ctx := context.Background()
			f.armTreasury(t)
			before := f.balance(t, f.driver)
			f.failAtDoor(t, c.fault, c.reason)
			if got := f.balance(t, f.driver) - before; got != 0 {
				t.Fatalf("قُيّد للسائق %d لحظةَ الإنهاء — **والتعويضُ بعد موافقة العمليات** (٢٠٢٦-١٠-٠٢)", got)
			}
			var fault string
			if err := f.pool.QueryRow(ctx, `SELECT COALESCE(fault,'') FROM orders WHERE id = $1`,
				f.orderID).Scan(&fault); err != nil {
				t.Fatal(err)
			}
			if fault != c.fault {
				t.Errorf("الذنبُ %q والمتوقّع %q — **والذنبُ ما كتبه المكتب**", fault, c.fault)
			}
			gotFault, suggested, found := f.pendingRequest(t)
			if found != c.pending {
				t.Fatalf("طلبُ تعويضٍ معلَّق = %v والمتوقّع %v", found, c.pending)
			}
			if found && (gotFault != c.fault || suggested != c.suggested) {
				t.Errorf("الطلبُ المعلَّق (%s · %d) والمتوقّع (%s · %d)", gotFault, suggested, c.fault, c.suggested)
			}
		})
	}
}

// TestMerchantBlocked_FaultFromReason_PendingNotPaid **تعذّرُ المتجر: يعود
// إلى المكتب، والذنبُ من السبب، والتعويضُ معلَّقٌ لا مدفوع.**
func TestMerchantBlocked_FaultFromReason_PendingNotPaid(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	_, treasury := f.armTreasury(t)
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "", "merchant_closed"); err != nil {
		t.Fatalf("تعذّرُ المتجر رُدّ: %v", err)
	}
	var status, fault, reason string
	if err := f.pool.QueryRow(ctx, `
		SELECT status, COALESCE(fault,''), COALESCE(fail_reason,'') FROM orders WHERE id = $1`,
		f.orderID).Scan(&status, &fault, &reason); err != nil {
		t.Fatal(err)
	}
	if status != "accepted" || fault != "merchant" || reason != "merchant_closed" {
		t.Errorf("(%s · %s · %s) والمتوقّع (accepted · merchant · merchant_closed)", status, fault, reason)
	}
	if got := f.balance(t, f.driver); got != 0 {
		t.Errorf("قُيّد للسائق %d — **والتعويضُ بموافقة العمليات**", got)
	}
	if got := f.balance(t, treasury); got != 0 {
		t.Errorf("تحرّكت الخزينةُ %d بلا موافقة", got)
	}
	if gotFault, suggested, found := f.pendingRequest(t); !found || gotFault != "merchant" || suggested != 5_000 {
		t.Errorf("الطلبُ المعلَّق (%v · %s · %d) والمتوقّع (true · merchant · 5000)", found, gotFault, suggested)
	}
	// **والمطالبةُ على المتجر تُفتح بما دُفع** — ولم يُدفع شيءٌ بعد.
	var disputes int
	if err := f.pool.QueryRow(ctx,
		`SELECT count(*) FROM disputes WHERE order_id = $1`, f.orderID).Scan(&disputes); err != nil {
		t.Fatal(err)
	}
	if disputes != 0 {
		t.Errorf("فُتح نزاعٌ قبل أن يُدفع شيء: %d", disputes)
	}
}

// TestNoCancelAfterPickup **«الطلبُ ما بيلتغي بعد ما يصير عند السائق».**
func TestNoCancelAfterPickup(t *testing.T) {
	for _, st := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		t.Run(st, func(t *testing.T) {
			f := setup(t, st, 100_000, 10_000, 0)
			ctx := context.Background()
			for _, roles := range [][]string{{"ops"}, {"admin"}} {
				if _, err := f.svc.Transition(ctx, f.driver, roles, f.orderID, "cancelled", "ألغى بالهاتف"); !errors.Is(err, orders.ErrBadTransition) {
					t.Fatalf("%v ألغى طلباً في %s: %v", roles, st, err)
				}
			}
			if got := f.statusOf(t); got != st {
				t.Fatalf("تبدّلت الحالُ إلى %q", got)
			}
			// **والإعادةُ إلى الطابور باقية** — مخرجُ العمليات الحيّ.
			if _, err := f.svc.Transition(ctx, f.driver, []string{"ops"}, f.orderID, "dispatching", "تبديلُ سائق"); err != nil {
				t.Fatalf("الإعادةُ إلى الطابور رُدّت: %v", err)
			}
		})
	}
}

// TestMerchantDelivery_BlockedAtPickup_NeverStuck **التوصيلةُ تعذّرت عند متجرها —
// وتبقى حيّةً بمخرجٍ قائم.**
//
// **كانت تُردّ إلى `accepted`** — ولا وجودَ لها في خارطة التوصيلة: لا تحويلَ
// ولا إلغاءَ ولا إعادة. **طلبٌ معلّقٌ إلى الأبد.**
func TestMerchantDelivery_BlockedAtPickup_NeverStuck(t *testing.T) {
	for _, exit := range []string{"dispatching", "cancelled"} {
		t.Run(exit, func(t *testing.T) {
			f := setup(t, "at_pickup", 0, 10_000, 0)
			ctx := context.Background()
			f.armTreasury(t)
			if _, err := f.pool.Exec(ctx, `
				UPDATE orders SET kind = 'merchant_delivery', recipient_name = 'مستلم',
				       recipient_phone = '0999000000', fee_payer = 'merchant'
				WHERE id = $1`, f.orderID); err != nil {
				t.Fatalf("تعذّر جعلُه توصيلة: %v", err)
			}
			if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
				f.orderID, "failed", "", "merchant_closed"); err != nil {
				t.Fatalf("تعذّرُ المتجر رُدّ: %v", err)
			}
			if st := f.statusOf(t); st != "at_pickup" {
				t.Fatalf("صارت التوصيلةُ %q — **و`accepted` لا مخرجَ منها في خارطتها**", st)
			}
			if _, _, found := f.pendingRequest(t); !found {
				t.Error("لا طلبَ تعويضٍ معلَّق — والسائقُ قاد إلى بابٍ مغلق")
			}
			// **والمخرجُ قائم** — العملياتُ تعيدها، **والإلغاءُ بعد التحويل للمالك**
			// (وضعُ «المنصة تدير»).
			roles := []string{"ops"}
			if exit == "cancelled" {
				roles = []string{"admin"}
			}
			if _, err := f.svc.Transition(ctx, f.driver, roles, f.orderID, exit, "قرارُ العمليات"); err != nil {
				t.Fatalf("العملياتُ لا تملك %s: %v — **التوصيلةُ عالقة**", exit, err)
			}
		})
	}
}
