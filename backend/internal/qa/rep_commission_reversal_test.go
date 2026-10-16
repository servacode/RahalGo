package qa

import "testing"

// ══════════════════════════════════════════════════════════════════════
// **`XG-10` — عمولةُ المندوب تبقى له عند الاسترداد** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// «المندوبُ ما إلو ذنب» — **لكلّ أنواع الطلبات.** فلا تُعكَس عمولتُه، **والمنصّةُ تتحمّلها
// افتراضاً**، والمكتبُ يملك مع الاسترجاع أن يحمّلها المتجرَ أو السائقَ ديناً بمقدارها
// (`rep_share` · سببُ `refund_rep_share`).
//
// # وقرارٌ نُسخ
//
// **كان العقدُ** (`RQ-5` · ٢٠٢٦-٠٩-٠٥): «ORDER REVENUE REVERSED → RELATED REP COMMISSION
// REVERSED» — **وكانت هذه الحرّاسُ تشترط العكس.** **ونسخه المالكُ نصّاً**، فصارت تشترط البقاء.

// withMargin **هامشٌ صريحٌ للصنف** — **وبلاه لا ربحَ فلا عمولة.**
//
// **وسعرُ البيع محسوبٌ لا مخزَّن** (`service.go:608`): يُبنى من كلفة
// المتجر وهامشِ الصنف أو قسمِه (`margin_override`). **وحقلُ `price` في
// `menu_items` لا يدخل الحسبة** — **فمن ضبط السعرَ وحدَه لم يضبط شيئاً.**
func withMargin(t *testing.T, h *Harness, it *Item, margin int64) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE menu_items SET margin_override = $2 WHERE id = $1::uuid`,
		it.ID, margin); err != nil {
		t.Fatalf("ضبطُ الهامش: %v", err)
	}
}

// xg10Delivered **طلبٌ مسلَّمٌ من متجرٍ جلبه مندوب — وعمولتُه مقيَّدة.**
func xg10Delivered(t *testing.T, h *Harness) (oid, merchantID string, rep, drv *User, paid int64) {
	t.Helper()
	treasury(t, h)
	// **وعمولةُ المندوب افتراضُها صفرٌ عمداً** — **حتّى يضبطها المالك.**
	h.Setting("merchants.commission_percent", "20")
	h.Setting("sales.commission_percent", "10")
	f := h.Factory()
	rep = h.NewUser("sales")
	m := f.Merchant(OwnedByRep(rep.ID))
	item := h.NewItemPriced(m, 5000, 8000)
	withMargin(t, h, item, 3000)
	cust := h.Customer()

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 2))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ = made.JSON()["id"].(string)
	drv = h.driverOf(oid)
	deliverOrder(t, h, oid, drv)
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		  WHERE ref = $1 AND kind = 'commission'`, oid).Scan(&paid); err != nil {
		t.Fatalf("قراءةُ العمولة: %v", err)
	}
	if paid <= 0 {
		t.Skip("لم تُقيَّد عمولةٌ للمندوب — والسيناريو يشترطها")
	}
	return oid, m.ID, rep, drv, paid
}

// xg10Kept **صافي عمولة الطلب ودينُ المندوب** — بعد الاسترداد.
func xg10Kept(t *testing.T, h *Harness, oid string, rep *User) (net, debt int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = $1 AND kind = 'commission'), 0),
		       COALESCE((SELECT commission_debt FROM users WHERE id = $2::uuid), 0)`,
		oid, rep.ID).Scan(&net, &debt); err != nil {
		t.Fatalf("قراءةُ العمولة بعد الاسترداد: %v", err)
	}
	return net, debt
}

// xg10ShareDebt **دينُ نصيب المندوب على طرفٍ عن هذا الطلب.**
func xg10ShareDebt(t *testing.T, h *Harness, oid, party, partyID string) int64 {
	t.Helper()
	var v int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM financial_obligations
		 WHERE cause = 'refund_rep_share' AND order_id = $1::uuid
		   AND party_kind = $2 AND party_id = $3::uuid`, oid, party, partyID).Scan(&v); err != nil {
		t.Fatalf("قراءةُ الدين: %v", err)
	}
	return v
}

// TestFIN_XG10_RepCommissionKeptOnRefund **المنصّةُ تتحمّل افتراضاً — والمندوبُ يبقى بعمولته.**
func TestFIN_XG10_RepCommissionKeptOnRefund(t *testing.T) {
	h := New(t)
	base := financialBaseline(t, h, "FI-04", "FI-05", "FI-06", "FI-09", "FI-11")
	oid, merchantID, rep, drv, paid := xg10Delivered(t, h)

	admin := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "XG-10 — المندوب ما إلو ذنب"}); got.Code >= 400 {
		t.Fatalf("الاستردادُ سقط: %s", got)
	}
	net, debt := xg10Kept(t, h, oid, rep)
	if net != paid {
		t.Errorf("XG-10 — **عمولةُ المندوب عُكست**: قُيّد %d وصافيه بعد الاسترداد %d — "+
			"والقرارُ (٢٠٢٦-١٠-١٦) أنّها تبقى له", paid, net)
	}
	if debt != 0 {
		t.Errorf("مندوبٌ دُيّن %d على طلبٍ لا ذنبَ له فيه", debt)
	}
	if d := xg10ShareDebt(t, h, oid, "merchant", merchantID) + xg10ShareDebt(t, h, oid, "driver", drv.ID); d != 0 {
		t.Errorf("دينُ نصيبٍ %d بلا اختيار — والافتراضُ المنصّة", d)
	}
	assertNewViolations(t, h, base, "FI-04", "FI-05", "FI-06", "FI-09", "FI-11")
}

// TestFIN_XG10_RepShareOnMerchant **«المتجر» ⇒ دينٌ على متجر الطلب بمقدار العمولة.**
func TestFIN_XG10_RepShareOnMerchant(t *testing.T) {
	h := New(t)
	base := financialBaseline(t, h, "FI-04", "FI-05", "FI-06", "FI-09", "FI-11")
	oid, merchantID, rep, _, paid := xg10Delivered(t, h)

	admin := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "على المتجر", "rep_share": "merchant"}); got.Code >= 400 {
		t.Fatalf("الاستردادُ سقط: %s", got)
	}
	if net, _ := xg10Kept(t, h, oid, rep); net != paid {
		t.Errorf("عمولةُ المندوب %d بعد الاسترداد — والمنتظَر %d", net, paid)
	}
	if got := xg10ShareDebt(t, h, oid, "merchant", merchantID); got != paid {
		t.Errorf("دينُ المتجر %d — والمنتظَر %d (نصيبُ المندوب)", got, paid)
	}
	assertNewViolations(t, h, base, "FI-04", "FI-05", "FI-06", "FI-09", "FI-11")
}

// TestFIN_XG10_RepShareOnDriver **«السائق» ⇒ دينٌ على سائق الطلب بمقدارها.**
func TestFIN_XG10_RepShareOnDriver(t *testing.T) {
	h := New(t)
	oid, _, _, drv, paid := xg10Delivered(t, h)
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(),
			`DELETE FROM financial_obligations WHERE party_kind = 'driver' AND party_id = $1::uuid`, drv.ID)
		_, _ = h.Pool.Exec(ctxBG(), `UPDATE users SET driver_debt = 0 WHERE id = $1::uuid`, drv.ID)
	})
	admin := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "على السائق", "rep_share": "driver"}); got.Code >= 400 {
		t.Fatalf("الاستردادُ سقط: %s", got)
	}
	if got := xg10ShareDebt(t, h, oid, "driver", drv.ID); got != paid {
		t.Errorf("دينُ السائق %d — والمنتظَر %d", got, paid)
	}
}

// TestFIN_XG10_RepShareBadChoiceRefused **اختيارٌ لا يُعرف، أو مع غير الاسترجاع، يُردّ قبل أيّ أثر.**
func TestFIN_XG10_RepShareBadChoiceRefused(t *testing.T) {
	h := New(t)
	oid, _, _, _, _ := xg10Delivered(t, h)
	admin := h.NewUser("admin")
	got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "x", "rep_share": "customer"})
	if got.Code != 400 || got.Err() != "refund_bad_rep_share" {
		t.Fatalf("اختيارٌ مجهولٌ ردّ %d / %s — والمنتظَر 400 refund_bad_rep_share", got.Code, got.Err())
	}
	var st string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT status FROM orders WHERE id = $1::uuid`, oid).Scan(&st)
	if st != "delivered" {
		t.Fatalf("الحالُ %q بعد اختيارٍ مردود", st)
	}
}

// TestFIN_XG10_RepWithdrewThenRefund **مندوبٌ سحب عمولتَه — والاستردادُ يمرّ ولا يُدينه.**
//
// **وهي `PG-6`**: حقُّ الزبون لا يسقط برصيد طرفٍ ثالث — **وصار أبسط**: لا عكسَ فلا عجز.
func TestFIN_XG10_RepWithdrewThenRefund(t *testing.T) {
	h := New(t)
	base := financialBaseline(t, h, "FI-04", "FI-05", "FI-06", "FI-09", "FI-11")
	oid, _, rep, _, _ := xg10Delivered(t, h)
	f := h.Factory()
	var bal int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`, rep.ID).Scan(&bal)
	if bal > 0 {
		f.Credit(rep.ID, -bal, "payout")
	}
	admin := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "XG-10 · PG-6"}); got.Code >= 400 {
		t.Fatalf("**حقُّ الزبون سقط برصيد المندوب** — الردّ %d", got.Code)
	}
	if _, debt := xg10Kept(t, h, oid, rep); debt != 0 {
		t.Errorf("مندوبٌ سحب عمولتَه فدُيّن %d — والقرارُ أنّها له", debt)
	}
	var negative int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM wallets WHERE balance < 0 AND NOT is_treasury`).Scan(&negative)
	if negative > 0 {
		t.Errorf("%d محفظةً برصيدٍ سالبٍ وليست خزينة", negative)
	}
	assertNewViolations(t, h, base, "FI-04", "FI-05", "FI-06", "FI-09", "FI-11")
}
