package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

func (s *Server) handleGetMenu(w http.ResponseWriter, r *http.Request) {
	menu, err := s.catalog.GetMenu(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, menu)
}

func (s *Server) handleCreateSection(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.SectionInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	sec, err := s.catalog.CreateSection(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, sec)
}

// handleUpdateSection تعديلُ اسم القسم أو صورته.
func (s *Server) handleUpdateSection(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.SectionInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	sec, err := s.catalog.UpdateSection(r.Context(), userIDFrom(r), chi.URLParam(r, "sectionID"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sec)
}

func (s *Server) handleDeleteSection(w http.ResponseWriter, r *http.Request) {
	if err := s.catalog.DeleteSection(r.Context(), userIDFrom(r), chi.URLParam(r, "sectionID"), clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.MenuItemInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	id, err := s.catalog.CreateItem(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.MenuItemInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.catalog.UpdateItem(r.Context(), userIDFrom(r), chi.URLParam(r, "itemID"), *req, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (s *Server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	if err := s.catalog.DeleteItem(r.Context(), userIDFrom(r), chi.URLParam(r, "itemID"), clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}
