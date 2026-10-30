package orders

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
)

// AmanahPickupNote **ما يقرؤه السائقُ عنواناً للاستلام** في الأمانة.
const AmanahPickupNote = "موقع الأمانة"

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
	tag, err := s.db.Exec(ctx, `
		UPDATE orders
		   SET pickup_override = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
		       pickup_override_note = $4
		 WHERE id = $1 AND kind = 'custom' AND custom_mode = 'amanah'
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
