package orders_test

import (
	"context"
	"testing"
)

// TestCompensation_FollowsFault التعويضُ يتبع الذنبَ لا تقديرَ أحد.
//
// **قرارُ المالك: تلقائيٌّ بلا يد.** ولو تُرك لحكمٍ لاحق **لصار قاعدةً تُنفَّذ
// بيدٍ — وقاعدةٌ تُنفَّذ بيدٍ ليست قاعدة، هي عادة.**
//
// ويفحص **الطرفين معاً**: أن يُعوَّض حين يستحقّ، **وألّا يُعوَّض حين لا يستحقّ**.
// **ومنحٌ بلا منعٍ ليس قاعدة، هو كرم.**
func TestCompensation_FollowsFault(t *testing.T) {
	cases := []struct {
		name   string
		reason string
		want   int64 // ٥٠٪ من رسم توصيلٍ ١٠٬٠٠٠
	}{
		{"عنوانٌ وهميّ — الحقُّ على الزبون", "address_wrong", 5_000},
		{"الزبونُ غائب", "customer_absent", 5_000},
		{"الزبونُ رفض", "customer_refused", 5_000},
		// **وذنبُ السائق لا تعويضَ فيه** — وهو ما يجعل وجودَ اللفظ في القائمة
		// اختباراً لصدقه لا زينةً.
		{"السائقُ تأخّر — الحقُّ عليه", "driver_late", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := setup(t, "at_dropoff", 100_000, 10_000, 0)
			ctx := context.Background()
			f.armTreasury(t)

			before := f.balance(t, f.driver)
			if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
				f.orderID, "failed", "", c.reason); err != nil {
				t.Fatalf("الإفشال فشل: %v", err)
			}
			if got := f.balance(t, f.driver) - before; got != c.want {
				t.Errorf("تعويضُ السائق = %d، والمتوقّع %d", got, c.want)
			}
		})
	}
}

// TestCompensation_UnknownReasonRejected رمزٌ مجهولٌ لا يُنسب إلى أحد.
//
// **ومجهولٌ لا يُحكَم به**: نسبتُه إلى الزبون تُعوّض بلا وجه، ونسبتُه إلى
// السائق تحرمه بلا وجه. **والسكوتُ أعدلُ من حكمٍ على غير بيّنة.**
func TestCompensation_UnknownReasonRejected(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)

	before := f.balance(t, f.driver)
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "شيءٌ ما", "سببٌ لا وجودَ له"); err != nil {
		t.Fatalf("الإفشال فشل: %v", err)
	}
	if got := f.balance(t, f.driver) - before; got != 0 {
		t.Errorf("عُوّض على سببٍ مجهول: %d", got)
	}
}
