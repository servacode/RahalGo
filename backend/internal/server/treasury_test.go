package server

// **قسمُ الخزينة** — قراراتُ المالك ٢٠٢٦-١٠-٠٤.
//
//	سحبُ الأدمن      قيدٌ بنوعه في الخزينة باسمه + سطرٌ خارجٌ من الصندوق
//	الصندوق          تسليمُ السائق والسحبُ المصروفُ والمرتدُّ كلٌّ بسطره ومرجعه
//	الإغلاقُ اليوميّ   المتوقَّعُ والمعدودُ والفرق، ومراجعةٌ من شخصٍ ثانٍ
//	النقص            لا يصير خسارةً قبل يوم، ولا بيد من أغلق، وعندها قيدٌ في الخزينة
//	الموافقات        صفحةٌ تجمع الجداول وتمنع المقترحَ من الموافقة

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/realtime"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

type treasuryFixture struct {
	pool     *pgxpool.Pool
	srv      *Server
	treasury string
	a, b     string // موظّفا ماليّة
}

func newTreasuryFixture(t *testing.T) *treasuryFixture {
	t.Helper()
	pool := testdb.Pool(t)
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	hub := realtime.NewHub(quiet)
	walletSvc := wallet.NewService(pool)
	store := settings.NewStore(pool)
	cb := cashbox.NewService(pool, store)
	f := &treasuryFixture{
		pool: pool,
		srv: &Server{
			pg: pool, logger: quiet, hub: hub, wallet: walletSvc, cashbox: cb,
			orders:   orders.NewService(pool, nil, walletSvc, cb, nil, quiet),
			settings: store, notify: notifications.New(pool, hub, quiet),
		},
		a: testdb.NewUser(t, pool, "admin"),
		b: testdb.NewUser(t, pool, "admin"),
	}
	ctx := context.Background()
	tid, err := wallet.EnsureTreasury(ctx, pool)
	if err != nil || tid == "" {
		t.Fatalf("لا خزينة: %v", err)
	}
	f.treasury = tid
	return f
}

var treasuryCaps = []string{string(authz.FinanceRead), string(authz.FinanceManage),
	string(authz.FinanceExport), string(authz.TreasuryManage)}

func (f *treasuryFixture) call(actor, method, path string, params map[string]string, body string,
	h http.HandlerFunc) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	for k, v := range params {
		rc.URLParams.Add(k, v)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, actor)
	ctx = context.WithValue(ctx, ctxRoles, []string{"admin"})
	ctx = context.WithValue(ctx, ctxCaps, treasuryCaps)
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

func (f *treasuryFixture) one(t *testing.T, sql string, args ...any) int64 {
	t.Helper()
	var v int64
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func (f *treasuryFixture) fiGreen(t *testing.T, ids ...string) {
	t.Helper()
	vs, err := fininv.Run(context.Background(), f.pool, ids...)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		t.Errorf("خرق: %s", v)
	}
}

// ── سحبُ الأدمن ──────────────────────────────────────────────────────

func TestTREASURY_AdminWithdrawalIsLabelledAndLeavesCashbox(t *testing.T) {
	f := newTreasuryFixture(t)
	ctx := context.Background()
	if _, err := f.srv.wallet.Apply(ctx, f.treasury, 50_000, "platform_profit", "", "تمهيد", nil); err != nil {
		t.Fatal(err)
	}
	bal := f.one(t, `SELECT balance FROM wallets WHERE user_id = $1`, f.treasury)

	// **لا يُسحب أكثرُ من الرصيد.**
	w := f.call(f.a, "POST", "/admin/treasury/withdrawals", nil,
		`{"amount":`+itoa64(bal+1)+`,"note":"زائد"}`, f.srv.handleTreasuryWithdraw)
	if w.Code != http.StatusConflict {
		t.Fatalf("سحبٌ فوق الرصيد ردّ %d", w.Code)
	}
	// **ولا بلا ملاحظة.**
	w = f.call(f.a, "POST", "/admin/treasury/withdrawals", nil, `{"amount":1000}`, f.srv.handleTreasuryWithdraw)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("سحبٌ بلا ملاحظة ردّ %d", w.Code)
	}
	w = f.call(f.a, "POST", "/admin/treasury/withdrawals", nil,
		`{"amount":7000,"note":"=HYPERLINK(1)"}`, f.srv.handleTreasuryWithdraw)
	if w.Code != http.StatusCreated {
		t.Fatalf("السحب ردّ %d: %s", w.Code, w.Body)
	}
	var res struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if got := f.one(t, `SELECT amount FROM wallet_transactions
		WHERE kind = 'treasury_withdrawal' AND ref = $1 AND user_id = $2`, res.Data.ID, f.treasury); got != -7000 {
		t.Fatalf("قيدُ السحب %d", got)
	}
	if got := f.one(t, `SELECT amount FROM office_cash_entries
		WHERE source = 'treasury_withdrawal' AND direction = 'out' AND ref = $1`, res.Data.ID); got != 7000 {
		t.Fatalf("سطرُ الصندوق %d", got)
	}
	// **الكشفُ يسمّيه ويحمل الرصيدَ الجاري.**
	w = f.call(f.a, "GET", "/admin/treasury/statement?kind=treasury_withdrawal", nil, "", f.srv.handleTreasuryStatement)
	var st struct {
		Data struct {
			Lines []treasuryLine `json:"lines"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &st)
	if len(st.Data.Lines) == 0 || st.Data.Lines[0].KindAr != "سحب الأدمن من رصيد الخزينة" ||
		st.Data.Lines[0].Balance != bal-7000 {
		t.Fatalf("الكشف: %+v", st.Data.Lines)
	}
	// **والتصديرُ لا معادلةَ فيه.**
	w = f.call(f.a, "GET", "/admin/treasury/statement/export?kind=treasury_withdrawal", nil, "",
		f.srv.handleTreasuryStatementExport)
	recs, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(w.Body.String(), string(rune(0xFEFF))))).ReadAll()
	if err != nil || len(recs) < 2 {
		t.Fatalf("ملفٌّ غيرُ مقروء: %v", err)
	}
	for _, rec := range recs {
		for _, cell := range rec {
			if strings.HasPrefix(cell, "=") || strings.HasPrefix(cell, "@") {
				t.Fatalf("خليّةٌ بمعادلة: %q", cell)
			}
		}
	}
	f.fiGreen(t, "FI-12", "FI-15")
}

// ── الصندوق: السحبُ المصروفُ والمرتدّ ────────────────────────────────

func TestTREASURY_PayoutPaidAndReversedMoveOfficeCashbox(t *testing.T) {
	f := newTreasuryFixture(t)
	ctx := context.Background()
	d := testdb.NewUser(t, f.pool, "driver")
	if _, err := f.srv.wallet.Apply(ctx, d, 30_000, "topup", "", "تمهيد", nil); err != nil {
		t.Fatal(err)
	}
	var pid string
	if err := f.pool.QueryRow(ctx, `INSERT INTO payout_requests (user_id, amount) VALUES ($1, 20000)
		RETURNING id::text`, d).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if err := f.srv.wallet.ReserveTx(ctx, f.pool, d, 20_000); err != nil {
		t.Fatal(err)
	}
	for _, st := range []string{"paid", "reversed"} {
		w := f.call(f.a, "POST", "/admin/payouts/"+pid+"/decide", map[string]string{"id": pid},
			`{"status":"`+st+`","method":"cash","decision":"صُرف نقداً من المكتب"}`, f.srv.handleDecidePayout)
		if w.Code != http.StatusOK {
			t.Fatalf("%s ردّ %d: %s", st, w.Code, w.Body)
		}
	}
	if got := f.one(t, `SELECT count(*) FROM office_cash_entries
		WHERE ref = $1 AND ((source = 'payout_paid' AND direction = 'out')
		                 OR (source = 'payout_reversed' AND direction = 'in')) AND amount = 20000`, pid); got != 2 {
		t.Fatalf("سطورُ الصندوق للسحب %d — يُنتظر اثنان (خارجٌ ثمّ داخل)", got)
	}
	f.fiGreen(t, "FI-15.c")
}

// ── الصندوق: تسليمُ السائق بمرجع سطره ────────────────────────────────

func TestTREASURY_DriverSettleRefIsItsCashEntry(t *testing.T) {
	f := newTreasuryFixture(t)
	ctx := context.Background()
	d := testdb.NewUser(t, f.pool, "driver")
	if err := f.srv.cashbox.Collect(ctx, d, 40_000, "", nil); err != nil {
		t.Fatal(err)
	}
	w := f.call(f.a, "POST", "/admin/drivers/"+d+"/settle", map[string]string{"id": d},
		`{"amount":25000,"note":"تسليم"}`, f.srv.handleDriverSettle)
	if w.Code != http.StatusOK {
		t.Fatalf("التسليم ردّ %d: %s", w.Code, w.Body)
	}
	if got := f.one(t, `SELECT count(*) FROM office_cash_entries c
		JOIN driver_cash_entries e ON e.id::text = c.ref AND e.kind = 'settlement' AND e.driver_id = $1
		WHERE c.source = 'driver_settle' AND c.direction = 'in' AND c.amount = 25000`, d); got != 1 {
		t.Fatalf("سطرُ الصندوق لا يشير إلى سطر التسليم (%d)", got)
	}
	f.fiGreen(t, "FI-15.a")
}

// ── الإغلاقُ اليوميّ والنقص ───────────────────────────────────────────

func resetCashCloses(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	for _, q := range []string{
		`UPDATE office_cash_entries SET close_id = NULL`,
		`DELETE FROM office_cash_shortfalls`,
		`DELETE FROM office_cash_closes`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTREASURY_DailyCloseReviewAndShortfallLoss(t *testing.T) {
	f := newTreasuryFixture(t)
	resetCashCloses(t, f.pool)
	ctx := context.Background()
	book := f.one(t, `SELECT COALESCE(sum(CASE WHEN direction='in' THEN amount ELSE -amount END),0)
		FROM office_cash_entries`)
	counted := book - 3000 // ناقصٌ ثلاثة آلاف
	if counted < 0 {
		if _, err := f.pool.Exec(ctx, `INSERT INTO office_cash_entries (direction, amount, source, ref)
			VALUES ('in', $1, 'shortfall_found', gen_random_uuid()::text)`, -counted+10_000); err != nil {
			t.Fatal(err)
		}
		counted = 10_000 - 3000
		book = 10_000
	}
	w := f.call(f.a, "POST", "/admin/cashbox/closes", nil, `{"counted":`+itoa64(counted)+`}`,
		f.srv.handleCreateCashClose)
	if w.Code != http.StatusCreated {
		t.Fatalf("الإغلاق ردّ %d: %s", w.Code, w.Body)
	}
	var cr struct {
		Data struct {
			Close cashCloseRow `json:"close"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &cr)
	c := cr.Data.Close
	if c.Expected != book || c.Difference != -3000 {
		t.Fatalf("المتوقَّع %d والفرق %d — يُنتظر %d و-3000", c.Expected, c.Difference, book)
	}
	// **إغلاقٌ ثانٍ والأوّلُ معلَّق يُردّ.**
	if w := f.call(f.b, "POST", "/admin/cashbox/closes", nil, `{"counted":1}`, f.srv.handleCreateCashClose); w.Code != http.StatusConflict {
		t.Fatalf("إغلاقٌ ثانٍ ردّ %d", w.Code)
	}
	// **ومن عدّ لا يراجع نفسَه.**
	p := map[string]string{"id": c.ID}
	if w := f.call(f.a, "POST", "/x", p, `{}`, f.srv.handleDecideCashClose(true)); w.Code != http.StatusForbidden {
		t.Fatalf("مراجعةُ النفس ردّت %d", w.Code)
	}
	if w := f.call(f.b, "POST", "/x", p, `{}`, f.srv.handleDecideCashClose(true)); w.Code != http.StatusOK {
		t.Fatalf("المراجعة ردّت %d: %s", w.Code, w.Body)
	}
	var sid string
	if err := f.pool.QueryRow(ctx, `SELECT id::text FROM office_cash_shortfalls WHERE close_id = $1 AND amount = 3000`,
		c.ID).Scan(&sid); err != nil {
		t.Fatalf("لم يُسجَّل النقص: %v", err)
	}
	sp := map[string]string{"id": sid}
	// **قبل يومٍ لا يصير خسارة.**
	if w := f.call(f.b, "POST", "/x", sp, `{}`, f.srv.handleDecideShortfall(true)); w.Code != http.StatusConflict {
		t.Fatalf("خسارةٌ قبل يوم ردّت %d", w.Code)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE office_cash_shortfalls SET eligible_at = now() - interval '1 minute' WHERE id = $1`, sid); err != nil {
		t.Fatal(err)
	}
	// **ولا بيد من أغلق.**
	if w := f.call(f.a, "POST", "/x", sp, `{}`, f.srv.handleDecideShortfall(true)); w.Code != http.StatusForbidden {
		t.Fatalf("خسارةٌ بيد من أغلق ردّت %d", w.Code)
	}
	if w := f.call(f.b, "POST", "/x", sp, `{}`, f.srv.handleDecideShortfall(true)); w.Code != http.StatusOK {
		t.Fatalf("اعتمادُ الخسارة ردّ %d: %s", w.Code, w.Body)
	}
	if got := f.one(t, `SELECT amount FROM wallet_transactions WHERE kind = 'platform_expense' AND ref = $1 AND user_id = $2`,
		sid, f.treasury); got != -3000 {
		t.Fatalf("قيدُ الخسارة %d", got)
	}
	// **والإغلاقُ التالي يبدأ من المعدود.**
	pos, err := cashboxPosition(ctx, f.pool)
	if err != nil || pos.Opening != counted || pos.OpenLines != 0 {
		t.Fatalf("الموضعُ بعد الإغلاق: %+v %v", pos, err)
	}
	f.fiGreen(t, "FI-15")
}

// ── الموافقاتُ الموحّدة ───────────────────────────────────────────────

func TestTREASURY_ApprovalsPageBlocksProposer(t *testing.T) {
	f := newTreasuryFixture(t)
	u := testdb.NewUser(t, f.pool, "customer")
	var rid string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO wallet_requests (user_id, kind, amount, note, proposed_by)
		VALUES ($1, 'topup', 1000, 'اختبار', $2) RETURNING id::text`, u, f.a).Scan(&rid); err != nil {
		t.Fatal(err)
	}
	find := func(actor string) approvalItem {
		w := f.call(actor, "GET", "/admin/approvals", nil, "", f.srv.handleApprovals)
		var res struct {
			Data struct {
				Items []approvalItem `json:"items"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		for _, it := range res.Data.Items {
			if it.ID == rid {
				return it
			}
		}
		t.Fatalf("الطلبُ غائبٌ عن الموافقات: %s", w.Body)
		return approvalItem{}
	}
	if it := find(f.a); it.CanApprove || it.Key != "wallet_requests" {
		t.Fatalf("المقترحُ يستطيع الموافقة: %+v", it)
	}
	it := find(f.b)
	if !it.CanApprove || it.ApprovePath != "/api/v1/admin/wallet-requests/"+rid+"/approve" {
		t.Fatalf("غيرُ المقترح: %+v", it)
	}
	if _, err := f.pool.Exec(context.Background(), `UPDATE wallet_requests SET status = 'rejected' WHERE id = $1`, rid); err != nil {
		t.Fatal(err)
	}
}

// ── صحّةُ الدفتر وإضافةُ الأنواع ──────────────────────────────────────

func TestTREASURY_HealthRunsLedgerChecks(t *testing.T) {
	f := newTreasuryFixture(t)
	w := f.call(f.a, "GET", "/admin/treasury/health", nil, "", f.srv.handleTreasuryHealth)
	var res struct {
		Data struct {
			Checks []ledgerCheck `json:"checks"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	ids := map[string]bool{}
	for _, c := range res.Data.Checks {
		ids[c.ID] = true
	}
	for _, want := range []string{"KINDS", "FI-02.a", "FI-12.d", "FI-15.e"} {
		if !ids[want] {
			t.Fatalf("الفحصُ %s غائب (%d فحصاً)", want, len(res.Data.Checks))
		}
	}
}

func TestTREASURY_WalletKindsAddKeepsLiveKinds(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := fininv.SchemaKinds(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `SELECT wallet_kinds_add('zz_probe_kind')`); err != nil {
		t.Fatal(err)
	}
	after, err := fininv.SchemaKinds(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before)+1 {
		t.Fatalf("قبل %d وبعد %d", len(before), len(after))
	}
	if m, s := fininv.KindDrift(before); len(m)+len(s) != 0 {
		t.Fatalf("انحرافٌ قائم: %v %v", m, s)
	}
}
