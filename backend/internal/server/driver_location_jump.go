package server

// ══════════════════════════════════════════════════════════════════════
// **قفزةُ موقعٍ مستحيلةٌ تُهمَل** (قرارُ المالك ٢٠٢٦-١٠-٠٣ مساءً: «أسرعُ من مركبة»)
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس في تجربة القبول**: نبضةٌ من الرقّة ثمّ نبضةٌ من دمشق بعد ثانيةٍ — **فكُتب
// السائقُ في دمشق** على خريطة العمليّات، وقربُ التوزيع يُقاس منها. **ومن زيّف موضعَه
// اختار أين يُرى.**
//
// # والحكمُ بالسرعة لا بالمسافة
//
// **ثلاثمئةُ كيلومترٍ بعد خمس ساعاتٍ مشوارٌ؛ وبعد ثانيةٍ كذبة.** فتُقسم المسافةُ على
// الزمن بين النقطتين **بوقت التقاطهما** — وأسرعُ من `maxPlausibleMps` تُهمَل.
//
// # ولا تُحبس الحقيقةُ خلف كذبةٍ قديمة
//
//   - **حدٌّ أدنى للمسافة** (`jumpMinMeters`): رجفةُ القمر مئةُ مترٍ في ثانية — سرعةٌ
//     «مستحيلةٌ» على الورق وليست قفزة.
//   - **ونافذةٌ زمنيّة** (`jumpWindow`): بعدها تُقبل النقطةُ أيّاً كانت — فموضعٌ مكتوبٌ
//     خطأً (أوّلُ قراءةٍ بعد نفقٍ) لا يحبس السائقَ ما دام يعمل.
//
// # والردُّ نجاحٌ بلا كتابة
//
// **التطبيقُ الذي يتلقّى خطأً يعيد المحاولةَ إلى الأبد** بنقطةٍ لن تُقبل — فيُردّ ٢٠٠
// ولا يُكتب الموضعُ ولا الأثر، **ويُسجَّل في سجلّ الخادم** ليُرى من يقفز.

import (
	"context"
	"time"

	"github.com/servacode/rahalgo/backend/internal/routing"
)

const (
	// maxPlausibleMps **أسرعُ ما يُصدَّق** — ٧٠ م/ث ≈ ٢٥٠ كم/س: لا درّاجةَ ولا سيّارةَ
	// في مدينة.
	maxPlausibleMps = 70.0
	// jumpMinMeters **أقصرُ قفزةٍ تُحسب** — دونها رجفةُ قمرٍ لا قفزة.
	jumpMinMeters = 1000.0
	// jumpWindow **بعدها لا يُقاس على الموضع المكتوب** — شاخ فلا يحكم.
	jumpWindow = 10 * time.Minute
)

// lastFix **آخرُ موضعٍ مكتوبٍ ووقتُه** — وغيابُه `ok=false`.
type lastFix struct {
	lat, lng float64
	at       time.Time
	ok       bool
}

func (s *Server) lastFixOf(ctx context.Context, driverID string) lastFix {
	var lat, lng *float64
	var at *time.Time
	if err := s.pg.QueryRow(ctx, `
		SELECT ST_Y(last_location::geometry), ST_X(last_location::geometry), last_location_at
		FROM users WHERE id = $1`, driverID).Scan(&lat, &lng, &at); err != nil ||
		lat == nil || lng == nil || at == nil {
		return lastFix{}
	}
	return lastFix{lat: *lat, lng: *lng, at: *at, ok: true}
}

// impossibleJump **أتقفز هذه النقطةُ من تلك أسرعَ من مركبة؟**
func impossibleJump(from lastFix, lat, lng float64, at time.Time) bool {
	if !from.ok {
		return false
	}
	dt := at.Sub(from.at)
	if dt < 0 {
		dt = -dt
	}
	if dt > jumpWindow {
		return false
	}
	d := routing.MetersBetween(routing.Point{Lat: from.lat, Lng: from.lng}, routing.Point{Lat: lat, Lng: lng})
	if d < jumpMinMeters {
		return false
	}
	secs := dt.Seconds()
	if secs < 1 {
		secs = 1
	}
	return d/secs > maxPlausibleMps
}

// logJump **يُسجَّل من قفز** — في سجلّ الخادم لا في القاعدة.
func (s *Server) logJump(driverID string, from lastFix, lat, lng float64, at time.Time) {
	s.logger.Warn("الموقع: قفزةٌ مستحيلةٌ أُهملت", "driver", driverID,
		"from_lat", from.lat, "from_lng", from.lng, "to_lat", lat, "to_lng", lng,
		"seconds", at.Sub(from.at).Seconds())
}
