package orders

// ══════════════════════════════════════════════════════════════════════
// **«ديون الطلبات»** — قرارُ المالك ٢٠٢٦-١١-٢٠
// ══════════════════════════════════════════════════════════════════════
//
// الزبونُ يقدر يدفع **الطلبَ كلَّه من المحفظة**. وبالنموذج ٢ الكابتنُ دفع للمتجر من جيبه عند الاستلام،
// **والزبونُ ما دفعله شي** — فما دفعه وما قبضه دينٌ عالمنصّة للكابتن:
//
//	الدين      = ما دفعه للمتجر − ما قبضه من الزبون عن البضاعة (نقداً أو بمعادل الدولار) — إن كان موجباً
//	المحفظة    ما بينزل فيها شي — **المبلغُ يبقى بالخزينة** والخزينةُ لا تعدّه ربحاً (`creditTreasuryGoodsOutside`)
//	المكتب     يحاسبه باليد ويكبس «تم الدفع» ⇒ الصفُّ `paid` + سطرٌ خارجٌ بصندوق المكتب (`order_debt_paid`)
//	التصحيح    طلبٌ سُجّل تسليمُه خطأً: المفتوحُ يسقط (`void`)، والمدفوعُ يصير ديناً على الكابتن
//	           (يستردّ ما دفعه من المتجر — القرار ٤ — فلا يأخذه مرّتين)
//
// **والنموذج ١ لا يُمسّ** — سلفةُ الدرج فيه تردّ للكابتن ما دفعه (`recordAdvanceReturn`).

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/officecash"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// ErrNoOpenOrderDebts **ما في ديون مفتوحة تنعطى** — انعطت قبل أو ما في شي.
var ErrNoOpenOrderDebts = httpx.NewError(http.StatusConflict,
	"no_open_order_debts", "errors.no_open_order_debts")

// recordOrderDriverDebt **يكتب دينَ الطلب للكابتن** إن دفع أكثر ممّا قبض — النموذج ٢ وحدَه.
// **ومُعادُ النداء لا يكرّر** (`(order_id, driver_id)` فريد).
func recordOrderDriverDebt(ctx context.Context, q wallet.Querier, orderID string, driverID *string,
	paid, collected int64, moneyModel int16) error {
	if driverID == nil || moneyModel != MoneyModelGoodsOutside {
		return nil
	}
	amount := paid - collected
	if amount <= 0 {
		return nil
	}
	_, err := q.Exec(ctx, `
		INSERT INTO order_driver_debts (order_id, driver_id, amount)
		VALUES ($1, $2, $3)
		ON CONFLICT (order_id, driver_id) DO NOTHING`, orderID, *driverID, amount)
	return err
}

// voidOrderDriverDebts **طلبٌ سُجّل تسليمُه خطأً** — المفتوحُ يسقط، والمدفوعُ دينٌ على الكابتن.
func voidOrderDriverDebts(ctx context.Context, q wallet.Querier, orderID, actorID string) error {
	rows, err := q.Query(ctx, `
		UPDATE order_driver_debts
		   SET state = CASE WHEN state = 'open' THEN 'void' ELSE state END,
		       voided_at = CASE WHEN state = 'open' THEN now() ELSE voided_at END
		 WHERE order_id = $1 AND state IN ('open', 'paid')
		RETURNING id::text, driver_id::text, amount, state`, orderID)
	if err != nil {
		return err
	}
	type paidRow struct {
		id, driver string
		amount     int64
	}
	var paid []paidRow
	for rows.Next() {
		var r paidRow
		var state string
		if err := rows.Scan(&r.id, &r.driver, &r.amount, &state); err != nil {
			rows.Close()
			return err
		}
		if state == "paid" {
			paid = append(paid, r)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range paid {
		if _, err := obligations.CreateRef(ctx, q, obligations.PartyDriver, r.driver, r.amount,
			obligations.CauseUndeliveredOrder, orderID, "order_debt:"+r.id, &actorID); err != nil {
			return err
		}
	}
	return nil
}

// ── سطحُ اللوحة ─────────────────────────────────────────────────────

// OrderDebtDriver **سطرُ كابتنٍ في «ديون الطلبات»** — مجموعُ ما إله عالمكتب.
type OrderDebtDriver struct {
	DriverID   string     `json:"driver_id"`
	DriverName string     `json:"driver_name"`
	Phone      string     `json:"phone"`
	OpenTotal  int64      `json:"open_total"`
	OpenCount  int        `json:"open_count"`
	PaidTotal  int64      `json:"paid_total"`
	Oldest     *time.Time `json:"oldest_open_at,omitempty"`
}

// OrderDebtRow **طلبٌ واحدٌ** — رقمُه ومبلغُه وحالُه.
type OrderDebtRow struct {
	ID          string     `json:"id"`
	OrderID     string     `json:"order_id"`
	OrderNumber int64      `json:"order_number"`
	Amount      int64      `json:"amount"`
	State       string     `json:"state"`
	CreatedAt   time.Time  `json:"created_at"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	PaidByName  *string    `json:"paid_by_name,omitempty"`
}

// OrderDebtsSummary **كلُّ كابتنٍ إله شي** — المفتوحُ أوّلاً بالأكبر.
func (s *Service) OrderDebtsSummary(ctx context.Context) ([]OrderDebtDriver, error) {
	rows, err := s.db.Query(ctx, `
		SELECT d.driver_id::text, COALESCE(u.full_name, ''), COALESCE(u.phone, ''),
		       COALESCE(sum(d.amount) FILTER (WHERE d.state = 'open'), 0)::bigint,
		       count(*) FILTER (WHERE d.state = 'open')::int,
		       COALESCE(sum(d.amount) FILTER (WHERE d.state = 'paid'), 0)::bigint,
		       min(d.created_at) FILTER (WHERE d.state = 'open')
		  FROM order_driver_debts d
		  JOIN users u ON u.id = d.driver_id
		 WHERE d.state <> 'void'
		 GROUP BY d.driver_id, u.full_name, u.phone
		 ORDER BY 4 DESC, 2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OrderDebtDriver{}
	for rows.Next() {
		var r OrderDebtDriver
		if err := rows.Scan(&r.DriverID, &r.DriverName, &r.Phone, &r.OpenTotal, &r.OpenCount,
			&r.PaidTotal, &r.Oldest); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// OrderDebtsOf **طلباتُ كابتنٍ واحد** — المفتوحُ أوّلاً ثمّ الأحدث.
func (s *Service) OrderDebtsOf(ctx context.Context, driverID string, limit int) ([]OrderDebtRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.Query(ctx, `
		SELECT d.id::text, d.order_id::text, COALESCE(o.number, 0), d.amount, d.state, d.created_at,
		       d.paid_at, p.full_name
		  FROM order_driver_debts d
		  JOIN orders o ON o.id = d.order_id
		  LEFT JOIN users p ON p.id = d.paid_by
		 WHERE d.driver_id = $1::uuid AND d.state <> 'void'
		 ORDER BY (d.state = 'open') DESC, d.created_at DESC
		 LIMIT $2`, driverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OrderDebtRow{}
	for rows.Next() {
		var r OrderDebtRow
		if err := rows.Scan(&r.ID, &r.OrderID, &r.OrderNumber, &r.Amount, &r.State, &r.CreatedAt,
			&r.PaidAt, &r.PaidByName); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// OrderDebtsOpenTotal **مجموعُ ما للكابتن عالمكتب** — لتطبيقه.
func (s *Service) OrderDebtsOpenTotal(ctx context.Context, driverID string) (int64, int, error) {
	var total int64
	var n int
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0)::bigint, count(*)::int
		  FROM order_driver_debts WHERE driver_id = $1::uuid AND state = 'open'`, driverID).Scan(&total, &n)
	return total, n, err
}

// PayOrderDebtsTx **«تم الدفع»** — يشطب ديونَ كابتنٍ (المحدَّدةَ أو كلَّ المفتوح) ويكتب لكلّ واحدٍ سطراً
// خارجاً بصندوق المكتب بمرجعه. يُرجع المجموعَ المدفوع وعددَه — داخل معاملة المستدعي (مع سجلّ التدقيق).
func (s *Service) PayOrderDebtsTx(ctx context.Context, q dbtx.Querier, driverID string, ids []string,
	actorID string) (int64, int, error) {
	var idFilter any
	if len(ids) > 0 {
		idFilter = ids
	}
	rows, err := q.Query(ctx, `
		UPDATE order_driver_debts d
		   SET state = 'paid', paid_at = now(), paid_by = $3::uuid
		 WHERE d.driver_id = $1::uuid AND d.state = 'open'
		   AND ($2::uuid[] IS NULL OR d.id = ANY($2::uuid[]))
		RETURNING d.id::text, d.amount, (SELECT number FROM orders o WHERE o.id = d.order_id)`,
		driverID, idFilter, actorID)
	if err != nil {
		return 0, 0, err
	}
	type paidRow struct {
		id     string
		amount int64
		number *int64
	}
	var paid []paidRow
	for rows.Next() {
		var r paidRow
		if err := rows.Scan(&r.id, &r.amount, &r.number); err != nil {
			rows.Close()
			return 0, 0, err
		}
		paid = append(paid, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	if len(paid) == 0 || (len(ids) > 0 && len(paid) != len(ids)) {
		return 0, 0, ErrNoOpenOrderDebts
	}
	var total int64
	for _, r := range paid {
		note := "دين طلب للكابتن — الزبون دفع كل الطلب من المحفظة"
		if r.number != nil {
			note = "دين طلب #" + strconv.FormatInt(*r.number, 10) + " للكابتن — الزبون دفع كل الطلب من المحفظة"
		}
		if err := officecash.Record(ctx, q, officecash.Entry{
			Direction: officecash.Out, Amount: r.amount, Source: officecash.SourceOrderDebtPaid,
			Ref: r.id, UserID: driverID, Actor: actorID, Note: note,
		}); err != nil {
			return 0, 0, err
		}
		total += r.amount
	}
	return total, len(paid), nil
}
