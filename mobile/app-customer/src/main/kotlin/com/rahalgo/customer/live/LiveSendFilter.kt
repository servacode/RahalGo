package com.rahalgo.customer.live

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لا يُرسَل إلّا الدقيق — وأحسنُ ما وُجد إن طال الانتظار** (قرارُ المالك ٢٠٢٦-١٠-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **المالكُ يريد الكابتنَ عند الزبون بالأمتار.** **والهاتفُ يقيس ٣–٥ أمتارٍ في الشارع
 * و٢٠–٥٠ داخلَ البيت** — فقراءةٌ بدقّةِ أربعين متراً تقول «هو في الحارة» لا «هو عند
 * الباب».
 *
 * # فثلاثةُ أحكام
 *
 *   - **الدقيقُ يُرسَل** — دقّتُه دون حدّ اللوحة (`orders.live_location_max_accuracy_m`،
 *     افتراضُه ٣٠ ويصل من ردّ المحرّك نفسِه).
 *   - **والتقريبيُّ يُحبَس** — ويُحفظ أحسنُه.
 *   - **وثلاثون ثانيةً بلا دقيق؟ يُرسَل أحسنُ ما فيها** — **فلا يبقى الكابتنُ بلا موضع**،
 *     والمحرّكُ يَسِمه «تقريبيّاً» فيُرى ولا يتحوّل إليه الطريق.
 *
 * **ولا أسرعَ من مرّةٍ كلَّ ثماني ثوانٍ** مهما سلّم النظام.
 *
 * **صافٍ بلا أندرويد** — الوقتُ يُمرَّر، **فيُختبر بلا جهاز** (`LiveSendFilterTest`).
 */
class LiveSendFilter(
    /** **حدُّ الدقّة بالمتر** — ويُبدَّل بما يردّه المحرّك. */
    @Volatile var maxAccuracyM: Double = DEFAULT_MAX_ACCURACY_M,
) {
    /** **قراءةٌ من الهاتف** — `accuracyM` فارغةٌ إن لم يقل الجهازُ كم يخطئ. */
    data class Reading(val lat: Double, val lng: Double, val accuracyM: Double?, val atMs: Long)

    private var windowStart = -1L
    private var lastSent = -1L
    private var best: Reading? = null

    /** **أدقيقةٌ هي؟** — **ومجهولةُ الدقّة ليست دقيقة.** */
    fun precise(r: Reading): Boolean = r.accuracyM != null && r.accuracyM <= maxAccuracyM

    /**
     * **قراءةٌ جديدة** — ويردّ ما يُرسَل الآن، أو `null`: لا شيء.
     */
    @Synchronized
    fun offer(r: Reading): Reading? {
        if (windowStart < 0) windowStart = r.atMs
        if (best == null || better(r, best!!)) best = r
        if (lastSent >= 0 && r.atMs - lastSent < MIN_GAP_MS) return null
        if (precise(r)) return sent(r, r.atMs)
        // **ودقيقةٌ وصلت قبل أوانها** — تُرسَل الآن لا بعد ثلاثين ثانية.
        val b = best!!
        if (precise(b)) return sent(b, r.atMs)
        if (r.atMs - windowStart >= FALLBACK_MS) return sent(b, r.atMs)
        return null
    }

    /** **طلبٌ آخرُ أو مشاركةٌ جديدة** — يُنسى ما مضى. */
    @Synchronized
    fun reset() {
        windowStart = -1L
        lastSent = -1L
        best = null
    }

    /** **أُرسلت الآن** — والنافذةُ والفجوةُ تُقاسان من وقت الإرسال لا من وقت القراءة. */
    private fun sent(r: Reading, nowMs: Long): Reading {
        lastSent = nowMs
        windowStart = nowMs
        best = null
        return r
    }

    /** **أصغرُ دقّةٍ أحسن** — **ومجهولةُ الدقّة أسوأُ الكلّ**، والأحدثُ عند التساوي. */
    private fun better(a: Reading, b: Reading): Boolean {
        val x = a.accuracyM ?: Double.MAX_VALUE
        val y = b.accuracyM ?: Double.MAX_VALUE
        return x <= y
    }

    companion object {
        /** **افتراضُ الحدّ** — كافتراض اللوحة: ثلاثون متراً. */
        const val DEFAULT_MAX_ACCURACY_M = 30.0

        /** **ثلاثون ثانيةً بلا دقيقٍ ثمّ أحسنُ ما وُجد** — قرارُ المالك. */
        const val FALLBACK_MS = 30_000L

        /** **ولا أسرعَ من هذا** — النظامُ قد يسلّم أكثرَ ممّا طُلب. */
        const val MIN_GAP_MS = 8_000L
    }
}
