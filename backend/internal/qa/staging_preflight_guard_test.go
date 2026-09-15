package qa

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حارسُ ما قبل المَسّ — والهدفُ يقول من هو** (`SPRE`، ٢٠٢٦-٠٩-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// # ما قِيس
//
// **و`promote.sh` يسأل الهويّةَ قبل التبديل منذ ٢٠٢٦-٠٩-١٣** — **لكنّ
// `staging/deploy.sh` كان يرفع البوّابةَ والويبَ والقاعدةَ والذاكرةَ
// قبل أن يبلغه**: **فأوّلُ مَسٍّ يسبق أوّلَ سؤال.**
//
// # ولماذا يُمشى السكربتُ ولا يُقرأ
//
// **وقراءةُ نصٍّ تُثبت أنّ السطرَ مكتوبٌ لا أنّه يمنع** — **وشرطٌ
// مقلوبٌ يمرّ القراءةَ ويسقط في الميدان.** **فيُشغَّل الحارسُ بخادمِ
// هويّةٍ يقول `production` ويُقاس أنّه وقف.**

func spreScript(t *testing.T) string {
	t.Helper()
	p := filepath.Join(repoRoot(t), "deploy", "preflight-env.sh")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("لم أجد حارسَ ما قبل المَسّ: %v", err)
	}
	return p
}

func spreIdentity(t *testing.T, env string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"environment":   env,
				"staging":       env == "staging",
				"source_commit": strings.Repeat("b", 40),
			}})
		}))
	t.Cleanup(srv.Close)
	return srv
}

func spreRun(t *testing.T, url, want string) (string, bool) {
	t.Helper()
	sh := spreScript(t)
	cmd := exec.Command("bash", sh, url, want)
	cmd.Dir = filepath.Dir(sh)
	out, err := cmd.CombinedOutput()
	return string(out), err == nil
}

// ══════════════════════════════════════════════════════════════════════
// **SPRE1 · وهدفٌ يقول «إنتاج» يوقف نشرَ التجهيز**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو الشاهدُ السالبُ السابع الذي طلبه المالك حرفاً.**
func TestSPRE1_ProductionTargetAbortsStagingDeploy(t *testing.T) {
	fake := spreIdentity(t, "production")
	out, ok := spreRun(t, fake.URL+"/api/v1/public/identity", "staging")
	t.Logf("SPRE1 المخرَج:\n%s", strings.TrimSpace(out))

	if ok {
		t.Fatal("**مضى النشرُ وهدفُه يقول إنّه الإنتاج** — " +
			"**والمَسُّ يقع قبل أن يُسأل أحد.**")
	}
	if !strings.Contains(out, "production") || !strings.Contains(out, "staging") {
		t.Error("**الرفضُ لم يُسمِّ البيئتين** — " +
			"**ومن قرأ رسالةً لا تقول ما اختلف أعاد المحاولةَ عمياء.**")
	}
	// **ولا يُقال «مرّ» في مخرَجٍ رافض.**
	if strings.Contains(out, "PREFLIGHT_TARGET_ENV=") {
		t.Error("**طبع مرورَ الحارس ثمّ رفض** — **ومخرَجٌ يناقض نفسَه يُقرأ خطأً.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **SPRE2 · وهدفٌ يقول «تجهيز» يمضي**
// ══════════════════════════════════════════════════════════════════════
//
// **وحارسٌ يرفض كلَّ شيءٍ ليس حارساً.**
func TestSPRE2_StagingTargetPasses(t *testing.T) {
	fake := spreIdentity(t, "staging")
	out, ok := spreRun(t, fake.URL+"/api/v1/public/identity", "staging")
	t.Logf("SPRE2 المخرَج:\n%s", strings.TrimSpace(out))

	if !ok {
		t.Fatalf("**رُدّ هدفٌ صحيح**: %s", strings.TrimSpace(out))
	}
	if !strings.Contains(out, "PREFLIGHT_TARGET_ENV=staging") {
		t.Errorf("**مرّ ولم يقل بمَ مرّ**: %s", strings.TrimSpace(out))
	}
}

// ══════════════════════════════════════════════════════════════════════
// **SPRE3 · والصامتُ ليس إنتاجاً — ويُمنَع بالتشدّد وحدَه**
// ══════════════════════════════════════════════════════════════════════
//
// **ومكدّسٌ لم يُقلع بعدُ لا يردّ** — **وأوّلُ نشرٍ لا سلفَ له**:
// **فمنعُ الصمت مطلقاً يمنع النشرَ الأوّلَ إلى الأبد.**
func TestSPRE3_SilentTargetIsNotProduction(t *testing.T) {
	// **وعنوانٌ لا يسمعه أحد** — **مغلقٌ بلا خادم.**
	dead := "http://127.0.0.1:9/api/v1/public/identity"

	out, ok := spreRun(t, dead, "staging")
	if !ok {
		t.Fatalf("**مُنع نشرٌ أوّلُ لأنّ المكدّسَ لم يُقلع بعد**: %s", strings.TrimSpace(out))
	}
	if !strings.Contains(out, "صامت") {
		t.Errorf("**مرّ الصمتُ بلا قول**: %s", strings.TrimSpace(out))
	}

	// **ومن أراد التشدّدَ منعه.**
	sh := spreScript(t)
	cmd := exec.Command("bash", sh, dead, "staging")
	cmd.Dir = filepath.Dir(sh)
	cmd.Env = append(os.Environ(), "STRICT=1")
	if b, err := cmd.CombinedOutput(); err == nil {
		t.Errorf("**مضى مع التشدّد وهدفُه صامت**: %s", strings.TrimSpace(string(b)))
	}
}

// ══════════════════════════════════════════════════════════════════════
// **SPRE4 · والسؤالُ قبل أوّل مَسّ في مسار التجهيز**
// ══════════════════════════════════════════════════════════════════════
//
// **وترتيبُ السطور هو العقدُ هنا** — **وحارسٌ صحيحٌ بعد `up` إنذارٌ
// لا منع.**
func TestSPRE4_GuardRunsBeforeFirstMutation(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "deploy", "staging", "deploy.sh"))
	if err != nil {
		t.Fatalf("مسارُ نشر التجهيز: %v", err)
	}
	// **والتعليقُ يذكر الخطوات ولا ينفّذها** — **فيُقرأ ما يُنفَّذ
	// وحدَه**، **وإلّا سمّى الشرحُ خطوةً قبل موضعها.**
	live := []string{}
	for _, ln := range strings.Split(string(b), "\n") {
		if trimmed := strings.TrimSpace(ln); trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		live = append(live, ln)
	}
	src := strings.Join(live, "\n")

	guard := strings.Index(src, "preflight-env.sh")
	if guard < 0 {
		t.Fatal("**لا سؤالَ قبل المَسّ في مسار نشر التجهيز.**")
	}
	// **وأوّلُ مَسٍّ للمكدّس** — **رفعُ الحاويات، وبناءُ الأثر قبله.**
	for _, mutation := range []string{"docker compose", "promote.sh", "build-artifact.sh"} {
		at := strings.Index(src, mutation)
		if at < 0 {
			t.Fatalf("**ذهبت خطوةُ %s من المسار** — **والحارسُ يحرس ما لم يعد موجوداً.**", mutation)
		}
		if at < guard {
			t.Errorf("**%s يسبق السؤالَ** — **فالمَسُّ قبل أن يُعرَف الهدف.**", mutation)
		}
	}
}
