package orders

import (
	"context"
	"time"
)

// lastStopPointSQL **آخرُ محطّةٍ استُلمت** في طلبٍ بعدّة محطّات — وNULL لما سواه. على الاسم `o`.
const lastStopPointSQL = `(SELECT lm.location FROM order_parts lp JOIN merchants lm ON lm.id = lp.merchant_id
	WHERE lp.order_id = o.id AND o.stops > 1 AND lp.status = 'picked_up'
	ORDER BY lp.picked_up_at DESC NULLS LAST LIMIT 1)`

// legPoint نقطةٌ في طريق.
type legPoint struct{ lat, lng float64 }

// sumLegSeconds **زمنُ طريقٍ بمحطّاته = مجموعُ سيقانه** — وساقٌ لم تُقَس تُسقط القياسَ كلَّه:
// **رقمٌ ناقصٌ يتّهم سائقاً بتأخيرٍ لم يقع.**
func sumLegSeconds(ctx context.Context, r RouteReader, pts []legPoint) int {
	if r == nil || len(pts) < 2 {
		return 0
	}
	total := 0
	for i := 1; i < len(pts); i++ {
		sec := r.Seconds(ctx, pts[i-1].lat, pts[i-1].lng, pts[i].lat, pts[i].lng)
		if sec <= 0 {
			return 0
		}
		total += sec
	}
	return total
}

// captureStopsToStoreETA **زمنُ طريق السائق إلى المتجر في طلبٍ بعدّة محطّات** (D12): من موضعه إلى
// المحطّة الأولى ثمّ ما بعدها بترتيبها، **حتّى آخر محطّةٍ حيّة** — والتسليمُ عند نقطة الطارئ.
// **ويردّ كاذباً لطلب المتجر الواحد** فيُقاس كما كان.
func (s *Service) captureStopsToStoreETA(orderID string) bool {
	if s.db == nil {
		return false
	}
	var stops int
	if err := s.db.QueryRow(context.Background(), `SELECT stops FROM orders WHERE id = $1`, orderID).
		Scan(&stops); err != nil || stops <= 1 {
		return false
	}
	if s.router == nil {
		return true
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), routeETATimeout)
		defer cancel()
		rows, err := s.db.Query(ctx, `
			SELECT lat, lng FROM (
			    SELECT 0 AS k, ST_Y(du.last_location::geometry) AS lat, ST_X(du.last_location::geometry) AS lng
			    FROM orders o JOIN users du ON du.id = o.driver_id
			    WHERE o.id = $1::uuid AND du.last_location IS NOT NULL
			      AND du.last_location_at > now() - interval '15 minutes'
			    UNION ALL
			    SELECT p.seq, ST_Y(pt::geometry), ST_X(pt::geometry)
			    FROM order_parts p JOIN orders o ON o.id = p.order_id
			    JOIN merchants m ON m.id = p.merchant_id
			    CROSS JOIN LATERAL (SELECT CASE WHEN p.status = 'picked_up'
			                                    THEN COALESCE(o.pickup_override, m.location)
			                                    ELSE m.location END AS pt) x
			    WHERE p.order_id = $1::uuid AND x.pt IS NOT NULL
			      AND (p.status = 'accepted' OR (p.status = 'picked_up' AND p.picked_up_by IS DISTINCT FROM o.driver_id))
			) q ORDER BY k`, orderID)
		if err != nil {
			return
		}
		var pts []legPoint
		for rows.Next() {
			var p legPoint
			if rows.Scan(&p.lat, &p.lng) == nil {
				pts = append(pts, p)
			}
		}
		rows.Close()
		sec := sumLegSeconds(ctx, s.router, pts)
		if sec <= 0 {
			return
		}
		if _, err := s.db.Exec(ctx, `UPDATE orders SET to_store_eta_sec = $2 WHERE id = $1::uuid`,
			orderID, sec); err != nil && s.logger != nil {
			s.logger.Warn("route eta: حفظ المحطّات", "error", err, "order", orderID)
		}
	}()
	return true
}

// stopsStageExtra **مهلةُ «في الطريق للمتجر» تزيد بكلّ محطّةٍ بعد الأولى** (D12): وقفةُ تسليمٍ
// عند كلّ متجر = `orders.handover_timeout_min`. **وطلبُ المتجر الواحد لا يُمسّ.**
func (o *Order) stopsStageExtra(handoverMin int) time.Duration {
	if o.Stops <= 1 || handoverMin <= 0 {
		return 0
	}
	live := 0
	for _, p := range o.Parts {
		switch p.Status {
		case PartPending, PartAccepted, PartPickedUp:
			live++
		}
	}
	if len(o.Parts) == 0 {
		live = o.Stops
	}
	if live <= 1 {
		return 0
	}
	return time.Duration(live-1) * time.Duration(handoverMin) * time.Minute
}
