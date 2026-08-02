package server

// إنذاراتُ المتجر — يراها صاحبُها، وتُصدرها العمليات.

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

type warningRow struct {
	ID          string    `json:"id"`
	Reason      string    `json:"reason"`
	Note        string    `json:"note"`
	OrderNumber *int64    `json:"order_number"`
	CreatedAt   time.Time `json:"created_at"`
}

const warningsSelect = `
	SELECT w.id, w.reason, w.note, o.number, w.created_at
	FROM merchant_warnings w
	LEFT JOIN orders o ON o.id = w.order_id
	WHERE w.merchant_id = $1
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

// handleIssueWarning إنذارٌ يدويٌّ بلا طلبٍ يشهد عليه.
//
// **ولا كلَّ إنذارٍ يُشتقّ من طلب**: متجرٌ رفع أسعارَه عن المتّفق، أو أساء إلى
// سائق، أو تكرّر تأخيرُه بلا فشلٍ مسجَّل. **ومن لا مكانَ لإنذاره لا يُنذَر
// إلّا بمكالمةٍ لا أثر لها** — فلا تُعدّ، ولا يُحتجّ بها يوم الحظر.
func (s *Server) handleIssueWarning(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Reason string `json:"reason"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والسببُ إلزاميّ**: إنذارٌ بلا سبب لا يُصحَّح ولا يُحتجّ به.
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		s.respondErr(w, errValidation)
		return
	}
	merchantID := chi.URLParam(r, "id")
	actor := userIDFrom(r)
	var id string
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO merchant_warnings (merchant_id, reason, note, issued_by)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		merchantID, reason, clip(req.Note, 500), actor).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}

	var ownerID *string
	_ = s.pg.QueryRow(r.Context(),
		`SELECT owner_user_id FROM merchants WHERE id = $1`, merchantID).Scan(&ownerID)
	if ownerID != nil {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: *ownerID, Kind: notifications.KindOrder,
			Title: notifTitles.warningIssued, Body: clip(req.Note, 200),
			Entity: "merchant", EntityID: merchantID, Href: "/portal/reviews",
		})
	}
	s.audit(r, "ops.merchant_warning_issued", "merchant", merchantID, map[string]any{
		"reason": reason,
	})
	s.touch("merchant", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}
