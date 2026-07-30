package identity

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("identity: not found")

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

func (r *Repo) getUserBy(ctx context.Context, where, arg string) (*User, string, error) {
	var u User
	var passwordHash *string
	err := r.db.QueryRow(ctx, `
		SELECT u.id, u.phone, u.full_name, u.status, u.password_hash, u.invite_code, u.created_at,
		       COALESCE(array_agg(ur.role_code) FILTER (WHERE ur.role_code IS NOT NULL), '{}')
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		WHERE `+where+` GROUP BY u.id`, arg).
		Scan(&u.ID, &u.Phone, &u.FullName, &u.Status, &passwordHash, &u.InviteCode, &u.CreatedAt, &u.Roles)
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
func (r *Repo) ListUsers(ctx context.Context, query, role string, limit, offset int) ([]User, int, error) {
	where := `WHERE ($1 = '' OR u.phone ILIKE '%'||$1||'%' OR u.full_name ILIKE '%'||$1||'%')
	          AND ($2 = '' OR EXISTS (SELECT 1 FROM user_roles fr WHERE fr.user_id = u.id AND fr.role_code = $2))`

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM users u `+where, query, role).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.phone, u.full_name, u.status, u.password_hash IS NOT NULL, u.invite_code, u.created_at,
		       COALESCE(array_agg(ur.role_code) FILTER (WHERE ur.role_code IS NOT NULL), '{}')
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		`+where+`
		GROUP BY u.id
		ORDER BY u.created_at DESC
		LIMIT $3 OFFSET $4`, query, role, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Phone, &u.FullName, &u.Status, &u.HasPassword, &u.InviteCode, &u.CreatedAt, &u.Roles); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// UpdateUser يعدّل الاسم و/أو الحالة — يعيد ErrNotFound لمعرف غير موجود.
func (r *Repo) UpdateUser(ctx context.Context, userID string, fullName, status *string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE users SET
			full_name = COALESCE($2, full_name),
			status    = COALESCE($3, status),
			updated_at = now()
		WHERE id = $1`, userID, fullName, status)
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

func (r *Repo) GrantRole(ctx context.Context, userID, role string, grantedBy *string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_code, granted_by) VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`, userID, role, grantedBy)
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

func (r *Repo) SetPassword(ctx context.Context, userID, hash string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, userID, hash)
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

// --- Refresh Tokens ---

func (r *Repo) StoreRefresh(ctx context.Context, userID, tokenHash string, ttl time.Duration, userAgent, ip string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent, ip)
		VALUES ($1, $2, now() + $3, $4, $5)`, userID, tokenHash, ttl, userAgent, ip)
	return err
}

// RevokeRefresh يُبطل التوكن ويعيد صاحبه — يعيد ErrNotFound إذا كان غير صالح.
func (r *Repo) RevokeRefresh(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	err := r.db.QueryRow(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()
		RETURNING user_id`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return userID, err
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
