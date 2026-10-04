package server

// الأرباح والخسائر — قرارات المالك ٢٠٢٦-١٠-٠٤.
//
// كلّ اختبار يعمل على يوم ماضٍ خاصّ به (القيود تُرجَع إليه بعد كتابتها)،
// فلا تدخل قيود اختبارات أخرى في حسابه.

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// profitsFx أدوات اختبار الأرباح: خادم وخزينة وزبون وسائق ومتجر.
type profitsFx struct {
	srv      *Server
	treasury string
	customer string
	driver   string
	merchant string
	marker   string
}

func newProfitsFx(t *testing.T, marker string) profitsFx {
	t.Helper()
	srv := overviewServer(t)
	ctx := context.Background()
	f := profitsFx{srv: srv, marker: marker,
		customer: testdb.NewUser(t, srv.pg, "customer"),
		driver:   testdb.NewUser(t, srv.pg, "driver"),
	}
	f.treasury = srv.orders.TreasuryID(ctx)
	if f.treasury == "" {
		f.treasury = testdb.NewUser(t, srv.pg, "admin")
		if _, err := srv.pg.Exec(ctx, `
			INSERT INTO wallets (user_id, balance, is_treasury) VALUES ($1, 0, true)
			ON CONFLICT (user_id) DO UPDATE SET is_treasury = true`, f.treasury); err != nil {
			t.Fatalf("الخزينة: %v", err)
		}
	}
	var cat string
	if err := srv.pg.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&cat); err != nil {
		t.Fatalf("لا تصنيفات: %v", err)
	}
	if err := srv.pg.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent)
		VALUES ('متجر فحص الأرباح', $1, 10) RETURNING id`, cat).Scan(&f.merchant); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = srv.pg.Exec(context.Background(), `DELETE FROM order_items WHERE merchant_id = $1`, f.merchant)
		_, _ = srv.pg.Exec(context.Background(), `DELETE FROM orders WHERE merchant_id = $1`, f.merchant)
		_, _ = srv.pg.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, f.merchant)
	})
	return f
}

// order طلب بحالته وأعمدة ماله. والمسلّم فيه صنف واحد: بيع ١٠٠٠٠ وشراء ٩٠٠٠.
func (f profitsFx) order(t *testing.T, status string, at time.Time) string {
	t.Helper()
	ctx := context.Background()
	var delivered any
	if status == "delivered" || status == "refunded" {
		delivered = at
	}
	var id string
	if err := f.srv.pg.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, driver_fee, discount, total, wallet_paid, cash_due,
			platform_commission, created_at, delivered_at,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ($1, $2, $3, $4, 'عنوان فحص الأرباح',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'wallet', 10000, 2000, 1800, 500, 11500, 11500, 0, 900, $5, $6,
			`+qaSnapSQL()+`)
		RETURNING id`, f.customer, f.merchant, f.driver, status, at, delivered).Scan(&id); err != nil {
		t.Fatalf("طلب %s: %v", status, err)
	}
	if _, err := f.srv.pg.Exec(ctx, `
		INSERT INTO order_items (order_id, merchant_id, name, unit_price, merchant_price, qty, options)
		VALUES ($1, $2, 'صنف فحص', 10000, 9000, 1, '[]')`, id, f.merchant); err != nil {
		t.Fatalf("صنف: %v", err)
	}
	return id
}

// post قيد يمرّ بالمحفظة (فيتحرّك الرصيد)، ثمّ يُرجَع وقته إلى at.
func (f profitsFx) post(t *testing.T, user string, amount int64, kind, ref string, at time.Time) {
	t.Helper()
	ctx := context.Background()
	if _, err := f.srv.wallet.Apply(ctx, user, amount, kind, ref, f.marker, nil); err != nil {
		t.Fatalf("قيد %s %d: %v", kind, amount, err)
	}
	if _, err := f.srv.pg.Exec(ctx, `
		UPDATE wallet_transactions SET created_at = $3
		WHERE id = (SELECT max(id) FROM wallet_transactions WHERE user_id = $1 AND note = $2)`,
		user, f.marker, at); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.srv.pg.Exec(context.Background(),
			`UPDATE wallets SET balance = balance - $2 WHERE user_id = $1`, user, amount)
		_, _ = f.srv.pg.Exec(context.Background(),
			`DELETE FROM wallet_transactions WHERE user_id = $1 AND note = $2 AND kind = $3 AND amount = $4`,
			user, f.marker, kind, amount)
	})
}

// wr طلب حركة يدويّة موافَق عليه — مرجع لقيد الخزينة المقابل.
func (f profitsFx) wr(t *testing.T, kind string, debit bool, amount int64) string {
	t.Helper()
	var id string
	if err := f.srv.pg.QueryRow(context.Background(), `
		INSERT INTO wallet_requests (user_id, kind, debit, amount, note, status, proposed_by)
		VALUES ($1, $2, $3, $4, 'حركة فحص', 'approved', $5) RETURNING id::text`,
		f.customer, kind, debit, amount, f.treasury).Scan(&id); err != nil {
		t.Fatalf("طلب حركة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.srv.pg.Exec(context.Background(), `DELETE FROM wallet_requests WHERE id = $1`, id)
	})
	return id
}

type profitsPlatformResp struct {
	Basis     string       `json:"basis"`
	Orders    int          `json:"orders"`
	Sales     int64        `json:"sales"`
	Income    int64        `json:"income"`
	Lines     []profitLine `json:"lines"`
	LinesSum  int64        `json:"lines_sum"`
	Net       int64        `json:"net"`
	CheckOK   bool         `json:"check_ok"`
	CheckDiff int64        `json:"check_diff"`
	Pending   int64        `json:"pending"`
	Outside   int64        `json:"outside"`
}

func (r profitsPlatformResp) line(k string) int64 {
	for _, l := range r.Lines {
		if l.Key == k {
			return l.Amount
		}
	}
	return 0
}

func readPlatformProfits(t *testing.T, srv *Server, day string) profitsPlatformResp {
	t.Helper()
	w := callAs(srv.handleProfits, "GET", "/x?tab=platform&from="+day+"&to="+day, "", "", financeCaps)
	var out profitsPlatformResp
	if err := json.Unmarshal(dataOf(t, w), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// ══════════════════════════════════════════════════════════════════════
// ١ · مجموع البنود = الصافي — بكلّ أبوابه، وبتاريخ واحد
// ══════════════════════════════════════════════════════════════════════
func TestProfits_LinesReconcileToNet(t *testing.T) {
	f := newProfitsFx(t, "فحص الأرباح ١")
	ctx := context.Background()
	day := damascusAt(t, "2001-03-11 12:00")
	rep := testdb.NewUser(t, f.srv.pg, "sales")

	// طلب مسلّم: هامش ١٠٠٠ + عمولة ٩٠٠ + توصيل ٢٠٠ − خصم ٥٠٠ = ١٦٠٠،
	// ونصيب المندوب ١٦٠ ⇒ الخزينة ١٤٤٠.
	// قيد الاستلام في اليوم السابق (٣٤٠٠) وقيد التسليم في يومه (−١٩٦٠) —
	// والطلب يُحسب كاملاً في يوم آخر قيد له.
	a := f.order(t, "delivered", day)
	f.post(t, f.treasury, 3400, "platform_profit", a, damascusAt(t, "2001-03-10 23:00"))
	f.post(t, f.treasury, -1960, "platform_profit", a, day)
	f.post(t, rep, 160, "commission", a, day)

	b := f.order(t, "failed", day)
	f.post(t, f.treasury, -3000, "platform_profit", b, day)
	c := f.order(t, "cancelled", day)
	f.post(t, f.treasury, -700, "platform_profit", c, day)
	f.post(t, f.treasury, -400, "platform_expense", b, day) // تعويض سائق
	f.post(t, f.treasury, -300, "reward", a, day)           // دعوة
	f.post(t, f.treasury, -200, "reward", "", day)          // هدف
	f.post(t, f.treasury, 250, "penalty", "", day)

	// مصروف صُرف في يومنا وسُجّل بعد ثلاثة أسابيع — يُحسب في يوم صرفه.
	var cat, exp string
	if err := f.srv.pg.QueryRow(ctx, `SELECT id FROM expense_categories LIMIT 1`).Scan(&cat); err != nil {
		t.Fatalf("لا أبواب مصروف: %v", err)
	}
	if err := f.srv.pg.QueryRow(ctx, `
		INSERT INTO expenses (category_id, amount, note, spent_at) VALUES ($1, 1000, 'فحص', '2001-03-11')
		RETURNING id::text`, cat).Scan(&exp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = f.srv.pg.Exec(context.Background(), `DELETE FROM expenses WHERE id = $1`, exp) })
	f.post(t, f.treasury, -1000, "operating_expense", exp, damascusAt(t, "2001-04-02 10:00"))

	r := readPlatformProfits(t, f.srv, "2001-03-11")
	want := map[string]int64{
		"margin": 1000, "commission": 900, "delivery_share": 200, "discount": -500,
		"settle_diff": 0, "rep_share": -160, "lost_failed": -3000, "lost_cancelled": -700,
		"compensations": -400, "referrals": -300, "targets": -200, "opex": -1000, "penalties": 250,
	}
	for k, v := range want {
		if got := r.line(k); got != v {
			t.Errorf("سطر %s = %d، المتوقَّع %d", k, got, v)
		}
	}
	if r.Net != -3910 || r.LinesSum != r.Net || !r.CheckOK {
		t.Fatalf("الصافي %d ومجموع البنود %d والتحقّق %v — المتوقَّع −3910 ومتساويان", r.Net, r.LinesSum, r.CheckOK)
	}
	if r.Orders != 1 || r.Sales != 11500 || r.Income != 1600 {
		t.Fatalf("طلبات %d مبيعات %d دخل %d", r.Orders, r.Sales, r.Income)
	}
	if r.Basis != "ledger_entry_damascus" {
		t.Fatalf("الأساس غير مكتوب: %q", r.Basis)
	}
	// واليوم السابق لا يحمل قيد الاستلام وحده.
	if prev := readPlatformProfits(t, f.srv, "2001-03-10"); prev.Net != 0 {
		t.Fatalf("قيد الاستلام انقسم على يوم سابق: %d", prev.Net)
	}
}

// ══════════════════════════════════════════════════════════════════════
// ٢ · سحب الأدمن والإيداع اليدويّ ليسا ربحاً
// ══════════════════════════════════════════════════════════════════════
func TestProfits_WithdrawalsAndManualDepositsExcluded(t *testing.T) {
	f := newProfitsFx(t, "فحص الأرباح ٢")
	day := damascusAt(t, "2001-05-07 12:00")

	f.post(t, f.treasury, 600, "penalty", "", day)
	deposit := f.wr(t, "adjustment", false, 5000) // إيداع يدويّ لمحفظة ⇒ يخرج من الخزينة
	f.post(t, f.treasury, -5000, "platform_expense", deposit, day)
	take := f.wr(t, "adjustment", true, 700) // خصم يدويّ من محفظة ⇒ يدخل الخزينة
	f.post(t, f.treasury, 700, "platform_profit", take, day)
	f.post(t, f.treasury, 2000, "topup", "", day)
	f.post(t, f.treasury, -2000, "payout", "", day) // سحب الأدمن من رصيد الخزينة

	r := readPlatformProfits(t, f.srv, "2001-05-07")
	if r.Net != 600 {
		t.Fatalf("الصافي %d — السحب والإيداع دخلا الربح (المتوقَّع 600)", r.Net)
	}
	if r.Outside != -4300 {
		t.Fatalf("خارج الربح %d، المتوقَّع −4300", r.Outside)
	}
	if !r.CheckOK || r.line("compensations") != 0 || r.line("recovered") != 0 {
		t.Fatalf("الحركة اليدويّة ظهرت في بند: %+v", r.Lines)
	}
}

// ══════════════════════════════════════════════════════════════════════
// ٣ · خسارة الفاشل والملغى سطر لحاله
// ══════════════════════════════════════════════════════════════════════
func TestProfits_FailedAndCancelledLossLine(t *testing.T) {
	f := newProfitsFx(t, "فحص الأرباح ٣")
	day := damascusAt(t, "2001-06-15 12:00")
	b := f.order(t, "failed", day)
	f.post(t, f.treasury, 8100, "platform_profit", b, damascusAt(t, "2001-06-15 09:00"))
	f.post(t, f.treasury, -11500, "platform_profit", b, day)
	c := f.order(t, "rejected", day)
	f.post(t, f.treasury, -250, "platform_profit", c, day)
	d := f.order(t, "refunded", day)
	f.post(t, f.treasury, -1100, "platform_profit", d, day)
	f.post(t, f.treasury, -400, "platform_expense", b, day)

	r := readPlatformProfits(t, f.srv, "2001-06-15")
	if r.line("lost_failed") != -3400 || r.line("lost_cancelled") != -250 || r.line("lost_refunded") != -1100 {
		t.Fatalf("خسائر: فاشلة %d ملغاة %d مستردّة %d",
			r.line("lost_failed"), r.line("lost_cancelled"), r.line("lost_refunded"))
	}
	// والتعويض سطر غير سطر الخسارة.
	if r.line("compensations") != -400 || r.Net != -5150 || !r.CheckOK {
		t.Fatalf("تعويضات %d صافي %d تحقّق %v", r.line("compensations"), r.Net, r.CheckOK)
	}
}

// ══════════════════════════════════════════════════════════════════════
// ٤ · مال طلب لم يُسلَّم «ربح معلّق» خارج الصافي
// ══════════════════════════════════════════════════════════════════════
func TestProfits_PendingProfitOutsideNet(t *testing.T) {
	f := newProfitsFx(t, "فحص الأرباح ٤")
	day := damascusAt(t, "2001-07-20 12:00")
	o := f.order(t, "on_the_way", day)
	f.post(t, f.treasury, 3400, "platform_profit", o, day)
	f.post(t, f.treasury, 150, "penalty", "", day)

	r := readPlatformProfits(t, f.srv, "2001-07-20")
	if r.Pending != 3400 {
		t.Fatalf("الربح المعلّق %d، المتوقَّع 3400", r.Pending)
	}
	if r.Net != 150 || !r.CheckOK || r.Orders != 0 {
		t.Fatalf("مال طلب لم يُسلَّم دخل الصافي: صافي %d طلبات %d", r.Net, r.Orders)
	}

	// وحين يُسلَّم يخرج من المعلّق ويدخل الربح في يوم قيد تسليمه.
	if _, err := f.srv.pg.Exec(context.Background(),
		`UPDATE orders SET status = 'delivered', delivered_at = $2 WHERE id = $1`, o, day); err != nil {
		t.Fatal(err)
	}
	r = readPlatformProfits(t, f.srv, "2001-07-20")
	if r.Pending != 0 || r.Net != 3550 || r.Orders != 1 {
		t.Fatalf("بعد التسليم: معلّق %d صافي %d طلبات %d", r.Pending, r.Net, r.Orders)
	}
}

// ══════════════════════════════════════════════════════════════════════
// ٥ · المندوب لا يظهر في تبويب الزبائن، والعمولة غير المكافأة
// ══════════════════════════════════════════════════════════════════════
func TestProfits_RepSeparateFromCustomers(t *testing.T) {
	f := newProfitsFx(t, "فحص الأرباح ٥")
	ctx := context.Background()
	day := damascusAt(t, "2001-08-03 12:00")
	rep := testdb.NewUser(t, f.srv.pg, "sales")
	if _, err := f.srv.pg.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_code) VALUES ($1, 'customer') ON CONFLICT DO NOTHING`, rep); err != nil {
		t.Fatal(err)
	}
	f.post(t, rep, 900, "commission", "", day)
	f.post(t, rep, 200, "reward", "", day) // مكافأة هدف
	f.post(t, f.customer, 300, "reward", f.customer, day)

	type tabResp struct {
		Columns []string   `json:"columns"`
		Rows    []partyRow `json:"rows"`
	}
	read := func(tab string) tabResp {
		w := callAs(f.srv.handleProfits, "GET",
			"/x?tab="+tab+"&from=2001-08-03&to=2001-08-03&per_page=100", "", "", financeCaps)
		var out tabResp
		if err := json.Unmarshal(dataOf(t, w), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	val := func(r tabResp, user, col string) (int64, bool) {
		for _, row := range r.Rows {
			if row.UserID != user {
				continue
			}
			for i, c := range r.Columns {
				if c == col {
					return row.Values[i], true
				}
			}
		}
		return 0, false
	}

	cust := read("customers")
	if _, found := val(cust, rep, "bonus"); found {
		t.Fatal("المندوب ظهر في تبويب الزبائن")
	}
	if v, _ := val(cust, f.customer, "referral"); v != 300 {
		t.Fatalf("مكافأة دعوة الزبون %d", v)
	}
	reps := read("reps")
	c, _ := val(reps, rep, "commission")
	b, _ := val(reps, rep, "bonus")
	if c != 900 || b != 200 {
		t.Fatalf("المندوب: عمولة %d مكافأة %d — يجب أن يكونا عمودين", c, b)
	}
}

// ══════════════════════════════════════════════════════════════════════
// ٦ · مدى مقلوب يُردّ برسالته
// ══════════════════════════════════════════════════════════════════════
func TestProfits_InvertedRangeRejected(t *testing.T) {
	srv := overviewServer(t)
	w := callAs(srv.handleProfits, "GET", "/x?tab=platform&from=2026-10-04&to=2026-10-01", "", "", financeCaps)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("مدى مقلوب ردّ %d", w.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// وكلّ تبويب يُنادى — بمدى وبلا مدى
// ══════════════════════════════════════════════════════════════════════
func TestProfits_EveryTabAnswers(t *testing.T) {
	srv := overviewServer(t)
	for _, tab := range []string{"platform", "customers", "reps", "drivers", "merchants"} {
		for _, q := range []string{"&from=2026-08-01&to=2026-08-31", ""} {
			w := callAs(srv.handleProfits, "GET", "/x?tab="+tab+q, "", "", financeCaps)
			if w.Code != 200 {
				t.Fatalf("تبويب %q (%s) ردّ %d — %s", tab, q, w.Code, w.Body.String())
			}
		}
	}
}
