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
		MerchantStars int    `json:"merchant_stars"`
		DriverStars   *int   `json:"driver_stars"`
		Comment       string `json:"comment"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	err = s.orders.RateOrder(r.Context(), userIDFrom(r), rolesFrom(r),
		chi.URLParam(r, "id"), req.MerchantStars, req.DriverStars, req.Comment)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// المتجر والسائق يعرفان بالتقييم فوراً (السمعة تتغيّر لحظياً)
	var ownerID, driverID *string
	_ = s.pg.QueryRow(r.Context(), `
		SELECT mm.owner_user_id, o.driver_id
		FROM orders o JOIN merchants mm ON mm.id = o.merchant_id
		WHERE o.id = $1`, chi.URLParam(r, "id")).Scan(&ownerID, &driverID)
	for _, uid := range []*string{ownerID, driverID} {
		if uid != nil {
			s.notify.Notify(r.Context(), notifications.Input{
				UserID: *uid, Kind: notifications.KindRating,
				Title: notifTitles.ratingNew, Entity: "rating",
				EntityID: chi.URLParam(r, "id"), Href: "/portal/reviews",
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
	res, err := s.support.List(r.Context(), q.Get("status"), page, perPage)
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
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: t.CustomerID, Kind: notifications.KindTicket,
		Title: notifTitles.ticketOpened, Body: t.Subject,
		Entity: "ticket", EntityID: t.ID, Href: "/orders",
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
	// الطرف الآخر يعرف بالرد فوراً (لا يردّ الموظف على نفسه)
	if t.CustomerID != userIDFrom(r) {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: t.CustomerID, Kind: notifications.KindTicket,
			Title: notifTitles.ticketReply, Body: t.Subject,
			Entity: "ticket", EntityID: t.ID, Href: "/orders",
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
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: t.CustomerID, Kind: notifications.KindTicket,
		Title: notifTitles.ticketResolved, Body: t.Resolution,
		Entity: "ticket", EntityID: t.ID, Href: "/orders",
	})
	httpx.JSON(w, http.StatusOK, t)
}
