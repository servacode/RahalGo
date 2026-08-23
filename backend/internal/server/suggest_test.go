package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestSuggestAndHasOptions يحرس بابين فُتحا معاً ٢٠٢٦-٠٨-٢٢.
//
// # الأوّل · **صنفٌ بخياراتٍ إلزاميّةٍ يُسقط الطلبَ إن لم تُعرف**
//
// **المحرّكُ يرفض** (`orders/service.go`): اختيارٌ أقلُّ من `min_select`
// يردّ `ErrBadItems` — **والطلبُ كلُّه يسقط**، لا الصنفُ وحدَه.
//
// **والشاشةُ لا تعرف قبل الضغطة** إلّا بحقلٍ في قائمة القسم: نداءٌ
// لكلّ بطاقةٍ يعني عشرين نداءً في شبكةٍ من عشرين. **فحقلٌ واحدٌ يفرّق
// بين نافذةِ اختيارٍ تُفتح وطلبٍ يسقط بلا سبب.**
//
// # والثاني · **اقتراحُ ما لا يُطلب يُسقط الطلبَ كذلك**
//
// **«يُطلب معه» يدخل السلّةَ بضغطةٍ واحدة** — فلو اقتُرح صنفٌ من مدينةٍ
// أخرى **لَرفضه المحرّكُ عند الإرسال**، ولا يفهم الزبونُ لماذا سقط
// طلبُه بسبب كولا أضافها باقتراحنا.
//
// **ولا يُقترح ما في السلّة**: اقتراحُ ما اشتراه يقول له إنّا لا نقرأ
// سلّته.
func TestSuggestAndHasOptions(t *testing.T) {
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

	// **مدينةٌ بعيدةٌ عن كلّ مزروعة** — فلا يلتقط الترشيحُ مدينةً أخرى
	// فينجح الاختبارُ لسببٍ غيرِ الذي يفحصه.
	const lat, lng = 20.0, 60.0
	var cityID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO cities (name, center, radius_m, active)
		VALUES ('مدينةُ الاقتراح', ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, 5000, true)
		RETURNING id`, lng, lat).Scan(&cityID); err != nil {
		t.Fatalf("تعذّرت المدينة: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM cities WHERE id = $1`, cityID) })

	var merchantID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, status, city_id, location)
		VALUES ('متجرُ الاقتراح', $1, 10, 'active', $2,
		        ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography)
		RETURNING id`, categoryID, cityID, lng, lat).Scan(&merchantID); err != nil {
		t.Fatalf("تعذّر المتجر: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchantID) })

	var secID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'الرئيسية')
		RETURNING id`, merchantID).Scan(&secID); err != nil {
		t.Fatalf("تعذّر قسمُ القائمة: %v", err)
	}

	// **قسمان**: طعامٌ يُطلب، ومشروبٌ يُقترح معه.
	// **والاسمُ يُبحث عنه أوّلاً**: «مشروبات باردة» قد يكون مزروعاً في
	// القاعدة، **وقسمان بالاسم نفسِه يجعلان الاقتراحَ يقرأ أحدَهما فارغاً.**
	section := func(name string, sort int) (string, bool) {
		var id string
		if err := pool.QueryRow(ctx,
			`SELECT id FROM platform_sections WHERE name = $1`, name).Scan(&id); err == nil {
			_, _ = pool.Exec(ctx, `UPDATE platform_sections SET active = true WHERE id = $1`, id)
			return id, false
		}
		if err := pool.QueryRow(ctx, `
			INSERT INTO platform_sections (name, icon, sort_order, active)
			VALUES ($1, 'food', $2, true) RETURNING id`, name, sort).Scan(&id); err != nil {
			t.Fatalf("تعذّر القسم %s: %v", name, err)
		}
		return id, true
	}
	foodID, foodNew := section("قسمُ طعامِ الاقتراح", 97)
	drinkID, _ := section("مشروبات باردة", 12)
	if foodNew {
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, foodID)
		})
	}

	item := func(psID, name string, price int64) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO menu_items (merchant_id, section_id, platform_section_id,
			                        name, merchant_price, price, available, approved)
			VALUES ($1, $2, $3, $4, $5, $5, true, true) RETURNING id`,
			merchantID, secID, psID, name, price).Scan(&id); err != nil {
			t.Fatalf("تعذّر الصنف %s: %v", name, err)
		}
		return id
	}
	burger := item(foodID, "برغرُ الاقتراح", 30000)
	cola := item(drinkID, "كولا الاقتراح", 5000)
	juice := item(drinkID, "عصيرُ الاقتراح", 7000)

	// **وللبرغر حجمٌ إلزاميّ** — وهو ما يجب أن تعرفه الشاشة.
	var groupID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO modifier_groups (item_id, name, min_select, max_select)
		VALUES ($1, 'الحجم', 1, 1) RETURNING id`, burger).Scan(&groupID); err != nil {
		t.Fatalf("تعذّرت المجموعة: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/sections/{id}/items", srv.handlePublicSectionItems)
	r.Get("/suggest", srv.handleSuggestWith)

	read := func(path string) []map[string]any {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s ردّ %d — %s", path, w.Code, w.Body.String())
		}
		var out struct {
			Data struct {
				Items []map[string]any `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("%s ردٌّ لا يُفكّ: %v — %s", path, err, w.Body.String())
		}
		return out.Data.Items
	}

	// ── ١ · الحقلُ يقول الحقّ في القائمة ──────────────────────────────
	for _, it := range read("/sections/" + foodID + "/items") {
		if it["id"] != burger {
			continue
		}
		if it["has_options"] != true {
			t.Errorf("صنفٌ بمجموعةٍ إلزاميّةٍ يُعرض بلا علامةٍ — "+
				"فتضغط الشاشةُ «أضف» ويسقط الطلبُ كلُّه: %v", it["has_options"])
		}
	}
	for _, it := range read("/sections/" + drinkID + "/items") {
		if it["id"] == cola && it["has_options"] != false {
			t.Errorf("صنفٌ بلا خياراتٍ يُعرض وكأنّ له خيارات — نافذةٌ فارغةٌ تُفتح")
		}
	}

	// ── ٢ · الاقتراحُ يعرض المشروبَ ولا يعرض ما في السلّة ─────────────
	got := read("/suggest?lat=20.0&lng=60.0&exclude=" + cola)
	var sawCola, sawJuice, sawBurger bool
	for _, it := range got {
		switch it["id"] {
		case cola:
			sawCola = true
		case juice:
			sawJuice = true
		case burger:
			sawBurger = true
		}
	}
	if sawCola {
		t.Error("اقتُرح ما في السلّة — والزبونُ يقرؤها أنّا لا نقرأ سلّته")
	}
	if !sawJuice {
		t.Error("لم يُقترح مشروبٌ متاحٌ في مدينته — والنقطةُ بلا فائدة")
	}
	if sawBurger {
		t.Error("اقتُرح طعامٌ على طعامٍ — والاقتراحُ مشروبٌ وحلوى لا وجبةٌ ثانية")
	}

	// ── ٣ · ولا يُقترح من مدينةٍ أخرى ─────────────────────────────────
	//
	// **موضعٌ بعيدٌ عن كلّ مدينة** — فلا مدينةَ تُطابق، ولا يُقترح شيء.
	if far := read("/suggest?lat=-40.0&lng=-70.0"); len(far) > 0 {
		t.Errorf("اقتُرح %d صنفاً لمن هو خارجَ كلّ مدينة — "+
			"ويسقط طلبُه كلُّه حين يضيفه", len(far))
	}
}
