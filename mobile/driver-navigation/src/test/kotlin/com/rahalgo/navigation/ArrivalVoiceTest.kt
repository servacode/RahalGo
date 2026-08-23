package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **من القراءة إلى الجملة — الرفائدُ هـ و ط و ي و ك**
 * ══════════════════════════════════════════════════════════════════
 *
 * **والحارسُ وحدَه لا يكفي دليلاً**: لا بدّ أن يُرى **ما يُنطق فعلاً**
 * — من `NavEngine` إلى `VoicePlanner` كما تعمل في الجهاز.
 *
 * (إغلاقُ دلالات الوصول، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 */
class ArrivalVoiceTest {

    private fun engineWithRoute(
        target: GeoPoint?,
        trip: TripTarget = TripTarget.DROPOFF,
    ): Pair<NavEngine, NavRoute> {
        val planner = VoicePlanner()
        planner.target = trip
        val engine = NavEngine(voice = planner)
        val route = RouteFixtures.straight()
        engine.setRoute(route)
        engine.arrivalTarget = target
        return engine to route
    }

    /** **يقود المسارَ كلَّه ثمّ يقف عند نهايته.** */
    private fun driveToEnd(engine: NavEngine, route: NavRoute): MutableList<VoiceCue> {
        val said = mutableListOf<VoiceCue>()
        for (f in RouteFixtures.driveAlong(route)) said += engine.onFix(f).cues
        val last = route.geometry.last()
        repeat(8) { i ->
            said += engine.onFix(NavFix(last.lat, last.lng, 5f, 0f, 0f, 100_000L + i * 1000L)).cues
        }
        return said
    }

    /** **دعوى الوصول** — لا مناورةُ الوصول. */
    private fun arrivals(said: List<VoiceCue>) =
        said.filter { it.kind == CueKind.ARRIVE && it.stage == CueStage.EVENT }

    /** **كلُّ ما نُطق فيه ادّعاءُ وصول** — أيّاً كان مصدرُه. */
    private fun claimsArrival(said: List<VoiceCue>) =
        said.filter { it.text.contains("وصلت") }

    private fun endPlus(offsetM: Double): GeoPoint {
        val end = RouteFixtures.straight().geometry.last()
        return GeoPoint(end.lat + offsetM / 111320.0, end.lng)
    }

    // ══════════════════════════════════════════════════════════════
    // **أ · الهدفُ عند نهاية الخطّ — والبند ١١: مرّةً واحدة**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `أ - الهدفُ على خمسة أمتارٍ فنداءٌ واحدٌ لا اثنان`() {
        /**
         * **أمرُ المالك (البند ١١) نصّاً**: «يجب أن يسمع السائق Final
         * Arrival Cue مرة واحدة فقط».
         *
         * **وهذه هي الحالةُ التي كانت تُنتج ندائين**: مناورةُ `ARRIVE`
         * تقول «لقد وصلت» **ثمّ** الدعوى تقول «وصلت إلى وجهة التوصيل».
         */
        val (engine, route) = engineWithRoute(endPlus(5.0))
        val said = driveToEnd(engine, route)

        assertEquals("دعوى الوصول: ${said.map { it.text }}", 1, arrivals(said).size)
        assertEquals(
            "نُطق ادّعاءُ وصولٍ أكثرَ من مرّة: ${claimsArrival(said).map { it.text }}",
            1, claimsArrival(said).size,
        )
        assertTrue(arrivals(said).first().text.contains("وجهة التوصيل"))
    }

    // ══════════════════════════════════════════════════════════════
    // **هـ · انتهى الخطُّ بعيداً — والبند ٨**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `هـ - مئةٌ وثمانون فلا لقد وصلت ولا دعوى`() {
        /**
         * **أمرُ المالك (البند ٨)**: «لا أقبل أن تقول مناورة ARRIVE
         * "لقد وصلت" عندما business target لم تُبلغ».
         */
        val (engine, route) = engineWithRoute(endPlus(180.0))
        val said = driveToEnd(engine, route)

        assertTrue(
            "نُطق ادّعاءُ وصولٍ كاذب: ${claimsArrival(said).map { it.text }}",
            claimsArrival(said).isEmpty(),
        )
        // **والملاحةُ لم تُكسَر** — الإرشادُ عمل كما يعمل.
        assertTrue("لم يُنطق شيءٌ أصلاً — فالرفيدةُ لا تقيس", said.isNotEmpty())
        // **والهندسةُ تقول: بلغتُ نهايةَ الخطّ** — ولم تتبدّل دلالتُها.
        assertTrue(engine.state?.progress?.arrived == true)
        assertEquals(ArrivalPhase.LAST_MILE_TO_TARGET, engine.state?.arrivalPhase)
    }

    @Test
    fun `و - ثمّ يقترب فتُقال مرّةً واحدة`() {
        val (engine, route) = engineWithRoute(endPlus(180.0))
        val said = driveToEnd(engine, route)
        assertTrue(claimsArrival(said).isEmpty())

        val t = engine.arrivalTarget!!
        repeat(6) { i ->
            said += engine.onFix(NavFix(t.lat, t.lng, 5f, 0f, 0f, 200_000L + i * 1000L)).cues
        }
        assertEquals(
            "الدعوى: ${said.map { it.text }}", 1, arrivals(said).size,
        )
        assertEquals(1, claimsArrival(said).size)
    }

    // ══════════════════════════════════════════════════════════════
    // **ي و ك · وجهةُ المتجر ووجهةُ الزبون — البند ١٢**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `ي - لفظُ الاستلام`() {
        val (engine, route) = engineWithRoute(endPlus(3.0), TripTarget.PICKUP)
        val said = driveToEnd(engine, route)
        assertEquals(1, arrivals(said).size)
        assertTrue(
            arrivals(said).first().text,
            arrivals(said).first().text.contains("نقطة الاستلام"),
        )
        assertEquals(1, claimsArrival(said).size)
    }

    @Test
    fun `ك - ولفظُ التوصيل — والمبدأُ واحد`() {
        /**
         * **البند ١٢**: «ولا تجعل واحدة strict والأخرى مبنية على
         * route endpoint».
         */
        val (engine, route) = engineWithRoute(endPlus(180.0), TripTarget.PICKUP)
        val said = driveToEnd(engine, route)
        assertTrue(
            "الاستلامُ ادّعى وصولاً عند نهاية الخطّ: ${claimsArrival(said).map { it.text }}",
            claimsArrival(said).isEmpty(),
        )
    }

    // ══════════════════════════════════════════════════════════════
    // **و · جودةُ القراءة**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `قراءةٌ مرفوضةٌ عند الهدف لا تُعلن`() {
        val (engine, route) = engineWithRoute(endPlus(3.0))
        val said = mutableListOf<VoiceCue>()
        for (f in RouteFixtures.driveAlong(route)) said += engine.onFix(f).cues
        if (claimsArrival(said).isNotEmpty()) return
        val t = engine.arrivalTarget!!
        repeat(8) { i ->
            said += engine.onFix(NavFix(t.lat, t.lng, 200f, 0f, 0f, 300_000L + i * 1000L)).cues
        }
        assertTrue(
            "قراءةٌ مرفوضةٌ أعلنت وصولاً: ${claimsArrival(said).map { it.text }}",
            claimsArrival(said).isEmpty(),
        )
    }

    // ══════════════════════════════════════════════════════════════
    // **ولا يُمسح الهدفُ بإعادة الحساب**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `الهدفُ يبقى بعد إعادةِ الحساب`() {
        val (engine, route) = engineWithRoute(endPlus(180.0))
        engine.setRoute(route)
        assertEquals(endPlus(180.0), engine.arrivalTarget)
    }

    // ══════════════════════════════════════════════════════════════
    // **ومن لا هدفَ له لا تنكسر ملاحتُه**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `بلا هدفٍ تبقى مناورةُ الوصول تُنطق`() {
        val (engine, route) = engineWithRoute(null)
        val said = driveToEnd(engine, route)
        assertTrue(
            "كُتمت مناورةُ الوصول بلا سبب: ${said.map { it.text }}",
            said.any { it.text.contains("وصلت") },
        )
    }
}
