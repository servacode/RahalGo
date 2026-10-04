package server

// ══════════════════════════════════════════════════════════════════════
// **رسالةُ الدخول — لكلّ حسابٍ جديدٍ ولكلّ إعادةِ كلمة** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// «تصله رسالةٌ للتسجيل برقمه وكلمة السرّ… ونذكر له أنّه لازم يغيّرها» — **بشروط
// الأمان**: الكلمةُ يولّدها النظام، تنتهي بعد ٧٢ ساعة، تُبدَّل عند أوّل دخول،
// **ومعها رابطُ تطبيق دوره من مركز التنزيل.** ومن له حسابٌ قائمٌ وصار صاحبَ متجرٍ
// **لا تُولَّد له كلمة** — رسالةُ «صار عندك متجر» مع الرابط.
//
// **والكلمةُ لا تُعرض على الموظّف ولا تُكتب في سجلّ**: تخرج في الرسالة وحدَها.
// فإن تعذّر الإرسالُ (البوتُ غيرُ مقترن) قيل ذلك، **وزرُّ «إعادة إرسال» في الملفّ.**

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// appKeyForRoles **أيُّ تطبيقٍ لهذا الحساب** — بدوره الأساسيّ. فارغٌ: لوحةُ الإدارة.
func appKeyForRoles(roles []string) string {
	for _, pair := range [][2]string{{"driver", "driver"}, {"sales", "rep"},
		{"merchant", "merchant"}} {
		for _, r := range roles {
			if r == pair[0] {
				return pair[1]
			}
		}
	}
	for _, r := range roles {
		if r != "customer" {
			return "" // موظّف — اللوحة
		}
	}
	return "customer"
}

// siteBase أصلُ الموقع العامّ — أوّلُ أصلٍ مسموحٍ بلا نجمة. فارغٌ في التطوير.
func (s *Server) siteBase() string {
	if s.cfg == nil {
		return ""
	}
	for _, o := range s.cfg.WebOrigins {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o != "" && !strings.Contains(o, "*") {
			return o
		}
	}
	return ""
}

// appLink رابطُ التطبيق المناسب من «مركز التنزيل» — أو بابُ اللوحة للموظّف.
func (s *Server) appLink(appKey string) string {
	if appKey == "" {
		return s.siteBase() + "/adminrahalgo"
	}
	return s.siteBase() + "/download/" + appKey
}

// rolesOf أدوارُ الحساب من القاعدة.
func (s *Server) rolesOf(ctx context.Context, userID string) []string {
	var roles []string
	_ = s.pg.QueryRow(ctx, `SELECT COALESCE(array_agg(role_code ORDER BY role_code), '{}')
		FROM user_roles WHERE user_id = $1`, userID).Scan(&roles)
	return roles
}

// welcomeText نصُّ الرسالة — **لا شيءَ فيها يُخمَّن**: الرقمُ والكلمةُ والمهلةُ والرابط.
func welcomeText(phone, password, link string, hours int64, reset bool) string {
	head := "أهلاً بك في رحّال غو — أُنشئ حسابك."
	if reset {
		head = "رحّال غو — أُعيد ضبط كلمة مرور حسابك."
	}
	return head + "\n" +
		"الرقم: " + phone + "\n" +
		"كلمة المرور المؤقتة: " + password + "\n" +
		"تنتهي بعد " + strconv.FormatInt(hours, 10) + " ساعة إن لم تُستعمل، " +
		"وعليك تغييرها عند أول دخول.\n" +
		"حمّل التطبيق من: " + link
}

// sendText يُرسل رسالةً حرّةً بالبوت — ويقول أوصلت أم لا.
func (s *Server) sendText(ctx context.Context, phone, text string) bool {
	if !s.merchantReady() {
		return false
	}
	if err := s.merchant.SendText(ctx, phone, text); err != nil {
		s.logger.Warn("رسالةُ الحساب لم تُرسَل", "error", err)
		return false
	}
	return true
}

// issueWelcome **يولّد كلمةً مؤقّتةً ويرسلها** — لحسابٍ جديدٍ أو لإعادةِ كلمة.
func (s *Server) issueWelcome(ctx context.Context, actor, userID, ip string, reset bool) (bool, time.Time, error) {
	return s.issueWelcomeApp(ctx, actor, userID, ip, "", reset)
}

// issueWelcomeFor رسالةُ حسابٍ جديد — و`app` يختار الرابط (`panel` للموظّف).
func (s *Server) issueWelcomeFor(ctx context.Context, actor, userID, ip, app string) (bool, time.Time, error) {
	return s.issueWelcomeApp(ctx, actor, userID, ip, app, false)
}

func (s *Server) issueWelcomeApp(ctx context.Context, actor, userID, ip, app string, reset bool) (bool, time.Time, error) {
	action := "admin.welcome_issued"
	if reset {
		action = "admin.password_reset"
	}
	plain, expires, err := s.identity.IssueTempPassword(ctx, actor, userID, ip, action)
	if err != nil {
		return false, time.Time{}, err
	}
	var phone string
	if err := s.pg.QueryRow(ctx, `SELECT phone FROM users WHERE id = $1`, userID).Scan(&phone); err != nil {
		return false, expires, err
	}
	key := appKeyForRoles(s.rolesOf(ctx, userID))
	switch app {
	case "panel":
		key = ""
	case "customer", "driver", "rep", "merchant":
		key = app
	}
	link := s.appLink(key)
	sent := s.sendText(ctx, phone, welcomeText(phone, plain, link,
		s.identity.TempPasswordHours(ctx), reset))
	if sent {
		_, _ = s.pg.Exec(ctx, `UPDATE users SET welcome_sent_at = now() WHERE id = $1`, userID)
	}
	s.auditCtx(ctx, actor, ip, "admin.welcome_message", "user", userID,
		map[string]any{"sent": sent, "reset": reset, "link": link})
	return sent, expires, nil
}

// notifyNewStoreOwner **«صار عندك متجر»** — لمن له حسابٌ قائم: لا كلمةَ تُولَّد.
func (s *Server) notifyNewStoreOwner(ctx context.Context, actor, userID, storeName, ip string) bool {
	var phone string
	if err := s.pg.QueryRow(ctx, `SELECT phone FROM users WHERE id = $1`, userID).Scan(&phone); err != nil {
		return false
	}
	text := "رحّال غو — صار عندك متجر: " + storeName + "\n" +
		"ادخل بتطبيق المتجر برقمك وكلمة سرّك نفسها.\n" +
		"حمّل التطبيق من: " + s.appLink("merchant")
	sent := s.sendText(ctx, phone, text)
	s.auditCtx(ctx, actor, ip, "admin.store_owner_notified", "user", userID,
		map[string]any{"sent": sent, "store": storeName})
	return sent
}

// auditCtx قيدُ تدقيقٍ بلا طلبٍ في اليد — أفضلُ جهد.
func (s *Server) auditCtx(ctx context.Context, actor, ip, action, entity, entityID string, meta map[string]any) {
	var a *string
	if actor != "" {
		a = &actor
	}
	s.identity.AuditBestEffort(ctx, a, action, entity, entityID, ip, meta)
}

// handleAdminResendWelcome **«إعادة إرسال»** — كلمةٌ مؤقّتةٌ جديدةٌ ورسالةٌ جديدة.
func (s *Server) handleAdminResendWelcome(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	sent, expires, err := s.issueWelcome(r.Context(), userIDFrom(r), id, clientIP(r), false)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": sent, "expires_at": expires})
}
