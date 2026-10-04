package server

// ══════════════════════════════════════════════════════════════════════
// **عقدُ «السوق» في لوحة الإدارة** (`MKT`، قراراتُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
//	MKT1  لا موافقةَ على الصنف — لا مفتاحَ ولا طابورَ ولا صنفَ معلَّق
//	MKT2  لا قسمَ باسمٍ موجودٍ ولو اختلف التشكيل («حلويات» = «حلويّات»)
//	MKT3  حذفُ قسمٍ فيه أصنافٌ ينقلها أوّلاً إلى قسمٍ يختاره الموظّف
//	MKT4  دمجُ الأقسام المكرّرة — يبقى واحدٌ وتنتقل إليه الأصنافُ كلُّها
//	MKT5  «المضافُ حديثاً» — الأحدثُ أوّلاً، وعدّادٌ يُصفَّر بفتح السوق

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// mktCall **ينادي معالِجاً كما يناديه الموجّه** — بمعلَماتِ مسارٍ وصاحبِ طلب.
func mktCall(t *testing.T, h http.HandlerFunc, method, target, body string,
	params map[string]string, uid string) *httptest.ResponseRecorder {
	t.Helper()
	var rd *strings.Reader
	if body != "" {
		rd = strings.NewReader(body)
	} else {
		rd = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rc := chi.NewRouteContext()
	for k, v := range params {
		rc.URLParams.Add(k, v)
	}
	c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	c = context.WithValue(c, ctxRoles, []string{"admin"})
	if uid != "" {
		c = context.WithValue(c, ctxUserID, uid)
	}
	w := httptest.NewRecorder()
	h(w, req.WithContext(c))
	return w
}

func mktJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v — %s", err, w.Body.String())
	}
	if d, ok := out["data"].(map[string]any); ok {
		return d
	}
	return out
}

// mktSection **قسمٌ يُنشأ للفحص ويُحذف بعده.**
func mktSection(t *testing.T, f *driverFixture, name string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO platform_sections (name, sort_order) VALUES ($1, 960)
		RETURNING id::text`, name).Scan(&id); err != nil {
		t.Fatalf("تعذّر القسم %s: %v", name, err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM menu_items WHERE platform_section_id = $1`, id)
		_, _ = f.pool.Exec(c, `DELETE FROM platform_sections WHERE id = $1`, id)
	})
	return id
}

// ── MKT1 · لا موافقةَ على الصنف ─────────────────────────────────────
//
// «ما في داعي للموافقة على الصنف أساساً» — **يُضيفه المتجرُ فيظهر فوراً،
// وتُشال المراجعةُ وإعدادُها من كلّ مكان.**
func TestMKT1_NoApprovalNeededForNewItems(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	// **والمفتاحُ ذهب من الكتالوج** — فلا يُشغَّل يوماً من الإعدادات.
	if _, ok := settings.Lookup("merchants.menu_requires_approval"); ok {
		t.Fatal("**مفتاحُ مراجعة الأصناف ما زال في الكتالوج** — والمالكُ أمر بشيله")
	}
	// **ولا صنفَ معلَّقٌ في القاعدة كلِّها** — هجرةُ ٠٢٠٠ أقرّت ما كان.
	var pending int
	if err := f.pool.QueryRow(ctx,
		`SELECT count(*) FROM menu_items WHERE NOT approved`).Scan(&pending); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if pending != 0 {
		t.Fatalf("**%d صنفاً ينتظر موافقةً لا وجودَ لها**", pending)
	}

	// **والمندوبُ يضيف صنفاً — فيظهر فوراً، ولا كلمةَ عن مراجعة.**
	rep := testdb.NewUser(t, f.pool, "sales")
	owner := testdb.NewUser(t, f.pool, "merchant")
	var cat string
	if err := f.pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&cat); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	var mid string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, owner_user_id, sales_rep_user_id, status, category_id)
		VALUES ('متجرُ فحصِ السوق', $1, $2, 'active', $3) RETURNING id`,
		owner, rep, cat).Scan(&mid); err != nil {
		t.Fatalf("تعذّر المتجر: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM menu_items WHERE merchant_id = $1`, mid)
		_, _ = f.pool.Exec(c, `DELETE FROM merchants WHERE id = $1`, mid)
	})
	sec := mktSection(t, f, "قسمُ فحصِ السوق ١")

	w := mktCall(t, f.srv.handleRepCreateItem, http.MethodPost, "/x",
		`{"name":"صنفٌ بلا مراجعة","price":1000,"platform_section_id":"`+sec+`"}`,
		map[string]string{"id": mid}, rep)
	if w.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الصنف ردّ %d — %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "pending_review") {
		t.Fatalf("**الردُّ ما زال يتكلّم عن مراجعة**: %s", w.Body.String())
	}
	var approved, available bool
	if err := f.pool.QueryRow(ctx, `
		SELECT approved, available FROM menu_items WHERE merchant_id = $1`, mid).
		Scan(&approved, &available); err != nil {
		t.Fatalf("تعذّرت قراءةُ الصنف: %v", err)
	}
	if !approved || !available {
		t.Fatalf("**الصنفُ لم يظهر فوراً**: approved=%v available=%v", approved, available)
	}
}

// ── MKT2 · لا قسمَ باسمٍ موجود — ولو اختلف التشكيل ──────────────────
func TestMKT2_DuplicateSectionNameRefusedNormalized(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	base := "حلويات فحص " + time.Now().Format("150405.000")

	w := mktCall(t, f.srv.handleCreatePlatformSection, http.MethodPost, "/x",
		`{"name":"`+base+`"}`, nil, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ القسم ردّ %d — %s", w.Code, w.Body.String())
	}
	firstID, _ := mktJSON(t, w)["id"].(string)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, firstID)
	})

	// **الاسمُ نفسُه بشدّةٍ وتنوينٍ ومسافتين** — هو هو.
	variant := strings.Replace(base, "حلويات", "حلويّات", 1)
	variant = strings.Replace(variant, "فحص", "فحصٍ ", 1)
	for _, name := range []string{base, variant, "  " + variant + " "} {
		w := mktCall(t, f.srv.handleCreatePlatformSection, http.MethodPost, "/x",
			`{"name":"`+name+`"}`, nil, "")
		if w.Code != http.StatusConflict ||
			!strings.Contains(w.Body.String(), "section_name_taken") {
			t.Fatalf("**قُبل قسمٌ مكرّر** %q: %d — %s", name, w.Code, w.Body.String())
		}
	}

	// **والإعادةُ تسمية قسمٍ آخرَ إلى الاسم نفسِه مرفوضةٌ كذلك.**
	other := mktSection(t, f, "قسمٌ آخر "+base)
	w = mktCall(t, f.srv.handleUpdatePlatformSection, http.MethodPatch, "/x",
		`{"name":"`+variant+`"}`, map[string]string{"id": other}, "")
	if w.Code != http.StatusConflict {
		t.Fatalf("**قُبلت تسميةٌ إلى اسمٍ موجود**: %d — %s", w.Code, w.Body.String())
	}
	// **وحفظُ القسم باسمه لا يُرفض.**
	w = mktCall(t, f.srv.handleUpdatePlatformSection, http.MethodPatch, "/x",
		`{"name":"`+variant+`"}`, map[string]string{"id": firstID}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("**رُفض حفظُ القسم باسمه**: %d — %s", w.Code, w.Body.String())
	}

	// **والقاعدةُ نفسُها لا تقبل اثنين** — الفهرسُ هو الحارسُ الأخير.
	if _, err := f.pool.Exec(ctx,
		`INSERT INTO platform_sections (name) VALUES ($1)`, base); err == nil {
		_, _ = f.pool.Exec(ctx, `DELETE FROM platform_sections WHERE name = $1 AND id <> $2`, base, firstID)
		t.Fatal("**قبلت القاعدةُ قسماً مكرّراً** — لا فهرسَ يحرسه")
	}
}

// ── MKT3 · حذفُ قسمٍ فيه أصنافٌ ينقلها أوّلاً ─────────────────────
func TestMKT3_DeleteSectionMovesItemsToChosenSection(t *testing.T) {
	x := newSectionDeleteFixture(t)
	a := x.addItem(t, "صنفٌ يُنقل ١")
	b := x.addItem(t, "صنفٌ يُنقل ٢")
	target := mktSection(t, x.f, "قسمُ الوجهة "+t.Name())

	// **والوجهةُ لا تكون القسمَ نفسَه.**
	w := mktCall(t, x.f.srv.handleDeletePlatformSection, http.MethodDelete,
		"/x?move_to="+x.sectionID, "", map[string]string{"id": x.sectionID}, "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("**قُبلت وجهةٌ هي القسمُ نفسُه**: %d — %s", w.Code, w.Body.String())
	}

	w = mktCall(t, x.f.srv.handleDeletePlatformSection, http.MethodDelete,
		"/x?move_to="+target, "", map[string]string{"id": x.sectionID}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("**رُدَّ الحذفُ مع النقل**: %d — %s", w.Code, w.Body.String())
	}
	if moved, _ := mktJSON(t, w)["moved_items"].(float64); moved != 2 {
		t.Fatalf("نُقل %v صنفاً لا ٢", moved)
	}
	if x.sectionExists(t, x.sectionID) {
		t.Fatal("**بقي القسمُ بعد حذفه**")
	}
	for _, id := range []string{a, b} {
		var got string
		if err := x.f.pool.QueryRow(context.Background(),
			`SELECT platform_section_id::text FROM menu_items WHERE id = $1`, id).
			Scan(&got); err != nil {
			t.Fatalf("**ضاع الصنف مع قسمه**: %v", err)
		}
		if got != target {
			t.Fatalf("**الصنفُ لم ينتقل إلى الوجهة**: %q", got)
		}
	}
}

// ── MKT4 · دمجُ الأقسام المكرّرة ────────────────────────────────────
//
// **يُبنى التكرارُ داخل معاملةٍ تُرجَع** — الفهرسُ الفريدُ يُرفع فيها وحدَها،
// **فلا يرى أحدٌ خارجَها شيئاً.**
func TestMKT4_MergeDuplicateSectionsKeepsOneWithAllItems(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	// **وبعد الهجرة لا تكرارَ في القاعدة** — قبل أيّ بناء.
	var dupGroups int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM (
			SELECT section_name_key(name) FROM platform_sections
			GROUP BY 1 HAVING count(*) > 1) d`).Scan(&dupGroups); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if dupGroups != 0 {
		t.Fatalf("**%d اسماً مكرّراً بقي بعد الدمج**", dupGroups)
	}

	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("معاملة: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DROP INDEX platform_sections_name_key_uq`); err != nil {
		t.Fatalf("رفعُ الفهرس: %v", err)
	}
	section := func(name string, active bool) string {
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO platform_sections (name, active) VALUES ($1, $2) RETURNING id::text`,
			name, active).Scan(&id); err != nil {
			t.Fatalf("قسم %s: %v", name, err)
		}
		return id
	}
	var menuSec string
	if err := tx.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'قائمةُ الدمج')
		RETURNING id::text`, f.merchantID).Scan(&menuSec); err != nil {
		t.Fatalf("قائمة: %v", err)
	}
	item := func(sec string) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO menu_items (merchant_id, section_id, platform_section_id,
				name, price, merchant_price, available)
			VALUES ($1, $2, $3, 'صنفُ الدمج', 1000, 1000, true)`,
			f.merchantID, menuSec, sec); err != nil {
			t.Fatalf("صنف: %v", err)
		}
	}
	keeper := section("مكسرات فحص الدمج", true)    // فعّالٌ بصنفٍ واحد
	retired := section("مكسّرات فحص الدمج", false) // مُطفأٌ بصنفين — يُدمج رغم أصنافه
	extra := section("مُكسرات  فحصِ الدمج", true)  // فعّالٌ فارغ
	item(keeper)
	item(retired)
	item(retired)

	var merged int
	if err := tx.QueryRow(ctx, `SELECT merge_duplicate_platform_sections()`).Scan(&merged); err != nil {
		t.Fatalf("الدمج: %v", err)
	}
	if merged != 2 {
		t.Fatalf("دُمج %d لا ٢", merged)
	}
	var left []string
	rows, err := tx.Query(ctx, `
		SELECT id::text FROM platform_sections
		 WHERE id = ANY($1::uuid[])`, []string{keeper, retired, extra})
	if err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		left = append(left, id)
	}
	rows.Close()
	if len(left) != 1 || left[0] != keeper {
		t.Fatalf("**بقي %v والمنتظَرُ الفعّالُ الأعمر وحدَه %s**", left, keeper)
	}
	var n int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM menu_items WHERE platform_section_id = $1`, keeper).Scan(&n); err != nil {
		t.Fatalf("عدّ: %v", err)
	}
	if n != 3 {
		t.Fatalf("**في القسم الباقي %d صنفاً لا ٣ — ضاع صنفٌ بالدمج**", n)
	}
}

// ── MKT5 · «المضافُ حديثاً» ──────────────────────────────────────────
func TestMKT5_NewItemsFirstAndCounterResetsOnOpen(t *testing.T) {
	x := newSectionDeleteFixture(t)
	f := x.f
	ctx := context.Background()
	staff := testdb.NewUser(t, f.pool, "admin")

	count := func() int {
		w := mktCall(t, f.srv.handleMarketNewCount, http.MethodGet, "/x", "", nil, staff)
		if w.Code != http.StatusOK {
			t.Fatalf("العدّاد ردّ %d — %s", w.Code, w.Body.String())
		}
		n, _ := mktJSON(t, w)["count"].(float64)
		return int(n)
	}
	seen := func() {
		w := mktCall(t, f.srv.handleMarketSeen, http.MethodPost, "/x", "", nil, staff)
		if w.Code != http.StatusOK {
			t.Fatalf("فتحُ السوق ردّ %d — %s", w.Code, w.Body.String())
		}
	}

	// **صنفٌ قديمٌ** — قبل عشرة أيّام.
	old := x.addItem(t, "صنفٌ قديم")
	if _, err := f.pool.Exec(ctx,
		`UPDATE menu_items SET created_at = now() - interval '10 days' WHERE id = $1`, old); err != nil {
		t.Fatalf("تقديمُ الصنف: %v", err)
	}
	seen()
	if c := count(); c != 0 {
		t.Fatalf("**بعد فتح السوق العدّادُ %d لا صفر**", c)
	}

	first := x.addItem(t, "صنفٌ جديدٌ أوّل")
	time.Sleep(5 * time.Millisecond)
	second := x.addItem(t, "صنفٌ جديدٌ ثانٍ")
	if c := count(); c != 2 {
		t.Fatalf("**أُضيف صنفان والعدّادُ %d**", c)
	}

	w := mktCall(t, f.srv.handleMarketItems, http.MethodGet,
		"/x?section="+x.sectionID, "", nil, staff)
	if w.Code != http.StatusOK {
		t.Fatalf("الأصنافُ ردّت %d — %s", w.Code, w.Body.String())
	}
	items, _ := mktJSON(t, w)["items"].([]any)
	if len(items) != 3 {
		t.Fatalf("الأصنافُ %d لا ٣", len(items))
	}
	want := []struct {
		id    string
		isNew bool
	}{{second, true}, {first, true}, {old, false}}
	for i, wt := range want {
		row, _ := items[i].(map[string]any)
		if row["id"] != wt.id || row["is_new"] != wt.isNew {
			t.Fatalf("**الصفُّ %d: %v/%v والمنتظَرُ %s/%v** — الأحدثُ أوّلاً بعلامة «جديد»",
				i, row["id"], row["is_new"], wt.id, wt.isNew)
		}
		if row["merchant_name"] == "" || row["created_at"] == nil {
			t.Fatalf("**الصفُّ بلا اسمِ متجرٍ أو وقتِ إضافة**: %v", row)
		}
	}

	// **ويُصفَّر بفتح السوق** — لهذا الموظّف وحدَه.
	seen()
	if c := count(); c != 0 {
		t.Fatalf("**فُتح السوقُ والعدّادُ %d**", c)
	}
	other := testdb.NewUser(t, f.pool, "admin")
	w = mktCall(t, f.srv.handleMarketNewCount, http.MethodGet, "/x", "", nil, other)
	if n, _ := mktJSON(t, w)["count"].(float64); n < 2 {
		t.Fatalf("**صُفِّر عدّادُ زميلٍ لم يفتح السوق**: %v", n)
	}
}
