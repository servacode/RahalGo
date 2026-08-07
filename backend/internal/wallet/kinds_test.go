package wallet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestLedgerKindsHaveArabicLabels **كلُّ نوعٍ يكتبه المحرّكُ في الدفتر له اسم.**
//
// (شكوى المالك ٢٠٢٦-٠٨-٠٧: «يوجد الكثير من النصوص الإنكليزيّة بكلّ اللوحات».
//
//	وقِيس على الشاشة: `platform_profit` و`platform_expense` معروضتان خامّتين
//	في خزنة الإدارة اثنتَي عشرةَ مرّةً في صفحةٍ واحدة.)
//
// # وهي ثالثةُ مواضع العائلة
//
// **سبقتها حالاتُ الطلب** («استرجاع طلب (cancelled)») **وأنواعُ صندوق
// السائق** (`order_collection` و`settlement`). **والقاسمُ واحد**: اسمٌ
// يُكتب في Go واسمٌ يُسمّى في المعجم، **ولا شيءَ يجمع بينهما.**
//
// **والشيفرةُ مكتوبةٌ لتسقط على الرمز الخام عند الفشل** — فلا خطأ ولا صمت،
// **إنّما إنكليزيّةٌ في شاشة مال.**
//
// # فيُقرأ الطرفان ويُقارَنان
//
// **تُستخرج الأنواعُ من الشيفرة نفسِها** — كلُّ `ApplyTx` في الحزمة كلِّها،
// لا من قائمةٍ تُصان بيدٍ فتشيخ.
func TestLedgerKindsHaveArabicLabels(t *testing.T) {
	kinds := map[string]bool{}
	root := ".."
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		// ApplyTx(ctx, q, id, amount, "<النوع>", ...)
		for _, m := range regexp.MustCompile(`ApplyTx\([^)]*?"([a-z_]+)"`).FindAllStringSubmatch(string(src), -1) {
			kinds[m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("المشي على الشيفرة: %v", err)
	}
	if len(kinds) < 3 {
		t.Fatalf("لم أجد إلّا %d نوعاً — تبدّل شكلُ النداء والحارسُ صار أعمى", len(kinds))
	}

	raw, err := os.ReadFile("../../../web/packages/i18n/src/locales/ar.json")
	if err != nil {
		t.Skipf("المعجمُ غيرُ متاح: %v", err)
	}
	var dict map[string]any
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatalf("المعجمُ غيرُ صالح: %v", err)
	}
	shared, _ := dict["shared"].(map[string]any)
	node, _ := shared["txKinds"].(map[string]any)
	if node == nil {
		t.Fatal("لا shared.txKinds في المعجم")
	}

	for kind := range kinds {
		v, ok := node[kind]
		if !ok {
			t.Errorf("النوع %q بلا اسمٍ في shared.txKinds — يُعرض خامّاً في شاشة مال", kind)
			continue
		}
		s, _ := v.(string)
		// **ولا حرفَ لاتينيٍّ في الاسم** — «commission عمولة» يمرّ الوجودَ
		// ويبقى غيرَ مفهوم.
		if strings.ContainsAny(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			t.Errorf("اسمُ النوع %q هو %q وفيه حرفٌ لاتينيّ", kind, s)
		}
	}
}
