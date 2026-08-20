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

// routeCacheVersion **نسخةُ صيغةِ المسار المخزَّن.**
//
// **وتُرفع كلَّما تبدّل ما يُحفظ** — انظر `routeCached`.
const routeCacheVersion = "v2"

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
	// **المسافةُ ثابتةٌ والزمنُ يتبع السرعة**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «ممكن سائق يقود ٢٠ وممكن سائق يقود ٣٥،
	//  معقول يبقى الزمن نفسه؟».)
	//
	// **وهو ما تفعله كلُّ خرائط الدنيا**: الطريقُ واحدٌ لا يتبدّل،
	// **ووقتُ الماشي غيرُ وقت الراكب.**
	//
	// **وسرعتُه مقيسةٌ لا مضبوطة** — ولا رقمَ من إعداداتٍ يخمّنه أحد.
	speed := s.driverSpeedKmh(r.Context(), userIDFrom(r))
	duration := route.DurationS
	if speed > 0 {
		duration = route.DistanceM / (float64(speed) * 1000 / 3600)
	}

	out := map[string]any{
		"available":  true,
		"distance_m": math.Round(route.DistanceM),
		"duration_s": math.Round(duration),
		// engine_duration_s **ما قاله المحرّك نفسُه** — يُقارَن ولا يُعرض.
		"engine_duration_s": math.Round(route.DurationS),
		"points":            pts,
	}

	// ══════════════════════════════════════════════════════════════════
	// **وبياناتُ الملاحة حقولٌ إضافيّةٌ لا بديلة**
	// ══════════════════════════════════════════════════════════════════
	//
	// (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «الحقول الحالية تبقى… وتضاف بيانات
	//  Navigation كحقول جديدة اختيارية».)
	//
	// **والنسخةُ المنشورةُ لا تعرفها فتتجاهلها** — `ignoreUnknownKeys`
	// في كوتلن. **وحقلٌ يُبدَّل معناه يكسر من لم يحدّث.**
	//
	// **ولا تُرسَل حين لا ملاحة** — محرّكٌ بلا خطواتٍ أو مسارٌ من
	// مخبأٍ قديم: **وحقلٌ فارغٌ يُقرأ «لا مناورات» وهو ما نريد.**
	if route.HasNavigation() {
		out["cumulative_m"] = route.CumulativeM
		out["maneuvers"] = route.Maneuvers
	}
	httpx.JSON(w, http.StatusOK, out)
}

// routeCached يقرأ من الذاكرة أوّلا — **ثمّ يسأل ويحفظ.**
func (s *Server) routeCached(ctx context.Context, from, to routing.Point) (*routing.Route, error) {
	if !s.route.Enabled() {
		return nil, routing.ErrNoEngine
	}
	// ══════════════════════════════════════════════════════════════════
	// **ونسخةٌ في المفتاح منذ المرحلة ٢**
	// ══════════════════════════════════════════════════════════════════
	//
	// (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «نفّذ خطّة `route:v2:` … **ولا تحاول
	//  تفسير Cache v1 القديم على أنّه Route Navigation جديد**».)
	//
	// **ومسارٌ خُزّن قبل الخطوات يُفكّ بلا مناورات** — فيصير «متاحاً»
	// بلا إرشاد، **ولا خطأَ يظهر**: الحقولُ الجديدةُ غائبةٌ فتُقرأ
	// أصفاراً.
	//
	// **ومفتاحٌ جديدٌ لا إبطالٌ صريح** — القديمُ يموت وحدَه بعد عشر
	// دقائق، **ولا يُمسّ مفتاحُ سائقٍ يقود الآن.**
	key := "route:" + routeCacheVersion + ":" +
		cell(from.Lat) + "," + cell(from.Lng) + ";" + cell(to.Lat) + "," + cell(to.Lng)
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
// ليست عادتَه. فدون العشرين يُنزَل إلى المصدر الذي بعده.
//
// # وثلاثةُ مصادرَ كلُّها مقيسة — ولا واحدَ منها تخمين
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «احذف هذا الإعداد ولا نعتمد عليه أبداً».)
//
//	سرعتُه هو            ←  نبضاتُ جهازه في أسبوعين
//	متوسّطُ السائقين     ←  لمن لا تاريخَ له بعد — وهو واقعُ المدينة
//	مدّةُ محرّك المسارات  ←  حين لا سائقَ قاد بعد أصلاً (صفرٌ هنا)
//
// **ورقمٌ يكتبه المالكُ تخميناً يُنتج وقتاً وهميّا**: يُقال للزبون «عشرُ
// دقائق» وهي خمسٌ أو عشرون، **ولا أحدَ يعرف أيُّهما.**
//
// # وحدٌّ أعلى وأدنى
//
// **قراءةٌ شاذّةٌ واحدةٌ تفسد المتوسّط** — ومن مرّ في سيّارةِ صديقه على
// طريقٍ سريع يُقرأ ثمانين. **فيُحبس بين خمسٍ وستّين.**
//
// **وصفرٌ يعني «لا أعرف»** — فيُترك الحسابُ لمحرّك المسارات.
func (s *Server) driverSpeedKmh(ctx context.Context, driverID string) int64 {
	if kmh := s.speedOf(ctx, `driver_id = $1::uuid`, driverID); kmh > 0 {
		return kmh
	}
	// **ومتوسّطُ السائقين واقعُ المدينة** — لا رقمَ يُكتب عنها.
	return s.speedOf(ctx, `true`, driverID)
}

// speedOf متوسّطُ سرعة السير بالكيلومتر — **وصفرٌ يعني لا يُعرف.**
func (s *Server) speedOf(ctx context.Context, where, driverID string) int64 {
	var avgMps *float64
	var n int
	err := s.pg.QueryRow(ctx, `
		SELECT avg(speed_mps), count(*) FROM driver_track
		WHERE `+where+`
		  AND recorded_at > now() - interval '14 days'
		  AND speed_mps IS NOT NULL AND speed_mps > 1.4`, driverID).Scan(&avgMps, &n)
	if err != nil || avgMps == nil || n < 20 {
		return 0
	}
	kmh := int64(*avgMps * 3.6)
	if kmh < 5 || kmh > 60 {
		return 0
	}
	return kmh
}

// cell يقرّب الإحداثيَّ إلى أربع منازل — **نحوَ أحدَ عشرَ مترا.**
func cell(v float64) string { return strconv.FormatFloat(v, 'f', 4, 64) }

// ══════════════════════════════════════════════════════════════════════
// **ومحرّكُ الطلبات يسأل الخريطةَ من هنا**
// ══════════════════════════════════════════════════════════════════════
//
// **الواجهةُ ضيّقةٌ عمدا** (`orders.RouteReader`): سؤالٌ واحدٌ عن ثوانٍ
// — **ولا رسمَ طريقٍ ولا خطواتٍ ولا بدائل.** ومحرّكُ الطلبات لا يحتاج
// حزمةَ التوجيه كلَّها ليعرف كم يستغرق مشوار.
//
// **ويمرّ بالمخبأ نفسِه** (`routeCached`) — سائقان يُسندان إلى المتجر
// نفسِه من الحيّ نفسِه **لا يسألان الخريطةَ مرّتين.**

// orderRouter يصل محرّكَ الطلبات بمحرّك الخرائط.
type orderRouter struct{ s *Server }

// Seconds كم ثانيةً بين نقطتين على الشارع — **وصفرٌ «لا يُعرف».**
func (r orderRouter) Seconds(ctx context.Context, fromLat, fromLng, toLat, toLng float64) int {
	route, err := r.s.routeCached(ctx,
		routing.Point{Lat: fromLat, Lng: fromLng},
		routing.Point{Lat: toLat, Lng: toLng})
	if err != nil || route == nil || route.DurationS <= 0 {
		return 0
	}
	return int(route.DurationS)
}
