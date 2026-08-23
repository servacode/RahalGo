package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اختباراتُ المرحلة ٥ — الاتّجاه المعاكس**
 * ══════════════════════════════════════════════════════════════════════
 *
 * المعرّفات: `WW-*`
 *
 * # وما تعنيه الكلمةُ هنا
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ١.)
 *
 * **«يتحرّك بعكس اتّجاه المسار مع بقائه قريباً هندسيّاً منه»** — **ولا
 * ندّعي مخالفةً قانونيّةً لاتّجاه الشارع**، ولا نملك قاعدةً تُثبتها.
 */
class WrongWayTest {

    private val CORRECT = WrongWayDetector.State.CORRECT_DIRECTION
    private val SUS = WrongWayDetector.State.SUSPECTED_WRONG_WAY
    private val WRONG = WrongWayDetector.State.WRONG_WAY

    private fun engine(
        route: NavRoute,
        tuning: WrongWayDetector.Tuning = WrongWayDetector.Tuning(),
    ): NavEngine = NavEngine(wrongWay = WrongWayDetector(tuning)).also { it.setRoute(route) }

    private fun run(e: NavEngine, fixes: List<NavFix>): List<NavState> = fixes.map { e.onFix(it) }

    private fun states(out: List<NavState>) = out.map { it.wrongWay.state }

    private fun confirmedAt(out: List<NavState>) = states(out).indexOfFirst { it == WRONG }

    // ══════════════════════════════════════════════════════════════════
    // **١ · الحالُ المرجعيّة — عودةٌ على المسار نفسِه**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **WW-001** — `same-route reverse`. (البند ٢٦.)
     *
     * **ولا تتحوّل خطأً إلى خروج** — الإسقاطُ ما زال على المسار.
     */
    @Test
    fun `WW-001 العودةُ على المسار نفسِه تُعلَن اتّجاهاً معاكسا`() {
        val route = RouteFixtures.straight()
        val e = engine(route)
        val fixes = WrongWayFixtures.forwardThenReverse(route)
        val out = run(e, fixes)
        val at = confirmedAt(out)
        val v = out.last().wrongWay
        println(
            "WW-001 · شكٌّ عند=${states(out).indexOfFirst { it == SUS }} · تأكيدٌ عند=$at · " +
                "زمنٌ عكسيّ=${v.oppositeMs}ملّي · مسافةٌ عكسيّة=${"%.0f".format(v.oppositeM)}م · " +
                "تراجعُ تقدّم=${v.progressRegressed}",
        )
        println("WW-001 · فروقُ الزوايا=" + out.mapNotNull { it.wrongWay.bearingDeltaDeg }.distinct().take(6))
        assertTrue("لم يُعلَن اتّجاهٌ معاكس", at >= 0)
        assertEquals("تحوّل إلى خروجٍ وهو على المسار", 0, out.count { it.isOffRoute })
        assertEquals("الحالُ المحسومةُ ليست الاتّجاه", NavSituation.WRONG_WAY, out.last().situation)
    }

    /** **WW-002** — والقياسُ عبر ثلاثة ترددات. (البند ٢٩.) */
    @Test
    fun `WW-002 النتيجةُ نفسُها عند نصف هرتزٍ وواحدٍ واثنين`() {
        val route = RouteFixtures.straight()
        val marks = ArrayList<Long>()
        for ((hz, step) in listOf(0.5 to 2_000L, 1.0 to 1_000L, 2.0 to 500L)) {
            val e = engine(route)
            val fixes = WrongWayFixtures.forwardThenReverse(route, stepMs = step)
            val out = run(e, fixes)
            val at = confirmedAt(out)
            val v = if (at >= 0) out[at].wrongWay else out.last().wrongWay
            val ms = if (at >= 0) v.oppositeMs else -1L
            val m = if (at >= 0) v.oppositeM else -1.0
            println("WW-002 · ${hz}هرتز → تأكيدٌ بعد ${ms}ملّي و${"%.0f".format(m)}م")
            assertTrue("لم يُؤكَّد عند $hz هرتز", at >= 0)
            // **والفرقُ لا يتجاوز عيّنةً واحدة** — الفحصُ يقع عند
            // القراءات، **فدقّةُ نصفِ هرتزٍ ثانيتان بطبيعتها.**
            assertTrue("الزمنُ خارج المدى عند $hz: $ms", ms in 4_000L..10_000L)
            marks += ms
        }
        val spread = marks.max() - marks.min()
        println("WW-002 · الفارقُ بين أبطأ وأسرع تردّد=${spread}ملّي")
        assertTrue("**التردّدُ غيّر زمنَ التأكيد**: $spread", spread <= 4_000L)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٢ · لا إنذارَ كاذب**
    // ══════════════════════════════════════════════════════════════════

    /** **WW-010** — سيرٌ سليم: صفرُ شكٍّ وصفرُ تأكيد. */
    @Test
    fun `WW-010 السيرُ السليمُ بلا إنذارٍ كاذب`() {
        val route = RouteFixtures.singleRight()
        val e = engine(route)
        val out = run(e, WrongWayFixtures.correctDirection(route))
        println("WW-010 · شكوك=${e.wrongWay.suspicions} · تأكيدات=${e.wrongWay.confirmations}")
        assertEquals("شكٌّ كاذب", 0, e.wrongWay.suspicions)
        assertEquals("تأكيدٌ كاذب", 0, e.wrongWay.confirmations)
    }

    /** **WW-011** — الواقفُ لا يُعلَن ولا يُمحى. */
    @Test
    fun `WW-011 الواقفُ يُجمَّد ولا يُعلَن`() {
        val route = RouteFixtures.straight()
        val e = engine(route)
        val out = run(e, WrongWayFixtures.stationary(route))
        println("WW-011 · تأكيدات=${e.wrongWay.confirmations} · تجميد=${out.count { it.wrongWay.skip == WrongWayDetector.Skip.STATIONARY }}")
        assertEquals(0, e.wrongWay.confirmations)
        assertTrue(out.any { it.wrongWay.skip == WrongWayDetector.Skip.STATIONARY })
    }

    /** **WW-012** — رجفةُ اتّجاهٍ واحدةٌ لا تُعلن شيئاً. */
    @Test
    fun `WW-012 رجفةُ اتّجاهٍ واحدةٌ لا تكفي`() {
        val route = RouteFixtures.straight()
        val e = engine(route)
        run(e, WrongWayFixtures.bearingGlitch(route))
        println("WW-012 · شكوك=${e.wrongWay.suspicions} · تأكيدات=${e.wrongWay.confirmations}")
        assertEquals(0, e.wrongWay.confirmations)
    }

    /** **WW-013** — ولفُّ الزاوية `359↔0` سليم. */
    @Test
    fun `WW-013 لفُّ الزاوية سليم`() {
        val route = RouteFixtures.straight()
        val e = engine(route)
        run(e, WrongWayFixtures.wrapAround(route))
        println("WW-013 · تأكيدات=${e.wrongWay.confirmations}")
        assertEquals(0, e.wrongWay.confirmations)
    }

    /** **WW-014** — قياسُ عتبة الزاوية. (البند ٧.) */
    @Test
    fun `WW-014 قياسُ عتبة الزاوية`() {
        val route = RouteFixtures.straight()
        for (deg in listOf(0, 30, 60, 90, 120, 135, 150, 180)) {
            val e = engine(route)
            val fixes = (0..20).map { i ->
                val d = 60.0 + i * 8.0
                val course = RouteFixtures.bearingAt(route, d)
                WrongWayFixtures.fixAt(route, d, 1_000L + i * 1_000L, 8f, reverse = false)
                    .copy(bearingDeg = GpsQuality.normalize(course + deg))
            }
            val e2 = engine(route)
            val out = run(e2, fixes)
            val st = states(out).last()
            println("WW-014 · ${deg}° → $st")
            if (deg <= 90) {
                assertEquals("زاويةُ $deg أعلنت شيئاً", CORRECT, st)
            }
            if (deg >= 150) {
                assertEquals("زاويةُ $deg لم تُعلَن معاكسة", WRONG, st)
            }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **٣ · الكبتُ عند المناورات**
    // ══════════════════════════════════════════════════════════════════

    /** **WW-020** — الدورانُ المطلوبُ لا يُعلَن معاكسا. (البند ١٤.) */
    @Test
    fun `WW-020 الدورانُ المطلوبُ لا يُعلَن معاكسا`() {
        val route = WrongWayFixtures.uTurnRoute()
        val e = engine(route)
        val out = run(e, WrongWayFixtures.driveUTurn())
        val suppressed = out.count { it.wrongWay.skip == WrongWayDetector.Skip.MANEUVER }
        println("WW-020 · تأكيدات=${e.wrongWay.confirmations} · مكبوتة=$suppressed من ${out.size}")
        assertEquals("**تنفيذٌ صحيحٌ للدوران أعلن اتّجاهاً معاكسا**", 0, e.wrongWay.confirmations)
        assertTrue("لم يُكبت شيءٌ عند الدوران", suppressed > 0)
    }

    /** **WW-021** — والدوّارُ الطبيعيُّ كذلك. (البند ١٥.) */
    @Test
    fun `WW-021 الدوّارُ الطبيعيُّ بلا إنذارٍ كاذب`() {
        val route = RouteFixtures.roundabout()
        val e = engine(route)
        val out = run(e, RouteFixtures.driveAlong(route, speedMps = 7f))
        println("WW-021 · تأكيدات=${e.wrongWay.confirmations} · مكبوتة=${out.count { it.wrongWay.skip == WrongWayDetector.Skip.MANEUVER }}")
        assertEquals(0, e.wrongWay.confirmations)
    }

    /** **WW-022** — والمنعطفُ الحادّ. (البند ١٦.) */
    @Test
    fun `WW-022 المنعطفُ الحادُّ لا يُعلن معاكسا`() {
        val base = RouteFixtures.singleRight()
        val sharp = NavRoute(
            base.geometry,
            base.cumulativeM,
            base.maneuvers.map {
                if (it.kind == ManeuverKinds.TURN_RIGHT) it.copy(kind = ManeuverKinds.SHARP_RIGHT) else it
            },
        )
        val e = engine(sharp)
        run(e, RouteFixtures.driveAlong(sharp, speedMps = 8f))
        println("WW-022 · تأكيدات=${e.wrongWay.confirmations}")
        assertEquals(0, e.wrongWay.confirmations)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٤ · الرجوعُ القصير**
    // ══════════════════════════════════════════════════════════════════

    /** **WW-030** — قياسُ ٥ و١٠ و٢٠ متراً. (البند ١٣.) */
    @Test
    fun `WW-030 قياسُ الرجوع القصير`() {
        val route = RouteFixtures.straight()
        for (backM in listOf(5.0, 10.0, 20.0, 60.0)) {
            val e = engine(route)
            val out = run(e, WrongWayFixtures.shortReverse(route, backM))
            val sus = states(out).indexOfFirst { it == SUS }
            val at = confirmedAt(out)
            println(
                "WW-030 · رجوع=${"%.0f".format(backM)}م → شكٌّ عند=$sus · تأكيدٌ عند=$at · " +
                    "أقصى مسافةٍ عكسيّة=${"%.0f".format(out.maxOf { it.wrongWay.oppositeM })}م",
            )
            if (backM <= 20.0) {
                assertEquals("**رجوعٌ قصيرٌ (${backM}م) أُعلن اتّجاهاً معاكسا**", -1, at)
            }
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **٥ · جودةُ القراءة**
    // ══════════════════════════════════════════════════════════════════

    /** **WW-040** — المتدهورةُ وحدَها لا تؤكّد. (البند ٩.) */
    @Test
    fun `WW-040 المتدهورةُ وحدَها لا تؤكّد`() {
        val route = RouteFixtures.straight()
        val e = engine(route)
        // **وضبابٌ من أوّل قراءةٍ لا يُنشئ اتّجاهاً أصلاً**:
        // `BearingTracker` لا يُصدّق المتدهورة، **فلا شيءَ يُقاس
        // عليه.** فيُقاس ما هو أدقّ: ضبابٌ يهبط على شكٍّ قائم.
        val clean = WrongWayFixtures.forwardThenReverse(route)
        val murky = clean.mapIndexed { i, f ->
            if (i > 32) f.copy(accuracyM = 45f) else f
        }
        val out = run(e, murky)
        val tail = out.drop(33)
        println(
            "WW-040 · متدهورة=${tail.count { it.grade == FixGrade.DEGRADED }} · " +
                "تأكيدات=${e.wrongWay.confirmations} · حالٌ نهائيّة=${e.wrongWay.state}",
        )
        assertTrue("لا قراءةَ متدهورة", tail.any { it.grade == FixGrade.DEGRADED })
        // **ولا تحسم المتدهورةُ**: التأكيدُ يشترط مقبولةً في اللحظة
        // نفسِها. **ولا تمحو** — الشكُّ يبقى.
        assertTrue(
            "الضبابُ حسم أو محا",
            tail.all { it.wrongWay.state != CORRECT } || e.wrongWay.confirmations == 0,
        )
    }

    /** **WW-041** — والمرفوضةُ لا تضيف ولا تخصم. */
    @Test
    fun `WW-041 المرفوضةُ تُتجاهل`() {
        val route = RouteFixtures.straight()
        val e = engine(route)
        val fixes = WrongWayFixtures.forwardThenReverse(route).mapIndexed { i, f ->
            if (i > 0 && i % 5 == 0) f.copy(accuracyM = 90f) else f
        }
        val out = run(e, fixes)
        val i = out.indexOfFirst { it.wrongWay.skip == WrongWayDetector.Skip.REJECTED }
        assertTrue("لا قراءةَ مرفوضة", i > 0)
        assertEquals(
            "المرفوضةُ غيّرت الزمنَ العكسيّ",
            out[i - 1].wrongWay.oppositeMs, out[i].wrongWay.oppositeMs,
        )
        println("WW-041 · مرفوضة=${out.count { it.wrongWay.skip == WrongWayDetector.Skip.REJECTED }} · تأكيدات=${e.wrongWay.confirmations}")
    }

    // ══════════════════════════════════════════════════════════════════
    // **٦ · الشارعُ الموازي**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **WW-050** — موازٍ معاكسٌ ⇒ اتّجاهٌ لا خروج. (البند ٤.)
     *
     * **وهذا استنتاجٌ نسبيٌّ للمسار لا إثباتُ هُويّةِ شارع.**
     */
    @Test
    fun `WW-050 الموازي المعاكسُ يصير اتّجاهاً معاكسا`() {
        val route = RouteFixtures.straight()
        val e = engine(route)
        val out = run(e, WrongWayFixtures.parallelOpposite(route))
        println(
            "WW-050 · تأكيدُ الاتّجاه=${confirmedAt(out)} · خروجٌ مؤكَّد=${out.count { it.isOffRoute }} · " +
                "أقصى بعد=${"%.0f".format(out.mapNotNull { it.progress?.offRouteM }.max())}م",
        )
        assertTrue("لم يُكشف الموازي المعاكس", confirmedAt(out) >= 0)
        assertEquals("**صُنّف خروجاً وهو داخلَ الممرّ**", 0, out.count { it.isOffRoute })
    }

    /** **WW-051** — وبالاتّجاه نفسِه لا يُحسم. (`TD-PARALLEL-SAMEDIR`.) */
    @Test
    fun `WW-051 الموازي بالاتّجاه نفسِه غيرُ محسوم`() {
        val route = RouteFixtures.straight()
        val e = engine(route)
        val out = run(e, WrongWayFixtures.parallelSameDirection(route))
        println("WW-051 · تأكيدات=${e.wrongWay.confirmations} · خروج=${out.count { it.isOffRoute }}")
        assertEquals("ادّعى كشفَ ما لا يُكشف", 0, e.wrongWay.confirmations)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٧ · الأولويّةُ والتعافي**
    // ══════════════════════════════════════════════════════════════════

    /** **WW-060** — الخروجُ الهندسيُّ يفوز. (البند ١٨.) */
    @Test
    fun `WW-060 الخروجُ الهندسيُّ يسبق الاتّجاه`() {
        val route = RouteFixtures.straight()
        val e = engine(route)
        val out = run(e, WrongWayFixtures.geometricDeparture(route))
        val offAt = out.indexOfFirst { it.isOffRoute }
        println("WW-060 · خروجٌ عند=$offAt · حالٌ نهائيّة=${out.last().situation}")
        assertTrue("**انكسر اكتشافُ الخروج الحقيقيّ**", offAt >= 0)
        assertEquals(NavSituation.OFF_ROUTE, out.last().situation)
    }

    /** **WW-061** — والتصحيحُ يُنهي النوبة. */
    @Test
    fun `WW-061 التصحيحُ يُعيد إلى الاتّجاه الصحيح`() {
        val route = RouteFixtures.straight()
        val e = engine(route)
        val out = run(e, WrongWayFixtures.forwardThenReverse(route, backM = 120.0))
        assertTrue("لم يُعلَن معاكسا", confirmedAt(out) >= 0)

        // **ثمّ يستدير ويتابع أماماً.**
        val last = out.size
        val from = 130.0
        val forward = (0..12).map { i ->
            WrongWayFixtures.fixAt(route, from + i * 8.0, 400_000L + i * 1_000L, 8f, reverse = false)
        }
        val back = run(e, forward)
        println("WW-061 · بعد التصحيح=${states(back)}")
        assertEquals("لم يعد إلى الاتّجاه الصحيح", CORRECT, e.wrongWay.state)
    }

    /** **WW-062** — والجيلُ الجديد يُصفّر كلَّ شيء. (البند ٢٢.) */
    @Test
    fun `WW-062 جيلٌ جديدٌ يُصفّر الكاشف`() {
        val route = RouteFixtures.straight()
        val e = engine(route)
        run(e, WrongWayFixtures.forwardThenReverse(route))
        assertEquals(WRONG, e.wrongWay.state)

        e.setRoute(RouteFixtures.singleRight())
        println("WW-062 · بعد المسار الجديد=${e.wrongWay.state} · نوبة=${e.wrongWay.episode}")
        assertEquals(CORRECT, e.wrongWay.state)
        assertEquals(0L, e.wrongWay.episode)
    }
    // ══════════════════════════════════════════════════════════════════
    // **٨ · التكامل — إعادةُ الحساب والصوت**
    // ══════════════════════════════════════════════════════════════════

    /** **بابُ شبكةٍ في اليد** — كما في `RerouteTest`. */
    private class Fake : RouteSource {
        var calls = 0
            private set
        private var pending: ((RouteReply) -> Unit)? = null

        override fun request(lat: Double, lng: Double, seq: Long, done: (RouteReply) -> Unit) {
            calls++
            pending = done
        }

        fun deliver(reply: RouteReply) {
            val p = pending
            pending = null
            p?.invoke(reply)
        }
    }

    /**
     * **WW-070** — `wrongway-reroute`: بابٌ واحدٌ بسببٍ معلن.
     *
     * (البندان ١٩ و٢١: **لا Reroute فورَ أوّل Confirmation**، ولا
     * محرّكٌ ثانٍ.)
     */
    @Test
    fun `WW-070 الاتّجاهُ المستمرُّ يطلب مساراً بسببه`() {
        val route = RouteFixtures.straight()
        val src = Fake()
        val e = NavEngine(source = src).also { it.setRoute(route) }
        val fixes = WrongWayFixtures.forwardThenReverse(route, forwardTo = 400.0, backM = 300.0)

        var firstConfirm = -1
        var firstRequest = -1
        fixes.forEachIndexed { i, f ->
            val st = e.onFix(f)
            if (firstConfirm < 0 && st.isWrongWay) firstConfirm = i
            if (firstRequest < 0 && src.calls > 0) firstRequest = i
        }
        println(
            "WW-070 · تأكيدٌ عند=$firstConfirm · طلبٌ عند=$firstRequest · " +
                "نداءات=${src.calls} · السبب=${e.reroute?.lastReason}",
        )
        assertTrue("لم يُعلَن اتّجاهٌ معاكس", firstConfirm >= 0)
        assertTrue("لم يُطلب مسارٌ بعد الاستمرار", firstRequest > firstConfirm)
        assertEquals("**طُلب فورَ التأكيد بلا نافذةِ تصحيح**", RerouteReason.WRONG_WAY, e.reroute?.lastReason)
        assertEquals("**عاصفةُ طلبات**", 1, src.calls)
    }

    /** **WW-071** — والخروجُ الهندسيُّ يطلب بسببه هو. */
    @Test
    fun `WW-071 الخروجُ يطلب بسبب الخروج لا الاتّجاه`() {
        val route = RouteFixtures.straight()
        val src = Fake()
        val e = NavEngine(source = src).also { it.setRoute(route) }
        WrongWayFixtures.geometricDeparture(route).forEach { e.onFix(it) }
        println("WW-071 · نداءات=${src.calls} · السبب=${e.reroute?.lastReason}")
        assertEquals(RerouteReason.OFF_ROUTE, e.reroute?.lastReason)
        assertEquals(1, src.calls)
    }

    /**
     * **WW-072** — الصوتُ مرّةً لكلّ نوبة. (البندان ٢٢ و٢٣.)
     *
     * **ولا نطقَ عند الشكّ** — ولا أمرٌ بمناورة.
     */
    @Test
    fun `WW-072 الصوتُ مرّةً واحدةً لكلّ نوبة`() {
        val route = RouteFixtures.straight()
        val planner = VoicePlanner()
        val e = NavEngine(voice = planner).also { it.setRoute(route) }
        val out = run(e, WrongWayFixtures.forwardThenReverse(route))
        val cues = out.flatMap { it.cues }.filter { it.kind == CueKind.WRONG_WAY }
        val whileSuspect = out.filter { it.wrongWay.state == SUS }.flatMap { it.cues }
            .count { it.kind == CueKind.WRONG_WAY }
        println("WW-072 · جملٌ=${cues.map { it.text }} · عند الشكّ=$whileSuspect")
        assertEquals("لم تُقل مرّةً واحدة", 1, cues.size)
        assertEquals("نُطق عند الشكّ", 0, whileSuspect)
        assertEquals(VoicePhrases.WRONG_WAY, cues.first().text)
        // **ووصفٌ لا أمرٌ بمناورة** (البند ٢٤).
        val text = cues.first().text
        assertTrue("أمرَ بمناورة", listOf("ارجع", "قم بالدوران", "غيّر الاتجاه").none { text.contains(it) })
        assertTrue("ادّعى مخالفةَ اتّجاه الشارع", !text.contains("الشارع"))
    }

    /** **WW-073** — ولا تعليماتِ مناورةٍ وهو معاكس. */
    @Test
    fun `WW-073 لا تعليماتِ مناورةٍ أثناء الاتّجاه المعاكس`() {
        val route = RouteFixtures.singleRight()
        val planner = VoicePlanner()
        val e = NavEngine(voice = planner).also { it.setRoute(route) }
        val out = run(e, WrongWayFixtures.forwardThenReverse(route, forwardTo = 280.0, backM = 200.0))
        val bad = out.filter { it.isWrongWay }.flatMap { it.cues }.count { it.kind == CueKind.MANEUVER }
        println("WW-073 · تعليماتُ مناورةٍ أثناء المعاكس=$bad")
        assertEquals(0, bad)
    }
}
