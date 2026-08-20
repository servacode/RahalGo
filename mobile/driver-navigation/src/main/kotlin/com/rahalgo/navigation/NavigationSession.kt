package com.rahalgo.navigation

import android.content.Context
import android.util.Log
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
    private val pipeline: NavPipeline = NavPipeline(probe = ::log),
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
    private var firstFixAt = 0L
    private var lastFixAt = 0L
    private var minGapMs = Long.MAX_VALUE
    private var maxGapMs = 0L

    /** **مدّةُ الجلسة بالميلي** — من أوّل قراءةٍ إلى آخرها. */
    val spanMs: Long get() = if (firstFixAt == 0L) 0L else lastFixAt - firstFixAt

    /** **متوسّطُ الفاصل الفعليّ** — لا المطلوب. (البند ٥ من التقرير.) */
    val meanGapMs: Long get() = if (samples < 2) 0L else spanMs / (samples - 1)

    val minGap: Long get() = if (minGapMs == Long.MAX_VALUE) 0L else minGapMs
    val maxGap: Long get() = maxGapMs

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
        firstFixAt = 0L
        lastFixAt = 0L
        minGapMs = Long.MAX_VALUE
        maxGapMs = 0L
        engine.onFix = { fix ->
            // **والفاصلُ الفعليُّ يُقاس هنا** — قبل أيّ ترشيح:
            // **المرفوضةُ وصلت أيضاً**، وتردّدُ الجهاز يُقاس بما أعطى
            // لا بما قُبل.
            if (firstFixAt == 0L) {
                firstFixAt = fix.atMs
            } else {
                val gap = fix.atMs - lastFixAt
                if (gap in 1..600_000) {
                    if (gap < minGapMs) minGapMs = gap
                    if (gap > maxGapMs) maxGapMs = gap
                }
            }
            lastFixAt = fix.atMs
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
     * **سطرُ الخلاصة** — يُطبع عند الإغلاق ويُقرأ في التقرير.
     *
     * **ولا يُطبع في كلّ قراءة** — أمرُ المالك: «بدون إغراق Logs
     * الإنتاجية».
     */
    fun summary(): String =
        "قراءات=$samples مقبولة=${seen - rejected - degraded} متدهورة=$degraded " +
            "مرفوضة=$rejected (دقّة=${pipeline.rejectedAccuracy} " +
            "قفزة=${pipeline.rejectedTeleport} قديمة=${pipeline.rejectedStale}) " +
            "الفاصل: متوسّط=${meanGapMs}ملّي أدنى=${minGap} أقصى=${maxGap} " +
            "الدقّة: أفضل=${pipeline.bestAccuracyM} أسوأ=${pipeline.worstAccuracyM} " +
            "متوسّط=${"%.1f".format(pipeline.meanAccuracyM)} " +
            "المدّة=${spanMs}ملّي"

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
        Log.i(TAG, "خلاصةُ الملاحة — ${summary()}")
        engine.stop()
        engine.onFix = null
        running = false
        render = null
    }

    private companion object {
        const val TAG = "RahalGo/nav"

        /**
         * **يُسجّل المرفوضَ والمتدهورَ وحدَهما** — والمقبولةُ هي
         * الأغلبيّة، **وسطرٌ لكلّ قراءةٍ في الثانية يملأ السجلَّ فلا
         * يُقرأ منه شيء.**
         */
        fun log(fix: NavFix, grade: FixGrade, reason: RejectReason) {
            if (grade == FixGrade.ACCEPTED) return
            Log.d(
                TAG,
                "قراءة $grade ${if (reason != RejectReason.NONE) reason else ""} " +
                    "دقّة=${fix.accuracyM} سرعة=${fix.speedMps} اتّجاه=${fix.bearingDeg}",
            )
        }
    }
}
