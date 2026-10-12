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
