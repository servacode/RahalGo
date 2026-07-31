package orders

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/wallet"
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

	// كل التسويات المالية **داخل** معاملة الانتقال: إمّا تتم الحالة والمال معاً
	// أو لا يتم شيء. كانت تُنفَّذ بعد الإيداع وأخطاؤها تُبتلع في السجل، فيصير
	// الطلب مُسلَّماً بلا عمولة ولا نقد مقيَّد — خلل مالي صامت لا أثر له.
	var done settled
	if err := s.settle(ctx, tx, settlement{
		orderID: orderID, from: from, to: to, actorID: actorID,
		customerID: customerID, driverID: driverID,
		walletPaid: walletPaid, cashDue: cashDue,
	}, &done); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	updated, err := s.GetByID(ctx, orderID)
	if err == nil {
		s.publishOrder(updated)
		// بعد الإيداع: فشل الإشعار لا يُبطل تسليماً وقع فعلاً
		s.notifyTransition(ctx, orderID, to, note)
		s.notifyCommission(ctx, done.repID, orderID, done.commissionPaid)
	}
	return updated, err
}

// settled ما وقع فعلاً من تسويات — يُملأ داخل المعاملة ويُقرأ بعد نجاحها
// لإطلاق الإشعارات. متغيّر محلي لكل طلب: الخدمة مشتركة بين كل الطلبات المتزامنة
// فلا يجوز أن تحمل حالة طلب بعينه.
type settled struct {
	repID          string
	commissionPaid int64
}

// settlement مدخلات التسوية المالية لانتقال واحد.
type settlement struct {
	orderID, from, to, actorID, customerID string
	driverID                               *string
	walletPaid, cashDue                    int64
}

// settle ينفّذ كل الأثر المالي لانتقال الحالة داخل معاملة المستدعي.
//
// القاعدة المحاسبية المعتمدة — الاسترجاع يعكس ما قيّده التسليم بالضبط:
//   - نهاية غير التسليم **قبل** التسليم: يُعاد المدفوع من المحفظة فقط
//     (النقد لم يُحصَّل أصلاً).
//   - استرجاع **بعد** التسليم: يُعاد **كامل المبلغ** إلى محفظة الزبون — لأنه
//     دفع النقد فعلاً للسائق — وتُعكس عمولة المندوب وتُصفَّر عمولة المنصة.
//     صندوق السائق يبقى كما هو عن قصد: النقد الذي قبضه ما زال بحوزته ويدين به
//     للمنصة، والمنصة هي من ردّت للزبون. هكذا يتوازن الطرفان بلا رصيد سالب.
func (s *Service) settle(ctx context.Context, q wallet.Querier, in settlement, out *settled) error {
	// (1) نهاية غير التسليم قبل التسليم — استرجاع ما دُفع من المحفظة
	if refundOnEnter(in.to) && in.from != StDelivered && in.walletPaid > 0 {
		if _, err := s.wallet.ApplyTx(ctx, q, in.customerID, in.walletPaid, "refund",
			in.orderID, fmt.Sprintf("استرجاع طلب (%s)", in.to), &in.actorID); err != nil {
			return err
		}
	}

	// (2) التسليم — تحصيل النقد وتسوية العمولات
	if in.to == StDelivered {
		if in.cashDue > 0 && in.driverID != nil {
			if err := s.cashbox.CollectTx(ctx, q, *in.driverID, in.cashDue, in.orderID, &in.actorID); err != nil {
				return err
			}
		}
		return s.settleCommissions(ctx, q, in.orderID, in.actorID, out)
	}

	// (3) استرجاع بعد التسليم — عكس كل ما سبق
	if refundOnEnter(in.to) && in.from == StDelivered {
		if total := in.walletPaid + in.cashDue; total > 0 {
			if _, err := s.wallet.ApplyTx(ctx, q, in.customerID, total, "refund",
				in.orderID, "استرجاع طلب مُسلَّم", &in.actorID); err != nil {
				return err
			}
		}
		return s.reverseCommissions(ctx, q, in.orderID, in.actorID)
	}
	return nil
}

// settleCommissions يحسب عمولة المنصة من المتجر (لقطة على الطلب)،
// ويقيّد نسبة المندوب منها لمحفظته تلقائياً (PLAN §6.3 + قرار 13).
func (s *Service) settleCommissions(ctx context.Context, q wallet.Querier, orderID, actorID string, out *settled) error {
	var subtotal int64
	var merchantPct int
	var repID *string
	err := q.QueryRow(ctx, `
		SELECT o.subtotal, m.commission_percent, m.sales_rep_user_id
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1`, orderID).Scan(&subtotal, &merchantPct, &repID)
	if err != nil {
		return err
	}

	platformCommission := subtotal * int64(merchantPct) / 100
	if _, err := q.Exec(ctx,
		`UPDATE orders SET platform_commission = $2 WHERE id = $1`,
		orderID, platformCommission); err != nil {
		return err
	}
	if platformCommission == 0 || repID == nil {
		return nil
	}

	repCommission, err := s.repShare(ctx, q, platformCommission)
	if err != nil || repCommission <= 0 {
		return err
	}
	if _, err := s.wallet.ApplyTx(ctx, q, *repID, repCommission, "commission",
		orderID, "عمولة مندوب عن طلب مسلَّم", &actorID); err != nil {
		return err
	}
	out.repID, out.commissionPaid = *repID, repCommission
	return nil
}

// reverseCommissions يعكس أثر التسليم المالي عند استرجاع طلب مُسلَّم:
// قيد مضاد لعمولة المندوب (الدفاتر لا تُعدَّل ولا تُحذف — تُصحَّح بقيد مقابل)
// وتصفير لقطة عمولة المنصة كي لا تتضخّم التقارير وفواتير المتاجر.
func (s *Service) reverseCommissions(ctx context.Context, q wallet.Querier, orderID, actorID string) error {
	var platformCommission int64
	var repID *string
	err := q.QueryRow(ctx, `
		SELECT o.platform_commission, m.sales_rep_user_id
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1`, orderID).Scan(&platformCommission, &repID)
	if err != nil {
		return err
	}
	if _, err := q.Exec(ctx,
		`UPDATE orders SET platform_commission = 0 WHERE id = $1`, orderID); err != nil {
		return err
	}
	if platformCommission == 0 || repID == nil {
		return nil
	}

	repCommission, err := s.repShare(ctx, q, platformCommission)
	if err != nil || repCommission <= 0 {
		return err
	}
	// قد يكون رصيد المندوب أقلّ من العمولة (سحبها) — عندها يُرفض القيد بـ
	// insufficient_balance، وهو رفض صحيح: الدَّين يُسوّى يدوياً من المالية.
	_, err = s.wallet.ApplyTx(ctx, q, *repID, -repCommission, "adjustment",
		orderID, "عكس عمولة مندوب — طلب مُسترجَع", &actorID)
	return err
}

// repShare نصيب المندوب من عمولة المنصة — نسبة ديناميكية من الإعدادات.
func (s *Service) repShare(ctx context.Context, q wallet.Querier, platformCommission int64) (int64, error) {
	var repPct float64
	if err := q.QueryRow(ctx, `
		SELECT COALESCE((SELECT (value#>>'{}')::float8 FROM app_settings
		                 WHERE key = 'sales.commission_percent'), 10)`).Scan(&repPct); err != nil {
		return 0, err
	}
	return int64(float64(platformCommission) * repPct / 100), nil
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
