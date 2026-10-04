package authzdoc

// ══════════════════════════════════════════════════════════════════════
// **صلاحيّاتُ الواجهة من مصدر المحرّك** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// **كانت أزرارُ ملفّ الحساب تفحص قدرةً والمحرّكُ يطلب أخرى** — دورُ «الثقة
// والسلامة» لا يرى زرَّ الإيقاف وهو صاحبُه، ودورٌ معه «إدارة الأدوار» يراه
// ويُردّ. **فالواجهةُ تقرأ جدولَ المحرّك نفسَه**: يُولَّد ملفٌّ في الويب من
// `authz.Rules()`، **ويُسقط البناءَ إن شاخ** (`TestWebPolicyIsCurrent`).

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// WebPolicyPath موضعُ الملفّ المولَّد في الويب — من مجلّد الحزمة.
const WebPolicyPath = "../../../web/apps/rahalgo/src/lib/adminPolicy.gen.ts"

// WebPolicyTS نصُّ الملفّ — قاعدةٌ لكلّ سطرٍ في جدول السياسة، مرتّبةٌ ثابتة.
func WebPolicyTS() string {
	type row struct{ method, pattern, cap string }
	var rows []row
	for _, r := range authz.Rules() {
		rows = append(rows, row{r.Method, r.Pattern, string(r.Need)})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].pattern != rows[j].pattern {
			return rows[i].pattern < rows[j].pattern
		}
		return rows[i].method < rows[j].method
	})
	var b strings.Builder
	b.WriteString("// مولَّدٌ من `backend/internal/authz/policy.go` — لا يُحرَّر باليد.\n")
	b.WriteString("// cd backend && go run ./cmd/authzdoc\n")
	b.WriteString("export const ADMIN_POLICY: ReadonlyArray<readonly [string, string, string]> = [\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "  [%q, %q, %q],\n", r.method, r.pattern, r.cap)
	}
	b.WriteString("];\n")
	return b.String()
}

// WriteWebPolicy يكتب الملفَّ إن تغيّر.
func WriteWebPolicy(path string) error {
	out := WebPolicyTS()
	if cur, _ := os.ReadFile(path); string(cur) == out {
		return nil
	}
	return os.WriteFile(path, []byte(out), 0o644)
}
