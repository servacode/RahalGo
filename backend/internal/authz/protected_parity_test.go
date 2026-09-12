package authz

// ══════════════════════════════════════════════════════════════════════
//  **المحميُّ في المحرّك هو المحميُّ في اللوحة — حرفاً بحرف**
// ══════════════════════════════════════════════════════════════════════
//
// # لماذا يُقرأ ملفُّ واجهةٍ من اختبارِ محرّك
//
// **الحدُّ هنا** (`protected.go`) **والإخفاءُ هناك** (`rolemeta.ts`) —
// **وقائمتان لشيءٍ واحدٍ تفترقان يوماً**، وافتراقُهما صامت:
//
//	**تنقص في اللوحة** ⇒ حبّةٌ تُعرَض ونقرةٌ تُردّ ٤٠٣ بلا سببٍ مفهوم
//	**تنقص هنا**       ⇒ **دورٌ يُخفى ولا يُحرَس** — **وذاك أخطرُ**
//
// **والحارسُ في المحرّك لا في نصوص الويب بقصد**: **نصوصُ الويب تدخل
// سياقَ بنائه** — **فحارسٌ يُضاف هناك يُبدّل بصمةَ أثرٍ مُثبَتٍ**،
// وهذا ما اشترط المالكُ ألّا يقع (٢٠٢٦-٠٩-١٢).
//
// **وسابقتُه `TestCORS_WebClientHeadersAreAllowed`** — عقدٌ طرفاه في
// مستودعين يُقرأ من طرفٍ واحد.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// webProtectedRe **سطرٌ في خريطة التصنيف يقول `protected`.**
//
//	owner_super_admin: "protected",
var webProtectedRe = regexp.MustCompile(`(?m)^\s*([a-z_]+)\s*:\s*"protected"\s*,`)

func TestProtectedRolesMatchWebPolicy(t *testing.T) {
	path := filepath.Join("..", "..", "..", "web", "apps", "rahalgo", "src", "lib", "rolemeta.ts")
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("تعذّر قراءةُ سياسة اللوحة (%s): %v — "+
			"**وحارسٌ بلا مرجعٍ لا يقيس شيئاً**", path, err)
	}
	// **ويُطوى التعليقُ قبل الفحص** — **وشرحٌ يذكر رمزاً ليس تصنيفاً له.**
	src := stripTS(string(raw))
	web := []string{}
	for _, m := range webProtectedRe.FindAllStringSubmatch(src, -1) {
		web = append(web, m[1])
	}
	if len(web) == 0 {
		t.Fatal("**لم يُقرأ دورٌ محميٌّ واحدٌ من سياسة اللوحة** — " +
			"**فإمّا تبدّل شكلُ التصنيف وإمّا أُفرغ**، وكلاهما يُخفي الافتراق")
	}

	mine := ProtectedRoles()
	sort.Strings(mine)
	sort.Strings(web)
	if strings.Join(mine, "|") != strings.Join(web, "|") {
		t.Errorf("**المحميُّ يفترق بين المحرّك واللوحة**\n"+
			"   المحرّك = %s\n   اللوحة  = %s\n"+
			"   **ونقصٌ في المحرّك دورٌ يُخفى ولا يُحرَس** — "+
			"**ونقصٌ في اللوحة حبّةٌ تُعرَض ونقرةٌ تُردّ بلا سبب.**",
			strings.Join(mine, " · "), strings.Join(web, " · "))
	}
	t.Logf("المحميُّ واحدٌ في الطرفين: %s", strings.Join(mine, " · "))
}

// stripTS **يطوي تعليقَ TypeScript.**
func stripTS(src string) string {
	var out strings.Builder
	for i := 0; i < len(src); i++ {
		if src[i] == '/' && i+1 < len(src) {
			switch src[i+1] {
			case '/':
				for i < len(src) && src[i] != '\n' {
					i++
				}
				out.WriteByte('\n')
				continue
			case '*':
				if j := strings.Index(src[i+2:], "*/"); j >= 0 {
					i += 2 + j + 1
					continue
				}
				return out.String()
			}
		}
		out.WriteByte(src[i])
	}
	return out.String()
}
