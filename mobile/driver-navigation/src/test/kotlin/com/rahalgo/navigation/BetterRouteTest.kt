package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/** **طريقٌ أفضل أثناء السير** — طلبُ المالك ٢٠٢٦-١٠-٠٢. */
class BetterRouteTest {

    private fun opt(route: NavRoute, sec: Double) =
        RouteOption("r" + sec.toInt(), route, engineDurationS = sec, distanceM = route.totalM)

    /** **طريقٌ يبدأ من موضعه على المستقيم ثمّ ينحرف شرقاً.** */
    private fun branch(fromNorthM: Double, eastM: Double): NavRoute {
        val pts = listOf(
            GeoPoint(RouteFixtures.north(fromNorthM), RouteFixtures.LNG0),
            GeoPoint(RouteFixtures.north(fromNorthM + 100.0), RouteFixtures.LNG0),
            GeoPoint(RouteFixtures.north(fromNorthM + 100.0), RouteFixtures.LNG0 + eastM * 0.0000111),
        )
        var acc = 0.0
        val cum = DoubleArray(pts.size)
        for (i in 1 until pts.size) {
            acc += GpsQuality.metersBetween(pts[i - 1].lat, pts[i - 1].lng, pts[i].lat, pts[i].lng)
            cum[i] = acc
        }
        return NavRoute(pts, cum, emptyList())
    }

    @Test
    fun `الأسرع بفارق كاف يعرض بنقطة افتراقه على طريقه`() {
        val current = RouteFixtures.straight()
        val better = branch(200.0, 300.0)
        val out = BetterRoute.evaluate(current, 200.0, 300.0, listOf(opt(better, 200.0)))
        assertEquals(1, out.size)
        assertEquals(-100.0, out[0].deltaDurationS, 0.01)
        // **يفترق بعد مئة مترٍ من موضعه** ⇒ على طريقه عند ٣٠٠م تقريباً.
        assertTrue("الافتراق=${out[0].decisionDivergenceM}", out[0].decisionDivergenceM in 290.0..320.0)
    }

    @Test
    fun `الفرق الصغير لا يعرض`() {
        val current = RouteFixtures.straight()
        val out = BetterRoute.evaluate(current, 200.0, 300.0, listOf(opt(branch(200.0, 300.0), 285.0)))
        assertTrue(out.isEmpty())
    }

    @Test
    fun `ما يطابق طريقه ليس بديلا`() {
        val current = RouteFixtures.straight()
        val same = branch(200.0, 0.0)
        val out = BetterRoute.evaluate(current, 200.0, 300.0, listOf(opt(same, 100.0)))
        assertTrue(out.isEmpty())
    }
}
