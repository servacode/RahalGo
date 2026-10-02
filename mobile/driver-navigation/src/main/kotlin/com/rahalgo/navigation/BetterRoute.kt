package com.rahalgo.navigation

import kotlin.math.max

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طريقٌ أفضل أثناء السير** — كغوغل: يُعرض خطّاً أزرقَ فاتحاً والسائقُ يختار
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-١٠-٠٢: «بنفس وقت المشي لازم يبحث عن طرقٍ أقصر… يطلع طريقاً أزرقَ
 *  باهتاً والشخصُ يختار أو يبقى على مساره».)
 *
 * **تُسأل كلَّ دقيقتين من موضعه الآن** — وما يردّه الخادمُ (الموصى به وبدائلُه) يُقاس على
 * **ما بقي من طريقه هو**:
 *
 * - **ما يطابق طريقَه لا يُعرض** — ليس بديلاً.
 * - **ولا يُعرض إلّا ما يوفّر فعلاً**: ثلاثين ثانيةً على الأقلّ **و**عُشرَ الوقت الباقي —
 *   فرقٌ صغيرٌ لا يستحقّ أن يُشغَل به وهو يقود.
 * - **ونقطةُ الافتراق تُحسب على طريقه** (تقدّمُه الآن + ما يشتركان فيه) — فيختفي البديلُ
 *   إذا جاوزها، **كما تختفي بدائلُ بداية الرحلة.**
 */
object BetterRoute {

    const val MIN_SAVE_S = 30.0
    const val MIN_SAVE_FRACTION = 0.10

    /** **أبعدُ من هذا عن طريقه = افترق.** */
    const val SPLIT_M = 25.0

    /**
     * @param current الطريقُ الذي يُرشَد عليه.
     * @param progressM تقدّمُه عليه الآن.
     * @param remainingSec الوقتُ الباقي عليه بتقدير المحرّك نفسِه (فالمقارنةُ بوحدةٍ واحدة).
     * @param candidates ما ردّه الخادمُ من موضعه الآن.
     * @return الأفضلُ وحدَه، بفرقه عن طريقه ونقطةِ افتراقه عليه — الأسرعُ أوّلاً.
     */
    fun evaluate(
        current: NavRoute,
        progressM: Double,
        remainingSec: Double,
        candidates: List<RouteOption>,
    ): List<RouteOption> {
        if (remainingSec <= 0 || progressM < 0) return emptyList()
        val remainingM = max(0.0, current.totalM - progressM)
        val need = max(MIN_SAVE_S, MIN_SAVE_FRACTION * remainingSec)
        return candidates.mapNotNull { c ->
            val save = remainingSec - c.engineDurationS
            if (save < need) return@mapNotNull null
            val shared = sharedPrefixM(current, progressM, c.route)
            // **ويطابق طريقَه كلَّه تقريباً ⇒ ليس بديلاً** (فرقُ الوقت من تقدير المحرّك لا من طريقٍ آخر).
            if (shared >= c.route.totalM - SPLIT_M) return@mapNotNull null
            c.copy(
                deltaDurationS = c.engineDurationS - remainingSec,
                deltaDistanceM = c.distanceM - remainingM,
                decisionDivergenceM = progressM + shared,
                firstDivergenceM = progressM + shared,
            )
        }.sortedBy { it.engineDurationS }
    }

    /**
     * **كم يمشي المرشَّحُ على طريقه قبل أن يفترق** — أوّلُ رأسٍ يبعد أكثرَ من [SPLIT_M]
     * عمّا بقي من طريقه.
     */
    fun sharedPrefixM(current: NavRoute, progressM: Double, candidate: NavRoute): Double {
        val g = current.geometry
        val c = current.cumulativeM
        var from = 0
        while (from < c.size - 1 && c[from + 1] < progressM) from++
        val cg = candidate.geometry
        val cc = candidate.cumulativeM
        for (i in cg.indices) {
            if (distanceToPolyline(cg[i], g, from) > SPLIT_M) {
                return if (i == 0) 0.0 else cc[i - 1]
            }
        }
        return candidate.totalM
    }

    private fun distanceToPolyline(p: GeoPoint, g: List<GeoPoint>, from: Int): Double {
        var best = Double.MAX_VALUE
        for (i in from until g.size - 1) {
            val d = segmentDistance(p, g[i], g[i + 1])
            if (d < best) best = d
        }
        return best
    }

    /** **بُعدُ نقطةٍ عن مقطع** — بإسقاطٍ مستوٍ محلّيّ (يكفي لمئات الأمتار). */
    private fun segmentDistance(p: GeoPoint, a: GeoPoint, b: GeoPoint): Double {
        val k = Math.cos(Math.toRadians(p.lat))
        val ax = (a.lng - p.lng) * 111_320.0 * k
        val ay = (a.lat - p.lat) * 111_320.0
        val bx = (b.lng - p.lng) * 111_320.0 * k
        val by = (b.lat - p.lat) * 111_320.0
        val dx = bx - ax
        val dy = by - ay
        val len2 = dx * dx + dy * dy
        val t = if (len2 == 0.0) 0.0 else (-(ax * dx + ay * dy) / len2).coerceIn(0.0, 1.0)
        val x = ax + t * dx
        val y = ay + t * dy
        return Math.sqrt(x * x + y * y)
    }
}
