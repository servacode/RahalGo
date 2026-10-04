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
