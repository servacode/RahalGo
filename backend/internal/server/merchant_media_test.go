package server

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestMerchantKindsAreASubset **ولا يرفع تاجرٌ شعارَ المنصة.**
//
// (طلبُ المالك ٢٠٢٦-٠٨-٠٧: «كلُّ منتجٍ يكون له صورة» — فوُلدت نقطةُ رفعٍ
// في بوّابة المتجر.)
//
// # ما يحرسه
//
// **نقطةُ الرفع تأخذ النوعَ من جسم الطلب.** فمن ملك الرفعَ ملك — بلا قائمةٍ
// بيضاء — كلَّ نوعٍ في `validKinds`: **`platform_logo` و`site_background`
// و`delivery_proof`.** وحراسةُ الدور تقول «هذا تاجر» **ولا تقول ماذا يرفع.**
//
// **وأثرُه أنّ تاجراً واحداً يبدّل شعارَ المنصة كلِّها** — أو يدسّ صورةً في
// مكان إثبات تسليم.
//
// # ولماذا يُقرأ الملفّ ولا يُنادى المعالج
//
// **المعالجُ يحتاج خادماً وقاعدةَ بياناتٍ وملفَّ صورةٍ ومصادقة** — واختبارٌ
// يبني نصفَ المنصّة ليفحص سطرَ شرطٍ واحدٍ لا يُكتب. **والقائمتان نصّان في
// ملفّين**، فتُقرآن وتُقارنان.
func TestMerchantKindsAreASubset(t *testing.T) {
	allowed := parseKinds(t, "media_handlers.go", `merchantKinds = map\[string\]bool\{([^}]*)\}`)
	if len(allowed) == 0 {
		t.Fatal("لم أجد merchantKinds — تبدّل شكلُها والحارسُ صار أعمى")
	}
	all := parseKinds(t, "../media/media.go", `validKinds = map\[string\]bool\{([^}]*)\}`)
	if len(all) == 0 {
		t.Fatal("لم أجد validKinds")
	}

	// **ولا نوعَ للتاجر خارجَ ما يعرفه المحرّك** — وإلّا رُدَّ رفعُه بخطأٍ
	// لا يفهمه صاحبُ المطعم.
	for k := range allowed {
		if !all[k] {
			t.Errorf("النوع %q مسموحٌ للتاجر وليس في validKinds — رفعٌ يُردّ دائماً", k)
		}
	}

	// **وأنواعُ المنصة محرَّمةٌ عليه صراحةً** — لا بالغياب بل بالفحص.
	for _, forbidden := range []string{
		"platform_logo", "auth_background", "site_background", "delivery_proof", "banner", "avatar",
	} {
		if allowed[forbidden] {
			t.Errorf("النوع %q مسموحٌ للتاجر — وهو من أنواع المنصة", forbidden)
		}
	}
}

func parseKinds(t *testing.T, file, pattern string) map[string]bool {
	t.Helper()
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("تعذّر قراءة %s: %v", file, err)
	}
	mm := regexp.MustCompile(pattern).FindStringSubmatch(string(src))
	if mm == nil {
		return nil
	}
	out := map[string]bool{}
	for _, k := range regexp.MustCompile(`"([a-z_]+)"\s*:\s*true`).FindAllStringSubmatch(mm[1], -1) {
		out[strings.TrimSpace(k[1])] = true
	}
	return out
}
