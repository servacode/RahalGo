package qa

import "testing"

// ══════════════════════════════════════════════════════════════════════
// **`XG-11` — حقُّ الاسترداد لا يُشترَط برصيدِ من بعده**
// ══════════════════════════════════════════════════════════════════════
//
// # العقدُ المعتمد (قرارُ المالك في إغلاق ما قبل `P-5`)
//
//	CUSTOMER REFUND ENTITLEMENT MUST NOT DEPEND ON CURRENT
//	DOWNSTREAM ACTOR BALANCES
//
// # ولماذا حارسٌ يؤكّد لا اختبارٌ يسجّل
//
// **كان في المِسنَد اختبارٌ يقيس الحالَ ويكتبه في السجلّ** —
// `TestFIN_MerchantWithdrewThenRefund` — **ويمرّ في الحالين.** وذلك
// صحيحٌ لغرضه (`P-4` كان يجرد لا يُصلح)، **ولا يصلح حارسَ انحدار**:
// **لو أُصلح العيبُ ثمّ عاد لَمرَّ الاختبارُ صامتاً.**
//
// **وهذا يؤكّد العقدَ نفسَه** — فيسقط إن انكسر، اليومَ أو بعد سنة.

// TestFIN_XG11_RefundIndependentOfMerchantBalance **حارسُ العقد.**
//
// # السيناريو
//
//	طلبٌ يُسلَّم           ⇒ يُقيَّد مستحقُّ المتجر
//	المتجرُ يسحب مستحقَّه ⇒ رصيدُه صفر
//	الإدارةُ تستردّ       ⇒ **يجب أن يصل الزبونَ مالُه كاملاً**
//
// **والمتجرُ لا يُعفى**: ما عجز عنه رصيدُه يبقى ديناً عليه يُقتطَع من
// أوّل مستحقٍّ قادم — **وهي آليّةٌ قائمةٌ ومُثبَتةٌ في مسار ردّ البضاعة**
// (`goods.go`)، **ولم تكن في مسار الاسترداد.**
func TestFIN_XG11_RefundIndependentOfMerchantBalance(t *testing.T) {
	h := New(t)
	treasury(t, h)
	// **والحَكَمُ محرّكُ `P-4` لا توكيدٌ ماليٌّ أكتبه** — **والقاعدةُ
	// مشتركةٌ فيها ركامُ سيناريوهاتٍ سابقة**، فيُصوَّر القائمُ ثمّ
	// يُسأل: **ما الذي جدَّ بفعلي أنا؟**
	base := financialBaseline(t, h, "FI-04", "FI-05", "FI-06", "FI-11")
	f := h.Factory()
	m := f.Merchant()
	item := h.NewItemFor(m, 4000)
	cust := h.Customer()

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	// ── ما قُيّد للمتجر فعلاً ─────────────────────────────────
	var owner string
	var earned int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT m.owner_user_id::text,
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = $2 AND kind = 'merchant_earning'), 0)
		FROM merchants m WHERE m.id = $1::uuid`, m.ID, oid).Scan(&owner, &earned); err != nil {
		t.Fatalf("قراءةُ المستحقّ: %v", err)
	}
	if earned <= 0 {
		t.Skip("لم يُقيَّد مستحقٌّ للمتجر — والسيناريو يشترطه")
	}

	// ── وما دفعه الزبونُ من محفظته ────────────────────────────
	//
	// **وهو ما يجب أن يعود إليه** — لا أقلّ.
	var walletPaid int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(wallet_paid, 0) FROM orders WHERE id = $1::uuid`,
		oid).Scan(&walletPaid); err != nil {
		t.Fatalf("قراءةُ المدفوع: %v", err)
	}

	// ── المتجرُ يسحب مستحقَّه فيصير رصيدُه صفراً ───────────────
	var bal int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`,
		owner).Scan(&bal); err != nil {
		t.Fatalf("قراءةُ الرصيد: %v", err)
	}
	if bal > 0 {
		f.Credit(owner, -bal, "payout")
	}
	var after int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1::uuid`,
		owner).Scan(&after)
	if after != 0 {
		t.Fatalf("أردتُ رصيداً صفراً وصار %d — والسيناريو لا يقوم", after)
	}
	t.Logf("مستحقُّ المتجر %d · سُحب · الرصيدُ الآن %d", earned, after)

	// ══════════════════════════════════════════════════════════════
	// **العقد**
	// ══════════════════════════════════════════════════════════════
	admin := h.NewUser("admin")
	got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "XG-11 — حارسُ العقد"})

	if got.Code >= 400 {
		t.Fatalf(`XG-11 — **حقُّ الزبون سقط برصيدِ طرفٍ ثالث**

  الاستردادُ ردَّ %d بعد أن سحب المتجرُ مستحقَّه (%d).
  **والزبونُ لم يفعل شيئاً** — ودفع من محفظته %d.

  والعقدُ المعتمد:
      CUSTOMER REFUND ENTITLEMENT MUST NOT DEPEND ON
      CURRENT DOWNSTREAM ACTOR BALANCES

  الردّ: %s`, got.Code, earned, walletPaid, got)
	}

	// ── وصل المالُ فعلاً؟ ─────────────────────────────────────
	//
	// **ولا يكفي أن يردّ البابُ `200`** — **استردادٌ بصفرٍ نجاحٌ كاذب.**
	var refunded int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		  WHERE ref = $1 AND kind = 'refund'`, oid).Scan(&refunded); err != nil {
		t.Fatalf("قراءةُ المسترَدّ: %v", err)
	}
	if walletPaid > 0 && refunded < walletPaid {
		t.Errorf("المستردُّ %d ودفع الزبونُ %d — **ونقصٌ في حقّه**", refunded, walletPaid)
	}
	t.Logf("المستردُّ للزبون = %d (دفع %d)", refunded, walletPaid)

	// ── والمتجرُ لا يُعفى ─────────────────────────────────────
	//
	// **ما عجز عنه رصيدُه يبقى ديناً** — ولا يُمحى بعجزه.
	var debt, reversed int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT debt FROM merchants WHERE id = $1::uuid), 0),
		       COALESCE((SELECT -sum(amount) FROM wallet_transactions
		                  WHERE ref = $2 AND kind = 'merchant_earning'
		                    AND amount < 0), 0)`,
		m.ID, oid).Scan(&debt, &reversed); err != nil {
		t.Fatalf("قراءةُ الدَّين: %v", err)
	}
	t.Logf("عُكس من المحفظة %d · وقُيّد ديناً %d · والمستحقُّ كان %d",
		reversed, debt, earned)
	if reversed+debt < earned {
		t.Errorf(`**مستحقٌّ تبخّر**: قُيّد %d، وعُكس %d، ودُيّن %d.
  **والفرقُ %d لا يعرف أحدٌ أين ذهب.**`,
			earned, reversed, debt, earned-reversed-debt)
	}

	// ── ولا رصيدَ سالبٌ لغير الخزينة ──────────────────────────
	//
	// **وهو قيدُ القاعدة نفسُه** — يُسأل صراحةً لأنّ الإصلاحَ يمسّه.
	var negative int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM wallets WHERE balance < 0 AND NOT is_treasury`).
		Scan(&negative); err != nil {
		t.Fatalf("فحصُ السالب: %v", err)
	}
	if negative > 0 {
		t.Errorf("%d محفظةً برصيدٍ سالبٍ وليست خزينة", negative)
	}

	// ══════════════════════════════════════════════════════════════
	// **وحكمُ `P-4` على ما جدَّ — بنطاقٍ مقيسٍ لا شامل**
	// ══════════════════════════════════════════════════════════════
	//
	// **والإصلاحُ يمسّ الدفتر** — **فلا يُقبَل بقولي إنّه سليم.**
	//
	// # ولماذا لا يُسأل المحرّكُ كلُّه
	//
	// **في الحزمة نفسِها اختباراتٌ تحقن فساداً عمداً** — `P-4` بناها
	// لتُثبت أنّ المحرّكَ يكشف: **رصيدٌ يُفسَد بيده** (`FI-02.a`)
	// **وقيدٌ يُكرَّر** (`FI-05.c`) **وصندوقٌ يُخالف قيودَه**
	// (`FI-10.a`). **وهي تفعل ذلك بقصد، وهو صوابُها.**
	//
	// **والقاعدةُ مشتركة** — فيصل أثرُها إلى من بعدها. **وسؤالُ
	// المحرّك كلِّه هنا يجعل حارسي يحمرّ بفعل غيره**، وذلك أسوأُ من
	// ألّا يسأل: **حمرةٌ لا يملك صاحبُها إصلاحَها تُطفَأ بالتجاهل.**
	//
	// # وما يُسأل عنه
	//
	// **ما يمسُّه هذا الإصلاحُ بعينه**: تغطيةُ الطلب بمستحقٍّ
	// (`FI-04`) · **والتكرار** (`FI-05`) · **وحفظُ اقتصاد الطلب**
	// (`FI-06`) · **ونزاهةُ الاسترداد** (`FI-11`).
	//
	// **وأمّا الأرصدةُ والصناديقُ فمفحوصةٌ أعلاه بصفوف هذا الاختبار
	// وحدَها** — **وذلك أدقُّ من مسحٍ عامٍّ في قاعدةٍ ملوَّثة.**
	assertNewViolations(t, h, base, "FI-04", "FI-05", "FI-06", "FI-11")
}

// TestFIN_XG11_RefundUntouchedWhenMerchantSolvent **والحالُ السليمةُ تبقى.**
//
// **ومتجرٌ لم يسحب يُعكَس مستحقُّه كاملاً بلا دَين** — **فالإصلاحُ لا
// يفتح باباً لتديينِ من يملك.**
func TestFIN_XG11_RefundUntouchedWhenMerchantSolvent(t *testing.T) {
	h := New(t)
	treasury(t, h)
	f := h.Factory()
	m := f.Merchant()
	item := h.NewItemFor(m, 4000)
	cust := h.Customer()

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	var earned int64
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		  WHERE ref = $1 AND kind = 'merchant_earning'`, oid).Scan(&earned)
	if earned <= 0 {
		t.Skip("لم يُقيَّد مستحقٌّ")
	}

	admin := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "XG-11 — الحالُ السليمة"}); got.Code >= 400 {
		t.Fatalf("استردادٌ من متجرٍ مليءٍ سقط: %s", got)
	}

	var debt, net int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT debt FROM merchants WHERE id = $1::uuid), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE ref = $2 AND kind = 'merchant_earning'), 0)`,
		m.ID, oid).Scan(&debt, &net); err != nil {
		t.Fatal(err)
	}
	if debt != 0 {
		t.Errorf("متجرٌ مليءٌ ودُيّن %d — **والدَّينُ لمن عجز لا لمن يملك**", debt)
	}
	if net != 0 {
		t.Errorf("صافي مستحقّ المتجر بعد العكس %d ولا صفر", net)
	}
}
