package server

import (
	"net/http"
	"slices"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// كشف محفظة مستخدم — أدمن/مالية/عمليات (قراءة).
func (s *Server) handleAdminWalletStatement(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	st, err := s.wallet.StatementFor(r.Context(), chi.URLParam(r, "id"), limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

// حركة يدوية على المحفظة (شحن/تعويض/تسوية/سحب) — أدمن/مالية فقط.
func (s *Server) handleAdminWalletApply(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Amount int64  `json:"amount"`
		Kind   string `json:"kind"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// الحركات اليدوية المسموحة من اللوحة فقط — حركات الطلبات يصدرها المحرك
	if !slices.Contains([]string{"topup", "compensation", "adjustment", "payout"}, req.Kind) {
		s.respondErr(w, errValidation)
		return
	}
	actor := userIDFrom(r)
	balance, err := s.wallet.Apply(r.Context(), chi.URLParam(r, "id"),
		req.Amount, req.Kind, "", req.Note, &actor)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"balance": balance})
}
