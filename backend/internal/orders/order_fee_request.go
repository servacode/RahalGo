package orders

// ══════════════════════════════════════════════════════════════════════
// **«اطلب تعديل الأجرة» — الكابتن يطلب، والإدارة توافق أو ترفض** (قرار المالك ٢٠٢٦-١١-١٨)
// ══════════════════════════════════════════════════════════════════════
//
// «تعديل السعر على طلبية الواتس اب ايضا من الكرت ومن تطبيق الكابتن» — والمالك اختار: الكابتن **يطلب**
// والإدارة **توافق**. فالكابتن ما بيغيّر المصاري بإيده:
//
//	طلب الكابتن  ←  صفّ معلّق بـ`order_fee_requests` (الأجرة الجديدة والسبب والأجرة وقتها)، وإشعار للمكتب،
//	                 وبيطلع على كارت الطلب باللوحة. طلب جديد بيحلّ محل المعلّق.
//	موافقة       ←  `EditDeliveryFee` نفسه — نفس الحسبة (الزبون بيدفع الزيادة) ونفس القيود (نقد انقبض،
//	                 محفظة ما بتكفي) ونفس التدقيق والإشعارات — والصفّ بيتعلّم بنفس المعاملة.
//	رفض          ←  بسبب إلزامي، والكابتن بيوصله إشعار.
//
// **والأجرة تغيّرت تحت الطلب** (المكتب عدّلها قبل الموافقة) ⇒ الموافقة بتنرد `fee_request_stale`
// والطلب بيصير «قديم» — والكابتن بيبعت غيره. **وطلب الواتساب وحده** هلق — متل ما طلب المالك.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

var (
	// ErrFeeRequestStage **مو طلب واتساب، أو انتهى، أو لسا بلا كابتن.**
	ErrFeeRequestStage = httpx.NewError(http.StatusConflict, "fee_request_stage", "errors.fee_request_stage")
	// ErrFeeRequestSame **نفس الأجرة الحالية.**
	ErrFeeRequestSame = httpx.NewError(http.StatusUnprocessableEntity, "fee_request_same", "errors.fee_request_same")
	// ErrFeeRequestNotPending **ما في طلب معلّق** — انقرّر أو انسحب.
	ErrFeeRequestNotPending = httpx.NewError(http.StatusConflict, "fee_request_not_pending", "errors.fee_request_not_pending")
	// ErrFeeRequestStale **الأجرة تغيّرت من وقت الطلب** — الكابتن بيبعت غيره.
	ErrFeeRequestStale = httpx.NewError(http.StatusConflict, "fee_request_stale", "errors.fee_request_stale")
)

// FeeRequestView **طلب الأجرة كما يراه الكابتن والمكتب.**
type FeeRequestView struct {
	ID           string `json:"id"`
	OldFee       int64  `json:"old_fee"`
	Fee          int64  `json:"fee"`
	Reason       string `json:"reason"`
	Status       string `json:"status"`
	RejectReason string `json:"reject_reason,omitempty"`
	CreatedAt    string `json:"created_at"`
	DriverName   string `json:"driver_name,omitempty"`
}

const feeRequestCols = `r.id::text, r.old_fee, r.fee, r.reason, r.status, r.reject_reason,
	to_char(r.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
	COALESCE((SELECT u.full_name FROM users u WHERE u.id = r.driver_id), '')`

func scanFeeRequest(row pgx.Row) (*FeeRequestView, error) {
	v := &FeeRequestView{}
	err := row.Scan(&v.ID, &v.OldFee, &v.Fee, &v.Reason, &v.Status, &v.RejectReason, &v.CreatedAt, &v.DriverName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return v, err
}

// RequestFeeChange **الكابتن يطلب أجرة جديدة لطلب الواتساب اللي معه.**
func (s *Service) RequestFeeChange(ctx context.Context, orderID, driverID string, fee int64, reason string) (*FeeRequestView, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, ErrOrderEditReason
	}
	if utf8.RuneCountInString(reason) > 500 {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if fee < 1 || fee > maxEditFee {
		return nil, ErrFeeEditRange
	}
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, httpx.ErrNotFound
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status, origin, driver string
	var number, current int64
	err = tx.QueryRow(ctx, `
		SELECT status, origin, COALESCE(driver_id::text, ''), number, delivery_fee + promo_delivery_waived
		  FROM orders WHERE id = $1 FOR UPDATE`, orderID).Scan(&status, &origin, &driver, &number, &current)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && driver != driverID) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if origin != OriginWhatsApp || terminal(status) {
		return nil, ErrFeeRequestStage
	}
	if fee == current {
		return nil, ErrFeeRequestSame
	}
	if _, err := tx.Exec(ctx, `
		UPDATE order_fee_requests SET status = 'superseded', decided_at = now()
		 WHERE order_id = $1 AND status = 'pending'`, orderID); err != nil {
		return nil, err
	}
	v, err := scanFeeRequest(tx.QueryRow(ctx, `
		WITH ins AS (
		  INSERT INTO order_fee_requests (order_id, driver_id, old_fee, fee, reason)
		  VALUES ($1, $2, $3, $4, $5) RETURNING *)
		SELECT `+feeRequestCols+` FROM ins r`, orderID, driverID, current, fee, reason))
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		VALUES ($1, $2, $2, $3, $4, $3)`, orderID, status, driverID,
		fmt.Sprintf("الكابتن طلب تعديل الأجرة %s ← %s ل.س: %s", groupDigits(current), groupDigits(fee), reason)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if s.notify != nil {
		s.notify.NotifyOps(ctx, notifications.Input{
			Kind: notifications.KindOrder, Title: t.feeRequestOps,
			Body:   fmt.Sprintf("#%d — %s ← %s ل.س: %s", number, groupDigits(current), groupDigits(fee), reason),
			Entity: "order", EntityID: orderID, Href: "/dashboard/orders?id=" + orderID,
		})
	}
	if o, err := s.GetByID(ctx, orderID); err == nil {
		s.publishOrder(o)
	}
	return v, nil
}

// DriverFeeRequest **آخر طلب أجرة للكابتن على هالطلب** — معلّق أو آخر قرار، وnil: ما في.
func (s *Service) DriverFeeRequest(ctx context.Context, orderID, driverID string) (*FeeRequestView, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, httpx.ErrNotFound
	}
	return scanFeeRequest(s.db.QueryRow(ctx, `SELECT `+feeRequestCols+` FROM order_fee_requests r
		 WHERE r.order_id = $1 AND r.driver_id = $2 AND r.status IN ('pending', 'approved', 'rejected', 'stale')
		 ORDER BY r.created_at DESC LIMIT 1`, orderID, driverID))
}

// PendingFeeRequest **الطلب المعلّق للمكتب** — وnil: ما في.
func (s *Service) PendingFeeRequest(ctx context.Context, orderID string) (*FeeRequestView, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, httpx.ErrNotFound
	}
	return scanFeeRequest(s.db.QueryRow(ctx, `SELECT `+feeRequestCols+` FROM order_fee_requests r
		 WHERE r.order_id = $1 AND r.status = 'pending'`, orderID))
}

// ApproveFeeRequest **المكتب يوافق** — من باب `EditDeliveryFee` نفسه، **ومعاينةٌ بلا أثر** بـ`dryRun`.
func (s *Service) ApproveFeeRequest(ctx context.Context, actorID, orderID, requestID string, dryRun bool,
	hook func(context.Context, dbtx.Querier, EditFeeResult) error) (*Order, EditFeeResult, error) {
	var res EditFeeResult
	if _, err := uuid.Parse(requestID); err != nil {
		return nil, res, ErrFeeRequestNotPending
	}
	r, err := s.PendingFeeRequest(ctx, orderID)
	if err != nil {
		return nil, res, err
	}
	if r == nil || r.ID != requestID {
		return nil, res, ErrFeeRequestNotPending
	}
	var current int64
	var driverID string
	if err := s.db.QueryRow(ctx, `
		SELECT o.delivery_fee + o.promo_delivery_waived, r.driver_id::text
		  FROM orders o JOIN order_fee_requests r ON r.order_id = o.id WHERE r.id = $1`, requestID).
		Scan(&current, &driverID); err != nil {
		return nil, res, err
	}
	if current != r.OldFee {
		if !dryRun {
			_, _ = s.db.Exec(ctx, `UPDATE order_fee_requests SET status = 'stale', decided_at = now(),
				decided_by = NULLIF($2, '')::uuid WHERE id = $1 AND status = 'pending'`, requestID, actorID)
			s.notifyFeeRequestDecision(ctx, orderID, driverID, t.feeRequestRejectedDriver,
				"الأجرة تغيّرت من وقت طلبك — ابعت طلب جديد إذا لازم")
		}
		return nil, res, ErrFeeRequestStale
	}
	req := EditFeeRequest{Fee: r.Fee, Reason: "طلب الكابتن: " + r.Reason, DryRun: dryRun}
	o, res, err := s.EditDeliveryFee(ctx, actorID, orderID, req,
		func(ctx context.Context, q dbtx.Querier, res EditFeeResult) error {
			tag, err := q.Exec(ctx, `UPDATE order_fee_requests SET status = 'approved', decided_at = now(),
				decided_by = NULLIF($2, '')::uuid WHERE id = $1 AND status = 'pending'`, requestID, actorID)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return ErrFeeRequestNotPending
			}
			if hook != nil {
				return hook(ctx, q, res)
			}
			return nil
		})
	return o, res, err
}

// RejectFeeRequest **المكتب يرفض بسبب** — والكابتن بيوصله.
func (s *Service) RejectFeeRequest(ctx context.Context, actorID, orderID, requestID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrOrderEditReason
	}
	if utf8.RuneCountInString(reason) > 500 {
		return httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if _, err := uuid.Parse(requestID); err != nil {
		return ErrFeeRequestNotPending
	}
	if _, err := uuid.Parse(orderID); err != nil {
		return httpx.ErrNotFound
	}
	var driverID string
	err := s.db.QueryRow(ctx, `
		UPDATE order_fee_requests SET status = 'rejected', decided_at = now(),
		       decided_by = NULLIF($3, '')::uuid, reject_reason = $4
		 WHERE id = $1 AND order_id = $2 AND status = 'pending'
		RETURNING driver_id::text`, requestID, orderID, actorID, reason).Scan(&driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrFeeRequestNotPending
	}
	if err != nil {
		return err
	}
	_, _ = s.db.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		SELECT id, status, status, NULLIF($2, '')::uuid, $3, driver_id FROM orders WHERE id = $1`,
		orderID, actorID, "المكتب رفض طلب تعديل الأجرة: "+reason)
	s.notifyFeeRequestDecision(ctx, orderID, driverID, t.feeRequestRejectedDriver, "السبب: "+reason)
	if o, err := s.GetByID(ctx, orderID); err == nil {
		s.publishOrder(o)
	}
	return nil
}

func (s *Service) notifyFeeRequestDecision(ctx context.Context, orderID, driverID, title, detail string) {
	if s.notify == nil || driverID == "" {
		return
	}
	var number int64
	_ = s.db.QueryRow(ctx, `SELECT number FROM orders WHERE id = $1`, orderID).Scan(&number)
	s.notify.Notify(ctx, notifications.Input{
		UserID: driverID, Kind: notifications.KindOrder, Title: title,
		Body:   fmt.Sprintf("#%d — %s", number, detail),
		Entity: "order", EntityID: orderID, Href: "/portal",
		Apps: []string{notifications.AppDriver},
	})
}
