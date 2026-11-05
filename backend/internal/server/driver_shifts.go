package server

// ══════════════════════════════════════════════════════════════════════
// **سجلّ الدوام** (قرار المالك ٢٠٢٦-١١-٠٥) — كل فتح وسكّر صفّ، أيّاً كان من سكّره
// ══════════════════════════════════════════════════════════════════════
//
// الكتابة كلها من `internal/shifts` (والحارس `TestShifts_NoDirectOnShiftWrites` يمنع غيرها).
// وهون ما يحتاجه الخادم حولها: نسخة التطبيق من ترويسته.

import (
	"net/http"
	"strconv"
	"strings"
)

// appBuild **نسخةُ تطبيق السائق من ترويسته** (`X-RahalGo-Version`) — وصفرٌ لغيره:
// الويبُ والاختبارات وأيُّ تطبيقٍ غيرُ السائق. **والصفرُ يُعامَل نسخةً قديمة.**
func appBuild(r *http.Request) int {
	if !strings.HasSuffix(r.Header.Get(clientKindHeader), "-driver") {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(r.Header.Get(versionHeader)))
	if err != nil || n < 0 {
		return 0
	}
	return n
}
