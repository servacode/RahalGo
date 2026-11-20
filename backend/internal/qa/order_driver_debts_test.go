package qa

// ══════════════════════════════════════════════════════════════════════
// **«ديون الطلبات»** (قرار المالك ٢٠٢٦-١١-٢٠)
// ══════════════════════════════════════════════════════════════════════
//
// الزبون بيدفع **كل الطلب من المحفظة** بالنموذج ٢ ← الكابتن دفع للمتجر من جيبه وما قبض شي ←
// **دين عالمنصة للكابتن** بقد اللي دفعه، ما بينزل بمحفظته، والخزينة ما بتحسبه ربح.
// والمكتب بيكبس «تم الدفع» ← الدين بينشطب وسطر خارج بصندوق المكتب.

import (
	"net/http"
	"testing"
)

func odebt(t *testing.T, h *Harness, oid string) (amount int64, state string) {
	t.Helper()
	_ = h.Pool.QueryRow(ctxBG(), `SELECT amount, state FROM order_driver_debts WHERE order_id = $1::uuid`, oid).
		Scan(&amount, &state)
	return
}

func TestOrderDebts_FullFromWallet(t *testing.T) {
	h := New(t)
	item, _ := m2Prep(t, h)
	base := financialBaseline(t, h, m2Checks...)

	// ── ١ · «الكل من المحفظة» صار مقبول بالنموذج ٢ ──────────────────────────
	cust := h.Customer()
	oid := wsMake(t, h, cust, wsBody(item, 2, "", "full"), 50_000)
	if m := moneyModelOf(t, h, oid); m != 2 {
		t.Fatalf("نموذج الطلب %d — يُنتظر ٢", m)
	}
	o := wsRead(t, h, oid)
	if o.scope != "full" || o.walletPaid != o.total || o.cashDue != 0 {
		t.Fatalf("«الكل من المحفظة»: %+v — يُنتظر كل المجموع من المحفظة بلا نقد", o)
	}

	drv := h.driverOf(oid)
	wsDrive(t, h, oid, drv, "at_pickup", "picked_up")
	if a, _ := odebt(t, h, oid); a != 0 {
		t.Fatalf("**دين قبل التسليم**: %d", a)
	}
	wsDrive(t, h, oid, drv, "on_the_way", "at_dropoff", "delivered")

	// ── ٢ · بالتسليم: دين ١٨٠٠٠ مفتوح، ولا شي بمحفظة الكابتن غير أجرته، والصندوق صفر ──
	f := m2Read(t, h, oid)
	if a, st := odebt(t, h, oid); a != 18_000 || st != "open" {
		t.Fatalf("**دين الطلب**: %d %q — يُنتظر 18000 مفتوح (اللي دفعه للمتجر)", a, st)
	}
	var driverOther int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallet_transactions
		WHERE ref = $1 AND user_id = $2::uuid AND kind <> 'driver_earning'`, oid, drv.ID).Scan(&driverOther)
	if driverOther != 0 || f.box != 0 {
		t.Errorf("**حق البضاعة نزل بمحفظة الكابتن أو انقبض**: قيود %d صندوق %d", driverOther, f.box)
	}
	if want := f.walletPaid - f.driverFee - 18_000; f.profit != want {
		t.Errorf("**ربح المنصة %d** — يُنتظر المحفظة %d − أجر الكابتن %d − دينه 18000 = %d",
			f.profit, f.walletPaid, f.driverFee, want)
	}

	// ── ٣ · الكابتن بيشوفه، واللوحة بتجمعه ─────────────────────────────────
	if r := h.GET("/api/v1/driver/order-debts", drv.Token); r.Code != http.StatusOK || r.JSON()["open_total"] != float64(18_000) {
		t.Errorf("الكابتن: %s", r)
	}
	fin := h.NewUser("admin")
	if r := h.GET("/api/v1/admin/finance/order-debts", fin.Token); r.Code != http.StatusOK || r.JSON()["open_total"] != float64(18_000) {
		t.Errorf("اللوحة: %s", r)
	}
	if r := h.GET("/api/v1/admin/finance/order-debts", h.NewUser("operations").Token); !capDenied(r) {
		t.Errorf("**العمليات شافت ديون الطلبات بلا صلاحية المالية**: %s", r)
	}

	// ── ٤ · «تم الدفع»: الدين انشطب وسطر خارج بالصندوق، ومرة وحدة بس ─────────
	if r := h.POST("/api/v1/admin/finance/order-debts/"+drv.ID+"/pay", fin.Token, map[string]any{}); r.Code != http.StatusOK || r.JSON()["paid_total"] != float64(18_000) {
		t.Fatalf("تم الدفع: %s", r)
	}
	if _, st := odebt(t, h, oid); st != "paid" {
		t.Errorf("بعد الدفع: %q", st)
	}
	var out int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		WHERE source = 'order_debt_paid' AND user_id = $1::uuid AND direction = 'out'`, drv.ID).Scan(&out)
	if out != 18_000 {
		t.Errorf("**سطر الصندوق**: %d — يُنتظر 18000 خارج", out)
	}
	if r := h.POST("/api/v1/admin/finance/order-debts/"+drv.ID+"/pay", fin.Token, map[string]any{}); r.Code != http.StatusConflict {
		t.Errorf("**دفع مرتين**: %s", r)
	}
	if after := m2Read(t, h, oid); after.profit != f.profit {
		t.Errorf("**الدفع حرّك ربح المنصة**: %d ← %d", f.profit, after.profit)
	}

	// ── ٥ · «التوصيل من المحفظة» ما بيكتب دين (قبض البضاعة كاش) ─────────────
	c2 := h.Customer()
	o2 := wsMake(t, h, c2, wsBody(item, 1, "", "delivery"), 5_000)
	d2 := h.driverOf(o2)
	wsDrive(t, h, o2, d2, "at_pickup", "picked_up", "on_the_way", "at_dropoff", "delivered")
	if a, _ := odebt(t, h, o2); a != 0 {
		t.Errorf("**دين على «التوصيل من المحفظة»**: %d", a)
	}

	assertNewViolations(t, h, base, m2Checks...)
}

// TestOrderDebts_CustomFullFromWallet **الطلب الخاص كله من المحفظة** ← دين بقد ثمن البضاعة.
func TestOrderDebts_CustomFullFromWallet(t *testing.T) {
	h := New(t)
	m2Prep(t, h)
	base := financialBaseline(t, h, m2Checks...)

	cc := h.Customer()
	h.Factory().Credit(cc.ID, 50_000, "topup")
	cx := h.POSTKey("/api/v1/orders/custom", cc.Token, uniq("odc"), customBody("أغراض من السوق — كله من المحفظة"))
	if cx.Code >= 400 {
		t.Fatalf("إنشاء الخاص: %s", cx)
	}
	cid, _ := cx.JSON()["id"].(string)
	drv := h.driverOf(cid)
	if r := h.POST("/api/v1/driver/orders/"+cid+"/agree", drv.Token,
		map[string]any{"goods_amount": 5000, "fee": 700}); r.Code >= 400 {
		t.Fatalf("توثيق العرض: %s", r)
	}
	var ver int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT quote_version FROM orders WHERE id = $1::uuid`, cid).Scan(&ver)
	if r := h.POST("/api/v1/orders/"+cid+"/confirm-quote", cc.Token, map[string]any{
		"payment_method": "wallet", "wallet_scope": "full", "expected_total": 5700, "expected_quote_version": ver,
	}); r.Code >= 400 {
		t.Fatalf("تأكيد «الكل من المحفظة»: %s", r)
	}
	wsDrive(t, h, cid, drv, "picked_up", "on_the_way", "at_dropoff", "delivered")
	if a, st := odebt(t, h, cid); a != 5000 || st != "open" {
		t.Fatalf("**دين الخاص**: %d %q — يُنتظر 5000 مفتوح", a, st)
	}
	var goodsToDriver int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND user_id = $2::uuid`, cid, drv.ID).Scan(&goodsToDriver)
	var fee int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT driver_fee FROM orders WHERE id = $1::uuid`, cid).Scan(&fee)
	if goodsToDriver != fee {
		t.Errorf("**محفظة الكابتن من الطلب %d** — يُنتظر أجرته %d بس", goodsToDriver, fee)
	}
	assertNewViolations(t, h, base, m2Checks...)
}
