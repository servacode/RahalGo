package com.rahalgo.navigation

import kotlin.math.abs
import kotlin.math.cos
import kotlin.math.max
import kotlin.math.min

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أين السائقُ على المسار — بنافذةٍ لا بمسحٍ أعمى**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٢، أمرُ المالك ٢٠٢٦-٠٨-٢٠: «لا أريد خوارزميّة: ابحث عن
 *  أقرب نقطةٍ في كامل Polyline كلّ ثانيةٍ بشكلٍ أعمى».)
 *
 * # ولماذا المسحُ الكاملُ خطأٌ مرّتين
 *
 * **الأولى في الأداء**: مسارُ الرقّة–دمشق **٢١١٣ رأساً** (قِيس)،
 * ومسحُها كلَّ ثانيةٍ إهدارٌ بلا مقابل.
 *
 * **والثانيةُ أخطر — في الصواب**: المسارُ يمرّ قرب نفسِه، ويلتفّ عند
 * دوّار، ويسير في شارعين متوازيين. **فأقربُ نقطةٍ في المسار كلِّه قد
 * تكون بعد كيلومترين** — فيقفز التقدّمُ من ٢٥٪ إلى ٧٠٪ **والسائقُ لم
 * يتحرّك.**
 *
 * # فالنافذةُ حولَ ما كان
 *
 *     [آخرُ تقدّمٍ − ٣٠م  ,  آخرُ تقدّمٍ + أقصى ما يمكن قطعُه]
 *
 * **وما وراءها لا يُقارَن به أصلاً** — فلا يُقفز إليه ولو كان ألصق.
 *
 * # والاتّجاهُ يفصل الشارعين المتوازيين
 *
 * **نقطتان متساويتا البعد، والذاهبُ شرقاً ليس الذاهبَ غرباً.** فتُرجَّح
 * القطعةُ التي يوافق اتّجاهُها اتّجاهَ السائق.
 *
 * **وهو متاحٌ عندنا من المرحلة ١ مجّاناً** — `BearingTracker`.
 *
 * # ولا نداءَ شبكةٍ هنا
 *
 * (أمرُ المالك: «لا تستخدم `/match` من الخادم كلّ ثانية».)
 *
 * **وحسابٌ محضٌ يُختبر بلا جهاز.**
 */
object RouteProjector {

    /** **حدُّ الرجوع في النافذة** — وقوفٌ وارتدادُ GPS. */
    const val BACK_WINDOW_M = 30.0

    /** **أدنى مدىً أماميّ** — حين تكون السرعةُ مجهولةً أو صفراً. */
    const val MIN_FORWARD_M = 60.0

    /** **معاملُ الاحتياط على المسافة الممكنة** — لتأخّرِ قراءةٍ أو تسارع. */
    const val FORWARD_SLACK = 1.5

    /** **وزنُ مخالفةِ الاتّجاه** — متراً لكلّ درجةٍ من الانحراف. */
    const val BEARING_PENALTY_M_PER_DEG = 0.6

    /** **دون هذه السرعة لا يُرجَّح باتّجاه** — انظر `BearingTracker`. */
    const val MIN_BEARING_SPEED_MPS = 2f

    /** **نتيجةُ الإسقاط.** */
    data class Hit(
        /** **المسافةُ المقطوعةُ على المسار** بالمتر. */
        val progressM: Double,
        /** **بُعدُ السائق عن الخطّ** — يُقرأ في التشخيص. */
        val offRouteM: Double,
        /** **فهرسُ القطعة** — تسريعٌ للنداء التالي. */
        val segment: Int,
    )

    /**
     * **يُسقط نقطةً على المسار داخل نافذةٍ حول آخر تقدّم.**
     *
     * @param lastProgressM آخرُ تقدّمٍ معروف — **صفرٌ في البداية.**
     * @param dtSec الزمنُ منذ آخر قراءةٍ مقبولة.
     * @param speedMps سرعةُ السائق — وفارغةٌ تعني «مجهولة».
     * @param bearingDeg اتّجاهُه المنعَّم — وفارغٌ يعني «لا ترجيح».
     */
    fun project(
        route: NavRoute,
        lat: Double,
        lng: Double,
        lastProgressM: Double,
        dtSec: Double,
        speedMps: Float?,
        bearingDeg: Float?,
        /**
         * ══════════════════════════════════════════════════════════════
         * **وأوّلُ إسقاطٍ يمسح المسارَ كلَّه**
         * ══════════════════════════════════════════════════════════════
         *
         * **ولا تقدّمَ سابقٌ يُبنى عليه** — والنافذةُ حول الصفر تحبس
         * السائقَ عند أوّل المسار.
         *
         * **وهي حالٌ واقعيّةٌ لا نادرة**: من فتح التطبيقَ وهو في
         * منتصف الطريق، أو أُعيدت جلستُه بعد انقطاع. **فيبقى سهمُه
         * عند البداية والمسافةُ لا تنقص.**
         *
         * **ومرّةً واحدةً في الجلسة** — فالتكلفةُ لا تتكرّر.
         */
        fullScan: Boolean = false,
    ): Hit? {
        val n = route.geometry.size
        if (n < 2 || route.cumulativeM.size != n) return null

        // **وأقصى ما يمكن قطعُه** — سرعةٌ في زمنٍ مع احتياط.
        val forward = max(
            MIN_FORWARD_M,
            (speedMps?.toDouble() ?: 0.0) * max(dtSec, 0.0) * FORWARD_SLACK,
        )
        val loM = if (fullScan) 0.0 else max(0.0, lastProgressM - BACK_WINDOW_M)
        val hiM = if (fullScan) route.totalM else min(route.totalM, lastProgressM + forward)

        val from = indexAtOrBefore(route.cumulativeM, loM)
        val to = min(n - 2, indexAtOrBefore(route.cumulativeM, hiM))

        var best: Hit? = null
        var bestScore = Double.MAX_VALUE
        val trustBearing = bearingDeg != null &&
            (speedMps == null || speedMps >= MIN_BEARING_SPEED_MPS)

        for (i in from..max(from, to)) {
            val a = route.geometry[i]
            val b = route.geometry[i + 1]
            val (t, dist) = pointToSegment(lat, lng, a, b)
            var score = dist
            if (trustBearing) {
                // **والقطعةُ التي تسير عكسَه تُعاقَب** — انظر أعلاه.
                val segBearing = GpsQuality.courseBetween(
                    NavFix(a.lat, a.lng, 0f, null, null, 0),
                    NavFix(b.lat, b.lng, 0f, null, null, 0),
                )
                val off = abs(GpsQuality.angleDelta(bearingDeg!!, segBearing))
                score += off * BEARING_PENALTY_M_PER_DEG
            }
            if (score < bestScore) {
                bestScore = score
                val segLen = route.cumulativeM[i + 1] - route.cumulativeM[i]
                best = Hit(
                    progressM = route.cumulativeM[i] + t * segLen,
                    offRouteM = dist,
                    segment = i,
                )
            }
        }
        return best
    }

    /**
     * **أوّلُ رأسٍ مسافتُه ≤ المطلوب** — بحثٌ ثنائيّ.
     *
     * **ومسحٌ خطّيٌّ هنا يُعيد ما هربنا منه** — ٢١١٣ رأساً في كلّ
     * قراءة.
     */
    fun indexAtOrBefore(cumulative: DoubleArray, meters: Double): Int {
        if (cumulative.isEmpty()) return 0
        var lo = 0
        var hi = cumulative.size - 1
        while (lo < hi) {
            val mid = (lo + hi + 1) / 2
            if (cumulative[mid] <= meters) lo = mid else hi = mid - 1
        }
        return lo
    }

    /**
     * **إسقاطُ نقطةٍ على قطعة** — يردّ `(t, المسافة بالمتر)`.
     *
     * **والحسابُ في مستوٍ محلّيّ**: على مئة مترٍ **الفرقُ بين الكرة
     * والمستوي أقلُّ من سنتيمتر**، **وحسابٌ كرويٌّ لكلّ قطعةٍ في كلّ
     * قراءةٍ يأكل معالجاً بلا مقابل.**
     */
    fun pointToSegment(lat: Double, lng: Double, a: GeoPoint, b: GeoPoint): Pair<Double, Double> {
        // **ودرجةُ الطول تضيق مع العرض** — فتُصحَّح بجيب التمام،
        // **وبلاها يصير المتر شرقاً أطولَ من المتر شمالاً.**
        val kx = 111_320.0 * cos(Math.toRadians(lat))
        val ky = 110_540.0
        val ax = (a.lng - lng) * kx
        val ay = (a.lat - lat) * ky
        val bx = (b.lng - lng) * kx
        val by = (b.lat - lat) * ky
        val dx = bx - ax
        val dy = by - ay
        val len2 = dx * dx + dy * dy
        if (len2 <= 1e-9) {
            return 0.0 to Math.hypot(ax, ay)
        }
        var t = -(ax * dx + ay * dy) / len2
        t = t.coerceIn(0.0, 1.0)
        val px = ax + t * dx
        val py = ay + t * dy
        return t to Math.hypot(px, py)
    }
}
