package com.rahalgo.driver.orders

import com.rahalgo.ui.LastPoint

/** **أقدمُ موضعٍ يُقبل لِما يُرسَل «من مكاني»** — دقيقتان، كنبضة الخادم التي تصحّحه. */
internal const val HERE_MAX_AGE_MS = 120_000L

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«من مكاني الآن» — لا من مكان آخر تسليم** (قرارُ المالك ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت آخرُ قراءةٍ للملاحة تُؤخذ ولو انطفأت الملاحة** — وهي تنطفئ مع آخر طلبٍ ولا تُمحى
 * قراءتُها. **فمعلمٌ يُرسَل من القائمة بين طلبين يُحفظ مكانَ آخر تسليم**، وكلُّ ما يُرسَل قبل
 * الطلب التالي في النقطة نفسِها بالضبط. (`routeOrigin` يشرط الملاحةَ من قبل.)
 *
 * **فالملاحةُ ما دامت شغّالة**، وإلّا آخرُ موضعٍ لخدمة الدوام **إن كان حديثاً**، وإلّا `null`
 * — **فيُطلب موضعٌ جديدٌ من النظام** (`hereFresh`). **وموضعٌ بلا وقتٍ لا يُعرف عمرُه** فلا يُقبل.
 *
 * **دالّةٌ صافية** — والوقتُ `elapsedRealtime` يُمرَّر، فتُقاس بلا جهاز.
 */
internal fun pickHere(
    following: Boolean,
    nav: Pair<Double, Double>?,
    last: LastPoint.Point?,
    nowElapsedMs: Long,
    maxAgeMs: Long = HERE_MAX_AGE_MS,
): Pair<Double, Double>? {
    if (following && nav != null) return nav
    val p = last ?: return null
    if (p.atMs <= 0L || p.atMs > nowElapsedMs || nowElapsedMs - p.atMs > maxAgeMs) return null
    return p.lat to p.lng
}
