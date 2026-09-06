package qa

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **`XG-32` — مكافأةُ الهدف تُحسَب بمعاملتِها**
// ══════════════════════════════════════════════════════════════════════
//
// **العدُّ كان يقرأ من المَسبَح** — **والمتجرُ الذي يُنشأ داخلَ معاملة
// التحويل غيرُ مُثبَّتٍ بعدُ فلا يراه.** فتتأخّر المكافأةُ تحويلاً
// كاملاً، **ومن بلغ هدفَه بالضبط ووقف لا يُكافأ أبداً.**
//
// **والعقدُ لم يتبدّل**: من حقّق هدفَه الشهريَّ تُقيَّد مكافأتُه
// **مرّةً واحدةً بالضبط.**

// convertLeadFor يحوّل مرشَّحاً جديداً لمندوبٍ ويُرجع رمزَ الردّ.
func convertLeadFor(t *testing.T, h *Harness, f *Factory, admin *User,
	repID, categoryID string) (code int, phone string) {
	t.Helper()
	phone = f.NS.Phone()
	leadID := newLeadFor(t, h, repID, categoryID, phone)
	return h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token,
		map[string]any{"status": "converted", "note": "XG-32"}).Code, phone
}

// targetFacts قيودُ المكافأة ورصيدُ المحفظة.
type targetFacts struct {
	Rewards int
	Sum     int64
	Wallet  int64
	Ledger  int64 // مجموعُ قيود `reward` في الدفتر
}

func targetFactsOf(t *testing.T, h *Harness, repID string) targetFacts {
	t.Helper()
	var f targetFacts
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT count(*) FROM incentives
		         WHERE user_id = $1::uuid AND for_target),
		       COALESCE((SELECT sum(amount) FROM incentives
		                  WHERE user_id = $1::uuid AND for_target), 0),
		       COALESCE((SELECT balance FROM wallets WHERE user_id = $1::uuid), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                  WHERE user_id = $1::uuid AND kind = 'reward'), 0)`,
		repID).Scan(&f.Rewards, &f.Sum, &f.Wallet, &f.Ledger); err != nil {
		t.Fatalf("قراءةُ وقائع الهدف: %v", err)
	}
	return f
}

// targetFixture مندوبٌ وتصنيفٌ وهدفٌ بالمقدار المطلوب.
func targetFixture(t *testing.T, h *Harness, target int) (f *Factory, repID, categoryID string) {
	t.Helper()
	h.Setting("sales.monthly_target", itoaQA(target))
	h.Setting("sales.target_reward", "5000")
	// **ومصنعٌ واحد** — **وكلُّ `h.Factory()` تُعيد ترقيمَ الهواتف**
	// فيتصادم هاتفُ المرشَّح بهاتف المندوب.
	f = h.Factory()
	rep := f.RepAccount()
	if err := h.Pool.QueryRow(ctxBG(),
		`INSERT INTO categories (name, active) VALUES ($1, true) RETURNING id::text`,
		uniq("تصنيفُ هدفٍ ")).Scan(&categoryID); err != nil {
		t.Fatalf("تصنيف: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = h.Pool.Exec(c, `DELETE FROM incentives WHERE user_id = $1::uuid`, rep.ID)
		_, _ = h.Pool.Exec(c, `DELETE FROM categories WHERE id = $1::uuid`, categoryID)
	})
	return f, rep.ID, categoryID
}

func itoaQA(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// ══════════════════════════════════════════════════════════════════════
// **١ · هدفٌ=1 · أوّلُ تحويلٍ يستحقّ ⇒ مكافأةٌ واحدةٌ فوراً**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا بعينه ما كان يسقط**: العدُّ لا يرى متجرَ معاملتِه.
func TestXG32_FirstConversionGrantsRewardImmediately(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	f, repID, categoryID := targetFixture(t, h, 1)

	code, phone := convertLeadFor(t, h, f, admin, repID, categoryID)
	if code >= 400 {
		t.Fatalf("التحويل: %d", code)
	}
	merchants := countRows(t, h, `SELECT count(*) FROM merchants WHERE phone = $1`, phone)
	got := targetFactsOf(t, h, repID)
	t.Logf("هدفٌ=1 · تحويلٌ أوّل: متاجرُ=%d · مكافآتُ=%d بمجموع %d · "+
		"محفظةٌ=%d · دفترٌ=%d", merchants, got.Rewards, got.Sum, got.Wallet, got.Ledger)

	if merchants != 1 {
		t.Errorf("متاجرُ %d والمتوقَّع واحد", merchants)
	}
	if got.Rewards != 1 {
		t.Errorf("**مكافآتُ %d والمتوقَّع واحدة** — **والعدُّ لا يرى "+
			"متجرَ معاملتِه** (`XG-32`)", got.Rewards)
	}
	if got.Sum != 5000 {
		t.Errorf("مجموعُ المكافأة %d والمتوقَّع 5000", got.Sum)
	}
	if got.Ledger != got.Sum {
		t.Errorf("**الدفترُ %d والمكافأةُ %d** — ولا يتطابقان", got.Ledger, got.Sum)
	}
	if got.Wallet != got.Sum {
		t.Errorf("**المحفظةُ %d والدفترُ %d**", got.Wallet, got.Sum)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · هدفٌ=2 ⇒ الأوّلُ لا شيءَ والثاني مكافأةٌ واحدة**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُصلَح التأخيرُ بجعل كلِّ تحويلٍ يُكافأ** — **العتبةُ عتبة.**
func TestXG32_TargetTwoGrantsOnSecondOnly(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	f, repID, categoryID := targetFixture(t, h, 2)

	if code, _ := convertLeadFor(t, h, f, admin, repID, categoryID); code >= 400 {
		t.Fatalf("التحويلُ الأوّل: %d", code)
	}
	one := targetFactsOf(t, h, repID)
	if code, _ := convertLeadFor(t, h, f, admin, repID, categoryID); code >= 400 {
		t.Fatalf("التحويلُ الثاني: %d", code)
	}
	two := targetFactsOf(t, h, repID)
	t.Logf("هدفٌ=2: بعد الأوّل مكافآتُ=%d · وبعد الثاني %d بمجموع %d",
		one.Rewards, two.Rewards, two.Sum)

	if one.Rewards != 0 {
		t.Errorf("**كوفئ قبل بلوغ العتبة**: %d قيداً", one.Rewards)
	}
	if two.Rewards != 1 {
		t.Errorf("**مكافآتُ %d بعد بلوغ العتبة والمتوقَّع واحدة**", two.Rewards)
	}
	if two.Sum != 5000 {
		t.Errorf("المجموعُ %d والمتوقَّع 5000", two.Sum)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٣+٤ · سقوطٌ قبل التثبيت ثمّ إعادة ⇒ مكافأةٌ واحدةٌ لا صفرٌ ولا اثنتان**
// ══════════════════════════════════════════════════════════════════════
func TestXG32_FailureRollbackThenRetryGrantsOnce(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	f, repID, categoryID := targetFixture(t, h, 1)

	phone := f.NS.Phone()
	leadID := newLeadFor(t, h, repID, categoryID, phone)
	body := map[string]any{"status": "converted", "note": "XG-32"}

	fp := h.Arm("XG32/commit-conversion", "merchant_leads", "UPDATE",
		1, "status", "converted")
	failed := h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token, body)
	fp.MustFire(t)

	mid := targetFactsOf(t, h, repID)
	merchants := countRows(t, h, `SELECT count(*) FROM merchants WHERE phone = $1`, phone)
	var status string
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT status FROM merchant_leads WHERE id = $1::uuid`, leadID).Scan(&status)
	t.Logf("سقوطٌ (%d): متاجرُ=%d · المرشَّحُ=%q · مكافآتُ=%d · محفظةٌ=%d",
		failed.Code, merchants, status, mid.Rewards, mid.Wallet)

	if merchants != 0 || mid.Rewards != 0 || mid.Wallet != 0 {
		t.Errorf("**أثرٌ بقي بعد السقوط**: متاجرُ=%d · مكافآتُ=%d · محفظةٌ=%d",
			merchants, mid.Rewards, mid.Wallet)
	}
	if status != "new" {
		t.Errorf("المرشَّحُ %q — **ويجب أن يبقى قابلاً للإعادة**", status)
	}

	if retry := h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token, body); retry.Code >= 400 {
		t.Fatalf("الإعادةُ سقطت: %d", retry.Code)
	}
	after := targetFactsOf(t, h, repID)
	t.Logf("بعد الإعادة: مكافآتُ=%d بمجموع %d · محفظةٌ=%d · دفترٌ=%d",
		after.Rewards, after.Sum, after.Wallet, after.Ledger)
	if after.Rewards != 1 {
		t.Errorf("**مكافآتُ %d بعد إعادةٍ ناجحة والمتوقَّع واحدة**", after.Rewards)
	}
	if after.Wallet != after.Ledger || after.Ledger != after.Sum {
		t.Errorf("محفظةٌ=%d · دفترٌ=%d · مكافأةٌ=%d — ولا تتطابق",
			after.Wallet, after.Ledger, after.Sum)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · تحويلاتٌ تاليةٌ لا تُكافأ ثانيةً**
// ══════════════════════════════════════════════════════════════════════
//
// **والقيدُ الفريدُ `(user_id, period)` هو الحارسُ الأخير** — يُثبَت
// عملُه ولا يُتّكَل عليه لإخفاء خطأ حساب.
func TestXG32_FurtherConversionsDoNotRepeatReward(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	f, repID, categoryID := targetFixture(t, h, 1)

	for i := 0; i < 3; i++ {
		if code, _ := convertLeadFor(t, h, f, admin, repID, categoryID); code >= 400 {
			t.Fatalf("التحويلُ %d: %d", i+1, code)
		}
	}
	got := targetFactsOf(t, h, repID)
	t.Logf("ثلاثةُ تحويلاتٍ وهدفٌ=1: مكافآتُ=%d بمجموع %d · محفظةٌ=%d",
		got.Rewards, got.Sum, got.Wallet)
	if got.Rewards != 1 {
		t.Errorf("**المكافأةُ تكرّرت**: %d قيداً بمجموع %d", got.Rewards, got.Sum)
	}

	// **والقيدُ في القاعدة يرفض الثانيَ ولو سقط الحساب.**
	var period string
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT period FROM incentives WHERE user_id = $1::uuid AND for_target LIMIT 1`,
		repID).Scan(&period)
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO incentives (user_id, kind, amount, reason, for_target, period)
		VALUES ($1::uuid, 'reward', 5000, 'تكرارٌ مباشر', true, $2)`,
		repID, period); err == nil {
		t.Error("**القاعدةُ قبلت مكافأةً ثانيةً للمرحلة نفسِها** — " +
			"`incentives_one_target_per_month` لا يحرس")
	} else {
		t.Logf("القاعدةُ ردّت المكافأةَ الثانية — الحارسُ في المخطَّط")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · حدُّ الشهر — عدُّ الشهر الماضي لا يُرضي هدفَ هذا الشهر**
// ══════════════════════════════════════════════════════════════════════
func TestXG32_PreviousMonthDoesNotSatisfyTarget(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	f, repID, categoryID := targetFixture(t, h, 2)

	// **متجرٌ منسوبٌ للمندوب لكنّه من شهرٍ مضى.**
	if code, phone := convertLeadFor(t, h, f, admin, repID, categoryID); code >= 400 {
		t.Fatalf("التحويلُ التمهيديّ: %d", code)
	} else if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE merchants SET created_at = now() - interval '45 days'
		 WHERE phone = $1`, phone); err != nil {
		t.Fatalf("إشاخةُ المتجر: %v", err)
	}
	// **وتُمحى مكافأةُ ذلك الشهر إن وقعت** — فالسؤالُ عن هذا الشهر.
	_, _ = h.Pool.Exec(ctxBG(),
		`DELETE FROM incentives WHERE user_id = $1::uuid`, repID)

	if code, _ := convertLeadFor(t, h, f, admin, repID, categoryID); code >= 400 {
		t.Fatalf("تحويلُ هذا الشهر: %d", code)
	}
	got := targetFactsOf(t, h, repID)
	t.Logf("متجرٌ من شهرٍ مضى + متجرٌ هذا الشهر · هدفٌ=2 ⇒ مكافآتُ=%d",
		got.Rewards)
	if got.Rewards != 0 {
		t.Errorf("**كوفئ بعدٍّ من شهرٍ مضى**: %d قيداً — "+
			"**والهدفُ شهريٌّ لا تراكميّ**", got.Rewards)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٧ · تحويلان متزامنان يعبران العتبةَ معاً ⇒ مكافأةٌ واحدة**
// ══════════════════════════════════════════════════════════════════════
//
// **تداخلٌ حقيقيٌّ مقيس.** **والقيدُ الفريدُ هو الفاصل** — والحسابُ
// وحدَه لا يكفي حين يقرأ الاثنان معاً.
func TestXG32_ConcurrentTargetCrossingGrantsOnce(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	f, repID, categoryID := targetFixture(t, h, 1)

	leadA := newLeadFor(t, h, repID, categoryID, f.NS.Phone())
	leadB := newLeadFor(t, h, repID, categoryID, f.NS.Phone())
	body := map[string]any{"status": "converted", "note": "XG-32"}

	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "تحويلٌ-أ", Do: func(ctx context.Context) any {
			return h.POST("/api/v1/admin/leads/"+leadA+"/status", admin.Token, body)
		}},
		Actor{Name: "تحويلٌ-ب", Do: func(ctx context.Context) any {
			return h.POST("/api/v1/admin/leads/"+leadB+"/status", admin.Token, body)
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}
	got := targetFactsOf(t, h, repID)
	t.Logf("تحويلان متزامنان يعبران العتبةَ: مكافآتُ=%d بمجموع %d · "+
		"محفظةٌ=%d · دفترٌ=%d — %s",
		got.Rewards, got.Sum, got.Wallet, got.Ledger, r)

	if got.Rewards > 1 {
		t.Errorf("**مكافأتان من عبورٍ واحد**: %d قيداً بمجموع %d",
			got.Rewards, got.Sum)
	}
	if got.Rewards != 1 {
		t.Errorf("**لم تُقيَّد مكافأةٌ** والعتبةُ عُبرت: %d", got.Rewards)
	}
	if got.Wallet != got.Ledger || got.Ledger != got.Sum {
		t.Errorf("محفظةٌ=%d · دفترٌ=%d · مكافأةٌ=%d", got.Wallet, got.Ledger, got.Sum)
	}
}
