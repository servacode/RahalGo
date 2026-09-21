package server

// **شكاواه هو — وأين وصلت.**
//
// # المسألة
//
// كان الزبونُ يفتح شكوى ثمّ **لا يجد لها أثراً في أيّ شاشة**: التذاكرُ كلُّها
// تحت `/admin` — لمكتب المنصة وحدَه. **وما رآه بعدها سطرٌ في بطاقة الطلب يقول
// رقمَها ولا يقول حالَها.**
//
// **ومن اشتكى ولم يرَ جواباً ظنّ أنّ شكواه ضاعت** — فيشتكي ثانيةً، أو يتّصل،
// **أو يسكت ويذهب.** والسكوتُ أسوأ: نخسر الزبونَ ولا نعرف لماذا.
//
// (شهده المالك ٢٠٢٦-٠٨-٠٣: «الشكاوي ليس لها مكان بحساب المستخدم، أين يرى ما
// حالة الشكوى؟ لا يوجد».)
//
// # وما يُعرض وما لا يُعرض
//
// **حالتُها، وردُّ المنصة، والتعويضُ إن وقع** — وهي ما يخصّه. **ولا يُعرض
// اسمُ المتجر** ولا شيءٌ يدلّ عليه: القاعدةُ نفسُها في `customer_privacy.go`،
// **وبابٌ خلفيٌّ في شاشةِ شكاوى يهدم ما تحرسه شاشةُ الطلبات.**

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// handleMyTickets شكاوى الزبون نفسِه — الأحدثُ أوّلاً.
func (s *Server) handleMyTickets(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT t.id::text, t.number, o.number, t.subject, COALESCE(t.reason, ''),
		       t.status, COALESCE(t.compensation, 0), COALESCE(t.resolution, ''),
		       t.created_at, t.resolved_at
		FROM tickets t
		LEFT JOIN orders o ON o.id = t.order_id
		WHERE t.customer_id = $1
		ORDER BY t.created_at DESC
		LIMIT 100`, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type row struct {
		ID          string `json:"id"`
		Number      int64  `json:"number"`
		OrderNumber *int64 `json:"order_number"`
		Subject     string `json:"subject"`
		Reason      string `json:"reason"`
		Status      string `json:"status"`
		// Compensation ما عُوّض به — **يُعرض لأنّه ماله.**
		Compensation int64 `json:"compensation"`
		// Resolution كلمةُ المنصة عند الإغلاق — **وشكوى تُغلق بلا كلمة تُقرأ
		// تجاهلاً**، ولو كان القرارُ في صالحه.
		Resolution string     `json:"resolution"`
		CreatedAt  time.Time  `json:"created_at"`
		ResolvedAt *time.Time `json:"resolved_at"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.ID, &x.Number, &x.OrderNumber, &x.Subject, &x.Reason,
			&x.Status, &x.Compensation, &x.Resolution, &x.CreatedAt, &x.ResolvedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"tickets": out})
}

// handleMyTicketDetail **تذكرتُه هو — بردودها** (`CUST-SUP-013`، PRQ-2).
//
// **كان الردُّ يُقرأ تحت `/admin/tickets` وحدَه** — **فالزبونُ يفتح شكوى ثمّ
// لا يرى جوابَ المنصّة.** فيُعرَض له تفصيلُها بردودها، **وتذكرةُ غيره لا
// تُقرأ**: العزلُ في الخادم، **و٤٠٤ لا ٤٠٣** فلا يُكشَف وجودُ ما ليس له.
func (s *Server) handleMyTicketDetail(w http.ResponseWriter, r *http.Request) {
	t, err := s.support.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if t.CustomerID != userIDFrom(r) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

// handleMyTicketReply **يردّ على تذكرته ما دامت مفتوحة** (`CUST-SUP-014`، PRQ-2).
//
// **الملكيّةُ أوّلاً** (تذكرةُ غيره لا يُردّ عليها)، **ثمّ الحالُ**:
// `support.Reply` يردّ `ticket_resolved` على المحلولة، **فلا ردَّ على مغلقة.**
// **والمكتبُ يعلم بردّه** — لا يُبتلع كما لا يُبتلع ردُّه هو.
func (s *Server) handleMyTicketReply(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Body string `json:"body"`
	}](r)
	if err != nil || strings.TrimSpace(req.Body) == "" {
		s.respondErr(w, errValidation)
		return
	}
	id := chi.URLParam(r, "id")
	t0, err := s.support.Get(r.Context(), id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if t0.CustomerID != userIDFrom(r) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	t, err := s.support.Reply(r.Context(), userIDFrom(r), id, req.Body)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.notify.NotifyOps(r.Context(), notifications.Input{
		Kind: notifications.KindTicket, Title: notifTitles.ticketReply, Body: t.Subject,
		Entity: "ticket", EntityID: t.ID, Href: "/dashboard/tickets",
	})
	httpx.JSON(w, http.StatusOK, t)
}
