package com.rahalgo.driver.orders

import com.rahalgo.driver.trip.ArrivalPoint
import com.rahalgo.driver.trip.arrivalPoint
import com.rahalgo.driver.trip.autoArrivalTarget
import com.rahalgo.driver.trip.currentStop
import com.rahalgo.driver.trip.multiStopNext
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.PickupStop
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **طلبٌ من عدّة متاجر — جهةُ السائق** (٢٠٢٦-١١-٠٦): محطّةٌ لكلّ متجر، و«انطلقت» بعد الأخيرة وحدَها.
 */
class MultiStopTest {

    private fun stop(id: String, seq: Int, mine: Boolean = false, arrived: Boolean = false, status: String = "accepted") =
        PickupStop(
            merchantId = id,
            seq = seq,
            name = "متجر $id",
            lat = 35.9 + seq / 100.0,
            lng = 39.0,
            status = status,
            mine = mine,
            arrivedAt = if (arrived) "2026-11-06T10:00:00Z" else null,
        )

    private fun order(status: String = "assigned", vararg s: PickupStop, next: String? = "A") =
        DriverOrder(id = "o1", status = status, stops = s.toList(), nextStop = next, navLat = 35.91, navLng = 39.0)

    private val json = Json { ignoreUnknownKeys = true }

    // ══════════════════════════════════════════════════════════════════
    // **الطابور: المحطّةُ تُحفظ وتُقرأ**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `queue round trip keeps the stop`() {
        val q = listOf(
            PendingStep("o1", "at_pickup", "k1", stop = "A"),
            PendingStep("o1", "picked_up", "k2", 35.9, 39.0, "A"),
            PendingStep("o2", "picked_up", "k3"),
        )
        val back = StepQueue.decode(StepQueue.encode(q))
        assertEquals(q, back)
        assertNull(back[2].stop)
    }

    @Test
    fun `an old queue file without stop still decodes`() {
        val back = StepQueue.decode("""[{"order_id":"o1","to":"picked_up","key":"k"}]""")
        assertEquals(listOf(PendingStep("o1", "picked_up", "k")), back)
    }

    // ══════════════════════════════════════════════════════════════════
    // **العرضُ المحلّيّ: محطّةٌ غيرُ أخيرةٍ تُبقي الطلبَ at_pickup**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `non final pickup keeps at_pickup and marks only that stop`() {
        val o = order("at_pickup", stop("A", 1, arrived = true), stop("B", 2))
        val out = StepQueue.overlay(listOf(o), listOf(PendingStep("o1", "picked_up", "k", stop = "A"))).single()
        assertEquals("at_pickup", out.status)
        val a = out.stops!!.first { it.merchantId == "A" }
        val b = out.stops!!.first { it.merchantId == "B" }
        assertTrue(a.mine)
        assertEquals("picked_up", a.status)
        assertFalse(b.mine)
        // **والتاليةُ B، والحقولُ المسطّحةُ تتبعها.**
        assertEquals("B", out.nextStop)
        assertEquals("متجر B", out.merchantName)
    }

    @Test
    fun `final pickup moves the order on`() {
        val o = order("at_pickup", stop("A", 1, mine = true, status = "picked_up"), stop("B", 2, arrived = true), next = "B")
        val out = StepQueue.overlay(
            listOf(o),
            listOf(PendingStep("o1", "picked_up", "k", stop = "B"), PendingStep("o1", "on_the_way", "k2")),
        ).single()
        assertEquals("on_the_way", out.status)
        assertNull(out.nextStop)
    }

    @Test
    fun `arrival at a stop marks it and advances assigned to at_pickup`() {
        val o = order("assigned", stop("A", 1), stop("B", 2))
        val out = StepQueue.overlay(listOf(o), listOf(PendingStep("o1", "at_pickup", "k", stop = "A"))).single()
        assertEquals("at_pickup", out.status)
        assertNotNull(out.stops!!.first { it.merchantId == "A" }.arrivedAt)
        assertNull(out.stops!!.first { it.merchantId == "B" }.arrivedAt)
    }

    @Test
    fun `offline at A then B reaches on_the_way only after B`() {
        val o = order("assigned", stop("A", 1), stop("B", 2))
        val q = listOf(
            PendingStep("o1", "at_pickup", "k1", stop = "A"),
            PendingStep("o1", "picked_up", "k2", stop = "A"),
        )
        assertEquals("at_pickup", StepQueue.overlay(listOf(o), q).single().status)
        val q2 = q + listOf(
            PendingStep("o1", "at_pickup", "k3", stop = "B"),
            PendingStep("o1", "picked_up", "k4", stop = "B"),
            PendingStep("o1", "on_the_way", "k5"),
        )
        assertEquals("on_the_way", StepQueue.overlay(listOf(o), q2).single().status)
    }

    @Test
    fun `on_the_way is chained only after the final stop`() {
        val o = order("at_pickup", stop("A", 1, arrived = true), stop("B", 2))
        assertFalse(StepQueue.finalAfter(o, "A"))
        val afterA = StepQueue.overlay(listOf(o), listOf(PendingStep("o1", "picked_up", "k", stop = "A"))).single()
        assertTrue(StepQueue.finalAfter(afterA, "B"))
    }

    @Test
    fun `a removed stop does not block the final pickup`() {
        val o = order("at_pickup", stop("A", 1, arrived = true), stop("B", 2, status = "removed"))
        assertTrue(StepQueue.finalAfter(o, "A"))
    }

    @Test
    fun `single store orders overlay exactly as before`() {
        val o = DriverOrder(id = "o1", status = "assigned")
        val out = StepQueue.overlay(
            listOf(o),
            listOf(PendingStep("o1", "at_pickup", "k1"), PendingStep("o1", "picked_up", "k2")),
        ).single()
        assertEquals("on_the_way", out.status)
        assertNull(out.stops)
    }

    // ══════════════════════════════════════════════════════════════════
    // **الطريقُ والوصول: لكلّ محطّة**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `route leg switches from A to B`() {
        val atA = order("at_pickup", stop("A", 1, arrived = true), stop("B", 2))
        val atB = StepQueue.overlay(listOf(atA), listOf(PendingStep("o1", "picked_up", "k", stop = "A"))).single()
        assertEquals("o1:stop:A", routeLegOf(atA, null))
        assertEquals("o1:stop:B", routeLegOf(atB, null))
        assertNotEquals(routeLegOf(atA, null), routeLegOf(atB, null))
    }

    @Test
    fun `single store leg key is unchanged`() {
        assertEquals("o1:merchant", routeLegOf(DriverOrder(id = "o1", status = "assigned"), null))
    }

    @Test
    fun `auto arrival is per stop`() {
        // **المحطّةُ التالية لم يُسجَّل وصولُها** → تُعلَن.
        assertEquals("at_pickup", autoArrivalTarget("at_pickup", false, stopPending = true))
        assertEquals("at_pickup", autoArrivalTarget("assigned", false, stopPending = true))
        // **وبعد وصولها لا شيء حتّى تُستلم.**
        assertNull(autoArrivalTarget("at_pickup", false, stopPending = false))
        // **وغيرُ المتعدّد كما كان.**
        assertEquals("at_pickup", autoArrivalTarget("assigned", false))
        assertNull(autoArrivalTarget("at_pickup", false))
        assertEquals("at_dropoff", autoArrivalTarget("on_the_way", false, stopPending = false))
    }

    @Test
    fun `arrival point targets the next uncollected stop while at_pickup`() {
        val o = order("at_pickup", stop("A", 1, mine = true, status = "picked_up"), stop("B", 2), next = "B")
            .copy(navLat = 35.92, navLng = 39.0)
        assertEquals(ArrivalPoint.At(35.92, 39.0), arrivalPoint(o))
        val arrived = order("at_pickup", stop("A", 1, mine = true, status = "picked_up"), stop("B", 2, arrived = true), next = "B")
        assertEquals(ArrivalPoint.NotArriving, arrivalPoint(arrived))
        assertEquals(ArrivalPoint.NotArriving, arrivalPoint(DriverOrder(id = "x", status = "at_pickup")))
    }

    @Test
    fun `next action follows the current stop`() {
        val fresh = order("assigned", stop("A", 1), stop("B", 2))
        assertEquals("at_pickup", multiStopNext(fresh)?.status)
        val arrived = order("at_pickup", stop("A", 1, arrived = true), stop("B", 2))
        assertEquals("picked_up", multiStopNext(arrived)?.status)
        // **وبعد الاستلام الأخير الفعلُ العاديّ (في الطريق للزبون).**
        assertNull(multiStopNext(order("picked_up", stop("A", 1, mine = true), stop("B", 2, mine = true), next = null)))
        assertNull(multiStopNext(DriverOrder(id = "x", status = "assigned")))
    }

    @Test
    fun `current stop skips removed and collected ones and a handover stays pending`() {
        val o = order(
            "at_pickup",
            stop("A", 1, status = "removed"),
            stop("B", 2, mine = true, status = "picked_up"),
            stop("C", 3, status = "picked_up").copy(handover = true),
            next = null,
        )
        assertEquals("C", currentStop(o)?.merchantId)
    }

    // ══════════════════════════════════════════════════════════════════
    // **النموذج: بـ`stops` وبدونها**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `parses a multi stop order`() {
        val text = """{"id":"o1","status":"assigned","next_stop":"A","stops":[
          {"merchant_id":"A","seq":1,"name":"مطعم","phone":"0911","address":"x","lat":35.9,"lng":39.0,
           "status":"accepted","ready":true,"ready_at":null,"prep_minutes":15,"arrived_at":null,"picked_up_at":null,
           "mine":false,"handover":false,"items":[{"id":"i1","name":"برغر","qty":2}],"pay_amount":12000,"pays_by":"cash"},
          {"merchant_id":"B","seq":2,"name":"صيدلية","status":"picked_up","mine":false,"handover":true,"pay_amount":0,"pays_by":"wallet"}]}"""
        val o = json.decodeFromString(DriverOrder.serializer(), text)
        assertEquals("A", o.nextStop)
        assertEquals(2, o.stops!!.size)
        assertEquals(12000L, o.stops!![0].payAmount)
        assertEquals(2, o.stops!![0].items.single().qty)
        assertTrue(o.stops!![1].handover)
        assertEquals("wallet", o.stops!![1].paysBy)
    }

    @Test
    fun `parses an order without stops as today`() {
        val o = json.decodeFromString(DriverOrder.serializer(), """{"id":"o1","status":"assigned","merchant_name":"م"}""")
        assertNull(o.stops)
        assertNull(o.nextStop)
    }
}
