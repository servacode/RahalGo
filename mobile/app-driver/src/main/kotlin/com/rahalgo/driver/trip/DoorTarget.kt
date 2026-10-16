package com.rahalgo.driver.trip

import com.rahalgo.shared.model.CustomerLive

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إلى أين يُقاد بعد الاستلام — الدبّوسُ أم الزبونُ الآن** (قرارُ المالك ٢٠٢٦-١٠-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **دبابيسُ الرقّة كثيراً ما تخطئ** — فإن شارك الزبونُ موقعَه المباشر وكان أبعدَ
 * من حدّ اللوحة (`orders.live_location_switch_m`) عن الدبّوس **قِيد السائقُ إليه.**
 *
 * # والحكمُ حكمُ الخادم نفسُه
 *
 * **البُعدُ والحدُّ يصلان من المحرّك** (`pin_gap_m` · `switch_m`) — وهو يحسب الطريقَ
 * بالشرط نفسِه (`LivePoint.Switch`). **فلا يرسم التطبيقُ وجهةً والخادمُ يحسب غيرَها.**
 *
 * # وما شاخ يعود إلى الدبّوس — بكلمة
 *
 * **دقيقتان بلا نقطةٍ جديدة** — **والعمرُ يُقاس هنا بين قراءتين** لا في الخادم وحدَه:
 * زبونٌ أطفأ هاتفَه تبقى آخرُ نقطةٍ في يد السائق حتّى القراءة التالية.
 * **ومن كان يُقاد إلى الزبون ثمّ عاد إلى الدبّوس يُقال له لماذا** (`staleLive`).
 */
sealed interface DoorTarget {
    /** **الدبّوس** — و`staleLive`: كان يُقاد إلى الزبون فشاخ موقعُه. */
    data class Pin(val staleLive: Boolean = false) : DoorTarget

    /** **موقعُ الزبون الآن** — بعيدٌ عن الدبّوس وحديث. */
    data class Live(val lat: Double, val lng: Double, val ageSec: Long) : DoorTarget
}

/** **عمرُ النقطة الأقصى** — كـ`orders.LiveFreshFor` في المحرّك: دقيقتان. */
const val LIVE_FRESH_SEC = 120L

/**
 * **عمرُ النقطة بالثواني** — و`null` إن لم يُقرأ وقتُها: **ونقطةٌ بلا وقتٍ لا تُصدَّق.**
 *
 * **وساعةُ الهاتف قد تسبق الخادم** — فالسالبُ صفر، لا «من المستقبل».
 */
fun liveAgeSec(live: CustomerLive, nowMs: Long): Long? {
    val at = runCatching { java.time.OffsetDateTime.parse(live.at).toInstant().toEpochMilli() }
        .getOrNull() ?: return null
    return ((nowMs - at) / 1000).coerceAtLeast(0)
}

/** **أحديثةٌ هي؟** — دون دقيقتين، **ومجهولةُ الوقت شائخة.** */
fun liveFresh(live: CustomerLive?, nowMs: Long): Boolean {
    val age = live?.let { liveAgeSec(it, nowMs) } ?: return false
    return age <= LIVE_FRESH_SEC
}

/**
 * **الوجهةُ بعد الاستلام.**
 *
 * **ويتحوّل الطريقُ إلى الزبون بشرطين** (قرارُ المالك ٢٠٢٦-١٠-١٦): **أبعدُ من حدّ اللوحة
 * عن الدبّوس، ودقيقٌ** (`precise` من المحرّك). **والتقريبيُّ يبقى علامةً على الخريطة
 * بدائرة دقّته** — ولا يسحب السائقَ عن دبّوسٍ قد يكون صحيحاً.
 *
 * @param hadLive **أكان يُقاد إلى الزبون قبل هذه القراءة؟** — فإن غاب الموقعُ أو شاخ
 * قيل له «رجعنا للعنوان»، **ومن لم يُقَد إليه قطّ لا يُشغَل بكلمة.**
 */
fun doorTarget(live: CustomerLive?, hadLive: Boolean, nowMs: Long): DoorTarget {
    if (live == null) return DoorTarget.Pin(staleLive = hadLive)
    val age = liveAgeSec(live, nowMs)
    if (age == null || age > LIVE_FRESH_SEC) return DoorTarget.Pin(staleLive = hadLive)
    if (live.precise && live.pinGapM > live.switchM.toDouble()) return DoorTarget.Live(live.lat, live.lng, age)
    return DoorTarget.Pin()
}
