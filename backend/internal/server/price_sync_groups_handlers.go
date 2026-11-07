package server

// ══════════════════════════════════════════════════════════════════════
// **توزيع أصناف الأمين على أقسام السوق — الأبواب** (قرار المالك ٢٠٢٦-١١-٠٧)
// ══════════════════════════════════════════════════════════════════════
//
//	GET  /merchant/stores/{id}/price-sync/groups     مجموعات الأمين وقسم كلّ وحدة والمقترح
//	PUT  /merchant/stores/{id}/price-sync/groups     قسم لكلّ مجموعة — وبتنتقل أصنافها (مو اليدويّة)
//	POST /merchant/stores/{id}/price-sync/auto-sort  «وزّع الأصناف الباقية تلقائياً» من أسمائها
//	GET  /merchant/stores/{id}/price-sync/pending    أصناف قسم الانتظار صفحة صفحة (?after=&limit=&q=)
//	POST /merchant/stores/{id}/price-sync/move       نقل أصناف مختارة لقسم بالإيد ⇒ يدويّ
//
// المنطق كلّه بـ `catalog/pricesync_groups.go`، والحرّاس نفس باقي أبواب المزامنة.

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// priceSyncReadGuard **صاحب المتجر وعنده الأمين** — للقراءة.
func (s *Server) priceSyncReadGuard(w http.ResponseWriter, r *http.Request) (string, bool) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return "", false
	}
	if err := s.requireAmeen(r.Context(), merchantID); err != nil {
		s.respondErr(w, err)
		return "", false
	}
	return merchantID, true
}

// priceSyncWriteGuard **صاحب المتجر، مو موقوف، وعنده الأمين** — للكتابة.
func (s *Server) priceSyncWriteGuard(w http.ResponseWriter, r *http.Request) (string, bool) {
	merchantID := chi.URLParam(r, "id")
	if err := s.merchantWriteGuard(r, merchantID); err != nil {
		s.respondErr(w, err)
		return "", false
	}
	if err := s.requireAmeen(r.Context(), merchantID); err != nil {
		s.respondErr(w, err)
		return "", false
	}
	return merchantID, true
}

// handleMerchantPriceSyncGroups **مجموعات الأمين** وكم صنف لسّا بالانتظار.
func (s *Server) handleMerchantPriceSyncGroups(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := s.priceSyncReadGuard(w, r)
	if !ok {
		return
	}
	view, err := s.catalog.PriceSyncGroups(r.Context(), merchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, view)
}

// handleMerchantPriceSyncSetGroups **قسم لكلّ مجموعة** — وأصنافها بتنتقل إلّا اليدويّة.
func (s *Server) handleMerchantPriceSyncSetGroups(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := s.priceSyncWriteGuard(w, r)
	if !ok {
		return
	}
	req, err := decode[[]catalog.GroupChoice](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	moved, err := s.catalog.SetPriceSyncGroups(r.Context(), merchantID, userIDFrom(r), *req)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("menu", "merchant:"+merchantID)
	httpx.JSON(w, http.StatusOK, map[string]any{"moved": moved})
}

// handleMerchantPriceSyncAutoSort **«وزّع الأصناف الباقية تلقائياً».**
func (s *Server) handleMerchantPriceSyncAutoSort(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := s.priceSyncWriteGuard(w, r)
	if !ok {
		return
	}
	res, err := s.catalog.AutoSortPending(r.Context(), merchantID, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("menu", "merchant:"+merchantID)
	httpx.JSON(w, http.StatusOK, res)
}

// handleMerchantPriceSyncPending **أصناف قسم الانتظار** للمراجعة بالإيد.
func (s *Server) handleMerchantPriceSyncPending(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := s.priceSyncReadGuard(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	page, err := s.catalog.PendingItems(r.Context(), merchantID, q.Get("after"), q.Get("q"), limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, page)
}

// handleMerchantPriceSyncMove **«انقل المحدّد إلى قسم…»** — والنقل بالإيد يدويّ.
func (s *Server) handleMerchantPriceSyncMove(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := s.priceSyncWriteGuard(w, r)
	if !ok {
		return
	}
	req, err := decode[struct {
		ItemIDs   []string `json:"item_ids"`
		SectionID string   `json:"platform_section_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	moved, err := s.catalog.MoveSyncItems(r.Context(), merchantID, userIDFrom(r), req.ItemIDs, req.SectionID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("menu", "merchant:"+merchantID)
	httpx.JSON(w, http.StatusOK, map[string]any{"moved": moved})
}
