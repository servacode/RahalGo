package settings

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// keyRead **نداءُ إعدادٍ بمفتاحٍ مكتوب.**
//
// **ولا يُطابَق شكلُ المفتاح، إنّما شكلُ النداء** — `"orders.csv"` نصٌّ
// و`"auth.logout"` فعلُ تدقيق، **وكلاهما على هيئة مفتاح.** ومن طابق الهيئةَ
// وحدَها يُنذر على ما ليس إعداداً، **وحارسٌ يُنذر كذباً يُطفَأ.**
//
// **والسياقُ لا يُسمّى `ctx` دائماً**: المعالجاتُ تمرّر `r.Context()`
// والخدماتُ تمرّر `ctx`. **وأوّلُ صياغةٍ لهذا الحارس طابقت `ctx` وحدَه
// فعميت عن ثلاثين مفتاحاً في حزمة الخادم** — ومرّت، فبدت الشيفرةُ نظيفةً
// وهي نصفُ مقروءة. **فيُقبل أيُّ وسيطٍ أوّل.**
var keyRead = regexp.MustCompile(
	`(?:\.Get(?:Int|Bool|String|Float)?|\b\w*[sS]etting\w*)\([\w.()]+, "([a-z0-9_.]+)"`)

// TestEverySettingReadExistsInCatalog **ما يقرؤه المحرّك يجب أن يكون في
// الفهرس.**
//
// (كشفه سؤالُ المالك 2026-08-09: «هل يوجد أيُّ خيارٍ للمستخدم يجب أن يكون
//
//	ديناميكيّاً يتحكّم به الأدمن، غيرُ موجودٍ أو نسيناه؟» — فقيس، فوُجد
//	`orders.source_proximity_m`.)
//
// # ولماذا الغيابُ أخطرُ من الخطأ
//
// **`GetInt` على مفتاحٍ مجهولٍ لا تُخطئ ولا تُسجّل** — تردّ صفراً
// (`settings.go:84`: الاحتياطيُّ صفرٌ ما لم يجد `Lookup` مدخلاً).
//
// **فيصير القرارُ عدماً لا افتراضاً**: `orders.source_proximity_m` نصفُ قطرِ
// قربٍ يُعفي المتجاورَين من رسم المصدر الإضافيّ، **وصفرٌ يعني أنّ كلَّ متجرين
// متباعدان** — فيُفرض الرسمُ على متجرين في شارعٍ واحد، **وهو نقيضُ ما يقوله
// تعليقُ الشيفرة نفسُه.**
//
// **ولا يظهر في شاشةٍ ولا في سجلّ**: الطلبُ يمرّ والرقمُ يُجمع، ولا يعرف
// أحدٌ أنّ قراراً لم يُتّخذ.
//
// # وهو الوجهُ الثاني لحارس الأسماء
//
// **`TestEverySettingHasArabicLabel` يحرس ما بين الفهرس والمعجم** — وهذا
// يحرس ما بين الفهرس والشيفرة. **والمفتاحُ لا يوجد حقّاً إلّا إذا اجتمع له
// الثلاثة**: مدخلٌ يُضبط، واسمٌ يُقرأ، وقارئٌ يعمل به.
func TestEverySettingReadExistsInCatalog(t *testing.T) {
	live := map[string]bool{}
	for _, d := range Catalog {
		live[d.Key] = true
	}

	type site struct{ key, file string }
	var orphans []site
	seen := map[string]bool{}
	reads := 0

	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return nil //nolint:nilerr // ملفٌّ غيرُ مقروءٍ يُتخطّى ولا يُسقط الجرد
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, m := range keyRead.FindAllStringSubmatch(string(src), -1) {
			reads++
			k := m[1]
			// **ونقطةٌ واحدةٌ تفصل العائلةَ عن المفتاح** — وما ليس على هذا
			// الشكل ليس مفتاحَ إعدادٍ أصلاً.
			if strings.Count(k, ".") != 1 || live[k] || seen[k] {
				continue
			}
			seen[k] = true
			orphans = append(orphans, site{k, filepath.ToSlash(path)})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("تعذّر المشي على الشيفرة: %v", err)
	}

	// **وأرضيّةٌ تكشف الحارسَ الأعمى.**
	//
	// **يومَ يتبدّل اسمُ دالّةِ القراءة يتوقّف النمطُ عن المطابقة** — فيمرّ
	// الحارسُ صامتاً وهو لا يقرأ شيئاً. **وحارسٌ لا يرى أسوأُ من لا حارس**:
	// يمنح ثقةً بلا سند.
	if reads < 40 {
		t.Fatalf("الحارسُ لا يرى: %d نداءَ إعدادٍ فقط — أتبدّل شكلُ النداء؟", reads)
	}

	sort.Slice(orphans, func(i, j int) bool { return orphans[i].key < orphans[j].key })
	for _, o := range orphans {
		t.Errorf("المفتاح %q يُقرأ في %s ولا مدخلَ له في الفهرس — "+
			"يردّ صفراً أبداً، ولا يضبطه المالك", o.key, o.file)
	}
}
