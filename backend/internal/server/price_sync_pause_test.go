package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestPriceSyncPause_KeepsProgramWritesNothing **«إيقاف» صار مؤقّتاً** (قرارُ المالك ٢٠٢٦-١١-٠٧:
// «إذا أعطى إيقاف ما في داعي لإعادة التحميل — بينزل مرّة وحدة بس»).
//
// موقوف ⇒ السؤالُ يقول "paused":true ولا يدفع إلى قراءة، **وما يرفعه برنامجٌ قديم (2.0.1)
// يُقبل بـ٢٠٠ ولا يُكتب منه شيء.** وشغّل ⇒ يُطبَّق الرفع. والحذفُ من الكمبيوتر كما كان.
func TestPriceSyncPause_KeepsProgramWritesNothing(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx, `
		UPDATE merchants SET uses_ameen = true, owner_user_id = $2, price_sync_token_hash = $3,
		       price_sync_requested_at = now(), price_sync_want_schema = true
		 WHERE id = $1`, f.merchantID, owner, syncTokenHash("rgs_pause")); err != nil {
		t.Fatal(err)
	}
	var secID, itemID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO platform_sections (name, sort_order) VALUES ('قسمُ إيقافٍ للاختبار', 951) RETURNING id`).Scan(&secID); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, platform_section_id, name, price, merchant_price, available, external_code)
		VALUES ($1, $2, 'سكّر', 100, 100, true, 'P-1') RETURNING id`, f.merchantID, secID).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM menu_items WHERE id = $1`, itemID)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, secID)
	})

	agent := func(h http.HandlerFunc, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Sync-Token", "rgs_pause")
		w := httptest.NewRecorder()
		h(w, req)
		return w
	}
	poll := func() string {
		w := agent(f.srv.handlePriceSyncPollV2, "/public/price-sync/poll", `{"version":"2.0.1"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("السؤال: %d %s", w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	price := func() int64 {
		var p int64
		_ = f.pool.QueryRow(ctx, `SELECT merchant_price FROM menu_items WHERE id = $1`, itemID).Scan(&p)
		return p
	}
	runs := func() int {
		var n int
		_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM price_sync_runs WHERE merchant_id = $1`, f.merchantID).Scan(&n)
		return n
	}

	// **الحرّاس**: غيرُ صاحب المتجر لا يوقف.
	stranger := testdb.NewUser(t, f.pool, "merchant")
	if w := f.call(f.srv.handleMerchantPriceSyncPause, http.MethodPost, "/x", f.merchantID, stranger, []string{"merchant"}, ""); w.Code != http.StatusForbidden {
		t.Fatalf("غريبٌ أوقف: %d %s", w.Code, w.Body.String())
	}
	if b := poll(); !strings.Contains(b, `"paused":false`) {
		t.Fatalf("غريبٌ غيّر الحال: %s", b)
	}

	// **إيقاف مؤقّت.**
	if w := f.call(f.srv.handleMerchantPriceSyncPause, http.MethodPost, "/x", f.merchantID, owner, []string{"merchant"}, ""); w.Code != http.StatusOK {
		t.Fatalf("الإيقاف: %d %s", w.Code, w.Body.String())
	}
	b := poll()
	for _, want := range []string{`"paused":true`, `"sync_now":false`, `"want_schema":false`, `"stop":false`} {
		if !strings.Contains(b, want) {
			t.Fatalf("السؤالُ وهو موقوف بلا %s: %s", want, b)
		}
	}
	before := runs()
	if w := agent(f.srv.handlePriceSyncAgent, "/public/price-sync", `{"rows":[{"code":"P-1","price":140,"qty":5}],"currency":"SYP"}`); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"ignored":true`) {
		t.Fatalf("رفعُ 2.0.1 وهو موقوف: %d %s", w.Code, w.Body.String())
	}
	if p := price(); p != 100 {
		t.Fatalf("**رفعٌ وهو موقوف غيّر السعر**: %d", p)
	}
	if n := runs(); n != before {
		t.Fatalf("**رفعٌ وهو موقوف سُجّل تقريراً**: %d → %d", before, n)
	}
	if w := agent(f.srv.handlePriceSyncSchema, "/public/price-sync/schema", `{"tables":[]}`); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"ignored":true`) {
		t.Fatalf("الجداولُ وهو موقوف: %d %s", w.Code, w.Body.String())
	}
	var schemaSaved bool
	_ = f.pool.QueryRow(ctx, `SELECT price_sync_schema IS NOT NULL FROM merchants WHERE id = $1`, f.merchantID).Scan(&schemaSaved)
	if schemaSaved {
		t.Fatal("**جداولُ وهو موقوف حُفظت**")
	}
	if w := agent(f.srv.handlePriceSyncAgentFile, "/public/price-sync/file", ``); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"ignored":true`) {
		t.Fatalf("ملفُّ المجلّد وهو موقوف: %d %s", w.Code, w.Body.String())
	}
	if w := f.call(f.srv.handleMerchantPriceSyncInfo, http.MethodGet, "/x", f.merchantID, owner, []string{"merchant"}, ""); !strings.Contains(w.Body.String(), `"paused":true`) {
		t.Fatalf("لوحةُ المتجر لا ترى «موقوفة مؤقّتاً»: %s", w.Body.String())
	}
	var audited int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE action = 'merchant.price_sync_pause' AND entity_id = $1`, f.merchantID).Scan(&audited)
	if audited != 1 {
		t.Fatalf("الإيقافُ لم يُسجَّل في السجلّ: %d", audited)
	}

	// **شغّل المزامنة** ⇒ يُطبَّق الرفع.
	if w := f.call(f.srv.handleMerchantPriceSyncResume, http.MethodPost, "/x", f.merchantID, owner, []string{"merchant"}, ""); w.Code != http.StatusOK {
		t.Fatalf("التشغيل: %d %s", w.Code, w.Body.String())
	}
	if b := poll(); !strings.Contains(b, `"paused":false`) || !strings.Contains(b, `"sync_now":true`) {
		t.Fatalf("السؤالُ بعد التشغيل: %s", b)
	}
	if w := agent(f.srv.handlePriceSyncAgent, "/public/price-sync", `{"rows":[{"code":"P-1","price":140,"qty":5}],"currency":"SYP"}`); w.Code != http.StatusOK || strings.Contains(w.Body.String(), `"ignored"`) {
		t.Fatalf("الرفعُ بعد التشغيل: %d %s", w.Code, w.Body.String())
	}
	if p := price(); p != 140 {
		t.Fatalf("الرفعُ بعد التشغيل لم يُطبَّق: %d", p)
	}

	// **الحذفُ من الكمبيوتر كما كان** — "stop":true فيزيل نفسَه.
	f.call(f.srv.handleMerchantPriceSyncStop, http.MethodPost, "/x", f.merchantID, owner, []string{"merchant"}, "")
	if b := poll(); !strings.Contains(b, `"stop":true`) {
		t.Fatalf("الحذفُ لم يصل: %s", b)
	}
}

// TestPriceSyncPause_NeedsAmeen **متجرٌ بلا الأمين لا يوقف ولا يشغّل.**
func TestPriceSyncPause_NeedsAmeen(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET uses_ameen = false, owner_user_id = $2 WHERE id = $1`, f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	for _, h := range []http.HandlerFunc{f.srv.handleMerchantPriceSyncPause, f.srv.handleMerchantPriceSyncResume} {
		if w := f.call(h, http.MethodPost, "/x", f.merchantID, owner, []string{"merchant"}, ""); w.Code == http.StatusOK {
			t.Fatalf("متجرٌ بلا الأمين نجح: %s", w.Body.String())
		}
	}
	var paused bool
	_ = f.pool.QueryRow(ctx, `SELECT price_sync_paused FROM merchants WHERE id = $1`, f.merchantID).Scan(&paused)
	if paused {
		t.Fatal("**متجرٌ بلا الأمين أوقف**")
	}
}
