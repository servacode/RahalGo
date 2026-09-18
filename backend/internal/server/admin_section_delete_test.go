package server

// ══════════════════════════════════════════════════════════════════════
// **عقدُ حذفِ قسم السوق** (`SD`، ٢٠٢٦-٠٩-١٨)
// ══════════════════════════════════════════════════════════════════════
//
// # عقدان لا يجتمعان
//
// **`menu_items.platform_section_id` موصوفٌ `NOT NULL`** (هجرة ٠١١٨،
// **بقرار المالك ٢٠٢٦-٠٨-٢٢**: «إضافة الصنف القسم إلزامي») —
// **ومفتاحُه بقي `ON DELETE SET NULL`** من هجرة ٠٠٥٧، **يومَ كان الفراغُ
// مشروعاً** ونصُّها يقوله: «وفراغُه مشروع: صنفٌ لم يُصنَّف بعد».
//
// **فتبدّل عقدُ العمود ولم يتبعه عقدُ المفتاح.**
//
// **والحذفُ حينئذٍ لا يُفرِّغ الحقلَ ولا يمنع نفسَه**: **يسقط بخرقِ
// `NOT NULL` (`23502`)** — **و`respondErr` لا يعرف هذا الرمز فيردّ
// خمسَمئة.** **فالأدمن يقرأ «عطبٌ في الخادم» عن قاعدةٍ تعمل تماماً.**
//
// # والعقدُ المقصود مستخرَجٌ لا مُخترَع
//
//	`SET NULL`  ⇐  **ميّت**: قرارُ ٠١١٨ يمنع صنفاً بلا قسم
//	`CASCADE`   ⇐  **مرفوضٌ نصّاً** في تعليق المعالِج: «ولو حُذف معه
//	                لَضاعت أسعارٌ وخياراتٌ بُنيت على مدى شهور»
//	الإطفاء     ⇐  **قائمٌ يعمل**: `PATCH active=false` — وهو وحدَه ما
//	                تناديه لوحةُ الأدمن، وهو ما فعلته هجرةُ ٠١٥٦
//
// **فلم يبقَ للحذف إلّا أن يُرفض ما دام القسمُ مشغولاً** — **ويُقال
// للأدمن لماذا، لا أن يُردّ بخمسِمئة صامتة.**

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// sectionDeleteFixture **قسمٌ ومتجرٌ وقائمة** — وأصنافُه تُضاف عند الطلب.
type sectionDeleteFixture struct {
	f             *driverFixture
	sectionID     string
	menuSectionID string
}

func newSectionDeleteFixture(t *testing.T) *sectionDeleteFixture {
	t.Helper()
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	x := &sectionDeleteFixture{f: f}

	// **والقسمُ يُنشأ أوّلاً ليُحذَف آخراً** — `t.Cleanup` عكسُ التسجيل،
	// **وأصنافُه تذهب بذهاب متجرها.**
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, sort_order) VALUES ($1, 950)
		RETURNING id::text`, "قسمُ عقدِ الحذف "+t.Name()).Scan(&x.sectionID); err != nil {
		t.Fatalf("تعذّر القسم: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = f.pool.Exec(c, `DELETE FROM menu_items WHERE platform_section_id = $1`, x.sectionID)
		if _, err := f.pool.Exec(c,
			`DELETE FROM platform_sections WHERE id = $1`, x.sectionID); err != nil {
			t.Errorf("تعذّر حذفُ القسم بعد الفحص: %v", err)
		}
	})

	if err := f.pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'قائمةُ عقدِ الحذف')
		RETURNING id::text`, f.merchantID).Scan(&x.menuSectionID); err != nil {
		t.Fatalf("تعذّرت قائمةُ المتجر: %v", err)
	}
	return x
}

// addItem **صنفٌ يشغل القسم.**
func (x *sectionDeleteFixture) addItem(t *testing.T, name string) string {
	t.Helper()
	var id string
	if err := x.f.pool.QueryRow(context.Background(), `
		INSERT INTO menu_items (merchant_id, section_id, platform_section_id,
			name, price, merchant_price, approved, available)
		VALUES ($1, $2, $3, $4, 1000, 1000, true, true)
		RETURNING id::text`,
		x.f.merchantID, x.menuSectionID, x.sectionID, name).Scan(&id); err != nil {
		t.Fatalf("تعذّر الصنف: %v", err)
	}
	return id
}

// del **ينادي بابَ الحذف كما يناديه الأدمن.**
func (x *sectionDeleteFixture) del(t *testing.T, id string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, "/x", nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", id)
	c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	c = context.WithValue(c, ctxRoles, []string{"admin"})
	w := httptest.NewRecorder()
	x.f.srv.handleDeletePlatformSection(w, req.WithContext(c))
	return w
}

// patch **بابُ التعديل — وبه يقع الإطفاء.**
func (x *sectionDeleteFixture) patch(t *testing.T, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", id)
	c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	c = context.WithValue(c, ctxRoles, []string{"admin"})
	w := httptest.NewRecorder()
	x.f.srv.handleUpdatePlatformSection(w, req.WithContext(c))
	return w
}

func (x *sectionDeleteFixture) sectionExists(t *testing.T, id string) bool {
	t.Helper()
	var n int
	if err := x.f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM platform_sections WHERE id = $1`, id).Scan(&n); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	return n > 0
}

// ── ١ · قسمٌ مشغولٌ يُرفض حذفُه بجوابٍ صريح ─────────────────────────
//
// **وهذا موضعُ العطب**: **اليومَ يردّ خمسَمئة** — «عطبٌ في الخادم» عن
// حالةٍ مفهومةٍ تماماً.
func TestSD1_ReferencedSectionIsRefusedExplicitly(t *testing.T) {
	x := newSectionDeleteFixture(t)
	itemID := x.addItem(t, "صنفٌ يشغل القسم")

	w := x.del(t, x.sectionID)

	if w.Code == http.StatusInternalServerError {
		t.Fatalf("**ردَّ خمسَمئة على حالةٍ مفهومة** — %s\n"+
			"**والأدمن يقرأ «عطبٌ في الخادم» عن قاعدةٍ تعمل.**", w.Body.String())
	}
	if w.Code != http.StatusConflict {
		t.Fatalf("**منتظَرٌ ٤٠٩ والموجودُ %d** — %s", w.Code, w.Body.String())
	}

	// **والجوابُ يقول السبب** — لا رمزاً أصمّ.
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v", err)
	}

	// **ولا صنفَ يتيمٌ بعد الرفض** — القسمُ باقٍ والصنفُ فيه.
	if !x.sectionExists(t, x.sectionID) {
		t.Fatal("**ذهب القسمُ رغم الرفض**")
	}
	var got string
	if err := x.f.pool.QueryRow(context.Background(),
		`SELECT platform_section_id::text FROM menu_items WHERE id = $1`, itemID).
		Scan(&got); err != nil {
		t.Fatalf("تعذّرت قراءةُ الصنف: %v", err)
	}
	if got != x.sectionID {
		t.Fatalf("**انقطع الصنفُ عن قسمه**: %q", got)
	}
}

// ── ٢ · وقسمٌ فارغٌ يُحذف كما كان ───────────────────────────────────
func TestSD2_UnreferencedSectionIsDeleted(t *testing.T) {
	x := newSectionDeleteFixture(t)

	w := x.del(t, x.sectionID)
	if w.Code != http.StatusOK {
		t.Fatalf("**رُدَّ حذفُ قسمٍ فارغ**: %d — %s", w.Code, w.Body.String())
	}
	if x.sectionExists(t, x.sectionID) {
		t.Fatal("**بقي القسمُ بعد حذفٍ ناجح**")
	}
}

// ── ٣ · والإطفاءُ هو طريقُ التقاعد، ولا يمسّ الأصناف ────────────────
//
// **وهو ما تناديه اللوحةُ فعلاً** — ولا زرَّ حذفٍ فيها أصلاً.
func TestSD3_RetireKeepsItemsAttached(t *testing.T) {
	x := newSectionDeleteFixture(t)
	itemID := x.addItem(t, "صنفٌ يبقى بعد الإطفاء")

	w := x.patch(t, x.sectionID, `{"active":false}`)
	if w.Code != http.StatusOK {
		t.Fatalf("**رُدَّ الإطفاء**: %d — %s", w.Code, w.Body.String())
	}

	var active bool
	if err := x.f.pool.QueryRow(context.Background(),
		`SELECT active FROM platform_sections WHERE id = $1`, x.sectionID).
		Scan(&active); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if active {
		t.Fatal("**بقي القسمُ عاملاً بعد الإطفاء**")
	}

	// **والصنفُ باقٍ في قسمه** — **والإطفاءُ يُخفي ولا يقطع.**
	var got string
	if err := x.f.pool.QueryRow(context.Background(),
		`SELECT platform_section_id::text FROM menu_items WHERE id = $1`, itemID).
		Scan(&got); err != nil {
		t.Fatalf("تعذّرت قراءةُ الصنف: %v", err)
	}
	if got != x.sectionID {
		t.Fatalf("**انقطع الصنفُ بالإطفاء**: %q", got)
	}
}

// ── ٤ · ولا صنفَ بلا قسمٍ في القاعدة كلِّها ─────────────────────────
//
// **حارسُ البنية**: **`NOT NULL` يمنعه، وهذا يقوله صراحةً** — فمن رخّى
// العمودَ يوماً رأى الأحمرَ هنا.
func TestSD4_NoOrphanItemsExist(t *testing.T) {
	f := newDriverFixture(t, 0)
	var orphans int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM menu_items WHERE platform_section_id IS NULL`).
		Scan(&orphans); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if orphans != 0 {
		t.Fatalf("**%d صنفاً بلا قسم** — **وصنفٌ بلا قسمٍ لا يراه أحد.**", orphans)
	}
}
