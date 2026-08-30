package orders_test

import (
	"context"
	"testing"
)

// TestRepTarget_GrantedOnDelivery **مكافأةُ هدف الشهر تُصرف للمندوب.**
//
// ══════════════════════════════════════════════════════════════════════
// **وكانت لا تُصرف قطّ**
// ══════════════════════════════════════════════════════════════════════
//
// `GrantTargetIfReached` تعرف دورَ `sales` منذ كُتبت: تقرأ
// `sales.monthly_target` و`sales.target_reward`، **ولها استعلامُها الخاصّ**
// يعدّ طلبات متاجره المسلَّمة.
//
// **ولم يكن في المنصّة كلِّها موضعٌ يناديها به** — السائقُ وحدَه
// (`transitions.go`). **فالشريطُ في تطبيق المندوب يمتلئ ويصير أخضرَ ويُكتب
// «تحقق الهدف — أحسنت»، ولا يدخل قرشٌ محفظتَه.**
//
// **ولا خطأَ يظهر ولا سطرَ في سجلّ**: عطبٌ لا يشتكي أطولُ عمراً من عطبٍ
// يصرخ. يظنّ المندوبُ المكافأةَ متأخّرةً أو أنّه أخطأ في العدّ.
//
// (كشفه جردُ تطبيق المندوب ٢٠٢٦-٠٨-٣٠، وأُصلح بإذن المالك.)
//
// # ولماذا هدفٌ من طلبٍ واحد
//
// **الفحصُ للنداء لا للعدّ**: عتبةٌ عاليةٌ تحتاج طلباتٍ كثيرةً في التهيئة
// **ولا تُثبت شيئاً زائداً** — إن نُودي المسارُ صحّ، وإن لم يُنادَ سقط.
func TestRepTarget_GrantedOnDelivery(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()

	// **هدفٌ وطلبٌ واحدٌ ومكافأةٌ معلومة.**
	//
	// **والاثنان شرطان**: `GrantTargetIfReached` تخرج صامتةً إن كان أحدُهما
	// صفراً — **وهو حالُ المنصّة اليوم**، فيُضبطان هنا صراحةً.
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES
		  ('sales.monthly_target', '1'::jsonb),
		  ('sales.target_reward', '5000'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("تعذّر ضبط الهدف والمكافأة: %v", err)
	}
	// **وتُمحى بعده** — القاعدةُ مشتركةٌ بين اختبارات الحزمة، **ومكافأةُ
	// هدفٍ تبقى مضبوطةً تدخل محافظَ اختباراتٍ تحسب العمولةَ وحدَها**
	// فتسقط ستّةٌ منها بأرقامٍ أكبرَ بخمسة آلاف.
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `
			DELETE FROM app_settings
			WHERE key IN ('sales.monthly_target', 'sales.target_reward')`)
	})

	before := f.balance(t, f.rep)

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
		f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل: %v", err)
	}

	// **والعمولةُ والمكافأةُ يدخلان معاً** — فيُقاس الفرقُ لا الرصيدُ
	// المطلق: **اختبارٌ يثبّت رقماً واحداً يسقط كلّما تبدّلت نسبةُ
	// العمولة**، وهي ليست موضوعَه.
	after := f.balance(t, f.rep)
	commission := int64(1_000) // ١٠٪ من هامشِ ١٠٬٠٠٠ — كما في `settlement_test`
	if got := after - before; got != commission+5_000 {
		t.Errorf("ما دخل محفظةَ المندوب = %d، والمتوقّع %d (عمولةٌ %d + مكافأةُ هدفٍ 5000)",
			got, commission+5_000, commission)
	}

	// **وقيدُ المكافأة يُكتب في `incentives` كذلك** — وهو ما يحمل الفهرسَ
	// الفريدَ الذي يمنع صرفَها مرّتين في الشهر.
	var n int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM incentives
		WHERE user_id = $1 AND for_target = true`, f.rep).Scan(&n); err != nil {
		t.Fatalf("تعذّرت قراءة الحوافز: %v", err)
	}
	if n != 1 {
		t.Errorf("قيودُ مكافأة الهدف = %d، والمتوقّع 1", n)
	}
}

// TestRepTarget_NotGrantedWhenRepIsBuyer **ولا يبلغ هدفَه بشرائه من متجره.**
//
// **وهي الحجّةُ نفسُها التي تمنعه من عمولة طلبه لنفسه**: من عدّ طلباتٍ
// اشتراها بيده صار بلوغُ الهدف بيده، **والمكافأةُ تُصرف على جلبِ بيعٍ لا
// على شراءٍ من نفسه.**
func TestRepTarget_NotGrantedWhenRepIsBuyer(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()

	if _, err := f.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES
		  ('sales.monthly_target', '1'::jsonb),
		  ('sales.target_reward', '5000'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("تعذّر ضبط الهدف والمكافأة: %v", err)
	}
	// **وتُمحى بعده** — القاعدةُ مشتركةٌ بين اختبارات الحزمة، **ومكافأةُ
	// هدفٍ تبقى مضبوطةً تدخل محافظَ اختباراتٍ تحسب العمولةَ وحدَها**
	// فتسقط ستّةٌ منها بأرقامٍ أكبرَ بخمسة آلاف.
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `
			DELETE FROM app_settings
			WHERE key IN ('sales.monthly_target', 'sales.target_reward')`)
	})
	// **والمندوبُ هو زبونُ هذا الطلب.**
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET customer_id = $2 WHERE id = $1`, f.orderID, f.rep); err != nil {
		t.Fatalf("تعذّر جعل المندوب زبوناً: %v", err)
	}

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
		f.orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم فشل: %v", err)
	}

	var n int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM incentives
		WHERE user_id = $1 AND for_target = true`, f.rep).Scan(&n); err != nil {
		t.Fatalf("تعذّرت قراءة الحوافز: %v", err)
	}
	if n != 0 {
		t.Errorf("صُرفت مكافأةُ هدفٍ على شرائه من متجره — القيود = %d، والمتوقّع 0", n)
	}
}
