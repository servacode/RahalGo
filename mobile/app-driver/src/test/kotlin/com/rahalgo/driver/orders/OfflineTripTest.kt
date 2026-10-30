package com.rahalgo.driver.orders

import com.rahalgo.shared.model.DriverMe
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.OrderRoute
import com.rahalgo.shared.model.RouteManeuver
import java.io.IOException
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **الرحلةُ تكمّل بلا إنترنت** (قرارُ المالك ٢٠٢٦-١٠-٣٠).
 */
class OfflineTripTest {

    private val route = OrderRoute(
        available = true,
        distanceM = 1800.0,
        durationS = 300.0,
        points = listOf(listOf(35.95, 39.0), listOf(35.96, 39.01)),
        cumulativeM = listOf(0.0, 1800.0),
        maneuvers = listOf(RouteManeuver(kind = "DEPART"), RouteManeuver(kind = "ARRIVE", atIndex = 1)),
        routeId = "r-1",
    )

    // ══════════════════════════════════════════════════════════════════
    // **١ · دورةٌ فشلت لا تمسح الطريق** — كان يُمحى بعد ١٥ ثانية من الانقطاع
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `a failed refresh keeps the route`() {
        // **ما تردّه `loadOnce` بلا شبكة** — قائمةٌ فارغةٌ وخطأ؛ وساقُ القائمة الفارغة فارغة.
        val offline = LoadOutcome(emptyList(), emptyList(), null, null, IOException("لا شبكة"))
        val k = routeKeep(offline, leg = null, heldLeg = "o1:merchant", held = route)
        assertEquals(RouteKeep.Hold, k)
    }

    @Test
    fun `a failed refresh through the real loader keeps the route`() = runBlocking {
        val feed = object : OrdersFeed {
            override suspend fun queue(): List<DriverOrder> = throw IOException("لا شبكة")
            override suspend fun orders(): List<DriverOrder> = throw IOException("لا شبكة")
            override suspend fun me(): DriverMe = throw IOException("لا شبكة")
            override suspend fun route(orderId: String): OrderRoute = throw IOException("لا شبكة")
        }
        val out = loadOnce(feed, "o1")
        assertNotNull(out.error)
        assertEquals(RouteKeep.Hold, routeKeep(out, null, "o1:customer", route))
    }

    @Test
    fun `a successful refresh on a new leg still swaps the route`() {
        val ok = LoadOutcome(emptyList(), listOf(DriverOrder(id = "o1")), DriverMe(), route)
        assertEquals(RouteKeep.Swap(route, "o1:customer"), routeKeep(ok, "o1:customer", "o1:merchant", null))
    }

    // ══════════════════════════════════════════════════════════════════
    // **٢ · الرحلةُ المحفوظةُ تُقرأ كما كُتبت**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `saved trip round trip`() {
        val trip = SavedTrip(
            mine = listOf(
                DriverOrder(id = "o1", number = 1219, status = "on_the_way", kind = "standard", navLat = 35.9, navLng = 39.1),
                DriverOrder(id = "o2", number = 1220, status = "assigned"),
            ),
            route = route,
            leg = "o1:customer",
            openId = "o1",
            savedAtMs = 1_000L,
        )
        val back = TripStore.decode(TripStore.encode(trip))
        assertEquals(trip, back)
    }

    @Test
    fun `saved trip without route round trip`() {
        val trip = SavedTrip(listOf(DriverOrder(id = "o1")), null, null, null, 5L)
        assertEquals(trip, TripStore.decode(TripStore.encode(trip)))
    }

    @Test
    fun `stale or broken saved trip is ignored`() {
        val dir = kotlin.io.path.createTempDirectory().toFile()
        try {
            TripStore.write(dir, TripStore.encode(SavedTrip(listOf(DriverOrder(id = "o1")), null, null, null, 0L)))
            assertNull(TripStore.read(dir, TripStore.MAX_AGE_MS + 1))
            assertNotNull(TripStore.read(dir, 10L))
            java.io.File(dir, TripStore.FILE).writeText("{")
            assertNull(TripStore.read(dir, 10L))
        } finally {
            dir.deleteRecursively()
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **٣ · طابورُ الخطوات**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `queue round trip`() {
        val q = listOf(
            PendingStep("o1", "picked_up", "k1"),
            PendingStep("o1", "on_the_way", "k2"),
            PendingStep("o2", StepQueue.GOODS, "k3", 35.9, 39.0),
        )
        assertEquals(q, StepQueue.decode(StepQueue.encode(q)))
    }

    @Test
    fun `delivered is never queued`() {
        assertTrue("delivered" !in StepQueue.QUEUEABLE)
        assertEquals(emptyList<PendingStep>(), StepQueue.decode("""[{"order_id":"o1","to":"delivered","key":"k"}]"""))
    }

    @Test
    fun `replay sends in order with the same keys`() = runBlocking {
        val sent = mutableListOf<String>()
        val q = listOf(PendingStep("o1", "at_pickup", "k1"), PendingStep("o1", "picked_up", "k2"), PendingStep("o1", "on_the_way", "k3"))
        val r = StepQueue.replay(q) { sent += it.to + ":" + it.key }
        assertEquals(listOf("at_pickup:k1", "picked_up:k2", "on_the_way:k3"), sent)
        assertTrue(r.remaining.isEmpty())
        assertTrue(r.rejected.isEmpty())
    }

    @Test
    fun `replay stops on network loss and keeps the rest`() = runBlocking {
        val q = listOf(PendingStep("o1", "at_pickup", "k1"), PendingStep("o1", "picked_up", "k2"))
        val r = StepQueue.replay(q) { if (it.key == "k2") throw IOException("انقطعت") }
        assertEquals(listOf(q[1]), r.remaining)
    }

    @Test
    fun `server rejection drops that order's later steps and is reported`() = runBlocking {
        val q = listOf(
            PendingStep("o1", "at_pickup", "k1"),
            PendingStep("o1", "picked_up", "k2"),
            PendingStep("o2", "at_dropoff", "k3"),
        )
        val sent = mutableListOf<String>()
        val r = StepQueue.replay(q) {
            sent += it.key
            if (it.key == "k1") throw IllegalStateException("invalid_transition")
        }
        assertEquals(listOf("k1", "k3"), sent)
        assertEquals(1, r.rejected.size)
        assertTrue(r.remaining.isEmpty())
    }

    @Test
    fun `overlay advances statuses and never goes back`() {
        val mine = listOf(
            DriverOrder(id = "o1", status = "assigned"),
            DriverOrder(id = "o2", status = "at_dropoff"),
            DriverOrder(id = "o3", status = "failed"),
        )
        val q = listOf(
            PendingStep("o1", "at_pickup", "k1"),
            PendingStep("o1", "picked_up", "k2"),
            PendingStep("o2", "at_pickup", "k3"),
            PendingStep("o3", StepQueue.GOODS, "k4"),
        )
        val out = StepQueue.overlay(mine, q)
        assertEquals(listOf("o1" to "on_the_way", "o2" to "at_dropoff"), out.map { it.id to it.status })
    }
}
