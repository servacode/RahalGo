package server

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

type ctxKey string

const (
	ctxUserID ctxKey = "user_id"
	ctxRoles  ctxKey = "roles"
	// ctxSID **عائلةُ الجلسة التي حملت الطلب** — `XG-40`.
	//
	// **ويلزم حين يُستثنى «هذا الجهاز» من إبطالٍ عامّ** — **ولا
	// يُستنتَج من ترويسةٍ ولا من «آخر جلسة»**: يُقرأ من الرمز
	// الموثَّق نفسِه.
	ctxSID ctxKey = "session_id"
)

var (
	errUnauthorized = httpx.NewError(http.StatusUnauthorized, "unauthorized", "errors.unauthorized")

	// errAuthUnavailable **تعذّر التحقّقُ من الجلسة** — `R16`.
	//
	// **و٥٠٣ لا ٤٠١**: **«لا أستطيع التحقّق» غيرُ «رمزُك مُبطَل»** —
	// **والثاني كذبٌ يُخرج صاحبَ جلسةٍ سليمةٍ ويطلب منه دخولاً جديداً
	// لن ينفعه.** **والأوّلُ يقول «عاود» وهو الصدق.**
	errAuthUnavailable = httpx.NewError(http.StatusServiceUnavailable,
		"auth_unavailable", "errors.auth_unavailable")
	errForbidden = httpx.NewError(http.StatusForbidden, "forbidden", "errors.forbidden")
)

// RequireAuth يتحقق من توكن الوصول ويحقن الهوية والأدوار في السياق.
func (s *Server) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			httpx.Error(w, errUnauthorized)
			return
		}
		claims, err := s.tokens.VerifyAccess(token)
		if err != nil {
			httpx.Error(w, errUnauthorized)
			return
		}
		// إنفاذ حالة الحساب: توقيع التوكن سليم لا يكفي — الموقوف/المحظور
		// يُرفض فوراً (بكاش Redis قصير) حتى لو بقيت صلاحية توكنه.
		// ══════════════════════════════════════════════════════════
		// **والتعليقُ يمنع الجديدَ ولا يشلّ القائم** — `XG-22`
		// ══════════════════════════════════════════════════════════
		//
		// **كان يردّ `403` على كلّ نداءٍ فورَ التعليق** — **فسائقٌ
		// عُلِّق وهو يحمل طلباً لا يستطيع تسليمَه**، والطلبُ يبقى
		// معلَّقاً بمن لا يقدر والزبونُ ينتظر.
		//
		// **و`blocked` بابٌ آخر**: يقف عند كلّ شيءٍ ولا استثناءَ فيه.
		// **والاستثناءُ للعاديّ وحدَه، وبشروطٍ ثلاثةٍ مجتمعة** — انظر
		// `suspension.go`.
		switch s.identity.ActiveStatus(r.Context(), claims.Subject) {
		case "active":
		case "suspended":
			if !s.suspendedMayContinue(r.Context(), r, claims.Subject, claims.Roles) {
				httpx.Error(w, errForbidden)
				return
			}
		default:
			httpx.Error(w, errForbidden)
			return
		}
		// جلسة أُنهيت لا يبقى توكنها صالحاً حتى انتهاء مهلته: الخروج فوري
		// على كل تطبيقات المنصة لا على التطبيق الذي طلبه وحده.
		// ══════════════════════════════════════════════════════════
		// **وغيابُ خبرٍ ليس خبراً بالسلامة** — `R16`
		// ══════════════════════════════════════════════════════════
		//
		// **كانت `SessionRevoked` تردّ `bool`** — **فخطأُ `Redis`
		// يُقرأ «ليست مُبطَلة»**، **وجلسةٌ أُبطلت تعود تعمل بسقوط
		// خبيئة.**
		//
		// **والحالُ ثلاثٌ لا اثنتان**، **والثالثةُ ٥٠٣.**
		state, dbRoles, err := s.identity.CheckSession(r.Context(), claims.SID)
		switch {
		case err != nil:
			s.logger.Error("التوثيق: تعذّر التحقّقُ من الجلسة",
				"outcome", "auth_validation_unavailable", "error", err)
			httpx.Error(w, errAuthUnavailable)
			return
		case state == identity.SessionRevoked:
			httpx.Error(w, errUnauthorized)
			return
		}
		// ══════════════════════════════════════════════════════════
		// **والتخويلُ من الحقيقة الموثوقة لا من الرمز** — `R15`
		// ══════════════════════════════════════════════════════════
		//
		// **كانت `ctxRoles` تُملأ من `claims.Roles`** — **فدورٌ سُحب
		// يبقى نافذاً حتّى تنتهي مهلةُ الرمز**، ربعَ ساعة.
		// **ومن سُحب منه دورُ المكتب يبقى فيه.**
		//
		// **والقاعدةُ تُسأل هنا أصلاً منذ `R16`** — **فأدوارُ اللحظة
		// تأتي مع الجواب بلا استعلامٍ ثانٍ ولا خبيئةٍ تُخترَع.**
		//
		// **والتوثيقُ غيرُ التخويل**: **الجلسةُ تبقى صالحةً**
		// والصلاحيّةُ المسحوبةُ تقف. **ولا تُبطَل جلسةٌ لأنّ دوراً
		// تبدّل.**
		//
		// **وصنفُ التوكنات بلا معرّفِ جلسةٍ يبقى على ادّعاءاته** —
		// **لا جلسةَ له تُسأل عنها**، وهو مُعلَنٌ في `CheckSession`.
		roles := claims.Roles
		if claims.SID != "" {
			roles = dbRoles
		}
		ctx := context.WithValue(r.Context(), ctxUserID, claims.Subject)
		ctx = context.WithValue(ctx, ctxRoles, roles)
		ctx = context.WithValue(ctx, ctxSID, claims.SID)
		s.touchPresence(claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRoles يسمح فقط لمن يحمل أحد الأدوار المذكورة (يُستخدم بعد RequireAuth).
func (s *Server) RequireRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles, _ := r.Context().Value(ctxRoles).([]string)
			for _, want := range roles {
				if slices.Contains(userRoles, want) {
					next.ServeHTTP(w, r)
					return
				}
			}
			httpx.Error(w, errForbidden)
		})
	}
}

func userIDFrom(r *http.Request) string {
	id, _ := r.Context().Value(ctxUserID).(string)
	return id
}

// touchPresence يحدّث "آخر ظهور" مع النشاط الموثق — بتهدئة 60 ثانية
// كي لا يثقل كل طلب بقاعدة البيانات.
func (s *Server) touchPresence(userID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = s.pg.Exec(ctx, `
			UPDATE users SET last_seen_at = now()
			WHERE id = $1 AND (last_seen_at IS NULL OR last_seen_at < now() - interval '60 seconds')`,
			userID)
	}()
}

// isUUID تحقق خفيف من صيغة UUID لتفادي أخطاء 500 عند تمرير معرف مسار غير صالح.
func isUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, c := range v {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
