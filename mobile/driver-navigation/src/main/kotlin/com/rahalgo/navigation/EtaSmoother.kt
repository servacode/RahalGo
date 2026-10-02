package com.rahalgo.navigation

import kotlin.math.max
import kotlin.math.min

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الوقتُ الباقي — يتبع السرعةَ ولا يقفز** (المرحلة ٤ من `docs/navigation/PLAN.md`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-١٠-٠٢: «المسافةُ والوقتُ حسبَ السرعة» — **وقِيس**: «١٫٣ كم · ١ د»
 *  ثابتان طوالَ الرحلة، **و«١ د» لـ١٫٣ كم تعني ٧٨ كم/س**: زمنُ محرّك الطرق للسيّارة.)
 *
 * # ما يصنعه
 *
 * ١. **زمنُ الطريق أساسٌ** (`remainingSec` من المحرّك) — يعرف الشوارعَ البطيئة.
 * ٢. **ومعدّلُ سرعته هو** (وهو يسير وحدَه، لا الوقفات) يُخلَط معه، **ويزيد وزنُه كلّما
 *    طال سيرُه** حتّى ثلاثةِ أرباع بعد دقيقة — طلبُ المالك: «الموتورُ قد يمشي ٣٠ أو ٥٠
 *    أو ٧٠، والقياسُ حسبَ السرعة مثلَ غوغل».
 * ٣. **ولا أسرعَ من ٩٠ كم/س وسطيّاً** — حدٌّ يمنع رقماً مستحيلاً لا أكثر. (كان ٤٥،
 *    **فمن يسير بـ٧٠ يُقال له وقتٌ أطولُ من الحقيقيّ.**)
 * ٤. **والمعروضُ يقترب من المحسوب ولا يقفز** — ربعُ الفرق في كلّ قراءة، **إلّا أن
 *    يتبدّل الطريقُ** (إعادة حساب) فيُأخذ الجديد كما هو.
 */
class EtaSmoother {

    private var shownSec = Double.NaN
    private var lastTotalM = Double.NaN

    /** **معدّلُ سرعته وهو يسير** — نحو دقيقةٍ أخيرة، والوقفاتُ لا تدخل. */
    var paceMps = Double.NaN
        private set
    private var movingFixes = 0

    fun reset() {
        shownSec = Double.NaN
        lastTotalM = Double.NaN
        paceMps = Double.NaN
        movingFixes = 0
    }

    /**
     * @param remainingM ما بقي بالمتر.
     * @param routeSec زمنُ المحرّك لما بقي — سالبٌ: لا يُعرف.
     * @param speedMps السرعةُ الملساء — صفرٌ عند الوقوف.
     * @param routeTotalM طولُ الطريق كلِّه — تبدّلُه يعني طريقاً جديداً.
     */
    fun update(remainingM: Double, routeSec: Double, speedMps: Double, routeTotalM: Double): Double {
        if (remainingM < 0) return -1.0
        val floorSec = remainingM / MAX_CITY_MPS
        if (speedMps >= MOVING_MPS) {
            paceMps = if (paceMps.isNaN()) speedMps else paceMps + PACE_ALPHA * (speedMps - paceMps)
            movingFixes++
        }
        var target = if (routeSec >= 0) routeSec else remainingM / FALLBACK_MPS
        if (!paceMps.isNaN()) {
            val w = MAX_PACE_WEIGHT * (movingFixes.toDouble() / PACE_RAMP_FIXES).coerceAtMost(1.0)
            target = (1 - w) * target + w * (remainingM / paceMps)
        }
        target = max(target, floorSec)
        val fresh = shownSec.isNaN() || routeTotalM != lastTotalM
        lastTotalM = routeTotalM
        shownSec = if (fresh) target else shownSec + ALPHA * (target - shownSec)
        // **ولا يزيد المعروضُ على المحسوب كثيراً عند الاقتراب** — آخرُ الأمتار تُقرأ صادقة.
        shownSec = min(shownSec, max(target * 1.5, target + 60.0))
        return shownSec
    }

    companion object {
        const val MOVING_MPS = 3.0
        const val MAX_CITY_MPS = 25.0 // ٩٠ كم/س — حدُّ المستحيل لا حدُّ المدينة
        const val FALLBACK_MPS = 7.0 // ٢٥ كم/س
        const val ALPHA = 0.25
        const val PACE_ALPHA = 0.05 // ≈ دقيقةٌ أخيرةٌ بقراءةٍ كلَّ ثانية
        const val PACE_RAMP_FIXES = 60
        const val MAX_PACE_WEIGHT = 0.75
    }
}
