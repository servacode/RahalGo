package qa

// الفشلُ الجزئيّ — **`P-6` البنود ٦…١١ و١٥ و٢٣.**
//
// **ولا تُصلَح شيفرةُ منتجٍ هنا** (البند ٣١): ما كشفه حقنُ فشلٍ يُعلَن
// `DEFECT REPRODUCED` أو `RISK CONFIRMED` باسم سجلّه، **ويُعرَض على
// المالك ولا يُجمَّد من تلقائي.**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

// ══════════════════════════════════════════════════════════════════════
// **٦ و٧ · `D2` / `XQ-3` — التحويلُ الجزئيّ**
// ══════════════════════════════════════════════════════════════════════

// convFixture مرشَّحٌ جاهزٌ للتحويل بمندوبه وتصنيفه.
type convFixture struct {
	leadID, phone, categoryID string
	rep                       *Rep
}

func newConvFixture(t *testing.T, h *Harness, f *Factory) convFixture {
	t.Helper()
	var catID string
	if err := h.Pool.QueryRow(ctxBG(),
		`INSERT INTO categories (name, active) VALUES ($1, true) RETURNING id::text`,
		uniq("تصنيف QA ")).Scan(&catID); err != nil {
		t.Fatalf("تصنيف: %v", err)
	}
	rep := f.RepAccount()
	phone := f.NS.Phone()
	lead := newLead(t, h, rep.ID, catID, phone)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = h.Pool.Exec(ctx, `DELETE FROM merchants WHERE phone = $1`, phone)
		_, _ = h.Pool.Exec(ctx, `DELETE FROM categories WHERE id = $1::uuid`, catID)
	})
	return convFixture{leadID: lead, phone: phone, categoryID: catID, rep: rep}
}

// convState الحقيقةُ الباقيةُ بعد التحويل — **تُقرأ من القاعدة كاملةً.**
type convState struct {
	Merchants  int
	LeadStatus string
	LeadLinked bool
	OwnerPwSet bool
	RepReward  int64
	RepBalance int64
	Incentives int
}

func readConvState(t *testing.T, h *Harness, c convFixture) convState {
	t.Helper()
	var s convState
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT count(*) FROM merchants WHERE phone = $1),
		       (SELECT status FROM merchant_leads WHERE id = $2::uuid),
		       (SELECT merchant_id IS NOT NULL FROM merchant_leads WHERE id = $2::uuid),
		       COALESCE((SELECT COALESCE(password_hash,'') <> '' FROM users WHERE phone = $1), false),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                 WHERE user_id = $3::uuid AND kind = 'reward'), 0),
		       COALESCE((SELECT balance FROM wallets WHERE user_id = $3::uuid), 0),
		       (SELECT count(*) FROM incentives WHERE user_id = $3::uuid AND for_target)`,
		c.phone, c.leadID, c.rep.ID).
		Scan(&s.Merchants, &s.LeadStatus, &s.LeadLinked, &s.OwnerPwSet,
			&s.RepReward, &s.RepBalance, &s.Incentives); err != nil {
		t.Fatalf("قراءةُ الحال: %v", err)
	}
	return s
}

func (s convState) log(t *testing.T, label string) {
	t.Helper()
	t.Logf("%-22s متاجرُ=%d · المرشَّحُ=%q · مربوطٌ=%v · كلمةٌ=%v · مكافأةٌ=%d · رصيدُ المندوب=%d · حوافزُ=%d",
		label, s.Merchants, s.LeadStatus, s.LeadLinked, s.OwnerPwSet,
		s.RepReward, s.RepBalance, s.Incentives)
}

// TestFAIL_D2_ConvertLeadPartialStates **البندان ٦ و٧ — مصفوفةُ التحويل.**
//
// **وخطواتُ `convertLead` مقيسةٌ من الشيفرة** (`leads_handlers.go:774…816`):
//
//	١ CreateMerchant                 ← خطؤها يُردّ
//	٢ UPDATE users … full_name       ← خطؤها مُهمَل
//	٣ UPDATE users … password_hash   ← خطؤها مُهمَل
//	٤ UPDATE merchants … district_id ← خطؤها مُهمَل
//	٥ grantSalesTargetIfAny          ← **تدفع مالاً** · خطؤها مُهمَل
//	٦ UPDATE merchant_leads … converted ← **التثبيت** · خطؤها يُردّ
//
// **والسبعُ كلُّها بلا معاملةٍ واحدة.**
func TestFAIL_D2_ConvertLeadPartialStates(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	admin := h.NewUser("admin")
	h.Setting("sales.monthly_target", "1")
	h.Setting("sales.target_reward", "5000")

	convert := func(c convFixture) Res {
		return h.POST("/api/v1/admin/leads/"+c.leadID+"/status", admin.Token,
			map[string]any{"status": "converted"})
	}

	// ── الأساسُ بلا فشل ────────────────────────────────────────────
	t.Run("بلا فشل", func(t *testing.T) {
		c := newConvFixture(t, h, f)
		got := convert(c)
		s := readConvState(t, h, c)
		s.log(t, "NO FAILURE")
		if got.Code >= 400 {
			t.Fatalf("التحويلُ رُدّ: %s", got)
		}
		if s.Merchants != 1 || s.LeadStatus != "converted" || !s.LeadLinked {
			t.Errorf("الأساسُ نفسُه غيرُ سليم")
		}
	})

	// ── الفشلُ بعد الخطوة ١ (إنشاءُ المتجر) ─────────────────────────
	t.Run("فشلٌ في إنشاء المتجر", func(t *testing.T) {
		c := newConvFixture(t, h, f)
		fp := h.Arm("D2/step-1-create-merchant", "merchants", "INSERT", 1, "phone", c.phone)
		got := convert(c)
		fp.MustFire(t)
		s := readConvState(t, h, c)
		s.log(t, "FAIL AT STEP 1")
		t.Logf("الردّ: %d %s", got.Code, got.Err())
		if s.Merchants != 0 {
			t.Errorf("متجرٌ بقي رغم فشل إنشائه: %d", s.Merchants)
		}
		if s.LeadStatus != "new" {
			t.Errorf("حالُ المرشَّح %q بعد فشلٍ في الخطوة الأولى", s.LeadStatus)
		}
		if s.RepReward != 0 {
			t.Errorf("MONEY BEFORE WORK: مكافأةٌ %d ولا متجرَ أُنشئ", s.RepReward)
		}
		if got.Code < 400 {
			t.Errorf("رُدّ نجاحٌ ولا متجرَ — **كذبٌ على الإدارة**")
		}
		if s.Merchants == 0 && s.LeadStatus == "new" && got.Code >= 400 {
			t.Logf("STEP-1 FAILURE = CLEAN — لا أثرَ ولا مالَ · والردُّ خطأ · وRETRY ممكن")
		}
	})

	// ── الفشلُ في الخطوة ٦ — **الحالُ التي منعها `XQ-3` نصّاً** ──────
	t.Run("فشلٌ في تثبيت التحويل", func(t *testing.T) {
		c := newConvFixture(t, h, f)
		before := readConvState(t, h, c)
		before.log(t, "BEFORE")

		// **النقطةُ على تثبيت المرشَّح وحدَه** — بعد المتجر وبعد المكافأة.
		fp := h.Arm("D2/step-6-commit-conversion", "merchant_leads", "UPDATE", 1, "id", c.leadID)
		got := convert(c)
		fp.MustFire(t)

		s := readConvState(t, h, c)
		s.log(t, "AFTER FAIL AT STEP 6")
		t.Logf("الردّ: %d %s", got.Code, got.Err())

		forbidden := s.Merchants > 0 && s.RepReward > 0 && s.LeadStatus != "converted"
		if forbidden {
			t.Logf("XQ-3 FORBIDDEN STATE = PROVEN")
			t.Logf("  PARTIAL CONVERSION + REWARD PAID + CONVERSION NOT COMMITTED")
			t.Logf("  المتجرُ %d · والمكافأةُ %d · وحالُ المرشَّح %q",
				s.Merchants, s.RepReward, s.LeadStatus)
			t.Logf("D2 PARTIAL CONVERSION = DEFECT REPRODUCED")
		} else if s.Merchants > 0 && s.LeadStatus != "converted" {
			t.Logf("PARTIAL — متجرٌ أُنشئ والمرشَّحُ %q · ولا مكافأةَ (العتبةُ لم تُبلَغ)",
				s.LeadStatus)
			t.Logf("D2 PARTIAL CONVERSION = DEFECT REPRODUCED (بلا شقِّ المال)")
		} else {
			t.Errorf("لم تقع حالٌ جزئيّةٌ — **وD2 يقول إنّها تقع. يُراجَع.**")
		}

		// ── والتعافي (البند ٢٣) ─────────────────────────────────────
		retry := convert(c)
		after := readConvState(t, h, c)
		after.log(t, "AFTER RETRY")
		t.Logf("الإعادة: %d", retry.Code)
		if after.Merchants > 1 {
			t.Logf("RETRY SAFETY = BROKEN — الإعادةُ أنشأت متجراً ثانياً (%d)", after.Merchants)
		} else if after.LeadStatus == "converted" {
			t.Logf("RETRY SAFETY = RECOVERS — الإعادةُ ثبّتت التحويلَ بلا متجرٍ ثانٍ")
		}
		if after.RepReward > before.RepReward*2 && before.RepReward > 0 {
			t.Errorf("DOUBLE REWARD: المكافأةُ صارت %d", after.RepReward)
		}

		// **والحَكَمُ محرّكُ `P-4`** (البند ٢٢).
		assertNewViolations(t, h, financialBaseline(t, h, "FI-03", "FI-05"), "FI-03", "FI-05")
	})
}

// ══════════════════════════════════════════════════════════════════════
// **٨ · `D5` — المصروفُ والخزينة**
// ══════════════════════════════════════════════════════════════════════

// TestFAIL_D5_ExpenseWithoutTreasuryDebit **البند ٨.**
//
// **وخطوتان مقيستان** (`expenses_handlers.go:262`):
//
//	١ INSERT INTO expenses          ← يُردّ خطؤها
//	٢ wallet.Apply(operating_expense) ← يُردّ خطؤها **بعد أن وقعت الأولى**
//
// **وسقوطُ الثانية يترك مصروفاً بلا خصم** — وهو ما ادّعاه `D5` ساكناً،
// **ويُثبَت هنا بحقنِ فشلٍ حيّ.**
func TestFAIL_D5_ExpenseTreasuryPartial(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")

	var catID string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO expense_categories (name, sort_order, active)
		VALUES ($1, 1, true) RETURNING id::text`, uniq("بندُ مصروف QA ")).Scan(&catID); err != nil {
		t.Fatalf("بندُ المصروف: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = h.Pool.Exec(ctx, `DELETE FROM expenses WHERE category_id = $1::uuid`, catID)
		_, _ = h.Pool.Exec(ctx, `DELETE FROM expense_categories WHERE id = $1::uuid`, catID)
	})

	base := financialBaseline(t, h)

	// **النقطةُ على قيدِ الخزينة وحدَه** — بالنوع، فلا تُصيب قيداً آخر.
	fp := h.Arm("D5/step-2-treasury-debit", "wallet_transactions", "INSERT",
		1, "kind", "operating_expense")

	got := h.POST("/api/v1/admin/expenses", admin.Token,
		map[string]any{"category_id": catID, "amount": 33_000, "note": "P-6"})
	fp.MustFire(t)
	t.Logf("الردّ: %d %s", got.Code, got.Err())

	var expenses, ledger int
	var posted int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT count(*) FROM expenses WHERE category_id = $1::uuid),
		       (SELECT count(*) FROM wallet_transactions t JOIN expenses e ON e.id::text = t.ref
		        WHERE e.category_id = $1::uuid),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t JOIN expenses e ON e.id::text = t.ref
		        WHERE e.category_id = $1::uuid), 0)`, catID).Scan(&expenses, &ledger, &posted)
	t.Logf("مصاريفُ=%d · قيودٌ=%d · مجموعُها=%d", expenses, ledger, posted)

	if expenses == 1 && ledger == 0 {
		t.Logf("D5 TREASURY/EXPENSE = DEFECT REPRODUCED")
		t.Logf("  مصروفٌ مسجَّلٌ بلا خصمِ خزينة — **وتقريرُ الأرباح يقول ربحاً لم يقع**")
		if got.Code < 400 {
			t.Logf("  ADMIN VISIBILITY = INVISIBLE — رُدَّ نجاحٌ ولا شيءَ يقول إنّ نصفَ العمليّة سقط")
		} else {
			t.Logf("  ADMIN VISIBILITY = PARTIAL — رُدَّ %d · والمصروفُ باقٍ في القائمة", got.Code)
		}
	} else if expenses == 0 {
		t.Errorf("لم يبقَ مصروفٌ — **وD5 يقول إنّ الأولى تقع والثانيةَ تسقط. يُراجَع.**")
	}

	// **والحارسُ يمسك الحالَ** — وهو ما بُني في `P-4`.
	vs := newViolations(h, base, "FI-04.d")
	if len(vs) > 0 {
		t.Logf("FI CHECK AFTER FAILURE = CAUGHT — %s", vs[0])
	} else {
		t.Errorf("FI-04.d لم يمسك مصروفاً بلا خصم — **الحارسُ أعمى**")
	}

	// ── والتعافي ────────────────────────────────────────────────────
	retry := h.POST("/api/v1/admin/expenses", admin.Token,
		map[string]any{"category_id": catID, "amount": 33_000, "note": "P-6 إعادة"})
	var expenses2, ledger2 int
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT count(*) FROM expenses WHERE category_id = $1::uuid),
		       (SELECT count(*) FROM wallet_transactions t JOIN expenses e ON e.id::text = t.ref
		        WHERE e.category_id = $1::uuid)`, catID).Scan(&expenses2, &ledger2)
	t.Logf("بعد الإعادة: مصاريفُ=%d · قيودٌ=%d · الردّ=%d", expenses2, ledger2, retry.Code)
	if expenses2 > expenses && ledger2 == ledger+1 {
		t.Logf("RETRY = NEW EXPENSE — **والأوّلُ اليتيمُ باقٍ ولا مسارَ يُصلحه**")
	}
}

// newViolations ما استجدّ — **ولا تُسقط**: الخرقُ هنا هو المقصود.
func newViolations(h *Harness, base fininv.Baseline, ids ...string) []fininv.Violation {
	vs, err := fininv.Run(ctxBG(), h.Pool, ids...)
	if err != nil {
		return nil
	}
	return base.New(vs)
}
