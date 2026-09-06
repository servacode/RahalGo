package gate_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **البوّابةُ لا تُوسّخ ما تقيسه**
// ══════════════════════════════════════════════════════════════════════
//
// # العلّة
//
// **كانت `RELEASE_GATE.json` تُكتب داخلَ ما يُتتبَّع في كلّ تشغيل**،
// وتحمل `run_id` عشوائيّاً و`generated_at` من الساعة. **فمرشَّحُ إصدارٍ
// نظيفٌ يصير `dirty` بأثرٍ ولّدته البوّابةُ نفسُها** — **وهي تقيس
// النظافةَ وتنقضها في النَّفَس ذاته.**
//
// **ولا يصلح «لا تكتب إن لم يتبدّل المحتوى»**: المحتوى يتبدّل حتماً.
//
// # وكيف يُثبَت
//
// **تُقارَن حالُ الشجرة قبلَ وبعد** — **لا يُشترَط أن تكون نظيفةً
// ابتداءً.** **ومن اشترط ذلك كتب حارساً لا يعمل إلّا على جهازٍ فارغ**،
// **ولا يُشغَّل فلا يحرس شيئاً.** المهمُّ ألّا تُضيف البوّابةُ ولا تحذف.
func TestGateRunLeavesTreeUnchanged(t *testing.T) {
	root := repoRoot(t)
	before := gitStatus(t, root)

	cmd := exec.Command("go", "run", "./cmd/releasegate")
	cmd.Dir = filepath.Join(root, "backend")
	out, err := cmd.CombinedOutput()
	// **والخروجُ ١ يعني «غيرُ جاهزٍ للإنتاج» لا عطباً** — والعطبُ ٢.
	if err != nil && !strings.Contains(string(out), "READY FOR PRODUCTION") {
		t.Fatalf("تعذّر تشغيلُ البوّابة: %v\n%s", err, out)
	}

	after := gitStatus(t, root)
	if added := diff(after, before); len(added) > 0 {
		t.Errorf("**البوّابةُ وسّخت الشجرة** — أضافت %v.\n"+
			"**ومرشَّحٌ نظيفٌ لا يصير متّسخاً بأثرٍ ولّدته هي.**", added)
	}
	if gone := diff(before, after); len(gone) > 0 {
		t.Errorf("**البوّابةُ محت أثراً قائماً**: %v", gone)
	}
	t.Logf("قبلَ التشغيل %d بنداً · بعدَه %d · ولا فرق", len(before), len(after))
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("المجلّدُ الحاليّ: %v", err)
	}
	// internal/gate ⇒ backend ⇒ الجذر.
	return filepath.Dir(filepath.Dir(filepath.Dir(wd)))
}

func gitStatus(t *testing.T, root string) map[string]bool {
	t.Helper()
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("لا مستودعَ git هنا: %v", err)
	}
	set := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		if s := strings.TrimSpace(line); s != "" {
			set[s] = true
		}
	}
	return set
}

func diff(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
