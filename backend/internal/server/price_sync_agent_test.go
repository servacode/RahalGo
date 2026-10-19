package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPriceSyncAgent_TokenPicksStore **ساعي الأمين برمزه** (قرارُ المالك ٢٠٢٦-١٠-١٩) — الرمزُ
// يدلّ على المتجر ويحدّث أصنافه، **ورمزٌ غلطٌ أو غائبٌ يُردّ ولا يمسّ شيئاً.**
func TestPriceSyncAgent_TokenPicksStore(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET price_sync_token_hash = $2 WHERE id = $1`,
		f.merchantID, syncTokenHash("rgs_test")); err != nil {
		t.Fatal(err)
	}
	var itemID string
	var secID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, sort_order) VALUES ('قسمُ مزامنةٍ للاختبار', 950) RETURNING id`).Scan(&secID); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, platform_section_id, name, price, merchant_price, available, external_code)
		VALUES ($1, $2, 'رز', 100, 100, true, 'A-1') RETURNING id`, f.merchantID, secID).Scan(&itemID); err != nil {
		t.Fatalf("صنفُ المتجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM menu_items WHERE id = $1`, itemID)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, secID)
	})
	post := func(tok string) int {
		req := httptest.NewRequest(http.MethodPost, "/public/price-sync",
			strings.NewReader(`{"rows":[{"code":"A-1","price":140,"qty":0}],"currency":"SYP"}`))
		req.Header.Set("Content-Type", "application/json")
		if tok != "" {
			req.Header.Set("X-Sync-Token", tok)
		}
		w := httptest.NewRecorder()
		f.srv.handlePriceSyncAgent(w, req)
		return w.Code
	}
	if code := post(""); code != http.StatusUnauthorized {
		t.Fatalf("بلا رمز: %d", code)
	}
	if code := post("rgs_wrong"); code != http.StatusUnauthorized {
		t.Fatalf("رمزٌ غلط: %d", code)
	}
	var price int64
	var avail bool
	_ = f.pool.QueryRow(ctx, `SELECT merchant_price, available FROM menu_items WHERE id = $1`, itemID).Scan(&price, &avail)
	if price != 100 || !avail {
		t.Fatal("**رمزٌ مرفوضٌ مسّ الصنف**")
	}
	if code := post("rgs_test"); code != http.StatusOK {
		t.Fatalf("الرمزُ الصحيح: %d", code)
	}
	_ = f.pool.QueryRow(ctx, `SELECT merchant_price, available FROM menu_items WHERE id = $1`, itemID).Scan(&price, &avail)
	if price != 140 || avail {
		t.Fatalf("بعد المزامنة: %d %v — **والمطلوب ١٤٠ غير متوفر**", price, avail)
	}
}
