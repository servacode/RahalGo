package qa

// ══════════════════════════════════════════════════════════════════════
// **«تصحيح: الطلب انلغى»** — قرارُ المالك ٢٠٢٦-١٠-٢٨
// ══════════════════════════════════════════════════════════════════════
//
// السائقُ ضغط «تم التسليم» خطأً والطلبُ أُلغي فعلاً (#1099 في الإنتاج). فيُصحّحه الأدمن:
// **يصير الطلبُ ملغياً، ويُعكَس ما سجّله التسليمُ على السائق والمندوب، ويبقى للمتجر ما قبضه** —
// فتصير قيودُ الطلب كقيود طلبٍ انتهى بعد الاستلام بلا تسليم.
//
// **والتوأمُ هو الشاهد**: طلبٌ بالشكل نفسِه يُنهيه المكتبُ بعد الاستلام (`door-resolution`)
// — ومجاميعُ قيوده لكلّ نوعٍ يجب أن تساوي مجاميعَ الطلب المصحَّح.

import (
	"testing"
	"time"
)

// undeliverSettings **شكلُ #1099**: عمولةُ متجر ١٠٪ · المندوبُ ١٠٪ منها · أجرةُ السائق ١٠٠ ·
// وعرضُ توصيلٍ مجّانيٍّ فعّال (الزبونُ لا يدفع التوصيل والخزينةُ تموّل أجرةَ السائق).
func undeliverSettings(t *testing.T, h *Harness) {
	t.Helper()
	h.oldScaleMoney()
	treasury(t, h)
	waiverSettings(h)
	h.Setting("delivery.fee", "100")
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM free_delivery_campaigns`) })
	now := time.Now().UTC()
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO free_delivery_campaigns (active, starts_at, ends_at, kinds)
		VALUES (true, $1, $2, ARRAY['standard'])`, now.Add(-time.Hour), now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
}

type undeliverFx struct {
	oid  string
	drv  *User
	cust *User
	rep  *Rep
	m    *Merchant
}

// undeliverPlaced **طلبٌ عاديٌّ من متجرٍ «نقداً» بسعر ١٢٠٠ ومندوب** — لم يتحرّك بعد الإسناد.
func undeliverPlaced(t *testing.T, h *Harness, f *Factory, payment string) undeliverFx {
	t.Helper()
	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID), Commission(10))
	coSetCash(t, h, m.ID)
	item := h.NewItemFor(m, 1200)
	cust := h.Customer()
	body := orderBody(item, 1)
	if payment == "wallet" {
		fundWallet(t, h, h.NewUser("admin"), cust.ID, 1200)
		body["payment_method"] = "wallet"
	}
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("undeliver"), body)
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	cashDue, walletPaid, fee, driverFee, _ := coOrderMoney(t, h, oid)
	if cashDue+walletPaid != 1200 || fee != 0 || driverFee != 100 {
		t.Fatalf("الطلب: نقد %d محفظة %d أجرة الزبون %d أجر السائق %d — يُنتظر مجموع ١٢٠٠ وأجرة ٠ وأجر ١٠٠",
			cashDue, walletPaid, fee, driverFee)
	}
	return undeliverFx{oid: oid, drv: h.driverOf(oid), cust: cust, rep: rep, m: m}
}

// undeliverTwin **التوأم**: الشكلُ نفسُه، يُنهيه المكتبُ وهو في الطريق — بلا تسليم.
func undeliverTwin(t *testing.T, h *Harness, f *Factory, payment string) string {
	t.Helper()
	x := undeliverPlaced(t, h, f, payment)
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way"} {
		if r := h.POST("/api/v1/driver/orders/"+x.oid+"/transition", x.drv.Token,
			map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("التوأم إلى %s: %s", to, r)
		}
	}
	admin := h.NewUser("admin")
	if r := h.POST("/api/v1/admin/orders/"+x.oid+"/door-resolution", admin.Token, map[string]any{
		"action": "return_to_office", "fault": "platform", "return_to": "office",
		"note": "الطلب أُلغي والبضاعة مع السائق",
	}); r.Code >= 400 {
		t.Fatalf("إنهاءُ التوأم بعد الاستلام: %s", r)
	}
	return x.oid
}

// moneyShape **قيودُ الطلب مجموعةً لكلّ نوع** — والأصفارُ تُحذف (عكسٌ تامٌّ يساوي لا قيد).
func moneyShape(t *testing.T, h *Harness, oid string) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT 'w:' || kind, sum(amount)::bigint FROM wallet_transactions WHERE ref = $1 GROUP BY kind
		UNION ALL
		SELECT 'box', COALESCE(sum(amount), 0)::bigint FROM driver_cash_entries WHERE ref = $1
		UNION ALL
		SELECT 'drawer_out', COALESCE(sum(amount), 0)::bigint FROM office_cash_entries
		 WHERE ref = $1 AND direction = 'out'
		UNION ALL
		SELECT 'debt', COALESCE(sum(amount), 0)::bigint FROM financial_obligations WHERE order_id::text = $1`, oid)
	if err != nil {
		t.Fatalf("شكلُ المال: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var v int64
		if err := rows.Scan(&k, &v); err != nil {
			t.Fatal(err)
		}
		if v != 0 {
			out[k] = v
		}
	}
	return out
}

func sameShape(t *testing.T, label string, got, want map[string]int64) {
	t.Helper()
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %s = %d — والتوأمُ الملغى بعد الاستلام %d\nالمصحَّح: %v\nالتوأم:   %v", label, k, got[k], v, got, want)
		}
	}
	for k, v := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("%s: %s = %d في المصحَّح ولا مثيلَ له في التوأم\nالمصحَّح: %v\nالتوأم:   %v", label, k, v, got, want)
		}
	}
}

func undeliver(h *Harness, by *User, oid, reason string) Res {
	return h.POST("/api/v1/admin/orders/"+oid+"/undeliver", by.Token, map[string]any{"reason": reason})
}

func boxOf(t *testing.T, h *Harness, oid string) int64 {
	t.Helper()
	var v int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM driver_cash_entries WHERE ref = $1`, oid).Scan(&v)
	return v
}

// TestUNDELIVER_1099ShapeMatchesCancelAfterPickup **الحالةُ نفسُها: نقدٌ، والسائقُ دفع للمتجر
// ١٠٨٠ عند الاستلام، ثمّ ضغط «تم التسليم» خطأً.**
//
//	قبل التصحيح  صندوقُ السائق ١٢٠ · أجرُه ١٠٠ · المندوب ١٢ · ربحُ المنصّة ٨
//	بعد التصحيح  ملغى · الصندوقُ صفر · الأجرُ صفر · المندوبُ صفر · المتجرُ ١٠٨٠ كما هو ·
//	             الزبونُ لم يُمسّ · الدرجُ يردّ ١٠٨٠ سلفةً · ربحُ المنصّة −١٠٨٠ — كالتوأم بالضبط
func TestUNDELIVER_1099ShapeMatchesCancelAfterPickup(t *testing.T) {
	h := New(t)
	undeliverSettings(t, h)
	f := h.Factory()
	base := financialBaseline(t, h)

	x := undeliverPlaced(t, h, f, "cash")
	custBefore := walletBalance(t, h, x.cust.ID)
	deliverOrder(t, h, x.oid, x.drv)

	if got := cashHeld(t, h, x.drv.ID); got != 120 {
		t.Fatalf("صندوقُ السائق بعد التسليم %d — يُنتظر ١٢٠ (١٢٠٠ − ١٠٨٠)", got)
	}
	if got := ledgerByRef(t, h, x.oid, "driver_earning"); got != 100 {
		t.Fatalf("أجرُ السائق %d — يُنتظر ١٠٠", got)
	}
	if got := repCommissionOf(t, h, x.oid); got != 12 {
		t.Fatalf("نصيبُ المندوب %d — يُنتظر ١٢", got)
	}
	if got := ledgerByRef(t, h, x.oid, "platform_profit"); got != 8 {
		t.Fatalf("ربحُ المنصّة %d — يُنتظر ٨ (١٢٠ − ١٠٠ − ١٢)", got)
	}

	// **غيرُ الأدمن لا يملكه** — ولو ملك التدخّلَ في الطلبات.
	if r := undeliver(h, h.NewUser("operations"), x.oid, "السائق ضغط تم التسليم خطأً"); r.Code < 400 {
		t.Fatalf("**موظّفُ العمليّات صحّح التسليم**: %s", r)
	}
	// **والسببُ إلزاميّ.**
	if r := undeliver(h, h.NewUser("admin"), x.oid, "  "); r.Code != 400 || r.Err() != "undeliver_reason_required" {
		t.Fatalf("تصحيحٌ بلا سبب: %s", r)
	}

	admin := h.NewUser("admin")
	reason := "السائق ضغط تم التسليم خطأً والزبون ألغى الطلب"
	var lastTx int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(max(id), 0) FROM wallet_transactions`).Scan(&lastTx)
	r := undeliver(h, admin, x.oid, reason)
	if r.Code != 200 {
		t.Fatalf("التصحيح: %s", r)
	}
	// **ما كتبه التصحيحُ بعينه** — يُطبع شاهداً.
	if rows, err := h.Pool.Query(ctxBG(), `
		SELECT 'wallet', kind, amount, note FROM wallet_transactions WHERE ref = $1 AND id > $2
		UNION ALL
		SELECT 'box', kind, amount, note FROM driver_cash_entries WHERE ref = $1 AND kind = 'collection_reversed'
		UNION ALL
		SELECT 'drawer', source, amount, note FROM office_cash_entries WHERE ref = $1`, x.oid, lastTx); err == nil {
		for rows.Next() {
			var where, kind, note string
			var amount int64
			if rows.Scan(&where, &kind, &amount, &note) == nil {
				t.Logf("قيدُ التصحيح: %s · %s · %d · %s", where, kind, amount, note)
			}
		}
		rows.Close()
	}
	if got := h.statusOf(x.oid); got != "cancelled" {
		t.Errorf("الحالُ بعد التصحيح %q — يُنتظر cancelled", got)
	}
	var cancelReason string
	var events, audits int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(cancel_reason, '') FROM orders WHERE id = $1`, x.oid).Scan(&cancelReason)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_events WHERE order_id = $1
		AND from_status = 'delivered' AND to_status = 'cancelled' AND note = $2 AND actor_id = $3::uuid`,
		x.oid, reason, admin.ID).Scan(&events)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM audit_log WHERE action = 'ops.order_undeliver'
		AND entity = 'order' AND entity_id = $1`, x.oid).Scan(&audits)
	if cancelReason != reason || events != 1 || audits != 1 {
		t.Errorf("الأثر: سبب %q · حدث %d · تدقيق %d — يُنتظر السببُ وحدثٌ وتدقيق", cancelReason, events, audits)
	}

	if got := boxOf(t, h, x.oid); got != 0 {
		t.Errorf("**صندوقُ السائق عن الطلب %d — يُنتظر صفر** (لم يقبض من الزبون شيئاً)", got)
	}
	if got := cashHeld(t, h, x.drv.ID); got != 0 {
		t.Errorf("صندوقُ السائق %d — يُنتظر صفر", got)
	}
	if got := ledgerByRef(t, h, x.oid, "driver_earning"); got != 0 {
		t.Errorf("**أجرُ السائق الصافي %d — يُنتظر صفر** (لم يوصل)", got)
	}
	if got := walletBalance(t, h, x.drv.ID); got != 0 {
		t.Errorf("محفظةُ السائق %d — يُنتظر صفر", got)
	}
	if got := repCommissionOf(t, h, x.oid); got != 0 {
		t.Errorf("**عمولةُ المندوب الصافية %d — يُنتظر صفر**", got)
	}
	if got := walletBalance(t, h, x.rep.ID); got != 0 {
		t.Errorf("محفظةُ المندوب %d — يُنتظر صفر", got)
	}
	if amount, state, via, _ := coSettlement(t, h, x.oid, x.m.ID); amount != 1080 || state != "cash_paid" || via != "driver" {
		t.Errorf("**تسويةُ المتجر تبدّلت**: %d %s %s — يُنتظر ١٠٨٠ cash_paid driver", amount, state, via)
	}
	if got := ledgerByRef(t, h, x.oid, "merchant_cash_paid"); got != -1080 {
		t.Errorf("قيدُ دفع المتجر %d — يُنتظر −١٠٨٠ كما هو", got)
	}
	if got := walletBalance(t, h, x.cust.ID); got != custBefore {
		t.Errorf("**محفظةُ الزبون تحرّكت في طلبٍ نقديّ**: %d ⇒ %d", custBefore, got)
	}
	if got := ledgerByRef(t, h, x.oid, "platform_profit"); got != -1080 {
		t.Errorf("**ربحُ المنصّة %d — يُنتظر −١٠٨٠** (ما دفعه السائقُ للمتجر خسارةٌ عليها)", got)
	}

	// **والتوأم** — الطلبُ نفسُه أُنهي بعد الاستلام بلا تسليم.
	twin := undeliverTwin(t, h, f, "cash")
	sameShape(t, "نقد", moneyShape(t, h, x.oid), moneyShape(t, h, twin))

	assertNewViolations(t, h, base)
}

// TestUNDELIVER_WalletPaidRefundsCustomer **دفع الزبونُ من محفظته** — فيُردّ له ما دفع، والباقي كالنقد.
func TestUNDELIVER_WalletPaidRefundsCustomer(t *testing.T) {
	h := New(t)
	undeliverSettings(t, h)
	f := h.Factory()
	base := financialBaseline(t, h)

	x := undeliverPlaced(t, h, f, "wallet")
	deliverOrder(t, h, x.oid, x.drv)
	if got := walletBalance(t, h, x.cust.ID); got != 0 {
		t.Fatalf("محفظةُ الزبون بعد الدفع %d — يُنتظر صفر", got)
	}
	if r := undeliver(h, h.NewUser("admin"), x.oid, "الطلب لم يصل الزبون"); r.Code != 200 {
		t.Fatalf("التصحيح: %s", r)
	}
	if got := walletBalance(t, h, x.cust.ID); got != 1200 {
		t.Errorf("**محفظةُ الزبون %d — يُنتظر ١٢٠٠** (يُردّ له ما دفع)", got)
	}
	if got := ledgerByRef(t, h, x.oid, "driver_earning"); got != 0 {
		t.Errorf("أجرُ السائق الصافي %d — يُنتظر صفر", got)
	}
	if got := repCommissionOf(t, h, x.oid); got != 0 {
		t.Errorf("عمولةُ المندوب الصافية %d — يُنتظر صفر", got)
	}
	if got := ledgerByRef(t, h, x.oid, "platform_profit"); got != -1080 {
		t.Errorf("ربحُ المنصّة %d — يُنتظر −١٠٨٠", got)
	}
	twin := undeliverTwin(t, h, f, "wallet")
	sameShape(t, "محفظة", moneyShape(t, h, x.oid), moneyShape(t, h, twin))
	assertNewViolations(t, h, base)
}

// TestUNDELIVER_SecondCallChangesNothing **مرّةً واحدة**: النداءُ الثاني يُردّ ولا يكتب قيداً.
func TestUNDELIVER_SecondCallChangesNothing(t *testing.T) {
	h := New(t)
	undeliverSettings(t, h)
	f := h.Factory()
	x := undeliverPlaced(t, h, f, "cash")
	deliverOrder(t, h, x.oid, x.drv)
	admin := h.NewUser("admin")
	if r := undeliver(h, admin, x.oid, "تسليم بالخطأ"); r.Code != 200 {
		t.Fatalf("الأوّل: %s", r)
	}
	n := orderLedgerCount(t, h, x.oid)
	shape := moneyShape(t, h, x.oid)
	if r := undeliver(h, admin, x.oid, "تسليم بالخطأ"); r.Code != 409 || r.Err() != "undeliver_not_delivered" {
		t.Fatalf("**النداءُ الثاني**: %s", r)
	}
	if got := orderLedgerCount(t, h, x.oid); got != n {
		t.Errorf("**النداءُ الثاني كتب %d قيداً**", got-n)
	}
	sameShape(t, "ثانٍ", moneyShape(t, h, x.oid), shape)
}

// TestUNDELIVER_RefusesOtherKinds **الخاصُّ و«لدي توصيلة» لهما مسارُ مالٍ آخر** — يُردّان بلا أثر.
func TestUNDELIVER_RefusesOtherKinds(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")

	custom := newCustomOrder(t, h, h.Customer())
	if r := undeliver(h, admin, custom, "خطأ"); r.Code != 409 || r.Err() != "undeliver_not_standard" {
		t.Errorf("طلبٌ خاصّ: %s", r)
	}

	m := newMDFx(t, h, "منطقةُ التصحيح")
	fundWallet(t, h, m.admin, m.fx.M.Owner.ID, 2000)
	id, drv := mdmCreate(t, h, m, "merchant", "", "undeliver-md")
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdmDeliver(t, h, id, drv)
	n := orderLedgerCount(t, h, id)
	if r := undeliver(h, admin, id, "خطأ"); r.Code != 409 || r.Err() != "undeliver_not_standard" {
		t.Errorf("«لدي توصيلة»: %s", r)
	}
	if got := h.statusOf(id); got != "delivered" {
		t.Errorf("الحالُ تبدّل إلى %q", got)
	}
	if got := orderLedgerCount(t, h, id); got != n {
		t.Errorf("**الرفضُ كتب %d قيداً**", got-n)
	}

	// **وغيرُ المسلَّم يُردّ** — طلبٌ عاديٌّ لم يُسلَّم بعد.
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("undeliver-open"), orderBody(item, 1))
	open, _ := made.JSON()["id"].(string)
	if r := undeliver(h, admin, open, "خطأ"); r.Code != 409 || r.Err() != "undeliver_not_delivered" {
		t.Errorf("طلبٌ لم يُسلَّم: %s", r)
	}
}

// TestUNDELIVER_PartiesAlreadyWithdrew **السائقُ والمندوبُ طلبا سحبَ رصيدهما، والسائقُ سلّم صندوقَه.**
//
//	الأجرُ والعمولةُ لا يحتملهما المتاحُ ⇒ دَينٌ على كلٍّ منهما (`undelivered_order`) ولا رصيدَ سالب
//	والسائقُ سلّم ١٢٠ لم يقبضها من الزبون ⇒ الدرجُ يردّها له مع السلفة (١٠٨٠ + ١٢٠)
func TestUNDELIVER_PartiesAlreadyWithdrew(t *testing.T) {
	h := New(t)
	undeliverSettings(t, h)
	f := h.Factory()
	base := financialBaseline(t, h)
	fin := h.NewUser("finance")

	x := undeliverPlaced(t, h, f, "cash")
	deliverOrder(t, h, x.oid, x.drv)
	for _, u := range []struct {
		who *User
		amt int64
	}{{x.drv, 100}, {x.rep.User, 12}} {
		if r := h.POST("/api/v1/me/payouts", u.who.Token, map[string]any{"amount": u.amt}); r.Code >= 400 {
			t.Fatalf("طلبُ السحب: %s", r)
		}
	}
	if r := h.POSTKey("/api/v1/admin/drivers/"+x.drv.ID+"/settle", fin.Token, uniq("settle"),
		map[string]any{"amount": 120, "note": "تسليم صندوق"}); r.Code != 200 {
		t.Fatalf("تسليمُ الصندوق: %s", r)
	}
	drvBal, repBal := walletBalance(t, h, x.drv.ID), walletBalance(t, h, x.rep.ID)

	if r := undeliver(h, h.NewUser("admin"), x.oid, "تسليم بالخطأ"); r.Code != 200 {
		t.Fatalf("التصحيح: %s", r)
	}
	if got := walletBalance(t, h, x.drv.ID); got != drvBal {
		t.Errorf("محفظةُ السائق %d — يُنتظر %d بلا مساس (متاحُه محجوزٌ لسحبه)", got, drvBal)
	}
	if got := walletBalance(t, h, x.rep.ID); got != repBal {
		t.Errorf("محفظةُ المندوب %d — يُنتظر %d بلا مساس", got, repBal)
	}
	debt := func(party, id string) int64 {
		var v int64
		_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount - settled), 0) FROM financial_obligations
			WHERE party_kind = $1 AND party_id = $2::uuid AND cause = 'undelivered_order' AND order_id = $3::uuid`,
			party, id, x.oid).Scan(&v)
		return v
	}
	if got := debt("driver", x.drv.ID); got != 100 {
		t.Errorf("**دَينُ السائق %d — يُنتظر ١٠٠**", got)
	}
	if got := debt("rep", x.rep.ID); got != 12 {
		t.Errorf("**دَينُ المندوب %d — يُنتظر ١٢**", got)
	}
	if got := cashHeld(t, h, x.drv.ID); got != 0 {
		t.Errorf("صندوقُ السائق %d — لا يهبط تحت الصفر", got)
	}
	var out int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		WHERE ref = $1 AND direction = 'out'`, x.oid).Scan(&out)
	if out != 1200 {
		t.Errorf("**الخارجُ من الدرج %d — يُنتظر ١٢٠٠** (السلفة ١٠٨٠ + ما سلّمه السائقُ ١٢٠)", out)
	}
	if h.statusOf(x.oid) != "cancelled" {
		t.Errorf("الحالُ لم يصر ملغياً")
	}
	assertNewViolations(t, h, base)
}
