package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **آلةُ حالِ الاختيار — البنود ٤١ إلى ٤٦**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ واجهة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 *
 * **وهذه اختباراتٌ خالصة** — لا Compose ولا جهاز. **والمنطقُ كلُّه
 * هنا**، والواجهةُ تعرضه.
 */
class RouteChoiceMachineTest {

    private val originLat = 35.9500
    private val originLng = 39.0050

    private fun option(id: String, divergeM: Double = -1.0, dm: Double = 0.0, ds: Double = 0.0) =
        RouteOption(
            routeId = id,
            route = RouteFixtures.straight(),
            engineDurationS = 900.0 + ds,
            distanceM = 11000.0 + dm,
            deltaDistanceM = dm,
            deltaDurationS = ds,
            decisionDivergenceM = divergeM,
        )

    private fun choices(alts: List<RouteOption>) = RouteChoices(
        setId = "set-1",
        generation = 4,
        target = RouteTarget.PICKUP,
        originLat = originLat,
        originLng = originLng,
        recommended = option("rec"),
        alternatives = alts,
    )

    private fun ctx(
        generation: Long = 4,
        target: RouteTarget = RouteTarget.PICKUP,
        lat: Double = originLat,
        lng: Double = originLng,
        progressM: Double = 0.0,
        ageMs: Long = 0,
        healthy: Boolean = true,
    ) = RouteChoiceMachine.Context(generation, target, lat, lng, progressM, ageMs, healthy)

    // ══════════════════════════════════════════════════════════════
    // **البند ٤١ — الحال**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `لا بدائلَ فلا لوحة`() {
        // **البند ٥**: «لا نريد أخذ مساحة من شاشة السائق بلا فائدة».
        assertEquals(RouteChoiceUi.Hidden, RouteChoiceMachine.present(choices(emptyList()), ctx()))
        assertEquals(RouteChoiceUi.Hidden, RouteChoiceMachine.present(null, ctx()))
    }

    @Test
    fun `بديلٌ واحدٌ فخياران`() {
        val ui = RouteChoiceMachine.present(choices(listOf(option("a"))), ctx())
        assertTrue("$ui", ui is RouteChoiceUi.Available)
        assertEquals(2, (ui as RouteChoiceUi.Available).choices.all.size)
    }

    @Test
    fun `بديلان فثلاثةُ خيارات`() {
        val ui = RouteChoiceMachine.present(choices(listOf(option("a"), option("b"))), ctx())
        assertEquals(3, (ui as RouteChoiceUi.Available).choices.all.size)
    }

    @Test
    fun `معاينةُ بديلٍ لا تمسّ الملاحة`() {
        // **البند ١٣** — «Preview تغيير بصري فقط».
        val engine = NavEngine(voice = VoicePlanner())
        val route = RouteFixtures.straight()
        engine.setRoute(route)
        for (f in RouteFixtures.driveAlong(route).take(10)) engine.onFix(f)
        val genBefore = engine.generation

        val ui = RouteChoiceMachine.present(choices(listOf(option("a"))), ctx())
        val previewing = RouteChoiceMachine.onSelect(ui, "a", ctx())

        assertTrue("$previewing", previewing is RouteChoiceUi.Previewing)
        assertEquals("a", previewing.previewRouteId)
        assertEquals("الجيلُ لم يتبدّل", genBefore, engine.generation)
        assertEquals("والمسارُ لم يتبدّل", route, engine.currentRoute)
    }

    @Test
    fun `إلغاءُ المعاينة لا يمسّ الملاحة`() {
        // **البند ١٦** — «ولا حاجة لاستدعاء setRoute لأن Recommended
        // هي Route الفعالة أصلًا».
        val ui = RouteChoiceMachine.present(choices(listOf(option("a"))), ctx())
        val previewing = RouteChoiceMachine.onSelect(ui, "a", ctx())
        val cancelled = RouteChoiceMachine.cancelPreview(previewing)

        assertTrue("$cancelled", cancelled is RouteChoiceUi.Available)
        assertNull(cancelled.previewRouteId)
    }

    @Test
    fun `ضغطُ الموصى به أثناء المعاينة يُلغيها`() {
        // **البند ٢٧** — ولا `setRoute`.
        val c = choices(listOf(option("a")))
        val previewing = RouteChoiceMachine.onSelect(
            RouteChoiceMachine.present(c, ctx()), "a", ctx(),
        )
        val back = RouteChoiceMachine.onSelect(previewing, "rec", ctx())
        assertTrue("$back", back is RouteChoiceUi.Available)
        assertNull(back.previewRouteId)
    }

    @Test
    fun `اعتمادٌ صالحٌ يُقبل مرّةً`() {
        val c = choices(listOf(option("a")))
        val previewing = RouteChoiceMachine.onSelect(
            RouteChoiceMachine.present(c, ctx()), "a", ctx(),
        )
        val out = RouteChoiceMachine.commit(previewing, ctx())
        assertTrue("$out", out is RouteChoiceMachine.Commit.Accept)
        assertEquals("a", (out as RouteChoiceMachine.Commit.Accept).option.routeId)
        // **وبعد التسليم تختفي المجموعة** — البند ١٧.
        assertEquals(RouteChoiceUi.Hidden, RouteChoiceMachine.afterCommitted())
    }

    @Test
    fun `اعتمادٌ من جيلٍ مضى يُرفض ولا يُمسّ المسار`() {
        val c = choices(listOf(option("a")))
        val previewing = RouteChoiceMachine.onSelect(
            RouteChoiceMachine.present(c, ctx()), "a", ctx(),
        )
        val out = RouteChoiceMachine.commit(previewing, ctx(generation = 5))
        assertTrue("$out", out is RouteChoiceMachine.Commit.Reject)
        assertEquals(
            SelectRoute.Why.STALE,
            (out as RouteChoiceMachine.Commit.Reject).why,
        )
        assertTrue("والحالُ يصير Stale", out.next is RouteChoiceUi.Stale)
    }

    @Test
    fun `ولا اعتمادَ بلا معاينة`() {
        // **البند ١١** — لا اختيارَ بلمسةٍ واحدة.
        val available = RouteChoiceMachine.present(choices(listOf(option("a"))), ctx())
        val out = RouteChoiceMachine.commit(available, ctx())
        assertTrue("$out", out is RouteChoiceMachine.Commit.Reject)
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ٤٢ — التقادمُ الفرديّ**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `البدائلُ تنتهي فرديّاً`() {
        val c = choices(
            listOf(
                option("A", divergeM = 125.0),
                option("B", divergeM = 800.0),
            ),
        )

        // ── عند ١٠٠م: كلاهما ──────────────────────────────────────
        val at100 = RouteChoiceMachine.present(c, ctx(progressM = 100.0))
        assertEquals(2, (at100 as RouteChoiceUi.Available).choices.alternatives.size)

        // ── وعند ٢٠٠م: «أ» تزول و«ب» تبقى ────────────────────────
        val at200 = RouteChoiceMachine.present(c, ctx(progressM = 200.0))
        val alive = (at200 as RouteChoiceUi.Available).choices.alternatives
        assertEquals(1, alive.size)
        assertEquals("B", alive[0].routeId)

        // ── وعند ٩٠٠م: اللوحةُ تختفي ─────────────────────────────
        val at900 = RouteChoiceMachine.present(c, ctx(progressM = 900.0))
        assertEquals(RouteChoiceUi.Hidden, at900)
    }

    @Test
    fun `ومعاينةُ بديلٍ فات فرعُه تُرفض`() {
        val c = choices(listOf(option("A", divergeM = 125.0), option("B", divergeM = 800.0)))
        val ui = RouteChoiceMachine.present(c, ctx(progressM = 300.0))
        val out = RouteChoiceMachine.onSelect(ui, "A", ctx(progressM = 300.0))
        assertTrue("$out", out is RouteChoiceUi.Stale)
    }

    @Test
    fun `والاعتمادُ بعد تجاوز الفرع أثناء المعاينة يُرفض`() {
        // **البند ١٩** — السائقُ يفتح المعاينةَ ثمّ يتحرّك ثمّ يضغط.
        val c = choices(listOf(option("A", divergeM = 125.0)))
        val previewing = RouteChoiceMachine.onSelect(
            RouteChoiceMachine.present(c, ctx(progressM = 50.0)), "A", ctx(progressM = 50.0),
        )
        assertTrue(previewing is RouteChoiceUi.Previewing)

        val out = RouteChoiceMachine.commit(previewing, ctx(progressM = 400.0))
        assertTrue("$out", out is RouteChoiceMachine.Commit.Reject)
        // **والمنتهيةُ تُزال** — البند ١٩.
        val next = (out as RouteChoiceMachine.Commit.Reject).next
        assertTrue(next is RouteChoiceUi.Stale)
        assertTrue(
            "البديلُ المنتهي أُزيل",
            next.choicesOrNull?.alternatives?.none { it.routeId == "A" } == true,
        )
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ٤٣ — الوجهةُ وإعادةُ الحساب**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `تبدّلُ الوجهة يمسح`() {
        val c = choices(listOf(option("a")))
        val ui = RouteChoiceMachine.present(c, ctx(target = RouteTarget.DROPOFF))
        assertEquals(RouteChoiceUi.Hidden, ui)
    }

    @Test
    fun `ونجاحُ إعادة الحساب يمسح`() {
        val c = choices(listOf(option("a")))
        assertEquals(RouteChoiceUi.Hidden, RouteChoiceMachine.present(c, ctx(generation = 5)))
    }

    @Test
    fun `وأثناء إعادة الحساب لا اختيار`() {
        // **البند ٢٣** — «route-choice interaction enabled only while
        // the current navigation situation is healthy».
        val c = choices(listOf(option("a")))
        assertEquals(RouteChoiceUi.Hidden, RouteChoiceMachine.present(c, ctx(healthy = false)))

        val previewing = RouteChoiceUi.Previewing(c, "a")
        val out = RouteChoiceMachine.commit(previewing, ctx(healthy = false))
        assertTrue("$out", out is RouteChoiceMachine.Commit.Reject)
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ٤٤ — دونَ اتّصال**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `المعاينةُ والاعتمادُ يعملان بلا شبكة`() {
        /**
         * **أمرُ المالك**: «لا Network call عند SelectRoute. هذا
         * اختبار إلزامي».
         *
         * **والدليلُ بنيويّ**: `RouteChoiceMachine` و`SelectRoute`
         * **دالّاتٌ خالصةٌ لا تعرف شبكةً** — لا `HttpSource` ولا
         * `DriverApi` في توقيعٍ ولا في تبعيّة. **والمسارُ محمَّلٌ في
         * `RouteOption.route` كاملاً** بهندسته ومناوراته (البند ٢١
         * من المرحلة ٧).
         */
        val c = choices(listOf(option("a")))
        val previewing = RouteChoiceMachine.onSelect(
            RouteChoiceMachine.present(c, ctx()), "a", ctx(),
        )
        val out = RouteChoiceMachine.commit(previewing, ctx())
        assertTrue("$out", out is RouteChoiceMachine.Commit.Accept)

        // **والمسارُ جاهزٌ للملاحة فوراً** — لا نداءَ ثانٍ.
        val chosen = (out as RouteChoiceMachine.Commit.Accept).option.route
        assertTrue("هندسةٌ كاملة", chosen.geometry.size >= 2)
        assertTrue("ومناوراتٌ", chosen.maneuvers.isNotEmpty())
        assertTrue("وصالحٌ للملاحة", chosen.usable)
    }

    @Test
    fun `والتسليمُ يزيد الجيلَ ويُصفّر الكواشف`() {
        val engine = NavEngine(voice = VoicePlanner())
        val route = RouteFixtures.straight()
        engine.setRoute(route)
        for (f in RouteFixtures.driveAlong(route).take(10)) engine.onFix(f)
        val genBefore = engine.generation

        val alternative = RouteFixtures.singleRight()
        engine.setRoute(alternative)

        assertEquals(genBefore + 1, engine.generation)
        assertEquals(alternative, engine.currentRoute)
    }

    // ══════════════════════════════════════════════════════════════
    // **وحالُ الاعتماد يعطّل الأزرار**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `أثناء الاعتماد لا تُقبل ضغطة`() {
        val c = choices(listOf(option("a"), option("b")))
        val committing = RouteChoiceUi.Committing(c, "a")
        assertTrue(committing.busy)
        assertEquals(committing, RouteChoiceMachine.onSelect(committing, "b", ctx()))
        assertEquals(committing, RouteChoiceMachine.cancelPreview(committing))
    }

    @Test
    fun `والمسحُ الصريحُ يُخفي`() {
        assertEquals(RouteChoiceUi.Hidden, RouteChoiceMachine.clear())
    }
}
