package androidmap

// ══════════════════════════════════════════════════════════════════════
// **كلُّ رمزٍ يبلغ الهاتفَ له نصٌّ عربيّ** — `XG-45`
// ══════════════════════════════════════════════════════════════════════
//
// # لماذا هنا
//
// **الدليلُ كان في حارسٍ يعمل بـ`pnpm`** — **ولا يقرؤه `P-9` ولا
// `P-10`**، فبقيت الفجوةُ بلا دليلٍ مربوط. **وهذه الحزمةُ موضعُ ما
// يقوله المحرّكُ عن أندرويد** (`P-9` يقرأ منها).
//
// # والتصنيفُ مصدرُه واحد
//
// **لا تُنسَخ القائمةُ هنا**: **تُقرأ من الحارس نفسِه**
// (`web/scripts/check-app-error-codes.mjs`) — **فمن بدّل التصنيفَ
// هناك رآه هنا، ولا معجمان يفترقان.**
//
// # وما يُقاس
//
//	١ كلُّ رمزٍ مُعلَنٍ أنّه يبلغ الهاتف له مدخلٌ في خريطة أندرويد
//	٢ ولا رمزَ في `server` بلا تصنيف — يبلغ أو لا يبلغ
//
// **والثاني هو ما سقط**: **`auth_unavailable` يخرج من وسيط التوثيق**،
// **وكان الحارسُ يتخطّى `internal/server` كلَّها** لأنّ فيها أبوابَ
// الإدارة. **فقُرئ لاتينيّةً على شاشةٍ عربيّة.**

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// setOf يقرأ مجموعةَ رموزٍ مُعلَنةً في الحارس.
func setOf(t *testing.T, src, name string) map[string]bool {
	t.Helper()
	re := regexp.MustCompile(`(?s)const ` + name + ` = new Set\(\[(.*?)\]\)`)
	m := re.FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("**لم تُقرأ مجموعةُ %s من الحارس** — تبدّل شكلُه", name)
	}
	out := map[string]bool{}
	for _, c := range regexp.MustCompile(`"([a-z0-9_]+)"`).FindAllStringSubmatch(m[1], -1) {
		out[c[1]] = true
	}
	if len(out) == 0 {
		t.Fatalf("**مجموعةُ %s فارغة** — وحارسٌ يقرأ فراغاً يمرّ دائماً", name)
	}
	return out
}

func TestXG45_EveryMobileReachableCodeHasArabic(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	guard, err := os.ReadFile(filepath.Join(root, "web", "scripts", "check-app-error-codes.mjs"))
	if err != nil {
		t.Skipf("حارسُ الرموز غيرُ متاح: %v", err)
	}
	mobile := setOf(t, string(guard), "SERVER_MOBILE")
	admin := setOf(t, string(guard), "SERVER_ADMIN")

	kt, err := os.ReadFile(filepath.Join(root, "mobile", "ui", "src", "main",
		"kotlin", "com", "rahalgo", "ui", "ApiErrors.kt"))
	if err != nil {
		t.Skipf("خريطةُ أندرويد غيرُ متاحة: %v", err)
	}
	known := map[string]bool{}
	for _, m := range regexp.MustCompile(`"([a-z0-9_]+)" to R\.string\.`).
		FindAllStringSubmatch(string(kt), -1) {
		known[m[1]] = true
	}
	if len(known) < 50 {
		t.Fatalf("**قُرئ %d رمزاً من خريطة أندرويد** — والقراءةُ ناقصة", len(known))
	}

	// ── ١ ── ما يبلغ الهاتفَ له نصّ ────────────────────────────────
	var mute []string
	for code := range mobile {
		if !known[code] {
			mute = append(mute, code)
		}
	}
	sort.Strings(mute)
	for _, c := range mute {
		t.Errorf("**رمزٌ يبلغ الهاتفَ بلا نصٍّ عربيّ**: `%s` — "+
			"**يُعرض خامّاً على شاشةٍ عربيّة.** (`XG-45`)", c)
	}

	// ── ٢ ── ولا رمزَ في `server` بلا تصنيف ────────────────────────
	//
	// **و`server` ليست كلُّها إدارة** — **وهذا هو ما أخطأه الحارسُ
	// الأوّل.**
	dir := filepath.Join(root, "backend", "internal", "server")
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("قراءةُ حزمة الخادم: %v", err)
	}
	born := regexp.MustCompile(`(?s)httpx\.NewError\(.*?"([a-z0-9_]+)"\s*,\s*"[a-z0-9_.]+"`)
	var unclassified []string
	for _, e := range ents {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			t.Fatalf("قراءةُ %s: %v", n, err)
		}
		for _, m := range born.FindAllStringSubmatch(string(b), -1) {
			c := m[1]
			if !mobile[c] && !admin[c] && !known[c] {
				unclassified = append(unclassified, c+" ("+n+")")
			}
		}
	}
	sort.Strings(unclassified)
	seen := map[string]bool{}
	for _, u := range unclassified {
		if seen[u] {
			continue
		}
		seen[u] = true
		t.Errorf("**رمزٌ في `server` بلا تصنيف**: %s — "+
			"**أيبلغ الهاتفَ أم للإدارة وحدَها؟** (`XG-45`)", u)
	}
	t.Logf("XG-45: يبلغ الهاتفَ=%d · للإدارة=%d · خريطةُ أندرويد=%d · بلا تصنيف=%d",
		len(mobile), len(admin), len(known), len(seen))
}
