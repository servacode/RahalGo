package settings_test

// **كلُّ مفتاحٍ يُقرأ في الشيفرة يجب أن يكون في الكتالوج.**
//
// (كشفه فحصُ التغطية ٢٠٢٦-٠٨-٠٧، بقرار المالك: «نعم ابدأ».)
//
// # المرض
//
// `GetInt` لمفتاحٍ ليس في الكتالوج **لا يخطئ ولا يسجّل**: يقرأ صفَّه من
// `app_settings` إن وُجد، **وإن لم يوجد ردَّ صفراً بصمت.**
//
// **وأخطرُ منه أنّ الكتالوج هو لوحةُ الإدارة**: شاشةُ الإعدادات تدور على
// `settings.Catalog`، و`Set` ترفض ما ليس فيه بـ`ErrUnknownKey`.
//
// **فمفتاحٌ خارجَ الكتالوج قيمتُه مجمَّدةٌ إلى الأبد**: زرعتها ترحيلةٌ مرّةً
// ولا يستطيع المالكُ تغييرَها ولا يراها أصلاً. **وهي بعينُها ما بُني
// الكتالوجُ ليمنعه** — «قرارا أمانٍ يتّخذهما المالك لا قرارا نشرٍ ينتظران
// مبرمجاً».
//
// وُجد خمسةَ عشرَ منها: حدُّ العناوين · حظرُ الإلغاء · زمنُ التحضير ·
// أقلُّ مبلغِ سحبٍ · نافذةُ الشكوى · وغيرُها. **وواحدٌ منها بلا صفٍّ أصلاً**
// (`drivers.require_delivery_photo`) — **فالميزةُ مطفأةٌ إلى الأبد** ولا
// مفتاحَ لإشعالها.
//
// # ولماذا اختبارٌ لا مراجعة
//
// **لا شيءَ يصرخ**: الشيفرةُ تُبنى، والنداءُ يعمل، والقيمةُ تُقرأ. **وحدَه
// من يفتح لوحةَ الإعدادات يبحث عن مفتاحٍ فلا يجده** — ولا يعرف لماذا.

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// الشكلُ المطلوب: `.GetInt(ctx, "orders.foo")` وأخواتُها.
var settingsRead = regexp.MustCompile(`\.Get(?:String|Int|Bool|Raw)\(\s*[A-Za-z_.()]+\s*,\s*"([a-z_]+\.[a-z_]+)"`)

func TestEveryKeyReadInCodeIsInCatalog(t *testing.T) {
	root := filepath.Join("..", "..")

	known := map[string]bool{}
	for _, d := range settingsCatalog() {
		known[d.Key] = true
	}

	missing := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "node_modules" || name == ".git" || name == ".next" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// **وحزمةُ الإعدادات نفسُها مستثناة** — فيها الكتالوجُ ومحرّكُه.
		if strings.Contains(filepath.ToSlash(path), "/internal/settings/") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range settingsRead.FindAllStringSubmatch(string(src), -1) {
			if !known[m[1]] {
				rel, _ := filepath.Rel(root, path)
				missing[m[1]] = append(missing[m[1]], filepath.ToSlash(rel))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("تعذّر المشي في الشجرة: %v", err)
	}

	if len(missing) == 0 {
		return
	}
	keys := make([]string, 0, len(missing))
	for k := range missing {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("**مفاتيحُ إعدادٍ تُقرأ في الشيفرة وليست في الكتالوج:**\n\n")
	for _, k := range keys {
		b.WriteString("\t" + k + "  ←  " + strings.Join(dedupe(missing[k]), ", ") + "\n")
	}
	b.WriteString("\n**وقيمتُها مجمّدةٌ إلى الأبد**: لوحةُ الإعدادات تدور على الكتالوج، " +
		"و`Set` ترفض ما ليس فيه. **ومن لا صفَّ له في الجدول يُقرأ صفراً بصمت.**\n" +
		"الإصلاح: يُضاف كلٌّ منها إلى `Catalog` بنوعه وافتراضه ومداه، وإلى " +
		"`Groups`، وإلى المعجم (`admin.settings.keys`).")
	t.Fatal(b.String())
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := in[:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
