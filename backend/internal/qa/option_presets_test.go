package qa

// خياراتُ الصنف الجاهزة ونسخُ صنفٍ والأسعارُ دفعةً واحدة (قرارُ المالك ٢٠٢٦-١٠-٢٧).

import (
	"testing"
)

func presetNames(r Res, chosenOnly bool) map[string]bool {
	out := map[string]bool{}
	list, _ := r.JSON()["presets"].([]any)
	for _, x := range list {
		p, _ := x.(map[string]any)
		if !chosenOnly || p["chosen"] == true {
			out[p["name"].(string)] = true
		}
	}
	return out
}

func presetID(r Res, name string) string {
	list, _ := r.JSON()["presets"].([]any)
	for _, x := range list {
		p, _ := x.(map[string]any)
		if p["name"] == name {
			return p["id"].(string)
		}
	}
	return ""
}

// **المطعمُ يرى خياراتِ تصنيفه، ثمّ يختار لنفسه، ثمّ يعود إلى تصنيفه.**
func TestOptionPresets_CategoryThenStoreChoice(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE merchants SET category_id = (SELECT id FROM categories WHERE name = 'مطاعم' LIMIT 1)
		 WHERE id = $1`, m.ID); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/merchant/stores/" + m.ID + "/option-presets"

	r := h.GET(path, m.Owner.Token)
	if r.Code != 200 {
		t.Fatalf("القراءة: %s", r)
	}
	chosen := presetNames(r, true)
	if !chosen["الحجم"] || !chosen["الإضافات"] || chosen["النمرة"] || r.JSON()["custom"] != false {
		t.Fatalf("المطعمُ يرى خياراتِ تصنيفه وحدَها: %v", chosen)
	}
	if all := presetNames(r, false); !all["النمرة"] || !all["المقاس"] {
		t.Fatalf("والقائمةُ الكاملةُ تصله ليختار منها: %v", all)
	}

	shoe := presetID(r, "النمرة")
	r = h.Call("PUT", path, m.Owner.Token, map[string]any{"preset_ids": []string{shoe}}, nil)
	if r.Code != 200 {
		t.Fatalf("الحفظ: %s", r)
	}
	chosen = presetNames(r, true)
	if len(chosen) != 1 || !chosen["النمرة"] || r.JSON()["custom"] != true {
		t.Fatalf("اختيارُه يعلو على تصنيفه: %v", chosen)
	}

	r = h.Call("PUT", path, m.Owner.Token, map[string]any{"reset": true}, nil)
	if chosen = presetNames(r, true); !chosen["الحجم"] || chosen["النمرة"] {
		t.Fatalf("والإرجاعُ يعيده إلى تصنيفه: %v", chosen)
	}

	// **ومتجرُ غيرِه لا يُقرأ ولا يُكتب.**
	other := f.Merchant()
	if r := h.GET(path, other.Owner.Token); r.Code != 403 {
		t.Fatalf("متجرُ غيرِه: %d", r.Code)
	}
}

// **والمندوبُ يضبطها لعميله، ولا يضبطها لغير عميله.**
func TestOptionPresets_RepForClientOnly(t *testing.T) {
	h := New(t)
	f := h.Factory()
	rep := h.NewUser("sales")
	m := f.Merchant(OwnedByRep(rep.ID))
	stranger := f.Merchant()

	r := h.GET("/api/v1/rep/stores/"+m.ID+"/option-presets", rep.Token)
	if r.Code != 200 {
		t.Fatalf("عميلُه: %s", r)
	}
	id := presetID(r, "المقاس")
	r = h.Call("PUT", "/api/v1/rep/stores/"+m.ID+"/option-presets", rep.Token,
		map[string]any{"preset_ids": []string{id}}, nil)
	if c := presetNames(r, true); len(c) != 1 || !c["المقاس"] {
		t.Fatalf("حفظُ المندوب: %s", r)
	}
	if r := h.GET("/api/v1/rep/stores/"+stranger.ID+"/option-presets", rep.Token); r.Code != 403 {
		t.Fatalf("غيرُ عميله: %d", r.Code)
	}
}

// **«نمرة ٤٢ خلصت»** — الخيارُ المطفأُ يُحفظ مطفأً، **وغيابُ الحقل «متوفّر»**
// فلا تطفئ التطبيقاتُ القديمةُ خياراً بصمتها.
func TestOptionAvailability_SavedPerOption(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	seed := h.NewItemFor(m, 1000)

	r := h.PATCH("/api/v1/merchant/menu/items/"+seed.ID, m.Owner.Token, map[string]any{
		"modifiers": []any{map[string]any{
			"name": "النمرة", "min_select": 1, "max_select": 1,
			"options": []any{
				map[string]any{"name": "41", "price_delta": 0},
				map[string]any{"name": "42", "price_delta": 0, "available": false},
			},
		}},
	})
	if r.Code != 200 {
		t.Fatalf("الحفظ: %s", r)
	}
	got := map[string]bool{}
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT o.name, o.available FROM modifier_options o
		JOIN modifier_groups g ON g.id = o.group_id WHERE g.item_id = $1`, seed.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var n string
		var a bool
		_ = rows.Scan(&n, &a)
		got[n] = a
	}
	rows.Close()
	if !got["41"] || got["42"] {
		t.Fatalf("التوفّرُ لكلّ خيار: %v", got)
	}
}

// **نسخُ صنفٍ بخياراته** — والنسخةُ صنفٌ مستقلّ.
func TestDuplicateItem_CopiesOptions(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	seed := h.NewItemFor(m, 800)
	h.PATCH("/api/v1/merchant/menu/items/"+seed.ID, m.Owner.Token, map[string]any{
		"modifiers": []any{map[string]any{
			"name": "اللون", "min_select": 1, "max_select": 1,
			"options": []any{map[string]any{"name": "أسود"}, map[string]any{"name": "أحمر", "available": false}},
		}},
	})

	r := h.POST("/api/v1/merchant/menu/items/"+seed.ID+"/duplicate", m.Owner.Token, map[string]any{})
	if r.Code != 201 {
		t.Fatalf("النسخ: %s", r)
	}
	newID, _ := r.JSON()["id"].(string)
	if newID == "" || newID == seed.ID {
		t.Fatalf("معرّفُ النسخة: %v", r.JSON())
	}
	var name string
	var price int64
	var opts, off int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT i.name, i.merchant_price,
		       (SELECT count(*) FROM modifier_options o JOIN modifier_groups g ON g.id = o.group_id WHERE g.item_id = i.id),
		       (SELECT count(*) FROM modifier_options o JOIN modifier_groups g ON g.id = o.group_id WHERE g.item_id = i.id AND NOT o.available)
		  FROM menu_items i WHERE i.id = $1`, newID).Scan(&name, &price, &opts, &off); err != nil {
		t.Fatal(err)
	}
	if price != 800 || opts != 2 || off != 1 || name == seed.Name {
		t.Fatalf("النسخة: اسم=%q سعر=%d خيارات=%d مطفأ=%d", name, price, opts, off)
	}

	other := f.Merchant()
	if r := h.POST("/api/v1/merchant/menu/items/"+seed.ID+"/duplicate", other.Owner.Token, map[string]any{}); r.Code != 403 {
		t.Fatalf("صنفُ غيرِه لا يُنسخ: %d", r.Code)
	}
}

// **«زيد كلّ أسعاري ١٠٪»** — معاينةٌ لا تكتب، ثمّ تطبيق.
func TestBulkPrice_PreviewThenApply(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	a := h.NewItemFor(m, 100)
	path := "/api/v1/merchant/stores/" + m.ID + "/menu/prices"

	r := h.POST(path, m.Owner.Token, map[string]any{"percent": 10})
	if r.Code != 200 || num(r.JSON()["count"]) != 1 || r.JSON()["applied"] != false {
		t.Fatalf("المعاينة: %s", r)
	}
	var p int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT merchant_price FROM menu_items WHERE id = $1`, a.ID).Scan(&p)
	if p != 100 {
		t.Fatalf("المعاينةُ كتبت: %d", p)
	}

	r = h.POST(path, m.Owner.Token, map[string]any{"percent": 10, "confirm": true})
	if r.Code != 200 || r.JSON()["applied"] != true {
		t.Fatalf("التطبيق: %s", r)
	}
	_ = h.Pool.QueryRow(ctxBG(), `SELECT merchant_price FROM menu_items WHERE id = $1`, a.ID).Scan(&p)
	if p != 110 {
		t.Fatalf("السعرُ بعد ١٠٪: %d والمنتظَر ١١٠", p)
	}

	if r := h.POST(path, m.Owner.Token, map[string]any{"percent": 500, "confirm": true}); r.Code != 400 {
		t.Fatalf("نسبةٌ خارج الحدّ: %d", r.Code)
	}
	other := f.Merchant()
	if r := h.POST(path, other.Owner.Token, map[string]any{"percent": 10, "confirm": true}); r.Code != 403 {
		t.Fatalf("متجرُ غيرِه: %d", r.Code)
	}
}

// **اللوحةُ تنشئ قالباً وتربطه بتصنيف** — فيطلع لمتاجره.
func TestAdminOptionPresets_CreateLinkDelete(t *testing.T) {
	h := New(t)
	f := h.Factory()
	admin := h.NewUser("admin")
	m := f.Merchant()
	var cat string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT category_id::text FROM merchants WHERE id = $1`, m.ID).Scan(&cat)

	name := "قالب QA " + m.ID[:6]
	r := h.POST("/api/v1/admin/option-presets", admin.Token, map[string]any{
		"name": name, "min_select": 1, "max_select": 1,
		"options":      []any{map[string]any{"name": "أ"}, map[string]any{"name": " "}},
		"category_ids": []string{cat},
	})
	if r.Code != 200 {
		t.Fatalf("الإنشاء: %s", r)
	}
	id, _ := r.JSON()["id"].(string)
	defer h.DEL("/api/v1/admin/option-presets/"+id, admin.Token)

	r = h.GET("/api/v1/merchant/stores/"+m.ID+"/option-presets", m.Owner.Token)
	if !presetNames(r, true)[name] {
		t.Fatalf("القالبُ المربوطُ بتصنيفه يطلع للمتجر: %s", r)
	}
	if r := h.POST("/api/v1/admin/option-presets", m.Owner.Token, map[string]any{"name": "x"}); r.Code != 403 {
		t.Fatalf("صاحبُ متجرٍ لا يدير القوالب: %d", r.Code)
	}
	if r := h.DEL("/api/v1/admin/option-presets/"+id, admin.Token); r.Code != 200 {
		t.Fatalf("الحذف: %s", r)
	}
}
