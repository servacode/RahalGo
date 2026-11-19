package orders

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// AmanahPickupNote **ما يقرؤه السائقُ عنواناً للاستلام** في الأمانة.
const AmanahPickupNote = "موقع الأمانة"

// اتّجاهُ الأمانة — `orders.amanah_direction` (قرارُ المالك ٢٠٢٦-١١-١٩).
const (
	AmanahReceive = "receive"
	AmanahSend    = "send"
)

// AmanahSenderNote **عنوانُ الاستلام في «أبعت أمانة»** — عنوانُ الزبون المرسِل.
const AmanahSenderNote = "عنوان المرسل"

// SetAmanahSendTx **«بدّي أبعت أمانة»** (قرارُ المالك ٢٠٢٦-١١-١٩) — في معاملة الإنشاء: الاتّجاهُ،
// **والاستلامُ من عنوان الزبون** (`pickup_override` — نقطةُ الطلب نفسُها، فُحصت تغطيتُها عند الإنشاء)،
// **والتسليمُ لموقع المستلِم**: وصل قبل الإرسال؟ يُكتب الآن. وإلّا `dropoff_known = false` حتّى يصل
// (`ApplyAmanahShared`) — فلا يُوجَّه الكابتنُ إلى عنوان المرسل ظنّاً أنّه المستلم.
func (s *Service) SetAmanahSendTx(ctx context.Context, q dbtx.Querier, orderID string, recipLat, recipLng *float64) (bool, error) {
	if _, err := q.Exec(ctx, `
		UPDATE orders
		   SET amanah_direction = 'send', pickup_override = dropoff, pickup_override_note = $2,
		       dropoff_known = false
		 WHERE id = $1 AND kind = 'custom' AND custom_mode = 'amanah'`, orderID, AmanahSenderNote); err != nil {
		return false, err
	}
	if recipLat == nil || recipLng == nil || !ValidPoint(*recipLat, *recipLng) {
		return false, nil
	}
	// **وموقعُ مستلِمٍ برّا التغطية لا يُسقط الطلب** — يبقى «غيرَ معروف» والكابتنُ يتّصل به.
	if err := s.requirePlaceLaunched(ctx, q, *recipLat, *recipLng); err != nil {
		return false, nil
	}
	if _, err := s.RequireServiceable(ctx, q, *recipLat, *recipLng); err != nil {
		return false, nil
	}
	_, err := q.Exec(ctx, `
		UPDATE orders SET dropoff = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, dropoff_known = true
		 WHERE id = $1`, orderID, *recipLat, *recipLng)
	return err == nil, err
}

// SetAmanahPickupTx **موقعُ الأمانة على الخريطة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — يلغي قرارَ
// ٢٠٢٦-١٠-١٦ «لا خانات استلام، السائق يسأل بالدردشة».
//
// **الزبونُ يحدّد النقطةَ والسائقُ يراها على خريطته** — فتُكتب في `pickup_override`
// نفسِه الذي يقرؤه كلُّ ما يحسب نقطةَ الاستلام (طابورُ السائق والملاحةُ والتوزيعُ
// بالقرب)، **ولا عمودَ ثانٍ يُعلَّم كلُّ قارئٍ به.**
//
// **وتُفحَص كنقطة التسليم**: صحّةٌ ثمّ محافظةٌ مُطلَقةٌ ثمّ تغطية — **فلا يُطلب
// سائقٌ ليستلم من مدينةٍ لا خدمةَ فيها.** وفي معاملة الإنشاء نفسِها.
func (s *Service) SetAmanahPickupTx(ctx context.Context, q dbtx.Querier, orderID string, lat, lng float64) error {
	if !ValidPoint(lat, lng) {
		return ErrBadPoint
	}
	if err := s.requirePlaceLaunched(ctx, q, lat, lng); err != nil {
		return err
	}
	if _, err := s.RequireServiceable(ctx, q, lat, lng); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `
		UPDATE orders
		   SET pickup_override = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
		       pickup_override_note = $4
		 WHERE id = $1 AND kind = 'custom' AND custom_mode = 'amanah'`,
		orderID, lat, lng, AmanahPickupNote)
	return err
}

// ApplyAmanahShared **صاحبُ الأمانة شارك موقعَه بعد إنشاء الطلب** (قرارُ المالك ٢٠٢٦-١٠-٣٠).
//
// الزبونُ طلب الموقعَ منه («اطلب الموقع منه») ثمّ أرسل الطلبَ قبل أن يصل — **فالنقطةُ تُكتب
// في الطلب حين تصل** وتُبثّ لغرفة السائق والزبون والمكتب، فتظهر على خريطة السائق بلا تحديث.
//
// **وبعد الاستلام أو الإقفال لا تُمسّ** — الأمانةُ صارت في يد السائق. ويردّ أكُتبت.
func (s *Service) ApplyAmanahShared(ctx context.Context, orderID string, lat, lng float64) (bool, error) {
	if !ValidPoint(lat, lng) {
		return false, ErrBadPoint
	}
	if err := s.requirePlaceLaunched(ctx, s.db, lat, lng); err != nil {
		return false, err
	}
	if _, err := s.RequireServiceable(ctx, s.db, lat, lng); err != nil {
		return false, err
	}
	// **«أبعت أمانة»: الموقعُ الواصلُ موقعُ المستلِم** (قرارُ المالك ٢٠٢٦-١١-١٩) — نقطةُ التسليم، **ولحدّ
	// التسليم** لا الاستلام: الكابتنُ يحمل الأمانةَ وينتظره.
	tag, err := s.db.Exec(ctx, `
		UPDATE orders
		   SET dropoff = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, dropoff_known = true,
		       updated_at = now()
		 WHERE id = $1 AND kind = 'custom' AND custom_mode = 'amanah' AND amanah_direction = 'send'
		   AND delivered_at IS NULL AND closed_at IS NULL`, orderID, lat, lng)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 1 {
		if o, err := s.GetByID(ctx, orderID); err == nil {
			s.publishOrder(o)
			if s.notify != nil && o.DriverID != nil {
				s.notify.Notify(ctx, notifications.Input{
					UserID: *o.DriverID, Kind: notifications.KindOrder,
					Title:  "وصل موقع مستلم الأمانة — صار على الخريطة",
					Body:   "#" + itoa(o.Number),
					Entity: "order", EntityID: o.ID, Href: "/portal",
					Apps: []string{notifications.AppDriver},
				})
			}
		}
		return true, nil
	}
	tag, err = s.db.Exec(ctx, `
		UPDATE orders
		   SET pickup_override = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
		       pickup_override_note = $4
		 WHERE id = $1 AND kind = 'custom' AND custom_mode = 'amanah' AND amanah_direction = 'receive'
		   AND picked_up_at IS NULL AND closed_at IS NULL`,
		orderID, lat, lng, AmanahPickupNote)
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
