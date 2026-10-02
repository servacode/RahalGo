package com.rahalgo.navigation

import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **كم ثانيةً من الخطأ حتّى يُطلب طريقٌ جديد** — طلبُ المالك ٢٠٢٦-١٠-٠٢: «إذا أخطأ السائقُ
 * أو تجاوز الطريق، الخريطةُ تعثر مباشرةً على طريقٍ جديد».
 *
 * **ينعطف تسعين درجةً عن مسارٍ مستقيم** بسرعةٍ ودقّةٍ معلومتين، ويُقاس الزمنُ من أوّل قراءةٍ
 * خارجَ الطريق حتّى أوّل طلب. (والشبكةُ نفسُها ١٥٠–٣٠٠ مل — قِيس.)
 */
class RerouteLatencyTest {

    private class StampSource : RouteSource {
        var firstAtMs = -1L
        var now = 0L
        override fun request(lat: Double, lng: Double, seq: Long, done: (RouteReply) -> Unit) {
            if (firstAtMs < 0) firstAtMs = now
            done(RouteReply.Failed(RerouteFailure.NETWORK))
        }
    }

    private fun secondsToReroute(speedMps: Float, accuracyM: Float): Double {
        val src = StampSource()
        val engine = NavEngine(source = src)
        val route = RouteFixtures.straight()
        engine.setRoute(route)
        val along = RouteFixtures.driveAlong(route, speedMps = speedMps, accuracyM = accuracyM)
        val turnAt = along.size / 3
        var t = 0L
        for (f in along.take(turnAt)) {
            t = f.atMs
            src.now = t
            engine.onFix(f)
        }
        val corner = along[turnAt - 1]
        val leaveMs = corner.atMs + 1_000L
        // **ثمّ شرقاً بزاوية قائمة** — قراءةٌ كلَّ ثانية.
        for (i in 1..60) {
            t = corner.atMs + i * 1_000L
            src.now = t
            engine.onFix(
                NavFix(
                    corner.lat, corner.lng + i * speedMps * 0.0000111,
                    accuracyM, speedMps, 90f, t,
                ),
            )
            if (src.firstAtMs >= 0) break
        }
        return if (src.firstAtMs < 0) -1.0 else (src.firstAtMs - leaveMs) / 1000.0
    }

    @Test
    fun `الانعطافُ الخاطئ يُطلب له طريقٌ خلال ثوانٍ`() {
        val rows = listOf(
            Triple("٣٠ كم/س دقّة ٦م", 8.3f, 6f),
            Triple("٢٠ كم/س دقّة ١٠م", 5.5f, 10f),
            Triple("٥٠ كم/س دقّة ٦م", 14f, 6f),
            Triple("٣٠ كم/س دقّة ٢٠م", 8.3f, 20f),
        )
        for ((label, v, acc) in rows) {
            val s = secondsToReroute(v, acc)
            println("REROUTE-LAT · $label ⇒ ${"%.0f".format(s)} ث (${"%.0f".format(s * v)} م بعد المنعطف)")
            assertTrue("$label: لم يُطلب طريقٌ في دقيقة", s >= 0)
        }
    }
}
