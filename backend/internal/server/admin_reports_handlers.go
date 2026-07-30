package server

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// تقرير تشغيلي ومالي لمدى زمني (افتراضياً آخر 7 أيام).
func (s *Server) handleReports(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	to, err := time.Parse("2006-01-02", q.Get("to"))
	if err != nil {
		to = time.Now()
	}
	from, err := time.Parse("2006-01-02", q.Get("from"))
	if err != nil {
		from = to.AddDate(0, 0, -6)
	}
	// نهاية اليوم "إلى" شاملة
	toEnd := to.AddDate(0, 0, 1)

	ctx := r.Context()

	// الملخص
	var summary struct {
		OrdersTotal     int     `json:"orders_total"`
		Delivered       int     `json:"delivered"`
		Cancelled       int     `json:"cancelled"`
		GrossSales      int64   `json:"gross_sales"`
		DeliveryFees    int64   `json:"delivery_fees"`
		Commissions     int64   `json:"commissions"`
		WalletPaid      int64   `json:"wallet_paid"`
		CashCollected   int64   `json:"cash_collected"`
		AvgDeliveryMin  float64 `json:"avg_delivery_min"`
		ActiveCustomers int     `json:"active_customers"`
	}
	err = s.pg.QueryRow(ctx, `
		SELECT
			count(*),
			count(*) FILTER (WHERE status = 'delivered'),
			count(*) FILTER (WHERE status IN ('cancelled','rejected','failed')),
			COALESCE(sum(total) FILTER (WHERE status = 'delivered'), 0),
			COALESCE(sum(delivery_fee) FILTER (WHERE status = 'delivered'), 0),
			COALESCE(sum(platform_commission) FILTER (WHERE status = 'delivered'), 0),
			COALESCE(sum(wallet_paid) FILTER (WHERE status = 'delivered'), 0),
			COALESCE(sum(cash_due) FILTER (WHERE status = 'delivered'), 0),
			COALESCE(round(avg(EXTRACT(EPOCH FROM delivered_at - created_at) / 60)
				FILTER (WHERE status = 'delivered' AND delivered_at IS NOT NULL))::float8, 0),
			count(DISTINCT customer_id)
		FROM orders WHERE created_at >= $1 AND created_at < $2`, from, toEnd).
		Scan(&summary.OrdersTotal, &summary.Delivered, &summary.Cancelled,
			&summary.GrossSales, &summary.DeliveryFees, &summary.Commissions,
			&summary.WalletPaid, &summary.CashCollected, &summary.AvgDeliveryMin,
			&summary.ActiveCustomers)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// السلسلة اليومية (كل الأيام حتى الفارغة)
	type day struct {
		Day       string `json:"day"`
		Orders    int    `json:"orders"`
		Delivered int    `json:"delivered"`
		Sales     int64  `json:"sales"`
	}
	daily := []day{}
	rows, err := s.pg.Query(ctx, `
		SELECT d::date::text,
		       COALESCE(count(o.id), 0),
		       COALESCE(count(o.id) FILTER (WHERE o.status = 'delivered'), 0),
		       COALESCE(sum(o.total) FILTER (WHERE o.status = 'delivered'), 0)
		FROM generate_series($1::date, $2::date, '1 day') d
		LEFT JOIN orders o ON o.created_at::date = d::date
		GROUP BY d ORDER BY d`, from, to)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for rows.Next() {
		var dd day
		if err := rows.Scan(&dd.Day, &dd.Orders, &dd.Delivered, &dd.Sales); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		daily = append(daily, dd)
	}
	rows.Close()

	// أفضل المتاجر
	type topMerchant struct {
		Name      string `json:"name"`
		Delivered int    `json:"delivered"`
		Sales     int64  `json:"sales"`
	}
	topMerchants := []topMerchant{}
	rows, err = s.pg.Query(ctx, `
		SELECT m.name, count(*), COALESCE(sum(o.total), 0)
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.status = 'delivered' AND o.created_at >= $1 AND o.created_at < $2
		GROUP BY m.id, m.name ORDER BY sum(o.total) DESC LIMIT 5`, from, toEnd)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for rows.Next() {
		var t topMerchant
		if err := rows.Scan(&t.Name, &t.Delivered, &t.Sales); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		topMerchants = append(topMerchants, t)
	}
	rows.Close()

	// أفضل السائقين
	type topDriver struct {
		Name      string `json:"name"`
		Phone     string `json:"phone"`
		Delivered int    `json:"delivered"`
		Cash      int64  `json:"cash"`
	}
	topDrivers := []topDriver{}
	rows, err = s.pg.Query(ctx, `
		SELECT COALESCE(u.full_name, ''), u.phone, count(*), COALESCE(sum(o.cash_due), 0)
		FROM orders o JOIN users u ON u.id = o.driver_id
		WHERE o.status = 'delivered' AND o.created_at >= $1 AND o.created_at < $2
		GROUP BY u.id, u.full_name, u.phone ORDER BY count(*) DESC LIMIT 5`, from, toEnd)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var t topDriver
		if err := rows.Scan(&t.Name, &t.Phone, &t.Delivered, &t.Cash); err != nil {
			s.respondErr(w, err)
			return
		}
		topDrivers = append(topDrivers, t)
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": from.Format("2006-01-02"), "to": to.Format("2006-01-02"),
		"summary": summary, "daily": daily,
		"top_merchants": topMerchants, "top_drivers": topDrivers,
	})
}
