package notifications_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **جردُ الإشعارات — كلُّ خبرٍ يرنّ في تطبيقه ويفتح على شيءٍ موجود**
// ══════════════════════════════════════════════════════════════════════
//
// (أمرُ المالك ٢٠٢٦-٠٨-١٣: «اعمل جرداً للإشعارات مثل الجرد بحساب
//
//	المستخدم الذي فعلناه سابقاً».)
//
// # ما كشفه الجرد
//
// **١ · «طلبٌ جديدٌ بنفس مسارك» كان يرنّ في كلّ تطبيقات صاحبه** — لأنّ
// `Apps` لم تُملأ، **وفارغُها يعني كلَّها.** والحسابُ نفسُه قد يكون
// زبوناً، **فيرنّ طلبُ عملٍ في تطبيق الزبون.** والقاعدةُ مكتوبةٌ في
// `orders/notify.go` منذ زمن — **ونُسيت في موضعٍ ثانٍ**، وهو ما تمنعه
// هذه الفحوص.
//
// **٢ · وجهتُه كانت `/portal`** — **ولا مسارَ بهذا الاسم في الموقع
// أصلاً**: جذورُه `driver` و`store` و`rep` و`dashboard` و`(site)`.
// **ورابطٌ يفتح على لا شيءٍ يُقرأ عطبا.**
//
// **٣ · وإسنادُ الطلب كان يفتح `/orders`** — وهي صفحةُ طلبات الزبون:
// **من أُسند إليه طلبٌ فضغط الخبرَ وجد مشترياته هو.**
//
// # ولماذا فحصٌ على النصّ لا على السلوك
//
// **الوجهةُ والتطبيقُ نصّان يُكتبان في بنيةٍ ولا يُقرآن في اختبارِ
// سلوك**: النداءُ ينجح، والإشعارُ يُحفظ، **والخطأُ لا يظهر إلّا في يد
// سائقٍ في الشارع.**

var reNotify = regexp.MustCompile(`notify\.Notify\w*\(`)

// TestDriverNotificationsRingInDriverApp **ما يخصُّ السائقَ يرنّ عنده وحدَه.**
func TestDriverNotificationsRingInDriverApp(t *testing.T) {
	for _, site := range notifySites(t) {
		if !site.driver {
			continue
		}
		if !strings.Contains(site.blob, "AppDriver") && !strings.Contains(site.blob, `"driver"`) {
			t.Errorf("%s:%d — إشعارُ سائقٍ بلا Apps: يرنّ في كلّ تطبيقاته", site.file, site.line)
		}
	}
}

// TestNotificationHrefsExist **كلُّ وجهةٍ لها صفحةٌ في الموقع.**
//
// **والجذورُ تُقرأ من الموقع نفسِه** — لا من قائمةٍ تُصان بيدٍ فتشيخ.
func TestNotificationHrefsExist(t *testing.T) {
	roots := webRoots(t)
	if len(roots) == 0 {
		t.Skip("الموقعُ غيرُ متاح")
	}
	re := regexp.MustCompile(`Href:\s*"(/[a-zA-Z0-9_/-]*)"`)
	for _, site := range notifySites(t) {
		for _, m := range re.FindAllStringSubmatch(site.blob, -1) {
			href := m[1]
			root := strings.Split(strings.TrimPrefix(href, "/"), "/")[0]
			if root == "" {
				continue // الجذرُ نفسُه
			}
			if !roots[root] {
				t.Errorf("%s:%d — الوجهةُ %q لا صفحةَ لها في الموقع", site.file, site.line, href)
			}
		}
	}
}

type site struct {
	file   string
	line   int
	blob   string
	driver bool
}

// notifySites **كلُّ موضعِ إرسالٍ في المحرّك ومعه ثمانيةَ عشرَ سطراً بعده.**
func notifySites(t *testing.T) []site {
	t.Helper()
	var out []site
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lines := strings.Split(string(raw), "\n")
		for i, l := range lines {
			if !reNotify.MatchString(l) {
				continue
			}
			end := i + 18
			if end > len(lines) {
				end = len(lines)
			}
			blob := strings.Join(lines[i:end], "\n")
			out = append(out, site{
				file: filepath.ToSlash(path), line: i + 1, blob: blob,
				// **ومَن يخصُّ السائقَ يُعرف من متلقّيه** — لا من نصّه.
				driver: regexp.MustCompile(`UserID:\s*[^,\n]*[Dd]river`).MatchString(blob),
			})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("لم أجد موضعَ إرسالٍ واحدا — تبدّل شكلُ النداء والحارسُ صار أعمى")
	}
	return out
}

// webRoots **جذورُ مسارات الموقع** — من مجلّدات `app/`.
func webRoots(t *testing.T) map[string]bool {
	t.Helper()
	roots := map[string]bool{}
	base := filepath.Join("..", "..", "..", "web", "apps", "rahalgo", "src", "app")
	entries, err := os.ReadDir(base)
	if err != nil {
		return roots
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "(") {
			// **مجموعةٌ بلا مسار** — صفحاتُها في الجذر.
			inner, err := os.ReadDir(filepath.Join(base, name))
			if err != nil {
				continue
			}
			for _, in := range inner {
				if in.IsDir() {
					roots[in.Name()] = true
				}
			}
			continue
		}
		roots[name] = true
	}
	return roots
}
