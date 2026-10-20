package qa

import (
	"context"
	"testing"
)

// TestRepTarget_TransferredStoreCountsForNewRep **المتجرُ المنقولُ قبل اكتماله يُحسب للمندوب
// الجديد** (قرارُ المالك ٢٠٢٦-١٠-٢٠: «كلشي ننقله الى المندوب الجديد لازم ينحسب») — يكتمل
// عنده فيُحسب له ويُكافأ، **ولا شيءَ لمن فتحه.**
func TestRepTarget_TransferredStoreCountsForNewRep(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	f, opener, categoryID := targetFixture(t, h, 1)
	// **صنفٌ بصورته شرطٌ** — فلا يكتمل المتجرُ عند التحويل، ويُنقل قبل اكتماله.
	h.Setting("sales.min_items", "1")
	if code, _ := convertLeadFor(t, h, f, admin, opener, categoryID); code >= 400 {
		t.Fatalf("التحويل: %d", code)
	}
	var mid string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM merchants WHERE opened_by_rep_id = $1::uuid`,
		opener).Scan(&mid); err != nil {
		t.Fatalf("المتجر: %v", err)
	}
	if qualifiedOf(t, h, mid) {
		t.Fatal("المتجرُ اكتمل عند التحويل — والاختبارُ يحتاجه ناقصاً")
	}
	newRep := f.RepAccount()
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(), `DELETE FROM incentives WHERE user_id = $1::uuid`, newRep.ID)
	})
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET sales_rep_user_id = $2::uuid WHERE id = $1::uuid`,
		mid, newRep.ID); err != nil {
		t.Fatal(err)
	}

	// **ويكتمل عند الجديد**: ساعاتٌ وموقعٌ وصنفٌ بصورته.
	prepareMerchantsOf(t, h, opener)
	var media, sec string
	if err := h.Pool.QueryRow(ctxBG(), `INSERT INTO media (kind, path, thumb_path, width, height, bytes)
		VALUES ('menu_item', 'qa/x.jpg', 'qa/x_t.jpg', 10, 10, 10) RETURNING id::text`).Scan(&media); err != nil {
		t.Fatalf("صورة: %v", err)
	}
	if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM platform_sections LIMIT 1`).Scan(&sec); err != nil {
		t.Fatalf("قسم: %v", err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO menu_items
		(merchant_id, name, price, merchant_price, platform_section_id, image_media_id)
		VALUES ($1::uuid, $2, 100, 100, $3::uuid, $4::uuid)`, mid, uniq("صنف "), sec, media); err != nil {
		t.Fatalf("صنف: %v", err)
	}
	h.API.SweepRepQualify(context.Background())
	if !qualifiedOf(t, h, mid) {
		t.Fatal("اكتملت الشروطُ عند المندوب الجديد ولم يُحسب")
	}
	if got := targetFactsOf(t, h, newRep.ID).Rewards; got != 1 {
		t.Fatalf("مكافأةُ المندوب الجديد = %d — والمتوقَّع ١", got)
	}
	if got := targetFactsOf(t, h, opener).Rewards; got != 0 {
		t.Fatalf("مكافأةُ من فتحه = %d — والمتجرُ لم يعد له", got)
	}
}
