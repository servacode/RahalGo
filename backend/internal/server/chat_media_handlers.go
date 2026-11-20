package server

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/comms"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/textguard"
)

// handleSendOrderAttachment **صورةٌ أو رسالةٌ صوتيّة في حديث الطلب** (قرارُ المالك ٢٠٢٦-١١-١٢).
//
// `POST /orders/{id}/messages/attachment` — `multipart`: `file` و`kind` (`photo` | `audio`)
// و`duration_ms` للصوت و`body` تعليقٌ اختياريّ. **والبابُ بابُ النصّ نفسُه**: `Permit` تردّ ٤٠٤
// لمن ليس طرفاً، والقناةُ المغلقةُ تُرفض، وفاصلُ المعدّل نفسُه — **وكلُّه قبل أن يُكتب ملفّ.**
func (s *Server) handleSendOrderAttachment(w http.ResponseWriter, r *http.Request) {
	lim := s.media.MaxBytes(r.Context())
	if lim < media.MaxAudioBytes {
		lim = media.MaxAudioBytes
	}
	r.Body = http.MaxBytesReader(w, r.Body, lim+64<<10)
	if err := r.ParseMultipartForm(lim); err != nil {
		s.respondErr(w, media.ErrTooLarge)
		return
	}
	kind := r.FormValue("kind")
	if kind != comms.AttachPhoto && kind != comms.AttachAudio {
		s.respondErr(w, errValidation)
		return
	}
	durationMs := 0
	if kind == comms.AttachAudio {
		d, err := strconv.Atoi(strings.TrimSpace(r.FormValue("duration_ms")))
		if err != nil || d <= 0 {
			s.respondErr(w, errValidation)
			return
		}
		if d > comms.MaxAudioMs {
			// **وتسجيلٌ توقّف عند الدقيقة قد يزيد أجزاءً من الثانية** — يُقصّ لا يُرفض.
			if d > comms.MaxAudioMs+1000 {
				s.respondErr(w, media.ErrAudioTooLarge)
				return
			}
			d = comms.MaxAudioMs
		}
		durationMs = d
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		s.respondErr(w, errValidation)
		return
	}
	defer file.Close()

	// **والتعليقُ يمرّ بحارس النصّ نفسِه** — الشتيمةُ تُخفى والرقمُ يُخفى.
	caption := r.FormValue("body")
	masked, err := s.guardText(r.Context(), tf("body", &caption, maxChatBody, textguard.Chat))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	caption, contacts := textguard.MaskContacts(caption, "***")

	p, err := s.comms.Permit(r.Context(), chi.URLParam(r, "id"), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.comms.PreflightAttachment(r.Context(), p); err != nil {
		s.respondErr(w, err)
		return
	}

	in := comms.AttachmentInput{Kind: kind, DurationMs: durationMs}
	var photoHits []textguard.ContactHit
	if kind == comms.AttachPhoto {
		mm, err := s.media.Save(r.Context(), p.SelfID, "chat_photo", file)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		in.MediaID, in.Path, in.Thumb = mm.ID, mm.URL, mm.ThumbURL
		// **فيها رقم جوال؟** (٢٠٢٦-١١-٢٠) ⇒ بتنحجز وما بتوصل.
		if text, hits := s.scanChatPhoto(r.Context(), mm.URL); len(hits) > 0 {
			in.Held, in.ScanText = true, clip(text, 2000)
			photoHits = hits
		}
	} else {
		raw, err := io.ReadAll(io.LimitReader(file, media.MaxAudioBytes+1))
		if err != nil {
			s.respondErr(w, err)
			return
		}
		mm, err := s.media.SaveAudio(r.Context(), p.SelfID, raw)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		in.MediaID, in.Path = mm.ID, mm.URL
	}

	msg, err := s.comms.SendAttachment(r.Context(), p, caption, in)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **الصورة المحجوزة**: موسومة للإدارة، وتنبيه، والمرسل بيعرف إنها ما انبعتت — ولا إشعار للطرف التاني.
	if in.Held {
		if err := s.comms.FlagContact(r.Context(), msg.ID, "صورة فيها رقم: "+textguard.ContactSummary(photoHits)); err != nil {
			s.logger.Warn("الحديث: تعذّر وسمُ صورةٍ محجوزة", "error", err)
		}
		s.alertContactLeak(r.Context(), p)
		s.touch("order", "ops")
		s.respondErr(w, comms.ErrPhotoHeld)
		return
	}
	if len(masked) > 0 {
		if err := s.comms.Flag(r.Context(), msg.ID, masked[0]); err != nil {
			s.logger.Warn("الحديث: تعذّر وسمُ لفظٍ مسيء", "error", err)
		}
		s.alertOffensive(r.Context(), masked, orderRef(p.Number), "order", p.OrderID, "/dashboard/orders")
	}
	if len(contacts) > 0 {
		if err := s.comms.FlagContact(r.Context(), msg.ID, textguard.ContactSummary(contacts)); err != nil {
			s.logger.Warn("الحديث: تعذّر وسمُ رقمٍ مخفيّ", "error", err)
		}
		s.alertContactLeak(r.Context(), p)
	}
	// **والإشعارُ يقول ما وصل** — «📷 صورة» أو «🎤 رسالة صوتية».
	s.notifyChatPeer(r.Context(), p, comms.Fallback(kind))
	if kind == comms.AttachAudio {
		s.queueVoiceScan(p, msg.ID, in.Path)
	}
	httpx.JSON(w, http.StatusCreated, msg)
}
