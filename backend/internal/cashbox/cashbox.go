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
	// **ورمزُه رمزُه لا رمزُ السقف** (قرارُ المالك ٢٠٢٦-١٠-١٥): كان `cash_limit_exceeded`
	// فيقرأ السائقُ «بلغت سقف النقد» **وهو تحت سقفه** — فلا يفهم لماذا رُدّ. **والسببان
	// مختلفان والفعلُ واحد** (سلّم ما بذمّتك)، **فالرسالةُ تقول السببَ الحقيقيّ**:
	// «تأخّرت بتسليم المصاري». واللوحةُ تقرأ مفتاحَها (`errors.cash_overdue`).
	ErrCashOverdue = httpx.NewError(http.StatusConflict, "cash_overdue", "errors.cash_overdue")
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
	// ══════════════════════════════════════════════════════════════════
	// **تفصيلُ طلب السطر** — لسطور التحصيل و«ما استلمت العمولة» وحدَها (٢٠٢٦-١٠-٢٦)
	// ══════════════════════════════════════════════════════════════════
	//
	// **حقولٌ مضافةٌ لا مبدَّلة** — التطبيقاتُ القديمةُ تقرأ ما فوقها كما كان.
	// وتغيب (`omitempty`) عن سطور التسليم والتسوية: لا طلبَ وراءها.
	//
	// OrderID معرّفُ الطلب — لزرّ «ما استلمت العمولة».
	OrderID *string `json:"order_id,omitempty"`
	// CustomerName **من دفع** — اسمُ الزبون، أو المستلِمُ في «لدي توصيلة».
	CustomerName *string `json:"customer_name,omitempty"`
	// MerchantName المتجرُ — ومتاجرُ الطلب متعدّد المصادر مجموعةً بـ« · ».
	MerchantName *string `json:"merchant_name,omitempty"`
	// FeePart **أجرةُ التوصيل التي دفعها الزبون** (`delivery_fee`) — صفرٌ في التوصيل المجانيّ.
	FeePart *int64 `json:"fee_part,omitempty"`
	// CommissionPart **عمولةُ المنصّة كما قُيّدت أصلاً** — قبل أيّ محو.
	//
	// **و`FeePart + CommissionPart` = مبلغُ التحصيل** في الطلب النقديّ العاديّ والمتجرُ «نقداً»
	// دفعه السائق. **ويفترقان** بهامش التسعير (سعرُ البيع فوق سعر الشراء) وبخصم الكود،
	// وفي متجرٍ «محفظة» (التحصيلُ يشمل البضاعة) وفي طلبٍ دُفع بعضُه من المحفظة.
	CommissionPart *int64 `json:"commission_part,omitempty"`
	// CommissionWaived **ما مُحي من عمولة الطلب** («ما استلمت العمولة») — صفرٌ إن لم يُمحَ.
	CommissionWaived *int64 `json:"commission_waived,omitempty"`
	// WaivedReason سببُ المحو كما كتبه الموظّف.
	WaivedReason *string `json:"waived_reason,omitempty"`
	// FreeWaived **ما أعفاه عرضُ/كودُ التوصيل المجانيّ** (`promo_delivery_waived`) — صفرٌ إن لا عرض.
	FreeWaived *int64 `json:"free_waived,omitempty"`
}

type Statement struct {
	Held int64 `json:"held"`
	// Limit **سقفُه الفعليّ** — الخاصُّ إن ضُبط وإلّا العامّ (قرارُ المالك ٢٠٢٦-١٠-١٥).
	// كان العامَّ دائماً، **فيرى الموظّفُ سقفاً غيرَ الذي يمنعه** (`GuardTx` يقرأ الخاصّ).
	Limit int64 `json:"limit"`
	// LimitIsOverride **السقفُ خاصٌّ بهذا السائق** — لا العامّ.
	LimitIsOverride bool    `json:"limit_is_override"`
	Entries         []Entry `json:"entries"`
	// **الدفترُ يُقلَّب ولا يُقصّ** (قرارُ المالك ٢٠٢٦-١٠-١٥) — كان آخرَ خمسين بلا
	// كلمة، فيظنّها مَن يراجع كلَّ ما وقع. **والمجموعُ يُقال ولو عُرضت صفحةٌ واحدة.**
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	// OpenOrders **طلباتٌ بيده لم تُغلق ونقدُها** — نصفُ التعرّض الذي لا يظهر في
	// القيود (القيدُ يُكتب عند التسليم). **وهي ما يحسبه `Exposure` بعينه.**
	OpenOrders []OpenCashOrder `json:"open_orders"`
	// OpenCash **مجموعُ نقدها** — و`Held + OpenCash` هو التعرّضُ الذي يقارنه الحارس.
	OpenCash int64 `json:"open_cash"`
}

// OpenCashOrder **طلبٌ بيد السائق لم يُغلق** — بنقده المنتظَر.
//
// **و`ExpectedCash` هو `cash_due` نفسُه** — الرقمُ الذي يحسبه الحارس (`Exposure`).
// وهو **حدٌّ أعلى** لا ما سيبقى بيده تماماً: ما يدفعه للمتاجر عند الاستلام يُطرح
// عند التسليم لا قبله (انظر التعليقَ على `Exposure`).
type OpenCashOrder struct {
	OrderID      string    `json:"order_id"`
	Number       int64     `json:"number"`
	Status       string    `json:"status"`
	ExpectedCash int64     `json:"expected_cash"`
	CreatedAt    time.Time `json:"created_at"`
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

// StatementFor الصفحةُ الأولى من الكشف — للنداءات القديمة.
func (s *Service) StatementFor(ctx context.Context, driverID string, limit int) (*Statement, error) {
	return s.StatementPage(ctx, driverID, limit, 1)
}

// StatementPage **كشفُ صندوق السائق صفحةً صفحة** — ومعه سقفُه الفعليّ وطلباتُه المفتوحة.
func (s *Service) StatementPage(ctx context.Context, driverID string, limit, page int) (*Statement, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	if page < 1 {
		page = 1
	}
	st := &Statement{Entries: []Entry{}, OpenOrders: []OpenCashOrder{},
		Limit: s.Limit(ctx), Page: page, PerPage: limit}
	var err error
	if st.Held, err = s.Held(ctx, driverID); err != nil {
		return nil, err
	}
	// **والسقفُ الخاصُّ يغلب العامّ** — القراءةُ نفسُها في `GuardTx`.
	var override *int64
	if err := s.db.QueryRow(ctx, `SELECT cash_limit_override FROM users WHERE id = $1`,
		driverID).Scan(&override); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if override != nil {
		st.Limit, st.LimitIsOverride = *override, true
	}
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM driver_cash_entries WHERE driver_id = $1`, driverID).
		Scan(&st.Total); err != nil {
		return nil, err
	}
	// **وتفصيلُ الطلب لسطور التحصيل والمحو وردِّ نقد المستلِم وحدَها** — والتسليمُ بلا طلب.
	rows, err := s.db.Query(ctx, `
		SELECT e.id, e.amount, e.kind, e.ref, o.number,
		       e.note, e.created_by, e.created_at,
		       o.id::text,
		       CASE WHEN o.id IS NULL THEN NULL
		            WHEN o.kind = 'merchant_delivery'
		            THEN COALESCE(NULLIF(btrim(o.recipient_name), ''), cu.full_name, '')
		            ELSE COALESCE(cu.full_name, NULLIF(btrim(o.recipient_name), ''), '') END,
		       CASE WHEN o.id IS NULL THEN NULL ELSE COALESCE(
		           (SELECT string_agg(DISTINCT m.name, ' · ' ORDER BY m.name)
		              FROM order_items oi JOIN merchants m ON m.id = oi.merchant_id
		             WHERE oi.order_id = o.id),
		           (SELECT m.name FROM merchants m
		             WHERE m.id = COALESCE(o.merchant_id, o.partner_merchant_id)),
		           '') END,
		       o.delivery_fee,
		       o.platform_commission + o.commission_waived,
		       o.commission_waived,
		       CASE WHEN o.id IS NULL THEN NULL ELSE COALESCE(o.commission_waived_reason, '') END,
		       o.promo_delivery_waived
		FROM driver_cash_entries e
		LEFT JOIN orders o ON e.kind IN ('order_collection', 'commission_waived', 'recipient_cash_returned')
		                  AND o.id::text = e.ref
		LEFT JOIN users cu ON cu.id = o.customer_id
		WHERE e.driver_id = $1
		ORDER BY e.id DESC LIMIT $2 OFFSET $3`, driverID, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Amount, &e.Kind, &e.Ref, &e.OrderNumber, &e.Note, &e.CreatedBy, &e.CreatedAt,
			&e.OrderID, &e.CustomerName, &e.MerchantName, &e.FeePart, &e.CommissionPart,
			&e.CommissionWaived, &e.WaivedReason, &e.FreeWaived); err != nil {
			return nil, err
		}
		st.Entries = append(st.Entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	// **والطلباتُ المفتوحةُ بشرط `Exposure` نفسِه** — نصّان يفترقان يوماً.
	orows, err := s.db.Query(ctx, `
		SELECT o.id::text, o.number, o.status, o.cash_due, o.created_at
		FROM orders o
		WHERE o.driver_id = $1::uuid AND o.closed_at IS NULL
		ORDER BY o.created_at ASC`, driverID)
	if err != nil {
		return nil, err
	}
	defer orows.Close()
	for orows.Next() {
		var o OpenCashOrder
		if err := orows.Scan(&o.OrderID, &o.Number, &o.Status, &o.ExpectedCash, &o.CreatedAt); err != nil {
			return nil, err
		}
		st.OpenCash += o.ExpectedCash
		st.OpenOrders = append(st.OpenOrders, o)
	}
	return st, orows.Err()
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

// WaiveCommissionTx **«ما استلمت العمولة»** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — المتجرُ أخذ من
// السائق ثمنَ البضاعة كاملاً فلم تبقَ العمولةُ بيده: سطرٌ سالبٌ بمقدارها بمرجع الطلب،
// والسببُ في الملاحظة. **داخل معاملة المستدعي** — مع تصحيح المتجر والمندوب والخزينة.
func (s *Service) WaiveCommissionTx(ctx context.Context, q Querier, driverID string,
	amount int64, orderID, reason, actorID string) (int64, error) {
	if amount <= 0 {
		return 0, ErrOverSettle
	}
	return s.applyTx(ctx, q, driverID, -amount, "commission_waived", orderID, reason, &actorID)
}

// ReturnRecipientCashTx **«رجّعت النقد للمستلِم»** (قرارُ المالك ٢٠٢٦-١٠-٢٧) — توصيلةُ «إحضار»
// قبض فيها السائقُ الأجرةَ من المستلِم عند الاستلام ثمّ فشلت بغير ذنبه (المنصّةُ أو السائق):
// **ردّ السائقُ النقدَ بيده**، فسطرٌ سالبٌ بمقداره بمرجع التوصيلة يقابل سطرَ القبض —
// فلا يبقى بذمّته ولا تحسبه الخزينةُ مدفوعاً لها. **داخل معاملة الانتقال.**
func (s *Service) ReturnRecipientCashTx(ctx context.Context, q Querier, driverID string,
	amount int64, orderID string, actorID *string) error {
	if amount <= 0 {
		return nil
	}
	_, err := s.applyTx(ctx, q, driverID, -amount, "recipient_cash_returned", orderID,
		"رجع النقد للمستلم — التوصيلة ما تمت", actorID)
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
