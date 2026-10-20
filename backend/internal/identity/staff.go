package identity

// ══════════════════════════════════════════════════════════════════════
// **موظّفو المتجر — يدخلون برقمهم ويعملون باسم المتجر** (قرارُ المالك ٢٠٢٦-١٠-٢٠)
// ══════════════════════════════════════════════════════════════════════
//
// «الصباحي يفتح من حسابه عادي والمسائي كمان… حسابنا مع المطعم مو مع الموظف».
//
// **الموظّفُ ليس حساباً.** رقمُه وكلمةُ سرّه تفتحان جلسةً **باسم حساب صاحب المتجر**
// — فكلُّ ما رُبط بصاحب المتجر (الطلباتُ والأصنافُ والمحفظةُ والإشعارات) يعمل له كما
// هو، **ولا يُمسّ موضعٌ واحدٌ من ربط المتجر بصاحبه.**
//
// **وما يفترق:**
//   - جلستُه لا تُزيح جلسةَ صاحبه ولا جلسةَ موظّفٍ آخر (`merchant_staff_sessions`).
//   - رمزُ وصوله يحمله (`Claims.Staff`) — فتُمنع عنه أبوابُ الحساب الشخصيّة.
//   - يدخل تطبيقَ المتجر ولوحتَه وحدَهما.
//   - حذفُه يُخرجه فوراً من كلّ أجهزته.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// MaxStaffPerStore **حدُّ الموظّفين لصاحب المتجر.**
const MaxStaffPerStore = 5

// ReasonStaffRemoved **حُذف الموظّف** — جلساتُه تُنهى بسببها.
const ReasonStaffRemoved = "staff_removed"

var (
	// ErrSessionStaffRemoved **أُزيل حسابُ الموظّف من المتجر.**
	ErrSessionStaffRemoved = httpx.NewError(http.StatusUnauthorized, "session_staff_removed", "errors.unauthorized")
	// ErrStaffLimit **بلغ صاحبُ المتجر حدَّ الموظّفين.**
	ErrStaffLimit = httpx.NewError(http.StatusConflict, "staff_limit", "errors.conflict")
	// ErrStaffPhoneTaken **الرقمُ موظّفٌ عند متجرٍ أو صاحبُ متجر.**
	ErrStaffPhoneTaken = httpx.NewError(http.StatusConflict, "staff_phone_taken", "errors.conflict")
)

// Staff **موظّفٌ كما يُعرض** — بلا كلمةِ سرّه.
type Staff struct {
	ID          string     `json:"id"`
	Phone       string     `json:"phone"`
	Name        string     `json:"name"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

type staffCtxKey struct{}

// withStaff **يحمل الموظّفَ إلى إصدار الجلسة** — من الدخول وحدَه.
func withStaff(ctx context.Context, staffID string) context.Context {
	return context.WithValue(ctx, staffCtxKey{}, staffID)
}

func staffFrom(ctx context.Context) string {
	v, _ := ctx.Value(staffCtxKey{}).(string)
	return v
}

// staffClient **أيُفتح هذا العميلُ لموظّف؟** — تطبيقُ المتجر، أو الويب (لوحةُ المتجر).
func staffClient(client string) bool {
	if client == ClientWeb {
		return true
	}
	_, role := RequiredRole(client)
	return role == "merchant"
}

// ── المخزن ────────────────────────────────────────────────────────────

func (r *Repo) staffByPhone(ctx context.Context, phone string) (id, ownerID, hash string, err error) {
	err = r.db.QueryRow(ctx, `SELECT id::text, owner_user_id::text, password_hash
		FROM merchant_staff WHERE phone = $1`, phone).Scan(&id, &ownerID, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", ErrNotFound
	}
	return
}

// StaffOfSession **موظّفُ هذه الجلسة** — وفارغٌ لجلسة صاحب حساب.
func (r *Repo) StaffOfSession(ctx context.Context, sessionID string) string {
	var id string
	_ = r.db.QueryRow(ctx, `SELECT staff_id::text FROM merchant_staff_sessions
		WHERE session_id = $1::uuid`, sessionID).Scan(&id)
	return id
}

func (r *Repo) linkStaffSession(ctx context.Context, sessionID, staffID string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO merchant_staff_sessions (session_id, staff_id) VALUES ($1::uuid, $2::uuid)
		ON CONFLICT (session_id) DO NOTHING`, sessionID, staffID)
	if err == nil {
		_, _ = r.db.Exec(ctx, `UPDATE merchant_staff SET last_login_at = now() WHERE id = $1::uuid`, staffID)
	}
	return err
}

// revokeStaffSessions **يُنهي جلساتِ موظّفٍ** — على عميلٍ بعينه، أو كلَّها إن كان فارغاً.
func (r *Repo) revokeStaffSessions(ctx context.Context, staffID, client, reason string) (sids, tokens []string, err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		UPDATE refresh_tokens SET revoked_at = now(), revoked_reason = $3
		 WHERE revoked_at IS NULL AND expires_at > now()
		   AND ($2 = '' OR client = $2)
		   AND session_id IN (SELECT session_id FROM merchant_staff_sessions WHERE staff_id = $1::uuid)
		RETURNING session_id::text`, staffID, client, reason)
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			rows.Close()
			return nil, nil, err
		}
		if !seen[sid] {
			seen[sid] = true
			sids = append(sids, sid)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(sids) > 0 {
		trows, derr := tx.Query(ctx,
			`DELETE FROM device_tokens WHERE session_id = ANY($1::uuid[]) RETURNING token`, sids)
		if derr != nil {
			return nil, nil, derr
		}
		for trows.Next() {
			var tok string
			if trows.Scan(&tok) == nil && tok != "" {
				tokens = append(tokens, tok)
			}
		}
		trows.Close()
	}
	return sids, tokens, tx.Commit(ctx)
}

// ── الدخول ────────────────────────────────────────────────────────────

// loginStaff **دخولُ موظّفٍ برقمه** — جلسةٌ باسم صاحب متجره. `ErrInvalidCredentials`
// إن لم يكن الرقمُ موظّفاً أو لم تطابق الكلمة.
func (s *Service) loginStaff(ctx context.Context, phone, password, userAgent, ip string) (*AuthResult, error) {
	staffID, ownerID, hash, err := s.repo.staffByPhone(ctx, phone)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	match, err := auth.VerifyPasswordTyped(password, hash)
	if err != nil {
		return nil, err
	}
	if !match {
		s.repo.Audit(ctx, nil, "auth.staff_password_failed", "merchant_staff", staffID, ip, nil)
		return nil, ErrInvalidCredentials
	}
	owner, _, err := s.repo.UserByID(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	return s.issueSession(withStaff(ctx, staffID), owner, userAgent, ip, "auth.staff_login", "")
}

// ── الإدارة ───────────────────────────────────────────────────────────

// ListStaff **موظّفو صاحب المتجر.**
func (s *Service) ListStaff(ctx context.Context, ownerID string) ([]Staff, error) {
	rows, err := s.repo.db.Query(ctx, `
		SELECT id::text, phone, name, created_at, last_login_at FROM merchant_staff
		 WHERE owner_user_id = $1::uuid ORDER BY created_at`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Staff{}
	for rows.Next() {
		var st Staff
		if err := rows.Scan(&st.ID, &st.Phone, &st.Name, &st.CreatedAt, &st.LastLoginAt); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// AddStaff **يضيف موظّفاً ويردّ كلمةَ سرّه** — مولَّدةً، تُرسَل له ولا تُخزَّن إلّا مشفّرة.
//
// **ورقمُ صاحبِ متجرٍ لا يصير موظّفاً**: دخولُه يذهب إلى حسابه هو، فلا يصل الموظّف.
func (s *Service) AddStaff(ctx context.Context, actorID, ownerID, rawPhone, name, ip string) (*Staff, string, error) {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return nil, "", ErrInvalidPhone
	}
	name = strings.TrimSpace(name)
	var ownsStore, isMerchant bool
	if err := s.repo.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM merchants WHERE owner_user_id = $1::uuid),
		       EXISTS (SELECT 1 FROM users u JOIN user_roles ur ON ur.user_id = u.id
		                WHERE u.phone = $2 AND ur.role_code = 'merchant')`,
		ownerID, phone).Scan(&ownsStore, &isMerchant); err != nil {
		return nil, "", err
	}
	if !ownsStore {
		return nil, "", ErrNotFound
	}
	if isMerchant {
		return nil, "", ErrStaffPhoneTaken
	}
	var n int
	if err := s.repo.db.QueryRow(ctx, `SELECT count(*) FROM merchant_staff WHERE owner_user_id = $1::uuid`,
		ownerID).Scan(&n); err != nil {
		return nil, "", err
	}
	if n >= MaxStaffPerStore {
		return nil, "", ErrStaffLimit
	}
	plain, err := GenerateTempPassword(int(s.intSetting(ctx, "security.password_min_length", minPasswordLn)))
	if err != nil {
		return nil, "", err
	}
	hash, err := auth.HashPassword(plain)
	if err != nil {
		return nil, "", err
	}
	var st Staff
	err = s.repo.db.QueryRow(ctx, `
		INSERT INTO merchant_staff (owner_user_id, phone, name, password_hash, created_by)
		VALUES ($1::uuid, $2, $3, $4, NULLIF($5, '')::uuid)
		ON CONFLICT (phone) DO NOTHING
		RETURNING id::text, phone, name, created_at, last_login_at`,
		ownerID, phone, name, hash, actorID).Scan(&st.ID, &st.Phone, &st.Name, &st.CreatedAt, &st.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrStaffPhoneTaken
	}
	if err != nil {
		return nil, "", err
	}
	s.repo.Audit(ctx, nilIfEmpty(actorID), "merchant.staff_added", "merchant_staff", st.ID, ip,
		map[string]any{"owner": ownerID, "phone": phone})
	return &st, plain, nil
}

// RemoveStaff **يحذف موظّفاً ويُخرجه فوراً من كلّ أجهزته.**
func (s *Service) RemoveStaff(ctx context.Context, actorID, ownerID, staffID, ip string) error {
	var exists bool
	if err := s.repo.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM merchant_staff
		WHERE id::text = $1 AND owner_user_id = $2::uuid)`, staffID, ownerID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	if err := s.endStaffSessions(ctx, staffID, "", ReasonStaffRemoved); err != nil {
		return err
	}
	if _, err := s.repo.db.Exec(ctx, `DELETE FROM merchant_staff WHERE id = $1::uuid`, staffID); err != nil {
		return err
	}
	s.repo.Audit(ctx, nilIfEmpty(actorID), "merchant.staff_removed", "merchant_staff", staffID, ip,
		map[string]any{"owner": ownerID})
	return nil
}

// endStaffSessions **يُنهي جلساتِ موظّفٍ** — في القاعدة وفي مرآة Redis، ويُبلغ أجهزتَه.
func (s *Service) endStaffSessions(ctx context.Context, staffID, client, reason string) error {
	sids, tokens, err := s.repo.revokeStaffSessions(ctx, staffID, client, reason)
	if err != nil {
		return err
	}
	for _, sid := range sids {
		s.rdb.Set(ctx, sessionRevokedKey(sid), reason, s.tokens.AccessTTL()+time.Minute)
		s.pub.Publish(SessionTopic(sid), map[string]any{"type": "session_revoked", "reason": reason})
	}
	if reason == ReasonSuperseded && len(tokens) > 0 && s.logoutNotifier != nil {
		s.logoutNotifier(ctx, tokens)
	}
	return nil
}

func nilIfEmpty(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
