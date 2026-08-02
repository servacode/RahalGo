package orders_test

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestTreasury_ProfitIsWhatRemains ربحُ المنصة ما بقي بعد الجميع.
//
// **رقمٌ واحد يُثبت الحسبة كلَّها.** بضاعةٌ ١٠٠٬٠٠٠ وتوصيلٌ ١٠٬٠٠٠ وعمولةُ
// منصةٍ ١٠٪:
//
//	الزبون يدفع        ١١٠٬٠٠٠
//	− المتجر            ٩٠٬٠٠٠
//	− السائق             ٧٬٠٠٠   (٧٠٪ من رسم التوصيل)
//	− المندوب            ١٬٠٠٠   (١٠٪ من عمولة المنصة)
//	────────────────────────────
//	= الخزينة          ١٢٬٠٠٠
//
// ولو حُسب الربحُ بالجمع (عمولة + توصيل − سائق − مندوب) لكانت حسبةً ثانيةً
// بجانب الأولى — **وحسبتان تفترقان يوماً**. فهو بالطرح: **ما دفعه الزبون ناقصَ
// ما قُيّد فعلاً للأطراف.**
func TestTreasury_ProfitIsWhatRemains(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()

	// **المتجرُ بمالكٍ وإلّا لم يُقيَّد له شيء** — وعندها يبقى نصيبُه في
	// الخزينة فيظهر ربحٌ ١٠٢ ألفاً لا ١٢. (وهو ما كشفه هذا الاختبار أوّلَ
	// تشغيل: **الحسبةُ صحيحة والعُدّةُ ناقصة.**)
	owner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx,
		`UPDATE merchants SET owner_user_id = $2 WHERE id = $1`,
		f.merchantID, owner); err != nil {
		t.Fatalf("تعذّر ربط المالك: %v", err)
	}

	treasury := testdb.NewUser(t, f.pool, "admin")
	if _, err := f.pool.Exec(ctx, `
		UPDATE wallets SET is_treasury = true WHERE user_id = $1`, treasury); err != nil {
		t.Fatalf("تعذّر وسمُ الخزينة: %v", err)
	}
	store := settings.NewStore(f.pool)
	if err := store.SetInternal(ctx, "platform.treasury_user_id", treasury); err != nil {
		t.Fatalf("تعذّر ضبط الخزينة: %v", err)
	}
	t.Cleanup(func() {
		_ = store.SetInternal(context.Background(), "platform.treasury_user_id", "")
	})
	f.svc.SetSettings(store)

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
		f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل: %v", err)
	}

	if got := f.balance(t, f.rep); got != 1_000 {
		t.Fatalf("عمولة المندوب = %d، والمتوقّع 1000", got)
	}
	if got := f.balance(t, f.driver); got != 7_000 {
		t.Fatalf("أجر السائق = %d، والمتوقّع 7000", got)
	}
	if got := f.balance(t, owner); got != 81_000 {
		t.Fatalf("مستحقّ المتجر = %d، والمتوقّع 81000 (سعرُ الشراء − عمولة)", got)
	}
	// **٢١٬٠٠٠ لا ١٢٬٠٠٠ — والفرقُ تسعةُ آلافٍ صارت لنا لا للمتجر.**
	//
	// دخلَ ١١٠٬٠٠٠ (بضاعةٌ بسعر البيع + توصيل)، وخرج ٨١٬٠٠٠ للمتجر و٧٬٠٠٠
	// للسائق و١٬٠٠٠ للمندوب. **والفرقُ عن النموذج القديم هامشُنا**: كان
	// يُدفع للمتجر لأن مستحقَّه كان يُحسب من سعر البيع.
	if got := f.balance(t, treasury); got != 21_000 {
		t.Errorf("ربح المنصة = %d، والمتوقّع 21000", got)
	}
}

// TestTreasury_UnsetDoesNotBlockDelivery خزينةٌ غير مختارة لا تُعطّل تسليماً.
//
// **طلبٌ يُرفض لأن المالك لم يفتح صفحةَ الإعدادات خسارةٌ لا تُحتمَل.** يُسلَّم
// الطلبُ وتُقيَّد أنصبةُ الأطراف، ويبقى الدفترُ ناقصَ طرفٍ حتى تُختار.
func TestTreasury_UnsetDoesNotBlockDelivery(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()
	store := settings.NewStore(f.pool)
	if err := store.SetInternal(ctx, "platform.treasury_user_id", ""); err != nil {
		t.Fatalf("تعذّر ضبط الخزينة: %v", err)
	}
	f.svc.SetSettings(store)

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
		f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل بلا خزينة: %v", err)
	}
	if got := f.balance(t, f.driver); got != 7_000 {
		t.Errorf("أجر السائق = %d، والمتوقّع 7000", got)
	}
}
