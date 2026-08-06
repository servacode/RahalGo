package support

import (
	"os"
	"strings"
	"testing"
)

// TestSubjectsCarryNoOrderNumber **لا رقمَ في العنوان — الطلبُ خانةٌ بذاته.**
//
// (شهده المالك ٢٠٢٦-٠٨-٠٧ في صفّ البلاغ: «مكرّرة بلا فائدة».)
//
// **والعنوانُ يُكتب مرّةً في قاعدة البيانات ويبقى** — فمن أعاد الرقمَ إليه
// غداً لا يرى شيئاً ينكسر، **ويراه الزبونُ مرّتين في سطرٍ واحد.**
//
// **ولا يُقرأ العنوانُ من الشيفرة بل من ملفّها** — لأنّه يُبنى داخلَ دالّتين
// طويلتين تحتاجان قاعدةَ بياناتٍ وطلباً مغلقاً وسائقاً. **واختبارٌ يحتاج
// نصفَ المنصّة ليفحص سطراً واحداً لا يُكتب**، فيبقى السطرُ بلا حارس.
func TestSubjectsCarryNoOrderNumber(t *testing.T) {
	for _, f := range []string{"complaints.go", "driver_report.go"} {
		src := readFile(t, f)
		for _, line := range strings.Split(src, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if !strings.Contains(trimmed, "subject =") && !strings.Contains(trimmed, "subject :=") {
				continue
			}
			// **ورقمُ الطلب يدخل بـ`FormatInt` أو بـ`%d`** — كلاهما ممنوع.
			if strings.Contains(trimmed, "FormatInt") || strings.Contains(trimmed, "%d") ||
				strings.Contains(trimmed, "number") {
				t.Errorf("%s: العنوانُ يحمل رقمَ الطلب — وهو خانةٌ قائمةٌ بذاتها\n  %s", f, trimmed)
			}
		}
	}
}

// readFile يقرأ ملفَّ الحزمة نفسِها — **والاختبارُ يجري في مجلّدها.**
func readFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("تعذّر قراءة %s: %v", name, err)
	}
	return string(b)
}
