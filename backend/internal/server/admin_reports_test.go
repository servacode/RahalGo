package server

// **التقارير — قرارُ المالك ٢٠٢٦-١٠-٠٤.** وما يُحرَس:
//
//   - **المالُ يُحذف في الخادم** عمّن لا يقرأ المال — ويصل للماليّة.
//   - **ربحُ المنصّة = صافي صفحة الأرباح** للمدى نفسِه.
//   - **اليومُ يومُ دمشق** في المدى والمخطّط والافتراض.
//   - **المستردُّ يُعدّ لحاله**، **والفترةُ السابقةُ المساويةُ تُحسب.**
//   - **الزوّارُ أجهزةُ الزبائن وحدَها.**
//   - **أفضلُ المتاجر: لكلّ مصدرٍ حصّتُه.**
//   - **مدًى مقلوبٌ أو أطولُ من سنةٍ يُرفض برسالته.**
//   - **تصديرُ الدفتر: الهاتفُ لمن يقرأ الأرقام، والنوعُ بالعربيّة.**
//
// **والطلباتُ تُكتب في سنواتٍ بعيدة** (٢٠٠٢ و٢٠٠٣) — **فلا يختلط بها ما
// خلّفه غيرُها في القاعدة.**

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

type reportOut struct {
	MoneyVisible   bool            `json:"money_visible"`
	Summary        reportSummary   `json:"summary"`
	Previous       reportSummary   `json:"previous"`
	PreviousFrom   string          `json:"previous_from"`
	PreviousTo     string          `json:"previous_to"`
	Daily          []reportDay     `json:"daily"`
	TopMerchants   []reportStore   `json:"top_merchants"`
	TopDrivers     []reportDriver  `json:"top_drivers"`
	Visitors       *reportVisitors `json:"visitors"`
	VisitorsFailed bool            `json:"visitors_failed"`
}

var (
	analystCaps = []string{string(authz.AnalyticsRead)}
	financeCaps = []string{string(authz.AnalyticsRead), string(authz.FinanceRead)}
)

func readReport(t *testing.T, srv *Server, query string, caps []string) reportOut {
	t.Helper()
	var out reportOut
	if err := json.Unmarshal(dataOf(t, callAs(srv.handleReports, "GET",
		"/admin/reports?"+query, "", "", caps)), &out); err != nil {
		t.Fatalf("التقريرُ لا يُقرأ: %v", err)
	}
	return out
}

// reportFixture متجران وزبونٌ وسائق.
type reportFixture struct {
	srv              *Server
	customer, driver string
	storeA, storeB   string
}

func newReportFixture(t *testing.T) reportFixture {
	t.Helper()
	srv := overviewServer(t)
	ctx := context.Background()
	f := reportFixture{srv: srv,
		customer: testdb.NewUser(t, srv.pg, "customer"),
		driver:   testdb.NewUser(t, srv.pg, "driver"),
	}
	var categoryID string
	if err := srv.pg.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	mk := func(name string) string {
		var id string
		if err := srv.pg.QueryRow(ctx, `
			INSERT INTO merchants (name, category_id, commission_percent)
			VALUES ($1, $2, 10) RETURNING id`, name, categoryID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = srv.pg.Exec(context.Background(), `DELETE FROM orders WHERE merchant_id = $1`, id)
			_, _ = srv.pg.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, id)
		})
		return id
	}
	f.storeA = mk("متجرُ تقارير أ")
	f.storeB = mk("متجرُ تقارير ب")
	return f
}

// order **طلبٌ بوقت إنشائه** — والمسلَّمُ يُسلَّم بعده بنصف ساعة.
func (f reportFixture) order(t *testing.T, status string, created time.Time, total int64) string {
	t.Helper()
	var delivered any
	if status == "delivered" {
		delivered = created.Add(30 * time.Minute)
	}
	var id string
	if err := f.srv.pg.QueryRow(context.Background(), `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due,
			platform_commission, created_at, delivered_at,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ($1, $2, $3, $4, 'عنوانُ تقرير',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', $5 - 2000, 2000, $5, 0, $5, 1000, $6, $7,
			`+qaSnapSQL()+`)
		RETURNING id`, f.customer, f.storeA, f.driver, status, total, created, delivered).Scan(&id); err != nil {
		t.Fatalf("طلبٌ %s: %v", status, err)
	}
	t.Cleanup(func() {
		_, _ = f.srv.pg.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id)
	})
	return id
}

func damascusAt(t *testing.T, s string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Damascus")
	if err != nil {
		t.Skipf("لا منطقةَ دمشق: %v", err)
	}
	v, err := time.ParseInLocation("2006-01-02 15:04", s, loc)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// ── ١ · المالُ يُحذف في الخادم ─────────────────────────────────────────

func TestReports_MoneyHiddenForNonFinance(t *testing.T) {
	f := newReportFixture(t)
	f.order(t, "delivered", damascusAt(t, "2002-05-10 12:00"), 12000)
	q := "from=2002-05-10&to=2002-05-10"

	w := callAs(f.srv.handleReports, "GET", "/admin/reports?"+q, "", "", analystCaps)
	body := string(dataOf(t, w))
	for _, key := range []string{"gross_sales", "commissions", "platform_profit",
		"delivery_fees", "wallet_paid", "cash_collected", `"sales"`, `"cash"`} {
		if strings.Contains(body, key) {
			t.Fatalf("مبلغٌ %s وصل لمن لا يقرأ المال: %s", key, body)
		}
	}
	if r := readReport(t, f.srv, q, analystCaps); r.MoneyVisible || r.Summary.Delivered != 1 {
		t.Fatalf("أرقامُ الطلبات تبقى، والمالُ مخفيّ: %+v", r.Summary)
	}

	r := readReport(t, f.srv, q, financeCaps)
	if !r.MoneyVisible || r.Summary.GrossSales == nil || *r.Summary.GrossSales != 12000 ||
		r.Summary.Commissions == nil || *r.Summary.Commissions != 1000 ||
		r.Summary.PlatformProfit == nil {
		t.Fatalf("الماليّةُ لا ترى المال: %+v", r.Summary)
	}
	if len(r.Daily) != 1 || r.Daily[0].Sales == nil || *r.Daily[0].Sales != 12000 {
		t.Fatalf("مبيعاتُ اليوم للماليّة: %+v", r.Daily)
	}
}

// ── ٢ · ربحُ المنصّة = صافي صفحة الأرباح ───────────────────────────────

func TestReports_PlatformProfitEqualsProfitsPage(t *testing.T) {
	srv := overviewServer(t)
	ctx := context.Background()
	treasury := srv.orders.TreasuryID(ctx)
	if treasury == "" {
		treasury = testdb.NewUser(t, srv.pg, "admin")
		if _, err := srv.pg.Exec(ctx, `
			INSERT INTO wallets (user_id, balance, is_treasury) VALUES ($1, 0, true)
			ON CONFLICT (user_id) DO UPDATE SET is_treasury = true`, treasury); err != nil {
			t.Fatalf("تعذّرت الخزينة: %v", err)
		}
		t.Cleanup(func() {
			_, _ = srv.pg.Exec(context.Background(), `DELETE FROM wallet_transactions WHERE user_id = $1`, treasury)
			_, _ = srv.pg.Exec(context.Background(), `DELETE FROM wallets WHERE user_id = $1`, treasury)
		})
	}
	// **قيدٌ في الخزينة اليومَ** — فلا يتساوى الرقمان صفراً بلا معنى.
	const note = "قيدُ فحص التقارير"
	if _, err := srv.wallet.Apply(ctx, treasury, 54_321, "platform_profit", "", note, nil); err != nil {
		t.Fatalf("القيد: %v", err)
	}
	t.Cleanup(func() {
		_, _ = srv.pg.Exec(context.Background(),
			`DELETE FROM wallet_transactions WHERE user_id = $1 AND note = $2`, treasury, note)
		_, _ = srv.pg.Exec(context.Background(),
			`UPDATE wallets SET balance = balance - 54321 WHERE user_id = $1`, treasury)
	})

	today, _, _ := overviewDays(time.Now())
	day := today.Format("2006-01-02")
	q := "from=" + day + "&to=" + day

	w := callAs(srv.handleProfits, "GET", "/admin/profits?tab=platform&"+q, "", "", financeCaps)
	var prof struct {
		Net int64 `json:"net"`
	}
	if err := json.Unmarshal(dataOf(t, w), &prof); err != nil {
		t.Fatal(err)
	}
	r := readReport(t, srv, q, financeCaps)
	if r.Summary.PlatformProfit == nil || *r.Summary.PlatformProfit != prof.Net {
		t.Fatalf("ربحُ المنصّة في التقارير %v وصافي صفحة الأرباح %d", r.Summary.PlatformProfit, prof.Net)
	}
	if prof.Net == 0 {
		t.Fatal("الصافي صفر — الفحصُ لا يقيس شيئاً")
	}
}

// ── ٣ · يومُ دمشق ──────────────────────────────────────────────────────

func TestReports_DamascusDayBucketing(t *testing.T) {
	f := newReportFixture(t)
	// **الواحدةُ والنصفُ فجراً بدمشق** — وهي في اليوم السابق بغرينتش.
	at := damascusAt(t, "2003-02-16 01:30")
	if at.UTC().Day() != 15 {
		t.Fatalf("الفحصُ يفترض أنّ الوقتَ في اليوم السابق بغرينتش: %v", at.UTC())
	}
	f.order(t, "pending", at, 5000)

	if r := readReport(t, f.srv, "from=2003-02-16&to=2003-02-16", analystCaps); r.Summary.OrdersTotal != 1 {
		t.Fatalf("طلبُ الفجر لم يُعدّ في يومه بدمشق: %d", r.Summary.OrdersTotal)
	}
	if r := readReport(t, f.srv, "from=2003-02-15&to=2003-02-15", analystCaps); r.Summary.OrdersTotal != 0 {
		t.Fatalf("طلبُ الفجر عُدّ في يوم غرينتش: %d", r.Summary.OrdersTotal)
	}
	r := readReport(t, f.srv, "from=2003-02-15&to=2003-02-16", analystCaps)
	if len(r.Daily) != 2 || r.Daily[0].Orders != 0 || r.Daily[1].Day != "2003-02-16" || r.Daily[1].Orders != 1 {
		t.Fatalf("المخطّطُ اليوميّ ليس بيوم دمشق: %+v", r.Daily)
	}
}

func TestReports_DefaultRangeEndsOnDamascusToday(t *testing.T) {
	// **الحاديةَ عشرةَ والنصفُ ليلاً بغرينتش = الواحدةُ والنصفُ فجراً بدمشق.**
	now := time.Date(2026, 10, 4, 22, 30, 0, 0, time.UTC)
	f, err := parseReportFilter(map[string][]string{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if f.toS() != "2026-10-05" || f.fromS() != "2026-09-29" {
		t.Fatalf("الافتراض %s ← %s — **ويومُ دمشق ٢٠٢٦-١٠-٠٥**", f.fromS(), f.toS())
	}
}

// ── ٤ · المستردُّ لحاله، والفترةُ السابقة ──────────────────────────────

func TestReports_RefundedAndPreviousPeriod(t *testing.T) {
	f := newReportFixture(t)
	f.order(t, "refunded", damascusAt(t, "2002-07-12 10:00"), 8000)
	f.order(t, "delivered", damascusAt(t, "2002-07-13 10:00"), 8000)
	// **الفترةُ السابقةُ المساوية** لـ ١٢–١٣ هي ١٠–١١.
	f.order(t, "delivered", damascusAt(t, "2002-07-10 10:00"), 8000)
	f.order(t, "delivered", damascusAt(t, "2002-07-11 10:00"), 8000)
	f.order(t, "cancelled", damascusAt(t, "2002-07-11 11:00"), 8000)

	r := readReport(t, f.srv, "from=2002-07-12&to=2002-07-13", financeCaps)
	if r.Summary.Refunded != 1 || r.Summary.Delivered != 1 || r.Summary.Cancelled != 0 {
		t.Fatalf("المستردُّ لا يُعدّ لحاله: %+v", r.Summary)
	}
	if r.PreviousFrom != "2002-07-10" || r.PreviousTo != "2002-07-11" {
		t.Fatalf("الفترةُ السابقة %s ← %s", r.PreviousFrom, r.PreviousTo)
	}
	if r.Previous.OrdersTotal != 3 || r.Previous.Delivered != 2 || r.Previous.Cancelled != 1 ||
		r.Previous.GrossSales == nil || *r.Previous.GrossSales != 16000 {
		t.Fatalf("أرقامُ الفترة السابقة: %+v", r.Previous)
	}
	if a := readReport(t, f.srv, "from=2002-07-12&to=2002-07-13", analystCaps); a.Previous.GrossSales != nil {
		t.Fatal("مالُ الفترة السابقة وصل لمن لا يقرأ المال")
	}
}

// ── ٥ · الزوّارُ أجهزةُ الزبائن ────────────────────────────────────────

func TestReports_VisitorsCountCustomerDevicesOnly(t *testing.T) {
	srv := overviewServer(t)
	ctx := context.Background()
	before := readReport(t, srv, "", analystCaps)
	if before.Visitors == nil {
		t.Fatal("الزوّارُ لم يُقرؤوا")
	}
	add := func(role, app string) {
		uid := testdb.NewUser(t, srv.pg, role)
		tok := "REPORTS-" + app + "-" + uid
		if _, err := srv.pg.Exec(ctx, `
			INSERT INTO device_tokens (token, user_id, platform, app, last_seen_at)
			VALUES ($1, $2, 'android', $3, now())`, tok, uid, app); err != nil {
			t.Fatalf("جهاز %s: %v", app, err)
		}
		t.Cleanup(func() {
			_, _ = srv.pg.Exec(context.Background(), `DELETE FROM device_tokens WHERE token = $1`, tok)
		})
	}
	add("customer", "customer")
	add("driver", "driver")
	add("merchant", "merchant")

	after := readReport(t, srv, "", analystCaps)
	if d := after.Visitors.DevicesToday - before.Visitors.DevicesToday; d != 1 {
		t.Fatalf("أجهزةُ اليوم زادت %d لا ١ — **السائقُ والمتجرُ ليسا زوّاراً**", d)
	}
	if d := after.Visitors.Devices7 - before.Visitors.Devices7; d != 1 {
		t.Fatalf("أجهزةُ الأسبوع زادت %d لا ١", d)
	}
}

// ── ٦ · أفضلُ المتاجر: لكلّ مصدرٍ حصّتُه ───────────────────────────────

func TestReports_TopStoresCreditEachSource(t *testing.T) {
	f := newReportFixture(t)
	id := f.order(t, "delivered", damascusAt(t, "2002-09-03 12:00"), 12000)
	if _, err := f.srv.pg.Exec(context.Background(), `
		INSERT INTO order_items (order_id, name, unit_price, qty, merchant_id, merchant_price)
		VALUES ($1, 'صنفٌ من أ', 3000, 1, $2, 2500), ($1, 'صنفٌ من ب', 3500, 2, $3, 3000)`,
		id, f.storeA, f.storeB); err != nil {
		t.Fatalf("الأصناف: %v", err)
	}
	r := readReport(t, f.srv, "from=2002-09-03&to=2002-09-03", financeCaps)
	got := map[string]int64{}
	for _, s := range r.TopMerchants {
		if s.Sales == nil || s.Delivered != 1 {
			t.Fatalf("متجرٌ بلا مبيعات أو بعدٍّ خاطئ: %+v", s)
		}
		got[s.ID] = *s.Sales
	}
	if got[f.storeA] != 3000 || got[f.storeB] != 7000 || len(got) != 2 {
		t.Fatalf("الحصص %v — أ ٣٠٠٠ وب ٧٠٠٠", got)
	}
}

// ── ٧ · المدى المقلوب والطويل ──────────────────────────────────────────

func TestReports_RejectsInvertedAndLongRanges(t *testing.T) {
	srv := overviewServer(t)
	for q, key := range map[string]string{
		"from=2026-10-04&to=2026-10-01": "errors.report_range_inverted",
		"from=2024-01-01&to=2026-01-01": "errors.report_range_too_long",
		"kind=bogus":                    "errors.validation",
	} {
		w := callAs(srv.handleReports, "GET", "/admin/reports?"+q, "", "", analystCaps)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), key) {
			t.Fatalf("%s ردّ %d: %s — والمنتظرُ %s", q, w.Code, w.Body.String(), key)
		}
	}
}

// ── ٨ · تصديرُ الدفتر ──────────────────────────────────────────────────

func TestLedgerExport_PhoneGatedAndArabic(t *testing.T) {
	srv := overviewServer(t)
	ctx := context.Background()
	uid := testdb.NewUser(t, srv.pg, "customer")
	var phone string
	if err := srv.pg.QueryRow(ctx, `SELECT phone::text FROM users WHERE id = $1`, uid).Scan(&phone); err != nil {
		t.Fatal(err)
	}
	const note = "قيدُ فحص تصدير الدفتر"
	if _, err := srv.wallet.Apply(ctx, uid, 777, "adjustment", "", note, nil); err != nil {
		t.Fatalf("القيد: %v", err)
	}
	t.Cleanup(func() {
		_, _ = srv.pg.Exec(context.Background(), `DELETE FROM wallet_transactions WHERE user_id = $1`, uid)
	})
	today, _, _ := overviewDays(time.Now())
	day := today.Format("2006-01-02")
	q := "/admin/ledger/export?from=" + day + "&to=" + day
	exportCaps := []string{string(authz.FinanceExport)}

	line := func(body string) string {
		for _, l := range strings.Split(body, "\n") {
			if strings.Contains(l, note) {
				return l
			}
		}
		t.Fatalf("قيدُ الفحص غائبٌ عن الملف")
		return ""
	}

	w := callAs(srv.handleLedgerExport, "GET", q, "", "", exportCaps)
	if w.Code != http.StatusOK {
		t.Fatalf("ردّ %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	head := strings.SplitN(body, "\n", 2)[0]
	if strings.Contains(head, "الهاتف") || strings.Contains(body, phone) {
		t.Fatal("الهاتفُ في الملف لمن لا يملك قراءةَ الأرقام")
	}
	l := line(body)
	if !strings.Contains(l, "تسوية إدارية") || strings.Contains(l, "adjustment") {
		t.Fatalf("نوعُ القيد ليس بالعربيّة: %s", l)
	}

	w = callAs(srv.handleLedgerExport, "GET", q, "", "",
		append(exportCaps, string(authz.UsersContactRead)))
	body = w.Body.String()
	if !strings.Contains(strings.SplitN(body, "\n", 2)[0], "الهاتف") || !strings.Contains(line(body), phone) {
		t.Fatal("صاحبُ قراءة الأرقام لا يرى الهاتف")
	}
	if w := callAs(srv.handleLedgerExport, "GET", "/admin/ledger/export?from=2026-10-04&to=2026-10-01",
		"", "", exportCaps); w.Code != http.StatusBadRequest {
		t.Fatalf("مدًى مقلوبٌ ردّ %d", w.Code)
	}
}

// TestLedgerKindsCoverEveryKind **لكلّ نوعِ قيدٍ اسمٌ عربيّ** — ويطابق
// معجمَ الشاشة حيث يسمّيه.
func TestLedgerKindsCoverEveryKind(t *testing.T) {
	for k := range fininv.Kinds {
		if ledgerKinds[k] == "" {
			t.Errorf("نوعُ القيد %q بلا اسمٍ عربيٍّ في التصدير", k)
		}
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "web", "packages", "i18n", "src", "locales", "ar.json"))
	if err != nil {
		t.Skipf("المعجمُ غيرُ متاح: %v", err)
	}
	var dict struct {
		Shared struct {
			TxKinds map[string]string `json:"txKinds"`
		} `json:"shared"`
	}
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatal(err)
	}
	for k, v := range ledgerKinds {
		if dict.Shared.TxKinds[k] != v {
			t.Errorf("%q: التصدير «%s» والمعجم «%s»", k, v, dict.Shared.TxKinds[k])
		}
	}
}
