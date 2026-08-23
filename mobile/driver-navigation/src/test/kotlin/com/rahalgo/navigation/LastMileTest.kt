package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **آخرُ ميل — الرفائدُ أ إلى ي**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ آخر ميل، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 *
 * **والعيبُ المقيسُ قبل الإصلاح** (`LastMileTraceTest`): أربعون قراءةً
 * بسرعة ١٦ م/ث بعد نهاية الخطّ → **`OFF_ROUTE` مؤكّدة وأربعةُ طلباتِ
 * إعادةِ حساب.**
 */
class LastMileTest {

    /** **مصدرٌ يعدّ الطلبات ولا يشبك** — فيُعرف أطُلبت إعادةُ حساب. */
    private class CountingSource : RouteSource {
        var asked = 0
        override fun request(lat: Double, lng: Double, seq: Long, done: (RouteReply) -> Unit) {
            asked++
            done(RouteReply.Failed(RerouteFailure.NETWORK))
        }
    }

    private class Rig(offsetM: Double, trip: TripTarget = TripTarget.DROPOFF) {
        val src = CountingSource()
        val planner = VoicePlanner().also { it.target = trip }
        val engine = NavEngine(voice = planner, source = src)
        val route: NavRoute = RouteFixtures.straight()
        val said = mutableListOf<VoiceCue>()
        var t = 0L
        val end: GeoPoint get() = route.geometry.last()

        init {
            engine.setRoute(route)
            engine.arrivalTarget = GeoPoint(end.lat + offsetM / 111320.0, end.lng)
        }

        fun drive() {
            for (f in RouteFixtures.driveAlong(route)) {
                said += engine.onFix(f).cues
                t = f.atMs
            }
        }

        /** **يمشي شمالاً من نهاية الخطّ** — نحوَ الهدف. */
        fun walkTo(m: Double, samples: Int = 4, speed: Float = 3f) {
            repeat(samples) {
                t += 3000
                val p = GeoPoint(end.lat + m / 111320.0, end.lng)
                said += engine.onFix(NavFix(p.lat, p.lng, 5f, speed, 0f, t)).cues
            }
        }

        fun arrivals() = said.count { it.kind == CueKind.ARRIVE && it.stage == CueStage.EVENT }
        fun routeEnds() = said.count { it.kind == CueKind.ROUTE_END }
        fun claims() = said.count { it.text.contains("وصلت") }
    }

    // ══════════════════════════════════════════════════════════════
    // **أ · الهدفُ عند نهاية الخطّ — لا نداءَ نهاية**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `أ - خمسةُ أمتارٍ فوصولٌ في القراءة نفسِها ونداءٌ واحد`() {
        val r = Rig(5.0)
        r.drive()
        r.walkTo(5.0)
        assertEquals("دعوى الوصول: ${r.said.map { it.text }}", 1, r.arrivals())
        assertEquals("نداءُ نهايةِ مسارٍ لا محلَّ له", 0, r.routeEnds())
        assertEquals(1, r.claims())
    }

    // ══════════════════════════════════════════════════════════════
    // **ب–د · خمسون ومئةٌ وعشرون ومئةٌ وثمانون**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `ب - خمسون متراً`() = lastMileScenario(50.0)

    @Test
    fun `ج - مئةٌ وعشرون متراً`() = lastMileScenario(120.0)

    @Test
    fun `د - مئةٌ وثمانون متراً`() = lastMileScenario(180.0)

    private fun lastMileScenario(offsetM: Double) {
        val r = Rig(offsetM)
        r.drive()

        // ── بلغ النهاية ───────────────────────────────────────────
        assertEquals(ArrivalPhase.LAST_MILE_TO_TARGET, r.engine.state?.arrivalPhase)
        assertEquals("نداءُ نهاية المسار مرّةً واحدة", 1, r.routeEnds())
        assertEquals("ولا دعوى وصول", 0, r.arrivals())
        assertEquals("ولا ادّعاءَ وصولٍ أيّاً كان", 0, r.claims())

        // ── ثمّ يمشي نحو الهدف ────────────────────────────────────
        r.walkTo(offsetM / 2)
        assertEquals(ArrivalPhase.LAST_MILE_TO_TARGET, r.engine.state?.arrivalPhase)
        assertNoNavAlarm(r)

        // ── فيصل ──────────────────────────────────────────────────
        r.walkTo(offsetM)
        assertEquals(ArrivalPhase.BUSINESS_TARGET_ARRIVED, r.engine.state?.arrivalPhase)
        assertEquals("دعوى الوصول مرّةً واحدة", 1, r.arrivals())
        assertEquals("ونداءُ النهاية لم يتكرّر", 1, r.routeEnds())
        assertEquals(1, r.claims())
        assertNoNavAlarm(r)
    }

    /** **ولا إنذارَ ملاحيّاً بعد نهاية الخطّ** — البندان ٤ و٧. */
    private fun assertNoNavAlarm(r: Rig) {
        val s = r.engine.state!!
        assertEquals("خروجٌ عن مسارٍ منتهٍ", OffRouteDetector.State.ON_ROUTE, s.offRoute.state)
        assertEquals("اتّجاهٌ معاكسٌ على مسارٍ منتهٍ",
            WrongWayDetector.State.CORRECT_DIRECTION, s.wrongWay.state)
        assertEquals("إعادةُ حساب", RerouteStatus.NONE, s.reroute)
        assertEquals("طلبُ مسارٍ جديد", 0, r.src.asked)
        assertEquals("جيلٌ جديد", 1L, r.engine.generation)
        assertEquals(NavSituation.ON_ROUTE, s.situation)
    }

    // ══════════════════════════════════════════════════════════════
    // **هـ · يبتعد عن الهندسة نحوَ الهدف**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `هـ - مئةُ مترٍ عن الهندسة نحوَ الهدف فلا إنذار`() {
        val r = Rig(200.0)
        r.drive()
        // **بسرعة قيادةٍ وقراءاتٍ كثيفة** — أقسى ما يكون على الكاشف.
        for (m in listOf(20.0, 40.0, 60.0, 80.0, 100.0, 120.0)) {
            r.walkTo(m, samples = 6, speed = 16f)
        }
        assertNoNavAlarm(r)
        assertEquals(ArrivalPhase.LAST_MILE_TO_TARGET, r.engine.state?.arrivalPhase)
    }

    // ══════════════════════════════════════════════════════════════
    // **و · يبتعد عن الهدف نفسِه**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `و - يبتعد عن الهدف فلا وصولَ ولا إنذار`() {
        /**
         * **أمرُ المالك (البند ٧)**: «لا Voice warning · لا auto
         * reroute · فقط لا تعلن arrival».
         */
        val r = Rig(100.0)
        r.drive()
        val end = r.end
        // **جنوباً — عكسَ الهدف.**
        for (i in 1..8) {
            r.t += 3000
            val p = GeoPoint(end.lat - i * 30.0 / 111320.0, end.lng)
            r.said += r.engine.onFix(NavFix(p.lat, p.lng, 5f, 12f, 180f, r.t)).cues
        }
        assertEquals(0, r.arrivals())
        assertEquals("نداءُ النهاية لم يتكرّر", 1, r.routeEnds())
        assertNoNavAlarm(r)
    }

    // ══════════════════════════════════════════════════════════════
    // **ز · تبدّلُ الهدف يفكّ آخرَ ميل**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `ز - هدفٌ جديدٌ يفكّ الحال`() {
        val r = Rig(180.0)
        r.drive()
        assertTrue(r.engine.guidanceEnded)

        r.engine.arrivalTarget = GeoPoint(r.end.lat + 900.0 / 111320.0, r.end.lng)
        assertTrue("لم يُفكّ المزلاجُ بتبدّل الهدف", !r.engine.guidanceEnded)
    }

    // ══════════════════════════════════════════════════════════════
    // **ح · مسارٌ جديدٌ يعيد الملاحةَ نظيفة**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `ح - مسارٌ جديدٌ يستأنف الملاحة`() {
        val r = Rig(180.0)
        r.drive()
        assertTrue(r.engine.guidanceEnded)
        val genBefore = r.engine.generation

        r.engine.setRoute(RouteFixtures.straight())
        assertTrue("لم يُفكّ المزلاجُ بمسارٍ جديد", !r.engine.guidanceEnded)
        assertNotEquals(genBefore, r.engine.generation)

        // **والكواشفُ تعمل ثانيةً** — يُغذّى المسارُ الجديد.
        val f = RouteFixtures.driveAlong(r.engine.currentRoute!!).first()
        val st = r.engine.onFix(f)
        assertEquals(NavSituation.ON_ROUTE, st.situation)
        assertEquals(ArrivalPhase.EN_ROUTE, st.arrivalPhase)
    }

    // ══════════════════════════════════════════════════════════════
    // **ط و ي · ولا تكرارَ مهما تكرّرت القراءات**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `ط - قراءاتٌ كثيرةٌ بعد النهاية ونداءُ النهاية واحد`() {
        val r = Rig(150.0)
        r.drive()
        repeat(10) { r.walkTo(20.0 + it * 2.0, samples = 3) }
        assertEquals(1, r.routeEnds())
    }

    @Test
    fun `ي - وقراءاتٌ كثيرةٌ داخلَ حدّ الوصول ودعوى واحدة`() {
        val r = Rig(60.0)
        r.drive()
        repeat(12) { r.walkTo(60.0, samples = 3) }
        assertEquals(1, r.arrivals())
        assertEquals(1, r.claims())
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ١٦ — ولا شبكةَ في آخر ميل**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `دون اتّصال — آخرُ ميلٍ لا يطلب شيئاً من الشبكة`() {
        /**
         * **و`CountingSource` هي المنفذُ الوحيدُ إلى الشبكة** في
         * `NavEngine` — فصفرُ طلبٍ يعني صفرَ نداء.
         */
        val r = Rig(180.0)
        r.drive()
        for (m in listOf(30.0, 60.0, 90.0, 120.0, 150.0, 180.0)) {
            r.walkTo(m, samples = 5, speed = 10f)
        }
        assertEquals("طُلب من الشبكة في آخر ميل", 0, r.src.asked)
        assertEquals(ArrivalPhase.BUSINESS_TARGET_ARRIVED, r.engine.state?.arrivalPhase)
        assertEquals(1, r.arrivals())
        assertEquals(1, r.routeEnds())
    }
}
