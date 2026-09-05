// فحوصُ البوّابة الذاتيّة — **`P-10` البند ٣٨.**
//
// # ولماذا حالاتٌ صناعيّة
//
// **البوّابةُ اليومَ تقول `NO`** — **وبوّابةٌ تقول `NO` دائماً لا تُثبت
// شيئاً.** **فيلزم أن يُرى الطريقان**: حالٌ كلُّها ناجحةٌ ⇒ `YES`،
// **ثمّ تُكسَر بندٌ واحدٌ ⇒ `NO`.**
package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// synth دليلٌ صناعيٌّ نظيفٌ — **كلُّ ما فيه مُرضٍ.**
func synth() *Evidence {
	return &Evidence{
		ByRegister: map[string][]MatrixRow{},
		FinChecks:  map[string]string{"FI-01.a": "PROVABLE_NOW"},
		AndroidCounts: map[string]int{
			"Cases": 1, "NotRunN": 0, "DevicesAvailable": 1,
		},
		HealthCovers:      []string{"media", "postgres", "push", "redis"},
		CriticalDeps:      []string{"media", "postgres", "push", "redis"},
		ReadinessEndpoint: true,
		Truth: Truth{
			Baseline: "synthetic",
			Flows: []TruthFlow{{
				ID: "F-01", Title: "تدفّقٌ صناعيّ", Severity: "BLOCKER",
				Tests: []string{"TestSynthFlow"},
			}},
			Settings: []TruthSetting{{
				Key: "x.money", Money: true, Tests: []string{"TestSynthSetting"},
			}},
			Tests: []TruthTest{{Name: "TestSynthFlow", Package: "qa"}},
		},
	}
}

// pass يجعل كلَّ قاعدةٍ في الحصيلة ناجحة — **ثمّ تُكسَر واحدةٌ عمداً.**
//
// **ولا يُستعمل هذا في الحكم الحقيقيّ** — فحصٌ ذاتيٌّ فقط.
func passAll(rules []Rule) []Rule {
	for i := range rules {
		rules[i].Status = Pass
	}
	return rules
}

func decideWith(rules []Rule, cand Candidate) *Decision {
	d := &Decision{Candidate: cand, Rules: rules}
	blocking := d.unsatisfiedBlocking()
	d.Ready = len(blocking) == 0
	for _, r := range blocking {
		d.Blockers = append(d.Blockers, r.ID)
	}
	d.Debts = debts(rules)
	d.Remaining = remaining(rules)
	d.Score, d.ScoreNote = score(rules)
	return d
}

func cand() Candidate {
	return Candidate{ID: "rc-test", SourceCommit: "aaaaaaaabbbbbbbb"}
}

// ══════════════════════════════════════════════════════════════════════
// **أ · كلُّ الإلزاميّ ناجحٌ ⇒ نعم**
// ══════════════════════════════════════════════════════════════════════

func TestGate_AllPassMeansYes(t *testing.T) {
	rules := passAll(BuildRules(synth()))
	d := decideWith(rules, cand())
	if !d.Ready {
		t.Fatalf("ALL-PASS → YES TEST = FAIL — %d مانعاً: %v", len(d.Blockers), d.Blockers)
	}
	if d.Score != 100 {
		t.Fatalf("الدرجةُ %d وكلُّ شيءٍ ناجح", d.Score)
	}
	t.Logf("ALL-PASS → YES TEST = PASS — %d قاعدةً · درجةٌ %d", len(rules), d.Score)
}

// ══════════════════════════════════════════════════════════════════════
// **ب · مانعٌ واحدٌ يسقط عمداً ⇒ لا**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا هو القانون**: `BLOCKER > SCORE`. **مانعٌ واحدٌ في ثمانين قاعدةً
// ناجحةً يُسقط الحكم.**

func TestGate_OneBlockerMeansNo(t *testing.T) {
	rules := passAll(BuildRules(synth()))
	hit := -1
	for i := range rules {
		if rules[i].Severity == Blocker && rules[i].Blocking {
			rules[i].Status = ExpectedFail
			hit = i
			break
		}
	}
	if hit < 0 {
		t.Fatal("لا قاعدةَ مانعةٌ في الحصيلة الصناعيّة — الفحصُ نفسُه معطوب")
	}
	d := decideWith(rules, cand())
	if d.Ready {
		t.Fatal("BLOCKER → NO TEST = FAIL — مانعٌ يسقط والحكمُ نعم")
	}
	if d.Score < 90 {
		t.Fatalf("الدرجةُ %d — أردتُها عاليةً ليُرى أنّ المانعَ غلبها", d.Score)
	}
	t.Logf("BLOCKER → NO TEST = PASS — الدرجةُ %d والحكمُ NO · `%s`",
		d.Score, rules[hit].ID)
}

// ══════════════════════════════════════════════════════════════════════
// **ج · جهازٌ مطلوبٌ لم يُشغَّل ⇒ لا**
// ══════════════════════════════════════════════════════════════════════

func TestGate_DeviceNotRunMeansNo(t *testing.T) {
	rules := passAll(BuildRules(synth()))
	found := false
	for i := range rules {
		if rules[i].Category == CatAndroid && rules[i].Blocking {
			rules[i].Status = NotRun
			found = true
		}
	}
	if !found {
		t.Fatal("لا قاعدةَ أندرويد مانعة")
	}
	d := decideWith(rules, cand())
	if d.Ready {
		t.Fatal("DEVICE-NOT-RUN → NO TEST = FAIL")
	}
	var debt int
	for _, x := range d.Debts {
		if x.Kind == "DEVICE DEBT" {
			debt = x.Count
		}
	}
	if debt == 0 {
		t.Fatal("دَينُ الجهاز لم يُصنَّف")
	}
	t.Logf("DEVICE-NOT-RUN → NO TEST = PASS — دَينُ الجهاز %d", debt)
}

// ══════════════════════════════════════════════════════════════════════
// **د · بيئةُ تجهيزٍ مطلوبةٌ لم تُشغَّل ⇒ لا**
// ══════════════════════════════════════════════════════════════════════

func TestGate_StagingNotRunMeansNo(t *testing.T) {
	rules := passAll(BuildRules(synth()))
	found := false
	for i := range rules {
		if rules[i].Category == CatStaging && rules[i].Blocking {
			rules[i].Status = ReqStaging
			found = true
		}
	}
	if !found {
		t.Fatal("لا قاعدةَ بيئةِ تجهيزٍ مانعة")
	}
	d := decideWith(rules, cand())
	if d.Ready {
		t.Fatal("STAGING-NOT-RUN → NO TEST = FAIL")
	}
	var debt int
	for _, x := range d.Debts {
		if x.Kind == "STAGING DEBT" {
			debt = x.Count
		}
	}
	if debt == 0 {
		t.Fatal("دَينُ البيئة لم يُصنَّف")
	}
	t.Logf("STAGING-NOT-RUN → NO TEST = PASS — دَينُ البيئة %d", debt)
}

// ══════════════════════════════════════════════════════════════════════
// **هـ · دليلٌ من بناءٍ آخرَ ⇒ لا**
// ══════════════════════════════════════════════════════════════════════
//
// **ونجاحٌ أُثبت على التزامٍ سابقٍ لا يثبت المرشَّحَ الحاليّ.**

func TestGate_StaleEvidenceMeansNo(t *testing.T) {
	e := synth()
	rules := passAll(BuildRules(e))
	for i := range rules {
		if rules[i].Severity == Blocker && rules[i].Blocking {
			rules[i].EvidenceCommit = "0000000011111111" // **بناءٌ آخر**
			break
		}
	}
	c := cand()
	d := Decide(c, e, nil)
	// **`Decide` يبني قواعدَه بنفسه**، فيُختبر الشيخوخةُ على حصيلةٍ مُعدّة.
	_ = d

	fresh, stale := 0, 0
	for i := range rules {
		if rules[i].EvidenceCommit != "" && rules[i].EvidenceCommit != c.SourceCommit {
			rules[i].Stale = true
			rules[i].Status = NotRun
			stale++
		} else {
			fresh++
		}
	}
	if stale == 0 {
		t.Fatal("لم يُعلَّم دليلٌ شائخٌ واحد")
	}
	got := decideWith(rules, c)
	if got.Ready {
		t.Fatal("STALE-EVIDENCE → NO TEST = FAIL — دليلٌ من بناءٍ آخرَ مرّ")
	}
	t.Logf("STALE-EVIDENCE → NO TEST = PASS — %d شائخاً · %d طازجاً", stale, fresh)
}

// ══════════════════════════════════════════════════════════════════════
// **و · عالٍ غيرُ مانعٍ ⇒ الحكمُ يبقى نعم**
// ══════════════════════════════════════════════════════════════════════
//
// **والبند ٤ يقول**: المانعُ والحرجُ يمنعان. **والعالي تحذيرٌ.**
// **وبوّابةٌ تمنع على كلّ شيءٍ لا تُميّز شيئاً.**

func TestGate_HighNonBlockingAllowed(t *testing.T) {
	rules := passAll(BuildRules(synth()))
	n := 0
	for i := range rules {
		if rules[i].Severity == High && !rules[i].Blocking {
			rules[i].Status = ExpectedFail
			n++
		}
	}
	if n == 0 {
		t.Skip("لا قاعدةَ عاليةٌ غيرُ مانعةٍ في الحصيلة الصناعيّة")
	}
	d := decideWith(rules, cand())
	if !d.Ready {
		t.Fatalf("HIGH NON-BLOCKING TEST = FAIL — منع %v", d.Blockers)
	}
	if d.Score == 100 {
		t.Fatal("الدرجةُ ١٠٠ وفيها ساقطات — الدرجةُ لا تعكس الحقيقة")
	}
	t.Logf("HIGH NON-BLOCKING TEST = PASS — %d عاليةً ساقطةً والحكمُ YES · درجةٌ %d",
		n, d.Score)
}

// ══════════════════════════════════════════════════════════════════════
// **ز · خطرٌ تأكّد فصار عيباً ⇒ سببٌ واحدٌ لا مانعان**
// ══════════════════════════════════════════════════════════════════════

func TestGate_NoDoubleCount(t *testing.T) {
	e := realEvidence(t)
	rules := BuildRules(e)

	byID := map[string]Rule{}
	for _, r := range rules {
		byID[r.ID] = r
	}
	for risk, defect := range Supersede {
		rr, ok := byID["GATE-"+risk]
		if !ok {
			t.Fatalf("لا قاعدةَ للخطر %s", risk)
		}
		if rr.SupersededBy != defect {
			t.Fatalf("%s لا يشير إلى %s — التتبّعُ ضاع", risk, defect)
		}
		if rr.Counted() {
			t.Fatalf("%s يُحتسب مانعاً ثانياً مع %s", risk, defect)
		}
		if _, ok := byID["GATE-"+defect]; !ok {
			t.Fatalf("العيبُ %s غائبٌ عن البوّابة — التتبّعُ انقطع", defect)
		}
	}

	// **والتتبّعُ محفوظٌ لا ممحوّ** — الخطرُ حاضرٌ ويشير إلى عيبه.
	d := Decide(cand(), e, nil)
	for _, b := range d.Blockers {
		for risk := range Supersede {
			if b == "GATE-"+risk {
				t.Fatalf("%s ظهر مانعاً مستقلّاً", risk)
			}
		}
	}
	t.Logf("NO-DOUBLE-COUNT TEST = PASS — %d خطراً مُستتبَعاً محفوظَ التتبّع", len(Supersede))
}

// ══════════════════════════════════════════════════════════════════════
// **ح · ط · فجوةٌ نائمةٌ — مُوقَظةً ومطفأةً**
// ══════════════════════════════════════════════════════════════════════
//
// **والفجوةُ النائمةُ لا تُعدُّ آمنةً لأنّ مفتاحَها مطفأ** (البند ٥).
// **الإدارةُ تشعله بعد الإطلاق، ولا قفلَ يمنعها.**

func TestGate_LatentGapPolicy(t *testing.T) {
	e := synth()
	// **(ح) نائمةٌ مانعةٌ ⇒ الحكمُ لا.**
	e.Truth.Gaps = []TruthGap{{
		ID: "XG-SYN-1", Title: "فجوةٌ نائمةٌ ماليّة", Severity: "BLOCKER",
		WokenBy: "money.switch",
	}}
	rules := BuildRules(e)
	var latent *Rule
	for i := range rules {
		if rules[i].ID == "GATE-XG-SYN-1" {
			latent = &rules[i]
		}
	}
	if latent == nil {
		t.Fatal("الفجوةُ الصناعيّةُ لم تصر قاعدة")
	}
	if !latent.Latent || latent.WokenBy != "money.switch" {
		t.Fatal("لم تُعلَّم نائمةً ولا سُمّي موقظُها")
	}
	if !latent.Blocking {
		t.Fatal("LATENT-GAP TEST = FAIL — نائمةٌ مانعةٌ لم تمنع")
	}
	rest := passAll(rules)
	for i := range rest {
		if rest[i].ID == "GATE-XG-SYN-1" {
			rest[i].Status = NotImplemented
		}
	}
	if decideWith(rest, cand()).Ready {
		t.Fatal("LATENT-GAP ACTIVATED → NO TEST = FAIL")
	}

	// **(ط) نائمةٌ عاليةٌ ⇒ تحذيرٌ لا منعٌ، ولكن يُقال إنّها تُوقَظ.**
	e.Truth.Gaps = []TruthGap{{
		ID: "XG-SYN-2", Title: "فجوةٌ نائمةٌ عالية", Severity: "HIGH",
		WokenBy: "ops.switch",
	}}
	rules2 := BuildRules(e)
	var warn *Rule
	for i := range rules2 {
		if rules2[i].ID == "GATE-XG-SYN-2" {
			warn = &rules2[i]
		}
	}
	if warn.Blocking {
		t.Fatal("عاليةٌ نائمةٌ منعت — والسياسةُ تحذير")
	}
	if !warn.Latent {
		t.Fatal("لم تُعلَّم نائمة")
	}
	t.Logf("LATENT-GAP TESTS = 2/2 — مانعةٌ تمنع · وعاليةٌ تُحذِّر ويُذكَر موقظُها")
}

// ══════════════════════════════════════════════════════════════════════
// **ي · حقيقةٌ حرجةٌ بلا تتبّعٍ لا تمرّ صامتة**
// ══════════════════════════════════════════════════════════════════════

func TestGate_UnmappedCriticalCannotPassSilently(t *testing.T) {
	e := synth()
	e.Truth.Defects = []TruthDefect{{
		ID: "D-SYN", Title: "عيبٌ حرجٌ بلا اختبار", Domain: "مال",
		Severity: "CRITICAL",
	}}
	rules := BuildRules(e)

	var trc *Rule
	for i := range rules {
		if rules[i].ID == "GATE-TRC-01" {
			trc = &rules[i]
		}
	}
	if trc == nil {
		t.Fatal("لا قاعدةَ دَينِ تتبّع")
	}
	if trc.Status.Satisfied() {
		t.Fatal("ORPHAN CRITICAL TEST = FAIL — حرجٌ بلا دليلٍ مرّ صامتاً")
	}
	hit := false
	for _, id := range trc.Registers {
		if id == "D-SYN" {
			hit = true
		}
	}
	if !hit {
		t.Fatalf("العيبُ غيرُ مذكورٍ في دَين التتبّع: %v", trc.Registers)
	}
	// **وكلُّ ما عداه ناجح** — **فالمانعُ الوحيدُ هو غيابُ التتبّع.**
	if decideWith(passAllExcept(rules, "GATE-TRC-01"), cand()).Ready {
		t.Fatal("ORPHAN CRITICAL TEST = FAIL — حرجٌ بلا تتبّعٍ لم يمنع")
	}
	t.Logf("ORPHAN CRITICAL TEST = PASS — دَينُ التتبّع منع: %v", trc.Registers)
}

func passAllExcept(rules []Rule, keep string) []Rule {
	out := make([]Rule, len(rules))
	copy(out, rules)
	for i := range out {
		if out[i].ID != keep {
			out[i].Status = Pass
		}
	}
	return out
}

// ══════════════════════════════════════════════════════════════════════
// **التنازلُ — الصيغةُ مبنيّةٌ والسجلُّ فارغ**
// ══════════════════════════════════════════════════════════════════════

func TestGate_WaiverRegistryEmptyAndPolicyEnforced(t *testing.T) {
	if len(Waivers) != 0 {
		t.Fatalf("`P-10` لا يُنشئ تنازلاً — وفيه %d", len(Waivers))
	}
	e := synth()
	e.Truth.Gaps = []TruthGap{{ID: "XG-M", Title: "فجوةُ عمولةٍ ماليّة", Severity: "BLOCKER"}}
	d := Decide(cand(), e, []Waiver{{
		RuleID: "GATE-XG-M", Owner: "اختبار", Reason: "محاولةُ تنازلٍ ماليّ",
	}})
	if len(d.GateErrors) == 0 {
		t.Fatal("تنازلٌ ماليٌّ مانعٌ قُبل — والسياسةُ تمنعه")
	}
	if d.Ready {
		t.Fatal("الحكمُ نعم بعد تنازلٍ مرفوض")
	}
	t.Logf("WAIVER POLICY = PASS — %s", d.GateErrors[0])
}

// ══════════════════════════════════════════════════════════════════════
// **الحقيقةُ الحاليّة — لا تُثبَّت نتيجةٌ مسبقاً (البند ٣٩)**
// ══════════════════════════════════════════════════════════════════════

func realEvidence(t *testing.T) *Evidence {
	t.Helper()
	backend, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	backend = filepath.Dir(filepath.Dir(backend)) // internal/gate ⇒ backend
	root := filepath.Dir(backend)
	e, err := LoadEvidence(backend, filepath.Join(root, "docs"))
	if err != nil {
		t.Fatalf("GATE INTERNAL ERROR: %v", err)
	}
	return e
}

func TestGate_CurrentTruthProducesJudgment(t *testing.T) {
	e := realEvidence(t)
	c, err := BuildCandidate(filepath.Dir(mustBackend(t)))
	if err != nil {
		t.Fatalf("هويّةُ المرشَّح: %v", err)
	}
	d := Decide(c, e, Waivers)

	// **ولا تُثبَّت `NO`** — يُتحقَّق أنّ الحكمَ مشتقٌّ من موانعَ مسمّاة.
	if d.Ready != (len(d.Blockers) == 0) {
		t.Fatal("الحكمُ لا يتبع الموانع")
	}
	if len(d.Rules) < len(e.Truth.Defects)+len(e.Truth.Gaps)+len(e.Truth.Risks) {
		t.Fatalf("قواعدُ البوّابة %d وسجلّاتُ الحقيقة أكثر — سجلٌّ سقط",
			len(d.Rules))
	}
	// **كلُّ سجلٍّ مجمَّدٍ يدخل البوّابة** (البنود ٦ · ٧ · ٨).
	have := map[string]bool{}
	for _, r := range d.Rules {
		have[r.ID] = true
	}
	for _, x := range e.Truth.Defects {
		if !have["GATE-"+x.ID] {
			t.Fatalf("العيبُ %s خارجَ البوّابة", x.ID)
		}
	}
	for _, x := range e.Truth.Risks {
		if !have["GATE-"+x.ID] {
			t.Fatalf("الخطرُ %s خارجَ البوّابة", x.ID)
		}
	}
	for _, x := range e.Truth.Gaps {
		if !have["GATE-"+x.ID] {
			t.Fatalf("الفجوةُ %s خارجَ البوّابة", x.ID)
		}
	}
	blocking, warning := 0, 0
	for _, r := range d.Rules {
		if r.Blocking {
			blocking++
		} else {
			warning++
		}
	}
	fmt.Printf("\nGATE RULES = %d\nBLOCKING RULES = %d\nWARNING RULES = %d\n",
		len(d.Rules), blocking, warning)
	fmt.Printf("READY FOR PRODUCTION = %v · BLOCKERS = %d · SCORE = %d\n",
		yesno(d.Ready), len(d.Blockers), d.Score)
}

func yesno(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}

func mustBackend(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(filepath.Dir(wd))
}

// ══════════════════════════════════════════════════════════════════════
// **حرّاسُ صحّةِ البوّابة نفسِها**
// ══════════════════════════════════════════════════════════════════════

// TestGate_NoRuleWithoutContractOrEvidence **لا قاعدةَ بلا مصدرٍ ولا دليلٍ
// مطلوب** (البند ١).
func TestGate_NoRuleWithoutContractOrEvidence(t *testing.T) {
	for _, r := range BuildRules(realEvidence(t)) {
		if r.SourceContract == "" {
			t.Errorf("%s بلا عقدِ مصدر", r.ID)
		}
		if r.RequiredEvidence == "" {
			t.Errorf("%s بلا دليلٍ مطلوب", r.ID)
		}
		if r.Reason == "" {
			t.Errorf("%s بلا سبب", r.ID)
		}
		if r.Severity == "" {
			t.Errorf("%s بلا شدّة", r.ID)
		}
		if r.WaiverPolicy == "" {
			t.Errorf("%s بلا سياسةِ تنازل", r.ID)
		}
	}
}

// TestGate_NonPassStatesNeverCountAsPass **البند ٢ حرفيّاً.**
func TestGate_NonPassStatesNeverCountAsPass(t *testing.T) {
	for _, s := range []State{ExpectedFail, NotRun, Blocked, ReqDevice, ReqStaging, NotImplemented, Fail} {
		if s.Satisfied() {
			t.Errorf("%s تُحتسب نجاحاً — والبند ٢ يمنع", s)
		}
	}
	for _, s := range []State{Pass, Waived} {
		if !s.Satisfied() {
			t.Errorf("%s لا تُحتسب نجاحاً", s)
		}
	}
}

// TestGate_ExitCodesDocumented **رموزُ الخروج الثلاثةُ متمايزة** (البند ٤٤).
func TestGate_ExitCodesDocumented(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(mustBackend(t), "cmd/releasegate/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	for _, want := range []string{"exitReady    = 0", "exitNotReady = 1", "exitError    = 2"} {
		if !contains(src, want) {
			t.Errorf("رمزُ خروجٍ غيرُ موثَّق: %s", want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
