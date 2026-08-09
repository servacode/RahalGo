package server

// إنذاراتُ المتجر — يراها صاحبُها، وتُصدرها العمليات.

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type warningRow struct {
	ID          string    `json:"id"`
	Reason      string    `json:"reason"`
	Note        string    `json:"note"`
	OrderNumber *int64    `json:"order_number"`
	CreatedAt   time.Time `json:"created_at"`
}

// warningsSelect **إنذاراتُ صاحب المتجر — من الجدول الموحَّد.**
//
// **وتُطلَب بمعرّف المتجر ويُقرأ صاحبُه** — الشاشةُ تعرف متجرَها لا حسابَه،
// **والإنذارُ على الحساب.** (وحّد الجدولَين ٢٠٢٦-٠٨-٠٩.)
const warningsSelect = `
	SELECT w.id, w.reason, w.note, o.number, w.created_at
	FROM warnings w
	LEFT JOIN orders o ON o.id = w.order_id
	WHERE w.user_id = (SELECT owner_user_id FROM merchants WHERE id = $1)
	ORDER BY w.created_at DESC LIMIT 100`

func (s *Server) scanWarnings(w http.ResponseWriter, r *http.Request, merchantID string) {
	rows, err := s.pg.Query(r.Context(), warningsSelect, merchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []warningRow{}
	for rows.Next() {
		var x warningRow
		if err := rows.Scan(&x.ID, &x.Reason, &x.Note, &x.OrderNumber, &x.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"warnings": out})
}

// handleMerchantWarnings ما أُنذر به صاحبُ المتجر.
//
// **ومن أُنذر يرى إنذارَه.** إشعارٌ يمرّ في الشريط يُقرأ مرّةً ويُنسى، **ثمّ
// يُحظر المتجرُ ولم يكن يعلم أنّ عليه شيئاً** — والحظرُ الذي يفاجئ يُفقدنا
// متجراً كان يصحّح لو عرف.
func (s *Server) handleMerchantWarnings(w http.ResponseWriter, r *http.Request) {
	var merchantID string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT id FROM merchants WHERE owner_user_id = $1 LIMIT 1`,
		userIDFrom(r)).Scan(&merchantID); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.scanWarnings(w, r, merchantID)
}

// handleAdminMerchantWarnings الإنذاراتُ كما تراها العمليات.
func (s *Server) handleAdminMerchantWarnings(w http.ResponseWriter, r *http.Request) {
	s.scanWarnings(w, r, chi.URLParam(r, "id"))
}
