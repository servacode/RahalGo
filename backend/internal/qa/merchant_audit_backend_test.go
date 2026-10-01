package qa

// **بندا الخادم من فحص تطبيق المتجر** — ٢٠٢٦-١٠-٠١.
//
// - **#4**: صنفٌ يُضاف من المتجر وضاع ردُّه فأُعيد **كان يُدرج مرّتين** —
//   بابُ المندوب محروسٌ منذ `DUP-LEAD` وبابُ المتجر لا.
// - **#5**: تقريرُ «اليوم» كان يقطع اليومَ بساعة UTC — **فطلبُ الواحدة ليلاً
//   بتوقيت دمشق يقع في أمس.** والتوصيلةُ كانت تُعدّ طلباً مبيعاً.

import (
	"testing"
	"time"
)

func TestMA04_MerchantItemCreateIsIdempotent(t *testing.T) {
	h := New(t)
	f := h.Factory()
	rep := h.NewUser("sales")
	m := f.Merchant(OwnedByRep(rep.ID))
	seed := h.NewItemFor(m, 1500)
	tok := h.TokenFor(m.Owner.ID, "merchant")

	body := map[string]any{"name": "صنفُ إعادة المتجر", "price": 700, "platform_section_id": seed.SectionID}
	path := "/api/v1/merchant/stores/" + m.ID + "/menu/items"
	key := "ma04-" + m.ID

	first := h.POSTKey(path, tok, key, body)
	if first.Code != 201 {
		t.Fatalf("الأوّل: %d / %s", first.Code, first.Err())
	}
	second := h.POSTKey(path, tok, key, body)
	if second.Code != 201 {
		t.Fatalf("الإعادة: %d / %s", second.Code, second.Err())
	}
	if first.JSON()["id"] != second.JSON()["id"] {
		t.Fatalf("**ردّان بمعرّفين** — والمنتظَرُ ردُّ الأوّل بعينه")
	}
	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM menu_items WHERE merchant_id = $1 AND name = 'صنفُ إعادة المتجر'`,
		m.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("**أُدرج الصنفُ %d مرّات بالمفتاح نفسِه**", n)
	}
}

func TestMA05_ReportTodayIsDamascusDay(t *testing.T) {
	h := New(t)
	item := h.NewItem(1000)
	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("ma05"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)

	// **الواحدةُ ليلاً بتوقيت دمشق** — وهي في UTC الأمس.
	loc, _ := time.LoadLocation("Asia/Damascus")
	now := time.Now().In(loc)
	early := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, loc)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET created_at = $2 WHERE id = $1`, oid, early); err != nil {
		t.Fatal(err)
	}

	var owner string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT owner_user_id::text FROM merchants WHERE id = $1`, item.MerchantID).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	tok := h.TokenFor(owner, "merchant")
	today := now.Format("2006-01-02")
	r := h.GET("/api/v1/merchant/stores/"+item.MerchantID+"/reports?from="+today+"&to="+today, tok)
	if r.Code != 200 {
		t.Fatalf("التقرير: %d / %s", r.Code, r.Err())
	}
	sum, _ := r.JSON()["summary"].(map[string]any)
	if got, _ := sum["orders"].(float64); got != 1 {
		t.Fatalf("**طلبُ الواحدة ليلاً لم يُعدّ في اليوم**: orders=%v — %s", sum["orders"], r)
	}
	days, _ := r.JSON()["days"].([]any)
	var onToday float64
	for _, d := range days {
		row, _ := d.(map[string]any)
		if row["date"] == today {
			onToday, _ = row["orders"].(float64)
		}
	}
	if onToday != 1 {
		t.Fatalf("**الرسمُ يضع الطلبَ في يومٍ آخر**: %v", days)
	}
}

// TestMA04b_OfferCreateIsIdempotent **عرضٌ أُعيد بالمفتاح نفسِه لا يُنشأ مرّتين.**
func TestMA04b_OfferCreateIsIdempotent(t *testing.T) {
	h := New(t)
	fx := newOfferFx(t, h, h.Factory(), 1000)
	body := map[string]any{"title": "x", "menu_item_id": fx.Item.ID, "discount_percent": 10}
	path := "/api/v1/merchant/stores/" + fx.M.ID + "/offers"
	key := "ma04b-" + fx.M.ID
	first := h.POSTKey(path, fx.Tok, key, body)
	second := h.POSTKey(path, fx.Tok, key, body)
	if first.Code != 200 || second.Code != 200 {
		t.Fatalf("%d / %d — %s", first.Code, second.Code, second.Err())
	}
	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM offers WHERE menu_item_id = $1`, fx.Item.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("**أُنشئ العرضُ %d مرّات بالمفتاح نفسِه**", n)
	}
}
