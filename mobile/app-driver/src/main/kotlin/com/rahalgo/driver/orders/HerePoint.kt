package com.rahalgo.driver.orders

import com.rahalgo.ui.LastPoint

/** **أقدمُ موضعٍ يُقبل لِما يُرسَل «من مكاني»** — دقيقتان، كنبضة الخادم التي تصحّحه. */
internal const val HERE_MAX_AGE_MS = 120_000L

/**
 * **أقدمُ قراءةٍ تُقبل من «اقرأ موضعي الآن»** — عشرُ ثوانٍ (مراجعة ٢٠٢٦-١١-٠٥). **وعمرُها من
 * لحظة القياس** (`elapsedRealtimeNanos`) لا من لحظة الوصول.
 */
internal const val HERE_FRESH_MAX_AGE_MS = 10_000L

/** **مهلةُ «اقرأ موضعي الآن»** — عشرُ ثوانٍ ثمّ يُرجع إلى آخر موضعٍ حديث. */
internal const val HERE_FRESH_TIMEOUT_MS = 10_000L

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

/**
 * ══════════════════════════════════════════════════════════════════════
 * **موضعُ المعلم: قراءةٌ تُقاس الآن قبل آخر موضعٍ معروف** (مراجعة ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **`LastPoint.atMs` وقتُ وصول القراءة إلى التطبيق لا وقتُ قياسها** — وبين الطلبات تُجمَّع
 * القراءاتُ حتّى دقيقتين (`setMaxUpdateDelayMillis`) **ولا تأتي جديدةٌ إلّا بعد عشرين متراً.**
 * فالسائقُ الذي وقف عند الجامع وأرسل في أوّل دقيقة **كان معلمُه يُحفظ قبله بمئات الأمتار**، ولا
 * يلحقه تصحيحُ الخادم (يقارن بالنقطة نفسِها)، **فيفوت فحصَ «موجود من قبل» مكرَّرٌ.**
 *
 * **فالملاحةُ ما دامت شغّالة** (قراءةٌ كلَّ ثانية)، **وإلّا قراءةٌ تُقاس الآن** (`fresh`)، **وإلّا**
 * — لا إشارةَ في عشر ثوانٍ — **آخرُ موضعٍ حديثٍ للدوام** (`pickHere`).
 */
internal fun pickLandmarkHere(
    following: Boolean,
    nav: Pair<Double, Double>?,
    fresh: Pair<Double, Double>?,
    last: LastPoint.Point?,
    nowElapsedMs: Long,
): Pair<Double, Double>? {
    if (following && nav != null) return nav
    if (fresh != null) return fresh
    return pickHere(following = false, nav = null, last = last, nowElapsedMs = nowElapsedMs)
}

/**
 * **أقِيست القراءةُ للتوّ؟** — بعمرها من لحظة القياس (`Location.elapsedRealtimeNanos`)؛
 * `getCurrentLocation` قد تردّ مخبوءاً قديماً. **وصفرٌ أو قراءةٌ من المستقبل لا تُقبل.**
 */
internal fun measuredJustNow(
    fixElapsedNanos: Long,
    nowElapsedNanos: Long,
    maxAgeMs: Long = HERE_FRESH_MAX_AGE_MS,
): Boolean {
    if (fixElapsedNanos <= 0L || fixElapsedNanos > nowElapsedNanos) return false
    return (nowElapsedNanos - fixElapsedNanos) / 1_000_000 <= maxAgeMs
}
