package qa

// **كلُّ تقييمٍ بسطره** — المالك ٢٠٢٦-١٠-٢٥: الشاشةُ تعرض كلَّ طلبٍ قُيّم، والمنخفضَ وحدَه بفلتر،
// **والبابُ لمن يملك الدعم** — والزبونُ يُردّ.

import (
	"net/http"
	"testing"
)

func TestAdminRatingsList_ShowsEachRating(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	cust := h.Customer()
	oid, _ := activeOrderFor(t, h)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET customer_id = $2::uuid WHERE id = $1::uuid`, oid, cust.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_ratings (order_id, customer_id, platform_stars, driver_stars)
		VALUES ($1::uuid, $2::uuid, 2, 5) ON CONFLICT (order_id) DO UPDATE SET platform_stars = 2`, oid, cust.ID); err != nil {
		t.Fatal(err)
	}

	find := func(path string) map[string]any {
		r := h.GET(path, admin.Token)
		if r.Code != http.StatusOK {
			t.Fatalf("%s: %d / %s", path, r.Code, r.Err())
		}
		list, _ := r.JSON()["ratings"].([]any)
		for _, x := range list {
			if row, _ := x.(map[string]any); row["order_id"] == oid {
				return row
			}
		}
		return nil
	}
	row := find("/api/v1/admin/ratings/list")
	if row == nil || row["platform_stars"] != float64(2) {
		t.Fatalf("التقييمُ غائبٌ أو نجومُه خطأ: %v", row)
	}
	if find("/api/v1/admin/ratings/list?low=1") == nil {
		t.Fatal("تقييمٌ بنجمتين غائبٌ عن «الضعيفة بس»")
	}
	if got := h.GET("/api/v1/admin/ratings/list", cust.Token).Code; got != http.StatusForbidden {
		t.Fatalf("الزبونُ بلغ قائمةَ التقييمات: %d — والمنتظَر 403", got)
	}
}
