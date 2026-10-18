package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestGroupStoreSections **أصنافُ المتجر تحت أقسامها بترتيب الأقسام** — بلا قاعدة.
//
// **وقسمٌ بلا صنفٍ لا يُعنوَن، وصنفٌ قسمُه غائبٌ لا يُعرض.**
func TestGroupStoreSections(t *testing.T) {
	heads := []storeSectionHead{{ID: "s2", Name: "ب"}, {ID: "s1", Name: "أ"}, {ID: "s3", Name: "فارغ"}}
	items := []publicItem{
		{ID: "a", SectionID: "s1"},
		{ID: "b", SectionID: "s2"},
		{ID: "c", SectionID: "s1"},
		{ID: "x", SectionID: "ghost"},
	}
	got := groupStoreSections(heads, items)
	if len(got) != 2 {
		t.Fatalf("عددُ الأقسام = %d، والمتوقّع ٢ (الفارغُ لا يُعنوَن): %+v", len(got), got)
	}
	if got[0].ID != "s2" || got[1].ID != "s1" {
		t.Fatalf("ترتيبُ الأقسام = %s,%s، والمتوقّع s2,s1 كما وصلت", got[0].ID, got[1].ID)
	}
	if len(got[1].Items) != 2 || got[1].Items[0].ID != "a" || got[1].Items[1].ID != "c" {
		t.Fatalf("أصنافُ s1 = %+v، والمتوقّع a ثمّ c بترتيبها", got[1].Items)
	}
	for _, s := range got {
		for _, it := range s.Items {
			if it.ID == "x" {
				t.Fatal("صنفٌ قسمُه غائبٌ عُرض")
			}
		}
	}
	if out := groupStoreSections(nil, nil); out == nil || len(out) != 0 {
		t.Fatalf("لا أقسام ⇒ قائمةٌ فارغةٌ لا null: %v", out)
	}
}

// storesFixture متاجرُ ومدينةٌ وأقسامٌ لاختبارات الشريط والصفحة.
type storesFixture struct {
	srv      *Server
	r        *chi.Mux
	cityLat  float64
	cityLng  float64
	category string
}

func newStoresFixture(t *testing.T) *storesFixture {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	catalogSvc := catalog.NewService(pool, nil)
	settingsStore := settings.NewStore(pool)
	catalogSvc.SetSettings(settingsStore)
	srv := &Server{pg: pool, logger: quiet, hub: realtime.NewHub(quiet),
		catalog: catalogSvc, settings: settingsStore}

	f := &storesFixture{srv: srv, cityLat: 21.0, cityLng: 61.0}
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&f.category); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	r := chi.NewRouter()
	r.Get("/home", srv.handlePublicHome)
	r.Get("/stores/{id}", srv.handlePublicStore)
	r.Get("/sections/{id}/items", srv.handlePublicSectionItems)
	f.r = r
	return f
}

func (f *storesFixture) city(t *testing.T) string {
	t.Helper()
	pool := f.srv.pg
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO cities (name, center, radius_m, active)
		VALUES ('مدينةُ المتاجر المميّزة', ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, 5000, true)
		RETURNING id`, f.cityLng, f.cityLat).Scan(&id); err != nil {
		t.Fatalf("تعذّرت المدينة: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM cities WHERE id = $1`, id) })
	return id
}

// merchant متجرٌ في المدينة إن أُعطيت، وإلّا بلا مدينة.
func (f *storesFixture) merchant(t *testing.T, name, status string, featured bool, order int, cityID string) string {
	t.Helper()
	pool := f.srv.pg
	var id string
	var city any
	if cityID != "" {
		city = cityID
	}
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO merchants (name, category_id, commission_percent, status, featured, featured_order,
		                       city_id, location)
		VALUES ($1, $2, 10, $3, $4, $5, $6::uuid,
		        ST_SetSRID(ST_MakePoint($7, $8), 4326)::geography)
		RETURNING id`, name, f.category, status, featured, order, city, f.cityLng, f.cityLat).Scan(&id); err != nil {
		t.Fatalf("تعذّر المتجر %s: %v", name, err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, id) })
	return id
}

func (f *storesFixture) section(t *testing.T, name string, sort int, active bool) string {
	t.Helper()
	pool := f.srv.pg
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO platform_sections (name, icon, sort_order, active)
		VALUES ($1, 'food', $2, $3) RETURNING id`, name, sort, active).Scan(&id); err != nil {
		t.Fatalf("تعذّر القسم %s: %v", name, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, id)
	})
	return id
}

// item صنفٌ في قسم منصّةٍ — و`psID` الفارغُ صنفٌ بلا قسم منصّة.
func (f *storesFixture) item(t *testing.T, merchantID, psID, name string, available bool) string {
	t.Helper()
	pool := f.srv.pg
	ctx := context.Background()
	var secID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'الرئيسية')
		RETURNING id`, merchantID).Scan(&secID); err != nil {
		t.Fatalf("تعذّر قسمُ القائمة: %v", err)
	}
	var ps any
	if psID != "" {
		ps = psID
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, section_id, platform_section_id,
		                        name, merchant_price, price, available, approved)
		VALUES ($1, $2, $3::uuid, $4, 10000, 10000, $5, true) RETURNING id`,
		merchantID, secID, ps, name, available).Scan(&id); err != nil {
		t.Fatalf("تعذّر الصنف %s: %v", name, err)
	}
	return id
}

func (f *storesFixture) get(t *testing.T, path string) (int, []byte) {
	t.Helper()
	w := httptest.NewRecorder()
	f.r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Code, w.Body.Bytes()
}

// TestFeaturedStores_OrderAndScope **الشريطُ بترتيب الأدمن ثمّ الاسم، وسوقُ المدينة وحدَه.**
//
// **وغيرُ المميّز والموقوفُ لا يظهران** — ومتجرٌ خارجَ مدينة الزبون لا يظهر لمن
// أرسل موضعَه، **ويظهر لمن لم يرسل** (العميلُ القديمُ يرى ما كان يرى).
func TestFeaturedStores_OrderAndScope(t *testing.T) {
	f := newStoresFixture(t)
	city := f.city(t)
	// **ترتيبٌ ٢ ثمّ ١ ثمّ ١** — والمتساويان بالاسم.
	third := f.merchant(t, "متجر مميز ج", "active", true, 2, city)
	second := f.merchant(t, "متجر مميز ب", "active", true, 1, city)
	first := f.merchant(t, "متجر مميز أ", "active", true, 1, city)
	plain := f.merchant(t, "متجر عادي", "active", false, 0, city)
	off := f.merchant(t, "متجر مميز موقوف", "inactive", true, 0, city)
	// **وآخرُ بلا مدينة** — خارجَ سوقِ من أرسل موضعَه.
	away := f.merchant(t, "متجر مميز بعيد", "active", true, 0, "")

	read := func(path string) []string {
		code, body := f.get(t, path)
		if code != http.StatusOK {
			t.Fatalf("%s ردّ %d — %s", path, code, body)
		}
		var out struct {
			Data struct {
				Featured []featuredStore `json:"featured_stores"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("ردٌّ لا يُفكّ: %v", err)
		}
		ids := []string{}
		for _, s := range out.Data.Featured {
			ids = append(ids, s.ID)
			if s.ID == first && !s.OpenNow {
				t.Errorf("متجرٌ بلا دوامٍ يُقرأ مغلقاً — والقاعدةُ «لا صفوفَ تعني مفتوحاً»")
			}
		}
		return ids
	}
	pick := func(ids []string, want ...string) []string {
		set := map[string]bool{}
		for _, w := range want {
			set[w] = true
		}
		out := []string{}
		for _, id := range ids {
			if set[id] {
				out = append(out, id)
			}
		}
		return out
	}
	all := []string{first, second, third, plain, off, away}

	scoped := pick(read("/home?lat=21.0&lng=61.0"), all...)
	want := []string{first, second, third}
	if len(scoped) != len(want) {
		t.Fatalf("الشريطُ بالموضع = %v، والمتوقّع %v (لا عاديّ ولا موقوف ولا بعيد)", scoped, want)
	}
	for i := range want {
		if scoped[i] != want[i] {
			t.Fatalf("ترتيبُ الشريط = %v، والمتوقّع %v (الترتيبُ ثمّ الاسم)", scoped, want)
		}
	}

	open := pick(read("/home"), all...)
	if len(open) != 4 || open[0] != away {
		t.Fatalf("بلا موضع = %v، والمتوقّع البعيدَ أوّلاً (ترتيبُه ٠) ثمّ الثلاثة", open)
	}
}

// TestPublicStore_GroupingAndLeaks **صفحةُ المتجر: أقسامُه بترتيبها، ولا يتسرّب منها ما لا يظهر في قسمه.**
func TestPublicStore_GroupingAndLeaks(t *testing.T) {
	f := newStoresFixture(t)
	store := f.merchant(t, "متجر الصفحة", "active", true, 0, "")
	other := f.merchant(t, "متجر آخر", "active", false, 0, "")

	late := f.section(t, "قسم الصفحة الثاني", 902, true)
	early := f.section(t, "قسم الصفحة الأول", 901, true)
	dead := f.section(t, "قسم الصفحة المطفأ", 900, false)

	a := f.item(t, store, early, "صنف أ", true)
	b := f.item(t, store, late, "صنف ب", true)
	sleepy := f.item(t, store, early, "صنف غير متوفر", false)
	hidden := f.item(t, store, dead, "صنف قسمه مطفأ", true)
	// **ولا صنفَ بلا قسم** — `menu_items.platform_section_id` إلزاميّ في القاعدة، فالحالةُ لا تقع.
	foreign := f.item(t, other, early, "صنف متجر آخر", true)

	code, body := f.get(t, "/stores/"+store)
	if code != http.StatusOK {
		t.Fatalf("الصفحةُ ردّت %d — %s", code, body)
	}
	var out struct {
		Data struct {
			Store    storeHeader `json:"store"`
			Sections []struct {
				ID    string           `json:"id"`
				Name  string           `json:"name"`
				Items []map[string]any `json:"items"`
			} `json:"sections"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v", err)
	}
	if out.Data.Store.ID != store || out.Data.Store.Name != "متجر الصفحة" || !out.Data.Store.OpenNow {
		t.Errorf("الرأس = %+v", out.Data.Store)
	}
	if out.Data.Store.PrepMinutes <= 0 {
		t.Errorf("مدّةُ التحضير = %d، والمتوقّع افتراضُ المتجر", out.Data.Store.PrepMinutes)
	}
	if len(out.Data.Sections) != 2 || out.Data.Sections[0].ID != early || out.Data.Sections[1].ID != late {
		t.Fatalf("الأقسام = %+v، والمتوقّع الأولَ ثمّ الثاني (بترتيب المنصّة، والمطفأُ غائب)", out.Data.Sections)
	}
	// **والمتاحُ أوّلاً داخل القسم** — وغيرُ المتوفّر يُعرض ولا يُخفى، كالقسم.
	firstSec := out.Data.Sections[0].Items
	if len(firstSec) != 2 || firstSec[0]["id"] != a || firstSec[1]["id"] != sleepy {
		t.Fatalf("أصنافُ القسم الأوّل = %v، والمتوقّع أ ثمّ غيرَ المتوفّر", firstSec)
	}
	if firstSec[1]["available"] != false {
		t.Errorf("غيرُ المتوفّر خرج متاحاً")
	}
	if out.Data.Sections[1].Items[0]["id"] != b {
		t.Errorf("القسمُ الثاني لا يحمل ب")
	}
	// **وشكلُ الصنف شكلُ القسم** — بلا سعر الشراء وهامشه، وباسم متجره.
	for _, k := range []string{"merchant_price", "margin_override"} {
		if _, ok := firstSec[0][k]; ok {
			t.Errorf("الحقلُ %q خرج إلى الزبون", k)
		}
	}
	for _, k := range []string{"price", "has_options", "source_closed", "merchant_name", "section_id"} {
		if _, ok := firstSec[0][k]; !ok {
			t.Errorf("الحقلُ %q غائبٌ عن صنف الصفحة — والبطاقةُ تقرؤه", k)
		}
	}
	for _, s := range out.Data.Sections {
		for _, it := range s.Items {
			switch it["id"] {
			case hidden, foreign:
				t.Errorf("صنفٌ لا يظهر في قسمه تسرّب من الصفحة: %v", it["name"])
			}
		}
	}
}

// TestPublicStore_NotFound **ما لا وجودَ له أو غيرُ فعّالٍ أو خارجَ السوق ⇒ ٤٠٤.**
func TestPublicStore_NotFound(t *testing.T) {
	f := newStoresFixture(t)
	off := f.merchant(t, "متجر موقوف للصفحة", "inactive", true, 0, "")
	away := f.merchant(t, "متجر خارج السوق", "active", true, 0, "")
	for _, path := range []string{
		"/stores/not-a-uuid",
		"/stores/00000000-0000-0000-0000-000000000000",
		"/stores/" + off,
		// **وبموضعٍ لا يصله** — متجرٌ بلا مدينةٍ خارجَ كلّ سوق.
		"/stores/" + away + "?lat=21.0&lng=61.0",
	} {
		if code, body := f.get(t, path); code != http.StatusNotFound {
			t.Errorf("%s ردّ %d، والمتوقّع ٤٠٤ — %s", path, code, body)
		}
	}
	// **وبلا موضعٍ يُفتح** — العميلُ القديمُ لا يُرشَّح.
	if code, _ := f.get(t, "/stores/"+away); code != http.StatusOK {
		t.Errorf("بلا موضعٍ ردّ %d، والمتوقّع ٢٠٠", code)
	}
}

// TestPublicStore_TodayHours **دوامُ اليوم في الرأس** (٢٠٢٦-١٠-١٨) — ٢٤ ساعة، أو من … لـ …، أو مغلقٌ اليوم،
// **وبلا صفوفِ دوامٍ ٢٤ ساعة.**
func TestPublicStore_TodayHours(t *testing.T) {
	f := newStoresFixture(t)
	sec := f.section(t, "قسم الدوام", 950, true)
	loc, _ := time.LoadLocation("Asia/Damascus")
	dow := int(time.Now().In(loc).Weekday())
	cases := []struct {
		name               string
		rows               bool
		closed             bool
		open, close        string
		wantClosed, want24 bool
		wantOpen, wantShut string
	}{
		{name: "بلا دوام", rows: false, want24: true},
		{name: "٢٤ ساعة", rows: true, open: "00:00", close: "00:00", want24: true},
		{name: "عادي", rows: true, open: "09:00", close: "23:00", wantOpen: "09:00", wantShut: "23:00"},
		{name: "عطلة", rows: true, closed: true, open: "09:00", close: "23:00", wantClosed: true},
	}
	for _, c := range cases {
		id := f.merchant(t, "متجر "+c.name, "active", false, 0, "")
		f.item(t, id, sec, "صنف "+c.name, true)
		if c.rows {
			for d := 0; d < 7; d++ {
				if _, err := f.srv.pg.Exec(context.Background(), `
					INSERT INTO merchant_hours (merchant_id, day_of_week, closed, open_time, close_time)
					VALUES ($1, $2, $3, $4::time, $5::time)`,
					id, d, c.closed && d == dow, c.open, c.close); err != nil {
					t.Fatalf("%s: تعذّر الدوام: %v", c.name, err)
				}
			}
		}
		code, body := f.get(t, "/stores/"+id)
		if code != http.StatusOK {
			t.Fatalf("%s: %d — %s", c.name, code, body)
		}
		var out struct {
			Data struct {
				Store struct {
					Today struct {
						Closed bool   `json:"closed"`
						Open24 bool   `json:"open_24h"`
						Open   string `json:"open"`
						Close  string `json:"close"`
					} `json:"today"`
				} `json:"store"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatal(err)
		}
		g := out.Data.Store.Today
		if g.Closed != c.wantClosed || g.Open24 != c.want24 || g.Open != c.wantOpen || g.Close != c.wantShut {
			t.Errorf("%s: دوامُ اليوم = %+v", c.name, g)
		}
	}
}
