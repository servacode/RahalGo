package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestPriceSyncPending_NotifyOnceRemindOnce **«في N صنف من الأمين بلا قسم»** (قرار المالك ٢٠٢٦-١١-٠٧):
// مزامنة حقيقيّة من الساعي تركت أصناف بالانتظار ⇒ إشعار واحد للمتجر وللإدارة، ومزامنة تانية بلا أصناف
// جديدة ما بتكرّر؛ وبعد يومين تذكير واحد بس؛ و«حطّ الباقي بقسم منوّعات» بينقلهن ويسكّر الدورة.
func TestPriceSyncPending_NotifyOnceRemindOnce(t *testing.T) {
	f := newDriverFixture(t, 0)
	f.srv.notify = notifications.New(f.pool, f.srv.hub, f.srv.logger)
	ctx := context.Background()
	owner := testdb.NewUser(t, f.pool, "merchant")
	admin := testdb.NewUser(t, f.pool, "admin")
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET owner_user_id = $2, uses_ameen = true, price_sync_token_hash = $3,
		price_sync_pending_notified_at = NULL, price_sync_pending_reminded_at = NULL WHERE id = $1`,
		f.merchantID, owner, syncTokenHash("rgs_pending")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM menu_items WHERE merchant_id = $1`, f.merchantID)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM notifications WHERE entity = 'price_sync' AND entity_id = $1`, f.merchantID)
	})
	// أسماء ما إلها قسم مقترح ⇒ بتروح للانتظار.
	agent := func(rows string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/public/price-sync",
			strings.NewReader(`{"rows":[`+rows+`],"currency":"SYP","create_missing":true}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Sync-Token", "rgs_pending")
		w := httptest.NewRecorder()
		f.srv.handlePriceSyncAgent(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("الساعي: %d %s", w.Code, w.Body.String())
		}
	}
	count := func(user string) (n int, body string) {
		t.Helper()
		if err := f.pool.QueryRow(ctx, `
			SELECT count(*), COALESCE(max(body), '') FROM notifications
			 WHERE user_id = $1 AND entity = 'price_sync' AND entity_id = $2`, user, f.merchantID).Scan(&n, &body); err != nil {
			t.Fatal(err)
		}
		return n, body
	}

	agent(`{"code":"PN-1","name":"زقزوق ١٢٣ قرقش","price":5},{"code":"PN-2","name":"فرفور ٤٥٦ دبدوب","price":5}`)
	n, body := count(owner)
	if n != 1 || !strings.Contains(body, "2") {
		t.Fatalf("إشعار المتجر بعد المزامنة: %d %q — والمطلوب واحد فيه العدد", n, body)
	}
	if n, _ := count(admin); n != 1 {
		t.Fatalf("إشعار الإدارة: %d — والمطلوب واحد", n)
	}
	var apps []string
	var href string
	if err := f.pool.QueryRow(ctx, `SELECT push_apps, href FROM notifications WHERE user_id = $1 AND entity = 'price_sync'`, owner).
		Scan(&apps, &href); err != nil || len(apps) != 1 || apps[0] != notifications.AppMerchant || href != "/store/price-sync" {
		t.Fatalf("إشعار المتجر: %v %q %v", apps, href, err)
	}

	// الساعي بيزامن كمان مرّة بلا أصناف جديدة ⇒ ولا إشعار.
	agent(`{"code":"PN-1","name":"زقزوق ١٢٣ قرقش","price":6}`)
	if n, _ := count(owner); n != 1 {
		t.Fatalf("مزامنة بلا أصناف جديدة كرّرت الإشعار: %d", n)
	}

	// قبل يومين ما في تذكير.
	if sent, err := f.srv.RemindPriceSyncPending(ctx, time.Now().Add(47*time.Hour)); err != nil || sent != 0 {
		t.Fatalf("قبل يومين: %d %v", sent, err)
	}
	// بعد يومين تذكير واحد — وجولة تانية ما بتعيده.
	later := time.Now().Add(49 * time.Hour)
	sent, err := f.srv.RemindPriceSyncPending(ctx, later)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := f.srv.RemindPriceSyncPending(ctx, later.Add(72*time.Hour)); sent < 1 || again != 0 {
		t.Fatalf("التذكير: %d ثمّ %d — والمطلوب واحد وبعدين ولا شي", sent, again)
	}
	if n, _ := count(owner); n != 2 {
		t.Fatalf("إشعارات المتجر بعد التذكير: %d — والمطلوب ٢", n)
	}
	if n, _ := count(admin); n != 2 {
		t.Fatalf("إشعارات الإدارة بعد التذكير: %d — والمطلوب ٢", n)
	}

	// «حطّ الباقي بقسم منوّعات».
	base := "/merchant/stores/" + f.merchantID + "/price-sync"
	other := testdb.NewUser(t, f.pool, "merchant")
	if w := f.call(f.srv.handleMerchantPriceSyncMoveRest, http.MethodPost, base+"/move-rest", f.merchantID, other, []string{"merchant"}, ""); w.Code == http.StatusOK {
		t.Fatalf("متجر غيره نقل: %d", w.Code)
	}
	w := f.call(f.srv.handleMerchantPriceSyncMoveRest, http.MethodPost, base+"/move-rest", f.merchantID, owner, []string{"merchant"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("move-rest: %d %s", w.Code, w.Body.String())
	}
	var res struct {
		Data struct {
			Moved int `json:"moved"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || res.Data.Moved != 2 {
		t.Fatalf("move-rest: %s %v", w.Body.String(), err)
	}
	var stamped bool
	if err := f.pool.QueryRow(ctx, `SELECT price_sync_pending_notified_at IS NOT NULL FROM merchants WHERE id = $1`, f.merchantID).
		Scan(&stamped); err != nil || stamped {
		t.Fatalf("الدورة ما تسكّرت بعد النقل: %v %v", stamped, err)
	}

	// مزامنة جديدة تركت أصناف ⇒ دورة جديدة بإشعار جديد.
	agent(`{"code":"PN-3","name":"شمشوم ٧٨٩ بقبوق","price":5}`)
	if n, _ := count(owner); n != 3 {
		t.Fatalf("مزامنة جديدة تركت أصناف: %d — والمطلوب ٣", n)
	}
}
