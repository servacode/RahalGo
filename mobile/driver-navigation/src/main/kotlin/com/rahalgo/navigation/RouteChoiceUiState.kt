package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════
 * **حالُ اختيار المسار — معاينةٌ ثمّ اعتماد**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ واجهة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ١١ و١٢ و١٣.)
 *
 * # **ولماذا خطوتان لا واحدة**
 *
 * **أمرُ المالك نصّاً**: «لمسة خاطئة على الخريطة أثناء القيادة يجب
 * ألا تغيّر الملاحة والصوت والمناورات فورًا».
 *
 * **والضغطةُ الواحدة تعني**: جيلٌ جديد، وتقدّمٌ يُصفَّر، ونداءاتٌ
 * صوتيّةٌ تُعاد، وكاشفان يبدآن من الصفر. **وذلك ثمنٌ لا يُدفع بخطأٍ
 * في يدٍ على مِقود.**
 *
 * # **والمعاينةُ بصريّةٌ محضة** (البند ١٣)
 *
 * **لا `setRoute` ولا `RouteProgress` ولا `VoicePlanner` ولا
 * الكاشفان ولا `routeGeneration`.** **خطٌّ يُبرَز على الخريطة، لا
 * أكثر.**
 *
 * # **وهذا كلُّه في نموذج العرض** (البند ١٢)
 *
 * **لا في حالٍ محلّيٍّ داخل Composable** — فيضيع بالتدوير، **والسائقُ
 * يدير جهازَه في الحامل وهو يقود.**
 */
sealed interface RouteChoiceUi {

    /** **لا شيءَ يُعرض** — ولا بديلَ ولا مساحةَ تُؤخذ (البند ٥). */
    data object Hidden : RouteChoiceUi

    /** **خياراتٌ متاحةٌ ولا معاينة.** */
    data class Available(val choices: RouteChoices) : RouteChoiceUi

    /** **معاينةٌ جاريةٌ** — بصريّةٌ محضة. */
    data class Previewing(
        val choices: RouteChoices,
        val routeId: String,
    ) : RouteChoiceUi

    /** **اعتمادٌ قيدَ التنفيذ** — بين الفحص والتسليم. */
    data class Committing(
        val choices: RouteChoices,
        val routeId: String,
    ) : RouteChoiceUi

    /**
     * **المسارُ لم يعد متاحاً** — البند ١٩.
     *
     * **رسالةٌ قصيرةٌ غيرُ مزعجة**، ثمّ يعود الحالُ إلى ما تبقّى.
     */
    data class Stale(val choices: RouteChoices?) : RouteChoiceUi

    /** **وسقوطُ الجلب** — البند ٣٠: لا شاشةَ خطأ، والملاحةُ تستمرّ. */
    data object Error : RouteChoiceUi

    /** **الخياراتُ الحاضرةُ إن وُجدت.** */
    val choicesOrNull: RouteChoices?
        get() = when (this) {
            is Available -> choices
            is Previewing -> choices
            is Committing -> choices
            is Stale -> choices
            else -> null
        }

    /** **المسارُ المعايَنُ الآن** — أو لا شيء. */
    val previewRouteId: String?
        get() = when (this) {
            is Previewing -> routeId
            is Committing -> routeId
            else -> null
        }

    /** **أثمّة اعتمادٌ جارٍ؟** — فتُعطَّل الأزرار. */
    val busy: Boolean get() = this is Committing
}

/**
 * ══════════════════════════════════════════════════════════════════
 * **آلةُ حالِ الاختيار — خالصةٌ تُختبر بلا واجهة**
 * ══════════════════════════════════════════════════════════════════
 *
 * (البنود ١١ و١٦ و١٧ و١٨ و١٩ و٢٠ و٣٦.)
 *
 * **ولا Compose هنا ولا Android** — قرارٌ محضٌ يُقرأ ويُختبر.
 */
object RouteChoiceMachine {

    /** **سياقُ اللحظة** — ما تحتاجه كلُّ خطوة. */
    data class Context(
        val generation: Long,
        val target: RouteTarget,
        val originLat: Double,
        val originLng: Double,
        /** **ما قطعه السائقُ على الموصى به** — سالبٌ إن جُهل. */
        val progressM: Double,
        val ageMs: Long,
        /**
         * ══════════════════════════════════════════════════════════
         * **أحالُ الملاحة صحيح؟** — إغلاقُ صحّة الواجهة، البند ١٠
         * ══════════════════════════════════════════════════════════
         *
         * **يُبنى من `NavState.situation` لا من كاشفٍ واحد.**
         *
         * **وكان `offRoute == ON_ROUTE && reroute == NONE` وهو ناقص**:
         * `WrongWayDetector` **يقيس الاتّجاهَ نسبةً إلى المسار**،
         * **فقد يكون السائقُ داخلَ الممرّ تماماً وهو يسير عكسَه** —
         * `offRoute` تقول `ON_ROUTE` والحالُ `WRONG_WAY`.
         *
         * **فيُستعمل `RouteChoiceHealth.of(state)`** — وهو يقرأ
         * الحالَ المحسومةَ من المرحلة ٥.
         */
        val healthy: Boolean = true,
        val tuning: RouteChoiceExpiry.Tuning = RouteChoiceExpiry.Tuning(),
    )

    /**
     * **ما يُعرض** — البند ٣٦.
     *
     * **وتظهر اللوحةُ بخمسة شروطٍ مجتمعة**: خياراتٌ موجودة، وبديلٌ
     * حيٌّ واحدٌ على الأقلّ، والوجهةُ مطابقة، والجيلُ صالح، وحالُ
     * الملاحة يسمح.
     */
    fun present(choices: RouteChoices?, ctx: Context): RouteChoiceUi {
        val c = choices ?: return RouteChoiceUi.Hidden
        val alive = aliveAlternatives(c, ctx)
        if (alive.isEmpty()) return RouteChoiceUi.Hidden
        if (!ctx.healthy) return RouteChoiceUi.Hidden
        return RouteChoiceUi.Available(c.copy(alternatives = alive))
    }

    /**
     * **البدائلُ الحيّةُ الآن** — البند ٢٠.
     *
     * **وكلُّ بديلٍ يموت بقراره لا بقرار غيره**: بديلٌ يتفرّع عند
     * ١٢٥م يختفي بعدها، **وآخرُ عند ٨٠٠ يبقى.**
     */
    fun aliveAlternatives(choices: RouteChoices, ctx: Context): List<RouteOption> {
        val setReason = RouteChoiceExpiry.check(
            choices, ctx.generation, ctx.target,
            ctx.originLat, ctx.originLng, ctx.progressM, ctx.ageMs, ctx.tuning,
        )
        // **وما بطل جماعيّاً لا يُنقذه فرد** — جيلٌ تبدّل أو وجهةٌ
        // تبدّلت **تُبطل المجموعةَ كلَّها.**
        if (setReason != RouteChoiceExpiry.Reason.FRESH &&
            setReason != RouteChoiceExpiry.Reason.PAST_DIVERGENCE
        ) {
            return emptyList()
        }
        return choices.alternatives.filter {
            it.isSelectableAt(ctx.progressM, ctx.tuning.pastDivergenceSlackM)
        }
    }

    /**
     * **ضغطةٌ على خيار** — البندان ١١ و١٦.
     *
     * **الموصى به يُلغي المعاينة** ولا يستدعي شيئاً — **فهو الفعّالُ
     * أصلاً.** **والبديلُ يدخل معاينةً لا اعتماداً.**
     */
    fun onSelect(current: RouteChoiceUi, routeId: String, ctx: Context): RouteChoiceUi {
        val c = current.choicesOrNull ?: return current
        if (current.busy) return current

        // **الموصى به** — إلغاءُ معاينةٍ إن كانت.
        if (routeId == c.recommended.routeId) {
            return RouteChoiceUi.Available(c)
        }

        val alive = aliveAlternatives(c, ctx)
        if (alive.none { it.routeId == routeId }) {
            // **بديلٌ فات فرعُه** — يُبلَّغ ولا يُعايَن.
            return RouteChoiceUi.Stale(c.copy(alternatives = alive))
        }
        return RouteChoiceUi.Previewing(c.copy(alternatives = alive), routeId)
    }

    /** **إلغاءُ المعاينة** — البند ١٦. */
    fun cancelPreview(current: RouteChoiceUi): RouteChoiceUi {
        if (current.busy) return current
        val c = current.choicesOrNull ?: return RouteChoiceUi.Hidden
        return RouteChoiceUi.Available(c)
    }

    /**
     * **بدءُ الاعتماد** — البند ١٨.
     *
     * **يُفحص أوّلاً ثمّ يُسلَّم** — **ولا واجهةٌ تتفاءل.** فالترتيب:
     *
     *	فحص  →  تسليمُ الملاحة  →  الواجهةُ تعكس ما وقع
     */
    sealed interface Commit {
        /** **مقبول** — يُسلَّم إلى الجلسة. */
        data class Accept(val option: RouteOption, val next: RouteChoiceUi) : Commit

        /** **مرفوض** — ولا يُمسّ المسارُ الحاليّ (البند ١٩). */
        data class Reject(val why: SelectRoute.Why, val next: RouteChoiceUi) : Commit
    }

    fun commit(current: RouteChoiceUi, ctx: Context): Commit {
        val c = current.choicesOrNull
        val routeId = current.previewRouteId
        if (c == null || routeId == null) {
            return Commit.Reject(SelectRoute.Why.UNKNOWN_ROUTE, RouteChoiceUi.Hidden)
        }
        if (!ctx.healthy) {
            return Commit.Reject(SelectRoute.Why.STALE, RouteChoiceUi.Stale(c))
        }

        val verdict = SelectRoute.evaluate(
            choices = c,
            setId = c.setId,
            routeId = routeId,
            currentGeneration = ctx.generation,
            currentTarget = ctx.target,
            originLat = ctx.originLat,
            originLng = ctx.originLng,
            progressM = ctx.progressM,
            ageMs = ctx.ageMs,
            tuning = ctx.tuning,
        )

        return when (verdict) {
            is SelectRoute.Verdict.Accept ->
                Commit.Accept(verdict.option, RouteChoiceUi.Committing(c, routeId))

            is SelectRoute.Verdict.Reject -> {
                /**
                 * **والمرفوضُ يُزال من المعروض** (البند ١٩): «ألغِ
                 * Preview · أزل Alternative المنتهية · لا تغيّر
                 * Current navigation».
                 */
                val alive = aliveAlternatives(c, ctx).filter { it.routeId != routeId }
                Commit.Reject(
                    verdict.why,
                    RouteChoiceUi.Stale(c.copy(alternatives = alive)),
                )
            }
        }
    }

    /**
     * **بعد نجاح التسليم** — البند ١٧.
     *
     * **والمجموعةُ القديمةُ انتهت بنيويّاً**: `setRoute` زادت الجيل،
     * **فما بُني على الجيل السابق لا يُقارَن به.**
     */
    fun afterCommitted(): RouteChoiceUi = RouteChoiceUi.Hidden

    /**
     * **ومسحٌ صريح** — عند تبدّل الوجهة أو نجاح إعادة الحساب أو نهاية
     * الرحلة (البندان ٢٢ و٢٣).
     */
    fun clear(): RouteChoiceUi = RouteChoiceUi.Hidden
}

/**
 * ══════════════════════════════════════════════════════════════════
 * **صحّةُ الحال للاختيار — من الحال الموحَّدة**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ صحّة الواجهة، البنود ١٠ و١١ و١٢.)
 *
 * **ولا يُبنى على كاشفٍ واحد**: `NavSituation` تحسم بينهما بالترتيب
 * الذي قرّره المالكُ في المرحلة ٥ — **إعادةُ الحساب فوق كلّ شيء، ثمّ
 * الخروج، ثمّ الاتّجاه المعاكس، ثمّ الشكوك.**
 *
 * # **والشكُّ يمنع أيضاً**
 *
 * **`SUSPECTED_OFF_ROUTE` و`SUSPECTED_WRONG_WAY` تمنعان** — أمرُ
 * المالك: «SUSPECTED_OFF_ROUTE · OFF_ROUTE · SUSPECTED_WRONG_WAY ·
 * WRONG_WAY · REROUTING → لا Preview commit».
 *
 * **والسببُ أنّ الشكَّ لحظةُ قرارٍ**: السائقُ يوشك أن يترك الممرَّ،
 * **وتبديلُ مسارٍ في تلك اللحظة يزيد التشويش.**
 */
object RouteChoiceHealth {

    /**
     * **أيُسمح بالاختيار الآن؟**
     *
     * **ولا حالَ يعني: لم تبدأ الملاحة بعد** — والاختيارُ مسموحٌ،
     * فلا شيءَ يُشوَّش.
     */
    fun of(state: NavState?): Boolean {
        val s = state ?: return true
        if (s.reroute != RerouteStatus.NONE) return false
        return s.situation == NavSituation.ON_ROUTE
    }

    /**
     * **وما يمنع** — للتشخيص لا للعرض.
     */
    fun blockedBy(state: NavState?): NavSituation? {
        val s = state ?: return null
        return s.situation.takeIf { it != NavSituation.ON_ROUTE }
    }
}
