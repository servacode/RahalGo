package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func ameenFlag(t *testing.T, f *driverFixture) bool {
	t.Helper()
	var on bool
	if err := f.pool.QueryRow(context.Background(), `SELECT uses_ameen FROM merchants WHERE id = $1`, f.merchantID).Scan(&on); err != nil {
		t.Fatal(err)
	}
	return on
}

// TestAmeenFlag_DefaultOffAndRefusals: new store is off; every price-sync route refuses with ameen_off.
func TestAmeenFlag_DefaultOffAndRefusals(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET owner_user_id = $2, uses_ameen = false,
		price_sync_token_hash = $3 WHERE id = $1`, f.merchantID, owner, syncTokenHash("rgs_off")); err != nil {
		t.Fatal(err)
	}
	if ameenFlag(t, f) {
		t.Fatal("expected off")
	}
	for name, h := range map[string]http.HandlerFunc{
		"info":      f.srv.handleMerchantPriceSyncInfo,
		"file":      f.srv.handleMerchantPriceSyncFile,
		"token":     f.srv.handleMerchantPriceSyncToken,
		"request":   f.srv.handleMerchantPriceSyncRequest,
		"interval":  f.srv.handleMerchantPriceSyncInterval,
		"download":  f.srv.handleMerchantPriceSyncAgentDownload,
		"stop":      f.srv.handleMerchantPriceSyncStop,
		"groups":    f.srv.handleMerchantPriceSyncGroups,
		"setGroups": f.srv.handleMerchantPriceSyncSetGroups,
		"autoSort":  f.srv.handleMerchantPriceSyncAutoSort,
		"pending":   f.srv.handleMerchantPriceSyncPending,
		"move":      f.srv.handleMerchantPriceSyncMove,
		"moveRest":  f.srv.handleMerchantPriceSyncMoveRest,
	} {
		w := f.call(h, http.MethodPost, "/x", f.merchantID, owner, []string{"merchant"}, `{}`)
		if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "ameen_off") {
			t.Fatalf("%s: %d %s", name, w.Code, w.Body.String())
		}
	}
	// agent with a valid token is refused too
	req := httptest.NewRequest(http.MethodPost, "/public/price-sync", strings.NewReader(`{"rows":[]}`))
	req.Header.Set("X-Sync-Token", "rgs_off")
	w := httptest.NewRecorder()
	f.srv.handlePriceSyncAgent(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("agent: %d", w.Code)
	}
}

// TestAmeenFlag_OwnerAndAdminToggle: owner and admin can toggle; another owner cannot.
func TestAmeenFlag_OwnerAndAdminToggle(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	other := testdb.NewUser(t, f.pool, "merchant")
	admin := testdb.NewUser(t, f.pool, "admin")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET owner_user_id = $2, uses_ameen = false WHERE id = $1`, f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	path := "/merchant/stores/" + f.merchantID + "/settings"
	w := f.call(f.srv.handleMerchantSettings, http.MethodPatch, path, f.merchantID, other, []string{"merchant"}, `{"uses_ameen":true}`)
	if w.Code == http.StatusOK || ameenFlag(t, f) {
		t.Fatalf("other owner toggled: %d", w.Code)
	}
	w = f.call(f.srv.handleMerchantSettings, http.MethodPatch, path, f.merchantID, owner, []string{"merchant"}, `{"uses_ameen":true}`)
	if w.Code != http.StatusOK || !ameenFlag(t, f) {
		t.Fatalf("owner on: %d %s", w.Code, w.Body.String())
	}
	// payload exposes it
	mf := &merchantFixture{pool: f.pool, srv: f.srv}
	st := mf.stores(t, owner)
	if len(st) == 0 || st[0]["uses_ameen"] != true {
		t.Fatalf("payload: %v", st)
	}
	// saving another field keeps the flag
	f.call(f.srv.handleMerchantSettings, http.MethodPatch, path, f.merchantID, owner, []string{"merchant"}, `{"default_prep_minutes":20}`)
	if !ameenFlag(t, f) {
		t.Fatal("flag lost on unrelated save")
	}
	// now routes pass the gate
	w = f.call(f.srv.handleMerchantPriceSyncInfo, http.MethodGet, "/x", f.merchantID, owner, []string{"merchant"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("info when on: %d %s", w.Code, w.Body.String())
	}
	w = f.call(f.srv.handleMerchantSettings, http.MethodPatch, path, f.merchantID, owner, []string{"merchant"}, `{"uses_ameen":false}`)
	if w.Code != http.StatusOK || ameenFlag(t, f) {
		t.Fatal("owner off")
	}
	w = f.call(f.srv.handleUpdateMerchant, http.MethodPatch, "/admin/merchants/"+f.merchantID, f.merchantID, admin, []string{"admin"}, `{"uses_ameen":true}`)
	if w.Code != http.StatusOK || !ameenFlag(t, f) {
		t.Fatalf("admin on: %d %s", w.Code, w.Body.String())
	}
}

// TestAmeenFlag_MigrationBackfill: stores with Ameen codes / sync usage become true; others stay false.
func TestAmeenFlag_MigrationBackfill(t *testing.T) {
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	var secID, itemID string
	if err := f.pool.QueryRow(ctx, `INSERT INTO platform_sections (name, sort_order) VALUES ('ameen flag sec', 951) RETURNING id`).Scan(&secID); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, platform_section_id, name, price, merchant_price, available, external_code)
		VALUES ($1, $2, 'x', 100, 100, true, 'Z-9') RETURNING id`, f.merchantID, secID).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM menu_items WHERE id = $1`, itemID)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM platform_sections WHERE id = $1`, secID)
	})
	var plain string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, phone, address_text)
		SELECT 'plain ameen flag', category_id, phone, address_text FROM merchants WHERE id = $1 RETURNING id`, f.merchantID).Scan(&plain); err != nil {
		t.Skipf("cannot clone store: %v", err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, plain) })
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET uses_ameen = false WHERE id IN ($1, $2)`, f.merchantID, plain); err != nil {
		t.Fatal(err)
	}
	sql, err := os.ReadFile(filepath.Join("..", "migrate", "migrations", "0514_merchant_uses_ameen.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, string(sql)); err != nil {
		t.Fatal(err)
	}
	var withCode, without bool
	_ = f.pool.QueryRow(ctx, `SELECT uses_ameen FROM merchants WHERE id = $1`, f.merchantID).Scan(&withCode)
	_ = f.pool.QueryRow(ctx, `SELECT uses_ameen FROM merchants WHERE id = $1`, plain).Scan(&without)
	if !withCode || without {
		t.Fatalf("backfill: withCode=%v plain=%v", withCode, without)
	}
}
