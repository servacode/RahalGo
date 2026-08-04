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
	if got := f.balance(t, owner); got != 81_000 {
		t.Fatalf("مستحقّ المتجر عند الاستلام = %d، والمتوقّع 81000", got)
	}
	// **نقديٌّ فلم تقبض المنصةُ شيئاً** — فالخزينةُ سالبةٌ بما دفعت.
	if got := f.balance(t, treasury); got != -81_000 {
		t.Errorf("الخزينة عند الاستلام = %d، والمتوقّع -81000", got)
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

	if got := f.balance(t, owner); got != 81_000 {
		t.Errorf("قُيّد للمتجر مرّتين: %d", got)
	}
	if got := f.balance(t, f.driver); got != 7_000 {
		t.Errorf("أجر السائق = %d، والمتوقّع 7000", got)
	}
	if got := f.balance(t, f.rep); got != 1_000 {
		t.Errorf("عمولة المندوب = %d، والمتوقّع 1000", got)
	}
	// **١١٠٬٠٠٠ − (٨١٬٠٠٠ + ٧٬٠٠٠ + ١٬٠٠٠) = ٢١٬٠٠٠** — نفسُ رقمٍ قبل التقسيم.
	if got := f.balance(t, treasury); got != 21_000 {
		t.Errorf("الخزينة بعد التسليم = %d، والمتوقّع 21000", got)
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
	if got := f.balance(t, owner); got != 81_000 {
		t.Errorf("مستحقّ المتجر = %d، والمتوقّع 81000 يبقى كما هو", got)
	}
	if got := f.balance(t, f.rep); got != 0 {
		t.Errorf("المندوب قبض عن طلبٍ فشل: %d", got)
	}
	// **والخسارةُ الفعلية = ما دُفع للمتجر** — مكتوبةٌ في الدفتر بلا تقرير.
	//
	// **وهي سعرُ الشراء ناقصَ عمولتنا لا سعرُ البيع**: ما خسرناه ما دفعناه،
	// **والهامشُ الذي لم نقبضه ربحٌ فائتٌ لا خسارةٌ واقعة** — وقاعدةُ المالك
	// «الفعلية لا الافتراضية».
	if got := f.balance(t, treasury); got != -81_000 {
		t.Errorf("خسارة المنصة = %d، والمتوقّع -81000", got)
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
	// **قُبض ١١٠٬٠٠٠ ودُفع ٨١٬٠٠٠** — فالباقي عندها ٢٩٬٠٠٠ حتى تدفع الأجور.
	if got := f.balance(t, treasury); got != 29_000 {
		t.Errorf("الخزينة عند الاستلام = %d، والمتوقّع 29000", got)
	}

	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
			f.orderID, st, ""); err != nil {
			t.Fatalf("%s فشل: %v", st, err)
		}
	}
	if got := f.balance(t, owner); got != 81_000 {
		t.Errorf("مستحقّ المتجر = %d", got)
	}
	if got := f.balance(t, treasury); got != 21_000 {
		t.Errorf("الخزينة = %d، والمتوقّع 21000 — نفسُ رقم النقديّ", got)
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
	// **والخزينةُ تُوسَم في محفظتها** — صفةٌ في الحساب لا مفتاحٌ في الإعدادات.
	treasury = testdb.NewUser(t, f.pool, "admin")
	if _, err := f.pool.Exec(ctx,
		`UPDATE wallets SET is_treasury = true WHERE user_id = $1`, treasury); err != nil {
		t.Fatalf("تعذّر وسمُ الخزينة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`UPDATE wallets SET is_treasury = false WHERE user_id = $1`, treasury)
	})
	f.svc.SetSettings(settings.NewStore(f.pool))
	return owner, treasury
}
