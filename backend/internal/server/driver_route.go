package server

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/routing"
)

// ══════════════════════════════════════════════════════════════════════
// **مسارُ الطلب — بالشوارع لا بالهواء**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢.)
//
// # ولماذا نداءٌ وحدَه لا حقلٌ في قائمة الطلبات
//
// **قائمةُ العروض تُقرأ كلَّ ثوانٍ وفيها خمسةُ طلبات** — ومسارٌ لكلِّ واحدٍ
// منها خمسةُ نداءاتٍ في كلّ تحديث، **لسائقٍ لم يقرّر بعد أيَّها يأخذ.**
// والمسافةُ الهوائيّةُ تكفي للموازنة بينها: **أيُّها أقرب** لا كم بالضبط.
//
// **والمسارُ يلزم لواحدٍ فقط**: الذي في يده الآن — يُرسم خطُّه ويُقال وقتُه.
//
// # والطورُ يقرّر الطرفين
//
// **قبل الاستلام**: من موضعه إلى المتجر. **وبعده**: من المتجر إلى الزبون.
// **ومن رسم الطرفين معا** أعطاه خطّاً يمرّ بمكانٍ لا شأنَ له به الآن.
//
// # والذاكرةُ تقصر عمرَ الجواب لا الطريق
//
// **الطريقُ لا يتبدّل في عشر دقائق** — والسائقُ يتحرّك. **فالمفتاحُ نقطتان
// مقرّبتان** إلى نحو عشرة أمتار: من مشى خطوةً قرأ الجوابَ المخزّن، **ومن
// مشى شارعاً سأل من جديد.**
//
// **وبلا هذا يُسأل المحرّكُ مع كلّ نبضة موضع** — وهو ما يجعل خدمةً مجّانيّةً
// تختنق، **ولا يعطي دقّةً**: عشرةُ أمتارٍ لا تغيّر طريقاً.

const routeCacheTTL = 10 * time.Minute

// handleDriverOrderRoute يردّ خطَّ الطريق ومسافتَه ومدّتَه للطرف الحاليّ.
func (s *Server) handleDriverOrderRoute(w http.ResponseWriter, r *http.Request) {
	var (
		status           string
		fromLat, fromLng *float64
		pickLat, pickLng float64
		dropLat, dropLng float64
	)
	err := s.pg.QueryRow(r.Context(), `
		SELECT o.status,
		       ST_Y(du.last_location::geometry), ST_X(du.last_location::geometry),
		       -- **ونقطةُ الاستلام قد تكون بديلةً** (بضاعةٌ مع سائقٍ وقع له
		       -- طارئ) — فيُمشى إليها لا إلى المتجر.
		       ST_Y(COALESCE(o.pickup_override, m.location)::geometry),
		       ST_X(COALESCE(o.pickup_override, m.location)::geometry),
		       ST_Y(o.dropoff::geometry), ST_X(o.dropoff::geometry)
		FROM orders o
		LEFT JOIN merchants m ON m.id = o.merchant_id
		LEFT JOIN users du ON du.id = o.driver_id
		WHERE o.id = $1::uuid AND o.driver_id = $2::uuid`,
		chi.URLParam(r, "id"), userIDFrom(r)).
		Scan(&status, &fromLat, &fromLng, &pickLat, &pickLng, &dropLat, &dropLng)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **وموضعٌ لم يصل بعدُ يُقاس من نقطة الاستلام** — لا يُردّ فراغا:
	// **سائقٌ فتح التطبيق للتوّ** يستحقّ أن يرى طريقَ المشوار كلَّه.
	picked := status != "assigned" && status != "at_pickup"
	from := routing.Point{Lat: pickLat, Lng: pickLng}
	if !picked && fromLat != nil && fromLng != nil {
		from = routing.Point{Lat: *fromLat, Lng: *fromLng}
	}
	to := routing.Point{Lat: pickLat, Lng: pickLng}
	if picked {
		to = routing.Point{Lat: dropLat, Lng: dropLng}
	}

	route, err := s.routeCached(r.Context(), from, to)
	if err != nil {
		// **ولا يُردّ خطأ**: الشاشةُ ترسم خطَّها المستقيمَ كما كانت،
		// **وشاشةٌ تسقط لأنّ خدمةَ مساراتٍ نامت** أسوأُ من خطٍّ تقريبيّ.
		s.logger.Warn("المسار: تعذّر", "error", err)
		httpx.JSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}

	pts := make([][2]float64, 0, len(route.Geometry))
	for _, p := range route.Geometry {
		pts = append(pts, [2]float64{p.Lat, p.Lng})
	}

	// ══════════════════════════════════════════════════════════════════
	// **والمسافةُ من المحرّك — والمدّةُ من إعداد المالك**
	// ══════════════════════════════════════════════════════════════════
	//
	// **ومدّةُ OSRM محسوبةٌ من وسم `maxspeed` في الخريطة** — وهو نادرٌ في
	// شوارع الرقّة، **فيرجع المحرّكُ إلى سرعاتٍ افتراضيّةٍ بحسب نوع
	// الشارع.** وقيس (٢٠٢٦-٠٨-١٢): ١٥١٨ متراً في ٢٫٣ دقيقة — **أربعون
	// كيلومتراً في الساعة لدرّاجةٍ في سوق.**
	//
	// **والمالكُ ضبط عشرين** (`drivers.avg_speed_kmh`) لمدينةٍ بازدحامها
	// ووقفاتها، **وتقديرٌ متفائلٌ يصنع شكوى.**
	//
	// **فالمسافةُ صارت حقيقيّةً — وهي ما كان يكذب**، والسرعةُ تبقى رقماً
	// واحداً يفهمه المالكُ ويضبطه بعينه. **ومدّةُ المحرّك تُرسَل معها**
	// لتُقارَن بالواقع يوماً وتُبدَّل إن صحّت.
	speed := s.driverSpeedKmh(r.Context(), userIDFrom(r))
	duration := route.DurationS
	if speed > 0 {
		duration = route.DistanceM / (float64(speed) * 1000 / 3600)
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"available":  true,
		"distance_m": math.Round(route.DistanceM),
		"duration_s": math.Round(duration),
		// engine_duration_s **ما قاله المحرّك نفسُه** — يُقارَن ولا يُعرض.
		"engine_duration_s": math.Round(route.DurationS),
		"points":            pts,
	})
}

// routeCached يقرأ من الذاكرة أوّلا — **ثمّ يسأل ويحفظ.**
func (s *Server) routeCached(ctx context.Context, from, to routing.Point) (*routing.Route, error) {
	if !s.route.Enabled() {
		return nil, routing.ErrNoEngine
	}
	key := "route:" + cell(from.Lat) + "," + cell(from.Lng) + ";" + cell(to.Lat) + "," + cell(to.Lng)
	if s.rdb != nil {
		if raw, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
			var cached routing.Route
			if json.Unmarshal(raw, &cached) == nil {
				return &cached, nil
			}
		}
	}
	route, err := s.route.Route(ctx, from, to)
	if err != nil {
		return nil, err
	}
	if s.rdb != nil {
		if raw, err := json.Marshal(route); err == nil {
			s.rdb.Set(ctx, key, raw, routeCacheTTL)
		}
	}
	return route, nil
}

// ══════════════════════════════════════════════════════════════════════
// **سرعتُه هو — لا سرعةٌ في الإعدادات**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «الوقت والسرعة تتحدّد من سرعة الموتور
//
//	الحقيقي مو من إعدادات الأدمن».)
//
// **ورقمٌ واحدٌ لكلّ السائقين يظلم اثنين**: من يقود بحذرٍ يَعِد بما لا
// يفي، **ومن يعرف طرقَه يُوعَد عنه بأبطأَ ممّا يفعل** فيُحسب أبعدَ من
// طلبٍ هو أقربُ الناس إليه.
//
// # والسرعةُ من جهازه لا من نقطتين
//
// **الـGPS يقول سرعتَه لحظتَها** (`speed_mps`) — وهي قراءةُ حسّاسٍ.
// **وحسابُها من نقطتين يكذب عند أوّل قفزةٍ في الدقّة**: قفزةُ مئةِ مترٍ
// في ثانيتين تُقرأ مئةً وثمانين كيلومتراً في الساعة.
//
// # ولا يُحسب الوقوف
//
// **نصفُ نبضاته وهو واقفٌ عند إشارةٍ أو بابِ زبون** — ومتوسّطٌ يشملها
// يقول «ثمانية كيلومترات» لمن يسير بثلاثين. **فتُقرأ نبضاتُ السير
// وحدَها** (فوق ٥ كم/سا)، **والوقوفُ داخلٌ أصلاً في الطريق** الذي
// يحسبه المحرّك.
//
// # وقليلُ النبضات لا يُبنى عليه
//
// **سائقٌ في أوّل يومه** لا تاريخَ له، **وعشرُ نبضاتٍ في منعطفٍ واحد**
// ليست عادتَه. فدون العشرين يُرجَع إلى رقم المالك.
//
// # وحدٌّ أعلى وأدنى
//
// **قراءةٌ شاذّةٌ واحدةٌ تفسد المتوسّط** — ومن مرّ في سيّارةِ صديقه على
// طريقٍ سريع يُقرأ ثمانين. **فيُحبس بين خمسٍ وستّين.**
func (s *Server) driverSpeedKmh(ctx context.Context, driverID string) int64 {
	fallback := s.settings.GetInt(ctx, "drivers.avg_speed_kmh")
	var avgMps *float64
	var n int
	err := s.pg.QueryRow(ctx, `
		SELECT avg(speed_mps), count(*) FROM driver_track
		WHERE driver_id = $1::uuid
		  AND recorded_at > now() - interval '14 days'
		  AND speed_mps IS NOT NULL AND speed_mps > 1.4`, driverID).Scan(&avgMps, &n)
	if err != nil || avgMps == nil || n < 20 {
		return fallback
	}
	kmh := int64(*avgMps * 3.6)
	if kmh < 5 || kmh > 60 {
		return fallback
	}
	return kmh
}

// cell يقرّب الإحداثيَّ إلى أربع منازل — **نحوَ أحدَ عشرَ مترا.**
func cell(v float64) string { return strconv.FormatFloat(v, 'f', 4, 64) }
