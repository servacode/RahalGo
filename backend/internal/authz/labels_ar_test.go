package authz

// **لكلّ قدرةٍ اسمٌ ووصفٌ عربيّان في المعجم** (قرارُ المالك ٢٠٢٦-١٠-٠٤، بعد الدمج).
//
// شاشةُ الأدوار تعرض القدرةَ باسمها من `terms.capabilityNames` ووصفِها من
// `terms.capabilityDescriptions`؛ وقدرةٌ بلا اسمٍ تظهر برمزها الإنجليزيّ أو بوصف
// المحرّك مكانَ الاسم. **فقدرةٌ تُضاف في الشيفرة بلا سطرَيها في المعجم تُسقط البناء.**

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var arabicLetter = regexp.MustCompile(`[\x{0600}-\x{06FF}]`)

func TestROLES_EveryCapabilityHasArabicLabel(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "web", "packages", "i18n", "src", "locales", "ar.json"))
	if err != nil {
		t.Fatalf("المعجم: %v", err)
	}
	var dict struct {
		Terms struct {
			Names map[string]string `json:"capabilityNames"`
			Descs map[string]string `json:"capabilityDescriptions"`
		} `json:"terms"`
	}
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatalf("المعجم لا يُقرأ: %v", err)
	}
	caps := All()
	if len(caps) == 0 {
		t.Fatal("لا قدرةَ في الكتالوج — والحارسُ بلا مرجع")
	}
	for _, c := range caps {
		if n := dict.Terms.Names[string(c)]; !arabicLetter.MatchString(n) {
			t.Errorf("القدرة %s بلا اسمٍ عربيٍّ في terms.capabilityNames", c)
		}
		if d := dict.Terms.Descs[string(c)]; !arabicLetter.MatchString(d) {
			t.Errorf("القدرة %s بلا وصفٍ عربيٍّ في terms.capabilityDescriptions", c)
		}
	}
}

// TestSTEPUP_EverySensitiveActionHasArabicLabel **ولكلّ فعلٍ حسّاسٍ اسمٌ عربيّ
// في نافذة التأكيد** (`admin.stepUp.actions`).
//
// (كشفه فحصُ المال ٢٠٢٦-١٠-٠٥: نافذةُ «تأكيدُ فعلٍ حسّاس» كانت تعرض رمزَ الفعل
// الإنجليزيَّ لأفعالٍ أُضيفت إلى `sensitiveActions` بلا سطرٍ في المعجم.)
func TestSTEPUP_EverySensitiveActionHasArabicLabel(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "web", "packages", "i18n", "src", "locales", "ar.json"))
	if err != nil {
		t.Fatalf("المعجم: %v", err)
	}
	var dict struct {
		Admin struct {
			StepUp struct {
				Actions map[string]string `json:"actions"`
				Facts   map[string]string `json:"facts"`
			} `json:"stepUp"`
		} `json:"admin"`
	}
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatalf("المعجم لا يُقرأ: %v", err)
	}
	if len(sensitiveActions) == 0 {
		t.Fatal("لا فعلَ حسّاسٌ في الكتالوج — والحارسُ بلا مرجع")
	}
	for _, s := range sensitiveActions {
		if n := dict.Admin.StepUp.Actions[s.Action]; !arabicLetter.MatchString(n) {
			t.Errorf("الفعل الحسّاس %s بلا اسمٍ عربيٍّ في admin.stepUp.actions", s.Action)
		}
	}
	// **وحقولُ الفعل المعروضةُ في النافذة بأسمائها العربيّة** — `StepUpGate.tsx`.
	for _, k := range []string{"amount", "role", "capability", "status", "value"} {
		if !arabicLetter.MatchString(dict.Admin.StepUp.Facts[k]) {
			t.Errorf("حقلُ التأكيد %s بلا اسمٍ عربيٍّ في admin.stepUp.facts", k)
		}
	}
}
