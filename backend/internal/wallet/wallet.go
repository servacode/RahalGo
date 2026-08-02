// Package wallet المحفظة الداخلية — دفتر قيود دائم (قرار 17):
// الحركات لا تُعدَّل ولا تُحذف، الرصيد محصلة مصانة داخل معاملة واحدة،
// ولا يقبل النظام رصيداً سالباً بأي حال.
package wallet

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
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
	// الافتتاحي والختامي للمدى المعروض. لا معنى لكشف حساب بدونهما: من يبدأ من
	// منتصف التاريخ يجمع الأسطر فلا تساوي رصيده فيظنّ الخلل في المنصة.
	// **مصانان ليتوازنا دائماً**: الافتتاحي + مجموع المعروض = الختامي.
	Opening int64 `json:"opening"`
	Closing int64 `json:"closing"`
	// صحيحة إن قُصّت النتيجة عند السقف — كشف حساب ناقص يجب أن يقول إنه ناقص
	Truncated bool `json:"truncated"`
}

// StatementRange مدى كشف الحساب. الحدّان اختياريان — يُترك أيّهما فارغاً فيُفتح.
type StatementRange struct {
	From  *time.Time
	To    *time.Time
	Limit int
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

// StatementFor يعيد الرصيد وآخر الحركات — عرض اللوحة السريع.
func (s *Service) StatementFor(ctx context.Context, userID string, limit int) (*Statement, error) {
	return s.Statement(ctx, userID, StatementRange{Limit: limit})
}

// maxStatementRows سقف صلب لكشف الحساب. ليس ترقيماً بل حاجزُ ذاكرة: كشفٌ
// بمئة ألف سطر لا يُقرأ ولا يُطبع، وإرساله يخنق الخادم والمتصفح معاً.
// وحين يُبلَغ السقف يُعلَن (`truncated`) لا يُصمت عنه.
const maxStatementRows = 2000

// Statement كشف الحساب: حركات المدى، ورصيداه الافتتاحي والختامي.
//
// الرصيدان **مشتقّان بالطرح من الرصيد الحالي** لا بجمع الدفتر من أوّله: الرصيد
// الحالي حقيقة مصانة في الجدول، وجمع تاريخ كامل يكلّف بلا فائدة. والاشتقاق
// مبنيٌّ ليتوازن حتى حين يُقصّ الكشف عند السقف — الافتتاحي يُحسب من **المعروض
// فعلاً** لا من المدى كله، فيصحّ الجمع دائماً بين يدَي من يراجعه.
func (s *Service) Statement(ctx context.Context, userID string, rng StatementRange) (*Statement, error) {
	limit := rng.Limit
	open := rng.From != nil || rng.To != nil // مدى صريح: المستخدم يطلب كشفاً لا لمحة
	switch {
	case limit > 0 && limit <= maxStatementRows:
	case open:
		limit = maxStatementRows
	default:
		limit = 50
	}

	st := &Statement{Transactions: []Transaction{}}
	var err error
	if st.Balance, err = s.Balance(ctx, userID); err != nil {
		return nil, err
	}

	// الختامي = رصيد نهاية المدى: الحالي ناقص كل ما وقع بعده. كشفُ تموز يجب أن
	// يُقفل برصيد تموز لا برصيد اليوم.
	st.Closing = st.Balance
	if rng.To != nil {
		var after int64
		if err := s.db.QueryRow(ctx, `
			SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
			WHERE user_id = $1 AND created_at >= $2`, userID, *rng.To).Scan(&after); err != nil {
			return nil, err
		}
		st.Closing -= after
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
		  AND ($2::timestamptz IS NULL OR t.created_at >= $2)
		  AND ($3::timestamptz IS NULL OR t.created_at < $3)
		ORDER BY t.id DESC LIMIT $4`, userID, rng.From, rng.To, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		// نطلب صفاً زائداً لنعرف أن هناك المزيد — بلا استعلام عدٍّ ثانٍ
		if len(st.Transactions) == limit {
			st.Truncated = true
			break
		}
		var t Transaction
		if err := rows.Scan(&t.ID, &t.Amount, &t.Kind, &t.Ref, &t.Note, &t.CreatedBy,
			&t.ByName, &t.OrderNumber, &t.TicketNumber, &t.CreatedAt); err != nil {
			return nil, err
		}
		st.Transactions = append(st.Transactions, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// الافتتاحي من **المعروض** لا من المدى: لو قُصّ الكشف بقي الجمع صحيحاً.
	var shown int64
	for _, t := range st.Transactions {
		shown += t.Amount
	}
	st.Opening = st.Closing - shown
	return st, nil
}

// Querier ما يُنفَّذ عليه الاستعلام: المجمّع أو معاملة قائمة. يسمح لمن يملك
// معاملة (محرك الطلبات) بأن يُدخل حركة المحفظة **داخلها** فتُلغى معه إن فشل،
// بدل أن تنجح وحدها ويبقى المال معلّقاً بلا طلب.
type Querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	// Query لقراءةِ صفوفٍ متعدّدة داخل المعاملة نفسِها.
	//
	// **أُضيفت حين صار الطلبُ من مصدرين**: التسويةُ تقرأ مستحقَّ كلِّ مصدرٍ على
	// حدة، **وقراءةٌ خارج المعاملة تقرأ ما قبلها لا ما فيها** — فتُقسَّم
	// مستحقّاتٌ على بنودٍ لم تُكتب بعد.
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// Apply ينفّذ حركة (موجبة أو سالبة) ذرّياً بمعاملة خاصة بها.
// خصمٌ يتجاوز الرصيد يُرفض بـ insufficient_balance (قيد CHECK في القاعدة).
func (s *Service) Apply(ctx context.Context, userID string, amount int64, kind, ref, note string, actorID *string) (int64, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	balance, err := s.ApplyTx(ctx, tx, userID, amount, kind, ref, note, actorID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return balance, nil
}

// ApplyTx نفس Apply لكن داخل معاملة يملكها المستدعي — لا يفتح معاملة ولا يُنهيها.
func (s *Service) ApplyTx(ctx context.Context, q Querier, userID string, amount int64, kind, ref, note string, actorID *string) (int64, error) {
	if amount == 0 {
		return 0, ErrInvalidAmount
	}

	// ضمان وجود المحفظة أولاً ثم التحديث — لأن CHECK يُفحص على صف الإدراج
	// المقترح قبل اكتشاف التعارض، فإدراج مبلغ سالب مباشرة يفشل خطأً.
	if _, err := q.Exec(ctx, `
		INSERT INTO wallets (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return 0, err
	}

	var balance int64
	err := q.QueryRow(ctx, `
		UPDATE wallets SET balance = balance + $2, updated_at = now()
		WHERE user_id = $1
		RETURNING balance`, userID, amount).Scan(&balance)
	if isCheckViolation(err) {
		return 0, ErrInsufficient
	}
	if err != nil {
		return 0, err
	}

	if _, err := q.Exec(ctx, `
		INSERT INTO wallet_transactions (user_id, amount, kind, ref, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, amount, kind, ref, note, actorID); err != nil {
		return 0, err
	}
	return balance, nil
}

func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}
