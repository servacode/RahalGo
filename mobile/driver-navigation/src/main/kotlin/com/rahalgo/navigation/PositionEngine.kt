package com.rahalgo.navigation

import kotlin.math.max
import kotlin.math.min

/**
 * ══════════════════════════════════════════════════════════════════════
 * **موضعُ الملاحة — لا الموضعُ الخامّ** (المرحلة ٢ من خطّة الملاحة)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-١٠-٠٢: «السهمُ يختفي ويظهر، ومرّةً للخلف ومرّةً للأمام»؛
 *  والمواصفةُ، الأجزاء ٤ و٧ و٨: «لا تحرّك الماركر حسب الـGPS الخامّ — هذا مرفوض».)
 *
 * **قِيس في الفحص** (`docs/navigation/NAVIGATION-AUDIT.md` ٢.١–٢.٣): السهمُ يُرسم على
 * القراءة الخامّة بخطأٍ يبلغ ٦٠م، **ولا تثبيتَ عند الوقوف**، والتقدّمُ يزحف للأمام والسائقُ
 * واقف. **وقِيس على الجهاز داخلَ البيت: المزوّدُ يقول ٢٦ كم/س باتّجاهٍ ثابتٍ وهو واقف.**
 *
 * # ما يصنعه
 *
 * ١. **حالةُ الحركة بهسترة** — لا تُبنى على قراءةٍ واحدةٍ ولا على سرعة المزوّد وحدَها:
 *    - تبدأ الحركةُ بسرعةٍ ≥ ٢٫٥م/ث في قراءتين مقبولتين، **أو** بابتعادٍ عن نقطة الوقوف
 *      أكبرَ من max(١٥م، الدقّة).
 *    - ويعود الوقوفُ بسرعةٍ < ١م/ث مدّةَ ٣ ثوانٍ.
 * ٢. **عند الوقوف يُثبَّت الموضعُ والاتّجاهُ والتقدّم** — فلا يهتزّ السهمُ ولا يزحف المتبقّي.
 * ٣. **الإلصاقُ بالطريق المرسوم** حين يكون السائقُ عليه: البعدُ الجانبيُّ ≤ max(١٥م، ١٫٥×الدقّة)
 *    ولم يُعلَن خروجٌ عن المسار ⇒ **يُعرض على الخطّ** باتّجاه الخطّ. **وإلّا يُعرض الخامُّ**
 *    (كعلامة `IS_ROAD_SNAPPED_KEY` عند غوغل — `GOOGLE-NAVIGATION-BEHAVIOR-REFERENCE.md` ١).
 *
 * **والخامُّ لا يضيع**: الخادمُ والسجلُّ والوصولُ تقرأ القراءةَ كما هي — هذا موضعُ العرض
 * والتقدّم وحدَهما.
 */
class PositionEngine {

    enum class Motion { STATIONARY, MOVING }

    data class Output(
        val lat: Double,
        val lng: Double,
        /** **اتّجاهُ العرض** — فارغٌ يعني «أبقِ الأخير». */
        val bearingDeg: Float?,
        val motion: Motion,
        /** **ألُصق بالطريق؟** */
        val snapped: Boolean,
        /** **أيُقدَّم التقدّمُ بهذه القراءة؟** — لا عند الوقوف. */
        val advanceProgress: Boolean,
    )

    var motion: Motion = Motion.STATIONARY
        private set

    /**
     * **السرعةُ الملساءُ بالمتر/ث** — صفرٌ عند الوقوف.
     *
     * **ولا تُقرأ سرعةُ المزوّد الخامّةُ في الشاشة** (المواصفة ٢٢): تقريبُ الكاميرا
     * والسرعةُ المعروضة تقرآن هذه.
     */
    var speedMps: Double = 0.0
        private set

    private var holdLat = Double.NaN
    private var holdLng = Double.NaN
    private var holdBearing: Float? = null
    private var fastRun = 0
    private var awayRun = 0

    /** **مركزُ القراءات الخامّة أثناء الوقوف** — منه يُقاس الابتعاد، لا من المعروض. */
    private var anchorLat = Double.NaN
    private var anchorLng = Double.NaN
    private var anchorN = 0
    private var slowSinceMs = -1L
    private var lastFix: NavFix? = null

    fun reset() {
        motion = Motion.STATIONARY
        holdLat = Double.NaN
        holdLng = Double.NaN
        holdBearing = null
        fastRun = 0
        awayRun = 0
        anchorLat = Double.NaN
        anchorLng = Double.NaN
        anchorN = 0
        slowSinceMs = -1L
        lastFix = null
        speedMps = 0.0
    }

    /**
     * **حالةُ الحركة من القراءة** — تُنادى قبل التقدّم كي يُقفَل عند الوقوف.
     */
    fun classify(fix: NavFix, grade: FixGrade): Motion {
        if (grade == FixGrade.REJECTED) return motion
        val speed = speedOf(fix)
        lastFix = fix
        speedMps = speedMps + SPEED_ALPHA * (speed - speedMps)
        when (motion) {
            Motion.STATIONARY -> {
                // **والمتدهورةُ تشهد على الحركة أيضاً** — دقّةُ ٣٠–٥٠م في السوق شائعة،
                // **ومن لا يتحرّك إلّا بقراءةٍ ممتازةٍ يبقى واقفاً على الشاشة وهو يسير.**
                fastRun = if (speed >= START_MPS) fastRun + 1 else 0
                val away = if (anchorLat.isNaN()) {
                    0.0
                } else {
                    GpsQuality.metersBetween(anchorLat, anchorLng, fix.lat, fix.lng)
                }
                awayRun = if (away > max(START_AWAY_M, fix.accuracyM.toDouble())) awayRun + 1 else 0
                if (fastRun >= START_RUN || awayRun >= START_RUN) {
                    motion = Motion.MOVING
                    slowSinceMs = -1L
                    awayRun = 0
                    anchorLat = Double.NaN
                    anchorN = 0
                } else {
                    // **المركزُ متوسّطٌ متحرّكٌ للقراءات** — الضجيجُ يتعادل حوله.
                    if (anchorLat.isNaN()) {
                        anchorLat = fix.lat
                        anchorLng = fix.lng
                        anchorN = 1
                    } else {
                        anchorN = min(anchorN + 1, ANCHOR_WINDOW)
                        anchorLat += (fix.lat - anchorLat) / anchorN
                        anchorLng += (fix.lng - anchorLng) / anchorN
                    }
                }
            }

            Motion.MOVING -> {
                if (speed < STOP_MPS) {
                    if (slowSinceMs < 0) slowSinceMs = fix.atMs
                    if (fix.atMs - slowSinceMs >= STOP_HOLD_MS) {
                        motion = Motion.STATIONARY
                        fastRun = 0
                        holdLat = Double.NaN // يُثبَّت على أوّل عرضٍ بعد الوقوف
                        speedMps = 0.0
                    }
                } else {
                    slowSinceMs = -1L
                }
            }
        }
        if (motion == Motion.STATIONARY) speedMps = 0.0
        return motion
    }

    /**
     * **موضعُ العرض** — بعد التقدّم (لموضع الإسقاط) وكاشف الخروج.
     */
    fun place(
        fix: NavFix,
        grade: FixGrade,
        rawBearing: Float?,
        route: NavRoute?,
        progress: RouteProgress.State?,
        offRoute: Boolean,
    ): Output {
        // **الواقفُ لا يتحرّك** — يُثبَّت على آخر ما عُرض.
        if (motion == Motion.STATIONARY) {
            if (holdLat.isNaN()) {
                val (lat, lng, snapped) = snap(fix, grade, route, progress, offRoute)
                holdLat = lat
                holdLng = lng
                if (snapped && route != null && progress != null) {
                    holdBearing = routeBearingAt(route, progress.progressM) ?: holdBearing
                }
                return Output(lat, lng, holdBearing, motion, snapped, advanceProgress = false)
            }
            return Output(holdLat, holdLng, holdBearing, motion, snapped = false, advanceProgress = false)
        }

        val (lat, lng, snapped) = snap(fix, grade, route, progress, offRoute)
        val bearing = if (snapped && route != null && progress != null) {
            routeBearingAt(route, progress.progressM) ?: rawBearing
        } else {
            rawBearing
        }
        if (bearing != null) holdBearing = bearing
        holdLat = lat
        holdLng = lng
        return Output(lat, lng, bearing ?: holdBearing, motion, snapped, advanceProgress = true)
    }

    private fun snap(
        fix: NavFix,
        grade: FixGrade,
        route: NavRoute?,
        progress: RouteProgress.State?,
        offRoute: Boolean,
    ): Triple<Double, Double, Boolean> {
        if (route == null || progress == null || offRoute || grade == FixGrade.REJECTED) {
            return Triple(fix.lat, fix.lng, false)
        }
        val limit = max(SNAP_MIN_M, SNAP_ACCURACY_FACTOR * fix.accuracyM.toDouble())
        if (progress.offRouteM > limit) return Triple(fix.lat, fix.lng, false)
        val p = pointAt(route, progress.progressM) ?: return Triple(fix.lat, fix.lng, false)
        return Triple(p.lat, p.lng, true)
    }

    private fun speedOf(fix: NavFix): Double {
        val reported = fix.speedMps?.toDouble()
        // **سرعةُ المزوّد لا تُصدَّق بدقّةٍ رديئة** — قِيس: ٢٦ كم/س داخلَ بيتٍ بدقّة ٣٠٠م.
        if (reported != null && fix.accuracyM <= TRUST_SPEED_ACCURACY_M) return reported
        val prev = lastFix ?: return 0.0
        val dt = (fix.atMs - prev.atMs) / 1000.0
        if (dt <= 0.25 || dt > 10.0) return 0.0
        val d = GpsQuality.metersBetween(prev, fix)
        // **والإزاحةُ داخلَ دقّتَي القراءتين ضجيجٌ لا حركة.**
        if (d <= max(4.0, (prev.accuracyM + fix.accuracyM).toDouble())) return 0.0
        return d / dt
    }

    companion object {
        const val START_MPS = 2.5
        const val START_RUN = 2
        const val START_AWAY_M = 15.0
        const val STOP_MPS = 1.0
        const val STOP_HOLD_MS = 3_000L
        const val SNAP_MIN_M = 15.0
        const val SNAP_ACCURACY_FACTOR = 1.5
        const val TRUST_SPEED_ACCURACY_M = 25f
        const val SPEED_ALPHA = 0.4
        const val ANCHOR_WINDOW = 20

        /** **النقطةُ على الطريق عند مسافةٍ منه** — بحثٌ ثنائيٌّ في التراكميّ. */
        fun pointAt(route: NavRoute, distanceM: Double): GeoPoint? {
            val g = route.geometry
            val c = route.cumulativeM
            if (g.size < 2 || c.size != g.size) return null
            val d = min(max(distanceM, 0.0), c[c.size - 1])
            var lo = 0
            var hi = c.size - 1
            while (hi - lo > 1) {
                val mid = (lo + hi) ushr 1
                if (c[mid] <= d) lo = mid else hi = mid
            }
            val seg = c[hi] - c[lo]
            val t = if (seg <= 0.0) 0.0 else (d - c[lo]) / seg
            val a = g[lo]
            val b = g[hi]
            return GeoPoint(a.lat + (b.lat - a.lat) * t, a.lng + (b.lng - a.lng) * t)
        }

        /** **اتّجاهُ الطريق عند مسافةٍ منه** — مماسُّ المقطع. */
        fun routeBearingAt(route: NavRoute, distanceM: Double): Float? {
            val g = route.geometry
            val c = route.cumulativeM
            if (g.size < 2 || c.size != g.size) return null
            val d = min(max(distanceM, 0.0), c[c.size - 1])
            var i = 0
            while (i < c.size - 2 && c[i + 1] < d) i++
            val a = g[i]
            val b = g[i + 1]
            val y = Math.sin(Math.toRadians(b.lng - a.lng)) * Math.cos(Math.toRadians(b.lat))
            val x = Math.cos(Math.toRadians(a.lat)) * Math.sin(Math.toRadians(b.lat)) -
                Math.sin(Math.toRadians(a.lat)) * Math.cos(Math.toRadians(b.lat)) *
                Math.cos(Math.toRadians(b.lng - a.lng))
            if (x == 0.0 && y == 0.0) return null
            return GpsQuality.normalize(Math.toDegrees(Math.atan2(y, x)).toFloat())
        }
    }
}
