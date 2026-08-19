package qa

// **الطبقةُ الثالثة — النزاهةُ الماليّة والتحقّقُ من المُدخَل.**
//
// المعرّفات: `FIN-*` · `VAL-*` · الوسم: `@api @security @critical @release`
//
// # القاعدةُ الواحدة
//
// **كلُّ رقمٍ يُحاسَب عليه يُحسب في الخادم** — والعميلُ يقترح ولا يقرّر.
// **فيُدسّ السعرُ والمجموعُ والإجماليُّ والحسمُ في الجسم**، ويُقاس أنّ
// الردَّ لم يتغيّر.
//
// # ولماذا يُقارَن بطلبٍ نظيف
//
// **اختبارٌ يقول «رُفض» يمرّ في خادمٍ يرفض كلَّ شيء** — فيُنشأ طلبان:
// نظيفٌ ومدسوس، **ويُقاس أنّهما بالمبلغ نفسه.**

import (
	"fmt"
	"net/http"
	"testing"
)

// TestFIN_TamperedFieldsIgnored **الحقولُ المدسوسةُ تُتجاهَل.**
func TestFIN_TamperedFieldsIgnored(t *testing.T) {
	h := New(t)
	u := h.Customer()
	item := h.NewItem(1000)

	clean := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), orderBody(item, 2))
	if clean.Code >= 400 {
		t.Fatalf("FIN: تعذّر إنشاءُ الطلب النظيف: %s", clean)
	}
	want := fmt.Sprint(clean.JSON()["total"])

	tampering := []struct {
		ID    string
		Field string
		Value any
	}{
		{"FIN-001", "total", 1},
		{"FIN-002", "subtotal", 1},
		{"FIN-003", "delivery_fee", 0},
		{"FIN-004", "discount", 999999},
		{"FIN-005", "price", 1},
		{"FIN-006", "commission", 0},
	}
	for _, c := range tampering {
		t.Run(c.ID, func(t *testing.T) {
			body := orderBody(item, 2)
			body[c.Field] = c.Value
			got := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), body)
			if got.Code >= 400 {
				return // **الرفضُ جوابٌ صحيحٌ أيضا**
			}
			if g := fmt.Sprint(got.JSON()["total"]); g != want {
				t.Errorf("%s دُسّ %s=%v فتغيّر الإجماليّ: %s ≠ %s",
					c.ID, c.Field, c.Value, g, want)
			}
		})
	}
}

// TestFIN_ItemPriceComesFromDB **والسعرُ من القاعدة لا من الجسم.**
func TestFIN_ItemPriceComesFromDB(t *testing.T) {
	h := New(t)
	u := h.Customer()
	item := h.NewItem(3000)

	body := orderBody(item, 1)
	body["items"].([]map[string]any)[0]["price"] = 1
	body["items"].([]map[string]any)[0]["unit_price"] = 1

	got := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), body)
	if got.Code >= 400 {
		return
	}
	sub := fmt.Sprint(got.JSON()["subtotal"])
	if sub == "1" || sub == "" {
		t.Errorf("FIN-010 سعرُ الصنف أُخذ من الجسم: المجموع=%s والسعرُ في القاعدة=%d",
			sub, item.Cost)
	}
}

// badInputs **مُدخَلاتٌ يجب أن تُرفض.**
var badInputs = []struct {
	ID   string
	Body func(*Item) map[string]any
}{
	{"VAL-001", func(i *Item) map[string]any {
		b := orderBody(i, 1)
		delete(b, "items")
		return b
	}},
	{"VAL-002", func(i *Item) map[string]any {
		return map[string]any{"items": []map[string]any{}, "lat": 35.9, "lng": 39.0,
			"address_text": "س", "payment_method": "cash"}
	}},
	{"VAL-003", func(i *Item) map[string]any { return orderBody(i, -5) }},
	{"VAL-004", func(i *Item) map[string]any { return orderBody(i, 0) }},
	{"VAL-005", func(i *Item) map[string]any { return orderBody(i, 999999999) }},
	{"VAL-006", func(i *Item) map[string]any {
		b := orderBody(i, 1)
		b["items"].([]map[string]any)[0]["menu_item_id"] = "00000000-0000-0000-0000-000000000000"
		return b
	}},
	{"VAL-007", func(i *Item) map[string]any {
		b := orderBody(i, 1)
		b["items"].([]map[string]any)[0]["menu_item_id"] = "ليس-معرّفا"
		return b
	}},
	{"VAL-008", func(i *Item) map[string]any {
		b := orderBody(i, 1)
		b["lat"], b["lng"] = 999.0, 999.0
		return b
	}},
	{"VAL-009", func(i *Item) map[string]any {
		b := orderBody(i, 1)
		b["payment_method"] = "بيتكوين"
		return b
	}},
}

// TestVAL_BadInputRejected **والمُدخَلُ الفاسدُ يُردّ لا يُقبل.**
func TestVAL_BadInputRejected(t *testing.T) {
	h := New(t)
	u := h.Customer()
	item := h.NewItem(1000)
	for _, c := range badInputs {
		t.Run(c.ID, func(t *testing.T) {
			got := h.POSTKey("/api/v1/orders", u.Token, uniq("k"), c.Body(item))
			if got.Code < 400 {
				t.Errorf("%s مُدخَلٌ فاسدٌ قُبل: %s", c.ID, got)
			}
			if got.Code >= 500 {
				t.Errorf("%s مُدخَلٌ فاسدٌ أسقط الخادمَ ٥٠٠ بدل ٤٠٠: %s", c.ID, got)
			}
		})
	}
}

// TestVAL_MalformedJSON **وجسمٌ معطوبٌ لا يُسقط الخادم.**
func TestVAL_MalformedJSON(t *testing.T) {
	h := New(t)
	u := h.Customer()
	for _, raw := range []string{`{`, `[]`, `null`, `"نصّ"`, ``} {
		got := h.Call("POST", "/api/v1/orders", u.Token, nil,
			map[string]string{"X-QA-Raw": raw})
		if got.Code >= 500 {
			t.Errorf("VAL-020 جسمٌ %q ردّ %d — الخادمُ يسقط لا يرفض", raw, got.Code)
		}
	}
	_ = http.StatusOK
}

// TestVAL_EmptyQuote **وسلّةٌ فارغةٌ ليست تسعيرة** — (BUG-006، أُصلح
// ٢٠٢٦-٠٨-١٩؛ ويبقى الحارسُ أبدا).
func TestVAL_EmptyQuote(t *testing.T) {
	h := New(t)
	got := h.POST("/api/v1/public/quote", "", map[string]any{
		"items": []any{}, "lat": 35.9506, "lng": 39.0094,
	})
	if got.Code != http.StatusBadRequest {
		t.Errorf("VAL-030 تسعيرةٌ بسلّةٍ فارغة: يُنتظر 400 ووقع %s", got)
	}
}

// TestVAL_MissingSection **وقسمٌ لا وجودَ له يُردّ ٤٠٤** — (BUG-005).
func TestVAL_MissingSection(t *testing.T) {
	h := New(t)
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "ليس-معرّفا"} {
		got := h.GET("/api/v1/public/sections/"+id+"/items", "")
		if got.Code != http.StatusNotFound {
			t.Errorf("VAL-031 قسمٌ %q: يُنتظر 404 ووقع %s", id, got)
		}
	}
}
