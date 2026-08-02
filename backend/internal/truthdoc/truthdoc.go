// Package truthdoc يُولّد **مصدرَ الحقيقة الواحد** من الشيفرة نفسِها.
//
// # المسألة
//
// كانت في المشروع أربعُ وثائقَ **كلٌّ منها يقول إنّه المرجع**: خطّةٌ وخارطةٌ
// وسياسةٌ ودورةٌ تشغيلية. **وقرارٌ يُكتب في واحدةٍ لا يُنقل إلى الثلاث**،
// فيُبنى بندٌ على قرارٍ نُسخ منه غيرُه.
//
// **وأخطرُ من التعدّد أن تشيخ الوثيقة**: تُغيَّر قاعدةٌ في الشيفرة ويبقى السطرُ
// يقول القديم. **وقد وقع** (٢٠٢٦-٠٨-٠٢): كُتب استثناءٌ ضيّقٌ للمالك، ونُفِّذ
// تجاوزاً شاملاً — **والوثيقةُ تقول إنّ الأزرار نُزعت وهي ظاهرة.**
//
// # والحلّ: نصفٌ يُولَّد ونصفٌ يُكتب
//
//	ما يصف **سلوكَ المحرّك**   ←  يُولَّد منه فلا يكذب
//	ما يصف **قرارَ المالك**    ←  يُكتب بلفظه ويُقتبس نصّاً
//
// **ولا يُخلط الاثنان في سطر.** وفواصلُ HTML المعلّقة تحرس الحدّ: ما بينها
// يُكتب آلياً، **وما خارجها لا يُمسّ.**
//
// # والحارسُ يمنع الشيخوخة
//
// `TestTruthDocIsCurrent` يُولّد ويقارن. **فمن غيّر قاعدةً ونسي الوثيقةَ
// يسقط بناؤه** — ولا يكتشف أحدٌ الفرقَ بعد شهر.
package truthdoc

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// Path موضعُ الوثيقة من جذر المستودع.
const Path = "../../../docs/TRUTH.md"

var blockRe = regexp.MustCompile(`(?s)<!-- gen:([a-z-]+) -->.*?<!-- /gen:([a-z-]+) -->`)

// Sections ما يُولَّد — المفتاحُ اسمُ الفاصل.
func Sections() map[string]string {
	return map[string]string{
		"platform-managed": orders.TruthTable(false),
		"self-managed":     orders.TruthTable(true),
		"fail-reasons":     orders.FailReasonsTable(),
		"terminal":         orders.TerminalStates() + "\n",
		"settings":         settingsTable(),
	}
}

// settingsTable مفاتيحُ الإعدادات المؤثّرة في الدورة — **من الفهرس لا من ذاكرة.**
//
// **ومفتاحٌ يُذكر في وثيقةٍ ولا وجودَ له** يُبحث عنه في اللوحة فلا يُوجد،
// **ومفتاحٌ موجودٌ لا يُذكر** يبقى على افتراضه أبداً ولا أحد يعلم أنّه يملك
// تغييرَه.
func settingsTable() string {
	groups := []settings.Group{
		settings.GroupOrders, settings.GroupDrivers,
		settings.GroupMerchants, settings.GroupPricing,
	}
	labels := map[settings.Group]string{
		settings.GroupOrders:    "الطلبات",
		settings.GroupDrivers:   "السائقون",
		settings.GroupMerchants: "المتاجر",
		settings.GroupPricing:   "التسعير",
	}
	var b strings.Builder
	b.WriteString("| المفتاح | المجموعة | النوع | الافتراضيّ |\n|---|---|---|---|\n")
	for _, g := range groups {
		for _, d := range settings.Catalog {
			if d.Group != g {
				continue
			}
			b.WriteString(fmt.Sprintf("| `%s` | %s | %s | `%v` |\n",
				d.Key, labels[g], d.Kind, d.Default))
		}
	}
	return b.String()
}

// Render يُدخل الأقسام المولَّدة في نصِّ الوثيقة ويعيد النتيجة.
//
// **ولا يكتب ما لا فاصلَ له**: قسمٌ مولَّدٌ بلا موضعٍ في الوثيقة **خطأٌ يُقال**
// لا شيءٌ يُلحق في آخرها.
func Render(doc string) (string, error) {
	secs := Sections()
	seen := map[string]bool{}
	var bad error
	out := blockRe.ReplaceAllStringFunc(doc, func(m string) string {
		g := blockRe.FindStringSubmatch(m)
		open, closeName := g[1], g[2]
		if open != closeName {
			bad = fmt.Errorf("فاصلٌ مفتوحٌ %q ومُغلَقٌ %q", open, closeName)
			return m
		}
		body, ok := secs[open]
		if !ok {
			bad = fmt.Errorf("فاصلٌ في الوثيقة بلا مُولِّد: %q", open)
			return m
		}
		seen[open] = true
		return fmt.Sprintf("<!-- gen:%s -->\n%s<!-- /gen:%s -->", open, body, open)
	})
	if bad != nil {
		return "", bad
	}
	for k := range secs {
		if !seen[k] {
			return "", fmt.Errorf("مُولِّدٌ بلا فاصلٍ في الوثيقة: %q", k)
		}
	}
	return out, nil
}

// Write يقرأ الوثيقة ويُعيد كتابتها بالأقسام المولَّدة.
func Write(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := Render(string(raw))
	if err != nil {
		return err
	}
	if out == string(raw) {
		return nil
	}
	return os.WriteFile(path, []byte(out), 0o644)
}
