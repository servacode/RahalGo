package com.rahalgo.navigation

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/**
 * ══════════════════════════════════════════════════════════════════════
 * **جلسةُ ملاحة — تُفتح وتُغلق، وما بينهما تتحرّك الشاشة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ١، بأمر المالك ٢٠٢٦-٠٨-٢٠.)
 *
 *     start()  →  محرّكُ موقعٍ سريعٍ محلّيّاً
 *                 → سلسلةٌ ترشّح وتنعّم
 *                 → `render` تتبدّل فترسم الشاشة
 *     stop()   →  يعود الجهازُ إلى وضعه الطبيعيّ
 *
 * # وما لا تفعله
 *
 * **لا ترسل نقطةً واحدةً إلى الخادم.** (أمرُ المالك: «رفع تردّد GPS
 * محليّاً لا يعني رفع تردّد الإرسال… لا أريد طلب شبكة كلّ ثانية».)
 *
 * **وخدمةُ الورديّة تعمل كما هي بجانبها** — هي التي تُبلّغ الخادم،
 * بتردّدها القديم وفلترها القديم. **ولم تُمسّ.**
 *
 * # ولا تعرف طلباً
 *
 * **تعرف أنّها تعمل أو لا تعمل** — ومن يقرّر متى يفتحها هو تطبيقُ
 * السائق حين تبدأ رحلة. **وهذا يُبقي الوحدةَ صالحةً لأيّ ملاحةٍ
 * لاحقة.**
 */
class NavigationSession(
    private val context: Context,
    private val engine: LocationEngine = LocationEngine(context),
    private val pipeline: NavPipeline = NavPipeline(),
) {

    /**
     * **ما تُرسمه الخريطةُ الآن** — وفارغٌ يعني «لا ملاحة».
     *
     * **وحالةُ Compose لا متغيّرٌ عاديّ** — الشاشةُ تُعاد رسماً مع كلّ
     * قراءةٍ بلا أن تسأل.
     */
    var render by mutableStateOf<NavRender?>(null)
        private set

    /** **أتعمل الآن؟** */
    var running by mutableStateOf(false)
        private set

    private var stepId = 0L

    /** **عدّاداتُ القياس** — تُقرأ في تقرير المرحلة. */
    val samples: Int get() = engine.samples
    val seen: Int get() = pipeline.seen
    val rejected: Int get() = pipeline.rejected
    val degraded: Int get() = pipeline.degraded
    val headingDeg: Float? get() = pipeline.headingDeg

    /**
     * **يفتح الجلسة.**
     *
     * **ويردّ `false` إن لم يُمنح الإذن** — فتبقى الشاشةُ كما كانت
     * ولا تسقط.
     */
    fun start(): Boolean {
        if (running) return true
        pipeline.reset()
        stepId = 0L
        engine.onFix = { fix ->
            val step = pipeline.onFix(fix)
            // **والمرفوضةُ لا تُبدّل ما يُرسم** — تبقى الأيقونةُ حيث
            // هي. **وهذا هو «لا تسمح لقراءةٍ سيّئةٍ أن تقفز شارعا».**
            NavRender.of(step, ++stepId)?.let { render = it }
        }
        val ok = engine.start()
        running = ok
        if (!ok) engine.onFix = null
        return ok
    }

    /**
     * **يغلق الجلسة ويعيد الجهازَ إلى وضعه.**
     *
     * (معيارُ المالك: «إنهاء جلسة الملاحة يعيد Location Mode للوضع
     *  الطبيعيّ».)
     *
     * **و`render` تُمحى** — فتعود الخريطةُ إلى سلوكها المألوف: **ومن
     * أنهى رحلتَه فوجد خريطتَه مائلةً مستديرةً ظنّ أنّ شيئاً عطب.**
     */
    fun stop() {
        if (!running) return
        engine.stop()
        engine.onFix = null
        running = false
        render = null
    }
}
