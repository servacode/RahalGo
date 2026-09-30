package deployguard

// حارسُ سياسةِ استبقاء أرشيفات الإصدار — `deploy/retention.sh`.
//
// ══════════════════════════════════════════════════════════════════════
//  لماذا اختبارٌ لسكربتِ نشر
// ══════════════════════════════════════════════════════════════════════
//
// **السياسةُ التي سبقته كانت مكتوبةً ولا تعمل**: تُنقّي
// `$STAGING_ROOT/artifacts` والأرشيفاتُ تُكتب في `/srv/rahalgo/artifacts`
// لأنّ `ARTIFACT_DIR` أُسند بلا `export`. **فبقيت حمراءَ بلا أن يُنبّه أحد**
// حتّى امتلأ القرصُ (٢٠٢٦-٠٩-٣٠) فسقطت قاعدةُ التجهيز.
//
// **ومنطقُ حذفٍ بلا اختبارٍ منطقُ حذفٍ لا يُعرَف ما يحذف.**
//
// ══════════════════════════════════════════════════════════════════════
//  ويُختبَر على مجلّدٍ مؤقّتٍ لا على الخادم
// ══════════════════════════════════════════════════════════════════════
//
// **`plan` تطبع ولا تحذف** — فتُقاس نيّتُها قبل أن تُنفَّذ، **ولا يقترب
// الاختبارُ من إصدارٍ عامل.**

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// scriptPath **يصعد إلى جذر المستودع** — والاختبارُ يعمل من مجلّد حزمته.
func scriptPath(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("مجلّدُ العمل: %v", err)
	}
	for i := 0; i < 8; i++ {
		p := filepath.Join(dir, "deploy", "retention.sh")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("لم يُعثر على deploy/retention.sh بالصعود من مجلّد الاختبار")
	return ""
}

func bashPath(t *testing.T) string {
	t.Helper()
	if p, err := exec.LookPath("bash"); err == nil {
		return p
	}
	// **وويندوز يحمل bash مع Git** — يُبحَث في موضعه المعتاد.
	if runtime.GOOS == "windows" {
		for _, p := range []string{
			`C:\Program Files\Git\bin\bash.exe`,
			`C:\Program Files (x86)\Git\bin\bash.exe`,
		} {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	t.Skip("لا bash على هذا الجهاز — والحارسُ يحتاجه")
	return ""
}

// unixPath **يحوّل مسارَ ويندوز إلى ما يفهمه bash** — `C:\x` ⇒ `/c/x`.
func unixPath(p string) string {
	if runtime.GOOS != "windows" {
		return p
	}
	p = filepath.ToSlash(p)
	if len(p) > 2 && p[1] == ':' {
		return "/" + strings.ToLower(p[:1]) + p[2:]
	}
	return p
}

// release **يكتب ملفّي إصدارٍ بوقتٍ محدّد** — الأرشيفُ وبيانُه.
func release(t *testing.T, dir, token string, age time.Duration) []string {
	t.Helper()
	var out []string
	for _, name := range []string{
		"rahalgo-api-release-" + token + ".tar",
		"release-api-" + token + ".json",
	} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatalf("كتابةُ %s: %v", name, err)
		}
		when := time.Now().Add(-age)
		if err := os.Chtimes(p, when, when); err != nil {
			t.Fatalf("ضبطُ وقتِ %s: %v", name, err)
		}
		out = append(out, name)
	}
	return out
}

func runPlan(t *testing.T, dir string, keep string, protected ...string) ([]string, error) {
	t.Helper()
	args := append([]string{unixPath(scriptPath(t)), "plan", unixPath(dir), keep}, protected...)
	cmd := exec.Command(bashPath(t), args...)
	var errb strings.Builder
	cmd.Stderr = &errb
	out, err := cmd.Output()
	if err != nil {
		return nil, &planError{msg: errb.String(), err: err}
	}
	var got []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			got = append(got, filepath.Base(line))
		}
	}
	sort.Strings(got)
	return got, nil
}

type planError struct {
	msg string
	err error
}

func (e *planError) Error() string { return e.msg + " | " + e.err.Error() }

// ── أ · يُحتفظ بآخر خمسةٍ ويُحذف ما قبلها ─────────────────────────────
func TestRetention_KeepsNewestFive(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// ثمانيةُ إصداراتٍ، الأحدثُ أوّلاً في العُمر
	tokens := []string{"aaaaaaa1", "bbbbbbb2", "ccccccc3", "ddddddd4",
		"eeeeeee5", "fffffff6", "ggggggg7", "hhhhhhh8"}
	for i, tok := range tokens {
		release(t, dir, tok, time.Duration(i+1)*time.Hour)
	}

	got, err := runPlan(t, dir, "5")
	if err != nil {
		t.Fatalf("plan سقطت: %v", err)
	}
	// الأحدثُ خمسةٌ هي الخمسةُ الأولى في القائمة (أقلُّها عُمراً)
	want := []string{
		"rahalgo-api-release-fffffff6.tar", "release-api-fffffff6.json",
		"rahalgo-api-release-ggggggg7.tar", "release-api-ggggggg7.json",
		"rahalgo-api-release-hhhhhhh8.tar", "release-api-hhhhhhh8.json",
	}
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("خطّةُ الحذف خاطئة\nوقع:    %v\nمنتظَر: %v", got, want)
	}
}

// ── ب · والمحميُّ لا يُحذف ولو كان أقدمَ الكلّ ────────────────────────
//
// **وهذا بيتُ القصيد**: إصدارٌ عاملٌ شاخ تاريخُه **يُحذف أرشيفُه فيسقط
// الرجوعُ إليه** — وهو ما يمنعه هذا البند.
func TestRetention_NeverDeletesProtected(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i, tok := range []string{"new00001", "new00002", "new00003", "old99999"} {
		release(t, dir, tok, time.Duration(i+1)*time.Hour)
	}
	// الاستبقاءُ واحدٌ فقط — فكلُّ ما عدا الأحدثَ مرشَّحٌ للحذف
	got, err := runPlan(t, dir, "1", "old99999")
	if err != nil {
		t.Fatalf("plan سقطت: %v", err)
	}
	for _, f := range got {
		if strings.Contains(f, "old99999") {
			t.Fatalf("خُطِّط حذفُ إصدارٍ محميّ: %s — **وأرشيفُ العامل لا يُحذف**", f)
		}
	}
	if len(got) == 0 {
		t.Fatal("لم يُخطَّط حذفُ شيءٍ مع استبقاءٍ واحدٍ وأربعةِ إصدارات — الاختبارُ لا يقيس شيئاً")
	}
}

// ── ج · وما ليس أرشيفَ إصدارٍ لا يُمسّ ────────────────────────────────
//
// **ونسخُ القاعدةِ خصوصاً** — لها سياستُها.
func TestRetention_IgnoresForeignFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i, tok := range []string{"k0000001", "k0000002"} {
		release(t, dir, tok, time.Duration(i+1)*time.Hour)
	}
	foreign := []string{
		"staging-before-wipe-20260929T212622Z.dump", // نسخةُ قاعدة
		"rahalgo.sql.gz",
		"notes.txt",
		"rahalgo-api-release-.tar.partial",
	}
	for _, name := range foreign {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := runPlan(t, dir, "1")
	if err != nil {
		t.Fatalf("plan سقطت: %v", err)
	}
	for _, f := range got {
		for _, bad := range foreign {
			if f == bad {
				t.Fatalf("خُطِّط حذفُ ملفٍّ ليس أرشيفَ إصدار: %s", f)
			}
		}
	}
}

// ── د · ويرفض مجلّدَ نسخٍ أو مساراً نسبيّاً ───────────────────────────
//
// **والرفضُ قبل القراءة** — فلا يُحذف شيءٌ في مجلّدٍ لم يُصرَّح به.
func TestRetention_RefusesUnsafeDirs(t *testing.T) {
	base := t.TempDir()
	cases := map[string]string{
		"مجلّدُ نسخٍ اسمُه backups": filepath.Join(base, "backups"),
		"اسمُه ليس artifacts":       filepath.Join(base, "releases"),
		"مسارُ pgdata":              filepath.Join(base, "pgdata", "artifacts"),
	}
	for name, dir := range cases {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := runPlan(t, dir, "5"); err == nil {
			t.Fatalf("%s: قُبل وكان يجب أن يُرفض (%s)", name, dir)
		}
	}
	// ومسارٌ نسبيٌّ يُرفض كذلك
	if _, err := runPlan(t, "artifacts", "5"); err == nil {
		t.Fatal("مسارٌ نسبيٌّ قُبل — والرفضُ شرطُ الأمان")
	}
}

// ── هـ · وعددُ استبقاءٍ غيرُ صالحٍ يُرفض ──────────────────────────────
func TestRetention_RefusesBadKeepCount(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	release(t, dir, "z0000001", time.Hour)
	for _, bad := range []string{"0", "-1", "خمسة", ""} {
		if _, err := runPlan(t, dir, bad); err == nil {
			t.Fatalf("عددُ استبقاءٍ %q قُبل — والصفرُ يعني حذفَ الكلّ", bad)
		}
	}
}

// ── و · والغلافُ يُصدّر ARTIFACT_DIR — وإلّا كتب في مجلّد الإنتاج ─────
//
// **هذا هو العطبُ الذي أسقط القرص** (٢٠٢٦-٠٩-٣٠): الإسنادُ بلا تصدير،
// **والابنةُ لا ترث ما لم يُصدَّر**، فتهبط إلى `/srv/rahalgo/artifacts`.
func TestDeployWrapper_ExportsArtifactDir(t *testing.T) {
	root := filepath.Dir(filepath.Dir(scriptPath(t)))
	src, err := os.ReadFile(filepath.Join(root, "deploy", "staging", "rahalgo-staging-deploy.sh"))
	if err != nil {
		t.Fatalf("قراءةُ غلاف النشر: %v", err)
	}
	if !strings.Contains(string(src), "export ARTIFACT_DIR") {
		t.Fatal("غلافُ النشر لا يُصدّر ARTIFACT_DIR — **فأرشيفاتُ التجهيز تُكتب في مجلّد الإنتاج " +
			"ولا تُنقّى أبداً**، وهو سببُ امتلاء القرص ٢٠٢٦-٠٩-٣٠")
	}
	if !strings.Contains(string(src), "retention.sh") {
		t.Fatal("غلافُ النشر لا ينادي deploy/retention.sh — **وسياسةٌ لا يناديها أحدٌ سياسةٌ لا وجودَ لها**")
	}
}
