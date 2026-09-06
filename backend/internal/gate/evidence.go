package gate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ══════════════════════════════════════════════════════════════════════
// **طبقةُ الدليل — تقرأ ولا تُعلن**
// ══════════════════════════════════════════════════════════════════════
//
// **كلُّ رقمٍ هنا مقروءٌ من `TEST_TRUTH` أو من المصفوفات الثمان**
// (البند ٤٧). **ولا يُكتب `27` ولا `24` ولا `26` ولا `95` بيد.**

// TruthDefect عيبٌ كما في الحقيقة.
type TruthDefect struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Domain   string `json:"domain"`
	Severity string `json:"severity"`
	// Fixed **دليلُ الإصلاح** — تقرؤه البوّابةُ فلا تعدّه مانعاً.
	//
	// **وكان يُقرأ للفجوات وحدَها**، **فعيبٌ أُغلق بدليلٍ وحرّاسٍ يبقى
	// `EXPECTED_FAIL` إلى الأبد.**
	Fixed  string   `json:"fixed,omitempty"`
	Tests  []string `json:"tests"`
	Status string   `json:"status"`
}

// TruthRisk خطرٌ كما في الحقيقة.
type TruthRisk struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Domain string   `json:"domain"`
	Flows  []string `json:"flows"`
	Tests  []string `json:"tests"`
	Status string   `json:"status"`
}

// TruthGap فجوةٌ كما في الحقيقة.
type TruthGap struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Severity string   `json:"severity"`
	WokenBy  string   `json:"woken_by"`
	Fixed    string   `json:"fixed"`
	Tests    []string `json:"tests"`
	Status   string   `json:"status"`
}

// TruthSetting إعدادٌ مُغيِّرٌ للسلوك.
type TruthSetting struct {
	Key       string   `json:"key"`
	Group     string   `json:"group"`
	Kind      string   `json:"kind"`
	Sensitive bool     `json:"sensitive"`
	Money     bool     `json:"money"`
	Flows     []string `json:"flows"`
	Tests     []string `json:"tests"`
	Status    string   `json:"status"`
}

// TruthFlow تدفّقٌ عابرٌ للأنظمة.
type TruthFlow struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Severity string   `json:"severity"`
	Money    bool     `json:"money"`
	Defects  []string `json:"defects"`
	Risks    []string `json:"risks"`
	Gaps     []string `json:"gaps"`
	Tests    []string `json:"tests"`
	Status   string   `json:"status"`
}

// TruthTest دالّةُ اختبارٍ مجرودة.
type TruthTest struct {
	Name    string   `json:"name"`
	File    string   `json:"file"`
	Package string   `json:"package"`
	Purpose string   `json:"purpose"`
	Modes   []string `json:"modes"`
	Orphan  bool     `json:"orphan"`
}

// Truth ما يُقرأ من `TEST_TRUTH.json`.
type Truth struct {
	Baseline     string         `json:"baseline"`
	Flows        []TruthFlow    `json:"flows"`
	Defects      []TruthDefect  `json:"defects"`
	Risks        []TruthRisk    `json:"risks"`
	Gaps         []TruthGap     `json:"gaps"`
	Settings     []TruthSetting `json:"settings"`
	Tests        []TruthTest    `json:"tests"`
	Stale        []string       `json:"stale"`
	CoverageGaps []string       `json:"coverage_gaps"`
}

// MatrixRow صفُّ دليلٍ من إحدى المصفوفات — **بصيغةٍ واحدةٍ لها كلِّها.**
//
// **والمصفوفاتُ تختلف حقولاً وتتّفق جوهراً**: معرِّفٌ · سجلّاتٌ يمسّها ·
// نتيجةٌ · دليل.
type MatrixRow struct {
	Matrix    string   `json:"matrix"`
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Registers []string `json:"registers"`
	Result    string   `json:"result"`
	Evidence  string   `json:"evidence"`
	Tests     []string `json:"tests"`
	Kind      string   `json:"kind,omitempty"`
	App       string   `json:"app,omitempty"`
}

// Evidence كلُّ ما تستند إليه البوّابة.
type Evidence struct {
	Truth Truth

	// Rows صفوفُ المصفوفات كلِّها مجموعةً.
	Rows []MatrixRow
	// ByRegister **الدليلُ مفهرَساً بالسجلّ** — `D26` ⇒ ما يمسُّه.
	ByRegister map[string][]MatrixRow

	// FinChecks حالُ ثوابت `P-4`.
	FinChecks map[string]string
	// FinCounts عدّادُ `FINANCIAL_INVARIANTS`.
	FinCounts map[string]int

	// Android عدّادُ مصفوفة الأجهزة.
	AndroidCounts map[string]int
	// AndroidDevices الأجهزةُ وحالُها.
	AndroidDevices []MatrixRow

	// Health ما يفحصه `/healthz` فعلاً — **مقروءٌ من الشيفرة.**
	HealthCovers []string
	// CriticalDeps التبعيّاتُ الحرجةُ للتشغيل — **مشتقّةٌ من الإقلاع.**
	CriticalDeps []string
	// ReadinessEndpoint هل ثمّةَ بابُ جهوزيّةٍ منفصل؟
	ReadinessEndpoint bool

	// Impact حصيلةُ `P-9` الأخيرة — لقياس دَين التتبّع.
	ImpactAddressable map[string][]string
}

// LoadEvidence يجمع الدليلَ كلَّه — **ويسقط إن نقص، ولا يُخمّن.**
func LoadEvidence(backendRoot, docsRoot string) (*Evidence, error) {
	ev := &Evidence{
		ByRegister: map[string][]MatrixRow{},
		FinChecks:  map[string]string{},
	}
	sys := filepath.Join(docsRoot, "testing/system")

	if err := readJSON(filepath.Join(sys, "TEST_TRUTH.json"), &ev.Truth); err != nil {
		return nil, fmt.Errorf("حقيقةُ الاختبار: %w", err)
	}
	if len(ev.Truth.Defects) == 0 || len(ev.Truth.Gaps) == 0 {
		return nil, fmt.Errorf("حقيقةُ الاختبار فارغةٌ من السجلّات — بوّابةٌ بلا أساس")
	}

	if err := ev.loadMatrices(sys); err != nil {
		return nil, err
	}
	if err := ev.probeRuntime(backendRoot); err != nil {
		return nil, err
	}
	ev.indexRegisters()
	return ev, nil
}

func readJSON(path string, into any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, into)
}

// loadMatrices يقرأ المصفوفاتِ الستَّ ذاتِ النتائج.
//
// **وكلُّ مصفوفةٍ تُترجَم إلى `MatrixRow`** — **فالبوّابةُ لا تعرف صيغةَ
// كلٍّ منها، تعرف الجوهر.**
func (e *Evidence) loadMatrices(sys string) error {
	// ── الثوابتُ الماليّة ────────────────────────────────────────
	var fin struct {
		Checks []struct {
			ID     string `json:"id"`
			Family string `json:"family"`
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"checks"`
		Counts map[string]int `json:"counts"`
	}
	if err := readJSON(filepath.Join(sys, "FINANCIAL_INVARIANTS.json"), &fin); err != nil {
		return fmt.Errorf("الثوابتُ الماليّة: %w", err)
	}
	e.FinCounts = fin.Counts
	for _, c := range fin.Checks {
		e.FinChecks[c.ID] = c.Status
		e.Rows = append(e.Rows, MatrixRow{
			Matrix: "FINANCIAL_INVARIANTS", ID: c.ID, Title: c.Name,
			Result: c.Status, Kind: c.Family,
		})
	}

	// ── التزامن ──────────────────────────────────────────────────
	var conc struct {
		Races []struct {
			ID        string   `json:"id"`
			Title     string   `json:"title"`
			Registers []string `json:"registers"`
			Result    string   `json:"result"`
			Evidence  string   `json:"evidence"`
			Tests     []string `json:"tests"`
		} `json:"races"`
	}
	if err := readJSON(filepath.Join(sys, "CONCURRENCY_MATRIX.json"), &conc); err != nil {
		return fmt.Errorf("مصفوفةُ التزامن: %w", err)
	}
	for _, r := range conc.Races {
		e.Rows = append(e.Rows, MatrixRow{Matrix: "CONCURRENCY", ID: r.ID,
			Title: r.Title, Registers: r.Registers, Result: r.Result,
			Evidence: r.Evidence, Tests: r.Tests})
	}

	// ── حقنُ الفشل ───────────────────────────────────────────────
	var fail struct {
		Flows []struct {
			ID        string   `json:"id"`
			Title     string   `json:"title"`
			Registers []string `json:"registers"`
			Result    string   `json:"result"`
			Evidence  string   `json:"evidence"`
			Tests     []string `json:"tests"`
		} `json:"flows"`
	}
	if err := readJSON(filepath.Join(sys, "FAILURE_INJECTION_MATRIX.json"), &fail); err != nil {
		return fmt.Errorf("مصفوفةُ حقن الفشل: %w", err)
	}
	for _, r := range fail.Flows {
		e.Rows = append(e.Rows, MatrixRow{Matrix: "FAILURE", ID: r.ID,
			Title: r.Title, Registers: r.Registers, Result: r.Result,
			Evidence: r.Evidence, Tests: r.Tests})
	}

	// ── عقودُ الأحداث ────────────────────────────────────────────
	var ev struct {
		Events []struct {
			ID        string   `json:"id"`
			Title     string   `json:"title"`
			Registers []string `json:"registers"`
			Status    string   `json:"status"`
			Evidence  string   `json:"evidence"`
			Tests     []string `json:"tests"`
		} `json:"events"`
	}
	if err := readJSON(filepath.Join(sys, "EVENT_CONTRACT_MATRIX.json"), &ev); err != nil {
		return fmt.Errorf("مصفوفةُ عقود الأحداث: %w", err)
	}
	for _, r := range ev.Events {
		e.Rows = append(e.Rows, MatrixRow{Matrix: "EVENTS", ID: r.ID,
			Title: r.Title, Registers: r.Registers, Result: r.Status,
			Evidence: r.Evidence, Tests: r.Tests})
	}

	// ── أندرويد ──────────────────────────────────────────────────
	var and struct {
		Cases []struct {
			ID        string   `json:"id"`
			App       string   `json:"app"`
			Title     string   `json:"title"`
			Kind      string   `json:"kind"`
			Registers []string `json:"registers"`
			Test      string   `json:"test"`
			Result    string   `json:"result"`
			Evidence  string   `json:"evidence"`
		} `json:"cases"`
		Counts map[string]int `json:"counts"`
	}
	if err := readJSON(filepath.Join(sys, "ANDROID_TEST_MATRIX.json"), &and); err != nil {
		return fmt.Errorf("مصفوفةُ أندرويد: %w", err)
	}
	e.AndroidCounts = and.Counts
	for _, r := range and.Cases {
		e.Rows = append(e.Rows, MatrixRow{Matrix: "ANDROID", ID: r.ID,
			Title: r.Title, Registers: r.Registers, Result: r.Result,
			Evidence: r.Evidence, Tests: []string{r.Test}, Kind: r.Kind, App: r.App})
	}

	var dev struct {
		Devices []struct {
			ID     string `json:"id"`
			Model  string `json:"model"`
			Status string `json:"status"`
			Why    string `json:"why"`
		} `json:"devices"`
	}
	if err := readJSON(filepath.Join(sys, "ANDROID_DEVICE_MATRIX.json"), &dev); err != nil {
		return fmt.Errorf("مصفوفةُ الأجهزة: %w", err)
	}
	for _, d := range dev.Devices {
		e.AndroidDevices = append(e.AndroidDevices, MatrixRow{
			Matrix: "DEVICES", ID: d.ID, Title: d.Model, Result: d.Status, Evidence: d.Why})
	}

	// ── أثرُ التغيير — **لقياس دَين التتبّع لا لاختيار الاختبارات** ──
	var imp struct {
		DomainRules []struct {
			Name string `json:"name"`
		} `json:"domain_rules"`
	}
	// **وغيابُه لا يُسقط البوّابة** — يُسجَّل نقصاً في التتبّع.
	_ = readJSON(filepath.Join(sys, "CHANGE_IMPACT_RULES.json"), &imp)
	return nil
}

// indexRegisters يفهرس الدليلَ بالسجلّ الذي يمسّه.
func (e *Evidence) indexRegisters() {
	for _, r := range e.Rows {
		for _, reg := range r.Registers {
			e.ByRegister[reg] = append(e.ByRegister[reg], r)
		}
	}
}

// probeRuntime يقيس ما لا تقوله المصفوفات — **من الشيفرة نفسِها.**
func (e *Evidence) probeRuntime(backendRoot string) error {
	h, err := os.ReadFile(filepath.Join(backendRoot, "internal/server/health.go"))
	if err != nil {
		return fmt.Errorf("فحصُ الصحّة: %w", err)
	}
	src := string(h)
	for dep, marker := range map[string]string{
		"postgres": "s.pg.Ping",
		"redis":    "s.rdb.Ping",
		"media":    "uploads",
		"push":     "push",
		"migrate":  "migrat",
	} {
		if strings.Contains(strings.ToLower(src), strings.ToLower(marker)) {
			e.HealthCovers = append(e.HealthCovers, dep)
		}
	}
	sort.Strings(e.HealthCovers)

	srv, err := os.ReadFile(filepath.Join(backendRoot, "internal/server/server.go"))
	if err != nil {
		return fmt.Errorf("الموجّه: %w", err)
	}
	e.ReadinessEndpoint = strings.Contains(string(srv), `"/readyz"`) ||
		strings.Contains(string(srv), `"/ready"`)

	// **التبعيّاتُ الحرجةُ تُشتقّ من متطلّبات الإقلاع** — لا تُكتب.
	cfg, err := os.ReadFile(filepath.Join(backendRoot, "internal/config/config.go"))
	if err != nil {
		return fmt.Errorf("الضبط: %w", err)
	}
	c := string(cfg)
	for env, dep := range map[string]string{
		"DATABASE_URL": "postgres",
		"REDIS_URL":    "redis",
	} {
		if strings.Contains(c, env) {
			e.CriticalDeps = append(e.CriticalDeps, dep)
		}
	}
	// **ومخزنُ الوسائط تبعيّةٌ حيّةٌ أيضاً** — الصورُ تُقرأ منه في كلّ طلب.
	if _, err := os.Stat(filepath.Join(backendRoot, "internal/media")); err == nil {
		e.CriticalDeps = append(e.CriticalDeps, "media")
	}
	// **والدفعُ تبعيّةٌ خارجيّة** — و`P-7` أثبت أنّ سقوطَها صامت.
	if _, err := os.Stat(filepath.Join(backendRoot, "internal/push")); err == nil {
		e.CriticalDeps = append(e.CriticalDeps, "push")
	}
	sort.Strings(e.CriticalDeps)
	return nil
}

// ── قراءاتٌ مشتقّة ───────────────────────────────────────────────

// DefectByID عيبٌ بمعرّفه.
func (e *Evidence) DefectByID(id string) (TruthDefect, bool) {
	for _, d := range e.Truth.Defects {
		if d.ID == id {
			return d, true
		}
	}
	return TruthDefect{}, false
}

// FlowSeverityOf أشدُّ تدفّقٍ يمسُّ سجلّاً — **شدّةُ الخطر تُستعار منه.**
//
// **وسجلُّ المخاطر لا عمودَ شدّةٍ فيه** (قِيس في `P-10`)، **فتُشتقّ من
// أخطر تدفّقٍ يحمله بدل أن تُخترَع.**
func (e *Evidence) FlowSeverityOf(register string) Severity {
	best := Severity("")
	for _, f := range e.Truth.Flows {
		hit := false
		for _, xs := range [][]string{f.Defects, f.Risks, f.Gaps} {
			for _, x := range xs {
				if x == register {
					hit = true
				}
			}
		}
		if !hit {
			continue
		}
		s := Severity(f.Severity)
		if best == "" || s.rank() > best.rank() {
			best = s
		}
	}
	if best == "" {
		return Unknown
	}
	return best
}

// EvidenceFor أسطرُ الدليل التي تمسُّ سجلّاً.
func (e *Evidence) EvidenceFor(register string) []MatrixRow { return e.ByRegister[register] }

// ResultOf أقوى نتيجةٍ مسجَّلةٍ لسجلّ — **والأسوأُ يغلب.**
//
// **ونتيجةٌ واحدةٌ سيّئةٌ لا يمحوها عشرُ نتائجَ حسنة.**
func (e *Evidence) ResultOf(register string) (string, string) {
	rank := map[string]int{
		"DEFECT_REPRODUCED": 6, "RISK_CONFIRMED": 6, "EXPECTED_FAIL": 5,
		"UNPROVEN": 4, "NOT_RUN": 4, "PARTIAL": 3, "HELD": 2,
		"PROVEN": 1, "PASS": 1,
	}
	worst, why := "", ""
	for _, r := range e.ByRegister[register] {
		if worst == "" || rank[r.Result] > rank[worst] {
			worst, why = r.Result, fmt.Sprintf("%s/%s: %s", r.Matrix, r.ID, r.Evidence)
		}
	}
	return worst, why
}
