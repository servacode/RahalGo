package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestPriceSyncSections_Routes **أبواب توزيع الأقسام**: قائمة الانتظار صفحة صفحة، والنقل بالإيد
// بيصير يدويّ، و«وزّع تلقائياً» بيردّ العدد — ومتجر غيره ممنوع.
func TestPriceSyncSections_Routes(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	other := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET owner_user_id = $2, uses_ameen = true WHERE id = $1`,
		f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	var target string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, sort_order) VALUES ('قسم نقل للاختبار', 951) RETURNING id::text`).Scan(&target); err != nil {
		t.Fatal(err)
	}
	var pend string
	if err := f.pool.QueryRow(ctx, `
		SELECT id::text FROM platform_sections WHERE section_name_key(name) = section_name_key('من الأمين — بانتظار التصنيف')
		 ORDER BY active DESC LIMIT 1`).Scan(&pend); err != nil {
		if err := f.pool.QueryRow(ctx, `INSERT INTO platform_sections (name, sort_order, active)
			VALUES ('من الأمين — بانتظار التصنيف', 9999, false) RETURNING id::text`).Scan(&pend); err != nil {
			t.Fatal(err)
		}
	}
	var ids []string
	for i, name := range []string{"مادة انتظار ١", "مادة انتظار ٢", "مادة انتظار ٣"} {
		var id string
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO menu_items (merchant_id, platform_section_id, name, price, merchant_price, available,
			                        section_source, sort_order)
			VALUES ($1, $2, $3, 10, 10, false, 'pending', $4) RETURNING id::text`, f.merchantID, pend, name, i+1).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM menu_items WHERE merchant_id = $1`, f.merchantID)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, target)
	})

	base := "/merchant/stores/" + f.merchantID + "/price-sync"
	w := f.call(f.srv.handleMerchantPriceSyncPending, http.MethodGet, base+"/pending?limit=2", f.merchantID, owner, []string{"merchant"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("pending: %d %s", w.Code, w.Body.String())
	}
	var env struct {
		Data struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
			Total int    `json:"total"`
			Next  string `json:"next"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	page := env.Data
	if page.Total != 3 || len(page.Items) != 2 || page.Next == "" || page.Items[0].ID != ids[0] {
		t.Fatalf("الصفحة: %+v %s", page, w.Body.String())
	}

	body := `{"item_ids":["` + ids[0] + `","` + ids[1] + `"],"platform_section_id":"` + target + `"}`
	if w := f.call(f.srv.handleMerchantPriceSyncMove, http.MethodPost, base+"/move", f.merchantID, other, []string{"merchant"}, body); w.Code == http.StatusOK {
		t.Fatalf("متجر غيره نقل: %d", w.Code)
	}
	w = f.call(f.srv.handleMerchantPriceSyncMove, http.MethodPost, base+"/move", f.merchantID, owner, []string{"merchant"}, body)
	if w.Code != http.StatusOK {
		t.Fatalf("move: %d %s", w.Code, w.Body.String())
	}
	var src, sec string
	if err := f.pool.QueryRow(ctx, `SELECT section_source, platform_section_id::text FROM menu_items WHERE id = $1`, ids[0]).
		Scan(&src, &sec); err != nil || src != "manual" || sec != target {
		t.Fatalf("بعد النقل: %s %s %v", src, sec, err)
	}

	w = f.call(f.srv.handleMerchantPriceSyncAutoSort, http.MethodPost, base+"/auto-sort", f.merchantID, owner, []string{"merchant"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("auto-sort: %d %s", w.Code, w.Body.String())
	}
	var res struct {
		Data struct {
			Waiting int `json:"waiting"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || res.Data.Waiting != 1 {
		t.Fatalf("auto-sort: %s %v", w.Body.String(), err)
	}

	w = f.call(f.srv.handleMerchantPriceSyncGroups, http.MethodGet, base+"/groups", f.merchantID, owner, []string{"merchant"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("groups: %d %s", w.Code, w.Body.String())
	}
	w = f.call(f.srv.handleMerchantPriceSyncSetGroups, http.MethodPut, base+"/groups", f.merchantID, owner, []string{"merchant"},
		`[{"group_key":"مجموعة","platform_section_id":"`+target+`"}]`)
	if w.Code != http.StatusOK {
		t.Fatalf("put groups: %d %s", w.Code, w.Body.String())
	}
}
