package com.rahalgo.navigation

import kotlin.math.abs
import kotlin.math.roundToLong

/**
 * ══════════════════════════════════════════════════════════════════
 * **أرقامُ المسار — صياغةٌ واحدةٌ لا صفاتٌ مطلقة**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ واجهة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ٧ و٨ و٩ و٣٣.)
 *
 * **أمرُ المالك نصّاً**: «Numbers, not unprovable labels».
 *
 * # **ولا «الأسرع» ولا «الأقصر»**
 *
 * **وزنُ محرّكنا `routability` لا `duration`** — وقِيس الفرقُ ٧٫٢٪ من
 * المسارات. **فمن سمّى مساراً «الأسرع» وصف ما لم يُحسَب.**
 *
 * **فالبديلُ يُوصف بفرقه**: «أقصرُ بـ٤٫٥ كم · +١٢ د».
 *
 * # **ولماذا هنا لا في Composable**
 *
 * **البند ٩**: «لا hardcode النصوص داخل Composable… استخدم formatter
 * مركزيًا لـ: distance · duration · signed deltas. ولا تنشئ `+-3` أو
 * `- -`».
 *
 * **وذلك ليس تنميقاً**: `"+" + "-3"` تعطي `+-3`، **وقيمةٌ سالبةٌ
 * تُنسَّق بعلامتها ثمّ تُسبَق بأخرى.** فالصياغةُ في موضعٍ واحدٍ
 * يُختبر.
 *
 * # **والأرقامُ من المحرّك لا من العرض** (البند ٧)
 *
 * `engineDurationS` و`distanceM` — **لا `durationS` المعدَّلةُ بسرعة
 * السائق.** **وإلّا صارت كلُّ المسارات بالسرعة نفسِها فاختفى الفرق.**
 */
object RouteMetricText {

    /**
     * **ما يُعرض عن خيار** — مفاتيحُ نصٍّ وقيمٌ مصاغة.
     *
     * **ولا نصَّ عربيٌّ هنا** — الصياغةُ تعطي أرقاماً ووحداتٍ،
     * **والتركيبُ في موارد النصّ** فيبقى RTL على حاله.
     */
    data class Metrics(
        /** **المدّة** — مثل «١٥ د». */
        val duration: String,
        /** **المسافة** — مثل «١١٫٠ كم». */
        val distance: String,
    )

    /** **فرقٌ مُوقَّع** — النوعُ والمقدارُ مفصولان عن النصّ. */
    data class Delta(
        val kind: Kind,
        /** **المقدارُ مصاغاً بلا علامة** — «٤٫٥ كم» أو «١٢ د». */
        val magnitude: String,
    ) {
        enum class Kind {
            SHORTER,
            LONGER,
            FASTER,
            SLOWER,
        }
    }

    // ══════════════════════════════════════════════════════════════
    // **الصياغة**
    // ══════════════════════════════════════════════════════════════

    /**
     * **المسافة** — بالمتر دونَ الكيلومتر، وبمنزلةٍ فوقه.
     *
     * **ولا كسورَ في المئات**: «٨٥٠ م» أوضحُ من «٠٫٨٥ كم» لسائقٍ
     * يقود.
     */
    fun distance(meters: Double): String {
        val m = abs(meters)
        return when {
            m < 1000 -> "${roundTo(m, 10.0).toLong()} م"
            m < 10_000 -> "${trim(roundTo(m / 1000.0, 0.1))} كم"
            else -> "${roundTo(m / 1000.0, 1.0).toLong()} كم"
        }
    }

    /**
     * **المدّة** — بالدقيقة، وبالساعة فوق الستّين.
     *
     * **ولا ثوانٍ**: السائقُ لا يقرؤها، **والفرقُ بالثانية ضجيج.**
     */
    fun duration(seconds: Double): String {
        val s = abs(seconds)
        val minutes = (s / 60.0).roundToLong()
        if (minutes < 60) return "$minutes د"
        val h = minutes / 60
        val m = minutes % 60
        return if (m == 0L) "$h س" else "$h س $m د"
    }

    /** **مقاييسُ خيارٍ للعرض.** */
    fun metricsOf(option: RouteOption): Metrics = Metrics(
        duration = duration(option.engineDurationS),
        distance = distance(option.distanceM),
    )

    /**
     * **فرقُ المسافة** — أو `null` إن كان دونَ عتبة العرض.
     *
     * **وفرقٌ صغيرٌ لا يُعرض**: «أقصرُ بـ٢٠ م» ليست معلومةً لسائق،
     * **وتزحم بطاقةً صغيرة.**
     */
    fun distanceDelta(meters: Double, minMeters: Double = MIN_DISTANCE_DELTA_M): Delta? {
        if (abs(meters) < minMeters) return null
        return Delta(
            kind = if (meters < 0) Delta.Kind.SHORTER else Delta.Kind.LONGER,
            magnitude = distance(meters),
        )
    }

    /** **وفرقُ المدّة** — من مدّة المحرّك لا من المعدَّلة. */
    fun durationDelta(seconds: Double, minSeconds: Double = MIN_DURATION_DELTA_S): Delta? {
        if (abs(seconds) < minSeconds) return null
        return Delta(
            kind = if (seconds < 0) Delta.Kind.FASTER else Delta.Kind.SLOWER,
            magnitude = duration(seconds),
        )
    }

    /**
     * **فروقُ بديلٍ عن الموصى به** — مرتَّبةً كما تُقرأ.
     *
     * **المسافةُ أوّلاً ثمّ الزمن** — فالمسافةُ هي المقايضةُ الغالبةُ
     * في بياناتنا: **٢٩ من ٣١ بديلاً غيرِ مهيمَنٍ عليه «أقصرُ
     * وأبطأ».**
     *
     * **وقائمةٌ فارغةٌ ممكنة**: بديلٌ يختلف هندسةً ويتقارب رقماً.
     */
    fun deltasOf(option: RouteOption): List<Delta> = listOfNotNull(
        distanceDelta(option.deltaDistanceM),
        durationDelta(option.deltaDurationS),
    )

    /** **أدنى فرقٍ يُعرض** — دونَه ضجيجٌ لا معلومة. */
    const val MIN_DISTANCE_DELTA_M = 100.0
    const val MIN_DURATION_DELTA_S = 30.0

    private fun roundTo(v: Double, step: Double): Double =
        (v / step).roundToLong() * step

    /** **يحذف الصفرَ العالق** — «٤٫٠ كم» تصير «٤ كم». */
    private fun trim(v: Double): String {
        val rounded = (v * 10).roundToLong()
        val whole = rounded / 10
        val frac = rounded % 10
        return if (frac == 0L) "$whole" else "$whole٫$frac"
    }
}
