package server

// **قراراتُ المالك ٢٠٢٦-١٠-١٥ — الخزينة** (ما يُقاس بأبوابٍ قائمة).
//
//	بطاقاتُ الربح      صافي اليوم والأسبوع والشهر ومن أوّل يوم = «صافي الربح» في صفحة الأرباح
//	المصروفُ الكاش     يخرج من الدرج بسطر `expense_cash`، وإلغاؤه يردّه، ومن برّا لا يمسّه
//	السالب             الدرجُ يصير سالباً بمصروفٍ كاش — ولا يُمنع
//	شحنُ المحفظة       بحثُه لا يردّ حسابَ الاحتباس النقديّ
//	ما ينتظر           مجموعُه لا يعدّ تعويضاً بلا مبلغٍ بعد، والتعويضُ يُوافَق من هنا
//	كشفُ الحساب        قيودُ ربح الطلب الواحد سطرٌ واحد
//
// **والقراءةُ من الجواب خاماً** (`map`) — فالاختبارُ يُبنى على الشيفرة القديمة ويسقط بحكمه
// لا بترجمته.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// raw **جسمُ الجواب خاماً** — `data` إن كان مغلَّفاً.
func raw(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("جوابٌ غيرُ مقروء: %v — %s", err, body)
	}
	if d, ok := env["data"].(map[string]any); ok {
		return d
	}
	return env
}

func jnum(v any) int64 {
	f, _ := v.(float64)
	return int64(f)
}

// ── ١ · بطاقاتُ صافي الربح ────────────────────────────────────────────

func TestTREASURY_ProfitCardsEqualProfitsNet(t *testing.T) {
	f := newTreasuryFixture(t)
	ctx := context.Background()
	// **ربحٌ اليومَ** — فلا يكون الصافي صفراً في كلّ مدًى صدفةً.
	if _, err := f.srv.wallet.Apply(ctx, f.treasury, 4321, "platform_profit", "", "بطاقات الربح", nil); err != nil {
		t.Fatal(err)
	}
	w := f.call(f.a, "GET", "/admin/treasury/overview", nil, "", f.srv.handleTreasuryOverview)
	if w.Code != http.StatusOK {
		t.Fatalf("النظرةُ ردّت %d: %s", w.Code, w.Body)
	}
	cards, _ := raw(t, w.Body.Bytes())["profit_cards"].([]any)
	if len(cards) != 4 {
		t.Fatalf("بطاقاتُ الربح %d — يُنتظر أربع (اليوم · الأسبوع · الشهر · من أوّل يوم)", len(cards))
	}
	want := []string{"day", "week", "month", "all"}
	for i, c := range cards {
		card := c.(map[string]any)
		if card["period"] != want[i] {
			t.Fatalf("البطاقةُ %d مدّتُها %v — يُنتظر %s", i, card["period"], want[i])
		}
		from, to := card["from"].(string), card["to"].(string)
		pw := f.call(f.a, "GET", "/admin/profits?from="+from+"&to="+to, nil, "", f.srv.handleProfits)
		if pw.Code != http.StatusOK {
			t.Fatalf("الأرباح ردّت %d: %s", pw.Code, pw.Body)
		}
		if got, net := jnum(card["net"]), jnum(raw(t, pw.Body.Bytes())["net"]); got != net {
			t.Fatalf("بطاقةُ %s = %d وصافي صفحة الأرباح للمدى نفسِه (%q→%q) = %d", want[i], got, from, to, net)
		}
	}
	// **والأسبوعُ آخرُ سبعة أيّام** — كزرّ «هذا الأسبوع» في صفحة الأرباح.
	wk := cards[1].(map[string]any)
	f0, _ := time.Parse("2006-01-02", wk["from"].(string))
	t0, _ := time.Parse("2006-01-02", wk["to"].(string))
	if t0.Sub(f0) != 6*24*time.Hour {
		t.Fatalf("الأسبوعُ %v→%v — يُنتظر سبعةُ أيّام", wk["from"], wk["to"])
	}
}

// ── ٢ · المصروفُ الكاش والدرج ────────────────────────────────────────

func newExpenseCategory(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO expense_categories (name) VALUES ('باب الدرج ' || gen_random_uuid()::text)
		RETURNING id::text`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func createdExpenseID(t *testing.T, body []byte) string {
	t.Helper()
	id, _ := raw(t, body)["id"].(string)
	if id == "" {
		t.Fatalf("لا معرّفَ للمصروف: %s", body)
	}
	return id
}

func TestTREASURY_CashExpenseLeavesDrawerAndVoidReturns(t *testing.T) {
	f := newTreasuryFixture(t)
	cat := newExpenseCategory(t, f.pool)

	// **كاشٌ من الدرج**: سطرٌ خارجٌ بمقداره ومرجعِه.
	w := f.call(f.a, "POST", "/admin/expenses", nil,
		`{"category_id":"`+cat+`","amount":7000,"note":"قرطاسية","paid_from":"cash"}`, f.srv.handleCreateExpense)
	if w.Code != http.StatusCreated {
		t.Fatalf("المصروفُ الكاش ردّ %d: %s", w.Code, w.Body)
	}
	cash := createdExpenseID(t, w.Body.Bytes())
	if got := f.one(t, `SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		WHERE source = 'expense_cash' AND direction = 'out' AND ref = $1`, cash); got != 7000 {
		t.Fatalf("سطرُ الدرج للمصروف الكاش %d — يُنتظر 7000 خارجاً", got)
	}

	// **وإلغاؤه يردّه** — بيد غير من سجّل.
	vw := f.call(f.b, "POST", "/admin/expenses/"+cash+"/void", map[string]string{"id": cash},
		`{"reason":"سُجّل خطأً"}`, f.srv.handleVoidExpense)
	if vw.Code != http.StatusOK {
		t.Fatalf("الإلغاءُ ردّ %d: %s", vw.Code, vw.Body)
	}
	if got := f.one(t, `SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		WHERE source = 'expense_cash_void' AND direction = 'in' AND ref = $1`, cash); got != 7000 {
		t.Fatalf("سطرُ ردّ الإلغاء %d — يُنتظر 7000 داخلاً", got)
	}

	// **ومن برّا الدرج لا يمسّه.**
	w = f.call(f.a, "POST", "/admin/expenses", nil,
		`{"category_id":"`+cat+`","amount":3000,"note":"حوالة","paid_from":"outside"}`, f.srv.handleCreateExpense)
	if w.Code != http.StatusCreated {
		t.Fatalf("المصروفُ من برّا ردّ %d: %s", w.Code, w.Body)
	}
	outside := createdExpenseID(t, w.Body.Bytes())
	if got := f.one(t, `SELECT count(*) FROM office_cash_entries WHERE ref = $1`, outside); got != 0 {
		t.Fatalf("مصروفٌ من برّا الدرج كتب %d سطراً في الدرج", got)
	}
	if got := f.one(t, `SELECT count(*) FROM expenses WHERE id = $1 AND paid_from = 'outside'`, outside); got != 1 {
		t.Fatal("الطريقةُ لم تُحفظ على صفّ المصروف")
	}
	f.fiGreen(t, "FI-15")
}

func TestTREASURY_DrawerGoesNegativeAfterCashExpense(t *testing.T) {
	f := newTreasuryFixture(t)
	resetCashCloses(t, f.pool)
	ctx := context.Background()
	// **درجٌ عُدّ صفراً** — إغلاقٌ مُعتمَدٌ يبتلع ما قبله.
	var cid string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO office_cash_closes (day, opening, expected, counted, difference, amount,
		                                status, proposed_by, decided_by, decided_at)
		VALUES ('2000-01-01', 0, 0, 0, 0, 0, 'approved', $1, $2, now()) RETURNING id::text`, f.a, f.b).
		Scan(&cid); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE office_cash_entries SET close_id = $1 WHERE close_id IS NULL`, cid); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resetCashCloses(t, f.pool) })
	cat := newExpenseCategory(t, f.pool)
	w := f.call(f.a, "POST", "/admin/expenses", nil,
		`{"category_id":"`+cat+`","amount":5000,"note":"كهرباء","paid_from":"cash"}`, f.srv.handleCreateExpense)
	if w.Code != http.StatusCreated {
		t.Fatalf("مصروفٌ كاشٌ من درجٍ فارغ ردّ %d — السالبُ مسموح: %s", w.Code, w.Body)
	}
	pos, err := cashboxPosition(ctx, f.pool)
	if err != nil {
		t.Fatal(err)
	}
	if pos.Expected != -5000 {
		t.Fatalf("المتوقَّعُ بعد مصروفٍ كاشٍ من درجٍ فارغ %d — يُنتظر -5000", pos.Expected)
	}
}

// ── ٣ · بحثُ شحن المحفظة ──────────────────────────────────────────────

func TestTREASURY_WalletLookupExcludesCashHolding(t *testing.T) {
	f := newTreasuryFixture(t)
	ctx := context.Background()
	var holder string
	err := f.pool.QueryRow(ctx, `SELECT user_id::text FROM wallets WHERE is_cash_holding LIMIT 1`).Scan(&holder)
	if err != nil {
		holder = testdb.NewUser(t, f.pool, "admin")
		if _, err := f.pool.Exec(ctx, `
			INSERT INTO wallets (user_id, balance, is_cash_holding) VALUES ($1, 0, true)
			ON CONFLICT (user_id) DO UPDATE SET is_cash_holding = true`, holder); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = f.pool.Exec(context.Background(), `UPDATE wallets SET is_cash_holding = false WHERE user_id = $1`, holder)
		})
	}
	var oldName *string
	_ = f.pool.QueryRow(ctx, `SELECT full_name FROM users WHERE id = $1`, holder).Scan(&oldName)
	marker := fmt.Sprintf("احتباس%d", time.Now().UnixNano())
	if _, err := f.pool.Exec(ctx, `UPDATE users SET full_name = $2 WHERE id = $1`, holder, marker); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `UPDATE users SET full_name = $2 WHERE id = $1`, holder, oldName)
	})
	// **وحسابٌ عاديٌّ بالاسم نفسِه يُردّ** — فالبحثُ يعمل والاستبعادُ للنظام وحدَه.
	person := testdb.NewUser(t, f.pool, "customer")
	if _, err := f.pool.Exec(ctx, `UPDATE users SET full_name = $2 WHERE id = $1`, person, marker+" زبون"); err != nil {
		t.Fatal(err)
	}
	w := f.call(f.a, "GET", "/admin/treasury/wallet-lookup?q="+marker, nil, "", f.srv.handleWalletLookup)
	accounts, _ := raw(t, w.Body.Bytes())["accounts"].([]any)
	seenPerson := false
	for _, a := range accounts {
		id := a.(map[string]any)["id"]
		if id == holder {
			t.Fatalf("بحثُ الشحن ردّ حسابَ الاحتباس النقديّ: %s", w.Body)
		}
		if id == person {
			seenPerson = true
		}
	}
	if !seenPerson {
		t.Fatalf("البحثُ لم يردّ الحسابَ العاديّ: %s", w.Body)
	}
}

// ── ٤ · ما ينتظر قراراً ───────────────────────────────────────────────

func TestTREASURY_ApprovalsTotalExcludesUndecidedCompensation(t *testing.T) {
	f := newTreasuryFixture(t)
	pf := newProfitsFx(t, "موافقات-تعويض")
	ctx := context.Background()
	order := pf.order(t, "failed", time.Now())
	d2 := testdb.NewUser(t, f.pool, "driver")
	var zero, decided string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO driver_compensation_requests (order_id, driver_id, fault, suggested_amount)
		VALUES ($1, $2, 'customer', 0) RETURNING id::text`, order, pf.driver).Scan(&zero); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO driver_compensation_requests (order_id, driver_id, fault, suggested_amount)
		VALUES ($1, $2, 'customer', 3000) RETURNING id::text`, order, d2).Scan(&decided); err != nil {
		t.Fatal(err)
	}
	w := f.call(f.b, "GET", "/admin/approvals", nil, "", f.srv.handleApprovals)
	res := raw(t, w.Body.Bytes())
	items, _ := res["items"].([]any)
	var sum int64
	found := map[string]map[string]any{}
	for _, x := range items {
		it := x.(map[string]any)
		if it["amount_undecided"] != true {
			sum += jnum(it["amount"])
		}
		if id, _ := it["id"].(string); id == zero || id == decided {
			found[id] = it
		}
	}
	if len(found) != 2 {
		t.Fatalf("التعويضان غائبان عن الموافقات: %s", w.Body)
	}
	if found[zero]["amount_undecided"] != true || found[decided]["amount_undecided"] == true {
		t.Fatalf("المقترحُ صفراً ليس «بلا مبلغ بعد»: %v · %v", found[zero], found[decided])
	}
	if jnum(res["undecided_count"]) < 1 {
		t.Fatalf("عددُ ما بلا مبلغٍ بعد %v", res["undecided_count"])
	}
	if jnum(res["total"]) != sum {
		t.Fatalf("مجموعُ ما ينتظر %d — ومجموعُ ما قُرّر مبلغُه %d", jnum(res["total"]), sum)
	}
	// **ويُوافَق عليه من هنا** — ببابه نفسِه وبمبلغٍ يُعدَّل.
	it := found[decided]
	if it["approve_path"] != "/api/v1/admin/compensations/"+decided+"/approve" || it["amount_editable"] != true {
		t.Fatalf("التعويضُ لا يُوافَق من الموافقات: %v", it)
	}
}

// ── ٥ · كشفُ الحساب: «ربح طلب #N» ─────────────────────────────────────

func TestTREASURY_StatementGroupsOrderProfit(t *testing.T) {
	f := newTreasuryFixture(t)
	pf := newProfitsFx(t, "كشف-ربح-طلب")
	now := time.Now()
	order := pf.order(t, "delivered", now)
	pf.post(t, pf.treasury, 900, "platform_profit", order, now)
	pf.post(t, pf.treasury, 300, "platform_profit", order, now)
	day := now.In(damascusLoc()).Format("2006-01-02")
	w := f.call(f.a, "GET", "/admin/treasury/statement?from="+day+"&to="+day+"&kind=platform_profit&per_page=200",
		nil, "", f.srv.handleTreasuryStatement)
	lines, _ := raw(t, w.Body.Bytes())["lines"].([]any)
	var rows []map[string]any
	for _, x := range lines {
		l := x.(map[string]any)
		if l["ref"] == order {
			rows = append(rows, l)
		}
	}
	if len(rows) != 1 {
		t.Fatalf("قيودُ ربح الطلب %d سطراً — يُنتظر سطرٌ واحد: %v", len(rows), rows)
	}
	if r := rows[0]; jnum(r["amount"]) != 1200 || jnum(r["entries"]) != 2 || r["grouped"] != true ||
		!strings.EqualFold(fmt.Sprint(r["order_id"]), order) {
		t.Fatalf("سطرُ «ربح طلب»: %v", r)
	}
}
