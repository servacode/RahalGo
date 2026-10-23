package qa

// **«التوصيل المجاني» من اللوحة إلى «لدي توصيلة»** — قرارُ المالك ٢٠٢٦-١٠-٢٣.
//
// العرضُ يُنشأ من بابه، **والمتجرُ يرى الأجرةَ صفراً قبل الطلب ويطلب بها**، والسائقُ
// بأجره كاملاً، **ومرّةٌ واحدةٌ لكلّ متجرٍ بالعرض**، واللوحةُ تعدّ ما انعطى.

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestFreeDelivery_MerchantOncePerStore(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ التوصيل المجاني")
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM free_delivery_campaigns`) })
	_, _ = h.Pool.Exec(ctxBG(), `UPDATE free_delivery_campaigns SET active = false`)

	// ── عرضٌ مفعَّلٌ لا يشمل شيئاً يُردّ ─────────────────────────────────
	now := time.Now().UTC()
	bad := h.POST("/api/v1/admin/free-delivery", m.admin.Token, map[string]any{
		"active": true, "starts_at": now.Add(-time.Hour), "ends_at": now.Add(24 * time.Hour), "kinds": []string{},
	})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("عرضٌ مفعَّلٌ بلا أنواع: %d — والمنتظَر 400", bad.Code)
	}
	made := h.POST("/api/v1/admin/free-delivery", m.admin.Token, map[string]any{
		"active": true, "starts_at": now.Add(-time.Hour), "ends_at": now.Add(24 * time.Hour),
		"kinds": []string{"merchant"},
	})
	if made.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ العرض: %d / %s", made.Code, made.Err())
	}

	// ── المتجرُ يرى الأجرةَ صفراً قبل الطلب ────────────────────────────
	q := h.GET(fmt.Sprintf("%s?lat=%f&lng=%f", m.url("/delivery-quote"), m.dropLat, m.dropLng), m.fx.Tok)
	if fee, _ := q.JSON()["fee"].(float64); fee != 0 {
		t.Fatalf("الأجرةُ في العرض %v — والمنتظَر 0: %s", q.JSON()["fee"], q)
	}
	if w, _ := q.JSON()["free_waived"].(float64); w != 500 {
		t.Fatalf("المعفى %v — والمنتظَر 500", q.JSON()["free_waived"])
	}

	// ── الأولى مجانيّةٌ والسائقُ بأجره، والثانيةُ بأجرتها ──────────────────
	first := m.create(t, h, "recipient", uniq("fd1"))
	var fee, cash, driverFee, waived int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT delivery_fee, cash_due, driver_fee, promo_delivery_waived
		FROM orders WHERE id = $1`, first).Scan(&fee, &cash, &driverFee, &waived); err != nil {
		t.Fatal(err)
	}
	if fee != 0 || cash != 0 || waived != 500 || driverFee != 450 {
		t.Fatalf("الأولى: أجرة %d · نقد %d · معفى %d · السائق %d — والمنتظَر 0 · 0 · 500 · 450",
			fee, cash, waived, driverFee)
	}
	second := m.create(t, h, "recipient", uniq("fd2"))
	if err := h.Pool.QueryRow(ctxBG(), `SELECT delivery_fee FROM orders WHERE id = $1`, second).Scan(&fee); err != nil {
		t.Fatal(err)
	}
	if fee != 500 {
		t.Fatalf("**الثانيةُ مجانيّةٌ أيضاً**: أجرة %d — والمرّةُ واحدةٌ لكلّ متجر", fee)
	}

	// ── واللوحةُ تعدّ ما انعطى ──────────────────────────────────────────
	g := h.GET("/api/v1/admin/free-delivery", m.admin.Token)
	c, _ := g.JSON()["campaign"].(map[string]any)
	if c == nil || c["claims"] != float64(1) || c["waived"] != float64(500) || c["running"] != true {
		t.Fatalf("ملخّصُ العرض: %s", g)
	}

	// ── والإطفاءُ يعيد الأجرة ──────────────────────────────────────────
	id, _ := c["id"].(string)
	off := h.PATCH("/api/v1/admin/free-delivery/"+id, m.admin.Token, map[string]any{
		"active": false, "starts_at": now.Add(-time.Hour), "ends_at": now.Add(24 * time.Hour),
		"kinds": []string{"merchant"},
	})
	if off.Code != http.StatusOK {
		t.Fatalf("الإطفاء: %d / %s", off.Code, off.Err())
	}
	q2 := h.GET(fmt.Sprintf("%s?lat=%f&lng=%f", m.url("/delivery-quote"), m.dropLat, m.dropLng), m.fx.Tok)
	if fee, _ := q2.JSON()["fee"].(float64); fee != 500 {
		t.Fatalf("بعد الإطفاء الأجرة %v — والمنتظَر 500", q2.JSON()["fee"])
	}
}
