// Package cashbox الصندوق النقدي للسائقين — نقد المنصة بحوزتهم.
// دفتر قيود دائم بنمط المحفظة: التحصيل عند التسليم، والتسليم للمالية بتسوية.
package cashbox

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

var (
	ErrOverSettle  = httpx.NewError(http.StatusConflict, "over_settle", "errors.over_settle")
	ErrLimitExceed = httpx.NewError(http.StatusConflict, "cash_limit_exceeded", "errors.cash_limit_exceeded")
	// ErrCashOverdue **نقدٌ بيده لم يُسلَّم منذ المدّة والإيقافُ مُشعَل** — قرارُ المالك ٢٠٢٦-١٠-٠٤.
	//
	// **والرمزُ رمزُ السقف نفسُه** — تطبيقُ السائق مجمَّدٌ ويعرف هذا الرمزَ بنصٍّ عربيّ
	// («سلّم صندوقك أوّلاً»)، **والفعلُ المطلوب منه واحد**. ومفتاحُ الرسالة مختلف،
	// فاللوحةُ تقول السببَ الدقيق.
	ErrCashOverdue = httpx.NewError(http.StatusConflict, "cash_limit_exceeded", "errors.cash_overdue")
)

type Entry struct {
	ID     int64  `json:"id"`
	Amount int64  `json:"amount"`
	Kind   string `json:"kind"`
	Ref    string `json:"ref"`
	// OrderNumber **رقمُ الطلب الذي جاء منه المبلغ** — لقيود التحصيل وحدَها.
	OrderNumber *int64    `json:"order_number"`
	Note        string    `json:"note"`
	CreatedBy   *string   `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
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
	// **والاحتياطيُّ افتراضُ الفهرس لا رقمٌ مكتوبٌ هنا** (٢٠٢٦-١٠-٠٩) — كان ٥٠٠٬٠٠٠
	// بيدٍ، **ورقمان لمعنًى واحدٍ يفترقان** حين يتبدّل الافتراض.
	return s.settings.GetInt(ctx, "drivers.cash_limit")
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
		SELECT e.id, e.amount, e.kind, e.ref,
		       (SELECT o.number FROM orders o
		         WHERE e.kind = 'order_collection' AND o.id::text = e.ref),
		       e.note, e.created_by, e.created_at
		FROM driver_cash_entries e WHERE e.driver_id = $1
		ORDER BY e.id DESC LIMIT $2`, driverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Amount, &e.Kind, &e.Ref, &e.OrderNumber, &e.Note, &e.CreatedBy, &e.CreatedAt); err != nil {
			return nil, err
		}
		st.Entries = append(st.Entries, e)
	}
	return st, rows.Err()
}

// Querier ما يُنفَّذ عليه الاستعلام: المجمّع أو معاملة قائمة — كي يقيّد محرك
// الطلبات نقد التسليم **داخل** معاملة الانتقال نفسها لا بعدها.
type Querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// apply حركة ذرّية بمعاملة خاصة بها (نمط المحفظة نفسه).
func (s *Service) apply(ctx context.Context, driverID string, amount int64, kind, ref, note string, actorID *string) (int64, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	held, err := s.applyTx(ctx, tx, driverID, amount, kind, ref, note, actorID)
	if err != nil {
		return 0, err
	}
	return held, tx.Commit(ctx)
}

// applyTx نفس apply داخل معاملة يملكها المستدعي.
func (s *Service) applyTx(ctx context.Context, q Querier, driverID string, amount int64, kind, ref, note string, actorID *string) (int64, error) {
	if _, err := q.Exec(ctx, `
		INSERT INTO driver_cash_boxes (driver_id) VALUES ($1)
		ON CONFLICT (driver_id) DO NOTHING`, driverID); err != nil {
		return 0, err
	}
	var held int64
	err := q.QueryRow(ctx, `
		UPDATE driver_cash_boxes SET held = held + $2, updated_at = now()
		WHERE driver_id = $1 RETURNING held`, driverID, amount).Scan(&held)
	if isCheckViolation(err) {
		return 0, ErrOverSettle
	}
	if err != nil {
		return 0, err
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO driver_cash_entries (driver_id, amount, kind, ref, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)`, driverID, amount, kind, ref, note, actorID); err != nil {
		return 0, err
	}
	return held, nil
}

// Collect تحصيل نقد طلب مسلَّم (يستدعيه محرك الطلبات عند التسليم).
func (s *Service) Collect(ctx context.Context, driverID string, amount int64, orderID string, actorID *string) error {
	if amount <= 0 {
		return nil
	}
	_, err := s.apply(ctx, driverID, amount, "order_collection", orderID, "", actorID)
	return err
}

// CollectTx تحصيل نقد التسليم داخل معاملة الانتقال — فإن فشل الانتقال لم يبقَ قيد يتيم.
func (s *Service) CollectTx(ctx context.Context, q Querier, driverID string, amount int64, orderID string, actorID *string) error {
	if amount <= 0 {
		return nil
	}
	_, err := s.applyTx(ctx, q, driverID, amount, "order_collection", orderID, "", actorID)
	return err
}

// Settle تسليم الصندوق للمالية (كلياً أو جزئياً).
func (s *Service) Settle(ctx context.Context, driverID string, amount int64, note string, actorID string) (int64, error) {
	if amount <= 0 {
		return 0, ErrOverSettle
	}
	return s.apply(ctx, driverID, -amount, "settlement", "", note, &actorID)
}

// SettleTx كـ`Settle` **في معاملةٍ مُمرَّرة** — `XG-33`.
//
// **ولا معاملةَ داخل معاملة**: العملُ وعلامةُ تثبيت منع التكرار
// **يُثبَّتان معاً أو لا يُثبَّت أحدُهما.**
func (s *Service) SettleTx(ctx context.Context, q Querier, driverID string,
	amount int64, note, actorID string) (int64, error) {
	if amount <= 0 {
		return 0, ErrOverSettle
	}
	return s.applyTx(ctx, q, driverID, -amount, "settlement", "", note, &actorID)
}

func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}
