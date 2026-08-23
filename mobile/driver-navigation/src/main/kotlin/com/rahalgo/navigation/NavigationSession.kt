package com.rahalgo.navigation

import android.content.Context
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import kotlinx.coroutines.launch

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
    /**
     * **مسجّلُ الرحلة — أداةُ فحصٍ لا جزءٌ من الملاحة.**
     *
     * (أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
     *
     * **وفارغٌ في الإنتاج**: التطبيقُ يمرّره في بناء التطوير وحدَه.
     * **وتسجيلُ مسارِ كلّ سائقٍ في كلّ رحلةٍ ليس فحصاً بل تتبّعا.**
     *
     * **ولا يُقرأ منه شيءٌ في المنطق** — يُكتب فيه ولا يُسأل.
     */
    private val recorder: TraceRecorder? = null,
    /**
     * **بابُ إعادة الحساب — وفارغٌ يعني «لا إعادة».**
     *
     * **ويُمرَّر من تطبيق السائق وحدَه** — هذه الوحدةُ لا تعرف
     * `Backend` ولا `DriverApi`، **وهو ما يحرسه `NavSessionTest`.**
     */
    source: RouteSource? = null,
    private val pipeline: NavPipeline = NavPipeline(probe = { f, g, r ->
        log(f, g, r)
        // **ويُسجَّل الرديءُ كما يُسجَّل الجيّد** — **ورحلةٌ فيها
        // المقبولُ وحدَه لا تختبر مرشِّحاً بل تختبر ما نجا منه.**
        recorder?.add(f, g, r)
    }),
    /**
     * ══════════════════════════════════════════════════════════════════
     * **والمحرّكُ هو من يحسب — لا هذه الجلسةُ ولا الشاشة**
     * ══════════════════════════════════════════════════════════════════
     *
     * (المرحلة ٣أ، أمرُ المالك ٢٠٢٦-٠٨-٢٠: «ولا تجعل UI نفسَها تحسب
     *  `RouteProgress` أو `OffRoute`».)
     *
     * **وهذه الجلسةُ غلافُ أندرويد**: تفتح محرّكَ المواقع وتغلقه،
     * وتحمل حالةَ Compose. **والحسابُ كلُّه في `NavEngine` الخالص** —
     * فيُعاد تشغيلُ رحلةٍ عليه بلا هاتف.
     */
    /**
     * **مخطِّطُ الصوت** — وفارغٌ يعني «لا إرشادَ صوتيّ». (المرحلة ٤.)
     *
     * **ولا يقول شيئاً** — يُخرج قراراً في `NavState.cues`.
     */
    voice: VoicePlanner? = null,
    private val engineCore: NavEngine = NavEngine(pipeline, source = source, voice = voice),
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

    /**
     * ══════════════════════════════════════════════════════════════════
     * **حالُ الملاحة كاملةً — تقرؤها الشاشةُ ولا تحسبها**
     * ══════════════════════════════════════════════════════════════════
     *
     * (المرحلة ٣أ.)
     *
     * **فيها الموضعُ والتقدّمُ وما بقي والمناورتان وحالُ الخروج** —
     * انظر `NavState`.
     *
     * **وفارغةٌ تعني «لم تصل قراءةٌ بعد»** — لا «لا مسار».
     */
    var nav by mutableStateOf<NavState?>(null)
        private set

    /**
     * **يُسلَّم المسارَ وحدةً واحدة.**
     *
     * **ويُنادى كلَّما تبدّل مسارُ الطلب** — وفارغٌ يعني «لا ملاحة»:
     * **الخريطةُ ترسم كما كانت ولا يسقط شيء.**
     *
     * **ويُنسى معه كلُّ تقدّمٍ وكلُّ شكّ** — انظر `NavEngine.setRoute`.
     */
    fun setRoute(route: NavRoute?) {
        engineCore.setRoute(route)
        nav = null
    }

    /**
     * ══════════════════════════════════════════════════════════════
     * **الهدفُ التجاريّ — إغلاقُ نقطة الالتقاط، ٢٠٢٦-٠٨-٢١**
     * ══════════════════════════════════════════════════════════════
     *
     * **المتجرُ أو بابُ الزبون كما في القاعدة** — لا نهايةُ
     * الخطّ التي اختارها المحرّك.
     *
     * **ويُنادى مع كلّ تبدّلِ طور** — فوجهةُ المتجر غيرُ
     * وجهةِ الزبون.
     *
     * **ولا يُمسح مع المسار**: إعادةُ حسابٍ تُبدّل الخطّ
     * **ولا تُبدّل المتجر** — ولو مُسح لعاد الوصولُ الكاذب
     * بعد كلّ إعادةِ حساب (البند ١٥).
     */
    fun setArrivalTarget(target: GeoPoint?) {
        engineCore.arrivalTarget = target
    }

    /** **الهدفُ النافذ** — يُقرأ في التشخيص والاختبار. */
    val arrivalTarget: GeoPoint? get() = engineCore.arrivalTarget

    /**
     * **هويّةُ المسار النافذ** — المرحلة ٨ب.
     *
     * **يُسلّمها التطبيقُ مع كلّ مسارٍ يُركّب** — أوّليّاً كان أو
     * مُعادَ حسابٍ أو بديلاً مختاراً.
     */
    var routeId: String?
        get() = engineCore.routeId
        set(value) {
            engineCore.routeId = value
        }

    /**
     * **طلبُ ارتباطٍ ينتظر الإرسال** — أو `null`.
     *
     * **ولا يُرسله المحرّك**: `driver-navigation` لا تعرف شبكة.
     */
    val correlationRequest: CorrelationRequest? get() = engineCore.correlationRequest

    /** **يُسلَّم حكمَ الخادم.** */
    fun onCorrelation(result: RoadCorrelation) {
        engineCore.parallel.onResult(result, nav?.progress, engineCore.lastAcceptedFix)
    }

    /** **أثبتَ الخادمُ أنّه على طريقٍ موازٍ؟** */
    val confirmedParallel: Boolean get() = engineCore.parallel.confirmedParallel

    /** **المسارُ النافذُ الآن** — يُقرأ في التشخيص. */
    val route: NavRoute? get() = engineCore.currentRoute

    /**
     * **جيلُ المسار** — إغلاقُ واجهة ٧، ٢٠٢٦-٠٨-٢١.
     *
     * **قارئٌ فقط** — يكشف ما يحسبه `NavEngine` ولا يغيّره.
     *
     * **وتحتاجه لوحةُ الاختيار**: كلُّ جلبةِ بدائلَ تحمل جيلَها،
     * **ويُقارَن عند الاعتماد** — فاختيارٌ من مجموعةٍ سبقت إعادةَ
     * حسابٍ يُرفض (البند ٢٠ من المرحلة ٧).
     *
     * **ولا منطقَ جديدٌ هنا** — سطرُ تفويضٍ لا أكثر.
     */
    val generation: Long get() = engineCore.generation

    /** **حالُ إعادة الحساب** — تقرؤها الشاشةُ ولا تحسبها. */
    val rerouteStatus: RerouteStatus get() = nav?.reroute ?: RerouteStatus.NONE

    /** **مخطِّطُ الصوت** — يُحقَن فيه طرفُ الرحلة. */
    val voicePlanner: VoicePlanner? get() = engineCore.voice

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

    /** **رفيدةُ هذه الجلسة** — أو فارغٌ إن لم يكن مسجّل. */
    val traceFile: java.io.File? get() = recorder?.file

    /** **أتعطّل المسجّل؟** — يُقرأ في التقرير لا في المنطق. */
    val traceFailed: Boolean get() = recorder?.failed == true

    /**
     * **يفتح الجلسة.**
     *
     * **ويردّ `false` إن لم يُمنح الإذن** — فتبقى الشاشةُ كما كانت
     * ولا تسقط.
     */
    fun start(): Boolean {
        if (running) return true
        engineCore.reset()
        nav = null
        // **ويُفتح الملفُّ قبل أوّل قراءة** — **وفشلُه لا يمنع
        // الملاحة**: انظر `TraceRecorder`.
        recorder?.start()
        stepId = 0L
        firstFixAt = 0L
        lastFixAt = 0L
        minGapMs = Long.MAX_VALUE
        maxGapMs = 0L
        engine.onFix = { fix -> consume(fix) }
        val ok = engine.start()
        running = ok
        if (!ok) engine.onFix = null
        return ok
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **مجرى القراءة الواحد — ولا يعرف من أين جاءت**
     * ══════════════════════════════════════════════════════════════════
     *
     * **استُخرج من `start` ولم يتبدّل سطرٌ منه** (٢٠٢٦-٠٨-٢٣): كان
     * جسمَ `engine.onFix` نفسَه.
     *
     * **ولذلك قيمةٌ واحدة**: **ما يمشي في الإعادة هو ما يمشي في
     * الشارع حرفاً بحرف** — لا نسخةٌ ثانيةٌ تفترق يومَ يُصلَح أحدُها،
     * **ومختبَرٌ يجرّب طريقاً غيرَ طريق الإنتاج يعطي طمأنينةً كاذبة.**
     */
    private fun consume(fix: NavFix) {
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
        val out = engineCore.onFix(fix)
        nav = out
        // **والمرفوضةُ لا تُبدّل ما يُرسم** — تبقى الأيقونةُ حيث
        // هي. **وهذا هو «لا تسمح لقراءةٍ سيّئةٍ أن تقفز شارعا».**
        NavRender.of(out, ++stepId)?.let { render = it }
    }

    // ══════════════════════════════════════════════════════════════════
    // **الإعادة — رحلةٌ تمشي والسائقُ جالس**
    // ══════════════════════════════════════════════════════════════════
    //
    // (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «تقدر تخلّي شاشةَ الرحلة مفتوحةً قدّامي
    //  وتشوفني شلون الطلبُ يمشي عالخريطة على المتجر وعلى الزبون بدون
    //  أن أتحرّك أنا؟»)
    //
    // # ولا تمسّ طبقةَ الأقمار
    //
    // **قرارُ المالك السابق قائمٌ** (٢٠٢٦-٠٨-٢١): لا يُزوَّر
    // `FusedLocationProviderClient` — **لأنّ الإنتاج يعتمده، ومحاكاةٌ
    // لا تمرّ منه لا تمثّل ما سيعمل به السائق.**
    //
    // **وهذه لا تزوّره ولا تلمسه**: `LocationEngine` لا يُستدعى
    // أصلاً في الإعادة. **تدخل من فوقه إلى المجرى نفسِه.**
    //
    // # وما تُثبته وما لا تُثبته
    //
    // **تُثبت**: الإسقاطَ على المسار · التقدّمَ والزمنَ المتبقّي ·
    // كشفَ الخروج وإعادةَ التوجيه · الكاميرا والسهمَ والدوران ·
    // الصوتَ · الوصول.
    //
    // **ولا تُثبت**: قراءةَ الأقمار نفسَها — دقّتَها وتذبذبَها
    // وانقطاعَها بين البنايات. **وتلك لا يثبتها إلّا الشارع.**
    //
    // # ولا تُبنى إلّا في بناء التطوير
    //
    // **والحارسُ في التطبيق لا هنا**: الوحدةُ تقبل أن تُعاد رحلة،
    // **والتطبيقُ وحدَه يعرف أيَّ بناءٍ هو** — كما في `TraceRecorder`.

    /** **أفي إعادةٍ الآن؟** — ليُقال في الشاشة، فلا يُظنّ حقيقيّاً. */
    var replaying by mutableStateOf(false)
        private set

    private var replayJob: kotlinx.coroutines.Job? = null

    /**
     * **يمشي على قراءاتٍ مصنوعة.**
     *
     * @param fixes ما يُغذّى به المجرى — بترتيبه الزمنيّ.
     * @param scope مجالُ الشاشة — **فتنتهي الإعادةُ بانتهائها**، ولا
     *              يبقى مؤقّتٌ يعمل في شاشةٍ هُدمت.
     * @param stepMs الفاصلُ بين قراءتين على الشاشة.
     */
    fun startReplay(
        fixes: List<NavFix>,
        scope: kotlinx.coroutines.CoroutineScope,
        stepMs: Long = 1000L,
    ) {
        if (fixes.isEmpty()) return
        stopReplay()
        // ══════════════════════════════════════════════════════════════
        // **ويُسكَت مجرى الأقمار ما دامت الإعادةُ تمشي**
        // ══════════════════════════════════════════════════════════════
        //
        // **وإلّا دخل المجرى الواحدَ مصدران**: قراءةٌ مصنوعةٌ تقول
        // «أنا على بعد كيلومتر» وقراءةٌ حقيقيّةٌ تقول «أنا في غرفتي»
        // — **فتقفز الأيقونةُ بينهما كلَّ ثانية**، ويُقرأ العطبُ في
        // الملاحة وهو في المختبَر.
        //
        // **ولا يُعاد تشغيلُه عند الإيقاف** — يبقى القرارُ بيد
        // «ابدأ الملاحة»: **ومن أعاد فتحَ شيءٍ لم يفتحه هو يفاجئ
        // صاحبَه.**
        engine.onFix = null
        engine.stop()
        engineCore.reset()
        nav = null
        stepId = 0L
        firstFixAt = 0L
        lastFixAt = 0L
        minGapMs = Long.MAX_VALUE
        maxGapMs = 0L
        replaying = true
        running = true
        replayJob = scope.launch {
            for (f in fixes) {
                consume(f)
                kotlinx.coroutines.delay(stepMs)
            }
            replaying = false
        }
    }

    fun stopReplay() {
        replayJob?.cancel()
        replayJob = null
        replaying = false
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
        nav = null
        // **ويُغلق الملفُّ إغلاقاً سليماً** — **وملفٌّ لم يُغلق يفقد
        // آخرَ ما في مخزنه**، وهو غالباً أهمُّ ما في الرحلة: نهايتُها.
        recorder?.stop()?.let { Log.i(TAG, "رفيدةُ الرحلة: ${'$'}{it.absolutePath}") }
        recorder?.takeIf { it.failed }?.let {
            Log.w(TAG, "تعطّل مسجّلُ الرحلة — ${'$'}{it.failure}")
        }
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
