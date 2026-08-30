package server

// ══════════════════════════════════════════════════════════════════════
// **مراجعةُ القائمة تحرس بابَ المندوب كما تحرس بابَ المتجر**
// ══════════════════════════════════════════════════════════════════════
//
// **وكانت تحرس نصفَ الأبواب**: المندوبُ يستعمل معالجاتِ الإدارة، **فورث
// إعفاءَ الأدمن من المراجعة** — يُشغَّل المفتاحُ فيُحجب ما يكتبه المتجر
// **ويمرّ ما يكتبه المندوبُ إلى السوق.**
//
// **وإعدادٌ يحرس نصفَ الأبواب أخطرُ من إعدادٍ لا يحرس شيئاً**: من شغّله
// ظنّ القوائمَ كلَّها محروسة فبنى عليه.
//
// (كشفه سؤالُ المالك ٢٠٢٦-٠٨-٣٠، وأُصلح بإذنه.)

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestRepMenuWrite_HeldForReviewLikeMerchant(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	rep := testdb.NewUser(t, f.pool, "sales")
	owner := testdb.NewUser(t, f.pool, "merchant")

	var cat string
	if err := f.pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&cat); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	var mid string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, owner_user_id, sales_rep_user_id, status, category_id)
		VALUES ('متجرُ فحصِ المراجعة', $1, $2, 'active', $3) RETURNING id`,
		owner, rep, cat).Scan(&mid); err != nil {
		t.Fatalf("تعذّر المتجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, mid)
	})

	var sec string
	if err := f.pool.QueryRow(ctx, `SELECT id FROM platform_sections LIMIT 1`).Scan(&sec); err != nil {
		t.Skipf("لا أقسامَ سوقٍ في القاعدة: %v", err)
	}

	// **والمفتاحُ يُرفع** — وهو الحالُ الذي كان يتسرّب منه المندوب.
	f.setSetting(t, "merchants.menu_requires_approval", true)
	t.Cleanup(func() { f.setSetting(t, "merchants.menu_requires_approval", false) })

	// **ويُنادى المعالجُ كما يناديه الموجّه.**
	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost,
			"/api/v1/rep/stores/"+mid+"/menu/items", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rc := chi.NewRouteContext()
		rc.URLParams.Add("id", mid)
		c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
		c = context.WithValue(c, ctxUserID, rep)
		w := httptest.NewRecorder()
		f.srv.handleRepCreateItem(w, req.WithContext(c))
		return w
	}

	w := post(`{"name":"صنفُ المندوب","price":1000,"platform_section_id":"` + sec + `"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الصنف ردّ %d — %s", w.Code, w.Body.String())
	}

	// ══════════════════════════════════════════════════════════════════
	// **والدليلُ في القاعدة لا في الردّ**
	// ══════════════════════════════════════════════════════════════════
	//
	// **`pending_review` في الردّ يقول ما نوى المعالج** — والعمودُ يقول
	// ما وقع. **ورايةٌ تُرفع في JSON ولا تُكتب في الصفّ تُخفي العطبَ
	// نفسَه** الذي جاء هذا الاختبارُ من أجله.
	var approved bool
	if err := f.pool.QueryRow(ctx, `
		SELECT approved FROM menu_items
		WHERE merchant_id = $1 ORDER BY created_at DESC LIMIT 1`, mid).Scan(&approved); err != nil {
		t.Fatalf("تعذّرت قراءةُ الصنف: %v", err)
	}
	if approved {
		t.Error("صنفُ المندوب نُشر في السوق والمراجعةُ مرفوعة — " +
			"**والمفتاحُ يحرس بابَ المتجر ويترك بابَه مفتوحاً**")
	}
}

// TestRepMenuWrite_AvailabilityIsNotHeld **والتوفّرُ وحدَه لا يُعلّق الصنف.**
//
// **«نفد الصنف» قرارُ مطبخٍ في لحظته** (الهجرة ٠٠٦٤) — **ومراجعتُه تجعل
// المتجرَ يبيع ما نفد حتّى نستيقظ.** والعكسُ أسوأ: **من أعاد صنفاً بعد
// أن ورد ينتظر مراجعةً ليبيع ما بين يديه.**
func TestRepMenuWrite_AvailabilityIsNotHeld(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	rep := testdb.NewUser(t, f.pool, "sales")
	owner := testdb.NewUser(t, f.pool, "merchant")

	var cat string
	if err := f.pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&cat); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	var mid string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, owner_user_id, sales_rep_user_id, status, category_id)
		VALUES ('متجرُ فحصِ التوفّر', $1, $2, 'active', $3) RETURNING id`,
		owner, rep, cat).Scan(&mid); err != nil {
		t.Fatalf("تعذّر المتجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, mid)
	})

	var sec string
	if err := f.pool.QueryRow(ctx, `SELECT id FROM platform_sections LIMIT 1`).Scan(&sec); err != nil {
		t.Skipf("لا أقسامَ سوقٍ في القاعدة: %v", err)
	}
	var item string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, name, price, merchant_price, available, approved, platform_section_id)
		VALUES ($1, 'صنفٌ منشور', 1000, 900, true, true, $2) RETURNING id`, mid, sec).Scan(&item); err != nil {
		t.Fatalf("تعذّر الصنف: %v", err)
	}

	f.setSetting(t, "merchants.menu_requires_approval", true)
	t.Cleanup(func() { f.setSetting(t, "merchants.menu_requires_approval", false) })

	req := httptest.NewRequest(http.MethodPatch,
		"/api/v1/rep/menu/items/"+item, strings.NewReader(`{"available":false}`))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("itemID", item)
	c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	c = context.WithValue(c, ctxUserID, rep)
	w := httptest.NewRecorder()
	f.srv.handleRepUpdateItem(w, req.WithContext(c))
	if w.Code != http.StatusOK {
		t.Fatalf("قلبُ التوفّر ردّ %d — %s", w.Code, w.Body.String())
	}

	var approved, available bool
	if err := f.pool.QueryRow(ctx,
		`SELECT approved, available FROM menu_items WHERE id = $1`, item).
		Scan(&approved, &available); err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if !approved {
		t.Error("قلبُ التوفّر علّق الصنفَ للمراجعة — **وصاحبُه ينتظر إذناً ليقول «نفد»**")
	}
	if available {
		t.Error("التوفّرُ لم يُقلب أصلاً")
	}
}
