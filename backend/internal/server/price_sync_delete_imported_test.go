package server

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestPriceSyncDeleteImported_OnlyAmeenCreated **«احذف الأصناف المستوردة من الأمين»** (قرارُ
// المالك ٢٠٢٦-١١-٠٨: «أ»): يحذف ما أنشأته المزامنةُ وحدَه — **والصنفُ المُضاف باليد يبقى ولو
// رُبط برقم مادّة** — والعددُ يظهر في معلومات الصفحة قبل الحذف، والغريبُ لا يحذف.
func TestPriceSyncDeleteImported_OnlyAmeenCreated(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx,
		`UPDATE merchants SET uses_ameen = true, owner_user_id = $2 WHERE id = $1`, f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	var secID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, sort_order) VALUES ('قسمُ حذفِ المستورد للاختبار', 952) RETURNING id`).Scan(&secID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM menu_items WHERE platform_section_id = $1`, secID)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, secID)
	})
	add := func(name, code string, imported bool) string {
		var id string
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO menu_items (merchant_id, platform_section_id, name, price, merchant_price, available,
			                        external_code, ameen_created)
			VALUES ($1, $2, $3, 100, 100, true, NULLIF($4, ''), $5) RETURNING id`,
			f.merchantID, secID, name, code, imported).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	add("مستوردٌ ١", "D-1", true)
	add("مستوردٌ ٢", "D-2", true)
	manualLinked := add("يدويٌّ مربوط", "D-3", false)
	manual := add("يدويّ", "", false)

	info := f.call(f.srv.handleMerchantPriceSyncInfo, http.MethodGet, "/x", f.merchantID, owner, []string{"merchant"}, "")
	if info.Code != http.StatusOK || !strings.Contains(info.Body.String(), `"imported_count":2`) {
		t.Fatalf("العددُ قبل الحذف: %d %s", info.Code, info.Body.String())
	}

	stranger := testdb.NewUser(t, f.pool, "merchant")
	if w := f.call(f.srv.handleMerchantPriceSyncDeleteImported, http.MethodPost, "/x", f.merchantID, stranger, []string{"merchant"}, ""); w.Code != http.StatusForbidden {
		t.Fatalf("غريبٌ حذف: %d %s", w.Code, w.Body.String())
	}

	w := f.call(f.srv.handleMerchantPriceSyncDeleteImported, http.MethodPost, "/x", f.merchantID, owner, []string{"merchant"}, "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"deleted":2`) {
		t.Fatalf("الحذف: %d %s", w.Code, w.Body.String())
	}
	var left []string
	rows, err := f.pool.Query(ctx, `SELECT id::text FROM menu_items WHERE platform_section_id = $1 ORDER BY name`, secID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		left = append(left, id)
	}
	rows.Close()
	if len(left) != 2 || !(slices.Contains(left, manualLinked) && slices.Contains(left, manual)) {
		t.Fatalf("**حُذف صنفٌ يدويّ** أو بقي مستورد: %v", left)
	}
}
