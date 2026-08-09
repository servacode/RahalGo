package server

/*
**الإنذارُ يبلغ كلَّ دور — ويصل صاحبَه.**

(شكوى المالك ٢٠٢٦-٠٨-٠٩: «وجّه إنذار للسائق… بس ما وصل الإنذار» · «اجعله لكلّ
 الأدوار حتّى الزبون — ممكن يكون تعامل بسوء مع السائق، بالنهاية كمان السائق
 بشر ويقدّم خدمة».)

# ما كان

**جدولُ الإنذارات للمتاجر وحدَها.** فحين وُجّه إنذارٌ لسائقٍ **لم يُسجَّل شيء**
— كانت ملاحظةً نصّيّةً في سجلّ المتابعة **تصل الزبونَ لا السائق.**

**وإنذارٌ لا يُسجَّل لا يُعدّ ولا يُحتجّ به يومَ الحظر** — ولا يعرف صاحبُه
أنّه أُنذر.

# وجدولٌ واحدٌ لا أربعة

**أربعةُ جداولَ لأربعة أدوارٍ أربعُ نسخٍ من قاعدةٍ واحدة**: يُضاف حقلٌ في
إحداها ويُنسى في الثلاث. **والإنذارُ إنذارٌ مهما كان دورُ صاحبه.**

# ويُنسب إلى حسابٍ لا إلى كيان

**المتجرُ كيانٌ وصاحبُه شخص** — ومن أُنذر هو من يقرأ، **والحسابُ هو ما يُحظَر
لا اللافتة.**
*/

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

type warningItem struct {
	ID        string    `json:"id"`
	Reason    string    `json:"reason"`
	Note      string    `json:"note"`
	Role      string    `json:"role_code"`
	OrderNum  *int64    `json:"order_number"`
	CreatedAt time.Time `json:"created_at"`
}

const warningItemSelect = `
	SELECT w.id::text, w.reason, w.note, w.role_code, o.number, w.created_at
	FROM warnings w
	LEFT JOIN orders o ON o.id = w.order_id`

func (s *Server) scanWarningItems(w http.ResponseWriter, r *http.Request, q string, args ...any) {
	rows, err := s.pg.Query(r.Context(), q, args...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []warningItem{}
	for rows.Next() {
		var x warningItem
		if err := rows.Scan(&x.ID, &x.Reason, &x.Note, &x.Role, &x.OrderNum, &x.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"warnings": out})
}

// handleMyWarnings **إنذاراتي — يراها صاحبُها.**
//
// **ومن أُنذر ولا يعلم لا يُصلح شيئاً**: يُحظَر يوماً فيُفاجأ، **ويظنّ الظلمَ
// حيث كان خبر.**
func (s *Server) handleMyWarnings(w http.ResponseWriter, r *http.Request) {
	s.scanWarningItems(w, r, warningItemSelect+`
		WHERE w.user_id = $1 ORDER BY w.created_at DESC LIMIT 50`, userIDFrom(r))
}

// handleAdminUserWarnings **ما على هذا الحساب** — كما تراه العمليات.
func (s *Server) handleAdminUserWarnings(w http.ResponseWriter, r *http.Request) {
	s.scanWarningItems(w, r, warningItemSelect+`
		WHERE w.user_id = $1 ORDER BY w.created_at DESC LIMIT 100`, chi.URLParam(r, "id"))
}

// handleIssueUserWarning **إنذارٌ على حسابٍ — أيَّ دورٍ كان.**
func (s *Server) handleIssueUserWarning(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Reason   string  `json:"reason"`
		Note     string  `json:"note"`
		OrderID  *string `json:"order_id"`
		TicketID *string `json:"ticket_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والسببُ إلزاميّ**: إنذارٌ بلا سببٍ لا يُصحَّح ولا يُحتجّ به.
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		s.respondErr(w, errValidation)
		return
	}
	userID := chi.URLParam(r, "id")

	// **ودورُه وقتَ الإنذار يُثبَّت** — لا يُشتقّ عند القراءة.
	//
	// **فمن كان سائقاً ثمّ صار مندوباً** تُقرأ إنذاراتُه القديمةُ باسم دوره
	// الجديد، **ويبدو المندوبُ سيّئَ السجلّ في عملٍ لم يعمله.**
	var role string
	if err := s.pg.QueryRow(r.Context(), `
		SELECT COALESCE((SELECT role_code FROM user_roles WHERE user_id = $1
		                 ORDER BY granted_at LIMIT 1), 'customer')`, userID).
		Scan(&role); err != nil {
		s.respondErr(w, err)
		return
	}

	var id string
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO warnings (user_id, role_code, reason, note, order_id, ticket_id, issued_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text`,
		userID, role, reason, clip(req.Note, 500), req.OrderID, req.TicketID,
		userIDFrom(r)).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}

	// **ويصل صاحبَه — وهو ما لم يكن.**
	//
	// **إنذارٌ لا يبلغ من أُنذر ليس إنذاراً**: هو سطرٌ في دفترٍ يُقرأ يومَ
	// الحظر، **ولا فرصةَ لصاحبه أن يُصلح.** (وهي شكوى المالك بعينها.)
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: userID, Kind: notifications.KindOrder,
		Title: notifTitles.warningOnYou, Body: clip(req.Note, 200),
		Entity: "user", EntityID: userID, Href: "/portal/complaints",
	})
	s.audit(r, "ops.warning_issued", "user", userID, map[string]any{"reason": reason})
	s.touch("user", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}
