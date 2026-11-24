package prizes

// ══════════════════════════════════════════════════════════════════════
// **دفترُ رصيد الجوائز** — منحٌ وصرفٌ وردٌّ وانتهاء
// ══════════════════════════════════════════════════════════════════════
//
// **كالمحفظة**: صفُّ رصيدٍ مقفولٌ بالتحديث (`FOR UPDATE`) وقيدٌ لكلّ حركةٍ لا يُعدَّل،
// **والرصيدُ لا ينزل تحت الصفر** (قيدُ `CHECK`). **ومجموعُ القيود = الرصيد دائماً** —
// فالانتهاءُ قيدٌ سالبٌ لا تصفيرٌ صامت، والردُّ بعد الانتهاء ردٌّ ثمّ انتهاءٌ في اللحظة.

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ErrInsufficient **صرفٌ يتجاوز الرصيد.**
var ErrInsufficient = httpx.NewError(http.StatusConflict, "prize_balance_insufficient",
	"errors.prize_balance_insufficient")

// Balance **رصيدٌ نافذ** — والمنتهي صفر.
type Balance struct {
	Amount    int64      `json:"amount"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// BalanceOf **رصيدُ الزبون الآن** — بلا قفل.
func (s *Service) BalanceOf(ctx context.Context, q dbtx.Querier, userID string) (Balance, error) {
	var b Balance
	err := q.QueryRow(ctx, `
		SELECT balance, expires_at FROM prize_balances WHERE user_id = $1`, userID).
		Scan(&b.Amount, &b.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Balance{}, nil
	}
	if err != nil {
		return Balance{}, err
	}
	if b.ExpiresAt == nil || !s.now().Before(*b.ExpiresAt) || b.Amount <= 0 {
		return Balance{}, nil
	}
	return b, nil
}

// lockRow **يقفل صفَّ الرصيد** وينشئه إن غاب، **ويُنهي رصيداً انتهى قبل أيّ حركة.**
func (s *Service) lockRow(ctx context.Context, q dbtx.Querier, userID string) (Balance, error) {
	if _, err := q.Exec(ctx, `
		INSERT INTO prize_balances (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return Balance{}, err
	}
	var b Balance
	if err := q.QueryRow(ctx, `
		SELECT balance, expires_at FROM prize_balances WHERE user_id = $1 FOR UPDATE`, userID).
		Scan(&b.Amount, &b.ExpiresAt); err != nil {
		return Balance{}, err
	}
	if b.Amount > 0 && (b.ExpiresAt == nil || !s.now().Before(*b.ExpiresAt)) {
		if err := expireLocked(ctx, q, userID, b); err != nil {
			return Balance{}, err
		}
		b.Amount = 0
	}
	return b, nil
}

// expireLocked **قيدُ انتهاءٍ بالرصيد كلِّه** — على صفٍّ مقفول.
func expireLocked(ctx context.Context, q dbtx.Querier, userID string, b Balance) error {
	if b.Amount <= 0 {
		return nil
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO prize_entries (user_id, kind, amount, expires_at)
		VALUES ($1, 'expire', $2, $3)`, userID, -b.Amount, b.ExpiresAt); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `
		UPDATE prize_balances SET balance = 0, updated_at = now() WHERE user_id = $1`, userID)
	return err
}

// LockedBalanceTx **الرصيدُ النافذُ مقفولاً** — يُقرأ قبل حساب الطلب، فطلبان متزامنان
// لا يصرفان الرصيدَ نفسَه: الثاني ينتظر القفلَ ثمّ يقرأ ما بقي.
func (s *Service) LockedBalanceTx(ctx context.Context, q dbtx.Querier, userID string) (int64, error) {
	b, err := s.lockRow(ctx, q, userID)
	if err != nil {
		return 0, err
	}
	return b.Amount, nil
}

// AwardTx **يمنح جائزةً** — تنتهي آخرَ الشهر الذي يلي المنح (`AwardExpiry`).
func (s *Service) AwardTx(ctx context.Context, q dbtx.Querier, userID string, amount int64, drawMonth time.Time) error {
	if amount <= 0 {
		return nil
	}
	if _, err := s.lockRow(ctx, q, userID); err != nil {
		return err
	}
	exp := AwardExpiry(s.now())
	if _, err := q.Exec(ctx, `
		UPDATE prize_balances SET balance = balance + $2, expires_at = $3, updated_at = now()
		 WHERE user_id = $1`, userID, amount, exp); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `
		INSERT INTO prize_entries (user_id, kind, amount, draw_month, expires_at)
		VALUES ($1, 'award', $2, $3::date, $4)`, userID, amount, drawMonth.Format("2006-01-02"), exp)
	return err
}

// SpendTx **يصرف من الرصيد على طلب** — داخل معاملة الطلب. يُرفض ما يتجاوز النافذ.
func (s *Service) SpendTx(ctx context.Context, q dbtx.Querier, userID, orderID string, amount int64) error {
	if amount <= 0 {
		return nil
	}
	b, err := s.lockRow(ctx, q, userID)
	if err != nil {
		return err
	}
	if b.Amount < amount {
		return ErrInsufficient
	}
	if _, err := q.Exec(ctx, `
		UPDATE prize_balances SET balance = balance - $2, updated_at = now() WHERE user_id = $1`,
		userID, amount); err != nil {
		return err
	}
	_, err = q.Exec(ctx, `
		INSERT INTO prize_entries (user_id, kind, amount, order_id, expires_at)
		VALUES ($1, 'spend', $2, $3, $4)`, userID, -amount, orderID, b.ExpiresAt)
	return err
}

// RefundOrderTx **يردّ ما صُرف على طلبٍ فوق `keep`** — ويردّ المردود.
//
// **الصافي من الدفتر** (الصرفُ ناقصَ ما رُدّ) — **فالنداءُ مرّتين لا يردّ مرّتين.** والردُّ
// إلى رصيدٍ لم ينتهِ بعد يعود إليه بانتهائه نفسِه، **وبعد الانتهاء يُكتب ردّاً ثمّ انتهاءً**
// فلا يُحيي رصيداً انتهى ويبقى الدفترُ يجمع إلى الرصيد.
func (s *Service) RefundOrderTx(ctx context.Context, q dbtx.Querier, orderID string, keep int64) (int64, error) {
	var userID *string
	var net int64
	var exp *time.Time
	if err := q.QueryRow(ctx, `
		SELECT max(user_id::text), -COALESCE(sum(amount), 0),
		       max(expires_at) FILTER (WHERE kind = 'spend')
		  FROM prize_entries
		 WHERE order_id = $1 AND kind IN ('spend', 'refund')`, orderID).
		Scan(&userID, &net, &exp); err != nil {
		return 0, err
	}
	if userID == nil || net <= keep {
		return 0, nil
	}
	if keep < 0 {
		keep = 0
	}
	back := net - keep
	b, err := s.lockRow(ctx, q, *userID)
	if err != nil {
		return 0, err
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO prize_entries (user_id, kind, amount, order_id, expires_at)
		VALUES ($1, 'refund', $2, $3, $4)`, *userID, back, orderID, exp); err != nil {
		return 0, err
	}
	if exp == nil || !s.now().Before(*exp) {
		// **انتهى شهرُه** — يُكتب انتهاؤه في اللحظة ولا يمسّ الرصيدَ القائم.
		_, err := q.Exec(ctx, `
			INSERT INTO prize_entries (user_id, kind, amount, order_id, expires_at)
			VALUES ($1, 'expire', $2, $3, $4)`, *userID, -back, orderID, exp)
		return 0, err
	}
	// **وشهرُه ما زال** — يعود إلى الرصيد بانتهائه نفسِه.
	newExp := exp
	if b.ExpiresAt != nil && b.Amount > 0 && b.ExpiresAt.After(*exp) {
		newExp = b.ExpiresAt
	}
	if _, err := q.Exec(ctx, `
		UPDATE prize_balances SET balance = balance + $2, expires_at = $3, updated_at = now()
		 WHERE user_id = $1`, *userID, back, newExp); err != nil {
		return 0, err
	}
	return back, nil
}

// ExpireDue **يُنهي كلَّ رصيدٍ مضى شهرُه** — ويردّ كم أُنهي.
func (s *Service) ExpireDue(ctx context.Context) (int, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		SELECT user_id::text, balance, expires_at FROM prize_balances
		 WHERE balance > 0 AND (expires_at IS NULL OR expires_at <= $1)
		 FOR UPDATE SKIP LOCKED`, s.now())
	if err != nil {
		return 0, err
	}
	type due struct {
		user string
		b    Balance
	}
	var all []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.user, &d.b.Amount, &d.b.ExpiresAt); err != nil {
			rows.Close()
			return 0, err
		}
		all = append(all, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, d := range all {
		if err := expireLocked(ctx, tx, d.user, d.b); err != nil {
			return 0, err
		}
	}
	return len(all), tx.Commit(ctx)
}
