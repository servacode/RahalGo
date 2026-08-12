package server

// **العلامةُ الخلفيّةُ تُنهي النصَّ الخام في Go — ولو كانت في تعليقِ SQL.**
//
// **وقعت ثلاثَ مرّاتٍ في يومٍ واحد** (٢٠٢٦-٠٨-١٢): تعليقٌ داخل استعلامٍ
// فيه اسمُ عمودٍ بين علامتين خلفيّتين، **فانتهى النصُّ عندها** وصار ما
// بعده شيفرةً — والمترجمُ يشكو من سطرٍ بعيدٍ عن الموضع:
//
//	invalid digit '9' in octal literal
//	expected declaration, found 'if'
//
// **والقاعدةُ مكتوبةٌ في `CLAUDE.md` ولم تمنعها** — لأنّ الكتابةَ لا
// تمنع، **والحارسُ يمنع.**
//
// # ولماذا اختبارٌ لا فحصٌ في البناء
//
// **البناءُ يمسكها أصلاً** — لكنّه يمسكها برسالةٍ تضلّل. **وهذا يمسكها
// باسمها وموضعها**، فيُصلَح في دقيقةٍ بدل أن يُبحث في ملفٍّ كامل.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoBacktickInsideRawSQL **لا علامةَ خلفيّةً داخل استعلام.**
//
// **والفحصُ على التعليقات وحدَها** (`--`): هي موضعُ الوقوع — **والشيفرةُ
// نفسُها لا تُكتب فيها علاماتٌ خلفيّة** أصلاً.
func TestNoBacktickInsideRawSQL(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("تعذّرت قراءة الملفّات: %v", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("تعذّرت قراءة %s: %v", file, err)
		}

		// **عدُّ العلامات يقول أين نحن**: فرديٌّ يعني داخلَ نصٍّ خام.
		inRaw := false
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			// **والهروبُ الصحيحُ يُستثنى**: من كتبها بضمّ نصّين
			// (`+"`+"`"+`"+`) أنهى النصَّ عمداً وأعاده — **وهو ما
			// نريده**، لا ما نمنعه.
			escaped := strings.Contains(line, `"+`) && strings.Contains(line, `+"`)
			if inRaw && !escaped && strings.HasPrefix(trimmed, "--") &&
				strings.Contains(line, "`") {
				t.Errorf("%s:%d **علامةٌ خلفيّةٌ في تعليق SQL** — تُنهي "+
					"النصَّ الخام فيصير ما بعده شيفرة:\n\t%s",
					file, i+1, trimmed)
			}
			inRaw = inRaw != (strings.Count(line, "`")%2 == 1)
		}
	}
}
