package server

// **الدرجُ وكشفُ الطلب — أبوابٌ جديدة** (قراراتُ المالك ٢٠٢٦-١٠-١٥).
//
//	رصيدُ البداية   يصير المتوقَّعَ، ومرّةً واحدة
//	الإيداع         سطرٌ داخلٌ في الدرج ولا يمسّ رصيدَ الخزينة
//	التقرير         رصيدُ آخر الفترة = المتوقَّع
//	المراجعة        المالكُ وحدَه يرى زرَّ مراجعة إغلاقه (`can_approve`)
//	تفصيلُ الطلب    سطورُه تُجمع إلى مبلغ «ربح طلب #N»

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestTREASURY_OpeningBalanceFeedsExpected(t *testing.T) {
	f := newTreasuryFixture(t)
	resetCashCloses(t, f.pool)
	t.Cleanup(func() { resetCashCloses(t, f.pool) })
	ctx := context.Background()

	w := f.call(f.a, "POST", "/admin/cashbox/opening", nil, `{"counted":12345,"note":"عدُّ أوّل يوم"}`,
		f.srv.handleCashOpening)
	if w.Code != http.StatusCreated {
		t.Fatalf("رصيدُ البداية ردّ %d: %s", w.Code, w.Body)
	}
	pos, err := cashboxPosition(ctx, f.pool)
	if err != nil {
		t.Fatal(err)
	}
	if pos.Opening != 12345 || pos.Expected != 12345 || pos.OpenLines != 0 {
		t.Fatalf("المتوقَّعُ بعد رصيد البداية: %+v — يُنتظر 12345 بلا سطورٍ مفتوحة", pos)
	}
	// **مرّةً واحدة.**
	if w := f.call(f.b, "POST", "/admin/cashbox/opening", nil, `{"counted":1}`, f.srv.handleCashOpening); w.Code != http.StatusConflict {
		t.Fatalf("رصيدُ بدايةٍ ثانٍ ردّ %d", w.Code)
	}
	// **ولا يحجز يومَ الإغلاق اليوميّ.**
	if w := f.call(f.a, "POST", "/admin/cashbox/closes", nil, `{"counted":12345}`, f.srv.handleCreateCashClose); w.Code != http.StatusCreated {
		t.Fatalf("إغلاقُ اليوم بعد رصيد البداية ردّ %d: %s", w.Code, w.Body)
	}
}

func TestTREASURY_OwnerDepositIsDrawerInNotTreasury(t *testing.T) {
	f := newTreasuryFixture(t)
	resetCashCloses(t, f.pool)
	t.Cleanup(func() { resetCashCloses(t, f.pool) })
	ctx := context.Background()
	if w := f.call(f.a, "POST", "/admin/cashbox/opening", nil, `{"counted":0}`, f.srv.handleCashOpening); w.Code != http.StatusCreated {
		t.Fatalf("رصيدُ البداية ردّ %d: %s", w.Code, w.Body)
	}
	balBefore := f.one(t, `SELECT balance FROM wallets WHERE user_id = $1`, f.treasury)
	txBefore := f.one(t, `SELECT count(*) FROM wallet_transactions WHERE user_id = $1`, f.treasury)

	// **بلا سبب لا.**
	if w := f.call(f.a, "POST", "/admin/cashbox/deposits", nil, `{"amount":1000}`, f.srv.handleCashDeposit); w.Code != http.StatusBadRequest {
		t.Fatalf("إيداعٌ بلا سبب ردّ %d", w.Code)
	}
	w := f.call(f.a, "POST", "/admin/cashbox/deposits", nil, `{"amount":2500,"note":"من جيبي للفكّة"}`,
		f.srv.handleCashDeposit)
	if w.Code != http.StatusCreated {
		t.Fatalf("الإيداعُ ردّ %d: %s", w.Code, w.Body)
	}
	ref, _ := raw(t, w.Body.Bytes())["ref"].(string)
	if got := f.one(t, `SELECT amount FROM office_cash_entries
		WHERE source = 'owner_deposit' AND direction = 'in' AND ref = $1`, ref); got != 2500 {
		t.Fatalf("سطرُ الإيداع %d", got)
	}
	if got := f.one(t, `SELECT balance FROM wallets WHERE user_id = $1`, f.treasury); got != balBefore {
		t.Fatalf("الإيداعُ حرّك رصيدَ الخزينة: %d ← %d", balBefore, got)
	}
	if got := f.one(t, `SELECT count(*) FROM wallet_transactions WHERE user_id = $1`, f.treasury); got != txBefore {
		t.Fatal("الإيداعُ كتب قيداً في الخزينة — ليس ربحاً")
	}
	if got := f.one(t, `SELECT count(*) FROM audit_log
		WHERE action = 'finance.cash_owner_deposit' AND entity_id::text = $1`, ref); got != 1 {
		t.Fatal("الإيداعُ بلا أثرٍ في السجلّ")
	}
	pos, err := cashboxPosition(ctx, f.pool)
	if err != nil || pos.Expected != 2500 {
		t.Fatalf("المتوقَّعُ بعد الإيداع %d (%v) — يُنتظر 2500", pos.Expected, err)
	}
	// **وتقريرُ اليوم يقول الشيءَ نفسَه** — رصيدُ آخره هو المتوقَّع.
	rw := f.call(f.a, "GET", "/admin/cashbox/report?period=day", nil, "", f.srv.handleCashboxReport)
	rep := raw(t, rw.Body.Bytes())
	if jnum(rep["end_balance"]) != pos.Expected {
		t.Fatalf("رصيدُ آخر اليوم %d والمتوقَّع %d: %s", jnum(rep["end_balance"]), pos.Expected, rw.Body)
	}
	f.fiGreen(t, "FI-15")
}

func TestTREASURY_CashCloseCanApproveFlag(t *testing.T) {
	f := newTreasuryFixture(t)
	resetCashCloses(t, f.pool)
	t.Cleanup(func() { resetCashCloses(t, f.pool) })
	// **من الماليّة لا مدير** — إغلاقُ المدير يُعتمَد فوراً (قرارُ المالك ٢٠٢٦-١٠-٢٧)، فلا زرَّ يُعرض.
	clerk := testdb.NewUser(t, f.pool, "finance")
	if w := f.call(clerk, "POST", "/admin/cashbox/closes", nil, `{"counted":0}`, f.srv.handleCreateCashClose); w.Code != http.StatusCreated {
		t.Fatalf("الإغلاقُ ردّ %d: %s", w.Code, w.Body)
	}
	flag := func(actor string) any {
		w := f.call(actor, "GET", "/admin/cashbox", nil, "", f.srv.handleCashbox)
		pc, _ := raw(t, w.Body.Bytes())["pending_close"].(map[string]any)
		return pc["can_approve"]
	}
	// **قرارُ ٢٠٢٦-١٠-٢٢:** من أغلق يراجع نفسَه.
	if flag(clerk) != true {
		t.Fatal("من أغلق لا يرى زرَّ مراجعة نفسِه")
	}
	if flag(f.b) != true {
		t.Fatal("غيرُ من أغلق لا يرى زرَّ المراجعة")
	}
}

func TestTREASURY_OrderBreakdownSumsToRow(t *testing.T) {
	f := newTreasuryFixture(t)
	pf := newProfitsFx(t, "تفصيل-ربح-طلب")
	now := time.Now()
	order := pf.order(t, "delivered", now)
	// **الطلبُ في الفحص**: هامش ١٠٠٠ + عمولة ٩٠٠ + توصيل ٢٠٠٠ − سائق ١٨٠٠ − خصم ٥٠٠ = ١٦٠٠.
	// **والخزينةُ قيّدت ١٧٠٠** — فالفرقُ ١٠٠ سطرٌ ظاهر لا مخفيّ.
	pf.post(t, pf.treasury, 1000, "platform_profit", order, now)
	pf.post(t, pf.treasury, 700, "platform_profit", order, now)
	day := accountToday(t, f.srv) // يومُ الحساب يبدأ بفتح الدوام (٢٠٢٦-١٠-٢٦)
	w := f.call(f.a, "GET", "/admin/treasury/statement/orders/"+order+"?from="+day+"&to="+day,
		map[string]string{"id": order}, "", f.srv.handleTreasuryOrderBreakdown)
	if w.Code != http.StatusOK {
		t.Fatalf("التفصيلُ ردّ %d: %s", w.Code, w.Body)
	}
	res := raw(t, w.Body.Bytes())
	lines, _ := res["lines"].([]any)
	var sum, diff int64
	keys := map[string]int64{}
	for _, x := range lines {
		l := x.(map[string]any)
		keys[l["key"].(string)] = jnum(l["amount"])
		sum += jnum(l["amount"])
		if l["key"] == "diff" {
			diff = jnum(l["amount"])
		}
	}
	if jnum(res["total"]) != 1700 || sum != 1700 {
		t.Fatalf("مجموعُ التفصيل %d ومبلغُ السطر %d — يُنتظر 1700 كلاهما: %s", sum, jnum(res["total"]), w.Body)
	}
	if diff != 100 || keys["margin"] != 1000 || keys["commission"] != 900 ||
		keys["delivery_fee"] != 2000 || keys["driver_fee"] != -1800 || keys["discount"] != -500 {
		t.Fatalf("سطورُ التفصيل: %v", keys)
	}
}
