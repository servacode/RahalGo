package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// pagedMenuFixture a store with many items in three market sections, an owner, and a router
// holding both the old full menus and the new paged readers.
type pagedMenuFixture struct {
	*storesFixture
	merchantID string
	owner      string
	secs       []string
	mr         *chi.Mux
}

func newPagedMenuFixture(t *testing.T, n int) *pagedMenuFixture {
	t.Helper()
	sf := newStoresFixture(t)
	f := &pagedMenuFixture{storesFixture: sf}
	pool := sf.srv.pg
	ctx := context.Background()
	f.merchantID = sf.merchant(t, "paged menu store", "active", false, 0, "")
	f.owner = testdb.NewUser(t, pool, "merchant")
	if _, err := pool.Exec(ctx, `UPDATE merchants SET owner_user_id = $2 WHERE id = $1`, f.merchantID, f.owner); err != nil {
		t.Fatal(err)
	}
	tag := f.merchantID[:8]
	f.secs = []string{
		sf.section(t, "paged sec c "+tag, 9301, true),
		sf.section(t, "paged sec a "+tag, 9300, true),
		sf.section(t, "paged sec b "+tag, 9302, true),
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM menu_items WHERE merchant_id = $1`, f.merchantID)
	})
	// n items spread over the three sections: duplicate names (ties broken by id), every
	// seventh unavailable, every fifth with a margin override, every eleventh priced in USD,
	// every thirteenth with an Ameen code.
	if _, err := pool.Exec(ctx, `
		INSERT INTO menu_items (merchant_id, platform_section_id, name, price, merchant_price, available,
		                        margin_override, price_usd_cents, external_code, sort_order)
		SELECT $1, ($2::uuid[])[1 + g % 3], 'item ' || lpad((g % ($3 / 2 + 1))::text, 6, '0'),
		       1000 + g, 1000 + g, g % 7 <> 0,
		       CASE WHEN g % 5 = 0 THEN 250 END,
		       CASE WHEN g % 11 = 0 THEN 150 END,
		       CASE WHEN g % 13 = 0 THEN 'AM' || g END,
		       g % 4
		FROM generate_series(1, $3) g`, f.merchantID, f.secs, n); err != nil {
		t.Fatal(err)
	}
	// modifiers on a few items, with distinct sort orders
	if _, err := pool.Exec(ctx, `
		WITH it AS (SELECT id FROM menu_items WHERE merchant_id = $1 AND external_code IS NOT NULL ORDER BY id LIMIT 20),
		     g AS (INSERT INTO modifier_groups (item_id, name, min_select, max_select, sort_order)
		           SELECT id, 'size', 0, 1, 1 FROM it RETURNING id)
		INSERT INTO modifier_options (group_id, name, price_delta, sort_order)
		SELECT g.id, o.n, o.d, o.s FROM g, (VALUES ('small', 0, 1), ('big', 500, 2)) o(n, d, s)`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	mr := chi.NewRouter()
	mr.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := context.WithValue(r.Context(), ctxUserID, r.Header.Get("X-Test-User"))
			c = context.WithValue(c, ctxRoles, []string{"merchant"})
			next.ServeHTTP(w, r.WithContext(c))
		})
	})
	mr.Get("/merchant/stores/{id}/menu", sf.srv.handleMerchantMenu)
	mr.Get("/merchant/stores/{id}/menu/summary", sf.srv.handleMerchantMenuSummary)
	mr.Get("/merchant/stores/{id}/menu/items", sf.srv.handleMerchantMenuItems)
	mr.Get("/public/stores/{id}", sf.srv.handlePublicStore)
	mr.Get("/public/stores/{id}/summary", sf.srv.handlePublicStoreSummary)
	mr.Get("/public/stores/{id}/items", sf.srv.handlePublicStoreItems)
	f.mr = mr
	return f
}

func (f *pagedMenuFixture) get(t *testing.T, user, path string) (int, []byte) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Test-User", user)
	w := httptest.NewRecorder()
	f.mr.ServeHTTP(w, req)
	return w.Code, w.Body.Bytes()
}

type pagedItemsResp struct {
	Data struct {
		Items     []json.RawMessage `json:"items"`
		NextAfter string            `json:"next_after"`
		Total     int               `json:"total"`
	} `json:"data"`
}

type idOnly struct {
	ID string `json:"id"`
}

// walk pages through a paged endpoint and returns the raw items in order, the first-page total
// and the number of pages.
func (f *pagedMenuFixture) walk(t *testing.T, user, base string) ([]json.RawMessage, int, int) {
	t.Helper()
	var all []json.RawMessage
	after, total, pages := "", -2, 0
	for {
		u := base
		if after != "" {
			u += "&after=" + url.QueryEscape(after)
		}
		code, body := f.get(t, user, u)
		if code != http.StatusOK {
			t.Fatalf("%s: %d %s", u, code, body)
		}
		var p pagedItemsResp
		if err := json.Unmarshal(body, &p); err != nil {
			t.Fatal(err)
		}
		if pages == 0 {
			total = p.Data.Total
		} else if p.Data.Total != -1 {
			t.Fatalf("later page carries total %d", p.Data.Total)
		}
		pages++
		all = append(all, p.Data.Items...)
		if p.Data.NextAfter == "" {
			break
		}
		after = p.Data.NextAfter
		if pages > 1000 {
			t.Fatal("paging does not end")
		}
	}
	return all, total, pages
}

func ids(t *testing.T, raw []json.RawMessage) []string {
	t.Helper()
	out := make([]string, len(raw))
	seen := map[string]bool{}
	for i, r := range raw {
		var x idOnly
		if err := json.Unmarshal(r, &x); err != nil {
			t.Fatal(err)
		}
		if seen[x.ID] {
			t.Fatalf("item %s repeated across pages (at %d)", x.ID, i)
		}
		seen[x.ID] = true
		out[i] = x.ID
	}
	return out
}

// TestPagedMenu_Merchant: the owner's paged menu holds exactly the items of the full menu, each
// one identical (price, currency, modifiers…), in a stable order with no repeats or gaps across
// pages; section, search and availability filters; the summary counts; other owners refused;
// the old full menu unchanged.
func TestPagedMenu_Merchant(t *testing.T) {
	const n = 1200
	f := newPagedMenuFixture(t, n)
	pool := f.srv.pg
	ctx := context.Background()
	stranger := testdb.NewUser(t, pool, "merchant")
	base := "/merchant/stores/" + f.merchantID

	for _, p := range []string{"/menu/summary", "/menu/items"} {
		if code, _ := f.get(t, stranger, base+p); code != http.StatusForbidden {
			t.Fatalf("%s for another owner: %d", p, code)
		}
	}

	// the old full menu, as before: every sectioned item, merchant price in `price`
	code, body := f.get(t, f.owner, base+"/menu")
	if code != http.StatusOK {
		t.Fatalf("old menu: %d", code)
	}
	var old struct {
		Data []struct {
			ID    string            `json:"id"`
			Items []json.RawMessage `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &old); err != nil {
		t.Fatal(err)
	}
	oldItems := map[string]string{}
	for _, s := range old.Data {
		for _, it := range s.Items {
			var x idOnly
			_ = json.Unmarshal(it, &x)
			oldItems[x.ID] = string(it)
		}
	}
	if len(oldItems) != n {
		t.Fatalf("old menu items = %d, want %d", len(oldItems), n)
	}

	// summary
	code, body = f.get(t, f.owner, base+"/menu/summary")
	if code != http.StatusOK {
		t.Fatalf("summary: %d %s", code, body)
	}
	var sum struct {
		Data struct {
			Total       int  `json:"total"`
			Unavailable int  `json:"unavailable"`
			NoPhoto     int  `json:"no_photo"`
			WaItemsOn   bool `json:"wa_items_on"`
			Sections    []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Count       int    `json:"count"`
				Unavailable int    `json:"unavailable"`
			} `json:"sections"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &sum); err != nil {
		t.Fatal(err)
	}
	// the "add items by WhatsApp" hint: on only when the global switch AND the store flag are on
	// (owner 2026-11-11). The global default is off.
	if sum.Data.WaItemsOn {
		t.Fatalf("wa_items_on must be false while the global switch is off")
	}
	waOn := func(global, store bool) bool {
		if _, err := pool.Exec(ctx, `UPDATE merchants SET wa_items_enabled = $2 WHERE id = $1`, f.merchantID, store); err != nil {
			t.Fatal(err)
		}
		f.srv.settings.Set(ctx, "merchants.wa_items_enabled", global, nil)
		_, b := f.get(t, f.owner, base+"/menu/summary")
		var r struct {
			Data struct {
				WaItemsOn bool `json:"wa_items_on"`
			} `json:"data"`
		}
		_ = json.Unmarshal(b, &r)
		return r.Data.WaItemsOn
	}
	for _, c := range []struct{ g, s, want bool }{{false, true, false}, {true, false, false}, {true, true, true}} {
		if got := waOn(c.g, c.s); got != c.want {
			t.Fatalf("wa_items_on(global=%v store=%v) = %v, want %v", c.g, c.s, got, c.want)
		}
	}
	f.srv.settings.Set(ctx, "merchants.wa_items_enabled", false, nil)
	var wantOff int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM menu_items WHERE merchant_id = $1 AND platform_section_id IS NOT NULL AND NOT available`, f.merchantID).Scan(&wantOff)
	if sum.Data.Total != n || sum.Data.Unavailable != wantOff || sum.Data.NoPhoto != n {
		t.Fatalf("summary totals: %+v (want total %d off %d nophoto %d)", sum.Data, n, wantOff, n)
	}
	if len(sum.Data.Sections) != 3 || sum.Data.Sections[0].ID != f.secs[1] ||
		sum.Data.Sections[1].ID != f.secs[0] || sum.Data.Sections[2].ID != f.secs[2] {
		t.Fatalf("summary sections order: %+v", sum.Data.Sections)
	}

	// every page, two page sizes: same sequence, no repeats, exactly the full menu
	all, total, pages := f.walk(t, f.owner, base+"/menu/items?limit=200")
	if total != n || len(all) != n || pages != 6 {
		t.Fatalf("pages: total=%d items=%d pages=%d", total, len(all), pages)
	}
	seq := ids(t, all)
	odd, _, _ := f.walk(t, f.owner, base+"/menu/items?limit=77")
	seq2 := ids(t, odd)
	for i := range seq {
		if seq[i] != seq2[i] {
			t.Fatalf("order differs between page sizes at %d", i)
		}
	}
	// the expected order: section (sort, name, id), then item name, then id
	rows, err := pool.Query(ctx, `
		SELECT i.id FROM menu_items i JOIN platform_sections ps ON ps.id = i.platform_section_id
		WHERE i.merchant_id = $1 ORDER BY ps.sort_order, ps.name, ps.id, i.name, i.id`, f.merchantID)
	if err != nil {
		t.Fatal(err)
	}
	k := 0
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		if k >= len(seq) || seq[k] != id {
			rows.Close()
			t.Fatalf("order at %d", k)
		}
		k++
	}
	rows.Close()
	// each item identical to the full menu's
	for i, raw := range all {
		if got, want := string(raw), oldItems[seq[i]]; got != want {
			t.Fatalf("item differs from the full menu:\n got %s\nwant %s", got, want)
		}
	}

	// a section: exactly its count
	for _, s := range sum.Data.Sections {
		items, tot, _ := f.walk(t, f.owner, base+"/menu/items?limit=150&section="+s.ID)
		if tot != s.Count || len(items) != s.Count {
			t.Fatalf("section %s: total %d items %d want %d", s.Name, tot, len(items), s.Count)
		}
		ids(t, items)
	}
	// unavailable only
	off, tot, _ := f.walk(t, f.owner, base+"/menu/items?filter=unavailable&limit=50")
	if tot != wantOff || len(off) != wantOff {
		t.Fatalf("unavailable: %d/%d want %d", tot, len(off), wantOff)
	}
	// available only — the offer item picker (owner 2026-11-08: the offers page was slow)
	on, tot, _ := f.walk(t, f.owner, base+"/menu/items?filter=available&limit=150")
	if tot != len(all)-wantOff || len(on) != len(all)-wantOff {
		t.Fatalf("available: %d/%d want %d", tot, len(on), len(all)-wantOff)
	}
	// search by name and by exact Ameen code
	var wantQ int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM menu_items WHERE merchant_id = $1 AND platform_section_id IS NOT NULL AND name LIKE '%00012%'`, f.merchantID).Scan(&wantQ)
	found, tot, _ := f.walk(t, f.owner, base+"/menu/items?q=00012&limit=5")
	if wantQ == 0 || tot != wantQ || len(found) != wantQ {
		t.Fatalf("search: %d/%d want %d", tot, len(found), wantQ)
	}
	found, tot, _ = f.walk(t, f.owner, base+"/menu/items?q=AM26")
	if tot != 1 || len(found) != 1 {
		t.Fatalf("code search: %d", tot)
	}
	// LIKE wildcards are literal
	if _, tot, _ = f.walk(t, f.owner, base+"/menu/items?q=%25"); tot != 0 {
		t.Fatalf("%% matched %d", tot)
	}
	// one item by id (editor, duplicate)
	one, tot, _ := f.walk(t, f.owner, base+"/menu/items?item="+seq[10])
	if tot != 1 || string(one[0]) != oldItems[seq[10]] {
		t.Fatalf("item by id: %d", tot)
	}
	// bad cursor and filter
	if code, _ := f.get(t, f.owner, base+"/menu/items?after=@@"); code != http.StatusBadRequest {
		t.Fatalf("bad cursor: %d", code)
	}
	if code, _ := f.get(t, f.owner, base+"/menu/items?filter=nope"); code != http.StatusBadRequest {
		t.Fatalf("bad filter: %d", code)
	}
	// the item a cursor points at is deleted between two pages: the next page still follows it
	code, body = f.get(t, f.owner, base+"/menu/items?limit=100")
	var p1 pagedItemsResp
	_ = json.Unmarshal(body, &p1)
	if _, err := pool.Exec(ctx, `DELETE FROM menu_items WHERE id = $1`, seq[99]); err != nil {
		t.Fatal(err)
	}
	code, body = f.get(t, f.owner, base+"/menu/items?limit=1&after="+url.QueryEscape(p1.Data.NextAfter))
	var p2 pagedItemsResp
	_ = json.Unmarshal(body, &p2)
	if code != http.StatusOK || len(p2.Data.Items) != 1 || ids(t, p2.Data.Items)[0] != seq[100] {
		t.Fatalf("after a deleted cursor item: %d %s", code, body)
	}
}

// TestPagedMenu_Customer: the customer's paged store holds the items of the store page with the
// same sale prices, in a stable order without repeats; section counts and search.
func TestPagedMenu_Customer(t *testing.T) {
	const n = 1100
	f := newPagedMenuFixture(t, n)
	base := "/public/stores/" + f.merchantID

	code, body := f.get(t, "", base+"/summary")
	if code != http.StatusOK {
		t.Fatalf("summary: %d %s", code, body)
	}
	var sum struct {
		Data struct {
			Store struct {
				ID string `json:"id"`
			} `json:"store"`
			Sections []struct {
				ID    string `json:"id"`
				Count int    `json:"count"`
			} `json:"sections"`
			Total int `json:"total"`
		} `json:"data"`
	}
	_ = json.Unmarshal(body, &sum)
	if sum.Data.Store.ID != f.merchantID || sum.Data.Total != n || len(sum.Data.Sections) != 3 {
		t.Fatalf("summary: %+v", sum.Data)
	}

	all, total, _ := f.walk(t, "", base+"/items?limit=200")
	if total != n || len(all) != n {
		t.Fatalf("items: total %d got %d", total, len(all))
	}
	seq := ids(t, all)
	odd, _, _ := f.walk(t, "", base+"/items?limit=61")
	seq2 := ids(t, odd)
	for i := range seq {
		if seq[i] != seq2[i] {
			t.Fatalf("order differs between page sizes at %d", i)
		}
	}
	// prices as the old store page (which caps at 500 items) for every item both show
	code, body = f.get(t, "", base)
	if code != http.StatusOK {
		t.Fatalf("old page: %d", code)
	}
	var old struct {
		Data struct {
			Sections []struct {
				Items []json.RawMessage `json:"items"`
			} `json:"sections"`
		} `json:"data"`
	}
	_ = json.Unmarshal(body, &old)
	oldByID := map[string]string{}
	for _, s := range old.Data.Sections {
		for _, it := range s.Items {
			var x idOnly
			_ = json.Unmarshal(it, &x)
			oldByID[x.ID] = string(it)
		}
	}
	if len(oldByID) != 500 {
		t.Fatalf("old page items = %d (cap 500)", len(oldByID))
	}
	same := 0
	for i, raw := range all {
		if want, ok := oldByID[seq[i]]; ok {
			if string(raw) != want {
				t.Fatalf("item differs from the store page:\n got %s\nwant %s", raw, want)
			}
			same++
		}
	}
	if same != 500 {
		t.Fatalf("compared %d", same)
	}
	for _, s := range sum.Data.Sections {
		items, tot, _ := f.walk(t, "", base+"/items?section="+s.ID)
		if tot != s.Count || len(items) != s.Count {
			t.Fatalf("section: %d/%d want %d", tot, len(items), s.Count)
		}
		// the available first inside a section
		seenOff := false
		for _, raw := range items {
			var x struct {
				Available bool `json:"available"`
			}
			_ = json.Unmarshal(raw, &x)
			if !x.Available {
				seenOff = true
			} else if seenOff {
				t.Fatal("an available item after an unavailable one")
			}
		}
	}
	if _, tot, _ := f.walk(t, "", base+"/items?q=00012"); tot == 0 {
		t.Fatal("search found nothing")
	}
	if code, _ := f.get(t, "", "/public/stores/not-a-uuid/items"); code != http.StatusNotFound {
		t.Fatalf("bad store: %d", code)
	}
}

// TestPagedMenu_BigStoreFirstPage: with 15,000 items the first page is small and quick, for the
// store owner and the customer. Also logs a small store, old against new.
func TestPagedMenu_BigStoreFirstPage(t *testing.T) {
	const n = 15000
	f := newPagedMenuFixture(t, n)
	timeIt := func(user, path string) (time.Duration, int) {
		start := time.Now()
		code, body := f.get(t, user, path)
		d := time.Since(start)
		if code != http.StatusOK {
			t.Fatalf("%s: %d", path, code)
		}
		var gz bytes.Buffer
		zw := gzip.NewWriter(&gz)
		_, _ = zw.Write(body)
		_ = zw.Close()
		t.Logf("  gzip %d bytes", gz.Len())
		return d, len(body)
	}
	m := "/merchant/stores/" + f.merchantID
	c := "/public/stores/" + f.merchantID
	// warm the pool and the plan cache once
	f.get(t, f.owner, m+"/menu/items")
	for _, x := range []struct{ user, path string }{
		{f.owner, m + "/menu/summary"},
		{f.owner, m + "/menu/items"},
		{"", c + "/summary"},
		{"", c + "/items"},
	} {
		d, size := timeIt(x.user, x.path)
		t.Logf("15000 items: %-28s %6d bytes  %v", x.path[len(x.path)-min(len(x.path), 28):], size, d)
		if size > 100*1024 {
			t.Fatalf("%s: first answer %d bytes > 100 KB", x.path, size)
		}
	}
	d, size := timeIt(f.owner, m+"/menu")
	t.Logf("15000 items: old full menu %d bytes  %v", size, d)

	small := newPagedMenuFixture(t, 40)
	sm := "/merchant/stores/" + small.merchantID
	small.get(t, small.owner, sm+"/menu")
	oldStart := time.Now()
	for i := 0; i < 5; i++ {
		small.get(t, small.owner, sm+"/menu")
	}
	oldD := time.Since(oldStart) / 5
	newStart := time.Now()
	for i := 0; i < 5; i++ {
		small.get(t, small.owner, sm+"/menu/summary")
		small.get(t, small.owner, sm+"/menu/items")
	}
	newD := time.Since(newStart) / 5
	t.Logf("40 items: old full menu %v, summary+first page %v", oldD, newD)
}
