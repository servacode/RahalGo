package deploycheck_test

// ══════════════════════════════════════════════════════════════════════
// **ملفُّ التركيب يُقرأ قبل النشر لا عنده**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-٣٠ بعد أن أسقط النشرة.)
//
// # الحادثة
//
// **`NEXT_PUBLIC_MAP_STYLE_URL` كُتب مرّتين في كتلة الويب** — إحداهما
// خارجَ `args` بمحاذاةٍ خاطئة. فردّ المُحلِّل:
//
//	did not find expected key (L96.C7-L118.C9)
//
// **ودخل مع إصلاح الخرائط فبقي نائماً أسبوعاً** — **لا اختبارَ يقرأ
// هذا الملفَّ ولا حارس**، فلا بناءٌ يسقط ولا تحقّقُ أنواعٍ يشتكي.
//
// **وظهر عند أوّل `docker compose build` على خادمٍ حيّ** — بعد رفعِ
// حزمةٍ وفكِّها فوق القائم. **وأرخصُ لحظةٍ لاكتشاف خطأِ صياغةٍ هي قبل
// أن تُلمس الآلة.**
//
// # وما يُفحص
//
// **الصياغةُ أوّلاً** — وهي التي وقعت. **ثمّ بنيةٌ صغيرةٌ تُقرأ**:
// خدماتٌ موجودةٌ بأسمائها، **ولا مفتاحَ مكرَّرٌ في خريطةٍ واحدة.**
//
// **ولا يُفحص المعنى**: أنّ الصورةَ تُبنى أو أنّ المنفذَ صحيح — تلك
// أشياءُ يقولها التشغيل. **وحارسٌ يدّعي أكثرَ ممّا يرى يُطفأ يومَ
// يكذب.**

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// composePath ملفُّ التركيب — من جذر المستودع.
const composePath = "../../../deploy/compose.yml"

func TestComposeIsValidYAML(t *testing.T) {
	raw, err := os.ReadFile(filepath.Clean(composePath))
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ %s: %v", composePath, err)
	}

	// ══════════════════════════════════════════════════════════════════
	// **و`KnownFields` لا يُشدَّد هنا**
	// ══════════════════════════════════════════════════════════════════
	//
	// **صياغةُ compose تتّسع بإصدارات دوكر** — وحارسٌ يرفض مفتاحاً
	// جديداً صحيحاً **يُسقط بناءً سليماً**، فيُطفأ. **إنّما تُفحص
	// الصياغةُ وحدَها**: هي التي أسقطت النشرة.
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf(`**صياغةُ %s معطوبة** — والنشرةُ ستسقط عند أوّل بناء.

%v

الإصلاح: صحّح المحاذاةَ — والخطأُ الغالبُ سطرٌ خارج كتلته.`, composePath, err)
	}

	// **والخدماتُ تُقرأ** — ملفٌّ يُحلَّل ولا خدمةَ فيه ليس ملفَّ تركيب.
	services, ok := doc["services"].(map[string]any)
	if !ok || len(services) == 0 {
		t.Fatal("لا خدماتٍ في ملفّ التركيب — **وملفٌّ يُحلَّل ولا خدمةَ فيه ليس تركيباً**")
	}

	// **والستُّ حاوياتٍ التي تحمل المنصّة** — من نزعت واحدةً منها بلا
	// قصدٍ يعرف قبل أن يصل الخادم.
	for _, name := range []string{"api", "web", "caddy", "postgres", "redis"} {
		if _, found := services[name]; !found {
			t.Errorf("خدمةُ %q غائبةٌ عن ملفّ التركيب", name)
		}
	}
}

// TestComposeHasNoDuplicateKeys **ولا مفتاحَ مكرَّرٌ في خريطةٍ واحدة.**
//
// **وyaml.v3 يقبل التكرارَ صامتاً في `map[string]any`** — يأخذ الأخيرَ
// ويمضي. **فمفتاحٌ كُتب مرّتين بقيمتين يُقرأ سليماً ويعمل بغير ما نوى
// كاتبُه**، وهو أخطرُ من خطأِ صياغةٍ يُسقط البناء: **هذا يُرى، وذاك
// يعمل خطأً.**
//
// **والشجرةُ الخام تُظهره** — `yaml.Node` يحفظ كلَّ مفتاحٍ كما كُتب.
func TestComposeHasNoDuplicateKeys(t *testing.T) {
	raw, err := os.ReadFile(filepath.Clean(composePath))
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		t.Skip("الصياغةُ معطوبةٌ — يقولها الفحصُ الأوّل")
	}
	walk(t, &root, "")
}

// walk **يمشي في الشجرة ويعدّ مفاتيحَ كلّ خريطة.**
func walk(t *testing.T, n *yaml.Node, path string) {
	t.Helper()
	if n.Kind == yaml.MappingNode {
		seen := map[string]int{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i].Value
			if line, dup := seen[key]; dup {
				t.Errorf("المفتاحُ %q مكرَّرٌ في %s — السطران %d و%d. "+
					"**يُقرأ سليماً ويعمل بغير ما نويت.**",
					key, pathOr(path), line, n.Content[i].Line)
			}
			seen[key] = n.Content[i].Line
			walk(t, n.Content[i+1], path+"/"+key)
		}
		return
	}
	for _, c := range n.Content {
		walk(t, c, path)
	}
}

func pathOr(p string) string {
	if p == "" {
		return "الجذر"
	}
	return p
}
