package server

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// handleUploadMedia يستقبل صورة multipart (حقل file + حقل kind)
// ويعيد سجل الوسائط بمساريه — يُربط لاحقاً بالكيان عبر معرفه.
func (s *Server) handleUploadMedia(w http.ResponseWriter, r *http.Request) {
	// الحد أعلى قليلاً من حد الصورة لاستيعاب غلاف multipart
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

	m, err := s.media.Save(r.Context(), userIDFrom(r), r.FormValue("kind"), file)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, m)
}
