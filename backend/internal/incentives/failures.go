package incentives

// ══════════════════════════════════════════════════════════════════════
// **مكافأةُ هدفٍ تعثّرت — تُسجَّل وتُعاد** (قسمُ الأهداف، ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// **كان خطؤها يُبلَع** عند إنشاء متجرٍ من الإدارة: لا سطرَ في السجلّ،
// والمكافأةُ تتأخّر للمتجر الجاي — **ومن وقف عند الهدف بالضبط لا يقبضها
// أبداً.** والآن صفٌّ في `incentive_grant_failures` يظهر تنبيهاً في الصفحة،
// ويُعاد دوريّاً (`RetryFailures`) ومن زرٍّ (`RetryFailure`) — **لشهره.**

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// RecordFailure **يكتب العثرة ويقولها في السجلّ** — ولا يُسقط شيئاً.
func (s *Service) RecordFailure(ctx context.Context, userID, role, month string, cause error) {
	s.logf("مكافأةُ الهدف تعذّرت", "user", userID, "role", role, "month", month, "error", cause)
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	if len(msg) > 500 {
		msg = msg[:500]
	}
	if _, err := s.db.Exec(ctx, `
		INSERT INTO incentive_grant_failures (user_id, role, month, last_error)
		VALUES ($1::uuid, $2, $3, $4)
		ON CONFLICT (user_id, role, month) WHERE resolved_at IS NULL
		DO UPDATE SET attempts = incentive_grant_failures.attempts + 1,
		              last_error = excluded.last_error, updated_at = now()`,
		userID, role, month, msg); err != nil {
		s.logf("تعذّر تسجيلُ عثرة المكافأة", "user", userID, "error", err)
	}
}

// Paid **ما دُفع في إعادة** — ليُبشَّر صاحبُه.
type Paid struct {
	UserID string
	Role   string
	Amount int64
}

// RetryFailure **يعيد مكافأةً تعثّرت** — لشهرها، ويُغلق الصفَّ إن مرّت.
func (s *Service) RetryFailure(ctx context.Context, id string) (Paid, error) {
	var p Paid
	var month string
	err := s.db.QueryRow(ctx, `
		SELECT user_id::text, role, month FROM incentive_grant_failures
		 WHERE id = $1::uuid AND resolved_at IS NULL`, id).Scan(&p.UserID, &p.Role, &month)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, httpx.ErrNotFound
	}
	if err != nil {
		return p, err
	}
	paid, gerr := s.grantForMonth(ctx, p.UserID, p.Role, month)
	if gerr != nil {
		s.RecordFailure(ctx, p.UserID, p.Role, month, gerr)
		return p, gerr
	}
	p.Amount = paid
	_, err = s.db.Exec(ctx, `
		UPDATE incentive_grant_failures SET resolved_at = now(), updated_at = now()
		 WHERE id = $1::uuid`, id)
	return p, err
}

// RetryFailures **جولةٌ على كلّ ما تعثّر** — من الحلقة الدوريّة.
func (s *Service) RetryFailures(ctx context.Context) []Paid {
	rows, err := s.db.Query(ctx, `
		SELECT id::text FROM incentive_grant_failures
		 WHERE resolved_at IS NULL ORDER BY created_at LIMIT 100`)
	if err != nil {
		return nil
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	var out []Paid
	for _, id := range ids {
		if p, err := s.RetryFailure(ctx, id); err == nil && p.Amount > 0 {
			out = append(out, p)
		}
	}
	return out
}
