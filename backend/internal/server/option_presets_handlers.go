package server

// ══════════════════════════════════════════════════════════════════════
// **خياراتُ الصنف الجاهزة · نسخُ صنف · الأسعارُ دفعةً واحدة** (٢٠٢٦-١٠-٢٧)
// ══════════════════════════════════════════════════════════════════════
//
// المتجرُ والمندوبُ يكتبان بالمنطق نفسِه — **والحارسُ وحدَه يختلف**:
// `merchantWriteGuard` للمتجر، و`repMenuGuard` على مسار المندوب.

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ── المتجر ─────────────────────────────────────────────────────────────

func (s *Server) handleMerchantOptionPresets(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, id) {
		s.respondErr(w, errForbidden)
		return
	}
	s.readStorePresets(w, r, id)
}

func (s *Server) handleMerchantSetOptionPresets(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.merchantWriteGuard(r, id); err != nil {
		s.respondErr(w, err)
		return
	}
	s.writeStorePresets(w, r, id)
}

func (s *Server) handleMerchantDuplicateItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	if err := s.itemWriteGuard(r, itemID); err != nil {
		s.respondErr(w, err)
		return
	}
	s.duplicateItem(w, r, itemID)
}

func (s *Server) handleMerchantBulkPrice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.merchantWriteGuard(r, id); err != nil {
		s.respondErr(w, err)
		return
	}
	s.bulkPrice(w, r, id)
}

// ── المندوب (الحارسُ `repMenuGuard` على المسار) ─────────────────────────

func (s *Server) handleRepOptionPresets(w http.ResponseWriter, r *http.Request) {
	s.readStorePresets(w, r, chi.URLParam(r, "id"))
}

func (s *Server) handleRepSetOptionPresets(w http.ResponseWriter, r *http.Request) {
	s.writeStorePresets(w, r, chi.URLParam(r, "id"))
}

func (s *Server) handleRepDuplicateItem(w http.ResponseWriter, r *http.Request) {
	s.duplicateItem(w, r, chi.URLParam(r, "itemID"))
}

func (s *Server) handleRepBulkPrice(w http.ResponseWriter, r *http.Request) {
	s.bulkPrice(w, r, chi.URLParam(r, "id"))
}

// ── المشترك ────────────────────────────────────────────────────────────

func (s *Server) readStorePresets(w http.ResponseWriter, r *http.Request, merchantID string) {
	presets, custom, err := s.catalog.StoreOptionPresets(r.Context(), merchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"presets": presets, "custom": custom})
}

// writeStorePresets **`preset_ids` تستبدل الاختيار، و`reset` يعيده إلى التصنيف.**
func (s *Server) writeStorePresets(w http.ResponseWriter, r *http.Request, merchantID string) {
	in, err := decode[struct {
		PresetIDs []string `json:"preset_ids"`
		Reset     bool     `json:"reset"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var ids *[]string
	if !in.Reset {
		list := in.PresetIDs
		if list == nil {
			list = []string{}
		}
		ids = &list
	}
	if err := s.catalog.SetStoreOptionPresets(r.Context(), userIDFrom(r), merchantID, ids, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	s.readStorePresets(w, r, merchantID)
}

func (s *Server) duplicateItem(w http.ResponseWriter, r *http.Request, itemID string) {
	id, err := s.catalog.DuplicateItem(r.Context(), userIDFrom(r), itemID, s.itemNameMax(r.Context()), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("menu", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) bulkPrice(w http.ResponseWriter, r *http.Request, merchantID string) {
	in, err := decode[catalog.BulkPriceInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.catalog.BulkPrice(r.Context(), userIDFrom(r), merchantID, *in, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if res.Applied {
		s.touch("menu", "merchant:"+merchantID)
	}
	httpx.JSON(w, http.StatusOK, res)
}

// ── اللوحة ─────────────────────────────────────────────────────────────

func (s *Server) handleAdminOptionPresets(w http.ResponseWriter, r *http.Request) {
	presets, err := s.catalog.AllOptionPresets(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"presets": presets})
}

func (s *Server) handleAdminSaveOptionPreset(w http.ResponseWriter, r *http.Request) {
	in, err := decode[catalog.OptionPresetInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	id, err := s.catalog.SaveOptionPreset(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), *in, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id})
}

func (s *Server) handleAdminDeleteOptionPreset(w http.ResponseWriter, r *http.Request) {
	if err := s.catalog.DeleteOptionPreset(r.Context(), userIDFrom(r), chi.URLParam(r, "id"), clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}
