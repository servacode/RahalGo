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
 * ٢. **وسرعتُه هو تُخلَط معه** حين يسير (≥ ٣م/ث): نصفٌ ونصف — **فمن يسير بـ٢٠ كم/س
 *    لا يُقال له زمنُ سيّارةٍ بـ٦٠.**
 * ٣. **ولا أسرعَ من ٤٥ كم/س وسطيّاً في المدينة** — حدٌّ أدنى للزمن لا يكذب على الزبون.
 * ٤. **والمعروضُ يقترب من المحسوب ولا يقفز** — ربعُ الفرق في كلّ قراءة، **إلّا أن
 *    يتبدّل الطريقُ** (إعادة حساب) فيُأخذ الجديد كما هو.
 */
class EtaSmoother {

    private var shownSec = Double.NaN
    private var lastTotalM = Double.NaN

    fun reset() {
        shownSec = Double.NaN
        lastTotalM = Double.NaN
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
        var target = if (routeSec >= 0) routeSec else remainingM / FALLBACK_MPS
        if (speedMps >= MOVING_MPS) {
            target = 0.5 * target + 0.5 * (remainingM / speedMps)
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
        const val MAX_CITY_MPS = 12.5 // ٤٥ كم/س
        const val FALLBACK_MPS = 7.0 // ٢٥ كم/س
        const val ALPHA = 0.25
    }
}
