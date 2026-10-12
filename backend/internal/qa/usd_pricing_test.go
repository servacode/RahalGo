package qa

// ══════════════════════════════════════════════════════════════════════
//  **أسعارٌ بالدولار وسعرُ صرفٍ للمتجر** (طلبُ المالك ٢٠٢٦-١٠-١٢)
// ══════════════════════════════════════════════════════════════════════
//
// كلُّ صنفٍ يختار عملتَه، **والدولاريُّ يُحسب بالليرة لأقرب ٥** من سعر صرف
// المتجر، **ويتبعه متى تبدّل** — والليريُّ لا يُمسّ. **وقفزةٌ فوق ١٠٪ تُطلب
// تأكيداً بمثالٍ من الأصناف.**

import (
	"net/http"
	"strings"
	"testing"
)

func usdItem(t *testing.T, h *Harness, m mdFx, body map[string]any) string {
	t.Helper()
	body["platform_section_id"] = usdSection(t, h)
	r := h.POSTKey(m.url("/menu/items"), m.fx.Tok, uniq("usd"), body)
	if r.Code != http.StatusCreated {
		t.Fatalf("إضافةُ الصنف: %d %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	return id
}

func usdSection(t *testing.T, h *Harness) string {
	t.Helper()
	var id string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM platform_sections WHERE active ORDER BY sort_order LIMIT 1`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func usdPrice(t *testing.T, h *Harness, id string) (syp int64, usd *int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `SELECT merchant_price, price_usd_cents FROM menu_items WHERE id = $1`, id).Scan(&syp, &usd); err != nil {
		t.Fatal(err)
	}
	return
}

func TestUSD01_DollarItemFollowsRatePoundItemStays(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "USD01")

	// **بلا سعر صرفٍ لا صنفَ دولاريّ.**
	if r := h.POSTKey(m.url("/menu/items"), m.fx.Tok, uniq("usd0"), map[string]any{
		"name": "شاحن", "price_currency": "USD", "price_usd": 3, "platform_section_id": usdSection(t, h),
	}); r.Err() != "exchange_rate_required" {
		t.Fatalf("**صنفٌ دولاريٌّ بلا سعر صرف** — %d %s", r.Code, r.Err())
	}

	if r := h.PATCH(m.url("/exchange-rate"), m.fx.Tok, map[string]any{"rate": 115}); r.Code != http.StatusOK {
		t.Fatalf("سعرُ الصرف: %d %s", r.Code, r.Err())
	}
	phone := usdItem(t, h, m, map[string]any{
		"name": "موبايل", "price_currency": "USD", "price_usd": 1.5,
		"modifiers": []map[string]any{{"name": "الذاكرة", "min_select": 1, "max_select": 1,
			"options": []map[string]any{{"name": "128", "price_delta_usd": 0}, {"name": "256", "price_delta_usd": 0.5}}}},
	})
	cover := usdItem(t, h, m, map[string]any{"name": "كفر", "price": 200})

	// 1.5 × 115 = 172.5 ⇒ 175 · 0.5 × 115 = 57.5 ⇒ 60
	if syp, usd := usdPrice(t, h, phone); syp != 175 || usd == nil || *usd != 150 {
		t.Fatalf("**الدولاريُّ لم يُحسب لأقرب ٥**: %d %v", syp, usd)
	}
	var delta int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT o.price_delta FROM modifier_options o JOIN modifier_groups g ON g.id = o.group_id
		WHERE g.item_id = $1 AND o.name = '256'`, phone).Scan(&delta)
	if delta != 60 {
		t.Fatalf("**الإضافةُ الدولاريّةُ لم تُحسب**: %d", delta)
	}

	// **قفزةٌ فوق ١٠٪ تُطلب تأكيداً** — ولا يتغيّر شيءٌ قبله.
	if r := h.PATCH(m.url("/exchange-rate"), m.fx.Tok, map[string]any{"rate": 11500}); r.Code != http.StatusConflict || r.Err() != "exchange_rate_jump" {
		t.Fatalf("**قفزةٌ مئويّةٌ مرّت بلا تأكيد** — %d %s", r.Code, r.Err())
	}
	if syp, _ := usdPrice(t, h, phone); syp != 175 {
		t.Fatalf("**تبدّل السعرُ قبل التأكيد**: %d", syp)
	}

	// 1.5 × 120 = 180 · 0.5 × 120 = 60 — والكفرُ ٢٠٠ كما هو.
	if r := h.PATCH(m.url("/exchange-rate"), m.fx.Tok, map[string]any{"rate": 120}); r.Code != http.StatusOK {
		t.Fatalf("سعرُ الصرف الجديد: %d %s", r.Code, r.Err())
	}
	if syp, _ := usdPrice(t, h, phone); syp != 180 {
		t.Fatalf("**الدولاريُّ لم يتبع سعرَ الصرف**: %d", syp)
	}
	if syp, usd := usdPrice(t, h, cover); syp != 200 || usd != nil {
		t.Fatalf("**الليريُّ مُسّ**: %d %v", syp, usd)
	}

	// **وقائمةُ المتجر تقول عملةَ كلّ صنف.**
	menu := h.GET(m.url("/menu"), m.fx.Tok)
	if menu.Code != http.StatusOK || !containsBytes(menu.Body, `"price_currency":"USD"`) || !containsBytes(menu.Body, `"merchant_price_usd":1.5`) {
		t.Fatalf("**القائمةُ بلا عملة الصنف**: %s", menu)
	}
}

func containsBytes(b []byte, s string) bool {
	return len(b) > 0 && strings.Contains(string(b), s)
}

// TestUSD02_CustomerSeesDollarsPaysPounds — **الخيار ١** (٢٠٢٦-١٠-١٢): متجرٌ اختار
// العرضَ بالدولار يرى زبونُه `price_usd` بجانب الليرة، **ويبقى `price` بالليرة
// هو ما يُدفع**؛ ومتجرٌ بالليرة بلا دولار.
func TestUSD02_CustomerSeesDollarsPaysPounds(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "USD02")
	if r := h.PATCH(m.url("/exchange-rate"), m.fx.Tok, map[string]any{"rate": 100}); r.Code != http.StatusOK {
		t.Fatalf("سعرُ الصرف: %d %s", r.Code, r.Err())
	}
	item := usdItem(t, h, m, map[string]any{"name": "سماعة", "price": 500})
	cust := h.NewUser("customer")

	r := h.GET("/api/v1/public/items/"+item, cust.Token)
	if r.Code != http.StatusOK || strings.Contains(string(r.Body), `"price_usd"`) {
		t.Fatalf("**متجرٌ بالليرة ظهر بالدولار**: %s", r)
	}
	if s := h.PATCH(m.url("/settings"), m.fx.Tok, map[string]any{"display_currency": "USD"}); s.Code != http.StatusOK {
		t.Fatalf("عملةُ العرض: %d %s", s.Code, s.Err())
	}
	r = h.GET("/api/v1/public/items/"+item, cust.Token)
	it, _ := r.JSON()["item"].(map[string]any)
	price, _ := it["price"].(float64)
	usd, _ := it["price_usd"].(float64)
	if price <= 0 || usd <= 0 {
		t.Fatalf("**بلا سعرٍ بالدولار أو بالليرة**: %s", r)
	}
	if want := float64(int64(price/100*100+0.5)) / 100; usd != want {
		t.Fatalf("**الدولارُ ليس الليرةَ ÷ سعرِ الصرف**: price=%v usd=%v want=%v", price, usd, want)
	}
}

// TestUSD03_RepReadsRateNeverWrites — **سعرُ الصرف لصاحب المتجر وحدَه** (قرارُ المالك
// ٢٠٢٦-١٠-١٢): المندوبُ يقرؤه من تفصيل متجره، **ولا بابَ له يكتبه.**
func TestUSD03_RepReadsRateNeverWrites(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "USD03")
	if r := h.PATCH(m.url("/exchange-rate"), m.fx.Tok, map[string]any{"rate": 110}); r.Code != http.StatusOK {
		t.Fatalf("سعرُ الصرف من المتجر: %d %s", r.Code, r.Err())
	}
	if r := h.PATCH("/api/v1/rep/stores/"+m.fx.M.ID+"/exchange-rate", m.fx.RepTok, map[string]any{"rate": 120}); r.Code < 400 {
		t.Fatalf("**المندوبُ كتب سعرَ الصرف**: %d", r.Code)
	}
	d := h.GET("/api/v1/rep/merchants/"+m.fx.M.ID, m.fx.RepTok)
	if d.Code != http.StatusOK || !strings.Contains(string(d.Body), `"exchange_rate":110`) {
		t.Fatalf("**المندوبُ لا يرى سعرَ صرف متجره**: %s", d)
	}
}

// TestUSD04_DollarShownAsWrittenPoundRoundedTo5c — ملاحظتا المالك (٢٠٢٦-١٠-١٢):
// صنفٌ كُتب ١٫٧٥$ يراه الزبونُ ١٫٧٥$ لا ١٫٧٤$، **وصنفٌ ليريٌّ بـ١٧٥ يُعرض ١٫٥٠$**
// (لأقرب ٥ سنتات)، **وإضافتُه الدولاريّةُ تُعرض بالدولار.**
func TestUSD04_DollarShownAsWrittenPoundRoundedTo5c(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "USD04")
	if r := h.PATCH(m.url("/exchange-rate"), m.fx.Tok, map[string]any{"rate": 115}); r.Code != http.StatusOK {
		t.Fatalf("سعرُ الصرف: %d %s", r.Code, r.Err())
	}
	if r := h.PATCH(m.url("/settings"), m.fx.Tok, map[string]any{"display_currency": "USD"}); r.Code != http.StatusOK {
		t.Fatalf("عملةُ العرض: %d %s", r.Code, r.Err())
	}
	burger := usdItem(t, h, m, map[string]any{"name": "برغر", "price_currency": "USD", "price_usd": 1.75,
		"modifiers": []map[string]any{{"name": "الحجم", "min_select": 1, "max_select": 1,
			"options": []map[string]any{{"name": "عادي", "price_delta_usd": 0}, {"name": "دبل", "price_delta_usd": 0.5}}}}})
	wrap := usdItem(t, h, m, map[string]any{"name": "شاورما", "price": 175})
	cust := h.NewUser("customer")

	r := h.GET("/api/v1/public/items/"+burger, cust.Token)
	it, _ := r.JSON()["item"].(map[string]any)
	if usd, _ := it["price_usd"].(float64); usd != 1.75 {
		t.Fatalf("**الدولاريُّ لم يُعرض بدولاره**: %v — %s", it["price_usd"], r)
	}
	if !strings.Contains(string(r.Body), `"price_delta_usd":0.5`) {
		t.Fatalf("**الإضافةُ الدولاريّةُ لم تُعرض بالدولار**: %s", r)
	}
	r = h.GET("/api/v1/public/items/"+wrap, cust.Token)
	it, _ = r.JSON()["item"].(map[string]any)
	if usd, _ := it["price_usd"].(float64); usd != 1.5 {
		t.Fatalf("**الليريُّ لم يُدوَّر لأقرب ٥ سنتات**: %v", it["price_usd"])
	}
}
