package catalog

import (
	"context"
	"errors"
	"strings"
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

// TestDrinks_ImageReachesItemOption **صورةُ المشروب تصل الزبونَ مع الصنف** (قرارُ المالك
// ٢٠٢٦-١٠-٣٠): تُحفظ في القائمة، وتُنسخ إلى الخيار، **وتخرج مصغّرتُها من خيارات الصنف.
// والمشروبُ بلا صورةٍ يبقى كما كان — بلا رابط.**
func TestDrinks_ImageReachesItemOption(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	var mediaID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO media (kind, path, thumb_path, width, height, bytes)
		VALUES ('menu_item', 'test/drink.jpg', 'test/drink_thumb.jpg', 10, 10, 8)
		RETURNING id::text`).Scan(&mediaID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchant_drinks WHERE merchant_id = $1`, f.merchantID)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM media WHERE id = $1`, mediaID)
	})
	item := f.item(t, "شاورما", nil)

	ds, err := f.svc.SetDrinks(ctx, f.actorID, f.merchantID, []Drink{
		{Name: "عيران", Price: 5, ImageMediaID: &mediaID}, {Name: "ببسي", Price: 7},
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("حفظُ القائمة: %v", err)
	}
	if len(ds) != 2 || ds[0].ImageMediaID == nil || *ds[0].ImageMediaID != mediaID ||
		ds[0].ImageThumbURL == nil || !strings.HasSuffix(*ds[0].ImageThumbURL, "test/drink_thumb.jpg") ||
		ds[1].ImageMediaID != nil || ds[1].ImageThumbURL != nil {
		t.Fatalf("القائمةُ لم تحمل الصورة كما يجب: %+v", ds)
	}

	groups, err := f.svc.ItemModifiers(ctx, item)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, g := range groups {
		if g.Name != DrinksGroupName {
			continue
		}
		for _, o := range g.Options {
			switch o.Name {
			case "عيران":
				seen++
				if o.ImageThumbURL == nil || !strings.HasSuffix(*o.ImageThumbURL, "test/drink_thumb.jpg") {
					t.Fatalf("خيارُ العيران بلا مصغّرة: %+v", o)
				}
			case "ببسي":
				seen++
				if o.ImageThumbURL != nil {
					t.Fatalf("ببسي بلا صورةٍ خرج برابط: %s", *o.ImageThumbURL)
				}
			}
		}
	}
	if seen != 2 {
		t.Fatalf("خياراتُ المشروبات في الصنف: %d — والمطلوب ٢", seen)
	}

	// **ومعرّفٌ لا يوجد يُردّ تحقّقاً** — لا ٥٠٠ من المفتاح الأجنبيّ.
	bad := "00000000-0000-0000-0000-000000000001"
	if _, err := f.svc.SetDrinks(ctx, f.actorID, f.merchantID, []Drink{{Name: "عيران", Price: 5, ImageMediaID: &bad}}, "127.0.0.1"); !errors.Is(err, ErrNameRequired) {
		t.Fatalf("صورةٌ لا توجد: %v — والمطلوب تحقّق", err)
	}
}

// TestDrinks_RestaurantsOnly **المشروباتُ للمطاعم وحدَها** (قرارُ المالك ٢٠٢٦-١٠-٣٠):
// متجرٌ تصنيفُه بلا `has_drinks` **لا تُنسخ إلى أصنافه، ولا تُحفظ له، وقائمتُه فارغة.**
func TestDrinks_RestaurantsOnly(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `
		UPDATE categories SET has_drinks = false
		 WHERE id = (SELECT category_id FROM merchants WHERE id = $1)`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	if ok, err := f.svc.DrinksAllowed(ctx, f.merchantID); err != nil || ok {
		t.Fatalf("متجرٌ غيرُ مطعمٍ مسموحٌ له: %v %v", ok, err)
	}
	if _, err := f.svc.SetDrinks(ctx, f.actorID, f.merchantID, []Drink{{Name: "عيران", Price: 5}}, "127.0.0.1"); !errors.Is(err, ErrDrinksNotAllowed) {
		t.Fatalf("حفظُ مشروباتٍ لغير مطعم: %v — والمطلوب drinks_not_allowed", err)
	}

	// **وقائمةٌ بقيت من قبل لا تُنسخ** — صنفٌ جديدٌ يُنشأ بلا مجموعة.
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO merchant_drinks (merchant_id, name, price, sort_order) VALUES ($1, 'عيران', 5, 0)`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchant_drinks WHERE merchant_id = $1`, f.merchantID)
	})
	item := f.item(t, "حذاء", nil)
	if n, _ := drinkGroups(t, f, item); n != 0 {
		t.Fatalf("غيرُ المطعم نال مجموعةَ مشروبات: %d", n)
	}
	if ds, err := f.svc.Drinks(ctx, f.merchantID); err != nil || len(ds) != 0 {
		t.Fatalf("قائمةُ غير المطعم: %+v %v — والمطلوب فارغة", ds, err)
	}
}
