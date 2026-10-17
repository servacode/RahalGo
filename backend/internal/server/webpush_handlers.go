package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/push"
)

// ══════════════════════════════════════════════════════════════════════
// **إشعاراتُ المتصفّح — ثلاثةُ أبواب** (قرارُ المالك ٢٠٢٦-١٠-١٧)
// ══════════════════════════════════════════════════════════════════════
//
//	GET    /public/webpush-key  المفتاحُ العامّ — وفارغٌ يعني «غيرُ مهيّأ»
//	POST   /me/webpush          اشتراكُ هذا المتصفّح لصاحب الجلسة
//	DELETE /me/webpush          إلغاؤه — عند الخروج أو بطلبه

// errWebPushDisabled **الخادمُ بلا مفاتيح** — فلا اشتراك.
var errWebPushDisabled = httpx.NewError(http.StatusServiceUnavailable,
	"webpush_disabled", "errors.webpush_disabled")

// handleWebPushKey يردّ المفتاحَ العامّ.
func (s *Server) handleWebPushKey(w http.ResponseWriter, r *http.Request) {
	key := s.push.WebPushPublicKey()
	httpx.JSON(w, http.StatusOK, map[string]any{"enabled": key != "", "public_key": key})
}

// handleWebPushSubscribe يحفظ اشتراكَ المتصفّح.
//
// **والتطبيقُ يُقرأ من الطلب ويُتحقَّق بالدور**: لوحةُ المتجر تطلب `merchant`،
// **ومن لا دورَ متجرٍ له يُرفض** — فلا يستقبل حسابٌ عاديٌّ طلباتِ متجرٍ ليس له.
func (s *Server) handleWebPushSubscribe(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
		// customer · merchant — أيُّ واجهةٍ اشتركت.
		App string `json:"app"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	app := strings.TrimSpace(req.App)
	if app == "" {
		app = push.WebAppCustomer
	}
	if app != push.WebAppCustomer && app != push.WebAppMerchant {
		s.respondErr(w, errValidation)
		return
	}
	if app == push.WebAppMerchant && !hasRole(rolesFrom(r), "merchant") {
		s.respondErr(w, errForbidden)
		return
	}
	sid, _ := r.Context().Value(ctxSID).(string)
	err = s.push.SubscribeWeb(r.Context(), userIDFrom(r), sid, app,
		strings.TrimSpace(req.Endpoint), strings.TrimSpace(req.Keys.P256dh),
		strings.TrimSpace(req.Keys.Auth), r.UserAgent())
	switch {
	case errors.Is(err, push.ErrWebPushDisabled):
		s.respondErr(w, errWebPushDisabled)
		return
	case errors.Is(err, push.ErrWebPushBadSubscription):
		s.respondErr(w, errValidation)
		return
	case err != nil:
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"subscribed": true, "app": app})
}

// handleWebPushUnsubscribe يحذف اشتراكَ المتصفّح — مقيَّداً بصاحبه.
func (s *Server) handleWebPushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Endpoint string `json:"endpoint"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.push.UnsubscribeWeb(r.Context(), userIDFrom(r), strings.TrimSpace(req.Endpoint)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"removed": true})
}
