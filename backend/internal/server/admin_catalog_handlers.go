package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := s.catalog.ListCategories(r.Context(), r.URL.Query().Get("active") == "1")
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, cats)
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.CategoryInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	c, err := s.catalog.CreateCategory(r.Context(), userIDFrom(r), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, c)
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.CategoryInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	c, err := s.catalog.UpdateCategory(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (s *Server) handleListMerchants(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	res, err := s.catalog.ListMerchants(r.Context(),
		q.Get("query"), q.Get("category_id"), q.Get("status"), q.Get("rep_id"), page, perPage)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleCreateMerchant(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.MerchantInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	m, err := s.catalog.CreateMerchant(r.Context(), userIDFrom(r), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, m)
}

func (s *Server) handleUpdateMerchant(w http.ResponseWriter, r *http.Request) {
	req, err := decode[catalog.MerchantInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	m, err := s.catalog.UpdateMerchant(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *req, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, m)
}

// handleAdminGetMerchant متجرٌ بعينه — **لملفّه المفرد.**
//
// كان الوصولُ إليه بالبحث في القائمة باسمه، **ومتجران متشابها الاسم يُخلطان.**
// وكلُّ ما يخصّه — قائمتُه وساعاتُه ومخالفاتُه — كان في نوافذَ منبثقةٍ داخل
// جدول: **تُفتح واحدةً وتُغلق لتُفتح أخرى، ولا تُرى صورتُه مجتمعةً.**
func (s *Server) handleAdminGetMerchant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	mr, err := s.catalog.MerchantByID(r.Context(), id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, mr)
}
