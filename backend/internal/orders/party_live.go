package orders

// ══════════════════════════════════════════════════════════════════════
// **الطرفُ الآخر يشارك موقعَه المباشر من تطبيقنا** (قرارُ المالك ٢٠٢٦-١١-٢٨)
// ══════════════════════════════════════════════════════════════════════
//
// مستلمُ «أبعت أمانة» وصاحبُ «أستقبل أمانة» ومستلمُ توصيلة المتجر — **إذا رقمُه عليه حسابُ زبون وقال
// «نعم»**، تطبيقُه يبعت موقعَه كلَّ عشر ثوانٍ. **أوّلُ نقطةٍ تملأ طلبَ الموقع كما يملؤه ردُّ الواتساب**
// (`server.locReqReceived`)، وما بعدها هنا:
//
//   - **طرفُ التسليم** (مستلمُ «أبعت» ومستلمُ «توصيل»): يُكتب موقعاً مباشراً للطلب بنفس الجدول اللي
//     يقرؤه الكابتن (`order_live_locations` ← `LiveFor`) — **كاتبٌ ثانٍ جنب `SaveLive`** بدل ما نفتح
//     شرطَ «صاحب الطلب وحدَه». والصفُّ باسم الطرف نفسِه (`customer_id`).
//   - **طرفُ الاستلام** (صاحبُ «أستقبل» وصاحبُ «إحضار»): الكابتن رايح لعنده قبل الاستلام، وشاشتُه ما
//     بتعرض الموقع المباشر قبل الاستلام — فتنكتب نقطةُ الاستلام نفسُها (`pickup_override`، أو `dropoff`
//     بـ«إحضار» لأنّه هونيك مكانُ الاستلام) **ويُبثّ الطلب**، بلا أيّ تعديل بتطبيق الكابتن.

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// أدوارُ الطرف الآخر — من نوع الطلب واتّجاهه.
const (
	// PartyDropoff **هو نقطةُ التسليم** — لحدّ التسليم.
	PartyDropoff = "dropoff"
	// PartyPickup **هو نقطةُ الاستلام بالأمانة** (`pickup_override`) — لحدّ الاستلام.
	PartyPickup = "pickup"
	// PartyFetch **هو نقطةُ الاستلام بـ«إحضار» المتجر** (عمودُ `dropoff`) — لحدّ الاستلام.
	PartyFetch = "fetch"
)

// partyMoveM **أقلُّ حركةٍ تُعيد كتابة نقطة الاستلام** — بالمتر؛ رجفةُ الـGPS ما بتحرّك الكابتن.
const partyMoveM = 15

// SavePartyLive **يكتب آخرَ نقطةٍ لطرف التسليم** — والطلبُ ما سُلّم ولا انقفل. ويردّ سائقَ الطلب ليُنبَّه.
//
// **والتحقّقُ من أنّه الطرفُ فعلاً بالخادم قبله** (`server.partyFind`) — هون الكتابةُ وحدَها، بجملةٍ
// وحدة فلا نافذةَ ينقفل فيها الطلب بين الفحص والكتابة.
func (s *Service) SavePartyLive(ctx context.Context, orderID, partyUserID string,
	lat, lng float64, accuracyM *float64) (string, error) {
	var driverID *string
	err := s.db.QueryRow(ctx, `
		WITH o AS (
			SELECT id, driver_id FROM orders
			WHERE id = $1::uuid AND closed_at IS NULL AND delivered_at IS NULL
			  AND status <> ALL($6::text[])
		), up AS (
			INSERT INTO order_live_locations (order_id, customer_id, point, accuracy_m, at)
			SELECT o.id, $2::uuid, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5, now()
			FROM o
			ON CONFLICT (order_id) DO UPDATE
			SET customer_id = EXCLUDED.customer_id, point = EXCLUDED.point,
			    accuracy_m = EXCLUDED.accuracy_m, at = EXCLUDED.at
			RETURNING order_id
		)
		SELECT o.driver_id::text FROM o JOIN up ON up.order_id = o.id`,
		orderID, partyUserID, lng, lat, accuracyM, partyEndedStatuses).Scan(&driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrLiveInactive
	}
	if err != nil {
		return "", err
	}
	if _, err := s.db.Exec(ctx, `
		UPDATE orders SET live_first_at = now(), live_first_source = $2
		 WHERE id = $1::uuid AND live_first_at IS NULL`, orderID, LiveSourceApp); err != nil {
		s.logger.Warn("موقع الطرف الآخر: تعليمُ أوّل موقع سقط", "order", orderID, "error", err)
	}
	if driverID == nil {
		return "", nil
	}
	return *driverID, nil
}

// StopPartyLive **«إيقاف» من الطرف** — تُمحى نقطتُه هو وحدَه (لا نقطةُ غيره). ويردّ سائقَ الطلب.
func (s *Service) StopPartyLive(ctx context.Context, orderID, partyUserID string) (string, error) {
	var driverID *string
	err := s.db.QueryRow(ctx, `
		WITH d AS (
			DELETE FROM order_live_locations WHERE order_id = $1::uuid AND customer_id = $2::uuid
		)
		SELECT driver_id::text FROM orders WHERE id = $1::uuid`, orderID, partyUserID).Scan(&driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", httpx.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if driverID == nil {
		return "", nil
	}
	return *driverID, nil
}

// partyEndedStatuses **حالاتٌ انتهى فيها الطلب** — لا تُكتب فيها نقطة.
var partyEndedStatuses = []string{StDelivered, StCancelled, StFailed, StRejected, StRefunded}

// MovePartyPickup **طرفُ الاستلام تحرّك** — تُعاد كتابةُ نقطة الاستلام ويُبثّ الطلب للكابتن.
//
// **بعد أوّل نقطة وحدَها** (النقطةُ مكتوبة أصلاً — فحصُ التغطية جرى عندها)، **وقبل الاستلام**،
// **وإذا تحرّك فوق `partyMoveM`**. ويردّ أنكتبت.
func (s *Service) MovePartyPickup(ctx context.Context, orderID, role string, lat, lng float64) (bool, error) {
	if !ValidPoint(lat, lng) {
		return false, ErrBadPoint
	}
	var q string
	switch role {
	case PartyPickup:
		q = `UPDATE orders
		        SET pickup_override = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, updated_at = now()
		      WHERE id = $1 AND pickup_override IS NOT NULL
		        AND ((kind = 'custom' AND custom_mode = 'amanah') OR wa_amanah)
		        AND picked_up_at IS NULL AND closed_at IS NULL AND status <> ALL($5::text[])
		        AND ST_Distance(pickup_override, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography) > $4`
	case PartyFetch:
		q = `UPDATE orders
		        SET dropoff = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, updated_at = now()
		      WHERE id = $1 AND dropoff_known AND dropoff IS NOT NULL
		        AND kind = 'merchant_delivery' AND delivery_mode = 'fetch'
		        AND picked_up_at IS NULL AND closed_at IS NULL AND status <> ALL($5::text[])
		        AND ST_Distance(dropoff, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography) > $4`
	default:
		return false, nil
	}
	tag, err := s.db.Exec(ctx, q, orderID, lat, lng, partyMoveM, partyEndedStatuses)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if o, err := s.GetByID(ctx, orderID); err == nil {
		s.publishOrder(o)
	}
	return true, nil
}
