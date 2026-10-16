package orders

// ══════════════════════════════════════════════════════════════════════
// **«الكابتن قريب منك» — مرّةً واحدةً لكلّ طلب** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// **المالكُ يريد الكابتنَ عند الزبون بالأمتار** — والطريقُ وحدَه لا يكفي: زبونٌ
// في الطابق الرابع لا يرى الدرّاجةَ تحت البيت. **فحين يدخل الكابتنُ حدَّ
// `orders.near_notice_m` من الوجهة يُقال للزبون أن يطلع إلى الباب أو الشبّاك.**
//
// # والوجهةُ وجهةُ الطريق نفسُها
//
// **موقعُ الزبون الدقيقُ إن تحوّل الطريقُ إليه، وإلّا الدبّوس** — بالحكم الواحد
// (`LivePoint.Switch`). **فلا يُنبَّه الزبونُ «قريبٌ منك» والكابتنُ قريبٌ من
// دبّوسٍ خطأ لا منه.**
//
// # ومتى
//
// **بعد الاستلام وحدَه** — قبله الكابتنُ عند المتجر، **وقربُه من البيت مصادفةُ
// طريقٍ لا وصول.** **ولأنواع الزبون كلِّها** (`liveKinds`): العاديّ والمشتريات
// والأمانة — **و«لدي توصيلة» خارجها**: مستلِمُها ليس على التطبيق.
//
// # ومرّةً واحدة
//
// **علامةٌ في الطلب نفسِه** (`0451_order_near_notice.sql`) **تُكتب بشرطها** —
// فنبضتان تصلان معاً لا تُرنّان مرّتين.

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// nearStatuses **ما بعد الاستلام** — الكابتنُ في طريقه إلى الزبون.
var nearStatuses = []string{StPickedUp, StOnTheWay, StAtDropoff}

// NearNotice **طلبٌ دخل كابتنُه حدَّ القرب** — ومن يُنبَّه.
type NearNotice struct {
	OrderID    string
	CustomerID string
	Number     int64
}

// NearNoticeDue **ما يستحقّ التنبيهَ الآن من طلبات هذا السائق** — ويُعلَّم قبل أن يُردّ.
//
// **يُقاس من الموضع الذي كتبه السائقُ للتوّ** إلى وجهة كلّ طلبٍ في يده بعد
// الاستلام، **وما دخل الحدَّ يُعلَّم بشرط أنّه لم يُعلَّم** — فما يُردّ يُنبَّه
// صاحبُه مرّةً، **ولو نادى هذا بابان معاً.**
func (s *Service) NearNoticeDue(ctx context.Context, driverID string, lat, lng float64,
	nearM int64, rule LiveRule) ([]NearNotice, error) {
	rows, err := s.db.Query(ctx, `
		WITH me AS (SELECT ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography AS at)
		SELECT o.id::text,
		       ST_Distance(o.dropoff, me.at),
		       l.order_id IS NOT NULL,
		       l.accuracy_m::float8,
		       COALESCE(ST_Distance(l.point, o.dropoff), -1),
		       COALESCE(ST_Distance(l.point, me.at), -1)
		FROM orders o
		CROSS JOIN me
		LEFT JOIN order_live_locations l ON l.order_id = o.id
		     AND l.at > now() - make_interval(secs => $6::int)
		WHERE o.driver_id = $1::uuid AND o.closed_at IS NULL AND o.near_notice_at IS NULL
		  AND o.status = ANY($4::text[]) AND o.kind = ANY($5::text[])
		  AND o.dropoff IS NOT NULL`,
		driverID, lng, lat, nearStatuses, liveKinds, int(LiveFreshFor/time.Second))
	if err != nil {
		return nil, err
	}
	var due []string
	for rows.Next() {
		var (
			id              string
			pinDist         float64
			hasLive         bool
			accuracy        *float64
			pinGap, liveDst float64
		)
		if err := rows.Scan(&id, &pinDist, &hasLive, &accuracy, &pinGap, &liveDst); err != nil {
			rows.Close()
			return nil, err
		}
		dist := pinDist
		if hasLive {
			p := &LivePoint{AccuracyM: accuracy, PinGapM: pinGap, SwitchM: rule.SwitchM,
				Precise: precise(accuracy, rule.MaxAccuracyM)}
			// **والحكمُ نفسُه الذي يحسب به الطريق** — دقيقٌ وبعيدٌ عن الدبّوس ⇒ الزبونُ الوجهة.
			if p.Switch() {
				dist = liveDst
			}
		}
		if dist <= float64(nearM) {
			due = append(due, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]NearNotice, 0, len(due))
	for _, id := range due {
		n := NearNotice{OrderID: id}
		var customer *string
		err := s.db.QueryRow(ctx, `
			UPDATE orders SET near_notice_at = now()
			WHERE id = $1::uuid AND near_notice_at IS NULL AND closed_at IS NULL
			RETURNING customer_id::text, number`, id).Scan(&customer, &n.Number)
		// **وسبقه بابٌ آخرُ إليه** — عُلّم ونُبّه هناك، فلا يُنبَّه ثانيةً.
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return out, err
		}
		if customer == nil || *customer == "" {
			continue
		}
		n.CustomerID = *customer
		out = append(out, n)
	}
	return out, nil
}
