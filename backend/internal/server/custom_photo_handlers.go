package server

import (
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
)

// maxPendingCustomPhotos **سقفُ الصور المرفوعة غير المربوطة بطلب** لكلّ زبون في اليوم —
// فنقطةُ رفعٍ بلا سقفٍ تملأ القرصَ.
const maxPendingCustomPhotos = 12

// handleMyCustomPhoto **الزبونُ يرفع صورةَ قائمته أو الغرض** (قرارُ المالك ٢٠٢٦-١١-١١).
//
// **النوعُ مثبَّتٌ `custom_photo`** — لا نثق بقيمة العميل. **والردُّ معرّفٌ ورابطٌ موقَّع**؛
// يُرسَل المعرّفُ مع `POST /orders/custom` (`photo_ids`).
func (s *Server) handleMyCustomPhoto(w http.ResponseWriter, r *http.Request) {
	lim := s.media.MaxBytes(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, lim+64<<10)
	if err := r.ParseMultipartForm(lim); err != nil {
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
	var pending int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*) FROM media m
		 WHERE m.kind = 'custom_photo' AND m.created_by = $1
		   AND m.created_at > now() - interval '1 day'
		   AND NOT EXISTS (SELECT 1 FROM order_custom_photos p WHERE p.media_id = m.id)`,
		uid).Scan(&pending); err != nil {
		s.respondErr(w, err)
		return
	}
	if pending >= maxPendingCustomPhotos {
		s.respondErr(w, errValidation)
		return
	}
	mm, err := s.media.Save(r.Context(), uid, "custom_photo", file)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"id":        mm.ID,
		"url":       media.SignedURL(strings.TrimPrefix(mm.URL, "/media/")),
		"thumb_url": media.SignedURL(strings.TrimPrefix(mm.ThumbURL, "/media/")),
	})
}
