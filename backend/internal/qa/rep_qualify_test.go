package qa

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **متجرُ المندوب يُحسب حين يجهز لا حين يُسجَّل** (قرارُ المالك ٢٠٢٦-١٠-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// ساعاتُ دوام · موقعٌ على الخريطة · أصنافٌ بصورها بعددٍ من اللوحة
// (`sales.min_items`). وبشهر اكتمالها يُعدّ، ومن اكتمل يبقى محسوباً.

// readyMerchantsOf **يُكمل شروطَ كلِّ متجرٍ فتحه المندوب** ثمّ يُمسَح —
// للاختبارات التي تسأل عن الصرف لا عن الشروط (و`sales.min_items` صفرٌ فيها).
func readyMerchantsOf(t *testing.T, h *Harness, repID string) {
	t.Helper()
	prepareMerchantsOf(t, h, repID)
	h.API.SweepRepQualify(context.Background())
}

func prepareMerchantsOf(t *testing.T, h *Harness, repID string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO merchant_hours (merchant_id, day_of_week, closed, open_time, close_time)
		SELECT m.id, 0, false, '09:00', '22:00' FROM merchants m
		 WHERE m.opened_by_rep_id = $1::uuid
		ON CONFLICT DO NOTHING`, repID); err != nil {
		t.Fatalf("ساعاتُ الدوام: %v", err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE merchants SET location = ST_SetSRID(ST_MakePoint(39.01, 35.95), 4326)::geography
		 WHERE opened_by_rep_id = $1::uuid AND location IS NULL`, repID); err != nil {
		t.Fatalf("الموقع: %v", err)
	}
}

func qualifiedOf(t *testing.T, h *Harness, merchantID string) bool {
	t.Helper()
	var ok bool
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT rep_qualified_at IS NOT NULL FROM merchants WHERE id = $1::uuid`,
		merchantID).Scan(&ok); err != nil {
		t.Fatalf("قراءةُ الاحتساب: %v", err)
	}
	return ok
}

func TestRepQualify_ThreeConditionsThenCountedOnce(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	f, repID, categoryID := targetFixture(t, h, 1)
	h.Setting("sales.min_items", "2")

	if code, _ := convertLeadFor(t, h, f, admin, repID, categoryID); code >= 400 {
		t.Fatalf("التحويل: %d", code)
	}
	var mid string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM merchants
		WHERE opened_by_rep_id = $1::uuid`, repID).Scan(&mid); err != nil {
		t.Fatalf("المتجر: %v", err)
	}
	sweep := func() { h.API.SweepRepQualify(context.Background()) }

	// ١ · مسجَّلٌ وحده ⇒ لا يُحسب ولا مكافأة.
	sweep()
	if qualifiedOf(t, h, mid) || targetFactsOf(t, h, repID).Rewards != 0 {
		t.Fatalf("**حُسب متجرٌ بلا شروط** — والقاعدة: يجهز أوّلاً")
	}

	// ٢ · ساعاتٌ وموقعٌ بلا أصناف ⇒ ينتظر.
	prepareMerchantsOf(t, h, repID)
	sweep()
	if qualifiedOf(t, h, mid) {
		t.Fatalf("**حُسب بلا أصناف** والحدُّ ٢")
	}

	// ٣ · صنفان أحدُهما بلا صورة ⇒ ينتظر.
	var media string
	if err := h.Pool.QueryRow(ctxBG(), `INSERT INTO media (kind, path, thumb_path, width, height, bytes)
		VALUES ('menu_item', 'qa/x.jpg', 'qa/x_t.jpg', 10, 10, 10) RETURNING id::text`).Scan(&media); err != nil {
		t.Fatalf("صورة: %v", err)
	}
	var sec string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM platform_sections LIMIT 1`).Scan(&sec); err != nil {
		t.Fatalf("قسم: %v", err)
	}
	add := func(img *string) {
		if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO menu_items
			(merchant_id, name, price, merchant_price, platform_section_id, image_media_id)
			VALUES ($1::uuid, $2, 100, 100, $3::uuid, $4::uuid)`, mid, uniq("صنف "), sec, img); err != nil {
			t.Fatalf("صنف: %v", err)
		}
	}
	add(&media)
	add(nil)
	sweep()
	if qualifiedOf(t, h, mid) {
		t.Fatalf("**صنفٌ بلا صورة عُدّ** — والأصنافُ بصورها")
	}

	// ٤ · الثاني بصورة ⇒ يُحسب ويُصرف مرّةً واحدة.
	add(&media)
	sweep()
	if !qualifiedOf(t, h, mid) {
		t.Fatalf("**اكتملت الشروط ولم يُحسب**")
	}
	sweep()
	if got := targetFactsOf(t, h, repID); got.Rewards != 1 || got.Sum != 5000 {
		t.Fatalf("مكافآتُ=%d بمجموع %d — والمتوقَّع واحدةٌ بـ5000", got.Rewards, got.Sum)
	}

	// ٥ · نقصت أصنافُه بعدها ⇒ يبقى محسوباً.
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE menu_items SET available = false WHERE merchant_id = $1::uuid`, mid); err != nil {
		t.Fatal(err)
	}
	sweep()
	if !qualifiedOf(t, h, mid) {
		t.Fatalf("**سُحب الاحتسابُ بعد نقص الأصناف** — والقرار: يظلّ محسوباً")
	}
}
