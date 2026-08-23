package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **اختيارُ المسار وسباقاتُه — البنود ٢٠ و٢١ و٢٣ و٢٤**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك نصّاً** (البند ٢٠ من التحليل):
 *
 *	RouteSet generation 4 معروضة
 *	↓ reroute يحدث
 *	↓ generation 5 مركَّبة
 *	↓ السائق يضغط Alternative قديمة من generation 4
 *	يجب رفضها.
 */
class RouteChoicesTest {

    private val originLat = 35.9500
    private val originLng = 39.0050

    private fun option(id: String, distanceM: Double, durationS: Double, divergeM: Double = -1.0) =
        RouteOption(
            routeId = id,
            route = RouteFixtures.straight(),
            engineDurationS = durationS,
            distanceM = distanceM,
            deltaDistanceM = if (id == "rec") 0.0 else distanceM - 3000,
            deltaDurationS = if (id == "rec") 0.0 else durationS - 300,
            decisionDivergenceM = divergeM,
            firstDivergenceM = divergeM,
        )

    private fun choices(
        generation: Long = 4,
        target: RouteTarget = RouteTarget.PICKUP,
        alts: List<RouteOption> = listOf(option("alt-1", 3400.0, 360.0, 500.0)),
        selected: String? = null,
    ) = RouteChoices(
        setId = "set-1",
        generation = generation,
        target = target,
        originLat = originLat,
        originLng = originLng,
        recommended = option("rec", 3000.0, 300.0),
        alternatives = alts,
        selectedRouteId = selected,
    )

    // ══════════════════════════════════════════════════════════════
    // **النموذج**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `المختارُ افتراضاً هو الموصى به`() {
        val c = choices()
        assertEquals("rec", c.selected.routeId)
        assertTrue(c.recommended.isRecommended)
        assertFalse(c.alternatives[0].isRecommended)
    }

    @Test
    fun `ولا خيارَ حين لا بدائل`() {
        // **البند ١ من قرار المالك**: «لا يجب أن تظهر UI بدائل إذا لم
        // توجد بدائل مفيدة».
        val c = choices(alts = emptyList())
        assertFalse(c.hasChoice)
        assertEquals(1, c.all.size)
        assertEquals("rec", c.selected.routeId)
    }

    @Test
    fun `المعرِّفُ لا الفهرس`() {
        // **البند ١٤** — `alternatives[0]` قد يصير `[1]`.
        val c = choices(
            alts = listOf(
                option("alt-a", 3400.0, 360.0),
                option("alt-b", 3200.0, 340.0),
            ),
        )
        assertEquals("alt-b", c.option("alt-b")?.routeId)
        assertEquals(null, c.option("لا-وجودَ-له"))
    }

    // ══════════════════════════════════════════════════════════════
    // **الاختيارُ وسباقاته**
    // ══════════════════════════════════════════════════════════════

    private fun evaluate(
        c: RouteChoices?,
        setId: String = "set-1",
        routeId: String = "alt-1",
        generation: Long = 4,
        target: RouteTarget = RouteTarget.PICKUP,
        lat: Double = originLat,
        lng: Double = originLng,
        progressM: Double = 0.0,
        ageMs: Long = 0,
    ) = SelectRoute.evaluate(
        c, setId, routeId, generation, target, lat, lng, progressM, ageMs,
    )

    @Test
    fun `اختيارٌ صالحٌ يُقبل`() {
        val v = evaluate(choices())
        assertTrue("$v", v is SelectRoute.Verdict.Accept)
        assertEquals("alt-1", (v as SelectRoute.Verdict.Accept).option.routeId)
    }

    @Test
    fun `اختيارٌ من جيلٍ مضى يُرفض`() {
        // **السيناريو الذي سمّاه المالك بعينه.**
        val displayed = choices(generation = 4)
        val v = evaluate(displayed, generation = 5)
        assertTrue("$v", v is SelectRoute.Verdict.Reject)
        assertEquals(SelectRoute.Why.STALE, (v as SelectRoute.Verdict.Reject).why)
    }

    @Test
    fun `اختيارٌ إلى وجهةٍ تبدّلت يُرفض`() {
        // **البند ٢١** — بدائلُ إلى الاستلام بعد `picked_up`.
        val displayed = choices(target = RouteTarget.PICKUP)
        val v = evaluate(displayed, target = RouteTarget.DROPOFF)
        assertTrue("$v", v is SelectRoute.Verdict.Reject)
        assertEquals(SelectRoute.Why.STALE, (v as SelectRoute.Verdict.Reject).why)
    }

    @Test
    fun `اختيارٌ من مجموعةٍ أخرى يُرفض`() {
        val v = evaluate(choices(), setId = "set-2")
        assertEquals(
            SelectRoute.Why.UNKNOWN_SET,
            (v as SelectRoute.Verdict.Reject).why,
        )
    }

    @Test
    fun `اختيارُ معرِّفٍ لا وجودَ له يُرفض`() {
        val v = evaluate(choices(), routeId = "شبح")
        assertEquals(
            SelectRoute.Why.UNKNOWN_ROUTE,
            (v as SelectRoute.Verdict.Reject).why,
        )
    }

    @Test
    fun `ولا مجموعةَ يعني رفضاً لا انهياراً`() {
        val v = evaluate(null)
        assertEquals(
            SelectRoute.Why.UNKNOWN_SET,
            (v as SelectRoute.Verdict.Reject).why,
        )
    }

    @Test
    fun `إعادةُ اختيار المختار تُرفض`() {
        // **فلا جيلٌ جديدٌ بلا سبب** — وإعادةُ التركيب تُصفّر التقدّم.
        val v = evaluate(choices(selected = "alt-1"))
        assertEquals(
            SelectRoute.Why.ALREADY_SELECTED,
            (v as SelectRoute.Verdict.Reject).why,
        )
    }

    // ══════════════════════════════════════════════════════════════
    // **التقادم — البند ٢٤**
    // ══════════════════════════════════════════════════════════════

    private fun expiry(
        c: RouteChoices = choices(),
        generation: Long = 4,
        target: RouteTarget = RouteTarget.PICKUP,
        lat: Double = originLat,
        lng: Double = originLng,
        progressM: Double = 0.0,
        ageMs: Long = 0,
    ) = RouteChoiceExpiry.check(c, generation, target, lat, lng, progressM, ageMs)

    @Test
    fun `الطازجُ طازج`() {
        assertEquals(RouteChoiceExpiry.Reason.FRESH, expiry())
    }

    @Test
    fun `تبدّلُ الجيل يُبطل`() {
        assertEquals(
            RouteChoiceExpiry.Reason.GENERATION_CHANGED,
            expiry(generation = 5),
        )
    }

    @Test
    fun `تبدّلُ الوجهة يُبطل`() {
        assertEquals(
            RouteChoiceExpiry.Reason.TARGET_CHANGED,
            expiry(target = RouteTarget.DROPOFF),
        )
    }

    @Test
    fun `حركةُ الأصل ثلاثَ مئةٍ تُبطل`() {
        // **٣٠٠م شمالاً ≈ 0.0027°.**
        assertEquals(
            RouteChoiceExpiry.Reason.ORIGIN_MOVED,
            expiry(lat = originLat + 0.0028),
        )
        assertEquals(
            "ومئتان لا تُبطل",
            RouteChoiceExpiry.Reason.FRESH,
            expiry(lat = originLat + 0.0018),
        )
    }

    @Test
    fun `العمرُ يُبطل`() {
        assertEquals(
            RouteChoiceExpiry.Reason.TOO_OLD,
            expiry(ageMs = 11 * 60 * 1000),
        )
    }

    @Test
    fun `تجاوزُ موضع القرار يُبطل قبل الثلاثمئة`() {
        /**
         * **البند ١٢** — «driver progress < first meaningful
         * divergence → SelectRoute accepted. ثم: driver passes first
         * meaningful divergence → SelectRoute rejected STALE. حتى لو:
         * originMoved <300m · age <10m · generation unchanged ·
         * target unchanged».
         *
         * **وقِيس في حلبَ ١١٩م** — فبديلٌ يتفرّع عندها **يبطل بعدها
         * لا بعد ثلاثِ مئة.**
         */
        val early = choices(alts = listOf(option("alt-1", 3400.0, 360.0, divergeM = 125.0)))

        // ── قبل القرار: يُقبل ─────────────────────────────────────
        assertEquals(
            RouteChoiceExpiry.Reason.FRESH,
            expiry(early, progressM = 100.0),
        )
        val before = evaluate(early, progressM = 100.0)
        assertTrue("قبل القرار يُقبل: $before", before is SelectRoute.Verdict.Accept)

        // ── وبعده: يُرفض ──────────────────────────────────────────
        //
        // **والحرّاسُ الأربعةُ كلُّها سليمة**: الجيلُ والوجهةُ لم
        // يتبدّلا، والأصلُ لم يتحرّك، والعمرُ صفر.
        assertEquals(
            RouteChoiceExpiry.Reason.PAST_DIVERGENCE,
            expiry(early, progressM = 200.0),
        )
        val after = evaluate(early, progressM = 200.0)
        assertTrue("بعد القرار يُرفض: $after", after is SelectRoute.Verdict.Reject)
        assertEquals(
            SelectRoute.Why.STALE,
            (after as SelectRoute.Verdict.Reject).why,
        )

        // **والأصلُ لم يتحرّك مئةً بعد** — فلولا هذا الشرطُ لبقي
        // البديلُ معروضاً وقد فات.
        assertEquals(
            RouteChoiceExpiry.Reason.FRESH,
            expiry(early, progressM = 100.0, lat = originLat + 0.0009),
        )
    }

    @Test
    fun `كلُّ بديلٍ يموت بقراره لا بقرار غيره`() {
        /**
         * **البند ٧** — «إذا السائق تجاوز الفرع الأول على
         * Recommended: الـAlternative الأصلية لا يجب أن تبقى
         * selectable حتى 900m».
         *
         * **بديلٌ يتفرّع عند ١٢٥م وآخرُ عند ٨٠٠**: من جاوز ١٢٥
         * **فقد الأوّلَ وحدَه.**
         */
        val two = choices(
            alts = listOf(
                option("alt-early", 3400.0, 360.0, divergeM = 125.0),
                option("alt-late", 3300.0, 350.0, divergeM = 800.0),
            ),
        )

        // ── عند ٣٠٠م: الأوّلُ مات والثاني حيّ ─────────────────────
        assertEquals(
            "والمجموعةُ حيّةٌ ما دام فيها بديل",
            RouteChoiceExpiry.Reason.FRESH,
            expiry(two, progressM = 300.0),
        )
        val dead = evaluate(two, routeId = "alt-early", progressM = 300.0)
        assertTrue("المبكّرُ يُرفض: $dead", dead is SelectRoute.Verdict.Reject)
        assertEquals(SelectRoute.Why.STALE, (dead as SelectRoute.Verdict.Reject).why)

        val alive = evaluate(two, routeId = "alt-late", progressM = 300.0)
        assertTrue("والمتأخّرُ يُقبل: $alive", alive is SelectRoute.Verdict.Accept)

        // ── وعند ٩٠٠م: كلاهما مات ─────────────────────────────────
        assertEquals(
            RouteChoiceExpiry.Reason.PAST_DIVERGENCE,
            expiry(two, progressM = 900.0),
        )
    }

    @Test
    fun `وبديلٌ لا يتفرّع افتراقاً ذا معنى لا يُبطله تقدّم`() {
        // **موضعٌ سالبٌ يعني: لا قرارَ فيه** — فلا يُقاس عليه.
        val c = choices(alts = listOf(option("alt-1", 3400.0, 360.0, divergeM = -1.0)))
        assertEquals(RouteChoiceExpiry.Reason.FRESH, expiry(c, progressM = 5000.0))
        assertTrue(evaluate(c, progressM = 5000.0) is SelectRoute.Verdict.Accept)
    }

    @Test
    fun `وتقدّمٌ مجهولٌ لا يُبطل`() {
        val c = choices(alts = listOf(option("alt-1", 3400.0, 360.0, divergeM = 125.0)))
        assertEquals(RouteChoiceExpiry.Reason.FRESH, expiry(c, progressM = -1.0))
        assertTrue(evaluate(c, progressM = -1.0) is SelectRoute.Verdict.Accept)
    }

    // ══════════════════════════════════════════════════════════════
    // **الاختيارُ يبني الملاحة — ولا يُعاد بناءُ الأنظمة** (البند ٢٣)
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `تسليمُ المختار يزيد الجيل ويُصفّر الكواشف`() {
        /**
         * **أمرُ المالك**: «والآليات الحالية تتولى: generation++ ·
         * new RouteProgress · OffRoute reset · WrongWay reset · Voice
         * generation reset. لا تعيد كتابة هذه الأنظمة».
         *
         * **وهذا الاختبارُ يُثبت أنّها تتولّاها فعلاً** — فلا شيءَ
         * جديدٌ كُتب.
         */
        val engine = NavEngine(voice = VoicePlanner())
        val first = RouteFixtures.straight()
        engine.setRoute(first)
        for (f in RouteFixtures.driveAlong(first).take(10)) engine.onFix(f)

        val genBefore = engine.generation
        val progressBefore = engine.state?.progress?.progressM ?: 0.0
        assertTrue("تقدّمٌ وقع", progressBefore > 0)

        // ── يُختار بديل ───────────────────────────────────────────
        val alternative = RouteFixtures.singleRight()
        engine.setRoute(alternative)

        assertEquals("الجيلُ زاد", genBefore + 1, engine.generation)
        assertEquals("والمسارُ تبدّل", alternative, engine.currentRoute)

        /**
         * **والقيادةُ على الجديد تُثبت أنّ الكواشفَ صُفّرت.**
         *
         * **ولا يُفحص التقدّمُ في القراءة الأولى**: المصفاةُ ترفض
         * قفزةً إلى الوراء — والسائقُ كان على تسعين متراً من القديم
         * **فعادت القراءةُ إلى أوّل الجديد.** **والمرفوضةُ لا تُغذّي
         * التقدّم** (المرحلة ٣أ)، **فيُقرأ ما بعدها.**
         *
         * **ولو لم تُصفَّر الكواشفُ لأعلن الجديدُ خروجاً فوراً** —
         * فالهندسةُ تبدّلت كلُّها.
         */
        var last: NavState? = null
        for (f in RouteFixtures.driveAlong(alternative).take(12)) last = engine.onFix(f)

        assertTrue("المسارُ محمَّل", last!!.hasRoute)
        assertEquals(
            "ولا خروجَ على المسار الجديد",
            OffRouteDetector.State.ON_ROUTE,
            last.offRoute.state,
        )
        assertEquals(
            "والاتّجاهُ المعاكسُ صُفّر",
            WrongWayDetector.State.CORRECT_DIRECTION,
            last.wrongWay.state,
        )
        assertEquals("والجيلُ لم يزد ثانيةً", genBefore + 1, engine.generation)
    }

    @Test
    fun `والمعروضُ لا يُبدّل المُلاحَ عليه`() {
        // **البند ١٦ من التحليل** — «تحميل RouteSet جديد لا يعني
        // تلقائيًا تغيير NavRoute الحالية».
        val engine = NavEngine()
        val current = RouteFixtures.straight()
        engine.setRoute(current)
        val gen = engine.generation

        // **تُبنى خياراتٌ ولا تُسلَّم** — والمحرّكُ لم يُمسّ.
        val c = choices()
        assertTrue(c.hasChoice)
        assertEquals(gen, engine.generation)
        assertEquals(current, engine.currentRoute)
    }
}
