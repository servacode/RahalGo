package server

// قسمُ طلبات السحب — قراراتُ المالك ٢٠٢٦-١٠-٠٤.
//
//	١ · السحبُ المرتجعُ يُقيَّد بنوعه «إرجاع سحب» (`payout_reversal`) لا «استرجاع»
//	٢ · الرفضُ والفشلُ والارتدادُ بسببٍ إجباريّ
//	٤ · لا سحبَ يدويّاً من نافذة المحفظة — بالخادم لا بالشاشة وحدَها
//	٥ · لا أحدَ يوافق على سحبه هو
//	٦ · الصرفُ نقداً من المكتب سطرُ خروجٍ في صندوق المكتب
//	ومعها: عنوانُ إشعارٍ لكلّ حال · مبلغٌ تغيّر بعد فتح النافذة يُردّ ·
//	والبطاقاتُ الأربع والدورُ في القائمة.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/fininv"
)

// newPayout طلبُ سحبٍ معلَّقٌ عبر المسار الحقيقيّ — يحجز مالَه.
func (f *payoutFixture) newPayout(t *testing.T, role string, amount int64) (uid, id string) {
	t.Helper()
	uid = f.user(t, role, 4*amount)
	w := f.request(uid, role, amount)
	if w.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الطلب ردّ %d — %s", w.Code, w.Body.String())
	}
	if err := f.pool.QueryRow(context.Background(),
		`SELECT id::text FROM payout_requests WHERE user_id = $1 AND status = 'pending'`,
		uid).Scan(&id); err != nil {
		t.Fatalf("قراءةُ الطلب: %v", err)
	}
	return uid, id
}

// decide قرارٌ على طلب — بجسمٍ كما ترسله الشاشة.
func (f *payoutFixture) decide(actor string, roles []string, id string, body map[string]any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/admin/payouts/"+id+"/decide", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", id)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, actor)
	ctx = context.WithValue(ctx, ctxRoles, roles)
	w := httptest.NewRecorder()
	f.srv.handleDecidePayout(w, req.WithContext(ctx))
	return w
}

var financeRoles = []string{"finance"}

func (f *payoutFixture) financeUser(t *testing.T) string {
	t.Helper()
	return f.user(t, "finance", 0)
}

// TestPAYOUTS_ReversalHasItsOwnKind **البند ١**: الارتدادُ «إرجاع سحب» لا «استرجاع».
func TestPAYOUTS_ReversalHasItsOwnKind(t *testing.T) {
	f := newPayoutFixture(t)
	fin := f.financeUser(t)
	uid, id := f.newPayout(t, "driver", f.min)

	if w := f.decide(fin, financeRoles, id, map[string]any{"status": "paid", "method": "transfer"}); w.Code != 200 {
		t.Fatalf("الصرف ردّ %d — %s", w.Code, w.Body.String())
	}
	if w := f.decide(fin, financeRoles, id, map[string]any{"status": "reversed", "decision": "رجعت الحوالة"}); w.Code != 200 {
		t.Fatalf("الارتداد ردّ %d — %s", w.Code, w.Body.String())
	}
	var rev, refund int64
	if err := f.pool.QueryRow(context.Background(), `
		SELECT COALESCE(sum(amount) FILTER (WHERE kind = 'payout_reversal'), 0),
		       count(*) FILTER (WHERE kind = 'refund')
		  FROM wallet_transactions WHERE ref = $1 AND user_id = $2`, id, uid).Scan(&rev, &refund); err != nil {
		t.Fatalf("قراءةُ الدفتر: %v", err)
	}
	if rev != f.min || refund != 0 {
		t.Fatalf("إرجاعُ السحب %d (يُنتظر %d) واسترجاعٌ %d (يُنتظر 0) — "+
			"**والسحبُ المرتجعُ لا يُكتب «استرجاع»**", rev, f.min, refund)
	}
	// **وفحوصُ الدفتر تعرفه** — عقدُ النوع قائمٌ ويشير إلى ثوابتَ موجودة.
	c, ok := fininv.Kinds["payout_reversal"]
	if !ok || c.RefTarget != "payout_requests" {
		t.Fatalf("نوعُ «إرجاع سحب» بلا عقدٍ في فحوص الدفتر")
	}
}

// TestPAYOUTS_ReasonRequired **البند ٢**: الرفضُ والفشلُ والارتدادُ بسبب.
func TestPAYOUTS_ReasonRequired(t *testing.T) {
	f := newPayoutFixture(t)
	fin := f.financeUser(t)
	_, id := f.newPayout(t, "merchant", f.min)
	for _, st := range []string{"rejected", "failed"} {
		if w := f.decide(fin, financeRoles, id, map[string]any{"status": st, "decision": "  "}); w.Code != http.StatusBadRequest {
			t.Fatalf("%s بلا سبب ردّ %d — يُنتظر 400", st, w.Code)
		}
	}
	if w := f.decide(fin, financeRoles, id, map[string]any{"status": "paid", "method": "transfer"}); w.Code != 200 {
		t.Fatalf("الصرف ردّ %d — %s", w.Code, w.Body.String())
	}
	if w := f.decide(fin, financeRoles, id, map[string]any{"status": "reversed"}); w.Code != http.StatusBadRequest {
		t.Fatalf("الارتدادُ بلا سبب ردّ %d — يُنتظر 400", w.Code)
	}
}

// TestPAYOUTS_NoSelfApproval **البند ٥**: لا أحدَ يقرّر في سحبه هو.
func TestPAYOUTS_NoSelfApproval(t *testing.T) {
	f := newPayoutFixture(t)
	uid, id := f.newPayout(t, "driver", f.min)
	// **حسابٌ بدورين**: سائقٌ ومعه دورُ الماليّة.
	if _, err := f.pool.Exec(context.Background(),
		`INSERT INTO user_roles (user_id, role_code) VALUES ($1, 'finance')`, uid); err != nil {
		t.Fatalf("دورٌ ثانٍ: %v", err)
	}
	w := f.decide(uid, []string{"driver", "finance"}, id, map[string]any{"status": "paid", "method": "transfer"})
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "self_approve") {
		t.Fatalf("صرفَ لنفسه بردّ %d — %s", w.Code, w.Body.String())
	}
	var st string
	_ = f.pool.QueryRow(context.Background(), `SELECT status FROM payout_requests WHERE id = $1`, id).Scan(&st)
	if st != "pending" {
		t.Fatalf("الحالُ %q بعد الردّ — يُنتظر pending", st)
	}
}

// TestPAYOUTS_CashLeavesOfficeCashbox **البند ٦**: النقدُ من المكتب يخرج من صندوقه.
func TestPAYOUTS_CashLeavesOfficeCashbox(t *testing.T) {
	f := newPayoutFixture(t)
	fin := f.financeUser(t)
	_, cashID := f.newPayout(t, "driver", f.min)
	_, bankID := f.newPayout(t, "merchant", f.min)

	if w := f.decide(fin, financeRoles, cashID, map[string]any{"status": "paid", "method": "cash"}); w.Code != 200 {
		t.Fatalf("الصرف نقداً ردّ %d — %s", w.Code, w.Body.String())
	}
	if w := f.decide(fin, financeRoles, bankID, map[string]any{"status": "paid", "method": "transfer"}); w.Code != 200 {
		t.Fatalf("الصرف حوالةً ردّ %d — %s", w.Code, w.Body.String())
	}
	count := func(ref string) (n int, sum int64) {
		_ = f.pool.QueryRow(context.Background(), `
			SELECT count(*), COALESCE(sum(amount), 0) FROM office_cash_entries
			 WHERE direction = 'out' AND ref = $1`, ref).Scan(&n, &sum)
		return
	}
	if n, sum := count(cashID); n != 1 || sum != f.min {
		t.Fatalf("الصرفُ نقداً: %d سطراً بمجموع %d — يُنتظر سطرٌ بـ%d", n, sum, f.min)
	}
	if n, _ := count(bankID); n != 0 {
		t.Fatalf("الحوالةُ كتبت %d سطراً في صندوق المكتب — يُنتظر صفر", n)
	}
	var via string
	_ = f.pool.QueryRow(context.Background(), `SELECT paid_via FROM payout_requests WHERE id = $1`, cashID).Scan(&via)
	if via != "cash" {
		t.Fatalf("طريقةُ الصرف %q — يُنتظر cash", via)
	}
	if w := f.decide(fin, financeRoles, cashID, map[string]any{"status": "paid", "method": "cheque"}); w.Code < 400 {
		t.Fatalf("طريقةٌ مجهولةٌ قُبلت بردّ %d", w.Code)
	}
}

// TestPAYOUTS_AmountChangedIsRefused **تأكيدُ كلمة السرّ مربوطٌ بالمبلغ** —
// والمبلغُ المرسَلُ يُطابَق بالمحفوظ.
func TestPAYOUTS_AmountChangedIsRefused(t *testing.T) {
	f := newPayoutFixture(t)
	fin := f.financeUser(t)
	_, id := f.newPayout(t, "driver", f.min)
	if w := f.decide(fin, financeRoles, id, map[string]any{"status": "paid", "method": "transfer", "amount": f.min + 1}); w.Code != http.StatusConflict {
		t.Fatalf("مبلغٌ مختلفٌ ردّ %d — يُنتظر 409", w.Code)
	}
	if w := f.decide(fin, financeRoles, id, map[string]any{"status": "paid", "method": "transfer", "amount": f.min}); w.Code != 200 {
		t.Fatalf("المبلغُ نفسُه ردّ %d — %s", w.Code, w.Body.String())
	}
	act, ok := authz.LookupSensitive("POST", "/payouts/{}/decide")
	if !ok {
		t.Fatal("قرارُ السحب خرج من معجم التأكيد")
	}
	has := map[string]bool{}
	for _, p := range act.Params {
		has[p] = true
	}
	if !has["status"] || !has["amount"] || has["approve"] {
		t.Fatalf("حقولُ البصمة %v — يُنتظر status وamount (وapprove حقلٌ لا يُرسَل)", act.Params)
	}
}

// TestPAYOUTS_NotificationTitlePerState **لكلّ حالٍ عنوانُها** — لا «صُرف» على الفشل.
func TestPAYOUTS_NotificationTitlePerState(t *testing.T) {
	f := newPayoutFixture(t)
	fin := f.financeUser(t)
	title := func(id string) string {
		var s string
		_ = f.pool.QueryRow(context.Background(), `
			SELECT title FROM notifications WHERE entity = 'payout' AND entity_id = $1
			 ORDER BY created_at DESC, id DESC LIMIT 1`, id).Scan(&s)
		return s
	}
	_, a := f.newPayout(t, "driver", f.min)
	f.decide(fin, financeRoles, a, map[string]any{"status": "processing"})
	if got := title(a); got != notifTitles.payoutProcessing {
		t.Fatalf("قيد الصرف بعنوان %q", got)
	}
	f.decide(fin, financeRoles, a, map[string]any{"status": "failed", "decision": "الحساب مغلق"})
	if got := title(a); got != notifTitles.payoutFailed {
		t.Fatalf("الفشل بعنوان %q", got)
	}
	_, b := f.newPayout(t, "merchant", f.min)
	f.decide(fin, financeRoles, b, map[string]any{"status": "paid", "method": "transfer"})
	f.decide(fin, financeRoles, b, map[string]any{"status": "reversed", "decision": "رجعت"})
	if got := title(b); got != notifTitles.payoutReversed {
		t.Fatalf("الارتدادُ بعنوان %q", got)
	}
	seen := map[string]bool{}
	for _, s := range []string{notifTitles.payoutPaid, notifTitles.payoutRejected,
		notifTitles.payoutProcessing, notifTitles.payoutFailed, notifTitles.payoutReversed} {
		if s == "" || seen[s] {
			t.Fatalf("عنوانٌ فارغٌ أو مكرَّر: %q", s)
		}
		seen[s] = true
	}
}

// TestPAYOUTS_NoManualPayoutKind **البند ٤**: لا سحبَ يدويّاً من نافذة المحفظة —
// الخادمُ يردّه ولا يكفي غيابُه من الشاشة.
func TestPAYOUTS_NoManualPayoutKind(t *testing.T) {
	f := newPayoutFixture(t)
	fin := f.financeUser(t)
	target := f.user(t, "driver", 4*f.min)
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+target+"/wallet",
		strings.NewReader(`{"amount":1000,"kind":"payout","note":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", target)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, fin)
	ctx = context.WithValue(ctx, ctxRoles, financeRoles)
	w := httptest.NewRecorder()
	f.srv.handleAdminWalletApply(w, req.WithContext(ctx))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "wallet_payout_not_here") {
		t.Fatalf("سحبٌ يدويٌّ ردّ %d — %s", w.Code, w.Body.String())
	}
}

// TestPAYOUTS_ListCardsAndRole **البطاقاتُ الأربع والدورُ في القائمة.**
//
// **ولا يُمحى شيءٌ من القاعدة** — طلبٌ مصروفٌ يُمحى يترك قيدَ سحبٍ بلا طلبٍ
// فيُسقط فحصَ الدفتر FI-01.e. فيُقاس الفرقُ قبل الطلبات وبعدها.
func TestPAYOUTS_ListCardsAndRole(t *testing.T) {
	f := newPayoutFixture(t)
	fin := f.financeUser(t)

	type stats struct {
		WaitingCount    int   `json:"waiting_count"`
		WaitingSum      int64 `json:"waiting_sum"`
		ProcessingCount int   `json:"processing_count"`
		ProcessingSum   int64 `json:"processing_sum"`
		PaidMonthCount  int   `json:"paid_month_count"`
		PaidMonthSum    int64 `json:"paid_month_sum"`
		ReturnedCount   int   `json:"returned_month_count"`
		ReturnedSum     int64 `json:"returned_month_sum"`
	}
	type row struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		UserRole      string `json:"user_role"`
		DecidedByName string `json:"decided_by_name"`
	}
	type res struct {
		Data struct {
			Total        int   `json:"total"`
			PendingTotal int64 `json:"pending_total"`
			Stats        stats `json:"stats"`
			Payouts      []row `json:"payouts"`
		} `json:"data"`
	}
	get := func(q string) res {
		req := httptest.NewRequest(http.MethodGet, "/x?"+q, nil)
		c := context.WithValue(req.Context(), ctxRoles, []string{"admin"})
		w := httptest.NewRecorder()
		f.srv.handleAdminPayouts(w, req.WithContext(c))
		if w.Code != 200 {
			t.Fatalf("ردّ %d — %s", w.Code, w.Body.String())
		}
		var out res
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	before := get("page=1")
	beforeDrivers := get("page=1&role=driver").Data.Total
	beforeProc := get("page=1&status=processing").Data.Total

	_, _ = f.newPayout(t, "driver", f.min)     // يبقى معلَّقاً
	_, p2 := f.newPayout(t, "merchant", f.min) // قيد الصرف
	_, p3 := f.newPayout(t, "sales", f.min)    // مصروف
	_, p4 := f.newPayout(t, "driver", f.min)   // فشل
	f.decide(fin, financeRoles, p2, map[string]any{"status": "processing"})
	f.decide(fin, financeRoles, p3, map[string]any{"status": "paid", "method": "transfer"})
	f.decide(fin, financeRoles, p4, map[string]any{"status": "failed", "decision": "خطأ"})

	after := get("page=1")
	b, a := before.Data.Stats, after.Data.Stats
	if a.WaitingCount-b.WaitingCount != 2 || a.WaitingSum-b.WaitingSum != 2*f.min ||
		after.Data.PendingTotal-before.Data.PendingTotal != 2*f.min {
		t.Fatalf("بانتظار الصرف زاد %d/%d — يُنتظر 2/%d **ويشمل قيد الصرف**",
			a.WaitingCount-b.WaitingCount, a.WaitingSum-b.WaitingSum, 2*f.min)
	}
	if a.ProcessingCount-b.ProcessingCount != 1 || a.ProcessingSum-b.ProcessingSum != f.min {
		t.Fatalf("قيد الصرف زاد %d", a.ProcessingCount-b.ProcessingCount)
	}
	if a.PaidMonthCount-b.PaidMonthCount != 1 || a.PaidMonthSum-b.PaidMonthSum != f.min {
		t.Fatalf("مصروف هالشهر زاد %d", a.PaidMonthCount-b.PaidMonthCount)
	}
	if a.ReturnedCount-b.ReturnedCount != 1 || a.ReturnedSum-b.ReturnedSum != f.min {
		t.Fatalf("مرتجع/فاشل هالشهر زاد %d", a.ReturnedCount-b.ReturnedCount)
	}
	drivers := get("page=1&role=driver&per_page=100")
	if drivers.Data.Total-beforeDrivers != 2 {
		t.Fatalf("فلترُ السائق زاد %d — يُنتظر 2", drivers.Data.Total-beforeDrivers)
	}
	for _, r := range drivers.Data.Payouts {
		if r.UserRole != "driver" {
			t.Fatalf("دورٌ %q في فلتر السائق", r.UserRole)
		}
		if r.ID == p4 && r.DecidedByName == "" {
			t.Fatal("مين قرّر غائبٌ عن طلبٍ مبتوت")
		}
	}
	if got := get("page=1&role=customer"); got.Data.Total != 0 {
		t.Fatalf("دورٌ لا يسحب ردّ %d", got.Data.Total)
	}
	if got := get("page=1&status=processing"); got.Data.Total-beforeProc != 1 {
		t.Fatalf("فلترُ قيد الصرف زاد %d", got.Data.Total-beforeProc)
	}
}
