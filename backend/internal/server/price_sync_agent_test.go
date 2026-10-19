package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
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

// TestPriceSyncPoll_SyncNow **الساعي يسأل فيُعرف أنّه حيّ، و«مزامنة الآن» تصله ثمّ تُمحى.**
func TestPriceSyncPoll_SyncNow(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET price_sync_token_hash = $2, price_sync_requested_at = NULL WHERE id = $1`,
		f.merchantID, syncTokenHash("rgs_poll")); err != nil {
		t.Fatal(err)
	}
	poll := func() string {
		req := httptest.NewRequest(http.MethodGet, "/public/price-sync/poll", nil)
		req.Header.Set("X-Sync-Token", "rgs_poll")
		req.Header.Set("X-Agent-Version", "1.0.0")
		w := httptest.NewRecorder()
		f.srv.handlePriceSyncPoll(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("السؤال: %d %s", w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	if b := poll(); !strings.Contains(b, `"sync_now":false`) || !strings.Contains(b, `"interval_minutes":1`) {
		t.Fatalf("بلا طلبٍ قال: %s — **والافتراضُ دقيقة**", b)
	}
	var seen *string
	_ = f.pool.QueryRow(ctx, `SELECT price_sync_agent_version FROM merchants WHERE id = $1`, f.merchantID).Scan(&seen)
	if seen == nil || *seen != "1.0.0" {
		t.Fatal("**السؤالُ لم يُسجِّل الساعي حيّاً**")
	}
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET price_sync_requested_at = now() WHERE id = $1`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	if b := poll(); !strings.Contains(b, `"sync_now":true`) {
		t.Fatalf("«مزامنة الآن» لم تصل: %s", b)
	}
	f.srv.agentSynced(ctx, f.merchantID)
	if b := poll(); !strings.Contains(b, `"sync_now":false`) {
		t.Fatalf("«مزامنة الآن» لم تُمحَ بعد المزامنة: %s", b)
	}
}

// TestValidSyncQuery **قراءةٌ لا كتابة** — كمبيوترُ المول يحمل حساباتِه.
func TestValidSyncQuery(t *testing.T) {
	ok := []string{"", "SELECT Code AS code, Price AS price FROM mt000", "with x as (select 1 as price) select * from x"}
	bad := []string{"DELETE FROM mt000", "SELECT 1; DROP TABLE mt000", "SELECT * INTO t2 FROM mt000",
		"UPDATE mt000 SET x=1", "EXEC xp_cmdshell 'dir'", "SELECT 1 -- x", "select * from openrowset('a','b','c')"}
	for _, q := range ok {
		if !validSyncQuery(q) {
			t.Fatalf("رُدّت قراءةٌ سليمة: %q", q)
		}
	}
	for _, q := range bad {
		if validSyncQuery(q) {
			t.Fatalf("**قُبلت جملةٌ تكتب**: %q", q)
		}
	}
}

// TestPriceSyncAgentV2_ConfigStatusDownload **الساعي يأخذ إعدادَه ويبعث حالَه، والتحميلُ يحمل الرمز**
// (قرارُ المالك ٢٠٢٦-١٠-١٩: «ملف واحد… ما بدنا المحاسب يتلبك»).
func TestPriceSyncAgentV2_ConfigStatusDownload(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "RahalGoSync.exe"), []byte("MZ-fake-agent"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agent-version"), []byte("2.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENT_DIR", dir)
	owner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx, `
		UPDATE merchants SET owner_user_id = $2, price_sync_sql_query = 'SELECT Code AS code, Price AS price FROM mt000',
		       price_sync_interval_minutes = 3 WHERE id = $1`, f.merchantID, owner); err != nil {
		t.Fatal(err)
	}

	// **التحميلُ: الملفُّ ثمّ العلامةُ ثمّ الرمز** — ورمزٌ جديدٌ يُحفظ.
	w := f.call(f.srv.handleMerchantPriceSyncAgentDownload, http.MethodGet, "/x", f.merchantID, owner, []string{"merchant"}, "")
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.HasPrefix(body, "MZ-fake-agent") || !strings.Contains(body, agentMarker) {
		t.Fatalf("التحميل: %d %q", w.Code, body)
	}
	var cfg struct{ Server, Token string }
	if err := json.Unmarshal([]byte(body[strings.Index(body, agentMarker)+len(agentMarker):]), &cfg); err != nil || !strings.HasPrefix(cfg.Token, "rgs_") {
		t.Fatalf("إعدادُ الذيل: %v %+v", err, cfg)
	}

	// **السؤال: يبعث حالَه ويأخذ إعدادَه.**
	req := httptest.NewRequest(http.MethodPost, "/public/price-sync/poll",
		strings.NewReader(`{"version":"2.0.0","error":"ما لقيت قاعدة الأمين","status":{"mode":"sql"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sync-Token", cfg.Token)
	rw := httptest.NewRecorder()
	f.srv.handlePriceSyncPollV2(rw, req)
	if rw.Code != http.StatusOK {
		t.Fatalf("السؤال: %d %s", rw.Code, rw.Body.String())
	}
	for _, want := range []string{`"interval_minutes":3`, `"latest_version":"2.0.0"`, `"sql_query":"SELECT Code AS code`, `"want_schema":true`, `"stop":false`} {
		if !strings.Contains(rw.Body.String(), want) {
			t.Fatalf("الإعدادُ بلا %s: %s", want, rw.Body.String())
		}
	}
	var agentErr *string
	_ = f.pool.QueryRow(ctx, `SELECT price_sync_agent_error FROM merchants WHERE id = $1`, f.merchantID).Scan(&agentErr)
	if agentErr == nil || !strings.Contains(*agentErr, "قاعدة الأمين") {
		t.Fatal("**خطأُ الساعي لم يُحفظ** — فلا يُقرأ في اللوحة")
	}

	// **«إيقاف» يصله.**
	f.call(f.srv.handleMerchantPriceSyncStop, http.MethodPost, "/x", f.merchantID, owner, []string{"merchant"}, "")
	rw = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/public/price-sync/poll", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sync-Token", cfg.Token)
	f.srv.handlePriceSyncPollV2(rw, req)
	if !strings.Contains(rw.Body.String(), `"stop":true`) {
		t.Fatalf("«إيقاف» لم يصل: %s", rw.Body.String())
	}
}
