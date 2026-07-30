package server

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type ctxKey string

const (
	ctxUserID ctxKey = "user_id"
	ctxRoles  ctxKey = "roles"
)

var (
	errUnauthorized = httpx.NewError(http.StatusUnauthorized, "unauthorized", "errors.unauthorized")
	errForbidden    = httpx.NewError(http.StatusForbidden, "forbidden", "errors.forbidden")
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
		ctx := context.WithValue(r.Context(), ctxUserID, claims.Subject)
		ctx = context.WithValue(ctx, ctxRoles, claims.Roles)
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
