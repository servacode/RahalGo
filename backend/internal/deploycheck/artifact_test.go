package deploycheck_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ══════════════════════════════════════════════════════════════════════
// **ابنِ مرّةً ورقِّ الأثرَ عينَه — حارسٌ دائم** (دورةُ ٧١)
// ══════════════════════════════════════════════════════════════════════
//
// # لماذا حارسٌ لا ملاحظةٌ في مراجعة
//
// **وعودةُ `build:` سطرٌ واحدٌ يمرّ في مراجعةٍ عجلى** — **وأثرُه أنّ
// الإنتاجَ يبني ثنائيّاً لم يختبره أحد وهو يظنّه ترقية.**
// **وذاك عطبٌ لا يظهر إلّا حين يظهر فرقٌ بين بناءين.**

const stagingComposePath = "../../../deploy/staging/compose.staging.yml"

func loadCompose(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ %s: %v", path, err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("%s ليس YAML صالحاً: %v", path, err)
	}
	svcs, _ := doc["services"].(map[string]any)
	if svcs == nil {
		t.Fatalf("%s بلا `services`", path)
	}
	return svcs
}

func apiService(t *testing.T, path string) map[string]any {
	t.Helper()
	svcs := loadCompose(t, path)
	api, _ := svcs["api"].(map[string]any)
	if api == nil {
		t.Fatalf("%s بلا خدمة `api`", path)
	}
	return api
}

// TestC71_BackendNeverBuildsAtDeploy **لا بناءَ للمحرّك في أيّ بيئة.**
func TestC71_BackendNeverBuildsAtDeploy(t *testing.T) {
	for _, p := range []string{composePath, stagingComposePath} {
		api := apiService(t, p)
		if _, has := api["build"]; has {
			t.Errorf("**%s يبني المحرّكَ وقتَ النشر** — "+
				"**فأثرٌ اختُبر في بيئةٍ ليس الأثرَ العاملَ في أخرى.** "+
				"(دورةُ ٧١: `image:` لا `build:`)", p)
		}
		if _, has := api["image"]; !has {
			t.Errorf("**%s لا يعلن أثراً صريحاً** — **والنشرُ اختيارُ "+
				"أثرٍ لا بناؤه.**", p)
		}
	}
}

// TestC71_ImageFailsClosed **ولا بناءَ صامتٌ بديلاً عن أثرٍ غائب.**
//
// **و`${VAR:?}` تُسقط الأمرَ إن لم يُحدَّد** — **و`${VAR:-something}`
// تمضي بصمتٍ إلى شيءٍ آخر، وهو عينُ ما يُحذَر منه.**
func TestC71_ImageFailsClosed(t *testing.T) {
	for _, p := range []string{composePath, stagingComposePath} {
		api := apiService(t, p)
		img, _ := api["image"].(string)
		if !strings.Contains(img, "RAHALGO_API_IMAGE") {
			t.Errorf("**%s لا يقرأ الأثرَ من `RAHALGO_API_IMAGE`** — %q", p, img)
			continue
		}
		if !strings.Contains(img, ":?") {
			t.Errorf("**%s يقبل أثراً غيرَ محدَّدٍ بلا سقوط** — %q · "+
				"**والفشلُ يجب أن يكون مغلقاً.**", p, img)
		}
		if strings.Contains(img, ":-") {
			t.Errorf("**%s فيه قيمةٌ احتياطيّةٌ صامتة** — %q · "+
				"**فيُنشَر ما لم يُختبَر.**", p, img)
		}
		if strings.Contains(img, "latest") {
			t.Errorf("**%s يسمّي وسماً متحرّكاً** — %q · "+
				"**و`latest` تقول «آخرُ ما بُني» لا «ما اعتُمد».**", p, img)
		}
	}
}

// TestC71_PromotionToolingExists **والأداتان قائمتان وتحرسان.**
//
// **ونصٌّ مفقودٌ يُحوّل العقدَ إلى نيّة** — **فيُعاد الناسُ إلى
// `docker compose up` اليدويّ وينكسر كلُّ ما سبق.**
func TestC71_PromotionToolingExists(t *testing.T) {
	for path, must := range map[string][]string{
		"../../../deploy/build-artifact.sh": {
			"SOURCE_COMMIT", "BUILD_ID", "release-", "docker image save", "sha256sum",
		},
		"../../../deploy/promote.sh": {
			"--no-build", "EXPECTED", "docker image inspect", "source_commit",
		},
	} {
		b, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			t.Fatalf("**أداةُ الترقية مفقودة**: %s — %v", path, err)
		}
		for _, needle := range must {
			if !strings.Contains(string(b), needle) {
				t.Errorf("**%s فقد عنصراً من عقده**: %q", path, needle)
			}
		}
	}
}

// TestC71_BuildIdentityIsInjectedAtBuild **الهويّةُ تُحقَن في الأثر.**
//
// **وهويّةٌ تُقرأ من متغيّر تشغيلٍ تُبدَّل بعد النشر** — **فتقول
// الصورةُ ما لا تعمله.**
func TestC71_BuildIdentityIsInjectedAtBuild(t *testing.T) {
	b, err := os.ReadFile(filepath.Clean("../../Dockerfile"))
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ Dockerfile: %v", err)
	}
	src := string(b)
	for _, arg := range []string{"SOURCE_COMMIT", "BUILD_ID"} {
		if !strings.Contains(src, "ARG "+arg) {
			t.Errorf("**Dockerfile لا يستقبل `%s` وقتَ البناء** — "+
				"**فالهويّةُ تصير متغيّرَ تشغيلٍ يُبدَّل.**", arg)
		}
	}
	// **ولا تُكتَب الهويّةُ في `environment` لأيّ بيئة** — **وذاك
	// بابُ تزويرٍ صامت.**
	for _, p := range []string{composePath, stagingComposePath} {
		api := apiService(t, p)
		env, _ := api["environment"].(map[string]any)
		for _, k := range []string{"SOURCE_COMMIT", "BUILD_ID"} {
			if _, has := env[k]; has {
				t.Errorf("**%s يحقن `%s` وقتَ التشغيل** — "+
					"**هويّةٌ تُغيَّر بعد البناء ليست هويّة.**", p, k)
			}
		}
	}
}
