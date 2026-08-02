package server

// بابُ الشكوى عند الزبون — لا في رقم هاتفٍ يتّصل به.

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/support"
)

// handleComplaintReasons ما يملك الزبونُ اختيارَه على هذا الطلب.
//
// **والقائمةُ تتبع حالةَ الطلب**: «لم يصلني طلبي» على طلبٍ لم يُسلَّم لغو —
// حالتُه تقول ذلك أصلاً، **وسؤالُ المستخدم عمّا نعرفه يجعله يشكّ فيما نعرف.**
func (s *Server) handleComplaintReasons(w http.ResponseWriter, r *http.Request) {
	var status string
	if err := s.pg.QueryRow(r.Context(),
		`SELECT status FROM orders WHERE id = $1 AND customer_id = $2`,
		chi.URLParam(r, "id"), userIDFrom(r)).Scan(&status); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"reasons": support.ReasonsFor(status)})
}

// handleOpenComplaint يفتح تذكرةً على طلبِ صاحبها.
func (s *Server) handleOpenComplaint(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Reason string `json:"reason"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	t, err := s.support.Complaint(r.Context(), userIDFrom(r), orderID,
		req.Reason, clip(req.Note, 1000))
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **ورقمُ الشكوى يُقال له.**
	//
	// من شكا ولم يُعطَ رقماً **لا يملك أن يسأل عنها**: يتّصل فيقول «شكوتُ
	// أمس» فيُبحث عنه بالاسم. **ورقمٌ في يده يجعل المتابعةَ ممكنةً من الطرفين.**
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: t.CustomerID, Kind: notifications.KindTicket,
		Title: notifTitles.ticketOpened, Body: t.Subject,
		Entity: "ticket", EntityID: t.ID, Href: "/orders/" + orderID,
	})
	// **والعملياتُ تُخبَر فوراً لا حين تفتح اللوحة.**
	//
	// «لم يصلني طلبي» خبرُ مالٍ خرج بلا مقابل، **وساعةُ تأخيرٍ في قراءته
	// ساعةٌ يبتعد فيها من يُسأل.**
	s.notify.NotifyOps(r.Context(), notifications.Input{
		Kind: notifications.KindTicket, Title: notifTitles.ticketNewOps,
		Body:   t.Subject + " — " + req.Reason,
		Entity: "ticket", EntityID: t.ID, Href: "/dashboard/tickets",
	})
	s.audit(r, "customer.complaint_opened", "ticket", t.ID, map[string]any{
		"order_id": orderID, "reason": req.Reason,
	})
	httpx.JSON(w, http.StatusCreated, t)
}

// handleMyComplaint شكوى الزبون على طلبه إن فتحها — ليراها في صفحة الطلب.
//
// **وبلا هذه لا يعرف أنّه شكا**: يضغط الزرَّ ثمّ يعود بعد ساعةٍ فيجده كما هو،
// **فيضغطه ثانيةً ويُردّ عليه «شكواك مفتوحة»** — وردٌّ يقول له ما كان ينبغي
// أن يراه بنفسه.
func (s *Server) handleMyComplaint(w http.ResponseWriter, r *http.Request) {
	var id string
	err := s.pg.QueryRow(r.Context(), `
		SELECT t.id FROM tickets t JOIN orders o ON o.id = t.order_id
		WHERE t.order_id = $1 AND o.customer_id = $2
		ORDER BY t.created_at DESC LIMIT 1`,
		chi.URLParam(r, "id"), userIDFrom(r)).Scan(&id)
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"ticket": nil})
		return
	}
	t, err := s.support.Get(r.Context(), id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ticket": t})
}
