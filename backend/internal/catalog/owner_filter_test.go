package catalog_test

// **«متاجره» متاجرُه هو — لا متاجرُ المنصّة.**
//
// (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
//
// # العطبُ الذي كان
//
// **الاستعلامُ يعرف الاسمَ والتصنيفَ والحالَ والمندوبَ ولا يعرف المالك** —
// **وتبويبُ «متاجره» في ملفّ صاحب المتجر كان ينادي بلا شرطٍ أصلاً**
// (`query=`)، فيعرض **مئةَ متجرٍ من متاجر المنصّة** تحت عنوان «متاجرُ
// يملكها».
//
// **ولم يُرَ لأنّ في القاعدة متجراً واحداً** — فبدا صحيحاً. **وحين تصير
// عشرين يراها كلُّ صاحبِ متجرٍ في ملفّه**، والنظرةُ العامّةُ فوقَه تقول
// الصحيح: **رقمان متناقضان في صفحةٍ واحدة.**
//
// **وهذا يقيس بمتجرين لمالكين** — **ومتجرٌ واحدٌ لا يكشف شرطاً غائبا.**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestListMerchants_FiltersByOwner(t *testing.T) {
	pool := testdb.Pool(t)
	svc := catalog.NewService(pool, nil)
	ctx := context.Background()
	mine := testdb.NewUser(t, pool, "merchant")
	theirs := testdb.NewUser(t, pool, "merchant")
	rep := testdb.NewUser(t, pool, "sales")

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	mk := func(name, owner string, repID *string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO merchants (name, category_id, owner_user_id, sales_rep_user_id,
				commission_percent, location)
			VALUES ($1, $2, $3, $4, 10,
				ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography)
			RETURNING id::text`, name, categoryID, owner, repID).Scan(&id); err != nil {
			t.Fatalf("تعذّر متجرُ %q: %v", name, err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, id)
		})
		return id
	}
	mkID := mk("متجري", mine, &rep)
	mk("متجرُ غيري", theirs, nil)

	// ── بمالكه ────────────────────────────────────────────────────────
	page, err := svc.ListMerchants(ctx, "", "", "", "", mine, 1, 50)
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if page.Total != 1 || len(page.Merchants) != 1 {
		t.Fatalf("%d متجراً و%d مجموعاً — **وملفٌّ يعرض متاجرَ غيره يُقرأ ملكاً**",
			len(page.Merchants), page.Total)
	}
	if page.Merchants[0].ID != mkID {
		t.Fatalf("عُرض متجرٌ آخر: %s", page.Merchants[0].Name)
	}

	// ── وبلا مالكٍ يُعرض الجميع ───────────────────────────────────────
	//
	// **وهو الصوابُ لشاشة المتاجر** — **والعطبُ كان في مناداتها من ملفّ
	// شخصٍ بلا شرط.**
	all, err := svc.ListMerchants(ctx, "", "", "", "", "", 1, 50)
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if all.Total < 2 {
		t.Fatalf("بلا شرطٍ ردَّ %d — **وشاشةُ المتاجر تعرض الجميع**", all.Total)
	}

	// ── ومندوبُه يبقى كما كان ────────────────────────────────────────
	byRep, err := svc.ListMerchants(ctx, "", "", "", rep, "", 1, 50)
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if byRep.Total != 1 || byRep.Merchants[0].ID != mkID {
		t.Fatalf("مُرشِّحُ المندوب انكسر: %d — **وإصلاحُ شرطٍ يكسر أخاه**", byRep.Total)
	}
}
