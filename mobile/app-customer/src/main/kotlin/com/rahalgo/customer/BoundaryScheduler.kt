package com.rahalgo.customer

import android.os.SystemClock
import com.rahalgo.shared.model.Availability
import com.rahalgo.shared.model.Ordering
import com.rahalgo.ui.Refresh
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.time.OffsetDateTime

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مُوقِّتُ الحدّ — التطبيقُ يتحوّل عند حدِّ الوقت بنفسِه** (Batch 5، C)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العلّةُ التي يسدّها
 *
 * **حالُ الاستقبال تُحسَب في الخادم لحظةً بلحظة، لكنّها لا تُدفَع عند
 * الحدّ**: لا مهمّةَ دوريّةً في الخادم تُطلق حدثاً حين ينتهي الدوام،
 * والبثُّ لا يصل إلّا على كتابةِ أدمن. **والتطبيقُ يُجدّد نفسَه عند
 * العودة/الكتابة/فعلِ المستخدم فقط.** فمن أبقى التطبيقَ مفتوحاً بلا لمسٍ
 * حتّى حدِّ الإغلاق القادم **بقيت شاشتُه تقول «مفتوح» بعد أن أُغلق.**
 *
 * # الحلّ — مُوقِّتٌ واحدٌ لأقربِ حدّ
 *
 * **يسلّح مؤقّتاً واحداً على أقربِ حدٍّ معلومٍ** (إغلاقُ المنصّة أو
 * المنطقة، أو فتحُها إن كانت مغلقة)، فإذا بلغه **أجبر إعادةَ التحقّق**
 * (تجديدُ الاستقبالِ + نبضةُ التحديثِ العامّة) فتنقلب الشاشةُ عند الحدّ
 * تماماً دون حدثٍ من الخادم ولا لمسٍ من المستخدم.
 *
 * # القواعد
 *
 *   • **ساعةُ الخادمُ هي المرجع** (`server_time`) لا ساعةُ الجهاز —
 *     والانتظارُ يُقاس بالساعةِ الرتيبة (`elapsedRealtime`) فلا يتأثّر
 *     بتبديلِ ساعةِ الجهاز.
 *   • **مؤقّتٌ واحدٌ فقط** — يُلغى ويُعاد تسليحُه عند كلّ قراءةٍ سلطويّة.
 *   • **يُلغى في الخلفيّة**؛ وعند العودةِ يُجبَر تجديدٌ ثمّ يُعاد التسليح.
 *   • **لا حلقةَ استطلاعٍ ولا مؤقّتٌ دوريّ** — نومٌ واحدٌ حتّى الحدّ.
 *   • **يعمل الحدّان: الإغلاقُ القادمُ والفتحُ القادم** — أقربُهما أوّلاً،
 *     ثمّ يُحسَب التالي بعد إعادةِ التحقّق.
 *   • **البثُّ اللحظيّ وإعادةُ الوصلِ يبقيان مسارَي تصحيحٍ إضافيّين.**
 */
object BoundaryScheduler {

    private var scope: CoroutineScope? = null
    private var foreground = false
    private var job: Job? = null

    // **مرجعُ الساعة** — لحظةُ الخادمِ عند آخرِ مزامنةِ منصّة، وقرينتُها
    // الرتيبةُ محليّاً. **والحدودُ مطلقةٌ بتوقيتِ الخادم فتُقاس بهما.**
    private var anchorServerMs = 0L
    private var anchorElapsed = 0L

    // **حدُّ كلِّ طبقةٍ** بتوقيتِ الخادمِ المطلق (٠ = لا حدَّ معلوم).
    private var platformBoundaryMs = 0L
    private var zoneBoundaryMs = 0L

    /** **يُربَط بنطاقِ التطبيق مرّةً** — عليه يُطلَق نومُ المؤقّت. */
    fun bind(s: CoroutineScope) {
        scope = s
    }

    /** **بعد كلِّ قراءةِ استقبالٍ ناجحة** — المنصّةُ تحمل `server_time`. */
    fun syncPlatform(o: Ordering) {
        val serverMs = parseIso(o.serverTime) ?: return // بلا مرجعِ ساعةٍ لا تسليح
        anchorServerMs = serverMs
        anchorElapsed = SystemClock.elapsedRealtime()
        platformBoundaryMs = boundaryOf(o.available, o.nextCloseAt, o.nextAvailableAt)
        rearm()
    }

    /** **بعد كلِّ قراءةِ إتاحةِ منطقةٍ** — التسعيرة/الإتاحة تحمل حدَّ المنطقة. */
    fun syncZone(av: Availability?) {
        zoneBoundaryMs = if (av == null) 0L else boundaryOf(av.available, av.nextCloseAt, av.nextAvailableAt)
        rearm()
    }

    /** **العودةُ إلى الواجهة** — يُجبَر تجديدٌ (من المنادي) ثمّ يُعاد التسليح. */
    fun onForeground() {
        foreground = true
        rearm()
    }

    /** **الذهابُ إلى الخلفيّة** — يُلغى المؤقّت، فلا نومٌ لشاشةٍ لا تُرى. */
    fun onBackground() {
        foreground = false
        job?.cancel()
        job = null
    }

    // **حدُّ الطبقة**: إن كانت مفتوحةً فحدُّها القادمُ إغلاقُها، وإلّا فتحُها.
    // **`internal` لتُقاسَ خالصةً** بلا ساعةِ أندرويد.
    internal fun boundaryOf(available: Boolean, nextClose: String, nextOpen: String): Long =
        parseIso(if (available) nextClose else nextOpen) ?: 0L

    internal fun parseIso(s: String): Long? =
        if (s.isBlank()) null
        else runCatching { OffsetDateTime.parse(s).toInstant().toEpochMilli() }.getOrNull()

    /**
     * **أقربُ حدٍّ في المستقبل** من بين حدَّي المنصّةِ والمنطقة — و`null`
     * إن لم يكن حدٌّ بعدَ المرجع. **خالصةٌ فتُقاس بلا ساعة.** (٠ = لا حدَّ.)
     */
    internal fun earliestFutureBoundary(platformMs: Long, zoneMs: Long, afterServerMs: Long): Long? =
        listOf(platformMs, zoneMs).filter { it > afterServerMs }.minOrNull()

    private fun rearm() {
        job?.cancel()
        job = null
        if (!foreground || anchorServerMs == 0L) return
        // **أقربُ حدٍّ في المستقبلِ فقط** — ولا يُسلَّح على حدٍّ مضى.
        val next = earliestFutureBoundary(platformBoundaryMs, zoneBoundaryMs, anchorServerMs) ?: return
        // **مدّةُ الخادمِ من المرجعِ إلى الحدّ**، مُسقَطةً على الساعةِ الرتيبة،
        // زائدَ هامشٍ صغيرٍ ليكون الخادمُ قد انقلب حين نُعيد السؤال.
        val targetElapsed = anchorElapsed + (next - anchorServerMs) + EPSILON_MS
        val waitMs = targetElapsed - SystemClock.elapsedRealtime()
        val s = scope ?: return
        job = s.launch {
            if (waitMs > 0) delay(waitMs)
            // **بلغَ الحدُّ — يُجبَر إعادةُ التحقّق للطبقتين.** والقراءاتُ
            // التاليةُ تُعيد `syncPlatform`/`syncZone` فيُعاد التسليحُ للتالي
            // (إغلاق ← فتح ← إغلاق…).
            Serving.refresh(this, force = true)
            Refresh.bump()
        }
    }

    /** **هامشٌ بعد الحدّ** — ليضمن أنّ الخادمَ انقلب قبل أن نعيد السؤال. */
    const val EPSILON_MS = 1500L

    /** **للفحوص وحدَها.** */
    fun resetForTest() {
        job?.cancel()
        job = null
        scope = null
        foreground = false
        anchorServerMs = 0L
        anchorElapsed = 0L
        platformBoundaryMs = 0L
        zoneBoundaryMs = 0L
    }
}
