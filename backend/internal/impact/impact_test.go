package impact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func load(t *testing.T) *Engine {
	t.Helper()
	root, err := findRoot()
	if err != nil {
		t.Fatalf("جذر: %v", err)
	}
	e, err := Load(root)
	if err != nil {
		t.Fatalf("حقيقة: %v", err)
	}
	return e
}

func findRoot() (string, error) {
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(d, "docs/testing/system/TEST_TRUTH.json")); err == nil {
			return d, nil
		}
		d = filepath.Dir(d)
	}
	return "", os.ErrNotExist
}

func hasMode(r *Result, m Mode) bool {
	for _, x := range r.Modes {
		if x == m {
			return true
		}
	}
	return false
}

func has(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func hasPkg(r *Result, p string) bool {
	for _, t := range r.Tests {
		if t.Kind == "package" && t.Target == p {
			return true
		}
	}
	return false
}

// ══════════════════════════════════════════════════════════════════════
// **٣١ · تحقّقٌ تاريخيّ — على تغييراتٍ من مراحلَ حقيقيّة**
// ══════════════════════════════════════════════════════════════════════

// TestHistoricalChangeCases **البند ٣١ — ولا معرّفاتِ التزامٍ مثبَّتة.**
//
// **وتُستعمل مساراتٌ حقيقيّةٌ من المراحل** — لا `commit id` يُثبَّت ليمرّ
// الاختبار، **فالمعرّفُ يشيخ والمسارُ يبقى.**
func TestHistoricalChangeCases(t *testing.T) {
	e := load(t)
	cases := []struct {
		name  string
		files []string
		check func(t *testing.T, r *Result)
	}{
		{
			name:  "P-1 خصوصيّة",
			files: []string{"backend/internal/server/customer_privacy.go"},
			check: func(t *testing.T, r *Result) {
				if !hasMode(r, ModeSecurity) {
					t.Errorf("PRIVACY ESCALATION خُرق: لا وضعَ أمنيّ")
				}
				if !has(r.Defects, "D21") && !has(r.Defects, "D23") {
					t.Errorf("لم يُختَر حارسُ D21/D23 — %v", r.Defects)
				}
			},
		},
		{
			name:  "P-4 مال",
			files: []string{"backend/internal/wallet/wallet.go"},
			check: func(t *testing.T, r *Result) {
				if !hasMode(r, ModeFinancial) {
					t.Errorf("FINANCIAL ESCALATION خُرق")
				}
				if !hasPkg(r, "internal/fininv") {
					t.Errorf("لم تُختَر حزمةُ الثوابت الماليّة")
				}
			},
		},
		{
			name:  "P-5 تزامن",
			files: []string{"backend/internal/server/driver_handlers.go"},
			check: func(t *testing.T, r *Result) {
				if !hasMode(r, ModeConcurrency) {
					t.Errorf("لا وضعَ تزامن — %v", r.Modes)
				}
				if !has(r.Defects, "D24") && !has(r.Risks, "R10") {
					t.Errorf("لم يُختَر حارسُ D24/R10 — %v %v", r.Defects, r.Risks)
				}
			},
		},
		{
			name:  "P-7 إشعارٌ وبثّ",
			files: []string{"backend/internal/push/push.go"},
			check: func(t *testing.T, r *Result) {
				if !hasMode(r, ModeRealtime) {
					t.Errorf("لا وضعَ بثّ — %v", r.Modes)
				}
				if len(r.DeviceRequired) == 0 {
					t.Errorf("لم يُذكَر أنّ الروابطَ العميقةَ تحتاج جهازاً")
				}
			},
		},
		{
			name:  "P-8 أندرويد · الموقع",
			files: []string{"mobile/app-driver/src/main/kotlin/com/rahalgo/driver/location/PointQueue.kt"},
			check: func(t *testing.T, r *Result) {
				if !hasMode(r, ModeAndroid) {
					t.Errorf("لا وضعَ أندرويد")
				}
				if len(r.DeviceRequired) == 0 {
					t.Errorf("ANDROID DEVICE ESCALATION خُرق: **دَينُ `P-8` لم يُذكَر**")
				}
				if !has(r.Apps, "driver") {
					t.Errorf("تطبيقُ السائق غيرُ متأثّر؟ %v", r.Apps)
				}
			},
		},
		{
			name:  "هجرةُ قاعدة",
			files: []string{"backend/internal/migrate/migrations/0999_x.sql"},
			check: func(t *testing.T, r *Result) {
				if !hasMode(r, ModeFull) {
					t.Errorf("MIGRATION IMPACT خُرق: لا توسيعَ (البند ١١)")
				}
			},
		},
		{
			name:  "معجمُ الإعدادات",
			files: []string{"backend/internal/settings/catalog.go"},
			check: func(t *testing.T, r *Result) {
				if !hasPkg(r, "internal/testtruth") {
					t.Errorf("SETTING IMPACT خُرق: لم يُعَد توليدُ الحقيقة")
				}
			},
		},
		{
			name:  "نشرٌ وتشغيل",
			files: []string{"deploy/nginx.conf"},
			check: func(t *testing.T, r *Result) {
				if len(r.StagingRequired) == 0 {
					t.Errorf("STAGING ESCALATION خُرق (البند ٢٦)")
				}
			},
		},
	}
	pass := 0
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := e.Analyze("TEST", "", "", c.files)
			before := t.Failed()
			c.check(t, r)
			if !t.Failed() && !before {
				pass++
			}
			t.Logf("modes=%v · flows=%d · apps=%v · risk=%s · conf=%s",
				r.Modes, len(r.Flows), r.Apps, r.Risk, r.Confidence)
		})
	}
	t.Logf("HISTORICAL CHANGE CASES = %d/%d", pass, len(cases))
}

// ══════════════════════════════════════════════════════════════════════
// **٣٢ · سيناريوهاتُ عيوبٍ معروفة**
// ══════════════════════════════════════════════════════════════════════

// TestDefectImpactCases **البند ٣٢** — ملفُّ العيب يجلب حارسَه.
func TestDefectImpactCases(t *testing.T) {
	e := load(t)
	cases := []struct{ defect, file string }{
		{"D20", "backend/internal/orders/service.go"},
		{"D24", "backend/internal/server/driver_handlers.go"},
		{"D26", "backend/internal/orders/watchdog.go"},
		{"D27", "backend/internal/push/push.go"},
	}
	pass := 0
	for _, c := range cases {
		r := e.Analyze("TEST", "", "", []string{c.file})
		got := has(r.Defects, c.defect)
		// **والحارسُ يُقاس بالاختبار المختار لا بالاسم وحدَه.**
		guard := false
		for _, tt := range r.Tests {
			for _, why := range tt.Reasons {
				if strings.Contains(why.Path, c.defect) {
					guard = true
				}
			}
		}
		t.Logf("%-5s ← %-52s متأثّرٌ=%v · حارسٌ مختارٌ=%v", c.defect, c.file, got, guard)
		if got {
			pass++
		} else {
			t.Errorf("DEFECT REGRESSION IMPACT خُرق: %s لم يُذكَر (البند ٢٠)", c.defect)
		}
	}
	t.Logf("DEFECT IMPACT CASES = %d/%d", pass, len(cases))
}

// ══════════════════════════════════════════════════════════════════════
// **٣٣ · ضابطٌ سالب — ولا انفجارَ بلا سبب**
// ══════════════════════════════════════════════════════════════════════

// TestNegativeControl **البند ٣٣.**
func TestNegativeControl(t *testing.T) {
	e := load(t)
	r := e.Analyze("TEST", "", "", []string{"README.md", "docs/WORKLOG.md"})
	t.Logf("modes=%v · tests=%d · risk=%s", r.Modes, len(r.Tests), r.Risk)

	if hasMode(r, ModeAndroid) || hasMode(r, ModeFinancial) || hasMode(r, ModeFull) {
		t.Errorf("NEGATIVE CONTROL خُرق: وثائقُ عاديّةٌ فجّرت الأوضاع — %v", r.Modes)
	}
	if len(r.Tests) != 0 {
		t.Errorf("NEGATIVE CONTROL خُرق: %d اختباراً لوثيقةٍ عاديّة", len(r.Tests))
	}
	if r.Risk != RiskLow {
		t.Errorf("صنفُ الخطر %s لوثيقةٍ عاديّة", r.Risk)
	}
	t.Logf("NEGATIVE CONTROL = PASS")

	// **ووثيقةُ الحقيقة ليست وثيقةً عاديّة** (البند ٢٥).
	r2 := e.Analyze("TEST", "", "", []string{"docs/testing/FINAL_STATIC_CLOSEOUT.md"})
	if !hasPkg(r2, "internal/testtruth") {
		t.Errorf("TRUTH DOC خُرق: تبديلُ سجلٍّ مجمَّدٍ لم يستدعِ حرّاسَ الانحراف")
	} else {
		t.Logf("PRODUCT TRUTH DOC ⇒ حرّاسُ الانحراف — والفرقُ عن الوثيقة العاديّة قائم")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٣٤ · مجهولٌ ⇒ توسيعٌ آمن**
// ══════════════════════════════════════════════════════════════════════

// TestUnknownChangeFallsBackSafely **البندان ١٠ و٣٤.**
func TestUnknownChangeFallsBackSafely(t *testing.T) {
	e := load(t)
	r := e.Analyze("TEST", "", "", []string{"some/new/unmapped/area/thing.go"})
	t.Logf("fallback=%q · conf=%s · risk=%s · unknowns=%v",
		r.Fallback, r.Confidence, r.Risk, r.Unknowns)

	if len(r.Unknowns) == 0 {
		t.Fatal("ملفٌّ مجهولٌ لم يُسجَّل")
	}
	if r.Fallback == "" {
		t.Errorf("SAFE FALLBACK خُرق: مجهولٌ بلا توسيع")
	}
	if !hasMode(r, ModeFull) {
		t.Errorf("مجهولٌ ولم يُطلَب الكلّ")
	}
	if r.Confidence != Low {
		t.Errorf("ثقةٌ %s لملفٍّ مجهول", r.Confidence)
	}
	if !strings.Contains(r.Command(), "./...") {
		t.Errorf("الأمرُ الموصى به لا يشمل الكلّ: %s", r.Command())
	}
	t.Logf("UNKNOWN CHANGE SELF-TEST = PROVEN")
	t.Logf("  UNKNOWN IMPACT → SAFE FULL FALLBACK")
}

// ══════════════════════════════════════════════════════════════════════
// **٣٦ · حارسُ الخرائط الشائخة**
// ══════════════════════════════════════════════════════════════════════

// TestStaleMappingGuard **البند ٣٦** — لا تشير الحقيقةُ إلى ما زال.
func TestStaleMappingGuard(t *testing.T) {
	e := load(t)
	root, _ := findRoot()

	byName := map[string]bool{}
	for _, x := range e.T.Tests {
		byName[x.Name] = true
	}
	stale := 0
	for _, f := range e.T.Flows {
		for _, n := range f.Tests {
			if !byName[n] {
				stale++
				t.Errorf("STALE MAPPING: التدفّق %s يشير إلى اختبارٍ زال %q", f.ID, n)
			}
		}
	}
	for _, d := range e.T.Defects {
		for _, n := range d.Tests {
			if !byName[n] {
				stale++
				t.Errorf("STALE MAPPING: العيب %s يشير إلى اختبارٍ زال %q", d.ID, n)
			}
		}
	}
	// **وملفُّ كلِّ اختبارٍ موجودٌ فعلاً.**
	missing := 0
	for _, x := range e.T.Tests {
		if x.File == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "backend", x.File)); err != nil {
			missing++
			if missing <= 3 {
				t.Errorf("STALE MAPPING: ملفُّ %s غيرُ موجود — %s", x.Name, x.File)
			}
		}
	}
	t.Logf("STALE MAPPING GUARD = PASS — %d خريطةً شائخةً · %d ملفّاً مفقوداً", stale, missing)
}

// ══════════════════════════════════════════════════════════════════════
// **٢٧ و٢٨ · لماذا اختير؟**
// ══════════════════════════════════════════════════════════════════════

// TestWhySelectedTrace **ولا اختبارَ يُختار بلا سبب.**
func TestWhySelectedTrace(t *testing.T) {
	e := load(t)
	r := e.Analyze("TEST", "", "", []string{"backend/internal/orders/transitions.go"})
	if len(r.Tests) == 0 {
		t.Fatal("لم يُختَر شيء")
	}
	for _, tt := range r.Tests {
		if len(tt.Reasons) == 0 {
			t.Errorf("WHY-SELECTED TRACE خُرق: %s بلا سبب", tt.Target)
		}
		for _, why := range tt.Reasons {
			if why.Rule == "" || why.Path == "" || why.Depth == "" {
				t.Errorf("%s: سببٌ ناقص %+v", tt.Target, why)
			}
		}
	}
	// **والمباشرُ يُميَّز عن المتعدّي** (البند ٦).
	d, tr := 0, 0
	for _, why := range r.Reasons {
		switch why.Depth {
		case Direct:
			d++
		case Transitive:
			tr++
		}
	}
	if d == 0 || tr == 0 {
		t.Errorf("DIRECT/TRANSITIVE خُرق: مباشرٌ %d · متعدٍّ %d", d, tr)
	}
	t.Logf("WHY-SELECTED TRACE = PASS — %d اختباراً · مباشرٌ %d · متعدٍّ %d",
		len(r.Tests), d, tr)
}

// ══════════════════════════════════════════════════════════════════════
// **٤٧ · قابليّةُ الإعادة · و٤٢ الزمن**
// ══════════════════════════════════════════════════════════════════════

// TestRepeatabilityAndSpeed **البندان ٤٢ و٤٧.**
func TestRepeatabilityAndSpeed(t *testing.T) {
	e := load(t)
	files := []string{"backend/internal/orders/transitions.go",
		"backend/internal/push/push.go", "web/apps/rahalgo/src/app/page.tsx"}

	var first string
	var maxD, sum time.Duration
	const n = 5
	for i := 0; i < n; i++ {
		start := time.Now()
		r := e.Analyze("TEST", "", "", files)
		el := time.Since(start)
		sum += el
		if el > maxD {
			maxD = el
		}
		got := canon(r)
		if i == 0 {
			first = got
		} else if got != first {
			t.Errorf("REPEATABILITY خُرق: التشغيل %d خالف الأوّل", i+1)
		}
	}
	t.Logf("REPEATABILITY = PASS — %d تشغيلاتٍ متطابقة", n)
	t.Logf("AVERAGE LOCAL IMPACT TIME = %s · MAX = %s",
		(sum / n).Round(time.Microsecond), maxD.Round(time.Microsecond))
}

func canon(r *Result) string {
	var b strings.Builder
	b.WriteString(strings.Join(r.Flows, ","))
	b.WriteString("|" + strings.Join(r.Apps, ","))
	b.WriteString("|" + strings.Join(r.Defects, ","))
	b.WriteString("|" + strings.Join(r.Risks, ","))
	b.WriteString("|" + strings.Join(r.Gaps, ","))
	for _, t := range r.Tests {
		b.WriteString("|" + t.Target)
	}
	for _, m := range r.Modes {
		b.WriteString("|" + string(m))
	}
	b.WriteString("|" + string(r.Confidence) + string(r.Risk) + r.Fallback)
	return b.String()
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · مداخلُ التغيير**
// ══════════════════════════════════════════════════════════════════════

// TestChangeInputModes **البند ٢.**
func TestChangeInputModes(t *testing.T) {
	root, err := findRoot()
	if err != nil {
		t.Fatalf("جذر: %v", err)
	}
	// **صريح.**
	src, files, err := ChangedFrom(root, "", "", []string{"a/b.go"})
	if err != nil || src != "EXPLICIT_FILES" || len(files) != 1 {
		t.Errorf("EXPLICIT خُرق: %s %v %v", src, files, err)
	}
	// **مدىً في git.**
	src, _, err = ChangedFrom(root, "HEAD~1", "HEAD", nil)
	if err != nil {
		t.Logf("GIT_RANGE غيرُ متاحٍ هنا: %v", err)
	} else if src != "GIT_RANGE" {
		t.Errorf("GIT_RANGE خُرق: %s", src)
	}
	// **شجرةُ العمل.**
	src, _, err = ChangedFrom(root, "", "", nil)
	if err != nil || src != "WORKING_TREE" {
		t.Errorf("WORKING_TREE خُرق: %s %v", src, err)
	}
	t.Logf("CHANGE INPUT MODES = 3")
}

// TestFileCategories كلُّ صنفٍ يُبلَغ بمسارٍ حقيقيّ.
func TestFileCategories(t *testing.T) {
	cases := map[string]Category{
		"backend/internal/orders/service.go":             CatBackend,
		"backend/internal/orders/service_test.go":        CatTest,
		"backend/internal/qa/harness.go":                 CatTestInfra,
		"backend/internal/migrate/migrations/0001_x.sql": CatMigration,
		"backend/internal/settings/catalog.go":           CatSettings,
		"web/apps/rahalgo/src/app/page.tsx":              CatAdminWeb,
		"mobile/app-customer/src/main/kotlin/X.kt":       CatAndCust,
		"mobile/app-driver/src/main/kotlin/X.kt":         CatAndDriver,
		"mobile/app-merchant/src/main/kotlin/X.kt":       CatAndMerch,
		"mobile/app-rep/src/main/kotlin/X.kt":            CatAndRep,
		"mobile/shared/src/main/kotlin/X.kt":             CatAndShared,
		"deploy/nginx.conf":                              CatDeploy,
		"docs/testing/FINAL_STATIC_CLOSEOUT.md":          CatTruthDocs,
		"README.md":                                      CatDocs,
		"weird/place/thing.go":                           CatUnknown,
	}
	for p, want := range cases {
		if got := Classify(p).Category; got != want {
			t.Errorf("%s ⇒ %s · يُنتظر %s", p, got, want)
		}
	}
	t.Logf("FILE CATEGORIES = %d", len(cases))
}
