package orders

import (
	"context"
	"time"
)

// CashBanInfo **منعُ النقد كما يراه الموظّف** — بسببه وتاريخِ انتهائه (قرارُ المالك ٢٠٢٦-١٠-٠٤).
//
// **والحسبةُ حسبةُ `cashBlocked` نفسُها**: الفشلُ بذنبه خلال المدّة وبعد آخر رفع.
// **والانتهاءُ** لحظةُ خروجِ الفشلةِ التي تُنزل العدَّ تحت الحدّ من النافذة.
type CashBanInfo struct {
	Blocked  bool       `json:"blocked"`
	Failures int64      `json:"failures"`
	Limit    int64      `json:"limit"`
	Days     int64      `json:"days"`
	Until    *time.Time `json:"until"`
	Orders   []int64    `json:"orders"`
	LiftedAt *time.Time `json:"lifted_at"`
}

// CashBan يقرأ حالَ منع النقد لزبون.
func (s *Service) CashBan(ctx context.Context, customerID string) (CashBanInfo, error) {
	out := CashBanInfo{Orders: []int64{}}
	if s.settings == nil {
		return out, nil
	}
	out.Limit = s.settings.GetInt(ctx, "customers.cash_ban_failures")
	out.Days = s.settings.GetInt(ctx, "customers.cash_ban_days")
	_ = s.db.QueryRow(ctx, `SELECT cash_ban_lifted_at FROM users WHERE id = $1`, customerID).Scan(&out.LiftedAt)
	if out.Limit <= 0 || out.Days <= 0 {
		return out, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT o.number, e.created_at
		FROM order_events e
		JOIN orders o ON o.id = e.order_id
		WHERE o.customer_id = $1
		  AND e.to_status = 'failed'
		  AND o.fault = 'customer'
		  AND e.created_at > now() - ($2::int * interval '1 day')
		  AND e.created_at > COALESCE((SELECT cash_ban_lifted_at FROM users
		                               WHERE id = $1), '-infinity'::timestamptz)
		ORDER BY e.created_at DESC`, customerID, out.Days)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	var at []time.Time
	for rows.Next() {
		var n int64
		var t time.Time
		if err := rows.Scan(&n, &t); err != nil {
			return out, err
		}
		out.Orders = append(out.Orders, n)
		at = append(at, t)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	out.Failures = int64(len(at))
	if out.Failures >= out.Limit {
		out.Blocked = true
		// الأحدثُ أوّلاً — فالفشلةُ رقمُ `Limit` هي التي يُنزل خروجُها العدَّ تحت الحدّ.
		u := at[out.Limit-1].Add(time.Duration(out.Days) * 24 * time.Hour)
		out.Until = &u
	}
	return out, nil
}
