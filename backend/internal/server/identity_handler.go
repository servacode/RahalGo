package server

// ══════════════════════════════════════════════════════════════════════
// **هويّةُ البيئة — `P-0` البند ٥**
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا بابٌ لا سطرٌ في سجلّ
//
// **من يفتح لوحةً لا يقرأ سجلَّ الخادم** — **ومن جرّب على «التجهيز»
// وهو الإنتاج لا يكتشف ذلك من سطرٍ طُبع عند الإقلاع قبل يومين.**
//
// **والبوّابةُ (`P-10`) تحتاجه أيضاً**: **مرشَّحٌ لا يقول أيَّ التزامٍ
// يشغّل ليس مرشَّحاً** (البند ٣٦).
//
// # ولا سرَّ فيه
//
// **لا وصلةَ قاعدةٍ ولا مفتاحَ توقيعٍ ولا هاتفَ أدمن** — **اسمُ بيئةٍ
// والتزامٌ وهجرة.** **وبابٌ يقول «أنا التجهيز» لا يفتح باباً لأحد.**

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/envguard"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleIdentity يقول ما هذه البيئةُ وأيَّ شيفرةٍ تشغّل.
func (s *Server) handleIdentity(w http.ResponseWriter, r *http.Request) {
	// **والهجرةُ تُقرأ من القاعدة لا من ملفٍّ على القرص** — **وملفٌّ
	// موجودٌ لا يعني أنّه طُبِّق.**
	var migration string
	_ = s.pg.QueryRow(r.Context(),
		`SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`).
		Scan(&migration)

	id := envguard.IdentityOf(s.cfg.Env, migration)
	// **ويُمنَع التخزينُ الوسيط** — **هويّةٌ مخزَّنةٌ تقول عن بناءٍ مضى.**
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, id)
}
