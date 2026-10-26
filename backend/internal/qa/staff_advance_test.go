package qa

// ══════════════════════════════════════════════════════════════════════
// **السلفة** — قرارُ المالك ٢٠٢٦-١٠-٢٦
// ══════════════════════════════════════════════════════════════════════
//
// «إذا موظّف أو سائق أو مندوب طلب سلفة لازم نقدر نضيفها، فتصير سالب عليه
// وتنقطع من محفظته.» **المكتبُ يعطيه من الدرج، ومحفظتُه تنقص ولو صارت سالبة،
// وما يدخلها بعدُ يسدّ السالب — ولا سحبَ وهي سالبة.**

import (
	"testing"

	"github.com/google/uuid"
)

func advanceCash(t *testing.T, h *Harness, reqID string) (int64, string) {
	t.Helper()
	var amount int64
	var dir string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0), COALESCE(max(direction), '')
		FROM office_cash_entries WHERE source = 'advance' AND ref = $1`, reqID).Scan(&amount, &dir)
	return amount, dir
}

func TestADVANCE_DriverAdvanceGoesNegativeAndFillsBack(t *testing.T) {
	h := New(t)
	h.oldScaleMoney()
	f := h.Factory()
	tid := treasury(t, h)
	base := financialBaseline(t, h)
	fin := h.NewUser("finance")
	approver := h.NewUser("admin")
	drv := f.Driver()
	tBefore := treasuryBalance(t, h, tid)

	r := walletPropose(h, fin, drv.ID, map[string]any{"amount": 500, "kind": "advance", "note": "سلفة على الراتب"})
	if r.Code != 201 {
		t.Fatalf("اقتراحُ السلفة: %s", r)
	}
	reqID, _ := r.JSON()["request_id"].(string)
	var debit bool
	_ = h.Pool.QueryRow(ctxBG(), `SELECT debit FROM wallet_requests WHERE id = $1::uuid`, reqID).Scan(&debit)
	if !debit {
		t.Errorf("**السلفةُ خصمٌ دائماً** — والطلبُ كُتب إيداعاً")
	}
	if b := walletBalance(t, h, drv.ID); b != 0 {
		t.Fatalf("**الاقتراحُ حرّك المال**: %d", b)
	}
	if ok := h.POST("/api/v1/admin/wallet-requests/"+reqID+"/approve", approver.Token, map[string]any{}); ok.Code != 200 {
		t.Fatalf("الموافقة: %s", ok)
	}

	// (١) المحفظةُ −٥٠٠ · قيدُ `advance` بمرجع الطلب · سطرٌ خارجٌ من الدرج · والخزينةُ كما هي.
	if b := walletBalance(t, h, drv.ID); b != -500 {
		t.Errorf("**رصيدُ السائق بعد السلفة %d والمنتظَر −٥٠٠**", b)
	}
	if v := ledgerByRef(t, h, reqID, "advance"); v != -500 {
		t.Errorf("قيدُ السلفة بمرجع الطلب %d", v)
	}
	if amt, dir := advanceCash(t, h, reqID); amt != 500 || dir != "out" {
		t.Errorf("**السلفةُ لم تخرج من الدرج**: %d %q", amt, dir)
	}
	if after := treasuryBalance(t, h, tid); after != tBefore {
		t.Errorf("**السلفةُ لا تمسّ الخزينة** (كالسحب النقديّ): قبل %d بعد %d", tBefore, after)
	}

	// **ولا خصمٌ آخرُ ينزله تحت سالب السلفة.**
	if err := h.walletApply(drv.ID, -1, "adjustment", ""); err == nil {
		t.Errorf("**خصمٌ عاديٌّ أنزل المحفظةَ تحت سالب السلفة**")
	}

	// (٢) أجرُ توصيلٍ ١٠٠ يسدّ من السالب ⇒ −٤٠٠، وسقفُ السالب يتقلّص معه.
	f.CreditRef(drv.ID, 100, "driver_earning", uuid.NewString())
	var bal, owed int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT balance, advance_owed FROM wallets WHERE user_id = $1::uuid`,
		drv.ID).Scan(&bal, &owed)
	if bal != -400 || owed != 400 {
		t.Errorf("**بعد أجرِ ١٠٠**: الرصيد %d والسقف %d — والمنتظَر −٤٠٠ و٤٠٠", bal, owed)
	}

	// (٣) ولا سحبَ والرصيدُ سالب.
	if p := h.POST("/api/v1/me/payouts", drv.Token, map[string]any{"amount": 100}); p.Code < 400 {
		t.Errorf("**طلبُ سحبٍ مرّ والرصيدُ سالب**: %s", p)
	}

	// (٥) ثوابتُ المال: الرصيدُ = القيود · السالبُ بسلفة · الدرجُ يطابق.
	assertNewViolations(t, h, base, "FI-02.a", "FI-02.b", "FI-11.d", "FI-15.e", "FI-15.i")
}

// (٤) **ولا سلفةَ لزبونٍ ولا لصاحب متجر.**
func TestADVANCE_RejectedForCustomer(t *testing.T) {
	h := New(t)
	h.oldScaleMoney()
	f := h.Factory()
	treasury(t, h)
	fin := h.NewUser("finance")
	cust := f.NewUserWith("customer")
	r := walletPropose(h, fin, cust.ID, map[string]any{"amount": 500, "kind": "advance", "note": "سلفة"})
	if r.Code != 400 || r.Err() != "advance_not_eligible" {
		t.Fatalf("**سلفةٌ لزبونٍ قُبلت**: %s", r)
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallet_requests WHERE user_id = $1::uuid`, cust.ID).Scan(&n)
	if n != 0 {
		t.Errorf("كُتب %d طلباً لزبون", n)
	}
}

// **والموظّفُ والمندوبُ مؤهَّلان** — والملاحظةُ إلزاميّة.
func TestADVANCE_StaffAndRepEligible(t *testing.T) {
	h := New(t)
	h.oldScaleMoney()
	f := h.Factory()
	treasury(t, h)
	fin := h.NewUser("finance")
	for _, role := range []string{"sales", "operations"} {
		u := f.NewUserWith(role)
		if r := walletPropose(h, fin, u.ID, map[string]any{"amount": 300, "kind": "advance", "note": "سلفة"}); r.Code != 201 {
			t.Errorf("%s: سلفةٌ رُدّت: %s", role, r)
		}
		if r := walletPropose(h, fin, u.ID, map[string]any{"amount": 300, "kind": "advance", "note": " "}); r.Err() != "wallet_note_required" {
			t.Errorf("%s: سلفةٌ بلا سبب: %s", role, r)
		}
	}
}
