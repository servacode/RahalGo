package identity

// ══════════════════════════════════════════════════════════════════════
//
//	**رمزُ الأدمن — طبقةٌ ثانيةٌ فوق كلمة المرور**
//
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «رمزُ دخولٍ ثانٍ من ٤ أرقام… مثل تحقّقٍ ثنائيٍّ
//
//	فقط للأدمن، لأنّه بنفس اللوحة تحسّباً للاختراق».)
//
// # ولماذا لا تُصدَر جلسةٌ «ناقصة»
//
// **البديلُ أن تُصدَر التوكنات ثمّ تُوسَم «لم يتحقّق»** فيُغلق كلُّ نقطةٍ
// إداريّةٍ على حدة. **وثمانٍ وتسعون نقطةً يُنسى حارسُ إحداها** — وهي عائلةُ
// العطب التي قاستها هذه المنصّة مرّتين اليوم (٣٧ مساراً بلا حارس معرّف،
// و٥ من ٩٨ بحارس دور).
//
// **فلا توكنَ يُصدَر قبل الرمز أصلاً.** ومن لا توكنَ له لا يبلغ شيئاً —
// **حارسٌ واحدٌ لا ثمانيةٌ وتسعون.**
//
// # والتحدّي في الذاكرة لا في القاعدة
//
// **بين الكلمة والرمز خطوةٌ تحتاج أن تتذكّر مَن يحاول.** ولو حُملت في
// الواجهة (معرّفُ المستخدم في الطلب) **لَأمكن تخطّي الكلمة كلَّها**: يرسل
// المهاجمُ معرّفَ المالك ورمزاً يخمّنه.
//
// **فالتحدّي رمزٌ عشوائيٌّ في Redis يعرف صاحبَه** — يعيش خمسَ دقائق ويُستهلك
// مرّةً واحدة. **ومن لم يعرف الكلمةَ لا يملك تحدّياً أصلاً.**
//
// # وخمسُ محاولاتٍ لا أكثر
//
// **أربعةُ أرقامٍ عشرةُ آلاف احتمال** — تُجرَّب كلُّها في دقائقَ بلا حدّ.
// **والحدُّ على المستخدم لا على العنوان**: من بدّل شبكتَه لم يبدّل حسابَه.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// pinLen **أربعةُ أرقامٍ** — بقرار المالك.
// RoleAdmin **صاحبُ المنصّة** — والاسمُ ثابتٌ لا نصٌّ يُكتب في موضعين.
const RoleAdmin = "admin"

const pinLen = 4

// pinMaxTries **خمسُ محاولاتٍ ثمّ قفل** — والقفلُ عشرُ دقائق.
const (
	pinMaxTries    = 5
	pinLockFor     = 10 * time.Minute
	pinChallengeTL = 5 * time.Minute
)

// pinResetPurpose **غرضُ رمزِ الاستعادة** — لا يُخلط برمز توثيقِ الرقم.
const pinResetPurpose = "pin_reset"

var (
	// ErrPinRequired **لا تُصدَر جلسةٌ قبل الرمز** — والواجهةُ تعرضه.
	ErrPinRequired = httpx.NewError(http.StatusUnauthorized, "pin_required", "errors.pin_required")
	// ErrPinInvalid **رمزٌ خاطئ** — ولا يُقال كم بقي من محاولات.
	ErrPinInvalid = httpx.NewError(http.StatusUnauthorized, "pin_invalid", "errors.pin_invalid")
	// ErrPinLocked **نفدت المحاولات.**
	ErrPinLocked = httpx.NewError(http.StatusTooManyRequests, "pin_locked", "errors.pin_locked")
	// ErrPinBadFormat **أربعةُ أرقامٍ لا غير.**
	ErrPinBadFormat = httpx.NewError(http.StatusBadRequest, "pin_format", "errors.pin_format")
	// ErrPinChallenge **تحدٍّ منتهٍ أو مستعمَل** — يعود إلى الكلمة من جديد.
	ErrPinChallenge = httpx.NewError(http.StatusUnauthorized, "pin_challenge", "errors.pin_challenge")
	// ErrPinNotSet **لا رمزَ ليُبدَّل.**
	ErrPinNotSet = httpx.NewError(http.StatusConflict, "pin_not_set", "errors.pin_not_set")
)

// NeedsPin **أيلزم هذا الحسابَ رمزٌ ثانٍ؟**
//
// **الأدمنُ وحدَه** — بقرار المالك. **والعملياتُ والماليةُ خارجَه**، ولو
// وُسّع لهم لاحقاً فالتبديلُ هنا في سطرٍ واحد.
func NeedsPin(roles []string) bool {
	for _, r := range roles {
		if r == RoleAdmin {
			return true
		}
	}
	return false
}

// validPin **أربعةُ محارفَ كلُّها أرقام.**
//
// **ولا يُقبل «١٢٣٤» بالأرقام العربيّة** — لوحاتُ المفاتيح تختلف، **ورمزٌ
// يُضبط بلوحةٍ ولا يُقبل بأخرى يُقفل صاحبَه على نفسه.**
func validPin(pin string) bool {
	if len(pin) != pinLen {
		return false
	}
	for _, c := range pin {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// PinChallenge **يبدأ خطوةَ الرمز** — يعيد تحدّياً يعرف صاحبَه.
func (s *Service) PinChallenge(ctx context.Context, userID string) (string, error) {
	raw, _, err := auth.NewOpaqueToken()
	if err != nil {
		return "", err
	}
	if err := s.rdb.Set(ctx, "pin:ch:"+raw, userID, pinChallengeTL).Err(); err != nil {
		return "", err
	}
	return raw, nil
}

// pinChallengeUser **يقرأ صاحبَ التحدّي ويستهلكه.**
//
// **والاستهلاكُ عند النجاح لا عند القراءة**: محاولةٌ خاطئةٌ تُبقيه حيّاً
// **وإلّا لَزم إدخالُ الكلمة من جديد بعد كلّ خطأ** — وهو عقابٌ على خطأِ
// إصبع، **ويدفع صاحبَه إلى كتابة الرمز في مكانٍ ما.**
func (s *Service) pinChallengeUser(ctx context.Context, challenge string) (string, error) {
	uid, err := s.rdb.Get(ctx, "pin:ch:"+challenge).Result()
	if errors.Is(err, redis.Nil) || uid == "" {
		return "", ErrPinChallenge
	}
	if err != nil {
		return "", err
	}
	return uid, nil
}

// pinTries **يعدّ المحاولات ويقفل عند الخامسة.**
func (s *Service) pinTries(ctx context.Context, userID string) error {
	key := "pin:try:" + userID
	n, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		// **وعطبُ الذاكرة لا يفتح الباب** — يُغلق.
		return ErrPinLocked
	}
	if n == 1 {
		s.rdb.Expire(ctx, key, pinLockFor)
	}
	if n > pinMaxTries {
		return ErrPinLocked
	}
	return nil
}

// VerifyPin **يتحقّق ويُصدر الجلسة.**
func (s *Service) VerifyPin(ctx context.Context, challenge, pin, userAgent, ip string) (*AuthResult, error) {
	uid, err := s.pinChallengeUser(ctx, challenge)
	if err != nil {
		return nil, err
	}
	if err := s.pinTries(ctx, uid); err != nil {
		return nil, err
	}
	hash, err := s.repo.AdminPinHash(ctx, uid)
	if err != nil {
		return nil, err
	}
	if hash == "" {
		return nil, ErrPinNotSet
	}
	ok, err := auth.VerifyPassword(pin, hash)
	if err != nil || !ok {
		return nil, ErrPinInvalid
	}
	user, _, err := s.repo.UserByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	// **ونجاحٌ يمسح العدّادَ ويستهلك التحدّي** — فلا يُعاد استعماله.
	s.rdb.Del(ctx, "pin:try:"+uid, "pin:ch:"+challenge)
	s.repo.Audit(ctx, &uid, "auth.pin_verified", "user", uid, ip, map[string]any{})
	return s.issueSession(ctx, user, userAgent, ip, "auth.pin_login", "")
}

// SetPinFirstTime **يضبط الرمزَ الأوّلَ ويُصدر الجلسة.**
//
// **ومن لا رمزَ له لا يُترك يدخل بلا ضبط** — (قرارُ المالك: «يدخله الأدمن»).
// **وخيارٌ يُؤجَّل يُنسى**، فيبقى أخطرُ حسابٍ بطبقةٍ واحدة.
func (s *Service) SetPinFirstTime(ctx context.Context, challenge, pin, userAgent, ip string) (*AuthResult, error) {
	uid, err := s.pinChallengeUser(ctx, challenge)
	if err != nil {
		return nil, err
	}
	if !validPin(pin) {
		return nil, ErrPinBadFormat
	}
	hash, err := s.repo.AdminPinHash(ctx, uid)
	if err != nil {
		return nil, err
	}
	// **ولا يُضبط فوق رمزٍ قائم** — تبديلُه يمرّ بالقديم.
	if hash != "" {
		return nil, ErrPinInvalid
	}
	enc, err := auth.HashPassword(pin)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetAdminPin(ctx, uid, enc); err != nil {
		return nil, err
	}
	user, _, err := s.repo.UserByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	s.rdb.Del(ctx, "pin:ch:"+challenge)
	s.repo.Audit(ctx, &uid, "auth.pin_set", "user", uid, ip, map[string]any{})
	return s.issueSession(ctx, user, userAgent, ip, "auth.pin_login", "")
}

// ChangePin **يبدّل الرمزَ من اللوحة — بالقديم.**
//
// (قرارُ المالك: «يستطيع تبديله من لوحة التحكّم».)
//
// **والقديمُ يُطلب** — **وجلسةٌ مسروقةٌ تبدّل الرمزَ بلا قديمٍ تُغلق المالكَ
// خارجَ لوحته** وتُبقي السارقَ داخلَها.
func (s *Service) ChangePin(ctx context.Context, userID, current, next, ip string) error {
	if !validPin(next) {
		return ErrPinBadFormat
	}
	if err := s.pinTries(ctx, userID); err != nil {
		return err
	}
	hash, err := s.repo.AdminPinHash(ctx, userID)
	if err != nil {
		return err
	}
	if hash == "" {
		return ErrPinNotSet
	}
	ok, err := auth.VerifyPassword(current, hash)
	if err != nil || !ok {
		return ErrPinInvalid
	}
	enc, err := auth.HashPassword(next)
	if err != nil {
		return err
	}
	if err := s.repo.SetAdminPin(ctx, userID, enc); err != nil {
		return err
	}
	s.rdb.Del(ctx, "pin:try:"+userID)
	s.repo.Audit(ctx, &userID, "auth.pin_changed", "user", userID, ip, map[string]any{})
	return nil
}

// ResetPinRequest **بابُ النجاة — رمزُ تحقّقٍ إلى واتساب المُوثَّق.**
//
// **ومن نسي رمزَه بلا هذا الباب أُقفلت لوحتُه على نفسه** — ولا أحدَ فوقه
// يفتحها له، **فهو المالك.**
//
// **ويُشترط رقمٌ موثَّق** — لا رقمُ الحساب: **من بدّل رقمَه ولم يوثّقه لا
// يفتح به باباً.**
func (s *Service) ResetPinRequest(ctx context.Context, userID, ip string) error {
	phone, err := s.repo.VerifiedWhatsApp(ctx, userID)
	if err != nil {
		return err
	}
	if phone == "" {
		return httpx.NewError(http.StatusConflict, "whatsapp_unverified", "errors.whatsapp_unverified")
	}
	return s.sendOTPFor(ctx, phone, pinResetPurpose, "otp:pin:", ip)
}

// ResetPinConfirm **يضبط رمزاً جديداً برمز واتساب.**
func (s *Service) ResetPinConfirm(ctx context.Context, userID, code, pin, ip string) error {
	if !validPin(pin) {
		return ErrPinBadFormat
	}
	phone, err := s.repo.VerifiedWhatsApp(ctx, userID)
	if err != nil {
		return err
	}
	if phone == "" {
		return httpx.NewError(http.StatusConflict, "whatsapp_unverified", "errors.whatsapp_unverified")
	}
	valid, err := s.repo.ConsumeOTP(ctx, phone, s.hashOTP(phone, strings.TrimSpace(code)), pinResetPurpose)
	if err != nil {
		return err
	}
	if !valid {
		return ErrOTPInvalid
	}
	enc, err := auth.HashPassword(pin)
	if err != nil {
		return err
	}
	if err := s.repo.SetAdminPin(ctx, userID, enc); err != nil {
		return err
	}
	s.rdb.Del(ctx, "pin:try:"+userID)
	s.repo.Audit(ctx, &userID, "auth.pin_reset", "user", userID, ip, map[string]any{})
	return nil
}

// HasPin **أللحساب رمزٌ مضبوط؟** — تقرؤها شاشةُ «حسابي».
func (s *Service) HasPin(ctx context.Context, userID string) (bool, *time.Time, error) {
	return s.repo.AdminPinState(ctx, userID)
}
