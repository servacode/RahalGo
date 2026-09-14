package com.rahalgo.customer

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.ui.AppCore
import com.rahalgo.shared.model.Ordering
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أنستقبل طلباً الآن؟ — ما يقوله الخادمُ لا ما تحسبه الشاشة** (`PH`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولمَ لا تُحسَب في الجهاز
 *
 * **والجدولُ بمنطقةِ دمشق** — **وساعةُ الهاتف تُبدَّل بإصبعٍ في
 * الإعدادات.** **ولو حسبت الشاشةُ «أنحن داخلَ الدوام» لَفتح من قدّم
 * ساعتَه بابَ الطلب على نفسه**، **ثمّ رُدّ من المحرّك فلم يفهم لماذا.**
 *
 * **فالخادمُ يقول، والشاشةُ تعرض.** **والمنعُ الحقيقيُّ عند الإنشاء
 * على كلّ حال** — **وهذه تكفيه مؤونةَ أن يملأ سلّةً ليُردّ في آخرها.**
 *
 * # ولمَ حالٌ للعمليّة لا حقلٌ في شاشة
 *
 * **والحالُ واحدةٌ للمنصّة كلِّها** — **لا لشاشةٍ بعينها.** **وسلّةٌ
 * تقرؤها وشاشةُ متجرٍ تقرؤها وشاشةُ الطلب المخصَّص** — **وثلاثُ نسخٍ
 * تفترق فتقول إحداها «مفتوح» والأخرى «مغلق» في اللحظة نفسِها.**
 *
 * # وتُجدَّد عند العودة لا عند الإقلاع وحدَه
 *
 * **ومن فتح التطبيقَ في الخامسة والدوامُ ينتهي السادسةَ ثمّ تركه في
 * الخلفيّة وعاد في الثامنة** — **يرى حالاً عمرُها ثلاثُ ساعات.**
 * **فيُطلب التجديدُ عند كلّ عودةٍ ذاتِ بال**، **وعند فتح السلّة.**
 *
 * **ولا يُجدَّد في كلّ عودةٍ مهما قصرت**: **من بدّل تطبيقاً لثانيةٍ
 * وعاد لا يحتاج نداءً** — **ونداءٌ في كلّ عودةٍ يُثقل شبكةً ضعيفة.**
 */
object Serving {

    /** **آخرُ ما قاله الخادم** — و`null` قبل أوّل قراءةٍ ناجحة. */
    var state: Ordering? by mutableStateOf(null)
        private set

    /** **لحظةُ آخرِ قراءةٍ ناجحة** — بساعةِ التشغيل لا بساعة الحائط. */
    var readAt: Long = 0L
        private set

    /**
     * **أيُستقبَل الطلبُ بحسب آخرِ ما نعلم؟**
     *
     * **وقبل أوّل قراءةٍ يُقال «نعم»** — **ولا يُمنَع أحدٌ لأنّ نداءً لم
     * يصل بعد**: **والمحرّكُ يردّه إن كان مغلقاً، والمنعُ بلا علمٍ أسوأُ
     * من ردٍّ بعلم.**
     */
    val available: Boolean get() = state?.available ?: true

    /** **سببُ المنع** — وفارغٌ يعني لا منع. */
    val reason: String get() = if (available) "" else state?.reason.orEmpty()

    /** **نصُّ المالك إن ضبطه.** */
    val message: String get() = if (available) "" else state?.message.orEmpty()

    /** **موعدُ العودة** — RFC 3339، وفارغٌ يعني «لا موعدَ معلوم». */
    val nextAt: String get() = if (available) "" else state?.nextAvailableAt.orEmpty()

    /**
     * stale **أشاخت الحالُ فتحتاج تجديداً؟**
     *
     * **والقياسُ بساعة التشغيل** (`elapsedRealtime`) — **لا بساعة
     * الحائط**: **من بدّل ساعةَ جهازه قفز العمرُ قفزةً**، **فتُجدَّد
     * الحالُ بلا سببٍ أو لا تُجدَّد وقد وجب.**
     */
    fun stale(nowElapsed: Long, maxAgeMs: Long = FRESH_MS): Boolean =
        state == null || nowElapsed - readAt >= maxAgeMs

    /** **تُكتب عند كلّ قراءةٍ ناجحة.** */
    fun put(o: Ordering, nowElapsed: Long) {
        state = o
        readAt = nowElapsed
    }

    /**
     * refreshIfStale **يسأل الخادمَ إن شاخت الحال — ويصمت إن لم تشخ.**
     *
     * **وسقوطُ النداء لا يُبدّل شيئاً** — **والحالُ القديمةُ أصدقُ من
     * لا حال**، **والمحرّكُ يردّ الطلبَ إن كان مغلقاً على كلّ حال.**
     */
    fun refreshIfStale(scope: kotlinx.coroutines.CoroutineScope) {
        if (!stale(android.os.SystemClock.elapsedRealtime())) return
        scope.launch {
            runCatching { com.rahalgo.shared.auth.AuthApi(AppCore.get().api).platform() }
                .onSuccess { put(it.ordering, android.os.SystemClock.elapsedRealtime()) }
        }
    }

    /** **تُصفَّر في الفحوص وحدَها** — **ولا يناديها منتَج.** */
    fun resetForTest() {
        state = null
        readAt = 0L
    }

    /**
     * FRESH_MS **عمرُ الحال المقبول** — دقيقتان.
     *
     * **وحدُّ الدوام دقيقةٌ لا ثانية** — **فحالٌ عمرُها دقيقتان تخطئ
     * في أسوأ الأحوال بدقيقتين قبل الإغلاق**، **والمحرّكُ يردّ عندها.**
     * **وأقصرُ من ذلك نداءٌ في كلّ لمسة.**
     */
    const val FRESH_MS: Long = 2 * 60 * 1000
}
