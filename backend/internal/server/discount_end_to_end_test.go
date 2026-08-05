package server

// **ما يُعرض هو ما يُقبض — أو لا معنى للخصم.**
//
// # لماذا وُجد هذا الاختبار
//
// قلتُ إنّ الخصمَ «يُطبَّق فعلاً» وأثبتُّ ذلك في `orders` **بقارئٍ ملفَّقٍ
// حقنتُه بيدي** — فأثبتُّ أنّ المحرّكَ يحترم قارئاً يقول «٢٠٪»، **ولم أُثبت
// أنّ عرضاً حقيقيّاً في الجدول يصل إليه، ولا أنّ الشاشةَ تقول ما يقوله.**
//
// **وشهده المالكُ على شاشته** (٢٠٢٦-٠٨-٠٥): بطاقةٌ تقول «٧٧٬٢٥٠» بشارة
// «−٢٥٪»، **والنافذةُ فوقها تقول «أضف للسلّة — ١٠٣٬٠٠٠».**
//
// **فالإثباتُ الجزئيُّ قيل كاملاً**، وهو أسوأُ من لا إثبات: **حارسٌ يُصدَّق
// وهو لا يحرس البابَ الذي يُظنّ به.**
//
// # وما يُقاس هنا
//
// **عرضٌ يُدرج في `offers` بيدٍ** — لا قارئٌ يُحقن — ثمّ:
//
//	`/public/sections/{id}/items`  بطاقةُ القسم
//	`/public/items/{id}`           نافذةُ الصنف **وزرُّ «أضف للسلّة»**
//
// **والثلاثةُ يجب أن تقول الرقمَ نفسَه** — والرقمُ نفسُه هو ما يُقيَّد في
// `order_items.unit_price` (يحرسه `orders/discount_applies_test.go`).

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/offers"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestDiscount_ShownPriceMatchesCharged(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := t.Context()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	catalogSvc := catalog.NewService(pool, nil)
	store := settings.NewStore(pool)
	catalogSvc.SetSettings(store)
	srv := &Server{pg: pool, logger: quiet, hub: realtime.NewHub(quiet),
		catalog: catalogSvc, settings: store}

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	var merchantID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, status)
		VALUES ('مطبخُ الخصم', $1, 'active') RETURNING id`, categoryID).Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر إنشاءُ متجر: %v", err)
	}
	var sectionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, active) VALUES ('قسمُ الخصم', true)
		RETURNING id`).Scan(&sectionID); err != nil {
		t.Fatalf("تعذّر إنشاءُ قسمِ منصّة: %v", err)
	}
	var menuSecID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'الرئيسية')
		RETURNING id`, merchantID).Scan(&menuSecID); err != nil {
		t.Fatalf("تعذّر إنشاءُ قسمِ قائمة: %v", err)
	}
	var itemID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, section_id, platform_section_id, name,
			merchant_price, price, available, approved)
		VALUES ($1, $2, $3, 'صنفُ الخصم', 100000, 100000, true, true)
		RETURNING id`, merchantID, menuSecID, sectionID).Scan(&itemID); err != nil {
		t.Fatalf("تعذّر إنشاءُ صنف: %v", err)
	}
	t.Cleanup(func() {
		c := t.Context()
		_, _ = pool.Exec(c, `DELETE FROM offers WHERE menu_item_id = $1`, itemID)
		_, _ = pool.Exec(c, `DELETE FROM menu_items WHERE id = $1`, itemID)
		_, _ = pool.Exec(c, `DELETE FROM menu_sections WHERE id = $1`, menuSecID)
		_, _ = pool.Exec(c, `DELETE FROM platform_sections WHERE id = $1`, sectionID)
		_, _ = pool.Exec(c, `DELETE FROM merchants WHERE id = $1`, merchantID)
	})

	r := chi.NewRouter()
	r.Get("/sections/{id}/items", srv.handlePublicSectionItems)
	r.Get("/items/{id}", srv.handlePublicItem)

	type priced struct {
		Price           int64  `json:"price"`
		PriceBefore     *int64 `json:"price_before"`
		DiscountPercent *int   `json:"discount_percent"`
	}
	readOne := func(path string) priced {
		t.Helper()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s ردّ %d", path, w.Code)
		}
		var out struct {
			Data struct {
				Item  *priced  `json:"item"`
				Items []priced `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("%s: %v — %s", path, err, w.Body.String())
		}
		if out.Data.Item != nil {
			return *out.Data.Item
		}
		for _, it := range out.Data.Items {
			return it
		}
		t.Fatalf("%s لم يُخرج صنفاً — **والاختبارُ لم يقس شيئاً**", path)
		return priced{}
	}

	// **الأساسُ أوّلاً** — وسعرُ البيع يُحسب من هوامشَ قد تتغيّر، فلا يُكتب رقماً.
	cardBase := readOne("/sections/" + sectionID + "/items")
	modalBase := readOne("/items/" + itemID)
	if cardBase.Price != modalBase.Price {
		t.Fatalf("البطاقةُ %d والنافذةُ %d **قبل أيّ خصم** — والخللُ أقدمُ من الخصم",
			cardBase.Price, modalBase.Price)
	}
	if cardBase.PriceBefore != nil {
		t.Fatalf("سعرٌ مشطوبٌ بلا عرض — **شارةُ خصمٍ كاذبة**")
	}

	var adminID string
	if err := pool.QueryRow(ctx, `SELECT id FROM users LIMIT 1`).Scan(&adminID); err != nil {
		t.Fatalf("لا مستخدمين: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO offers (kind, title, menu_item_id, discount_percent, borne_by,
			active, created_by)
		VALUES ('discount', 'خصمُ اختبار', $1, 25, $2, true, $3)`,
		itemID, offers.ByPlatform, adminID); err != nil {
		t.Fatalf("تعذّر إنشاءُ العرض: %v", err)
	}

	card := readOne("/sections/" + sectionID + "/items")
	modal := readOne("/items/" + itemID)
	want := offers.AfterDiscount(cardBase.Price, 25)
	t.Logf("الأساس=%d · البطاقة=%d · النافذة=%d · المنتظَر=%d",
		cardBase.Price, card.Price, modal.Price, want)

	if card.Price != want {
		t.Errorf("بطاقةُ القسم تقول %d والمنتظَر %d — **الخصمُ لا يُرى حيث يُتصفَّح**",
			card.Price, want)
	}
	// **والنافذةُ هي زرُّ «أضف للسلّة»** — وهي الشاشةُ الحاسمة: منها يدخل
	// السعرُ إلى السلّة، **ورقمٌ فيها يخالف البطاقةَ يُقرأ خدعةً لا خصماً.**
	if modal.Price != want {
		t.Errorf("نافذةُ الصنف تقول %d والمنتظَر %d — **وهي زرُّ الإضافة إلى السلّة**",
			modal.Price, want)
	}
	if modal.PriceBefore == nil || *modal.PriceBefore != cardBase.Price {
		t.Errorf("النافذةُ بلا سعرٍ مشطوب — **والمشطوبُ هو ما يجعل الخصمَ خصماً**")
	}
	if modal.DiscountPercent == nil || *modal.DiscountPercent != 25 {
		t.Errorf("النافذةُ بلا نسبة — **والنسبةُ تُقرأ قبل أن يُقارَن الرقمان**")
	}
}
