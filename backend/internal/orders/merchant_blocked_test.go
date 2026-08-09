package orders_test

// تعذّرٌ عند باب المتجر — **لا يُغلق الطلبَ ولا يُبلَّغ الزبون.**
//
// # الحادثة
//
// شهد المالكُ (٢٠٢٦-٠٨-٠٥): «إذا تعذّر استلامُ المنتج من المتجر والسائقُ أبلغ
// المنصة، يصل إلى الزبون فشلُ التسليم — **وهذا غلط**، لأنّه يوجد خيارُ بدل».
//
// **والفرقُ بين البابين هو كلُّ شيء:**
//
//	باب المتجر  ←  لا بضاعةَ خرجت · **بديلٌ قائم** · الطلبُ يعود للمكتب
//	باب الزبون  ←  البضاعةُ في الصندوق · لا بديل · **الطلبُ يُغلق**
//
// **وهما فعلٌ واحدٌ بزرٍّ واحدٍ في شاشة السائق** — وهو صواب: من يقف عند بابٍ
// **يقول ما وقع، والمحرّكُ يقرّر ما يعنيه.**

import (
	"context"
	"sync"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// recorder مُبلِّغٌ يكتب من نُودي بدل أن يُبلّغ.
//
// **وبغيره كان الاختبارُ أجوفَ**: عُدّةُ الاختبار لا تُبلّغ أحداً أصلاً، **فعدُّ
// إشعارات الزبون يجده صفراً في الحالين** — وينجح الاختبارُ ولو أُعيد الخللُ
// كاملاً.
type recorder struct {
	mu  sync.Mutex
	to  []string // من نُودي بالاسم
	ops int      // وكم نداءً للمكتب
}

func (r *recorder) Notify(_ context.Context, in notifications.Input) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.to = append(r.to, in.UserID)
}
func (r *recorder) NotifyRoles(context.Context, []string, notifications.Input) {}
func (r *recorder) NotifyOps(context.Context, notifications.Input) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ops++
}

// told هل نودي هذا الحساب.
func (r *recorder) told(userID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.to {
		if u == userID {
			return true
		}
	}
	return false
}

// TestMerchantBlocked_ReturnsToOfficeNotClosed **يعود للمكتب لا يُغلق.**
func TestMerchantBlocked_ReturnsToOfficeNotClosed(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)

	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "المطعمُ مغلق", "merchant_closed"); err != nil {
		t.Fatalf("تعذّر الإبلاغ: %v", err)
	}

	var status string
	var driverID *string
	var closed, sent bool
	if err := f.pool.QueryRow(ctx, `
		SELECT status, driver_id::text, closed_at IS NOT NULL,
		       sent_to_merchant_at IS NOT NULL
		FROM orders WHERE id = $1`, f.orderID).
		Scan(&status, &driverID, &closed, &sent); err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلب: %v", err)
	}

	if closed {
		t.Fatal("أُغلق الطلبُ — **وللمنصة بديلٌ في يدها: تبديلُ المتجر**، " +
			"وزبونٌ يُقال له «فشل» وثمّة حلٌّ زبونٌ خسرناه بلا سبب")
	}
	if status != "accepted" {
		t.Fatalf("الحالة %q لا «accepted» — **وهي حيث يظهر زرّا التحويل وتبديل المتجر**", status)
	}
	// **ولو عاد `dispatching` لَنزل إلى سائقٍ آخرَ يقف أمام المطعم المغلق
	// نفسِه** — ويتكرّر إلى أن ينتبه أحد.
	if driverID != nil {
		t.Fatalf("بقي السائقُ ملتصقاً به: %s — **والمشوارُ انتهى**", *driverID)
	}
	if sent {
		t.Fatal("بقي وسمُ الإبلاغ — **والتحويلُ يبدأ من أوّله إلى متجرٍ آخر**")
	}
}

// TestMerchantBlocked_CustomerIsNotTold **والزبونُ لا يعلم — لأنّ شيئاً لم
// يقع في حقّه.**
//
// **وهذا جوهرُ التصويب**: طلبُه قائمٌ ويُحوَّل إلى مطبخٍ آخر.
func TestMerchantBlocked_CustomerIsNotTold(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)

	rec := &recorder{}
	f.svc.SetNotifier(rec)

	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "", "merchant_refused"); err != nil {
		t.Fatalf("تعذّر الإبلاغ: %v", err)
	}

	if rec.told(f.customer) {
		t.Fatal("نودي الزبونُ — **وهو لم يُخطئ، ولم يخرج من المطبخ شيءٌ بعد، " +
			"وطلبُه قائمٌ يُحوَّل إلى مطبخٍ آخر**")
	}
	// **والصمتُ وحدَه ليس نجاحاً**: لو لم يُنادَ أحدٌ لَبقي الطلبُ في المكتب
	// **ولا أحدَ يعلم أنّ عليه تبديلَ المتجر** — وهو أسوأُ من إشعارٍ خاطئ.
	if rec.ops == 0 {
		t.Fatal("لم يُنادَ المكتب — **والطلبُ ينتظر تبديلاً لا يأتي**")
	}
}

// TestMerchantBlocked_StillCountsAgainstTheMerchant **ووجودُ البديل لا يُبرّئ.**
//
// **وهذا ما كاد يضيع في التصويب**: الإنذارُ كان معلّقاً بحالة `failed`،
// **وتحويلُ المسار قفز فوقه.** فمن أغلق بابَه عشر مرّاتٍ يبقى بلا مخالفة،
// **والمنصةُ تدفع التعويضَ في كلّ مرّةٍ وتبتلعه صامتة.**
//
// **بل هو هنا أوجب**: الزبونُ لم يُخطَر، **فلا شكوى تكشف المتجر.**
func TestMerchantBlocked_StillCountsAgainstTheMerchant(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)

	before, err := f.svc.MerchantViolations(ctx, f.pool, f.merchantID)
	if err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "", "merchant_closed"); err != nil {
		t.Fatalf("تعذّر الإبلاغ: %v", err)
	}
	after, err := f.svc.MerchantViolations(ctx, f.pool, f.merchantID)
	if err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}
	if after != before+1 {
		t.Errorf("العدّاد قبل=%d بعد=%d — **بابٌ مغلقٌ مرّ بلا أثر**", before, after)
	}

	var n int
	if err := f.pool.QueryRow(ctx,
		`SELECT count(*) FROM warnings WHERE order_id = $1`, f.orderID).Scan(&n); err != nil {
		t.Fatalf("تعذّرت قراءةُ الإنذارات: %v", err)
	}
	if n != 1 {
		t.Errorf("عددُ الإنذارات %d والمتوقّع 1 — **والإنذارُ سجلٌّ لا إشعار**", n)
	}
}

// TestMerchantBlocked_DriverIsStillCompensated **والسائقُ يُعوَّض عن مشواره.**
//
// قاد وعاد بلا شيء، **والذنبُ ليس ذنبَه** — وبالمعادلة نفسِها التي تعوّضه عند
// الفشل، فلا حسبةَ ثانيةً لواقعةٍ من الجنس نفسِه.
func TestMerchantBlocked_DriverIsStillCompensated(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)

	before := f.balance(t, f.driver)
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "", "merchant_closed"); err != nil {
		t.Fatalf("تعذّر الإبلاغ: %v", err)
	}
	// **٥٠٪ من أجرة التوصيل** — والمفتاحُ مبذورٌ في العُدّة.
	if got := f.balance(t, f.driver) - before; got != 5_000 {
		t.Fatalf("تعويضُ السائق %d والمتوقّع 5000 — **قاد وعاد بلا شيء**", got)
	}
}

// TestCustomerDoorFailure_StillCloses **وباب الزبون يبقى كما كان.**
//
// **ولا تُصلَح حالٌ بكسر أخرى**: البضاعةُ في الصندوق ولا بديل، **والطلبُ
// يُغلق ويُبلَّغ صاحبُه.**
func TestCustomerDoorFailure_StillCloses(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)

	rec := &recorder{}
	f.svc.SetNotifier(rec)

	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "", "customer_absent"); err != nil {
		t.Fatalf("الإفشال فشل: %v", err)
	}
	var status string
	var closed bool
	if err := f.pool.QueryRow(ctx,
		`SELECT status, closed_at IS NOT NULL FROM orders WHERE id = $1`, f.orderID).
		Scan(&status, &closed); err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلب: %v", err)
	}
	if status != "failed" || !closed {
		t.Fatalf("status=%q closed=%v — **وعند باب الزبون لا بديل**", status, closed)
	}
	// **ويُنادى صاحبُه** — بضاعتُه لم تصل، **وحقُّه أن يعلم.**
	if !rec.told(f.customer) {
		t.Fatal("لم يُنادَ الزبونُ — **وطلبُه أُغلق وهو لا يدري**")
	}
}
