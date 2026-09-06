package qa

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **`XG-29` — «المكافأةُ تُدفع قبل تثبيت التحويل»**
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **ستُّ كتاباتٍ بلا معاملة**: يُنشأ المتجرُ · **تُدفع المكافأة** ·
// **ثمّ** يُثبَّت التحويل. **فسقوطُ التثبيت يترك مالاً مدفوعاً عن
// تحويلٍ لم يقع.**
//
// # وما صار (دورةُ إصلاحٍ ٤)
//
// **الكلُّ في معاملةٍ واحدة**: المكافأةُ (`leads_handlers.go:895`) ثمّ
// تثبيتُ التحويل (`:901`) ثمّ `Commit` (`:905`). **ولا شيءَ منها دائمٌ
// قبل التثبيت.**
//
// **ولا يُغلَق مانعٌ لأنّ الدالّةَ صار في اسمها `Tx`** — **الاسمُ لا
// يُثبت.** فهذه الحرّاسُ تسأل الدفترَ لا الشيفرة.

// rewardFacts ما يقوله الدفترُ عن مندوبٍ ومرشَّحِه.
type rewardFacts struct {
	Merchants  int
	LeadStatus string
	Rewards    int   // قيودُ مكافأةِ هدفٍ في `incentives`
	RewardSum  int64 // مجموعُها
	Wallet     int64 // رصيدُ محفظة المندوب
}

func rewardFactsOf(t *testing.T, h *Harness, leadID, phone, repID string) rewardFacts {
	t.Helper()
	var f rewardFacts
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT count(*) FROM merchants WHERE phone = $1),
		       COALESCE((SELECT status FROM merchant_leads WHERE id = $2::uuid), ''),
		       (SELECT count(*) FROM incentives
		         WHERE user_id = $3::uuid AND for_target),
		       COALESCE((SELECT sum(amount) FROM incentives
		                  WHERE user_id = $3::uuid AND for_target), 0),
		       COALESCE((SELECT balance FROM wallets WHERE user_id = $3::uuid), 0)`,
		phone, leadID, repID).Scan(&f.Merchants, &f.LeadStatus,
		&f.Rewards, &f.RewardSum, &f.Wallet); err != nil {
		t.Fatalf("قراءةُ وقائع المكافأة: %v", err)
	}
	return f
}

// rewardFixture مندوبٌ ومرشَّحٌ وهدفٌ بلغه بمتجرٍ واحد.
//
// **والهدفُ واحد** — **فأوّلُ تحويلٍ يستحقّ**، ولا حاجةَ لبناء شهرٍ
// كاملٍ من المتاجر.
func rewardFixture(t *testing.T, h *Harness) (f *Factory, repID, categoryID string) {
	t.Helper()
	h.Setting("sales.monthly_target", "1")
	h.Setting("sales.target_reward", "5000")
	// **ومصنعٌ واحدٌ لا مصنعٌ في كلّ سطر**: **كلُّ `h.Factory()` تُعيد
	// ترقيمَ الهواتف من أوّله** — **فيتصادم هاتفُ المرشَّح بهاتف
	// المندوب**، ويُردّ التحويلُ `role_conflict` (سائقٌ لا يصير متجراً).
	f = h.Factory()
	rep := f.RepAccount()

	if err := h.Pool.QueryRow(ctxBG(),
		`INSERT INTO categories (name, active) VALUES ($1, true) RETURNING id::text`,
		uniq("تصنيفُ مكافأةٍ ")).Scan(&categoryID); err != nil {
		t.Fatalf("تصنيف: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = h.Pool.Exec(c, `DELETE FROM incentives WHERE user_id = $1::uuid`, rep.ID)
		_, _ = h.Pool.Exec(c, `DELETE FROM categories WHERE id = $1::uuid`, categoryID)
	})
	return f, rep.ID, categoryID
}

// ══════════════════════════════════════════════════════════════════════
// **١ · سقوطٌ قبل التثبيت ⇒ لا متجرَ ولا مكافأةَ والمرشَّحُ قابلٌ للإعادة**
// ══════════════════════════════════════════════════════════════════════
func TestXG29_FailureBeforeCommitLeavesNothing(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	f, repID, categoryID := rewardFixture(t, h)
	phone := f.NS.Phone()
	leadID := newLeadFor(t, h, repID, categoryID, phone)

	// **يُسقَط تثبيتُ التحويل** — وهو آخرُ كتابةٍ قبل `Commit`.
	fp := h.Arm("XG29/commit-conversion", "merchant_leads", "UPDATE",
		1, "status", "converted")
	got := h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token,
		map[string]any{"status": "converted", "note": "XG-29"})
	fp.MustFire(t)

	got2 := rewardFactsOf(t, h, leadID, phone, repID)
	t.Logf("سقوطٌ قبل التثبيت (%d): متاجرُ=%d · المرشَّحُ=%q · مكافآتُ=%d "+
		"بمجموع %d · محفظةُ المندوب=%d",
		got.Code, got2.Merchants, got2.LeadStatus, got2.Rewards,
		got2.RewardSum, got2.Wallet)

	if got2.Merchants != 0 {
		t.Errorf("**متجرٌ بقي والتثبيتُ سقط** (%d)", got2.Merchants)
	}
	if got2.Rewards != 0 || got2.RewardSum != 0 {
		t.Errorf("**مكافأةٌ دُفعت عن تحويلٍ لم يقع** — %d قيداً بمجموع %d — "+
			"**وهو `XG-29` بعينه.**", got2.Rewards, got2.RewardSum)
	}
	if got2.Wallet != 0 {
		t.Errorf("**محفظةُ المندوب تحرّكت** (%d) والتحويلُ لم يقع", got2.Wallet)
	}
	if got2.LeadStatus != "new" {
		t.Errorf("المرشَّحُ %q — **ويجب أن يبقى قابلاً للإعادة**", got2.LeadStatus)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · نجاحٌ ⇒ متجرٌ واحدٌ ومرشَّحٌ محوَّلٌ ومكافأةٌ مرّةً واحدةً إن استُحقّت**
// ══════════════════════════════════════════════════════════════════════
func TestXG29_SuccessGrantsRewardExactlyOnce(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	f, repID, categoryID := rewardFixture(t, h)

	// **متجران**: الأوّلُ يبلغ الهدفَ والثاني يُثبت ألّا تكرار.
	var last rewardFacts
	for i := 0; i < 2; i++ {
		phone := f.NS.Phone()
		leadID := newLeadFor(t, h, repID, categoryID, phone)
		if got := h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token,
			map[string]any{"status": "converted", "note": "XG-29"}); got.Code >= 400 {
			t.Fatalf("التحويلُ %d: %d %s", i+1, got.Code, got.Err())
		}
		last = rewardFactsOf(t, h, leadID, phone, repID)
		t.Logf("تحويلٌ %d: متاجرُ=%d · المرشَّحُ=%q · مكافآتُ=%d بمجموع %d · محفظةٌ=%d",
			i+1, last.Merchants, last.LeadStatus, last.Rewards, last.RewardSum, last.Wallet)
		if last.LeadStatus != "converted" {
			t.Errorf("التحويلُ %d: المرشَّحُ %q", i+1, last.LeadStatus)
		}
	}

	// **والهدفُ واحدٌ فالمكافأةُ مرّةٌ واحدةٌ لا اثنتان.**
	if last.Rewards != 1 {
		t.Errorf("**قيودُ المكافأة %d والمتوقَّع واحد** (المرحلةُ واحدةٌ "+
			"والهدفُ بُلغ) — بمجموع %d", last.Rewards, last.RewardSum)
	}
	if last.RewardSum != 5000 {
		t.Errorf("مجموعُ المكافأة %d والمتوقَّع 5000", last.RewardSum)
	}
	if last.Wallet != last.RewardSum {
		t.Errorf("**المحفظةُ %d والمكافأةُ %d** — ولا يتطابقان",
			last.Wallet, last.RewardSum)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · سقوطٌ ثمّ إعادة ⇒ مكافأةٌ مرّةً واحدةً لا مرّتين**
// ══════════════════════════════════════════════════════════════════════
func TestXG29_FailureThenRetryGrantsRewardOnce(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	f, repID, categoryID := rewardFixture(t, h)

	// **متجرٌ سابقٌ يبلغ به الهدفَ** — فالتحويلُ التالي مستحقٌّ فوراً.
	first := f.NS.Phone()
	firstLead := newLeadFor(t, h, repID, categoryID, first)
	if got := h.POST("/api/v1/admin/leads/"+firstLead+"/status", admin.Token,
		map[string]any{"status": "converted", "note": "تمهيد"}); got.Code >= 400 {
		t.Fatalf("التمهيد: %d", got.Code)
	}

	phone := f.NS.Phone()
	leadID := newLeadFor(t, h, repID, categoryID, phone)
	body := map[string]any{"status": "converted", "note": "XG-29"}

	fp := h.Arm("XG29/retry-commit", "merchant_leads", "UPDATE",
		1, "status", "converted")
	failed := h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token, body)
	fp.MustFire(t)
	mid := rewardFactsOf(t, h, leadID, phone, repID)

	retry := h.POST("/api/v1/admin/leads/"+leadID+"/status", admin.Token, body)
	after := rewardFactsOf(t, h, leadID, phone, repID)

	t.Logf("بعد السقوط (%d): متاجرُ=%d · مكافآتُ=%d بمجموع %d",
		failed.Code, mid.Merchants, mid.Rewards, mid.RewardSum)
	t.Logf("بعد الإعادة (%d): متاجرُ=%d · المرشَّحُ=%q · مكافآتُ=%d بمجموع %d · محفظةٌ=%d",
		retry.Code, after.Merchants, after.LeadStatus, after.Rewards,
		after.RewardSum, after.Wallet)

	if retry.Code >= 400 {
		t.Errorf("**الإعادةُ سقطت** (%d)", retry.Code)
	}
	if after.Merchants != 1 {
		t.Errorf("**متاجرُ %d بعد سقوطٍ وإعادة** والمتوقَّع واحد", after.Merchants)
	}
	if after.Rewards > 1 {
		t.Errorf("**المكافأةُ تكرّرت**: %d قيداً بمجموع %d — "+
			"**والإعادةُ لا تصير باباً لمكافأةٍ ثانية.**",
			after.Rewards, after.RewardSum)
	}
	if after.Wallet != after.RewardSum {
		t.Errorf("المحفظةُ %d والمكافأةُ %d", after.Wallet, after.RewardSum)
	}
}
