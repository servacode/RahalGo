package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════
 * **مجموعةُ المسارات وحالُ اختيارها**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 *
 * # **والمعروضُ ليس المُلاحَ عليه** (البند ١٦ من التحليل)
 *
 * **أمرُ المالك نصّاً**: «تحميل RouteSet جديد لا يعني تلقائيًا تغيير
 * NavRoute الحالية إن كان المستخدم يتنقل بالفعل. يجب التفريق بين:
 * available routes · selected navigation route».
 *
 * **فهذا الصنفُ يحمل المتاحَ**، **و`NavigationSession` تحمل المختار.**
 * ولا يمسّ أحدُهما الآخرَ إلّا بأمرٍ صريح: `SelectRoute`.
 *
 * # **ولا وسمَ «الأسرع» ولا «الأقصر»** (البند ٣٣)
 *
 * **البدائلُ أرقامٌ**: `deltaDistanceM` و`deltaDurationS`. **والواجهةُ
 * تصوغهما** — «أقصرُ بـ٤٫٥كم · +١٢د». **ولا صفةَ مطلقةٌ تُدَّعى.**
 *
 * **و`deltaDurationS` من مدّة المحرّك** لا من المعدَّلة بسرعة السائق
 * (البند ٤) — **وإلّا صارت كلُّ المسارات بالسرعة نفسِها فاختفى الفرق.**
 */
data class RouteChoices(
    /** **معرِّفُ الجلبة** — يميّزها عن جلبةٍ أخرى للطلب نفسِه. */
    val setId: String,

    /**
     * **جيلُ الملاحة وقتَ الطلب** — البند ٢٠.
     *
     * **يُقارَن عند الاختيار بجيلِ المحرّك الآن**، فاختيارٌ من مجموعةٍ
     * سبقت إعادةَ حسابٍ **يُرفض.**
     */
    val generation: Long,

    /**
     * **الوجهة** — البند ٢١.
     *
     * **بديلٌ إلى الاستلام لا يصلح بعد `picked_up`** — والوجهةُ حقلٌ
     * صريحٌ لا يُستنتج.
     */
    val target: RouteTarget,

    /** **الأصلُ الذي بُنيت منه** — به يُقاس التقادم. */
    val originLat: Double,
    val originLng: Double,

    /** **الموصى به** — `routes[0]` من المحرّك، ولا يُرقّى غيرُه. */
    val recommended: RouteOption,

    /** **وما نجا من الترشيح** — بترتيب توصية المحرّك. */
    val alternatives: List<RouteOption> = emptyList(),

    /** **ما يُلاحَ عليه الآن** — وفارغٌ يعني الموصى به. */
    val selectedRouteId: String? = null,

    val loading: Boolean = false,
    val error: String? = null,
) {
    /** **كلُّ الخيارات** — الموصى به أوّلاً. */
    val all: List<RouteOption> get() = listOf(recommended) + alternatives

    /** **المسارُ المختار** — والافتراضُ الموصى به. */
    val selected: RouteOption
        get() = all.firstOrNull { it.routeId == selectedRouteId } ?: recommended

    /** **أثمّة ما يُعرض للاختيار؟** — البند ١ من قرار المالك. */
    val hasChoice: Boolean get() = alternatives.isNotEmpty()

    fun option(routeId: String): RouteOption? = all.firstOrNull { it.routeId == routeId }
}

/** **وجهةُ المسار** — البند ٢١. */
enum class RouteTarget {
    PICKUP,
    DROPOFF,
    ;

    companion object {
        fun of(raw: String?): RouteTarget =
            if (raw.equals("dropoff", ignoreCase = true)) DROPOFF else PICKUP
    }
}

/**
 * **خيارٌ واحد** — مسارٌ جاهزٌ للملاحة وأرقامُه.
 *
 * **جاهزٌ للملاحة** (البند ٢١): الهندسةُ والتراكميّةُ والمناوراتُ
 * كلُّها في الجلبة نفسِها. **فلا نداءَ شبكةٍ ثانٍ عند الاختيار.**
 */
data class RouteOption(
    val routeId: String,
    val route: NavRoute,

    /** **مدّةُ المحرّك** — لا المعدَّلةُ بسرعة السائق (البند ٤). */
    val engineDurationS: Double,
    val distanceM: Double,

    /** **الفرقُ عن الموصى به** — صفرٌ للموصى به نفسِه. */
    val deltaDistanceM: Double = 0.0,
    val deltaDurationS: Double = 0.0,

    /** **نسبةُ الاشتراك** — للتشخيص لا للعرض. */
    val sharedRatio: Double = 0.0,

    /**
     * ══════════════════════════════════════════════════════════════
     * **موضعُ القرار الأوّل** — إغلاقُ صحّة ٧، البنود ٦ إلى ١٠
     * ══════════════════════════════════════════════════════════════
     *
     * **بدايةُ أوّل فترةِ افتراقٍ ذاتِ معنى** عن الموصى به، **مقيسةً
     * على الموصى به** — فهو الذي يقوده السائق.
     *
     * **والأولى لا الأخيرة**: مساران يفترقان عند ١٢٥م ثمّ يلتقيان
     * ثمّ يفترقان عند ٩٠٠ — **القرارُ عند ١٢٥.** ومن تجاوزه
     * **اتُّخذ قرارُه الأوّلُ فعلاً**، ولا يُعرض له بديلٌ يحاول
     * إرجاعَه إلى فرعٍ فات.
     *
     * **وسالبٌ يعني: لا يفترق افتراقاً ذا معنى** — فلا يُبطله تقدّم.
     */
    val decisionDivergenceM: Double = -1.0,

    /** **وأوّلُ افتراقٍ أيّاً كان طولُه** — للتشخيص لا للقرار. */
    val firstDivergenceM: Double = -1.0,
) {
    val isRecommended: Boolean get() = deltaDistanceM == 0.0 && deltaDurationS == 0.0

    /**
     * **أما زال هذا البديلُ قابلاً للاختيار من هنا؟**
     *
     * **لكلّ بديلٍ قرارُه** (البند ٧): بديلٌ يتفرّع عند ١٢٥م يموت
     * بعدها، **وآخرُ يتفرّع عند ٨٠٠ يبقى.** فالحكمُ فرديٌّ لا جماعيّ.
     */
    fun isSelectableAt(progressM: Double, slackM: Double): Boolean {
        if (progressM < 0) return true
        if (decisionDivergenceM < 0) return true
        return progressM <= decisionDivergenceM + slackM
    }
}

/**
 * ══════════════════════════════════════════════════════════════════
 * **التقادم — أربعةُ شروطٍ أيُّها وقع أبطل**
 * ══════════════════════════════════════════════════════════════════
 *
 * (البند ٢٤.)
 *
 * **أمرُ المالك نصّاً**: «إذا السائق اجتاز نقطة التفرع الفعلية لمسار
 * معيّن، قد تكون Alternative قديمة قبل 300m».
 *
 * **فالشرطُ الأقوى هو نقطةُ التفرّع** حين تُعرف: **بديلٌ يتفرّع عند
 * ١٢٥م يبطل بعد ١٢٥م**، لا بعد ثلاث مئة. **وحركةُ الأصل حارسٌ
 * محافظٌ لما لا يُعرف تفرّعُه.**
 */
object RouteChoiceExpiry {

    data class Tuning(
        /**
         * **حركةُ الأصل** — حارسٌ إضافيٌّ لا أساسيّ (البند ١٣).
         *
         * **والأدقُّ هو موضعُ القرار** حين يُعرف. **وهذه لما لا
         * يُعرف تفرّعُه**، ولأنّ الأصلَ قد يتحرّك بلا تقدّمٍ على
         * المسار (وقوفٌ ثمّ عودةٌ من طريقٍ آخر).
         *
         * **وقِيست مواضعُ القرار**: ٤٩٥م في الطبقة، و٣٢٩م في دير
         * الزور، **و١١٩م في حلب** — فثلاثُ مئةٍ حارسٌ لا يُفرِّط.
         */
        val originMovedM: Double = 300.0,

        /** **والعمر** — مساوٍ لعمر مخبأ الخادم. */
        val maxAgeMs: Long = 10 * 60 * 1000,

        /**
         * **وهامشٌ بعد موضع القرار.**
         *
         * **فمن جاوزه بأمتارٍ ما زال قد يعود** — والغالبُ أنّه التزم.
         * **والهامشُ يمنع إبطالاً على الحدّ** حيث يهتزّ التقدّمُ
         * بضجيج الموقع.
         */
        val pastDivergenceSlackM: Double = 50.0,
    )

    enum class Reason {
        FRESH,
        GENERATION_CHANGED,
        TARGET_CHANGED,
        ORIGIN_MOVED,
        PAST_DIVERGENCE,
        TOO_OLD,
    }

    /**
     * **أما زالت المجموعةُ صالحة؟**
     *
     * @param progressM ما قطعه السائقُ على الموصى به — أو سالبٌ إن جُهل.
     */
    fun check(
        choices: RouteChoices,
        currentGeneration: Long,
        currentTarget: RouteTarget,
        originLat: Double,
        originLng: Double,
        progressM: Double,
        ageMs: Long,
        tuning: Tuning = Tuning(),
    ): Reason {
        if (choices.generation != currentGeneration) return Reason.GENERATION_CHANGED
        if (choices.target != currentTarget) return Reason.TARGET_CHANGED
        if (ageMs >= tuning.maxAgeMs) return Reason.TOO_OLD

        val moved = GpsQuality.metersBetween(
            choices.originLat, choices.originLng, originLat, originLng,
        )
        if (moved >= tuning.originMovedM) return Reason.ORIGIN_MOVED

        /**
         * **والمجموعةُ تموت حين يموت آخرُ بديلٍ فيها.**
         *
         * **والحكمُ الفرديُّ هو الأهمّ** — `RouteOption.isSelectableAt`
         * يمنع اختيارَ بديلٍ فات فرعُه **ولو بقي في المجموعة غيرُه.**
         * **وهذا الشرطُ للمجموعة كلِّها**: لا شيءَ يُعرض بعدُ.
         */
        if (progressM >= 0 && choices.alternatives.isNotEmpty()) {
            val anyAlive = choices.alternatives.any {
                it.isSelectableAt(progressM, tuning.pastDivergenceSlackM)
            }
            if (!anyAlive) return Reason.PAST_DIVERGENCE
        }
        return Reason.FRESH
    }

    fun isFresh(reason: Reason): Boolean = reason == Reason.FRESH
}

/**
 * ══════════════════════════════════════════════════════════════════
 * **أمرُ الاختيار — ويُرفض ما لا يصحّ**
 * ══════════════════════════════════════════════════════════════════
 *
 * (البندان ٢٣ و١٧ من التحليل.)
 *
 * **أمرُ المالك نصّاً**: «Command واضح: SelectRoute(setId, routeId)…
 * وقبل التنفيذ: setId current · generation current · target current ·
 * RouteSet not expired».
 *
 * **ولا يُعاد بناءُ الملاحة هنا** — أمرُ المالك: «والآليات الحالية
 * تتولى: generation++ · new RouteProgress · OffRoute reset · WrongWay
 * reset · Voice generation reset. لا تعيد كتابة هذه الأنظمة».
 * **فـ`NavEngine.setRoute` تفعلها كلَّها بالفعل.**
 */
object SelectRoute {

    sealed interface Verdict {
        /** **يُنفَّذ** — والمسارُ يُسلَّم إلى الجلسة. */
        data class Accept(val option: RouteOption) : Verdict

        /** **يُرفض** — ولا شيءَ يتغيّر. */
        data class Reject(val why: Why) : Verdict
    }

    enum class Why {
        UNKNOWN_SET,
        UNKNOWN_ROUTE,
        STALE,
        ALREADY_SELECTED,
    }

    /**
     * **يفحص ثمّ يقرّر** — ولا يمسّ الجلسة.
     *
     * **والفحصُ قبل التنفيذ لا بعده**: بين عرضِ المجموعة وضغطةِ
     * السائق **قد تكون إعادةُ حسابٍ نجحت** (البند ٢٠)، **أو تبدّلت
     * وجهةُ الطلب** (البند ٢١).
     */
    fun evaluate(
        choices: RouteChoices?,
        setId: String,
        routeId: String,
        currentGeneration: Long,
        currentTarget: RouteTarget,
        originLat: Double,
        originLng: Double,
        progressM: Double,
        ageMs: Long,
        tuning: RouteChoiceExpiry.Tuning = RouteChoiceExpiry.Tuning(),
    ): Verdict {
        val c = choices ?: return Verdict.Reject(Why.UNKNOWN_SET)
        if (c.setId != setId) return Verdict.Reject(Why.UNKNOWN_SET)

        val expiry = RouteChoiceExpiry.check(
            c, currentGeneration, currentTarget, originLat, originLng, progressM, ageMs, tuning,
        )
        if (!RouteChoiceExpiry.isFresh(expiry)) return Verdict.Reject(Why.STALE)

        val option = c.option(routeId) ?: return Verdict.Reject(Why.UNKNOWN_ROUTE)
        if (c.selectedRouteId == routeId) return Verdict.Reject(Why.ALREADY_SELECTED)

        /**
         * **وقرارُ هذا البديل بعينه** — البند ٧.
         *
         * **فالمجموعةُ قد تكون حيّةً ببديلٍ يتفرّع متأخّراً**،
         * **والمضغوطُ هو الذي فات فرعُه.** فيُرفض هو وحدَه.
         */
        if (!option.isSelectableAt(progressM, tuning.pastDivergenceSlackM)) {
            return Verdict.Reject(Why.STALE)
        }
        return Verdict.Accept(option)
    }
}
