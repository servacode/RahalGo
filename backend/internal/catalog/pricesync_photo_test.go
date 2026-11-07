package catalog

import (
	"context"
	"testing"
)

type photoState struct {
	available, hold, created bool
	qty                      *float64
}

func (f *menuFixture) photoState(t *testing.T, code string) (string, photoState) {
	t.Helper()
	var id string
	var st photoState
	if err := f.pool.QueryRow(context.Background(), `
		SELECT id::text, available, photo_hold, ameen_created, ameen_qty FROM menu_items
		 WHERE merchant_id = $1 AND external_code = $2`, f.merchantID, code).
		Scan(&id, &st.available, &st.hold, &st.created, &st.qty); err != nil {
		t.Fatalf("الصنف %s: %v", code, err)
	}
	return id, st
}

func (f *menuFixture) media(t *testing.T, name string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO media (kind, path, thumb_path, width, height, bytes)
		VALUES ('menu_item', 'test/'||$1||'.jpg', 'test/'||$1||'_thumb.jpg', 10, 10, 8)
		RETURNING id::text`, name).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `UPDATE menu_items SET image_media_id = NULL WHERE image_media_id = $1`, id)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM media WHERE id = $1`, id)
	})
	return id
}

// TestSyncPrices_PhotoRule **«الصنف بلا صورة يبقى مخفياً، وبس تنزل صورة يصير متوفر»**
// (قرار المالك ٢٠٢٦-١١-٠٧): صنف من الأمين كميته فوق الصفر وبلا صورة بيضلّ مخفي، وأوّل صورة
// بتخلّيه متوفر — إلّا إذا الكمية صفر؛ وما منقلب شي غيّره المتجر بإيده؛ والصنف اليدوي متل قبل.
func TestSyncPrices_PhotoRule(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	opt := SyncOptions{Currency: CurrencySYP, Source: "file", ActorID: f.actorID, CreateMissing: true}

	// صنف يدوي مربوط بالأمين وبلا صورة — سلوكه ما بيتغيّر.
	hand := f.priced(t, "زيت يدوي", 100)
	if _, err := f.pool.Exec(ctx, `UPDATE menu_items SET external_code = 'PH-H', available = false WHERE id = $1`, hand); err != nil {
		t.Fatal(err)
	}

	rows := []SyncRow{
		{Code: "PH-A", Name: "صنف كميته موجودة", Price: f64(10), Qty: f64(5)},
		{Code: "PH-B", Name: "صنف كميته صفر", Price: f64(10), Qty: f64(0)},
		{Code: "PH-C", Name: "صنف بلا كمية", Price: f64(10)},
		{Code: "PH-D", Name: "صنف بيخفيه المتجر", Price: f64(10), Qty: f64(2)},
		{Code: "PH-H", Name: "زيت يدوي", Price: f64(100), Qty: f64(3)},
	}
	rep, err := f.svc.SyncPrices(ctx, f.merchantID, rows, opt)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Created != 4 || rep.KeptHiddenNoPhoto != 3 {
		t.Fatalf("التقرير: أُنشئ %d ومخفي لأنو بلا صورة %d — والمطلوب ٤ و٣", rep.Created, rep.KeptHiddenNoPhoto)
	}
	if st := f.state(t, hand); !st.available {
		t.Fatal("**الصنف اليدوي ما صار متوفر بكميته** — قاعدة الصورة لأصناف الأمين بس")
	}
	idA, a := f.photoState(t, "PH-A")
	idB, b := f.photoState(t, "PH-B")
	idC, c := f.photoState(t, "PH-C")
	idD, _ := f.photoState(t, "PH-D")
	if a.available || !a.hold || !a.created || a.qty == nil || *a.qty != 5 {
		t.Fatalf("PH-A: %+v — والمطلوب مخفي وبينطر صورة وكميته ٥", a)
	}
	if b.available || b.hold {
		t.Fatalf("PH-B: %+v — كميته صفر فما بينطر صورة", b)
	}
	if c.available || !c.hold {
		t.Fatalf("PH-C: %+v — بلا كمية بينطر صورته", c)
	}

	// المزامنة التانية: كمية فوق الصفر وبلا صورة ⇒ بيضلّ مخفي.
	rep2, err := f.svc.SyncPrices(ctx, f.merchantID, []SyncRow{
		{Code: "PH-A", Qty: f64(7)}, {Code: "PH-B", Qty: f64(0)},
	}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if rep2.MadeAvailable != 0 || rep2.KeptHiddenNoPhoto != 1 {
		t.Fatalf("المزامنة التانية: %+v — **صنف بلا صورة صار متوفر**", rep2)
	}
	if _, a = f.photoState(t, "PH-A"); a.available || !a.hold || *a.qty != 7 {
		t.Fatalf("PH-A بعد المزامنة التانية: %+v", a)
	}

	// أوّل صورة ⇒ متوفر. والمحرّر بيبعت التوفّر متل ما هو (false) مع الصورة.
	if err := f.svc.UpdateItem(ctx, f.actorID, idA, MenuItemInput{ImageMediaID: ptr(f.media(t, "pha")), Available: ptr(false)}, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, a = f.photoState(t, "PH-A"); !a.available || a.hold {
		t.Fatalf("PH-A بعد الصورة: %+v — **والمطلوب متوفر**", a)
	}
	// كميته صفر ⇒ الصورة ما بتظهّره.
	if err := f.svc.UpdateItem(ctx, f.actorID, idB, MenuItemInput{ImageMediaID: ptr(f.media(t, "phb"))}, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, b = f.photoState(t, "PH-B"); b.available {
		t.Fatal("**PH-B كميته صفر وصار متوفر بالصورة**")
	}
	// بلا كمية من الأمين ⇒ الصورة بتظهّره.
	if err := f.svc.UpdateItem(ctx, f.actorID, idC, MenuItemInput{ImageMediaID: ptr(f.media(t, "phc"))}, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, c = f.photoState(t, "PH-C"); !c.available {
		t.Fatal("PH-C ما صار متوفر بالصورة")
	}

	// المتجر خفّى الصنف بإيده بعد ما صار إلو صورة ⇒ صورة تانية ما بتقلبه.
	if err := f.svc.UpdateItem(ctx, f.actorID, idD, MenuItemInput{ImageMediaID: ptr(f.media(t, "phd1"))}, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.UpdateItem(ctx, f.actorID, idD, MenuItemInput{Available: ptr(false)}, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.UpdateItem(ctx, f.actorID, idD, MenuItemInput{ImageMediaID: ptr(f.media(t, "phd2"))}, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, d := f.photoState(t, "PH-D"); d.available {
		t.Fatal("**PH-D خفّاه المتجر بإيده وانقلب متوفر بالصورة**")
	}
	// وصنف بلا صورة خفّاه المتجر بإيده (بعد ما ظهّره) ما بينقلب بأوّل صورة كمان.
	if _, err := f.pool.Exec(ctx, `UPDATE menu_items SET image_media_id = NULL, photo_hold = true, available = true WHERE id = $1`, idD); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.UpdateItem(ctx, f.actorID, idD, MenuItemInput{Available: ptr(false)}, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.UpdateItem(ctx, f.actorID, idD, MenuItemInput{ImageMediaID: ptr(f.media(t, "phd3"))}, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, d := f.photoState(t, "PH-D"); d.available {
		t.Fatal("**PH-D اختار المتجر يخفيه وانقلب بأوّل صورة**")
	}

	// صنف يدوي بلا صورة وبلا علامة ما بيتغيّر توفّره بصورة.
	if _, err := f.pool.Exec(ctx, `UPDATE menu_items SET available = false WHERE id = $1`, hand); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.UpdateItem(ctx, f.actorID, hand, MenuItemInput{ImageMediaID: ptr(f.media(t, "phh"))}, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if st := f.state(t, hand); st.available {
		t.Fatal("**صنف يدوي مخفي صار متوفر بالصورة**")
	}
}

// TestMoveRestToMisc **«حطّ الباقي بقسم منوّعات»**: كلّ اللي بالانتظار بينتقل يدويّاً، والباقي ما بينمسّ.
func TestMoveRestToMisc(t *testing.T) {
	f := newMenuFixture(t)
	ctx := context.Background()
	old := suggestSection
	suggestSection = func(string) (string, bool) { return "", false }
	defer func() { suggestSection = old }()

	other := f.priced(t, "صنف بقسمه", 50)
	if _, err := f.svc.SyncPrices(ctx, f.merchantID, []SyncRow{
		{Code: "MR-1", Name: "مادة أولى", Price: f64(5)},
		{Code: "MR-2", Name: "مادة تانية", Price: f64(5)},
	}, SyncOptions{Currency: CurrencySYP, Source: "file", CreateMissing: true}); err != nil {
		t.Fatal(err)
	}
	if n, err := f.svc.PendingSyncCount(ctx, f.merchantID); err != nil || n != 2 {
		t.Fatalf("بالانتظار %d %v — والمطلوب ٢", n, err)
	}
	n, err := f.svc.MoveRestToMisc(ctx, f.merchantID, f.actorID)
	if err != nil || n != 2 {
		t.Fatalf("انتقل %d %v — والمطلوب ٢", n, err)
	}
	var misc, src string
	if err := f.pool.QueryRow(ctx, `
		SELECT ps.name, i.section_source FROM menu_items i JOIN platform_sections ps ON ps.id = i.platform_section_id
		 WHERE i.merchant_id = $1 AND i.external_code = 'MR-1'`, f.merchantID).Scan(&misc, &src); err != nil {
		t.Fatal(err)
	}
	if misc != MiscSectionName || src != SourceManual {
		t.Fatalf("MR-1 بـ%q مصدره %q — والمطلوب منوّعات ويدويّ", misc, src)
	}
	var sec string
	if err := f.pool.QueryRow(ctx, `SELECT platform_section_id::text FROM menu_items WHERE id = $1`, other).Scan(&sec); err != nil || sec != f.platformID {
		t.Fatalf("صنف بقسمه انتقل: %s %v", sec, err)
	}
	if n, _ := f.svc.PendingSyncCount(ctx, f.merchantID); n != 0 {
		t.Fatalf("لسّا بالانتظار %d", n)
	}
	if n, err := f.svc.MoveRestToMisc(ctx, f.merchantID, f.actorID); err != nil || n != 0 {
		t.Fatalf("المرّة التانية %d %v", n, err)
	}
}
