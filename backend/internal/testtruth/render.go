// المخرَجان — **واحدٌ للآلة وواحدٌ للإنسان.**
//
// **و`P-9` يستهلك JSON لا Markdown** (البند ١٥ من طلب المالك) — **فلا
// يُعاد تحليلُ لغةٍ طبيعيّةٍ في كلّ تشغيل.**
package testtruth

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// JSON الحقيقةُ للآلة — **مرتّبةٌ حتميّاً فلا يتبدّل الملفُّ بلا سبب.**
func (t *Truth) JSON() ([]byte, error) {
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// Report التقريرُ البشريّ.
func (t *Truth) Report() string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }

	w("# حقيقةُ الاختبار — التقرير\n\n")
	w("> **مولَّدٌ آليّاً** — `go run ./cmd/testtruth`.\n")
	w("> **ولا يُحرَّر بيد**، وحارسُ `TestTruthIsCurrent` يُسقط البناءَ إن شاخ.\n")
	w("> **والمصدرُ للآلة** [`TEST_TRUTH.json`](TEST_TRUTH.json).\n\n---\n\n")

	// ── المستخرَج ────────────────────────────────────────────────
	w("# ١ · ما استُخرج من الشيفرة\n\n")
	w("**ولا رقمَ منه مكتوبٌ بيد.**\n\n")
	w("| ما هو | العدد |\n|---|---|\n")
	d := t.Derived
	w("| أبوابٌ في الموجّه | **%d** |\n", d.Routes)
	w("| انتقالاتُ الطلب | **%d** |\n", d.Transitions)
	w("| أنواعُ قيدِ المحفظة | **%d** — `%s` |\n", len(d.LedgerKinds), strings.Join(d.LedgerKinds, "` · `"))
	w("| مواضعُ الإشعار | **%d** — منها **%d** موجَّهٌ بـ`Apps` |\n", d.NotifySites, d.NotifyTargeted)
	w("| غرفُ البثّ | **%d** — `%s` |\n", len(d.PublishRooms), strings.Join(d.PublishRooms, "` · `"))
	w("| تعريفاتُ الإعدادات | **%d** — منها **%d** مُغيِّرٌ للسلوك |\n", d.SettingDefs, d.BehaviourSettings)
	w("| حقولُ الطلب | **%d** — من عقد `P-1` |\n", d.OrderFields)
	w("| ملفّاتُ اختبار | **%d** |\n", d.TestFiles)
	w("| دوالُّ اختبار | **%d** |\n\n", d.TestFuncs)

	// ── الاختبارات ───────────────────────────────────────────────
	mapped, orphan, infra := 0, 0, 0
	for _, x := range t.Tests {
		switch {
		case x.Orphan:
			orphan++
		case x.Purpose != PurposeFeature && len(x.Flows) == 0 && len(x.Defects) == 0:
			infra++
		default:
			mapped++
		}
	}
	w("---\n\n# ٢ · الاختبارات\n\n")
	w("```\nTOTAL      = %d\nMAPPED     = %d\nINFRA      = %d\nORPHAN     = %d\n```\n\n",
		len(t.Tests), mapped, infra, orphan)
	if orphan > 0 {
		w("**واليتيمُ اختبارٌ لا يعرف ماذا يحرس** — **ولا يُسقط البناءَ اليومَ**،\n")
		w("**ويُربَط مرحلةً بعد مرحلة.** وأكثرُها في:\n\n")
		byPkg := map[string]int{}
		for _, x := range t.Tests {
			if x.Orphan {
				byPkg[x.Package]++
			}
		}
		keys := make([]string, 0, len(byPkg))
		for k := range byPkg {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return byPkg[keys[i]] > byPkg[keys[j]] })
		w("| الحزمة | يتيمٌ |\n|---|---|\n")
		for i, k := range keys {
			if i >= 10 {
				break
			}
			w("| `%s` | %d |\n", k, byPkg[k])
		}
		w("\n")
	}

	// ── التغطية ──────────────────────────────────────────────────
	w("---\n\n# ٣ · التغطية\n\n")
	w("| السجلّ | العدد | مربوطٌ | بلا اختبار |\n|---|---|---|---|\n")
	w("| **التدفّقات** | %d | %d | %d |\n", len(t.Flows), countCovered(t), len(t.Flows)-countCovered(t))
	w("| **العيوب** | %d | %d | %d |\n", len(t.Defects), countDefectTested(t), len(t.Defects)-countDefectTested(t))
	w("| **المخاطر** | %d | %d | %d |\n", len(t.Risks), countRiskTested(t), len(t.Risks)-countRiskTested(t))
	w("| **فجواتُ العقد** | %d | %d | %d |\n", len(t.Gaps), countGapTested(t), len(t.Gaps)-countGapTested(t))
	w("| **إعداداتُ السلوك** | %d | %d | %d |\n\n", len(t.Settings), countSettingTested(t), len(t.Settings)-countSettingTested(t))

	// ── العيوبُ بالتفصيل ─────────────────────────────────────────
	w("---\n\n# ٤ · العيوبُ وحارسُها\n\n")
	w("| ID | العنوان | الحال | الاختبارات |\n|---|---|---|---|\n")
	for _, x := range t.Defects {
		w("| **%s** | %s | `%s` | %s |\n", x.ID, trim(x.Title, 46), x.Status, list(x.Tests))
	}

	// ── الفجوات ──────────────────────────────────────────────────
	w("\n---\n\n# ٥ · فجواتُ العقد\n\n")
	w("| ID | الشدّة | يوقظه | الحال | الاختبارات |\n|---|---|---|---|---|\n")
	for _, x := range t.Gaps {
		woke := "—"
		if x.WokenBy != "" {
			woke = "`" + x.WokenBy + "`"
		}
		w("| **%s** | `%s` | %s | `%s` | %s |\n", x.ID, x.Severity, woke, x.Status, list(x.Tests))
	}

	// ── الشيخوخةُ والانحراف ──────────────────────────────────────
	w("\n---\n\n# ٦ · الشيخوخةُ والفجوات\n\n")
	w("```\nSTALE REFERENCES = %d\nCOVERAGE GAPS    = %d\n```\n\n", len(t.Stale), len(t.CoverageGaps))
	if len(t.Stale) > 0 {
		w("## مراجعُ شائخة — **وهي سقوطٌ لا تقرير**\n\n")
		for _, s := range t.Stale {
			w("- %s\n", s)
		}
		w("\n")
	}
	if len(t.CoverageGaps) > 0 {
		w("## فجواتُ تغطية — **ما يحتاج اختباراً ولا اختبارَ له**\n\n")
		max := 40
		for i, s := range t.CoverageGaps {
			if i >= max {
				w("- … و%d أخرى\n", len(t.CoverageGaps)-max)
				break
			}
			w("- %s\n", s)
		}
		w("\n")
	}
	return b.String()
}

func countCovered(t *Truth) int {
	n := 0
	for _, f := range t.Flows {
		if len(f.Tests) > 0 {
			n++
		}
	}
	return n
}
func countDefectTested(t *Truth) int {
	n := 0
	for _, x := range t.Defects {
		if len(x.Tests) > 0 {
			n++
		}
	}
	return n
}
func countRiskTested(t *Truth) int {
	n := 0
	for _, x := range t.Risks {
		if len(x.Tests) > 0 {
			n++
		}
	}
	return n
}
func countGapTested(t *Truth) int {
	n := 0
	for _, x := range t.Gaps {
		if len(x.Tests) > 0 {
			n++
		}
	}
	return n
}
func countSettingTested(t *Truth) int {
	n := 0
	for _, x := range t.Settings {
		if len(x.Tests) > 0 {
			n++
		}
	}
	return n
}

func list(xs []string) string {
	if len(xs) == 0 {
		return "—"
	}
	return "`" + strings.Join(xs, "` · `") + "`"
}

func trim(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
