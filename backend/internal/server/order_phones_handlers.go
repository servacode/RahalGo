package server

import (
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

// ══════════════════════════════════════════════════════════════════════
// **رقمُ الطلبات بحسب الدوام** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — هجرة ٠٤٦٣
// ══════════════════════════════════════════════════════════════════════
//
// «دوام كامل» رقمٌ واحدٌ كما كان، أو «صباحي ومسائي» برقمين وساعةِ تبديلٍ يحدّدها
// المتجر — **وتحويلُ الطلب على واتساب يختار رقمَ الدوام الحاليّ** (`merchant_dispatch.go`).

type orderPhones struct {
	Mode        string `json:"mode"`
	Morning     string `json:"morning_phone"`
	Evening     string `json:"evening_phone"`
	EveningFrom string `json:"evening_from"`
}

var hhmm = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func (s *Server) readOrderPhones(r *http.Request, merchantID string) (*orderPhones, error) {
	var p orderPhones
	err := s.pg.QueryRow(r.Context(), `
		SELECT order_phone_mode, COALESCE(order_phone_morning, ''), COALESCE(order_phone_evening, ''),
		       to_char(order_phone_evening_from, 'HH24:MI')
		FROM merchants WHERE id::text = $1`, merchantID).Scan(&p.Mode, &p.Morning, &p.Evening, &p.EveningFrom)
	if err != nil {
		return nil, httpx.ErrNotFound
	}
	return &p, nil
}

// writeOrderPhones **يحفظ الاختيار** — والمقسومُ يشترط رقمين صحيحين وساعةً صحيحة.
func (s *Server) writeOrderPhones(w http.ResponseWriter, r *http.Request, merchantID string) {
	req, err := decode[orderPhones](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	switch req.Mode {
	case "full":
	case "split":
		m, ok1 := identity.NormalizePhone(req.Morning)
		e, ok2 := identity.NormalizePhone(req.Evening)
		if !ok1 || !ok2 || !hhmm.MatchString(req.EveningFrom) {
			s.respondErr(w, errValidation)
			return
		}
		req.Morning, req.Evening = m, e
	default:
		s.respondErr(w, errValidation)
		return
	}
	if req.EveningFrom == "" {
		req.EveningFrom = "16:00"
	}
	if !hhmm.MatchString(req.EveningFrom) {
		s.respondErr(w, errValidation)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE merchants SET order_phone_mode = $2,
		       order_phone_morning = NULLIF($3, ''), order_phone_evening = NULLIF($4, ''),
		       order_phone_evening_from = $5::time, updated_at = now()
		WHERE id::text = $1`, merchantID, req.Mode, req.Morning, req.Evening, req.EveningFrom)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.auditCtx(r.Context(), userIDFrom(r), clientIP(r), "merchant.order_phones", "merchant", merchantID,
		map[string]any{"mode": req.Mode, "evening_from": req.EveningFrom})
	p, _ := s.readOrderPhones(r, merchantID)
	httpx.JSON(w, http.StatusOK, p)
}

func (s *Server) handleAdminGetOrderPhones(w http.ResponseWriter, r *http.Request) {
	p, err := s.readOrderPhones(r, chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (s *Server) handleAdminPutOrderPhones(w http.ResponseWriter, r *http.Request) {
	s.writeOrderPhones(w, r, chi.URLParam(r, "id"))
}

func (s *Server) handleMerchantGetOrderPhones(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, id) {
		s.respondErr(w, errForbidden)
		return
	}
	p, err := s.readOrderPhones(r, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (s *Server) handleMerchantPutOrderPhones(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.merchantWriteGuard(r, id); err != nil {
		s.respondErr(w, err)
		return
	}
	s.writeOrderPhones(w, r, id)
}
