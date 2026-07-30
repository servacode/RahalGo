package server

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// handleMeSummary بيانات التوب بار الموحّدة لأي مستخدم: الاسم، الصورة، رصيد المحفظة.
func (s *Server) handleMeSummary(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	var out struct {
		FullName    string  `json:"full_name"`
		AvatarThumb *string `json:"avatar_thumb_url"`
		Balance     int64   `json:"balance"`
	}
	err := s.pg.QueryRow(r.Context(), `
		SELECT u.full_name,
		       (SELECT m.thumb_path FROM media m WHERE m.id = u.avatar_media_id),
		       COALESCE((SELECT w.balance FROM wallets w WHERE w.user_id = u.id), 0)
		FROM users u WHERE u.id = $1`, uid).
		Scan(&out.FullName, &out.AvatarThumb, &out.Balance)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	out.AvatarThumb = media.URLForPtr(out.AvatarThumb) // "/media/" prefix (نمط الوسائط)
	httpx.JSON(w, http.StatusOK, out)
}

// handleMyAvatar يرفع صورة المستخدم لنفسه (نوع avatar) ويضبطها، ويعيد رابط المصغّرة.
func (s *Server) handleMyAvatar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, media.MaxUploadBytes+64<<10)
	if err := r.ParseMultipartForm(media.MaxUploadBytes); err != nil {
		s.respondErr(w, media.ErrTooLarge)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	defer file.Close()

	uid := userIDFrom(r)
	// النوع مثبّت "avatar" — لا نثق بقيمة العميل على نقطة عامة للمستخدمين.
	mm, err := s.media.Save(r.Context(), uid, "avatar", file)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.SetOwnAvatar(r.Context(), uid, mm.ID); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"avatar_thumb_url": mm.ThumbURL})
}

// handleDeleteMyAvatar يزيل صورة المستخدم.
func (s *Server) handleDeleteMyAvatar(w http.ResponseWriter, r *http.Request) {
	if err := s.identity.SetOwnAvatar(r.Context(), userIDFrom(r), ""); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"removed": true})
}
