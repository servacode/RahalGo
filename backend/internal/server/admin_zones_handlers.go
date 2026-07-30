package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ---------- أوقات الدوام ----------

func (s *Server) handleGetHours(w http.ResponseWriter, r *http.Request) {
	hours, err := s.catalog.GetHours(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, hours)
}

func (s *Server) handleSetHours(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Days []catalog.DayHours `json:"days"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.catalog.SetHours(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), req.Days, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// ---------- مناطق التغطية ----------

func (s *Server) handleListZones(w http.ResponseWriter, r *http.Request) {
	zones, err := s.catalog.ListZones(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, zones)
}

func (s *Server) handleCreateZone(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.ZoneInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	z, err := s.catalog.CreateZone(r.Context(), userIDFrom(r), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, z)
}

func (s *Server) handleUpdateZone(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.ZoneInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	z, err := s.catalog.UpdateZone(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, z)
}

func (s *Server) handleDeleteZone(w http.ResponseWriter, r *http.Request) {
	if err := s.catalog.DeleteZone(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}
