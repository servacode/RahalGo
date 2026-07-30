package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	res, err := s.identity.AdminListUsers(r.Context(), q.Get("query"), q.Get("role"), page, perPage)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	req, err := decode[identity.CreateUserInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	user, err := s.identity.AdminCreateUser(r.Context(), userIDFrom(r), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, user)
}

func (s *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	req, err := decode[identity.UpdateUserInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	user, err := s.identity.AdminUpdateUser(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (s *Server) handleAdminGrantRole(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Role string `json:"role"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.AdminGrantRole(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), req.Role, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"granted": true})
}

func (s *Server) handleAdminRevokeRole(w http.ResponseWriter, r *http.Request) {
	if err := s.identity.AdminRevokeRole(r.Context(), userIDFrom(r),
		chi.URLParam(r, "id"), chi.URLParam(r, "role"), clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"revoked": true})
}
