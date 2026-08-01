package orders

import (
	"context"
	"time"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// Alert تنبيه تصعيد لطلب عالق — يظهر أحمر في غرفة العمليات.
type Alert struct {
	OrderID       string  `json:"order_id"`
	Number        int64   `json:"number"`
	Status        string  `json:"status"`
	MerchantName  string  `json:"merchant_name"`
	CustomerPhone string  `json:"customer_phone"`
	Reason        string  `json:"reason"` // no_accept | no_driver | too_long
	Minutes       float64 `json:"minutes"`
}

// Alerts يفحص الطلبات العالقة وفق المهل الديناميكية (PLAN §6.1).
func (s *Service) Alerts(ctx context.Context) ([]Alert, error) {
	rows, err := s.db.Query(ctx, `
		WITH t AS (
			SELECT
				COALESCE((SELECT (value#>>'{}')::float8 FROM app_settings WHERE key='orders.accept_timeout_min'), 5)   AS accept_min,
				COALESCE((SELECT (value#>>'{}')::float8 FROM app_settings WHERE key='orders.driver_timeout_min'), 10)  AS driver_min,
				COALESCE((SELECT (value#>>'{}')::float8 FROM app_settings WHERE key='orders.delivery_timeout_min'), 60) AS delivery_min
		)
		SELECT o.id, o.number, o.status, m.name, cu.phone,
			CASE
				WHEN o.status = 'pending' AND o.created_at < now() - make_interval(mins => t.accept_min::int)
					THEN 'no_accept'
				WHEN o.status IN ('preparing','dispatching') AND o.updated_at < now() - make_interval(mins => t.driver_min::int)
					THEN 'no_driver'
				ELSE 'too_long'
			END AS reason,
			round(EXTRACT(EPOCH FROM now() - o.created_at) / 60) AS minutes
		FROM orders o
		CROSS JOIN t
		JOIN merchants m ON m.id = o.merchant_id
		JOIN users cu ON cu.id = o.customer_id
		WHERE o.closed_at IS NULL AND (
			(o.status = 'pending' AND o.created_at < now() - make_interval(mins => t.accept_min::int)) OR
			(o.status IN ('preparing','dispatching') AND o.updated_at < now() - make_interval(mins => t.driver_min::int)) OR
			(o.created_at < now() - make_interval(mins => t.delivery_min::int))
		)
		ORDER BY o.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := []Alert{}
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.OrderID, &a.Number, &a.Status, &a.MerchantName,
			&a.CustomerPhone, &a.Reason, &a.Minutes); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

// RunWatchdog حلقة الراصد: فحص دوري وبث التنبيهات لغرفة العمليات.
// يبث فقط عند تغير مجموعة التنبيهات — لا إزعاج متكرراً بلا جديد.
//
// **والبثّ يصل الشاشات المفتوحة وحدها.** فمن أغلق اللوحة ليلاً لم يصله شيء،
// والطلب يبقى عالقاً حتى الصباح. فصار الراصد يُنشئ كذلك **إشعاراً باقياً**
// يجده الموظّف حين يفتح، مرّةً واحدة لكل طلب (`alerted_at` علامةٌ في القاعدة
// لا في الذاكرة: ذاكرةٌ تُفقد بإعادة التشغيل فيُعاد إنذار الجميع دفعةً).
func (s *Service) RunWatchdog(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastKey := ""
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// **انتقالُ الدور مع نبضة الراصد** — لا مع نداءِ سائقٍ للطابور:
			// لو انتظرنا من يسأل لبقي طلبٌ محجوزاً لسائقٍ نائمٍ حتى يفتح
			// غيرُه التطبيق. **والزبونُ لا ينتظر أن يتذكّر أحدٌ أن ينظر.**
			s.SweepExpiredOffers(ctx)
			alerts, err := s.Alerts(ctx)
			if err != nil {
				s.logger.Error("watchdog scan failed", "error", err)
				continue
			}
			key := ""
			for _, a := range alerts {
				key += a.OrderID + a.Reason + "|"
			}
			if key == lastKey {
				continue
			}
			lastKey = key
			s.pub.Publish("ops", map[string]any{"type": "alerts", "alerts": alerts})
			if len(alerts) > 0 {
				s.logger.Warn("watchdog escalation", "count", len(alerts))
				s.escalate(ctx, alerts)
			}
		}
	}
}

// escalate يُنشئ إشعاراً باقياً لكل طلبٍ عَلِق ولم يُنذَر بعد.
//
// والعلامة تُوضع **قبل** الإشعار لا بعده: لو أُشعر ثم فشلت الكتابة لأُعيد
// الإشعار كل ثلاثين ثانية إلى الأبد. وإشعارٌ ضائع أهون من إشعارٍ يتكرّر
// مئتي مرّة — الثاني يُدرَّب المستخدم على تجاهله فيصير كالصمت.
func (s *Service) escalate(ctx context.Context, alerts []Alert) {
	if s.notify == nil {
		return
	}
	for _, a := range alerts {
		tag, err := s.db.Exec(ctx,
			`UPDATE orders SET alerted_at = now() WHERE id = $1 AND alerted_at IS NULL`, a.OrderID)
		if err != nil {
			s.logger.Error("watchdog: تعذّر وسم الإنذار", "order", a.OrderID, "error", err)
			continue
		}
		if tag.RowsAffected() == 0 {
			continue // أُنذر سابقاً
		}
		s.notify.NotifyOps(ctx, notifications.Input{
			Kind:     notifications.KindOrder,
			Title:    alertTitles[a.Reason],
			Body:     a.MerchantName,
			Entity:   "order",
			EntityID: a.OrderID,
			Href:     "/dashboard/orders",
		})
	}
}

// alertTitles نصوص التصعيد — مصدرٌ واحد بجانب بقية نصوص الإشعارات.
var alertTitles = map[string]string{
	"no_accept": "طلبٌ لم يقبله متجره",
	"no_driver": "طلبٌ بلا سائق",
	"too_long":  "طلبٌ تأخّر عن موعده",
}
