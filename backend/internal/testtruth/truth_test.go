// حارسُ حقيقةِ الاختبار — **الانحرافُ سقوطٌ لا تقرير.**
//
// (البند ١٢ من طلب المالك: `TRUTH DRIFT = BUILD/QA FAILURE`.)
//
// **وهو النمطُ العاملُ في المشروع لـ`TRUTH.md`** — يُعاد التوليدُ ويُقارَن،
// **فما شاخ أسقط البناء.**
package testtruth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

// roots جذرا المشروع من موضع هذه الحزمة.
func roots(t *testing.T) (backend, docs string) {
	t.Helper()
	wd, err := os.Getwd() // internal/testtruth
	if err != nil {
		t.Fatal(err)
	}
	backend = filepath.Dir(filepath.Dir(wd))
	docs = filepath.Join(filepath.Dir(backend), "docs")
	if _, err := os.Stat(filepath.Join(backend, "go.mod")); err != nil {
		t.Fatalf("لم أجد جذرَ المحرّك عند %s", backend)
	}
	return backend, docs
}

func build(t *testing.T) *Truth {
	t.Helper()
	backend, docs := roots(t)
	tr, err := Build(backend, docs)
	if err != nil {
		t.Fatalf("تعذّر بناءُ الحقيقة: %v", err)
	}
	return tr
}

// ══════════════════════════════════════════════════════════════════════
// **١ · الانحراف — أهمُّ حارسٍ في المرحلة**
// ══════════════════════════════════════════════════════════════════════

func TestTruthIsCurrent(t *testing.T) {
	tr := build(t)
	_, docs := roots(t)
	out := filepath.Join(docs, "testing", "system")

	for _, c := range []struct {
		name string
		want []byte
	}{
		{"TEST_TRUTH.json", mustJSON(t, tr)},
		{"TEST_TRUTH.md", []byte(tr.Report())},
	} {
		got, err := os.ReadFile(filepath.Join(out, c.name))
		if err != nil {
			t.Fatalf("%s غيرُ موجود — شغّلْ `go run ./cmd/testtruth`", c.name)
		}
		if norm(string(got)) != norm(string(c.want)) {
			t.Errorf("%s شاخ — أعِد التوليدَ بـ`go run ./cmd/testtruth`", c.name)
		}
	}
}

func mustJSON(t *testing.T, tr *Truth) []byte {
	t.Helper()
	b, err := tr.JSON()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// norm **نهاياتُ الأسطر ليست انحرافاً** — ويندوز يبدّلها عند السحب.
func norm(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

// ══════════════════════════════════════════════════════════════════════
// **٢ · لا عيبَ يجهله النظام**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ٥: **ممنوعٌ عيبٌ مسجَّلٌ لا يعرف النظامُ بوجوده.**)

func TestEveryDefectIsKnown(t *testing.T) {
	tr := build(t)
	if len(tr.Defects) == 0 {
		t.Fatal("لم يُقرأ عيبٌ واحدٌ من السجلّ المجمَّد")
	}
	seen := map[string]bool{}
	for _, d := range tr.Defects {
		if seen[d.ID] {
			t.Errorf("عيبٌ مكرَّرٌ في السجلّ: %s", d.ID)
		}
		seen[d.ID] = true
		if d.Title == "" {
			t.Errorf("%s بلا عنوان — تبدّلت صيغةُ السجلّ", d.ID)
		}
	}
	// **والمتّصلُ يكشف ثغرةً** — عيبٌ حُذف أو رقمٌ قُفز.
	for i := 1; i <= len(tr.Defects); i++ {
		id := "D" + itoa(i)
		if !seen[id] {
			t.Errorf("ثغرةٌ في ترقيم العيوب: %s مفقود", id)
		}
	}
	t.Logf("DEFECTS MAPPED = %d/%d", len(tr.Defects), len(tr.Defects))
}

// ══════════════════════════════════════════════════════════════════════
// **٣ · لكلّ خطرٍ استراتيجيّةُ تحقّق**
// ══════════════════════════════════════════════════════════════════════

func TestEveryRiskHasStrategy(t *testing.T) {
	tr := build(t)
	if len(tr.Risks) == 0 {
		t.Fatal("لم يُقرأ خطرٌ واحدٌ من السجلّ")
	}
	for _, r := range tr.Risks {
		if r.Strategy == "" {
			t.Errorf("%s بلا استراتيجيّةِ تحقّق — أضِفها إلى RiskStrategy", r.ID)
		}
	}
	t.Logf("RISKS MAPPED = %d/%d", len(tr.Risks), len(tr.Risks))
}

// ══════════════════════════════════════════════════════════════════════
// **٤ · لا مرجعَ شائخ**
// ══════════════════════════════════════════════════════════════════════
//
// (البند ١٤: **ممنوعٌ مرجعٌ وهميّ.**)

func TestNoStaleReferences(t *testing.T) {
	tr := build(t)
	for _, s := range tr.Stale {
		t.Errorf("مرجعٌ شائخ: %s", s)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · التدفّقاتُ الخمسةُ والثلاثون كلُّها معلَنة**
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا صارت خمسةً وثلاثين
//
// **`F-35` — خريطةُ العمليات**، بُنيت بطلب المالك ٢٠٢٦-٠٩-٠٦.
//
// **و`F-36` — حذفُ الحساب بطلب صاحبه**، **سُجّل في دورةِ ٥٧**:
// **مسارٌ قائمٌ في المنتَج لم يُسجَّل قطّ** — **ولا اختبارَ يمرّ
// به** (`XG-49`). **والتسجيلُ يُظهر نقصَ الدليل، وتركُه يُخفيه.**
//
// **والحارسُ لا يُضعَّف ليمرّ**: **بقي عدداً مقطوعاً وتُرقّم فجوتُه** —
// **ومن جعله `>= 34` سمح لتدفّقٍ أن يسقط صامتاً.** **وميزةٌ جديدةٌ
// تُعلَن بتبديل رقمٍ في سطرٍ يُقرأ، لا بحارسٍ يتساهل.**
const declaredFlows = 36

func TestAllFlowsDeclared(t *testing.T) {
	tr := build(t)
	if len(tr.Flows) != declaredFlows {
		t.Fatalf("التدفّقاتُ %d لا %d — تبدّل الكتالوج",
			len(tr.Flows), declaredFlows)
	}
	seen := map[string]bool{}
	for _, f := range tr.Flows {
		if seen[f.ID] {
			t.Errorf("تدفّقٌ مكرَّر: %s", f.ID)
		}
		seen[f.ID] = true
		if f.Title == "" || f.Severity == "" {
			t.Errorf("%s ناقصُ العنوان أو الشدّة", f.ID)
		}
		if len(f.NeedLevels) == 0 {
			t.Errorf("%s بلا طبقةٍ لازمة — قاعدةُ الاشتقاق مكسورة", f.ID)
		}
	}
	for i := 1; i <= declaredFlows; i++ {
		id := "F-" + pad2(i)
		if !seen[id] {
			t.Errorf("تدفّقٌ مفقود: %s", id)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **٦ · المستخرِجاتُ تُطابق مصادرَها**
// ══════════════════════════════════════════════════════════════════════
//
// **ومستخرِجٌ صامتٌ يردّ صفراً أخطرُ من مستخرِجٍ يسقط** — **ورقمٌ خاطئٌ
// يُصدَّق.** (وقع ثلاثَ مرّاتٍ في بناء هذه المرحلة: **٣ انتقالاً من ٥٥ ·
// ونوعٌ واحدٌ من ثلاثةَ عشر · و٩٢ إعداداً من ١١٨.**)

func TestExtractorsAgreeWithSource(t *testing.T) {
	tr := build(t)
	d := tr.Derived

	checks := []struct {
		name string
		got  int
		min  int
	}{
		{"أبوابٌ في الموجّه", d.Routes, 200},
		{"انتقالاتُ الطلب", d.Transitions, 50},
		{"أنواعُ قيدِ المحفظة", len(d.LedgerKinds), 9},
		{"مواضعُ الإشعار", d.NotifySites, 40},
		{"غرفُ البثّ", len(d.PublishRooms), 4},
		{"تعريفاتُ الإعدادات", d.SettingDefs, 110},
		{"حقولُ الطلب", d.OrderFields, 70},
		{"ملفّاتُ الاختبار", d.TestFiles, 150},
		{"دوالُّ الاختبار", d.TestFuncs, 400},
	}
	for _, c := range checks {
		if c.got < c.min {
			t.Errorf("%s = %d — والحدُّ الأدنى المعقول %d. **المستخرِجُ مكسورٌ أو المصدرُ تبدّل.**",
				c.name, c.got, c.min)
		}
	}

	// **والإعداداتُ المُغيِّرةُ للسلوك ٩٦** — كما في `CONFIG_IMPACT_MAP.md`.
	//
	// **وصارت ستّاً وتسعين في دورةِ ٣١** — `sales.commission_source`
	// (`XG-13`): **مفتاحٌ يبدّل قاعدةَ حسابِ مالٍ يُدفَع.**
	if d.BehaviourSettings != 96 {
		t.Errorf("إعداداتُ السلوك = %d لا ٩٦ — راجِعْ قاعدةَ `behaviour` أو المعجم",
			d.BehaviourSettings)
	}
	// **والموجَّهُ من الإشعارات ١١ من ٤٩** — حقيقةٌ مقيسةٌ في إغلاق المنظومة.
	if d.NotifyTargeted > d.NotifySites {
		t.Errorf("الموجَّهُ %d أكبرُ من المجموع %d — القياسُ مكسور",
			d.NotifyTargeted, d.NotifySites)
	}
	t.Logf("derived: أبوابٌ %d · انتقالاتٌ %d · أنواعٌ %d · إشعاراتٌ %d/%d · غرفٌ %d",
		d.Routes, d.Transitions, len(d.LedgerKinds), d.NotifyTargeted, d.NotifySites, len(d.PublishRooms))
	t.Logf("        إعداداتٌ %d (سلوكٌ %d) · حقولُ طلبٍ %d · اختباراتٌ %d في %d ملفّاً",
		d.SettingDefs, d.BehaviourSettings, d.OrderFields, d.TestFuncs, d.TestFiles)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func pad2(n int) string {
	s := itoa(n)
	if len(s) < 2 {
		return "0" + s
	}
	return s
}

// TestFinancialSettingsExist **البند ٩ من `P-4`** — مفاتيحُ الإعدادات
// الماليّةِ موجودةٌ في المعجم الفعليّ.
//
// **ولا يُنسَخ مفتاحٌ من ذاكرة.** `fininv.FinancialSettings` تُسمّي ستّةً
// وعشرين مفتاحاً تدخل حساباً ماليّاً، **وهذا يطابقها بما يستخرجه المولّدُ
// من `settings/catalog.go`** — **فمفتاحٌ يُعاد تسميتُه غداً يُسقط البناءَ
// بدل أن يصمت الحسابُ.**
func TestFinancialSettingsExist(t *testing.T) {
	backend, _ := roots(t)
	defs, err := Root(backend).SettingDefs()
	if err != nil {
		t.Fatalf("معجمُ الإعدادات: %v", err)
	}
	have := map[string]bool{}
	for _, d := range defs {
		have[d.Key] = true
	}
	missing := 0
	for _, key := range fininv.FinancialSettings {
		if !have[key] {
			t.Errorf("مفتاحٌ ماليٌّ لا وجودَ له في المعجم: %q", key)
			missing++
		}
	}
	if missing == 0 {
		t.Logf("FINANCIAL SETTINGS MAPPED = %d/%d", len(fininv.FinancialSettings), len(defs))
	}
}
