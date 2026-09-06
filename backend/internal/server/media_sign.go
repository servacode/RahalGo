package server

import (
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// handleSignMedia يُصدر رابطاً موقَّعاً لوسيطٍ شخصيّ — `D13`.
//
// ══════════════════════════════════════════════════════════════════════
// **ولا يُوقَّع إلّا ما هو محميٌّ فعلاً**
// ══════════════════════════════════════════════════════════════════════
//
// **ونقطةُ توقيعٍ تُوقّع أيَّ نصٍّ تُصبح مفتاحاً عامّاً** — فمن مرّر
// مساراً ليس في الجدول أخذ توقيعاً لملفٍّ يزرعه.
//
// **والصلاحيّةُ في المسار نفسِه** (`RequireRoles` على `/admin`): **من
// يرى ملفَّ إنسانٍ يرى صورتَه.**
func (s *Server) handleSignMedia(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimSpace(r.URL.Query().Get("path"))
	p = strings.TrimPrefix(p, "/media/")
	if p == "" || strings.Contains(p, "..") {
		s.respondErr(w, errValidation)
		return
	}
	if !s.media.IsProtected(r.Context(), p) {
		// **والعامُّ لا يُوقَّع** — رابطُه عارٍ أصلاً.
		httpx.JSON(w, http.StatusOK, map[string]any{"url": media.URLFor(p)})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"url": media.SignedURL(p)})
}
