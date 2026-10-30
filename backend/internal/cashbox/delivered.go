package cashbox

import (
	"context"
	"time"
)

// DeliveredOrder **طلبٌ سلّمه السائقُ** — نقداً كان أم لا (قرارُ المالك ٢٠٢٦-١٠-٣٠).
//
// كشفُ الصندوق لا يرى إلّا ما قُبض نقداً، **فطلبٌ دفعه المتجرُ من محفظته لا قيدَ له**
// فيظنّه المالكُ ضائعاً. هذه قراءةٌ للعرض وحدَه: لا تكتب شيئاً، ولا تدخل حساباً.
type DeliveredOrder struct {
	OrderID      string    `json:"order_id"`
	Number       int64     `json:"number"`
	Kind         string    `json:"kind"`
	CustomMode   string    `json:"custom_mode"`
	DeliveryMode string    `json:"delivery_mode"`
	FeePayer     string    `json:"fee_payer"`
	Payment      string    `json:"payment_method"`
	WalletPaid   int64     `json:"wallet_paid"`
	CashDue      int64     `json:"cash_due"`
	DeliveryFee  int64     `json:"delivery_fee"`
	Commission   int64     `json:"platform_commission"`
	Waived       int64     `json:"commission_waived"`
	FreeWaived   int64     `json:"free_waived"`
	MerchantName string    `json:"merchant_name"`
	DeliveredAt  time.Time `json:"delivered_at"`
	// CashCollected **ما قُيّد بذمّته من هذا الطلب** — مجموعُ سطور التحصيل، وصفرٌ لما لم يُقبض نقداً.
	CashCollected int64 `json:"cash_collected"`
}

// MaxDelivered حدُّ ما يُرجَع في نداءٍ واحد.
const MaxDelivered = 500

// Delivered **ما سلّمه السائقُ بين وقتين** — الأحدثُ أوّلاً، بحدٍّ أعلى `MaxDelivered`.
func (s *Service) Delivered(ctx context.Context, driverID string, from, to time.Time) ([]DeliveredOrder, error) {
	rows, err := s.db.Query(ctx, `
		SELECT o.id::text, o.number, o.kind, COALESCE(o.custom_mode, ''), COALESCE(o.delivery_mode, ''),
		       COALESCE(o.fee_payer, ''), o.payment_method, o.wallet_paid, o.cash_due,
		       o.delivery_fee, o.platform_commission, o.commission_waived, o.promo_delivery_waived,
		       COALESCE(
		           (SELECT string_agg(DISTINCT m.name, ' · ' ORDER BY m.name)
		              FROM order_items oi JOIN merchants m ON m.id = oi.merchant_id
		             WHERE oi.order_id = o.id),
		           (SELECT m.name FROM merchants m
		             WHERE m.id = COALESCE(o.merchant_id, o.partner_merchant_id)),
		           ''),
		       o.delivered_at,
		       COALESCE((SELECT sum(e.amount) FROM driver_cash_entries e
		                  WHERE e.driver_id = o.driver_id AND e.kind = 'order_collection'
		                    AND e.ref = o.id::text), 0)
		FROM orders o
		WHERE o.driver_id = $1::uuid AND o.status = 'delivered' AND o.delivered_at IS NOT NULL
		  AND o.delivered_at >= $2 AND o.delivered_at < $3
		ORDER BY o.delivered_at DESC
		LIMIT $4`, driverID, from, to, MaxDelivered)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DeliveredOrder{}
	for rows.Next() {
		var d DeliveredOrder
		if err := rows.Scan(&d.OrderID, &d.Number, &d.Kind, &d.CustomMode, &d.DeliveryMode,
			&d.FeePayer, &d.Payment, &d.WalletPaid, &d.CashDue,
			&d.DeliveryFee, &d.Commission, &d.Waived, &d.FreeWaived,
			&d.MerchantName, &d.DeliveredAt, &d.CashCollected); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
