package identity

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/media"
)

var ErrNotFound = errors.New("identity: not found")

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

func (r *Repo) pool() *pgxpool.Pool { return r.db }

func (r *Repo) getUserBy(ctx context.Context, where, arg string) (*User, string, error) {
	var u User
	var passwordHash *string
	err := r.db.QueryRow(ctx, `
		SELECT u.id, u.phone, u.full_name, u.status, u.password_hash, u.invite_code, u.must_change_password, am.thumb_path, u.last_seen_at, u.created_at,
		       COALESCE(array_agg(ur.role_code) FILTER (WHERE ur.role_code IS NOT NULL), '{}')
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN media am ON am.id = u.avatar_media_id
		WHERE `+where+` GROUP BY u.id, am.thumb_path`, arg).
		Scan(&u.ID, &u.Phone, &u.FullName, &u.Status, &passwordHash, &u.InviteCode, &u.MustChangePassword, &u.AvatarURL, &u.LastSeenAt, &u.CreatedAt, &u.Roles)
	u.AvatarURL = media.URLForPtr(u.AvatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	hash := ""
	if passwordHash != nil {
		hash = *passwordHash
		u.HasPassword = true
	}
	return &u, hash, nil
}

func (r *Repo) UserByPhone(ctx context.Context, phone string) (*User, string, error) {
	return r.getUserBy(ctx, "u.phone = $1", phone)
}

func (r *Repo) UserByID(ctx context.Context, id string) (*User, string, error) {
	return r.getUserBy(ctx, "u.id = $1", id)
}

func (r *Repo) UserByInviteCode(ctx context.Context, code string) (*User, string, error) {
	return r.getUserBy(ctx, "u.invite_code = $1", code)
}

// CreateUserWithRole ينشئ مستخدماً جديداً بدور واحد (ضمن معاملة).
func (r *Repo) CreateUserWithRole(ctx context.Context, phone, fullName, role string) (*User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	if err := tx.QueryRow(ctx,
		`INSERT INTO users (phone, full_name) VALUES ($1, $2) RETURNING id`,
		phone, fullName).Scan(&id); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_code) VALUES ($1, $2)`, id, role); err != nil {
		return nil, err
	}
	// المندوب/المتجر/السائق هم أيضاً زبائن (انظر GrantRole).
	if grantsCustomer(role) {
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_code) VALUES ($1, 'customer')
			 ON CONFLICT DO NOTHING`, id); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if role == "sales" {
		if err := r.EnsureInviteCode(ctx, id); err != nil {
			return nil, err
		}
	}
	u, _, err := r.UserByID(ctx, id)
	return u, err
}

// ListUsers بحث وترشيح وترقيم صفحات لإدارة المستخدمين.
func (r *Repo) ListUsers(ctx context.Context, query, role string, onlineOnly bool, status string, limit, offset int) ([]User, int, error) {
	// role الخاص "staff" = موظفو المنصة (عمليات + مالية)
	where := `WHERE ($1 = '' OR u.phone ILIKE '%'||$1||'%' OR u.full_name ILIKE '%'||$1||'%' OR u.invite_code ILIKE '%'||$1||'%')
	          AND ($2 = '' OR EXISTS (
	              SELECT 1 FROM user_roles fr WHERE fr.user_id = u.id
	              AND (fr.role_code = $2 OR ($2 = 'staff' AND fr.role_code IN ('ops','finance')))))
	          AND (NOT $3 OR u.last_seen_at > now() - interval '2 minutes')
	          AND ($4 = '' OR u.status = $4)`

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM users u `+where+` AND NOT EXISTS (SELECT 1 FROM user_roles ar WHERE ar.user_id = u.id AND ar.role_code = 'admin')`, query, role, onlineOnly, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.phone, u.full_name, u.status, u.password_hash IS NOT NULL, u.invite_code, am.thumb_path, u.last_seen_at, u.created_at,
		       COALESCE(array_agg(ur.role_code) FILTER (WHERE ur.role_code IS NOT NULL), '{}')
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN media am ON am.id = u.avatar_media_id
		`+where+`
		GROUP BY u.id, am.thumb_path
		HAVING NOT bool_or(ur.role_code = 'admin')
		ORDER BY u.created_at DESC
		LIMIT $5 OFFSET $6`, query, role, onlineOnly, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Phone, &u.FullName, &u.Status, &u.HasPassword, &u.InviteCode, &u.AvatarURL, &u.LastSeenAt, &u.CreatedAt, &u.Roles); err != nil {
			return nil, 0, err
		}
		u.AvatarURL = media.URLForPtr(u.AvatarURL)
		if false {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// SetFullName يضبط اسم المستخدم (يُستعمل عند إكمال تسجيل حساب زبون).
func (r *Repo) SetFullName(ctx context.Context, userID, name string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET full_name = $2, updated_at = now() WHERE id = $1`, userID, name)
	return err
}

// SetPhone يحدّث رقم هاتف المستخدم (بعد تأكيد الرمز على الرقم الجديد).
func (r *Repo) SetPhone(ctx context.Context, userID, phone string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET phone = $2, updated_at = now() WHERE id = $1`, userID, phone)
	return err
}

// SetAvatar يضبط صورة المستخدم لنفسه (mediaID فارغ = إزالة الصورة).
func (r *Repo) SetAvatar(ctx context.Context, userID, mediaID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET avatar_media_id = NULLIF($2, '')::uuid, updated_at = now()
		WHERE id = $1`, userID, mediaID)
	return err
}

// UpdateUser يعدّل الاسم و/أو الحالة — يعيد ErrNotFound لمعرف غير موجود.
func (r *Repo) UpdateUser(ctx context.Context, userID string, fullName, status, avatarMediaID, statusReason, phone, adminNotes *string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE users SET
			full_name = COALESCE($2, full_name),
			status    = COALESCE($3, status),
			avatar_media_id = CASE WHEN $4::text IS NULL THEN avatar_media_id
			                       ELSE NULLIF($4, '')::uuid END,
			status_reason = COALESCE($5, status_reason),
			phone     = COALESCE($6, phone),
			admin_notes = COALESCE($7, admin_notes),
			updated_at = now()
		WHERE id = $1`, userID, fullName, status, avatarMediaID, statusReason, phone, adminNotes)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) RevokeRole(ctx context.Context, userID, role string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role_code = $2`, userID, role)
	return err
}

// FieldRolesAreCustomers: هل يُمنح صاحبُ دورٍ ميدانيّ دورَ الزبون تلقائياً؟
//
// **مُطفأةٌ مؤقّتاً بقرار المالك (٢٠٢٦-٠٨-٠١) لأجل التجربة.**
//
// السببُ عمليّ لا مبدئيّ: حين يحمل السائقُ دورَ الزبون أيضاً، يستطيع أن يطلب
// طلباً ثم يوصّله بنفسه — فتختلط الأطراف في تجربةٍ يُراد منها أن تفصلها. وقد
// وقع ذلك فعلاً في أوّل طلبٍ حقيقيّ: ظهر «عمر الشيخ» زبوناً وهو السائق.
//
// **وهي ميزةٌ صحيحة تعود بعد التجربة**: السائق إنسانٌ يطلب عشاءه كما يوصّل
// عشاء غيره، ومنعُه منه في الإنتاج حرمانٌ بلا سبب.
//
// وموضعُ الإطفاء هنا وحده: `GrantRole` و`CreateUserWithRole` وزراعةُ البيانات
// كلُّها تمرّ به. **فالعودة سطرٌ واحد** — لا مطاردةَ مواضع.
// **وأُعيدت بقرار المالك ٢٠٢٦-٠٨-١٠**: «السائقُ يمكن أن يكون زبوناً، وكذلك
// المتجرُ والمندوبُ وصاحبُ المنصّة والموظّفون… يعني دورين فقط».
//
// **والخلطُ الذي أُطفئت لأجله لم يعد ممكناً**: الطلبُ لا يُعرض على سائقه
// (`offered_driver_id` بالدور)، **والتجربةُ التي كُتبت لها انتهت.**
const FieldRolesAreCustomers = true

// grantsCustomer يحدد الأدوار الميدانية التي يُمنح صاحبها دور الزبون تلقائياً
// (المندوب/المتجر/السائق) — لا الأدوار الداخلية (أدمن/عمليات/مالية).
// **وكلُّ دورٍ يجلب الزبونَ معه لا الميدانيَّةُ وحدَها.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «وصاحبُ المنصّة أيضاً، والموظّفون أيضاً».)
//
// **وصاحبُ المنصّة يطلب عشاءه كما يطلبه غيرُه** — ومن لا يستطيع أن يطلب من
// منصّته لا يرى ما يراه زبائنُه. **وهو أوّلُ من يجب أن يراه.**
func grantsCustomer(role string) bool {
	if !FieldRolesAreCustomers {
		return false
	}
	return role != RoleCustomer
}

func (r *Repo) GrantRole(ctx context.Context, userID, role string, grantedBy *string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_code, granted_by) VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`, userID, role, grantedBy)
	// المندوب وصاحب المتجر والسائق هم أيضاً زبائن — يحصلون تلقائياً على دور
	// الزبون كي تظهر لهم كل خيارات الزبون (تصفّح، طلب، محفظة) إضافة لخياراتهم.
	if err == nil && grantsCustomer(role) {
		_, err = r.db.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1, 'customer')
			ON CONFLICT DO NOTHING`, userID)
	}
	if err == nil && role == "sales" {
		err = r.EnsureInviteCode(ctx, userID)
	}
	return err
}

// EnsureInviteCode يولّد كود دعوة فريداً للمستخدم إن لم يكن لديه (لدور المندوب).
func (r *Repo) EnsureInviteCode(ctx context.Context, userID string) error {
	const charset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789" // بلا أحرف ملتبسة
	for attempt := 0; attempt < 5; attempt++ {
		b := make([]byte, 5)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		for i := range b {
			b[i] = charset[int(b[i])%len(charset)]
		}
		code := "RH-" + string(b)
		_, err := r.db.Exec(ctx, `
			UPDATE users SET invite_code = $2 WHERE id = $1 AND invite_code IS NULL`,
			userID, code)
		if err == nil {
			return nil
		}
	}
	return errors.New("identity: failed to generate invite code")
}

// SetPassword يضبط كلمة المرور — ويرفع إجبار التبديل لأن صاحب الحساب هو من ضبطها.
func (r *Repo) SetPassword(ctx context.Context, userID, hash string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET password_hash = $2, must_change_password = false, updated_at = now()
		WHERE id = $1`, userID, hash)
	return err
}

// SetTempPassword كلمة مرور وضعها طرف ثالث — تُجبر صاحب الحساب على تبديلها.
func (r *Repo) SetTempPassword(ctx context.Context, userID, hash string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET password_hash = $2, must_change_password = true, updated_at = now()
		WHERE id = $1`, userID, hash)
	return err
}

// --- OTP ---

func (r *Repo) CreateOTP(ctx context.Context, phone, codeHash, purpose string, ttl time.Duration) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO otp_codes (phone, code_hash, purpose, expires_at)
		VALUES ($1, $2, $3, now() + $4)`, phone, codeHash, purpose, ttl)
	return err
}

// ConsumeOTP يتحقق من آخر رمز فعّال ويستهلكه ذرّياً؛ يعيد false عند الفشل.
// كل محاولة فاشلة تُحتسب، وبعد 5 محاولات يُبطل الرمز.
func (r *Repo) ConsumeOTP(ctx context.Context, phone, codeHash, purpose string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE otp_codes SET consumed_at = now()
		WHERE id = (
			SELECT id FROM otp_codes
			WHERE phone = $1 AND purpose = $3 AND consumed_at IS NULL
			  AND expires_at > now() AND attempts < 5
			ORDER BY created_at DESC LIMIT 1
		) AND code_hash = $2`, phone, codeHash, purpose)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}
	// محاولة فاشلة — تُسجَّل على آخر رمز فعّال
	_, err = r.db.Exec(ctx, `
		UPDATE otp_codes SET attempts = attempts + 1
		WHERE id = (
			SELECT id FROM otp_codes
			WHERE phone = $1 AND purpose = $2 AND consumed_at IS NULL AND expires_at > now()
			ORDER BY created_at DESC LIMIT 1
		)`, phone, purpose)
	return false, err
}

// CheckOTP يتحقق من صحة الرمز **بلا أن يستهلكه** — للتحقّق المسبق في تسجيل
// حسابٍ جديد.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «يدخل الرقم، يضغط إرسال رمز، **لا تظهر المعلومات
//
//	إلّا بعد التحقّق من الرمز**، ثمّ تظهر معلومات إنشاء الحساب».)
//
// # ولماذا لا يُستهلك
//
// **الاستهلاكُ يُبطل الرمز** — ثمّ يملأ صاحبُه الاسمَ وكلمتَي المرور ويوافق
// ويضغط «إنشاء حساب»، **فيُرفض رمزُه الذي صُدِّق قبل دقيقة.**
//
// # وحدُّ المحاولات هو نفسُه
//
// **فحصٌ لا يعدّ الفاشلةَ يصير أداةَ تخمين**: رمزٌ من ستّة أرقامٍ يُكسر بمليون
// نداءٍ إن لم يُبطَل. **فتُحتسب هنا كما تُحتسب في `ConsumeOTP`**، وبعد خمسٍ
// يُبطل الرمزُ حتّى للصحيح.
//
// **والنافذةُ لم تتّسع**: الرمزُ كان صالحاً مدّةَ حياته قبلَ هذا وبعدَه —
// **وهذا يقرأ ولا يمدّ.**
func (r *Repo) CheckOTP(ctx context.Context, phone, codeHash, purpose string) (bool, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		SELECT id FROM otp_codes
		WHERE phone = $1 AND purpose = $3 AND consumed_at IS NULL
		  AND expires_at > now() AND attempts < 5 AND code_hash = $2
		ORDER BY created_at DESC LIMIT 1`, phone, codeHash, purpose).Scan(&id)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	// محاولة فاشلة — تُسجَّل على آخر رمز فعّال، كما في `ConsumeOTP` حرفاً بحرف
	_, err = r.db.Exec(ctx, `
		UPDATE otp_codes SET attempts = attempts + 1
		WHERE id = (
			SELECT id FROM otp_codes
			WHERE phone = $1 AND purpose = $2 AND consumed_at IS NULL AND expires_at > now()
			ORDER BY created_at DESC LIMIT 1
		)`, phone, purpose)
	return false, err
}

// --- Refresh Tokens ---

// StoreRefresh يخزّن توكن تجديد داخل عائلة جلسة ويعيد معرّفها.
// sessionID فارغ = عائلة جديدة تولّدها القاعدة — ونحتاج معرّفها لنضعه في توكن
// الوصول، فبه وحده يصير إبطال الجلسة فورياً.
func (r *Repo) StoreRefresh(ctx context.Context, userID, tokenHash string, ttl time.Duration, userAgent, ip, sessionID string) (string, error) {
	var sid any
	if sessionID != "" {
		sid = sessionID
	}
	var out string
	err := r.db.QueryRow(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent, ip, session_id)
		VALUES ($1, $2, now() + $3, $4, $5, COALESCE($6::uuid, gen_random_uuid()))
		RETURNING session_id::text`,
		userID, tokenHash, ttl, userAgent, ip, sid).Scan(&out)
	return out, err
}

// ActiveSessionIDs كل عائلات الجلسات الفعّالة لحساب — لإبطالها دفعة واحدة.
func (r *Repo) ActiveSessionIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT session_id::text FROM refresh_tokens
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var sid string
		if rows.Scan(&sid) == nil {
			out = append(out, sid)
		}
	}
	return out, rows.Err()
}

// RevokeRefresh يُبطل التوكن ويعيد صاحبه وعائلة جلسته — ErrNotFound إن كان غير صالح.
func (r *Repo) RevokeRefresh(ctx context.Context, tokenHash string) (userID, sessionID string, err error) {
	err = r.db.QueryRow(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()
		RETURNING user_id, session_id::text`, tokenHash).Scan(&userID, &sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return userID, sessionID, err
}

// ActiveSessionID عائلة الجلسة الفعّالة للحساب (أحدث توكن سارٍ). تُستعمل عند
// التسليم بين تطبيقات المنصة كي يبقى الانتقال داخل نفس الجلسة لا جلسة جديدة.
func (r *Repo) ActiveSessionID(ctx context.Context, userID string) (string, error) {
	var sid string
	err := r.db.QueryRow(ctx, `
		SELECT session_id::text FROM refresh_tokens
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC LIMIT 1`, userID).Scan(&sid)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return sid, err
}

// --- Audit ---

func (r *Repo) Audit(ctx context.Context, actorID *string, action, entity, entityID, ip string, details map[string]any) {
	d := details
	if d == nil {
		d = map[string]any{}
	}
	// فشل التدقيق لا يُفشل العملية الأصلية لكنه يُسجَّل
	_, _ = r.db.Exec(ctx, `
		INSERT INTO audit_log (actor_user_id, action, entity, entity_id, ip, details)
		VALUES ($1, $2, $3, $4, $5, $6)`, actorID, action, entity, entityID, ip, d)
}

// RevokeSession يُبطل كل توكنات عائلة جلسة واحدة (تمتد على تطبيقات المنصة).
func (r *Repo) RevokeSession(ctx context.Context, userID, sessionID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE user_id = $1 AND session_id = $2::uuid
		  AND revoked_at IS NULL AND expires_at > now()`, userID, sessionID)
	return err
}

// RevokeAllTokens يُبطل كل توكنات التجديد الفعالة لحساب — يعيد عددها.
func (r *Repo) RevokeAllTokens(ctx context.Context, userID string) (int, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()`, userID)
	return int(tag.RowsAffected()), err
}
