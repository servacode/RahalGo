package orders

// ══════════════════════════════════════════════════════════════════════
// **تعويضُ السائق بعد موافقة العمليات — لا لحظةَ الضغطة** (٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// `compensateDriverOnFail` كانت تقيّد نصفَ أجر السائق **في معاملة الفشل نفسِها**
// — بلا يد. **وقِيس على التجهيز**: الخادمُ يقبل سبباً لا يخصّ المرحلة، **ويدفع
// ٥٬٠٠٠ فوراً في كلّ مرّة، حتّى لـ«تأخّرتُ أنا».** فسائقٌ يكسب بضغطة.
//
// # فقرّر المالك: «بعد موافقة العمليات»
//
// **وما بقي من القرار القديم**: السائقُ يُعوَّض إن لم يكن الذنبُ ذنبَه (زبونٌ أو
// متجر)، **والمطالبةُ على المتجر تُفتح كما هي.** والذي تغيّر **متى** يُقبض: حين
// يوافق إنسانٌ لا لحظةَ الضغطة.
//
// # فالمحرّكُ يكتب طلباً معلَّقاً ولا يقيّد شيئاً
//
// صفٌّ في `driver_compensation_requests` بالذنب والسبب **والمبلغِ المقترَح**
// (المعادلةُ القديمةُ نفسُها)، **وتنبيهٌ للعمليات.** والموافقةُ من الباب اليدويّ
// القائم (`server/failure_aftermath.go`) — **وهو يحرس التكرارَ أصلاً**
// (`driver_already_compensated`)، فلا بابَ ثانٍ للمال.

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// حالاتُ طلب التعويض.
const (
	CompensationPending  = "pending"
	CompensationApproved = "approved"
	CompensationRejected = "rejected"
)

// CompensationRequest طلبُ تعويضٍ ينتظر — أو قُضي فيه.
type CompensationRequest struct {
	ID              string     `json:"id"`
	OrderID         string     `json:"order_id"`
	OrderNumber     int64      `json:"order_number"`
	OrderStatus     string     `json:"order_status"`
	DriverID        string     `json:"driver_id"`
	DriverName      string     `json:"driver_name"`
	Fault           string     `json:"fault"`
	FailReason      string     `json:"fail_reason"`
	SuggestedAmount int64      `json:"suggested_amount"`
	Status          string     `json:"status"`
	Amount          *int64     `json:"amount"`
	DecidedAt       *time.Time `json:"decided_at"`
	DecisionNote    string     `json:"decision_note"`
	CreatedAt       time.Time  `json:"created_at"`
}

// requestDriverCompensation **يكتب طلبَ تعويضٍ معلَّقاً — بدل القيد.**
//
// يُنادى من مسارين: الفشلُ عند باب الزبون (`settle` الخطوة ٤)، **وتعذّرُ
// المتجر** (`merchant_blocked.go`). **والذنبُ يُمرَّر لا يُقرأ**: في التوصيلة
// لا يُكتب على الطلب (يبقى حيّاً عند المتجر)، **والطلبُ ما زال يُعوَّض عنه.**
//
// ويردّ `true` إن كُتب طلبٌ جديد — **فيُنبَّه المكتبُ بعد التثبيت لا قبله.**
func (s *Service) requestDriverCompensation(ctx context.Context, q wallet.Querier,
	orderID string, driverID *string, fault, failReason string, driverFee int64) (bool, error) {
	if driverID == nil || s.settings == nil {
		return false, nil
	}
	// **ذنبُ السائق لا تعويضَ فيه** — ومن أخّر فبرد الطعامُ لا يُؤجَر على
	// تأخيره. **والمجهولُ لا يُنسب إلى أحد.**
	if fault != FaultCustomer && fault != FaultMerchant {
		return false, nil
	}
	// **وما عُوِّض لا يُطلَب له ثانيةً** — قيدٌ يدويٌّ سبق للطلب نفسِه.
	var already bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM wallet_transactions
		              WHERE ref = $1 AND kind = 'compensation' AND user_id = $2)`,
		orderID, *driverID).Scan(&already); err != nil {
		return false, err
	}
	if already {
		return false, nil
	}
	// **النسبةُ صفرٌ تعني «بلا تعويض»** (لفظُ الإعداد) — فلا يُطلَب شيء.
	pct := s.settings.GetInt(ctx, "drivers.failed_compensation_percent")
	if pct <= 0 {
		return false, nil
	}
	suggested := int64(0)
	if driverFee > 0 {
		suggested = driverFee * pct / 100
	}
	tag, err := q.Exec(ctx, `
		INSERT INTO driver_compensation_requests
		    (order_id, driver_id, fault, fail_reason, suggested_amount)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (order_id, driver_id) DO NOTHING`,
		orderID, *driverID, fault, failReason, suggested)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// alertCompensationPending **تنبيهُ المكتب بعد التثبيت** — طلبُ تعويضٍ ينتظره.
//
// **وبعد الإيداع لا داخلَه**: تنبيهٌ عن طلبٍ رُدّت معاملتُه يدعو العملياتِ إلى
// الموافقة على ما لا وجودَ له.
func (s *Service) alertCompensationPending(ctx context.Context, orderID string) {
	if s.notify == nil {
		return
	}
	var number, suggested int64
	var reason string
	if err := s.db.QueryRow(ctx, `
		SELECT o.number, r.suggested_amount, r.fail_reason
		FROM driver_compensation_requests r JOIN orders o ON o.id = r.order_id
		WHERE r.order_id = $1 AND r.status = 'pending'
		ORDER BY r.created_at DESC LIMIT 1`, orderID).
		Scan(&number, &suggested, &reason); err != nil {
		return
	}
	s.notify.NotifyOps(ctx, notifications.Input{
		Kind:     notifications.KindOrder,
		Title:    "تعويضُ سائقٍ ينتظر موافقتك",
		Body:     "#" + itoa(number) + " — " + reason + " — المقترَح " + itoa(suggested),
		Entity:   "order",
		EntityID: orderID,
		// **وينقر فيصل إلى قائمة الموافقة نفسِها** — لا إلى الطلبات يبحث فيها.
		Href: "/dashboard/compensations",
	})
}

// PendingCompensationTx **طلبُ التعويض المعلَّق لهذا الطلب — مقفولاً.**
//
// يُقرأ في معاملة الموافقة: **القفلُ يجعل ضغطتين متزامنتين واحدة.** ويردّ
// `nil` إن لم يكن ثمّة طلبٌ معلَّق.
func (s *Service) PendingCompensationTx(ctx context.Context, q wallet.Querier,
	orderID string) (*CompensationRequest, error) {
	var c CompensationRequest
	err := q.QueryRow(ctx, `
		SELECT id::text, order_id::text, driver_id::text, fault, fail_reason,
		       suggested_amount, status, created_at
		FROM driver_compensation_requests
		WHERE order_id = $1 AND status = 'pending'
		ORDER BY created_at ASC LIMIT 1
		FOR UPDATE`, orderID).
		Scan(&c.ID, &c.OrderID, &c.DriverID, &c.Fault, &c.FailReason,
			&c.SuggestedAmount, &c.Status, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// DecideCompensationTx **يُغلق الطلبَ المعلَّق** — موافقةً بمبلغٍ أو رفضاً.
//
// **ولا يقيّد مالاً**: القيدُ في الباب اليدويّ نفسِه وفي المعاملة نفسِها —
// **وهذه تكتب القرارَ بجانبه.**
func (s *Service) DecideCompensationTx(ctx context.Context, q wallet.Querier,
	id, status string, amount int64, actorID, note string) error {
	var amt any
	if status == CompensationApproved {
		amt = amount
	}
	_, err := q.Exec(ctx, `
		UPDATE driver_compensation_requests
		SET status = $2, amount = $3, decided_by = $4, decided_at = now(),
		    decision_note = $5
		WHERE id = $1 AND status = 'pending'`, id, status, amt, actorID, note)
	return err
}

// OpenMerchantClaimTx **المطالبةُ على المتجر تُفتح بما دُفع فعلاً** — عند
// الموافقة على تعويضٍ ذنبُه على المتجر، لا عند الضغطة.
func (s *Service) OpenMerchantClaimTx(ctx context.Context, q wallet.Querier,
	orderID string, amount int64) error {
	return s.openMerchantClaim(ctx, q, orderID, amount)
}

// PendingCompensations **ما ينتظر قراراً** — بالأقدم أوّلاً.
func (s *Service) PendingCompensations(ctx context.Context, limit, offset int) ([]CompensationRequest, int, error) {
	var total int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM driver_compensation_requests WHERE status = 'pending'`).
		Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT r.id::text, r.order_id::text, o.number, o.status,
		       r.driver_id::text, COALESCE(NULLIF(u.full_name, ''), u.phone::text, ''),
		       r.fault, r.fail_reason, r.suggested_amount, r.status, r.amount,
		       r.decided_at, r.decision_note, r.created_at
		FROM driver_compensation_requests r
		JOIN orders o ON o.id = r.order_id
		JOIN users u ON u.id = r.driver_id
		WHERE r.status = 'pending'
		ORDER BY r.created_at ASC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []CompensationRequest{}
	for rows.Next() {
		var c CompensationRequest
		if err := rows.Scan(&c.ID, &c.OrderID, &c.OrderNumber, &c.OrderStatus,
			&c.DriverID, &c.DriverName, &c.Fault, &c.FailReason,
			&c.SuggestedAmount, &c.Status, &c.Amount,
			&c.DecidedAt, &c.DecisionNote, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}
