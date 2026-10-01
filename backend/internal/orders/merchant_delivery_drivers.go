package orders

// ══════════════════════════════════════════════════════════════════════
// **لا توصيلةَ بلا سائقٍ قريب** — قرارُ المالك ٢٠٢٦-١٠-٠١
// ══════════════════════════════════════════════════════════════════════
//
// (نصُّه: «لازم نحمي هي الخطوة إذا كانت المنصّة خارج أوقات العمل والسائقين
//  خارج أوقات العمل أيضاً» · واختار: «سائقٌ بالدوام قريبٌ من المتجر».)
//
// **كانت التوصيلةُ تُرسَل والسائقون كلُّهم خارجَ الدوام** — فيبقى المتجرُ
// ينتظر سائقاً لا وجودَ له، **والغرضُ يبرد على الرفّ.**
//
// # والقُربُ قاعدةُ محرّك التوزيع نفسُها — لا قاعدةٌ ثانية
//
// بالدوام · فاعل · **موقعُه حديث** (`drivers.location_fresh_sec`) · **وداخلَ
// أقصى نصفِ قطر التوزيع حول المتجر** (`drivers.dispatch_radius_max_m`).
// **ومن عرّف «قريب» بغير ما يعرّفه به المحرّكُ قبل توصيلةً لن تُعرض على أحد.**
//
// **والقُربُ مطفأٌ** (`drivers.proximity_enabled = false`) ⇒ **يكفي سائقٌ
// بالدوام أينما كان** — كما يوزّع المحرّكُ حينها.
//
// **ولا يُحسب النقدُ ولا عددُ الطلبات**: سائقٌ مشغولٌ الآن يفرغ بعد دقائق —
// **والسؤالُ «هل في الشارع أحد؟» لا «من يأخذها الآن؟».**

import (
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ErrNoDriversNearby **لا سائقَ بالدوام قريبٌ من المتجر الآن.**
//
// **و٥٠٣ لا ٤٠٠** — «ليس الآن» لا «طلبُك خطأ»، **كعائلة `platform_closed_now`.**
var ErrNoDriversNearby = httpx.NewError(http.StatusServiceUnavailable,
	"no_drivers_nearby", "errors.no_drivers_nearby")

// requireDriverNearby يردّ `ErrNoDriversNearby` إن لم يكن في الشارع أحد.
func (s *Service) requireDriverNearby(ctx context.Context, q dbtx.Querier, merchantID string) error {
	proximity := s.settings == nil || s.settings.GetBool(ctx, "drivers.proximity_enabled")
	fresh, maxR := int64(0), int64(0)
	if proximity {
		fresh = s.settingInt(ctx, "drivers.location_fresh_sec")
		maxR = s.settingInt(ctx, "drivers.dispatch_radius_max_m")
	}
	var ok bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1
		    FROM users u
		    JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
		    LEFT JOIN merchants m ON m.id = $1
		    WHERE u.on_shift AND u.status = 'active'
		      AND ( NOT $2
		         OR ( u.last_location IS NOT NULL
		              AND u.last_location_at > now() - make_interval(secs => $3)
		              AND ( m.location IS NULL OR $4 <= 0
		                 OR ST_DWithin(u.last_location, m.location, $4) ) ) ))`,
		merchantID, proximity, fresh, maxR).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrNoDriversNearby
	}
	return nil
}
