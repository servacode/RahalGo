package qa

// ══════════════════════════════════════════════════════════════════════
// **عائلةُ الديون — سيناريوهاتٌ ما كان لها فحصٌ يقرأ المالَ كلَّه**
// ══════════════════════════════════════════════════════════════════════
//
//	١ · عقوبةٌ يدويّةٌ أكبرُ من المتاح: يُؤخذ المتاحُ للخزينة والباقي دين،
//	    ثمّ مكافأتان يدويّتان تسدّانه (الأقدمُ أوّلاً) ويعود المقتطَعُ للخزينة
//	٢ · نزاعٌ على سائق: يُخصم المتاحُ ويصير الباقي ديناً، ثمّ دفعةٌ نقديّةٌ
//	    بالمكتب تدخل الدرجَ والخزينة، ثمّ شطبُ الباقي بلا قيدٍ في الدفتر
//	٣ · إغلاقُ الدرج بزيادة: لا نقصَ ولا قيدَ خزينة، والمعدودُ أساسُ التالي؛
//	    وإغلاقٌ بنقصٍ وُجد مالُه: سطرٌ داخلٌ ولا خسارة
//
// **وكلُّ فحصٍ يقرأ ثوابتَ الدفتر بعده** (`fininv.Capture` ثمّ `fininv.Run`).

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/incentives"
	"github.com/servacode/rahalgo/backend/internal/officecash"
)

// mdebtAvail **المتاحُ في محفظته** — الرصيدُ ناقصَ المحجوز.
func mdebtAvail(t *testing.T, h *Harness, uid string) int64 {
	t.Helper()
	var v int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT balance - reserved FROM wallets WHERE user_id = $1::uuid), 0)`,
		uid).Scan(&v); err != nil {
		t.Fatalf("قراءةُ المتاح: %v", err)
	}
	return v
}

// mdebtSum مجموعُ قيودِ نوعٍ لمحفظةٍ بمرجعٍ بعينه.
func mdebtSum(t *testing.T, h *Harness, uid, kind, ref string) int64 {
	t.Helper()
	var v int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE user_id = $1::uuid AND kind = $2 AND ref = $3`, uid, kind, ref).Scan(&v); err != nil {
		t.Fatalf("قراءةُ القيود: %v", err)
	}
	return v
}

func mdebtDriverDebtCol(t *testing.T, h *Harness, uid string) int64 {
	t.Helper()
	var v int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT driver_debt FROM users WHERE id = $1::uuid`, uid).
		Scan(&v); err != nil {
		t.Fatalf("صورةُ دين السائق: %v", err)
	}
	return v
}

func mdebtObligationBySource(t *testing.T, h *Harness, src string) (id, cause string, amount int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT id::text, cause, amount FROM financial_obligations WHERE source_ref = $1`, src).
		Scan(&id, &cause, &amount); err != nil {
		t.Fatalf("**لا دينَ من المصدر %s**: %v", src, err)
	}
	return
}

// mdebtDeliveredDriver **سائقٌ سلّم طلباً نقديّاً** — فله أجرٌ في محفظته.
func mdebtDeliveredDriver(t *testing.T, h *Harness) *User {
	t.Helper()
	h.Setting("delivery.fee", "1000")
	oid := newOrderFor(t, h, h.NewItem(1000))
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)
	return drv
}

// ══════════════════════════════════════════════════════════════════════
// **١ · عقوبةٌ أكبرُ من المتاح ثمّ مكافأتان تسدّان الباقي**
// ══════════════════════════════════════════════════════════════════════
//
// مثال: متاحُه A، العقوبةُ A+700 ⇒ −A من محفظته و+A للخزينة (`penalty`)، ودينٌ ٧٠٠.
// مكافأةٌ ٥٠٠ ⇒ +٥٠٠ ثمّ −٥٠٠ سدادٌ، والخزينةُ −٥٠٠ ثمّ +٥٠٠ ⇒ دينٌ ٢٠٠.
// مكافأةٌ ١٠٠٠ ⇒ يبقى له ٨٠٠، والخزينةُ صافيها −٨٠٠، والدينُ يُغلق.
func TestMDEBT_PenaltyPartialRestDebtRewardsPayIt(t *testing.T) {
	h := New(t)
	tid := treasury(t, h)
	base := financialBaseline(t, h, debtsFI...)
	admin := h.NewUser("admin")
	drv := mdebtDeliveredDriver(t, h)
	avail := mdebtAvail(t, h, drv.ID)
	if avail <= 0 {
		t.Skip("لا أجرَ في محفظة السائق")
	}
	inc := debtsIncentives(h)

	t0 := treasuryBalance(t, h, tid)
	pen, err := inc.Grant(ctxBG(), admin.ID, drv.ID, incentives.KindPenalty, avail+700, "تأخّر متكرّر")
	if err != nil {
		t.Fatalf("العقوبة: %v", err)
	}
	if pen.Amount != avail+700 {
		t.Errorf("العقوبةُ سُجّلت بـ%d والمنتظَر كاملةً %d", pen.Amount, avail+700)
	}
	if got := mdebtAvail(t, h, drv.ID); got != 0 {
		t.Errorf("بقي في محفظته %d — والمتاحُ كلُّه يُؤخذ", got)
	}
	if got := mdebtSum(t, h, drv.ID, "penalty", ""); got != -avail {
		t.Errorf("قيدُ العقوبة على السائق %d والمنتظَر −%d (المتاحُ وحدَه)", got, avail)
	}
	if d := treasuryBalance(t, h, tid) - t0; d != avail {
		t.Errorf("الخزينةُ تغيّرت %d والمنتظَر +%d (المأخوذُ وحدَه)", d, avail)
	}
	obID, cause, amount := mdebtObligationBySource(t, h, pen.ID)
	if cause != "penalty_remainder" || amount != 700 {
		t.Errorf("الدين: %q بـ%d — والمنتظَر باقي عقوبةٍ ٧٠٠", cause, amount)
	}
	if c := mdebtDriverDebtCol(t, h, drv.ID); c != 700 {
		t.Errorf("صورةُ دين السائق %d والمنتظَر ٧٠٠", c)
	}

	// ── مكافأةٌ ٥٠٠: تسدّ ٥٠٠ ويبقى ٢٠٠ ──
	t1 := treasuryBalance(t, h, tid)
	r1, err := inc.Grant(ctxBG(), admin.ID, drv.ID, incentives.KindReward, 500, "شغل منيح")
	if err != nil {
		t.Fatalf("المكافأةُ الأولى: %v", err)
	}
	if got := mdebtSum(t, h, drv.ID, "reward", r1.ID); got != 500 {
		t.Errorf("المكافأةُ قُيّدت %d والمنتظَر كاملةً ٥٠٠", got)
	}
	if got := mdebtSum(t, h, drv.ID, "adjustment", r1.ID); got != -500 {
		t.Errorf("سدادُ الدين من المكافأة %d والمنتظَر −٥٠٠", got)
	}
	if got := mdebtSum(t, h, tid, "platform_profit", r1.ID); got != 500 {
		t.Errorf("عاد للخزينة %d بمرجع المكافأة والمنتظَر ٥٠٠", got)
	}
	if d := treasuryBalance(t, h, tid) - t1; d != 0 {
		t.Errorf("صافي الخزينة بالمكافأة الأولى %d والمنتظَر صفر (−٥٠٠ +٥٠٠)", d)
	}
	if got := openDebt(t, h, "driver", drv.ID); got != 200 {
		t.Errorf("الدينُ الباقي %d والمنتظَر ٢٠٠", got)
	}

	// ── مكافأةٌ ١٠٠٠: تسدّ ٢٠٠ ويبقى له ٨٠٠ ──
	t2 := treasuryBalance(t, h, tid)
	if _, err := inc.Grant(ctxBG(), admin.ID, drv.ID, incentives.KindReward, 1000, "شغل منيح"); err != nil {
		t.Fatalf("المكافأةُ الثانية: %v", err)
	}
	if got := mdebtAvail(t, h, drv.ID); got != 800 {
		t.Errorf("محفظتُه %d والمنتظَر ٨٠٠ (ألفٌ ناقصَ دينٍ ٢٠٠)", got)
	}
	if d := treasuryBalance(t, h, tid) - t2; d != -800 {
		t.Errorf("صافي الخزينة بالمكافأة الثانية %d والمنتظَر −٨٠٠", d)
	}
	settled, closed, methods := obligationState(t, h, obID)
	if settled != 700 || !closed || len(methods) != 2 || methods[0] != "earning" || methods[1] != "earning" {
		t.Errorf("الدينُ بعد المكافأتين: مسدَّد %d مغلق %v طرق %v", settled, closed, methods)
	}
	if c := mdebtDriverDebtCol(t, h, drv.ID); c != 0 {
		t.Errorf("صورةُ دين السائق %d بعد السداد", c)
	}
	assertNewViolations(t, h, base, debtsFI...)
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · نزاعٌ على سائق ⇒ دين ⇒ دفعةٌ بالمكتب ⇒ شطبُ الباقي**
// ══════════════════════════════════════════════════════════════════════
//
// مثال: متاحُه A والمطالبةُ A+1500 ⇒ −A من محفظته و+A للخزينة، ودينٌ ١٥٠٠.
// دفع ١٠٠٠ بالمكتب ⇒ +١٠٠٠ للدرج و+١٠٠٠ للخزينة، ودينٌ ٥٠٠.
// شطبُ المدير ⇒ يُغلق بلا قيد، والخزينةُ لا تتحرّك.
func TestMDEBT_DriverDisputeDebtOfficeCashThenWriteoff(t *testing.T) {
	h := New(t)
	tid := treasury(t, h)
	ids := append(append([]string{}, debtsFI...), "FI-15")
	base := financialBaseline(t, h, ids...)
	admin := h.NewUser("admin")
	fin := h.NewUser("finance")
	drv := mdebtDeliveredDriver(t, h)
	avail := mdebtAvail(t, h, drv.ID)
	if avail <= 0 {
		t.Skip("لا أجرَ في محفظة السائق")
	}

	mk := h.POSTKey("/api/v1/admin/disputes", admin.Token, uniq("dsp"), map[string]any{
		"party_role": "driver", "party_id": drv.ID, "reason": "بضاعة ضاعت معه",
		"amount": avail + 1500,
	})
	if mk.Code != 201 {
		t.Fatalf("فتحُ النزاع: %s", mk)
	}
	did, _ := mk.JSON()["id"].(string)

	t0 := treasuryBalance(t, h, tid)
	pr := h.POST("/api/v1/admin/disputes/"+did+"/propose", fin.Token,
		map[string]any{"action": "charge", "note": "يتحمّلها السائق"})
	if pr.Code != 201 || pr.JSON()["status"] != "pending" {
		t.Fatalf("اقتراحُ الماليّة ينتظر ولا ينفُذ: %s", pr)
	}
	resID, _ := pr.JSON()["resolution_id"].(string)
	if ok := h.POST("/api/v1/admin/dispute-resolutions/"+resID+"/approve", admin.Token,
		map[string]any{}); ok.Code != 200 {
		t.Fatalf("موافقةُ الحسم: %s", ok)
	}
	var status string
	var recovered int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT status, recovered FROM disputes WHERE id = $1::uuid`, did).
		Scan(&status, &recovered); err != nil {
		t.Fatal(err)
	}
	if status != "settled" || recovered != avail {
		t.Errorf("النزاع: %s بمسترَدٍّ %d — والمنتظَر محسومٌ بمسترَدٍّ %d", status, recovered, avail)
	}
	if got := mdebtSum(t, h, drv.ID, "adjustment", did); got != -avail {
		t.Errorf("الخصمُ من السائق %d والمنتظَر −%d", got, avail)
	}
	if got := mdebtSum(t, h, tid, "platform_profit", did); got != avail {
		t.Errorf("عاد للخزينة %d بمرجع النزاع والمنتظَر %d", got, avail)
	}
	if d := treasuryBalance(t, h, tid) - t0; d != avail {
		t.Errorf("الخزينةُ تغيّرت %d والمنتظَر +%d", d, avail)
	}
	ob, cause, amount := mdebtObligationBySource(t, h, did)
	if cause != "dispute_remainder" || amount != 1500 {
		t.Errorf("الدين: %q بـ%d — والمنتظَر باقي مطالبةٍ ١٥٠٠", cause, amount)
	}

	// ── دفعةٌ نقديّةٌ ١٠٠٠ بالمكتب: الماليّةُ تقترح والمديرُ يوافق ──
	t1 := treasuryBalance(t, h, tid)
	oc := h.POST("/api/v1/admin/obligations/"+ob+"/office-cash", fin.Token,
		map[string]any{"amount": 1000, "note": "دفع السائق بالمكتب"})
	if oc.Code != 201 || oc.JSON()["status"] != "pending" {
		t.Fatalf("اقتراحُ الدفعة: %s", oc)
	}
	reqID, _ := oc.JSON()["request_id"].(string)
	if ok := h.POST("/api/v1/admin/obligation-requests/"+reqID+"/approve", admin.Token,
		map[string]any{}); ok.Code != 200 {
		t.Fatalf("موافقةُ الدفعة: %s", ok)
	}
	var cash int64
	var cashUser string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT amount, COALESCE(user_id::text, '') FROM office_cash_entries
		 WHERE source = 'obligation_cash' AND ref = $1 AND direction = 'in'`, reqID).
		Scan(&cash, &cashUser); err != nil {
		t.Fatalf("**النقدُ لم يدخل الدرج**: %v", err)
	}
	if cash != 1000 || cashUser != drv.ID {
		t.Errorf("سطرُ الدرج %d لصاحبه %q — والمنتظَر ١٠٠٠ للسائق", cash, cashUser)
	}
	if d := treasuryBalance(t, h, tid) - t1; d != 1000 {
		t.Errorf("الخزينةُ بالدفعة %d والمنتظَر +١٠٠٠", d)
	}
	if got := openDebt(t, h, "driver", drv.ID); got != 500 {
		t.Errorf("الدينُ بعد الدفعة %d والمنتظَر ٥٠٠", got)
	}

	// ── شطبُ المدير ينفُذ فوراً: يُغلق الباقي ولا تتحرّك الخزينة ──
	t2 := treasuryBalance(t, h, tid)
	wo := h.POST("/api/v1/admin/obligations/"+ob+"/write-off", admin.Token,
		map[string]any{"note": "السائق ترك الشغل"})
	if wo.Code != 201 || wo.JSON()["applied"] != true {
		t.Fatalf("شطبُ المدير لم ينفُذ فوراً: %s", wo)
	}
	settled, closed, methods := obligationState(t, h, ob)
	if settled != 1500 || !closed || len(methods) != 2 ||
		methods[0] != "office_cash" || methods[1] != "written_off" {
		t.Errorf("الدينُ بعد الشطب: مسدَّد %d مغلق %v طرق %v", settled, closed, methods)
	}
	if d := treasuryBalance(t, h, tid) - t2; d != 0 {
		t.Errorf("**الشطبُ حرّك الخزينة** %d — والخسارةُ وقعت يومَ نشأ الدين", d)
	}
	var woAmount int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT amount FROM obligation_requests
		 WHERE obligation_id = $1::uuid AND kind = 'write_off' AND status = 'approved'`, ob).
		Scan(&woAmount); err != nil || woAmount != 500 {
		t.Errorf("طلبُ الشطب المقبول بـ%d (%v) والمنتظَر ٥٠٠ — تقرؤه صفحةُ الأرباح", woAmount, err)
	}
	if c := mdebtDriverDebtCol(t, h, drv.ID); c != 0 {
		t.Errorf("صورةُ دين السائق %d بعد الشطب", c)
	}
	assertNewViolations(t, h, base, ids...)
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · إغلاقُ الدرج: زيادةٌ لا تصير شيئاً، ونقصٌ وُجد مالُه لا يصير خسارة**
// ══════════════════════════════════════════════════════════════════════

// mdebtResetCashCloses **يفكّ الإغلاقات كلَّها** — كما يفعل فحصُ الخزينة في `server`:
// إغلاقٌ يوميٌّ واحدٌ ليوم الحساب، فاختبارٌ سبقه اليومَ يمنع هذا.
func mdebtResetCashCloses(t *testing.T, h *Harness) {
	t.Helper()
	for _, q := range []string{
		`UPDATE office_cash_entries SET close_id = NULL`,
		`DELETE FROM office_cash_shortfalls`,
		`DELETE FROM office_cash_closes`,
	} {
		if _, err := h.Pool.Exec(ctxBG(), q); err != nil {
			t.Fatal(err)
		}
	}
}

// mdebtDrawerAtLeast **يضمن في الدرج نقداً** بإيداعٍ من جيب المالك (لا قيدَ له في الدفتر).
func mdebtDrawerAtLeast(t *testing.T, h *Harness, min int64) int64 {
	t.Helper()
	book, err := officecash.Book(ctxBG(), h.Pool)
	if err != nil {
		t.Fatal(err)
	}
	if book < min {
		if err := officecash.Record(ctxBG(), h.Pool, officecash.Entry{
			Direction: officecash.In, Amount: min - book, Source: officecash.SourceOwnerDeposit,
			Ref: uniq("deposit"), Note: "تمهيدُ الدرج",
		}); err != nil {
			t.Fatal(err)
		}
		book = min
	}
	return book
}

func TestMDEBT_CashCloseExcessAndFoundShortfall(t *testing.T) {
	h := New(t)
	tid := treasury(t, h)
	clerk := h.NewUser("finance")
	admin := h.NewUser("admin")
	ids := []string{"FI-12", "FI-15"}

	// ── أ · زيادةٌ ٥٠٠ ──
	mdebtResetCashCloses(t, h)
	book := mdebtDrawerAtLeast(t, h, 2000)
	base := financialBaseline(t, h, ids...)
	t0 := treasuryBalance(t, h, tid)
	r := h.POST("/api/v1/admin/cashbox/closes", clerk.Token,
		map[string]any{"counted": book + 500, "note": "عدّ المسا"})
	if r.Code != 201 || r.JSON()["applied"] != false {
		t.Fatalf("إغلاقُ الماليّة ينتظر مراجعة: %s", r)
	}
	c, _ := r.JSON()["close"].(map[string]any)
	cid, _ := c["id"].(string)
	if exp, _ := c["expected"].(float64); int64(exp) != book {
		t.Errorf("المتوقَّع %v والمنتظَر %d", c["expected"], book)
	}
	if diff, _ := c["difference"].(float64); int64(diff) != 500 {
		t.Errorf("الفرق %v والمنتظَر +٥٠٠", c["difference"])
	}
	if ok := h.POST("/api/v1/admin/cashbox/closes/"+cid+"/approve", admin.Token,
		map[string]any{}); ok.Code != 200 {
		t.Fatalf("مراجعةُ الإغلاق: %s", ok)
	}
	var shortfalls int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM office_cash_shortfalls WHERE close_id = $1::uuid`,
		cid).Scan(&shortfalls)
	if shortfalls != 0 {
		t.Errorf("**زيادةٌ سُجّلت نقصاً** (%d)", shortfalls)
	}
	if d := treasuryBalance(t, h, tid) - t0; d != 0 {
		t.Errorf("الزيادةُ حرّكت الخزينة %d", d)
	}
	var opening int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT counted FROM office_cash_closes WHERE status = 'approved'
		 ORDER BY decided_at DESC, created_at DESC LIMIT 1`).Scan(&opening)
	if opening != book+500 {
		t.Errorf("أساسُ الإغلاق التالي %d والمنتظَر المعدود %d", opening, book+500)
	}
	if again := h.POST("/api/v1/admin/cashbox/closes", clerk.Token,
		map[string]any{"counted": book}); again.Err() != "cash_close_day_done" {
		t.Errorf("إغلاقٌ يوميٌّ ثانٍ ليوم الحساب نفسِه: %s", again)
	}
	assertNewViolations(t, h, base, ids...)

	// ── ب · نقصٌ ٤٠٠ ثمّ وُجد المال ──
	mdebtResetCashCloses(t, h)
	book = mdebtDrawerAtLeast(t, h, 2000)
	base = financialBaseline(t, h, ids...)
	t1 := treasuryBalance(t, h, tid)
	r = h.POST("/api/v1/admin/cashbox/closes", clerk.Token, map[string]any{"counted": book - 400})
	if r.Code != 201 {
		t.Fatalf("الإغلاق: %s", r)
	}
	c, _ = r.JSON()["close"].(map[string]any)
	cid, _ = c["id"].(string)
	if ok := h.POST("/api/v1/admin/cashbox/closes/"+cid+"/approve", admin.Token,
		map[string]any{}); ok.Code != 200 {
		t.Fatalf("مراجعةُ الإغلاق: %s", ok)
	}
	var sid string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT id::text FROM office_cash_shortfalls
		 WHERE close_id = $1::uuid AND amount = 400 AND status = 'pending'`, cid).Scan(&sid); err != nil {
		t.Fatalf("**لم يُسجَّل النقص ٤٠٠**: %v", err)
	}
	if res := h.POST("/api/v1/admin/cashbox/shortfalls/"+sid+"/resolve", clerk.Token,
		map[string]any{"note": "لقيناها بدرج تاني"}); res.Code != 200 {
		t.Fatalf("حلُّ النقص: %s", res)
	}
	var found int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		 WHERE source = 'shortfall_found' AND ref = $1 AND direction = 'in'`, sid).Scan(&found)
	if found != 400 {
		t.Errorf("المالُ الموجود دخل الدرج %d والمنتظَر ٤٠٠", found)
	}
	if loss := h.POST("/api/v1/admin/cashbox/shortfalls/"+sid+"/approve", admin.Token,
		map[string]any{}); loss.Err() != "request_decided" {
		t.Errorf("**نقصٌ محلولٌ صار خسارة**: %s", loss)
	}
	if got := mdebtSum(t, h, tid, "platform_expense", sid); got != 0 {
		t.Errorf("قيدُ خسارةٍ لنقصٍ وُجد مالُه: %d", got)
	}
	if d := treasuryBalance(t, h, tid) - t1; d != 0 {
		t.Errorf("الخزينةُ تحرّكت %d بنقصٍ وُجد مالُه", d)
	}
	// **والمتوقَّعُ التالي يعود إلى ما كان** — المعدودُ + الموجود.
	var openIn int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(CASE WHEN direction = 'in' THEN amount ELSE -amount END), 0)
		  FROM office_cash_entries WHERE close_id IS NULL`).Scan(&openIn)
	if (book-400)+openIn != book {
		t.Errorf("المتوقَّعُ بعد الحلّ %d والمنتظَر %d", (book-400)+openIn, book)
	}
	assertNewViolations(t, h, base, ids...)
}
