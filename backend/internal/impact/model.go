// Package impact محرّكُ أثر التغيير — **`P-9`.**
//
// # المشكلةُ الأصليّة
//
// **عند تعديل ملفٍّ أو إعدادٍ أو بابٍ أو تدفّق، لا يجوز أن يُقرأ المشروعُ
// كلُّه من الصفر لمعرفة ما انكسر.**
//
// # والقاعدةُ التي تحكم كلَّ سطرٍ هنا
//
//	NO FALSE CONFIDENCE
//
// **فما لم يُعرَف أثرُه يُوسَّع لا يُهمَل.** **و«لا أثر» تُقال بدليلٍ أو
// لا تُقال.**
//
// # ولا حقيقةَ ثانية (البند ١)
//
// **يُستهلَك ما وُلّد في `P-2…P-8`** — `TEST_TRUTH` والمصفوفاتُ الستّ.
// **ولا يُكتب هنا عددُ تدفّقٍ ولا اسمُ اختبارٍ بيد.**
package impact

import "sort"

// Category صنفُ الملفّ (البند ٣).
type Category string

const (
	CatBackend   Category = "PRODUCT_BACKEND"
	CatAdminWeb  Category = "ADMIN_WEB"
	CatAndCust   Category = "CUSTOMER_ANDROID"
	CatAndDriver Category = "DRIVER_ANDROID"
	CatAndMerch  Category = "MERCHANT_ANDROID"
	CatAndRep    Category = "REP_ANDROID"
	CatAndShared Category = "SHARED_ANDROID"
	CatMigration Category = "DATABASE_MIGRATION"
	CatSettings  Category = "SETTINGS_TRUTH"
	CatTestInfra Category = "TEST_INFRA"
	CatTest      Category = "TEST"
	CatDeploy    Category = "DEPLOY_OPS"
	CatDocs      Category = "DOCS_ONLY"
	CatTruthDocs Category = "PRODUCT_TRUTH_DOCS"
	CatUnknown   Category = "UNKNOWN"
)

// Mode وضعُ التشغيل (البند ٨).
type Mode string

const (
	ModeFast        Mode = "FAST"
	ModeImpacted    Mode = "IMPACTED"
	ModeFull        Mode = "FULL"
	ModeFinancial   Mode = "FINANCIAL"
	ModeSecurity    Mode = "SECURITY"
	ModeConcurrency Mode = "CONCURRENCY"
	ModeFailure     Mode = "FAILURE"
	ModeRealtime    Mode = "REALTIME"
	ModeAndroid     Mode = "ANDROID"
	ModeWeb         Mode = "WEB"
	ModeRelease     Mode = "RELEASE"
)

// Confidence ثقةُ التغطية (البند ٢٩).
type Confidence string

const (
	High   Confidence = "HIGH"
	Medium Confidence = "MEDIUM"
	Low    Confidence = "LOW"
)

// RiskClass صنفُ خطر التغيير (البند ٣٠).
type RiskClass string

const (
	RiskLow      RiskClass = "LOW"
	RiskMedium   RiskClass = "MEDIUM"
	RiskHigh     RiskClass = "HIGH"
	RiskCritical RiskClass = "CRITICAL"
)

// Depth مباشرٌ أم متعدٍّ (البند ٦).
type Depth string

const (
	Direct     Depth = "DIRECT"
	Transitive Depth = "TRANSITIVE"
)

// File ملفٌّ متغيّرٌ وتصنيفُه.
type File struct {
	Path     string   `json:"path"`
	Category Category `json:"category"`
	// Why لماذا صُنّف كذلك — **ولا تصنيفَ بلا سبب.**
	Why string `json:"why"`
}

// Reason سببُ اختيارِ أثرٍ — **وهو ما يجيب «لماذا هذا؟»** (البند ٢٧).
type Reason struct {
	// From الملفُّ الذي بدأ منه.
	From string `json:"from"`
	// Rule القاعدةُ التي أطلقته.
	Rule string `json:"rule"`
	// Path مسارُ الأثر مقروءاً.
	Path string `json:"path"`
	// Depth مباشرٌ أم متعدٍّ.
	Depth Depth `json:"depth"`
}

// TestSel اختبارٌ مختار.
type TestSel struct {
	// Target ما يُشغَّل — حزمةٌ أو نمطُ اسم.
	Target string `json:"target"`
	// Kind حزمةٌ أم اختبارٌ بعينه أم وحدةُ أندرويد.
	Kind string `json:"kind"`
	// Required إلزاميٌّ أم موصىً به (البند ٧).
	Required bool `json:"required"`
	// Reasons لماذا اختير.
	Reasons []Reason `json:"reasons"`
}

// Result حصيلةُ تحليلٍ واحد.
type Result struct {
	// ── هويّةُ التشغيل ─────────────────────────────────────────────
	Source string `json:"change_source"`
	Base   string `json:"base"`
	Head   string `json:"head"`

	Files []File `json:"changed_files"`

	// ── الأثر ──────────────────────────────────────────────────────
	Flows    []string `json:"impacted_flows"`
	Apps     []string `json:"impacted_apps"`
	Settings []string `json:"impacted_settings"`
	Defects  []string `json:"impacted_defects"`
	Risks    []string `json:"impacted_risks"`
	Gaps     []string `json:"impacted_gaps"`
	FinInv   []string `json:"impacted_financial_invariants"`
	Events   []string `json:"impacted_events"`

	Tests []TestSel `json:"selected_tests"`
	Modes []Mode    `json:"required_modes"`

	// ── ما لا يُثبَت محلّيّاً ───────────────────────────────────────
	DeviceRequired  []string `json:"real_device_required"`
	StagingRequired []string `json:"staging_required"`

	// ── الصدق ──────────────────────────────────────────────────────
	Unknowns   []string   `json:"unknowns"`
	Fallback   string     `json:"fallback,omitempty"`
	Confidence Confidence `json:"confidence"`
	Risk       RiskClass  `json:"risk_class"`
	Reasons    []Reason   `json:"reasons"`
	Warnings   []string   `json:"warnings,omitempty"`
}

// FullCommand **الأمرُ الكامل كما في اتّفاق العمل** — **ولا نسختان له.**
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا مهلةٌ صريحةٌ في توصيةٍ نصّيّة** (٢٠٢٦-٠٩-١٧)
// ══════════════════════════════════════════════════════════════════════
//
// **ومهلةُ `go` الافتراضيّةُ عشرُ دقائق** — **وحزمةُ `qa` وحدَها
// تتجاوزها** (قِيست ٩١٠ و١١٩٠ و١٢٩٨ ثانيةً). **فمن أخذ هذه التوصيةَ
// كما تُطبَع رأى `panic: test timed out`** — **وهو ليس سقوطَ فحص**،
// **فيُطارَد عطبٌ لا وجودَ له.**
//
// **وتوصيةٌ تُطبَع ولا تعمل أسوأُ من لا توصية.**
//
// (قرارُ المالك ٢٠٢٦-٠٩-١٧ — وهو نصُّ `CLAUDE.md` حرفاً،
//
//	يحرسه `TestFullFallbackMatchesWorkingAgreement`.)
const (
	// TestTimeout **سياسةُ المهلة الواحدة** — **ولا ثانيةَ لها.**
	//
	// **والأمرُ الضيّقُ يطلب `./internal/qa` في أكثرِ أحواله** —
	// **وهي وحدَها تجاوزت العشرَ دقائقِ الافتراضيّةَ في كلّ قياس.**
	// **فلو حملت الكاملةُ مهلةً والضيّقةُ لا، لَسقط الطريقُ الأكثرُ
	// سلوكاً** — **وهو ما وقع فعلاً** (قرارُ المالك ٢٠٢٦-٠٩-١٧).
	TestTimeout = "30m"

	// testCommandPrefix **بدايةُ كلّ أمرٍ يُوصى به** — كاملاً كان أو ضيّقاً.
	testCommandPrefix = "go test -timeout " + TestTimeout + " -count=1 -p 1"

	// FullCommand **الأمرُ الكامل كما في اتّفاق العمل.**
	FullCommand = testCommandPrefix + " ./..."
)

// Command أمرُ التشغيل الموصى به.
func (r *Result) Command() string {
	if r.Fallback != "" {
		return FullCommand + "   # " + r.Fallback
	}
	pkgs := map[string]bool{}
	for _, t := range r.Tests {
		if t.Kind == "package" {
			pkgs[t.Target] = true
		}
	}
	if len(pkgs) == 0 {
		return FullCommand + "   # لا هدفَ ضيّقٌ استُنتج"
	}
	var list []string
	for p := range pkgs {
		list = append(list, "./"+p)
	}
	sort.Strings(list)
	out := testCommandPrefix
	for _, p := range list {
		out += " " + p
	}
	return out
}

func addUniq(dst *[]string, vals ...string) {
	seen := map[string]bool{}
	for _, v := range *dst {
		seen[v] = true
	}
	for _, v := range vals {
		if v != "" && !seen[v] {
			seen[v] = true
			*dst = append(*dst, v)
		}
	}
}

func sortAll(r *Result) {
	for _, s := range []*[]string{&r.Flows, &r.Apps, &r.Settings, &r.Defects,
		&r.Risks, &r.Gaps, &r.FinInv, &r.Events, &r.DeviceRequired,
		&r.StagingRequired, &r.Unknowns, &r.Warnings} {
		sort.Strings(*s)
	}
	sort.Slice(r.Tests, func(i, j int) bool { return r.Tests[i].Target < r.Tests[j].Target })
	sort.Slice(r.Modes, func(i, j int) bool { return r.Modes[i] < r.Modes[j] })
	sort.Slice(r.Files, func(i, j int) bool { return r.Files[i].Path < r.Files[j].Path })
}
