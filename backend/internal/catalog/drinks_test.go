package catalog

import (
	"context"
	"testing"
)

// drinkGroups **مجموعاتُ «مشروبات» المنسوخة لصنف** — عددُها وأسماءُ خياراتها وأسعارُها.
func drinkGroups(t *testing.T, f *menuFixture, itemID string) (int, map[string]int64) {
	t.Helper()
	ctx := context.Background()
	var n int
	if err := f.pool.QueryRow(ctx,
		`SELECT count(*) FROM modifier_groups WHERE item_id = $1 AND store_drinks`, itemID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	rows, err := f.pool.Query(ctx, `
		SELECT o.name, o.price_delta FROM modifier_options o
		JOIN modifier_groups g ON g.id = o.group_id
		WHERE g.item_id = $1 AND g.store_drinks`, itemID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	opts := map[string]int64{}
	for rows.Next() {
		var name string
		var p int64
		if err := rows.Scan(&name, &p); err != nil {
			t.Fatal(err)
		}
		opts[name] = p
	}
	return n, opts
}

func (f *menuFixture) item(t *testing.T, name string, mods *[]ModifierGroupInput) string {
	t.Helper()
	id, err := f.svc.CreateItem(context.Background(), f.actorID, f.merchantID, MenuItemInput{
		SectionID: ptr(f.sectionID), Name: ptr(name), Price: ptr(int64(100)),
		PlatformSectionID: ptr(f.platformID), Modifiers: mods,
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("إنشاءُ الصنف: %v", err)
	}
	return id
}

// TestDrinks_CopiedToAllItems **القائمةُ تُكتب مرّةً وتنضاف لكلّ الأصناف** (قرارُ المالك
// ٢٠٢٦-١٠-١٩) — **والجديدُ ينالها، وحفظُ الصنف لا يمحوها، والمحرّرُ لا يراها، والدولارُ
// يتبع الصرف، والفارغةُ تمحوها.**
func TestDrinks_CopiedToAllItems(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	size := []ModifierGroupInput{{Name: "الحجم", MinSelect: 1, MaxSelect: 1,
		Options: []ModifierOptionInput{{Name: "عادي"}, {Name: "دبل", PriceDelta: 20}}}}
	a := f.item(t, "شاورما", &size)
	b := f.item(t, "برغر", nil)

	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET exchange_rate = 120 WHERE id = $1`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	usd := 0.5
	if _, err := f.svc.SetDrinks(ctx, f.actorID, f.merchantID, []Drink{
		{Name: "بيبسي", Price: 5}, {Name: "عصير", PriceUSD: &usd},
	}, "127.0.0.1"); err != nil {
		t.Fatalf("حفظُ القائمة: %v", err)
	}
	for _, id := range []string{a, b} {
		n, opts := drinkGroups(t, f, id)
		if n != 1 || opts["بيبسي"] != 5 || opts["عصير"] != 60 {
			t.Fatalf("الصنف %s: مجموعات %d خيارات %v — **والمطلوب مجموعةٌ واحدة: بيبسي ٥ وعصير ٦٠**", id, n, opts)
		}
	}

	// **والصنفُ الجديدُ ينالها.**
	c := f.item(t, "بيتزا", nil)
	if n, opts := drinkGroups(t, f, c); n != 1 || len(opts) != 2 {
		t.Fatalf("الصنفُ الجديد: مجموعات %d خيارات %v", n, opts)
	}

	// **وحفظُ الصنف بمجموعاته لا يمحوها.**
	if err := f.svc.UpdateItem(ctx, f.actorID, a, MenuItemInput{Modifiers: &size}, "127.0.0.1"); err != nil {
		t.Fatalf("حفظُ الصنف: %v", err)
	}
	if n, _ := drinkGroups(t, f, a); n != 1 {
		t.Fatalf("حفظُ الصنف محا المشروبات أو كرّرها: %d", n)
	}

	// **والمحرّرُ لا يراها** — مجموعةُ الحجم وحدَها.
	menu, err := f.svc.GetMenu(ctx, f.merchantID)
	if err != nil {
		t.Fatal(err)
	}
	for _, sec := range menu {
		for _, it := range sec.Items {
			for _, g := range it.Modifiers {
				if g.Name == DrinksGroupName {
					t.Fatalf("المحرّرُ يرى المشروبات في %s — **فيحفظها نسخةً ثانية**", it.Name)
				}
			}
		}
	}

	// **والدولارُ يتبع الصرف.**
	if _, err := f.svc.SetExchangeRate(ctx, f.actorID, f.merchantID, 130, true, "127.0.0.1"); err != nil {
		t.Fatalf("سعرُ الصرف: %v", err)
	}
	if _, opts := drinkGroups(t, f, b); opts["عصير"] != 65 || opts["بيبسي"] != 5 {
		t.Fatalf("بعد الصرف: %v — **والمطلوب عصير ٦٥ وبيبسي ٥**", opts)
	}
	if ds, _ := f.svc.Drinks(ctx, f.merchantID); len(ds) != 2 || ds[1].Price != 65 {
		t.Fatalf("القائمةُ لم تتبع الصرف: %+v", ds)
	}

	// **والفارغةُ تمحوها من الكلّ.**
	if _, err := f.svc.SetDrinks(ctx, f.actorID, f.merchantID, nil, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{a, b, c} {
		if n, _ := drinkGroups(t, f, id); n != 0 {
			t.Fatalf("القائمةُ الفارغة لم تمحُ المشروبات من %s", id)
		}
	}
}
