package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════════
 * **محرّكُ الملاحة — قراءةٌ تدخل، وحالٌ كاملةٌ تخرج**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٣أ، أمرُ المالك ٢٠٢٦-٠٨-٢٠.)
 *
 *     NavFix
 *       ↓  NavPipeline       ترشيحٌ وتنعيمٌ وحركة
 *       ↓  RouteProgress     أين هو · كم بقي · ما المناورة
 *       ↓  OffRouteDetector  أخرج عن المسار؟
 *     NavState
 *
 * # ولماذا انفصل عن `NavigationSession`
 *
 * **`NavigationSession` تحمل أندرويد**: سياقاً ومحرّكَ مواقعَ وحالةَ
 * Compose. **فلا تُبنى في اختبارِ وحدة** — ولذلك بقيت المرحلتان الأولى
 * والثانيةُ تُختبران على `NavPipeline` وحدَه.
 *
 * **وهذا الصنفُ لا يعرف أندرويد** — فتُعاد رحلةٌ كاملةٌ عليه: مسارٌ
 * وقراءاتٌ وحالٌ متوقَّعة. **وهو ما تقوم عليه رفائدُ المرحلة ٣.**
 *
 * # وما لا يفعله
 *
 * **لا نداءَ شبكةٍ ولا إعادةَ حساب.** (أمرُ المالك للمرحلة ٣أ: «في ٣أ
 * نريد إثباتَ أنّ النظام يستطيع أن يقول بثقة: السائق خرج عن المسار.
 * فقط».)
 *
 * **ولا يعرف طلباً ولا سائقاً** — يعرف مساراً ونقاطاً وأزمنة.
 */
class NavEngine(
    val pipeline: NavPipeline = NavPipeline(),
    val detector: OffRouteDetector = OffRouteDetector(),
    /** **كاشفُ الاتّجاه المعاكس** — المرحلة ٥. */
    val wrongWay: WrongWayDetector = WrongWayDetector(),
    /**
     * **بابُ الشبكة — وفارغٌ يعني «لا إعادةَ حساب».**
     *
     * **والمرحلةُ ٣أ كلُّها تعمل بلاه** — الكشفُ محلّيٌّ محض،
     * **وإعادةُ الحساب زيادةٌ لا شرط.**
     */
    source: RouteSource? = null,
    rerouteTuning: RerouteEngine.Tuning = RerouteEngine.Tuning(),
    /**
     * **مخطِّطُ الصوت — وفارغٌ يعني «لا إرشادَ صوتيّ».**
     *
     * (المرحلة ٤.)
     *
     * **ولا يقول شيئاً** — يُخرج قراراً في `NavState.cues`،
     * **والقولُ في تطبيق السائق.**
     */
    val voice: VoicePlanner? = null,
) {

    private var route: NavRoute? = null
    private var progress: RouteProgress? = null

    /**
     * ══════════════════════════════════════════════════════════════════
     * **جيلُ المسار — يزيد مع كلّ تسليم**
     * ══════════════════════════════════════════════════════════════════
     *
     * (البند ١٨.)
     *
     * **وجوابٌ بُني على جيلٍ مضى لا يستبدل مساراً أحدث** — ومنه حالُ
     * تغيّرِ طور الرحلة (البند ١٩): **من طلب طريقاً إلى المتجر ثمّ
     * استلم، تُسلَّم شاشتُه طريقاً إلى الزبون فيزيد الجيل**، فيُطرح
     * جوابُه القديم.
     */
    var generation = 0L
        private set

    /**
     * **آخرُ قراءةٍ مقبولةٍ تماماً** — نقطةُ بدء إعادة الحساب.
     *
     * (أمرُ المالك، البند ٥: «لا تستخدم Projected position على Route
     *  القديم… ولا PathSmoother visual output».)
     *
     * **وهي القراءةُ الخامّةُ بعد حارس الجودة** — لا موضعٌ مُنعَّمٌ
     * ولا مُسقَط: **من خرج عن المسار لا يُبدأ به من مسارٍ تركه.**
     */
    var lastAcceptedFix: NavFix? = null
        private set

    /** **محرّكُ إعادة الحساب** — وفارغٌ إن لم يُعطَ باباً. */
    val reroute: RerouteEngine? = source?.let {
        RerouteEngine(it, rerouteTuning, ::installRerouted) { generation }
    }

    /** **المسارُ النافذُ الآن** — يُقرأ في التشخيص. */
    val currentRoute: NavRoute? get() = route

    /**
     * ══════════════════════════════════════════════════════════════
     * **الهدفُ التجاريّ — إغلاقُ نقطة الالتقاط، ٢٠٢٦-٠٨-٢١**
     * ══════════════════════════════════════════════════════════════
     *
     * **المتجرُ أو بابُ الزبون — لا حيثُ ينتهي الخطّ.**
     *
     * **والمحرّكُ يلتقط على أقرب طريقٍ حتّى ٢٥٠م** (المرحلة ٨أ)،
     * **فنهايةُ الخطّ ليست الهدف.**
     *
     * **و`null` تعني: لم يُعطَ** — فيُعمل بالسلوك القديم،
     * **ولا تنكسر شاشةٌ لم تُحدَّث بعد.**
     */
    var arrivalTarget: GeoPoint? = null
        set(value) {
            // **وهدفٌ جديدٌ يفكّ آخرَ ميل** — البند ٦-ب.
            if (value != field) guidanceEnded = false
            field = value
        }

    /**
     * **أانتهى إرشادُ الطريق؟** — مِزلاجٌ لا يُفكّ بالحركة.
     *
     * **ولا يفكّه إلّا**: مسارٌ جديدٌ (`setRoute`) أو هدفٌ جديد.
     */
    var guidanceEnded: Boolean = false
        private set

    /**
     * ══════════════════════════════════════════════════════════════
     * **هويّةُ المسار النافذ — المرحلة ٨ب**
     * ══════════════════════════════════════════════════════════════
     *
     * **معرّفٌ محايدٌ أعطاه الخادمُ مع المسار** — لا يُخترع هنا
     * ولا يُشتقّ. **وبه يُقارَن الأثر.**
     *
     * **و`null` تعني: لا هويّة** — فلا مقارنةَ ولا تخمين
     * (البند ١٩: لا ارتدادَ إلى مسارٍ مضى).
     */
    var routeId: String? = null

    /**
     * **مُحلِّلُ الطريق الموازي** — ورايتُه مطفأةٌ افتراضاً.
     *
     * **البند ١**: المعايرةُ تنتظر قياساً ميدانيّاً في الرقّة.
     */
    val parallel: ParallelResolver =
        ParallelResolver(enabled = NavFeatures.parallelResolver)

    /**
     * **طلبُ ارتباطٍ يجب إرسالُه الآن** — أو `null`.
     *
     * **تقرؤه الشاشةُ بعد كلّ قراءة** — ولا يُرسله المحرّك بنفسه:
     * **لا شبكةَ في `driver-navigation`** (البند ١٤).
     */
    var correlationRequest: CorrelationRequest? = null
        private set

    /** **آخرُ حالٍ خرجت** — وفارغةٌ قبل أوّل قراءة. */
    var state: NavState? = null
        private set

    /**
     * ══════════════════════════════════════════════════════════════════
     * **والمسارُ يُبدَّل وحدةً واحدة**
     * ══════════════════════════════════════════════════════════════════
     *
     * (أمرُ المالك: «يجب أن تستقبل `NavigationSession` المسارَ كوحدةٍ
     *  واحدة».)
     *
     * **و`NavRoute` صنفٌ ثابتٌ** — هندستُه وتراكميّتُه ومناوراتُه فيه
     * معاً. **فالاستبدالُ إسنادُ مرجعٍ لا تعديلُ حقول**، ولا لحظةَ
     * تكون فيها هندسةٌ جديدةٌ مع مناوراتٍ قديمة.
     *
     * **وفارغٌ يعني «لا ملاحة»** — الخريطةُ ترسم وحدَها ولا يسقط شيء.
     */
    fun setRoute(next: NavRoute?) {
        route = next?.takeIf { it.usable }
        // **وتقدّمٌ جديدٌ لا تقدّمٌ مُعادُ الضبط** — كائنٌ جديدٌ على
        // مرجعٍ جديد، **فلا يبقى فيه أثرٌ من مسارٍ مضى.**
        progress = route?.let { RouteProgress(it) }
        // **ودليلُ الخروج القديمُ لا يلوّث الجديد** (البند ١٣).
        detector.reset()
        wrongWay.reset()
        // **ومسارٌ جديدٌ يفكّ آخرَ ميل** — البند ٦-ج: تستأنف الملاحةُ نظيفة.
        guidanceEnded = false
        generation++
        reroute?.reset()
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **والمسارُ الجديد يُجرَّب قبل أن يُعتمد**
     * ══════════════════════════════════════════════════════════════════
     *
     * (البند ١٦.)
     *
     * **يُسقَط عليه موضعُ السائق الآن**، فإن كان بعيداً عنه **فتركيبُه
     * يُخرجه فورا** — `install → OFF_ROUTE → reroute → install`،
     * **وتلك حلقةٌ لا تنتهي.**
     *
     * **ولا يُحكَم بالبصمة** (البند ١٥): **مسارٌ يعود كما هو قد يكون
     * أفضلَ طريقٍ بحقّ** — والسؤالُ أيصلح من موضعه الآن، لا أهو جديد.
     */
    private fun installRerouted(next: NavRoute, at: NavFix): InstallOutcome {
        if (!next.usable) return InstallOutcome.INVALID
        val hit = RouteProjector.project(
            next, at.lat, at.lng, 0.0, 0.0, at.speedMps, null, fullScan = true,
        ) ?: return InstallOutcome.INVALID
        if (hit.offRouteM > (reroute?.tuning?.maxStartOffsetM ?: Double.MAX_VALUE)) {
            return InstallOutcome.UNUSABLE
        }
        setRoute(next)
        // **ويُبذَر التقدّمُ بالقراءة نفسِها** — فلا تبقى الشاشةُ بلا
        // حالٍ حتّى تصل التالية.
        val seeded = progress?.onFix(at, pipeline.headingDeg)
        // **والحالُ المعروضةُ تتبدّل الآن لا عند القراءة التالية.**
        //
        // **وبلاها تعرض الشاشةُ ما بقي من مسارٍ استُبدل** — قِيس
        // (٢٠٢٦-٠٨-٢٠): ٣٩٠م من القديم بينما الجديدُ ٩٠٠.
        state = state?.copy(progress = seeded, hasRoute = true)
        return InstallOutcome.INSTALLED
    }

    /**
     * **يُغذّى قراءةً فيردّ ما يُعرض وما يُقرَّر.**
     *
     * **والمرفوضةُ تمرّ أيضاً** — فترى الشاشةُ أنّ شيئاً لم يتبدّل،
     * **ويرى الكاشفُ أنّها لم تصل.**
     */
    fun onFix(fix: NavFix): NavState {
        val step = pipeline.onFix(fix)

        // **ولا يُغذّى التقدّمُ إلّا بما قُبل** — `RouteProgress`
        // يقول ذلك نصّاً: «ولا تُغذّى المرفوضة».
        // **وتقدّمٌ حُسب الآن على المسار الحاليّ** — يُميَّز عمّا وُرث.
        //
        // **ولولا هذا التمييز**: قراءةٌ مرفوضةٌ بعد تركيب مسارٍ جديدٍ
        // تُعيد تقدّمَ المسار المنتهي — **فيُقفل مِزلاجُ آخرِ ميلٍ على
        // ملاحةٍ لم تبدأ بعد.** (وقع في الرفيدة «ح».)
        val fresh = if (step.grade == FixGrade.REJECTED) {
            null
        } else {
            progress?.onFix(fix, step.bearingDeg)
        }
        val p = fresh ?: state?.progress

        if (step.grade == FixGrade.ACCEPTED) lastAcceptedFix = fix

        // ══════════════════════════════════════════════════════════════
        // **وإرشادُ الطريق إذا انتهى انتهى — إغلاقُ آخر ميل**
        // ══════════════════════════════════════════════════════════════
        //
        // (قرارُ المالك ٢٠٢٦-٠٨-٢١، البندان ٤ و٥.)
        //
        // **العيبُ المقيس**: متجرٌ يبعد مئةً وثمانين متراً عن أقرب
        // طريق — يبلغ السائقُ نهايةَ الخطّ **ثمّ يتحرّك نحوَ الزبون
        // فيشتعل `OFF_ROUTE`** وتنطلق إعادةُ الحساب.
        //
        // **قِيس قبل الإصلاح** (أربعون قراءةً بسرعة ١٦ م/ث): `OFF_ROUTE`
        // مؤكّدة بنقاطٍ مُشبَعة، **وأربعةُ طلباتِ إعادةِ حساب.**
        //
        // **والمسارُ انتهى أصلاً** — فالابتعادُ عن هندسةٍ منتهيةٍ ليس
        // خروجاً بالمعنى الملاحيّ، **وإعادةُ حسابٍ تردّ النقطةَ نفسَها
        // حلقةٌ لا إرشاد.**
        //
        // **والكتمُ هنا لا في الكواشف** (البند ٥): لا `if (lastMile)`
        // داخلَ `OffRouteDetector` ولا `WrongWayDetector` ولا
        // `RerouteEngine` — **هذه طبقةُ التنسيق، وهي من تقرّر متى
        // تُسأل.**
        //
        // **ومِزلاجٌ لا شرطٌ لحظيّ**: من بلغ النهاية لا يعود إلى
        // الإرشاد لمجرّدِ أنّه ابتعد (البند ٦) — **ولا يُفكّه إلّا
        // مسارٌ جديدٌ أو هدفٌ جديد.**
        if (fresh?.arrived == true) guidanceEnded = true

        // ══════════════════════════════════════════════════════════════
        // **والاشتباهُ في طريقٍ موازٍ — المرحلة ٨ب**
        // ══════════════════════════════════════════════════════════════
        //
        // **ويُغذّى بالإسقاط لا بالتقدّم** — `lateralSignedM` تقول
        // الجهةَ، **و`RouteProgress` لا تحمل تاريخاً** (البند ١٧ من
        // التحليل: لا تصير مخزنَ تاريخِ مُحلِّل).
        //
        // **ولا شبكةَ هنا**: يُخرج طلباً تقرؤه الشاشةُ وترسله.

        val verdict = if (guidanceEnded) {
            detector.idle()
        } else {
            detector.onFix(fix, step.grade, step.bearingDeg, route, p)
        }
        val wrong = if (guidanceEnded) {
            wrongWay.idle()
        } else {
            wrongWay.onFix(fix, step.grade, step.bearingDeg, route, p, generation)
        }
        val lastState = state
        correlationRequest = parallel.onFix(
            fix, step.grade, fresh, lastState, routeId, generation,
        )

        var out = NavState(
            grade = step.grade,
            reason = step.reason,
            lat = step.targetLat,
            lng = step.targetLng,
            bearingDeg = step.bearingDeg,
            animationMs = step.animationMs,
            hasRoute = route != null,
            progress = p,
            offRoute = verdict,
            wrongWay = wrong,
            // **والوصولُ يُحكم على الهدف لا على نهاية الخطّ.**
            //
            // **و`progress.arrived` تبقى كما هي** — دلالتُها «بلغتُ
            // نهايةَ المسار»، **وهي صحيحةٌ لإرشادٍ وعرض.**
            hasArrivalTarget = arrivalTarget != null,
            // ══════════════════════════════════════════════════════════
            // **و«انتهى الإرشاد» يُقرأ من المِزلاج لا من القراءة**
            // ══════════════════════════════════════════════════════════
            //
            // **و`p.arrived` لحظيّة**: قراءةٌ مرفوضةٌ تُعيد تقدّمَ مسارٍ
            // مضى فتقول «بلغتُ النهاية» والرحلةُ لم تبدأ (الرفيدة «ح»)،
            // **وقراءةٌ بعيدةٌ تقول «لم أبلغ» بعد أن بلغ.**
            //
            // **والمِزلاجُ يُقفل بتقدّمٍ حُسب على المسار الحاليّ وحدَه،
            // ولا يُفكّ إلّا بمسارٍ جديدٍ أو هدفٍ جديد.**
            arrivalPhase = ArrivalGuard.phase(
                routeArrived = guidanceEnded,
                target = arrivalTarget,
                fix = fix,
                grade = step.grade,
            ),
            targetDistanceM = ArrivalGuard.distanceToTargetM(arrivalTarget, fix),
        ).also { it.wrongWayWindowElapsed = wrongWay.correctionWindowElapsed() }
        // **وإعادةُ الحساب تُنظَر بعد الحكم لا قبله** — ولا تُطلق إلّا
        // من قراءةٍ مقبولةٍ تماماً: **نقطةُ بدءٍ مشكوكٌ فيها تُنتج
        // مساراً مشكوكاً فيه.**
        val start = if (step.grade == FixGrade.ACCEPTED) fix else lastAcceptedFix
        val status = if (start == null) {
            reroute?.status ?: RerouteStatus.NONE
        } else {
            reroute?.consider(start, out.rerouteReason, fix.atMs) ?: RerouteStatus.NONE
        }
        out = out.copy(reroute = status)
        // **والصوتُ يُخطَّط بعد كلّ شيء** — يقرأ الحالَ النهائيّةَ
        // نفسَها التي تقرؤها الشاشة، **فلا يقول عن حالٍ لم تُعرض.**
        out = out.copy(cues = voice?.onState(out, generation, fix) ?: emptyList())
        state = out
        return out
    }

    /** **يُنسى كلُّ شيء** — عند بدء جلسةٍ جديدة. */
    fun reset() {
        pipeline.reset()
        detector.reset()
        wrongWay.reset()
        reroute?.reset()
        voice?.reset()
        progress = route?.let { RouteProgress(it) }
        lastAcceptedFix = null
        state = null
    }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالُ الملاحة كاملةً — تُقرأ ولا تُحسَب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك: «ولا تجعل UI نفسَها تحسب `RouteProgress` أو
 *  `OffRoute`».)
 *
 * **فالشاشةُ تقرأ حقولاً** — ولا تعرف إسقاطاً ولا نقاطَ دليل.
 */
data class NavState(
    val grade: FixGrade,
    val reason: RejectReason,
    /** **موضعُ الملاحة** — وفارغٌ إن رُفضت القراءة. */
    val lat: Double?,
    val lng: Double?,
    val bearingDeg: Float?,
    val animationMs: Long,
    val hasRoute: Boolean,
    /** **تقدّمُه على المسار** — وفارغٌ قبل أوّل إسقاطٍ أو بلا مسار. */
    val progress: RouteProgress.State?,
    val offRoute: OffRouteDetector.Verdict,
    /** **حكمُ الاتّجاه** — المرحلة ٥. */
    val wrongWay: WrongWayDetector.Verdict = WrongWayDetector.Verdict(
        WrongWayDetector.State.CORRECT_DIRECTION, null, 0L, 0.0, false,
        WrongWayDetector.Skip.NO_ROUTE, 0L,
    ),
    /** **حالُ إعادة الحساب** — فعلُ شبكةٍ لا دليلُ ملاحة. */
    val reroute: RerouteStatus = RerouteStatus.NONE,
    /**
     * **ما يجب أن يُقال عند هذه القراءة** — وفارغةٌ في الأغلب.
     *
     * **وقرارٌ لا فعل**: من يقرأها هو من ينطق. (المرحلة ٤.)
     */
    val cues: List<VoiceCue> = emptyList(),

    /**
     * **أيُّ الحالات الثلاث؟** — إغلاقُ دلالات الوصول.
     *
     * **وهذا غيرُ `progress.arrived`**: تلك تقول «بلغتُ نهايةَ
     * الخطّ» — **وهي صحيحةٌ للإرشاد، ولا تُقال للسائق.**
     */
    val arrivalPhase: ArrivalPhase = ArrivalPhase.EN_ROUTE,

    /**
     * **أللرحلة هدفٌ تجاريّ؟** — وبه تُكتم مناورةُ الوصول.
     *
     * **ومن لا هدفَ له يبقى على سلوكه القديم** — ملاحةٌ بلا
     * طلبٍ تسمع «لقد وصلت» كما كانت.
     */
    val hasArrivalTarget: Boolean = false,

    /** **كم يبعد عن هدفه** — وسالبٌ: لا هدفَ أو لا قراءة. */
    val targetDistanceM: Double = -1.0,
) {
    /** **كم بقي بالمتر** — وسالبٌ يعني «لا يُعرف». */
    val remainingM: Double get() = progress?.remainingM ?: -1.0

    /** **وكم بقي بالثانية.** */
    val remainingSec: Double get() = progress?.remainingSec ?: -1.0

    val currentManeuver: NavManeuver? get() = progress?.current
    val nextManeuver: NavManeuver? get() = progress?.next

    /** **أخرج عن المسار قطعاً؟** — الشكُّ وحدَه لا يُعرض للسائق. */
    val isOffRoute: Boolean get() = offRoute.state == OffRouteDetector.State.OFF_ROUTE

    /** **أيسير بعكس المسار قطعاً؟** */
    val isWrongWay: Boolean get() = wrongWay.state == WrongWayDetector.State.WRONG_WAY

    /** **وصل إلى الهدف التجاريّ** — ووحدَها تُطلق العبارة. */
    val arrivedAtTarget: Boolean get() = arrivalPhase == ArrivalPhase.BUSINESS_TARGET_ARRIVED

    /** **انتهى الخطّ ولمّا يصل** — يمشي آخرَ الأمتار بنفسه. */
    val lastMile: Boolean
        get() = arrivalPhase == ArrivalPhase.LAST_MILE_TO_TARGET

    /**
     * ══════════════════════════════════════════════════════════════════
     * **حالٌ واحدةٌ محسومة — فلا تتناقض الشاشةُ والصوت**
     * ══════════════════════════════════════════════════════════════════
     *
     * (المرحلة ٥، أمرُ المالك، البند ١٨.)
     *
     *     REROUTING  ←  فعلُ شبكةٍ يُعرض فوق كلّ شيء
     *     OFF_ROUTE  ←  ترك الممرَّ — يسبق الاتّجاه دائماً
     *     WRONG_WAY  ←  فيه ومعكوس
     *     ثمّ الشكوك، ثمّ على المسار
     */
    val situation: NavSituation
        get() = when {
            reroute == RerouteStatus.REROUTING -> NavSituation.REROUTING
            isOffRoute -> NavSituation.OFF_ROUTE
            isWrongWay -> NavSituation.WRONG_WAY
            offRoute.state == OffRouteDetector.State.SUSPECTED_OFF_ROUTE ->
                NavSituation.SUSPECTED_OFF_ROUTE
            wrongWay.state == WrongWayDetector.State.SUSPECTED_WRONG_WAY ->
                NavSituation.SUSPECTED_WRONG_WAY
            else -> NavSituation.ON_ROUTE
        }

    /**
     * **أيُطلَب مسارٌ جديدٌ الآن — ولماذا؟**
     *
     * **والخروجُ يسبق ولا ينتظر نافذةَ تصحيح** (أمرُ المالك، البند
     * ١٩): **من ترك الممرَّ لن يُصلحه أن يدور.**
     */
    val rerouteReason: RerouteReason?
        get() = when {
            isOffRoute -> RerouteReason.OFF_ROUTE
            isWrongWay && wrongWayWindowElapsed -> RerouteReason.WRONG_WAY
            else -> null
        }

    /** **تُملأ من المحرّك** — نافذةُ التصحيح مضت. */
    var wrongWayWindowElapsed: Boolean = false
        internal set
}

/** **الحالُ المحسومةُ للعرض والصوت** — المرحلة ٥. */
enum class NavSituation {
    ON_ROUTE,
    SUSPECTED_WRONG_WAY,
    SUSPECTED_OFF_ROUTE,
    WRONG_WAY,
    OFF_ROUTE,
    REROUTING,
}
