package com.rahalgo.navigation

import kotlin.math.abs
import kotlin.math.max
import kotlin.math.min

/**
 * ══════════════════════════════════════════════════════════════════════
 * **متى يُقال وماذا — ولا يعرف كيف يُقال**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٤، أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
 *
 *     NavState  →  VoicePlanner  →  VoiceCue
 *
 * **ولا `TextToSpeech` ولا `AudioManager` ولا سياقُ أندرويد** — أمرُ
 * المالك نصّاً: **«لا أريد Android side effects داخل محرّك الملاحة»**.
 * فتُعاد رحلةٌ كاملةٌ عليه وتُقرأ التعليماتُ قائمةً، **بلا سمّاعة.**
 *
 * # والزنادُ بالزمن لا بالمسافة
 *
 * (أمرُ المالك، البند ٥: «من يسير ١٠ كم/س ليس كمن يسير ٧٠ كم/س».)
 *
 * **ومئتا مترٍ عند خمسةَ عشرَ كم/س ثمانٍ وأربعون ثانية** — تنبيهٌ سابقٌ
 * لأوانه بكثير. **وعند سبعين كم/س عشرُ ثوانٍ** — متأخّرٌ جدّاً. **والرقمُ
 * واحدٌ والنتيجتان متناقضتان.**
 *
 * **فالعتبةُ ثوانٍ، وتُقيَّد بحدَّي مسافةٍ يمنعان السخف** في الطرفين.
 *
 * # وما لا يفعله
 *
 * **لا يقول ولا يُطابر ولا يمسك تركيزَ صوت** — يُخرج قراراً. **والطابورُ
 * والأولويّةُ في تطبيق السائق** (`VoiceOrchestrator`).
 */
class VoicePlanner(val tuning: VoiceTuning = VoiceTuning()) {

    /**
     * **طرفُ الرحلة** — يُحقَن ولا يُستنتَج.
     *
     * (أمرُ المالك، البند ٢٥: «لا تجعل `VoicePlanner` تعرف Order أو
     *  Status الخام».)
     */
    var target: TripTarget = TripTarget.PICKUP

    /** **آخرُ مسافةٍ إلى المناورة** — لكشف عبور العتبة. */
    private var lastDistanceM: Double = Double.NaN
    private var lastManeuverAtM: Double = Double.NaN
    private var lastGeneration: Long = -1L

    /** **لحظةُ آخر قراءةٍ مقبولةٍ تماماً** — بالمقياس الرتيب. */
    private var trustedAtMs: Long = 0L

    /** **نوبةُ إعادة الحساب الجارية** — فلا تتكرّر جملتُها. */
    private var rerouteEpisodeSaid = false
    private var rerouteFailSaid = false
    private var lastRerouteStatus: RerouteStatus = RerouteStatus.NONE

    /** **عدّاداتُ القياس** — تُقرأ في التقرير لا في المنطق. */
    var emitted = 0
        private set
    var gatedByGps = 0
        private set

    /**
     * ══════════════════════════════════════════════════════════════════
     * **يُغذّى حالاً فيردّ ما يجب أن يُقال — أو لا شيء**
     * ══════════════════════════════════════════════════════════════════
     *
     * **وقائمةٌ لا واحدة**: قد يقع الوصولُ ونهايةُ إعادةِ حسابٍ في
     * القراءة نفسِها.
     */
    fun onState(state: NavState, generation: Long, fix: NavFix): List<VoiceCue> {
        // **وجيلٌ جديدٌ يمحو كلَّ ما كان** — أمرُ المالك، البند ٢١.
        if (generation != lastGeneration) {
            lastGeneration = generation
            lastDistanceM = Double.NaN
            lastManeuverAtM = Double.NaN
            rerouteEpisodeSaid = false
            rerouteFailSaid = false
            arrivalSaid = false
            routeEndEpisode = -1L
            wrongWaySaidEpisode = -1L
        }

        // **والمقبولةُ وحدَها تُجدّد الثقة** — كما في المرحلة ٣أ.
        if (state.grade == FixGrade.ACCEPTED) trustedAtMs = fix.atMs

        val out = ArrayList<VoiceCue>(3)
        wrongWayCue(state, generation)?.let { out += it }
        rerouteCue(state, generation)?.let { out += it }
        arrivalCue(state, generation)?.let { out += it }
        routeEndCue(state, generation)?.let { out += it }
        maneuverCue(state, generation, fix)?.let { out += it }
        emitted += out.size
        return out
    }

    /** **يُنسى كلُّ شيء** — عند جلسةٍ جديدة. */
    fun reset() {
        lastDistanceM = Double.NaN
        lastManeuverAtM = Double.NaN
        lastGeneration = -1L
        trustedAtMs = 0L
        rerouteEpisodeSaid = false
        rerouteFailSaid = false
        arrivalSaid = false
        wrongWaySaidEpisode = -1L
        lastRerouteStatus = RerouteStatus.NONE
        emitted = 0
        gatedByGps = 0
    }

    // ══════════════════════════════════════════════════════════════════
    // **الاتّجاهُ المعاكس**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **مرّةً واحدةً لكلّ نوبة — وعند التأكيد وحدَه.**
     *
     * (قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ٢٢ و٢٣: «لا صوتَ عند
     *  `SUSPECTED_WRONG_WAY`… وإذا صحّح ثمّ دخل نوبةً جديدةً يمكن
     *  النطقُ مرّةً أخرى».)
     *
     * **ورقمُ النوبة في الهُويّة** — فالشكُّ لا ينطق، **والنوبةُ
     * الثانيةُ تنطق.**
     */
    private fun wrongWayCue(state: NavState, generation: Long): VoiceCue? {
        if (!state.isWrongWay) return null
        val episode = state.wrongWay.episode
        if (episode == wrongWaySaidEpisode) return null
        wrongWaySaidEpisode = episode
        return VoiceCue(
            id = CueId(generation, WRONG_WAY_KEY - episode, CueStage.EVENT),
            kind = CueKind.WRONG_WAY,
            stage = CueStage.EVENT,
            priority = tuning.priorityNow,
            validUntilProgressM = Double.MAX_VALUE,
            text = VoicePhrases.WRONG_WAY,
        )
    }

    private var wrongWaySaidEpisode = -1L

    // ══════════════════════════════════════════════════════════════════
    // **إعادةُ الحساب**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **جملةٌ واحدةٌ لكلّ نوبة.**
     *
     * (أمرُ المالك، البند ٢٤.)
     *
     * **و`OFF_ROUTE` وحدَه صامت**: «أنت خارج المسار» **قلقٌ بلا فعل** —
     * والسائقُ لا يملك أن يصنع شيئاً بها. **و«جارٍ إعادة الحساب» تقول
     * الشيءَ نفسَه ومعها ما سيقع.**
     *
     * **والنجاحُ صامتٌ أيضاً** — وتُقال أوّلُ تعليمةٍ من المسار الجديد
     * مباشرةً، **وهي أنفعُ من «تمّ بنجاح».**
     */
    private fun rerouteCue(state: NavState, generation: Long): VoiceCue? {
        val was = lastRerouteStatus
        lastRerouteStatus = state.reroute
        return when {
            state.reroute == RerouteStatus.REROUTING && !rerouteEpisodeSaid -> {
                rerouteEpisodeSaid = true
                event(generation, CueKind.REROUTE, VoicePhrases.REROUTING, tuning.priorityReroute)
            }
            state.reroute == RerouteStatus.REROUTE_FAILED && !rerouteFailSaid -> {
                rerouteFailSaid = true
                event(generation, CueKind.REROUTE, VoicePhrases.REROUTE_FAILED, tuning.priorityReroute)
            }
            // **وانتهاءُ النوبة يفتح البابَ لنوبةٍ تالية.**
            state.reroute == RerouteStatus.NONE && was != RerouteStatus.NONE -> {
                rerouteEpisodeSaid = false
                rerouteFailSaid = false
                null
            }
            else -> null
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **الوصول**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **من `arrived` الموثوق لا من `nearDestination`.**
     *
     * (أمرُ المالك، البند ٢٥. و`nearDestination` شأنُ عرضٍ قُرّر في
     * المرحلة الثانية.)
     */
    /**
     * ══════════════════════════════════════════════════════════════
     * **انتهى الخطُّ ولمّا يصل — البنود ٩ إلى ١٢**
     * ══════════════════════════════════════════════════════════════
     *
     * **ولا يُنطق إن وصل في القراءة نفسِها** (البند ١١): الدعوى
     * أَولى، **وندائان في لحظةٍ واحدةٍ ضجيج.**
     *
     * **وهويّتُه بالجيل** (البند ١٢) — فإعادةُ تركيبٍ أو قراءةٌ
     * جديدةٌ أو فتحُ شاشةٍ **لا تُعيده**. ومسارٌ جديدٌ نوبةٌ جديدة.
     */
    private fun routeEndCue(state: NavState, generation: Long): VoiceCue? {
        if (state.arrivalPhase != ArrivalPhase.LAST_MILE_TO_TARGET) return null
        // **ووصل في هذه القراءة؟ فالدعوى وحدَها** — لا هذا.
        if (arrivalSaid && state.arrivedAtTarget) return null
        if (routeEndEpisode == generation) return null
        routeEndEpisode = generation
        return VoiceCue(
            id = CueId(generation, ROUTE_END_KEY, CueStage.EVENT),
            kind = CueKind.ROUTE_END,
            stage = CueStage.EVENT,
            priority = tuning.priorityNow,
            validUntilProgressM = Double.MAX_VALUE,
            text = VoicePhrases.ROUTE_END,
        )
    }

    private fun arrivalCue(state: NavState, generation: Long): VoiceCue? {
        // ══════════════════════════════════════════════════════════════
        // **ومن `arrivedAtTarget` لا من نهاية الخطّ**
        // ══════════════════════════════════════════════════════════════
        //
        // (إغلاقُ نقطة الالتقاط، ٢٠٢٦-٠٨-٢١.)
        //
        // **واللفظُ لم يتبدّل** — الشرطُ وحدَه هو ما صُحّح.
        //
        // **ومتجرٌ يبعد ١٨٠م عن أقرب طريق** يُعطي مساراً ينتهي
        // على بُعد ١٨٠م منه — **فكانت تُقال «وصلت» والسائقُ لم يصل.**
        if (!state.arrivedAtTarget) return null
        // **ومرّةً واحدةً في الجيل** — والمنسّقُ يحرس أيضاً، **لكنّ
        // توليدَ الشيء ستّ مرّاتٍ في الثانية ضجيجٌ في القياس.**
        if (arrivalSaid) return null
        arrivalSaid = true
        return VoiceCue(
            id = CueId(generation, ARRIVAL_KEY, CueStage.EVENT),
            kind = CueKind.ARRIVE,
            stage = CueStage.EVENT,
            priority = tuning.priorityNow,
            validUntilProgressM = Double.MAX_VALUE,
            text = VoicePhrases.arrival(target),
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **المناورة**
    // ══════════════════════════════════════════════════════════════════

    private fun maneuverCue(state: NavState, generation: Long, fix: NavFix): VoiceCue? {
        val p = state.progress ?: return null
        // **ولا تعليماتِ مناورةٍ وهو خارجَ المسار** — تعليمةٌ من مسارٍ
        // تركه ضرر. (البند ١٤ من التحليل.)
        if (state.isOffRoute) return null
        // **ولا تعليماتِ مناورةٍ وهو معاكس** — «انعطف يميناً» لمن يسير
        // عكسَ المسار **تعليمةٌ على طريقٍ لا يسلكه.**
        if (state.isWrongWay) return null
        // **والمرفوضةُ لا تولّد شيئاً ولا تمحو شيئاً** (البند ١٠).
        if (state.grade == FixGrade.REJECTED) return null

        // ══════════════════════════════════════════════════════════════
        // **والمقصودةُ هي «الجارية» لا «التالية»**
        // ══════════════════════════════════════════════════════════════
        //
        // **ودلالةُ المرحلة الثانية تخالف الاسم**: `RouteProgress`
        // تُقدّم `current` بمجرّد مجاوزة السابقة، **فتصير «الجارية» هي
        // المناورةُ القادمةُ أمامنا**، و`next` هي التي بعدها.
        //
        // **وقِيس أثرُ الخطأ** (٢٠٢٦-٠٨-٢١): استهدافُ `next` جعل
        // التنبيهَ يقول «بعد أربع مئة متر، لقد وصلت» **وأمامَ السائق
        // منعطفٌ لم يُذكر.**
        //
        // (وهي الدلالةُ نفسُها التي كشفتها معايرةُ ٣أ في سماح
        //  المناورة.)
        val target = p.current ?: p.next ?: return null
        val distance = max(0.0, target.atDistanceM - p.progressM)
        val speed = trustedSpeed(fix, p)

        // **وتبدّلُ المناورة يمسح ذاكرةَ العبور** — فلا يُقاس عبورُ
        // عتبةٍ بين مناورتين مختلفتين.
        if (target.atDistanceM != lastManeuverAtM) {
            lastManeuverAtM = target.atDistanceM
            lastDistanceM = Double.NaN
        }
        val previous = lastDistanceM
        lastDistanceM = distance

        // ══════════════════════════════════════════════════════════════
        // **ومناورةُ الوصول تُكتم — البندان ٨ و٩**
        // ══════════════════════════════════════════════════════════════
        //
        // (إغلاقُ دلالات الوصول، ٢٠٢٦-٠٨-٢١.)
        //
        // **كانت تقول «لقد وصلت» عند نهاية الخطّ** — والمتجرُ
        // قد يكون على بُعد مئةٍ وثمانين متراً. **وذلك ليس لفظاً
        // فحسب** — هو معنًى منافسٌ لما تملكه دعوى الوصول.
        //
        // **فإن كان للرحلة هدفٌ تجاريّ فالعبارةُ ملكُ `ArrivalGuard`
        // وحدَه** — **ولا نداءين متتاليين** (البند ١١).
        //
        // **ومن لا هدفَ له يبقى على سلوكه القديم** — فلا تنكسر
        // ملاحةٌ بلا طلب.
        if (target.kind == ManeuverKinds.ARRIVE && state.hasArrivalTarget) return null

        val stage = stageFor(previous, distance, speed) ?: return null
        if (!allowed(stage, state, fix.atMs)) {
            gatedByGps++
            return null
        }
        return build(stage, target, p, distance, generation, speed)
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **العبورُ لا المساواة**
     * ══════════════════════════════════════════════════════════════════
     *
     * (أمرُ المالك، البند ٧: «لا يجوز فقد `APPROACH` لأنّنا لم نقرأ
     *  ١٥٠م حرفيّاً».)
     *
     * **وقراءتان بينهما ثانيةٌ عند سبعين كم/س تقطعان تسعةَ عشرَ متراً**
     * — **والعتبةُ تُعبَر بينهما ولا تُلمَس.**
     *
     * # وبدايةٌ داخلَ العتبة
     *
     * (البند ٨: «لا تقل PREPARE وAPPROACH وNOW دفعةً واحدة».)
     *
     * **فأوّلُ قراءةٍ تُعطي أنفعَ طورٍ ينطبق وحدَه** — من فتح التطبيقَ
     * على بُعد ستّين متراً يسمع `NOW`، **ولا يسمع ثلاثاً متلاحقة.**
     */
    private fun stageFor(previousM: Double, nowM: Double, speed: Double): CueStage? {
        // **والأدنى أولى** — من عبر العتبات الثلاثَ دفعةً واحدة يريد
        // «الآن» لا «بعد خمس مئة متر». (البند ٨.)
        return when {
            crosses(previousM, nowM, triggerM(tuning.nowSeconds, speed, tuning.nowMinM, tuning.nowMaxM)) ->
                CueStage.NOW

            crosses(
                previousM, nowM,
                triggerM(tuning.approachSeconds, speed, tuning.approachMinM, tuning.approachMaxM),
            ) -> CueStage.APPROACH

            crosses(
                previousM, nowM,
                triggerM(tuning.prepareSeconds, speed, tuning.prepareMinM, tuning.prepareMaxM),
            ) -> CueStage.PREPARE

            else -> null
        }
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **والحدُّ يَقُصُّ العتبةَ ولا يُصفّي القراءة**
     * ══════════════════════════════════════════════════════════════════
     *
     * **وقِيس أثرُ الخلط** (٢٠٢٦-٠٨-٢١): كان الشرطُ «الثواني دون
     * العتبة **و** المسافةُ داخلَ الحدّين». وعند خمسةَ عشرَ كم/س تعطي
     * خمسَ عشرةَ ثانيةً **٦٢٫٥ متراً**، والحدُّ الأدنى ستّون —
     * **فالشريطُ الصالحُ مترانِ ونصف**، والقراءةُ تقطع أربعةً في
     * الثانية. **فتتخطّاه ولا تلمسه، فيسكت `APPROACH` كلُّه.**
     *
     * **والصوابُ أن تُقصَّ العتبةُ نفسُها** — فتبقى **نقطةً تُعبَر لا
     * شريطاً يُصاب.**
     */
    fun triggerM(thresholdSec: Double, speedMps: Double, minM: Double, maxM: Double): Double =
        (thresholdSec * speedMps).coerceIn(minM, maxM)

    /**
     * **العبورُ لا المساواة.**
     *
     * (أمرُ المالك، البند ٧: «لا يجوز فقد `APPROACH` لأنّنا لم نقرأ
     *  ١٥٠م حرفيّاً».)
     *
     * **وأوّلُ قراءةٍ لا سابقَ لها** — فتُعدّ عبوراً إن انطبقت، وهو ما
     * يجعل من بدأ داخلَ العتبة يسمع أنفعَ طورٍ وحدَه. (البند ٨.)
     */
    private fun crosses(previousM: Double, nowM: Double, triggerAtM: Double): Boolean {
        if (nowM > triggerAtM) return false
        if (previousM.isNaN()) return true
        return previousM > triggerAtM
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **السرعةُ الموثوقة — ولا زمنُ الرحلة كلِّها**
     * ══════════════════════════════════════════════════════════════════
     *
     * (تصحيحُ المالك ٢٠٢٦-٠٨-٢٠، البند ٦: «لا أريد استخدام زمن الرحلة
     *  المتبقّي بالكامل كبديلٍ لسرعة الحركة نحو المناورة الحاليّة».)
     *
     * **والبديلُ سرعةُ الخطوة نفسِها من المحرّك**: `stepDistanceM /
     * stepDurationS` — **وهي سرعةُ هذا الشارع لا متوسّطُ المشوار**،
     * وشارعٌ داخليٌّ في المشوار الطويل يُحسب بسرعة الأوتوستراد لو أُخذ
     * المتوسّط.
     */
    fun trustedSpeed(fix: NavFix, p: RouteProgress.State): Double {
        val gps = fix.speedMps
        if (gps != null && gps >= tuning.minTrustedSpeedMps) return gps.toDouble()
        val step = p.current
        if (step != null && step.stepDurationS > 0 && step.stepDistanceM > 0) {
            val v = step.stepDistanceM / step.stepDurationS
            if (v >= tuning.minTrustedSpeedMps) return min(v, tuning.maxAssumedSpeedMps)
        }
        // **ولا تقديرَ صالحٌ ⇒ افتراضٌ محافظٌ مركزيّ.**
        return tuning.fallbackSpeedMps
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **بوّابةُ الثقة — و`NOW` أشدُّها**
     * ══════════════════════════════════════════════════════════════════
     *
     * (تصحيحُ المالك ٢٠٢٦-٠٨-٢٠، البند ٩: «`NOW` هي أكثرُ Instruction
     *  حساسيّةً للموقع… `انعطف الآن` قد تكون أخطرَ من الصمت».)
     *
     * **ودقّةُ خمسين متراً تعني موضعاً في دائرةٍ قطرُها مئة** — و«الآن»
     * فيها قد تعني تقاطعاً سابقاً أو لاحقاً.
     *
     * **والمقياسُ زمنٌ رتيبٌ لا عددُ قراءات** — كما في معايرة ٣أ.
     */
    private fun allowed(stage: CueStage, state: NavState, fixAt: Long): Boolean {
        if (state.grade == FixGrade.ACCEPTED) return true
        if (trustedAtMs == 0L || fixAt == 0L) return false
        val age = fixAt - trustedAtMs
        return when (stage) {
            CueStage.NOW -> age <= tuning.nowTrustWindowMs
            else -> age <= tuning.softTrustWindowMs
        }
    }

    /**
     * **يبني التعليمة — ويدمج المتقاربتين.**
     *
     * (أمرُ المالك، البند ١١: «يمكن أن يكون الدمجُ مبنيّاً أيضاً على
     *  seconds between maneuvers وليس المسافةَ فقط».)
     *
     * **فالفجوةُ تُقاس زمناً بسرعة السير** — ومئةُ مترٍ عند خمسةَ عشرَ
     * كم/س أربعٌ وعشرون ثانية (جملتان تتّسعان)، **وعند سبعين خمسُ ثوانٍ
     * (لا تتّسعان).**
     */
    private fun build(
        stage: CueStage,
        target: NavManeuver,
        p: RouteProgress.State,
        distanceM: Double,
        generation: Long,
        speedMps: Double,
    ): VoiceCue {
        val after = maneuverAfter(p, target)
        val speed = max(tuning.minTrustedSpeedMps.toDouble(), tuning.fallbackSpeedMps)
        val gapSec = after?.let { (it.atDistanceM - target.atDistanceM) / speed }
        val merge = after != null && gapSec != null &&
            gapSec <= tuning.combineSeconds &&
            after.kind != ManeuverKinds.ARRIVE &&
            stage != CueStage.PREPARE

        val spoken = if (stage == CueStage.NOW) null else roundMeters(distanceM)
        val text = if (merge) {
            VoicePhrases.combined(target, after!!, spoken)
        } else {
            VoicePhrases.maneuver(target, spoken)
        }
        return VoiceCue(
            id = CueId(generation, target.atDistanceM, stage),
            kind = CueKind.MANEUVER,
            stage = stage,
            maneuver = target,
            roundedDistanceM = spoken,
            priority = when (stage) {
                CueStage.NOW -> tuning.priorityNow
                CueStage.APPROACH -> tuning.priorityApproach
                else -> tuning.priorityPrepare
            },
            // **وبعد المناورة بسماحٍ قصيرٍ تسقط** — انظر `VoiceCue`.
            validUntilProgressM = target.atDistanceM + tuning.staleAfterM,
            firedAtM = distanceM,
            firedSpeedMps = speedMps,
            text = text,
        )
    }

    /**
     * **المناورةُ التي تلي المقصودة.**
     *
     * **و`RouteProgress` تعطي اثنتين لا ثلاثاً** — فإن كانت المقصودةُ
     * هي `next` فلا ثالثةَ عندنا، **ولا تُخترع.**
     */
    private fun maneuverAfter(p: RouteProgress.State, target: NavManeuver): NavManeuver? {
        val n = p.next ?: return null
        return if (n.atDistanceM > target.atDistanceM) n else null
    }

    /** **وهل قيل الوصولُ في هذا الجيل؟** — فلا يُعاد كلَّ ثانية. */
    private var arrivalSaid = false

    /** **نوبةُ نهاية المسار التي نُطقت** — البند ١٢. */
    private var routeEndEpisode: Long = -1L

    /** **تقريبُ المسافة إلى ما يُنطق.** */
    fun roundMeters(m: Double): Int {
        val steps = intArrayOf(50, 100, 150, 200, 300, 400, 500, 700, 1000)
        var best = steps[0]
        for (s in steps) if (abs(s - m) < abs(best - m)) best = s
        return best
    }

    private fun event(generation: Long, kind: CueKind, text: String, priority: Int) = VoiceCue(
        id = CueId(generation, if (kind == CueKind.REROUTE) REROUTE_KEY else ARRIVAL_KEY, CueStage.EVENT),
        kind = kind,
        stage = CueStage.EVENT,
        priority = priority,
        validUntilProgressM = Double.MAX_VALUE,
        text = text,
    )

    private companion object {
        /** **مفاتيحُ الأحداث** — لا مناورةَ لها، فتُميَّز برقمٍ مستحيل. */
        const val ARRIVAL_KEY = -1.0

        /** **مفتاحُ نداء نهاية المسار** — آخرُ ميل، ٢٠٢٦-٠٨-٢١. */
        const val ROUTE_END_KEY = -3.0
        const val REROUTE_KEY = -2.0

        /** **ومفتاحُ الاتّجاه المعاكس** — ورقمُ النوبة يُطرح منه. */
        const val WRONG_WAY_KEY = -1000.0
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **كلُّ رقمٍ في الصوت هنا — ولا رقمَ خارجَه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك، البند ٥: «لا Magic Numbers موزّعة».)
 */
data class VoiceTuning(
    /** **الأطوارُ بالثواني** — لا بالمسافة. */
    val prepareSeconds: Double = 45.0,
    val approachSeconds: Double = 15.0,
    val nowSeconds: Double = 5.0,

    /**
     * **وحدودُ مسافةٍ تمنع السخف.**
     *
     * **الواقفُ لا يُنبَّه عند كيلومترين، والمسرعُ لا بعد فوات
     * الأوان.**
     */
    val prepareMinM: Double = 150.0,
    val prepareMaxM: Double = 800.0,
    val approachMinM: Double = 60.0,
    val approachMaxM: Double = 300.0,
    val nowMinM: Double = 15.0,
    val nowMaxM: Double = 120.0,

    /** **دون هذه السرعة لا تُصدَّق قراءةُ الجهاز** — كـ`BearingTracker`. */
    val minTrustedSpeedMps: Float = BearingTracker.MIN_TRUSTED_MPS,

    /** **وسقفُ ما يُفترض من سرعة الخطوة** — ٢٥ م/ث ≈ ٩٠ كم/س. */
    val maxAssumedSpeedMps: Double = 25.0,

    /**
     * **والافتراضُ المحافظُ حين لا تقديرَ صالح.**
     *
     * **وثمانيةُ أمتارٍ ≈ ٢٩ كم/س** — سرعةُ مدينةٍ معقولة: **لا
     * تُبكّر التنبيهَ كثيراً ولا تؤخّره.**
     */
    val fallbackSpeedMps: Double = 8.0,

    /**
     * **وفجوةٌ دونها تُدمج المناورتان** — بالثواني لا بالمتر.
     *
     * **وقِيس في المرحلة ٢** أنّ بين مناورتين ثمانيةَ أمتارٍ فعلاً.
     */
    val combineSeconds: Double = 8.0,

    /** **وبعد المناورة بهذا تسقط تعليمتُها.** */
    val staleAfterM: Double = 25.0,

    /**
     * **ونافذةُ الثقة لـ`NOW`** — بالمقياس الرتيب.
     *
     * **وثلاثُ ثوانٍ**: من لم تصله قراءةٌ مقبولةٌ منذ ثلاثِ ثوانٍ **لا
     * يُقال له «انعطف الآن».**
     */
    val nowTrustWindowMs: Long = 3_000L,

    /** **وأوسعُ لما دونها** — التمهيدُ يحتمل تقريبا. */
    val softTrustWindowMs: Long = 10_000L,

    val priorityNow: Int = 3,
    val priorityReroute: Int = 2,
    val priorityApproach: Int = 1,
    val priorityPrepare: Int = 0,
)
