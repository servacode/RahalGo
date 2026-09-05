package gate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Waivers **سجلُّ التنازلات — فارغٌ عمداً** (البند ٣٧).
//
// **و`P-10` لا يُنشئ تنازلاً من نفسه.** الصيغةُ مبنيّةٌ لمن يأتي بقرار
// مالكٍ صريح، **ولا سطرَ فيها اليوم.**
var Waivers = []Waiver{}

// Decide الحكم — **البيانات تُنتجه ولا يُكتب مسبقاً** (البند ٣٩).
func Decide(cand Candidate, e *Evidence, waivers []Waiver) *Decision {
	rules := BuildRules(e)

	// ── التنازلات ────────────────────────────────────────────────
	//
	// **ولا تنازلَ على ما تمنعه السياسة** — المالُ والخصوصيّةُ والأمنُ
	// في شدّةٍ مانعةٍ أو حرجة.
	byID := map[string]Waiver{}
	var bad []string
	for _, w := range waivers {
		byID[w.RuleID] = w
	}
	for i := range rules {
		w, ok := byID[rules[i].ID]
		if !ok {
			continue
		}
		if rules[i].WaiverPolicy == WaiverForbidden {
			bad = append(bad, fmt.Sprintf("%s — **تنازلٌ ممنوعٌ بالسياسة ورُفض** (%s)",
				rules[i].ID, w.Owner))
			continue
		}
		rules[i].Status = Waived
		rules[i].Reason += fmt.Sprintf(" · **تنازلٌ معتمد**: %s — %s (ينتهي %s)",
			w.Owner, w.Reason, w.Expiry)
	}

	// ── طزاجةُ الدليل (البند ٣٤) ─────────────────────────────────
	//
	// **دليلٌ من بناءٍ قديمٍ لا يثبت بناءً جديداً.**
	for i := range rules {
		if rules[i].EvidenceCommit != "" && rules[i].EvidenceCommit != cand.SourceCommit {
			rules[i].Stale = true
			if rules[i].Status.Satisfied() {
				rules[i].Status = NotRun
				rules[i].Reason = fmt.Sprintf("**دليلٌ شائخ** — من `%s` والمرشَّحُ `%s` · %s",
					short(rules[i].EvidenceCommit), short(cand.SourceCommit), rules[i].Reason)
			}
		}
	}

	// ── هويّةُ المرشَّح تُحسَم هنا (البند ٣٣) ──────────────────────
	for i := range rules {
		if rules[i].ID != "GATE-FRESH-01" {
			continue
		}
		if cand.Dirty {
			rules[i].Status = Fail
			rules[i].Reason = "**شجرةُ العمل متّسخة** — " +
				"**فالمحكومُ عليه ليس ما سيُنشَر**، والدليلُ كلُّه على رملٍ متحرّك"
		} else if cand.BackendHash == "" {
			rules[i].Status = Fail
			rules[i].Reason = "**لا بصمةَ بناءٍ للخادم** — ولا مرشَّحَ بلا بصمة"
		} else {
			rules[i].Status = Pass
			rules[i].Reason = fmt.Sprintf("**التزامٌ نظيفٌ `%s` · هجرةٌ `%s` · %d بصمةَ تطبيق**",
				short(cand.SourceCommit), cand.Migration, len(cand.APKHashes))
		}
	}

	d := &Decision{
		Candidate: cand, Rules: rules, Waivers: waivers,
		TruthBase:  e.Truth.Baseline,
		Generated:  time.Now().UTC().Format(time.RFC3339),
		GateErrors: bad,
	}
	d.RunID = runID(cand, rules)

	// ── الحكم ────────────────────────────────────────────────────
	blocking := d.unsatisfiedBlocking()
	d.Ready = len(blocking) == 0 && len(bad) == 0

	for _, r := range blocking {
		d.Blockers = append(d.Blockers, fmt.Sprintf("[%s · %s] %s — %s",
			r.Severity, r.Status, r.Requirement, r.Reason))
	}
	for _, r := range rules {
		if !r.Counted() || r.Status.Satisfied() {
			continue
		}
		if r.Severity == Critical {
			d.CriticalU = append(d.CriticalU, r.ID+" — "+r.Requirement)
		}
		switch r.Status {
		case NotRun, ReqDevice, ReqStaging, Blocked:
			if r.Blocking {
				d.NotRunReq = append(d.NotRunReq, r.ID+" — "+r.Requirement)
			}
		}
		if !r.Blocking {
			d.Warnings = append(d.Warnings, fmt.Sprintf("[%s] %s — %s",
				r.Severity, r.ID, r.Requirement))
		}
	}

	d.Debts = debts(rules)
	d.Remaining = remaining(rules)
	d.Score, d.ScoreNote = score(rules)
	return d
}

// debts تصنيفُ الدَّين (البند ٤٠).
func debts(rules []Rule) []Debt {
	kinds := []struct {
		name string
		why  string
		hit  func(Rule) bool
	}{
		{"DEVICE DEBT", "**قبولٌ على جهازٍ حقيقيٍّ لم يبدأ**", func(r Rule) bool {
			return r.Category == CatAndroid || r.Category == CatNavigation || r.Category == CatBackground
		}},
		{"STAGING DEBT", "**بيئةُ تجهيزٍ مخصَّصةٌ لم تُنشأ — `P-0`**", func(r Rule) bool {
			return r.Category == CatStaging || r.Status == ReqStaging
		}},
		{"TRACEABILITY DEBT", "**سجلّاتٌ بلا دليلٍ يخصُّها**", func(r Rule) bool {
			return r.Category == CatTrace
		}},
		{"OPS DEBT", "**نسخٌ واستعادةٌ وصحّةٌ وجهوزيّةٌ ومراقبة**", func(r Rule) bool {
			return r.Category == CatBackup || r.Category == CatHealth ||
				r.Category == CatOps || r.Category == CatLoad || r.Category == CatPerf
		}},
	}
	var out []Debt
	for _, k := range kinds {
		var items []string
		for _, r := range rules {
			if k.hit(r) && !r.Status.Satisfied() {
				items = append(items, r.ID)
			}
		}
		if len(items) > 0 {
			sort.Strings(items)
			out = append(out, Debt{Kind: k.name, Count: len(items), Items: items, Why: k.why})
		}
	}
	return out
}

// remaining الطريقُ إلى نعم (البند ٤١) — **مرتَّبٌ بالخطورة.**
func remaining(rules []Rule) []string {
	var blk []Rule
	for _, r := range rules {
		if r.Counted() && !r.Status.Satisfied() && r.Blocking {
			blk = append(blk, r)
		}
	}
	sort.Slice(blk, func(i, j int) bool {
		if blk[i].Severity.rank() != blk[j].Severity.rank() {
			return blk[i].Severity.rank() > blk[j].Severity.rank()
		}
		return blk[i].ID < blk[j].ID
	})
	var out []string
	for _, r := range blk {
		out = append(out, fmt.Sprintf("%s [%s] %s ⇒ %s",
			r.ID, r.Severity, r.Requirement, r.RequiredEvidence))
	}
	return out
}

// score **عرضٌ مساعدٌ لا قرار** (البند ٤٦).
func score(rules []Rule) (int, string) {
	n, ok := 0, 0
	for _, r := range rules {
		if !r.Counted() {
			continue
		}
		n++
		if r.Status.Satisfied() {
			ok++
		}
	}
	if n == 0 {
		return 0, "لا قاعدةَ تُحتسب"
	}
	return ok * 100 / n,
		"**عرضٌ مساعدٌ لا قرار** — `BLOCKER > SCORE`، **ومانعٌ واحدٌ يُسقط الحكمَ مهما بلغت الدرجة**"
}

// runID هويّةُ التشغيل — **تتبدّل إن تبدّل المرشَّحُ أو حالُ قاعدة.**
func runID(c Candidate, rules []Rule) string {
	h := sha256.New()
	fmt.Fprint(h, c.ID, c.SourceCommit)
	for _, r := range rules {
		fmt.Fprint(h, r.ID, r.Status)
	}
	return "gate_" + hex.EncodeToString(h.Sum(nil))[:16]
}

func short(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

// ══════════════════════════════════════════════════════════════════════
// **هويّةُ المرشَّح (البند ٣٣)**
// ══════════════════════════════════════════════════════════════════════

// BuildCandidate يبني هويّةَ المرشَّح من المستودع.
//
// **وشجرةٌ متّسخةٌ ليست مرشَّحاً** — تُسجَّل ويُحكَم عليها.
func BuildCandidate(root string) (Candidate, error) {
	c := Candidate{Generated: time.Now().UTC().Format(time.RFC3339)}
	git := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	c.SourceCommit = git("rev-parse", "HEAD")
	if c.SourceCommit == "" {
		return c, fmt.Errorf("لا مستودعَ ولا التزام — **ولا بوّابةَ على غير مرشَّحٍ ثابت**")
	}
	c.Dirty = git("status", "--porcelain") != ""

	// **بصمةُ الشيفرة** — تُحسب من محتوى الشجرة كما هي، **فتختلف عن
	// الالتزام حين تتّسخ الشجرة.**
	c.BackendHash = treeHash(filepath.Join(root, "backend"), ".go")
	c.WebHash = treeHash(filepath.Join(root, "web"), ".tsx")
	for _, app := range []string{"app-customer", "app-driver", "app-merchant", "app-rep"} {
		h := treeHash(filepath.Join(root, "mobile", app), ".kt")
		if h != "" {
			c.APKHashes = append(c.APKHashes, app+":"+h)
		}
	}
	c.ConfigSnap = fileHash(filepath.Join(root, "backend/internal/settings/catalog.go"))
	c.Migration = lastMigration(filepath.Join(root, "backend/internal/migrate/migrations"))

	c.ID = fmt.Sprintf("rc-%s-%s", short(c.SourceCommit), short(c.BackendHash))
	if c.Dirty {
		c.ID += "-dirty"
	}
	return c, nil
}

func treeHash(dir, ext string) string {
	h := sha256.New()
	n := 0
	_ = filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || !strings.HasSuffix(p, ext) {
			return nil
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		fmt.Fprint(h, filepath.ToSlash(rel))
		h.Write(b)
		n++
		return nil
	})
	if n == 0 {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

func fileHash(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])[:16]
}

func lastMigration(dir string) string {
	files, _ := filepath.Glob(filepath.Join(dir, "*.sql"))
	if len(files) == 0 {
		return ""
	}
	sort.Strings(files)
	return filepath.Base(files[len(files)-1])
}
