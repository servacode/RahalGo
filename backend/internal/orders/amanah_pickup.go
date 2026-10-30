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
