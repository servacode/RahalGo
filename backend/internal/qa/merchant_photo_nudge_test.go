package qa

// **تذكيرُ الصباح بالأصناف بلا صور** (قرارُ المالك ٢٠٢٦-١١-١١): نحو العاشرة بدمشق، مرّةً في
// اليوم، لمن في متجره الفعّال أصنافٌ بلا صور — **ولا شيءَ إن كان العددُ صفراً، ولا للموقوف،
// ولا والمفتاحُ مطفأ.**

import (
	"strings"
	"testing"
	"time"
)

func pnAt(day, hour int) time.Time {
	loc := time.FixedZone("Damascus", 3*3600)
	return time.Date(2032, 5, day, hour, 5, 0, 0, loc)
}

func pnItems(t *testing.T, h *Harness, m *Merchant, n int, withPhoto bool) {
	t.Helper()
	for i := 0; i < n; i++ {
		it := h.NewItemPriced(m, 1000, 1000)
		if !withPhoto {
			continue
		}
		var mid string
		if err := h.Pool.QueryRow(ctxBG(), `
			INSERT INTO media (kind, path, thumb_path, width, height, bytes)
			VALUES ('menu_item', 'pn/x.jpg', 'pn/x_t.jpg', 10, 10, 1) RETURNING id::text`).Scan(&mid); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM media WHERE id = $1::uuid`, mid) })
		if _, err := h.Pool.Exec(ctxBG(), `UPDATE menu_items SET image_media_id = $2::uuid WHERE id = $1::uuid`, it.ID, mid); err != nil {
			t.Fatal(err)
		}
	}
}

func pnInbox(t *testing.T, h *Harness, userID string) []string {
	return obInbox(t, h, userID)
}

func TestPN01_SendsOncePerDayWithCount(t *testing.T) {
	h := New(t)
	h.Setting("merchants.photo_nudge_enabled", "true")
	m := h.Factory().Merchant()
	pnItems(t, h, m, 3, false)
	pnItems(t, h, m, 2, true)
	ctx := ctxBG()

	h.API.MerchantPhotoNudgeTickForTest(ctx, pnAt(1, 9)) // outside the hour
	if got := pnInbox(t, h, m.Owner.ID); len(got) != 0 {
		t.Fatalf("**أُرسل خارج ساعة الإرسال**: %v", got)
	}
	h.API.MerchantPhotoNudgeTickForTest(ctx, pnAt(1, 10))
	h.API.MerchantPhotoNudgeTickForTest(ctx, pnAt(1, 10)) // **نفسُ اليوم — لا ثانٍ**
	got := pnInbox(t, h, m.Owner.ID)
	if len(got) != 1 || !strings.Contains(got[0], "3") || !strings.Contains(got[0], "بلا صورة") {
		t.Fatalf("**يجب إشعارٌ واحدٌ بعدد ٣**: %v", got)
	}
	h.API.MerchantPhotoNudgeTickForTest(ctx, pnAt(2, 10)) // **اليومُ التالي — يصل**
	if got := pnInbox(t, h, m.Owner.ID); len(got) != 2 {
		t.Fatalf("**اليومُ التالي لم يصل**: %v", got)
	}
}

func TestPN02_SkipsZeroSuspendedAndOff(t *testing.T) {
	h := New(t)
	h.Setting("merchants.photo_nudge_enabled", "true")
	f := h.Factory()
	allPhotos := f.Merchant()
	pnItems(t, h, allPhotos, 2, true)
	susp := f.Merchant(MerchantSuspended())
	pnItems(t, h, susp, 2, false)
	closed := f.Merchant()
	pnItems(t, h, closed, 2, false)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET emergency_closed = true WHERE id = $1`, closed.ID); err != nil {
		t.Fatal(err)
	}
	h.API.MerchantPhotoNudgeTickForTest(ctxBG(), pnAt(3, 10))
	for name, id := range map[string]string{"كلُّها بصور": allPhotos.Owner.ID, "موقوف": susp.Owner.ID, "مغلق": closed.Owner.ID} {
		if got := pnInbox(t, h, id); len(got) != 0 {
			t.Fatalf("**وصل تذكيرٌ لـ%s**: %v", name, got)
		}
	}

	h.Setting("merchants.photo_nudge_enabled", "false")
	m := f.Merchant()
	pnItems(t, h, m, 1, false)
	if n := h.API.MerchantPhotoNudgeTickForTest(ctxBG(), pnAt(4, 10)); n != 0 {
		t.Fatalf("**أُرسل والمفتاحُ مطفأ** — %d", n)
	}
	if got := pnInbox(t, h, m.Owner.ID); len(got) != 0 {
		t.Fatalf("**وصل والمفتاحُ مطفأ**: %v", got)
	}
}
