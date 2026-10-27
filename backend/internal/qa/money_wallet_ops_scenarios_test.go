package qa

// ══════════════════════════════════════════════════════════════════════
// **عمليّاتُ المحفظة — سيناريوهاتُ المال من أوّلها لآخرها**
// ══════════════════════════════════════════════════════════════════════
//
// شحنٌ نقديٌّ (يدخل الدرج) · سلفةٌ (تخرج من الدرج والمحفظةُ سالبةٌ بحدّ السلفة،
// وما يدخل بعدها يسدّها، ولا سحبَ وهي سالبة) · تسويةٌ إيداعاً وخصماً (والطرفُ
// الآخرُ الخزينة) · تعويضٌ (من الخزينة) · سحبٌ نقداً أو حوالة، وارتدادُه.
//
// **وقرارُ المالك ٢٠٢٦-١٠-٢٧**: حركةُ المالك أو المدير تنفذ فوراً، **وطلبُ موظّف
// الماليّة ينتظر** (`approval.AutoApplies`). **وما فوق سقف التعويض لغير المالك يبقى
// معلَّقاً** ولو اقترحه مدير.
//
// كلُّ اختبارٍ يمشي بالموجّه الحقيقيّ، ويقيس المحفظةَ والخزينةَ والدرجَ معاً،
// **وثوابتُ الدفتر (`fininv`) لا يستجدّ فيها خرق.**

import (
	"testing"
)

// wopsDrawer **صافي ما دخل الدرج وخرج منه بمرجعٍ واحد** — الداخلُ موجبٌ والخارجُ سالب.
func wopsDrawer(t *testing.T, h *Harness, ref string) int64 {
	t.Helper()
	var v int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(CASE WHEN direction = 'in' THEN amount ELSE -amount END), 0)
		  FROM office_cash_entries WHERE ref = $1`, ref).Scan(&v); err != nil {
		t.Fatalf("الدرج: %v", err)
	}
	return v
}

// wopsWallet **الرصيدُ والمحجوزُ وما بقي من السلفة.**
func wopsWallet(t *testing.T, h *Harness, uid string) (balance, reserved, owed int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT balance      FROM wallets WHERE user_id = $1::uuid), 0),
		       COALESCE((SELECT reserved     FROM wallets WHERE user_id = $1::uuid), 0),
		       COALESCE((SELECT advance_owed FROM wallets WHERE user_id = $1::uuid), 0)`,
		uid).Scan(&balance, &reserved, &owed); err != nil {
		t.Fatalf("المحفظة: %v", err)
	}
	return
}

// wopsApplied **حركةُ مديرٍ نفذت فوراً** — ويُرجع معرّفَ طلبها.
func wopsApplied(t *testing.T, r Res, what string) string {
	t.Helper()
	if r.Code != 201 || r.JSON()["status"] != "approved" || r.JSON()["applied"] != true {
		t.Fatalf("**%s لم تنفذ فوراً**: %s", what, r)
	}
	id, _ := r.JSON()["request_id"].(string)
	if id == "" {
		t.Fatalf("%s بلا معرّف طلب: %s", what, r)
	}
	return id
}

// TestWALLETOPS_AdvanceLifecycleWithDrawerAndTreasury **السلفةُ من رصيدٍ موجب حتّى سدادها وسحبِ الباقي.**
//
// سائقٌ رصيدُه ١٠٠ (تسويةُ إيداعٍ من الخزينة) ⇒ سلفةُ ٥٠٠ نقداً من الدرج ⇒ −٤٠٠ وسقفُ
// السالب ٤٠٠ ⇒ لا خصمَ يُنزله أكثر ولا سحب ⇒ شحنٌ نقديٌّ ٦٠٠ ⇒ +٢٠٠ والسقفُ صفر ⇒ سحبُ
// ٢٠٠ يُحجز ⇒ سلفةٌ والسحبُ محجوزٌ تُردّ ⇒ صرفُ السحب نقداً ⇒ صفر.
//
// الدرج: −٥٠٠ +٦٠٠ −٢٠٠ = −١٠٠ · الخزينة: −١٠٠ (التسوية وحدَها؛ السلفةُ والشحنُ والسحبُ لا يمسّونها).
func TestWALLETOPS_AdvanceLifecycleWithDrawerAndTreasury(t *testing.T) {
	h := New(t)
	h.Setting("finance.manual_wallet_max", "5000")
	tid := treasury(t, h)
	base := financialBaseline(t, h)
	admin := h.NewUser("admin")
	drv := h.NewUser("driver")
	t0 := treasuryBalance(t, h, tid)

	// ── ١ · تسويةُ إيداعٍ ١٠٠ — من الخزينة إلى المحفظة ─────────────────
	adj := wopsApplied(t, walletPropose(h, admin, drv.ID,
		map[string]any{"amount": 100, "kind": "adjustment", "note": "أجر قديم"}), "تسويةُ المدير")
	if b, _, _ := wopsWallet(t, h, drv.ID); b != 100 {
		t.Fatalf("بعد التسوية: الرصيد %d والمنتظَر ١٠٠", b)
	}
	if v := ledgerByRef(t, h, adj, "platform_expense"); v != -100 {
		t.Errorf("**التسويةُ إيداعاً لم تُقيَّد على الخزينة** (%d)", v)
	}
	t1 := treasuryBalance(t, h, tid)
	if t0-t1 != 100 {
		t.Errorf("الخزينة: قبل %d بعد %d — والمنتظَر −١٠٠", t0, t1)
	}

	// ── ٢ · سلفةُ ٥٠٠ ⇒ −٤٠٠ والسقفُ ٤٠٠ · والدرجُ −٥٠٠ · والخزينةُ كما هي ──
	adv := wopsApplied(t, walletPropose(h, admin, drv.ID,
		map[string]any{"amount": 500, "kind": "advance", "note": "سلفة على الأجر"}), "سلفةُ المدير")
	if b, _, owed := wopsWallet(t, h, drv.ID); b != -400 || owed != 400 {
		t.Errorf("**بعد السلفة**: الرصيد %d والسقف %d — والمنتظَر −٤٠٠ و٤٠٠", b, owed)
	}
	if d := wopsDrawer(t, h, adv); d != -500 {
		t.Errorf("**السلفةُ لم تخرج من الدرج** (%d)", d)
	}
	if tt := treasuryBalance(t, h, tid); tt != t1 {
		t.Errorf("**السلفةُ مسّت الخزينة**: %d ← %d", t1, tt)
	}

	// ── ٣ · خصمٌ تحت سقف السالب يُردّ كلُّه — ولا طلبَ يبقى منه ─────────
	var nReq int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallet_requests WHERE user_id = $1::uuid`, drv.ID).Scan(&nReq)
	if r := walletPropose(h, admin, drv.ID,
		map[string]any{"amount": 1, "kind": "adjustment", "debit": true, "note": "خصم"}); r.Code < 400 {
		t.Errorf("**خصمٌ أنزل المحفظةَ تحت سالب السلفة**: %s", r)
	}
	var nAfter int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallet_requests WHERE user_id = $1::uuid`, drv.ID).Scan(&nAfter)
	if nAfter != nReq {
		t.Errorf("**خصمٌ مردودٌ ترك طلباً معلَّقاً**: %d ← %d", nReq, nAfter)
	}
	if b, _, _ := wopsWallet(t, h, drv.ID); b != -400 {
		t.Errorf("تبدّل الرصيدُ بخصمٍ مردود: %d", b)
	}
	// **ولا سحبَ والرصيدُ سالب.**
	if p := h.POST("/api/v1/me/payouts", drv.Token, map[string]any{"amount": 100}); p.Code < 400 {
		t.Errorf("**سحبٌ مرّ والرصيدُ سالب**: %s", p)
	}

	// ── ٤ · شحنٌ نقديٌّ ٦٠٠ يسدّ السالبَ أوّلاً ⇒ +٢٠٠ والسقفُ صفر ───────
	top := wopsApplied(t, walletPropose(h, admin, drv.ID,
		map[string]any{"amount": 600, "kind": "topup", "note": "سداد السلفة نقداً"}), "شحنُ المدير")
	if b, _, owed := wopsWallet(t, h, drv.ID); b != 200 || owed != 0 {
		t.Errorf("**بعد الشحن**: الرصيد %d والسقف %d — والمنتظَر ٢٠٠ وصفر", b, owed)
	}
	if d := wopsDrawer(t, h, top); d != 600 {
		t.Errorf("**الشحنُ لم يدخل الدرج** (%d)", d)
	}
	if tt := treasuryBalance(t, h, tid); tt != t1 {
		t.Errorf("**الشحنُ مسّ الخزينة**: %d ← %d", t1, tt)
	}

	// ── ٥ · سحبُ ٢٠٠ يُحجز · وسلفةٌ والسحبُ محجوزٌ تُردّ ──────────────────
	p := h.POST("/api/v1/me/payouts", drv.Token, map[string]any{"amount": 200})
	if p.Code != 201 {
		t.Fatalf("طلبُ السحب بعد السداد: %s", p)
	}
	pid, _ := p.JSON()["id"].(string)
	if b, res, _ := wopsWallet(t, h, drv.ID); b != 200 || res != 200 {
		t.Errorf("بعد طلب السحب: الرصيد %d المحجوز %d — والمنتظَر ٢٠٠ و٢٠٠", b, res)
	}
	if r := walletPropose(h, admin, drv.ID,
		map[string]any{"amount": 100, "kind": "advance", "note": "سلفة ثانية"}); r.Code < 400 {
		t.Errorf("**سلفةٌ مرّت ومالُه محجوزٌ لسحبٍ جارٍ**: %s", r)
	}
	if b, res, owed := wopsWallet(t, h, drv.ID); b != 200 || res != 200 || owed != 0 {
		t.Errorf("تبدّلت المحفظةُ بسلفةٍ مردودة: %d/%d/%d", b, res, owed)
	}

	// ── ٦ · صرفُ السحب نقداً ⇒ صفر · والدرجُ −٢٠٠ ───────────────────────
	if d := h.POST("/api/v1/admin/payouts/"+pid+"/decide", admin.Token,
		map[string]any{"status": "paid", "method": "cash", "decision": "صُرف نقداً"}); d.Code >= 400 {
		t.Fatalf("صرفُ السحب: %s", d)
	}
	if b, res, _ := wopsWallet(t, h, drv.ID); b != 0 || res != 0 {
		t.Errorf("بعد الصرف: %d/%d — والمنتظَر صفر", b, res)
	}
	if d := wopsDrawer(t, h, pid); d != -200 {
		t.Errorf("**السحبُ النقديُّ لم يخرج من الدرج** (%d)", d)
	}
	if net := wopsDrawer(t, h, adv) + wopsDrawer(t, h, top) + wopsDrawer(t, h, pid); net != -100 {
		t.Errorf("صافي الدرج لهذا السائق %d والمنتظَر −١٠٠", net)
	}
	if tt := treasuryBalance(t, h, tid); t0-tt != 100 {
		t.Errorf("الخزينةُ في آخر المشوار: %d ← %d — والمنتظَر −١٠٠ (التسوية وحدَها)", t0, tt)
	}
	assertNewViolations(t, h, base)
}

// TestWALLETOPS_FinanceDebitWaitsThenCreditsTreasury **خصمُ الماليّة ينتظر، وبعد الموافقة يعود للخزينة.**
//
// سائقٌ رصيدُه ١٠٠٠ ⇒ الماليّةُ تقترح خصمَ ٣٠٠ ⇒ لا يتحرّك شيء ⇒ المديرُ يوافق ⇒ ٧٠٠
// والخزينةُ +٣٠٠ (`platform_profit` بمرجع الطلب). واقتراحٌ ثانٍ يُرفض ⇒ لا مال.
func TestWALLETOPS_FinanceDebitWaitsThenCreditsTreasury(t *testing.T) {
	h := New(t)
	h.Setting("finance.manual_wallet_max", "5000")
	tid := treasury(t, h)
	base := financialBaseline(t, h)
	admin := h.NewUser("admin")
	fin := h.NewUser("finance")
	drv := h.NewUser("driver")

	wopsApplied(t, walletPropose(h, admin, drv.ID,
		map[string]any{"amount": 1000, "kind": "topup", "note": "نقد بالمكتب"}), "شحنُ المدير")
	t0 := treasuryBalance(t, h, tid)

	r := walletPropose(h, fin, drv.ID, map[string]any{"amount": 300, "kind": "adjustment", "debit": true, "note": "خصم غلط بالحساب"})
	if r.Code != 201 || r.JSON()["status"] != "pending" {
		t.Fatalf("**خصمُ الماليّة لم يبقَ معلَّقاً**: %s", r)
	}
	reqID, _ := r.JSON()["request_id"].(string)
	if b, _, _ := wopsWallet(t, h, drv.ID); b != 1000 {
		t.Fatalf("**تحرّك الرصيدُ قبل الموافقة**: %d", b)
	}
	if tt := treasuryBalance(t, h, tid); tt != t0 {
		t.Fatalf("**تحرّكت الخزينةُ قبل الموافقة**: %d ← %d", t0, tt)
	}
	if ok := h.POST("/api/v1/admin/wallet-requests/"+reqID+"/approve", admin.Token, map[string]any{}); ok.Code != 200 {
		t.Fatalf("موافقةُ المدير: %s", ok)
	}
	if b, _, _ := wopsWallet(t, h, drv.ID); b != 700 {
		t.Errorf("بعد الخصم: %d والمنتظَر ٧٠٠", b)
	}
	if v := ledgerByRef(t, h, reqID, "adjustment"); v != -300 {
		t.Errorf("قيدُ الخصم بمرجع الطلب %d", v)
	}
	if v := ledgerByRef(t, h, reqID, "platform_profit"); v != 300 {
		t.Errorf("**الخصمُ لم يعد إلى الخزينة** (%d)", v)
	}
	if tt := treasuryBalance(t, h, tid); tt-t0 != 300 {
		t.Errorf("الخزينة: %d ← %d — والمنتظَر +٣٠٠", t0, tt)
	}

	// **والرفضُ لا يحرّك مالاً.**
	r2 := walletPropose(h, fin, drv.ID, map[string]any{"amount": 200, "kind": "adjustment", "debit": true, "note": "خصم ثاني"})
	id2, _ := r2.JSON()["request_id"].(string)
	if rj := h.POST("/api/v1/admin/wallet-requests/"+id2+"/reject", admin.Token,
		map[string]any{"note": "ما في داعي"}); rj.Code != 200 {
		t.Fatalf("الرفض: %s", rj)
	}
	if b, _, _ := wopsWallet(t, h, drv.ID); b != 700 {
		t.Errorf("**الرفضُ حرّك الرصيد**: %d", b)
	}
	if tt := treasuryBalance(t, h, tid); tt-t0 != 300 {
		t.Errorf("**الرفضُ حرّك الخزينة**: %d", tt-t0)
	}
	assertNewViolations(t, h, base)
}

// TestWALLETOPS_CompensationAdminNowAboveCapWaits **تعويضُ السائق: المديرُ تحت السقف فوراً،
// وفوق السقف ينتظر المالك، والماليّةُ تنتظر دائماً.**
//
// سقفُ تعويض السائق ٤٠٠:
//
//	المديرُ ٣٠٠ ⇒ فوراً: السائق +٣٠٠ · الخزينة −٣٠٠ (`platform_expense` بمرجع الطلب)
//	المديرُ ٥٠٠ ⇒ معلَّق ⇒ المالكُ يوافق ⇒ السائق +٥٠٠ · الخزينة −٥٠٠
//	الماليّةُ ٢٠٠ ⇒ معلَّق ولا مال
func TestWALLETOPS_CompensationAdminNowAboveCapWaits(t *testing.T) {
	h := New(t)
	h.Setting("compensations.cap_driver", "400")
	tid := treasury(t, h)
	base := financialBaseline(t, h)
	admin := h.NewUser("admin")
	fin := h.NewUser("finance")
	owner := h.NewUser("owner_super_admin")
	item := h.NewItem(1000)

	propose := func(by *User, oid, drvID string, amount int64) Res {
		return h.POST("/api/v1/admin/compensations", by.Token, map[string]any{
			"order_id": oid, "user_id": drvID, "amount": amount,
			"fault": "platform", "reason": "other", "note": "تعويض فحص",
		})
	}
	reqOf := func(oid, drvID string) (id, status string, self bool) {
		t.Helper()
		if err := h.Pool.QueryRow(ctxBG(), `
			SELECT id::text, status, self_approved FROM driver_compensation_requests
			 WHERE order_id = $1::uuid AND driver_id = $2::uuid
			 ORDER BY created_at DESC LIMIT 1`, oid, drvID).Scan(&id, &status, &self); err != nil {
			t.Fatalf("طلبُ التعويض: %v", err)
		}
		return
	}

	// ── ١ · المديرُ تحت السقف ⇒ فوراً ────────────────────────────────────
	o1 := newOrderFor(t, h, item)
	d1 := h.driverOf(o1)
	tb := treasuryBalance(t, h, tid)
	r := propose(admin, o1, d1.ID, 300)
	if r.Code != 202 || r.JSON()["status"] != "approved" || r.JSON()["applied"] != true {
		t.Fatalf("**تعويضُ المدير تحت السقف لم ينفذ فوراً**: %s", r)
	}
	if b, _, _ := wopsWallet(t, h, d1.ID); b != 300 {
		t.Errorf("السائق بعد التعويض %d والمنتظَر ٣٠٠", b)
	}
	if v := ledgerByRef(t, h, o1, "platform_expense"); v != -300 {
		t.Errorf("**التعويضُ لم يخرج من الخزينة** بمرجع الطلب (%d)", v)
	}
	if tt := treasuryBalance(t, h, tid); tb-tt != 300 {
		t.Errorf("الخزينة: %d ← %d — والمنتظَر −٣٠٠", tb, tt)
	}
	if _, st, self := reqOf(o1, d1.ID); st != "approved" || !self {
		t.Errorf("طلبُ التعويض: %s · ذاتيّة=%v — والمنتظَر approved مع علامة", st, self)
	}

	// ── ٢ · المديرُ فوق السقف ⇒ معلَّق ⇒ المالكُ يوافق ───────────────────
	o2 := newOrderFor(t, h, item)
	d2 := h.driverOf(o2)
	tb = treasuryBalance(t, h, tid)
	r = propose(admin, o2, d2.ID, 500)
	if r.Code != 202 || r.JSON()["status"] != "pending" {
		t.Fatalf("**تعويضُ المدير فوق السقف لم يبقَ معلَّقاً**: %s", r)
	}
	if b, _, _ := wopsWallet(t, h, d2.ID); b != 0 {
		t.Errorf("**دُفع فوق السقف بلا المالك**: %d", b)
	}
	if tt := treasuryBalance(t, h, tid); tt != tb {
		t.Errorf("**تحرّكت الخزينةُ فوق السقف قبل المالك**: %d ← %d", tb, tt)
	}
	id2, _, _ := reqOf(o2, d2.ID)
	if ok := h.POST("/api/v1/admin/compensations/"+id2+"/approve", owner.Token,
		map[string]any{"amount": 500, "note": "موافقة المالك"}); ok.Code >= 400 {
		t.Fatalf("موافقةُ المالك فوق السقف: %s", ok)
	}
	if b, _, _ := wopsWallet(t, h, d2.ID); b != 500 {
		t.Errorf("السائقُ بعد موافقة المالك %d والمنتظَر ٥٠٠", b)
	}
	if tt := treasuryBalance(t, h, tid); tb-tt != 500 {
		t.Errorf("الخزينة: %d ← %d — والمنتظَر −٥٠٠", tb, tt)
	}

	// ── ٣ · الماليّةُ ⇒ معلَّق ولا مال ──────────────────────────────────
	o3 := newOrderFor(t, h, item)
	d3 := h.driverOf(o3)
	tb = treasuryBalance(t, h, tid)
	r = propose(fin, o3, d3.ID, 200)
	if r.Code != 202 || r.JSON()["status"] != "pending" {
		t.Fatalf("**تعويضُ الماليّة لم يبقَ معلَّقاً**: %s", r)
	}
	if b, _, _ := wopsWallet(t, h, d3.ID); b != 0 {
		t.Errorf("**دُفع تعويضُ الماليّة قبل الموافقة**: %d", b)
	}
	if tt := treasuryBalance(t, h, tid); tt != tb {
		t.Errorf("**تحرّكت الخزينةُ بطلب الماليّة**: %d ← %d", tb, tt)
	}
	assertNewViolations(t, h, base)
}

// TestWALLETOPS_CashPayoutReversedReturnsToDrawer **سحبٌ صُرف نقداً ثمّ ارتدّ يعود للمحفظة وللدرج.**
//
// مندوبٌ رصيدُه ٨٠٠ ⇒ سحبُ ٨٠٠ نقداً ⇒ المحفظة صفر والدرجُ −٨٠٠ ⇒ ارتدادٌ ⇒ المحفظة ٨٠٠
// (`payout_reversal`) والدرجُ +٨٠٠ (`payout_reversed`) ⇒ صافي الدرج صفر. والخزينةُ لا تتحرّك.
// **وسحبٌ ثانٍ يبدأ صرفُه ثمّ يفشل** ⇒ يُفكّ الحجزُ بلا خصمٍ ولا سطرٍ في الدرج.
func TestWALLETOPS_CashPayoutReversedReturnsToDrawer(t *testing.T) {
	h := New(t)
	h.Setting("finance.manual_wallet_max", "5000")
	tid := treasury(t, h)
	base := financialBaseline(t, h)
	admin := h.NewUser("admin")
	rep := h.NewUser("sales")

	wopsApplied(t, walletPropose(h, admin, rep.ID,
		map[string]any{"amount": 800, "kind": "topup", "note": "نقد بالمكتب"}), "شحنُ المدير")
	t0 := treasuryBalance(t, h, tid)

	p := h.POST("/api/v1/me/payouts", rep.Token, map[string]any{"amount": 800})
	if p.Code != 201 {
		t.Fatalf("طلبُ السحب: %s", p)
	}
	pid, _ := p.JSON()["id"].(string)
	decide := func(id string, body map[string]any) Res {
		return h.POST("/api/v1/admin/payouts/"+id+"/decide", admin.Token, body)
	}
	if d := decide(pid, map[string]any{"status": "paid", "method": "cash", "decision": "صُرف"}); d.Code >= 400 {
		t.Fatalf("الصرف: %s", d)
	}
	if b, res, _ := wopsWallet(t, h, rep.ID); b != 0 || res != 0 {
		t.Errorf("بعد الصرف: %d/%d", b, res)
	}
	if d := wopsDrawer(t, h, pid); d != -800 {
		t.Errorf("الدرجُ بعد الصرف %d والمنتظَر −٨٠٠", d)
	}
	if d := decide(pid, map[string]any{"status": "reversed", "decision": "رجّع المبلغ"}); d.Code >= 400 {
		t.Fatalf("الارتداد: %s", d)
	}
	if b, _, _ := wopsWallet(t, h, rep.ID); b != 800 {
		t.Errorf("**المالُ لم يعد للمحفظة بعد الارتداد**: %d", b)
	}
	if v := ledgerByRef(t, h, pid, "payout_reversal"); v != 800 {
		t.Errorf("قيدُ إرجاع السحب %d", v)
	}
	if d := wopsDrawer(t, h, pid); d != 0 {
		t.Errorf("**النقدُ المرتجعُ لم يعد للدرج** — الصافي %d", d)
	}
	var revIn int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		WHERE ref = $1 AND source = 'payout_reversed' AND direction = 'in'`, pid).Scan(&revIn)
	if revIn != 800 {
		t.Errorf("سطرُ «سحب ارتدّ» في الدرج %d", revIn)
	}

	// **ثانياً: بدأ الصرفُ ثمّ فشل ⇒ يُفكّ بلا خصم.**
	p2 := h.POST("/api/v1/me/payouts", rep.Token, map[string]any{"amount": 800})
	if p2.Code != 201 {
		t.Fatalf("السحبُ الثاني: %s", p2)
	}
	pid2, _ := p2.JSON()["id"].(string)
	if d := decide(pid2, map[string]any{"status": "processing"}); d.Code >= 400 {
		t.Fatalf("بدءُ الصرف: %s", d)
	}
	if b, res, _ := wopsWallet(t, h, rep.ID); b != 800 || res != 800 {
		t.Errorf("قيدَ الصرف: %d/%d — والمنتظَر ٨٠٠ محجوزة", b, res)
	}
	if d := decide(pid2, map[string]any{"status": "failed", "decision": "الحوالة ما وصلت"}); d.Code >= 400 {
		t.Fatalf("الفشل: %s", d)
	}
	if b, res, _ := wopsWallet(t, h, rep.ID); b != 800 || res != 0 {
		t.Errorf("بعد الفشل: %d/%d — والمنتظَر ٨٠٠ بلا حجز", b, res)
	}
	if d := wopsDrawer(t, h, pid2); d != 0 {
		t.Errorf("**سحبٌ فاشلٌ كتب في الدرج** (%d)", d)
	}
	if tt := treasuryBalance(t, h, tid); tt != t0 {
		t.Errorf("**السحبُ مسّ الخزينة**: %d ← %d", t0, tt)
	}
	assertNewViolations(t, h, base)
}
