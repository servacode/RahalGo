package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var errValidation = httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")

func decode[T any](r *http.Request) (*T, error) {
	var v T
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(&v); err != nil {
		return nil, errValidation
	}
	return &v, nil
}

// respondErr يحول أخطاء المجال إلى استجابة موحدة، ويخفي التفاصيل الداخلية.
func (s *Server) respondErr(w http.ResponseWriter, err error) {
	var appErr *httpx.AppError
	if errors.As(err, &appErr) {
		httpx.Error(w, appErr)
		return
	}
	s.logger.Error("internal error", "error", err)
	httpx.Error(w, httpx.ErrInternal)
}

func (s *Server) handleOTPRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.RequestOTP(r.Context(), req.Phone); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}

func (s *Server) handleOTPVerify(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.VerifyOTP(r.Context(), req.Phone, req.Code, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.LoginPassword(r.Context(), req.Phone, req.Password, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		RefreshToken string `json:"refresh_token"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.Refresh(r.Context(), req.RefreshToken, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		RefreshToken string `json:"refresh_token"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.Logout(r.Context(), req.RefreshToken, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"logged_out": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := s.identity.Me(r.Context(), userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (s *Server) handleSetPassword(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Password        string `json:"password"`
		CurrentPassword string `json:"current_password"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.SetPassword(r.Context(), userIDFrom(r), req.Password, req.CurrentPassword, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func clientIP(r *http.Request) string {
	return r.RemoteAddr
}

// handleMyLogins آخر دخولات الحساب — للمستخدم نفسه (شفافية أمان "هل كان هذا أنت؟").
func (s *Server) handleMyLogins(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT action, COALESCE(ip, ''), created_at
		FROM audit_log
		WHERE actor_user_id = $1
		  AND action IN ('auth.otp_login', 'auth.password_login', 'auth.password_failed')
		ORDER BY id DESC LIMIT 20`, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type login struct {
		Action    string    `json:"action"`
		IP        string    `json:"ip"`
		CreatedAt time.Time `json:"created_at"`
	}
	out := []login{}
	for rows.Next() {
		var l login
		if err := rows.Scan(&l.Action, &l.IP, &l.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, l)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// --- استعادة كلمة المرور: رمز على الهاتف ثم كلمة مرور جديدة ---

func (s *Server) handleResetRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.RequestPasswordReset(r.Context(), req.Phone); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}

func (s *Server) handleResetConfirm(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone    string `json:"phone"`
		Code     string `json:"code"`
		Password string `json:"password"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.ConfirmPasswordReset(r.Context(), req.Phone, req.Code, req.Password, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// --- إنشاء حساب زبون: رمز تأكيد ثم اسم وكلمة مرور. الزبون فقط، لا دور آخر. ---

func (s *Server) handleSignupRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.RequestSignup(r.Context(), req.Phone); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}

func (s *Server) handleSignupConfirm(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone    string `json:"phone"`
		Code     string `json:"code"`
		FullName string `json:"full_name"`
		Password string `json:"password"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, err := s.identity.ConfirmSignup(r.Context(), req.Phone, req.Code,
		strings.TrimSpace(req.FullName), req.Password, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// handleHandoff ينشئ رمز تسليم لمرّة واحدة (SSO): يفتح المستخدم تطبيقاً آخر مسجّلاً
// بلا كلمة مرور. الرمز قصير العمر (60ث) ويُستهلك مرّة واحدة.
func (s *Server) handleHandoff(w http.ResponseWriter, r *http.Request) {
	code, _, err := auth.NewOpaqueToken()
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// نمرّر عائلة جلسة المصدر مع الرمز: الانتقال بين لوحاتنا امتداد لنفس الجلسة،
	// فلا تُبطل اللوحة الجديدة جلسة اللوحة التي جاء منها.
	uid := userIDFrom(r)
	payload := uid + "|" + s.identity.ActiveSessionID(r.Context(), uid)
	if err := s.rdb.Set(r.Context(), "sso:"+code, payload, 60*time.Second).Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"code": code})
}

// handleSSO يستبدل رمز التسليم بجلسة (تسجيل دخول بلا كلمة مرور عبر رمز موثوق).
func (s *Server) handleSSO(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Code string `json:"code"`
	}](r)
	if err != nil || req.Code == "" {
		s.respondErr(w, errValidation)
		return
	}
	payload, err := s.rdb.GetDel(r.Context(), "sso:"+req.Code).Result() // استهلاك لمرّة واحدة
	if err != nil || payload == "" {
		s.respondErr(w, errUnauthorized)
		return
	}
	uid, sessionID, _ := strings.Cut(payload, "|")
	if uid == "" {
		s.respondErr(w, errUnauthorized)
		return
	}
	res, err := s.identity.IssueForUserID(r.Context(), uid, sessionID, r.UserAgent(), clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// handlePhoneChangeRequest يرسل رمزاً للرقم الجديد لتأكيد تغيير رقم الحساب.
func (s *Server) handlePhoneChangeRequest(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.RequestPhoneChange(r.Context(), userIDFrom(r), req.Phone); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true})
}

// handlePhoneChangeConfirm يتحقق من الرمز ويحدّث رقم الحساب.
func (s *Server) handlePhoneChangeConfirm(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.identity.ConfirmPhoneChange(r.Context(), userIDFrom(r), req.Phone, req.Code, clientIP(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}
