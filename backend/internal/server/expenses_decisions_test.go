package server

// **قراراتُ المالك ٢٠٢٦-١٠-٠٤ على مصروفات التشغيل.**
//
//	١ · فوق السقف اقتراحٌ يوافق عليه شخصٌ آخر، وتحته قيدٌ مباشر.
//	٢ · الشهرُ بتاريخ الصرف (`opexBySpendDate`).
//	٤ · الإيصالُ إلزاميٌّ فوق السقف.
//	٥ · الإلغاءُ بسببٍ ومن شخصٍ غيرِ من سجّل، والملغى يبقى ظاهراً.
//	ومعها: لا تاريخَ صرفٍ في المستقبل · مدًى مقلوبٌ يُقال · قيدُ الخزينة يسمّي الباب ·
//	سجلُّ تعديل الباب كامل.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

type opexFixture struct {
	f        *driverFixture
	treasury string
	catID    string
	ids      []string
}

func newOpexFixture(t *testing.T) *opexFixture {
	t.Helper()
	f := newDriverFixture(t, 0)
	ctx := context.Background()
	x := &opexFixture{f: f}
	x.treasury = f.srv.orders.TreasuryID(ctx)
	if x.treasury == "" {
		x.treasury = testdb.NewUser(t, f.pool, "admin")
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO wallets (user_id, balance, is_treasury) VALUES ($1, 0, true)
			ON CONFLICT (user_id) DO UPDATE SET is_treasury = true`, x.treasury); err != nil {
			t.Fatalf("تعذّرت الخزينة: %v", err)
		}
		t.Cleanup(func() {
			_, _ = f.pool.Exec(context.Background(), `DELETE FROM wallets WHERE user_id = $1`, x.treasury)
		})
	}
	if err := f.pool.QueryRow(ctx,
		`SELECT id::text FROM expense_categories WHERE name = 'إيجار'`).Scan(&x.catID); err != nil {
		t.Fatalf("لا بابَ «إيجار»: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		// **والحركةُ تُحذف ورصيدُها يُردّ** — فلا يسقط `FI-02.a` في moneycheck.
		unpost := `
			WITH d AS (DELETE FROM wallet_transactions WHERE ref = $1 RETURNING user_id, amount)
			UPDATE wallets w SET balance = w.balance - s.total
			FROM (SELECT user_id, sum(amount) AS total FROM d GROUP BY user_id) s
			WHERE w.user_id = s.user_id`
		for _, id := range x.ids {
			_, _ = f.pool.Exec(c, unpost, id)
			_, _ = f.pool.Exec(c, `DELETE FROM expense_requests WHERE id::text = $1 OR expense_id::text = $1`, id)
		}
		for _, id := range x.ids {
			_, _ = f.pool.Exec(c, `DELETE FROM expenses WHERE id::text = $1`, id)
		}
		_, _ = f.pool.Exec(c, `DELETE FROM media WHERE kind = 'expense_receipt' AND path LIKE 'test/%'`)
	})
	return x
}

func (x *opexFixture) call(t *testing.T, h http.HandlerFunc, actor, method, target, body, id string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rc := chi.NewRouteContext()
	if id != "" {
		rc.URLParams.Add("id", id)
	}
	c := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	c = context.WithValue(c, ctxUserID, actor)
	c = context.WithValue(c, ctxRoles, []string{"admin"})
	w := httptest.NewRecorder()
	h(w, req.WithContext(c))
	return w
}

func (x *opexFixture) balance(t *testing.T) int64 {
	t.Helper()
	b, err := x.f.srv.wallet.Balance(context.Background(), x.treasury)
	if err != nil {
		t.Fatalf("الرصيد: %v", err)
	}
	return b
}

func expData(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("ردٌّ لا يُفكّ: %v — %s", err, w.Body.String())
	}
	return env.Data
}

func expErrCode(w *httptest.ResponseRecorder) string {
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	return env.Error.Code
}

func (x *opexFixture) receipt(t *testing.T, by string) string {
	t.Helper()
	var id string
	if err := x.f.pool.QueryRow(context.Background(), `
		INSERT INTO media (kind, path, thumb_path, width, height, bytes, created_by)
		VALUES ('expense_receipt', 'test/r.jpg', 'test/r_t.jpg', 10, 10, 100, $1)
		RETURNING id::text`, by).Scan(&id); err != nil {
		t.Fatalf("إيصال: %v", err)
	}
	return id
}

// TestEXPENSES_AboveThresholdNeedsSecondApprover **القرار ١ و٤.**
func TestEXPENSES_AboveThresholdNeedsSecondApprover(t *testing.T) {
	x := newOpexFixture(t)
	s := x.f.srv
	clerk := testdb.NewUser(t, x.f.pool, "admin")
	boss := testdb.NewUser(t, x.f.pool, "admin")
	limit := s.expenseApprovalThreshold(context.Background())
	over := strconv.FormatInt(limit+1, 10)
	today := expenseToday()

	// فوق السقف بلا إيصال ⇒ يُردّ.
	w := x.call(t, s.handleCreateExpense, clerk, "POST", "/x",
		`{"category_id":"`+x.catID+`","amount":`+over+`,"spent_at":"`+today+`"}`, "")
	if w.Code != 400 || expErrCode(w) != "expense_receipt_required" {
		t.Fatalf("فوق السقف بلا إيصال قُبل: %d %s", w.Code, w.Body.String())
	}

	before := x.balance(t)
	rc := x.receipt(t, clerk)
	w = x.call(t, s.handleCreateExpense, clerk, "POST", "/x",
		`{"category_id":"`+x.catID+`","amount":`+over+`,"note":"إيجار تشرين","spent_at":"`+today+`","receipt_media_id":"`+rc+`"}`, "")
	if w.Code != 201 {
		t.Fatalf("الاقتراح: %d %s", w.Code, w.Body.String())
	}
	d := expData(t, w)
	if d["status"] != "pending" {
		t.Fatalf("فوق السقف قُيّد مباشرة: %v", d)
	}
	reqID, _ := d["id"].(string)
	x.ids = append(x.ids, reqID)
	if got := x.balance(t); got != before {
		t.Fatalf("الاقتراحُ أنقص الخزينة: %d ← %d", before, got)
	}

	// صاحبُ الاقتراح لا يوافق عليه.
	w = x.call(t, s.handleApproveExpenseRequest, clerk, "POST", "/x", `{}`, reqID)
	if w.Code != 403 || expErrCode(w) != "self_approve" {
		t.Fatalf("وافق صاحبُ الاقتراح على نفسه: %d %s", w.Code, w.Body.String())
	}

	// وغيرُه يوافق ⇒ مصروفٌ وقيدٌ يسمّي الباب.
	w = x.call(t, s.handleApproveExpenseRequest, boss, "POST", "/x", `{}`, reqID)
	if w.Code != 200 {
		t.Fatalf("الموافقة: %d %s", w.Code, w.Body.String())
	}
	expID, _ := expData(t, w)["expense_id"].(string)
	x.ids = append(x.ids, expID)
	if got := x.balance(t); got != before-(limit+1) {
		t.Fatalf("بعد الموافقة الرصيدُ %d والمنتظر %d", got, before-(limit+1))
	}
	var note, createdBy, approvedBy string
	if err := x.f.pool.QueryRow(context.Background(), `
		SELECT t.note, e.created_by::text, e.approved_by::text
		FROM wallet_transactions t JOIN expenses e ON e.id::text = t.ref
		WHERE t.ref = $1 AND t.kind = 'operating_expense'`, expID).
		Scan(&note, &createdBy, &approvedBy); err != nil {
		t.Fatalf("القيد: %v", err)
	}
	if note != "مصروف تشغيل — إيجار — إيجار تشرين" {
		t.Fatalf("نصُّ القيد %q", note)
	}
	if createdBy != clerk || approvedBy != boss {
		t.Fatalf("المسجِّل %s والموافق %s", createdBy, approvedBy)
	}
	// ولا تُعاد الموافقة.
	if w := x.call(t, s.handleApproveExpenseRequest, boss, "POST", "/x", `{}`, reqID); w.Code != 409 {
		t.Fatalf("وافق مرّتين: %d", w.Code)
	}
}

// TestEXPENSES_AtThresholdPostsDirectlyAndNamesCategory **تحت السقف قيدٌ مباشرٌ يسمّي الباب.**
func TestEXPENSES_AtThresholdPostsDirectlyAndNamesCategory(t *testing.T) {
	x := newOpexFixture(t)
	s := x.f.srv
	clerk := testdb.NewUser(t, x.f.pool, "admin")
	before := x.balance(t)
	w := x.call(t, s.handleCreateExpense, clerk, "POST", "/x",
		`{"category_id":"`+x.catID+`","amount":1500}`, "")
	if w.Code != 201 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	d := expData(t, w)
	id, _ := d["id"].(string)
	x.ids = append(x.ids, id)
	if d["status"] != "posted" || x.balance(t) != before-1500 {
		t.Fatalf("لم يُقيَّد مباشرة: %v", d)
	}
	var note string
	var spent time.Time
	_ = x.f.pool.QueryRow(context.Background(), `
		SELECT t.note, e.spent_at FROM wallet_transactions t JOIN expenses e ON e.id::text = t.ref
		WHERE t.ref = $1`, id).Scan(&note, &spent)
	// **والملاحظةُ الفارغةُ لا تُخفي الباب.**
	if note != "مصروف تشغيل — إيجار" {
		t.Fatalf("نصُّ القيد %q", note)
	}
	// **ويومُ الصرف الافتراضيُّ يومُ دمشق.**
	if spent.Format("2006-01-02") != expenseToday() {
		t.Fatalf("يومُ الصرف %s ويومُ دمشق %s", spent.Format("2006-01-02"), expenseToday())
	}
}

// TestEXPENSES_NoFutureSpendDate **لا صرفَ في الغد.**
func TestEXPENSES_NoFutureSpendDate(t *testing.T) {
	x := newOpexFixture(t)
	clerk := testdb.NewUser(t, x.f.pool, "admin")
	tomorrow := time.Now().In(damascusLoc()).AddDate(0, 0, 1).Format("2006-01-02")
	w := x.call(t, x.f.srv.handleCreateExpense, clerk, "POST", "/x",
		`{"category_id":"`+x.catID+`","amount":1000,"spent_at":"`+tomorrow+`"}`, "")
	if w.Code != 400 || expErrCode(w) != "expense_future_date" {
		t.Fatalf("قُبل تاريخٌ في المستقبل: %d %s", w.Code, w.Body.String())
	}
}

// TestEXPENSES_VoidNeedsReasonAndAnotherPersonAndStaysVisible **القرار ٥.**
func TestEXPENSES_VoidNeedsReasonAndAnotherPersonAndStaysVisible(t *testing.T) {
	x := newOpexFixture(t)
	s := x.f.srv
	clerk := testdb.NewUser(t, x.f.pool, "admin")
	boss := testdb.NewUser(t, x.f.pool, "admin")
	w := x.call(t, s.handleCreateExpense, clerk, "POST", "/x",
		`{"category_id":"`+x.catID+`","amount":2000,"note":"غلط","spent_at":"2026-08-03"}`, "")
	if w.Code != 201 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	id, _ := expData(t, w)["id"].(string)
	x.ids = append(x.ids, id)

	if w := x.call(t, s.handleVoidExpense, boss, "POST", "/x", `{}`, id); w.Code != 400 ||
		expErrCode(w) != "expense_void_reason_required" {
		t.Fatalf("أُلغي بلا سبب: %d %s", w.Code, w.Body.String())
	}
	if w := x.call(t, s.handleVoidExpense, clerk, "POST", "/x", `{"reason":"مكرر"}`, id); w.Code != 403 ||
		expErrCode(w) != "expense_void_self" {
		t.Fatalf("ألغاه من سجّله: %d %s", w.Code, w.Body.String())
	}
	before := x.balance(t)
	if w := x.call(t, s.handleVoidExpense, boss, "POST", "/x", `{"reason":"مكرر"}`, id); w.Code != 200 {
		t.Fatalf("الإلغاء: %d %s", w.Code, w.Body.String())
	}
	if x.balance(t) != before+2000 {
		t.Fatalf("لم يعُد المال")
	}

	// **والملغى يبقى ظاهراً بحالته وسببه — ولا يُعدّ في المجموع.**
	lw := x.call(t, s.handleListExpenses, boss, "GET", "/x?from=2026-08-03&to=2026-08-03&q=غلط", "", "")
	if lw.Code != 200 {
		t.Fatalf("القائمة: %d %s", lw.Code, lw.Body.String())
	}
	var list struct {
		Data struct {
			Expenses    []expenseRow `json:"expenses"`
			TotalAmount int64        `json:"total_amount"`
		} `json:"data"`
	}
	_ = json.Unmarshal(lw.Body.Bytes(), &list)
	found := false
	for _, e := range list.Data.Expenses {
		if e.ID == id {
			found = true
			if e.Status != "voided" || e.Reason != "مكرر" {
				t.Fatalf("الملغى ظهر %s بسبب %q", e.Status, e.Reason)
			}
		}
	}
	if !found {
		t.Fatal("الملغى اختفى من القائمة")
	}
	if list.Data.TotalAmount != 0 {
		t.Fatalf("الملغى يُعدّ: %d", list.Data.TotalAmount)
	}
}

// TestEXPENSES_InvertedRangeIsSaid **مدًى مقلوبٌ يُقال ولا يُقرأ «لا مصروفات».**
func TestEXPENSES_InvertedRangeIsSaid(t *testing.T) {
	x := newOpexFixture(t)
	w := x.call(t, x.f.srv.handleListExpenses, x.treasury, "GET", "/x?from=2026-09-30&to=2026-09-01", "", "")
	if w.Code != 400 || expErrCode(w) != "expense_bad_range" {
		t.Fatalf("المدى المقلوب: %d %s", w.Code, w.Body.String())
	}
}

// TestEXPENSES_MonthIsBySpendDate **القرار ٢ — إيجارُ أيلول المسجَّلُ اليومَ في أيلول.**
func TestEXPENSES_MonthIsBySpendDate(t *testing.T) {
	x := newOpexFixture(t)
	clerk := testdb.NewUser(t, x.f.pool, "admin")
	ctx := context.Background()
	base, err := opexBySpendDate(ctx, x.f.pool, "2026-07-01", "2026-07-31")
	if err != nil {
		t.Fatal(err)
	}
	w := x.call(t, x.f.srv.handleCreateExpense, clerk, "POST", "/x",
		`{"category_id":"`+x.catID+`","amount":7000,"spent_at":"2026-07-15"}`, "")
	if w.Code != 201 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	id, _ := expData(t, w)["id"].(string)
	x.ids = append(x.ids, id)
	got, err := opexBySpendDate(ctx, x.f.pool, "2026-07-01", "2026-07-31")
	if err != nil {
		t.Fatal(err)
	}
	if got != base+7000 {
		t.Fatalf("تمّوز %d والمنتظر %d — الشهرُ بتاريخ الصرف", got, base+7000)
	}
}

// TestEXPENSES_CategoryAuditIsComplete **إطفاءُ بابٍ يُكتب بما كان وما صار.**
func TestEXPENSES_CategoryAuditIsComplete(t *testing.T) {
	x := newOpexFixture(t)
	ctx := context.Background()
	var id string
	if err := x.f.pool.QueryRow(ctx, `
		INSERT INTO expense_categories (name, sort_order) VALUES ('باب فحص التدقيق', 50)
		RETURNING id::text`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = x.f.pool.Exec(context.Background(), `DELETE FROM expense_categories WHERE id::text = $1`, id)
	})
	w := x.call(t, x.f.srv.handleSaveExpenseCategory, x.treasury, "POST", "/x",
		`{"id":"`+id+`","active":false}`, "")
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	var raw []byte
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := x.f.pool.QueryRow(ctx, `
			SELECT details::text::bytea FROM audit_log
			WHERE action = 'finance.expense_category' AND entity_id = $1
			ORDER BY created_at DESC LIMIT 1`, id).Scan(&raw); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	var meta map[string]any
	_ = json.Unmarshal(raw, &meta)
	if meta["name"] != "باب فحص التدقيق" || meta["active"] != false || meta["old_active"] != true {
		t.Fatalf("السجلُّ ناقص: %s", raw)
	}
}
