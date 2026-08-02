package orders_test

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestMerchantPaidAtPickup المتجرُ يقبض عند خروج البضاعة لا عند التسليم.
//
// **قرارُ المالك**: المتجرُ ليس طرفاً في التوصيل — باع وسلّم وانتهى، وما يجري
// بعد ذلك بين المنصة والسائق والزبون لا يخصّه.
//
// وهذا الاختبارُ يفحص **ثلاثةَ أشياءَ لا واحداً**:
//
//  1. أن مستحقَّه يقع عند `picked_up`
//  2. أن **الخزينةَ تهبط تحت الصفر بينهما** — دفعت ولم تقبض بعد
//  3. أن المجموعَ عند التسليم **هو نفسُه قبل التقسيم** — فالتقسيمُ نقلُ توقيتٍ
//     لا تغييرُ حساب
func TestMerchantPaidAtPickup(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	owner, treasury := f.armTreasury(t)

	// ── الاستلامُ من المتجر ──────────────────────────────────────────────
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
		f.orderID, "picked_up", ""); err != nil {
		t.Fatalf("الاستلام فشل: %v", err)
	}
	if got := f.balance(t, owner); got != 90_000 {
		t.Fatalf("مستحقّ المتجر عند الاستلام = %d، والمتوقّع 90000", got)
	}
	// **نقديٌّ فلم تقبض المنصةُ شيئاً** — فالخزينةُ سالبةٌ بما دفعت.
	if got := f.balance(t, treasury); got != -90_000 {
		t.Errorf("الخزينة عند الاستلام = %d، والمتوقّع -90000", got)
	}
	if got := f.balance(t, f.driver); got != 0 {
		t.Errorf("السائق قبض قبل أن يسلّم: %d", got)
	}
	if got := f.balance(t, f.rep); got != 0 {
		t.Errorf("المندوب قبض قبل التسليم: %d", got)
	}

	// ── ثمّ الطريقُ والتسليم ─────────────────────────────────────────────
	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
			f.orderID, st, ""); err != nil {
			t.Fatalf("%s فشل: %v", st, err)
		}
	}

	if got := f.balance(t, owner); got != 90_000 {
		t.Errorf("قُيّد للمتجر مرّتين: %d", got)
	}
	if got := f.balance(t, f.driver); got != 7_000 {
		t.Errorf("أجر السائق = %d، والمتوقّع 7000", got)
	}
	if got := f.balance(t, f.rep); got != 1_000 {
		t.Errorf("عمولة المندوب = %d، والمتوقّع 1000", got)
	}
	// **١١٠٬٠٠٠ − (٩٠٬٠٠٠ + ٧٬٠٠٠ + ١٬٠٠٠) = ١٢٬٠٠٠** — نفسُ رقمٍ قبل التقسيم.
	if got := f.balance(t, treasury); got != 12_000 {
		t.Errorf("الخزينة بعد التسليم = %d، والمتوقّع 12000", got)
	}
}

// TestFailureAfterPickup_PlatformBearsIt الفشلُ بعد الاستلام خسارةٌ تُقيَّد وحدها.
//
// **وهذا ما أسقط سؤالَ «أاستردّ المتجرُ بضاعتَه؟»**: المنصةُ اشترت الطعامَ
// لحظةَ خروجه من المطبخ، **فهو ملكُها** — والخسارةُ تقع تلقائياً حيث يجب
// **بلا قرارٍ من أحد**.
func TestFailureAfterPickup_PlatformBearsIt(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	owner, treasury := f.armTreasury(t)

	for _, st := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
			f.orderID, st, ""); err != nil {
			t.Fatalf("%s فشل: %v", st, err)
		}
	}
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "الزبون لم يستلم"); err != nil {
		t.Fatalf("الإفشال فشل: %v", err)
	}

	// **المتجرُ يبقى بماله** — لا يخسر بمن أخطأ بعده.
	if got := f.balance(t, owner); got != 90_000 {
		t.Errorf("مستحقّ المتجر = %d، والمتوقّع 90000 يبقى كما هو", got)
	}
	if got := f.balance(t, f.rep); got != 0 {
		t.Errorf("المندوب قبض عن طلبٍ فشل: %d", got)
	}
	// **والخسارةُ الفعلية = ما دُفع للمتجر** — مكتوبةٌ في الدفتر بلا تقرير.
	if got := f.balance(t, treasury); got != -90_000 {
		t.Errorf("خسارة المنصة = %d، والمتوقّع -90000", got)
	}
}

// TestWalletOrder_TwoStagesSameTotal المحفظةُ تصل الرقمَ نفسه بمسارٍ آخر.
//
// **والفرقُ في التوقيت لا في النتيجة**: الزبونُ دفع عند الطلب، **فالخزينةُ
// موجبةٌ عند الاستلام** لا سالبة — ثمّ تنزل إلى نصيبها الحقيقيّ عند التسليم.
func TestWalletOrder_TwoStagesSameTotal(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 110_000)
	ctx := context.Background()
	owner, treasury := f.armTreasury(t)

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
		f.orderID, "picked_up", ""); err != nil {
		t.Fatalf("الاستلام فشل: %v", err)
	}
	// **قُبض ١١٠٬٠٠٠ ودُفع ٩٠٬٠٠٠** — فالباقي عندها ٢٠٬٠٠٠ حتى تدفع الأجور.
	if got := f.balance(t, treasury); got != 20_000 {
		t.Errorf("الخزينة عند الاستلام = %d، والمتوقّع 20000", got)
	}

	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
			f.orderID, st, ""); err != nil {
			t.Fatalf("%s فشل: %v", st, err)
		}
	}
	if got := f.balance(t, owner); got != 90_000 {
		t.Errorf("مستحقّ المتجر = %d", got)
	}
	if got := f.balance(t, treasury); got != 12_000 {
		t.Errorf("الخزينة = %d، والمتوقّع 12000 — نفسُ رقم النقديّ", got)
	}
}

// armTreasury يُهيّئ متجراً بمالكٍ وخزينةً مختارة.
func (f *fixture) armTreasury(t *testing.T) (owner, treasury string) {
	t.Helper()
	ctx := context.Background()
	owner = testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx,
		`UPDATE merchants SET owner_user_id = $2 WHERE id = $1`, f.merchantID, owner); err != nil {
		t.Fatalf("تعذّر ربط المالك: %v", err)
	}
	// **ولا يُوسَم العمودُ هنا**: المفتاحُ هو الحقيقة، والمحرّكُ يُصحّح العمودَ
	// عند أوّل قيد. **واختبارٌ يُهيّئ ما يُهيّئه النظامُ نفسُه يُخفي عطبَه.**
	treasury = testdb.NewUser(t, f.pool, "admin")
	store := settings.NewStore(f.pool)
	if err := store.SetInternal(ctx, "platform.treasury_user_id", treasury); err != nil {
		t.Fatalf("تعذّر ضبط الخزينة: %v", err)
	}
	t.Cleanup(func() {
		_ = store.SetInternal(context.Background(), "platform.treasury_user_id", "")
	})
	f.svc.SetSettings(store)
	return owner, treasury
}
