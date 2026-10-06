package identity

// ══════════════════════════════════════════════════════════════════════
// **قسمُ الحسابات — الدخولُ والرقم** (قراراتُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// # كلمةُ السرّ يولّدها النظام
//
// **كان الموظّفُ يكتبها بيده مرّتين** — فيعرفها هو وصاحبُها إلى الأبد، **ولا
// مدّةَ لها ولا رسالة.** والقرار: النظامُ يولّدها، **وتُرسَل برسالة**، **وتنتهي
// بعد ٧٢ ساعةً إن لم تُستعمل**، **ويُجبَر صاحبُها على تبديلها عند أوّل دخول** —
// في كلّ إنشاءِ حسابٍ وفي كلّ إعادةِ كلمة.
//
// # وتغييرُ الرقم ينقل الحسابَ كلَّه
//
// **بمحفظته وسجلّه** — فهو أخطرُ تعديل: **بكلمةِ سرِّ الموظّف** (خطوةُ تحقّق)،
// **ويُخرج كلَّ الجلسات**، **ويُبلَّغ الرقمُ القديم**، ويُسجَّل القديمُ والجديد.

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ErrTempPasswordExpired **انتهت مهلةُ الكلمة المؤقّتة** — تُطلب غيرُها من الإدارة.
var ErrTempPasswordExpired = httpx.NewError(http.StatusUnauthorized,
	"temp_password_expired", "errors.temp_password_expired")

// tempLetters **حروفٌ لا تلتبس** — لا `o` ولا `l` ولا `i`: تُقرأ من رسالةٍ على هاتف.
const tempLetters = "abcdefghjkmnpqrstuvwxyz"

// GenerateTempPassword **كلمةٌ سهلةٌ تُكتب** (قرارُ المالك ٢٠٢٦-١٠-٠٦: «صعبة جدّاً… لازم تكون سهلة،
// حرف وأرقام… وهو باسورد مؤقّت بالنهاية») — **حرفان ثمّ أرقام**، مثل `rk482915`، بطولِ حدِّ
// المنصّة وثمانيةٍ على الأقلّ. **والأمانُ من غيرها**: تنتهي بعد ساعات، **ويُجبَر صاحبُها على
// تبديلها عند أوّل دخول**، ومحاولاتُ الدخول محدودة.
func GenerateTempPassword(minLen int) (string, error) {
	n := 8
	if minLen > n {
		n = minLen
	}
	out := make([]byte, n)
	for i := range out {
		alphabet := "0123456789"
		if i < 2 {
			alphabet = tempLetters
		}
		k, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		out[i] = alphabet[k.Int64()]
	}
	return string(out), nil
}

// TempPasswordHours مهلةُ الكلمة المؤقّتة — من الإعدادات، وافتراضُها ٧٢.
func (s *Service) TempPasswordHours(ctx context.Context) int64 {
	h := s.intSetting(ctx, "security.temp_password_hours", 72)
	if h <= 0 {
		return 72
	}
	return h
}

// IssueTempPassword **يولّد كلمةً مؤقّتةً ويضعها** — ويُرجعها نصّاً لتُرسَل وحدَها.
//
// **وفعلُ استرداد**: تُقطع الجلساتُ كلُّها (حِقبةٌ وإبطال) كما في `AdminResetPassword`،
// **ولا يبقى صاحبُ وصولٍ قديمٍ داخلاً.** **والنصُّ لا يُكتب في سجلٍّ ولا تدقيق.**
func (s *Service) IssueTempPassword(ctx context.Context, actorID, userID, ip, action string) (string, time.Time, error) {
	if err := guardAccountAdmin(ctx, s.repo.pool(), actorID, userID); err != nil {
		return "", time.Time{}, err
	}
	plain, err := GenerateTempPassword(int(s.intSetting(ctx, "security.password_min_length", minPasswordLn)))
	if err != nil {
		return "", time.Time{}, err
	}
	hash, err := auth.HashPassword(plain)
	if err != nil {
		return "", time.Time{}, err
	}
	hours := s.TempPasswordHours(ctx)
	tx, err := s.repo.pool().Begin(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var expires time.Time
	if err := tx.QueryRow(ctx, `
		UPDATE users
		   SET password_hash = $2, must_change_password = true,
		       temp_password_expires_at = now() + ($3::int * interval '1 hour'),
		       sessions_revoked_at = now(), sessions_kept_session_id = NULL,
		       updated_at = now()
		 WHERE id = $1::uuid
		RETURNING temp_password_expires_at`, userID, hash, hours).Scan(&expires); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", time.Time{}, httpx.ErrNotFound
		}
		return "", time.Time{}, err
	}
	sids, err := revokeRefreshTx(ctx, tx, userID)
	if err != nil {
		return "", time.Time{}, err
	}
	if err := AuditTx(ctx, tx, &actorID, action, "user", userID, ip,
		map[string]any{"sessions_revoked": len(sids), "expires_at": expires,
			"system_generated": true}); err != nil {
		return "", time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", time.Time{}, err
	}
	s.afterRevoke(ctx, userID, sids)
	return plain, expires, nil
}

// revokeRefreshTx يُبطل رموزَ التجديد الحيّة ويُرجع معرّفاتِ جلساتها.
func revokeRefreshTx(ctx context.Context, q dbtx.Querier, userID string) ([]string, error) {
	rows, err := q.Query(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		 WHERE user_id = $1::uuid AND revoked_at IS NULL AND expires_at > now()
		RETURNING session_id::text`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sids []string
	for rows.Next() {
		var sid string
		if rows.Scan(&sid) == nil {
			sids = append(sids, sid)
		}
	}
	return sids, rows.Err()
}

// afterRevoke **بعد التثبيت**: مُسرِّعُ الرفض ووجهاتُ الدفع وخبيئةُ الحال.
func (s *Service) afterRevoke(ctx context.Context, userID string, sids []string) {
	if s.rdb != nil && s.tokens != nil {
		for _, sid := range sids {
			s.rdb.Set(ctx, sessionRevokedKey(sid), "1", s.tokens.AccessTTL()+time.Minute)
		}
	}
	if _, err := s.repo.DeleteDeviceTokensOfUser(ctx, userID); err != nil && s.logger != nil {
		s.logger.Error("تعذّر قطعُ وجهاتِ الدفع", "user", userID, "error", err)
	}
	if s.rdb != nil {
		s.invalidateStatusCache(ctx, userID)
	}
}

// TempPasswordPending **أكلمتُه مؤقّتةٌ من النظام لم تُبدَّل بعد؟** ومتى تنتهي.
func (s *Service) TempPasswordPending(ctx context.Context, userID string) (bool, *time.Time) {
	if s == nil || s.repo == nil || userID == "" {
		return false, nil
	}
	var exp *time.Time
	if err := s.repo.pool().QueryRow(ctx,
		`SELECT temp_password_expires_at FROM users WHERE id = $1::uuid AND must_change_password`,
		userID).Scan(&exp); err != nil || exp == nil {
		return false, nil
	}
	return true, exp
}

// tempExpired **كلمةٌ مؤقّتةٌ انقضت مهلتُها** — لا تفتح الحساب.
func (s *Service) tempExpired(ctx context.Context, userID string) bool {
	pending, exp := s.TempPasswordPending(ctx, userID)
	return pending && exp != nil && time.Now().After(*exp)
}

// AdminChangePhone **يبدّل رقمَ الحساب** — ويُخرج كلَّ جلساته ويُرجع الرقمَ القديم.
//
// **والتوثيقُ بواتساب يسقط مع الرقم** — كما في `SetPhone`: رقمٌ جديدٌ لم يُوثَّق.
func (s *Service) AdminChangePhone(ctx context.Context, actorID, userID, rawPhone, ip string, approvedBy string) (string, string, error) {
	if err := guardAccountAdmin(ctx, s.repo.pool(), actorID, userID); err != nil {
		return "", "", err
	}
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return "", "", ErrInvalidPhone
	}
	tx, err := s.repo.pool().Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var old string
	if err := tx.QueryRow(ctx, `SELECT phone FROM users WHERE id = $1::uuid FOR UPDATE`,
		userID).Scan(&old); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", httpx.ErrNotFound
		}
		return "", "", err
	}
	if old == phone {
		return old, phone, nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users
		   SET phone = $2,
		       whatsapp_phone = CASE WHEN whatsapp_phone = $2 THEN whatsapp_phone ELSE NULL END,
		       whatsapp_verified_at = CASE WHEN whatsapp_phone = $2 THEN whatsapp_verified_at ELSE NULL END,
		       sessions_revoked_at = now(), sessions_kept_session_id = NULL,
		       updated_at = now()
		 WHERE id = $1::uuid`, userID, phone); err != nil {
		if isUniqueViolation(err) {
			return "", "", ErrPhoneTaken
		}
		return "", "", err
	}
	sids, err := revokeRefreshTx(ctx, tx, userID)
	if err != nil {
		return "", "", err
	}
	details := map[string]any{"old_phone": old, "new_phone": phone,
		"sessions_revoked": len(sids)}
	if approvedBy != "" {
		details["approved_by"] = approvedBy
	}
	if err := AuditTx(ctx, tx, &actorID, "admin.phone_change", "user", userID, ip, details); err != nil {
		return "", "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}
	s.afterRevoke(ctx, userID, sids)
	return old, phone, nil
}

// AuditBestEffort قيدُ تدقيقٍ أفضلُ جهد — لمن يملك الخدمةَ لا الطلب.
func (s *Service) AuditBestEffort(ctx context.Context, actorID *string, action, entity, entityID, ip string, details map[string]any) {
	s.repo.Audit(ctx, actorID, action, entity, entityID, ip, details)
}

// SetTempExpiry **يضبط مهلةَ كلمةٍ مؤقّتةٍ كُتبت سلفاً** — لصاحب متجرٍ أُنشئ مع متجره.
func (s *Service) SetTempExpiry(ctx context.Context, userID string) (time.Time, error) {
	var exp time.Time
	err := s.repo.pool().QueryRow(ctx, `
		UPDATE users SET temp_password_expires_at = now() + ($2::int * interval '1 hour')
		 WHERE id = $1::uuid AND must_change_password
		RETURNING temp_password_expires_at`, userID, s.TempPasswordHours(ctx)).Scan(&exp)
	return exp, err
}
