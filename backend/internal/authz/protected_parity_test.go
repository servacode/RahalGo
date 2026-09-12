package authz

// ══════════════════════════════════════════════════════════════════════
//  **أصنافُ الأدوار في المحرّك واللوحة — جدولٌ واحدٌ حرفاً بحرف**
// ══════════════════════════════════════════════════════════════════════
//
// # لماذا يُقرأ ملفُّ واجهةٍ من اختبارِ محرّك
//
// **الحدُّ هنا** (`roleclass.go`) **والعرضُ هناك** (`rolemeta.ts`) —
// **وجدولان لشيءٍ واحدٍ يفترقان يوماً**، وافتراقُهما صامت:
//
//	**ينقص صفٌّ في اللوحة**  ⇒ حبّةٌ تُعرَض ونقرةٌ تُردّ بلا سببٍ مفهوم
//	**ينقص صفٌّ هنا**        ⇒ **دورٌ يُخفى ولا يُحرَس** — **وذاك أخطر**
//
// **ومثالُه مقيس**: **`ops` كان `staff` في اللوحة و`legacy` هنا** —
// **فتُعرَض حبّتُه للمنح ويردُّها المحرّك**، وهذا ما يمسكه هذا الفحص.
//
// **وموضعُه المحرّكُ لا نصوصُ الويب بقصد**: **نصوصُ `web/scripts` تدخل
// سياقَ بناء الويب** — **فحارسٌ يُضاف هناك يُبدّل بصمةَ الأثر.**
// **وسابقتُه `TestCORS_WebClientHeadersAreAllowed`.**

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// webClassRe **سطرٌ في جدول التصنيف**: `code: "class",`
var webClassRe = regexp.MustCompile(`(?m)^\s*([a-z_]+)\s*:\s*"(account_type|staff|elevated|protected|legacy)"\s*,`)

// webRoleClasses **جدولُ اللوحة كما هو مكتوبٌ فيها.**
func webRoleClasses(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "web", "apps", "rahalgo", "src", "lib", "rolemeta.ts")
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("تعذّر قراءةُ سياسة اللوحة (%s): %v — "+
			"**وحارسٌ بلا مرجعٍ لا يقيس شيئاً**", path, err)
	}
	// **ويُطوى التعليقُ قبل الفحص** — **وشرحٌ يذكر رمزاً ليس تصنيفاً له.**
	src := stripTS(string(raw))
	// **ويُقرأ جدولُ التصنيف وحدَه** — **لا كلُّ سطرٍ يشبهه في الملفّ**
	// (`GROUP_TITLES` سطورُها `staff: m.terms…` فلا تطابق، **والحصرُ
	// أحوطُ من الاتّكال على ذلك**).
	const head = "const ROLE_CLASSES: Record<string, RoleClass> = {"
	i := strings.Index(src, head)
	if i < 0 {
		t.Fatal("**لم يُوجَد جدولُ التصنيف في اللوحة** — **فإمّا تبدّل اسمُه وإمّا نُقل**، " +
			"وكلاهما يُخفي الافتراق")
	}
	body := src[i:]
	if j := strings.Index(body, "\n};"); j > 0 {
		body = body[:j]
	}
	out := map[string]string{}
	for _, m := range webClassRe.FindAllStringSubmatch(body, -1) {
		out[m[1]] = m[2]
	}
	return out
}

// TestRoleClassesMatchWebPolicy **الجدولان سواء.**
func TestRoleClassesMatchWebPolicy(t *testing.T) {
	web := webRoleClasses(t)
	if len(web) == 0 {
		t.Fatal("**لم يُقرأ صفٌّ واحدٌ من جدول اللوحة** — " +
			"**فإمّا تبدّل شكلُ التصنيف وإمّا أُفرغ**، وكلاهما يُخفي الافتراق")
	}

	mine := map[string]string{}
	for _, c := range []RoleClass{
		ClassAccountType, ClassStaff, ClassElevated, ClassProtected, ClassLegacy,
	} {
		for _, code := range RolesInClass(c) {
			mine[code] = string(c)
		}
	}

	diff := []string{}
	for code, cls := range mine {
		if web[code] != cls {
			diff = append(diff, code+": المحرّك="+cls+" اللوحة="+orNone(web[code]))
		}
	}
	for code, cls := range web {
		if _, ok := mine[code]; !ok {
			diff = append(diff, code+": المحرّك=(لا شيء) اللوحة="+cls)
		}
	}
	sort.Strings(diff)
	if len(diff) > 0 {
		t.Errorf("**أصنافُ الأدوار تفترق بين المحرّك واللوحة**:\n   %s\n"+
			"   **ونقصٌ في المحرّك دورٌ يُخفى ولا يُحرَس** — "+
			"**ونقصٌ في اللوحة حبّةٌ تُعرَض ونقرةٌ تُردّ بلا سبب.**",
			strings.Join(diff, "\n   "))
	}
	t.Logf("جدولٌ واحدٌ في الطرفين — %d رمزاً مصنَّفاً", len(mine))
}

// TestProtectedRolesMatchWebPolicy **والمحميُّ خاصّةً** — أخطرُ صفٍّ فيه.
func TestProtectedRolesMatchWebPolicy(t *testing.T) {
	web := webRoleClasses(t)
	got := []string{}
	for code, cls := range web {
		if cls == string(ClassProtected) {
			got = append(got, code)
		}
	}
	mine := ProtectedRoles()
	sort.Strings(mine)
	sort.Strings(got)
	if len(mine) == 0 {
		t.Fatal("**لا دورَ محميّاً في المحرّك** — **نقرةٌ واحدةٌ تمنح كلَّ قدرة**")
	}
	if strings.Join(mine, "|") != strings.Join(got, "|") {
		t.Errorf("**المحميُّ يفترق**: المحرّك = %s · اللوحة = %s",
			strings.Join(mine, " · "), orNone(strings.Join(got, " · ")))
	}
	t.Logf("المحميُّ واحدٌ في الطرفين: %s", strings.Join(mine, " · "))
}

func orNone(s string) string {
	if s == "" {
		return "(لا شيء)"
	}
	return s
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
