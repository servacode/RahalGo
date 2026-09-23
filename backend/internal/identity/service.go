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
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"github.com/servacode/rahalgo/backend/internal/dbtx"

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
	ErrTooManyAttempts    = httpx.NewError(http.StatusTooManyRequests, "too_many_attempts", "errors.too_many_attempts")
	ErrNameTooShort       = httpx.NewError(http.StatusBadRequest, "name_too_short", "errors.name_too_short")

	// ErrWrongCurrentPassword **كلمةُ المرور الحاليّة خاطئة — لا غير.**
	//
	// (شكوى المالك ٢٠٢٦-٠٨-١٨: «وضعتُ كلمةَ سرٍّ غلط فظهرت رسالةٌ تقول
	//  رقمُ الهاتف أو كلمةُ المرور غيرُ صحيحة، وهذا خطأ».)
	//
	// **كان يُردّ `invalid_credentials` — وهي رسالةُ شاشة الدخول**، تجمع
	// الرقمَ والكلمةَ لأنّ الدخولَ يجمعهما ولا يُقال أيُّهما أخطأ.
	//
	// **وهنا لا رقمَ في النموذج أصلاً**: صاحبُ الحساب داخلٌ بجلسته،
	// وحقلٌ واحدٌ أخطأ. **ورسالةٌ تذكر رقمَ هاتفٍ لا يُسأل عنه تُرسل
	// صاحبَها يفحص رقمَه** — ويجدُه صحيحاً فلا يعرف ما العطب.
	ErrWrongCurrentPassword = httpx.NewError(http.StatusUnauthorized,
		"wrong_current_password", "errors.wrong_current_password")
)

const (
	// احتياطيّاتٌ تعمل بها الخدمة إن لم تُربط باللوحة. **والقيمُ الفعليّةُ في
	// الإعدادات** — قراراتُ أمانٍ يتّخذها المالك لا قراراتِ نشرٍ تنتظر مبرمجاً.
	//
	// (نُقلت الثلاثةُ الأولى إلى اللوحة 2026-08-09 بقرار المالك بعد جردِ
	//  الثوابت: «نطبّقها كلَّها».)
	otpTTL       = 5 * time.Minute     // security.otp_ttl_min
	otpMaxPer15m = 3                   // security.otp_max_per_phone
	refreshTTL   = 30 * 24 * time.Hour // security.session_days

	minPasswordLn = 8

	// حدّ محاولات الدخول بكلمة المرور. رموز التحقق كانت محمية والكلمة مفتوحة —
	// وأرقام المتاجر ظاهرة في واجهة الزبون العامة، فالتخمين كان بلا سقف.
	loginMaxPerPhone = 5  // لكل رقم — يوقف تخمين حساب بعينه
	loginMaxPerIP    = 30 // لكل عنوان — أوسع: مقهى أو مكتب يشترك فيه عدة أشخاص
	loginFailWindow  = 15 * time.Minute

	// **وحدُّ الرموز للعنوان أيضاً — لا للرقم وحدَه.**
	//
	// (كشفه فحصُ المشروع ٢٠٢٦-٠٨-٠٧.)
	//
	// كان الحدُّ لكلّ رقمٍ فقط: ثلاثةٌ في ربع ساعة. **فمن ملك مضيفاً واحداً
	// أرسل ثلاثةً لكلّ رقمٍ في البلد ولم يبلغ حدّاً قطّ** — رسائلُ لا يريدها
	// أصحابُها، ورصيدُ واتساب يُحرق، **ورقمُ المنصة يُبلَّغ عنه سبَماً
	// فيُحجب.** وهو ما لا يُستدرَك بنشرة.
	//
	// **وأوسعُ من حدّ الرقم**: مقهىً أو مكتبٌ يشترك فيه عشرة — وحدٌّ ضيّقٌ
	// يمنع من لم يُخطئ. وثلاثون طلباً في ربع ساعةٍ من عنوانٍ واحدٍ يكفي
	// عشرةَ أشخاصٍ يخطئ كلٌّ منهم مرّتين.
	otpMaxPerIP = 30
)

type Service struct {
	repo   *Repo
	rdb    *redis.Client
	tokens *auth.TokenIssuer
	sender notify.OTPSender
	secret string // لتجزئة رموز OTP (HMAC)
	logger *slog.Logger
	// setting يقرأ إعداداً عددياً من اللوحة، أو يعيد الاحتياطي.
	//
	// **دالّة لا مخزن**: لو حُقن `*settings.Store` لاعتمدت حزمةُ الهوية على
	// حزمة الإعدادات، وهي أدنى منها في الترتيب. والدالّة تكسر الاتجاه بلا
	// واجهةٍ اصطناعية.
	setting func(ctx context.Context, key string, fallback int64) int64
}

func NewService(repo *Repo, rdb *redis.Client, tokens *auth.TokenIssuer, sender notify.OTPSender, secret string, logger *slog.Logger) *Service {
	return &Service{repo: repo, rdb: rdb, tokens: tokens, sender: sender, secret: secret, logger: logger}
}

// SetSettingReader يربط الخدمة بإعدادات اللوحة (تُنادى مرّة عند الإقلاع).
//
// وبلا ربطٍ تعمل الخدمة بالاحتياطيات — فاختبارات الهوية لا تحتاج مخزناً.
func (s *Service) SetSettingReader(f func(ctx context.Context, key string, fallback int64) int64) {
	s.setting = f
}

// intSetting القيمة من اللوحة أو الاحتياطي — لا تفشل أبداً.
func (s *Service) intSetting(ctx context.Context, key string, fallback int64) int64 {
	if s.setting == nil {
		return fallback
	}
	return s.setting(ctx, key, fallback)
}

// boolSetting **مفتاحٌ منطقيٌّ يُقرأ رقماً.**
//
// **و`coerceNum` تترجم `true` إلى واحدٍ و`false` إلى صفر** — فلا
// يحتاج جسرٌ ثانٍ. (انظر `settings/settings.go`.)
func (s *Service) boolSetting(ctx context.Context, key string, fallback bool) bool {
	f := int64(0)
	if fallback {
		f = 1
	}
	return s.intSetting(ctx, key, f) != 0
}

// otpLifetime مهلةُ الرمز — من اللوحة أو الاحتياطيّ.
func (s *Service) otpLifetime(ctx context.Context) time.Duration {
	return time.Duration(s.intSetting(ctx, "security.otp_ttl_min",
		int64(otpTTL/time.Minute))) * time.Minute
}

// otpQuota سقفُ الرموز للرقم الواحد في النافذة.
func (s *Service) otpQuota(ctx context.Context) int64 {
	return s.intSetting(ctx, "security.otp_max_per_phone", otpMaxPer15m)
}

// applyForcePolicy **يُقنّع علَمَ الإجبار حين يكون الخيارُ مُطفأً.**
//
// (قرارُ المالك 2026-08-09: «الحالةُ الافتراضيّةُ غيرُ مفعّل» — خطوةٌ زائدةٌ
//
//	تجعل من يملّ يغلق التطبيقَ قبل أن يُكمل.)
//
// **ولا يُمسّ العلَمُ في القاعدة**: هو واقعةٌ لا رأي — هذه الكلمةُ وضعها طرفٌ
// ثالث. **والتقنيعُ يخفي البوّابةَ ولا يمحو الخبر**، فمن شغّل الخيارَ بعد
// شهرٍ التقط كلَّ من وُضعت كلمتُه ولم يبدّلها.
//
// **وهنا لا في الويب**: خمسُ بوّاباتٍ تقرأ الحقلَ — **ولو قرأ كلٌّ منها
// الإعدادَ بنفسه لَاختلفت واحدةٌ يوماً.**
func (s *Service) applyForcePolicy(ctx context.Context, u *User) {
	if u == nil || !u.MustChangePassword {
		return
	}
	if s.intSetting(ctx, "security.force_password_change", 0) == 0 {
		u.MustChangePassword = false
	}
}

// sessionLife طولُ الجلسة قبل أن يُطلب الدخولُ من جديد.
//
// **ولا يُقصّر جلسةً قائمة**: مهلةُ الرمز تُكتب في الصفّ يومَ يُنشأ،
// **فمن كان داخلاً بمهلةٍ قديمةٍ يُكملها** ويأخذ الجديدةَ في تجديده التالي.
func (s *Service) sessionLife(ctx context.Context) time.Duration {
	return time.Duration(s.intSetting(ctx, "security.session_days",
		int64(refreshTTL/(24*time.Hour)))) * 24 * time.Hour
}

func (s *Service) hashOTP(phone, code string) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(phone + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

// RequestOTP يولّد رمزاً ويرسله عبر القناة المهيأة، مع حد معدل لكل رقم.
func (s *Service) RequestOTP(ctx context.Context, rawPhone, ip string) error {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return ErrInvalidPhone
	}

	if err := s.noteOTPSend(ctx, phone, ip); err != nil {
		return err
	}
	key := "otp:req:" + phone
	n, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	if n == 1 {
		s.rdb.Expire(ctx, key, 15*time.Minute)
	}
	if n > s.otpQuota(ctx) {
		return ErrOTPRateLimited
	}

	code, err := randomDigits(6)
	if err != nil {
		return err
	}
	if err := s.repo.CreateOTP(ctx, phone, s.hashOTP(phone, code), "login", s.otpLifetime(ctx)); err != nil {
		return err
	}
	if err := s.sender.SendOTP(ctx, phone, code); err != nil {
		return s.otpSendError(err)
	}
	return nil
}

// RequestPhoneChange يرسل رمز تحقق إلى الرقم الجديد لتأكيد تغيير رقم الحساب —
// لأي مستخدم لنفسه. الرقم الجديد يجب ألّا يكون مملوكاً لحساب آخر.
func (s *Service) RequestPhoneChange(ctx context.Context, userID, rawPhone, ip string) error {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return ErrInvalidPhone
	}
	if u, _, err := s.repo.UserByPhone(ctx, phone); err == nil && u.ID != userID {
		return ErrPhoneTaken
	}
	if err := s.noteOTPSend(ctx, phone, ip); err != nil {
		return err
	}
	key := "otp:chg:" + phone
	n, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	if n == 1 {
		s.rdb.Expire(ctx, key, 15*time.Minute)
	}
	if n > s.otpQuota(ctx) {
		return ErrOTPRateLimited
	}
	code, err := randomDigits(6)
	if err != nil {
		return err
	}
	if err := s.repo.CreateOTP(ctx, phone, s.hashOTP(phone, code), "phone_change", s.otpLifetime(ctx)); err != nil {
		return err
	}
	if err := s.sender.SendOTP(ctx, phone, code); err != nil {
		return s.otpSendError(err)
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

// noteOTPSend يحسب حدَّي الإرسال: للرقم وللعنوان.
//
// **والعنوانُ يُعدّ قبل الرقم**: من يقصف أرقاماً مختلفةً لا يبلغ حدَّ رقمٍ
// أبداً، **فحدُّ الرقم وحدَه لا يراه.**
//
// **ويُعدّ ولو رُفض**: وإلّا صار الرفضُ مجّانيّاً — يجرّب حتّى يمرّ.
func (s *Service) noteOTPSend(ctx context.Context, phone, ip string) error {
	if ip != "" {
		key := "otp:ip:" + ip
		n, err := s.rdb.Incr(ctx, key).Result()
		if err != nil {
			return err
		}
		if n == 1 {
			s.rdb.Expire(ctx, key, 15*time.Minute)
		}
		if n > otpMaxPerIP {
			return ErrOTPRateLimited
		}
	}
	_ = phone
	return nil
}

// sendOTPFor يُصدر رمزاً لغرض محدّد مع تحديد معدّل خاص بذلك الغرض.
// مسار واحد لكل رموز التحقق — لا يعيد كل تدفّق كتابة المنطق نفسه.
func (s *Service) sendOTPFor(ctx context.Context, phone, purpose, rateKey, ip string) error {
	if err := s.noteOTPSend(ctx, phone, ip); err != nil {
		return err
	}
	key := rateKey + phone
	n, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	if n == 1 {
		s.rdb.Expire(ctx, key, 15*time.Minute)
	}
	if n > s.otpQuota(ctx) {
		return ErrOTPRateLimited
	}
	code, err := randomDigits(6)
	if err != nil {
		return err
	}
	if err := s.repo.CreateOTP(ctx, phone, s.hashOTP(phone, code), purpose, s.otpLifetime(ctx)); err != nil {
		return err
	}
	if err := s.sender.SendOTP(ctx, phone, code); err != nil {
		return s.otpSendError(err)
	}
	return nil
}

// RequestPasswordReset يرسل رمزاً لاستعادة كلمة المرور. لا يكشف إن كان الرقم
// مسجّلاً أم لا (تعداد الحسابات) — الرد ناجح دائماً من وجهة نظر المتصل.
func (s *Service) RequestPasswordReset(ctx context.Context, rawPhone, ip string) error {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return ErrInvalidPhone
	}
	if _, _, err := s.repo.UserByPhone(ctx, phone); err != nil {
		return nil // رقم غير مسجّل: صمت مقصود
	}
	return s.sendOTPFor(ctx, phone, "reset", "otp:rst:", ip)
}

// ConfirmPasswordReset يتحقق من الرمز ويضبط كلمة مرور جديدة، ثم يفتح جلسة
// جديدة — فلا يعيد المستخدم إدخال ما ضبطه للتوّ.
func (s *Service) ConfirmPasswordReset(ctx context.Context, rawPhone, code, password, userAgent, ip string) (*AuthResult, error) {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return nil, ErrInvalidPhone
	}
	if int64(len(password)) < s.intSetting(ctx, "security.password_min_length", minPasswordLn) {
		return nil, ErrWeakPassword
	}
	valid, err := s.repo.ConsumeOTP(ctx, phone, s.hashOTP(phone, code), "reset")
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrOTPInvalid
	}
	user, _, err := s.repo.UserByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetPassword(ctx, user.ID, hash); err != nil {
		return nil, err
	}
	// ══════════════════════════════════════════════════════════════
	// **والاستعادةُ تُبطل الحسابَ كلَّه لا نوعَ عميلٍ واحداً** (`SEC8`)
	// ══════════════════════════════════════════════════════════════
	//
	// **وكان الإبطالُ يقع ضمناً في `issueFor`** — **وهي تُبطل عائلاتِ
	// نوعِ العميل الطالبِ وحدَها** (`revokeClientSessions`، هجرة
	// `0099`، وهو صوابُها في الدخول العاديّ: **دخولُ السائق من هاتفه
	// لا يُخرجه من متصفّحه**).
	//
	// **لكنّ الاستعادةَ ليست دخولاً عاديّاً**: **هي بابُ من فقد
	// حسابَه** — **ومن أعاد كلمتَه ليُخرج متطفّلاً كان يُخرجه من نوعٍ
	// واحدٍ ويُبقيه في الباقي.**
	//
	// **قِيس ٢٠٢٦-٠٩-١٣**: **عائلةُ تطبيقِ الزبون تنجو من استعادةٍ
	// طُلبت من الويب** — وعقدُ `R13` و`F-30` يقول «إعادةٌ وإخراجٌ
	// شامل».
	//
	// **والترتيبُ مقصود**: **الإبطالُ قبل إصدار الجلسة الجديدة** —
	// **وإلّا أبطل نفسَه.**
	if err := s.revokeAllSessions(ctx, user.ID); err != nil {
		return nil, err
	}
	s.repo.Audit(ctx, &user.ID, "auth.password_reset", "user", user.ID, ip, nil)
	// جلسة جديدة بعد إبطالٍ شامل — من سرق الحساب يخرج فوراً
	return s.issueFor(ctx, user, userAgent, ip, "auth.password_reset")
}

// RequestSignup يرسل رمز تأكيد لإنشاء حساب زبون جديد.
func (s *Service) RequestSignup(ctx context.Context, rawPhone, ip string) error {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return ErrInvalidPhone
	}
	// حساب موجود بكلمة مرور = ليس تسجيلاً جديداً
	if _, hash, err := s.repo.UserByPhone(ctx, phone); err == nil && hash != "" {
		return ErrPhoneTaken
	}
	return s.sendOTPFor(ctx, phone, "signup", "otp:sgn:", ip)
}

// ConfirmSignup ينشئ حساب **زبون** باسم وكلمة مرور بعد تأكيد الرقم.
// لا يُنشأ أي دور آخر من هنا إطلاقاً — المتجر/السائق/المندوب عبر الإدارة أو مندوب.
// VerifySignupCode يتحقّق من رمز التسجيل **بلا استهلاك** — خطوةٌ بين إرسال
// الرمز وبين ملء البيانات.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٦.)
//
// **ولا تُنشئ حساباً ولا تُصدر جلسة** — تقول «الرمزُ صحيح» وتسكت. **والإنشاءُ
// يبقى في `ConfirmSignup` حيث يُستهلك الرمزُ مرّةً واحدة.**
func (s *Service) VerifySignupCode(ctx context.Context, rawPhone, code string) error {
	return s.verifyCode(ctx, rawPhone, code, "signup")
}

// VerifyResetCode مثلُها لاستعادة كلمة المرور.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا يجوز أن يفتح الفورمُ بمجرّد إرسال طلب
//
//	استعادة».)
//
// **والاستعادةُ أخطرُ من التسجيل**: من فتح نموذجَ كلمةٍ جديدةٍ بمجرّد إرسال
// الرمز **يظنّ أنّه على وشك تغييرها**، ثمّ يُرفض بعد أن كتبها مرّتين.
func (s *Service) VerifyResetCode(ctx context.Context, rawPhone, code string) error {
	return s.verifyCode(ctx, rawPhone, code, "reset")
}

// verifyCode فحصٌ لا يستهلك — **واحدٌ للغرضين.**
//
// **ونسختان بغرضين تفترقان**: تُشدَّد إحداهما ويُنسى ما بجانبها.
func (s *Service) verifyCode(ctx context.Context, rawPhone, code, purpose string) error {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return ErrInvalidPhone
	}
	valid, err := s.repo.CheckOTP(ctx, phone, s.hashOTP(phone, code), purpose)
	if err != nil {
		return err
	}
	if !valid {
		return ErrOTPInvalid
	}
	return nil
}

func (s *Service) ConfirmSignup(ctx context.Context, rawPhone, code, fullName, password, userAgent, ip string) (*AuthResult, error) {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return nil, ErrInvalidPhone
	}
	if int64(len(password)) < s.intSetting(ctx, "security.password_min_length", minPasswordLn) {
		return nil, ErrWeakPassword
	}
	// ══════════════════════════════════════════════════════════════════
	// **والرمزُ عند التسجيل صار اختياريّاً**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك 2026-08-25: «تسجيلُ الدخول ما بدّو كود للزبون…
	//  وعند أوّل طلب يقال له: يجب توثيق الحساب».)
	//
	// # ولماذا نُقل الاحتكاك
	//
	// **وقُيّد رقمُ واتساب المنصّة مرّتين في يوم** — الثانيةُ بعد
	// رسالتَي تحقّقٍ اثنتين. **وواتساب يمنع الحساباتِ الشخصيّةَ من
	// مراسلة من لم يراسلها**، ولا تُصلح ذلك مهلةٌ ولا صياغةُ نصّ.
	//
	// **ومن حُبس على شاشة رمزٍ لا يصل لم يُنشئ حساباً أصلاً.**
	//
	// # والتوثيقُ لم يُلغَ، نُقل
	//
	// **والحسابُ غيرُ الموثَّق يتصفّح ولا يطلب** — والمنعُ قائمٌ سلفاً
	// في `orders/service.go` و`driver_handlers.go` و`leads_handlers.go`.
	// **فمن أراد أن يطلب وثّق نفسَه حينها** برسالةٍ يرسلها هو
	// (`wa_inbound.go`).
	//
	// # ومفتاحٌ يُعيد الشرطَ متى شئنا
	//
	// **ويوم تُشترى بوّابةُ رسائلَ يُرفع المفتاحُ فيعود الرمزُ شرطاً**
	// — بلا شيفرةٍ تتغيّر.
	//
	// ══════════════════════════════════════════════════════════════════
	// **والتسجيلُ لا يصير استعادةً ولا استيلاءً** (`CUST-DEF-001`، ٢٠٢٦-٠٩-١٩)
	// ══════════════════════════════════════════════════════════════════
	//
	// **وكان الرمزُ حين يُطفأ يُسقط كلَّ حارس**: **رقمُ حسابٍ قائمٍ يكفي لكتابة
	// كلمته واسمِه ومحوِ علَمِ «بدّل كلمتك» وإصدارِ جلسةٍ له** — **لكلّ دور**،
	// **والأدمنُ بلا PIN يُسلَّم تحدّيَ إنشاءِ رمزه.** **وقراءةُ الإعداد إن فشلت
	// عُدّت إطفاءً.** (`qa/signup_takeover_test.go` — `TestSU*`.)
	//
	// **فصار العقدُ ثلاثة أبواب لا يتداخل منها شيء:**
	//
	//	رقمٌ جديد            ⇐ حسابُ زبونٍ جديد — **والرمزُ كما يقول الإعداد**
	//	                        (**وفشلُ القراءة يطلبه**)
	//	حسابٌ قائمٌ بكلمة     ⇐ **مرفوضٌ دائماً** (`phone_taken`) — بابُه الدخولُ
	//	                        أو الاستعادة، **لا التسجيل**
	//	زبونٌ قائمٌ بلا كلمة  ⇐ **(من دخل بالرمز ولم يضع كلمة)** يُكمَل **برمزٍ
	//	                        دائماً** ولو كان الإعدادُ مطفأً — **فالرمزُ وحدَه
	//	                        يثبت أنّه صاحبُ الرقم**
	//
	// **وما عدا ذلك مرفوضٌ بلا كتابة**: **موقوفٌ أو محظور · أدمنٌ أو سائقٌ أو
	// متجرٌ أو مندوب** — **فلا يُمسّ حسابٌ ليس زبوناً عاديّاً من باب الزبائن.**
	//
	// **والمحاولاتُ تُعَدّ بعدّاد الدخول نفسِه** (`loginLocked`) — **لا حدٌّ
	// ثانٍ**: **كلُّ رفضٍ يُعَدّ على الرقم والعنوان، وكلُّ حسابٍ يُنشأ يُعَدّ
	// على العنوان.**
	if s.loginLocked(ctx, phone, ip) {
		return nil, ErrTooManyAttempts
	}
	refuse := func(e error) (*AuthResult, error) {
		s.noteLoginFail(ctx, phone, ip)
		return nil, e
	}
	consume := func() (bool, error) {
		if strings.TrimSpace(code) == "" {
			return false, nil
		}
		return s.repo.ConsumeOTP(ctx, phone, s.hashOTP(phone, code), "signup")
	}
	existing, existingHash, err := s.repo.UserByPhone(ctx, phone)
	switch {
	case errors.Is(err, ErrNotFound):
		// **والاحتياطيُّ مُشعَل** — فخطأُ قراءةِ الإعداد يطلب الرمزَ لا يُسقطه.
		if s.boolSetting(ctx, "auth.signup_verify", true) {
			valid, err := consume()
			if err != nil {
				return nil, err
			}
			if !valid {
				return refuse(ErrOTPInvalid)
			}
		}
	case err != nil:
		return nil, err
	case existingHash != "" || existing.Status != "active" || !onlyCustomer(existing.Roles):
		return refuse(ErrPhoneTaken)
	default:
		valid, err := consume()
		if err != nil {
			return nil, err
		}
		if !valid {
			if strings.TrimSpace(code) == "" {
				return refuse(ErrPhoneTaken)
			}
			return refuse(ErrOTPInvalid)
		}
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	user := existing
	if user == nil {
		user, err = s.repo.CreateUserWithRole(ctx, phone, fullName, "customer")
		if err != nil {
			return nil, err
		}
		if err := s.repo.SetPassword(ctx, user.ID, hash); err != nil {
			return nil, err
		}
		s.repo.Audit(ctx, &user.ID, "user.register", "user", user.ID, ip, nil)
		_, ipKey := loginKeys(phone, ip)
		s.countAttempts(ctx, ipKey)
	} else {
		// زبونٌ دخل بالرمز ولم يضع كلمة — **وأثبت الرقمَ برمزٍ الآن**.
		if err := s.repo.SetPassword(ctx, user.ID, hash); err != nil {
			return nil, err
		}
		if fullName != "" {
			_ = s.repo.SetFullName(ctx, user.ID, fullName)
		}
	}
	user, _, err = s.repo.UserByID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return s.issueFor(ctx, user, userAgent, ip, "auth.signup")
}

// loginKeys مفاتيح عدّ المحاولات الفاشلة: بالرقم وبالعنوان.
func loginKeys(phone, ip string) (string, string) {
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host // المنفذ العابر يتغيّر مع كل اتصال — المفتاح للعنوان
	}
	return "login:fail:p:" + phone, "login:fail:i:" + ip
}

// loginLocked هل تجاوز الرقم أو العنوان حدّ المحاولات الفاشلة؟
// عطل الكاش لا يقفل الدخول (يفشل مفتوحاً عمداً — كما ActiveStatus).
func (s *Service) loginLocked(ctx context.Context, phone, ip string) bool {
	pk, ik := loginKeys(phone, ip)
	if n, err := s.rdb.Get(ctx, pk).Int(); err == nil && int64(n) >= s.intSetting(ctx, "security.login_max_attempts", loginMaxPerPhone) {
		return true
	}
	if n, err := s.rdb.Get(ctx, ik).Int(); err == nil && n >= loginMaxPerIP {
		return true
	}
	return false
}

// noteLoginFail يعدّ محاولة فاشلة على الرقم والعنوان معاً.
func (s *Service) noteLoginFail(ctx context.Context, phone, ip string) {
	pk, ik := loginKeys(phone, ip)
	s.countAttempts(ctx, pk, ik)
}

// countAttempts **عدّادُ المحاولات الواحد** — للدخول وللتسجيل (`CUST-DEF-001`):
// **ونافذتُه نافذةُ الدخول.** **ولا عدّادَ ثانٍ بمنطقٍ ثانٍ.**
func (s *Service) countAttempts(ctx context.Context, keys ...string) {
	for _, k := range keys {
		if n, err := s.rdb.Incr(ctx, k).Result(); err == nil && n == 1 {
			s.rdb.Expire(ctx, k, loginFailWindow)
		}
	}
}

// onlyCustomer **زبونٌ عاديٌّ لا دورَ له غيرُه** — وحده يُكمَل من باب التسجيل.
func onlyCustomer(roles []string) bool {
	if len(roles) == 0 {
		return false
	}
	for _, r := range roles {
		if r != RoleCustomer {
			return false
		}
	}
	return true
}

// LoginPassword دخول بكلمة المرور (للموظفين والأدوار التشغيلية غالباً).
func (s *Service) LoginPassword(ctx context.Context, rawPhone, password, userAgent, ip string) (*AuthResult, error) {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return nil, ErrInvalidPhone
	}
	// الفحص قبل قراءة الحساب: لا نكشف وجود الرقم لمن تجاوز الحد
	if s.loginLocked(ctx, phone, ip) {
		return nil, ErrTooManyAttempts
	}
	user, hash, err := s.repo.UserByPhone(ctx, phone)
	if errors.Is(err, ErrNotFound) || (err == nil && hash == "") {
		// نعدّ المحاولة حتى لرقم غير مسجّل: وإلا صار الفرق في السلوك كاشفاً
		// لأي رقم له حساب (تعداد حسابات).
		s.noteLoginFail(ctx, phone, ip)
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
		s.noteLoginFail(ctx, phone, ip)
		s.repo.Audit(ctx, nil, "auth.password_failed", "user", user.ID, ip, nil)
		return nil, ErrInvalidCredentials
	}
	// نجاح: يمسح عدّاد الرقم فلا يُعاقَب صاحبه بمحاولاته السابقة
	pk, _ := loginKeys(phone, ip)
	s.rdb.Del(ctx, pk)
	// **وبصمةُ الكلمة التي أُثبتت توّاً تُمرَّر** — `XG-40`:
	// **تغييرٌ يقع بين التحقّق والإصدار يُنتج جلسةً بكلمةٍ ماتت.**
	return s.issueForVerified(ctx, user, userAgent, ip, "auth.password_login", hash)
}

// IssueForUserID يصدر جلسة لمستخدم بمعرّفه (لتسليم SSO عبر رمز موثوق لمرّة واحدة).
// sessionID عائلة جلسة المصدر: الانتقال بين لوحاتنا امتداد للجلسة نفسها لا جلسة
// جديدة، فلا يُطرد المستخدم من اللوحة التي جاء منها.
func (s *Service) IssueForUserID(ctx context.Context, userID, sessionID, userAgent, ip string) (*AuthResult, error) {
	user, _, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.issueSession(ctx, user, userAgent, ip, "auth.sso", sessionID)
}

// ActiveSessionID عائلة الجلسة الفعّالة للحساب — يستعملها تسليم SSO.
func (s *Service) ActiveSessionID(ctx context.Context, userID string) string {
	sid, err := s.repo.ActiveSessionID(ctx, userID)
	if err != nil {
		return ""
	}
	return sid
}

// issueFor يبدأ جلسة جديدة: يبطل كل ما سبق (قاعدة الجلسة الواحدة).
//
// ══════════════════════════════════════════════════════════════════════
// **وبابُ الأدمن لا يُفتح هنا — يقف عند الرمز**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «رمزُ دخولٍ ثانٍ من ٤ أرقام… فقط للأدمن».)
//
// **وموضعُه هنا لا في المعالِجات**: `issueFor` هي ما ينادي كلَّ دخولٍ يبدأ
// **من صفر** — كلمةٌ أو رمزٌ مؤقّتٌ أو استعادةُ كلمة. **وما لا يمرّ بها
// امتدادُ جلسةٍ تحقّقت** (تجديدُ توكن، وتسليمُ SSO بين أقسام اللوحة)،
// **ولا معنى لأن يُسأل الرمزَ مرّةً أخرى من هو داخلٌ أصلاً.**
//
// **وثلاثةُ معالِجاتٍ يُنسى أحدُها** — والحارسُ في المنبع لا يُنسى.
func (s *Service) issueFor(ctx context.Context, user *User, userAgent, ip, action string) (*AuthResult, error) {
	return s.issueForVerified(ctx, user, userAgent, ip, action, "")
}

// issueForVerified كـ`issueFor` **وتحمل بصمةَ الكلمة المُثبَتة** — `XG-40`.
func (s *Service) issueForVerified(ctx context.Context, user *User, userAgent, ip, action, verifiedHash string) (*AuthResult, error) {
	// **والتسجيلُ الجديد لا يُنشئ أدمن** — لكنّ الشرطَ يُقرأ من الأدوار لا
	// من الفعل، **فلو صار للأدمن مسارُ دخولٍ رابعٌ يوماً وقف عند الرمز أيضاً.**
	if NeedsPin(user.Roles) {
		challenge, err := s.PinChallenge(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		hash, err := s.repo.AdminPinHash(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		// **ولا توكنَ في الردّ** — خطوةٌ ناقصةٌ لا جلسةٌ ناقصة.
		return &AuthResult{
			PinRequired: true,
			PinSetup:    hash == "",
			Challenge:   challenge,
		}, nil
	}
	return s.issueSessionFor(ctx, user, userAgent, ip, action, "", verifiedHash)
}

// issueSession يصدر زوج توكنات.
//
// **sessionID فارغ = دخولٌ جديد** يُبطل جلساتِ هذا الحساب **من نوع العميل
// نفسِه**. **وموجود = امتدادٌ لنفس الجلسة** (تسليمٌ بين اللوحات أو تدويرُ
// توكن) فلا يُبطل إخوته.
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا صار الإبطالُ بالنوع لا بالحساب**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١١، في خطّة تطبيق أندرويد.)
//
// **كان دخولٌ جديدٌ يُبطل كلَّ الجلسات** — وهو صحيحٌ حين كانت المنصّةُ
// متصفّحاً وحدَه. **ومع الهاتف يصير حلقةً مقفلة**: السائقُ يفتح التطبيقَ
// فيخرج من الويب، ويفتح الويبَ فيخرج من التطبيق.
//
// **والقيدُ لا يُلغى بل يُضيَّق**: هو ما يمنع مشاركةَ الحسابات — سائقٌ
// يعطي رقمَه ورمزَه لآخرَ فيعملان معاً **فيقبضان على اسمٍ واحدٍ ولا تعرف
// المنصّةُ من سلّم.** **وسائقٌ بهاتفين لا يزال ممنوعاً**، لأنّ الثاني
// يُبطل الأوّل — وذاك ما كان يُحرَس فعلاً.
//
// **والسقفُ مغلقٌ بـ`normalizeClient`** — ثلاثُ قيمٍ لا رابعَ لها، **فلا
// يفتح أحدٌ جلساتٍ بلا حدٍّ بترويسةٍ يخترعها.**
func (s *Service) issueSession(ctx context.Context, user *User, userAgent, ip, action, sessionID string) (*AuthResult, error) {
	return s.issueSessionFor(ctx, user, userAgent, ip, action, sessionID, "")
}

// issueSessionFor كـ`issueSession` **وتحمل بصمةَ الكلمة المُثبَتة** —
// **فعائلةٌ جديدةٌ لا تُنشأ بكلمةٍ بُدّلت بين التحقّق والإصدار** (`XG-40`).
func (s *Service) issueSessionFor(ctx context.Context, user *User, userAgent, ip, action, sessionID, verifiedHash string) (*AuthResult, error) {
	// ══════════════════════════════════════════════════════════════
	// **وجلسةٌ قائمةٌ تُجدَّد وإن أُوقف صاحبُها** — `XG-39`
	// ══════════════════════════════════════════════════════════════
	//
	// **ومهلةُ رمز الوصول خمسَ عشرةَ دقيقة** — **ورحلةُ توصيلٍ قد
	// تطول.** **فمنعُ التجديد يقتل استثناءَ دورةِ ١١ بعد ربع ساعة،
	// ولو لم يُبطَل شيء.**
	//
	// **وأسوأُ من ذلك**: `Refresh` تُبطل الرمزَ القديمَ **قبل** أن
	// تنادي هذه (`RevokeRefresh`) — **فمحاولةٌ فاشلةٌ واحدةٌ تقتل
	// الجلسةَ كلَّها.**
	//
	// **والفرقُ بين جلسةٍ جديدةٍ وتجديدِ قائمةٍ هو `sessionID`**:
	// **فارغٌ يعني دخولاً أو جهازاً جديداً** — ويُمنَع؛ **وغيرُ
	// فارغٍ يعني عائلةً قائمةً** — وقد أثبت `RevokeRefresh` توّاً
	// أنّ لها صفّاً حيّاً غيرَ مُبطَلٍ ولا منتهٍ.
	//
	// **و`blocked` و`deleted` لا استثناءَ لهما** — **جلساتُهما
	// أُبطلت أصلاً، وهذا حارسٌ ثانٍ.**
	switch {
	case user.Status == "active":
	case user.Status == "suspended" && sessionID != "":
		// **جلسةٌ قائمةٌ لموقوفٍ عاديّ** — **توثيقٌ لا تخويل.**
	default:
		if user.Status == "suspended" {
			return nil, ErrUserSuspended
		}
		return nil, ErrUserBlocked
	}
	rawRefresh, refreshHash, err := auth.NewOpaqueToken()
	if err != nil {
		return nil, err
	}
	client := ClientFrom(ctx)
	// **دخولٌ جديدٌ يُبطل ما سبق من نوعه** — والإبطال يشمل قائمة Redis
	// كي يسري فوراً على توكنات الوصول القائمة.
	if sessionID == "" {
		if err := s.revokeClientSessions(ctx, user.ID, client); err != nil {
			return nil, err
		}
	}
	// نخزّن التجديد أولاً لنعرف عائلة الجلسة، ثم نضعها في توكن الوصول
	sid, err := s.repo.StoreRefreshFor(ctx, user.ID, refreshHash,
		s.sessionLife(ctx), userAgent, ip, sessionID, client, verifiedHash)
	if err != nil {
		return nil, err
	}
	access, exp, err := s.tokens.IssueAccess(user.ID, user.Roles, sid)
	if err != nil {
		return nil, err
	}
	s.repo.Audit(ctx, &user.ID, action, "user", user.ID, ip, nil)
	// **والمخرجُ الثاني للحقل** — `Me` هو الأوّل. **ولو قُنّع في أحدهما فقط
	// لَظهرت البوّابةُ بعد الدخول ثمّ اختفت عند أوّل تحديث** (أو العكس).
	s.applyForcePolicy(ctx, user)
	return &AuthResult{
		User: *user,
		Tokens: TokenPair{
			AccessToken:     access,
			AccessExpiresAt: exp,
			RefreshToken:    rawRefresh,
		},
	}, nil
}

// Refresh يدوّر توكن التحديث: يبطل القديم ويصدر زوجاً جديداً داخل نفس الجلسة —
// فتدوير لوحة لا يقطع اللوحة الأخرى المفتوحة لنفس الشخص.
func (s *Service) Refresh(ctx context.Context, rawRefresh, userAgent, ip string) (*AuthResult, error) {
	userID, sessionID, err := s.repo.RevokeRefresh(ctx, auth.HashToken(rawRefresh))
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
	return s.issueSession(ctx, user, userAgent, ip, "auth.refresh", sessionID)
}

// sessionRevokedKey مفتاح إبطال جلسة في Redis.
func sessionRevokedKey(sid string) string { return "sess:revoked:" + sid }

// ══════════════════════════════════════════════════════════════════════
// **وحُذفت `SessionRevoked`** — `R16`
// ══════════════════════════════════════════════════════════════════════
//
// **كانت تردّ `bool`**:
//
//	n, err := s.rdb.Exists(ctx, sessionRevokedKey(sid)).Result()
//	return err == nil && n > 0
//
// **وخطأُ الذاكرة يُقرأ «ليست مُبطَلة»** — **فجلسةٌ أُبطلت تعود تعمل
// بسقوط خبيئة.** (مقيسٌ بذاكرةٍ حقيقيّة: `401` ثمّ `200`.)
//
// **ولم تُترَك مُهمَلةً بل حُذفت**: **دالّةٌ باقيةٌ تُنادى** —
// **والشكلُ نفسُه هو ما أوقع العيب.**
//
// **وخلَفُها `CheckSession`** في `session_check.go` — ثلاثُ حالاتٍ
// لا اثنتان.

// revokeSession يُبطل عائلة جلسة: في القاعدة (توكنات التجديد) وفي Redis
// (توكنات الوصول القائمة). مدة المفتاح = عمر توكن الوصول، فبعدها لا يبقى توكن حيّ.
func (s *Service) revokeSession(ctx context.Context, userID, sid string) error {
	if sid == "" {
		return nil
	}
	if err := s.repo.RevokeSession(ctx, userID, sid); err != nil {
		return err
	}
	s.rdb.Set(ctx, sessionRevokedKey(sid), "1", s.tokens.AccessTTL()+time.Minute)
	return nil
}

// revokeAllSessions يُبطل كل جلسات الحساب — يستعمله الدخول الجديد وخروج الإدارة.
// revokeClientSessions يُبطل جلساتِ حسابٍ **من نوعِ عميلٍ واحد.**
//
// **وRedis شرطٌ لا زينة**: القاعدةُ تُبطل توكنَ التجديد، **وتوكنُ الوصول
// القائمُ يبقى صالحاً حتّى تنتهي مهلتُه** — والوسيطُ يسأل Redis في كلّ
// طلبٍ ليعرف أنّ الجلسةَ ماتت.
func (s *Service) revokeClientSessions(ctx context.Context, userID, client string) error {
	sids, err := s.repo.ClientSessionIDs(ctx, userID, client)
	if err != nil {
		return err
	}
	if _, err := s.repo.RevokeClientTokens(ctx, userID, client); err != nil {
		return err
	}
	for _, sid := range sids {
		s.rdb.Set(ctx, sessionRevokedKey(sid), "1", s.tokens.AccessTTL()+time.Minute)
	}
	return nil
}

func (s *Service) revokeAllSessions(ctx context.Context, userID string) error {
	sids, err := s.repo.ActiveSessionIDs(ctx, userID)
	if err != nil {
		return err
	}
	if _, err := s.repo.RevokeAllTokens(ctx, userID); err != nil {
		return err
	}
	for _, sid := range sids {
		s.rdb.Set(ctx, sessionRevokedKey(sid), "1", s.tokens.AccessTTL()+time.Minute)
	}
	// **ووجهاتُ الدفع تُقطَع معها** (`SEC`، ٢٠٢٦-٠٩-١٣) — **وإبطالٌ
	// شاملٌ يترك جهازاً يستقبل إشعاراً خاصّاً إبطالٌ ناقص.**
	//
	// **وسقوطُها لا يُنقض الإبطال**: **الجلساتُ قُطعت في القاعدة
	// وفي المُسرِّع** — **فيُسجَّل الخطأُ ولا يُردّ الفعلُ كلُّه.**
	if _, err := s.repo.DeleteDeviceTokensOfUser(ctx, userID); err != nil {
		s.logger.Error("تعذّر قطعُ وجهاتِ الدفع بعد إبطالٍ شامل",
			"user", userID, "error", err)
	}
	return nil
}

// Logout يُنهي الجلسة على **كل تطبيقات المنصة** لا على التطبيق الذي طلب الخروج.
// الجلسة الواحدة تمتد على الأربعة عبر عائلة session_id (هجرة 0026): إبطال توكن
// واحد كان يترك الحساب مفتوحاً في تبويب آخر — وعلى جهاز مشترك هذا خطر حقيقي.
func (s *Service) Logout(ctx context.Context, rawRefresh, deviceToken, ip string) error {
	// **ووجهةُ الدفع تُنهى مع الجلسة لا بعدها** — `D12`.
	//
	// **والعميلُ يمسح اعتمادَه محلّيّاً قبل النداء** — **فنداءٌ
	// موثَّقٌ برمز وصولٍ يُردّ ٤٠١.** **وهذا البابُ يوثَّق برمز
	// التجديد نفسِه**، **فيبقى قادراً على إنهاء الوجهة.**
	userID, sid, err := s.repo.RevokeRefreshAndDevice(
		ctx, auth.HashToken(rawRefresh), deviceToken)
	if errors.Is(err, ErrNotFound) {
		return nil // خروج توكن ميت = نجاح صامت
	}
	if err != nil {
		return err
	}
	if err := s.revokeSession(ctx, userID, sid); err != nil {
		return err
	}
	s.repo.Audit(ctx, &userID, "auth.logout", "user", userID, ip, nil)
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (*User, error) {
	user, _, err := s.repo.UserByID(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return nil, httpx.ErrNotFound
	}
	s.applyForcePolicy(ctx, user)
	return user, err
}

// SetOwnAvatar يضبط صورة المستخدم لنفسه (mediaID فارغ = إزالة).
func (s *Service) SetOwnAvatar(ctx context.Context, userID, mediaID string) error {
	return s.repo.SetAvatar(ctx, userID, mediaID)
}

// SetOwnName يغيّر المستخدمُ اسمَه بنفسه.
//
// # لماذا لزم
//
// **لم يكن له بابٌ ألبتّة**: صفحةُ «حسابي» تقرأ الاسمَ وتعرضه **ولا تكتبه** —
// ومن أخطأ في اسمه عند التسجيل، أو كُتب له بيد موظّفٍ في طلبٍ هاتفيّ، **يبقى
// عليه إلى الأبد** أو يتّصل بالمنصة ليُغيّره له إنسان.
//
// **والاسمُ يُقرأ في مواضعَ يهمّ فيها**: يناديه السائقُ عند الباب، ويُكتب في
// الفاتورة، ويظهر لغرفة العمليات حين يتّصل. **واسمٌ خاطئٌ يُربك الثلاثة.**
//
// (شهده المالك ٢٠٢٦-٠٨-٠٣: «بحسابي لا يوجد مكان لتبديل اسم الشخص».)
//
// # ولا يُقبل فارغاً
//
// **اسمٌ فارغٌ أسوأُ من اسمٍ خاطئ**: الخاطئُ يُنادى به فيُصحَّح، **والفارغُ
// يجعل السائقَ يقف أمام بابٍ لا يعرف من يطرقه**، وغرفةَ العمليات تقرأ سطراً
// بلا صاحب.
func (s *Service) SetOwnName(ctx context.Context, userID, name string) error {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) < 2 {
		return ErrNameTooShort
	}
	// **والحدُّ الأعلى بالمحارف لا بالبايتات**: الاسمُ العربيُّ محرفُه بايتان،
	// **وقصٌّ بالبايتات يقطع حرفاً في نصفه** فيخرج مربّعاً في الشاشة.
	r := []rune(name)
	if len(r) > 60 {
		name = string(r[:60])
	}
	return s.repo.SetFullName(ctx, userID, name)
}

// SetPassword **تغييرُ المرء لكلمته** — `XG-40`.
//
// **و`currentSID` عائلةُ الجلسة التي نفّذت التغيير** — تُقرأ من رمز
// الوصول الذي حمل الطلب. **وفارغةٌ تعني «لا استثناء»**: تُقطَع كلُّها،
// وهو الأسلمُ حين لا يُعرَف من ينفّذ.
func (s *Service) SetPassword(ctx context.Context, userID, password, currentPassword, ip, currentSID string) error {
	if int64(len(password)) < s.intSetting(ctx, "security.password_min_length", minPasswordLn) {
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
			// **والسببُ يُقال بعينه** — انظر `ErrWrongCurrentPassword`.
			return ErrWrongCurrentPassword
		}
	}
	newHash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	// ══════════════════════════════════════════════════════════════
	// **وكلمةٌ بُدّلت تُخرج من عرفها** — `XG-40`
	// ══════════════════════════════════════════════════════════════
	//
	// **كانت تكتب البصمةَ وتمضي** — **فمن شكّ أنّ أحداً يعرف كلمتَه
	// فبدّلها لم يُخرجه**: جلسةُ المتطفّل تعمل ورمزُ تجديده يدور.
	//
	// # وعقدُ المالك (٢٠٢٦-٠٩-٠٧)
	//
	//	تبقى **العائلةُ التي نفّذت التغيير** · وتُقطَع البواقي
	//
	// **ولا تُقطَع كلُّها** — **وإلّا أخرج نفسَه من الصفحة التي يقف
	// عليها**، وهو عكسُ ما طُلب.
	//
	// # والثلاثةُ في معاملةٍ واحدة
	//
	// **بصمةٌ وحِقبةٌ وإبطالٌ** — **فإن سقط أحدُها لم يقع شيء.**
	// **وكلمةٌ بُدّلت بلا إخراجٍ تُطمئن ولا تحمي.**
	//
	// **والكلمةُ الخاطئةُ لا تصل إلى هنا** — تُردّ قبل المعاملة، فلا
	// حِقبةَ ولا إبطال.
	revoked, err := s.repo.SetPasswordKeeping(ctx, userID, newHash, currentSID)
	if err != nil {
		return err
	}
	// **وبعد التثبيت**: مُسرِّعُ الرفض للعائلات المقطوعة.
	for _, sid := range revoked {
		s.rdb.Set(ctx, sessionRevokedKey(sid), "1", s.tokens.AccessTTL()+time.Minute)
	}
	// **ووجهاتُ المقطوعِ وحدَه** (`SEC4`) — **وعائلتُه باقيةٌ بعقد
	// `XG-40`، فوجهتُها تبقى معها.** **ومن حذف الكلَّ هنا أسكت
	// إشعاراتَ من لم يُخرَج.**
	if _, err := s.repo.DeleteDeviceTokensOfSessions(ctx, revoked); err != nil {
		s.logger.Error("تعذّر قطعُ وجهاتِ العائلات المقطوعة",
			"user", userID, "error", err)
	}
	s.repo.Audit(ctx, &userID, "auth.set_password", "user", userID, ip,
		map[string]any{"sessions_revoked": len(revoked), "kept": currentSID != ""})
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
// EnsureUserWithRoleTx كـ`EnsureUserWithRole` **في معاملةٍ مُمرَّرة**.
//
// **وُجدت لأجل تحويل المرشَّح** (`PF-01`): **إنشاءُ المتجر وصاحبِه
// والمكافأةُ والتثبيتُ عمليّةٌ واحدة** — **فإن سقط آخرُها لم يبقَ
// أوّلُها.**
//
// **والتدقيقُ خارجَها**: `Audit` أفضلُ جهدٍ بقرارٍ قائم (`PF-06`)،
// **وإدخالُه هنا يجعل سقوطَه يُسقط إنشاءَ متجرٍ صحيح.**
func (s *Service) EnsureUserWithRoleTx(ctx context.Context, q dbtx.Querier,
	actorID, rawPhone, role, fullName, password string) (string, error) {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return "", ErrInvalidPhone
	}
	var id string
	err := q.QueryRow(ctx, `SELECT id::text FROM users WHERE phone = $1`, phone).Scan(&id)
	if err == nil {
		// **قائمٌ ⇒ يُمنَح الدورَ وحدَه** — **ولا تُمَسّ كلمتُه.**
		if err := GrantRoleTx(ctx, q, id, role, &actorID); err != nil {
			return "", err
		}
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	id, err = CreateUserWithRoleTx(ctx, q, phone, fullName, role)
	if err != nil {
		return "", err
	}
	if password != "" {
		if len(password) < int(s.intSetting(ctx, "security.password_min_length", minPasswordLn)) {
			return "", ErrWeakPassword
		}
		hash, herr := auth.HashPassword(password)
		if herr != nil {
			return "", herr
		}
		if err := SetTempPasswordTx(ctx, q, id, hash); err != nil {
			return "", err
		}
	}
	return id, nil
}

func (s *Service) EnsureUserWithRole(ctx context.Context, actorID, rawPhone, role, fullName, password, ip string) (*User, error) {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return nil, ErrInvalidPhone
	}
	user, _, err := s.repo.UserByPhone(ctx, phone)
	if errors.Is(err, ErrNotFound) {
		// **وباسمه إن أُعطي** — **وحسابٌ برقمٍ بلا اسمٍ لا يُعرف
		// صاحبُه في جدول الحسابات حتّى يُفتح متجرُه.**
		user, err = s.repo.CreateUserWithRole(ctx, phone, fullName, role)
		// ══════════════════════════════════════════════════════════
		// **وكلمتُه تُوضع مؤقّتةً — للحساب الجديد وحدَه**
		// ══════════════════════════════════════════════════════════
		//
		// (قرارُ المالك ٢٠٢٦-٠٨-١٥: يخرج من النموذج **حسابٌ جاهزٌ
		//  ومتجرٌ جاهز**.)
		//
		// **ومؤقّتةٌ لا دائمة** (`SetTempPassword`): وضعها الأدمنُ
		// وأملاها على صاحبها، **وكلمةٌ يعرفها اثنان ليست كلمةَ سرّ**
		// — فيُطلب تبديلُها عند أوّل دخول.
		//
		// **ولا تُلمس كلمةُ حسابٍ قائم**: من كان في المنصّة ثمّ فُتح
		// له متجرٌ **لا تُبدَّل كلمتُه من نافذة متجر** — ولا يعرف هو
		// أنّها بُدّلت فيقف على بابه.
		if err == nil && password != "" {
			if len(password) < int(s.intSetting(ctx, "security.password_min_length", minPasswordLn)) {
				return nil, ErrWeakPassword
			}
			hash, herr := auth.HashPassword(password)
			if herr != nil {
				return nil, herr
			}
			if serr := s.repo.SetTempPassword(ctx, user.ID, hash); serr != nil {
				return nil, serr
			}
		}
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
	user, hash, err := s.repo.UserByPhone(ctx, phone)
	if errors.Is(err, ErrNotFound) {
		user, err = s.repo.CreateUserWithRole(ctx, phone, "", "admin")
		if err != nil {
			return err
		}
		s.logger.Info("bootstrap admin created", "phone", phone)
		hash = ""
	} else if err != nil {
		return err
	} else if err := s.repo.GrantRole(ctx, user.ID, "admin", nil); err != nil {
		return err
	}
	return s.bootstrapPassword(ctx, user.ID, hash)
}

// bootstrapPassword **كلمةُ المرور الأولى — وإلّا لا يدخل أحدٌ أصلاً.**
//
// ══════════════════════════════════════════════════════════════════════
// **الحلقةُ المفرغةُ التي كشفها المالك (٢٠٢٦-٠٨-١٠)**
// ══════════════════════════════════════════════════════════════════════
//
// «كيف رح يجي رمز وأساساً المنصّة لسّا مو مربوطة بواتساب؟»
//
// **وهو محقّ**: الحسابُ يُنشأ بلا كلمة مرور، **فلا سبيلَ للدخول إلّا برمز
// واتساب** — والبوتُ لا يُقترن إلّا من لوحةٍ لا تُفتح إلّا بدخول.
// **ثلاثةٌ يمسك بعضُها برقاب بعض، والنسخةُ الجديدةُ مقفلةٌ على صاحبها.**
//
// **فتُقبل كلمةٌ أولى من البيئة** (`ADMIN_PASSWORD`) — تُكسر بها الحلقة.
//
// # وهي مؤقّتةٌ بحكم الشيفرة لا بالنصيحة
//
// **تُكتب بـ`SetTempPassword`** فيُرفع علَمُ «بدّلها» — **فأوّلُ دخولٍ يفرض
// كلمةً يختارها هو.** وبعدها تصير القيمةُ في لوحة الاستضافة بلا قيمة.
//
// **ونصيحةٌ «بدّلها لاحقاً» تُنسى** — والعلَمُ لا يُنسى.
//
// # ولا تُدهَس كلمةٌ قائمة
//
// **تُضبط لمن لا كلمةَ له فقط.** ولولا هذا الشرطُ **لأعادت كلُّ نشرةٍ
// كلمةَ البيئة** — فيبدّلها المالكُ ثمّ تعود من ورائه، **ويبقى ما في
// الاستضافة مفتاحاً حيّاً إلى الأبد.**
func (s *Service) bootstrapPassword(ctx context.Context, userID, existingHash string) error {
	raw := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD"))
	if raw == "" || existingHash != "" {
		return nil
	}
	if int64(len(raw)) < minPasswordLn {
		return fmt.Errorf("identity: ADMIN_PASSWORD أقصر من %d محارف", minPasswordLn)
	}
	enc, err := auth.HashPassword(raw)
	if err != nil {
		return err
	}
	if err := s.repo.SetTempPassword(ctx, userID, enc); err != nil {
		return err
	}
	s.logger.Info("bootstrap admin password set — يُطلب تبديلُها عند أوّل دخول")
	return nil
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

// InvalidateStatusCache **مُصدَّرٌ للبذّارات على التجهيز** — من غيّر حالةَ
// مستخدمٍ خارجَ مسار الإدارة (بذّارُ QA) يجب أن يُبطل كاشَه بنفسِه، **وإلّا
// رأى الوسيطُ الحالةَ القديمةَ حتّى انتهاء المهلة (٣٠ث).**
func (s *Service) InvalidateStatusCache(ctx context.Context, userID string) {
	s.invalidateStatusCache(ctx, userID)
}

// QASetPassword **يضبط كلمةَ مرورِ مستخدمٍ مباشرةً** — للبذّارات على التجهيز
// وحدَها (يُدعى من بابِ QA المحروسِ بالبيئة). يُجزّئ بنفس argon2id ويضع
// `must_change_password=false` فيصير الدخولُ بالرقم+الكلمة عاديّاً بلا OTP.
func (s *Service) QASetPassword(ctx context.Context, userID, password string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return s.repo.SetPassword(ctx, userID, hash)
}

// QACreateOrGetCustomer **بيئةُ تجهيزٍ فقط** — يُنشئ (أو يُرجع إن وُجد) زبوناً
// يُستهلك لشهود حذف الحساب (CUST-06-026)، **فلا يُمسّ به زبونُ QA الأساسيّ.**
// يضبط له كلمةَ مرورٍ معلومةً فيُدخَل بالرقم+الكلمة بلا OTP. عكوسٌ: الحذفُ
// يحرّر الرقمَ فيُعاد إنشاؤُه.
func (s *Service) QACreateOrGetCustomer(ctx context.Context, rawPhone, fullName, password string) (string, error) {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return "", ErrInvalidPhone
	}
	if u, _, err := s.repo.UserByPhone(ctx, phone); err == nil && u != nil {
		if perr := s.QASetPassword(ctx, u.ID, password); perr != nil {
			return "", perr
		}
		return u.ID, nil
	}
	u, err := s.repo.CreateUserWithRole(ctx, phone, fullName, "customer")
	if err != nil {
		return "", err
	}
	if err := s.QASetPassword(ctx, u.ID, password); err != nil {
		return "", err
	}
	return u.ID, nil
}

// QAIssueDeleteCode **بيئةُ تجهيزٍ فقط** — يُصدر رمزَ حذفٍ حقيقيّاً (يُخزَّن
// مجزّأً بنفس مسار الإنتاج) ويُعيده، كي تُشهَد سيرورةُ الحذف في التطبيق دون
// قراءة السجلّ. **والأحدثُ هو ما يُستهلك** (`ConsumeOTP` يأخذ آخرَ رمزٍ فعّال)،
// فيُدعى بعد أن يطلب التطبيقُ الرمز.
func (s *Service) QAIssueDeleteCode(ctx context.Context, rawPhone string) (string, error) {
	return s.QAIssueCode(ctx, rawPhone, "delete")
}

// QAIssueCode **بيئةُ تجهيزٍ فقط** — يُصدر رمزَ OTP حقيقيّاً لأيّ غرضٍ
// (signup/reset/whatsapp/delete) ويُعيده، كي تُشهَد مساراتُ OTP في التطبيق
// على التجهيز (المزوّدُ dev يطبع في السجلّ لا يرسل واتساب). **لا يتجاوز
// التحقّق**: يُخزَّن مجزّأً بنفس `CreateOTP`+`otpLifetime`، و`ConsumeOTP` يتحقّق
// منه عاديّاً (مهلة، محاولات، الأحدثُ يُستهلك) — فتبقى حالاتُ القبول حقيقيّة.
func (s *Service) QAIssueCode(ctx context.Context, rawPhone, purpose string) (string, error) {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return "", ErrInvalidPhone
	}
	code, err := randomDigits(6)
	if err != nil {
		return "", err
	}
	if err := s.repo.CreateOTP(ctx, phone, s.hashOTP(phone, code), purpose, s.otpLifetime(ctx)); err != nil {
		return "", err
	}
	return code, nil
}

// otpSendError **يُمرّر سببَ الفشل حين يكون معروفاً — ولا يبتلعه.**
//
// ══════════════════════════════════════════════════════════════════════
// **ورسالةٌ واحدةٌ لعلَّتين تجعل صاحبَها ينتظر ما لا يأتي**
// ══════════════════════════════════════════════════════════════════════
//
// (شهده المالك ٢٠٢٦-٠٨-١١: جرّب رقماً بلا واتساب فقيل له «ضغطٌ على خادم
//
//	الرسائل — يرجى المحاولة بعد قليل».)
//
// **«أعد المحاولة» جوابٌ صحيحٌ لانقطاعِ شبكة، وكاذبٌ لرقمٍ ليس على واتساب**
// — الأوّلُ يُصلحه الانتظار، **والثاني لا يُصلحه إلّا تبديلُ الرقم.**
//
// **وكلُّ محاولةٍ تستهلك من حدّه** — فينتهي به المطاف مقفولاً على خطأٍ
// لم يكن خطأه.
//
// **وما لا يُعرف سببُه يبقى عامّاً** — ولا تُخترع له علّة.
func (s *Service) otpSendError(err error) error {
	var known *httpx.AppError
	if errors.As(err, &known) {
		return known
	}
	s.logger.Error("otp send failed", "error", err)
	return ErrOTPSendFailed
}
