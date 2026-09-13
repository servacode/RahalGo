package qa

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حارسُ بيئةِ الترقية — ولا يُبدَّل الإنتاجُ بشاهدِ تجهيز** (`ENVG`)
// ══════════════════════════════════════════════════════════════════════
//
// # ما وقع (٢٠٢٦-٠٩-١٣)
//
// **رُقّي الإنتاجُ ورابطُ التحقّق `http://localhost:8080` — وهو منفذُ
// التجهيز على الخادم نفسِه.** **فطبعت البوّابةُ `"environment":"staging"`
// شاهداً على ترقيةِ إنتاج.**
//
// **والإنتاجُ كان صحيحاً** — أُثبت منفصلاً بالنطاق العامّ وبمعرّف
// الصورة العاملة، **فقُبلت الدفعة.** **لكنّ البوّابةَ لم تمنع، وما لا
// يمنع يقع مرّةً ثانية.**
//
// # وما يُحرَس
//
//	TARGET_ENV مطلوبةٌ — **ولا ترقيةَ بلا بيئةٍ مقصودةٍ معلَنة**
//	ردُّ الرابطِ قبل التبديل يجب أن يقول البيئةَ عينَها
//	وتفاوتٌ ⇒ **وقوفٌ قبل `up`** لا تحذيرٌ بعده
//	وبعد النشر تُطابَق ثانياً
//
// # ولماذا يُمشى السكربتُ ولا يُقرأ
//
// **وقراءةُ نصٍّ تُثبت أنّ السطرَ مكتوبٌ لا أنّه يمنع** — **وشرطٌ
// مقلوبٌ يمرّ القراءةَ ويسقط في الميدان.** **فيُشغَّل `promote.sh`
// بخادمِ هويّةٍ مزيّفٍ يقول `staging`** — **ويُقاس أنّه وقف قبل أن
// يلمس `docker`.**

// envgScript **مسارُ مسار الترقية في المستودع.**
func envgScript(t *testing.T) string {
	t.Helper()
	p := filepath.Join(repoRoot(t), "deploy", "promote.sh")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("لم أجد مسارَ الترقية: %v", err)
	}
	return p
}

// envgFakeIdentity خادمُ هويّةٍ يقول بيئةً بعينها.
func envgFakeIdentity(t *testing.T, env string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			body := map[string]any{"data": map[string]any{
				"environment":   env,
				"source_commit": strings.Repeat("a", 40),
				"build_id":      "envg-fake",
			}}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(body)
		}))
	t.Cleanup(srv.Close)
	return srv
}

// envgRun يُشغّل مسارَ الترقية ويعيد مخرجَه ونجاحَه.
//
// **ومرجعُ الأثر غيرُ موجودٍ في مخزن الصور عمداً** — **فلو تجاوز
// الحارسَ لَسقط في الخطوة التي بعده**، **والفرقُ بين الرسالتين هو
// الدليل**: من وقف عند البيئة لم يبلغ مخزنَ الصور.
func envgRun(t *testing.T, targetEnv, health string) (string, bool) {
	t.Helper()
	sh := envgScript(t)
	cmd := exec.Command("bash", sh, "compose.yml", ".env",
		"rahalgo-api:envg-absent-"+uniq("x"), health)
	cmd.Dir = filepath.Dir(sh)
	cmd.Env = append(os.Environ(), "TARGET_ENV="+targetEnv)
	out, err := cmd.CombinedOutput()
	return string(out), err == nil
}

// ══════════════════════════════════════════════════════════════════════
// **ENVG1 · ترقيةُ إنتاجٍ بشاهدِ تجهيزٍ تُرفَض قبل التبديل**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو الشاهدُ السالبُ الذي طلبه المالك حرفاً.**
func TestENVG1_ProductionRefusesStagingIdentity(t *testing.T) {
	fake := envgFakeIdentity(t, "staging")
	out, ok := envgRun(t, "production", fake.URL+"/api/v1/public/identity")
	t.Logf("ENVG1 المخرَج:\n%s", envgTrim(out))

	if ok {
		t.Fatal("**مضى مسارُ الترقية ورابطُ التحقّق يخصُّ التجهيز** — " +
			"**والإنتاجُ يُبدَّل بشاهدٍ من بيئةٍ أخرى.**")
	}
	if !strings.Contains(out, "staging") || !strings.Contains(out, "production") {
		t.Error("ENVG1 **الرفضُ لم يُسمِّ البيئتين** — " +
			"**ومن قرأ رسالةً لا تقول ما اختلف أعاد المحاولةَ عمياء.**")
	}
	// **والوقوفُ قبل التبديل** — **ولم يبلغ مخزنَ الصور ولا `docker`.**
	for _, late := range []string{"RUNNING_IMAGE_ID", "PREFLIGHT_IMAGE_ID"} {
		if strings.Contains(out, late) {
			t.Errorf("ENVG1 **بلغ %s** — **فالحارسُ بعد خطوةٍ لا قبلَها.**", late)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ENVG2 · وبيئةٌ مطابقةٌ تمضي إلى ما بعد الحارس**
// ══════════════════════════════════════════════════════════════════════
//
// **وحارسٌ يرفض كلَّ شيءٍ ليس حارساً** — **فيُثبَت أنّه يُمضي المطابقَ.**
//
// **والمضيُّ يُقاس بما بعده**: **الأثرُ المطلوبُ غيرُ موجودٍ في المخزن**،
// **فالسقوطُ يجب أن يكون عليه لا على البيئة.**
func TestENVG2_MatchingEnvironmentPassesTheGate(t *testing.T) {
	fake := envgFakeIdentity(t, "production")
	out, ok := envgRun(t, "production", fake.URL+"/api/v1/public/identity")
	t.Logf("ENVG2 المخرَج:\n%s", envgTrim(out))

	if ok {
		t.Fatal("ENVG2 **مضى إلى النهاية وأثرُه غيرُ موجود** — **والمقدّمةُ مكسورة.**")
	}
	if !strings.Contains(out, "PREFLIGHT_TARGET_ENV=production") {
		t.Error("ENVG2 **البيئةُ المطابقةُ لم تمرّ الحارسَ** — " +
			"**فالحارسُ يرفض ما يجب أن يُمضيه.**")
	}
	// **والسقوطُ على الأثر لا على البيئة.**
	if !strings.Contains(out, "مخزن الصور") {
		t.Error("ENVG2 **لم يبلغ فحصَ وجودِ الأثر** — **فوقف على غير سببه.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ENVG3 · ولا ترقيةَ بلا بيئةٍ معلَنة**
// ══════════════════════════════════════════════════════════════════════
//
// **ومن نسيها لا يُترَك يُخمِّن** — **والصمتُ يعني «أيَّ بيئةٍ كانت».**
func TestENVG3_MissingTargetEnvRefuses(t *testing.T) {
	sh := envgScript(t)
	cmd := exec.Command("bash", sh, "compose.yml", ".env",
		"rahalgo-api:envg-absent", "http://127.0.0.1:1/identity")
	cmd.Dir = filepath.Dir(sh)
	// **وبيئةٌ نظيفةٌ بلا `TARGET_ENV`** — ولا تُورَث من المِسنَد.
	clean := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "TARGET_ENV=") {
			clean = append(clean, kv)
		}
	}
	cmd.Env = clean
	out, err := cmd.CombinedOutput()
	t.Logf("ENVG3 المخرَج:\n%s", envgTrim(string(out)))
	if err == nil {
		t.Fatal("ENVG3 **مضى بلا بيئةٍ مقصودة.**")
	}
	if !strings.Contains(string(out), "TARGET_ENV") {
		t.Error("ENVG3 **الرفضُ لم يُسمِّ ما ينقص.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ENVG4 · والمنادونَ كلُّهم يقولون بيئتَهم**
// ══════════════════════════════════════════════════════════════════════
//
// **وحارسٌ لا يُنادى لا يحرس** — **فمن نسخ سطرَ ترقيةٍ بلا `TARGET_ENV`
// سقط عليه هذا الفحصُ لا الميدان.**
func TestENVG4_EveryCallerDeclaresItsEnvironment(t *testing.T) {
	root := repoRoot(t)
	dir := filepath.Join(root, "deploy")
	var bad []string
	err := filepath.Walk(dir, func(p string, fi os.FileInfo, e error) error {
		if e != nil || fi.IsDir() || !strings.HasSuffix(p, ".sh") {
			return nil
		}
		if filepath.Base(p) == "promote.sh" {
			return nil
		}
		b, e2 := os.ReadFile(p)
		if e2 != nil {
			return nil
		}
		for _, line := range strings.Split(string(b), "\n") {
			if !strings.Contains(line, "promote.sh") {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
			if !strings.Contains(line, "TARGET_ENV=") {
				bad = append(bad, fmt.Sprintf("%s: %s",
					filepath.Base(p), strings.TrimSpace(line)))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("مشيُ المجلَّد: %v", err)
	}
	for _, b := range bad {
		t.Errorf("ENVG4 **نداءُ ترقيةٍ بلا بيئةٍ معلَنة** — %s", b)
	}
}

// envgTrim يقصّ المخرَجَ الطويلَ في السجلّ.
func envgTrim(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > 12 {
		lines = lines[len(lines)-12:]
	}
	return strings.Join(lines, "\n")
}
