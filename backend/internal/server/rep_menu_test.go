package server

// **المندوبُ يبني قائمةَ عميله — ولا يمسّ قائمةَ غيره.**
//
// (طلبُ المالك ٢٠٢٦-٠٨-١٨: «نفس الفورم الموجود عند مدير المنصّة والموجود
//  عند المتجر موجودٌ عند المندوب».)
//
// # ولماذا يُفحص الطرفان
//
// **بابٌ يُفتح لمن يستحقّه ولا يُغلق دون غيره ليس باباً** — **ومندوبٌ
// يعدّل قائمةَ متجرٍ ليس عميلَه يبدّل أسعارَ سوقٍ لا يعرفه صاحبُه.**
//
// **والمعرّفاتُ تُخمَّن أو تُقرأ من ردٍّ سابق** — فلا يكفي أن تُخفى
// الأزرارُ في الشاشة.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestRepMenuGuard_OnlyOwnClients(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()

	rep := testdb.NewUser(t, f.pool, "sales")
	other := testdb.NewUser(t, f.pool, "sales")
	owner := testdb.NewUser(t, f.pool, "merchant")

	// **والتصنيفُ مطلوبٌ في الجدول** — يُؤخذ أوّلُ موجودٍ أو يُنشأ.
	var cat string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO categories (name, icon, active)
		VALUES ('فحصُ المندوب', 'other', true)
		RETURNING id`).Scan(&cat); err != nil {
		t.Fatalf("تعذّر التصنيف: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM categories WHERE id = $1`, cat)
	})

	var mine string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, owner_user_id, sales_rep_user_id, status, category_id)
		VALUES ('متجرُ عميلي', $1, $2, 'active', $3) RETURNING id`,
		owner, rep, cat).Scan(&mine); err != nil {
		t.Fatalf("تعذّر إنشاءُ المتجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, mine)
	})

	// **ويُنادى الحارسُ كما يناديه الموجّه** — بمعرّفٍ في المسار.
	call := func(userID, merchantID string) int {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		rc := chi.NewRouteContext()
		rc.URLParams.Add("id", merchantID)
		c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
		c = context.WithValue(c, ctxUserID, userID)
		w := httptest.NewRecorder()
		f.srv.repMenuGuard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, req.WithContext(c))
		return w.Code
	}

	// **صاحبُ العميل يمرّ** — وقفلٌ لا يُفتح بمفتاحه عطبٌ لا حماية.
	if code := call(rep, mine); code != http.StatusOK {
		t.Fatalf("مندوبُ العميل رُدّ بـ%d — **والقفلُ يمنع صاحبَ المفتاح**", code)
	}

	// **ومندوبٌ آخرُ يُردّ** — وهو الغرضُ كلُّه.
	if code := call(other, mine); code == http.StatusOK {
		t.Fatal("مندوبٌ ليس صاحبَ العميل مرّ — **ويعدّل أسعارَ متجرٍ لا يعرفه**")
	}

	// ══════════════════════════════════════════════════════════════════
	// **ومسارٌ بلا معرّفٍ لا يمرّ**
	// ══════════════════════════════════════════════════════════════════
	//
	// **والافتراضُ منعٌ لا سماح**: من أضاف مساراً غداً ونسي معرّفَه يجد
	// باباً مغلقاً لا مفتوحاً — **وحارسٌ يسمح عند الشكّ ليس حارساً.**
	if code := call(rep, ""); code == http.StatusOK {
		t.Fatal("مسارٌ بلا معرّفٍ مرّ — **والافتراضُ يجب أن يكون منعاً**")
	}
}
