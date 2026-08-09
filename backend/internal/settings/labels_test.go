package settings

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

// TestEverySettingHasArabicLabel **كلُّ مفتاحٍ في الفهرس له اسمٌ يُقرأ.**
//
// (شكوى المالك 2026-08-09 بلقطةٍ من قسم السائقين: «هذا الخيار بالأجنبيّ، لم
//
//	أفهم لماذا بإعدادات السائقين» — وكان `drivers.direct_assign` مطبوعاً
//	بمفتاحه.)
//
// # ورابعُ مواضع العائلة
//
// **سبقتها**: حالاتُ الطلب · أنواعُ صندوق السائق · أنواعُ الدفتر · أفعالُ
// التدقيق. **والقاسمُ واحد**: اسمٌ يُكتب في Go واسمٌ يُسمّى في المعجم،
// **ولا شيءَ يجمع بينهما.**
//
// **والشاشةُ مكتوبةٌ لتسقط على المفتاح الخام عند الغياب** (`?? k`) — **فلا
// خطأ ولا صمت، إنّما إنكليزيّةٌ في شاشةٍ عربيّة.** ولا يكتشفها إلّا من يفتح
// ذاك القسمَ بعينه.
//
// # ولماذا هنا لا في حارس الويب
//
// **الفهرسُ في Go والمعجمُ في الويب** — والحارسُ يجب أن يقف حيث يُضاف
// المفتاح: **من كتب سطراً في `catalog.go` يسقط بناؤه في اللحظة**، لا بعد
// أسبوعٍ حين يفتح أحدٌ الشاشة.
//
// # ولا يُفحص التلميح
//
// **الاسمُ واجبٌ والتلميحُ اختياريّ** — (قرارُ المالك 2026-08-09: «هي النصوص
// كلها ما تلزم»). **ومفتاحٌ يُفهم من عنوانه لا يُشرح.**
func TestEverySettingHasArabicLabel(t *testing.T) {
	raw, err := os.ReadFile("../../../web/packages/i18n/src/locales/ar.json")
	if err != nil {
		t.Skipf("المعجمُ غيرُ متاح: %v", err)
	}
	var dict map[string]any
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatalf("المعجمُ غيرُ صالح: %v", err)
	}
	admin, _ := dict["admin"].(map[string]any)
	set, _ := admin["settings"].(map[string]any)
	keys, _ := set["keys"].(map[string]any)
	if keys == nil {
		t.Fatal("لا admin.settings.keys في المعجم")
	}

	var missing []string
	for _, d := range Catalog {
		node, ok := keys[d.Key].(map[string]any)
		if !ok {
			missing = append(missing, d.Key)
			continue
		}
		label, _ := node["label"].(string)
		if strings.TrimSpace(label) == "" {
			missing = append(missing, d.Key)
			continue
		}
		// **ولا حرفَ لاتينيٍّ في الاسم** — الاسمُ عربيٌّ أو ليس اسماً.
		if strings.ContainsAny(label, "abcdefghijklmnopqrstuvwxyz") {
			t.Errorf("اسمُ %q هو %q وفيه حرفٌ لاتينيّ", d.Key, label)
		}
	}
	sort.Strings(missing)
	for _, k := range missing {
		t.Errorf("المفتاح %q بلا اسمٍ في admin.settings.keys — يُعرض بمفتاحه في شاشةٍ عربيّة", k)
	}

	// **ولا اسمَ لمفتاحٍ لا وجودَ له.**
	//
	// **اسمٌ يبقى بعد حذف مفتاحه لا يضرّ اليوم** — **ويُضلّل غداً**: من بحث
	// عن المفتاح وجد اسمَه فظنّه قائماً. **والمعجمُ يُقرأ فهرساً لِما يُضبط.**
	live := map[string]bool{}
	for _, d := range Catalog {
		live[d.Key] = true
	}
	var stale []string
	for k := range keys {
		// **ونقطةٌ واحدةٌ تفصل العائلةَ عن المفتاح** — وما ليس على هذا الشكل
		// ليس اسمَ إعدادٍ أصلاً.
		if strings.Count(k, ".") == 1 && !live[k] {
			stale = append(stale, k)
		}
	}
	sort.Strings(stale)
	for _, k := range stale {
		t.Errorf("اسمٌ في المعجم لمفتاحٍ لا وجودَ له في الفهرس: %q", k)
	}
}
