package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleDriverOrderOutcome **ما آل إليه طلبٌ كان بيده** (٢٠٢٦-١٠-٠٢).
//
// **يسأله التطبيقُ حين يختفي طلبٌ من قائمته** — فيقول له «ألغى الزبونُ الطلب» أو
// «أعادته الإدارة إلى الطابور» **بدل أن يُغلق الملاحةَ ويقفز إلى الطلبات صامتاً.**
//
// **ومن سجلّ الأحداث لا من الدفع**: الإشعارُ قد لا يصل (هاتفٌ بلا خدمة دفع، أو
// إذنٌ مرفوض)، **والسجلُّ يبقى.** **ولا يُجاب إلّا عن طلبٍ كان بيده** — ولا شيءَ
// فيه يخصّ غيرَه: رقمٌ وحالٌ وسبب.
func (s *Server) handleDriverOrderOutcome(w http.ResponseWriter, r *http.Request) {
	out, err := s.orders.DriverOutcomeOf(r.Context(), chi.URLParam(r, "id"), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}
