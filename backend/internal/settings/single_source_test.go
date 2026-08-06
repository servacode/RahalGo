package settings_test

// **حارسُ المصدر الواحد** — لا يُقرأ `app_settings` إلّا من هذه الحزمة.
//
// # المرض
//
// كان خمسةَ عشرَ موضعاً يقرأ الجدولَ بيده:
//
//	COALESCE((SELECT (value#>>'{}')::int FROM app_settings
//	          WHERE key = 'drivers.cash_limit'), 500000)
//
// **والرقمُ في آخر السطر هو الداء**: افتراضٌ ثانٍ للمفتاح نفسِه، والفهرسُ يحمل
// الأوّل. **فيُغيَّر أحدُهما ويبقى الآخر** — ولا يظهر الفرقُ حتى تُمحى القيمةُ
// من القاعدة، فيعمل موضعٌ برقمٍ وموضعٌ بآخر **لمفتاحٍ واحد.**
//
// **ولا يظهر في أيّ خطأ**: كلا الرقمين صالح، وكلُّ استعلامٍ يعمل وحدَه.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «الأرقامُ تصدر من مكانٍ مركزيٍّ واحدٍ وليس من
// أماكنَ متفرّقة».)
//
// # ولماذا اختبارٌ لا مراجعة
//
// **نُظّفت خمسةَ عشرَ موضعاً مرّةً، وستعود** — يكتبها من يحتاج رقماً في
// استعلامٍ ولا يعرف أنّ للحزمة قارئاً. **وقاعدةٌ يحرسها الانتباهُ وحدَه قاعدةٌ
// تسقط.**

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// settingsCatalog يكشف الفهرسَ للاختبار الخارجيّ.
func settingsCatalog() []settings.Def { return settings.Catalog }

// settingsLookup يكشف البحثَ للاختبار الخارجيّ.
func settingsLookup(k string) (settings.Def, bool) { return settings.Lookup(k) }

// TestNoRawSettingsReads لا استعلامَ يقرأ `app_settings` خارجَ هذه الحزمة.
func TestNoRawSettingsReads(t *testing.T) {
	// **والجذرُ يُبنى من موضع الاختبار** — لا مسارٌ مطلقٌ يتبع جهازَ من كتبه.
	root := filepath.Join("..", "..")

	// **ومواضعُ مسموحة**: الترحيلاتُ تبذر وتنقل (وهي SQL لا Go)، والحزمةُ
	// نفسُها هي القارئ. **والتعليقاتُ تذكر الجدولَ لتشرح** — فيُفحص السطرُ
	// الفعليّ لا ذكرُ الاسم.
	skip := []string{
		filepath.Join("internal", "settings"),
		filepath.Join("internal", "migrate"),
		filepath.Join("scripts"),
	}

	var bad []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		for _, s := range skip {
			if strings.HasPrefix(rel, s) {
				return nil
			}
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for i, line := range strings.Split(string(src), "\n") {
			trimmed := strings.TrimSpace(line)
			// **التعليقاتُ تُستثنى** — تشرح الجدولَ ولا تقرؤه.
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "--") {
				continue
			}
			// **والداءُ ليس القراءةَ بل الاحتياطيَّ المكتوبَ معها.**
			//
			// نقطةُ اللوحة تقرأ الجدولَ كلَّه لتعرضه — **وهي قراءةٌ بلا
			// افتراضٍ فلا تُنشئ نسخةً ثانية.** والاختباراتُ تبذر قيماً
			// لتفحص سلوكاً، **والبذرُ كتابةٌ لا قراءة.**
			//
			// **و`COALESCE` مع `app_settings` هو التوقيعُ بعينه**: رقمٌ في
			// آخر السطر يقول «إن غاب المفتاح فهذا افتراضُه» — **والفهرسُ
			// يقوله أيضاً**، فصار للمفتاح افتراضان.
			if !strings.Contains(line, "app_settings") ||
				!strings.Contains(strings.ToUpper(line), "COALESCE") {
				continue
			}
			bad = append(bad, rel+":"+itoa(i+1)+" — "+trimmed)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("تعذّر المشي في الشجرة: %v", err)
	}
	if len(bad) > 0 {
		t.Fatalf("قراءةٌ مباشرةٌ لجدول الإعدادات خارجَ حزمتها — **والافتراضُ يُكتب مرّتين**:\n  %s\n\n"+
			"البديل: settings.Store.GetInt/GetString/GetBool — تقرأ الافتراضَ من الفهرس.",
			strings.Join(bad, "\n  "))
	}
}

// TestEveryCatalogKeyHasDefault كلُّ مفتاحٍ له افتراضٌ من نوعه.
//
// **وافتراضٌ ناقصٌ يُقرأ صفراً**: مفتاحُ مهلةٍ بلا افتراضٍ يجعل كلَّ طلبٍ
// متأخّراً في اللحظة الأولى، **ومفتاحُ سقفٍ بلا افتراضٍ يمنع كلَّ سائق.**
func TestEveryCatalogKeyHasDefault(t *testing.T) {
	for _, d := range settingsCatalog() {
		if d.Default == nil {
			t.Errorf("%s بلا افتراض", d.Key)
			continue
		}
		switch d.Kind {
		case "int", "money":
			if _, ok := d.Default.(int); !ok {
				t.Errorf("%s رقميٌّ وافتراضُه %T", d.Key, d.Default)
			}
		case "bool":
			if _, ok := d.Default.(bool); !ok {
				t.Errorf("%s منطقيٌّ وافتراضُه %T", d.Key, d.Default)
			}
		case "choice", "text":
			s, ok := d.Default.(string)
			if !ok {
				t.Errorf("%s نصّيٌّ وافتراضُه %T", d.Key, d.Default)
				continue
			}
			// **وافتراضُ الخيار من خياراته** — وإلّا عمل النظامُ بقيمةٍ
			// **لا تظهر في الشاشة أصلاً**، فلا يستطيع أحدٌ أن يعيده إليها.
			if d.Kind == "choice" && !contains(d.Options, s) {
				t.Errorf("%s افتراضُه %q وليس من خياراته %v", d.Key, s, d.Options)
			}
		}
	}
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
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

// TestShowWhen_PointsAtRealKeysAndValues **شرطُ الظهور يشير إلى ما هو قائم.**
//
// # المسألة
//
// `ShowWhen` يقول «لا تُظهرني إلّا إذا كان مفتاحُ كذا يساوي كذا». **وإن كان
// المفتاحُ المذكورُ غيرَ موجودٍ أو القيمةُ ليست من خياراته، لم يظهر الحقلُ
// أبداً** — لا في وضعٍ ولا في آخر.
//
// **وغيابٌ صامتٌ أسوأُ من خطأ**: يُضبط المفتاحُ في الفهرس ولا يُرى في الشاشة،
// **فيُظنّ أنّه لم يُضَف** فيُضاف ثانيةً باسمٍ آخر.
func TestShowWhen_PointsAtRealKeysAndValues(t *testing.T) {
	for _, d := range settingsCatalog() {
		if d.ShowWhen == nil {
			continue
		}
		on, ok := settingsLookup(d.ShowWhen.Key)
		if !ok {
			t.Errorf("%s مشروطٌ بمفتاحٍ لا وجودَ له: %s", d.Key, d.ShowWhen.Key)
			continue
		}
		// **وشرطٌ بلا قيمةٍ ولا `NotEmpty` لا يتحقّق أبداً.**
		if len(d.ShowWhen.Equals) == 0 && !d.ShowWhen.NotEmpty {
			t.Errorf("%s مشروطٌ بلا قيمة — فلا يظهر أبداً", d.Key)
			continue
		}
		if d.ShowWhen.NotEmpty {
			continue
		}
		// **والقيمةُ من خيارات المفتاح المشروط به** — وإلّا لم تتحقّق قطّ.
		if on.Kind != "choice" {
			continue
		}
		for _, want := range d.ShowWhen.Equals {
			if !contains(on.Options, want) {
				t.Errorf("%s مشروطٌ بـ%s=%q وهي ليست من خياراته %v",
					d.Key, on.Key, want, on.Options)
			}
		}
	}
}
