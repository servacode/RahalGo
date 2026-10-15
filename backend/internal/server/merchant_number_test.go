package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestMerchantNumber **رقمُ المتجر القصير — يُعطى وحدَه، ويُبحث به، ولا يخرج للزبون.**
//
// (طلبُ المالك ٢٠٢٦-١٠-١٦: «القصور #M-1045» و«القصور #M-1062».)
//
// **متجران باسمٍ عربيٍّ واحدٍ لا يُفرَّقان في اللوحة إلّا بفتح كلٍّ منهما.**
// فيحرس هذا ثلاثاً:
//
//   - **كلُّ متجرٍ يولد برقمٍ فريد** — من التسلسل لا من شيفرةٍ تكتبه، فلا
//     بابَ إنشاءٍ ينساه.
//   - **البحثُ في اللوحة بـ«M-1045» أو «1045» يجده** — وهو ما سيكتبه الموظّف.
//   - **وبطاقةُ الصنف عند الزبون لا تحمله** — الرقمُ للموظّفين وحدَهم.
func TestMerchantNumber(t *testing.T) {
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
	// **اسمان متطابقان** — وهي الحالُ التي وُجد الرقمُ لأجلها.
	const twin = "قصورُ اختبار الرقم"
	mk := func() (string, int64) {
		var id string
		var n int64
		if err := pool.QueryRow(ctx, `
			INSERT INTO merchants (name, category_id, commission_percent, status)
			VALUES ($1, $2, 10, 'active') RETURNING id::text, number`, twin, categoryID).
			Scan(&id, &n); err != nil {
			t.Fatalf("تعذّر إنشاء متجر: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, id)
		})
		return id, n
	}
	idA, numA := mk()
	idB, numB := mk()

	// ── يُعطى عند الإنشاء، فريداً، ومن ١٠٠١ فصاعداً ──────────────────────
	if numA < 1001 || numB < 1001 {
		t.Fatalf("رقمان %d و%d — **والترقيمُ يبدأ من ١٠٠١**", numA, numB)
	}
	if numA == numB {
		t.Fatalf("متجران بالرقم نفسِه %d — **والرقمُ وُجد ليفرّق بينهما**", numA)
	}
	if numB <= numA {
		t.Errorf("الثاني %d لا يلي الأوّل %d — **والجديدُ يأخذ التاليَ من التسلسل**", numB, numA)
	}
	var dups int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM (SELECT number FROM merchants GROUP BY number HAVING count(*) > 1) d`).
		Scan(&dups); err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}
	if dups != 0 {
		t.Fatalf("%d رقماً مكرّراً في المتاجر", dups)
	}
	// **ورقمٌ مكتوبٌ بيدٍ مكرّراً يُرفض من القاعدة** — لا من الشيفرة وحدَها.
	if _, err := pool.Exec(ctx, `UPDATE merchants SET number = $1 WHERE id = $2`, numA, idB); err == nil {
		t.Fatalf("قُبل رقمٌ مكرّر — **والقيدُ الفريدُ غائب**")
	}

	// ── وملفُّه في اللوحة يحمله ───────────────────────────────────────────
	mr, err := catalogSvc.MerchantByID(ctx, idA)
	if err != nil {
		t.Fatalf("تعذّر الملفّ: %v", err)
	}
	if mr.Number != numA {
		t.Errorf("ملفُّ المتجر يحمل %d، والمتوقّع %d", mr.Number, numA)
	}

	r := chi.NewRouter()
	r.Get("/admin/merchants", srv.handleListMerchants)
	r.Get("/admin/market/stores", srv.handleMarketStores)
	r.Get("/public/items/{id}", srv.handlePublicItem)
	r.Get("/public/sections/{id}/items", srv.handlePublicSectionItems)
	r.Get("/public/search", srv.handleSearchItems)
	get := func(path string) string {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s ردّ %d: %s", path, w.Code, w.Body.String())
		}
		return w.Body.String()
	}

	// ── والبحثُ برقمه يجده هو وحدَه — لا توأمَه ─────────────────────────
	n := strconv.FormatInt(numA, 10)
	for _, q := range []string{n, "M-" + n, "#M-" + n, "m" + n} {
		var list struct {
			Data struct {
				Merchants []struct {
					ID     string `json:"id"`
					Number int64  `json:"number"`
				} `json:"merchants"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(get("/admin/merchants?query="+url.QueryEscape(q))), &list); err != nil {
			t.Fatalf("ردٌّ غيرُ مقروء: %v", err)
		}
		found := false
		for _, x := range list.Data.Merchants {
			if x.ID == idB {
				t.Errorf("«%s» وجد التوأمَ أيضاً — **والرقمُ وُجد ليفرّق**", q)
			}
			if x.ID == idA {
				found = true
				if x.Number != numA {
					t.Errorf("القائمةُ تحمل %d، والمتوقّع %d", x.Number, numA)
				}
			}
		}
		if !found {
			t.Errorf("البحثُ في قائمة المتاجر بـ«%s» لم يجده", q)
		}

		var stores struct {
			Data struct {
				Stores []struct {
					ID     string `json:"id"`
					Number int64  `json:"number"`
				} `json:"stores"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(get("/admin/market/stores?q="+url.QueryEscape(q))), &stores); err != nil {
			t.Fatalf("ردٌّ غيرُ مقروء: %v", err)
		}
		if len(stores.Data.Stores) != 1 || stores.Data.Stores[0].ID != idA || stores.Data.Stores[0].Number != numA {
			t.Errorf("بحثُ صفحة المتاجر بـ«%s» ردّ %+v، والمتوقّع متجرُه وحدَه", q, stores.Data.Stores)
		}
	}

	// ── وبطاقةُ الصنف عند الزبون لا تحمله ────────────────────────────────
	var secID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'الرئيسية')
		RETURNING id`, idA).Scan(&secID); err != nil {
		t.Fatalf("تعذّر إنشاء قسم قائمة: %v", err)
	}
	var platformID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, icon, sort_order)
		VALUES ('قسمُ اختبار الرقم', 'food', 99) RETURNING id`).Scan(&platformID); err != nil {
		t.Fatalf("تعذّر إنشاء قسم: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, platformID)
	})
	var itemID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, section_id, platform_section_id,
		                        name, merchant_price, price, available)
		VALUES ($1, $2, $3, 'صنفُ رقمٍ لا يُرى', 10000, 10000, true)
		RETURNING id`, idA, secID, platformID).Scan(&itemID); err != nil {
		t.Fatalf("تعذّر إنشاء صنف: %v", err)
	}
	var item struct {
		Data struct {
			Item map[string]any `json:"item"`
		} `json:"data"`
	}
	body := get("/public/items/" + itemID)
	if err := json.Unmarshal([]byte(body), &item); err != nil {
		t.Fatalf("ردٌّ غيرُ مقروء: %v", err)
	}
	if item.Data.Item["merchant_id"] != idA {
		t.Fatalf("الصنفُ لم يخرج بمتجره — ردّ: %s", body)
	}
	for _, banned := range []string{"number", "merchant_number"} {
		if _, ok := item.Data.Item[banned]; ok {
			t.Errorf("الحقل %q خرج إلى الزبون في بطاقة الصنف", banned)
		}
	}
	if strings.Contains(body, "M-"+n) || strings.Contains(body, ":"+n+",") || strings.Contains(body, ":"+n+"}") {
		t.Errorf("رقمُ المتجر %s ظهر في بطاقة الصنف: %s", n, body)
	}
	// **وشبكةُ التصفّح كذلك** — بطاقاتُ القسم.
	var grid struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(get("/public/sections/"+platformID+"/items")), &grid); err != nil {
		t.Fatalf("ردٌّ غيرُ مقروء: %v", err)
	}
	if len(grid.Data.Items) != 1 {
		t.Fatalf("عددُ أصناف القسم = %d، والمتوقّع 1", len(grid.Data.Items))
	}
	for _, banned := range []string{"number", "merchant_number"} {
		if _, ok := grid.Data.Items[0][banned]; ok {
			t.Errorf("الحقل %q خرج إلى الزبون في شبكة التصفّح", banned)
		}
	}
	if search := get("/public/search?q=" + url.QueryEscape("صنفُ رقمٍ")); strings.Contains(search, "M-"+n) ||
		strings.Contains(search, `"number"`) {
		t.Errorf("رقمُ المتجر ظهر في بحث الزبون: %s", search)
	}
}
