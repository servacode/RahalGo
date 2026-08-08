package catalog

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestMediaPathsArePrefixed **كلُّ مسارِ وسائطٍ يخرج للواجهة يبدأ بـ`/media/`.**
//
// (كشفه المالك ٢٠٢٦-٠٨-٠٨ بلقطتين: صورةُ إثبات التسليم مكسورة، وصورُ أقسام
//
//	القائمة كذلك.)
//
// # ولماذا هذا يقع أصلاً
//
// **ما في القاعدة مفتاحُ تخزينٍ لا رابط**: `2026/08/x.png`. **والواجهةُ
// تنتظر مسارَ خادم**: `/media/2026/08/x.png` — تضيف إليه أصلَ المحرّك
// فيصير رابطاً كاملاً.
//
// **وبينهما دالّةٌ واحدة**: `media.URLForPtr`. **ومن نسي نداءَها خرج
// المفتاحُ عارياً** — فلا هو رابطٌ ولا مسارُ خادم: **يُطلب من منفذ اللوحة
// (3001) لا من المحرّك (8080) فيردّ 404، وحتى لو لُفّ بـ`mediaUrl` خرج
// `http://…:8080/2026/08/…` وردّ 404 كذلك.**
//
// **وصورةٌ مكسورةٌ في إثبات تسليمٍ تعني أنّ الإثباتَ غيرُ موجود** — وهو ما
// يُحتكم إليه في نزاع.
//
// # والقاعدةُ تُقاس على البنى لا على النداءات
//
// **الحقلُ الذي اسمُه `ImageURL`/`LogoURL`/`ThumbURL` وما شابه، إن مُلئ من
// عمودٍ في القاعدة، وجب أن يمرّ على `URLFor`.** ويُقاس هنا على الأبسط
// والأدقّ: **كلُّ `Scan(` يقرأ حقلَ رابطٍ يجب أن يليه — في الدالّة نفسِها —
// نداءُ `URLFor`.**
//
// **وثلاثةُ مواضعَ في هذا الملفّ وحدَه فاتت** بينما الأصنافُ في الدالّة
// نفسِها كانت تُبادَأ — **فالعلّةُ سهو، والسهوُ ما يُحرَس.**
func TestMediaPathsArePrefixed(t *testing.T) {
	// حقولٌ قيمتُها مسارُ وسائط — **ويُلتقط المتغيّرُ مع الحقل.**
	//
	// **ولا تكفي «في الدالّة نداءُ URLFor»**: أوّلُ صيغةٍ كتبتُها كانت كذلك،
	// **وزُرع فيها العيبُ الأصليُّ بعينه فلم تمسكه** — لأنّ الأصنافَ في
	// `GetMenu` تُبادَأ فمرّت الأقسامُ في ظلِّها. **فلكلّ حقلٍ حسابُه.**
	field := regexp.MustCompile(`&(\w+)\.(\w*(?:Image|Logo|Avatar|Photo|Thumb|Proof)\w*URL)\b`)
	scan := regexp.MustCompile(`(?s)\.(?:Scan|QueryRow)\(`)

	checked := 0
	for _, dir := range []string{".", "../orders", "../offers", "../identity"} {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			src := string(raw)
			// **الدالّةُ وحدةُ الفحص** — من قرأ في دالّةٍ بادَأ فيها.
			for _, fn := range splitFuncs(src) {
				if !scan.MatchString(fn.body) {
					continue
				}
				for _, m := range field.FindAllStringSubmatch(fn.body, -1) {
					ref := m[1] + "." + m[2] // مثال: sec.ImageURL
					checked++
					// **الإسنادُ بعينه لا مجرّدُ وجودِ الدالّة في الدالّة.**
					if strings.Contains(fn.body, ref+" = media.URLFor") ||
						strings.Contains(fn.body, ref+" = URLFor") {
						continue
					}
					t.Errorf("%s · %s: يقرأ %s من القاعدة ولا يُسنِد له media.URLForPtr — يخرج مفتاحُ تخزينٍ عارياً فتُكسر الصورة",
						path, fn.name, ref)
				}
			}
			return nil
		})
	}
	if checked < 5 {
		t.Fatalf("لم أفحص إلّا %d دالّة — تبدّل شكلُ القراءة والحارسُ صار أعمى", checked)
	}
}

type goFunc struct{ name, body string }

// splitFuncs يقصّ المصدرَ عند كلّ `func ` — تقريبٌ يكفي: **ما يهمّنا أن
// النداءَ والمبادأة في كتلةٍ واحدة.**
func splitFuncs(src string) []goFunc {
	var out []goFunc
	head := regexp.MustCompile(`(?m)^func (?:\([^)]*\) )?(\w+)`)
	locs := head.FindAllStringSubmatchIndex(src, -1)
	for i, l := range locs {
		end := len(src)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		out = append(out, goFunc{name: src[l[2]:l[3]], body: src[l[0]:end]})
	}
	return out
}
