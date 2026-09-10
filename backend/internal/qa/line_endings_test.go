// نهاياتُ الأسطر — **حارسُ أداةٍ لا حارسُ منتَج.**
//
// # لماذا وُجد
//
// **في دورةِ ٥١ رفعتُ تغييراً بـ`git stash` لأقيس الأساسَ بدونه** —
// **فأعاد git كتابةَ الملفّات بـ`CRLF`** (`core.autocrlf = true` ولا
// `.gitattributes` في المستودع).
//
// **وفي المستودع فحوصٌ تقرأ الشيفرةَ نصّاً**: `TestFIN_TransactionBoundaries`
// تبحث عن `"\n}\n"` لتقتطع جسمَ الدالّة. **ولا يوجد ذلك في `CRLF`** —
// **فتُقرأ بقيّةُ الملفّ كتلةً واحدة**، فتجد `.Begin(` فتقول «التسويةُ
// ذرّيّة» وهي ليست كذلك.
//
// **فسقطت ثلاثُ جولاتٍ كاملةٍ على شيفرةٍ سليمة** — **والسببُ أداتي لا
// المنتَج.** **وضاعت ساعةٌ في مقارنةٍ ملوَّثة.**
//
// # وما يفعله هذا الحارس
//
// **يمسك الانحرافَ في ثانيةٍ بدل ساعة** — **ولا يُصلح شيئاً ولا يمسّ
// إعداداتِ git**: **المستودعُ على `LF` وهذا يُثبته.**
//
// **ولا يُعالَج بـ`.gitattributes` هنا**: **إضافتُه تُعيد كتابةَ
// المستودع كلِّه** — **وذاك قرارُ مالكٍ لا أثرٌ جانبيٌّ لحارس.**
package qa

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLineEndingsAreLF **لا ملفَّ مصدرٍ بنهايات `CRLF`.**
//
// **ويُقاس ما في القرص لا ما في git** — **فالفحوصُ تقرأ القرص.**
func TestLineEndingsAreLF(t *testing.T) {
	root := filepath.Join("..", "..")

	var bad []string
	scanned := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// **وما لا نملكه لا يُقاس** — الوحداتُ الخارجيّةُ وأثرُ البناء.
			switch info.Name() {
			case "vendor", "node_modules", ".git", "uploads", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		scanned++
		if strings.Contains(string(b), "\r\n") {
			rel, _ := filepath.Rel(root, path)
			bad = append(bad, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("المشي في الشجرة: %v", err)
	}
	if scanned == 0 {
		t.Fatal("لم يُقرأ ملفٌّ واحد — **الحارسُ يمرّ على فراغ**")
	}

	for i, f := range bad {
		if i >= 12 {
			t.Errorf("  … و%d غيرُها", len(bad)-12)
			break
		}
		t.Errorf("  CRLF: %s", f)
	}
	if len(bad) > 0 {
		t.Errorf("**%d ملفَّ مصدرٍ بنهايات CRLF من %d** — "+
			"**وفحوصٌ تقرأ الشيفرةَ نصّاً تُخطئ قراءتَها** "+
			"(`TestFIN_TransactionBoundaries` تبحث عن \"\\n}\\n\"). "+
			"**والسببُ المعتادُ عمليّةُ git أعادت الكتابة** "+
			"(`core.autocrlf`).", len(bad), scanned)
	}
	t.Logf("نهاياتُ الأسطر: %d ملفَّ `.go` كلُّها LF", scanned)
}
