// Package shamcash **سجلّ شام كاش اليدويّ** — قرار المالك ٢٠٢٦-١١-١٠.
//
// الزبون أحياناً بيحوّل على حساب شام كاش تبع المالك بدل ما يدفع للسائق نقداً، **والمالك بيعطي السائق
// المبلغ نقداً من جيبه**، والسائق بيسلّم المكتب المبلغ كامل متل العادة.
//
// # وهاد سجلّ جانبيّ وبس
//
// **ولا سطر هون بيمسّ الدفتر ولا المحافظ ولا صندوق السائق ولا درج المكتب ولا الخزينة.** الحزمة ما
// بتستورد أيّ حزمة مال، وجداولها (`shamcash_movements`، `shamcash_opening`) ما بيقراها أيّ فحص مال.
// والاختبار (`server/shamcash_test.go`) بيقارن أثر طلبين مسلَّمين، واحد بشام كاش وواحد بلا، صفّاً صفّاً.
//
// # الرصيد
//
//	رصيد البداية + تحويلات السائقين (غير «ما وصل») + إيداعات المالك − سحوباته
//
// لكلّ عملة لحالها (الليرة والدولار ما بينجمعوا)، والملغى ما بينحسب، **وما صار قبل تاريخ رصيد
// البداية داخل فيه** فما بينعدّ مرّتين.
package shamcash

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// أنواع الحركة.
const (
	KindDriverTransfer = "driver_transfer"
	KindDeposit        = "deposit"
	KindWithdrawal     = "withdrawal"
)

// حالات تحويل السائق.
const (
	StatusClaimed    = "claimed"
	StatusNotArrived = "not_arrived"
)

// العملتان.
const (
	CurrencySYP = "SYP"
	CurrencyUSD = "USD"
)

var (
	// ErrState **الحركة بحالة ما بتسمح بهالخطوة** — مثلاً «سلّمت السائق» على تحويل «ما وصل»، أو
	// إلغاء تحويل سائق (بدالو «ما وصل»).
	ErrState = httpx.NewError(http.StatusConflict, "shamcash_state", "errors.shamcash_state")
)

// ValidCurrency **ليرة أو دولار** وبس.
func ValidCurrency(c string) bool { return c == CurrencySYP || c == CurrencyUSD }

// Movement **حركة بالسجلّ** كما بتنعرض.
type Movement struct {
	ID            string     `json:"id"`
	Kind          string     `json:"kind"`
	OrderID       *string    `json:"order_id"`
	OrderNumber   *int64     `json:"order_number"`
	DriverID      *string    `json:"driver_id"`
	DriverName    *string    `json:"driver_name"`
	Currency      string     `json:"currency"`
	Amount        int64      `json:"amount"`
	OccurredAt    time.Time  `json:"occurred_at"`
	Note          string     `json:"note"`
	Status        string     `json:"status"`
	NotArrivedAt  *time.Time `json:"not_arrived_at"`
	NotArrivedBy  *string    `json:"not_arrived_by_name"`
	CompensatedAt *time.Time `json:"compensated_at"`
	CompensatedBy *string    `json:"compensated_by_name"`
	VoidedAt      *time.Time `json:"voided_at"`
	VoidedBy      *string    `json:"voided_by_name"`
	VoidReason    string     `json:"void_reason"`
	CreatedByName *string    `json:"created_by_name"`
	CreatedAt     time.Time  `json:"created_at"`
	// ExceedsDue **السائق كتب مبلغاً أكبر من المطلوب من الزبون** (قرار المالك ٢٠٢٦-١١-١٠) — مقبولة ومعلَّمة
	// واللوحة بتظلّلها. `DueAtEntry` المطلوب من الزبون وقتها (ليرة) للعرض.
	ExceedsDue      bool   `json:"exceeds_due"`
	DueAtEntry      *int64 `json:"due_at_entry"`
	BeforeOpening   bool   `json:"before_opening"`
	CountsInBalance bool   `json:"counts_in_balance"`
}

// RecordTransfer **السائق قال «انقبض شام كاش» عند التسليم** — جوّا معاملة الانتقال نفسها.
//
// **ومرّة وحدة لكل طلب**: إعادة النداء (الشبكة) ما بتكتب صفّاً تاني ولا بتبدّل الأول.
func RecordTransfer(ctx context.Context, q dbtx.Querier, orderID, driverID, currency string, amount int64) error {
	if !ValidCurrency(currency) || amount <= 0 {
		return errors.New("shamcash: bad transfer")
	}
	// **مبلغ أكبر من المطلوب من الزبون ⇒ مقبول ومعلَّم** (بالليرة وحدها: الدولار ما إله مطلوب يُقارَن فيه).
	_, err := q.Exec(ctx, `
		INSERT INTO shamcash_movements (kind, order_id, driver_id, currency, amount, created_by,
		                                exceeds_due, due_at_entry)
		SELECT 'driver_transfer', o.id, $2::uuid, $3::text, $4::bigint, $2::uuid,
		       ($3::text = 'SYP' AND $4::bigint > o.cash_due),
		       CASE WHEN $3::text = 'SYP' THEN o.cash_due END
		  FROM orders o WHERE o.id = $1::uuid
		ON CONFLICT (order_id) WHERE kind = 'driver_transfer' DO NOTHING`,
		orderID, driverID, currency, amount)
	return err
}

// Filter **مرشّحات القائمة** — والفارغ يعني «الكل».
type Filter struct {
	Kind        string // driver_transfer | deposit | withdrawal
	DriverID    string
	Status      string // claimed | not_arrived
	Compensated string // yes | no
	From, To    string // YYYY-MM-DD بيوم دمشق، والحدّان داخلين
	Voided      bool   // يعرض الملغى كمان
}

func (f Filter) where() (string, []any) {
	conds := []string{"TRUE"}
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "$?", "$"+strconv.Itoa(len(args))))
	}
	if f.Kind != "" {
		add("m.kind = $?", f.Kind)
	}
	if f.DriverID != "" {
		add("m.driver_id = $?::uuid", f.DriverID)
	}
	if f.Status != "" {
		add("m.status = $?", f.Status)
	}
	switch f.Compensated {
	case "yes":
		conds = append(conds, "m.compensated_at IS NOT NULL")
	case "no":
		conds = append(conds, "m.kind = 'driver_transfer' AND m.compensated_at IS NULL")
	}
	if f.From != "" {
		add("(m.occurred_at AT TIME ZONE 'Asia/Damascus')::date >= $?::date", f.From)
	}
	if f.To != "" {
		add("(m.occurred_at AT TIME ZONE 'Asia/Damascus')::date <= $?::date", f.To)
	}
	if !f.Voided {
		conds = append(conds, "m.voided_at IS NULL")
	}
	return strings.Join(conds, " AND "), args
}

const movementCols = `
	m.id::text, m.kind, m.order_id::text, o.number, m.driver_id::text, d.full_name,
	m.currency, m.amount, m.occurred_at, m.note, m.status,
	m.not_arrived_at, na.full_name, m.compensated_at, cp.full_name, m.voided_at, vd.full_name, m.void_reason,
	cb.full_name, m.created_at,
	(op.as_of IS NOT NULL AND m.occurred_at < op.as_of), m.exceeds_due, m.due_at_entry`

const movementFrom = `
	FROM shamcash_movements m
	LEFT JOIN orders o ON o.id = m.order_id
	LEFT JOIN users d ON d.id = m.driver_id
	LEFT JOIN users na ON na.id = m.not_arrived_by
	LEFT JOIN users cp ON cp.id = m.compensated_by
	LEFT JOIN users vd ON vd.id = m.voided_by
	LEFT JOIN users cb ON cb.id = m.created_by
	LEFT JOIN shamcash_opening op ON op.currency = m.currency`

func scanMovement(row pgx.Row) (Movement, error) {
	var m Movement
	err := row.Scan(&m.ID, &m.Kind, &m.OrderID, &m.OrderNumber, &m.DriverID, &m.DriverName,
		&m.Currency, &m.Amount, &m.OccurredAt, &m.Note, &m.Status,
		&m.NotArrivedAt, &m.NotArrivedBy, &m.CompensatedAt, &m.CompensatedBy, &m.VoidedAt, &m.VoidedBy,
		&m.VoidReason, &m.CreatedByName, &m.CreatedAt, &m.BeforeOpening, &m.ExceedsDue, &m.DueAtEntry)
	m.CountsInBalance = m.VoidedAt == nil && m.Status == StatusClaimed && !m.BeforeOpening
	return m, err
}

// List **الحركات الأحدث أولاً** بمرشّحاتها وصفحتها.
func List(ctx context.Context, q dbtx.Querier, f Filter, limit, offset int) ([]Movement, int, error) {
	where, args := f.where()
	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM shamcash_movements m WHERE `+where, args...).
		Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := q.Query(ctx, `SELECT `+movementCols+movementFrom+` WHERE `+where+`
		ORDER BY m.occurred_at DESC, m.created_at DESC
		LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Movement{}
	for rows.Next() {
		m, err := scanMovement(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, m)
	}
	return out, total, rows.Err()
}

// Get **حركة وحدة.**
func Get(ctx context.Context, q dbtx.Querier, id string) (Movement, error) {
	m, err := scanMovement(q.QueryRow(ctx, `SELECT `+movementCols+movementFrom+` WHERE m.id = $1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return m, httpx.ErrNotFound
	}
	return m, err
}

// Balance **رصيد عملة وحدة** وتفصيله.
type Balance struct {
	Currency    string     `json:"currency"`
	Opening     int64      `json:"opening"`
	OpeningAsOf *time.Time `json:"opening_as_of"`
	Transfers   int64      `json:"transfers"`
	Deposits    int64      `json:"deposits"`
	Withdrawals int64      `json:"withdrawals"`
	Balance     int64      `json:"balance"`
	// NotArrived **تحويلات قال السائق إنها صارت وما وصلت** — برّا الرصيد، للعلم.
	NotArrived int64 `json:"not_arrived"`
	// OwedToDrivers **«ما سلّمته للسائقين لسّا»** — تحويلات وصلت وما عطيت السائق بدالها.
	OwedToDrivers int64 `json:"owed_to_drivers"`
}

// DriverOwed **ما بذمّة المالك لسائق** بعملة.
type DriverOwed struct {
	DriverID   string `json:"driver_id"`
	DriverName string `json:"driver_name"`
	Currency   string `json:"currency"`
	Amount     int64  `json:"amount"`
	Count      int    `json:"count"`
}

// Summary **الأرصدة وما للسائقين.**
type Summary struct {
	Balances []Balance    `json:"balances"`
	Owed     []DriverOwed `json:"owed_by_driver"`
}

// owedCond **تحويل وصل (أو ما انقال «ما وصل») والمالك ما عطى السائق بدالو.** — بلا قيد التاريخ:
// دين السائق دين ولو قبل رصيد البداية.
const owedCond = `m.kind = 'driver_transfer' AND m.status = 'claimed' AND m.compensated_at IS NULL`

// GetSummary **الرصيد لكل عملة** = البداية + التحويلات (غير «ما وصل») + الإيداعات − السحوبات.
func GetSummary(ctx context.Context, q dbtx.Querier) (Summary, error) {
	out := Summary{Balances: []Balance{}, Owed: []DriverOwed{}}
	rows, err := q.Query(ctx, `
		SELECT c.cur, COALESCE(op.amount, 0), op.as_of,
		       COALESCE(sum(m.amount) FILTER (WHERE m.kind = 'driver_transfer' AND m.status = 'claimed' AND in_range), 0),
		       COALESCE(sum(m.amount) FILTER (WHERE m.kind = 'deposit' AND in_range), 0),
		       COALESCE(sum(m.amount) FILTER (WHERE m.kind = 'withdrawal' AND in_range), 0),
		       COALESCE(sum(m.amount) FILTER (WHERE m.kind = 'driver_transfer' AND m.status = 'not_arrived'), 0),
		       COALESCE(sum(m.amount) FILTER (WHERE `+owedCond+`), 0)
		FROM (VALUES ('SYP'), ('USD')) c(cur)
		LEFT JOIN shamcash_opening op ON op.currency = c.cur
		LEFT JOIN LATERAL (
		    SELECT m.*, (op.as_of IS NULL OR m.occurred_at >= op.as_of) AS in_range
		    FROM shamcash_movements m
		    WHERE m.currency = c.cur AND m.voided_at IS NULL) m ON TRUE
		GROUP BY c.cur, op.amount, op.as_of
		ORDER BY c.cur DESC`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var b Balance
		if err := rows.Scan(&b.Currency, &b.Opening, &b.OpeningAsOf, &b.Transfers, &b.Deposits,
			&b.Withdrawals, &b.NotArrived, &b.OwedToDrivers); err != nil {
			rows.Close()
			return out, err
		}
		b.Balance = b.Opening + b.Transfers + b.Deposits - b.Withdrawals
		out.Balances = append(out.Balances, b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	rows, err = q.Query(ctx, `
		SELECT m.driver_id::text, COALESCE(d.full_name, ''), m.currency, sum(m.amount), count(*)
		FROM shamcash_movements m LEFT JOIN users d ON d.id = m.driver_id
		WHERE `+owedCond+`
		GROUP BY m.driver_id, d.full_name, m.currency
		ORDER BY sum(m.amount) DESC`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var o DriverOwed
		if err := rows.Scan(&o.DriverID, &o.DriverName, &o.Currency, &o.Amount, &o.Count); err != nil {
			return out, err
		}
		out.Owed = append(out.Owed, o)
	}
	return out, rows.Err()
}

// SetOpening **رصيد البداية لعملة** بتاريخه.
func SetOpening(ctx context.Context, q dbtx.Querier, currency string, amount int64, asOf time.Time, actor string) error {
	_, err := q.Exec(ctx, `
		INSERT INTO shamcash_opening (currency, amount, as_of, updated_by, updated_at)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, now())
		ON CONFLICT (currency) DO UPDATE
		   SET amount = EXCLUDED.amount, as_of = EXCLUDED.as_of,
		       updated_by = EXCLUDED.updated_by, updated_at = now()`,
		currency, amount, asOf, actor)
	return err
}

// AddOwner **إيداع أو سحب من المالك.**
func AddOwner(ctx context.Context, q dbtx.Querier, kind, currency string, amount int64, at time.Time, note, actor string) (string, error) {
	if (kind != KindDeposit && kind != KindWithdrawal) || !ValidCurrency(currency) || amount <= 0 {
		return "", errors.New("shamcash: bad owner movement")
	}
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO shamcash_movements (kind, currency, amount, occurred_at, note, created_by)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::uuid) RETURNING id::text`,
		kind, currency, amount, at, note, actor).Scan(&id)
	return id, err
}

// lockKind **يقفل الحركة ويرجّع نوعها وحالتها.**
func lockKind(ctx context.Context, q dbtx.Querier, id string) (kind, status string, compensated, voided bool, err error) {
	err = q.QueryRow(ctx, `
		SELECT kind, status, compensated_at IS NOT NULL, voided_at IS NOT NULL
		FROM shamcash_movements WHERE id = $1::uuid FOR UPDATE`, id).
		Scan(&kind, &status, &compensated, &voided)
	if errors.Is(err, pgx.ErrNoRows) {
		err = httpx.ErrNotFound
	}
	return
}

// Void **إلغاء إيداع أو سحب انكتب غلط** — بسبب. **وتحويل السائق ما بينلغى**: بدالو «ما وصل».
func Void(ctx context.Context, q dbtx.Querier, id, reason, actor string) error {
	kind, _, _, voided, err := lockKind(ctx, q, id)
	if err != nil {
		return err
	}
	if kind == KindDriverTransfer || voided {
		return ErrState
	}
	_, err = q.Exec(ctx, `
		UPDATE shamcash_movements SET voided_at = now(), voided_by = NULLIF($2, '')::uuid, void_reason = $3
		WHERE id = $1::uuid`, id, actor, reason)
	return err
}

// SetCompensated **«سلّمت السائق بدالها»** (أو التراجع عنها) — لتحويل سائق وصل وبس.
func SetCompensated(ctx context.Context, q dbtx.Querier, id string, on bool, actor string) error {
	kind, status, compensated, _, err := lockKind(ctx, q, id)
	if err != nil {
		return err
	}
	if kind != KindDriverTransfer || compensated == on || (on && status != StatusClaimed) {
		return ErrState
	}
	if on {
		_, err = q.Exec(ctx, `UPDATE shamcash_movements SET compensated_at = now(), compensated_by = NULLIF($2, '')::uuid
			WHERE id = $1::uuid`, id, actor)
	} else {
		_, err = q.Exec(ctx, `UPDATE shamcash_movements SET compensated_at = NULL, compensated_by = NULL
			WHERE id = $1::uuid`, id)
	}
	return err
}

// SetNotArrived **«ما وصل»** (أو «وصل» للتراجع) — بيطلع التحويل من الرصيد. **وتحويل عطيت السائق
// بدالو ما بينقال عنه «ما وصل»** قبل ما يتراجع عن التسليم.
func SetNotArrived(ctx context.Context, q dbtx.Querier, id string, on bool, actor string) error {
	kind, status, compensated, _, err := lockKind(ctx, q, id)
	if err != nil {
		return err
	}
	if kind != KindDriverTransfer || (status == StatusNotArrived) == on || (on && compensated) {
		return ErrState
	}
	if on {
		_, err = q.Exec(ctx, `UPDATE shamcash_movements SET status = 'not_arrived', not_arrived_at = now(),
			not_arrived_by = NULLIF($2, '')::uuid WHERE id = $1::uuid`, id, actor)
	} else {
		_, err = q.Exec(ctx, `UPDATE shamcash_movements SET status = 'claimed', not_arrived_at = NULL,
			not_arrived_by = NULL WHERE id = $1::uuid`, id)
	}
	return err
}

// DriverRef **سائق إلو تحويلات** — لمرشّح القائمة.
type DriverRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Drivers **السائقين اللي إلهم تحويلات شام كاش.**
func Drivers(ctx context.Context, q dbtx.Querier) ([]DriverRef, error) {
	rows, err := q.Query(ctx, `
		SELECT DISTINCT m.driver_id::text, COALESCE(d.full_name, '')
		FROM shamcash_movements m JOIN users d ON d.id = m.driver_id
		WHERE m.kind = 'driver_transfer'
		ORDER BY 2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DriverRef{}
	for rows.Next() {
		var d DriverRef
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
