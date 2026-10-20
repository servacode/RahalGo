package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **موظّفو المتجر** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — انظر `identity/staff.go`
// ══════════════════════════════════════════════════════════════════════

// errStaffNotAllowed **بابٌ لصاحب الحساب وحدَه** — الموظّفُ يعمل باسم المتجر ولا يمسّ
// كلمةَ سرّ صاحبه ولا رقمَه ولا اسمَه ولا يحذف حسابَه.
var errStaffNotAllowed = httpx.NewError(http.StatusForbidden, "staff_not_allowed", "errors.forbidden")

// staffFrom **موظّفُ هذا الطلب** — وفارغٌ لصاحب الحساب.
func staffFrom(r *http.Request) string {
	v, _ := r.Context().Value(ctxStaff).(string)
	return v
}

// notStaff **يغلق باباً شخصيّاً على الموظّف** — في الخادم لا بإخفاء زرّ.
func (s *Server) notStaff(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if staffFrom(r) != "" {
			s.respondErr(w, errStaffNotAllowed)
			return
		}
		h(w, r)
	}
}

// staffAddResponse **الموظّفُ وكلمةُ سرّه** — تُعرض مرّةً لمن أضافه، وتُرسَل له واتساب.
func (s *Server) addStaffFor(w http.ResponseWriter, r *http.Request, ownerID string) {
	req, err := decode[struct {
		Phone string `json:"phone"`
		Name  string `json:"name"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	st, plain, err := s.identity.AddStaff(r.Context(), userIDFrom(r), ownerID, req.Phone, req.Name, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والرسالةُ إلى الموظّف** — رقمُه، ثمّ كلمتُه، ثمّ كيف يدخل. وتعذّرُها لا يُسقط الإضافة:
	// الكلمةُ تُعرض لمن أضافه فيبلّغها بيده.
	text := "رحّال غو — صار عندك حساب موظّف بمتجر " + s.ownerStoreName(r, ownerID) + ".\n" +
		"بالرسالتين فوق: رقمك وكلمة المرور.\n" +
		"📱 ادخل بتطبيق المتجر: " + s.appLink("merchant") +
		iphoneLine("merchant", s.siteBase())
	sent := s.sendText(r.Context(), st.Phone, st.Phone) && s.sendText(r.Context(), st.Phone, plain) &&
		s.sendText(r.Context(), st.Phone, text)
	httpx.JSON(w, http.StatusCreated, map[string]any{"staff": st, "password": plain, "sent": sent})
}

// ownerStoreName **اسمُ متجر صاحب الحساب** — أوّلُها إن كانت فروعاً.
func (s *Server) ownerStoreName(r *http.Request, ownerID string) string {
	var name string
	_ = s.pg.QueryRow(r.Context(), `SELECT name FROM merchants WHERE owner_user_id = $1
		ORDER BY created_at LIMIT 1`, ownerID).Scan(&name)
	return strings.TrimSpace(name)
}

// merchantOwnerOf **صاحبُ هذا المتجر** — للإدارة، بمعرّف المتجر.
func (s *Server) merchantOwnerOf(r *http.Request, merchantID string) string {
	var owner string
	_ = s.pg.QueryRow(r.Context(), `SELECT COALESCE(owner_user_id::text, '') FROM merchants
		WHERE id::text = $1`, merchantID).Scan(&owner)
	return owner
}

// ── الإدارة: /admin/merchants/{id}/staff ───────────────────────────────

func (s *Server) handleAdminListStaff(w http.ResponseWriter, r *http.Request) {
	owner := s.merchantOwnerOf(r, chi.URLParam(r, "id"))
	if owner == "" {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	list, err := s.identity.ListStaff(r.Context(), owner)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"staff": list})
}

func (s *Server) handleAdminAddStaff(w http.ResponseWriter, r *http.Request) {
	owner := s.merchantOwnerOf(r, chi.URLParam(r, "id"))
	if owner == "" {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.addStaffFor(w, r, owner)
}

func (s *Server) handleAdminRemoveStaff(w http.ResponseWriter, r *http.Request) {
	owner := s.merchantOwnerOf(r, chi.URLParam(r, "id"))
	if owner == "" {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err := s.identity.RemoveStaff(r.Context(), userIDFrom(r), owner, chi.URLParam(r, "staffID"), clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"removed": true})
}

// ── صاحبُ المتجر: /merchant/staff ─────────────────────────────────────
//
// **وموظّفٌ لا يدير الموظّفين** (`notStaff` على المسار) — صاحبُ المتجر وحدَه.

func (s *Server) handleMerchantListStaff(w http.ResponseWriter, r *http.Request) {
	list, err := s.identity.ListStaff(r.Context(), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"staff": list})
}

func (s *Server) handleMerchantAddStaff(w http.ResponseWriter, r *http.Request) {
	s.addStaffFor(w, r, userIDFrom(r))
}

func (s *Server) handleMerchantRemoveStaff(w http.ResponseWriter, r *http.Request) {
	if err := s.identity.RemoveStaff(r.Context(), userIDFrom(r), userIDFrom(r), chi.URLParam(r, "staffID"), clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"removed": true})
}
