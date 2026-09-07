package gate

import (
	"path/filepath"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **عدّادان في تقريرٍ واحد** — `XG-37`
// ══════════════════════════════════════════════════════════════════════
//
// # ما وقع
//
// **الطرفيّةُ قالت `BLOCKERS = 23` والملفُّ فيه ٢٨ صفّاً `blocking:true`.**
//
// **والاسمُ كان يحمل معنيين**: «مانعةٌ **إن سقطت**» و«مانعةٌ **الآن**».
// **فمن بنى لوحةً أو حارساً على `blocking:true` أعلن ٢٨ وهي ٢٣** —
// **ولا يظهر الخطأُ إلّا بمقارنةٍ يدويّة.**
//
// # وما صار
//
//	blocking_capable    صنفٌ — ناجحةً كانت أو ساقطة
//	currently_blocking  حالٌ — وهو ما يُقرَّر به
//
// **والعددُ منشورٌ في `counts` لا يُستنتَج** — **فلا يعدّ قارئٌ بنفسه.**

// TestXG37_CountsAgreeAcrossAllThreeReaders **ثلاثةُ طرقٍ ورقمٌ واحد.**
//
// **والفحصُ يقرأ الحكمَ الحقيقيَّ لا مصنوعاً** — **فحارسٌ على بياناتٍ
// مخترَعةٍ يمرّ وهو أعمى.**
func TestXG37_CountsAgreeAcrossAllThreeReaders(t *testing.T) {
	d := decisionForTest(t)

	// ① ما تطبعه الطرفيّة.
	cli := len(d.Blockers)

	// ② ما يُنشَر في الملخَّص.
	summary := d.Counts.CurrentBlockers

	// ③ ما يُحسَب بترشيحٍ آليٍّ على الصفوف.
	var recomputed int
	for _, r := range d.Rules {
		if r.CurrentlyBlocking {
			recomputed++
		}
	}

	t.Logf("طرفيّة=%d · ملخَّص=%d · ترشيحٌ آليّ=%d", cli, summary, recomputed)

	if cli != summary || summary != recomputed {
		t.Errorf("**ثلاثةُ أعدادٍ لمانعٍ واحد**: طرفيّة=%d · ملخَّص=%d · "+
			"ترشيح=%d — **والبوّابةُ مصدرُ قرارِ الإطلاق.** (`XG-37`)",
			cli, summary, recomputed)
	}
}

// TestXG37_BlockingCapableIsNotACurrentBlocker **والصنفُ ليس حالاً.**
//
// **وقاعدةٌ ناجحةٌ تبقى `blocking_capable=true`** — **ولا تُعَدّ.**
func TestXG37_BlockingCapableIsNotACurrentBlocker(t *testing.T) {
	d := decisionForTest(t)
	c := d.Counts

	t.Logf("قواعدُ=%d · مانعةٌ صنفاً=%d · مانعةٌ حالاً=%d · "+
		"مستوفاةٌ مانعةُ الصنف=%d · تحذيراتٌ=%d · مُستتبَعةٌ=%d",
		c.TotalRules, c.BlockingCapable, c.CurrentBlockers,
		c.SatisfiedBlockingCapable, c.Warnings, c.Superseded)
	t.Logf("ومنها مانعةُ الصنفِ مُستتبَعةٌ=%d", c.SupersededBlockingCapable)

	if c.BlockingCapable != c.CurrentBlockers+c.SatisfiedBlockingCapable+
		c.SupersededBlockingCapable {
		t.Errorf("**صنفُ المانعات لا يساوي أحوالَه الثلاثة**: "+
			"%d ≠ %d + %d + %d", c.BlockingCapable, c.CurrentBlockers,
			c.SatisfiedBlockingCapable, c.SupersededBlockingCapable)
	}
	if c.TotalRules < c.BlockingCapable+c.Warnings {
		t.Errorf("**أعدادٌ تفوق مجموعَها**: %d < %d + %d",
			c.TotalRules, c.BlockingCapable, c.Warnings)
	}

	// **ولا يُصلَح الفرقُ بإخفاء ناجحة** — **فوجودُها دليلٌ على العمل.**
	if c.SatisfiedBlockingCapable == 0 {
		t.Log("**تنبيه**: لا قاعدةَ مانعةَ الصنف مستوفاةٌ — " +
			"**فإمّا لم يُنجَز شيءٌ وإمّا حُذفت الناجحات.**")
	}

	// **والتحذيرُ لا يكون مانعاً.**
	for _, r := range d.Rules {
		if r.CurrentlyBlocking && !r.Blocking {
			t.Errorf("**`%s` مانعةٌ حالاً وليست مانعةَ الصنف**", r.ID)
		}
	}
}

// TestXG37_SupersededNeverCounts **والمُستتبَعةُ لا مانعَ ثانٍ.**
func TestXG37_SupersededNeverCounts(t *testing.T) {
	d := decisionForTest(t)
	for _, r := range d.Rules {
		if r.SupersededBy != "" && r.CurrentlyBlocking {
			t.Errorf("**`%s` مُستتبَعةٌ بـ`%s` وتُعَدّ مانعاً** — "+
				"**سببٌ واحدٌ لا مانعان مصطنعان.**", r.ID, r.SupersededBy)
		}
	}
}

// decisionForTest **الحكمُ الحقيقيُّ على الشجرة الحاليّة.**
func decisionForTest(t *testing.T) *Decision {
	t.Helper()
	e := realEvidence(t)
	c, err := BuildCandidate(filepath.Dir(mustBackend(t)))
	if err != nil {
		t.Fatalf("هويّةُ المرشَّح: %v", err)
	}
	return Decide(c, e, Waivers)
}
