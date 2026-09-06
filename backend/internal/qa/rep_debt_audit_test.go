package qa

import "testing"

// ══════════════════════════════════════════════════════════════════════
// **تدقيقُ دورة الإصلاح ٢ — `XG-10` بأرقامه**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يكفي أن يمرّ الحارس** — **المالُ يُعَدُّ قبلَ وبعد.**

// repLedger صورةُ حالِ مندوبٍ وطلبٍ — **من الدفتر لا من المعادلة.**
type repLedger struct {
	Balance    int64
	Debt       int64
	OrderNet   int64
	OrderRows  int
	RefundPaid int64
}

func snapRep(t *testing.T, h *Harness, repID, orderID string) repLedger {
	t.Helper()
	var s repLedger
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT balance FROM wallets WHERE user_id = $1::uuid), 0),
		       COALESCE((SELECT commission_debt FROM users WHERE id = $1::uuid), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = $2 AND kind = 'commission' AND user_id = $1::uuid), 0),
		       (SELECT count(*) FROM wallet_transactions
		         WHERE ref = $2 AND kind = 'commission' AND user_id = $1::uuid),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = $2 AND kind = 'refund'), 0)`,
		repID, orderID).Scan(&s.Balance, &s.Debt, &s.OrderNet, &s.OrderRows, &s.RefundPaid); err != nil {
		t.Fatalf("قراءةُ الحال: %v", err)
	}
	return s
}

// repFixture مندوبٌ ومتجرُه وصنفٌ بهامشٍ معلوم.
func repFixture(t *testing.T, h *Harness) (rep *User, item *Item) {
	t.Helper()
	h.Setting("merchants.commission_percent", "20")
	h.Setting("sales.commission_percent", "10")
	f := h.Factory()
	rep = h.NewUser("sales")
	item = h.NewItemPriced(f.Merchant(OwnedByRep(rep.ID)), 5000, 8000)
	withMargin(t, h, item, 3000)
	return rep, item
}

func placeAndDeliver(t *testing.T, h *Harness, item *Item) string {
	t.Helper()
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 2))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	deliverOrder(t, h, oid, h.driverOf(oid))
	return oid
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · إعادةُ نداء الاسترداد لا تُضاعف شيئاً**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا استردادَ جزئيٌّ في المعمار** — `refunded` تأتي من `delivered`
// وحدَها وهي نهائيّةٌ بلا مخرج (`statuses.go`). **فالتكرارُ هو الخطرُ
// الوحيدُ الممكن، وهو ما يُختبَر.**
func TestFIN_XG10_RefundReplayDoesNotDoubleCharge(t *testing.T) {
	h := New(t)
	treasury(t, h)
	rep, item := repFixture(t, h)
	oid := placeAndDeliver(t, h, item)

	before := snapRep(t, h, rep.ID, oid)
	if before.OrderNet <= 0 {
		t.Skip("لم تُقيَّد عمولة")
	}
	admin := h.NewUser("admin")
	body := map[string]any{"to": "refunded", "note": "تكرار"}

	first := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token, body)
	if first.Code >= 400 {
		t.Fatalf("الاستردادُ الأوّل: %s", first)
	}
	after1 := snapRep(t, h, rep.ID, oid)

	// **النداءُ الثاني بالجسم نفسِه** — والحالُ صارت `refunded`.
	second := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token, body)
	after2 := snapRep(t, h, rep.ID, oid)

	t.Logf("قبل: عمولةٌ %d · بعد الأوّل: صافٍ %d دَينٌ %d قيودٌ %d · "+
		"بعد الثاني (%d): صافٍ %d دَينٌ %d قيودٌ %d · المستردُّ %d ← %d",
		before.OrderNet, after1.OrderNet, after1.Debt, after1.OrderRows,
		second.Code, after2.OrderNet, after2.Debt, after2.OrderRows,
		after1.RefundPaid, after2.RefundPaid)

	if second.Code < 400 {
		t.Errorf("نداءُ استردادٍ ثانٍ قُبل (%d) — **و`refunded` نهائيّة**", second.Code)
	}
	if after2.OrderRows != after1.OrderRows {
		t.Errorf("قيودُ العمولة تضاعفت: %d ← %d", after1.OrderRows, after2.OrderRows)
	}
	if after2.Debt != after1.Debt {
		t.Errorf("**الدَّينُ زاد بإعادة النداء**: %d ← %d — "+
			"**وإعادةُ نداءٍ لا تصير باباً لتديين المندوب**", after1.Debt, after2.Debt)
	}
	if after2.RefundPaid != after1.RefundPaid {
		t.Errorf("المستردُّ تضاعف: %d ← %d", after1.RefundPaid, after2.RefundPaid)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · التسويةُ من العمولات القادمة — بالأرقام**
// ══════════════════════════════════════════════════════════════════════
func TestFIN_XG10_DebtSettlementArithmetic(t *testing.T) {
	h := New(t)
	treasury(t, h)
	rep, item := repFixture(t, h)

	// ── طلبٌ أوّل: عمولةٌ تُسحب ثمّ يُستردّ ⇒ دَين ────────────
	oid1 := placeAndDeliver(t, h, item)
	s0 := snapRep(t, h, rep.ID, oid1)
	if s0.OrderNet <= 0 {
		t.Skip("لم تُقيَّد عمولة")
	}
	commission := s0.OrderNet
	h.Factory().Credit(rep.ID, -s0.Balance, "payout")

	admin := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid1+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "دَين"}); got.Code >= 400 {
		t.Fatalf("الاستردادُ سقط: %s", got)
	}
	s1 := snapRep(t, h, rep.ID, oid1)
	t.Logf("١) عمولةٌ %d · سُحبت · استُرِدّ ⇒ دَينٌ %d · رصيدٌ %d",
		commission, s1.Debt, s1.Balance)
	if s1.Debt != commission {
		t.Fatalf("الدَّينُ %d والعمولةُ كانت %d", s1.Debt, commission)
	}

	// ── طلبٌ ثانٍ: عمولةٌ تُقيَّد ثمّ يُقتطَع منها ───────────
	oid2 := placeAndDeliver(t, h, item)
	s2 := snapRep(t, h, rep.ID, oid2)
	t.Logf("٢) عمولةٌ جديدةٌ ⇒ صافي الطلب %d · دَينٌ %d · رصيدٌ قابلٌ للسحب %d",
		s2.OrderNet, s2.Debt, s2.Balance)

	// **والمعادلة**: دَينٌ سابقٌ − ما اقتُطع = الباقي.
	settled := s1.Debt - s2.Debt
	if settled <= 0 {
		t.Fatalf("لم يُقتطَع شيءٌ من الدَّين: %d ← %d", s1.Debt, s2.Debt)
	}
	// **والرصيدُ القابلُ للسحب = العمولةُ الجديدةُ − ما اقتُطع.**
	var gross int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'commission' AND amount > 0 AND user_id = $2::uuid`,
		oid2, rep.ID).Scan(&gross)
	if want := gross - settled; s2.Balance != want {
		t.Errorf("الرصيدُ %d والمتوقَّع %d (عمولةٌ %d − اقتطاعٌ %d)",
			s2.Balance, want, gross, settled)
	}
	t.Logf("   عمولةٌ خامٌّ %d · اقتُطع %d · بقي للسحب %d · ودَينٌ %d",
		gross, settled, s2.Balance, s2.Debt)

	// ── دليلُ التسوية في الدفتر ────────────────────────────────
	//
	// **ولا تسويةَ بلا سطرٍ يقولها.**
	var offsetRows int
	var offsetNote string
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT count(*), COALESCE(max(note), '')
		FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'commission' AND amount < 0 AND user_id = $2::uuid`,
		oid2, rep.ID).Scan(&offsetRows, &offsetNote)
	if offsetRows == 0 {
		t.Errorf("لا سطرَ في الدفتر يقول إنّ اقتطاعاً وقع")
	}
	t.Logf("   دليلُ التسوية: %d سطراً · %q", offsetRows, offsetNote)
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · متجرٌ عاجزٌ ومندوبٌ سحب — والزبونُ يستردّ**
// ══════════════════════════════════════════════════════════════════════
func TestFIN_XG10_CombinedMerchantAndRepInsufficiency(t *testing.T) {
	h := New(t)
	treasury(t, h)
	rep, item := repFixture(t, h)
	oid := placeAndDeliver(t, h, item)

	var ownerID string
	var mID string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT m.owner_user_id::text, m.id::text
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1::uuid`, oid).Scan(&ownerID, &mID); err != nil {
		t.Fatalf("قراءةُ المتجر: %v", err)
	}

	var merchantEarned, repPaid, walletPaid, cashDue int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = $1 AND kind = 'merchant_earning'), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = $1 AND kind = 'commission'), 0),
		       COALESCE((SELECT wallet_paid FROM orders WHERE id = $1::uuid), 0),
		       COALESCE((SELECT cash_due FROM orders WHERE id = $1::uuid), 0)`,
		oid).Scan(&merchantEarned, &repPaid, &walletPaid, &cashDue); err != nil {
		t.Fatalf("قراءةُ الأنصبة: %v", err)
	}
	if merchantEarned <= 0 || repPaid <= 0 {
		t.Skipf("السيناريو يشترط الاثنين: متجرٌ %d · مندوبٌ %d", merchantEarned, repPaid)
	}

	// **الطرفان يسحبان** — فلا رصيدَ عند أيٍّ منهما.
	f := h.Factory()
	for _, u := range []string{ownerID, rep.ID} {
		var b int64
		_ = h.Pool.QueryRow(ctxBG(),
			`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`, u).Scan(&b)
		if b > 0 {
			f.Credit(u, -b, "payout")
		}
	}
	t.Logf("مستحقُّ المتجر %d · عمولةُ المندوب %d · وكلاهما سحب", merchantEarned, repPaid)

	admin := h.NewUser("admin")
	got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "عجزٌ مركَّب"})
	if got.Code >= 400 {
		t.Fatalf(`**حقُّ الزبون سقط بعجزِ طرفين** — الردّ %d.
  **وعجزُ المندوب لا يجوز أن يُعيد XG-11 من بابٍ خلفيّ.**`, got.Code)
	}

	var refunded, mDebt, rDebt int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = $1 AND kind = 'refund'), 0),
		       COALESCE((SELECT debt FROM merchants WHERE id = $2::uuid), 0),
		       COALESCE((SELECT commission_debt FROM users WHERE id = $3::uuid), 0)`,
		oid, mID, rep.ID).Scan(&refunded, &mDebt, &rDebt); err != nil {
		t.Fatalf("قراءةُ النتيجة: %v", err)
	}
	t.Logf("المستردُّ %d (دفع محفظةً %d ونقداً %d) · دَينُ المتجر %d · التزامُ المندوب %d",
		refunded, walletPaid, cashDue, mDebt, rDebt)

	if want := walletPaid + cashDue; want > 0 && refunded != want {
		t.Errorf("المستردُّ %d والمتوقَّع %d", refunded, want)
	}
	if mDebt < merchantEarned {
		t.Errorf("دَينُ المتجر %d ومستحقُّه كان %d", mDebt, merchantEarned)
	}
	if rDebt < repPaid {
		t.Errorf("التزامُ المندوب %d وعمولتُه كانت %d", rDebt, repPaid)
	}

	var negative int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM wallets WHERE balance < 0 AND NOT is_treasury`).Scan(&negative)
	if negative > 0 {
		t.Errorf("%d محفظةً برصيدٍ سالبٍ وليست خزينة", negative)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٧ · حفظُ المال — لا يُخلَق ولا يضيع**
// ══════════════════════════════════════════════════════════════════════
//
// **والمعادلةُ الواحدة**: **ما خرج من الأطراف عائداً + ما صار التزاماً
// = ما كان قد قُيّد لهم.**
func TestFIN_XG10_ConservationAcrossRefund(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h, "FI-04", "FI-05", "FI-06", "FI-09", "FI-11")
	rep, item := repFixture(t, h)
	oid := placeAndDeliver(t, h, item)

	type side struct{ merchant, commission, refund, treasury int64 }
	read := func() side {
		var s side
		if err := h.Pool.QueryRow(ctxBG(), `
			SELECT COALESCE((SELECT sum(amount) FROM wallet_transactions
			                  WHERE ref = $1 AND kind = 'merchant_earning'), 0),
			       COALESCE((SELECT sum(amount) FROM wallet_transactions
			                  WHERE ref = $1 AND kind = 'commission'), 0),
			       COALESCE((SELECT sum(amount) FROM wallet_transactions
			                  WHERE ref = $1 AND kind = 'refund'), 0),
			       COALESCE((SELECT sum(amount) FROM wallet_transactions t
			                  JOIN wallets w ON w.user_id = t.user_id
			                 WHERE t.ref = $1 AND w.is_treasury), 0)`,
			oid).Scan(&s.merchant, &s.commission, &s.refund, &s.treasury); err != nil {
			t.Fatalf("قراءة: %v", err)
		}
		return s
	}
	before := read()
	if before.commission <= 0 {
		t.Skip("لم تُقيَّد عمولة")
	}
	t.Logf("قبل: متجرٌ %d · عمولةٌ %d · استردادٌ %d · خزينةٌ %d",
		before.merchant, before.commission, before.refund, before.treasury)

	admin := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "حفظُ المال"}); got.Code >= 400 {
		t.Fatalf("الاسترداد: %s", got)
	}
	after := read()

	var mDebt, rDebt, platformCommission int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT m.debt FROM merchants m
		                  JOIN orders o ON o.merchant_id = m.id WHERE o.id = $1::uuid), 0),
		       COALESCE((SELECT commission_debt FROM users WHERE id = $2::uuid), 0),
		       COALESCE((SELECT platform_commission FROM orders WHERE id = $1::uuid), 0)`,
		oid, rep.ID).Scan(&mDebt, &rDebt, &platformCommission)

	t.Logf("بعد: متجرٌ %d · عمولةٌ %d · استردادٌ %d · خزينةٌ %d · "+
		"دَينُ متجرٍ %d · التزامُ مندوبٍ %d · عمولةُ منصّةٍ %d",
		after.merchant, after.commission, after.refund, after.treasury,
		mDebt, rDebt, platformCommission)

	// ══════════════════════════════════════════════════════════════
	// **ومعادلةُ الحفظ**: ما عُكس + ما صار التزاماً = ما قُيّد
	// ══════════════════════════════════════════════════════════════
	//
	// **وما عُكس هو نقصانُ الصافي** — `قبل − بعد`. **فصافٍ صفرٌ يعني
	// عكساً كاملاً، لا تبخّراً.**
	//
	// **وكانت هذه المعادلةُ مقلوبةً في أوّل صياغة** (`بعد + دَين <
	// قبل`) **فقرأت العكسَ الكاملَ ضياعاً** — **والأرقامُ نفسُها كانت
	// تقول إنّ المالَ محفوظ.**
	if reversed := before.merchant - after.merchant; reversed+mDebt < before.merchant {
		t.Errorf("مستحقُّ متجرٍ تبخّر: قُيّد %d · عُكس %d · دَينٌ %d",
			before.merchant, reversed, mDebt)
	}
	if reversed := before.commission - after.commission; reversed+rDebt < before.commission {
		t.Errorf("عمولةُ مندوبٍ تبخّرت: قُيّدت %d · عُكس %d · التزامٌ %d",
			before.commission, reversed, rDebt)
	}
	// **وعمولةُ المنصّة تُصفَّر** — طلبٌ لم يقع لا ربحَ فيه.
	if platformCommission != 0 {
		t.Errorf("عمولةُ المنصّة بقيت %d على طلبٍ مُسترجَع", platformCommission)
	}

	assertNewViolations(t, h, base, "FI-04", "FI-05", "FI-06", "FI-09", "FI-11")
}
