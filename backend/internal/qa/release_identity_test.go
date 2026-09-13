package qa

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/envguard"
)

// ══════════════════════════════════════════════════════════════════════
// **هويّةُ الإصدار — والأثرُ يقول ما هو** (`RID`)
// ══════════════════════════════════════════════════════════════════════
//
// # ما وقع (٢٠٢٦-٠٩-١٣)
//
// **أثرٌ رُقّي إلى الإنتاج بهويّةٍ فارغة**: `source_commit=""` و
// `build_id=""`. **فصار الإنتاجُ لا يقول أيَّ التزامٍ يعمل**، **وبطاقةُ
// «الإصدار العامل» في شاشة المراقبة تُعرَض فارغة.**
//
// # والسببُ ليس حارساً غائباً
//
// **و`deploy/build-artifact.sh` كان يمرّر الوسيطين** ، **و
// `deploy/promote.sh` كان يرفض هويّةً ناقصة.** **لكنّ المستودعَ ليس على
// خادم الإنتاج** — فلم يعمل الأوّلُ هناك، **فكُتب سكربتٌ عارضٌ بديلٌ
// نسي `--build-arg`.**
//
// **وحارسٌ يُتجاوَز ليس حارساً** — **والعلاجُ أن يُستغنى عن التجاوز.**
//
// # وما يحرسه هذا الملفّ
//
//	١ · عقدُ `Dockerfile` قائمٌ — وسيطان يُختمان بـ`-ldflags`
//	٢ · ومسارُ البناء الموثوقُ يمرّرهما
//	٣ · ويسأل الأثرَ بعد بنائه — ويسقط مغلقاً
//	٤ · ومسارُ الترقية يسأله **قبل** التبديل لا بعده
//	٥ · ولا آليّةَ هويّةٍ ثانية
//	٦ · والبابان يعرضان ما في الأثر: `/public/identity` · `/ops/health`

// repoRoot جذرُ المستودع من موضع الحزمة.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("مجلّدُ العمل: %v", err)
	}
	// internal/qa ⇒ backend ⇒ الجذر
	return filepath.Dir(filepath.Dir(filepath.Dir(wd)))
}

func ridRead(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(parts...)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("قراءةُ %s: %v", p, err)
	}
	return string(b)
}

// TestRID1_DockerfileStampsIdentity **العقدُ قائمٌ في مِلفّ البناء.**
func TestRID1_DockerfileStampsIdentity(t *testing.T) {
	root := repoRoot(t)
	df := ridRead(t, root, "backend", "Dockerfile")
	for _, want := range []string{
		"ARG SOURCE_COMMIT",
		"ARG BUILD_ID",
		"envguard.buildCommit=${SOURCE_COMMIT}",
		"envguard.buildID=${BUILD_ID}",
	} {
		if !strings.Contains(df, want) {
			t.Errorf("**عقدُ الختم انكسر في `Dockerfile`**: غابت %q", want)
		}
	}
}

// TestRID2_AuthoritativeBuildPassesArgs **ومسارُ البناء يمرّرهما.**
func TestRID2_AuthoritativeBuildPassesArgs(t *testing.T) {
	root := repoRoot(t)
	sh := ridRead(t, root, "deploy", "build-artifact.sh")
	for _, want := range []string{
		`--build-arg "SOURCE_COMMIT=$SOURCE_COMMIT"`,
		`--build-arg "BUILD_ID=$BUILD_ID"`,
	} {
		if !strings.Contains(sh, want) {
			t.Errorf("**مسارُ البناء لا يمرّر الختم**: غاب %s", want)
		}
	}
	// **ويسأل الأثرَ بعد بنائه** — فلا يخرج أجوفُ الهويّة.
	if !strings.Contains(sh, "verify_identity") {
		t.Error("**البناءُ لا يسأل الأثرَ عن هويّته** — ويُقبَل ما لا يقول ما هو.")
	}
	if !strings.Contains(sh, "-identity") {
		t.Error("**لا نداءَ لبابِ الهويّة في البناء.**")
	}
	// **ويعمل بلا مستودع** — وإلّا عاد السكربتُ العارض.
	if !strings.Contains(sh, "SRC_ROOT") {
		t.Error("**مسارُ البناء يلزمه مستودعٌ دائماً** — **والمستودعُ ليس على الخادم، " +
			"فيُكتب بديلٌ عارضٌ ينسى الختم.** (وهو ما وقع ٢٠٢٦-٠٩-١٣.)")
	}
}

// TestRID3_PromotionGateIsBeforeTheSwitch **والسؤالُ قبل التبديل.**
//
// **وكشفٌ بعد الوقوع إنذارٌ لا منع** — **والإنتاجُ يكون قد صار على
// الأثر الأجوف.**
func TestRID3_PromotionGateIsBeforeTheSwitch(t *testing.T) {
	root := repoRoot(t)
	sh := ridRead(t, root, "deploy", "promote.sh")
	gate := strings.Index(sh, "-identity")
	switchAt := strings.Index(sh, "up -d --no-build")
	switch {
	case gate < 0:
		t.Error("**مسارُ الترقية لا يسأل الأثرَ عن هويّته قبل التبديل.**")
	case switchAt < 0:
		t.Error("**لم يُعثَر على موضع التبديل في مسار الترقية.**")
	case gate > switchAt:
		t.Error("**حارسُ الهويّة بعدَ التبديل** — **فيكشف العطبَ والإنتاجُ صار عليه.**")
	}
	// **ولا يُقبَل ختمٌ ناقص.**
	if !strings.Contains(sh, `"${#PREC}" -eq 40`) {
		t.Error("**الترقيةُ لا تتحقّق من طول بصمة الالتزام.**")
	}
}

// TestRID4_OneIdentityMechanismOnly **ولا آليّةَ هويّةٍ ثانية.**
//
// **وآليّتان تفترقان يومَ يُبدَّل إحداهما** — **ورقمان للحقيقة الواحدة
// أسوأُ من لا رقم.**
func TestRID4_OneIdentityMechanismOnly(t *testing.T) {
	root := repoRoot(t)
	// **ومصدرُ الختم واحد**: متغيّرا الحزمة، يُحقنان بـ`-ldflags`.
	eg := ridRead(t, root, "backend", "internal", "envguard", "envguard.go")
	if strings.Count(eg, "buildCommit") == 0 || strings.Count(eg, "buildID") == 0 {
		t.Fatal("**متغيّرا الختم غابا من `envguard`.**")
	}
	// **ولا يُقرآن من البيئة** — هويّةٌ تُبدَّل وقتَ التشغيل ليست هويّة.
	bad := regexp.MustCompile(`(?m)buildCommit\s*=\s*os\.Getenv|buildID\s*=\s*os\.Getenv`)
	if bad.MatchString(eg) {
		t.Error("**الختمُ يُقرأ من متغيّر بيئة** — **ويُبدَّل بعد النشر، فليس هويّة.**")
	}
	// **والويبُ لا يخترع بديلاً** — يعرض ما يقوله المحرّك.
	page := ridRead(t, root, "web", "apps", "rahalgo", "src", "app", "dashboard", "ops", "page.tsx")
	for _, forged := range []string{`source_commit ?? "`, `build_id ?? "`, "process.env.NEXT_PUBLIC_COMMIT"} {
		if strings.Contains(page, forged) {
			t.Errorf("**الويبُ يُلفّق هويّةً بديلة**: %s — **ويعرض ما ليس في الأثر.**", forged)
		}
	}
}

// TestRID5_BothEndpointsExposeIdentity **والبابان يعرضان ما في الأثر.**
//
// **وتُقاس القيمةُ من `envguard` لا من نصٍّ مكتوب** — **فالاختبارُ
// يمشي مختوماً وغيرَ مختوم، ويقيس التطابقَ لا الامتلاء.**
func TestRID5_BothEndpointsExposeIdentity(t *testing.T) {
	hh := New(t)
	wantCommit, wantBuild := envguard.BuildInfo()

	pub := hh.GET("/api/v1/public/identity", "")
	if pub.Code != http.StatusOK {
		t.Fatalf("`/public/identity` ⇒ %d", pub.Code)
	}
	body := string(pub.Body)
	for _, k := range []string{"source_commit", "build_id", "environment", "migration_version"} {
		if !strings.Contains(body, `"`+k+`"`) {
			t.Errorf("**حقلٌ غائبٌ من هويّة العلن**: %s", k)
		}
	}
	if wantCommit != "" && !strings.Contains(body, wantCommit) {
		t.Errorf("**هويّةُ العلن لا تقول ختمَ الأثر**: المنتظَر %s", wantCommit)
	}

	// **والمالكُ لا يملك `observability.read`** — قرارُ أقلِّ صلاحيّةٍ
	// قائم، **ولا يُنقَض لأجل اختبار.** فيُصنَع دورٌ بقدرته وحدَها.
	capRole(t, hh, "rid_obs", authz.ObservabilityRead)
	_, tok := roleUser(t, hh, "rid_obs")
	h := hh.GET("/api/v1/admin/ops/health", tok)
	if h.Code != http.StatusOK {
		t.Fatalf("`/admin/ops/health` ⇒ %d", h.Code)
	}
	hb := string(h.Body)
	for _, k := range []string{"source_commit", "build_id"} {
		if !strings.Contains(hb, `"`+k+`"`) {
			t.Errorf("**حقلٌ غائبٌ من صحّة التشغيل**: runtime.%s", k)
		}
	}
	if wantBuild != "" && !strings.Contains(hb, wantBuild) {
		t.Errorf("**صحّةُ التشغيل لا تقول ختمَ البناء**: المنتظَر %s", wantBuild)
	}
	t.Logf("✓ البابان يعرضان ختمَ الأثر (commit=%q build=%q)", wantCommit, wantBuild)
}

// TestRID6_IdentityFlagFailsClosed **وبابُ السؤال يسقط مغلقاً.**
//
// **ولا يُقاس بتشغيل الثنائيّة** — **بل بعقدها في الشيفرة**: رمزُ
// خروجٍ ثالثٌ للأجوف، وطباعةُ الحقلين.
func TestRID6_IdentityFlagFailsClosed(t *testing.T) {
	root := repoRoot(t)
	main := ridRead(t, root, "backend", "cmd", "api", "main.go")
	for _, want := range []string{
		`os.Args[1] == "-identity"`,
		"envguard.BuildInfo()",
		`"source_commit": commit`,
		`"build_id": id`,
		"os.Exit(3)",
	} {
		if !strings.Contains(main, want) {
			t.Errorf("**عقدُ بابِ الهويّة ناقص**: غاب %q", want)
		}
	}
	// **والسؤالُ قبل الإقلاع** — يعمل في صورةٍ بلا قاعدةٍ ولا شبكة.
	flagAt := strings.Index(main, `"-identity"`)
	runAt := strings.Index(main, "if err := run(logger)")
	if flagAt < 0 || runAt < 0 || flagAt > runAt {
		t.Error("**بابُ الهويّة بعد الإقلاع** — **فلا يُسأل أثرٌ بلا قاعدةٍ ولا شبكة.**")
	}
}
