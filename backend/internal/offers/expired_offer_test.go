package offers

// **عرضٌ انتهت مدّتُه لا يحجز صنفَه** — `OFFER-EXP`، ٢٠٢٦-٠٩-٣٠.
//
// # ما رآه المالك
//
// أنشأ عرضاً بالنسبة ثمّ بالمبلغ الثابت على صنفٍ واحد — **فسقط الاثنان
// بـ«تعذر الاتصال».** والشاشةُ عرضت الصنفَ متاحاً.
//
// # والتناقض
//
// **الشاشةُ تحجز ما هو «سارٍ» الآن** (`OfferStatus.discounting`)، **والقاعدةُ
// تحجز كلَّ ما علَمُه `active`** (`offers_one_live_per_item`) — **ولو انتهت
// مدّتُه.** **فعرضٌ لساعةٍ مضت يحجز صنفَه للأبد** حتّى يُنزَل بيد.
//
// # وسببٌ ثانٍ أمسكه هذا الفحص
//
// **المبلغُ الثابتُ سقط بقيد `offers_discount_complete`** — بقي من `0075`
// يشترط النسبة، **و`0168` لم تُرخِه.** فكلُّ عرضٍ بمبلغٍ رُفض في القاعدة
// (٥٠٠ ⇒ «تعذّر الاتصال»). **أصلحته `0169`**، ويحرسه فرعُ «مبلغ ثابت» أدناه.
//
// # والقاعدة — قرارُ المالك ٢٠٢٦-٠٩-٣٠
//
// «لازم نقدر نعمل عرض إيمت ما بدنا، ما إلو علاقة». **فالجديدُ يحلّ محلّ
// القائم أيّاً كان — منتهياً أو سارياً أو مجدولاً** — **ويبقى على الصنف عرضٌ
// قائمٌ واحد: الجديد.**

import (
	"context"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestCreate_ExpiredOfferDoesNotBlockItem(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := New(pool)
	actor := testdb.NewUser(t, pool, "admin")

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	var merchantID, menuSecID, platSecID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, active) VALUES ('قسمُ العرض المنتهي', true)
		RETURNING id`).Scan(&platSecID); err != nil {
		t.Fatalf("قسمُ منصّة: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, status)
		VALUES ('متجرُ العرض المنتهي', $1, 'active') RETURNING id`, categoryID).Scan(&merchantID); err != nil {
		t.Fatalf("متجر: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'الرئيسية')
		RETURNING id`, merchantID).Scan(&menuSecID); err != nil {
		t.Fatalf("قسم: %v", err)
	}
	newItem := func(name string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO menu_items (merchant_id, section_id, platform_section_id, name,
				merchant_price, price, available, approved)
			VALUES ($1, $2, $3, $4, 25000, 25000, true, true) RETURNING id`,
			merchantID, menuSecID, platSecID, name).Scan(&id); err != nil {
			t.Fatalf("صنف: %v", err)
		}
		return id
	}
	// **يُدرَج بيدٍ** — `Create` يرفض نهايةً مضت، **والعرضُ المنتهي في
	// الحياة وُلد سارياً ثمّ مضت مدّتُه.**
	seed := func(item string, starts, ends time.Time) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO offers (kind, title, menu_item_id, discount_percent, borne_by,
			                    starts_at, ends_at, active, created_by)
			VALUES ('discount', 'قديم', $1, 10, 'merchant', $2, $3, true, $4) RETURNING id`,
			item, starts, ends, actor).Scan(&id); err != nil {
			t.Fatalf("عرضٌ قديم: %v", err)
		}
		return id
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = pool.Exec(c, `DELETE FROM offers WHERE menu_item_id IN
			(SELECT id FROM menu_items WHERE merchant_id = $1)`, merchantID)
		_, _ = pool.Exec(c, `DELETE FROM menu_items WHERE merchant_id = $1`, merchantID)
		_, _ = pool.Exec(c, `DELETE FROM menu_sections WHERE id = $1`, menuSecID)
		_, _ = pool.Exec(c, `DELETE FROM merchants WHERE id = $1`, merchantID)
		_, _ = pool.Exec(c, `DELETE FROM platform_sections WHERE id = $1`, platSecID)
	})

	now := time.Now()
	by := ByMerchant
	create := func(item string, pct *int, amt *int64) error {
		ends := now.Add(24 * time.Hour)
		_, err := svc.Create(ctx, actor, Input{
			Title: "جديد", MenuItemID: &item, DiscountPercent: pct, DiscountAmount: amt,
			BorneBy: &by, EndsAt: &ends,
		}, func(p int64) int64 { return p })
		return err
	}
	pct := 15
	amt := int64(500)

	// ── ١ · المنتهي لا يحجز — بالنسبة وبالمبلغ كما جرّب المالك ──────────
	for _, tc := range []struct {
		name string
		pct  *int
		amt  *int64
	}{{"نسبة", &pct, nil}, {"مبلغ ثابت", nil, &amt}} {
		item := newItem("صنفٌ عرضُه انتهى — " + tc.name)
		old := seed(item, now.Add(-3*time.Hour), now.Add(-1*time.Hour))
		if err := create(item, tc.pct, tc.amt); err != nil {
			t.Fatalf("%s: عرضٌ منتهٍ حجز الصنف — ردَّ %v", tc.name, err)
		}
		var stillActive bool
		_ = pool.QueryRow(ctx, `SELECT active FROM offers WHERE id = $1`, old).Scan(&stillActive)
		if stillActive {
			t.Fatalf("%s: المنتهي بقي مرفوعاً بعد إنشاء الجديد", tc.name)
		}
	}

	// ── ٢ · السارِي والمجدولُ يُستبدلان كذلك — والجديدُ وحدَه قائم ──────
	for _, tc := range []struct {
		name         string
		starts, ends time.Time
	}{
		{"سارٍ", now.Add(-1 * time.Hour), now.Add(2 * time.Hour)},
		{"مجدول", now.Add(2 * time.Hour), now.Add(5 * time.Hour)},
	} {
		item := newItem("صنفٌ عرضُه " + tc.name)
		old := seed(item, tc.starts, tc.ends)
		if err := create(item, nil, &amt); err != nil {
			t.Fatalf("%s: العرضُ القائمُ منع الجديد — ردَّ %v", tc.name, err)
		}
		var oldActive bool
		var live int
		_ = pool.QueryRow(ctx, `SELECT active FROM offers WHERE id = $1`, old).Scan(&oldActive)
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM offers WHERE menu_item_id = $1 AND active`, item).Scan(&live)
		if oldActive || live != 1 {
			t.Fatalf("%s: القديمُ مرفوع=%v · القائمُ=%d — والمنتظَرُ الجديدُ وحدَه", tc.name, oldActive, live)
		}
	}
}
