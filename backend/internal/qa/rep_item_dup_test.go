package qa

// **صنفٌ أُعيد إرسالُه بالمفتاح نفسِه لا يُدرج مرّتين** — `DUP-LEAD`، ٢٠٢٦-٠٩-٣٠.
//
// **رُئي على جهاز المالك**: عميلٌ أُرسل وانقطعت الشبكةُ قبل الردّ، ثمّ أُعيد
// — **فسُجِّل مرّتين.** والعميلُ محروسٌ في الخادم وكان المفتاحُ لا يثبت في
// التطبيق. **وإضافةُ الصنف لم يكن لها حارسٌ في الخادم أصلاً** — فالتعثّرُ
// نفسُه يُكرّر الصنف ولو ثبت المفتاح.

import (
	"testing"
)

func TestDUPLEAD_RepItemCreateIsIdempotent(t *testing.T) {
	h := New(t)
	f := h.Factory()
	rep := h.NewUser("sales")
	m := f.Merchant(OwnedByRep(rep.ID))
	seed := h.NewItemFor(m, 1500) // لقسم المنصّة وحدَه

	body := map[string]any{"name": "صنفُ الإعادة", "price": 700, "platform_section_id": seed.SectionID}
	path := "/api/v1/rep/stores/" + m.ID + "/menu/items"
	key := "dup-item-" + m.ID

	first := h.POSTKey(path, rep.Token, key, body)
	if first.Code != 201 {
		t.Fatalf("الأوّل: %d / %s", first.Code, first.Err())
	}
	second := h.POSTKey(path, rep.Token, key, body)
	if second.Code != 201 {
		t.Fatalf("الإعادة: %d / %s — والمنتظَرُ ردُّ الأوّل بعينه", second.Code, second.Err())
	}

	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM menu_items WHERE merchant_id = $1 AND name = 'صنفُ الإعادة'`,
		m.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("**أُدرج الصنفُ %d مرّات بالمفتاح نفسِه** — والمنتظَرُ مرّة", n)
	}
}
