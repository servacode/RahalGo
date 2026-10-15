package server

// **قراراتُ المالك ٢٠٢٦-١٠-١٥ — المصروفاتُ والتعويضات.**
//
//	المصروفات: اسمُ الموظّف يُحفظ ويُبحث ويُجمع · تذكيرُ الشهر مرّةً · سببُ الرفض إلزاميٌّ
//	ويُعرض · علَما «يلغي/يوافق» من الخادم والموافقُ لا يلغي · البطاقاتُ تتبع البابَ
//	والبحث · الإيصالُ لمصروفٍ واحد.
//	التعويضات: لا «تعويض» من نافذة المحفظة · المرفوضُ يُقترح ثانيةً · التنبيهُ للماليّة
//	· والمستفيدُ يُخبَر عند الموافقة بلا ملاحظة المكتب.

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// listRaw **القائمةُ خاماً** — حقولٌ تُقرأ بأسمائها في الردّ لا ببنيةٍ في الشيفرة.
func (x *opexFixture) listRaw(t *testing.T, actor, query string) map[string]any {
	t.Helper()
	w := x.call(t, x.f.srv.handleListExpenses, actor, "GET", "/x?"+query, "", "")
	if w.Code != 200 {
		t.Fatalf("القائمة: %d %s", w.Code, w.Body.String())
	}
	return expData(t, w)
}

func rowsOf(d map[string]any) []map[string]any {
	out := []map[string]any{}
	list, _ := d["expenses"].([]any)
	for _, r := range list {
		if m, ok := r.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func rowByID(d map[string]any, id string) map[string]any {
	for _, r := range rowsOf(d) {
		if r["id"] == id {
			return r
		}
	}
	return nil
}

func (x *opexFixture) catByName(t *testing.T, name string) string {
	t.Helper()
	var id string
	if err := x.f.pool.QueryRow(context.Background(),
		`SELECT id::text FROM expense_categories WHERE name = $1`, name).Scan(&id); err != nil {
		t.Fatalf("لا بابَ %q: %v", name, err)
	}
	return id
}

func (x *opexFixture) create(t *testing.T, actor, body string) (int, map[string]any, string) {
	t.Helper()
	w := x.call(t, x.f.srv.handleCreateExpense, actor, "POST", "/x", body, "")
	if w.Code != 201 {
		return w.Code, nil, expErrCode(w)
	}
	d := expData(t, w)
	if id, _ := d["id"].(string); id != "" {
		x.ids = append(x.ids, id)
	}
	return w.Code, d, ""
}

// TestEXP1015_EmployeeNameStoredSearchableSummed **اسمُ الموظّف يُحفظ ويُبحث ويُجمع لكلّ موظّف.**
func TestEXP1015_EmployeeNameStoredSearchableSummed(t *testing.T) {
	x := newOpexFixture(t)
	clerk := testdb.NewUser(t, x.f.pool, "admin")
	salary := x.catByName(t, "رواتب")
	tag := "موظف-فحص-" + strconv.FormatInt(time.Now().UnixNano()%1_000_000, 10)
	other := tag + "-ب"

	// **الرواتبُ بلا اسمٍ تُردّ.**
	if code, _, ec := x.create(t, clerk, `{"category_id":"`+salary+`","amount":1000,"spent_at":"2025-01-10"}`); code != 400 ||
		ec != "expense_employee_required" {
		t.Fatalf("راتبٌ بلا اسم موظّف قُبل: %d %s", code, ec)
	}
	for _, c := range []struct {
		name   string
		amount int
	}{{tag, 1000}, {tag, 500}, {other, 2000}} {
		if code, _, ec := x.create(t, clerk, `{"category_id":"`+salary+`","amount":`+strconv.Itoa(c.amount)+
			`,"spent_at":"2025-01-10","employee_name":"`+c.name+`"}`); code != 201 {
			t.Fatalf("راتب %s: %d %s", c.name, code, ec)
		}
	}

	d := x.listRaw(t, clerk, "from=2025-01-01&to=2025-01-31&category_id="+salary+"&q="+url.QueryEscape(tag))
	rows := rowsOf(d)
	if len(rows) != 3 {
		t.Fatalf("**البحثُ باسم الموظّف لم يجد رواتبه** — %d صفّ", len(rows))
	}
	for _, r := range rows {
		if n, _ := r["employee_name"].(string); !strings.HasPrefix(n, tag) {
			t.Fatalf("**اسمُ الموظّف لم يُحفظ**: %v", r)
		}
	}
	sums := map[string]float64{}
	list, _ := d["by_employee"].([]any)
	for _, e := range list {
		m, _ := e.(map[string]any)
		n, _ := m["name"].(string)
		v, _ := m["total"].(float64)
		sums[n] = v
	}
	if sums[tag] != 1500 || sums[other] != 2000 {
		t.Fatalf("**مجموعُ كلّ موظّف خطأ**: %v", sums)
	}
}

// TestEXP1015_MonthlyReminderOncePerMonth **تذكيرُ الشهر مرّةً واحدةً أوّلَ الشهر.**
func TestEXP1015_MonthlyReminderOncePerMonth(t *testing.T) {
	x := newOpexFixture(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	x.f.srv.notify = notifications.New(x.f.pool, x.f.srv.hub, quiet)
	fin := testdb.NewUser(t, x.f.pool, "finance")
	ctx := context.Background()
	month := time.Date(2031, 3, 1, 0, 0, 0, 0, damascusLoc())
	t.Cleanup(func() {
		_, _ = x.f.pool.Exec(context.Background(),
			`DELETE FROM expense_monthly_reminders WHERE month = '2031-03-01'`)
	})
	count := func() int {
		var n int
		_ = x.f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications
			WHERE user_id = $1 AND href = '/dashboard/expenses'`, fin).Scan(&n)
		return n
	}
	// **وفي غير أوّل الشهر لا تذكير.**
	if sent, _ := x.f.srv.remindRecurringExpenses(ctx, month.AddDate(0, 0, 4)); sent || count() != 0 {
		t.Fatalf("ذُكِّر في اليوم الخامس")
	}
	for i := 0; i < 3; i++ {
		if _, err := x.f.srv.remindRecurringExpenses(ctx, month.Add(time.Duration(i)*time.Hour)); err != nil {
			t.Fatalf("التذكير: %v", err)
		}
	}
	if got := count(); got != 1 {
		t.Fatalf("**تذكيرُ الشهر وصل %d مرّة والمنتظَر مرّة**", got)
	}
}

// TestEXP1015_RejectNeedsReasonAndShowsIt **الرفضُ بسببٍ يُحفظ ويُعرض على صفّه.**
func TestEXP1015_RejectNeedsReasonAndShowsIt(t *testing.T) {
	x := newOpexFixture(t)
	s := x.f.srv
	clerk := testdb.NewUser(t, x.f.pool, "admin")
	boss := testdb.NewUser(t, x.f.pool, "admin")
	over := strconv.FormatInt(s.expenseApprovalThreshold(context.Background())+1, 10)
	rc := x.receipt(t, clerk)
	code, d, ec := x.create(t, clerk, `{"category_id":"`+x.catID+`","amount":`+over+
		`,"note":"رفض-فحص","spent_at":"2025-02-03","receipt_media_id":"`+rc+`"}`)
	if code != 201 {
		t.Fatalf("الاقتراح: %d %s", code, ec)
	}
	reqID, _ := d["id"].(string)
	if w := x.call(t, s.handleRejectExpenseRequest, boss, "POST", "/x", `{}`, reqID); w.Code != 400 ||
		expErrCode(w) != "expense_reject_reason_required" {
		t.Fatalf("**رُفض بلا سبب**: %d %s", w.Code, w.Body.String())
	}
	if w := x.call(t, s.handleRejectExpenseRequest, boss, "POST", "/x", `{"note":"الإيصال لا يطابق"}`, reqID); w.Code != 200 {
		t.Fatalf("الرفض: %d %s", w.Code, w.Body.String())
	}
	r := rowByID(x.listRaw(t, boss, "from=2025-02-01&to=2025-02-28&q="+url.QueryEscape("رفض-فحص")), reqID)
	if r == nil || r["status"] != "rejected" || r["reason"] != "الإيصال لا يطابق" {
		t.Fatalf("**سببُ الرفض لا يُعرض**: %v", r)
	}
}

// TestEXP1015_CanVoidCanApproveFlags **علَما الأزرار من الخادم — والموافقُ لا يلغي وحدَه.**
func TestEXP1015_CanVoidCanApproveFlags(t *testing.T) {
	x := newOpexFixture(t)
	s := x.f.srv
	clerk := testdb.NewUser(t, x.f.pool, "admin")
	boss := testdb.NewUser(t, x.f.pool, "admin")
	third := testdb.NewUser(t, x.f.pool, "admin")
	tag := "أعلام-فحص-" + strconv.FormatInt(time.Now().UnixNano()%1_000_000, 10)
	q := "from=2025-03-01&to=2025-03-31&q=" + url.QueryEscape(tag)

	_, d, _ := x.create(t, clerk, `{"category_id":"`+x.catID+`","amount":1000,"note":"`+tag+` مباشر","spent_at":"2025-03-04"}`)
	direct, _ := d["id"].(string)
	over := strconv.FormatInt(s.expenseApprovalThreshold(context.Background())+1, 10)
	rc := x.receipt(t, clerk)
	_, d, _ = x.create(t, clerk, `{"category_id":"`+x.catID+`","amount":`+over+`,"note":"`+tag+` اقتراح","spent_at":"2025-03-04","receipt_media_id":"`+rc+`"}`)
	reqID, _ := d["id"].(string)

	asClerk, asBoss := x.listRaw(t, clerk, q), x.listRaw(t, boss, q)
	if r := rowByID(asClerk, direct); r == nil || r["can_void"] != false {
		t.Fatalf("**من سجّل يرى «إلغاء»**: %v", r)
	}
	if r := rowByID(asBoss, direct); r == nil || r["can_void"] != true {
		t.Fatalf("غيرُه لا يرى «إلغاء»: %v", r)
	}
	if r := rowByID(asClerk, reqID); r == nil || r["can_approve"] != false {
		t.Fatalf("**صاحبُ الاقتراح يرى «موافقة»**: %v", r)
	}
	if r := rowByID(asBoss, reqID); r == nil || r["can_approve"] != true {
		t.Fatalf("غيرُه لا يرى «موافقة»: %v", r)
	}

	// **والموافقُ لا يلغي ما وافق عليه وحدَه.**
	w := x.call(t, s.handleApproveExpenseRequest, boss, "POST", "/x", `{}`, reqID)
	if w.Code != 200 {
		t.Fatalf("الموافقة: %d %s", w.Code, w.Body.String())
	}
	expID, _ := expData(t, w)["expense_id"].(string)
	x.ids = append(x.ids, expID)
	if r := rowByID(x.listRaw(t, boss, q), expID); r == nil || r["can_void"] != false {
		t.Fatalf("**الموافقُ يرى «إلغاء» على ما وافق عليه**: %v", r)
	}
	if w := x.call(t, s.handleVoidExpense, boss, "POST", "/x", `{"reason":"خطأ"}`, expID); w.Code != 403 ||
		expErrCode(w) != "expense_void_self" {
		t.Fatalf("**ألغى الموافقُ ما وافق عليه**: %d %s", w.Code, w.Body.String())
	}
	if r := rowByID(x.listRaw(t, third, q), expID); r == nil || r["can_void"] != true {
		t.Fatalf("الثالثُ لا يرى «إلغاء»: %v", r)
	}
	if w := x.call(t, s.handleVoidExpense, third, "POST", "/x", `{"reason":"خطأ"}`, expID); w.Code != 200 {
		t.Fatalf("الثالثُ لم يلغِ: %d %s", w.Code, w.Body.String())
	}
}

// TestEXP1015_CardsFollowCategoryAndSearch **بطاقاتُ الشهر تتبع البابَ والبحث.**
func TestEXP1015_CardsFollowCategoryAndSearch(t *testing.T) {
	x := newOpexFixture(t)
	clerk := testdb.NewUser(t, x.f.pool, "admin")
	net := x.catByName(t, "إنترنت")
	tag := "بطاقات-فحص-" + strconv.FormatInt(time.Now().UnixNano()%1_000_000, 10)
	x.create(t, clerk, `{"category_id":"`+x.catID+`","amount":1234,"note":"`+tag+`","spent_at":"2024-11-10"}`)
	x.create(t, clerk, `{"category_id":"`+net+`","amount":4321,"note":"غيره","spent_at":"2024-11-11"}`)

	sum := func(d map[string]any) map[string]any { m, _ := d["summary"].(map[string]any); return m }
	d := x.listRaw(t, clerk, "from=2024-11-01&to=2024-11-30&category_id="+x.catID)
	if s := sum(d); s["month_total"] != d["total_amount"] || s["top_category"] != "إيجار" {
		t.Fatalf("**البطاقاتُ لا تتبع الباب**: الشهر %v والمجموع %v وأكبرُ باب %v",
			s["month_total"], d["total_amount"], s["top_category"])
	}
	d = x.listRaw(t, clerk, "from=2024-11-01&to=2024-11-30&q="+url.QueryEscape(tag))
	if s := sum(d); s["month_total"] != float64(1234) {
		t.Fatalf("**البطاقاتُ لا تتبع البحث**: %v", s["month_total"])
	}
}

// TestEXP1015_ReceiptReuseRefused **الإيصالُ لمصروفٍ واحد.**
func TestEXP1015_ReceiptReuseRefused(t *testing.T) {
	x := newOpexFixture(t)
	clerk := testdb.NewUser(t, x.f.pool, "admin")
	rc := x.receipt(t, clerk)
	body := `{"category_id":"` + x.catID + `","amount":900,"spent_at":"2025-04-02","receipt_media_id":"` + rc + `"}`
	if code, _, ec := x.create(t, clerk, body); code != 201 {
		t.Fatalf("الأوّل: %d %s", code, ec)
	}
	if code, _, ec := x.create(t, clerk, body); code != 409 || ec != "expense_receipt_used" {
		t.Fatalf("**إيصالٌ واحدٌ لمصروفين**: %d %s", code, ec)
	}
}

// ── التعويضات ─────────────────────────────────────────────────────────

// TestCOMP1015_WalletCompensationKindRefused **لا «تعويض» من نافذة المحفظة — طريقُه صفحةُ التعويضات.**
func TestCOMP1015_WalletCompensationKindRefused(t *testing.T) {
	f := newDriverFixture(t, 1)
	fin := testdb.NewUser(t, f.pool, "finance")
	w := f.call(f.srv.handleAdminWalletApply, http.MethodPost, "/x", f.drivers[0], fin, []string{"finance"},
		`{"amount":1000,"kind":"compensation","note":"تعويض"}`)
	if w.Code != http.StatusBadRequest || errCode(t, w) != "wallet_compensation_not_here" {
		t.Fatalf("**قُبل تعويضٌ من نافذة المحفظة**: %d %s", w.Code, w.Body.String())
	}
}

// TestCOMP1015_RejectedDriverCompCanBeProposedAgain **المرفوضُ يُقترح ثانيةً.**
func TestCOMP1015_RejectedDriverCompCanBeProposedAgain(t *testing.T) {
	f := newDriverFixture(t, 1)
	compTreasury(t, f)
	d := f.drivers[0]
	o := f.problemOrderAt(t, "at_pickup", d)
	id := f.pendingRow(t, o, d, "merchant", "merchant_closed", 2500, "0")
	fin := testdb.NewUser(t, f.pool, "finance")
	if w := f.call(f.srv.handleRejectCompensationByID, http.MethodPost, "/x", id, fin, []string{"finance"},
		`{"note":"ناقص"}`); w.Code != 200 {
		t.Fatalf("الرفض: %d %s", w.Code, w.Body.String())
	}
	w := f.call(f.srv.handleProposeCompensation, http.MethodPost, "/x", "", fin, []string{"finance"},
		`{"order_id":"`+o+`","user_id":"`+d+`","amount":2000,"fault":"merchant","reason":"merchant_closed","note":"اقتراح ثان"}`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("**المرفوضُ لا يُقترح ثانيةً**: %d %s", w.Code, w.Body.String())
	}
	// **والحيُّ واحدٌ** — معلَّقٌ قائمٌ يمنع ثالثاً.
	w = f.call(f.srv.handleProposeCompensation, http.MethodPost, "/x", "", fin, []string{"finance"},
		`{"order_id":"`+o+`","user_id":"`+d+`","amount":2000,"fault":"merchant","reason":"merchant_closed","note":"ثالث"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("معلَّقان حيّان لطلبٍ وسائقٍ واحد: %d %s", w.Code, w.Body.String())
	}
}

// TestCOMP1015_PendingAlertToFinanceAndBeneficiaryNotified **التنبيهُ للماليّة لا للعمليّات،
// والمستفيدُ يُخبَر عند الموافقة بالمبلغ وبلا ملاحظة المكتب.**
func TestCOMP1015_PendingAlertToFinanceAndBeneficiaryNotified(t *testing.T) {
	f := newDriverFixture(t, 1)
	compTreasury(t, f)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	n := notifications.New(f.pool, f.srv.hub, quiet)
	f.srv.notify = n
	f.srv.orders.SetNotifier(n)
	f.srv.orders.SetSettings(settings.NewStore(f.pool))
	d := f.drivers[0]
	o := f.problemOrderAt(t, "at_pickup", d)
	proposer := testdb.NewUser(t, f.pool, "finance")
	approver := testdb.NewUser(t, f.pool, "finance")
	ops := testdb.NewUser(t, f.pool, "operations")
	ctx := context.Background()

	w := f.call(f.srv.handleProposeCompensation, http.MethodPost, "/x", "", proposer, []string{"finance"},
		`{"order_id":"`+o+`","user_id":"`+d+`","amount":3000,"fault":"merchant","reason":"merchant_closed","note":"سر-داخلي"}`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("الاقتراح: %d %s", w.Code, w.Body.String())
	}
	var body struct {
		Data struct {
			Proposed string `json:"proposed"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	alerts := func(user string) int {
		var c int
		_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications
			WHERE user_id = $1 AND href = '/dashboard/compensations'`, user).Scan(&c)
		return c
	}
	if alerts(approver) != 1 {
		t.Fatalf("**الماليّةُ لم تُنبَّه إلى تعويضٍ ينتظرها** (%d)", alerts(approver))
	}
	if alerts(ops) != 0 {
		t.Fatalf("**العمليّاتُ نُبّهت إلى تعويضٍ ليس لها**")
	}

	if code, b := f.approveCompW(body.Data.Proposed, approver, 3000); code != 200 {
		t.Fatalf("الموافقة: %d %s", code, b)
	}
	var title, text string
	var apps []string
	if err := f.pool.QueryRow(ctx, `SELECT title, body, push_apps FROM notifications
		WHERE user_id = $1 AND title LIKE 'انصرفلك تعويض%' ORDER BY created_at DESC LIMIT 1`, d).
		Scan(&title, &text, &apps); err != nil {
		t.Fatalf("**المستفيدُ لم يُخبَر بتعويضه**: %v", err)
	}
	if !strings.Contains(title+text, "3") || strings.Contains(title+text, "سر-داخلي") ||
		strings.Contains(title+text, "موافقة الاختبار") {
		t.Fatalf("نصُّ الخبر: %q · %q", title, text)
	}
	if len(apps) != 1 || apps[0] != notifications.AppDriver {
		t.Fatalf("الخبرُ لا يرنّ في تطبيق السائق وحدَه: %v", apps)
	}
}

var _ = httptest.NewRecorder
