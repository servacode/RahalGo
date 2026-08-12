package orders

import (
	"context"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
//
//	**زمنُ الطريق من الخريطة — يُلتقط في لحظته ويُجمَّد**
//
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «يجب أن يكون لدينا زمنٌ افتراضيٌّ تقريبيّ،
//
//	وعلامةُ الإكس — أكيد ما رح نعدم السائق أو المتجر، ولكن حركةٌ ذكيّةٌ
//	للمنصّة فقط».)
//
// # وواجهةٌ ضيّقةٌ لا حزمةُ توجيه
//
// **هذا المحرّكُ يسأل سؤالاً واحداً**: كم ثانيةً بين نقطتين على الشارع؟
// **ولا يعنيه رسمُ الطريق ولا خطواتُه ولا بدائلُه.** وحقنُ الحزمة كلِّها
// يفتح باباً لقراءاتٍ لا تخصّه.
//
// # ولا يوقف إسناداً
//
// **نداءٌ خارجيٌّ في طريق الإسناد يجعل تعطّلَ الخريطة تعطّلاً في
// المنصّة**: سائقٌ يضغط «أقبل» فينتظر مهلةَ اتّصالٍ ثمّ يُردّ بخطأ.
//
// **فيقع بعد أن يتمّ كلُّ شيء، في خيطٍ مستقلٍّ بمهلته** — وما يعود به
// يُكتب في صفٍّ قائمٍ أصلاً. **وإن سقط بقي العمودُ فارغاً**، وبقي الخطُّ
// بلا علامة: **لا أحدَ يُوسَم بتأخيرٍ لم يُقَس.**

// RouteReader ما يحتاجه المحرّكُ من الخريطة — **سؤالٌ واحد.**
//
// **وصفرٌ يعني «لا يُعرف»** — لا «فوريّ»: نقطتان بلا طريقٍ بينهما،
// أو محرّكُ خرائطَ متوقّف. **والجهلُ ليس سرعة.**
type RouteReader interface {
	Seconds(ctx context.Context, fromLat, fromLng, toLat, toLng float64) int
}

// SetRouter يحقن قارئَ الخريطة — **يُنادى مرّةً عند الإقلاع.**
func (s *Service) SetRouter(r RouteReader) { s.router = r }

// routeETATimeout **مهلةُ السؤال** — والخريطةُ على استضافةٍ قد تنام.
//
// **وطولُها لا يؤذي**: الخيطُ مستقلٌّ ولا ينتظره أحد. **وقصرُها يضيّع
// قياساً** حين تكون الخدمةُ تستيقظ.
const routeETATimeout = 20 * time.Second

// captureToStoreETA يلتقط زمنَ طريق السائق إلى المتجر عند إسناده.
//
// **ومن موضعه الآن** — لا من موضعه حين يُسأل بعد يومين.
func (s *Service) captureToStoreETA(orderID string) {
	s.captureETA(orderID, "to_store_eta_sec", `
		SELECT ST_Y(du.last_location::geometry), ST_X(du.last_location::geometry),
		       ST_Y(COALESCE(o.pickup_override, mr.location)::geometry),
		       ST_X(COALESCE(o.pickup_override, mr.location)::geometry)
		FROM orders o
		JOIN users du ON du.id = o.driver_id
		LEFT JOIN merchants mr ON mr.id = o.merchant_id
		WHERE o.id = $1::uuid
		  AND du.last_location IS NOT NULL
		  AND du.last_location_at > now() - interval '15 minutes'
		  AND COALESCE(o.pickup_override, mr.location) IS NOT NULL`)
}

// captureToDoorETA يلتقط زمنَ الطريق من المتجر إلى باب الزبون عند الاستلام.
func (s *Service) captureToDoorETA(orderID string) {
	s.captureETA(orderID, "to_door_eta_sec", `
		SELECT ST_Y(COALESCE(o.pickup_override, mr.location)::geometry),
		       ST_X(COALESCE(o.pickup_override, mr.location)::geometry),
		       ST_Y(o.dropoff::geometry), ST_X(o.dropoff::geometry)
		FROM orders o
		LEFT JOIN merchants mr ON mr.id = o.merchant_id
		WHERE o.id = $1::uuid
		  AND COALESCE(o.pickup_override, mr.location) IS NOT NULL
		  AND o.dropoff IS NOT NULL`)
}

// captureETA يسأل الخريطةَ عن طريقٍ ويحفظ ثوانيَه في عمودٍ من الطلب.
//
// **والعمودُ يُلصَق بالنصّ لا يُمرَّر وسيطا** — أسماءُ الأعمدة لا تُعطى
// كوسائط في SQL، **وهي هنا ثابتتان في هذا الملفّ لا تأتيان من نداءٍ
// خارجيّ.**
func (s *Service) captureETA(orderID, column, pointsSQL string) {
	if s.router == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), routeETATimeout)
		defer cancel()

		var fromLat, fromLng, toLat, toLng float64
		if err := s.db.QueryRow(ctx, pointsSQL, orderID).
			Scan(&fromLat, &fromLng, &toLat, &toLng); err != nil {
			// **ولا سجلَّ خطأٍ لغياب موضع** — سائقٌ أغلق تتبّعَه أمرٌ
			// عاديّ، **وسجلٌّ يمتلئ بالعاديّ يُخفي غيرَ العاديّ.**
			return
		}
		sec := s.router.Seconds(ctx, fromLat, fromLng, toLat, toLng)
		if sec <= 0 {
			return
		}
		if _, err := s.db.Exec(ctx,
			`UPDATE orders SET `+column+` = $2 WHERE id = $1::uuid`,
			orderID, sec); err != nil {
			s.logger.Warn("route eta: حفظ", "error", err, "order", orderID)
		}
	}()
}
