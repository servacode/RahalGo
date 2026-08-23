package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اختباراتُ المرحلة ٣ب — إعادةُ الحساب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * المعرّفات: `RRT-*`
 *
 * **والشبكةُ تُحاكى بدالّةٍ واحدة** — `FakeRouteSource` تمسك الجوابَ في
 * يدها وتطلقه متى شاء الاختبار. **وبها وحدَها تُختبر السباقات**:
 * جوابٌ يصل بعد أن عاد السائقُ، وجوابٌ من جيلٍ مضى.
 */
class RerouteTest {

    // ══════════════════════════════════════════════════════════════════
    // **بابُ شبكةٍ في اليد**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **بابٌ لا يردّ حتّى يُؤمَر.**
     *
     * **وهذا هو ما يجعل السباقَ قابلاً للاختبار** — نداءٌ يعود فوراً لا
     * يُنتج سباقاً أبداً.
     */
    class FakeRouteSource : RouteSource {
        var calls = 0
            private set
        val points = ArrayList<Pair<Double, Double>>()
        private var pending: ((RouteReply) -> Unit)? = null

        /** **الجوابُ المعدُّ** — يُطلق بـ`deliver`. */
        var reply: RouteReply = RouteReply.Failed(RerouteFailure.NETWORK)

        /** **أيردّ فوراً؟** — والافتراضُ لا، فيُمسَك الجواب. */
        var immediate = false

        override fun request(lat: Double, lng: Double, seq: Long, done: (RouteReply) -> Unit) {
            calls++
            points += lat to lng
            if (immediate) done(reply) else pending = done
        }

        /** **يُطلق الجوابَ المعدَّ** — أو المعطى. */
        fun deliver(what: RouteReply = reply) {
            val p = pending ?: return
            pending = null
            p(what)
        }

        val hasPending: Boolean get() = pending != null
    }

    private fun engine(
        route: NavRoute,
        source: FakeRouteSource,
        tuning: RerouteEngine.Tuning = RerouteEngine.Tuning(),
    ): NavEngine = NavEngine(source = source, rerouteTuning = tuning).also { it.setRoute(route) }

    /** **مسارٌ بديلٌ يبدأ من موضع الانحراف** — كما يردّه المحرّك. */
    private fun altRouteAt(fix: NavFix, meters: Double = 400.0): NavRoute {
        val pts = (0..10).map { GeoPoint(fix.lat + it * (meters / 10) * 0.000009, fix.lng) }
        return NavRoute.of(
            pts,
            listOf(
                NavManeuver(ManeuverKinds.DEPART, null, 0.0, 0, meters, meters / 10),
                NavManeuver(ManeuverKinds.ARRIVE, null, meters, 10, 0.0, 0.0),
            ),
        )
    }

    /** **يقود حتّى يُعلَن الخروج** — ويردّ ما وقع. */
    private fun driveUntilOffRoute(e: NavEngine, fixes: List<NavFix>): List<NavState> {
        val out = ArrayList<NavState>()
        for (f in fixes) {
            out += e.onFix(f)
            if (out.last().isOffRoute) break
        }
        return out
    }

    // ══════════════════════════════════════════════════════════════════
    // **١ · النجاح**
    // ══════════════════════════════════════════════════════════════════

    /** **RRT-001** — `offroute-reroute-success`. */
    @Test
    fun `RRT-001 خروجٌ مؤكَّدٌ يُطلق طلباً واحداً ويُركّب المسار`() {
        val (route, fixes, at) = OffRouteFixtures.realDeviation()
        val src = FakeRouteSource()
        val e = engine(route, src)
        val out = driveUntilOffRoute(e, fixes)
        val confirmedAt = out.size - 1

        assertTrue("لم يُعلَن خروج", out.last().isOffRoute)
        assertEquals("لم يُطلق طلبٌ واحد", 1, src.calls)
        assertEquals(RerouteStatus.REROUTING, out.last().reroute)

        // **والنقطةُ المرسلةُ هي القراءةُ المقبولةُ الخامّة** — لا
        // مُسقَطةٌ ولا مُنعَّمة.
        val start = e.lastAcceptedFix!!
        assertEquals(start.lat, src.points.last().first, 1e-9)
        assertEquals(start.lng, src.points.last().second, 1e-9)

        val before = e.currentRoute!!
        val alt = altRouteAt(start)
        src.deliver(RouteReply.Ok(alt))

        assertEquals(1, e.reroute!!.installs)
        assertNotEquals("الهندسةُ لم تتبدّل", RouteFingerprint.of(before), RouteFingerprint.of(e.currentRoute!!))
        println(
            "RRT-001 · انحرافٌ عند=$at · تأكيدٌ عند=$confirmedAt · " +
                "زمنُ التأكيد=${(fixes[confirmedAt].atMs - fixes[at].atMs) / 1000.0}ث · نداءات=${src.calls}",
        )
    }

    /** **RRT-002** — `new-route-progress-reset` + المناورةُ والزمن. */
    @Test
    fun `RRT-002 المسارُ الجديد يُعيد التقدّمَ والمناورةَ والزمن`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val src = FakeRouteSource()
        val e = engine(route, src)
        driveUntilOffRoute(e, fixes)
        val oldRemaining = e.state!!.remainingM
        val oldGeneration = e.generation

        val start = e.lastAcceptedFix!!
        src.deliver(RouteReply.Ok(altRouteAt(start, meters = 900.0)))

        assertTrue("الجيلُ لم يزد", e.generation > oldGeneration)
        assertEquals("دليلُ الخروج لوّث الجديد", OffRouteDetector.State.ON_ROUTE, e.detector.state)
        assertEquals(0.0, e.detector.score, 0.001)

        val st = e.state!!
        assertNotNull("التقدّمُ لم يُبذَر", st.progress)
        println(
            "RRT-002 · ما بقي قبل=${"%.0f".format(oldRemaining)} بعد=${"%.0f".format(st.remainingM)} · " +
                "زمنٌ=${"%.0f".format(st.remainingSec)} · مناورة=${st.currentManeuver?.kind}",
        )
        assertTrue("ما بقي لم يُقرأ من الجديد", st.remainingM > 800.0)
        assertTrue("الزمنُ لم يُقرأ من الجديد", st.remainingSec > 0.0)
        assertNotNull("المناورةُ لم تُقرأ من الجديد", st.currentManeuver)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٢ · طلبٌ واحدٌ لا غير**
    // ══════════════════════════════════════════════════════════════════

    /** **RRT-010** — `double-reroute-prevention`. */
    @Test
    fun `RRT-010 عشرون قراءةً خارجَ المسار ونداءٌ واحد`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val src = FakeRouteSource()
        val e = engine(route, src)
        // **ولا يُطلق الجوابُ أبداً** — فالطلبُ معلَّقٌ طولَ المشية.
        fixes.take(30).forEach { e.onFix(it) }
        println("RRT-010 · نداءات=${src.calls} · معلَّق=${src.hasPending}")
        assertEquals("أُطلق أكثرُ من نداء", 1, src.calls)
        assertEquals(RerouteEngine.Phase.REQUESTING, e.reroute!!.phase)
    }

    /** **RRT-011** — `suspect-no-reroute`: الشكُّ وحدَه لا يُطلق شيئاً. */
    @Test
    fun `RRT-011 الشكُّ وحدَه لا يُطلق نداء`() {
        val (route, fixes) = OffRouteFixtures.suspectThenReturn()
        val src = FakeRouteSource()
        val e = engine(route, src)
        val out = fixes.map { e.onFix(it) }
        assertTrue("لم يقع شكٌّ أصلاً", out.any { it.offRoute.state == OffRouteDetector.State.SUSPECTED_OFF_ROUTE })
        println("RRT-011 · نداءات=${src.calls}")
        assertEquals("الشكُّ أطلق نداء", 0, src.calls)
    }

    /** **RRT-012** — `reroute-while-stationary`: الواقفُ لا يُطلق شيئاً. */
    @Test
    fun `RRT-012 الواقفُ المنجرفُ لا يُطلق نداء`() {
        val (route, fixes) = OffRouteFixtures.stationaryDrift()
        val src = FakeRouteSource()
        val e = engine(route, src)
        fixes.forEach { e.onFix(it) }
        println("RRT-012 · نداءات=${src.calls}")
        assertEquals(0, src.calls)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٣ · الإخفاقُ والتهدئة**
    // ══════════════════════════════════════════════════════════════════

    /** **RRT-020** — `offroute-reroute-network-failure`: المسارُ القديمُ يبقى. */
    @Test
    fun `RRT-020 فشلُ الشبكة يُبقي المسارَ القديم`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val src = FakeRouteSource()
        val e = engine(route, src)
        driveUntilOffRoute(e, fixes)
        val before = RouteFingerprint.of(e.currentRoute!!)
        val maneuversBefore = e.currentRoute!!.maneuvers.size
        val remainingBefore = e.state!!.remainingM

        src.deliver(RouteReply.Failed(RerouteFailure.NETWORK))

        assertEquals("الهندسةُ ضاعت", before, RouteFingerprint.of(e.currentRoute!!))
        assertEquals("المناوراتُ ضاعت", maneuversBefore, e.currentRoute!!.maneuvers.size)
        assertEquals(RerouteEngine.Phase.COOLDOWN, e.reroute!!.phase)
        assertEquals(RerouteStatus.REROUTE_FAILED, e.reroute!!.status)
        assertEquals(RerouteFailure.NETWORK, e.reroute!!.lastFailure)
        println("RRT-020 · ما بقي=${"%.0f".format(remainingBefore)} · حال=${e.reroute!!.status}")
    }

    /** **RRT-021** — `reroute-cooldown`: لا عاصفةَ طلبات. */
    @Test
    fun `RRT-021 التهدئةُ تمنع عاصفةَ الطلبات`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val src = FakeRouteSource()
        src.immediate = true
        src.reply = RouteReply.Failed(RerouteFailure.NETWORK)
        val e = engine(route, src)
        // **خمسون قراءةً وكلُّها خارجَ المسار** — والفشلُ فوريّ.
        fixes.forEach { e.onFix(it) }
        println(
            "RRT-021 · نداءات=${src.calls} من ${fixes.size} قراءة · " +
                "تهدئةٌ باقية=${e.reroute!!.cooldownLeftMs(fixes.last().atMs)}ملّي",
        )
        assertTrue("عاصفةُ طلبات: ${src.calls}", src.calls <= 5)
        assertTrue("لم يُطلق شيءٌ أصلاً", src.calls >= 1)
    }

    /** **RRT-022** — `network-loss-return`: تعود الشبكةُ فينجح. */
    @Test
    fun `RRT-022 عودةُ الشبكة تُنجح إعادةَ الحساب`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val src = FakeRouteSource()
        val e = engine(route, src)
        driveUntilOffRoute(e, fixes)
        src.deliver(RouteReply.Failed(RerouteFailure.NETWORK))
        assertEquals(RerouteStatus.REROUTE_FAILED, e.reroute!!.status)

        // **والسائقُ يتابع والشبكةُ منقطعة** — لا نداءَ في التهدئة.
        //
        // **والقراءاتُ تُبنى من قراءة التأكيد لا من آخر الرفيدة** —
        // **تلك تبعد أربعَ مئةِ متر**، فتُقرأ حركةً ذاتَ معنىً وتُعيد
        // الطلبَ لسببٍ لا يفحصه هذا الاختبار.
        val last = e.lastAcceptedFix!!
        val callsAfterFail = src.calls
        (1..3).forEach { i ->
            e.onFix(last.copy(atMs = last.atMs + i * 1_000L, lng = last.lng + i * 0.00002))
        }
        assertEquals("نُودي أثناء التهدئة", callsAfterFail, src.calls)

        // **ثمّ تمرّ المهلةُ وتعود الشبكة** — ولو كان واقفاً.
        val later = last.copy(atMs = last.atMs + 6_000L)
        e.onFix(later)
        assertEquals("لم يُعَد الطلبُ بعد التهدئة", callsAfterFail + 1, src.calls)
        src.deliver(RouteReply.Ok(altRouteAt(e.lastAcceptedFix!!)))
        println("RRT-022 · نداءات=${src.calls} · تركيبات=${e.reroute!!.installs}")
        assertEquals(1, e.reroute!!.installs)
        assertEquals(RerouteStatus.NONE, e.reroute!!.status)
    }

    /** **RRT-023** — `reroute-after-movement`: إخفاقٌ غيرُ شبكيٍّ يحتاج معنى. */
    @Test
    fun `RRT-023 إخفاقُ المسار يحتاج حركةً أو صبرا`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val src = FakeRouteSource()
        val e = engine(route, src)
        driveUntilOffRoute(e, fixes)
        src.deliver(RouteReply.Failed(RerouteFailure.NO_ROUTE))
        val after = src.calls

        // **مهلةٌ مرّت والسائقُ لم يبرح** — ولا يُعاد الطلب.
        val last = e.lastAcceptedFix!!
        e.onFix(last.copy(atMs = last.atMs + 8_000L))
        assertEquals("أُعيد الطلبُ بلا معنى", after, src.calls)

        // **ثمّ تحرّك مئةَ متر** — فيُعاد.
        val moved = last.copy(atMs = last.atMs + 16_000L, lat = last.lat + 100 * 0.000009)
        e.onFix(moved)
        println("RRT-023 · نداءات=${src.calls} · آخرُ إخفاق=${e.reroute!!.lastFailure}")
        assertEquals("الحركةُ لم تُعِد الطلب", after + 1, src.calls)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٤ · السباقاتُ والأجيال**
    // ══════════════════════════════════════════════════════════════════

    /** **RRT-030** — `offroute-return-before-response`. */
    @Test
    fun `RRT-030 جوابٌ يصل بعد عودة السائق يُطرح`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val src = FakeRouteSource()
        val e = engine(route, src)
        driveUntilOffRoute(e, fixes)
        val kept = RouteFingerprint.of(e.currentRoute!!)

        // **ويعود إلى مساره قبل أن يصل الجواب.**
        val base = RouteFixtures.driveAlong(route, speedMps = 10f)
        val t0 = e.lastAcceptedFix!!.atMs
        val offsets = listOf(60.0, 30.0, 5.0, 0.0, 0.0)
        offsets.forEachIndexed { i, off ->
            e.onFix(
                base[10 + i].copy(
                    lng = RouteFixtures.LNG0 + off * 0.0000111,
                    atMs = t0 + (i + 1) * 1_000L,
                ),
            )
        }
        assertEquals("لم يعد إلى المسار", OffRouteDetector.State.ON_ROUTE, e.detector.state)

        src.deliver(RouteReply.Ok(altRouteAt(e.lastAcceptedFix!!)))
        println("RRT-030 · تركيبات=${e.reroute!!.installs} · آخرُ حال=${e.reroute!!.lastFailure}")
        assertEquals("جوابٌ مُلغىً استبدل مساراً صالحا", 0, e.reroute!!.installs)
        assertEquals("المسارُ تبدّل", kept, RouteFingerprint.of(e.currentRoute!!))
    }

    /** **RRT-031** — `route-generation-race` و`order-target-change-race`. */
    @Test
    fun `RRT-031 جوابٌ من جيلٍ مضى لا يستبدل مساراً أحدث`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val src = FakeRouteSource()
        val e = engine(route, src)
        driveUntilOffRoute(e, fixes)
        val genAtRequest = e.generation

        // **وأثناء الطلب يتبدّل طورُ الرحلة** — فتُسلَّم الشاشةُ مساراً
        // إلى الزبون. (البند ١٩.)
        val newTarget = RouteFixtures.singleRight()
        e.setRoute(newTarget)
        assertTrue("الجيلُ لم يزد", e.generation > genAtRequest)
        val installed = RouteFingerprint.of(e.currentRoute!!)

        src.deliver(RouteReply.Ok(altRouteAt(fixes.last())))
        println("RRT-031 · جيلُ الطلب=$genAtRequest · الجيلُ الآن=${e.generation}")
        assertEquals("جوابٌ من طورٍ مضى استبدل الوجهةَ الجديدة", installed, RouteFingerprint.of(e.currentRoute!!))
    }

    /** **RRT-032** — `stale-response`: جوابٌ لطلبٍ قديمٍ بعد طلبٍ أحدث. */
    @Test
    fun `RRT-032 جوابُ طلبٍ قديمٍ يُطرح`() {
        val route = RouteFixtures.straight()
        val src = FakeRouteSource()
        val e = engine(route, src)
        val alt = altRouteAt(RouteFixtures.driveAlong(route).first())
        // **جوابٌ برقمٍ لم يُطلق قطّ.**
        e.reroute!!.onReply(999L, e.generation, RouteReply.Ok(alt), RouteFixtures.driveAlong(route).first())
        println("RRT-032 · تركيبات=${e.reroute!!.installs} · حال=${e.reroute!!.lastFailure}")
        assertEquals(0, e.reroute!!.installs)
        assertEquals(RerouteFailure.STALE_RESPONSE, e.reroute!!.lastFailure)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٥ · صلاحيّةُ المسار الجديد**
    // ══════════════════════════════════════════════════════════════════

    /** **RRT-040** — `same-route-valid`: المسارُ نفسُه قد يكون صحيحا. */
    @Test
    fun `RRT-040 مسارٌ يعود كما هو ويصلح فيُركَّب`() {
        // ══════════════════════════════════════════════════════════════
        // **و«خرج قليلاً» حالٌ واقعيّةٌ لا مفتعلة**
        // ══════════════════════════════════════════════════════════════
        //
        // (أمرُ المالك، البند ١٥: «قد يكون السائقُ خرج قليلاً ثمّ أعاد
        //  OSRM نفسَ المسار لأنّه ما زال أفضلَ طريقٍ بالفعل».)
        //
        // **والشارعُ الموازي المعاكسُ هو الحالُ بعينها**: عشرون متراً
        // عن الخطّ — **دون عتبة الخروج**، والخروجُ أُعلن بالاتّجاه.
        // **فالمسارُ نفسُه يصلح من موضعه.**
        val (route, fixes) = OffRouteFixtures.parallelRoadOpposite()
        val src = FakeRouteSource()
        val e = engine(route, src)
        driveUntilOffRoute(e, fixes)
        assertEquals(1, src.calls)
        // **ونسخةٌ مطابقةٌ بصمةً** — والسائقُ قريبٌ منها.
        val same = NavRoute(route.geometry, route.cumulativeM.copyOf(), route.maneuvers)
        assertTrue("البصمتان لا تتطابقان", RouteFingerprint.same(route, same))
        src.deliver(RouteReply.Ok(same))
        println("RRT-040 · تركيبات=${e.reroute!!.installs} · بصمةٌ متطابقة=true")
        assertEquals("مسارٌ صالحٌ رُفض لأنّه مطابق", 1, e.reroute!!.installs)
    }

    /** **RRT-041** — `same-route-unusable`: مسارٌ يترك السائقَ بعيداً يُرفض. */
    @Test
    fun `RRT-041 مسارٌ يترك السائقَ بعيداً يُرفض ويُهدَّأ`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val src = FakeRouteSource()
        val e = engine(route, src)
        driveUntilOffRoute(e, fixes)
        val kept = RouteFingerprint.of(e.currentRoute!!)

        // **مسارٌ صحيحٌ في نفسِه على بُعد كيلومترين.**
        val far = altRouteAt(fixes.last().copy(lat = fixes.last().lat + 0.02))
        src.deliver(RouteReply.Ok(far))
        println(
            "RRT-041 · تركيبات=${e.reroute!!.installs} · مرفوضة=${e.reroute!!.rejected} · " +
                "حال=${e.reroute!!.lastFailure}",
        )
        assertEquals("رُكّب مسارٌ يُخرج السائقَ فورا", 0, e.reroute!!.installs)
        assertEquals(1, e.reroute!!.rejected)
        assertEquals(RerouteFailure.INVALID_ROUTE, e.reroute!!.lastFailure)
        assertEquals("المسارُ القديمُ ضاع", kept, RouteFingerprint.of(e.currentRoute!!))
    }

    /** **RRT-042** — البصمةُ تفرّق طريقين متساويَي الطول والعدد. */
    @Test
    fun `RRT-042 البصمةُ تفرّق ما لا يفرّقه الطولُ والعدد`() {
        val north = RouteFixtures.straight()
        val east = NavRoute.of(
            (0..10).map { GeoPoint(RouteFixtures.LAT0, RouteFixtures.east(it * 50.0)) },
            north.maneuvers,
        )
        println(
            "RRT-042 · رؤوس=${north.geometry.size}/${east.geometry.size} · " +
                "طول=${"%.0f".format(north.totalM)}/${"%.0f".format(east.totalM)} · " +
                "تشابه=${"%.2f".format(RouteFingerprint.similarity(north, east))}",
        )
        assertEquals("العددُ مختلف — فالاختبارُ لا يفحص شيئا", north.geometry.size, east.geometry.size)
        assertTrue("الطولان متباعدان", kotlin.math.abs(north.totalM - east.totalM) < 15.0)
        assertTrue("**البصمةُ لم تفرّق طريقين مختلفين**", !RouteFingerprint.same(north, east))
    }

    // ══════════════════════════════════════════════════════════════════
    // **٦ · بلا بابٍ إلى الشبكة**
    // ══════════════════════════════════════════════════════════════════

    /** **RRT-050** — محرّكٌ بلا مصدرٍ يعمل كاملاً ولا يسقط. */
    @Test
    fun `RRT-050 بلا بابِ شبكةٍ تعمل الملاحةُ كما هي`() {
        val (route, fixes, _) = OffRouteFixtures.realDeviation()
        val e = NavEngine().also { it.setRoute(route) }
        val out = fixes.map { e.onFix(it) }
        assertNull("بُني محرّكُ إعادةٍ بلا باب", e.reroute)
        assertTrue("لم يُكشف الخروج", out.any { it.isOffRoute })
        assertTrue(out.all { it.reroute == RerouteStatus.NONE })
    }

    // ══════════════════════════════════════════════════════════════════
    // **٧ · القياسُ الكامل — من الانحراف إلى المسار الجديد**
    // ══════════════════════════════════════════════════════════════════

    /** **RRT-060** — البند ٢٤: المدّةُ كلُّها لا زمنُ الكاشف وحدَه. */
    @Test
    fun `RRT-060 قياسُ المدّة من الانحراف إلى المسار الجديد`() {
        val (route, fixes, at) = OffRouteFixtures.realDeviation()
        val src = FakeRouteSource()
        val e = engine(route, src)
        val out = driveUntilOffRoute(e, fixes)
        val confirmed = out.size - 1

        val deviationMs = fixes[at].atMs
        val confirmedMs = fixes[confirmed].atMs
        val requestMs = e.reroute!!.lastRequestAtMs
        // **وزمنُ الشبكة يُحاكى بثانيتين** — ويُقرأ من ساعة الرحلة.
        val responseAt = fixes[confirmed].copy(atMs = confirmedMs + 2_000L)
        e.reroute!!.onReply(1L, e.generation, RouteReply.Ok(altRouteAt(e.lastAcceptedFix!!)), responseAt)
        val installedMs = e.reroute!!.lastInstallAtMs

        println(
            "RRT-060 · الانحرافُ عند=${deviationMs}ملّي · التأكيدُ=${confirmedMs} " +
                "(${(confirmedMs - deviationMs) / 1000.0}ث) · الطلبُ=${requestMs} · " +
                "التركيبُ=${installedMs} · **الإجماليُّ=${(installedMs - deviationMs) / 1000.0}ث** · " +
                "نداءات=${src.calls}",
        )
        assertEquals("أُطلق أكثرُ من نداء", 1, src.calls)
        assertEquals("الطلبُ لم يُطلق لحظةَ التأكيد", confirmedMs, requestMs)
        assertTrue("لم يُركَّب المسار", installedMs > 0)
        assertTrue("الإجماليُّ تجاوز خمسَ عشرةَ ثانية", (installedMs - deviationMs) / 1000.0 <= 15.0)
    }
}
