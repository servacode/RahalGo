package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ---------- أكواد الخصم ----------

func (s *Server) handleListPromos(w http.ResponseWriter, r *http.Request) {
	promos, err := s.catalog.ListPromos(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, promos)
}

func (s *Server) handleCreatePromo(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.PromoInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	p, err := s.catalog.CreatePromo(r.Context(), userIDFrom(r), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (s *Server) handleUpdatePromo(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.PromoInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	p, err := s.catalog.UpdatePromo(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

// ---------- البانرات ----------

func (s *Server) handleListBanners(w http.ResponseWriter, r *http.Request) {
	banners, err := s.catalog.ListBanners(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, banners)
}

func (s *Server) handleCreateBanner(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.BannerInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	b, err := s.catalog.CreateBanner(r.Context(), userIDFrom(r), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, b)
}

func (s *Server) handleUpdateBanner(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.BannerInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	b, err := s.catalog.UpdateBanner(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

func (s *Server) handleDeleteBanner(w http.ResponseWriter, r *http.Request) {
	if err := s.catalog.DeleteBanner(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// ---------- الإعدادات الديناميكية ----------

func (s *Server) handleListSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `SELECT key, value, updated_at FROM app_settings ORDER BY key`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type setting struct {
		Key       string          `json:"key"`
		Value     json.RawMessage `json:"value"`
		UpdatedAt time.Time       `json:"updated_at"`
	}
	out := []setting{}
	for rows.Next() {
		var st setting
		if err := rows.Scan(&st.Key, &st.Value, &st.UpdatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, st)
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (s *Server) handleSetSetting(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Value json.RawMessage `json:"value"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	actor := userIDFrom(r)
	var v any
	if err := json.Unmarshal(req.Value, &v); err != nil {
		s.respondErr(w, errValidation)
		return
	}
	if err := s.settings.Set(r.Context(), chi.URLParam(r, "key"), v, &actor); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}
