package qa

// ══════════════════════════════════════════════════════════════════════
// **المشروبُ بعددِه لا بعدد الصنف** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — هجرة 0487
// ══════════════════════════════════════════════════════════════════════
//
// شاهدُه على التجهيز طلبُ #1177: «٢ × شاورما + عيران» حُسب عيرانين، فصار
// المجموعُ ٢٠٨ والصحيحُ ١٦٠. **فالمشروبُ الذي أُرسل عددُه (`option_qty`) بندٌ
// مستقلٌّ بعدده، والعميلُ القديمُ الذي لا يرسله يبقى كما كان.**

import (
	"net/http"
	"testing"
)

type drinkFx struct {
	m            mdFx
	reg, dbl     string
	ayran, pepsi string
	kinza, mir   string
}

// newDrinkFx صنفٌ بسعر ٢٠ ومقاسٍ (عادي ٠ / دبل +١٠) ومشروباتُ المتجر الأربعة.
func newDrinkFx(t *testing.T, h *Harness, name string) drinkFx {
	t.Helper()
	m := newMDFxBare(t, h, name)
	// **المشروباتُ للمطاعم وحدَها** (هجرة 0488) — فتصنيفُ متجر الشاهد مطعم.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE categories SET has_drinks = true
		 WHERE id = (SELECT category_id FROM merchants WHERE id = $1::uuid)`, m.fx.M.ID); err != nil {
		t.Fatal(err)
	}
	patchItem(t, h, m.fx.Tok, m.fx.Item.ID, map[string]any{"price": 20, "modifiers": []any{
		map[string]any{"name": "الحجم", "min_select": 1, "max_select": 1, "options": []any{
			map[string]any{"name": "عادي", "price_delta": 0},
			map[string]any{"name": "دبل", "price_delta": 10},
		}},
	}})
	if r := h.Call("PUT", m.url("/drinks"), m.fx.Tok, map[string]any{"drinks": []any{
		map[string]any{"name": "عيران", "price": 5},
		map[string]any{"name": "ببسي", "price": 7},
		map[string]any{"name": "كنزا", "price": 6},
		map[string]any{"name": "ميرندا", "price": 7},
	}}, nil); r.Code != http.StatusOK {
		t.Fatalf("قائمةُ المشروبات: %s", r)
	}
	opt := func(name string) string {
		var id string
		if err := h.Pool.QueryRow(ctxBG(), `
			SELECT o.id::text FROM modifier_options o JOIN modifier_groups g ON g.id = o.group_id
			 WHERE g.item_id = $1 AND o.name = $2`, m.fx.Item.ID, name).Scan(&id); err != nil {
			t.Fatalf("الخيار %s: %v", name, err)
		}
		return id
	}
	return drinkFx{m: m, reg: opt("عادي"), dbl: opt("دبل"),
		ayran: opt("عيران"), pepsi: opt("ببسي"), kinza: opt("كنزا"), mir: opt("ميرندا")}
}

// items1177 سلّةُ الطلب #1177 — بالحقل الجديد أو بدونه.
func (d drinkFx) items1177(perDrink bool) []map[string]any {
	id := d.m.fx.Item.ID
	if !perDrink {
		return []map[string]any{
			{"menu_item_id": id, "qty": 2, "option_ids": []string{d.reg, d.ayran, d.pepsi}},
			{"menu_item_id": id, "qty": 3, "option_ids": []string{d.dbl, d.kinza, d.ayran, d.mir}},
		}
	}
	return []map[string]any{
		{"menu_item_id": id, "qty": 2, "option_ids": []string{d.reg, d.ayran, d.pepsi},
			"option_qty": map[string]int{d.ayran: 1, d.pepsi: 1}},
		// **والمشروبُ يُعرف من مفتاحه** — لا يلزم في `option_ids`.
		{"menu_item_id": id, "qty": 3, "option_ids": []string{d.dbl},
			"option_qty": map[string]int{d.kinza: 1, d.ayran: 1, d.mir: 1}},
	}
}

func (d drinkFx) order(t *testing.T, h *Harness, items []map[string]any) (Res, string) {
	t.Helper()
	r, oid, _ := d.orderAs(t, h, items)
	return r, oid
}

// orderAs الطلبُ ومعه توكنُ زبونه.
func (d drinkFx) orderAs(t *testing.T, h *Harness, items []map[string]any) (Res, string, string) {
	t.Helper()
	body := zoneBody(d.m.fx.Item, d.m.z.Lat, d.m.z.Lng)
	body["items"] = items
	tok := h.Customer().Token
	r := h.POST("/api/v1/orders", tok, body)
	oid, _ := r.JSON()["id"].(string)
	return r, oid, tok
}

func (d drinkFx) quote(t *testing.T, h *Harness, items []map[string]any) int64 {
	t.Helper()
	r := h.POST("/api/v1/public/quote", "", map[string]any{"items": items, "lat": d.m.z.Lat, "lng": d.m.z.Lng})
	if r.Code != http.StatusOK {
		t.Fatalf("التسعيرة: %s", r)
	}
	v, _ := r.JSON()["subtotal"].(float64)
	return int64(v)
}

func orderSubtotal(t *testing.T, h *Harness, oid string) int64 {
	t.Helper()
	var sub, lines int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT o.subtotal, (SELECT sum(unit_price * qty) FROM order_items WHERE order_id = o.id)
		  FROM orders o WHERE o.id = $1::uuid`, oid).Scan(&sub, &lines); err != nil {
		t.Fatal(err)
	}
	// **وجمعُ البنود هو المجموع** — فكلُّ قارئٍ لـ`unit_price * qty` يرى الحقيقة.
	if sub != lines {
		t.Fatalf("**مجموعُ الطلب %d وجمعُ بنوده %d**", sub, lines)
	}
	return sub
}

// **#1177 بالحقل الجديد ⇒ ١٦٠**، والمشروباتُ بنودٌ بعددها، والتسعيرةُ مثلُه.
func TestDrinkQty_1177_PerDrink(t *testing.T) {
	h := New(t)
	d := newDrinkFx(t, h, "DRQ-NEW")
	items := d.items1177(true)
	if q := d.quote(t, h, items); q != 160 {
		t.Fatalf("**التسعيرةُ %d والمنتظَر ١٦٠**", q)
	}
	r, oid, tok := d.orderAs(t, h, items)
	if r.Code != http.StatusOK && r.Code != http.StatusCreated {
		t.Fatalf("الطلب: %s", r)
	}
	if sub := orderSubtotal(t, h, oid); sub != 160 {
		t.Fatalf("**المجموعُ %d والمنتظَر ١٦٠**", sub)
	}

	// **البنود**: صنفان (عادي ×٢ ٢٠، دبل ×٣ ٣٠) ومشروباتُهما بعددها.
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT oi.name, oi.unit_price, oi.merchant_price, oi.qty,
		       COALESCE(p.qty, 0), oi.drink_option_id IS NOT NULL
		  FROM order_items oi LEFT JOIN order_items p ON p.id = oi.drink_for
		 WHERE oi.order_id = $1::uuid AND oi.drink_for IS NOT NULL`, oid)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string][3]int64{}
	for rows.Next() {
		var name string
		var up, mp, parentQty int64
		var qty int
		var hasOpt bool
		if err := rows.Scan(&name, &up, &mp, &qty, &parentQty, &hasOpt); err != nil {
			t.Fatal(err)
		}
		if up != mp || !hasOpt {
			t.Fatalf("**بندُ %s**: بيع %d شراء %d خيار %v", name, up, mp, hasOpt)
		}
		v := got[name]
		v[0] += int64(qty)
		v[1] = up
		got[name] = v
	}
	want := map[string][2]int64{"عيران": {2, 5}, "ببسي": {1, 7}, "كنزا": {1, 6}, "ميرندا": {1, 7}}
	if len(got) != len(want) {
		t.Fatalf("**بنودُ المشروبات**: %v", got)
	}
	for n, w := range want {
		if got[n][0] != w[0] || got[n][1] != w[1] {
			t.Fatalf("**%s**: %v والمنتظَر عدد %d سعر %d", n, got[n], w[0], w[1])
		}
	}
	// **ولا مشروبَ في خيارات الصنف** — فلا يُعرض مرّتين.
	var inOpts bool
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT EXISTS(SELECT 1 FROM order_items oi, jsonb_array_elements(oi.options) e
		               WHERE oi.order_id = $1::uuid AND e->>'group' = 'مشروبات')`, oid).Scan(&inOpts)
	if inOpts {
		t.Fatal("**المشروبُ بقي في خيارات الصنف**")
	}
	// **وقراءةُ الطلب تضع المشروبَ بعد صنفه** ومعه `drink_for`.
	g := h.GET("/api/v1/my/orders/"+oid, tok)
	if g.Code != http.StatusOK || !containsBytes(g.Body, `"drink_for"`) || !containsBytes(g.Body, `"drink_option_id"`) {
		t.Fatalf("**قراءةُ الطلب بلا بنود المشروبات**: %s", g)
	}
}

// **والعميلُ القديمُ بلا الحقل ⇒ ٢٠٨ كما كان** — لا يتبدّل سعرٌ على تطبيقٍ لم يُحدَّث.
func TestDrinkQty_1177_OldClientUnchanged(t *testing.T) {
	h := New(t)
	d := newDrinkFx(t, h, "DRQ-OLD")
	items := d.items1177(false)
	if q := d.quote(t, h, items); q != 208 {
		t.Fatalf("**التسعيرةُ القديمة %d والمنتظَر ٢٠٨**", q)
	}
	r, oid := d.order(t, h, items)
	if r.Code != http.StatusOK && r.Code != http.StatusCreated {
		t.Fatalf("الطلب: %s", r)
	}
	if sub := orderSubtotal(t, h, oid); sub != 208 {
		t.Fatalf("**المجموعُ %d والمنتظَر ٢٠٨**", sub)
	}
	var drinkRows int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM order_items WHERE order_id = $1::uuid AND drink_for IS NOT NULL`, oid).Scan(&drinkRows)
	if drinkRows != 0 {
		t.Fatalf("**العميلُ القديمُ صار له بنودُ مشروبات**: %d", drinkRows)
	}
}

// **العددُ ١..٢٠، والصفرُ «لم يُختر»، والمفتاحُ لغير المشروب يُردّ.**
func TestDrinkQty_Validation(t *testing.T) {
	h := New(t)
	d := newDrinkFx(t, h, "DRQ-VAL")
	id := d.m.fx.Item.ID
	line := func(q map[string]int, opts ...string) []map[string]any {
		return []map[string]any{{"menu_item_id": id, "qty": 1,
			"option_ids": append([]string{d.reg}, opts...), "option_qty": q}}
	}
	for name, c := range map[string]struct {
		items []map[string]any
		code  string
	}{
		"21":       {line(map[string]int{d.ayran: 21}), "bad_qty"},
		"سالب":     {line(map[string]int{d.ayran: -1}), "bad_qty"},
		"غيرمشروب": {line(map[string]int{d.dbl: 1}), "invalid_items"},
	} {
		r, _ := d.order(t, h, c.items)
		if r.Code < 400 || r.Err() != c.code {
			t.Fatalf("**%s**: %d %q والمنتظَر %q", name, r.Code, r.Err(), c.code)
		}
	}
	// ٢٠ تُقبل: ٢٠ + ٢٠×٥ = ١٢٠.
	if q := d.quote(t, h, line(map[string]int{d.ayran: 20})); q != 120 {
		t.Fatalf("**عيران ×٢٠**: %d والمنتظَر ١٢٠", q)
	}
	// **والصفرُ «لم يُختر»** ولو كان في `option_ids` — ٢٠ وحدَها.
	if q := d.quote(t, h, line(map[string]int{d.ayran: 0}, d.ayran)); q != 20 {
		t.Fatalf("**عيران ×٠**: %d والمنتظَر ٢٠", q)
	}
	// **و`{}` هو الحقلُ الجديد** — مشروبٌ في `option_ids` بلا عدد لا يُحسب.
	if q := d.quote(t, h, line(map[string]int{}, d.pepsi)); q != 20 {
		t.Fatalf("**ببسي بلا عدد**: %d والمنتظَر ٢٠", q)
	}
}

// **صنفٌ غير متوفّر يُخرج مشروباتِه معه** — ولا يبقى طلبٌ بمشروباتٍ بلا صنفها.
func TestDrinkQty_ItemUnavailableTakesDrinks(t *testing.T) {
	h := New(t)
	d := newDrinkFx(t, h, "DRQ-UNAV")
	r, oid := d.order(t, h, d.items1177(true))
	if r.Code != http.StatusOK && r.Code != http.StatusCreated {
		t.Fatalf("الطلب: %s", r)
	}
	var parent string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT id::text FROM order_items WHERE order_id = $1::uuid AND drink_for IS NULL AND qty = 2`,
		oid).Scan(&parent); err != nil {
		t.Fatal(err)
	}
	// **بابُ «صنفٌ غير متوفّر» نفسُه** — ٢ عادي ومشروباه (عيران ٥ + ببسي ٧) يخرجون.
	admin := h.NewUser("admin")
	o, res, err := h.Orders.MarkItemUnavailable(ctxBG(), admin.ID, oid, parent, 0, nil)
	if err != nil {
		t.Fatalf("صنفٌ غير متوفّر: %v", err)
	}
	if res.RemovedQty != 2 || o.Subtotal != 160-40-12 {
		t.Fatalf("**بعد الحذف**: محذوف %d ومجموع %d والمنتظَر ١٠٨", res.RemovedQty, o.Subtotal)
	}
	if sub := orderSubtotal(t, h, oid); sub != 108 {
		t.Fatalf("**جمعُ البنود بعد الحذف %d**", sub)
	}
	var left int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM order_items WHERE drink_for = $1::uuid`, parent).Scan(&left)
	if left != 0 {
		t.Fatalf("**بقيت مشروباتُ صنفٍ محذوف**: %d", left)
	}
}
