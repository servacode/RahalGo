package orders_test

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// TestCompensation_FollowsFault **الذنبُ يقرّر أيُطلَب تعويض — ولا يُقيَّد شيء.**
//
// **كان التعويضُ تلقائيّاً بلا يد — ونُسخ ٢٠٢٦-١٠-٠٢**: قِيس على التجهيز ٥٬٠٠٠
// تُدفع فوراً في كلّ ضغطة. **فالذنبُ يقرّر الاستحقاق، والعملياتُ توافق.**
//
// ويفحص **الطرفين معاً**: أن يُطلَب حين يستحقّ، **وألّا يُطلَب حين لا يستحقّ**.
func TestCompensation_FollowsFault(t *testing.T) {
	cases := []struct {
		name    string
		reason  string
		pending bool
	}{
		{"عنوانٌ وهميّ — الحقُّ على الزبون", "address_wrong", true},
		{"الزبونُ رفض", "customer_refused", true},
		// **وذنبُ السائق لا تعويضَ فيه** — وهو ما يجعل وجودَ اللفظ في القائمة
		// اختباراً لصدقه لا زينةً.
		{"السائقُ تأخّر — الحقُّ عليه", "driver_late", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := setup(t, "at_dropoff", 100_000, 10_000, 0)
			f.armTreasury(t)

			before := f.balance(t, f.driver)
			// **والذنبُ ما يقترحه السببُ ويكتبه المكتب** (مساءَ ٢٠٢٦-١٠-٠٢).
			f.failAtDoor(t, orders.SuggestedFault(c.reason), c.reason)
			if got := f.balance(t, f.driver) - before; got != 0 {
				t.Errorf("قُيّد للسائق %d بلا موافقة", got)
			}
			if _, _, found := f.pendingRequest(t); found != c.pending {
				t.Errorf("طلبُ التعويض = %v والمتوقّع %v", found, c.pending)
			}
		})
	}
}

// TestCompensation_UnknownReasonRejected **رمزٌ مجهولٌ يُردّ — لا يُنسب إلى أحد.**
//
// **كان يُقبل ويُغلق الطلبَ بلا ذنب** — والمحرّكُ اليومَ يردّ ما لا يخصّ المرحلة
// (٢٠٢٦-١٠-٠٢)، **والمجهولُ لا يخصّ مرحلةً أصلاً.**
func TestCompensation_UnknownReasonRejected(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)

	before := f.balance(t, f.driver)
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "شيءٌ ما", "سببٌ لا وجودَ له"); err == nil {
		t.Fatal("قُبل رمزٌ مجهول")
	}
	if got := f.balance(t, f.driver) - before; got != 0 {
		t.Errorf("عُوّض على سببٍ مجهول: %d", got)
	}
	if _, _, found := f.pendingRequest(t); found {
		t.Error("طُلب تعويضٌ على سببٍ مجهول")
	}
}

// TestMerchantFault_PendingAndNoClaimYet المتجرُ يعتذر — **طلبُ تعويضٍ معلَّق،
// والمطالبةُ على المتجر تنتظر ما يُدفع فعلاً.**
//
// # قرارُ المالك (٢٠٢٦-٠٨-٠٣) باقٍ
//
// **«نعم، المنصة تعوّضه — وبفتح نزاع مع المتجر لحلّ القصة.»** **والذي تغيّر
// (٢٠٢٦-١٠-٠٢) متى**: بعد موافقة العمليات — **ومعها يُفتح النزاعُ بما دُفع**
// (`server/failure_aftermath.go`، ويُختبر هناك).
func TestMerchantFault_PendingAndNoClaimYet(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)

	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "اعتذر عن الصنف", "merchant_refused"); err != nil {
		t.Fatalf("الإفشال فشل: %v", err)
	}
	if got := f.balance(t, f.driver); got != 0 {
		t.Fatalf("قُيّد للسائق %d بلا موافقة", got)
	}
	fault, suggested, found := f.pendingRequest(t)
	if !found || fault != "merchant" || suggested != 5_000 {
		t.Fatalf("الطلبُ المعلَّق (%v · %s · %d) والمتوقّع (true · merchant · 5000)", found, fault, suggested)
	}
	var claims int
	if err := f.pool.QueryRow(ctx,
		`SELECT count(*) FROM disputes WHERE order_id = $1 AND party_role = 'merchant'`,
		f.orderID).Scan(&claims); err != nil {
		t.Fatal(err)
	}
	if claims != 0 {
		t.Errorf("فُتح نزاعٌ بمالٍ لم يُدفع بعد: %d", claims)
	}
}
