// Package gate بوّابةُ الإطلاق بالدليل — **`P-10`.**
//
// # السؤالُ الوحيد
//
//	READY FOR PRODUCTION = YES / NO
//
// **ولا درجةَ تجميليّة.** **ولا رأيَ بشريٍّ غيرَ موثَّق.** **ولا «كلُّ شيءٍ
// تقريباً جيّد».**
//
// # والقانونُ الأعلى
//
//	BLOCKER > SCORE
//
// **فدرجةُ ٩٢ من ١٠٠ لا تتغلّب على مانعٍ واحد.**
//
// # ولا حقيقةَ ثالثة
//
// **تُقرأ `TEST_TRUTH` والمصفوفاتُ الثمان** — **ولا يُكتب هنا عددُ عيبٍ
// ولا شدّةُ فجوة.**
package gate

import "sort"

// State حالُ قاعدةٍ في البوّابة (البند ٢).
type State string

const (
	Pass           State = "PASS"
	Fail           State = "FAIL"
	NotRun         State = "NOT_RUN"
	ExpectedFail   State = "EXPECTED_FAIL"
	Blocked        State = "BLOCKED"
	NotImplemented State = "NOT_IMPLEMENTED"
	ReqDevice      State = "REQUIRES_DEVICE"
	ReqStaging     State = "REQUIRES_STAGING"
	Waived         State = "WAIVED"
)

// Satisfied **هل تُحتسب نجاحاً؟**
//
// **ولا يُحتسب نجاحاً**: `EXPECTED_FAIL` ولا `NOT_RUN` ولا `BLOCKED` ولا
// `REQUIRES_DEVICE` ولا `REQUIRES_STAGING` (البند ٢).
//
// **و`WAIVED` وحدَها تُحتسب** — **ولا تنازلَ في هذا الإصدار** (البند ٣).
func (s State) Satisfied() bool { return s == Pass || s == Waived }

// Severity شدّةٌ وفق `TQ-1` (البند ٤).
type Severity string

const (
	Blocker  Severity = "BLOCKER"
	Critical Severity = "CRITICAL"
	High     Severity = "HIGH"
	Medium   Severity = "MEDIUM"
	Low      Severity = "LOW"
	// Unknown **شدّةٌ لم تُصرَّح** — **وتُعامَل معاملةَ العالية لا الهيّنة.**
	Unknown Severity = "UNKNOWN"
)

// rank ترتيبُ الشدّة — الأعلى أخطر.
func (s Severity) rank() int {
	switch s {
	case Blocker:
		return 5
	case Critical:
		return 4
	case High, Unknown:
		return 3
	case Medium:
		return 2
	case Low:
		return 1
	}
	return 3
}

// Category مجالُ القاعدة.
type Category string

const (
	CatFinancial   Category = "FINANCIAL"
	CatPrivacy     Category = "PRIVACY"
	CatSecurity    Category = "SECURITY"
	CatOrder       Category = "ORDER_INTEGRITY"
	CatConcurrency Category = "CONCURRENCY"
	CatFailure     Category = "FAILURE_RECOVERY"
	CatNotify      Category = "NOTIFICATION_REALTIME"
	CatAndroid     Category = "ANDROID_REAL_DEVICE"
	CatNavigation  Category = "NAVIGATION"
	CatBackground  Category = "BACKGROUND_DOZE"
	CatStaging     Category = "STAGING"
	CatOps         Category = "OPERATIONAL_READINESS"
	CatSettings    Category = "SETTINGS_READINESS"
	CatRBAC        Category = "ADMIN_RBAC"
	CatAudit       Category = "AUDIT"
	CatMedia       Category = "MEDIA_SECURITY"
	CatPerf        Category = "PERFORMANCE"
	CatLoad        Category = "LOAD_SOAK"
	CatBackup      Category = "BACKUP_RESTORE"
	CatHealth      Category = "HEALTH_READINESS"
	CatTrace       Category = "TRACEABILITY_DEBT"
	CatCoverage    Category = "TEST_COVERAGE"
	CatFreshness   Category = "EVIDENCE_FRESHNESS"
	CatTestInfra   Category = "TEST_INFRA_DEBT"
)

// WaiverPolicy هل يجوز التنازل عن القاعدة أصلاً؟ (البند ٣٧)
type WaiverPolicy string

const (
	// WaiverForbidden **لا تنازلَ البتّة** — المالُ والخصوصيّةُ والأمنُ
	// في شدّةٍ مانعةٍ أو حرجة.
	WaiverForbidden WaiverPolicy = "FORBIDDEN"
	// WaiverOwnerOnly **بقرار مالكٍ صريحٍ فقط** — ولم يُنشأ في `P-10`.
	WaiverOwnerOnly WaiverPolicy = "OWNER_DECISION_REQUIRED"
)

// Waiver تنازلٌ معتمد — **الصيغةُ مبنيّةٌ والسجلُّ فارغ** (البند ٣٧).
//
// **و`P-10` لا يُنشئ تنازلاً من نفسه.**
type Waiver struct {
	RuleID     string `json:"rule_id"`
	Owner      string `json:"owner"`
	Reason     string `json:"reason"`
	Scope      string `json:"scope"`
	Expiry     string `json:"expiry"`
	Risk       string `json:"risk"`
	Mitigation string `json:"mitigation"`
}

// Rule قاعدةُ بوّابةٍ واحدة (البند ١).
type Rule struct {
	ID          string   `json:"rule_id"`
	Category    Category `json:"category"`
	Requirement string   `json:"requirement"`
	// SourceContract **من أين جاء الإلزام** — عقدٌ أو سجلٌّ أو قرارُ مالك.
	SourceContract string   `json:"source_contract"`
	Severity       Severity `json:"severity"`
	// Latent **فجوةٌ نائمةٌ يوقظها إعداد** (البند ٥).
	Latent  bool   `json:"latent,omitempty"`
	WokenBy string `json:"woken_by,omitempty"`

	RequiredEvidence string `json:"required_evidence"`
	Status           State  `json:"current_status"`

	// Blocking **صنفٌ لا حال**: «هذه القاعدة تمنع الإطلاقَ **إن لم
	// تُستوفَ**». **وهي صحيحةٌ في قاعدةٍ ناجحةٍ أيضاً.**
	//
	// **وكانت تُنشَر باسم `blocking`** — **فقرأتها الآلاتُ «مانعةٌ
	// الآن» فأعلنت ٢٨ مانعاً والبوّابةُ تقول ٢٣** (`XG-37`).
	Blocking bool `json:"blocking_capable"`

	// CurrentlyBlocking **حالٌ لا صنف**: **مانعةٌ الآن فعلاً.**
	//
	//	Blocking && Counted() && !Status.Satisfied()
	//
	// **وهي وحدَها ما يُعَدّ** — **ويُحسَب في `Decide` لا يُملأ يدويّاً.**
	CurrentlyBlocking bool `json:"currently_blocking"`

	Reason string `json:"reason"`

	Registers []string `json:"related_registers,omitempty"`
	Tests     []string `json:"tests,omitempty"`

	WaiverPolicy WaiverPolicy `json:"waiver_policy"`

	// SupersededBy **الخطرُ الذي صار عيباً يشير إلى عيبه** (البند ٧).
	//
	// **والقاعدةُ المُستتبَعة لا تُعدُّ مانعاً ثانياً** — سببٌ واحدٌ لا
	// مانعان مصطنعان.
	SupersededBy string `json:"superseded_by,omitempty"`

	// EvidenceCommit **التزامُ الدليل** — يُقارَن بالمرشَّح (البند ٣٤).
	EvidenceCommit string `json:"evidence_commit,omitempty"`
	// Stale **دليلٌ من بناءٍ غيرِ المرشَّح.**
	Stale bool `json:"stale_evidence,omitempty"`
}

// Counted **هل تُحتسب هذه القاعدةُ مانعاً مستقلّاً؟**
//
// **والمُستتبَعةُ لا تُحتسب** — `R22 → D26` سببٌ واحد.
func (r Rule) Counted() bool { return r.SupersededBy == "" }

// Candidate هويّةُ مرشَّح الإطلاق (البند ٣٣).
//
// **ولا بوّابةَ نهائيّةٌ تعمل على «شجرةِ عملٍ حاليّة»** — **المرشَّحُ ثابتٌ
// أو الحكمُ باطل.**
type Candidate struct {
	ID           string   `json:"release_candidate_id"`
	SourceCommit string   `json:"source_commit"`
	Dirty        bool     `json:"working_tree_dirty"`
	BackendHash  string   `json:"backend_build_hash"`
	WebHash      string   `json:"web_build_hash"`
	APKHashes    []string `json:"apk_hashes"`
	ConfigSnap   string   `json:"config_snapshot"`
	Migration    string   `json:"migration_version"`
	Generated    string   `json:"generated_at"`
}

// Debt دَينٌ مصنَّف (البند ٤٠).
type Debt struct {
	Kind  string   `json:"kind"`
	Count int      `json:"count"`
	Items []string `json:"items"`
	Why   string   `json:"why"`
}

// Decision حكمُ البوّابة.
type Decision struct {
	Candidate Candidate `json:"candidate"`
	Ready     bool      `json:"ready_for_production"`
	Rules     []Rule    `json:"rules"`
	Waivers   []Waiver  `json:"waivers"`

	// Counts **العددُ منشورٌ لا مُستنتَج** — **فلا يعدّ قارئٌ بنفسه
	// فيخطئ** (`XG-37`).
	Counts Counts `json:"counts"`

	Blockers  []string `json:"release_blockers"`
	CriticalU []string `json:"critical_unresolved"`
	NotRunReq []string `json:"required_validations_not_run"`
	Warnings  []string `json:"warnings"`

	Debts []Debt `json:"debts"`

	// Remaining **الطريقُ إلى نعم** (البند ٤١).
	Remaining []string `json:"remaining_requirements_to_ready"`

	// Score **عرضٌ مساعدٌ لا قرار** (البند ٤٦).
	Score      int      `json:"advisory_score"`
	ScoreNote  string   `json:"advisory_score_note"`
	RunID      string   `json:"run_id"`
	Generated  string   `json:"generated_at"`
	TruthBase  string   `json:"truth_baseline"`
	GateErrors []string `json:"gate_errors,omitempty"`
}

// Counts **معجمُ الأعداد** — **ولا اسمَ يحمل معنيين.**
type Counts struct {
	// TotalRules كلُّ قاعدةٍ في التقرير — **بما فيها المُستتبَعة.**
	TotalRules int `json:"total_rules"`
	// BlockingCapable **صنفاً** — ناجحةً كانت أو ساقطة.
	BlockingCapable int `json:"blocking_capable_rules"`
	// CurrentBlockers **حالاً** — **وهو الرقمُ الوحيدُ الذي يُقرَّر به.**
	CurrentBlockers int `json:"current_blockers"`
	// SatisfiedBlockingCapable مانعةٌ صنفاً ومستوفاةٌ حالاً.
	SatisfiedBlockingCapable int `json:"satisfied_blocking_capable"`
	// Warnings غيرُ مانعةٍ صنفاً وغيرُ مستوفاةٍ حالاً.
	Warnings int `json:"warnings"`
	// SupersededBlockingCapable مانعةٌ صنفاً ومُستتبَعةٌ فلا تُعَدّ.
	//
	// **ولا تُخلَط بالمستوفاة** — **«مستوفاةٌ» تعني عملاً أُنجز،
	// و«مُستتبَعةٌ» تعني سبباً واحداً بمانعٍ واحد.**
	SupersededBlockingCapable int `json:"superseded_blocking_capable"`
	// Superseded كلُّ مُستتبَعةٍ — مانعةَ الصنف كانت أو لا.
	Superseded int `json:"superseded_not_counted"`
}

// blocking القواعدُ المانعةُ المحتسَبةُ غيرُ المُرضاة.
func (d *Decision) unsatisfiedBlocking() []Rule {
	var out []Rule
	for _, r := range d.Rules {
		if r.Blocking && r.Counted() && !r.Status.Satisfied() {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity.rank() != out[j].Severity.rank() {
			return out[i].Severity.rank() > out[j].Severity.rank()
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// countOf **المصدرُ الواحدُ للأعداد** — **تنادِيه الطرفيّةُ والملفُّ
// والحارسُ جميعاً، فلا يفترقون.**
func countOf(rules []Rule) Counts {
	var c Counts
	c.TotalRules = len(rules)
	for _, r := range rules {
		if !r.Counted() {
			c.Superseded++
		}
		if r.Blocking {
			c.BlockingCapable++
			switch {
			case !r.Counted():
				c.SupersededBlockingCapable++
			case !r.Status.Satisfied():
				c.CurrentBlockers++
			default:
				c.SatisfiedBlockingCapable++
			}
			continue
		}
		if r.Counted() && !r.Status.Satisfied() {
			c.Warnings++
		}
	}
	return c
}
