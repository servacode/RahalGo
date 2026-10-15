package server

// ══════════════════════════════════════════════════════════════════════
// **قراراتُ المالك في الديون** (٢٠٢٦-١٠-١٥) — ما يمرّ بالنزاعات والأهداف
// ══════════════════════════════════════════════════════════════════════
//
//	٢ · مكافأةُ الهدف لمندوبٍ عليه دين تسدّ الدينَ أوّلاً
//	٤ · خصمُ النزاع من المتاح لا الرصيد، والباقي دينٌ والنزاعُ يُحسم
//	· · والزبونُ لا طرفَ دينٍ له — باقيه يبقى نزاعاً مفتوحاً كما كان

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestDEBT_DisputeChargeTakesAvailableRestBecomesDebt **يُؤخذ المتاحُ ويصير الباقي ديناً.**
//
// متجرٌ رصيدُه ٣٬٠٠٠ منها ١٬٠٠٠ محجوزة، والنزاعُ ٥٬٠٠٠: **يُخصم ٢٬٠٠٠** (لا ٣٬٠٠٠)،
// **و٣٬٠٠٠ دينٌ على المتجر** بسبب «باقي مطالبة»، **والنزاعُ يُحسم**.
func TestDEBT_DisputeChargeTakesAvailableRestBecomesDebt(t *testing.T) {
	f := newLDFixture(t)
	ctx := context.Background()
	f.fund(t, f.owner, 3_000)
	if err := f.srv.wallet.ReserveTx(ctx, f.pool, f.owner, 1_000); err != nil {
		t.Fatalf("الحجز: %v", err)
	}
	t.Cleanup(func() { _ = f.srv.wallet.ReleaseTx(context.Background(), f.pool, f.owner, 1_000) })
	id := f.dispute(t, 5_000)
	res := f.propose(t, id, "charge")
	w := f.call(f.approver, http.MethodPost, "/x", map[string]string{"id": res}, `{}`,
		f.srv.handleDecideDisputeResolution(true))
	if w.Code != http.StatusOK {
		t.Fatalf("الموافقةُ ردّت %d: %s", w.Code, w.Body.String())
	}
	if st, _, rec := f.row(t, id); st != "settled" || rec != 2_000 {
		t.Fatalf("النزاع: %s · مسترَدٌّ %d — والمتوقَّع محسومٌ بمسترَدٍّ ٢٠٠٠ (المتاحُ لا الرصيد)", st, rec)
	}
	var balance, reserved int64
	if err := f.pool.QueryRow(ctx, `SELECT balance, reserved FROM wallets WHERE user_id = $1`,
		f.owner).Scan(&balance, &reserved); err != nil {
		t.Fatal(err)
	}
	if balance != 1_000 || reserved != 1_000 {
		t.Errorf("الرصيدُ %d والمحجوزُ %d — **والمحجوزُ لا يُمسّ**", balance, reserved)
	}
	var amount int64
	var cause, src string
	if err := f.pool.QueryRow(ctx, `
		SELECT amount - settled, cause, COALESCE(source_ref, '') FROM financial_obligations
		 WHERE party_kind = 'merchant' AND party_id = $1 AND closed_at IS NULL`, f.merchantID).
		Scan(&amount, &cause, &src); err != nil {
		t.Fatalf("**لا دينَ على المتجر بالباقي**: %v", err)
	}
	if amount != 3_000 || cause != "dispute_remainder" || src != id {
		t.Errorf("الدين: %d · %q · %q — والمتوقَّع ٣٠٠٠ باقي مطالبةٍ من النزاع %s", amount, cause, src, id)
	}
	var notices int
	_ = f.pool.QueryRow(ctx, `
		SELECT count(*) FROM notifications WHERE user_id = $1 AND title = 'صار عليك دين'`,
		f.owner).Scan(&notices)
	if notices != 1 {
		t.Errorf("إشعاراتُ نشأة الدين للمالك %d والمنتظَر واحد", notices)
	}
	f.ledgerClean(t)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM financial_obligations WHERE party_kind = 'merchant' AND party_id = $1`, f.merchantID)
	})
}

// TestDEBT_CustomerDisputeRemainderStaysOpen **الزبونُ لا أرباحَ له** — باقيه يبقى نزاعاً.
func TestDEBT_CustomerDisputeRemainderStaysOpen(t *testing.T) {
	f := newLDFixture(t)
	ctx := context.Background()
	customer := testdb.NewUser(t, f.pool, "customer")
	f.fund(t, customer, 1_000)
	var id string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO disputes (party_role, party_user_id, reason, amount)
		VALUES ('customer', $1, 'لم يدفع', 3000) RETURNING id::text`, customer).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM disputes WHERE id = $1`, id) })
	res := f.propose(t, id, "charge")
	if w := f.call(f.approver, http.MethodPost, "/x", map[string]string{"id": res}, `{}`,
		f.srv.handleDecideDisputeResolution(true)); w.Code != http.StatusOK {
		t.Fatalf("الموافقةُ ردّت %d: %s", w.Code, w.Body.String())
	}
	if st, _, rec := f.row(t, id); st != "open" || rec != 1_000 {
		t.Fatalf("النزاع: %s · %d — والمتوقَّع مفتوحٌ بمسترَدٍّ ١٠٠٠", st, rec)
	}
	var n int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM financial_obligations WHERE party_id = $1`, customer).Scan(&n)
	if n != 0 {
		t.Errorf("نشأ دينٌ على زبون (%d)", n)
	}
}

// TestDEBT_TargetRewardPaysRepDebtFirst **مكافأةُ الهدف تسدّ دينَ المندوب أوّلاً.**
func TestDEBT_TargetRewardPaysRepDebtFirst(t *testing.T) {
	f := newIncFixture(t)
	f.setTargets(t, map[string]string{"sales.monthly_target": "1", "sales.target_reward": "3000"})
	ctx := context.Background()
	base, err := fininv.Capture(ctx, f.pool, "FI-03.a", "FI-02.a", "FI-02.b", "FI-13.b", "FI-13.c")
	if err != nil {
		t.Fatal(err)
	}
	rep := f.kept(t, "sales")
	if _, err := obligations.Create(ctx, f.pool, obligations.PartyRep, rep, 1_000,
		obligations.CauseLegacy, "", nil); err != nil {
		t.Fatalf("دينُ المندوب: %v", err)
	}
	f.newStore(t, rep)
	paid, err := f.srv.incentives.GrantTargetIfReachedTx(ctx, f.pool, rep, "sales")
	if err != nil || paid != 3_000 {
		t.Fatalf("المكافأة: %d (%v)", paid, err)
	}
	var open, balance int64
	_ = f.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(amount - settled), 0) FROM financial_obligations
		 WHERE party_kind = 'rep' AND party_id = $1 AND closed_at IS NULL`, rep).Scan(&open)
	_ = f.pool.QueryRow(ctx, `SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1`, rep).Scan(&balance)
	if open != 0 || balance != 2_000 {
		t.Errorf("**بقي دينٌ %d ورصيدُه %d** — والمكافأةُ تسدّ الدينَ أوّلاً (المنتظَر ٠ و٢٠٠٠)", open, balance)
	}
	var notices int
	_ = f.pool.QueryRow(ctx, `
		SELECT count(*) FROM notifications WHERE user_id = $1 AND title = 'انقطع من دينك'`, rep).Scan(&notices)
	if notices != 1 {
		t.Errorf("إشعاراتُ الاقتطاع %d والمنتظَر واحد", notices)
	}
	vs, err := fininv.Run(ctx, f.pool, "FI-03.a", "FI-02.a", "FI-02.b", "FI-13.b", "FI-13.c")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range base.New(vs) {
		line := fmt.Sprint(v.Rows)
		if v.Check.ID == "FI-03.a" || strings.Contains(line, rep) || strings.Contains(line, f.treasury) {
			t.Errorf("**خرقُ دفتر**: %s — %s", v.Check.ID, line)
		}
	}
}
