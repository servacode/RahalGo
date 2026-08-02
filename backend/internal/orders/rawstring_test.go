package orders

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoBacktickInsideRawStrings علامةٌ خلفيةٌ داخل نصٍّ خام تُنهيه.
//
// **وقعت خمسَ مرّاتٍ في هذه الجلسة وحدها**: يُكتب تعليقٌ عربيٌّ داخل استعلام
// SQL، ويُحاط اسمُ عمودٍ بعلامتين خلفيتين للتوضيح — **فتُنهي النصَّ الخام في
// موضعٍ لا يقصده أحد**، وتنهار الشيفرةُ بأخطاءٍ لا تدلّ على السبب:
// «missing ',' in argument list» في سطرٍ بعيدٍ عن العلّة.
//
// **والعددُ الفرديّ يكشفها**: كلُّ نصٍّ خام علامتان، فإن كان مجموعُها فردياً
// فثمّة نصٌّ لم يُغلق. **وحارسٌ يكشف في ثانيةٍ خيرٌ من عينٍ تبحث في عشرة أخطاء.**
func TestNoBacktickInsideRawStrings(t *testing.T) {
	// **العلامةُ نفسُها لا تُكتب هنا** — وإلّا صار الحارسُ أوّلَ من يُبلَّغ عنه.
	tick := string(rune(96))
	root := ".."
	var bad []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		if strings.Count(string(b), tick)%2 != 0 {
			bad = append(bad, p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("المسح تعذّر: %v", err)
	}
	if len(bad) > 0 {
		t.Errorf("علاماتٌ خلفيةٌ فردية — نصٌّ خام لم يُغلق:\n  %s",
			strings.Join(bad, "\n  "))
	}
}
