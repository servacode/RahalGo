package com.rahalgo.driver.trip

import com.rahalgo.shared.model.CustomerLive
import com.rahalgo.shared.model.DriverOrder
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **إلى أين يُقاد بعد الاستلام — الدبّوسُ أم الزبونُ الآن** (قرارُ المالك ٢٠٢٦-١٠-١٦).
 *
 * **بعيدٌ عن الدبّوس وحديثٌ ⇒ إليه** · **قريبٌ ⇒ الدبّوس** · **شائخٌ ⇒ الدبّوس بكلمة**
 * · **ولا وقتَ له ⇒ لا يُصدَّق.**
 */
class DoorTargetTest {

    private val now = java.time.OffsetDateTime.parse("2026-10-16T12:00:00+03:00").toInstant().toEpochMilli()

    private fun live(ageSec: Long, gapM: Double, switchM: Long = 150) = CustomerLive(
        lat = 35.956, lng = 39.008, accuracyM = 10.0,
        at = java.time.Instant.ofEpochMilli(now - ageSec * 1000).atOffset(java.time.ZoneOffset.ofHours(3)).toString(),
        pinGapM = gapM, switchM = switchM,
    )

    @Test
    fun `far from the pin and fresh switches the route to the customer`() {
        val t = doorTarget(live(ageSec = 20, gapM = 320.0), hadLive = false, nowMs = now)
        assertTrue(t is DoorTarget.Live)
        t as DoorTarget.Live
        assertEquals(35.956, t.lat, 1e-9)
        assertEquals(20L, t.ageSec)
    }

    @Test
    fun `the threshold comes from the dashboard setting`() {
        assertTrue(doorTarget(live(10, gapM = 320.0, switchM = 500), false, now) is DoorTarget.Pin)
        assertTrue(doorTarget(live(10, gapM = 320.0, switchM = 300), false, now) is DoorTarget.Live)
    }

    @Test
    fun `near the pin keeps the pin without a note`() {
        assertEquals(DoorTarget.Pin(staleLive = false), doorTarget(live(5, gapM = 60.0), hadLive = false, nowMs = now))
        // **ولو كان يُقاد إليه فعاد قريباً من دبّوسه** — لا «شاخ»: الموقعُ حديث.
        assertEquals(DoorTarget.Pin(staleLive = false), doorTarget(live(5, gapM = 60.0), hadLive = true, nowMs = now))
    }

    @Test
    fun `stale over two minutes falls back to the pin with a note`() {
        assertEquals(DoorTarget.Pin(staleLive = true), doorTarget(live(121, gapM = 900.0), hadLive = true, nowMs = now))
        // **وعند الحدّ نفسِه ما زال حديثاً.**
        assertTrue(doorTarget(live(120, gapM = 900.0), hadLive = true, nowMs = now) is DoorTarget.Live)
        // **ومن لم يُقَد إليه قطّ لا يُشغَل بكلمة.**
        assertEquals(DoorTarget.Pin(staleLive = false), doorTarget(live(300, gapM = 900.0), hadLive = false, nowMs = now))
    }

    @Test
    fun `a vanished point after switching returns to the pin with a note`() {
        assertEquals(DoorTarget.Pin(staleLive = true), doorTarget(null, hadLive = true, nowMs = now))
        assertEquals(DoorTarget.Pin(staleLive = false), doorTarget(null, hadLive = false, nowMs = now))
    }

    @Test
    fun `a point without a readable time is never trusted`() {
        val bad = live(0, gapM = 900.0).copy(at = "")
        assertEquals(DoorTarget.Pin(staleLive = true), doorTarget(bad, hadLive = true, nowMs = now))
        assertEquals(null, liveAgeSec(bad, now))
    }

    @Test
    fun `arrival is measured at the customer when the route switched`() {
        val o = DriverOrder(status = "on_the_way", lat = 35.9528, lng = 39.0079, dropoffKnown = true)
        assertEquals(ArrivalPoint.At(35.9528, 39.0079), arrivalPoint(o))
        assertEquals(ArrivalPoint.At(35.956, 39.008), arrivalPoint(o, 35.956 to 39.008))
        // **وبابٌ لا يُعرف يبقى «لا يُعرف»** — ولو وصل موقعٌ مباشر.
        assertEquals(ArrivalPoint.Unknown, arrivalPoint(o.copy(dropoffKnown = false), 35.956 to 39.008))
    }
}
