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

// TestMerchantFault_CompensatesAndOpensClaim المتجرُ يعتذر — **السائقُ يُعوَّض
// فوراً، والمطالبةُ تُفتح عليه.**
//
// # قرارُ المالك (٢٠٢٦-٠٨-٠٣)
//
// **«نعم، المنصة تعوّضه — وبفتح نزاع مع المتجر لحلّ القصة.»**
//
// # وكانت القاعدةُ تظلم السائق
//
// بُنيت سابقاً: «ذنبُ المتجر لا تعويضَ فيه — المنصةُ تتحمّل بضاعتَه وتعوّض
// سائقَها فلا نجمع عليها الاثنين». **فكان السائقُ يقود المشوارَ كاملاً ولا
// يأخذ شيئاً** بسبب متجرٍ اعتذر متأخّراً — **وهو لا يملك من أمر ذلك شيئاً.**
//
// # ولماذا يُدفع قبل الحسم
//
// **نزاعٌ يستغرق يوماً يترك من قاد مشوارَه بلا مقابلٍ يومَه كلَّه** — ومن قاد
// بلا مقابلٍ مرّةً يتردّد في الثانية. **والمطالبةُ تجري في مسارها.**
func TestMerchantFault_CompensatesAndOpensClaim(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)

	// **المتجرُ يعتذر والسائقُ عند بابه** — ذنبُه من القائمة لا من تقدير أحد.
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "اعتذر عن الصنف", "merchant_refused"); err != nil {
		t.Fatalf("الإفشال فشل: %v", err)
	}

	// **١ · السائقُ عُوِّض** — نصفُ رسم التوصيل (الافتراضيّ).
	comp := f.balance(t, f.driver)
	if comp <= 0 {
		t.Fatalf("السائقُ لم يُعوَّض عن مشوارٍ ضاع بذنب المتجر: %d", comp)
	}

	// **٢ · والنزاعُ فُتح بما دُفع** — لا برقمٍ يُحسب من جديد.
	//
	// **وموضعُه `disputes` لا صفُّ الإنذار** (هجرة `0063`): الإنذارُ سلوكٌ يُعدّ
	// ولا يُسوّى، **والنزاعُ مالٌ يُسوّى ويُغلق.** وبقاءُ المال في صفّ الإنذار
	// هو ما منع أن يكون للسائق أو الزبون نزاعٌ أصلاً.
	var claim int64
	var status string
	var warningID *string
	if err := f.pool.QueryRow(ctx, `
		SELECT amount, status, warning_id::text FROM disputes
		WHERE order_id = $1 AND party_role = 'merchant'`,
		f.orderID).Scan(&claim, &status, &warningID); err != nil {
		t.Fatalf("لم يُفتح نزاع: %v", err)
	}
	if claim != comp {
		t.Errorf("النزاع = %d والتعويضُ = %d — **يجب أن يتطابقا**", claim, comp)
	}
	// **ومفتوحٌ لا محسوم**: الخصمُ قرارُ إنسانٍ بعد أن يسمع المتجر.
	if status != "open" {
		t.Errorf("حُسم النزاعُ آلياً: %q — **ومالٌ يخرج قبل أن يُسأل نزاعٌ خُسر**", status)
	}
	// **والرابطُ يمنع نسختين من الحقيقة** — من قرأ الإنذارَ وجد كلفتَه.
	if warningID == nil {
		t.Error("النزاعُ بلا إنذارٍ مرجعيّ — **ومن قرأ الإنذارَ لا يجد كلفتَه**")
	}
}
