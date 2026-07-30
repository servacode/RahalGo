// Package cashbox الصندوق النقدي للسائقين — نقد المنصة بحوزتهم.
// دفتر قيود دائم بنمط المحفظة: التحصيل عند التسليم، والتسليم للمالية بتسوية.
package cashbox

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

var (
	ErrOverSettle  = httpx.NewError(http.StatusConflict, "over_settle", "errors.over_settle")
	ErrLimitExceed = httpx.NewError(http.StatusConflict, "cash_limit_exceeded", "errors.cash_limit_exceeded")
)

type Entry struct {
	ID        int64     `json:"id"`
	Amount    int64     `json:"amount"`
	Kind      string    `json:"kind"`
	Ref       string    `json:"ref"`
	Note      string    `json:"note"`
	CreatedBy *string   `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type Statement struct {
	Held    int64   `json:"held"`
	Limit   int64   `json:"limit"`
	Entries []Entry `json:"entries"`
}

type Service struct {
	db       *pgxpool.Pool
	settings *settings.Store
}

func NewService(db *pgxpool.Pool, settingsStore *settings.Store) *Service {
	return &Service{db: db, settings: settingsStore}
}

// Limit السقف النقدي الحالي (إعداد ديناميكي).
func (s *Service) Limit(ctx context.Context) int64 {
	var v float64
	if err := s.settings.Get(ctx, "drivers.cash_limit", &v); err != nil || v <= 0 {
		return 500000
	}
	return int64(v)
}

func (s *Service) Held(ctx context.Context, driverID string) (int64, error) {
	var held int64
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE((SELECT held FROM driver_cash_boxes WHERE driver_id = $1), 0)`,
		driverID).Scan(&held)
	return held, err
}

// OverLimit هل تجاوز السائق سقفه النقدي؟ (يمنع إسناد طلبات نقدية جديدة)
func (s *Service) OverLimit(ctx context.Context, driverID string) (bool, error) {
	held, err := s.Held(ctx, driverID)
	if err != nil {
		return false, err
	}
	return held >= s.Limit(ctx), nil
}

func (s *Service) StatementFor(ctx context.Context, driverID string, limit int) (*Statement, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	st := &Statement{Entries: []Entry{}, Limit: s.Limit(ctx)}
	var err error
	if st.Held, err = s.Held(ctx, driverID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, amount, kind, ref, note, created_by, created_at
		FROM driver_cash_entries WHERE driver_id = $1
		ORDER BY id DESC LIMIT $2`, driverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Amount, &e.Kind, &e.Ref, &e.Note, &e.CreatedBy, &e.CreatedAt); err != nil {
			return nil, err
		}
		st.Entries = append(st.Entries, e)
	}
	return st, rows.Err()
}

// apply حركة ذرّية: قيد + تحديث الرصيد (نمط المحفظة نفسه).
func (s *Service) apply(ctx context.Context, driverID string, amount int64, kind, ref, note string, actorID *string) (int64, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO driver_cash_boxes (driver_id) VALUES ($1)
		ON CONFLICT (driver_id) DO NOTHING`, driverID); err != nil {
		return 0, err
	}
	var held int64
	err = tx.QueryRow(ctx, `
		UPDATE driver_cash_boxes SET held = held + $2, updated_at = now()
		WHERE driver_id = $1 RETURNING held`, driverID, amount).Scan(&held)
	if isCheckViolation(err) {
		return 0, ErrOverSettle
	}
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO driver_cash_entries (driver_id, amount, kind, ref, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)`, driverID, amount, kind, ref, note, actorID); err != nil {
		return 0, err
	}
	return held, tx.Commit(ctx)
}

// Collect تحصيل نقد طلب مسلَّم (يستدعيه محرك الطلبات عند التسليم).
func (s *Service) Collect(ctx context.Context, driverID string, amount int64, orderID string, actorID *string) error {
	if amount <= 0 {
		return nil
	}
	_, err := s.apply(ctx, driverID, amount, "order_collection", orderID, "تحصيل طلب", actorID)
	return err
}

// Settle تسليم الصندوق للمالية (كلياً أو جزئياً).
func (s *Service) Settle(ctx context.Context, driverID string, amount int64, note string, actorID string) (int64, error) {
	if amount <= 0 {
		return 0, ErrOverSettle
	}
	return s.apply(ctx, driverID, -amount, "settlement", "", note, &actorID)
}

func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}
