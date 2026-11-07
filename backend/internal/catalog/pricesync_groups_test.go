package catalog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// groupsFixture **متجر وقسمين للتوزيع** — والقسمين قبل المتجر لينحذفوا بعده.
type groupsFixture struct {
	*menuFixture
	secA, secB string
	nameA      string
}

func newGroupsFixture(t *testing.T) *groupsFixture {
	t.Helper()
	ctx := context.Background()
	pool := testdb.Pool(t)
	g := &groupsFixture{nameA: "قسم توزيع أ للاختبار"}
	for _, x := range []struct {
		name string
		id   *string
	}{{g.nameA, &g.secA}, {"قسم توزيع ب للاختبار", &g.secB}} {
		if err := pool.QueryRow(ctx, `INSERT INTO platform_sections (name, sort_order) VALUES ($1, 950) RETURNING id::text`,
			x.name).Scan(x.id); err != nil {
			t.Fatal(err)
		}
		id := *x.id
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, id)
		})
	}
	g.menuFixture = newMenuFixture(t)
	if _, err := pool.Exec(ctx, `UPDATE merchants SET uses_ameen = true WHERE id = $1`, g.merchantID); err != nil {
		t.Fatal(err)
	}
	// **المقترح بالاختبار**: «بيبسي» ⇒ القسم أ، وغيره ما في اقتراح.
	old := suggestSection
	suggestSection = func(name string) (string, bool) {
		if strings.Contains(name, "بيبسي") {
			return g.nameA, true
		}
		return "", false
	}
	t.Cleanup(func() { suggestSection = old })
	return g
}

type itemSec struct {
	section, source, group string
}

func (g *groupsFixture) item(t *testing.T, code string) itemSec {
	t.Helper()
	var s itemSec
	if err := g.pool.QueryRow(context.Background(), `
		SELECT platform_section_id::text, section_source, COALESCE(external_group, '')
		  FROM menu_items WHERE merchant_id = $1 AND external_code = $2`, g.merchantID, code).
		Scan(&s.section, &s.source, &s.group); err != nil {
		t.Fatalf("%s: %v", code, err)
	}
	return s
}

func (g *groupsFixture) itemID(t *testing.T, code string) string {
	t.Helper()
	var id string
	if err := g.pool.QueryRow(context.Background(),
		`SELECT id::text FROM menu_items WHERE merchant_id = $1 AND external_code = $2`, g.merchantID, code).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (g *groupsFixture) sync(t *testing.T, rows []SyncRow) *SyncReport {
	t.Helper()
	rep, err := g.svc.SyncPrices(context.Background(), g.merchantID, rows, SyncOptions{
		Currency: CurrencySYP, Source: "file", ActorID: g.actorID, CreateMissing: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

// TestSyncPrices_PlacesNewItems **الصنف الجديد: قسم مجموعته، وإلّا من اسمه، وإلّا الانتظار.**
func TestSyncPrices_PlacesNewItems(t *testing.T) {
	g := newGroupsFixture(t)
	ctx := context.Background()
	if _, err := g.pool.Exec(ctx, `INSERT INTO price_sync_groups (merchant_id, group_key, group_name, platform_section_id)
		VALUES ($1, $2, 'مشروبات', $3)`, g.merchantID, groupKey("مشروبات"), g.secB); err != nil {
		t.Fatal(err)
	}
	rows := []SyncRow{
		{Code: "G1", Name: "عصير برتقال", Group: "مشروبات", Price: f64(10)},
		{Code: "N1", Name: "بيبسي علبة", Group: "غير مربوطة", Price: f64(10)},
		{Code: "P1", Name: "شي غريب", Price: f64(10)},
	}
	dry, err := g.svc.SyncPrices(ctx, g.merchantID, rows, SyncOptions{Currency: CurrencySYP, DryRun: true, CreateMissing: true})
	if err != nil {
		t.Fatal(err)
	}
	if dry.CreatedByGroup != 1 || dry.CreatedByName != 1 || dry.CreatedPending != 1 {
		t.Fatalf("«جرّب»: %+v", dry)
	}
	rep := g.sync(t, rows)
	if rep.Created != 3 || rep.CreatedByGroup != 1 || rep.CreatedByName != 1 || rep.CreatedPending != 1 {
		t.Fatalf("التقرير: %+v", rep)
	}
	pend, err := pendingSectionID(ctx, g.pool)
	if err != nil || pend == "" {
		t.Fatalf("قسم الانتظار: %q %v", pend, err)
	}
	if s := g.item(t, "G1"); s.section != g.secB || s.source != SourceGroup || s.group != "مشروبات" {
		t.Fatalf("G1: %+v", s)
	}
	if s := g.item(t, "N1"); s.section != g.secA || s.source != SourceName || s.group != "غير مربوطة" {
		t.Fatalf("N1: %+v", s)
	}
	if s := g.item(t, "P1"); s.section != pend || s.source != SourcePending {
		t.Fatalf("P1: %+v", s)
	}
	var n int
	var sec *string
	if err := g.pool.QueryRow(ctx, `SELECT count(*) OVER (), platform_section_id::text FROM price_sync_groups
		WHERE merchant_id = $1 AND group_key = $2`, g.merchantID, groupKey("غير مربوطة")).Scan(&n, &sec); err != nil || sec != nil {
		t.Fatalf("المجموعة الجديدة لازم تنحفظ بلا قسم: %v %v", sec, err)
	}
}

// TestPriceSyncGroups_PutMovesButNotManual **قسم المجموعة بينقل الانتظار والتلقائي، واليدويّ لا.**
func TestPriceSyncGroups_PutMovesButNotManual(t *testing.T) {
	g := newGroupsFixture(t)
	ctx := context.Background()
	g.sync(t, []SyncRow{
		{Code: "A1", Name: "شي أوّل", Group: "منظفات", Price: f64(5)},
		{Code: "A2", Name: "بيبسي كبير", Group: "منظفات", Price: f64(5)},
		{Code: "A3", Name: "شي تالت", Group: "منظفات", Price: f64(5)},
	})
	// A3 نقله صاحب المتجر بإيده.
	if _, err := g.svc.MoveSyncItems(ctx, g.merchantID, g.actorID, []string{g.itemID(t, "A3")}, g.platformID); err != nil {
		t.Fatal(err)
	}
	view, err := g.svc.PriceSyncGroups(ctx, g.merchantID)
	if err != nil {
		t.Fatal(err)
	}
	var found *SyncGroup
	for i := range view.Groups {
		if view.Groups[i].Key == groupKey("منظفات") {
			found = &view.Groups[i]
		}
	}
	if found == nil || found.Items != 3 || found.Waiting != 1 || found.SuggestedSectionID == nil || *found.SuggestedSectionID != g.secA {
		t.Fatalf("المجموعة: %+v", found)
	}
	moved, err := g.svc.SetPriceSyncGroups(ctx, g.merchantID, g.actorID, []GroupChoice{{Key: "منظفات", SectionID: g.secB}})
	if err != nil {
		t.Fatal(err)
	}
	if moved != 2 {
		t.Fatalf("انتقل %d والمطلوب ٢", moved)
	}
	for _, c := range []string{"A1", "A2"} {
		if s := g.item(t, c); s.section != g.secB || s.source != SourceGroup {
			t.Fatalf("%s: %+v", c, s)
		}
	}
	if s := g.item(t, "A3"); s.section != g.platformID || s.source != SourceManual {
		t.Fatalf("اليدويّ تحرّك: %+v", s)
	}
	// والمزامنة التالية بتحطّ صنف جديد بنفس المجموعة بقسمها.
	g.sync(t, []SyncRow{{Code: "A4", Name: "شي رابع", Group: "منظفات", Price: f64(5)}})
	if s := g.item(t, "A4"); s.section != g.secB || s.source != SourceGroup {
		t.Fatalf("A4: %+v", s)
	}
	// وقسم مطفي أو مو موجود بينرفض.
	if _, err := g.svc.SetPriceSyncGroups(ctx, g.merchantID, g.actorID, []GroupChoice{{Key: "منظفات", SectionID: "00000000-0000-0000-0000-000000000000"}}); err != ErrSectionRequired {
		t.Fatalf("قسم غلط: %v", err)
	}
}

// TestUpdateItem_SectionChangeIsManual **تعديل القسم من محرّر الصنف بيقلبه يدويّ — ونفس القسم لا.**
func TestUpdateItem_SectionChangeIsManual(t *testing.T) {
	g := newGroupsFixture(t)
	ctx := context.Background()
	g.sync(t, []SyncRow{{Code: "U1", Name: "بيبسي صغير", Price: f64(5)}})
	id := g.itemID(t, "U1")
	if err := g.svc.UpdateItem(ctx, g.actorID, id, MenuItemInput{PlatformSectionID: ptr(g.secA), Name: ptr("بيبسي صغير")}, ""); err != nil {
		t.Fatal(err)
	}
	if s := g.item(t, "U1"); s.source != SourceName {
		t.Fatalf("نفس القسم قلبه: %+v", s)
	}
	if err := g.svc.UpdateItem(ctx, g.actorID, id, MenuItemInput{PlatformSectionID: ptr(g.secB)}, ""); err != nil {
		t.Fatal(err)
	}
	if s := g.item(t, "U1"); s.source != SourceManual || s.section != g.secB {
		t.Fatalf("لازم يصير يدويّ: %+v", s)
	}
}

// TestAutoSortAndReview **«وزّع تلقائياً» من الاسم، والباقي بالمراجعة صفحة صفحة ونقل بالإيد.**
func TestAutoSortAndReview(t *testing.T) {
	g := newGroupsFixture(t)
	ctx := context.Background()
	g.sync(t, []SyncRow{
		{Code: "S1", Name: "مادة أولى", Price: f64(5)},
		{Code: "S2", Name: "مادة تانية", Price: f64(5)},
		{Code: "S3", Name: "مادة تالتة", Price: f64(5)},
	})
	// صار في اقتراح بعد الاستيراد (متل لمّا يندمج المقترح الحقيقي).
	old := suggestSection
	suggestSection = func(name string) (string, bool) {
		if name == "مادة أولى" {
			return g.nameA, true
		}
		return "", false
	}
	defer func() { suggestSection = old }()
	res, err := g.svc.AutoSortPending(ctx, g.merchantID, g.actorID)
	if err != nil {
		t.Fatal(err)
	}
	if res.ByName != 1 || res.Waiting != 2 {
		t.Fatalf("التوزيع: %+v", res)
	}
	if s := g.item(t, "S1"); s.section != g.secA || s.source != SourceName {
		t.Fatalf("S1: %+v", s)
	}
	p1, err := g.svc.PendingItems(ctx, g.merchantID, "", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if p1.Total != 2 || len(p1.Items) != 1 || p1.Next == "" {
		t.Fatalf("الصفحة ١: %+v", p1)
	}
	p2, err := g.svc.PendingItems(ctx, g.merchantID, p1.Next, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(p2.Items) != 1 || p2.Next != "" || p2.Items[0].ID == p1.Items[0].ID {
		t.Fatalf("الصفحة ٢: %+v", p2)
	}
	q, err := g.svc.PendingItems(ctx, g.merchantID, "", "تالتة", 50)
	if err != nil || q.Total != 1 || q.Items[0].Name != "مادة تالتة" {
		t.Fatalf("البحث: %+v %v", q, err)
	}
	n, err := g.svc.MoveSyncItems(ctx, g.merchantID, g.actorID, []string{p1.Items[0].ID, p2.Items[0].ID}, g.secB)
	if err != nil || n != 2 {
		t.Fatalf("النقل: %d %v", n, err)
	}
	for _, c := range []string{"S2", "S3"} {
		if s := g.item(t, c); s.section != g.secB || s.source != SourceManual {
			t.Fatalf("%s: %+v", c, s)
		}
	}
	if left, _ := g.svc.PendingItems(ctx, g.merchantID, "", "", 50); left.Total != 0 {
		t.Fatalf("لسّا في: %d", left.Total)
	}
}

// TestMigration0520_ExistingItemsStay **الهجرة ما بتحرّك شي**: الصنف الموجود يدويّ، وبس اللي
// قاعد بقسم الانتظار بيصير pending.
func TestMigration0520_ExistingItemsStay(t *testing.T) {
	g := newGroupsFixture(t)
	ctx := context.Background()
	normal := g.priced(t, "صنف قديم", 10)
	pend, err := ensurePendingSection(ctx, g.pool)
	if err != nil {
		t.Fatal(err)
	}
	var waiting string
	if err := g.pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, platform_section_id, name, price, merchant_price, available, section_source)
		VALUES ($1, $2, 'قديم بالانتظار', 1, 1, false, 'manual') RETURNING id::text`, g.merchantID, pend).Scan(&waiting); err != nil {
		t.Fatal(err)
	}
	sql, err := os.ReadFile(filepath.Join("..", "migrate", "migrations", "0520_price_sync_groups.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.pool.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("الهجرة مرّة تانية: %v", err)
	}
	src := func(id string) string {
		var s string
		if err := g.pool.QueryRow(ctx, `SELECT section_source FROM menu_items WHERE id = $1`, id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	if s := src(normal); s != SourceManual {
		t.Fatalf("الصنف القديم صار %s", s)
	}
	if s := src(waiting); s != SourcePending {
		t.Fatalf("صنف الانتظار صار %s", s)
	}
}
