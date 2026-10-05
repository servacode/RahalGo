package referrals

// حجزُ الدعوة بالرقم — **الصديقُ لا ينسخ رمزاً** (قرارُ المالك ٢٠٢٦-١٠-٠٥).
//
// # المسألة
//
// **من نزّل التطبيقَ ملفّاً مباشراً لا `referrer` له**، وكانت صفحةُ الدعوة
// تطلب منه أن ينسخ الرمزَ بيده ثمّ يلصقه — **ورمزٌ يُنسخ بيدٍ يضيع.**
//
// # الحلّ
//
// الصديقُ يكتب رقمَه في الصفحة ⇒ `ClaimInvite` يحفظ (رقم، رمز) ⇒ وعند تأكيد
// تسجيله **بلا `ref`** يقرأ `TakeClaim` الحجزَ برقمه فيُنسب ثمّ يُحذف.
//
// **حجزٌ واحدٌ للرقم والأحدثُ يغلب** — آخرُ رابطٍ ضغطه هو ما قصده.
// **ويسقط بعد ثلاثين يوماً** (`ClaimTTL`).

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ClaimTTL **عمرُ الحجز** — بعده لا يُنسب.
const ClaimTTL = 30 * 24 * time.Hour

// ErrPhoneRegistered **الرقمُ له حسابٌ أصلاً** — الدعوةُ لمن لم يسجّل.
//
// **ويُردّ برمزٍ يُفهم** لا بنجاحٍ صامت: من له حسابٌ يُقال له «افتح التطبيق»،
// **وحجزٌ لن يُقرأ أبداً وعدٌ كاذب.** والمنادي يحوّله إلى `phone_taken`.
var ErrPhoneRegistered = errors.New("referrals: phone already registered")

// ClaimInvite **يحجز الدعوةَ لرقمٍ لم يسجّل بعد.**
//
// `phone` مطبَّعٌ مسبقاً (`identity.NormalizePhone`) — **بصيغة `users.phone`
// نفسِها**، وإلّا لم يطابقه التسجيل.
func (s *Service) ClaimInvite(ctx context.Context, code, phone string) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return ErrBadCode
	}
	var exists bool
	if err := s.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE invite_code = $1)`, code).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrBadCode
	}
	var registered bool
	if err := s.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE phone = $1)`, phone).Scan(&registered); err != nil {
		return err
	}
	if registered {
		return ErrPhoneRegistered
	}
	// **والمنتهي يُكنَس هنا** — لا مهمّةَ مجدولةً لجدولٍ صغير.
	if _, err := s.db.Exec(ctx,
		`DELETE FROM invite_claims WHERE created_at < now() - make_interval(secs => $1)`,
		ClaimTTL.Seconds()); err != nil {
		return err
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO invite_claims (phone, invite_code) VALUES ($1, $2)
		ON CONFLICT (phone) DO UPDATE
		   SET invite_code = EXCLUDED.invite_code, created_at = now()`, phone, code)
	return err
}

// TakeClaim **يأخذ حجزَ الرقم ويحذفه** — وفارغٌ إن لم يكن أو انتهى.
//
// **والحذفُ والقراءةُ في نداءٍ واحد** (`DELETE … RETURNING`): تأكيدان متزامنان
// لا يأخذان الحجزَ معاً.
func (s *Service) TakeClaim(ctx context.Context, phone string) (string, error) {
	var code string
	var at time.Time
	err := s.db.QueryRow(ctx,
		`DELETE FROM invite_claims WHERE phone = $1 RETURNING invite_code, created_at`,
		phone).Scan(&code, &at)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if time.Since(at) > ClaimTTL {
		return "", nil
	}
	return code, nil
}
