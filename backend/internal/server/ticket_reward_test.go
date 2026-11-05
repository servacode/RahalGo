package server

// **مكافأةُ بلاغ «السائق طلب رقمي»: كودُ توصيلٍ مجّانيٍّ لمرّةٍ واحدة** (قرارُ المالك
// ٢٠٢٦-١١-٠٥، البند ٦) — بقدرتي الدعم والعروض، **وعلى البلاغ الصحيح وحدَه، ومرّةً واحدة.**

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func (f *driverFixture) callWithCaps(h http.HandlerFunc, id, userID string, caps []string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", id)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"admin"})
	ctx = context.WithValue(ctx, ctxCaps, caps)
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

func TestTicketReward_FreeDeliveryCodeOnce(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	f.srv.support = support.NewService(f.pool, f.srv.identity, f.srv.wallet)
	f.srv.notify = notifications.New(f.pool, f.srv.hub, f.srv.logger)
	ctx := context.Background()
	staff := testdb.NewUser(t, f.pool, "admin")

	deliveredOrder := func() (orderID, customer string) {
		orderID = f.problemOrderAt(t, "on_the_way", d)
		if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'delivered', delivered_at = now(), closed_at = now() WHERE id = $1`, orderID); err != nil {
			t.Fatal(err)
		}
		if err := f.pool.QueryRow(ctx, `SELECT customer_id::text FROM orders WHERE id = $1`, orderID).Scan(&customer); err != nil {
			t.Fatal(err)
		}
		return orderID, customer
	}
	o1, c1 := deliveredOrder()
	offTk, err := f.srv.support.Complaint(ctx, c1, o1, support.ReasonOffPlatform, "طلب رقمي")
	if err != nil {
		t.Fatalf("البلاغ: %v", err)
	}
	o2, c2 := deliveredOrder()
	lateTk, err := f.srv.support.Complaint(ctx, c2, o2, "late", "")
	if err != nil {
		t.Fatalf("شكوى التأخير: %v", err)
	}
	both := []string{"support.manage", "offers.manage"}

	// (١) بلا قدرة العروض ⇒ ٤٠٣.
	if w := f.callWithCaps(f.srv.handleTicketRewardFreeDelivery, offTk.ID, staff, []string{"support.manage"}); w.Code != http.StatusForbidden {
		t.Fatalf("**بلا offers.manage صُنع كود**: %d %s", w.Code, w.Body.String())
	}
	// (٢) شكوى بسببٍ آخر ⇒ ٤٠٩ reward_not_eligible.
	if w := f.callWithCaps(f.srv.handleTicketRewardFreeDelivery, lateTk.ID, staff, both); w.Code != http.StatusConflict ||
		!strings.Contains(w.Body.String(), "reward_not_eligible") {
		t.Fatalf("**مكافأةٌ على شكوى تأخير**: %d %s", w.Code, w.Body.String())
	}

	// (٣) البلاغُ الصحيح ⇒ كودٌ لمرّةٍ واحدة.
	w := f.callWithCaps(f.srv.handleTicketRewardFreeDelivery, offTk.ID, staff, both)
	if w.Code != http.StatusCreated {
		t.Fatalf("المكافأة: %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	code := out.Data.Code
	prefix := "RG" + strconv.FormatInt(offTk.Number, 10) + "-"
	if !strings.HasPrefix(code, prefix) {
		t.Fatalf("الكود %q لا يبدأ بـ %q", code, prefix)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM promo_codes WHERE code = $1`, code) })
	var kind, approval string
	var maxUses int
	var once, active bool
	if err := f.pool.QueryRow(ctx, `SELECT kind, max_uses, once_per_user, active, approval_state
		FROM promo_codes WHERE code = $1`, code).Scan(&kind, &maxUses, &once, &active, &approval); err != nil {
		t.Fatalf("الكود في القاعدة: %v", err)
	}
	if kind != "free_delivery" || maxUses != 1 || !once || !active {
		t.Errorf("**الكود**: kind=%s max_uses=%d once=%v active=%v approval=%s", kind, maxUses, once, active, approval)
	}
	var replies int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM ticket_replies WHERE ticket_id = $1 AND body LIKE '%' || $2 || '%'`,
		offTk.ID, code).Scan(&replies); err != nil {
		t.Fatal(err)
	}
	if replies != 1 {
		t.Errorf("**لا ردَّ على البلاغ يحمل الكود**: %d", replies)
	}
	var notified int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND body LIKE '%' || $2 || '%'`,
		c1, code).Scan(&notified); err != nil {
		t.Fatal(err)
	}
	if notified == 0 {
		t.Errorf("**الزبونُ لم يُخبَر بالكود**")
	}

	// (٤) المرّةُ الثانية ⇒ ٤٠٩ بالكود نفسِه.
	w2 := f.callWithCaps(f.srv.handleTicketRewardFreeDelivery, offTk.ID, staff, both)
	if w2.Code != http.StatusConflict || !strings.Contains(w2.Body.String(), "reward_already_granted") ||
		!strings.Contains(w2.Body.String(), code) {
		t.Fatalf("**المكافأةُ مُنحت مرّتين**: %d %s", w2.Code, w2.Body.String())
	}
	var n int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM promo_codes WHERE code LIKE $1 || '%'`, prefix).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("أكوادُ البلاغ: %d", n)
	}
}
