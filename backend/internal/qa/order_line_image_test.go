package qa

// ══════════════════════════════════════════════════════════════════════
// **صورةُ الصنف على سطر الطلب لقطةٌ لا قراءةٌ لاحقة** (قرارُ المالك ٢٠٢٦-١١-١٢) — هجرة 0546
// ══════════════════════════════════════════════════════════════════════
//
// السائقُ يرى «قميص» ولا يعرف أيَّ قميص. فتُنسخ صورةُ الصنف إلى سطره لحظةَ
// الطلب، **ولا يغيّرها تبديلُ المتجر لصورة الصنف بعدها.**

import (
	"net/http"
	"testing"
)

func lineImgMedia(t *testing.T, h *Harness, tag string) string {
	t.Helper()
	var id string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO media (kind, path, thumb_path, width, height, bytes)
		VALUES ('menu_item', $1, $2, 10, 10, 100) RETURNING id::text`,
		"test/line-"+tag+".jpg", "test/line-"+tag+"_t.jpg").Scan(&id); err != nil {
		t.Fatalf("وسيط: %v", err)
	}
	// **لا يبقى صفُّ وسيطٍ بلا ملفٍّ** — يكسر `mediacheck` على القاعدة نفسها.
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM media WHERE id = $1::uuid`, id)
	})
	return id
}

func lineImgOf(t *testing.T, h *Harness, r Res) (string, string) {
	t.Helper()
	if r.Code != http.StatusOK {
		t.Fatalf("قراءةُ الطلب: %s", r)
	}
	items, _ := r.JSON()["items"].([]any)
	if len(items) == 0 {
		t.Fatalf("لا بنود: %s", r)
	}
	it, _ := items[0].(map[string]any)
	u, _ := it["image_url"].(string)
	th, _ := it["image_thumb_url"].(string)
	return u, th
}

func TestOrderLineCarriesItemImageAndKeepsIt(t *testing.T) {
	h := New(t)
	m := newMDFxBare(t, h, "LINEIMG")
	first := lineImgMedia(t, h, "a")
	second := lineImgMedia(t, h, "b")
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE menu_items SET image_media_id = $2::uuid WHERE id = $1::uuid`, m.fx.Item.ID, first); err != nil {
		t.Fatal(err)
	}

	body := zoneBody(m.fx.Item, m.z.Lat, m.z.Lng)
	tok := h.Customer().Token
	r := h.POST("/api/v1/orders", tok, body)
	if r.Code != http.StatusOK && r.Code != http.StatusCreated {
		t.Fatalf("الطلب: %s", r)
	}
	oid, _ := r.JSON()["id"].(string)

	// ١ · السطرُ يحمل صورةَ الصنف لحظتَها.
	u, th := lineImgOf(t, h, h.GET("/api/v1/my/orders/"+oid, tok))
	if u != "/media/test/line-a.jpg" || th != "/media/test/line-a_t.jpg" {
		t.Fatalf("**الصورةُ لم تصل السطر**: %q %q", u, th)
	}

	// ٢ · وتبديلُ صورة الصنف بعدها لا يغيّر السطر.
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE menu_items SET image_media_id = $2::uuid WHERE id = $1::uuid`, m.fx.Item.ID, second); err != nil {
		t.Fatal(err)
	}
	u, _ = lineImgOf(t, h, h.GET("/api/v1/my/orders/"+oid, tok))
	if u != "/media/test/line-a.jpg" {
		t.Fatalf("**تبديلُ صورة الصنف غيّر سطرَ الطلب**: %q", u)
	}

	// ٣ · وقائمةُ الزبون تحمل الصورةَ كذلك، وردُّ المتجر.
	lst := h.GET("/api/v1/my/orders", tok)
	if lst.Code != http.StatusOK {
		t.Fatalf("القائمة: %s", lst)
	}
	ao := h.GET("/api/v1/admin/orders/"+oid, m.admin.Token)
	if au, _ := lineImgOf(t, h, ao); au != "/media/test/line-a.jpg" {
		t.Fatalf("ردُّ الإدارة بلا صورة: %q", au)
	}
	mo := h.GET("/api/v1/merchant/orders/"+oid, m.fx.Tok)
	if mu, _ := lineImgOf(t, h, mo); mu != "/media/test/line-a.jpg" {
		t.Fatalf("ردُّ المتجر بلا صورة: %q", mu)
	}
}
