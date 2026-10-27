package qa

import (
	"context"
	"strconv"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حركةُ المالك أو المدير تنفذ فوراً — والموافقةُ لطلب الموظّف وحدَه**
// ══════════════════════════════════════════════════════════════════════
//
// قرارُ المالك ٢٠٢٦-١٠-٢٧: «انا الادمن، وقت اقوم بحركة ما لازم يطلب مني موافقة.
// الموافقة فقط وقت موظف المالية يطلب. حركاتي لازم تتنفذ فوراً بدون موافقة».
//
// **والتنفيذُ بطريق الموافقة نفسِه** (`approval.AutoApplies` ثمّ بابُ الموافقة في
// المعاملة نفسِها) — فالقيدُ والسجلُّ وعلامةُ `self_approved` هي هي. **ويُقاس هنا**:
// الطلبُ مقبولٌ عند إنشائه، والمالُ تحرّك، والسجلُّ يقول «وافق على اقتراحه»،
// **وطلبُ الماليّة نفسُه يبقى معلَّقاً لا يتحرّك قبله شيء**، ولا خرقَ جديدٌ في الدفتر.

// auditDetailsFor **تفاصيلُ آخر سطر سجلٍّ لفعلٍ على كيان.**
func auditDetailsFor(t *testing.T, h *Harness, action, entityID string) string {
	t.Helper()
	var d string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT details::text FROM audit_log
		 WHERE action = $1 AND entity_id::text = $2
		 ORDER BY created_at DESC LIMIT 1`, action, entityID).Scan(&d); err != nil {
		t.Fatalf("سطرُ السجلّ %s: %v", action, err)
	}
	return d
}

func walletBalanceOf(t *testing.T, h *Harness, uid string) int64 {
	t.Helper()
	var v int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE((SELECT balance FROM wallets WHERE user_id = $1::uuid), 0)`,
		uid).Scan(&v)
	return v
}

// TestAPPLYNOW_AdminWalletTopupAndAdvance **شحنُ المدير وسلفتُه تنفذان فوراً، وشحنُ الماليّة ينتظر.**
func TestAPPLYNOW_AdminWalletTopupAndAdvance(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h)
	admin := h.NewUser("admin")
	fin := h.NewUser("finance")
	cust := h.Factory().NewUserWith("customer", Named("زبون تنفيذ فوري"))
	drv := h.NewUser("driver")
	const a = "/api/v1/admin"
	// **والمبالغُ تحت سقف الحركة اليدويّة** — السقفُ من الإعدادات لا رقمٌ ثابت.
	limit := h.Settings.GetInt(ctxBG(), "finance.manual_wallet_max")
	if limit < 3 {
		t.Skipf("سقفُ الحركة اليدويّة %d لا يتّسع للفحص", limit)
	}
	top, advAmt, small := limit, limit/2, limit/3

	// ── ١ · شحنُ المدير ─────────────────────────────────────────────────
	r := h.POSTKey(a+"/users/"+cust.ID+"/wallet", admin.Token, uniq("w"),
		map[string]any{"amount": top, "kind": "topup", "note": "نقد بالمكتب"})
	if r.Code != 201 {
		t.Fatalf("شحنُ المدير رُدّ: %s", r)
	}
	if r.JSON()["status"] != "approved" || r.JSON()["applied"] != true {
		t.Fatalf("**شحنُ المدير لم ينفذ فوراً**: %s", r)
	}
	reqID, _ := r.JSON()["request_id"].(string)
	if b := walletBalanceOf(t, h, cust.ID); b != top {
		t.Errorf("رصيدُ الزبون %d والمتوقَّع %d", b, top)
	}
	var status, decidedBy string
	var self bool
	if err := h.Pool.QueryRow(ctxBG(), `SELECT status, decided_by::text, self_approved
		FROM wallet_requests WHERE id = $1::uuid`, reqID).Scan(&status, &decidedBy, &self); err != nil {
		t.Fatal(err)
	}
	if status != "approved" || decidedBy != admin.ID || !self {
		t.Errorf("الطلب: %s · %s · ذاتيّة=%v", status, decidedBy, self)
	}
	var cash int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		WHERE ref = $1 AND direction = 'in'`, reqID).Scan(&cash)
	if cash != top {
		t.Errorf("**النقدُ لم يدخل صندوقَ المكتب** (%d)", cash)
	}
	if d := auditDetailsFor(t, h, "finance.wallet_request_approved", cust.ID); !strings.Contains(d, "proposer_approves_own") {
		t.Errorf("**السجلُّ لا يقول «وافق على اقتراحه»**: %s", d)
	}

	// ── ٢ · سلفةُ المدير ─────────────────────────────────────────────────
	adv := h.POSTKey(a+"/users/"+drv.ID+"/wallet", admin.Token, uniq("w"),
		map[string]any{"amount": advAmt, "kind": "advance", "note": "سلفة طارئة"})
	if adv.Code != 201 || adv.JSON()["status"] != "approved" {
		t.Fatalf("**سلفةُ المدير لم تنفذ فوراً**: %s", adv)
	}
	if b := walletBalanceOf(t, h, drv.ID); b != -advAmt {
		t.Errorf("رصيدُ السائق بعد السلفة %d والمتوقَّع %d", b, -advAmt)
	}

	// ── ٣ · شحنُ الماليّة ينتظر ─────────────────────────────────────────
	before := walletBalanceOf(t, h, cust.ID)
	p := h.POSTKey(a+"/users/"+cust.ID+"/wallet", fin.Token, uniq("w"),
		map[string]any{"amount": small, "kind": "topup", "note": "نقد من الموظف"})
	if p.Code != 201 || p.JSON()["status"] != "pending" || p.JSON()["applied"] != nil {
		t.Fatalf("**طلبُ الماليّة لم يبقَ معلَّقاً**: %s", p)
	}
	if b := walletBalanceOf(t, h, cust.ID); b != before {
		t.Errorf("**تحرّك الرصيدُ بطلب الماليّة قبل الموافقة**: %d ← %d", before, b)
	}
	pid, _ := p.JSON()["request_id"].(string)
	if ok := h.POST(a+"/wallet-requests/"+pid+"/approve", admin.Token, map[string]any{}); ok.Code != 200 {
		t.Fatalf("موافقةُ المدير على طلب الماليّة: %s", ok)
	}
	if b := walletBalanceOf(t, h, cust.ID); b != before+small {
		t.Errorf("بعد الموافقة %d والمتوقَّع %d", b, before+small)
	}
	assertNewViolations(t, h, base)
}

// TestAPPLYNOW_ExpenseAboveThreshold **مصروفُ المدير فوق السقف يُقيَّد فوراً، ومصروفُ الماليّة ينتظر.**
func TestAPPLYNOW_ExpenseAboveThreshold(t *testing.T) {
	h := New(t)
	tid := treasury(t, h)
	base := financialBaseline(t, h)
	admin := h.NewUser("admin")
	fin := h.NewUser("finance")
	var catID string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO expense_categories (name, sort_order, active)
		VALUES ($1, 1, true) RETURNING id::text`, uniq("بند فوري ")).Scan(&catID); err != nil {
		t.Fatalf("بندُ المصروف: %v", err)
	}
	receipt := func(by string) string {
		var id string
		if err := h.Pool.QueryRow(ctxBG(), `
			INSERT INTO media (kind, path, thumb_path, width, height, bytes, created_by)
			VALUES ('expense_receipt', 'test/apply-now.jpg', 'test/apply-now_t.jpg', 10, 10, 100, $1)
			RETURNING id::text`, by).Scan(&id); err != nil {
			t.Fatalf("إيصال: %v", err)
		}
		return id
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = h.Pool.Exec(ctx, `DELETE FROM expense_requests WHERE category_id = $1::uuid`, catID)
	})
	over := h.Settings.GetInt(ctxBG(), "finance.expense_approval_threshold") + 1

	before := treasuryBalance(t, h, tid)
	r := h.POST("/api/v1/admin/expenses", admin.Token, map[string]any{
		"category_id": catID, "amount": over, "note": "إيجار المدير", "receipt_media_id": receipt(admin.ID)})
	if r.Code != 201 || r.JSON()["status"] != "approved" || r.JSON()["applied"] != true {
		t.Fatalf("**مصروفُ المدير فوق السقف لم ينفذ فوراً**: %s", r)
	}
	expID, _ := r.JSON()["expense_id"].(string)
	if after := treasuryBalance(t, h, tid); before-after != over {
		t.Errorf("الخزينة: قبل %d بعد %d — والمتوقَّع −%d", before, after, over)
	}
	if d := auditDetailsFor(t, h, "finance.expense_request_approved", expID); !strings.Contains(d, "proposer_approves_own") {
		t.Errorf("**السجلُّ لا يقول «وافق على اقتراحه»**: %s", d)
	}

	mid := treasuryBalance(t, h, tid)
	p := h.POST("/api/v1/admin/expenses", fin.Token, map[string]any{
		"category_id": catID, "amount": over, "note": "إيجار الموظف", "receipt_media_id": receipt(fin.ID)})
	if p.Code != 201 || p.JSON()["status"] != "pending" {
		t.Fatalf("**مصروفُ الماليّة فوق السقف لم يبقَ معلَّقاً**: %s", p)
	}
	if after := treasuryBalance(t, h, tid); after != mid {
		t.Errorf("**نقصت الخزينةُ بطلب الماليّة قبل الموافقة**: %d ← %d", mid, after)
	}
	assertNewViolations(t, h, base)
}

// TestAPPLYNOW_ObligationOfficeCash **دفعُ دينٍ نقداً يسجّله المدير ينفذ فوراً، ودفعُ الماليّة ينتظر.**
func TestAPPLYNOW_ObligationOfficeCash(t *testing.T) {
	h := New(t)
	f := h.Factory()
	tid := treasury(t, h)
	base := financialBaseline(t, h)
	admin := h.NewUser("admin")
	fin := h.NewUser("finance")
	m := f.Merchant()
	ob := legacyDebt(t, h, "merchant", m.ID, 9000)
	before := treasuryBalance(t, h, tid)

	r := h.POST("/api/v1/admin/obligations/"+ob+"/office-cash", admin.Token,
		map[string]any{"amount": 4000, "note": "دفع بالمكتب"})
	if r.Code != 201 || r.JSON()["status"] != "approved" || r.JSON()["applied"] != true {
		t.Fatalf("**دفعُ المدير لم ينفذ فوراً**: %s", r)
	}
	if settled, _, methods := obligationState(t, h, ob); settled != 4000 || len(methods) != 1 {
		t.Errorf("الدين بعد دفع المدير: مسدَّد %d طرق %v", settled, methods)
	}
	if after := treasuryBalance(t, h, tid); after-before != 4000 {
		t.Errorf("الخزينة: قبل %d بعد %d — والمتوقَّع +4000", before, after)
	}
	if d := auditDetailsFor(t, h, "finance.obligation_request_approved", ob); !strings.Contains(d, "proposer_approves_own") {
		t.Errorf("**السجلُّ لا يقول «وافق على اقتراحه»**: %s", d)
	}

	p := h.POST("/api/v1/admin/obligations/"+ob+"/office-cash", fin.Token,
		map[string]any{"amount": 1000, "note": "دفع عند الموظف"})
	if p.Code != 201 || p.JSON()["status"] != "pending" {
		t.Fatalf("**دفعُ الماليّة لم يبقَ معلَّقاً**: %s", p)
	}
	if settled, _, _ := obligationState(t, h, ob); settled != 4000 {
		t.Errorf("**سُدّ الدينُ بطلب الماليّة قبل الموافقة** (%d)", settled)
	}
	assertNewViolations(t, h, base)
}

// TestAPPLYNOW_OwnFundsRuleUnchanged **مالُ الفاعل نفسِه لا يمسّه القرار**: مديرٌ طلب سحبَ
// ماله لا يقرّر فيه — وفي المنصّة غيرُه يملك القرار (`approval.Check` بـ`OwnFunds`).
func TestAPPLYNOW_OwnFundsRuleUnchanged(t *testing.T) {
	h := New(t)
	treasury(t, h)
	f := h.Factory()
	admin := h.NewUser("admin")
	_ = h.NewUser("admin") // **غيرُه يملك القرار** — فلا استثناءَ «المالكُ وحدَه».
	f.Credit(admin.ID, 5000, "topup")
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE wallets SET reserved = 2000 WHERE user_id = $1::uuid`, admin.ID); err != nil {
		t.Fatalf("حجزُ السحب: %v", err)
	}
	var pid string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO payout_requests (user_id, amount) VALUES ($1::uuid, 2000) RETURNING id::text`,
		admin.ID).Scan(&pid); err != nil {
		t.Fatalf("طلبُ السحب: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = h.Pool.Exec(ctx, `UPDATE wallets SET reserved = 0 WHERE user_id = $1::uuid`, admin.ID)
		_, _ = h.Pool.Exec(ctx, `DELETE FROM payout_requests WHERE id = $1::uuid`, pid)
	})
	r := h.POST("/api/v1/admin/payouts/"+pid+"/decide", admin.Token,
		map[string]any{"status": "paid", "decision": "سحب"})
	if r.Err() != "self_approve" {
		t.Errorf("**مديرٌ قرّر في سحب ماله**: %s", r)
	}
	if b := walletBalanceOf(t, h, admin.ID); b != 5000 {
		t.Errorf("تحرّك رصيدُ المدير: %s", strconv.FormatInt(b, 10))
	}
}
