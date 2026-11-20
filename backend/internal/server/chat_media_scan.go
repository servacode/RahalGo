package server

// ══════════════════════════════════════════════════════════════════════
// **فلترة صور وتسجيلات الدردشة** — قرارُ المالك ٢٠٢٦-١١-٢٠
// ══════════════════════════════════════════════════════════════════════
//
//	الصورة    السيرفر بيقرا الأرقام اللي فيها قبل ما توصل. فيها رقم جوال ⇒ **بتنحجز** (ما بتوصل للطرف
//	          التاني)، وبيطلع للمرسل «الصورة فيها رقم وما انبعتت»، وتنبيه للإدارة. والمكتب بيمرّرها.
//	التسجيل   بيوصل فوراً، وبالخلفية بيتحوّل لكلام مكتوب. انمسك فيه رقم ⇒ تنبيه للإدارة + إنذار
//	          تلقائي عالمرسل. وطوله بيضل دقيقة (قرار المالك).
//
// **وكله على سيرفرنا** (internal/mediascan). أداة مو مركّبة ⇒ الصورة والتسجيل بيمشوا متل قبل.

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/comms"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/mediascan"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/textguard"
)

// voiceScanSlots **تسجيل واحد بالمرة** — التحويل تقيل عالمعالج، والطلبات ما لازم تتأخر.
var voiceScanSlots = make(chan struct{}, 1)

// SetMediaScanner **للاختبار** — ماسحٌ وهميّ بدل الأدوات الحقيقية.
func (s *Server) SetMediaScanner(sc mediascan.Scanner) { s.scan = sc }

func (s *Server) mediaFilePath(url string) string {
	return filepath.Join(s.media.Dir(), filepath.FromSlash(strings.TrimPrefix(url, "/media/")))
}

// scanChatPhoto **بيقرا الصورة قبل ما تنبعت** — يُرجع ما قُرئ والأرقامَ الممسوكة.
func (s *Server) scanChatPhoto(ctx context.Context, url string) (string, []textguard.ContactHit) {
	if s.scan == nil || !s.scan.PhotoReady() {
		return "", nil
	}
	text, err := s.scan.OCR(ctx, s.mediaFilePath(url))
	if err != nil {
		s.logger.Warn("الدردشة: تعذّرت قراءة الصورة — بتنبعت متل ما هي", "error", err)
		return "", nil
	}
	return text, mediascan.PhoneHits(text)
}

// queueVoiceScan **بيحوّل التسجيل لكلام بالخلفية** بعد ما يوصل.
func (s *Server) queueVoiceScan(p *comms.Permission, messageID, url string) {
	if s.scan == nil || !s.scan.VoiceReady() {
		return
	}
	pp := *p
	go func() {
		voiceScanSlots <- struct{}{}
		defer func() { <-voiceScanSlots }()
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer cancel()
		s.ScanVoiceMessage(ctx, &pp, messageID, url)
	}()
}

// ScanVoiceMessage **التحويل والفحص** — ومُصدَّرة للاختبار.
func (s *Server) ScanVoiceMessage(ctx context.Context, p *comms.Permission, messageID, url string) {
	text, err := s.scan.Transcribe(ctx, s.mediaFilePath(url))
	if err != nil {
		s.logger.Warn("الدردشة: تعذّر تحويل التسجيل لكلام", "error", err)
		return
	}
	if err := s.comms.SetScanText(ctx, messageID, clip(text, 2000)); err != nil {
		s.logger.Warn("الدردشة: تعذّر حفظ كلام التسجيل", "error", err)
	}
	hits := mediascan.PhoneHits(text)
	if len(hits) == 0 {
		return
	}
	if err := s.comms.FlagContact(ctx, messageID, "تسجيل صوتي: "+textguard.ContactSummary(hits)); err != nil {
		s.logger.Warn("الدردشة: تعذّر وسم التسجيل", "error", err)
	}
	s.alertContactLeak(ctx, p)
	s.systemWarning(ctx, p.SelfID, p.OrderID, "رقم جوال بتسجيل صوتي بدردشة "+orderRef(p.Number))
}

// systemWarning **إنذار تلقائي من النظام** — نفس جدول الإنذارات، بلا مُصدِر.
func (s *Server) systemWarning(ctx context.Context, userID, orderID, note string) {
	var role string
	if err := s.pg.QueryRow(ctx, `
		SELECT COALESCE((SELECT role_code FROM user_roles WHERE user_id = $1
		                 ORDER BY granted_at LIMIT 1), 'customer')`, userID).Scan(&role); err != nil {
		s.logger.Warn("الإنذار التلقائي: تعذّرت قراءة الدور", "error", err)
		return
	}
	if _, err := s.pg.Exec(ctx, `
		INSERT INTO warnings (user_id, role_code, reason, note, order_id)
		VALUES ($1, $2, 'terms_violation', $3, $4)`, userID, role, clip(note, 500), orderID); err != nil {
		s.logger.Warn("الإنذار التلقائي: تعذّرت كتابته", "error", err)
		return
	}
	if s.notify != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: userID, Kind: notifications.KindAccount,
			Title: notifTitles.warningOnYou, Body: clip("إرسال رقم الجوال بالدردشة ممنوع — "+note, 200),
			Entity: "user", EntityID: userID,
			Apps: s.payeeWorkerApps(ctx, userID),
		})
	}
	s.touch("user", "ops")
}

// handleReleaseHeldChatPhoto **«مرّرها»** — المكتب شاف الصورة المحجوزة ومرّرها للطرف التاني.
func (s *Server) handleReleaseHeldChatPhoto(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	orderID, _, err := s.comms.Release(r.Context(), id, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.chat_photo_released", "order", orderID, map[string]any{"message_id": id})
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"released": true})
}
