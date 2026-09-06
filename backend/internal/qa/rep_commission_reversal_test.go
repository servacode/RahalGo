package qa

import "testing"

// ══════════════════════════════════════════════════════════════════════
// **`XG-10` — عمولةُ المندوب تُعكَس عند الاسترداد**
// ══════════════════════════════════════════════════════════════════════
//
// # العقدُ المعتمد (`RQ-5 = APPROVED` · ٢٠٢٦-٠٩-٠٥)
//
//	ORDER REVENUE REVERSED → RELATED REP COMMISSION REVERSED
//
// **«ولا يجوز: يسترد الزبونُ مالَه · وتنعكس مستحقّاتُ المتجر والمنصّة ·
// وتبقى عمولةُ المندوب قابلةً للسحب.»**
//
// **وإن كانت قد سُحبت تُعالَج التزاماً — ولا يُمحى التاريخ.**
//
// # وقرارٌ سابقٌ نُسخ
//
// **كان في `TRUTH.md` (٢٠٢٦-٠٨-٠٣): «استُرِدّ بعد التسليم ⇒ تبقى
// عمولتُه»** — **وسياقُه نزاعُ الطعام الفاسد**: من يتحمّل الخسارةَ حين
// يشتكي زبون.
//
// **والقرارُ الأحدث يتحدّث عن الاسترداد عامّاً وينسخه** (صُولح
// ٢٠٢٦-٠٩-٠٦). **وبقي التعليقُ القديمُ في `transitions.go` شهراً بعد
// نسخه** — **ووثيقتان تتناقضان صامتتين أسوأُ من لا وثيقة.**

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

// TestFIN_XG10_RepCommissionReversedOnRefund **الحالُ الأولى — رصيدٌ قائم.**
func TestFIN_XG10_RepCommissionReversedOnRefund(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h, "FI-04", "FI-05", "FI-06", "FI-09", "FI-11")
	// **وعمولةُ المندوب افتراضُها صفرٌ عمداً** — **حتّى يضبطها المالك.**
	// **فتُضبَط هنا صراحةً**: السيناريو يشترط عمولةً مقيَّدة.
	h.Setting("merchants.commission_percent", "20")
	h.Setting("sales.commission_percent", "10")
	f := h.Factory()
	rep := h.NewUser("sales")
	item := h.NewItemPriced(f.Merchant(OwnedByRep(rep.ID)), 5000, 8000)
	withMargin(t, h, item, 3000)
	cust := h.Customer()

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 2))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	var paid int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		  WHERE ref = $1 AND kind = 'commission'`, oid).Scan(&paid); err != nil {
		t.Fatalf("قراءةُ العمولة: %v", err)
	}
	if paid <= 0 {
		t.Skip("لم تُقيَّد عمولةٌ للمندوب — والسيناريو يشترطها")
	}
	t.Logf("عمولةُ المندوب المقيَّدة = %d", paid)

	admin := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "XG-10 — حارسُ العقد"}); got.Code >= 400 {
		t.Fatalf("الاستردادُ سقط: %s", got)
	}

	// ══════════════════════════════════════════════════════════════
	// **العقد**: صافي قيد العمولة عن هذا الطلب = صفر
	// ══════════════════════════════════════════════════════════════
	//
	// **ولا يُقاس بوجود قيدٍ سالبٍ وحدَه** — **الصافي هو العقد**،
	// وبه يُقرأ الدفترُ في كلّ حسبة.
	var net int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		  WHERE ref = $1 AND kind = 'commission'`, oid).Scan(&net); err != nil {
		t.Fatalf("قراءةُ الصافي: %v", err)
	}
	if net != 0 {
		t.Errorf(`XG-10 — **عمولةُ المندوب لم تُعكَس**

  قُيّد %d، وصافي الدفتر بعد الاسترداد %d.
  **والمنصّةُ ردّت للزبون ثمنَه وتركت للمندوب نصيبَه منه** — مالٌ خُلق.

  والعقدُ (RQ-5 · ٢٠٢٦-٠٩-٠٥):
      ORDER REVENUE REVERSED → RELATED REP COMMISSION REVERSED`, paid, net)
	}

	// **ولا دَينَ على من يملك** — رصيدُه كان يكفي.
	var debt int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(commission_debt, 0) FROM users WHERE id = $1::uuid`,
		rep.ID).Scan(&debt)
	if debt != 0 {
		t.Errorf("مندوبٌ رصيدُه يكفي ودُيّن %d", debt)
	}

	assertNewViolations(t, h, base, "FI-04", "FI-05", "FI-06", "FI-09", "FI-11")
}

// TestFIN_XG10_RepWithdrewThenRefund **الحالُ الثانية — سحب فلا يُوقف الزبون.**
//
// **وهي `PG-6` في عقد المندوب**: **العكسُ وحدَه يعيد مرضَ `XG-11`** —
// **مندوبٌ سحب عمولتَه يجعل الاستردادَ مستحيلاً** بقيد الصفر في القاعدة.
func TestFIN_XG10_RepWithdrewThenRefund(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h, "FI-04", "FI-05", "FI-06", "FI-09", "FI-11")
	// **وعمولةُ المندوب افتراضُها صفرٌ عمداً** — **حتّى يضبطها المالك.**
	// **فتُضبَط هنا صراحةً**: السيناريو يشترط عمولةً مقيَّدة.
	h.Setting("merchants.commission_percent", "20")
	h.Setting("sales.commission_percent", "10")
	f := h.Factory()
	rep := h.NewUser("sales")
	item := h.NewItemPriced(f.Merchant(OwnedByRep(rep.ID)), 5000, 8000)
	withMargin(t, h, item, 3000)
	cust := h.Customer()

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 2))
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	var paid int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		  WHERE ref = $1 AND kind = 'commission'`, oid).Scan(&paid)
	if paid <= 0 {
		t.Skip("لم تُقيَّد عمولةٌ للمندوب")
	}

	// **المندوبُ يسحب كلَّ ما عنده.**
	var bal int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`,
		rep.ID).Scan(&bal)
	if bal > 0 {
		f.Credit(rep.ID, -bal, "payout")
	}
	var now int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`,
		rep.ID).Scan(&now)
	if now != 0 {
		t.Fatalf("أردتُ رصيداً صفراً وصار %d", now)
	}
	t.Logf("عمولةٌ %d · سُحبت · الرصيدُ %d", paid, now)

	admin := h.NewUser("admin")
	got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "XG-10 · PG-6"})
	if got.Code >= 400 {
		t.Fatalf(`**حقُّ الزبون سقط برصيد المندوب** — الردّ %d.

  **والعكسُ بلا مسارِ التزامٍ يعيد مرضَ XG-11 من بابٍ آخر.**
  والعقد: **وإن كانت قد سُحبت تُعالَج التزاماً — ولا يُمحى التاريخ.**`, got.Code)
	}

	// ── ولا يُعفى المندوب ─────────────────────────────────────
	var reversed, debt int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT -sum(amount) FROM wallet_transactions
		                  WHERE ref = $1 AND kind = 'commission' AND amount < 0), 0),
		       COALESCE((SELECT commission_debt FROM users WHERE id = $2::uuid), 0)`,
		oid, rep.ID).Scan(&reversed, &debt); err != nil {
		t.Fatalf("قراءةُ الالتزام: %v", err)
	}
	t.Logf("عُكس %d · والتزامٌ %d · والمقيَّدُ كان %d", reversed, debt, paid)
	if reversed+debt < paid {
		t.Errorf(`**عمولةٌ تبخّرت**: قُيّد %d، وعُكس %d، والتزامٌ %d.
  **والفرقُ %d لا يعرف أحدٌ أين ذهب.**`, paid, reversed, debt, paid-reversed-debt)
	}

	var negative int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM wallets WHERE balance < 0 AND NOT is_treasury`).Scan(&negative)
	if negative > 0 {
		t.Errorf("%d محفظةً برصيدٍ سالبٍ وليست خزينة", negative)
	}

	assertNewViolations(t, h, base, "FI-04", "FI-05", "FI-06", "FI-09", "FI-11")
}

// TestFIN_XG10_DebtOffsetFromNextCommission **والالتزامُ يُسوّى لا يُنسى.**
//
// **وهذا تمامُ العقد**: **«تُسوّى من عمولاته المستقبليّة»** — **والتزامٌ
// يُقيَّد ولا يُحصَّل دفترٌ يتضخّم بلا معنى.**
func TestFIN_XG10_DebtOffsetFromNextCommission(t *testing.T) {
	h := New(t)
	treasury(t, h)
	// **وعمولةُ المندوب افتراضُها صفرٌ عمداً** — **حتّى يضبطها المالك.**
	// **فتُضبَط هنا صراحةً**: السيناريو يشترط عمولةً مقيَّدة.
	h.Setting("merchants.commission_percent", "20")
	h.Setting("sales.commission_percent", "10")
	f := h.Factory()
	rep := h.NewUser("sales")
	m := f.Merchant(OwnedByRep(rep.ID))
	item := h.NewItemPriced(m, 5000, 8000)
	withMargin(t, h, item, 3000)

	// ── طلبٌ أوّلُ يُسلَّم ثمّ تُسحب عمولتُه ثمّ يُستردّ ──────
	first := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 2))
	oid1, _ := first.JSON()["id"].(string)
	deliverOrder(t, h, oid1, h.driverOf(oid1))

	var paid1 int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		  WHERE ref = $1 AND kind = 'commission'`, oid1).Scan(&paid1)
	if paid1 <= 0 {
		t.Skip("لم تُقيَّد عمولة")
	}
	var bal int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`,
		rep.ID).Scan(&bal)
	if bal > 0 {
		f.Credit(rep.ID, -bal, "payout")
	}
	admin := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid1+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "XG-10 — التزام"}); got.Code >= 400 {
		t.Fatalf("الاستردادُ سقط: %s", got)
	}
	var debt int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(commission_debt, 0) FROM users WHERE id = $1::uuid`,
		rep.ID).Scan(&debt)
	if debt <= 0 {
		t.Skipf("لا التزامَ قُيّد (%d) — والسيناريو يشترطه", debt)
	}
	t.Logf("الالتزامُ بعد الاسترداد = %d", debt)

	// ── طلبٌ ثانٍ يُسلَّم — فتُقتطَع منه ────────────────────
	second := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 2))
	oid2, _ := second.JSON()["id"].(string)
	deliverOrder(t, h, oid2, h.driverOf(oid2))

	var debtAfter, net2 int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT commission_debt FROM users WHERE id = $1::uuid), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = $2 AND kind = 'commission'), 0)`,
		rep.ID, oid2).Scan(&debtAfter, &net2); err != nil {
		t.Fatalf("قراءةٌ بعد الطلب الثاني: %v", err)
	}
	t.Logf("بعد طلبٍ ثانٍ: الالتزامُ %d (كان %d) · صافي عمولةِ الطلب الثاني %d",
		debtAfter, debt, net2)
	if debtAfter >= debt {
		t.Errorf("الالتزامُ لم يُقتطَع من العمولة القادمة: كان %d وصار %d",
			debt, debtAfter)
	}
}
