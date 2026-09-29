// حزمةُ machinedoc — **عقدُ آلةِ الحالات ملفّاً يُقرأ بالآلة، ووثيقةً تُقرأ
// بالعين — ومصدرُهما واحد.**
//
// # ولماذا ملفّان لا ملفّ
//
// **السؤالان مختلفان.** مصفوفةُ القبول تسأل: «هل يملك السائقُ
// `at_dropoff → delivered` في وضع المنصّة؟» — **سؤالٌ يُجاب بـ`jq` لا
// بقراءة.** والمالكُ يسأل: «ما الذي يقع حين يرفض المتجرُ الطلبَ؟» —
// **سؤالٌ لا يُجاب بـJSON.**
//
// **فمن جعلهما ملفّاً واحداً خسر أحدَهما.**
//
// # وحارسٌ واحدٌ يحكم الاثنين
//
// `TestMachineDocIsCurrent` **يُعيد التوليدَ ويقارن** — فإن بدّل أحدٌ حدّاً
// في `statuses.go` ولم يُعِد التوليدَ **سقط البناء.** وهو عينُ ما يحرس
// `TRUTH.md` (`truthdoc_test.go:30`).
//
// **ولولا الحارسُ لكان الملفُّ وثيقةً مكتوبةً بيدٍ تلبس ثوبَ المولَّد** —
// وذلك حالُ `docs/testing/ORDER_TRANSITIONS_55.md` اليوم: **يزعم أنّه
// مولَّدٌ، ولا مولِّدَ له، ويحرسه عدُّ صفوفٍ لا محتوى.**
package machinedoc

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// JSONPath و DocPath موضعاهما من مجلّد الحزمة (`internal/machinedoc`).
const (
	JSONPath = "../../../docs/testing/system/ORDER_STATE_MACHINE.json"
	DocPath  = "../../../docs/ORDER-STATE-MACHINE.md"
)

var blockRe = regexp.MustCompile(`(?s)<!-- gen:([a-z-]+) -->.*?<!-- /gen:([a-z-]+) -->`)

// roleLabels تسمياتُ الأدوار — **بلفظها في الشاشة.**
var roleLabels = map[string]string{
	"customer": "الزبون",
	"merchant": "المتجر",
	"ops":      "العمليات",
	"admin":    "المالك",
	"driver":   "السائق",
}

// JSON نصُّ الملفّ المقروء بالآلة — **مرتَّبٌ ثابتاً** فلا يتغيّر بلا سبب.
func JSON() (string, error) {
	b, err := json.MarshalIndent(orders.BuildMachine(), "", " ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}

func labelRoles(rs []string) string {
	if len(rs) == 0 {
		return "**لا أحد**"
	}
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		if l, ok := roleLabels[r]; ok {
			out = append(out, l)
		} else {
			out = append(out, r)
		}
	}
	return strings.Join(out, " · ")
}

// edgeTable جدولُ حدودِ نوعٍ بعينه — **والوضعان في عمودَين متجاورَين.**
//
// **فالفرقُ بين الوضعَين هو ما يُقرأ**، ومن وضع كلَّ وضعٍ في جدولٍ منفصلٍ
// جعل القارئَ يُقلّب بين جدولين ليرى سطراً واحداً.
func edgeTable(kind string) string {
	m := orders.BuildMachine()
	var b strings.Builder
	b.WriteString("| من | إلى | الأدوارُ المُصرَّحة | في وضع المنصّة | في وضع المتاجر |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, e := range m.Edges {
		if e.Kind != kind {
			continue
		}
		decl := "—"
		if len(e.DeclaredRoles) > 0 {
			decl = "`" + strings.Join(e.DeclaredRoles, "`, `") + "`"
		}
		from := e.From
		if e.Inherited {
			from = e.From + " ↩"
		}
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %s | %s |\n",
			from, e.To, decl,
			labelRoles(e.EffectivePlatform), labelRoles(e.EffectiveMerchants)))
	}
	return b.String()
}

// nobodyTable الحدودُ التي لا يملكها أحدٌ في وضع المنصّة.
//
// **وهي أهمُّ جدولٍ هنا** — لأنّها القيودُ التي تُقرأ سهواً: **الشيفرةُ
// تسمح بالحدّ، والوضعُ يمنع كلَّ من يملكه**، فيبقى بابٌ مرسومٌ لا يفتحه أحد.
func nobodyTable() string {
	m := orders.BuildMachine()
	var b strings.Builder
	b.WriteString("| النوع | من | إلى | الأدوارُ المُصرَّحة | ومن أسقطها |\n")
	b.WriteString("|---|---|---|---|---|\n")
	driverOnly := map[string]bool{
		"at_pickup": true, "picked_up": true, "on_the_way": true,
		"at_dropoff": true, "delivered": true, "failed": true,
	}
	for _, e := range m.Edges {
		if !e.NobodyInPlatform {
			continue
		}
		why := "—"
		switch {
		case e.To == "preparing":
			why = "`P2` (وقبلَها `P1` للمتجر)"
		case driverOnly[e.To]:
			why = "`P3`"
		default:
			why = "`P1`"
		}
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | %s |\n",
			e.Kind, e.From, e.To, strings.Join(e.DeclaredRoles, "`, `"), why))
	}
	return b.String()
}

// declaredTable جدولُ ما صُرِّح — **بمصدره سطراً سطراً.**
func declaredTable(rows []orders.MachineDeclared) string {
	var b strings.Builder
	b.WriteString("| المعرّف | ما هو | المصدر |\n|---|---|---|\n")
	for _, d := range rows {
		b.WriteString(fmt.Sprintf("| `%s` | %s | `%s` |\n", d.ID, d.What, d.Source))
	}
	return b.String()
}

func statusTable() string {
	m := orders.BuildMachine()
	var b strings.Builder
	b.WriteString("| # | الرمز | اللفظ | نهائيّة؟ | تُعيد المال؟ |\n|---|---|---|---|---|\n")
	for _, s := range m.Statuses {
		yes := func(v bool) string {
			if v {
				return "**نعم**"
			}
			return "لا"
		}
		b.WriteString(fmt.Sprintf("| %d | `%s` | %s | %s | %s |\n",
			s.FlowIndex, s.Code, s.Label, yes(s.Terminal), yes(s.RefundOnEnter)))
	}
	return b.String()
}

// countsLine أرقامُ الآلة سطراً واحداً — **فمن قرأ العددَ عرف أنّه قرأ الكلّ.**
func countsLine() string {
	m := orders.BuildMachine()
	std, cust, nobody := 0, 0, 0
	for _, e := range m.Edges {
		switch e.Kind {
		case "standard":
			std++
		case "custom":
			cust++
		}
		if e.NobodyInPlatform {
			nobody++
		}
	}
	return fmt.Sprintf(
		"**%d حالةً · %d نهائيّةً · %d حدّاً قياسيّاً · %d حدّاً مخصَّصاً · "+
			"%d حدّاً لا يملكه أحدٌ في وضع المنصّة · %d سببَ تعذّر.**\n",
		len(m.Statuses), len(m.Terminal), std, cust, nobody, len(m.FailReasons))
}

// Sections ما يُولَّد في الوثيقة.
func Sections() map[string]string {
	m := orders.BuildMachine()
	return map[string]string{
		"counts":          countsLine(),
		"statuses":        statusTable(),
		"edges-standard":  edgeTable("standard"),
		"edges-custom":    edgeTable("custom"),
		"nobody-platform": nobodyTable(),
		"mode-rules":      declaredTable(m.ModeRules),
		"guards":          declaredTable(m.Guards),
		"auto-rules":      declaredTable(m.AutoRules),
		"implicit-edges":  declaredTable(m.ImplicitEdges),
		"fail-reasons":    failReasonsTable(),
	}
}

func failReasonsTable() string {
	m := orders.BuildMachine()
	var b strings.Builder
	b.WriteString("| السبب | الذنبُ على | يُعرض عند |\n|---|---|---|\n")
	for _, f := range m.FailReasons {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` |\n", f.Code, f.Fault, f.At))
	}
	return b.String()
}

// Render يُدخل الأقسام المولَّدة — **ويُخطئ إن بقي مُولِّدٌ بلا فاصل.**
func Render(doc string) (string, error) {
	secs := Sections()
	seen := map[string]bool{}
	var bad error
	out := blockRe.ReplaceAllStringFunc(doc, func(mm string) string {
		g := blockRe.FindStringSubmatch(mm)
		open, closeName := g[1], g[2]
		if open != closeName {
			bad = fmt.Errorf("فاصلٌ مفتوحٌ %q ومُغلَقٌ %q", open, closeName)
			return mm
		}
		body, ok := secs[open]
		if !ok {
			bad = fmt.Errorf("فاصلٌ في الوثيقة بلا مُولِّد: %q", open)
			return mm
		}
		seen[open] = true
		return fmt.Sprintf("<!-- gen:%s -->\n%s<!-- /gen:%s -->", open, body, open)
	})
	if bad != nil {
		return "", bad
	}
	missing := []string{}
	for k := range secs {
		if !seen[k] {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return "", fmt.Errorf("مُولِّدٌ بلا فاصلٍ في الوثيقة: %s",
			strings.Join(missing, ", "))
	}
	return out, nil
}

// Write يكتب الملفَّين — **الوثيقةَ والعقدَ المقروءَ بالآلة.**
func Write(docPath, jsonPath string) error {
	raw, err := os.ReadFile(docPath)
	if err != nil {
		return err
	}
	out, err := Render(string(raw))
	if err != nil {
		return err
	}
	if out != string(raw) {
		if err := os.WriteFile(docPath, []byte(out), 0o644); err != nil {
			return err
		}
	}
	js, err := JSON()
	if err != nil {
		return err
	}
	cur, _ := os.ReadFile(jsonPath)
	if string(cur) == js {
		return nil
	}
	return os.WriteFile(jsonPath, []byte(js), 0o644)
}
