package truthdoc_test

import (
	"os"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/truthdoc"
)

// TestTruthDocIsCurrent **الوثيقةُ تطابق المحرّك — وإلّا سقط البناء.**
//
// # لماذا اختبارٌ لا سكربت
//
// سكربتٌ يُنادى باليد **يُنسى في اليوم الذي يهمّ**. والاختبارُ يجري في كلّ
// دفعةٍ إلى المشترك، **فمن غيّر قاعدةً في المحرّك ونسي الوثيقةَ يعرف فوراً.**
//
// # وما يمسكه بالضبط
//
// وقع في ٢٠٢٦-٠٨-٠٢: كُتب في الوثيقة استثناءٌ **ضيّقٌ** للمالك، **ونُفِّذ في
// الشيفرة تجاوزاً شاملاً** — فبقيت ثلاثةُ أزرارٍ ظاهرةً له والوثيقةُ تقول
// إنّها نُزعت. **ولم يكتشفها إلّا المالكُ بعينه بعد أسبوع.**
//
// **ووثيقةٌ تكذب أخطرُ من غياب الوثيقة**: غيابُها يدفعك إلى قراءة الشيفرة،
// **وكذبُها يجعلك تبني على ما ليس.**
//
// # وكيف يُصلَح السقوط
//
// `go run ./cmd/truthdoc` — يُعيد التوليدَ في مكانه.
func TestTruthDocIsCurrent(t *testing.T) {
	raw, err := os.ReadFile(truthdoc.Path)
	if err != nil {
		t.Fatalf("تعذّرت قراءة %s: %v", truthdoc.Path, err)
	}
	want, err := truthdoc.Render(string(raw))
	if err != nil {
		t.Fatalf("تعذّر التوليد: %v", err)
	}
	if want != string(raw) {
		t.Errorf(`**الوثيقةُ لا تطابق المحرّك.**

غُيّرت قاعدةٌ في الشيفرة ولم تُحدَّث %s — **ووثيقةٌ تكذب أخطرُ من غيابها.**

الإصلاح:  cd backend && go run ./cmd/truthdoc`, truthdoc.Path)
	}
}

// TestEverySectionHasAnchor كلُّ قسمٍ مولَّدٍ له موضعٌ في الوثيقة.
//
// **ومُولِّدٌ بلا فاصلٍ يُكتب في الفراغ**: يُبنى الجدولُ ولا يراه أحد،
// **فيُظنّ أنّ الوثيقة تصفه وهي ساكتةٌ عنه.**
func TestEverySectionHasAnchor(t *testing.T) {
	raw, err := os.ReadFile(truthdoc.Path)
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	doc := string(raw)
	for name := range truthdoc.Sections() {
		if !contains(doc, "<!-- gen:"+name+" -->") {
			t.Errorf("القسم %q يُولَّد ولا فاصلَ له في الوثيقة", name)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
