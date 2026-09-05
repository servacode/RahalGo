// البناءُ — **يجمع المستخرَجَ إلى المُعلَن ويردّ الحقيقة.**
package testtruth

import (
	"fmt"
	"sort"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/qa"
)

// Build يبني الحقيقةَ من الشجرة والوثائق.
//
// **ولا يقرأ لغةً طبيعيّةً بحرّيّة** — **يقرأ جداولَ بصيغةٍ ثابتة**
// (السجلُّ المجمَّد) **وبِنًى في الشيفرة**، **ويُعلن ما لا يُقرأ.**
func Build(backendRoot, docsRoot string) (*Truth, error) {
	r := Root(backendRoot)
	t := &Truth{}

	// ── المستخرَج ─────────────────────────────────────────────────
	tests, fileCount, err := r.Tests()
	if err != nil {
		return nil, fmt.Errorf("جردُ الاختبارات: %w", err)
	}
	routes, err := r.Routes()
	if err != nil {
		return nil, fmt.Errorf("جردُ الأبواب: %w", err)
	}
	trans, err := r.Transitions(docsRoot)
	if err != nil {
		return nil, fmt.Errorf("جردُ الانتقالات: %w", err)
	}
	kinds, err := r.LedgerKinds()
	if err != nil {
		return nil, fmt.Errorf("جردُ أنواع القيد: %w", err)
	}
	nTotal, nTargeted, err := r.NotifySites()
	if err != nil {
		return nil, fmt.Errorf("جردُ الإشعارات: %w", err)
	}
	rooms, err := r.PublishRooms()
	if err != nil {
		return nil, fmt.Errorf("جردُ غرف البثّ: %w", err)
	}
	defs, err := r.SettingDefs()
	if err != nil {
		return nil, fmt.Errorf("جردُ الإعدادات: %w", err)
	}
	regDefects, regRisks, err := Register(docsRoot)
	if err != nil {
		return nil, fmt.Errorf("قراءةُ السجلّ المجمَّد: %w", err)
	}

	t.Derived = Derived{
		Routes: routes, Transitions: trans, LedgerKinds: kinds,
		NotifySites: nTotal, NotifyTargeted: nTargeted,
		PublishRooms: rooms,
		SettingDefs:  len(defs), BehaviourSettings: countBehaviour(defs),
		// **حقولُ الطلب من `P-1`** — تُستهلَك ولا تُعاد.
		OrderFields: len(qa.OrderFields()),
		TestFiles:   fileCount, TestFuncs: len(tests),
	}

	// ── الاختباراتُ: مستخرَجةٌ + مُعلَنة ──────────────────────────
	byFlow := map[string][]string{}
	byDefect := map[string][]string{}
	byRisk := map[string][]string{}
	byGap := map[string][]string{}
	bySetting := map[string][]string{}

	for i := range tests {
		fn := &tests[i]
		d, declared := TestMap[fn.Name]
		if !declared {
			// **حزمةُ بنيةٍ تحتيّةٍ لا تُعدّ يتيمة** — لكنّها تُسمّى.
			if p, ok := InfraPackages[fn.Package]; ok {
				fn.Purpose = p
				continue
			}
			fn.Orphan = true
			continue
		}
		fn.Level, fn.Purpose = d.Level, d.Purpose
		fn.Flows, fn.Defects, fn.Risks = d.Flows, d.Defects, d.Risks
		fn.Gaps, fn.Settings, fn.Modes, fn.Evidence = d.Gaps, d.Settings, d.Modes, d.Evidence
		for _, x := range d.Flows {
			byFlow[x] = append(byFlow[x], fn.Name)
		}
		for _, x := range d.Defects {
			byDefect[x] = append(byDefect[x], fn.Name)
		}
		for _, x := range d.Risks {
			byRisk[x] = append(byRisk[x], fn.Name)
		}
		for _, x := range d.Gaps {
			byGap[x] = append(byGap[x], fn.Name)
		}
		for _, x := range d.Settings {
			bySetting[x] = append(bySetting[x], fn.Name)
		}
	}
	t.Tests = tests

	// ── التدفّقات ─────────────────────────────────────────────────
	for _, f := range Flows {
		fl := Flow{
			ID: f.ID, Title: f.Title, Actor: f.Actor, Apps: f.Apps,
			Money: f.Money, Realtime: f.Realtime, Severity: f.Severity,
			NeedLevels: deriveLevels(f),
			Defects:    f.Defects, Risks: f.Risks, Gaps: f.Gaps,
			Tests: byFlow[f.ID],
		}
		switch {
		case len(fl.Tests) == 0:
			fl.Status = StatusNotImplemented
			t.CoverageGaps = append(t.CoverageGaps,
				fmt.Sprintf("%s (%s) — لا اختبارَ مرتبطٌ به", f.ID, f.Title))
		case len(fl.Tests) < len(fl.NeedLevels):
			fl.Status = StatusPartial
		default:
			fl.Status = StatusCovered
		}
		t.Flows = append(t.Flows, fl)
	}

	// ── العيوبُ — من السجلّ لا من قائمةٍ هنا ──────────────────────
	for _, d := range regDefects {
		def := Defect{ID: d.ID, Title: d.Title, Tests: byDefect[d.ID]}
		switch {
		case len(def.Tests) == 0:
			def.Status = StatusNoTestYet
			t.CoverageGaps = append(t.CoverageGaps,
				fmt.Sprintf("%s — لا اختبارَ انحدارٍ بعد", d.ID))
		default:
			// **`P-1` تُنتج `EXPECTED_FAIL` لا `PASS`** — والحالُ يقولها.
			def.Status = StatusExpectedFail
		}
		t.Defects = append(t.Defects, def)
	}

	// ── المخاطر ──────────────────────────────────────────────────
	for _, x := range regRisks {
		rk := Risk{ID: x.ID, Title: x.Title, Strategy: RiskStrategy[x.ID], Tests: byRisk[x.ID]}
		if rk.Strategy == "" {
			t.Stale = append(t.Stale, fmt.Sprintf("%s — خطرٌ بلا استراتيجيّةِ تحقّق", x.ID))
		}
		for _, f := range Flows {
			for _, id := range f.Risks {
				if id == x.ID {
					rk.Flows = append(rk.Flows, f.ID)
				}
			}
		}
		if len(rk.Tests) == 0 {
			rk.Status = StatusNoTestYet
			t.CoverageGaps = append(t.CoverageGaps,
				fmt.Sprintf("%s — لا اختبارَ يحسمه بعد", x.ID))
		} else {
			rk.Status = StatusPresent
		}
		t.Risks = append(t.Risks, rk)
	}

	// ── الفجوات ──────────────────────────────────────────────────
	for _, g := range Gaps {
		gp := Gap{ID: g.ID, Title: g.Title, Severity: g.Severity, WokenBy: g.WokenBy, Tests: byGap[g.ID]}
		if len(gp.Tests) == 0 {
			gp.Status = StatusNotImplemented
		} else {
			gp.Status = StatusExpectedFail
		}
		t.Gaps = append(t.Gaps, gp)
	}

	// ── الإعدادات ────────────────────────────────────────────────
	flowsByKey := settingFlows()
	for _, d := range defs {
		if !behaviour(d) {
			continue
		}
		s := Setting{
			Key: d.Key, Group: d.Group, Kind: d.Kind, Sensitive: d.Sensitive,
			Money: moneyKind(d), Flows: flowsByKey[d.Key], Tests: bySetting[d.Key],
		}
		if len(s.Flows) == 0 {
			s.Flows = groupFlows(d.Group)
		}
		s.Apps = appsOf(s.Flows)
		if len(s.Tests) == 0 {
			s.Status = StatusNoTestYet
		} else {
			s.Status = StatusPresent
		}
		t.Settings = append(t.Settings, s)
	}

	// ── المراجعُ الشائخة ─────────────────────────────────────────
	live := map[string]bool{}
	for _, fn := range tests {
		live[fn.Name] = true
	}
	names := make([]string, 0, len(TestMap))
	for n := range TestMap {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if !live[n] {
			t.Stale = append(t.Stale, fmt.Sprintf("TestMap يشير إلى %q ولا وجودَ لها", n))
		}
	}
	knownDefect := map[string]bool{}
	for _, d := range regDefects {
		knownDefect[d.ID] = true
	}
	for _, f := range Flows {
		for _, id := range f.Defects {
			if !knownDefect[id] {
				t.Stale = append(t.Stale, fmt.Sprintf("%s يشير إلى عيبٍ %s لا وجودَ له في السجلّ", f.ID, id))
			}
		}
	}
	knownGap := map[string]bool{}
	for _, g := range Gaps {
		knownGap[g.ID] = true
	}
	for _, f := range Flows {
		for _, id := range f.Gaps {
			if !knownGap[id] {
				t.Stale = append(t.Stale, fmt.Sprintf("%s يشير إلى فجوةٍ %s لا وجودَ لها", f.ID, id))
			}
		}
	}
	knownSetting := map[string]bool{}
	for _, d := range defs {
		knownSetting[d.Key] = true
	}
	for _, g := range Gaps {
		if g.WokenBy != "" && !knownSetting[g.WokenBy] {
			t.Stale = append(t.Stale, fmt.Sprintf("%s يوقظه مفتاحٌ %q لا وجودَ له", g.ID, g.WokenBy))
		}
	}

	// **والقوائمُ الفارغةُ تخرج `[]` لا `null`** — **فمن يستهلكها لا
	// يحرس نفسَه من العدم.** (`P-9` و`P-10` يقرآن هذا الملفّ.)
	if t.Stale == nil {
		t.Stale = []string{}
	}
	if t.CoverageGaps == nil {
		t.CoverageGaps = []string{}
	}
	sort.Strings(t.Stale)
	sort.Strings(t.CoverageGaps)
	return t, nil
}

// ══════════════════════════════════════════════════════════════════════
// **قواعدُ الاشتقاق — تُطبَّق ولا تُكتب لكلّ تدفّق**
// ══════════════════════════════════════════════════════════════════════

func deriveLevels(f FlowDecl) []Level {
	set := map[Level]bool{L4: true} // كلُّ تدفّقٍ يمرّ ببابٍ
	if f.Money {
		set[L3] = true
	}
	if f.Realtime {
		set[L5] = true
	}
	if len(f.Apps) > 1 {
		set[L8] = true
	}
	if f.Partial {
		set[L9] = true
	}
	if f.Race {
		set[L9] = true
	}
	out := make([]Level, 0, len(set))
	for _, l := range []Level{L1, L2, L3, L4, L5, L8, L9} {
		if set[l] {
			out = append(out, l)
		}
	}
	return out
}

// behaviour **أيُغيّر هذا المفتاحُ سلوكاً؟**
//
// **والمعيارُ محتوىً لا مجموعة** — كما في
// `docs/testing/CONFIG_IMPACT_MAP.md`: **«عرضٌ وهويّةٌ ونصوصُ صفحات ٢٣ من
// ١١٨»**.
//
// **واستثناءُ مجموعتَي `site` و`app` كاملتين ردّ ٨٦ لا ٩٥** — **لأنّ
// فيهما مفاتيحَ تغيّر السلوك فعلاً** (بوّاباتُ النسخة والروابط)،
// **وفي `platform` نصوصَ صفحاتٍ لا تغيّره.**
//
// **فالحكمُ بالنوع والاسم**: **صورةٌ أو ملفٌّ أو نصٌّ طويلٌ أو نقطةٌ =
// عرض** · **و`page.*` نصوصُ صفحات.**
func behaviour(d SettingDef) bool {
	switch d.Kind {
	case "media", "file", "longtext", "geo":
		return false
	}
	if strings.HasPrefix(d.Key, "page.") {
		return false
	}
	// **وتعتيمُ خلفيّةٍ عرضٌ كخلفيّتِه.**
	//
	// **واستثناءُ الصورة دون نسبةِ تعتيمها تفريقٌ بلا معنى** — كلاهما
	// يبدّل ما يُرى ولا يبدّل مساراً. **وبهما يستقيم العدُّ ٩٥/١١٨
	// كما في `CONFIG_IMPACT_MAP.md`.**
	if strings.HasSuffix(d.Key, ".background_dim") {
		return false
	}
	// **هويّةٌ وروابطُ تواصلٍ — تُعرَض ولا تُغيّر مساراً.**
	for _, p := range []string{
		"platform.name", "platform.logo", "platform.address",
		"platform.facebook", "platform.instagram", "platform.telegram",
		"platform.whatsapp", "platform.support_phone",
	} {
		if d.Key == p {
			return false
		}
	}
	return true
}

func countBehaviour(defs []SettingDef) int {
	n := 0
	for _, d := range defs {
		if behaviour(d) {
			n++
		}
	}
	return n
}

func moneyKind(d SettingDef) bool {
	if d.Kind == "money" || d.Kind == "percent" {
		return true
	}
	return strings.Contains(d.Key, "commission") ||
		strings.Contains(d.Key, "margin") ||
		strings.Contains(d.Key, "payout") ||
		strings.Contains(d.Key, "reward") ||
		strings.Contains(d.Key, "fee")
}

// settingFlows **الروابطُ الصريحةُ لمفاتيحَ بعينها** — وما عداها بالمجموعة.
func settingFlows() map[string][]string {
	return map[string][]string{
		"sales.commission_percent":     {"F-14", "F-23", "F-15", "F-33"},
		"sales.activation_orders":      {"F-23", "F-33"},
		"merchants.commission_percent": {"F-14", "F-23", "F-33"},
		"pricing.margin_fixed":         {"F-01", "F-14", "F-23", "F-33"},
		"payouts.min_amount":           {"F-24", "F-33"},
		"orders.auto_accept_min":       {"F-05", "F-33"},
		"sales.require_whatsapp":       {"F-21"},
		"sales.target_reward":          {"F-21", "F-23"},
		"sales.reward_2":               {"F-21", "F-23"},
		"sales.reward_3":               {"F-21", "F-23"},
	}
}

func groupFlows(group string) []string {
	switch group {
	case "drivers":
		return []string{"F-07", "F-08", "F-13", "F-27"}
	case "merchants":
		return []string{"F-04", "F-05", "F-28"}
	case "sales":
		return []string{"F-21", "F-23"}
	case "customers":
		return []string{"F-01", "F-17"}
	}
	return []string{"F-33"}
}

func appsOf(flows []string) []string {
	seen := map[string]bool{}
	for _, id := range flows {
		for _, f := range Flows {
			if f.ID != id {
				continue
			}
			for _, a := range f.Apps {
				seen[a] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for a := range seen {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}
