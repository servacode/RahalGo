package com.rahalgo.navigation

import kotlin.math.abs
import kotlin.math.hypot

/**
 * ══════════════════════════════════════════════════════════════════
 * **اختيارُ الخطّ باللمس — بالقطعة لا بالرأس**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ صحّة واجهة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ١٣ إلى ١٦.)
 *
 * # **العيبُ الذي أُصلح**
 *
 * **كانت المسافةُ تُقاس إلى أقرب رأس.** وذلك خاطئٌ بنيويّاً:
 *
 * **أمرُ المالك نصّاً**: «LineString يمكن أن تمر من تحت إصبع المستخدم
 * بينما أقرب vertex تبعد عشرات/مئات pixels».
 *
 * **ومثالُه**: مسارٌ «أ» قطعتُه من `(0,0)` إلى `(1000,0)` بلا رأسٍ
 * بينهما، **واللمسةُ عند `(500,5)`** — فهي على الخطّ بخمسة بكسلات.
 * **وأقربُ رأسٍ فيه يبعد ٥٠٠.** ومسارُ «ب» له رأسٌ عند `(500,30)`.
 *
 * **فبالرؤوس يفوز «ب» وهو أبعد**، **وبالقطع يفوز «أ» وهو تحتَ
 * الإصبع.**
 *
 * # **وعند التقارب لا يُخمَّن** (البند ١٥)
 *
 * **خطّان متقاربان تحتَ الإصبع لا يُفرَّق بينهما بيقين.** فالبطاقاتُ
 * تبقى وسيلةَ الاختيار، **ولا معاينةً تلقائيّةً من لمسةٍ ملتبسة.**
 *
 * # **وهذا صنفٌ خالص**
 *
 * **لا MapLibre ولا Android** — إحداثيّاتُ شاشةٍ وأرقام. **فيُختبر
 * بلا جهاز** (البند ١٤).
 */
object TapGeometry {

    /** **نقطةٌ على الشاشة** — بالبكسل. */
    data class ScreenPoint(val x: Float, val y: Float)

    data class Tuning(
        /**
         * **عتبةُ الالتباس** — البند ١٥.
         *
         * **بالـdp لا بالبكسل** — فشاشةٌ كثيفةٌ لا تصير أدقَّ إصبعاً.
         *
         * **وثمانيةٌ سُدسُ هدف اللمس** (٤٨dp): **فرقٌ أقلُّ منها لا
         * يفرّق بين خطّين تحتَ إصبعٍ عرضُه عشرةُ ملّيمترات.**
         */
        val ambiguityDp: Float = 8f,

        /**
         * **وأقصى بُعدٍ يُعدُّ لمسةً على الخطّ.**
         *
         * **نصفُ هدف اللمس** — فما وراءه ليس ما قصده.
         */
        val maxDistanceDp: Float = 24f,
    )

    /** **نتيجةُ اللمس** — للتشخيص كما للقرار. */
    sealed interface Hit {
        /** **خطٌّ واحدٌ واضح.** */
        data class One(val routeId: String, val distancePx: Float) : Hit

        /** **خطّان متقاربان** — البطاقاتُ تحسم (البند ١٥). */
        data class Ambiguous(val routeIds: List<String>, val deltaPx: Float) : Hit

        /** **لا خطَّ تحتَ اللمسة.** */
        data object None : Hit
    }

    /**
     * **يختار خطّاً أو لا يختار.**
     *
     * @param candidates **مرشَّحون من `queryRenderedFeatures`** —
     *   مصفاةٌ أوّليّةٌ لا حكم (البند ١٦).
     */
    fun classify(
        candidates: List<Pair<String, List<ScreenPoint>>>,
        tapX: Float,
        tapY: Float,
        density: Float,
        tuning: Tuning = Tuning(),
    ): Hit {
        if (candidates.isEmpty()) return Hit.None

        val maxPx = tuning.maxDistanceDp * density
        val ambiguityPx = tuning.ambiguityDp * density

        // **وأقربُ نقطةٍ إلى أيّ قطعةٍ مرسومة** — لا إلى رأس.
        val scored = candidates
            .map { (id, pts) -> id to distanceToPath(pts, tapX, tapY) }
            .filter { it.second <= maxPx }
            .sortedBy { it.second }

        if (scored.isEmpty()) return Hit.None
        if (scored.size == 1) return Hit.One(scored[0].first, scored[0].second)

        val delta = abs(scored[1].second - scored[0].second)
        if (delta < ambiguityPx) {
            /**
             * **ولا يُؤخذ الأوّلُ في الترتيب** — أمرُ المالك: «لا تخمّن
             * عشوائيًا». **فترتيبُ المتساويين ليس معلومةً.**
             */
            return Hit.Ambiguous(scored.map { it.first }, delta)
        }
        return Hit.One(scored[0].first, scored[0].second)
    }

    /** **معرِّفُ الخطّ الواضح** — أو `null` عند اللبس أو الغياب. */
    fun pick(
        candidates: List<Pair<String, List<ScreenPoint>>>,
        tapX: Float,
        tapY: Float,
        density: Float,
        tuning: Tuning = Tuning(),
    ): String? = (classify(candidates, tapX, tapY, density, tuning) as? Hit.One)?.routeId

    /**
     * **أدنى مسافةٍ من نقطةٍ إلى مسارٍ مرسوم** — بالبكسل.
     *
     * **إلى القطع لا إلى الرؤوس.**
     */
    fun distanceToPath(points: List<ScreenPoint>, x: Float, y: Float): Float {
        if (points.isEmpty()) return Float.MAX_VALUE
        if (points.size == 1) return hypot(points[0].x - x, points[0].y - y)
        var best = Float.MAX_VALUE
        for (i in 0 until points.size - 1) {
            val d = distanceToSegment(points[i], points[i + 1], x, y)
            if (d < best) best = d
        }
        return best
    }

    /** **مسافةُ نقطةٍ من قطعةٍ مستقيمة** — بإسقاطٍ محصورٍ بين الطرفين. */
    fun distanceToSegment(a: ScreenPoint, b: ScreenPoint, x: Float, y: Float): Float {
        val dx = b.x - a.x
        val dy = b.y - a.y
        if (dx == 0f && dy == 0f) return hypot(a.x - x, a.y - y)
        var t = ((x - a.x) * dx + (y - a.y) * dy) / (dx * dx + dy * dy)
        t = t.coerceIn(0f, 1f)
        return hypot(x - (a.x + t * dx), y - (a.y + t * dy))
    }
}
