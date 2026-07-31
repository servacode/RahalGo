package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

func rolesFrom(r *http.Request) []string {
	roles, _ := r.Context().Value(ctxRoles).([]string)
	return roles
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	res, err := s.orders.List(r.Context(), orders.ListFilter{
		Status:     q.Get("status"),
		MerchantID: q.Get("merchant_id"),
		CustomerID: q.Get("customer_id"),
		DriverID:   q.Get("driver_id"),
		Query:      q.Get("query"),
		OpenOnly:   q.Get("open") == "1",
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleOrderAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.orders.Alerts(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, alerts)
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	o, err := s.orders.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

func (s *Server) handleOrderTransition(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		To   string `json:"to"`
		Note string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o, err := s.orders.Transition(r.Context(), userIDFrom(r), rolesFrom(r),
		chi.URLParam(r, "id"), req.To, req.Note)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

func (s *Server) handleOrderAssign(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		DriverID string `json:"driver_id"`
		Note     string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o, err := s.orders.AssignDriver(r.Context(), userIDFrom(r), rolesFrom(r),
		chi.URLParam(r, "id"), req.DriverID, req.Note)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// السائق يعرف بإسناد الطلب فوراً (يستعمله تطبيقه)
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: req.DriverID, Kind: notifications.KindOrder,
		Title: notifTitles.driverAssigned, Entity: "order",
		EntityID: chi.URLParam(r, "id"), Href: "/orders",
	})
	httpx.JSON(w, http.StatusOK, o)
}
