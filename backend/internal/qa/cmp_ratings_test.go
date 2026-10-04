package qa

// **الزبونُ يقيّم السائقَ والمنصّة فقط** (قرارُ المالك ٢٠٢٦-١٠-٠٤: «ما يعرف
// المتجرَ ليقيّمه»).
//
//   - صاحبُ المتجر لا يُخبَر بتقييمٍ ليس فيه نجمةٌ له — السائقُ وحدَه يُخبَر.
//   - وملفُّ صاحب المتجر في اللوحة لا يعرض نجمةَ المنصّة «واردةً» إليه.

import (
	"testing"
	"time"
)

func TestCMP_CustomerRatingNotTheMerchants(t *testing.T) {
	h := New(t)
	treasury(t, h)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	var ownerID string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT owner_user_id::text FROM merchants WHERE id = $1::uuid`,
		item.MerchantID).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}

	got := h.POST("/api/v1/orders/"+oid+"/rating", cust.Token,
		map[string]any{"platform_stars": 1, "driver_stars": 5})
	if got.Code >= 400 {
		t.Fatalf("التقييم: %s", got)
	}

	count := func(uid string) int {
		var n int
		_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
			WHERE user_id = $1::uuid AND kind = 'rating' AND entity_id = $2`, uid, oid).Scan(&n)
		return n
	}
	deadline := time.Now().Add(5 * time.Second)
	for count(drv.ID) == 0 && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if count(drv.ID) == 0 {
		t.Fatal("السائقُ لم يُخبَر بتقييمه")
	}
	if n := count(ownerID); n != 0 {
		t.Fatalf("**صاحبُ المتجر أُخبر بتقييمٍ ليس له** (%d) — الزبونُ لا يقيّم المتجر", n)
	}

	admin := h.NewUser("admin")
	fb := h.GET("/api/v1/admin/users/"+ownerID+"/feedback", admin.Token)
	if fb.Code >= 400 {
		t.Fatalf("ملفُّ صاحب المتجر: %s", fb)
	}
	if v, ok := fb.JSON()["ratings_received_count"].(float64); !ok || v != 0 {
		t.Fatalf("**نجمةُ المنصّة عُدّت واردةً لصاحب المتجر**: %v", fb.JSON()["ratings_received_count"])
	}
}
