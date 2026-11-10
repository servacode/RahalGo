package orders

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryOrderInsertStampsMoneyModel **كلُّ بابٍ يُنشئ طلباً يختم نموذجَ ماله** (الدفعة B1).
//
// العمودُ افتراضُه ١، **فبابٌ جديدٌ نسي العمودَ يُنشئ طلباً قديمَ النموذج بصمتٍ** والمفتاحُ
// مشغَّل — ولا يراه أحدٌ حتّى يُسوّى طلبٌ بالقاعدة الخطأ. فكلُّ `INSERT INTO orders (` في
// الحزمة (غيرِ الاختبارات) يذكر `money_model` قبل `VALUES`.
func TestEveryOrderInsertStampsMoneyModel(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src := string(raw)
		for i := strings.Index(src, "INSERT INTO orders ("); i >= 0; {
			rest := src[i:]
			end := strings.Index(rest, "VALUES")
			if end < 0 {
				end = len(rest)
			}
			seen++
			if !strings.Contains(rest[:end], "money_model") {
				line := strings.Count(src[:i], "\n") + 1
				t.Errorf("%s:%d — **إنشاءُ طلبٍ بلا `money_model`**", f, line)
			}
			next := strings.Index(src[i+1:], "INSERT INTO orders (")
			if next < 0 {
				break
			}
			i = i + 1 + next
		}
	}
	if seen < 5 {
		t.Fatalf("وُجد %d بابَ إنشاء فقط — يُنتظر ٥ على الأقلّ (عاديّ · خاصّان · توصيلة · واتساب)", seen)
	}
}
