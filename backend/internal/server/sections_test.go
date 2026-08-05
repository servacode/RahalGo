package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestBrowse_HidesSource **اسمُ المتجر لا يخرج من التصفّح — ولا معرّفُه.**
//
// # لماذا هذا الاختبار بالذات
//
// **الإخفاءُ في الشاشة لا يكفي.** من فتح أدوات المتصفّح قرأ الردَّ كما هو،
// **ومعرّفٌ في الردّ يُفتح به `/public/merchants/{id}` فيُقرأ الاسمُ كاملاً.**
//
// **وهو أهمُّ حارسٍ في المرحلة الثانية كلِّها**: قرارُ المالك أن اسمَ المتجر
// مخفيٌّ لأن **زبوناً رآه يتّصل به مباشرةً في المرّة القادمة** — فيوفّر رسمَ
// التوصيل والمطعمُ يوفّر عمولتنا. **وكلُّ منصةِ توصيلٍ تموت من هذا الباب.**
//
// **وحقلٌ يُضاف يوماً بلا انتباه يهدم هذا كلَّه** — ولا يظهر في أيّ خطأ.
func TestBrowse_HidesSource(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	catalogSvc := catalog.NewService(pool, nil)
	settingsStore := settings.NewStore(pool)
	catalogSvc.SetSettings(settingsStore)
	srv := &Server{pg: pool, logger: quiet, hub: realtime.NewHub(quiet),
		catalog: catalogSvc, settings: settingsStore}

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	// **اسمٌ لا يشبه شيئاً** — كي يُبحث عنه في الردّ بلا لبس.
	const secret = "مطعمُ السرِّ المكتوم"
	var merchantID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, status)
		VALUES ($1, $2, 10, 'active') RETURNING id`, secret, categoryID).
		Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر إنشاء متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchantID)
	})

	var sectionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, icon, sort_order)
		VALUES ('قسمُ اختبار', 'food', 99) RETURNING id`).Scan(&sectionID); err != nil {
		t.Fatalf("تعذّر إنشاء قسم: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, sectionID)
	})

	var secID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'الرئيسية')
		RETURNING id`, merchantID).Scan(&secID); err != nil {
		t.Fatalf("تعذّر إنشاء قسم قائمة: %v", err)
	}
	var itemID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, section_id, platform_section_id,
		                        name, merchant_price, price, available)
		VALUES ($1, $2, $3, 'صنفُ اختبارٍ فريد', 10000, 10000, true)
		RETURNING id`, merchantID, secID, sectionID).Scan(&itemID); err != nil {
		t.Fatalf("تعذّر إنشاء صنف: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/sections", srv.handlePublicSections)
	r.Get("/sections/{id}/items", srv.handlePublicSectionItems)
	r.Get("/items/{id}", srv.handlePublicItem)
	r.Get("/search", srv.handleSearchItems)
	// **والصفحةُ الأولى كانت خارجَ الحراسة.**
	//
	// حرس هذا الاختبارُ الأقسامَ والأصنافَ والبحثَ منذ المرحلة الثانية،
	// **و`/home` تُرسل قائمةَ المتاجر كاملةً بأسمائها وشعاراتها** — وهي
	// أوّلُ ما يُفتح في المنصة. **فالبابُ الذي لم يُحرس هو الذي كان مفتوحاً.**
	r.Get("/home", srv.handlePublicHome)

	get := func(path string) string {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s ردّ %d", path, w.Code)
		}
		return w.Body.String()
	}

	for _, path := range []string{
		"/sections",
		"/sections/" + sectionID + "/items",
		"/items/" + itemID,
		"/search?q=" + "صنف",
		"/home",
	} {
		body := get(path)
		if strings.Contains(body, secret) {
			t.Errorf("%s سرّب اسمَ المتجر", path)
		}
		// **والمعرّفُ يكفي لكشف الاسم**: كان يُفتح به `/public/merchants/{id}`
		// **فحُذفت النقطةُ والصفحة** — ويبقى المعرّفُ محجوباً، فما حُذف اليوم
		// يُعاد غداً **والمعرّفُ المسرَّبُ يبقى في الردّ حتّى يُنتبَه له.**
		if strings.Contains(body, merchantID) {
			t.Errorf("%s سرّب معرّفَ المتجر — وبه يُقرأ الاسمُ كاملاً", path)
		}
	}

	// **وسعرُ الشراء كذلك لا يخرج.**
	//
	// الهامشُ مُعلَنٌ للمتجر ولا نكتمه، **لكنّ رقماً بعينه لكلّ صنفٍ يصل
	// الزبونَ يصل المتجرَ من بعده**، ومنافساً يبني قائمتَه على أرقامنا.
	// **والردُّ ملفوفٌ في `data`** — كسائر ردود المنصة (`httpx.JSON`).
	var out struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(get("/sections/"+sectionID+"/items")), &out); err != nil {
		t.Fatalf("ردٌّ غيرُ مقروء: %v", err)
	}
	if len(out.Data.Items) != 1 {
		t.Fatalf("عددُ الأصناف = %d، والمتوقّع 1", len(out.Data.Items))
	}
	for _, banned := range []string{"merchant_price", "merchant_id", "merchant_name", "margin_override"} {
		if _, ok := out.Data.Items[0][banned]; ok {
			t.Errorf("الحقل %q خرج إلى الزبون", banned)
		}
	}
}
