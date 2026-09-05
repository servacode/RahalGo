package impact

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Truth ما يُستهلَك من `TEST_TRUTH` — **ولا يُنسَخ** (البند ١).
type Truth struct {
	Tests []struct {
		Name    string   `json:"name"`
		File    string   `json:"file"`
		Package string   `json:"package"`
		Purpose string   `json:"purpose"`
		Flows   []string `json:"flows"`
		Defects []string `json:"defects"`
		Risks   []string `json:"risks"`
		Gaps    []string `json:"gaps"`
		Modes   []string `json:"modes"`
		Orphan  bool     `json:"orphan"`
	} `json:"tests"`
	Flows []struct {
		ID      string   `json:"id"`
		Apps    []string `json:"apps"`
		Money   bool     `json:"money"`
		Defects []string `json:"defects"`
		Risks   []string `json:"risks"`
		Gaps    []string `json:"gaps"`
		Tests   []string `json:"tests"`
	} `json:"flows"`
	Settings []struct {
		Key   string   `json:"key"`
		Money bool     `json:"money"`
		Flows []string `json:"flows"`
		Apps  []string `json:"apps"`
		Tests []string `json:"tests"`
	} `json:"settings"`
	Defects []struct {
		ID    string   `json:"id"`
		Tests []string `json:"tests"`
	} `json:"defects"`
	Risks []struct {
		ID    string   `json:"id"`
		Tests []string `json:"tests"`
	} `json:"risks"`
	Gaps []struct {
		ID       string   `json:"id"`
		Severity string   `json:"severity"`
		Tests    []string `json:"tests"`
	} `json:"gaps"`
}

// Engine محرّكُ الأثر.
type Engine struct {
	T Truth
	// root جذرُ المستودع.
	root string
}

// Load يقرأ الحقيقةَ المولَّدة.
func Load(root string) (*Engine, error) {
	b, err := os.ReadFile(filepath.Join(root, "docs/testing/system/TEST_TRUTH.json"))
	if err != nil {
		return nil, fmt.Errorf("حقيقةُ الاختبار غيرُ مقروءة: %w", err)
	}
	var t Truth
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("حقيقةُ الاختبار غيرُ مفهومة: %w", err)
	}
	if len(t.Tests) == 0 || len(t.Flows) == 0 {
		return nil, fmt.Errorf("حقيقةُ الاختبار فارغة — يُعاد التوليد")
	}
	return &Engine{T: t, root: root}, nil
}

// Analyze يحلّل مجموعةَ تغييرٍ ويعيد الأثر.
func (e *Engine) Analyze(source, base, head string, paths []string) *Result {
	r := &Result{Source: source, Base: base, Head: head, Confidence: High, Risk: RiskLow}

	for _, p := range paths {
		f := Classify(p)
		r.Files = append(r.Files, f)
		e.applyCategory(r, f)
		e.applyDomainRules(r, f)
	}

	// **ولا نتيجةَ فارغةٌ بلا سبب** (البند ١٠).
	if len(r.Files) == 0 {
		r.Fallback = "لا ملفَّ متغيّراً — لا أثر"
		r.Fallback = ""
		r.Confidence = High
	}

	e.expandFlows(r)
	e.selectTests(r)
	e.finalize(r)
	sortAll(r)
	return r
}

// applyCategory أثرُ الصنف نفسِه.
func (e *Engine) applyCategory(r *Result, f File) {
	add := func(rule, why string, d Depth) {
		r.Reasons = append(r.Reasons, Reason{From: f.Path, Rule: rule, Path: why, Depth: d})
	}
	switch f.Category {
	case CatUnknown:
		addUniq(&r.Unknowns, f.Path)
		r.Fallback = "UNKNOWN IMPACT → SAFE FULL FALLBACK"
		r.Confidence = Low
		modeSet(&r.Modes, ModeFull)
		add("UNKNOWN", "**لا قاعدةَ تصنّفه** ⇒ توسيعٌ آمنٌ إلى الكلّ", Direct)

	case CatMigration:
		modeSet(&r.Modes, ModeFull, ModeFinancial)
		addUniq(&r.Flows, "F-01", "F-14")
		add("MIGRATION", "هجرةٌ ⇒ تكاملُ القاعدة وثوابتُ المال (البند ١١)", Direct)
		e.addPkg(r, "internal/qa", true, Reason{From: f.Path, Rule: "MIGRATION",
			Path: "الهجراتُ تُطبَّق في مِسنَد `qa`", Depth: Direct})
		e.addPkg(r, "internal/fininv", true, Reason{From: f.Path, Rule: "MIGRATION",
			Path: "قيدُ أنواع الدفتر يُقرأ من القاعدة الحيّة", Depth: Transitive})

	case CatSettings:
		modeSet(&r.Modes, ModeFull)
		add("SETTINGS", "معجمُ الإعدادات ⇒ كلُّ ما يقرأ إعداداً (البند ١٢)", Direct)
		e.addPkg(r, "internal/testtruth", true, Reason{From: f.Path, Rule: "SETTINGS",
			Path: "عدُّ الإعدادات السلوكيّة يُستخرَج من المعجم", Depth: Direct})
		e.addPkg(r, "internal/qa", true, Reason{From: f.Path, Rule: "SETTINGS",
			Path: "الإعداداتُ تقود التسويةَ والحرّاس", Depth: Transitive})

	case CatTruthDocs:
		modeSet(&r.Modes, ModeFast)
		add("TRUTH_DOCS", "**حقيقةُ منتجٍ لا وثيقةٌ عاديّة** ⇒ إعادةُ توليدٍ وحرّاسُ انحراف (البند ٢٥)", Direct)
		e.addPkg(r, "internal/testtruth", true, Reason{From: f.Path, Rule: "TRUTH_DOCS",
			Path: "السجلّاتُ المجمَّدةُ تُستخرَج من هذه الوثائق", Depth: Direct})

	case CatTestInfra:
		modeSet(&r.Modes, ModeFast, ModeFull)
		add("TEST_INFRA", "**بنيةٌ تحتيّةٌ للاختبار ⇒ تحقّقٌ أوسعُ من اختبارِها وحدَه** (البند ٢٤)", Direct)
		for _, p := range []string{"internal/testtruth", "internal/fininv", "internal/racemap",
			"internal/failmap", "internal/androidmap", "internal/impact", "internal/qa"} {
			e.addPkg(r, p, true, Reason{From: f.Path, Rule: "TEST_INFRA",
				Path: "**المِسنَدُ لا يختبر نفسَه بما تبدّل وحدَه**", Depth: Transitive})
		}

	case CatTest:
		modeSet(&r.Modes, ModeImpacted)
		add("TEST_ONLY", "اختبارٌ تبدّل — **ولا يُفترَض تبدّلُ سلوكِ منتج** (البند ٢٣)", Direct)
		if pkg := goPkgOf(f.Path); pkg != "" {
			e.addPkg(r, pkg, true, Reason{From: f.Path, Rule: "TEST_ONLY",
				Path: "حزمةُ الاختبار المتبدّل", Depth: Direct})
		}

	case CatDocs:
		add("DOCS_ONLY", "توثيقٌ عاديّ — **لا أثرَ على السلوك**", Direct)

	case CatAdminWeb:
		addUniq(&r.Apps, "admin")

	case CatAndCust:
		addUniq(&r.Apps, "customer")
	case CatAndDriver:
		addUniq(&r.Apps, "driver")
	case CatAndMerch:
		addUniq(&r.Apps, "merchant")
	case CatAndRep:
		addUniq(&r.Apps, "rep")
	case CatAndShared:
		addUniq(&r.Apps, "customer", "driver", "merchant", "rep")
	}

	// **وكلُّ تبديلٍ في أندرويد يُذكِّر بدَين `P-8`** (البند ٤٤).
	switch f.Category {
	case CatAndCust, CatAndDriver, CatAndMerch, CatAndRep, CatAndShared:
		modeSet(&r.Modes, ModeAndroid)
		e.addPkg(r, "internal/androidmap", true, Reason{From: f.Path, Rule: "ANDROID",
			Path: "مصفوفةُ أندرويد وحرّاسُها", Depth: Direct})
		addUniq(&r.DeviceRequired,
			"**`REAL DEVICE ACCEPTANCE = NOT STARTED`** — و٦٢١ اختباراً محلّيّاً لا تُغني عنه")
	}
}

// applyDomainRules قواعدُ النطاق المقيسة.
func (e *Engine) applyDomainRules(r *Result, f File) {
	for _, rule := range domainRules {
		hit := false
		for _, m := range rule.Match {
			if strings.HasPrefix(f.Path, m) || strings.Contains(f.Path, m) {
				hit = true
				break
			}
		}
		if !hit {
			continue
		}
		reason := Reason{From: f.Path, Rule: rule.Name, Path: rule.Why, Depth: Direct}
		r.Reasons = append(r.Reasons, reason)
		addUniq(&r.Flows, rule.Flows...)
		addUniq(&r.Apps, rule.Apps...)
		modeSet(&r.Modes, rule.Modes...)
		for _, p := range rule.Packages {
			e.addPkg(r, p, true, reason)
		}
		if rule.Device != "" {
			addUniq(&r.DeviceRequired, rule.Device)
		}
		if rule.Staging != "" {
			addUniq(&r.StagingRequired, rule.Staging)
		}
	}
}

// expandFlows يوسّع من التدفّقات إلى ما تحمله (البند ٦ · متعدٍّ).
func (e *Engine) expandFlows(r *Result) {
	want := map[string]bool{}
	for _, f := range r.Flows {
		want[f] = true
	}
	for _, fl := range e.T.Flows {
		if !want[fl.ID] {
			continue
		}
		addUniq(&r.Apps, fl.Apps...)
		addUniq(&r.Defects, fl.Defects...)
		addUniq(&r.Risks, fl.Risks...)
		addUniq(&r.Gaps, fl.Gaps...)
		if fl.Money {
			modeSet(&r.Modes, ModeFinancial)
		}
		r.Reasons = append(r.Reasons, Reason{
			From: fl.ID, Rule: "FLOW_EXPANSION",
			Path:  fmt.Sprintf("%s ⇒ تطبيقاتُه %v وسجلّاتُه", fl.ID, fl.Apps),
			Depth: Transitive,
		})
	}
	// **والإعداداتُ التي تقود تدفّقاً متأثّراً تصير متأثّرة.**
	for _, s := range e.T.Settings {
		for _, sf := range s.Flows {
			if want[sf] {
				addUniq(&r.Settings, s.Key)
				if s.Money {
					modeSet(&r.Modes, ModeFinancial)
				}
				break
			}
		}
	}
}

// selectTests يختار الاختباراتِ من الحقيقة — **ولا اسمَ يُكتب بيد** (البند ٢٨).
func (e *Engine) selectTests(r *Result) {
	want := map[string]string{} // اسمُ الاختبار ⇒ السبب

	mark := func(names []string, why string) {
		for _, n := range names {
			if _, ok := want[n]; !ok {
				want[n] = why
			}
		}
	}
	for _, id := range r.Defects {
		for _, d := range e.T.Defects {
			if d.ID == id {
				mark(d.Tests, "حارسُ العيب "+id+" (البند ٢٠)")
			}
		}
	}
	for _, id := range r.Risks {
		for _, x := range e.T.Risks {
			if x.ID == id {
				mark(x.Tests, "تحقّقُ الخطر "+id+" (البند ٢١)")
			}
		}
	}
	for _, id := range r.Gaps {
		for _, g := range e.T.Gaps {
			if g.ID == id {
				mark(g.Tests, "فجوةُ عقدٍ "+id+" ("+g.Severity+") (البند ٢٢)")
				if g.Severity == "BLOCKER" || g.Severity == "CRITICAL" {
					r.Risk = worse(r.Risk, RiskHigh)
				}
			}
		}
	}
	for _, id := range r.Flows {
		for _, fl := range e.T.Flows {
			if fl.ID == id {
				mark(fl.Tests, "تدفّقٌ متأثّر "+id)
			}
		}
	}

	// **والاسمُ يُترجَم إلى حزمةٍ من الحقيقة نفسِها** — فلا مسارَ يُخمَّن.
	byName := map[string]string{}
	for _, t := range e.T.Tests {
		byName[t.Name] = t.Package
		if t.File != "" {
			byName[t.Name] = goPkgOf("backend/" + t.File)
		}
	}
	for name, why := range want {
		pkg := byName[name]
		if pkg == "" {
			addUniq(&r.Warnings, "**اختبارٌ في الخريطة لا يُعرَف ملفُّه**: "+name)
			continue
		}
		e.addTest(r, name, "test", true, Reason{From: name, Rule: "TRUTH_MAPPING",
			Path: why, Depth: Transitive})
		e.addPkg(r, pkg, true, Reason{From: name, Rule: "TRUTH_MAPPING",
			Path: "حزمةُ " + name, Depth: Transitive})
	}
}

// finalize يحسم الثقةَ والخطرَ والوضع.
func (e *Engine) finalize(r *Result) {
	hasFin, hasSec, hasAnd := false, false, false
	for _, m := range r.Modes {
		switch m {
		case ModeFinancial:
			hasFin = true
		case ModeSecurity:
			hasSec = true
		case ModeAndroid:
			hasAnd = true
		}
	}
	switch {
	case r.Fallback != "":
		r.Risk = worse(r.Risk, RiskHigh)
	case hasFin || hasSec:
		r.Risk = worse(r.Risk, RiskCritical)
	case hasAnd || len(r.Flows) > 0:
		r.Risk = worse(r.Risk, RiskMedium)
	}

	// **وثقةٌ منخفضةٌ توسّع ولا تختصر** (البند ٢٩).
	if r.Confidence == Low && r.Fallback == "" {
		r.Fallback = "LOW CONFIDENCE → SAFE FULL FALLBACK"
		modeSet(&r.Modes, ModeFull)
	}
	// **ووثائقُ محضةٌ لا تُفجّر الأوضاع** (البند ٣٣).
	if len(r.Modes) == 0 && len(r.Tests) == 0 {
		onlyDocs := len(r.Files) > 0
		for _, f := range r.Files {
			if f.Category != CatDocs {
				onlyDocs = false
			}
		}
		if onlyDocs {
			r.Confidence = High
			r.Risk = RiskLow
			r.Reasons = append(r.Reasons, Reason{Rule: "NEGATIVE_CONTROL",
				Path: "**وثائقُ محضةٌ ⇒ لا اختبار** — والمحافظةُ لا تعني الانفجار", Depth: Direct})
		}
	}
	if len(r.Modes) == 0 {
		modeSet(&r.Modes, ModeFast)
	}
}

func (e *Engine) addPkg(r *Result, pkg string, required bool, why Reason) {
	for i := range r.Tests {
		if r.Tests[i].Target == pkg && r.Tests[i].Kind == "package" {
			r.Tests[i].Reasons = append(r.Tests[i].Reasons, why)
			r.Tests[i].Required = r.Tests[i].Required || required
			return
		}
	}
	r.Tests = append(r.Tests, TestSel{Target: pkg, Kind: "package",
		Required: required, Reasons: []Reason{why}})
}

func (e *Engine) addTest(r *Result, name, kind string, required bool, why Reason) {
	for i := range r.Tests {
		if r.Tests[i].Target == name && r.Tests[i].Kind == kind {
			r.Tests[i].Reasons = append(r.Tests[i].Reasons, why)
			return
		}
	}
	r.Tests = append(r.Tests, TestSel{Target: name, Kind: kind,
		Required: required, Reasons: []Reason{why}})
}

// goPkgOf مسارُ حزمةِ Go من مسار ملفّ.
func goPkgOf(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if !strings.HasPrefix(p, "backend/") {
		return ""
	}
	d := filepath.ToSlash(filepath.Dir(strings.TrimPrefix(p, "backend/")))
	if d == "." {
		return ""
	}
	return d
}

func worse(a, b RiskClass) RiskClass {
	rank := map[RiskClass]int{RiskLow: 0, RiskMedium: 1, RiskHigh: 2, RiskCritical: 3}
	if rank[b] > rank[a] {
		return b
	}
	return a
}
