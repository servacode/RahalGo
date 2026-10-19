package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleRepSetLogo **المندوبُ يضع شعارَ متجر عميله** (طلبُ المالك ٢٠٢٦-١٠-١٩: «والمندوب يقدر
// كمان يضيف لوغو للمتجر») — والحارسُ `repMenuGuard` على المسار: **عميلُه وحدَه.**
//
// **والصورةُ صورتُه هو وبنوع الشعار** — كما عند صاحب المتجر (`handleMerchantSettings`):
// **ومعرّفٌ لصورةِ غيره يُردّ**، فلا يلبس متجرٌ شعارَ متجرٍ آخر بمعرّفٍ رآه.
func (s *Server) handleRepSetLogo(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	req, err := decode[struct {
		LogoMediaID string `json:"logo_media_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var mine bool
	if err := s.pg.QueryRow(r.Context(), `
		SELECT EXISTS (SELECT 1 FROM media
		                WHERE id::text = $1 AND kind = 'merchant_logo' AND created_by = $2)`,
		req.LogoMediaID, userIDFrom(r)).Scan(&mine); err != nil {
		s.respondErr(w, err)
		return
	}
	if !mine {
		s.respondErr(w, errValidation)
		return
	}
	if _, err := s.pg.Exec(r.Context(), `
		UPDATE merchants SET logo_media_id = $2::uuid, updated_at = now() WHERE id = $1`,
		merchantID, req.LogoMediaID); err != nil {
		s.respondErr(w, err)
		return
	}
	s.auditCtx(r.Context(), userIDFrom(r), clientIP(r), "merchant.logo_update", "merchant", merchantID,
		map[string]any{"logo_media_id": req.LogoMediaID, "by": "rep"})
	s.touch("menu", "merchant:"+merchantID)
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}
