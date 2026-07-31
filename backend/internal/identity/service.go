package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notify"
)

// أخطاء المجال — بصيغة الـAPI الموحدة ومفاتيح الترجمة المركزية.
var (
	ErrInvalidPhone       = httpx.NewError(http.StatusBadRequest, "invalid_phone", "errors.invalid_phone")
	ErrOTPRateLimited     = httpx.NewError(http.StatusTooManyRequests, "rate_limited", "errors.rate_limited")
	ErrOTPInvalid         = httpx.NewError(http.StatusUnauthorized, "invalid_otp", "auth.otpInvalid")
	ErrInvalidCredentials = httpx.NewError(http.StatusUnauthorized, "invalid_credentials", "errors.invalid_credentials")
	ErrUserBlocked        = httpx.NewError(http.StatusForbidden, "user_blocked", "errors.user_blocked")
	ErrUserSuspended      = httpx.NewError(http.StatusForbidden, "user_suspended", "errors.user_suspended")
	ErrInvalidRefresh     = httpx.NewError(http.StatusUnauthorized, "invalid_refresh", "errors.unauthorized")
	ErrWeakPassword       = httpx.NewError(http.StatusBadRequest, "weak_password", "errors.weak_password")
	ErrOTPSendFailed      = httpx.NewError(http.StatusServiceUnavailable, "otp_send_failed", "errors.otp_send_failed")
)

const (
	otpTTL        = 5 * time.Minute
	otpMaxPer15m  = 3
	refreshTTL    = 30 * 24 * time.Hour
	minPasswordLn = 8
)

type Service struct {
	repo   *Repo
	rdb    *redis.Client
	tokens *auth.TokenIssuer
	sender notify.OTPSender
	secret string // لتجزئة رموز OTP (HMAC)
	logger *slog.Logger
}

func NewService(repo *Repo, rdb *redis.Client, tokens *auth.TokenIssuer, sender notify.OTPSender, secret string, logger *slog.Logger) *Service {
	return &Service{repo: repo, rdb: rdb, tokens: tokens, sender: sender, secret: secret, logger: logger}
}

func (s *Service) hashOTP(phone, code string) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(phone + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

// RequestOTP يولّد رمزاً ويرسله عبر القناة المهيأة، مع حد معدل لكل رقم.
func (s *Service) RequestOTP(ctx context.Context, rawPhone string) error {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return ErrInvalidPhone
	}

	key := "otp:req:" + phone
	n, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	if n == 1 {
		s.rdb.Expire(ctx, key, 15*time.Minute)
	}
	if n > otpMaxPer15m {
		return ErrOTPRateLimited
	}

	code, err := randomDigits(6)
	if err != nil {
		return err
	}
	if err := s.repo.CreateOTP(ctx, phone, s.hashOTP(phone, code), "login", otpTTL); err != nil {
		return err
	}
	if err := s.sender.SendOTP(ctx, phone, code); err != nil {
		s.logger.Error("otp send failed", "error", err)
		return ErrOTPSendFailed
	}
	return nil
}

// RequestPhoneChange يرسل رمز تحقق إلى الرقم الجديد لتأكيد تغيير رقم الحساب —
// لأي مستخدم لنفسه. الرقم الجديد يجب ألّا يكون مملوكاً لحساب آخر.
func (s *Service) RequestPhoneChange(ctx context.Context, userID, rawPhone string) error {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return ErrInvalidPhone
	}
	if u, _, err := s.repo.UserByPhone(ctx, phone); err == nil && u.ID != userID {
		return ErrPhoneTaken
	}
	key := "otp:chg:" + phone
	n, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	if n == 1 {
		s.rdb.Expire(ctx, key, 15*time.Minute)
	}
	if n > otpMaxPer15m {
		return ErrOTPRateLimited
	}
	code, err := randomDigits(6)
	if err != nil {
		return err
	}
	if err := s.repo.CreateOTP(ctx, phone, s.hashOTP(phone, code), "phone_change", otpTTL); err != nil {
		return err
	}
	if err := s.sender.SendOTP(ctx, phone, code); err != nil {
		s.logger.Error("otp send failed", "error", err)
		return ErrOTPSendFailed
	}
	return nil
}

// ConfirmPhoneChange يتحقق من رمز الرقم الجديد ويحدّث رقم الحساب.
func (s *Service) ConfirmPhoneChange(ctx context.Context, userID, rawPhone, code, ip string) error {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return ErrInvalidPhone
	}
	if u, _, err := s.repo.UserByPhone(ctx, phone); err == nil && u.ID != userID {
		return ErrPhoneTaken
	}
	valid, err := s.repo.ConsumeOTP(ctx, phone, s.hashOTP(phone, code), "phone_change")
	if err != nil {
		return err
	}
	if !valid {
		return ErrOTPInvalid
	}
	if err := s.repo.SetPhone(ctx, userID, phone); err != nil {
		return err
	}
	s.repo.Audit(ctx, &userID, "auth.phone_change", "user", userID, ip, nil)
	return nil
}

// VerifyOTP يتحقق من الرمز؛ ينشئ حساب زبون تلقائياً للرقم الجديد، ويصدر التوكنات.
func (s *Service) VerifyOTP(ctx context.Context, rawPhone, code, userAgent, ip string) (*AuthResult, error) {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return nil, ErrInvalidPhone
	}

	valid, err := s.repo.ConsumeOTP(ctx, phone, s.hashOTP(phone, code), "login")
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrOTPInvalid
	}

	user, _, err := s.repo.UserByPhone(ctx, phone)
	if errors.Is(err, ErrNotFound) {
		user, err = s.repo.CreateUserWithRole(ctx, phone, "", "customer")
		if err == nil {
			s.repo.Audit(ctx, &user.ID, "user.register", "user", user.ID, ip, nil)
		}
	}
	if err != nil {
		return nil, err
	}
	return s.issueFor(ctx, user, userAgent, ip, "auth.otp_login")
}

// LoginPassword دخول بكلمة المرور (للموظفين والأدوار التشغيلية غالباً).
func (s *Service) LoginPassword(ctx context.Context, rawPhone, password, userAgent, ip string) (*AuthResult, error) {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return nil, ErrInvalidPhone
	}
	user, hash, err := s.repo.UserByPhone(ctx, phone)
	if errors.Is(err, ErrNotFound) || (err == nil && hash == "") {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	match, err := auth.VerifyPassword(password, hash)
	if err != nil {
		return nil, err
	}
	if !match {
		s.repo.Audit(ctx, nil, "auth.password_failed", "user", user.ID, ip, nil)
		return nil, ErrInvalidCredentials
	}
	return s.issueFor(ctx, user, userAgent, ip, "auth.password_login")
}

// IssueForUserID يصدر جلسة لمستخدم بمعرّفه (لتسليم SSO عبر رمز موثوق لمرّة واحدة).
func (s *Service) IssueForUserID(ctx context.Context, userID, userAgent, ip string) (*AuthResult, error) {
	user, _, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.issueFor(ctx, user, userAgent, ip, "auth.sso")
}

func (s *Service) issueFor(ctx context.Context, user *User, userAgent, ip, action string) (*AuthResult, error) {
	if user.Status == "suspended" {
		return nil, ErrUserSuspended
	}
	if user.Status != "active" {
		return nil, ErrUserBlocked
	}
	access, exp, err := s.tokens.IssueAccess(user.ID, user.Roles)
	if err != nil {
		return nil, err
	}
	rawRefresh, refreshHash, err := auth.NewOpaqueToken()
	if err != nil {
		return nil, err
	}
	// جلسة واحدة فقط لكل حساب: أحدث دخول يُبطل كل الجلسات السابقة. عند التدجيل
	// يكون القديم أُبطل قبلاً فلا يبقى نشط سوى الجديد. (يمهّد للتحقق بخطوتين لاحقاً.)
	if _, err := s.repo.RevokeAllTokens(ctx, user.ID); err != nil {
		return nil, err
	}
	if err := s.repo.StoreRefresh(ctx, user.ID, refreshHash, refreshTTL, userAgent, ip); err != nil {
		return nil, err
	}
	s.repo.Audit(ctx, &user.ID, action, "user", user.ID, ip, nil)
	return &AuthResult{
		User: *user,
		Tokens: TokenPair{
			AccessToken:     access,
			AccessExpiresAt: exp,
			RefreshToken:    rawRefresh,
		},
	}, nil
}

// Refresh يدوّر توكن التحديث: يبطل القديم ويصدر زوجاً جديداً.
func (s *Service) Refresh(ctx context.Context, rawRefresh, userAgent, ip string) (*AuthResult, error) {
	userID, err := s.repo.RevokeRefresh(ctx, auth.HashToken(rawRefresh))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidRefresh
	}
	if err != nil {
		return nil, err
	}
	user, _, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.issueFor(ctx, user, userAgent, ip, "auth.refresh")
}

func (s *Service) Logout(ctx context.Context, rawRefresh, ip string) error {
	userID, err := s.repo.RevokeRefresh(ctx, auth.HashToken(rawRefresh))
	if errors.Is(err, ErrNotFound) {
		return nil // خروج توكن ميت = نجاح صامت
	}
	if err == nil {
		s.repo.Audit(ctx, &userID, "auth.logout", "user", userID, ip, nil)
	}
	return err
}

func (s *Service) Me(ctx context.Context, userID string) (*User, error) {
	user, _, err := s.repo.UserByID(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return nil, httpx.ErrNotFound
	}
	return user, err
}

// SetOwnAvatar يضبط صورة المستخدم لنفسه (mediaID فارغ = إزالة).
func (s *Service) SetOwnAvatar(ctx context.Context, userID, mediaID string) error {
	return s.repo.SetAvatar(ctx, userID, mediaID)
}

func (s *Service) SetPassword(ctx context.Context, userID, password, currentPassword, ip string) error {
	if len(password) < minPasswordLn {
		return ErrWeakPassword
	}
	// من يملك كلمة مرور يجب أن يثبتها قبل تغييرها (صفحة "حسابي")
	_, hash, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if hash != "" {
		ok, err := auth.VerifyPassword(currentPassword, hash)
		if err != nil || !ok {
			return ErrInvalidCredentials
		}
	}
	newHash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if err := s.repo.SetPassword(ctx, userID, newHash); err != nil {
		return err
	}
	s.repo.Audit(ctx, &userID, "auth.set_password", "user", userID, ip, nil)
	return nil
}

// SalesRepByInviteCode يعيد المندوب صاحب كود الدعوة — أو خطأ واضحاً إن لم يوجد.
var ErrInvalidInviteCode = httpx.NewError(http.StatusBadRequest, "invalid_invite_code", "errors.invalid_invite_code")

func (s *Service) SalesRepByInviteCode(ctx context.Context, code string) (*User, error) {
	user, _, err := s.repo.UserByInviteCode(ctx, code)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidInviteCode
	}
	if err != nil {
		return nil, err
	}
	for _, r := range user.Roles {
		if r == "sales" {
			return user, nil
		}
	}
	return nil, ErrInvalidInviteCode
}

// EnsureUserWithRole يجد المستخدم برقم هاتفه (أو ينشئه) ويضمن حمله الدور المطلوب.
// تستخدمه الوحدات الأخرى لربط الحسابات (صاحب متجر، سائق...) — مع تدقيق كامل.
func (s *Service) EnsureUserWithRole(ctx context.Context, actorID, rawPhone, role, ip string) (*User, error) {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return nil, ErrInvalidPhone
	}
	user, _, err := s.repo.UserByPhone(ctx, phone)
	if errors.Is(err, ErrNotFound) {
		user, err = s.repo.CreateUserWithRole(ctx, phone, "", role)
		if err == nil {
			s.repo.Audit(ctx, &actorID, "admin.user_create", "user", user.ID, ip,
				map[string]any{"phone": phone, "roles": []string{role}})
		}
		return user, err
	}
	if err != nil {
		return nil, err
	}
	if err := s.repo.GrantRole(ctx, user.ID, role, &actorID); err != nil {
		return nil, err
	}
	return user, nil
}

// BootstrapAdmin يضمن وجود حساب أدمن أول (يُستدعى عند الإقلاع بهاتف من الإعدادات).
func (s *Service) BootstrapAdmin(ctx context.Context, rawPhone string) error {
	if rawPhone == "" {
		return nil
	}
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return fmt.Errorf("identity: invalid ADMIN_PHONE %q", rawPhone)
	}
	user, _, err := s.repo.UserByPhone(ctx, phone)
	if errors.Is(err, ErrNotFound) {
		user, err = s.repo.CreateUserWithRole(ctx, phone, "", "admin")
		if err == nil {
			s.logger.Info("bootstrap admin created", "phone", phone)
		}
		return err
	}
	if err != nil {
		return err
	}
	return s.repo.GrantRole(ctx, user.ID, "admin", nil)
}

func randomDigits(n int) (string, error) {
	out := make([]byte, n)
	for i := range out {
		d, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		out[i] = byte('0' + d.Int64())
	}
	return string(out), nil
}

// ActiveStatus يعيد حالة الحساب مع كاش Redis قصير (30ث) — للإنفاذ في الوسيط
// دون إثقال كل طلب بقاعدة البيانات. يعيد "active" افتراضياً عند أي تعذّر
// (لا نقفل النظام بسبب عطل كاش)، والإيقاف يمسح الكاش فيسري خلال 30ث كحد أقصى.
func (s *Service) ActiveStatus(ctx context.Context, userID string) string {
	key := "ustatus:" + userID
	if v, err := s.rdb.Get(ctx, key).Result(); err == nil {
		return v
	}
	var status string
	if err := s.repo.pool().QueryRow(ctx,
		`SELECT status FROM users WHERE id = $1`, userID).Scan(&status); err != nil {
		return "active"
	}
	s.rdb.Set(ctx, key, status, 30*time.Second)
	return status
}

func (s *Service) invalidateStatusCache(ctx context.Context, userID string) {
	s.rdb.Del(ctx, "ustatus:"+userID)
}
