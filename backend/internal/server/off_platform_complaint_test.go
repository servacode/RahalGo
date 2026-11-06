package server

// ══════════════════════════════════════════════════════════════════════
// **«السائق طلب رقمي أو عرض يوصّلني برّا التطبيق»** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٣)
// ══════════════════════════════════════════════════════════════════════
//
// **سببُ شكوى جديد** — على كلّ طلبٍ مغلقٍ كان له سائق. **يُنبّه الإدارةَ فوراً
// بعنوانه هو**، **ولا يُخبَر السائقُ** (يعرف رقمَ الطلب فيعرف من بلّغ)، **ولا يظهر في
// شاشة سمعته.** **وتطبيقُ الزبون القديم لا يراه**: يترجم المجهولَ «أخرى» فيفتح بلاغاً
// على سائقٍ بالغلط.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// callWithHeaders **كـ`call` وفوقه ترويساتُ التطبيق** — نوعُه ونسختُه.
func (f *driverFixture) callWithHeaders(h http.HandlerFunc, method, orderID, userID string,
	roles []string, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/x", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rc := chi.NewRouteContext()
	if orderID != "" {
		rc.URLParams.Add("id", orderID)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxRoles, roles)
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

func reasonCodes(t *testing.T, w *httptest.ResponseRecorder) map[string]bool {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("الأسباب: %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			Reasons []struct {
				Code string `json:"code"`
			} `json:"reasons"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Data.Reasons) == 0 {
		t.Fatalf("لا أسباب في الرد: %s", w.Body.String())
	}
	got := map[string]bool{}
	for _, r := range out.Data.Reasons {
		got[r.Code] = true
	}
	return got
}

func TestOffPlatformComplaint_ReasonListAndOpen(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	f.srv.support = support.NewService(f.pool, f.srv.identity, f.srv.wallet)
	f.srv.notify = notifications.New(f.pool, f.srv.hub, f.srv.logger)
	ctx := context.Background()
	ops := testdb.NewUser(t, f.pool, "operations")

	orderID := f.problemOrderAt(t, "on_the_way", d)
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'delivered', delivered_at = now(), closed_at = now() WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	var customer string
	if err := f.pool.QueryRow(ctx, `SELECT customer_id::text FROM orders WHERE id = $1`, orderID).Scan(&customer); err != nil {
		t.Fatal(err)
	}
	reasons := func(id, uid string, headers map[string]string) map[string]bool {
		t.Helper()
		return reasonCodes(t, f.callWithHeaders(f.srv.handleComplaintReasons, http.MethodGet, id, uid,
			[]string{"customer"}, "", headers))
	}

	// (١) الأسبابُ — للويب (بلا ترويسة) وللتطبيق الجديد، لا للقديم.
	if !reasons(orderID, customer, nil)["off_platform"] {
		t.Errorf("**الويبُ لا يرى السببَ الجديد**")
	}
	old := map[string]string{"X-RahalGo-Client": "android-customer", "X-RahalGo-Version": "35"}
	if reasons(orderID, customer, old)["off_platform"] {
		t.Errorf("**تطبيقُ الزبون القديم (35) يرى السببَ الجديد** — ويقرؤه «أخرى»")
	}
	fresh := map[string]string{"X-RahalGo-Client": "android-customer", "X-RahalGo-Version": "36"}
	if !reasons(orderID, customer, fresh)["off_platform"] {
		t.Errorf("**تطبيقُ الزبون الجديد (36) لا يرى السببَ الجديد**")
	}
	// **وطلبٌ لم يكن له سائقٌ قطّ لا يُشكى فيه سائق.**
	noDriver := f.dispatchingOrder(t, 10_000, 2_000)
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'cancelled', closed_at = now() WHERE id = $1`, noDriver); err != nil {
		t.Fatal(err)
	}
	var noDriverCustomer string
	if err := f.pool.QueryRow(ctx, `SELECT customer_id::text FROM orders WHERE id = $1`, noDriver).Scan(&noDriverCustomer); err != nil {
		t.Fatal(err)
	}
	if reasons(noDriver, noDriverCustomer, nil)["off_platform"] {
		t.Errorf("**طلبٌ بلا سائقٍ يعرض «السائق طلب رقمي»**")
	}
	if w := f.call(f.srv.handleOpenComplaint, http.MethodPost, "/x", noDriver, noDriverCustomer, []string{"customer"},
		`{"reason":"off_platform"}`); w.Code < 400 {
		t.Errorf("**بلاغٌ على طلبٍ بلا سائق قُبل**: %d %s", w.Code, w.Body.String())
	}

	// (٢) فتحُ البلاغ.
	w := f.call(f.srv.handleOpenComplaint, http.MethodPost, "/x", orderID, customer, []string{"customer"},
		`{"reason":"off_platform","note":"طلب رقمي ليوصلني لحالو"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("البلاغ: %d %s", w.Code, w.Body.String())
	}
	var ticketID, against string
	if err := f.pool.QueryRow(ctx, `SELECT id::text, COALESCE(against_user_id::text, '') FROM tickets
		WHERE order_id = $1 AND reason = 'off_platform'`, orderID).Scan(&ticketID, &against); err != nil {
		t.Fatal(err)
	}
	if against != d {
		t.Errorf("**البلاغُ ليس على السائق**: %q", against)
	}
	var opsTitles int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND title = $2`,
		ops, notifTitles.offPlatformComplaint).Scan(&opsTitles); err != nil {
		t.Fatal(err)
	}
	if opsTitles != 1 {
		t.Errorf("**الإدارةُ لم تُنبَّه بعنوان البلاغ**: %d", opsTitles)
	}
	var generic int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND title = $2 AND entity_id = $3`,
		ops, notifTitles.ticketNewOps, ticketID).Scan(&generic); err != nil {
		t.Fatal(err)
	}
	if generic != 0 {
		t.Errorf("**وصل الإدارةَ «شكوى جديدة» العامّ أيضاً** — عنوانان لبلاغٍ واحد")
	}
	var toDriver int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND entity_id = $2`,
		d, ticketID).Scan(&toDriver); err != nil {
		t.Fatal(err)
	}
	if toDriver != 0 {
		t.Errorf("**السائقُ أُخبر بالبلاغ عليه** — فيعرف من بلّغ")
	}

	// (٣) ولا يظهر في سمعة السائق.
	rep := f.call(f.srv.handleMeReputation, http.MethodGet, "/me/reputation", "", d, []string{"driver"}, "")
	if rep.Code != http.StatusOK {
		t.Fatalf("السمعة: %d %s", rep.Code, rep.Body.String())
	}
	var out struct {
		Data struct {
			Complaints []map[string]any `json:"complaints"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rep.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Data.Complaints) != 0 {
		t.Errorf("**البلاغُ في شاشة سمعة السائق**: %v", out.Data.Complaints)
	}
}

// TestOffPlatformComplaint_AgainstLastDriverAfterRelease **والسائقُ الذي سلّم الطلبَ لغيره
// أو تركه يبقى المشكوَّ عليه** — من سجلّ الطلب، لا من صفّه وحده.
func TestOffPlatformComplaint_AgainstLastDriverAfterRelease(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	f.srv.support = support.NewService(f.pool, f.srv.identity, f.srv.wallet)
	ctx := context.Background()
	orderID := f.dispatchingOrder(t, 10_000, 2_000)
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, driver_id)
		VALUES ($1, 'dispatching', 'assigned', $2)`, orderID, d); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'cancelled', closed_at = now(), driver_id = NULL WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	var customer string
	if err := f.pool.QueryRow(ctx, `SELECT customer_id::text FROM orders WHERE id = $1`, orderID).Scan(&customer); err != nil {
		t.Fatal(err)
	}
	tk, err := f.srv.support.Complaint(ctx, customer, orderID, "off_platform", "")
	if err != nil {
		t.Fatalf("البلاغ: %v", err)
	}
	if tk.AgainstUserID == nil || *tk.AgainstUserID != d {
		t.Errorf("**البلاغُ ليس على آخر سائقٍ حمل الطلب**: %v", tk.AgainstUserID)
	}
}
