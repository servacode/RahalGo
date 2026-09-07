package gate

import (
	"fmt"
	"sort"
	"strings"
)

// CategoryVerdict حكمُ مجالٍ كامل — **للعرض لا للقرار.**
type CategoryVerdict struct {
	Category Category `json:"category"`
	State    State    `json:"state"`
	Total    int      `json:"total"`
	Passing  int      `json:"passing"`
	Blocking int      `json:"blocking_unsatisfied"`
}

// Categories حكمُ كلِّ مجال.
//
// **وأسوأُ حالٍ في المجال هو حالُ المجال** — **ولا يُجمَّل بمتوسّط.**
func (d *Decision) Categories() []CategoryVerdict {
	agg := map[Category]*CategoryVerdict{}
	for _, r := range d.Rules {
		if !r.Counted() {
			continue
		}
		v, ok := agg[r.Category]
		if !ok {
			v = &CategoryVerdict{Category: r.Category, State: Pass}
			agg[r.Category] = v
		}
		v.Total++
		if r.Status.Satisfied() {
			v.Passing++
			continue
		}
		if r.Blocking {
			v.Blocking++
		}
		// **الأسوأُ يغلب** — و`FAIL` أسوأُ من `NOT_RUN`.
		if worse(r.Status, v.State) {
			v.State = r.Status
		}
	}
	var out []CategoryVerdict
	for _, v := range agg {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Category < out[j].Category })
	return out
}

func worse(a, b State) bool {
	rank := map[State]int{
		Pass: 0, Waived: 0, NotImplemented: 2, NotRun: 3,
		ReqStaging: 3, ReqDevice: 3, Blocked: 4, ExpectedFail: 5, Fail: 6,
	}
	return rank[a] > rank[b]
}

// StateOf حالُ مجالٍ بعينه — **للتقرير النهائيّ.**
func (d *Decision) StateOf(c Category) State {
	for _, v := range d.Categories() {
		if v.Category == c {
			return v.State
		}
	}
	return NotRun
}

// Human التقريرُ البشريُّ (البند ٤٣).
//
// **ولا قائمةَ خمسِمئةِ اختبار** — **الأسبابُ الجذريّةُ وحدَها.**
func (d *Decision) Human() string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }

	w("══════════════════════════════════════════════════════════════")
	w("  بوّابةُ إطلاق رحّال غو — بالدليل")
	w("══════════════════════════════════════════════════════════════")
	w("")
	w("RELEASE CANDIDATE ID = %s", d.Candidate.ID)
	w("SOURCE COMMIT        = %s", d.Candidate.SourceCommit)
	w("MIGRATION VERSION    = %s", d.Candidate.Migration)
	w("WORKING TREE DIRTY   = %v", d.Candidate.Dirty)
	w("RUN ID               = %s", d.RunID)
	base := d.TruthBase
	if base == "" {
		// **وحقلُ البصمة في `TEST_TRUTH` فارغٌ عمداً**: لو حمل التزاماً
		// **لَتبدّل الملفُّ في كلّ التزامٍ فأسقط حارسَ الانحراف**.
		// **فالطزاجةُ تُثبَت ببصمة المرشَّح لا به.**
		base = "— (فارغٌ عمداً · الطزاجةُ ببصمة المرشَّح)"
	}
	w("TRUTH BASELINE       = %s", base)
	w("")
	w("──────────────────────────────────────────────────────────────")
	if d.Ready {
		w("  READY FOR PRODUCTION = YES")
	} else {
		w("  READY FOR PRODUCTION = NO")
	}
	w("──────────────────────────────────────────────────────────────")
	w("")
	// **ولا اسمَ يحمل معنيين** (`XG-37`): **الصنفُ غيرُ الحال.**
	w("TOTAL RULES                  = %d", d.Counts.TotalRules)
	w("BLOCKING-CAPABLE RULES       = %d  (صنفاً — ناجحةً كانت أو ساقطة)",
		d.Counts.BlockingCapable)
	w("CURRENT BLOCKERS             = %d  ← **وبه يُقرَّر**",
		d.Counts.CurrentBlockers)
	w("SATISFIED BLOCKING-CAPABLE   = %d", d.Counts.SatisfiedBlockingCapable)
	w("SUPERSEDED BLOCKING-CAPABLE  = %d  (جذرُها في قاعدةٍ أخرى)",
		d.Counts.SupersededBlockingCapable)
	w("WARNINGS                     = %d", d.Counts.Warnings)
	w("SUPERSEDED (لا تُعَدّ)         = %d", d.Counts.Superseded)
	w("")
	w("BLOCKERS                     = %d  (= CURRENT BLOCKERS)", len(d.Blockers))
	w("CRITICAL UNRESOLVED          = %d", len(d.CriticalU))
	w("REQUIRED VALIDATIONS NOT RUN = %d", len(d.NotRunReq))
	for _, x := range d.Debts {
		w("%-28s = %d", x.Kind, x.Count)
	}
	w("")
	w("ADVISORY SCORE = %d/100", d.Score)
	w("  %s", d.ScoreNote)
	w("")

	if len(d.GateErrors) > 0 {
		w("⚠️  أخطاءُ بوّابة:")
		for _, x := range d.GateErrors {
			w("   %s", x)
		}
		w("")
	}

	w("── لماذا لا ─────────────────────────────────────────────────")
	w("")
	if len(d.Blockers) == 0 {
		w("  لا مانع.")
	}
	for i, x := range d.Blockers {
		if i >= 20 {
			w("  … و%d مانعاً آخرَ في RELEASE_GATE.json", len(d.Blockers)-20)
			break
		}
		w("%2d. %s", i+1, x)
	}
	w("")

	w("── ما ثبت وما لم يثبت ───────────────────────────────────────")
	w("")
	for _, v := range d.Categories() {
		mark := "✅"
		if v.State != Pass {
			mark = "❌"
		}
		w("%s %-24s %-16s %d/%d  · مانعٌ غيرُ مُرضىً %d",
			mark, v.Category, v.State, v.Passing, v.Total, v.Blocking)
	}
	w("")

	w("── الطريقُ إلى نعم ──────────────────────────────────────────")
	w("")
	w("REMAINING REQUIREMENTS TO READY = %d", len(d.Remaining))
	for i, x := range d.Remaining {
		if i >= 25 {
			w("  … و%d بنداً آخرَ في RELEASE_GATE.json", len(d.Remaining)-25)
			break
		}
		w("%2d. %s", i+1, x)
	}
	w("")
	w("TOP WARNINGS = %d (غيرُ مانعة)", len(d.Warnings))
	for i, x := range d.Warnings {
		if i >= 10 {
			w("  … و%d تحذيراً آخر", len(d.Warnings)-10)
			break
		}
		w("  · %s", x)
	}
	return b.String()
}
