package qa

// ══════════════════════════════════════════════════════════════════════
// **الطلبُ العاديُّ المدفوعُ نقداً — السلسلةُ كاملةً من الزبون إلى درج المكتب**
// ══════════════════════════════════════════════════════════════════════
//
// ثلاثُ فجواتٍ لم يمسكها اختبارٌ قائم:
//
//  1. **السلسلةُ كلُّها بأرقامٍ فيها تقريبٌ وهامشٌ وحصّةُ توصيل**: بضاعةٌ ١٠٠٥ (عمولةُ ١٠٪
//     = ١٠٠٫٥ ← ١٠١) · هامشٌ ١٠٠ · أجرةٌ ٥٠٠ وحصّةُ المنصّة ١٠٪ (أجرُ السائق ٤٥٠) · مندوبٌ
//     ١٠٪ من (١٠١ + ١٠٠) = ٢٠ · ثمّ يسلّم السائقُ صندوقَه للمكتب — **ومحوُ العمولة بعدها يُرفض.**
//  2. **محوُ العمولة والمندوبُ سحب رصيدَه** ⇒ نصيبُه دَينٌ عليه (`waived_rep_commission`).
//  3. **محوُ العمولة على طلبٍ من متجرين «نقداً» بنسبتين** ⇒ لكلّ متجرٍ عمولتُه.

import "testing"

// coFamilies ثوابتُ المال التي يمسّها الطلبُ النقديّ ومحوُ العمولة والدَّين والسحب.
var coFamilies = append(append([]string{}, waiverFamilies...), "FI-09", "FI-11")

func coSetCash(t *testing.T, h *Harness, merchantID string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET settlement_method = 'cash' WHERE id = $1::uuid`,
		merchantID); err != nil {
		t.Fatal(err)
	}
}

// coOrderMoney **أرقامُ الطلب كما كُتبت عند الإنشاء.**
func coOrderMoney(t *testing.T, h *Harness, oid string) (cashDue, walletPaid, deliveryFee, driverFee, commission int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT cash_due, wallet_paid, delivery_fee, driver_fee, platform_commission
		  FROM orders WHERE id = $1::uuid`, oid).
		Scan(&cashDue, &walletPaid, &deliveryFee, &driverFee, &commission); err != nil {
		t.Fatalf("قراءةُ الطلب: %v", err)
	}
	return
}

// coSettlement **صفُّ تسوية متجرٍ في طلب.**
func coSettlement(t *testing.T, h *Harness, oid, merchantID string) (amount int64, state, via string, waived int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT amount, state, COALESCE(paid_via, ''), commission_waived
		  FROM merchant_settlements WHERE order_id = $1::uuid AND merchant_id = $2::uuid`,
		oid, merchantID).Scan(&amount, &state, &via, &waived); err != nil {
		t.Fatalf("صفُّ التسوية: %v", err)
	}
	return
}

// TestCASHORD_FullChainMarginShareRepDrawer **طلبٌ نقديٌّ من متجرٍ «نقداً» — كلُّ طرفٍ بقرشه.**
//
//	الزبون     يدفع نقداً ١٦٠٥ = ١١٠٥ بضاعة (١٠٠٥ + هامش ١٠٠) + ٥٠٠ توصيل — ومحفظتُه لا تُمسّ
//	المتجر     يقبض من السائق عند الاستلام ٩٠٤ = ١٠٠٥ − ١٠١ — ولا شيءَ في محفظته
//	السائق     محفظتُه +٤٥٠ أجراً · صندوقُه ٧٠١ = ١٦٠٥ − ٩٠٤ (يدين بها للمكتب)
//	المندوب    ٢٠ = ١٠٪ من (عمولة ١٠١ + هامش ١٠٠) — وحصّةُ التوصيل ليست في قاعدته
//	الخزينة    ٢٣١ = ١٠١ + ١٠٠ + ٥٠ − ٢٠
//	الدرج      لا سلفةَ تُردّ · ثمّ يسلّم السائقُ ٧٠١ ⇒ داخلٌ ٧٠١ وصندوقُه صفر
func TestCASHORD_FullChainMarginShareRepDrawer(t *testing.T) {
	h := New(t)
	h.oldScaleMoney()
	treasury(t, h)
	waiverSettings(h)
	h.Setting("pricing.margin_fixed", "100")
	h.Setting("delivery.share_standard", "10")
	h.Setting("delivery.share_standard_mode", `"percent"`)
	base := financialBaseline(t, h, coFamilies...)
	fin := h.NewUser("finance")

	f := h.Factory()
	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID), Commission(10))
	coSetCash(t, h, m.ID)
	item := h.NewItemFor(m, 1005)
	cust := h.Customer()
	custBefore := walletBalance(t, h, cust.ID)

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("cashord"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	cashDue, walletPaid, fee, driverFee, _ := coOrderMoney(t, h, oid)
	if cashDue != 1605 || walletPaid != 0 || fee != 500 || driverFee != 450 {
		t.Fatalf("الطلبُ عند الإنشاء: نقد %d محفظة %d أجرة %d أجرُ السائق %d — يُنتظر ١٦٠٥ و٠ و٥٠٠ و٤٥٠",
			cashDue, walletPaid, fee, driverFee)
	}

	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	// المتجر: ١٠٠٥ − ١٠١ (النصفُ يصعد) = ٩٠٤، دفعها السائقُ عند الاستلام.
	if _, _, _, _, commission := coOrderMoney(t, h, oid); commission != 101 {
		t.Errorf("**عمولةُ المنصّة %d — يُنتظر ١٠١** (١٠٪ من ١٠٠٥ = ١٠٠٫٥ والنصفُ يصعد)", commission)
	}
	amount, state, via, _ := coSettlement(t, h, oid, m.ID)
	if amount != 904 || state != "cash_paid" || via != "driver" {
		t.Errorf("تسويةُ المتجر: %d %s %s — يُنتظر ٩٠٤ cash_paid driver", amount, state, via)
	}
	if got := ledgerByRef(t, h, oid, "merchant_cash_accrued"); got != 904 {
		t.Errorf("الاحتباسُ %d — يُنتظر ٩٠٤", got)
	}
	if got := ledgerByRef(t, h, oid, "merchant_cash_paid"); got != -904 {
		t.Errorf("قيدُ الدفع %d — يُنتظر −٩٠٤", got)
	}
	if got := ledgerByRef(t, h, oid, "merchant_earning"); got != 0 {
		t.Errorf("**مستحقٌّ في محفظة متجرٍ «نقداً»**: %d", got)
	}
	if got := walletBalance(t, h, m.Owner.ID); got != 0 {
		t.Errorf("محفظةُ صاحب المتجر %d — يُنتظر صفر", got)
	}

	// السائق: أجرُه في محفظته، والنقدُ ناقصَ ما دفعه في صندوقه.
	if got := ledgerByRef(t, h, oid, "driver_earning"); got != 450 {
		t.Errorf("أجرُ السائق %d — يُنتظر ٤٥٠", got)
	}
	if got := walletBalance(t, h, drv.ID); got != 450 {
		t.Errorf("محفظةُ السائق %d — يُنتظر ٤٥٠", got)
	}
	if got := cashHeld(t, h, drv.ID); got != 701 {
		t.Errorf("**صندوقُ السائق %d — يُنتظر ٧٠١** (١٦٠٥ − ٩٠٤)", got)
	}

	// المندوب والخزينة والزبون.
	if got := repCommissionOf(t, h, oid); got != 20 {
		t.Errorf("نصيبُ المندوب %d — يُنتظر ٢٠ (١٠٪ من ١٠١ + ١٠٠)", got)
	}
	if got := ledgerByRef(t, h, oid, "platform_profit"); got != 231 {
		t.Errorf("**ربحُ المنصّة %d — يُنتظر ٢٣١** (١٠١ + ١٠٠ + ٥٠ − ٢٠)", got)
	}
	if got := walletBalance(t, h, cust.ID); got != custBefore {
		t.Errorf("محفظةُ الزبون تحرّكت في طلبٍ نقديّ: %d ⇒ %d", custBefore, got)
	}
	var advances int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM office_cash_entries
		WHERE source = 'advance_return' AND ref = $1`, oid).Scan(&advances)
	if advances != 0 {
		t.Errorf("سلفةٌ رُدّت من الدرج ونقدُ الزبون يغطّيها: %d", advances)
	}

	// الدرج: السائقُ يسلّم صندوقَه كلَّه.
	if r := h.POSTKey("/api/v1/admin/drivers/"+drv.ID+"/settle", fin.Token, uniq("settle"),
		map[string]any{"amount": 701, "note": "تسليم صندوق"}); r.Code != 200 || num(r.JSON()["held"]) != 0 {
		t.Fatalf("تسليمُ الصندوق: %s", r)
	}
	var inDrawer int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		WHERE source = 'driver_settle' AND direction = 'in' AND user_id = $1::uuid`, drv.ID).Scan(&inDrawer)
	if inDrawer != 701 {
		t.Errorf("**داخلُ الدرج %d — يُنتظر ٧٠١**", inDrawer)
	}
	if got := cashHeld(t, h, drv.ID); got != 0 {
		t.Errorf("صندوقُ السائق بعد التسليم %d", got)
	}

	// **والمحوُ بعد التسليم يُرفض** — العمولةُ لم تعد بيد السائق، ولا يُنزَل صندوقُه تحت الصفر.
	n := orderLedgerCount(t, h, oid)
	if r := h.POST("/api/v1/admin/orders/"+oid+"/waive-commission", fin.Token,
		map[string]any{"reason": "المتجر أخذ الثمن كامل"}); r.Code != 409 || r.Err() != "commission_waiver_driver_cash_short" {
		t.Errorf("محوٌ بعد تسليم الصندوق: %s", r)
	}
	if got := orderLedgerCount(t, h, oid); got != n {
		t.Errorf("**المحوُ المرفوضُ قيّد %d سطراً**", got-n)
	}

	assertNewViolations(t, h, base, coFamilies...)
}

// TestCASHORD_WaiverRepAlreadyWithdrewBecomesDebt **محوُ العمولة والمندوبُ طلب سحبَ رصيده كلِّه.**
//
// بضاعةٌ ١٠٠٠ · عمولةٌ ١٠٠ · نصيبُ المندوب ١٠ · ثمّ يطلب سحبَ ١٠ فيُحجَز.
// **المحو**: صندوقُ السائق ٥٠٠ · لا قيدَ في محفظة المندوب (متاحُه صفر) · **دَينٌ ١٠ عليه**
// (`waived_rep_commission`) يُقتطَع من عمولةٍ قادمة · والخزينةُ تنقص ١٠٠ كاملةً الآن.
func TestCASHORD_WaiverRepAlreadyWithdrewBecomesDebt(t *testing.T) {
	h := New(t)
	h.oldScaleMoney()
	treasury(t, h)
	waiverSettings(h)
	base := financialBaseline(t, h, coFamilies...)
	fin := h.NewUser("finance")

	oid, drv, rep, _ := waiverOrder(t, h, "cash")
	if got := repCommissionOf(t, h, oid); got != 10 {
		t.Fatalf("نصيبُ المندوب %d — يُنتظر ١٠", got)
	}
	repBal := walletBalance(t, h, rep.ID)
	if repBal <= 0 {
		t.Fatalf("رصيدُ المندوب %d", repBal)
	}
	if r := h.POST("/api/v1/me/payouts", rep.Token, map[string]any{"amount": repBal}); r.Code >= 400 {
		t.Fatalf("طلبُ السحب: %s", r)
	}
	profitBefore := ledgerByRef(t, h, oid, "platform_profit")

	r := h.POST("/api/v1/admin/orders/"+oid+"/waive-commission", fin.Token,
		map[string]any{"reason": "المتجر رفض يخصم العمولة"})
	if r.Code != 200 || num(r.JSON()["waived"]) != 100 || num(r.JSON()["driver_held"]) != 500 {
		t.Fatalf("المحو: %s", r)
	}
	if got := cashHeld(t, h, drv.ID); got != 500 {
		t.Errorf("صندوقُ السائق %d — يُنتظر ٥٠٠", got)
	}
	// **لا عكسَ في المحفظة** — متاحُه محجوزٌ لسحبه.
	if got := walletBalance(t, h, rep.ID); got != repBal {
		t.Errorf("محفظةُ المندوب %d — يُنتظر %d بلا مساس", got, repBal)
	}
	if got := repCommissionOf(t, h, oid); got != 10 {
		t.Errorf("صافي عمولة المندوب على الطلب %d — يُنتظر ١٠ (العكسُ صار دَيناً)", got)
	}
	var debt int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount - settled), 0) FROM financial_obligations
		WHERE party_kind = 'rep' AND party_id = $1::uuid AND cause = 'waived_rep_commission'
		  AND order_id = $2::uuid`, rep.ID, oid).Scan(&debt)
	if debt != 10 {
		t.Errorf("**دَينُ المندوب %d — يُنتظر ١٠**", debt)
	}
	if got := ledgerByRef(t, h, oid, "platform_profit"); got != profitBefore-100 {
		t.Errorf("ربحُ المنصّة %d — يُنتظر %d (العمولةُ كلُّها، والدَّينُ يُستردّ لاحقاً)", got, profitBefore-100)
	}
	assertNewViolations(t, h, base, coFamilies...)
}

// TestCASHORD_TwoCashStoresWaiverSplitsPerStore **طلبٌ من متجرين «نقداً» بنسبتين ثمّ يُمحى.**
//
//	المتجر أ   ١٠٠٠ · ١٠٪ = ١٠٠ · يقبض ٩٠٠ من السائق
//	المتجر ب    ٥٠٠ · ٢٠٪ = ١٠٠ · يقبض ٤٠٠ من السائق
//	الزبون     ٢٠٠٠ نقداً (١٥٠٠ + ٥٠٠) · صندوقُ السائق ٧٠٠ · أجرُه ٥٠٠ · المندوبُ ٢٠ · الخزينة ١٨٠
//	المحو      ٢٠٠ ⇒ كلُّ صفٍّ يحمل ١٠٠ · الصندوقُ ٥٠٠ · المندوبُ صفر · الخزينةُ صفر
func TestCASHORD_TwoCashStoresWaiverSplitsPerStore(t *testing.T) {
	h := New(t)
	h.oldScaleMoney()
	treasury(t, h)
	waiverSettings(h)
	h.Setting("orders.max_sources", "2")
	h.Setting("orders.extra_source_fee", "0")
	h.Setting("orders.source_mid_percent", "0")
	h.Setting("orders.source_far_percent", "0")
	base := financialBaseline(t, h, coFamilies...)
	fin := h.NewUser("finance")

	f := h.Factory()
	rep := f.RepAccount()
	mA := f.Merchant(OwnedByRep(rep.ID), Commission(10))
	mB := f.Merchant(OwnedByRep(rep.ID), Commission(20))
	coSetCash(t, h, mA.ID)
	coSetCash(t, h, mB.ID)
	a := h.NewItemFor(mA, 1000)
	b := h.NewItemFor(mB, 500)

	body := orderBody(a, 1)
	body["items"] = []map[string]any{{"menu_item_id": a.ID, "qty": 1}, {"menu_item_id": b.ID, "qty": 1}}
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("cashord2"), body)
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	if cashDue, _, fee, driverFee, _ := coOrderMoney(t, h, oid); cashDue != 2000 || fee != 500 || driverFee != 500 {
		t.Fatalf("الطلب: نقد %d أجرة %d أجرُ السائق %d — يُنتظر ٢٠٠٠ و٥٠٠ و٥٠٠", cashDue, fee, driverFee)
	}
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	if _, _, _, _, commission := coOrderMoney(t, h, oid); commission != 200 {
		t.Errorf("عمولةُ المنصّة %d — يُنتظر ٢٠٠ (١٠٠ + ١٠٠)", commission)
	}
	for _, c := range []struct {
		id   string
		want int64
	}{{mA.ID, 900}, {mB.ID, 400}} {
		amount, state, via, _ := coSettlement(t, h, oid, c.id)
		if amount != c.want || state != "cash_paid" || via != "driver" {
			t.Errorf("تسويةُ متجر: %d %s %s — يُنتظر %d cash_paid driver", amount, state, via, c.want)
		}
	}
	if got := cashHeld(t, h, drv.ID); got != 700 {
		t.Errorf("**صندوقُ السائق %d — يُنتظر ٧٠٠** (٢٠٠٠ − ٩٠٠ − ٤٠٠)", got)
	}
	if got := ledgerByRef(t, h, oid, "driver_earning"); got != 500 {
		t.Errorf("أجرُ السائق %d — يُنتظر ٥٠٠", got)
	}
	if got := repCommissionOf(t, h, oid); got != 20 {
		t.Errorf("نصيبُ المندوب %d — يُنتظر ٢٠", got)
	}
	if got := ledgerByRef(t, h, oid, "platform_profit"); got != 180 {
		t.Errorf("ربحُ المنصّة %d — يُنتظر ١٨٠ (٢٠٠ − ٢٠)", got)
	}

	r := h.POST("/api/v1/admin/orders/"+oid+"/waive-commission", fin.Token,
		map[string]any{"reason": "المتجران أخذا الثمن كامل"})
	if r.Code != 200 || num(r.JSON()["waived"]) != 200 || num(r.JSON()["driver_held"]) != 500 {
		t.Fatalf("المحو: %s", r)
	}
	for _, id := range []string{mA.ID, mB.ID} {
		if _, _, _, waived := coSettlement(t, h, oid, id); waived != 100 {
			t.Errorf("**صفُّ متجرٍ يحمل محواً %d — يُنتظر ١٠٠**", waived)
		}
	}
	if got := cashHeld(t, h, drv.ID); got != 500 {
		t.Errorf("صندوقُ السائق بعد المحو %d — يُنتظر ٥٠٠", got)
	}
	if got := repCommissionOf(t, h, oid); got != 0 {
		t.Errorf("صافي عمولة المندوب %d — يُنتظر صفر", got)
	}
	if got := ledgerByRef(t, h, oid, "platform_profit"); got != 0 {
		t.Errorf("**ربحُ المنصّة بعد المحو %d — يُنتظر صفر** (١٨٠ − ٢٠٠ + ٢٠)", got)
	}
	assertNewViolations(t, h, base, coFamilies...)
}
