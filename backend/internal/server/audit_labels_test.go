package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestAuditActionsHaveArabicLabels **كلُّ فعلٍ يُسجَّل له اسمٌ يُقرأ.**
//
// (شكوى المالك ٢٠٢٦-٠٨-٠٧: «نصوصٌ إنكليزيّةٌ بكلّ اللوحات».)
//
// # ورابعُ مواضع العائلة
//
// **سبقتها**: حالاتُ الطلب («استرجاع طلب (cancelled)») · أنواعُ صندوق
// السائق (`order_collection`) · أنواعُ الدفتر (`platform_profit`).
//
// **والقاسمُ واحد**: اسمٌ يُكتب في Go واسمٌ يُسمّى في المعجم، **ولا شيءَ
// يجمع بينهما.** والشيفرةُ مكتوبةٌ لتسقط على الرمز الخام عند الفشل — **فلا
// خطأ ولا صمت، إنّما إنكليزيّةٌ على شاشة.**
//
// **وأثبت التكرارُ أنّ الإصلاحَ وحدَه لا يكفي**: أصلحتُ `menu.section_update`
// فظهر `finance.orders_exported` في التشغيل التالي. **وما يتكرّر يُحرَس.**
//
// # وتُستخرج الأفعالُ من الشيفرة لا من قائمةٍ تُصان
//
// **قائمةٌ تُكتب بيدٍ تشيخ يومَ يُضاف فعلٌ ولا يُذكر فيها** — وهو بعينه
// العطبُ الذي نحرسه.
func TestAuditActionsHaveArabicLabels(t *testing.T) {
	actions := map[string]bool{}
	// **والفعلُ يُسجَّل من الخادم ومن حزم المجال** — فيُمشى على الاثنين.
	for _, root := range []string{".", "../catalog", "../orders", "../identity", "../settings"} {
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			// audit(...) بأيّ ترتيبٍ للوسائط — الفعلُ نصٌّ فيه نقطة.
			for _, m := range regexp.MustCompile(`\baudit\([^)]*?"([a-z_]+\.[a-z_]+)"`).FindAllStringSubmatch(string(src), -1) {
				actions[m[1]] = true
			}
			return nil
		})
	}
	if len(actions) < 5 {
		t.Fatalf("لم أجد إلّا %d فعلاً — تبدّل شكلُ النداء والحارسُ صار أعمى", len(actions))
	}

	raw, err := os.ReadFile("../../../web/packages/i18n/src/locales/ar.json")
	if err != nil {
		t.Skipf("المعجمُ غيرُ متاح: %v", err)
	}
	var dict map[string]any
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatalf("المعجمُ غيرُ صالح: %v", err)
	}
	admin, _ := dict["admin"].(map[string]any)
	audit, _ := admin["audit"].(map[string]any)
	node, _ := audit["actions"].(map[string]any)
	if node == nil {
		t.Fatal("لا admin.audit.actions في المعجم")
	}

	for a := range actions {
		v, ok := node[a]
		if !ok {
			t.Errorf("الفعل %q بلا اسمٍ في admin.audit.actions — يُعرض خامّاً في سجلّ التدقيق", a)
			continue
		}
		s, _ := v.(string)
		if strings.ContainsAny(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			t.Errorf("اسمُ الفعل %q هو %q وفيه حرفٌ لاتينيّ", a, s)
		}
	}
}
