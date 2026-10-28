package qa

// ══════════════════════════════════════════════════════════════════════
// **طريقةُ سعر الخيارات** (قرارُ المالك ٢٠٢٦-١٠-٢٨) — هجرة 0482
// ══════════════════════════════════════════════════════════════════════
//
// «بدون سعر» صفرٌ مهما أُرسل، و«حسب الوزن» يُحسب من سعر الصنف لأقرب ٥،
// **ويتبعه من كلّ باب**: التعديل، والنسبةُ دفعةً، وسعرُ الصرف — **ويُحاسَب
// الطلبُ به**. ونسخةُ التطبيق القديمة لا تمحو الطريقة.

import (
	"net/http"
	"testing"
)

// weightGroup مجموعةُ «حسب الوزن» بأساسٍ وأوزانٍ بالغرام.
func weightGroup(base int, grams ...int) map[string]any {
	opts := []any{}
	for _, g := range grams {
		// **والاسمُ والسعرُ المُرسَلان يُهملان** — يُولَّدان من الغرامات.
		opts = append(opts, map[string]any{"name": "x", "grams": g, "price_delta": 7})
	}
	return map[string]any{
		"name": "الوزن", "min_select": 1, "max_select": 1,
		"price_mode": "weight", "base_grams": base, "options": opts,
	}
}

// optionDeltas فروقُ خيارات الصنف بأسمائها.
func optionDeltas(t *testing.T, h *Harness, itemID string) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT o.name, o.price_delta FROM modifier_options o
		JOIN modifier_groups g ON g.id = o.group_id WHERE g.item_id = $1 AND NOT g.store_drinks`, itemID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		var d int64
		_ = rows.Scan(&n, &d)
		out[n] = d
	}
	return out
}

func patchItem(t *testing.T, h *Harness, tok, itemID string, body map[string]any) {
	t.Helper()
	if r := h.PATCH("/api/v1/merchant/menu/items/"+itemID, tok, body); r.Code != http.StatusOK {
		t.Fatalf("حفظُ الصنف: %s", r)
	}
}

// **«بدون سعر» صفرٌ مهما أرسل التطبيق** — المقاسُ واللونُ بلا سعر.
func TestOptionModes_NoneForcesZero(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	it := h.NewItemFor(m, 1000)
	tok := h.TokenFor(m.Owner.ID, "merchant")

	patchItem(t, h, tok, it.ID, map[string]any{"modifiers": []any{map[string]any{
		"name": "اللون", "min_select": 1, "max_select": 1, "price_mode": "none",
		"options": []any{
			map[string]any{"name": "أسود", "price_delta": 500},
			map[string]any{"name": "أحمر", "price_delta": 300, "available": false},
		},
	}}})
	d := optionDeltas(t, h, it.ID)
	if d["أسود"] != 0 || d["أحمر"] != 0 || len(d) != 2 {
		t.Fatalf("**خيارٌ بلا سعرٍ حُفظ بسعر**: %v", d)
	}
	menu := h.GET("/api/v1/merchant/stores/"+m.ID+"/menu", tok)
	if !containsBytes(menu.Body, `"price_mode":"none"`) {
		t.Fatalf("**القائمةُ بلا طريقة السعر**: %s", menu)
	}
}

// **الوزنُ يُحسب من سعر الصنف لأقرب ٥** — ٦٠٠ لكلّ ١٠٠ غ: ٢٥٠ غ = ١٥٠٠،
// كيلو = ٦٠٠٠. **والاسمُ من الغرامات.**
func TestOptionModes_WeightComputesAndRounds(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	it := h.NewItemFor(m, 600)
	tok := h.TokenFor(m.Owner.ID, "merchant")

	patchItem(t, h, tok, it.ID, map[string]any{"modifiers": []any{weightGroup(100, 250, 500, 1000, 1500)}})
	d := optionDeltas(t, h, it.ID)
	want := map[string]int64{"250 غ": 900, "نصف كيلو": 2400, "كيلو": 5400, "1.5 كيلو": 8400}
	for n, v := range want {
		if got, ok := d[n]; !ok || got != v {
			t.Fatalf("**وزن %s**: %v (المنتظَر %d) — الكلّ %v", n, got, v, d)
		}
	}

	// **وأساسُ ٢٠٠ غ وسعرٌ لا يُقسم** — يُقرَّب لأقرب ٥.
	patchItem(t, h, tok, it.ID, map[string]any{"price": 333, "modifiers": []any{weightGroup(200, 250, 1000)}})
	d = optionDeltas(t, h, it.ID)
	// ٣٣٣ × ٢٥٠ ÷ ٢٠٠ = ٤١٦٫٢٥ ⇒ ٤١٥ ⇒ الفرق ٨٢ · ٣٣٣ × ١٠٠٠ ÷ ٢٠٠ = ١٦٦٥ ⇒ الفرق ١٣٣٢.
	if d["250 غ"] != 82 || d["كيلو"] != 1332 {
		t.Fatalf("**الوزنُ بأساس ٢٠٠ غ لم يُقرَّب لأقرب ٥**: %v", d)
	}
	var base *int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT base_grams FROM modifier_groups WHERE item_id = $1`, it.ID).Scan(&base)
	if base == nil || *base != 200 {
		t.Fatalf("**الوزنُ الأساسيّ لم يُحفظ**: %v", base)
	}
}

// **سعرُ الصنف يتبدّل من أيّ بابٍ فتتبعه الأوزان** — التعديلُ وحده،
// والنسبةُ دفعةً واحدة.
func TestOptionModes_WeightFollowsItemPrice(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	it := h.NewItemFor(m, 600)
	tok := h.TokenFor(m.Owner.ID, "merchant")
	patchItem(t, h, tok, it.ID, map[string]any{"modifiers": []any{weightGroup(100, 250, 1000)}})

	// **السعرُ وحدَه بلا الخيارات** — ٨٠٠: ٢٥٠ غ = ٢٠٠٠.
	patchItem(t, h, tok, it.ID, map[string]any{"price": 800})
	if d := optionDeltas(t, h, it.ID); d["250 غ"] != 1200 || d["كيلو"] != 7200 {
		t.Fatalf("**تبدّل السعرُ ولم تتبعه الأوزان**: %v", d)
	}

	// **وزيادةُ ١٠٪ دفعةً** — ٨٨٠: ٢٥٠ غ = ٢٢٠٠.
	r := h.POST("/api/v1/merchant/stores/"+m.ID+"/menu/prices", tok, map[string]any{"percent": 10, "confirm": true})
	if r.Code != http.StatusOK || r.JSON()["applied"] != true {
		t.Fatalf("النسبة: %s", r)
	}
	if d := optionDeltas(t, h, it.ID); d["250 غ"] != 1320 || d["كيلو"] != 7920 {
		t.Fatalf("**زيادةُ النسبة لم تصل الأوزان**: %v", d)
	}

	// **ونسخُ الصنف ينسخ الطريقة.**
	dup := h.POST("/api/v1/merchant/menu/items/"+it.ID+"/duplicate", tok, map[string]any{})
	newID, _ := dup.JSON()["id"].(string)
	if d := optionDeltas(t, h, newID); d["250 غ"] != 1320 {
		t.Fatalf("**النسخةُ فقدت الوزن**: %v", d)
	}
	var mode string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT price_mode FROM modifier_groups WHERE item_id = $1`, newID).Scan(&mode)
	if mode != "weight" {
		t.Fatalf("**النسخةُ بطريقة %q**", mode)
	}
}

// **والصنفُ الدولاريُّ يتبع سعرَ الصرف بأوزانه.**
func TestOptionModes_WeightFollowsExchangeRate(t *testing.T) {
	h := New(t)
	m := newMDFxBare(t, h, "OPW-USD")
	if r := h.PATCH(m.url("/exchange-rate"), m.fx.Tok, map[string]any{"rate": 115}); r.Code != http.StatusOK {
		t.Fatalf("سعرُ الصرف: %s", r)
	}
	id := usdItem(t, h, m, map[string]any{
		"name": "بن", "price_currency": "USD", "price_usd": 5,
		"modifiers": []any{weightGroup(100, 250)},
	})
	// ٥ × ١١٥ = ٥٧٥ · ٢٥٠ غ = ١٤٣٧٫٥ ⇒ ١٤٤٠ ⇒ الفرق ٨٦٥.
	if d := optionDeltas(t, h, id); d["250 غ"] != 865 {
		t.Fatalf("**الوزنُ الدولاريّ**: %v", d)
	}
	if r := h.PATCH(m.url("/exchange-rate"), m.fx.Tok, map[string]any{"rate": 120}); r.Code != http.StatusOK {
		t.Fatalf("سعرُ الصرف الجديد: %s", r)
	}
	// ٦٠٠ ⇒ ١٥٠٠ ⇒ الفرق ٩٠٠.
	if d := optionDeltas(t, h, id); d["250 غ"] != 900 {
		t.Fatalf("**تبدّل الصرفُ ولم تتبعه الأوزان**: %v", d)
	}
}

// **نسخةُ التطبيق القديمة لا ترسل الطريقة** — فتبقى بالوزن ولا يُقبل سعرُها.
func TestOptionModes_OldClientKeepsMode(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	it := h.NewItemFor(m, 600)
	tok := h.TokenFor(m.Owner.ID, "merchant")
	patchItem(t, h, tok, it.ID, map[string]any{"modifiers": []any{
		weightGroup(200, 250),
		map[string]any{"name": "اللون", "min_select": 1, "max_select": 1, "price_mode": "none",
			"options": []any{map[string]any{"name": "أسود"}}},
	}})

	// **ما يرسله تطبيقٌ قديم**: الأسماءُ والفروقُ بلا طريقةٍ ولا غرامات.
	patchItem(t, h, tok, it.ID, map[string]any{"price": 700, "modifiers": []any{
		map[string]any{"name": "الوزن", "min_select": 1, "max_select": 1,
			"options": []any{map[string]any{"name": "250 غ", "price_delta": 1}}},
		map[string]any{"name": "اللون", "min_select": 1, "max_select": 1,
			"options": []any{map[string]any{"name": "أسود", "price_delta": 999}}},
	}})
	var weightMode, colorMode string
	var base *int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT price_mode, base_grams FROM modifier_groups WHERE item_id = $1 AND name = 'الوزن'`, it.ID).Scan(&weightMode, &base)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT price_mode FROM modifier_groups WHERE item_id = $1 AND name = 'اللون'`, it.ID).Scan(&colorMode)
	if weightMode != "weight" || colorMode != "none" || base == nil || *base != 200 {
		t.Fatalf("**التطبيقُ القديمُ محا الطريقة**: وزن=%q أساس=%v لون=%q", weightMode, base, colorMode)
	}
	// ٧٠٠ × ٢٥٠ ÷ ٢٠٠ = ٨٧٥ ⇒ الفرق ١٧٥.
	if d := optionDeltas(t, h, it.ID); d["250 غ"] != 175 || d["أسود"] != 0 {
		t.Fatalf("**قُبل سعرُ التطبيق القديم**: %v", d)
	}
}

// **والطلبُ يُحاسَب بسعر الوزن** — ٦٠٠ لكلّ ١٠٠ غ، والزبونُ اختار ٢٥٠ غ.
func TestOptionModes_OrderChargesWeightPrice(t *testing.T) {
	h := New(t)
	m := newMDFxBare(t, h, "OPW-ORDER")
	patchItem(t, h, m.fx.Tok, m.fx.Item.ID, map[string]any{"price": 600,
		"modifiers": []any{weightGroup(100, 250, 1000)}})

	var optID string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT o.id::text FROM modifier_options o JOIN modifier_groups g ON g.id = o.group_id
		 WHERE g.item_id = $1 AND o.grams = 250`, m.fx.Item.ID).Scan(&optID); err != nil {
		t.Fatal(err)
	}
	body := zoneBody(m.fx.Item, m.z.Lat, m.z.Lng)
	body["items"] = []map[string]any{{"menu_item_id": m.fx.Item.ID, "qty": 1, "option_ids": []string{optID}}}
	made := h.POST("/api/v1/orders", h.Customer().Token, body)
	if made.Code != http.StatusOK && made.Code != http.StatusCreated {
		t.Fatalf("الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	var mp int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT merchant_price FROM order_items WHERE order_id = $1::uuid`, oid).Scan(&mp); err != nil {
		t.Fatal(err)
	}
	if mp != 1500 {
		t.Fatalf("**الطلبُ لم يُحاسَب بسعر الوزن**: سعرُ المتجر %d والمنتظَر ١٥٠٠", mp)
	}
}

// **القوالبُ الجاهزة بطرقها** — اللونُ بلا سعرٍ وبألوانٍ أكثر، والوزنُ بالغرامات.
func TestOptionModes_PresetsSeeded(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	r := h.GET("/api/v1/merchant/stores/"+m.ID+"/option-presets", m.Owner.Token)
	if r.Code != http.StatusOK {
		t.Fatalf("القراءة: %s", r)
	}
	list, _ := r.JSON()["presets"].([]any)
	modes := map[string]string{}
	var colors, weights []any
	for _, x := range list {
		p, _ := x.(map[string]any)
		name, _ := p["name"].(string)
		modes[name], _ = p["price_mode"].(string)
		switch name {
		case "اللون":
			colors, _ = p["options"].([]any)
		case "الوزن":
			weights, _ = p["options"].([]any)
		}
	}
	for n, want := range map[string]string{"المقاس": "none", "اللون": "none", "نمرة رجالي": "none", "نمرة نسائي": "none", "نمرة أطفال": "none", "مقاس أطفال": "none", "بدون": "none",
		"الوزن": "weight", "الحجم": "priced", "العبوة": "priced", "الإضافات": "priced"} {
		if modes[n] != want {
			t.Fatalf("**طريقةُ «%s»**: %q والمنتظَر %q", n, modes[n], want)
		}
	}
	if len(colors) < 10 {
		t.Fatalf("**الألوانُ لم تُزَد**: %v", colors)
	}
	if len(weights) == 0 || weights[0].(map[string]any)["grams"] == nil {
		t.Fatalf("**الوزنُ بلا غرامات**: %v", weights)
	}
}

// **النمراتُ ثلاثٌ والمقاسُ اثنان** (قرارُ المالك ٢٠٢٦-١٠-٢٨) — و«النمرة» صارت
// «نمرة رجالي» بمعرّفها، والأحذيةُ ترى الثلاث والألبسةُ مقاسَي الكبار والأطفال.
func TestOptionModes_ShoeAndKidsPresets(t *testing.T) {
	h := New(t)
	count := func(name string) int {
		var n int
		_ = h.Pool.QueryRow(ctxBG(), `SELECT jsonb_array_length(options) FROM option_presets WHERE name = $1`, name).Scan(&n)
		return n
	}
	for name, want := range map[string]int{"نمرة رجالي": 8, "نمرة نسائي": 7, "نمرة أطفال": 16, "مقاس أطفال": 12, "المقاس": 6} {
		if got := count(name); got != want {
			t.Fatalf("**«%s» بـ%d خيارات والمنتظَر %d**", name, got, want)
		}
	}
	var old int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM option_presets WHERE name = 'النمرة'`).Scan(&old)
	if old != 0 {
		t.Fatalf("**«النمرة» القديمة ما زالت قالباً مستقلّاً**")
	}
	// **والربطُ لكلّ تصنيفٍ موجود** — والتصنيفاتُ التي لا وجودَ لها في القاعدة لا تُفحص.
	for cat, presets := range map[string][]string{
		"أحذية": {"نمرة رجالي", "نمرة نسائي", "نمرة أطفال"},
		"ألبسة": {"المقاس", "مقاس أطفال", "اللون"},
		"ملابس": {"المقاس", "مقاس أطفال", "اللون"},
	} {
		var exists bool
		_ = h.Pool.QueryRow(ctxBG(), `SELECT EXISTS (SELECT 1 FROM categories WHERE name = $1)`, cat).Scan(&exists)
		if !exists {
			continue
		}
		for _, p := range presets {
			var linked bool
			_ = h.Pool.QueryRow(ctxBG(), `
				SELECT EXISTS (SELECT 1 FROM category_option_presets cop
				  JOIN categories c ON c.id = cop.category_id JOIN option_presets p ON p.id = cop.preset_id
				 WHERE c.name = $1 AND p.name = $2)`, cat, p).Scan(&linked)
			if !linked {
				t.Fatalf("**«%s» لا يطلع لتصنيف «%s»**", p, cat)
			}
		}
	}
}
