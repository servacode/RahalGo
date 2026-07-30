package orders

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// Transition ينفّذ انتقال حالة بعد التحقق من شرعيته للأدوار الفاعلة،
// ويسجل الحدث، ويطلق التسويات (استرجاع المحفظة، تحرير كود الخصم) عند الإغلاق.
func (s *Service) Transition(ctx context.Context, actorID string, actorRoles []string, orderID, to, note string) (*Order, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var from string
	var driverID *string
	var walletPaid, cashDue int64
	var customerID, promoCode string
	err = tx.QueryRow(ctx, `
		SELECT status, driver_id, wallet_paid, cash_due, customer_id, COALESCE(promo_code,'')
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&from, &driverID, &walletPaid, &cashDue, &customerID, &promoCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if !canTransition(from, to, actorRoles) {
		return nil, ErrBadTransition
	}
	// لا استلام بلا سائق مسند
	if (to == StAtPickup || to == StPickedUp) && driverID == nil {
		return nil, ErrNeedsDriver
	}

	set := `status = $2, updated_at = now()`
	switch to {
	case StAccepted:
		set += `, accepted_at = now()`
	case StPickedUp:
		set += `, picked_up_at = now()`
	case StDelivered:
		set += `, delivered_at = now(), closed_at = now()`
	case StDispatching:
		if from == StAssigned { // فك الإسناد
			set += `, driver_id = NULL`
		}
	}
	if terminal(to) && to != StDelivered {
		set += `, closed_at = now(), cancel_reason = ` + "$3"
	}

	args := []any{orderID, to}
	if terminal(to) && to != StDelivered {
		args = append(args, note)
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET `+set+` WHERE id = $1`, args...); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, $2, $3, $4, $5)`, orderID, from, to, actorID, note); err != nil {
		return nil, err
	}

	// تحرير كود الخصم عند الإلغاء/الرفض/الفشل (لا عند refunded — الخدمة قُدمت)
	if (to == StCancelled || to == StRejected || to == StFailed) && promoCode != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE promo_codes SET used_count = greatest(used_count - 1, 0) WHERE code = $1`,
			promoCode); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM promo_redemptions WHERE order_id = $1`, orderID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// استرجاع المدفوع من المحفظة تلقائياً عند أي نهاية غير التسليم
	if refundOnEnter(to) && walletPaid > 0 {
		if _, err := s.wallet.Apply(ctx, customerID, walletPaid, "refund",
			orderID, fmt.Sprintf("استرجاع طلب (%s)", to), &actorID); err != nil {
			s.logger.Error("wallet refund failed", "order", orderID, "error", err)
		}
	}

	// عند التسليم: نقد الطلب يُقيَّد على صندوق السائق + تسوية العمولات
	if to == StDelivered {
		if cashDue > 0 && driverID != nil {
			if err := s.cashbox.Collect(ctx, *driverID, cashDue, orderID, &actorID); err != nil {
				s.logger.Error("cash collect failed", "order", orderID, "error", err)
			}
		}
		if err := s.settleCommissions(ctx, orderID, actorID); err != nil {
			s.logger.Error("commission settle failed", "order", orderID, "error", err)
		}
	}

	return s.GetByID(ctx, orderID)
}

// settleCommissions يحسب عمولة المنصة من المتجر (لقطة على الطلب)،
// ويقيّد نسبة المندوب منها لمحفظته تلقائياً (PLAN §6.3 + قرار 13).
func (s *Service) settleCommissions(ctx context.Context, orderID, actorID string) error {
	var subtotal int64
	var merchantPct int
	var repID *string
	err := s.db.QueryRow(ctx, `
		SELECT o.subtotal, m.commission_percent, m.sales_rep_user_id
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1`, orderID).Scan(&subtotal, &merchantPct, &repID)
	if err != nil {
		return err
	}

	platformCommission := subtotal * int64(merchantPct) / 100
	if _, err := s.db.Exec(ctx,
		`UPDATE orders SET platform_commission = $2 WHERE id = $1`,
		orderID, platformCommission); err != nil {
		return err
	}
	if platformCommission == 0 || repID == nil {
		return nil
	}

	// نسبة المندوب من عمولة المنصة — إعداد ديناميكي
	var repPct float64
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE((SELECT (value#>>'{}')::float8 FROM app_settings
		                 WHERE key = 'sales.commission_percent'), 10)`).Scan(&repPct); err != nil {
		return err
	}
	repCommission := int64(float64(platformCommission) * repPct / 100)
	if repCommission <= 0 {
		return nil
	}
	_, err = s.wallet.Apply(ctx, *repID, repCommission, "commission",
		orderID, "عمولة مندوب عن طلب مسلَّم", &actorID)
	return err
}

// AssignDriver إسناد يدوي من العمليات: يتحقق أن الحساب سائق نشط ثم يسند وينقل الحالة.
func (s *Service) AssignDriver(ctx context.Context, actorID string, actorRoles []string, orderID, driverID, note string) (*Order, error) {
	var isDriver bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_roles ur JOIN users u ON u.id = ur.user_id
			WHERE ur.user_id = $1 AND ur.role_code = 'driver' AND u.status = 'active')`,
		driverID).Scan(&isDriver)
	if err != nil {
		return nil, err
	}
	if !isDriver {
		return nil, ErrNeedsDriver
	}

	var from string
	var cashDue int64
	err = s.db.QueryRow(ctx, `SELECT status, cash_due FROM orders WHERE id = $1`, orderID).
		Scan(&from, &cashDue)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	// الإسناد اليدوي مسموح من التحضير أو البحث عن سائق
	if from != StPreparing && from != StDispatching {
		return nil, ErrBadTransition
	}
	// السقف النقدي: لا طلبات نقدية لسائق تجاوز سقفه (PLAN §6.3)
	if cashDue > 0 {
		over, err := s.cashbox.OverLimit(ctx, driverID)
		if err != nil {
			return nil, err
		}
		if over {
			return nil, cashbox.ErrLimitExceed
		}
	}

	if _, err := s.db.Exec(ctx,
		`UPDATE orders SET driver_id = $2, updated_at = now() WHERE id = $1`,
		orderID, driverID); err != nil {
		return nil, err
	}
	// من التحضير: نمر عبر dispatching ثم assigned لسجل أحداث سليم
	if from == StPreparing {
		if _, err := s.Transition(ctx, actorID, actorRoles, orderID, StDispatching, note); err != nil {
			return nil, err
		}
	}
	return s.Transition(ctx, actorID, []string{"ops"}, orderID, StAssigned, note)
}
