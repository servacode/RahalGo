package orders

// ══════════════════════════════════════════════════════════════════════
// **موقعُ المستلِم يصل بعد إرسال «لدي توصيلة»** (بلاغُ المالك ٢٠٢٦-١١-٢٧)
// ══════════════════════════════════════════════════════════════════════
//
// المتجرُ ضغط «اطلب الموقع من المستلم» ثمّ أرسل التوصيلةَ قبل أن يردّ المستلِم — **فبقيت بلا نقطة
// ولم تصل الكابتنَ أبداً**: لا شيءَ كان يربط طلبَ الموقع بالتوصيلة. **والآن يُربط عند الإنشاء**
// (`server.linkMerchantDeliveryLocationTx`) **وحين يصل الموقعُ يُكتب هنا** ويُبثّ للكابتن والمتجر والمكتب.
//
// **والأجرةُ لا تتغيّر** — حُسبت عند الإنشاء من موقع المتجر وعليها وافق المتجر، **ولا مالَ يتبدّل
// بصمت.** النقطةُ وحدَها تُكتب (`dropoff` و`dropoff_known`).
//
// **وفي «توصيل» لحدّ التسليم، وفي «إحضار» لحدّ الاستلام** — النقطةُ هناك مكانُ الاستلام، وبعده
// الكابتنُ متّجهٌ إلى المتجر.

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// SetMerchantDeliveryPointTx **يكتب نقطةَ المستلِم في توصيلة متجرٍ بلا نقطة** — في معاملة من يناديه،
// ولا بثّ. **ونقطةٌ خارجَ التغطية لا تُكتب** (الكابتنُ يتّصل بالمستلِم كما كان) ولا تُسقط شيئاً.
// يردّ أكُتبت.
func (s *Service) SetMerchantDeliveryPointTx(ctx context.Context, q dbtx.Querier, orderID string, lat, lng float64) (bool, error) {
	if !ValidPoint(lat, lng) {
		return false, ErrBadPoint
	}
	if err := s.requirePlaceLaunched(ctx, q, lat, lng); err != nil {
		return false, err
	}
	if _, err := s.RequireServiceable(ctx, q, lat, lng); err != nil {
		return false, err
	}
	tag, err := q.Exec(ctx, `
		UPDATE orders
		   SET dropoff = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, dropoff_known = true,
		       updated_at = now()
		 WHERE id = $1 AND kind = 'merchant_delivery' AND NOT dropoff_known
		   AND closed_at IS NULL AND delivered_at IS NULL
		   AND (delivery_mode <> 'fetch' OR picked_up_at IS NULL)`, orderID, lat, lng)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// ApplyMerchantDeliveryShared **المستلِمُ أرسل موقعَه بعد إنشاء التوصيلة** — يُكتب ويُبثّ لغرف الطلب،
// ويُخبَر الكابتنُ المُسنَد: «وصل موقع المستلم — صار على الخريطة». يردّ أكُتبت.
func (s *Service) ApplyMerchantDeliveryShared(ctx context.Context, orderID string, lat, lng float64) (bool, error) {
	ok, err := s.SetMerchantDeliveryPointTx(ctx, s.db, orderID, lat, lng)
	if err != nil || !ok {
		return ok, err
	}
	o, err := s.GetByID(ctx, orderID)
	if err != nil {
		return true, nil
	}
	s.publishOrder(o)
	if s.notify != nil && o.DriverID != nil {
		title := "وصل موقع المستلم — صار على الخريطة"
		if o.DeliveryMode == DeliveryModeFetch {
			title = "وصل موقع الاستلام — صار على الخريطة"
		}
		s.notify.Notify(ctx, notifications.Input{
			UserID: *o.DriverID, Kind: notifications.KindOrder,
			Title:  title,
			Body:   "#" + itoa(o.Number),
			Entity: "order", EntityID: o.ID, Href: "/portal",
			Apps: []string{notifications.AppDriver},
		})
	}
	return true, nil
}
