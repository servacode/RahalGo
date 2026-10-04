package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/support"
)

// ---------- التقييم (واجهة الزبون — تُستخدم من التطبيق/الموقع) ----------

func (s *Server) handleRateOrder(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		PlatformStars int  `json:"platform_stars"`
		DriverStars   *int `json:"driver_stars"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	err = s.orders.RateOrder(r.Context(), userIDFrom(r), rolesFrom(r),
		chi.URLParam(r, "id"), req.PlatformStars, req.DriverStars)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **السائقُ وحدَه يعرف بالتقييم** — الزبونُ يقيّم السائقَ والمنصّة فقط
	// (قرارُ المالك ٢٠٢٦-١٠-٠٤: «ما يعرف المتجرَ ليقيّمه»). **كان صاحبُ المتجر
	// يُخبَر بتقييمٍ جديدٍ ليس فيه نجمةٌ له.**
	var driverID *string
	_ = s.pg.QueryRow(r.Context(), `
		SELECT o.driver_id FROM orders o WHERE o.id = $1`, chi.URLParam(r, "id")).Scan(&driverID)
	for _, uid := range []*string{driverID} {
		if uid != nil {
			s.notify.Notify(r.Context(), notifications.Input{
				UserID: *uid, Kind: notifications.KindRating,
				Title: notifTitles.ratingNew, Entity: "rating",
				EntityID: chi.URLParam(r, "id"), Href: "/portal/reviews",
				// **يُحفَظ ولا يرنّ** — (قرارُ المالك ٢٠٢٦-٠٨-١٤: «لا
				// نريد إشعاراتٍ كثيرةً بلا فائدة»).
				//
				// **ونجمةٌ تُعطى لا فعلَ فيها**: يقرؤها حين يفتح
				// تقييماتِه، **ورنّةٌ تقطع عليه طريقَه لأجلها** تُنفق
				// انتباهَه في غير موضعه.
				Silent: true,
			})
		}
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"rated": true})
}

// ---------- التذاكر ----------

func (s *Server) handleListTickets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	// **والمتأخّرةُ بمهلة الإعدادات** — بطاقةُ «شكاوى متأخّرة» في رئيسيّة
	// المدير تفتح هنا على العدد نفسِه (قرارُ المالك ٢٠٢٦-١٠-٠٤).
	f := support.TicketFilter{Status: q.Get("status")}
	if q.Get("late") == "1" {
		f.LateHours = s.ticketLateHours(r.Context())
	}
	res, err := s.support.ListFiltered(r.Context(), f, page, perPage)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleCreateTicket(w http.ResponseWriter, r *http.Request) {
	req, err := decode[support.CreateInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	t, err := s.support.Create(r.Context(), userIDFrom(r), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **فتحُ تذكرةٍ من اللوحة يُكتب** (قرارُ المالك ٢٠٢٦-١٠-٠٤، الرابع) — كان
	// الإغلاقُ وحدَه يُكتب.
	s.audit(r, "ops.ticket_created", "ticket", t.ID, map[string]any{
		"title": t.Subject, "customer_id": t.CustomerID,
	})
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: t.CustomerID, Kind: notifications.KindTicket,
		Title: notifTitles.ticketOpened, Body: t.Subject,
		Entity: "ticket", EntityID: t.ID, Href: "/portal/orders",
	})
	s.notify.NotifyOps(r.Context(), notifications.Input{
		Kind: notifications.KindTicket, Title: notifTitles.ticketNewOps, Body: t.Subject,
		Entity: "ticket", EntityID: t.ID, Href: "/dashboard/tickets",
	})
	httpx.JSON(w, http.StatusCreated, t)
}

func (s *Server) handleGetTicket(w http.ResponseWriter, r *http.Request) {
	t, err := s.support.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (s *Server) handleTicketReply(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Body string `json:"body"`
	}](r)
	if err != nil || req.Body == "" {
		s.respondErr(w, errValidation)
		return
	}
	t, err := s.support.Reply(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), req.Body)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والردُّ يُكتب بنصّه** — مقصوصاً عند خمسمئة حرف.
	replyNote := []rune(req.Body)
	if len(replyNote) > 500 {
		replyNote = replyNote[:500]
	}
	s.audit(r, "ops.ticket_replied", "ticket", t.ID, map[string]any{
		"title": t.Subject, "note": string(replyNote),
	})
	// **صاحبُ الشكوى يعرف بالردّ فوراً** (لا يردّ الموظف على نفسه) — لا زبونُ
	// الطلب: بلاغُ سائقٍ على زبونٍ كان يُخبر الزبونَ المشتكى عليه بردّ المكتب.
	if t.ComplainantID != userIDFrom(r) {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: t.ComplainantID, Kind: notifications.KindTicket,
			Title: notifTitles.ticketReply, Body: t.Subject,
			Entity: "ticket", EntityID: t.ID, Href: "/portal/orders",
		})
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (s *Server) handleTicketResolve(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Resolution   string `json:"resolution"`
		Compensation int64  `json:"compensation"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	t, err := s.support.Resolve(r.Context(), userIDFrom(r), chi.URLParam(r, "id"),
		req.Resolution, req.Compensation, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **التعويضُ اقتراحٌ لا دفع** — يُسجَّل حتى لو كان صفراً، فحلُّ الشكوى بلا
	// تعويضٍ قرارٌ أيضاً وقد يُراجَع. والدفعُ يُسجَّل عند موافقة الماليّة.
	s.audit(r, "finance.ticket_resolve", "ticket", t.ID, map[string]any{
		"compensation_proposed": req.Compensation, "resolution": t.Resolution,
		"customer_id": t.CustomerID, "complainant_id": t.ComplainantID,
	})

	// **وبتعويضٍ معلَّقٍ لم تُحلّ بعد** — «بانتظار المالية» لا «تم حل شكواك».
	title := notifTitles.ticketResolved
	if t.Status == support.StatusAwaitingFinance {
		title = notifTitles.ticketAwaitingFinance
	}
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: t.ComplainantID, Kind: notifications.KindTicket,
		Title: title, Body: t.Resolution,
		// **إلى صفحة شكاواه** — حيث يرى حالَها وردَّنا والتعويض.
		Entity: "ticket", EntityID: t.ID, Href: "/portal/complaints",
	})
	// **ولا رصيدَ يتحرّك هنا** — الاقتراحُ يظهر حيّاً في لوح طلبات المحفظة
	// للماليّة، **ورصيدُه يتحدّث حين توافق** (`handleDecideWalletRequest`).
	s.touchUser(t.ComplainantID, "ticket")
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, t)
}
