package server

import (
	"context"
	"encoding/json"
	"errors"
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
// ══════════════════════════════════════════════════════════════════════
// **ونسختان لا واحدة — المرحلة ٧، البند ١٧**
// ══════════════════════════════════════════════════════════════════════
//
// **أمرُ المالك نصّاً**: «لا أريد route single وroute alternatives
// يتشاركان Cache key ثم Client يأخذ Contract خاطئ».
//
// **والخطرُ ليس نظريّاً**: طلبٌ مفردٌ يملأ المخبأ، ثمّ طلبُ بدائلَ
// يقرؤه **فيردّ مساراً واحداً**، **ويظنّ التطبيقُ أنّ المحرّكَ لم يجد
// بدائل** — وهو لم يُسأل أصلاً.
//
// **ورُفعت `v2`→`v3`** لأنّ شكلَ القيمة تبدّل: `Route` صارت تحمل
// `EngineWeight` و`WeightName`.
const routeCacheVersion = "v3"

// **ونمطان**: مفردٌ ومجموعة.
const (
	cacheModeSingle = "single"
	cacheModeAlts   = "alternatives"
)

// ══════════════════════════════════════════════════════════════════════
// **وحبيبتان لا واحدة — إغلاقُ صحّة ٧، ٢٠٢٦-٠٨-٢١**
// ══════════════════════════════════════════════════════════════════════
//
// **أمرُ المالك نصّاً**: «لا أقبل أن request B يستلم Cached RouteSet
// المبنية للأصل A لمجرد أن A وB وقعا في خلية 4 decimals ≈11m».
//
// # **والقياسُ الأوّلُ كان معطوباً**
//
// **قِيس أوّلاً أنّ الأساسَ يتبدّل في ٣٠٪ عند إزاحة ٥م** — **وذلك
// خطأُ مقياسٍ لا خطأُ محرّك.** كانت الشبكةُ المكانيّةُ في
// `routeset.go` تسجّل القطعةَ عند طرفيها، **ورؤوسُ الطرق السريعة
// متباعدة**، فتسقط الخلايا الوسطى **فيبدو مساران متطابقان مختلفين.**
//
// **وبعد الإصلاح** (نفسُ العيّنات، ١٤٠ مقارنةً لكلّ مسافة):
//
//	إزاحة ٥م   الأساسُ تبدّل ٠٫٧٪
//	إزاحة ١٠م  الأساسُ تبدّل ١٫٤٪
//	إزاحة ٢٠م  الأساسُ تبدّل ٢٫٩٪
//
// # **والسؤالُ الصحيحُ ليس هذا**
//
// **بل**: أيردّ المخبأُ استجابةً غيرَ التي كان المحرّكُ سيولّدها لهذا
// الطلب؟ **فقِيس على أزواجٍ تقع في الخليّة نفسِها** (٥٤ زوجاً لكلّ
// دقّة، أسوأُ حالٍ — زاويتان متقابلتان):
//
//	٤ منازل   خليّة ١١٫١م × ٩٫١م   **تلوّثٌ ١٫٩٪**
//	٥ منازل   خليّة  ١٫١م × ٠٫٩م   **تلوّثٌ ٠٫٠٪**
//	٦ منازل   خليّة  ٠٫١م × ٠٫١م   **تلوّثٌ ٠٫٠٪**
//
// **والضابطُ سليم**: الأصلُ نفسُه مرّتين ⇒ ١٠٠٪ تطابق. **فالمحرّكُ
// حتميّ، وكلُّ اختلافٍ سببُه الأصل.**
//
// # **فالبدائلُ بخمسة منازل والمفردُ بأربعة**
//
// **والمفردُ لا يُمسّ** — أمرُ المالك: «لا أريد تغيير السلوك القديم
// بلا داعٍ». **وهو يُسأل مع كلّ نبضةِ موقع**، فحبيبتُه تشتري إصاباتٍ
// حقيقيّة.
//
// **والبدائلُ تُطلب عند أحداثٍ لا في حلقةٍ** (بدءُ رحلةٍ أو طلبٌ
// صريح) — **فالإصابةُ فيها قليلةُ القيمة أصلاً**، ولا يُدفع ثمنُ
// صحّةٍ مقابلها.
const (
	// **حبيبةُ المسار المفرد** — ٤ منازل ≈ ١١م، كما كانت.
	cellDecimalsSingle = 4

	// **وحبيبةُ البدائل** — ٥ منازل ≈ ١٫١م، تلوّثٌ صفر.
	cellDecimalsAlts = 5

	// ══════════════════════════════════════════════════════════════
	// **وحارسٌ ثانٍ فوق الحبيبة**
	// ══════════════════════════════════════════════════════════════
	//
	// **الحبيبةُ تُقلّل التلوّثَ إحصائيّاً، وهذا يمنعه قطعاً.**
	//
	// **يُحفظ الأصلُ الذي حُسبت منه المجموعة**، **ولا تُعاد إلّا
	// لطلبٍ أصلُه ضمنَ هذه المسافة.** فحتّى لو وقع طلبان على طرفَي
	// خليّةٍ **لا يتشاركان إلّا إن تقاربا فعلاً.**
	//
	// **ومترٌ ونصفٌ أوسعُ قليلاً من قطر الخليّة** (١٫١م × ٠٫٩م)،
	// **فلا يردّ إصابةً مشروعة.**
	altOriginToleranceM = 1.5
)

func (s *Server) handleDriverOrderRoute(w http.ResponseWriter, r *http.Request) {
	// **وموضعُ الجهاز يُقرأ أوّلاً** — **وقيمةٌ فاسدةٌ تُردّ قبل أن
	// تُتعب القاعدة**، ولا تُبتلع في خطأِ خادمٍ لاحق.
	local, perr := clientPoint(r)
	if perr != nil {
		s.respondErr(w, perr)
		return
	}
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

	// ══════════════════════════════════════════════════════════════════
	// **وموضعُ الجهاز يسبق موضعَ الخادم — إن أرسله**
	// ══════════════════════════════════════════════════════════════════
	//
	// (المرحلة ٣ب، بموافقة المالك ٢٠٢٦-٠٨-٢٠.)
	//
	// **وقِيس الفرق**: خدمةُ الورديّة ترسل كلَّ خمس ثوانٍ وبعد عشرين
	// متراً (`MIN_SEND_GAP_MS` و`MIN_MOVE_M`)، **وحلقةُ الملاحة تقرأ
	// كلَّ ثانيةٍ بلا حدّ مسافة.** فسائقٌ بخمسين كم/س **يكون موضعُه
	// عند الخادم متأخّراً تسعةً وستّين متراً** في أحسن أحواله.
	//
	// **وإعادةُ حسابٍ من موضعٍ متأخّرٍ تُنتج مساراً من حيث كان لا من
	// حيث هو** — وربّما أعادته إلى الشارع الذي تركه، **فأطلقت حلقة.**
	//
	// # ولا يُرسَل غيرُ النقطة
	//
	// **الجهازُ يرسل واقعةً والخادمُ يقرّر** — قاعدةُ المشروع. فلا
	// وجهةَ ولا مسافةَ ولا زمنَ يُقبل منه: **الطورُ يُقرأ من الطلب
	// لحظةَ النداء** (أسفلُه)، **والمسارُ يُحسب هنا.**
	//
	// **واختياريّان بالكامل** — نسخةٌ منشورةٌ لا ترسلهما تعمل حرفاً
	// بحرفٍ كما كانت.
	picked := status != "assigned" && status != "at_pickup"
	from := routing.Point{Lat: pickLat, Lng: pickLng}
	if !picked && fromLat != nil && fromLng != nil {
		from = routing.Point{Lat: *fromLat, Lng: *fromLng}
	}
	if local != nil {
		from = *local
	}
	to := routing.Point{Lat: pickLat, Lng: pickLng}
	if picked {
		to = routing.Point{Lat: dropLat, Lng: dropLng}
	}

	// ══════════════════════════════════════════════════════════════
	// **والبدائلُ بطلبٍ لا افتراضاً** — البند ١٦
	// ══════════════════════════════════════════════════════════════
	//
	// **أمرُ المالك نصّاً**: «old client لا يحمل payload البدائل إلا
	// إذا طلبها».
	//
	// **والعقدُ القديمُ يبقى بايتاً ببايت**: من لم يطلب لم يتغيّر له
	// شيء.
	wantAlternatives := r.URL.Query().Get("alternatives") == "true"

	// ══════════════════════════════════════════════════════════════════
	// **وبياناتُ الارتباط باشتراكٍ صريح — المرحلة ٨ب**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٢.)
	//
	// **والردُّ القديمُ لا يتبدّل بلاها**: المرحلةُ ٧ ثبّتت
	// أنّ `GET /route` يبقى كما هو، **ومن أراد زيادةً طلبها.**
	wantCorrelation := r.URL.Query().Get("correlation") == "true"

	target := "pickup"
	if picked {
		target = "dropoff"
	}

	routes, err := s.routeSetCached(r.Context(), from, to, wantAlternatives)
	if err != nil || len(routes) == 0 {
		// **ولا يُردّ خطأ**: الشاشةُ ترسم خطَّها المستقيمَ كما كانت،
		// **وشاشةٌ تسقط لأنّ خدمةَ مساراتٍ نامت** أسوأُ من خطٍّ تقريبيّ.
		// ══════════════════════════════════════════════════════════════
		// **والسببُ يُميّز في السجلّ — المرحلة ٨أ**
		// ══════════════════════════════════════════════════════════════
		//
		// **والعقدُ لم يتبدّل**: `available:false` كما كان — **فلا
		// توسيعَ API قبل أن تُثبَت الحاجة.**
		//
		// **لكنّ ثلاثةَ أسبابٍ تختلف علاجاً**: محرّكٌ نائم، وإحداثيّةٌ
		// خارجَ الحدّ، وطرفانِ لا سبيلَ بينهما. **ومن رآها سطراً
		// واحداً لم يعرف أيَّها وقع.**
		reason := "تعذّر"
		switch {
		case errors.Is(err, routing.ErrNoSegment):
			reason = "إحداثيّةٌ خارجَ حدّ الالتقاط"
		case errors.Is(err, routing.ErrNoRoute):
			reason = "لا سبيلَ بين النقطتين"
		case errors.Is(err, routing.ErrNoEngine):
			reason = "لا محرّك"
		}
		s.logger.Warn("المسار: "+reason, "error", err, "target", target)
		httpx.JSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}
	route := routes[0]

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

	// ══════════════════════════════════════════════════════════════════
	// **ومعرّفُ المسار لمن طلبه — المرحلة ٨ب**
	// ══════════════════════════════════════════════════════════════════
	//
	// **ومصادرُ المسار أربعة** — أوّلٌ، وإعادةُ حسابٍ،
	// وموصًى به، وبديلٌ مختار. **والثلاثةُ الأولى تمرّ من
	// هنا** — فبلا هذا السطر **يقود السائقُ مساراً لا هويّةَ
	// له**، ولا يُعرف بماذا يُقارَن أثرُه.
	if wantCorrelation {
		orderID := chi.URLParam(r, "id")
		rid := routing.RouteID(route, orderID, target)
		out["route_id"] = rid
		// **ويُحفظ سياقُ المقارنة** — من الخبيئة أو من
		// المحرّك سواءً (البند ٦).
		s.storeCorrelation(r.Context(), userIDFrom(r), orderID, target, rid,
			from, to, rid)
	}

	if wantAlternatives {
		orderID := chi.URLParam(r, "id")
		if !wantCorrelation {
			out["route_id"] = routing.RouteID(route, orderID, target)
		}
		out["target"] = target
		// **ومدّةُ المحرّك صريحةً** — البند ٤.
		//
		// **`duration_s` أعلاه معدَّلةٌ بسرعة السائق** وهي دلالةٌ
		// قديمةٌ لا تُكسر. **والمقارنةُ بين المسارات تكون على
		// `engine_duration_s` وحدَها**، وإلّا صارت كلُّ المسارات
		// بالسرعة نفسِها فاختفى الفرقُ الزمنيّ.
		out["weight_name"] = route.WeightName
		out["engine_weight"] = route.EngineWeight

		kept, dropped := routing.FilterAlternatives(route, routes[1:])
		alts := make([]map[string]any, 0, len(kept))
		for _, a := range kept {
			alts = append(alts, alternativePayload(a, route, orderID, target))
		}
		out["alternatives"] = alts
		out["alternatives_meta"] = map[string]any{
			"engine_returned":   len(routes),
			"after_filter":      len(kept),
			"dropped":           dropped,
			"primary_outranked": routing.PrimaryOutranked(route, routes[1:]),
		}
	}

	httpx.JSON(w, http.StatusOK, out)
}

// ══════════════════════════════════════════════════════════════════════
// **حمولةُ البديل — أرقامٌ لا صفات**
// ══════════════════════════════════════════════════════════════════════
//
// (البنود ٥ و٦ و٣٣.)
//
// **أمرُ المالك نصّاً**: «لا FASTEST · لا SHORTEST… الأفضل أن UI
// المستقبلية تعتمد الأرقام: distance delta · duration delta».
//
// **فلا وسمَ هنا** — فرقان بالمتر والثانية، **والواجهةُ تصوغهما.**
// و`delta_duration_s` **من مدّة المحرّك** لا من المعدَّلة بالسرعة.
func alternativePayload(a, primary *routing.Route, orderID, target string) map[string]any {
	pts := make([][2]float64, 0, len(a.Geometry))
	for _, p := range a.Geometry {
		pts = append(pts, [2]float64{p.Lat, p.Lng})
	}
	out := map[string]any{
		"route_id":          routing.RouteID(a, orderID, target),
		"distance_m":        math.Round(a.DistanceM),
		"engine_duration_s": math.Round(a.DurationS),
		"engine_weight":     a.EngineWeight,
		"delta_distance_m":  math.Round(a.DistanceM - primary.DistanceM),
		"delta_duration_s":  math.Round(a.DurationS - primary.DurationS),
		"shared_ratio":      math.Round(routing.SharedRatio(primary.Geometry, a.Geometry)*1000) / 1000,
		// ══════════════════════════════════════════════════════════
		// **موضعُ القرار الأوّل** — إغلاقُ صحّة ٧، البنود ٦ إلى ١٠
		// ══════════════════════════════════════════════════════════
		//
		// **مقيسٌ على المسار الموصى به** — فهو الذي يقوده السائق.
		//
		// **وأوّلُ فترةٍ ذاتِ معنى لا أوّلُ شاردة ولا آخرُ فترة**:
		// **فمن تجاوز هذا الموضعَ اتُّخذ قرارُه الأوّلُ فعلاً**،
		// **ولا يُعرض له بديلٌ يحاول إرجاعَه إلى فرعٍ فات.**
		"decision_divergence_m": math.Round(
			routing.MeaningfulDivergence(primary.Geometry, a.Geometry)),

		// **وأوّلُ افتراقٍ أيّاً كان** — للتشخيص لا للقرار.
		"first_divergence_m": math.Round(
			routing.FirstDivergenceM(primary.Geometry, a.Geometry)),
		"points": pts,
	}
	if a.HasNavigation() {
		// **والبديلُ جاهزٌ للملاحة** — البند ٢١: لا نداءَ ثانٍ عند
		// الاختيار.
		out["cumulative_m"] = a.CumulativeM
		out["maneuvers"] = a.Maneuvers
	}
	return out
}

// ══════════════════════════════════════════════════════════════════════
// **نقطةُ الجهاز — تُقرأ ويُتحقَّق منها، أو لا تكون**
// ══════════════════════════════════════════════════════════════════════
//
// (المرحلة ٣ب، أمرُ المالك ٢٠٢٦-٠٨-٢٠، البندان ٤ و٢١.)
//
//	لا `lat` ولا `lng`   ←  فارغٌ بلا خطأ — السلوكُ القديم حرفاً بحرف
//	واحدٌ دون الآخر       ←  خطأُ تحقّقٍ صريح — **زوجٌ أو لا شيء**
//	NaN أو ∞ أو نصٌّ      ←  خطأُ تحقّق
//	خارجَ المدى          ←  خطأُ تحقّق
//
// # ولا يُردّ موضعٌ صحيحٌ لأنّ موضعَ الخادم قديم
//
// (أمرُ المالك: «لا أريد Security check يرفض موقعاً صحيحاً لمجرّد أنّ
//
//	`users.last_location` قديمة».)
//
// **وموضعُ الخادم يتأخّر بحقّ**: انقطاعُ شبكةٍ · دفعاتٌ في الخلفيّة ·
// `MIN_SEND_GAP` · `MIN_MOVE` · طابور. **فمقارنةُ مسافةٍ به تردّ
// الصادقَ أكثرَ ممّا تردّ الكاذب.**
//
// **والضررُ المتصوَّر من نقطةٍ كاذبةٍ محدود**: السائقُ يرى مساراً خاطئاً
// لنفسه. **ولا يغيّر وجهةً ولا مالاً ولا حالَ طلب** — تلك كلُّها من
// الطلب في القاعدة.
//
// **فالحراسةُ حيث تنفع**: التخويلُ (الطلبُ له) والمدى والصحّةُ
// العدديّة.
// errBadPoint **نقطةٌ لا تصلح** — خطأُ طالبٍ لا خطأُ خادم.
var errBadPoint = httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")

func clientPoint(r *http.Request) (*routing.Point, error) {
	q := r.URL.Query()
	// **و«لم يُرسَل» غيرُ «أُرسل فارغا»** — `Get` تردّ فراغاً فيهما
	// معاً، **فيُسأل عن الوجود لا عن القيمة**: من كتب `?lat=&lng=`
	// قصد أن يرسل موضعاً ولم يرسله، **وذاك خطأُ طالبٍ لا سلوكٌ قديم.**
	if !q.Has("lat") && !q.Has("lng") {
		return nil, nil
	}
	rawLat, rawLng := q.Get("lat"), q.Get("lng")
	// **وزوجٌ أو لا شيء** — نصفُ نقطةٍ ليس نقطة، **وقبولُ نصفِها
	// بصمتٍ يعني مساراً من مكانٍ لم يقله أحد.**
	if rawLat == "" || rawLng == "" {
		return nil, errBadPoint
	}
	lat, err1 := strconv.ParseFloat(rawLat, 64)
	lng, err2 := strconv.ParseFloat(rawLng, 64)
	if err1 != nil || err2 != nil {
		return nil, errBadPoint
	}
	// **و`NaN` و`±Inf` تمرّ من `ParseFloat`** — تُكتب نصّاً فتُقبل،
	// **ثمّ تصير `null` في JSON أو تُسقط حساباً في المحرّك.**
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) {
		return nil, errBadPoint
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return nil, errBadPoint
	}
	return &routing.Point{Lat: lat, Lng: lng}, nil
}

// ══════════════════════════════════════════════════════════════════════
// **مخبأُ المجموعة — مفصولٌ عن المفرد** (البند ١٧)
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يتشاركان مفتاحاً**: طلبٌ مفردٌ يملأ المخبأ ثمّ طلبُ بدائلَ
// يقرؤه **فيردّ مساراً واحداً**، **فيظنّ التطبيقُ أنّ المحرّكَ لم
// يجد بدائل** — وهو لم يُسأل أصلاً.
func (s *Server) routeSetCached(
	ctx context.Context, from, to routing.Point, alternatives bool,
) ([]*routing.Route, error) {
	if !s.route.Enabled() {
		return nil, routing.ErrNoEngine
	}
	mode := cacheModeSingle
	decimals := cellDecimalsSingle
	if alternatives {
		mode = cacheModeAlts
		decimals = cellDecimalsAlts
	}
	key := "route:" + routeCacheVersion + ":" + mode + ":" +
		cellAt(from.Lat, decimals) + "," + cellAt(from.Lng, decimals) + ";" +
		cellAt(to.Lat, decimals) + "," + cellAt(to.Lng, decimals)

	if s.rdb != nil {
		if raw, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
			var entry cachedRoutes
			if json.Unmarshal(raw, &entry) == nil && len(entry.Routes) > 0 {
				if s.cacheEntryUsable(entry, from, alternatives) {
					return entry.Routes, nil
				}
			}
		}
	}

	var (
		out []*routing.Route
		err error
	)
	if alternatives {
		out, err = s.route.RouteSet(ctx, from, to)
	} else {
		var one *routing.Route
		one, err = s.route.Route(ctx, from, to)
		if one != nil {
			out = []*routing.Route{one}
		}
	}
	if err != nil {
		return nil, err
	}
	if s.rdb != nil {
		entry := cachedRoutes{Routes: out, OriginLat: from.Lat, OriginLng: from.Lng}
		if raw, err := json.Marshal(entry); err == nil {
			s.rdb.Set(ctx, key, raw, routeCacheTTL)
		}
	}
	return out, nil
}

// cachedRoutes **ما يُحفظ** — مساراتٌ **والأصلُ الذي حُسبت منه.**
//
// **والأصلُ ليس زينة**: به يُمنع التلوّثُ قطعاً لا إحصائيّاً (البند ٤).
type cachedRoutes struct {
	Routes    []*routing.Route `json:"routes"`
	OriginLat float64          `json:"origin_lat"`
	OriginLng float64          `json:"origin_lng"`
}

// cacheEntryUsable **أتصلح هذه المدخلةُ لهذا الطلب؟**
//
// **والمفردُ لا يُفحص أصلُه** — سلوكُه القديمُ محفوظٌ كما أمر المالك.
// **والبدائلُ تُفحص**: فلا تُعاد مجموعةٌ بُنيت من أصلٍ آخر.
func (s *Server) cacheEntryUsable(
	entry cachedRoutes, from routing.Point, alternatives bool,
) bool {
	if !alternatives {
		return true
	}
	// **ومدخلةٌ قديمةٌ بلا أصلٍ محفوظ** — من قبل هذا الإغلاق —
	// **تُرفض ولا تُقرأ.** والثمنُ نداءٌ واحدٌ للمحرّك.
	if entry.OriginLat == 0 && entry.OriginLng == 0 {
		return false
	}
	moved := routing.MetersBetween(
		routing.Point{Lat: entry.OriginLat, Lng: entry.OriginLng}, from,
	)
	return moved <= altOriginToleranceM
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
func cell(v float64) string { return cellAt(v, cellDecimalsSingle) }

// cellAt **يقصّ الإحداثيَّ إلى دقّةٍ معلومة.**
//
// **والدقّةُ ليست واحدةً للنمطين** — انظر أعلاه: المفردُ ٤ والبدائلُ ٥.
func cellAt(v float64, decimals int) string {
	return strconv.FormatFloat(v, 'f', decimals, 64)
}

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
