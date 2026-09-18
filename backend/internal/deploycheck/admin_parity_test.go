package deploycheck

// ======================================================================
//  **ولوحةُ التجهيز كلوحة الإنتاج — إلّا ما تفرضه البيئة**
// ======================================================================
//
// (تدقيقُ التكافؤ ٢٠٢٦-٠٩-١٩ · `docs/testing/ADMIN-STAGING-PRIMARY-PARITY.md`.)
//
// # ولماذا حارسٌ هنا
//
// **الصورةُ واحدةٌ للبيئتين** (`/config.js` وقتَ التشغيل) — **فلا يفترق
// سلوكُ اللوحة إلّا بتهيئة التشغيل.** **وتهيئةٌ تفترق بلا سببٍ من البيئة
// تُسقط أداةً في لوحةٍ دون أختها** — **ولا شيءَ يسقط ولا سطرَ في سجلّ.**
//
// # والمقارنةُ مقارنةُ قدرةٍ لا قيمة
//
// **أربعةُ مفاتيحَ تختلف بحكم البيئة** — العنوانُ والموقعُ واسمُ البيئة
// **ونمطُ الخريطة.** **وما عداها يجب أن يطابق الإنتاج.**
//
// # ولماذا نمطُ الخريطة من مفاتيح البيئة
//
// **`maps.rahalgo.com` مضيفُ إنتاج** (`envguard.ProductionHosts`)،
// **والتجهيزُ لا يسمّيه** (`TestStagingConfigNeverNamesProduction`).
// **وحاولتُ في هذا التدقيق أن أجعله يطابق الإنتاج فأسقطني ذلك الحارس —
// وهو محقّ.** **فالقيمةُ تخصّ البيئة** — **وللتجهيز خرائطُه منذ
// ٢٠٢٦-٠٩-١٩** (`TestStagingMapsAreServedFromStagingItself`).

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const productionComposePath = "../../../deploy/compose.yml"

// envSpecific **مفاتيحُ تختلف بحكم البيئة** — وغيرُها يطابق الإنتاج.
var envSpecific = map[string]bool{
	"RAHALGO_API_URL":       true,
	"RAHALGO_SITE_URL":      true,
	"RAHALGO_ENVIRONMENT":   true,
	"RAHALGO_MAP_STYLE_URL": true, // مضيفُ الخرائط مضيفُ إنتاج — envguard
}

// knownEmptyInStaging **فراغٌ مُسمّىً بسببه** — **وكلُّ فراغٍ غيرِه يُسقط
// الحارس.**
//
// **وكان فيه نمطُ الخريطة** حتّى وُجدت للتجهيز خرائطُه (٢٠٢٦-٠٩-١٩) —
// **فحُذف، وصار فراغُه يُسقط الحارس.**
var knownEmptyInStaging = map[string]string{}

// webRuntimeEnv **مفاتيحُ `RAHALGO_*` في خدمة الويب.**
func webRuntimeEnv(t *testing.T, path string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("قراءةُ %s: %v", path, err)
	}
	var doc struct {
		Services map[string]struct {
			Environment map[string]string `yaml:"environment"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("تحليلُ %s: %v", path, err)
	}
	web, ok := doc.Services["web"]
	if !ok {
		t.Fatalf("**لا خدمةَ ويبٍ في %s**", path)
	}
	out := map[string]string{}
	for k, v := range web.Environment {
		if strings.HasPrefix(k, "RAHALGO_") {
			out[k] = v
		}
	}
	return out
}

// effectiveDefault **ما يُستعمل إن غاب ملفُّ البيئة** — `${X:-def}` ⇒ `def`.
var defaultExpr = regexp.MustCompile(`^\$\{[A-Z0-9_]+:-(.*)\}$`)

func effectiveDefault(v string) string {
	if m := defaultExpr.FindStringSubmatch(v); m != nil {
		return m[1]
	}
	return v
}

// TestAdminRuntimeConfigSurfaceMatches **المفاتيحُ نفسُها في البيئتين.**
//
// **ومفتاحٌ في الإنتاج غائبٌ عن التجهيز يعني قدرةً لا تُختبَر** — والعكسُ
// قدرةٌ لا وجودَ لها حيث يعمل الناس.
func TestAdminRuntimeConfigSurfaceMatches(t *testing.T) {
	prod := webRuntimeEnv(t, productionComposePath)
	stg := webRuntimeEnv(t, stagingComposePath)
	keys := func(m map[string]string) []string {
		var s []string
		for k := range m {
			s = append(s, k)
		}
		sort.Strings(s)
		return s
	}
	for _, k := range keys(prod) {
		if _, ok := stg[k]; !ok {
			t.Errorf("**مفتاحُ تهيئةٍ في الإنتاج غائبٌ عن التجهيز**: %s", k)
		}
	}
	for _, k := range keys(stg) {
		if _, ok := prod[k]; !ok {
			t.Errorf("**مفتاحُ تهيئةٍ في التجهيز لا وجودَ له في الإنتاج**: %s", k)
		}
	}
}

// TestAdminRuntimeConfigHasNoEmptyDefault **ولا فراغَ يعطّل بصمت** —
// **إلّا ما سُمّي سببُه.**
//
// **والفارغُ لا يسقط شيئاً**: تُفتح اللوحةُ وتعمل ناقصة. **فكلُّ فراغٍ
// إمّا خطأٌ وإمّا قرارٌ مكتوب** — **ولا ثالث.**
func TestAdminRuntimeConfigHasNoEmptyDefault(t *testing.T) {
	for k, v := range webRuntimeEnv(t, stagingComposePath) {
		empty := strings.TrimSpace(effectiveDefault(v)) == ""
		why, known := knownEmptyInStaging[k]
		switch {
		case empty && !known:
			t.Errorf("**افتراضٌ فارغٌ في تهيئة ويب التجهيز بلا سببٍ مكتوب**: %s = %q\n\n"+
				"**والفارغُ لا يسقط** — تُفتح اللوحةُ وتعمل ناقصة.", k, v)
		case !empty && known:
			t.Errorf("**%s لم يعد فارغاً** — احذفه من `knownEmptyInStaging` "+
				"(وكان سببُه: %s)", k, why)
		}
	}
}

// TestStagingMapsAreServedFromStagingItself **وخرائطُ لوحة التجهيز من
// التجهيز نفسِه** — لا من مضيفٍ آخر ولا من الإنتاج.
//
// **و`envguard` يمنع مضيفَ الإنتاج** — **وهذا يشترط الأصلَ نفسَه**:
// **نمطٌ على مضيفٍ لا يخدمه التجهيزُ يعطّل الخريطةَ كالفراغ تماماً.**
//
// **ويشترط ما يخدمه**: كتلةَ `/maps/` في `Caddyfile.staging`، **وحاملَ
// الآثار للقراءة**، **وأداةَ النسخ.** **ونمطٌ يشير إلى بابٍ لا يُخدَم
// خريطةٌ معطّلةٌ لا يسقط لها شيء.**
func TestStagingMapsAreServedFromStagingItself(t *testing.T) {
	env := webRuntimeEnv(t, stagingComposePath)
	style := effectiveDefault(env["RAHALGO_MAP_STYLE_URL"])
	api := effectiveDefault(env["RAHALGO_API_URL"])
	host := func(u string) string {
		u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
		if i := strings.IndexByte(u, '/'); i >= 0 {
			u = u[:i]
		}
		return u
	}
	if host(style) == "" || host(style) != host(api) {
		t.Errorf("**نمطُ خرائط التجهيز ليس على مضيف التجهيز**\n"+
			"  النمط = %q\n  المحرّك = %q", style, api)
	}
	if !strings.Contains(style, "/maps/") {
		t.Errorf("**نمطُ الخريطة لا يمرّ بـ`/maps/`** — وهو ما يخدمه كاديّ التجهيز: %q", style)
	}

	caddy, err := os.ReadFile("../../../deploy/staging/Caddyfile.staging")
	if err != nil {
		t.Fatalf("قراءةُ Caddyfile.staging: %v", err)
	}
	for _, want := range []string{"handle_path /maps/*", "root * /srv/maps", "respond 404"} {
		if !strings.Contains(string(caddy), want) {
			t.Errorf("**كتلةُ الخرائط ناقصةٌ في Caddyfile.staging**: %q", want)
		}
	}
	raw := readStagingComposeRaw(t)
	if !strings.Contains(raw, ":/srv/maps:ro") {
		t.Error("**لا حاملَ لآثار الخرائط في كاديّ التجهيز** (`:/srv/maps:ro`)")
	}
	if _, err := os.Stat("../../../deploy/staging/maps-sync.sh"); err != nil {
		t.Error("**لا أداةَ لنسخ آثار الخرائط** (`deploy/staging/maps-sync.sh`)")
	}
}

// TestAdminRuntimeConfigMatchesProductionExceptEnvironment **وما لا يخصّ
// البيئةَ يطابق الإنتاجَ حرفاً.**
func TestAdminRuntimeConfigMatchesProductionExceptEnvironment(t *testing.T) {
	prod := webRuntimeEnv(t, productionComposePath)
	stg := webRuntimeEnv(t, stagingComposePath)
	for k, pv := range prod {
		if envSpecific[k] {
			continue
		}
		sv, ok := stg[k]
		if !ok {
			continue // **يحرسه `TestAdminRuntimeConfigSurfaceMatches`.**
		}
		if got := effectiveDefault(sv); got != pv {
			t.Errorf("**%s يفترق عن الإنتاج بلا سببٍ من البيئة**\n"+
				"  الإنتاج = %q\n  التجهيز = %q", k, pv, got)
		}
	}
}
