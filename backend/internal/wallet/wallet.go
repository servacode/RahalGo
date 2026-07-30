// Package wallet المحفظة الداخلية — دفتر قيود دائم (قرار 17):
// الحركات لا تُعدَّل ولا تُحذف، الرصيد محصلة مصانة داخل معاملة واحدة،
// ولا يقبل النظام رصيداً سالباً بأي حال.
package wallet

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	ErrInsufficient  = httpx.NewError(http.StatusConflict, "insufficient_balance", "errors.insufficient_balance")
	ErrInvalidAmount = httpx.NewError(http.StatusBadRequest, "invalid_amount", "errors.invalid_amount")
)

type Transaction struct {
	ID           int64     `json:"id"`
	Amount       int64     `json:"amount"`
	Kind         string    `json:"kind"`
	Ref          string    `json:"ref"`
	Note         string    `json:"note"`
	CreatedBy    *string   `json:"created_by"`
	ByName       *string   `json:"by_name"`       // منفّذ الحركة (اسم أو هاتف)
	OrderNumber  *int64    `json:"order_number"`  // إن كان المرجع طلباً
	TicketNumber *int64    `json:"ticket_number"` // إن كان المرجع تذكرة
	CreatedAt    time.Time `json:"created_at"`
}

type Statement struct {
	Balance      int64         `json:"balance"`
	Transactions []Transaction `json:"transactions"`
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service { return &Service{db: db} }

// Balance يعيد الرصيد الحالي (صفر لمن لا محفظة له بعد).
func (s *Service) Balance(ctx context.Context, userID string) (int64, error) {
	var balance int64
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE((SELECT balance FROM wallets WHERE user_id = $1), 0)`, userID).
		Scan(&balance)
	return balance, err
}

// StatementFor يعيد الرصيد وآخر الحركات.
func (s *Service) StatementFor(ctx context.Context, userID string, limit int) (*Statement, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	st := &Statement{Transactions: []Transaction{}}
	var err error
	if st.Balance, err = s.Balance(ctx, userID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT t.id, t.amount, t.kind, t.ref, t.note, t.created_by,
		       NULLIF(COALESCE(cb.full_name, cb.phone::text), ''),
		       o.number, tk.number, t.created_at
		FROM wallet_transactions t
		LEFT JOIN users cb ON cb.id = t.created_by
		LEFT JOIN orders o ON t.ref <> '' AND o.id::text = t.ref
		LEFT JOIN tickets tk ON t.ref <> '' AND tk.id::text = t.ref
		WHERE t.user_id = $1
		ORDER BY t.id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.Amount, &t.Kind, &t.Ref, &t.Note, &t.CreatedBy,
			&t.ByName, &t.OrderNumber, &t.TicketNumber, &t.CreatedAt); err != nil {
			return nil, err
		}
		st.Transactions = append(st.Transactions, t)
	}
	return st, rows.Err()
}

// Apply ينفّذ حركة (موجبة أو سالبة) ذرّياً: قيد في الدفتر + تحديث الرصيد معاً.
// خصمٌ يتجاوز الرصيد يُرفض بـ insufficient_balance (قيد CHECK في القاعدة).
func (s *Service) Apply(ctx context.Context, userID string, amount int64, kind, ref, note string, actorID *string) (int64, error) {
	if amount == 0 {
		return 0, ErrInvalidAmount
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// ضمان وجود المحفظة أولاً ثم التحديث — لأن CHECK يُفحص على صف الإدراج
	// المقترح قبل اكتشاف التعارض، فإدراج مبلغ سالب مباشرة يفشل خطأً.
	if _, err := tx.Exec(ctx, `
		INSERT INTO wallets (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return 0, err
	}

	var balance int64
	err = tx.QueryRow(ctx, `
		UPDATE wallets SET balance = balance + $2, updated_at = now()
		WHERE user_id = $1
		RETURNING balance`, userID, amount).Scan(&balance)
	if isCheckViolation(err) {
		return 0, ErrInsufficient
	}
	if err != nil {
		return 0, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO wallet_transactions (user_id, amount, kind, ref, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, amount, kind, ref, note, actorID); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return balance, nil
}

func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}
