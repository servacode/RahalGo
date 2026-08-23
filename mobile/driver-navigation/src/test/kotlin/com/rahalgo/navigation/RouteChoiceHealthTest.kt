package com.rahalgo.navigation

import kotlinx.coroutines.CancellationException
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **صحّةُ الحال والإلغاء — البنود ٥ و١٠ إلى ١٢**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ صحّة واجهة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 */
class RouteChoiceHealthTest {

    private fun state(
        situation: NavSituation,
        reroute: RerouteStatus = RerouteStatus.NONE,
    ): NavState {
        // **تُبنى الحالُ من كاشفَين حقيقيَّين** — لا من حقلٍ مزيَّف،
        // **فـ`situation` مشتقّةٌ لا مُسنَدة.**
        val off = when (situation) {
            NavSituation.OFF_ROUTE -> OffRouteDetector.State.OFF_ROUTE
            NavSituation.SUSPECTED_OFF_ROUTE -> OffRouteDetector.State.SUSPECTED_OFF_ROUTE
            else -> OffRouteDetector.State.ON_ROUTE
        }
        val wrong = when (situation) {
            NavSituation.WRONG_WAY -> WrongWayDetector.State.WRONG_WAY
            NavSituation.SUSPECTED_WRONG_WAY -> WrongWayDetector.State.SUSPECTED_WRONG_WAY
            else -> WrongWayDetector.State.CORRECT_DIRECTION
        }
        return NavState(
            grade = FixGrade.ACCEPTED,
            reason = RejectReason.NONE,
            lat = 35.95,
            lng = 39.005,
            bearingDeg = 0f,
            animationMs = 0L,
            hasRoute = true,
            progress = null,
            offRoute = OffRouteDetector.Verdict(
                off, 0.0, 36.0, 0.0, OffRouteDetector.Skip.NONE, false, false, false, false,
            ),
            wrongWay = WrongWayDetector.Verdict(
                wrong, null, 0L, 0.0, false, WrongWayDetector.Skip.NONE, 0L,
            ),
            reroute = reroute,
        )
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ١٠ — والصحّةُ من الحال الموحَّدة**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `على المسارِ صحيح`() {
        assertTrue(RouteChoiceHealth.of(state(NavSituation.ON_ROUTE)))
    }

    @Test
    fun `والاتّجاهُ المعاكسُ يمنع ولو كان داخلَ الممرّ`() {
        /**
         * **العيبُ الذي أُصلح**: كان الفحصُ `offRoute == ON_ROUTE`
         * وحدَه.
         *
         * **وكاشفُ الاتّجاه يقيس نسبةً إلى المسار** — **فالسائقُ قد
         * يكون داخلَ الممرّ تماماً وهو يسير عكسَه**، و`offRoute`
         * تقول `ON_ROUTE`.
         */
        val s = state(NavSituation.WRONG_WAY)
        assertEquals(
            "الرفيدةُ يجب أن تُوقع الفحصَ القديم",
            OffRouteDetector.State.ON_ROUTE,
            s.offRoute.state,
        )
        assertEquals(NavSituation.WRONG_WAY, s.situation)
        assertFalse("والفحصُ الجديد يمنع", RouteChoiceHealth.of(s))
        assertEquals(NavSituation.WRONG_WAY, RouteChoiceHealth.blockedBy(s))
    }

    @Test
    fun `والشكوكُ تمنع أيضاً`() {
        // **أمرُ المالك**: «SUSPECTED_OFF_ROUTE · SUSPECTED_WRONG_WAY
        // → لا Preview commit».
        assertFalse(RouteChoiceHealth.of(state(NavSituation.SUSPECTED_WRONG_WAY)))
        assertFalse(RouteChoiceHealth.of(state(NavSituation.SUSPECTED_OFF_ROUTE)))
    }

    @Test
    fun `والخروجُ وإعادةُ الحساب يمنعان`() {
        assertFalse(RouteChoiceHealth.of(state(NavSituation.OFF_ROUTE)))
        assertFalse(
            RouteChoiceHealth.of(state(NavSituation.ON_ROUTE, RerouteStatus.REROUTING)),
        )
    }

    @Test
    fun `ولا حالَ يعني لم تبدأ الملاحة`() {
        assertTrue(RouteChoiceHealth.of(null))
        assertNull(RouteChoiceHealth.blockedBy(null))
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ١٢ — الاتّجاهُ المعاكسُ والعودة**
    // ══════════════════════════════════════════════════════════════

    private fun option(id: String, divergeM: Double = -1.0) = RouteOption(
        routeId = id,
        route = RouteFixtures.straight(),
        engineDurationS = 900.0,
        distanceM = 11000.0,
        deltaDistanceM = -1500.0,
        deltaDurationS = 120.0,
        decisionDivergenceM = divergeM,
    )

    private fun choices(alts: List<RouteOption>) = RouteChoices(
        setId = "s",
        generation = 4,
        target = RouteTarget.PICKUP,
        originLat = 35.95,
        originLng = 39.005,
        recommended = option("rec"),
        alternatives = alts,
    )

    private fun ctx(progressM: Double = 0.0, healthy: Boolean = true) =
        RouteChoiceMachine.Context(4, RouteTarget.PICKUP, 35.95, 39.005, progressM, 0L, healthy)

    @Test
    fun `الاتّجاهُ المعاكسُ يُخفي اللوحةَ ويمنع الاعتماد`() {
        val c = choices(listOf(option("a", divergeM = 800.0)))
        val wrong = state(NavSituation.WRONG_WAY)
        val healthy = RouteChoiceHealth.of(wrong)

        // ── اللوحةُ تختفي ─────────────────────────────────────────
        assertEquals(
            RouteChoiceUi.Hidden,
            RouteChoiceMachine.present(c, ctx(healthy = healthy)),
        )

        // ── والاعتمادُ يُرفض ولو كانت معاينةٌ قائمة ───────────────
        val previewing = RouteChoiceUi.Previewing(c, "a")
        val out = RouteChoiceMachine.commit(previewing, ctx(healthy = healthy))
        assertTrue("$out", out is RouteChoiceMachine.Commit.Reject)
    }

    @Test
    fun `والملاحةُ لا تُمسّ حين يسوء الحال`() {
        /**
         * **البند ١١**: «لا تغير selected NavRoute · لا تغير
         * RouteProgress · لا setRoute».
         *
         * **والدليلُ بنيويّ**: `present` و`commit` **لا تلمسان
         * `NavEngine`** — تردّان حالَ عرضٍ وقراراً.
         */
        val engine = NavEngine(voice = VoicePlanner())
        val route = RouteFixtures.straight()
        engine.setRoute(route)
        for (f in RouteFixtures.driveAlong(route).take(10)) engine.onFix(f)
        val genBefore = engine.generation

        val c = choices(listOf(option("a")))
        RouteChoiceMachine.present(c, ctx(healthy = false))
        RouteChoiceMachine.commit(RouteChoiceUi.Previewing(c, "a"), ctx(healthy = false))

        assertEquals(genBefore, engine.generation)
        assertEquals(route, engine.currentRoute)
    }

    @Test
    fun `وعودةُ الصحّةِ تُعيد ما بقي طازجاً`() {
        val c = choices(listOf(option("a", divergeM = 800.0)))
        // **الحالُ ساء ثمّ عاد** — والتقدّمُ ما زال قبل الفرع.
        assertEquals(RouteChoiceUi.Hidden, RouteChoiceMachine.present(c, ctx(healthy = false)))
        val back = RouteChoiceMachine.present(c, ctx(progressM = 200.0, healthy = true))
        assertTrue("$back", back is RouteChoiceUi.Available)
    }

    @Test
    fun `ولا تعود بديلةٌ فات فرعُها`() {
        /**
         * **أمرُ المالك**: «returns ON_ROUTE لكن driver passed
         * decision divergence → لا تعود Alternative القديمة».
         *
         * **فالصحّةُ تُعيد الفحصَ لا الحال.**
         */
        val c = choices(listOf(option("a", divergeM = 125.0)))
        assertEquals(RouteChoiceUi.Hidden, RouteChoiceMachine.present(c, ctx(healthy = false)))
        assertEquals(
            "عادت الصحّةُ والفرعُ فات",
            RouteChoiceUi.Hidden,
            RouteChoiceMachine.present(c, ctx(progressM = 400.0, healthy = true)),
        )
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ٥ — والإلغاءُ لا يُبتلع**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `runCatching تبتلع الإلغاءَ — ولذلك لا تُستعمل`() {
        /**
         * **أمرُ المالك نصّاً**: «Kotlin runCatching يلتقط Throwable،
         * بما فيه CancellationException. لا أريد ابتلاع
         * cancellation».
         *
         * **وهذا الاختبارُ يُثبت الخطر** — فمن أعادها يوماً وجد نصّاً
         * يقول لماذا لا تصلح.
         */
        val swallowed = runCatching { throw CancellationException("أُلغيت") }.getOrNull()
        assertNull("`runCatching` ابتلعته", swallowed)

        // ── والنمطُ المعتمد: يُمرَّر الإلغاءُ ويُبتلع الخطأ ────────
        fun guarded(block: () -> String): String? = try {
            block()
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            null
        }

        var propagated = false
        try {
            guarded { throw CancellationException("أُلغيت") }
        } catch (e: CancellationException) {
            propagated = true
        }
        assertTrue("الإلغاءُ يُمرَّر", propagated)
        assertNull("والخطأُ يُبتلع", guarded { throw IllegalStateException("شبكة") })
        assertEquals("ok", guarded { "ok" })
    }

    @Test
    fun `ونمطُ الجالب هو المعتمد`() {
        /**
         * **يُقرأ من المصدر** — فلا يعود `runCatching` يوماً إلى
         * جالب البدائل بلا أن يُلاحَظ.
         *
         * **والتعاليقُ تُطرح** — ففي شرحِ الدالة نفسِها ذكرٌ
         * لـ`runCatching` يقول لماذا لا تصلح.
         */
        val src = java.io.File(
            "../app-driver/src/main/kotlin/com/rahalgo/driver/orders/OrdersViewModel.kt",
        )
        assertTrue("لم يُوجَد المصدر: " + src.absolutePath, src.isFile)

        val body = src.readText()
            .substringAfter("fun loadAlternatives(")
            .substringBefore("var navGeneration")
        val code = body.lineSequence()
            .map { it.trim() }
            .filterNot { it.startsWith("*") || it.startsWith("/*") || it.startsWith("//") }
            .joinToString(" | ")

        assertTrue("لم يُقرأ جسمُ الجالب", code.contains("alternatives = true"))
        assertFalse("الجالبُ يستعمل runCatching فيبتلع الإلغاء", code.contains("runCatching"))
        assertTrue("ولا يمرّر الإلغاء", code.contains("catch (e: kotlinx.coroutines.CancellationException)"))
        assertTrue("ولا يعيد رميَه", code.contains("throw e"))
    }
}
