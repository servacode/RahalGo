package orders_test

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestTwoSources_EachPaidItsOwn لكلِّ مصدرٍ مستحقُّه وعمولتُه.
//
// # ما يمسكه
//
// كانت التسويةُ تقرأ `orders.merchant_id` وحدَه — **فتدفع لصاحب المحطّة الأولى
// ثمنَ بضاعةِ الثاني**: يربح من لم يبع، **ويُحرم من باع.**
//
// **والعمولةُ لكلِّ متجرٍ بنسبته**: متجرٌ اتُّفق معه على ١٠٪ وآخرُ على ٢٠٪ **لا
// تجمعهما نسبةٌ واحدة** — ونسبةُ صاحب المحطّة الأولى ليست عقداً على غيره.
//
// # الحسبة
//
//	الأوّل   :  ٦٠٬٠٠٠ شراءً · ١٠٪ = ٦٬٠٠٠ عمولة · ٥٤٬٠٠٠ مستحقّاً
//	الثاني   :  ٤٠٬٠٠٠ شراءً · ٢٠٪ = ٨٬٠٠٠ عمولة · ٣٢٬٠٠٠ مستحقّاً
//	العمولة  :  ١٤٬٠٠٠ — **مجموعُ الاثنتين لا نسبةٌ واحدةٌ على المجموع**
func TestTwoSources_EachPaidItsOwn(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	owner, _ := f.armTreasury(t)

	// **متجرٌ ثانٍ بنسبةٍ مختلفة** — كي يُرى أن كلاً يُحاسَب بعقده.
	var categoryID string
	if err := f.pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	owner2 := testdb.NewUser(t, f.pool, "merchant")
	var merchant2 string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, owner_user_id)
		VALUES ('المصدرُ الثاني', $1, 20, $2) RETURNING id`, categoryID, owner2).
		Scan(&merchant2); err != nil {
		t.Fatalf("تعذّر إنشاء متجر ثانٍ: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchant2)
	})

	// الزراعةُ تضع بنداً واحداً بـ100000 بيعاً و90000 شراءً — نستبدله ببندين.
	if _, err := f.pool.Exec(ctx, `DELETE FROM order_items WHERE order_id = $1`, f.orderID); err != nil {
		t.Fatalf("تعذّر مسح البنود: %v", err)
	}
	for _, b := range []struct {
		merchant string
		cost     int64
	}{{f.merchantID, 60_000}, {merchant2, 40_000}} {
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO order_items (order_id, merchant_id, name, unit_price, merchant_price, qty, options)
			VALUES ($1, $2, 'صنف', $3, $3, 1, '[]'::jsonb)`,
			f.orderID, b.merchant, b.cost); err != nil {
			t.Fatalf("تعذّر إنشاء بند: %v", err)
		}
	}

	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
		f.orderID, "picked_up", ""); err != nil {
		t.Fatalf("الاستلام فشل: %v", err)
	}

	if got := f.balance(t, owner); got != 54_000 {
		t.Errorf("مستحقّ المصدر الأوّل = %d، والمتوقّع 54000 (60000 − 10%%)", got)
	}
	if got := f.balance(t, owner2); got != 32_000 {
		t.Errorf("مستحقّ المصدر الثاني = %d، والمتوقّع 32000 (40000 − 20%%)", got)
	}
	// **والعمولةُ مجموعُ الاثنتين** — لا نسبةٌ واحدةٌ على المجموع.
	//
	// لو حُسبت بنسبة الأوّل وحدَها لَكانت ١٠٬٠٠٠، **وبنسبة الثاني ٢٠٬٠٠٠** —
	// وكلاهما رقمٌ لم يتّفق عليه أحد.
	if got := f.platformCommission(t); got != 14_000 {
		t.Errorf("عمولة المنصة = %d، والمتوقّع 14000 (6000 + 8000)", got)
	}
}

// TestSources_CapEnforced سقفُ المصادر يُفرض في الخادم.
//
// **والسلّةُ لا تملك أن تمنع**: أخفينا عنها المصادرَ عمداً. **والخادمُ يعرف،
// وهو الموضعُ الذي لا يُلتفّ عليه.**
func TestSources_CapEnforced(t *testing.T) {
	f := setup(t, "pending", 100_000, 10_000, 0)
	ctx := context.Background()

	var categoryID string
	if err := f.pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	// **ثلاثةُ مصادر** — والسقفُ اثنان.
	ids := []string{}
	for _, name := range []string{"مصدرٌ ١", "مصدرٌ ٢", "مصدرٌ ٣"} {
		var mid, secID, itemID string
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO merchants (name, category_id, commission_percent, status)
			VALUES ($1, $2, 10, 'active') RETURNING id`, name, categoryID).Scan(&mid); err != nil {
			t.Fatalf("تعذّر إنشاء متجر: %v", err)
		}
		t.Cleanup(func() {
			_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, mid)
		})
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'الرئيسية')
			RETURNING id`, mid).Scan(&secID); err != nil {
			t.Fatalf("تعذّر إنشاء قسم: %v", err)
		}
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO menu_items (merchant_id, section_id, name, merchant_price, price, available)
			VALUES ($1, $2, $3, 10000, 10000, true) RETURNING id`,
			mid, secID, name).Scan(&itemID); err != nil {
			t.Fatalf("تعذّر إنشاء صنف: %v", err)
		}
		ids = append(ids, itemID)
	}

	items := func(n int) []orders.ItemInput {
		out := []orders.ItemInput{}
		for _, id := range ids[:n] {
			out = append(out, orders.ItemInput{MenuItemID: id, Qty: 1})
		}
		return out
	}

	// **مصدران يمرّان.**
	src, err := f.svc.SourcesOf(ctx, items(2))
	if err != nil {
		t.Fatalf("مصدران رُدّا: %v", err)
	}
	if len(src.IDs) != 2 {
		t.Errorf("عددُ المصادر = %d، والمتوقّع 2", len(src.IDs))
	}

	// **وثلاثةٌ تُعرف على أنها ثلاثة** — والردُّ في `Create`.
	src, err = f.svc.SourcesOf(ctx, items(3))
	if err != nil {
		t.Fatalf("ثلاثةٌ تعذّرت قراءتها: %v", err)
	}
	if len(src.IDs) != 3 {
		t.Errorf("عددُ المصادر = %d، والمتوقّع 3", len(src.IDs))
	}
}

// TestTwoSources_RefundReversesEach الاستردادُ يعكس لكلِّ مطبخٍ ما قُيّد له هو.
//
// # الخللُ الذي كشفته التجربةُ الحيّة (طلب #1009)
//
// كان العكسُ يجمع قيودَ `merchant_earning` كلَّها ثمّ **يخصم المجموعَ من صاحب
// المحطّة الأولى وحدَه.** ففي طلبٍ من مطبخين:
//
//	الأوّل  :  قبض ٤٣٬٢٠٠  ثمّ خُصم منه ٥٠٬٨٠٠  →  **رصيدٌ سالبٌ بما لم يقبض**
//	الثاني  :  قبض  ٧٬٦٠٠  ولم يُعكس عنه شيء   →  **مالٌ على طلبٍ مُسترَدّ**
//
// **وهي علّةُ التسوية الأمامية نفسُها في مرآتها**: أُصلحت هناك ونُسيت هنا.
//
// **ولم يكشفها اختبار**: كلُّ اختبارات الاسترداد كانت بمصدرٍ واحد، **والمصدرُ
// الواحد يُخفي الخلل تماماً** — المجموعُ هو نصيبُه.
func TestTwoSources_RefundReversesEach(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	owner, _ := f.armTreasury(t)

	var categoryID string
	if err := f.pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	owner2 := testdb.NewUser(t, f.pool, "merchant")
	var merchant2 string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, owner_user_id)
		VALUES ('المصدرُ الثاني للاسترداد', $1, 10, $2) RETURNING id`, categoryID, owner2).
		Scan(&merchant2); err != nil {
		t.Fatalf("تعذّر إنشاء متجر ثانٍ: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchant2)
	})

	if _, err := f.pool.Exec(ctx, `DELETE FROM order_items WHERE order_id = $1`, f.orderID); err != nil {
		t.Fatalf("تعذّر مسح البنود: %v", err)
	}
	for _, b := range []struct {
		merchant string
		cost     int64
	}{{f.merchantID, 60_000}, {merchant2, 40_000}} {
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO order_items (order_id, merchant_id, name, unit_price, merchant_price, qty, options)
			VALUES ($1, $2, 'صنف', $3, $3, 1, '[]'::jsonb)`,
			f.orderID, b.merchant, b.cost); err != nil {
			t.Fatalf("تعذّر إنشاء بند: %v", err)
		}
	}

	for _, st := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
			f.orderID, st, ""); err != nil {
			t.Fatalf("%s فشل: %v", st, err)
		}
	}
	if got := f.balance(t, owner); got != 54_000 {
		t.Fatalf("مستحقّ الأوّل قبل الاسترداد = %d، والمتوقّع 54000", got)
	}
	if got := f.balance(t, owner2); got != 36_000 {
		t.Fatalf("مستحقّ الثاني قبل الاسترداد = %d، والمتوقّع 36000", got)
	}

	if _, err := f.svc.Transition(ctx, f.driver, []string{"admin"},
		f.orderID, "refunded", "تجربة"); err != nil {
		t.Fatalf("الاسترداد فشل: %v", err)
	}

	// **وكلٌّ يعود إلى صفره** — لا الأوّلُ يهبط تحته ولا الثاني يبقى فوقه.
	if got := f.balance(t, owner); got != 0 {
		t.Errorf("المصدر الأوّل بعد الاسترداد = %d، والمتوقّع 0 — **خُصم منه ما لم يقبض**", got)
	}
	if got := f.balance(t, owner2); got != 0 {
		t.Errorf("المصدر الثاني بعد الاسترداد = %d، والمتوقّع 0 — **بقي بمالِ طلبٍ مُسترَدّ**", got)
	}
}
