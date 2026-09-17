package deploycheck

// ======================================================================
//  **وبيئةُ التجهيز لا تحسب المسافةَ بخطٍّ مستقيم**
// ======================================================================
//
// # ولماذا حارسٌ على ملفّ تركيب
//
// **كان `OSRM_URL` و`GEOCODER_URL` يفترغان في تركيب التجهيز**، **وكان
// ذلك عمداً**: «يُترَك فارغاً حتّى يُحتاج».
//
// **وأثرُه لا يُرى**: **المحرّكُ لا يسقط ولا يردّ خطأً** — **يُسجّل سطراً
// واحداً عند الإقلاع** (`المسارات: لا محرّك`) **ثمّ يحسب المسافةَ بخطٍّ
// مستقيمٍ فوق البيوت** ويردّها كأنّها طريق.
//
// **فكلُّ اختبارِ ملاحةٍ على التجهيز كان سيمرّ** — **وكلُّ عرضِ سعرٍ
// مبنيٍّ على مسافةٍ كان سيختلف عن الإنتاج بلا أن يقول أحدٌ شيئاً.**
// (اكتُشف ٢٠٢٦-٠٩-١٦ في تمهيد قبول `P-8`، **وقبل أن يُقاس طريقٌ واحد**.)
//
// **والعنونةُ مثلُها**: **`GEOCODER_URL` فارغةٌ تردّ نقطةً بلا اسم** —
// **فشاشةُ العنوان تعمل وتُظهر فراغاً**، ولا خطأَ في سجلّ.
//
// # ولماذا لا يكفي أن يُضبط اليومَ
//
// **من ضبطه اليومَ لم يمنع من يفرغه غداً** — **والفراغُ لا يسقط شيئاً**،
// فلا اختبارَ يحمرّ ولا نشرةَ تقف. **فالحارسُ هنا لا في رأسِ أحد.**

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// stagingComposePath تركيبُ التجهيز — من جذر المستودع.
const stagingComposePath = "../../../deploy/staging/compose.staging.yml"

// TestStagingComposeHasRoutingEngine **ومحرّكُ المسارات خدمةٌ في شبكة
// التجهيز نفسِها** — **لا في شبكة الإنتاج، ولا معدومٌ.**
func TestStagingComposeHasRoutingEngine(t *testing.T) {
	doc := readStagingCompose(t)
	services, ok := doc["services"].(map[string]any)
	if !ok || len(services) == 0 {
		t.Fatal("لا خدماتٍ في تركيب التجهيز")
	}
	if _, found := services["osrm"]; !found {
		t.Fatal("**لا خدمةَ `osrm` في تركيب التجهيز** — **والمحرّكُ الغائبُ لا يسقط:\n" +
			"يحسب المسافةَ بخطٍّ مستقيمٍ فوق البيوت ويردّها كأنّها طريق،\n" +
			"فيمرّ كلُّ اختبارِ ملاحةٍ كذباً.**\n\n" +
			"الإصلاح: أعِد خدمةَ `osrm` — والصورةُ مبنيّةٌ من `routing/Dockerfile`.")
	}

	// **ولا تُوصَل بشبكة الإنتاج لتبلغ محرّكَه** (`TQ-4`) — **وصلٌ لأجل
	// توفيرِ صورةٍ موجودةٍ أصلاً يكسر العزلَ بلا مقابل.**
	osrm, _ := services["osrm"].(map[string]any)
	if _, joined := osrm["networks"]; joined {
		t.Error("**خدمةُ `osrm` في التجهيز تُعلن شبكةً صريحة** — " +
			"**والافتراضُ شبكةُ المكدّس وحدَها**، وإعلانُ شبكةٍ بابُ وصلٍ بالإنتاج.")
	}
}

// TestStagingComposeRoutingDefaultsAreNotEmpty **ولا يُفترَض الفراغ.**
//
// **ومن نسي المتغيّرَ في `.env` وقع على الافتراض** — **ففراغُ الافتراض
// هو ما يصمت.**
func TestStagingComposeRoutingDefaultsAreNotEmpty(t *testing.T) {
	raw := readStagingComposeRaw(t)
	for _, bad := range []string{
		"OSRM_URL: ${STAGING_OSRM_URL:-}",
		"GEOCODER_URL: ${STAGING_GEOCODER_URL:-}",
	} {
		if strings.Contains(raw, bad) {
			t.Errorf("**افتراضٌ فارغٌ في تركيب التجهيز**: %s\n\n"+
				"**والفارغُ لا يسقط** — يُقلع المحرّكُ ويعمل ويكذب.\n"+
				"الإصلاح: ضع الافتراضَ الصحيحَ في التركيب نفسِه.", bad)
		}
	}
	// **والقيمتان تُقرآن صراحةً** — **حتّى لا يمرَّ افتراضٌ بصيغةٍ أخرى.**
	for _, want := range []string{
		"OSRM_URL: ${STAGING_OSRM_URL:-http://osrm:5000}",
		"GEOCODER_URL: ${STAGING_GEOCODER_URL:-https://nominatim.openstreetmap.org}",
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("**غاب الافتراضُ المنتظَر**: %s", want)
		}
	}
}

// TestStagingComposeKeepsProductionOutOfReach **ولا يُشار إلى الإنتاج.**
func TestStagingComposeKeepsProductionOutOfReach(t *testing.T) {
	// **والتعليقاتُ تُنزَع قبل الفحص** — **ورأسُ الملفّ يذكر أسماءَ
	// الإنتاج عمداً ليقابلها بأسماء التجهيز**، **وحارسٌ يقرأ التعليقَ
	// يحمرّ على توثيقٍ صحيح.** (وقع أوّلَ تشغيلٍ لهذا الحارس.)
	raw := stripComments(readStagingComposeRaw(t))
	for _, bad := range []string{"rahalgo-osrm-1", "rahalgo_default"} {
		if strings.Contains(raw, bad) {
			t.Errorf("**تركيبُ التجهيز يذكر %q** — **وهذا اسمُ إنتاج.**", bad)
		}
	}
	// ══════════════════════════════════════════════════════════════
	// **ومضيفُ الإنتاج يُطابَق مضيفاً لا سلسلةَ حروف** (٢٠٢٦-٠٩-١٧)
	// ══════════════════════════════════════════════════════════════
	//
	// **وكان الفحصُ `Contains("api.rahalgo.com")`** — **و
	// `staging-api.rahalgo.com` يحتوي تلك السلسلةَ حرفاً بحرف.**
	// **فمضيفُ التجهيز الحقيقيُّ يُقرأ مضيفَ إنتاج**، **ويحمرّ
	// الحارسُ على تركيبٍ صحيح.**
	//
	// **والقصدُ لم يتبدّل**: **`https://api.rahalgo.com` يبقى
	// مرفوضاً** — **والحدُّ هو حدُّ المضيف لا موضعُ الحروف.**
	if m := prodHost.FindString(raw); m != "" {
		t.Errorf("**تركيبُ التجهيز يشير إلى مضيف الإنتاج**: %q — "+
			"**ونداءُ تجهيزٍ يبلغ قاعدةَ الإنتاج لا يُستدرَك.**", strings.TrimSpace(m))
	}
}

func readStagingComposeRaw(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(stagingComposePath))
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ %s: %v", stagingComposePath, err)
	}
	// **وgit على ويندوز يكتب `CRLF`** — **وحارسٌ يطابق `LF` يحمرّ على
	// شجرةٍ سليمة.**
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

func readStagingCompose(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(readStagingComposeRaw(t)), &doc); err != nil {
		t.Fatalf("**صياغةُ تركيب التجهيز معطوبة**: %v", err)
	}
	return doc
}

// stripComments **يُبقي القيمَ ويطرح الشروح.**
//
// **ولا يُحلَّل YAML لهذا**: **الشرحُ يُطرح قبل أن يُقرأ نصٌّ**، **وما
// بقي هو ما ينفّذه دوكر.**
func stripComments(raw string) string {
	var b strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// prodHost **مضيفُ الإنتاج مطابَقاً بحدوده** — **لا كسلسلةٍ داخل غيرِه.**
//
// **و`staging-api.rahalgo.com` مضيفُ تجهيزٍ مشروع** — **يسبق اسمَه
// شَرطةٌ، فلا يُقرأ إنتاجاً.**
var prodHost = regexp.MustCompile(`(^|[^A-Za-z0-9.-])\.?api\.rahalgo\.com`)
