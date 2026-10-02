package com.rahalgo.driver.location

import com.rahalgo.ui.LastPoint
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * **نبضةُ الواقف لا تُعيد موضعاً ماتت أقمارُه** (٢٠٢٦-١٠-٠٢).
 *
 * **كانت تُعيد آخرَ موضعٍ بلا حدّ عمر** — فسائقٌ مات GPS هاتفه يبقى «حديثاً»
 * ويُعرض عليه أقربُ طلب.
 */
class HeartbeatTest {

    private val now = 10_000_000L
    private fun point(ageMs: Long, mocked: Boolean = false) =
        LastPoint.Point(35.95, 39.01, mocked = mocked, atMs = now - ageMs)

    @Test
    fun `a short silence waits`() {
        assertEquals(Heartbeat.Act.WAIT, Heartbeat.decide(now, now - 60_000, point(10_000)))
    }

    @Test
    fun `a live stationary point is re-sent`() {
        assertEquals(Heartbeat.Act.RESEND, Heartbeat.decide(now, now - 130_000, point(200_000)))
    }

    @Test
    fun `a point older than five minutes is never re-sent`() {
        assertEquals(Heartbeat.Act.REFRESH, Heartbeat.decide(now, now - 130_000, point(301_000)))
        assertEquals(Heartbeat.Act.REFRESH, Heartbeat.decide(now, now - 130_000, point(3_600_000)))
    }

    @Test
    fun `no point or no capture time asks the system`() {
        assertEquals(Heartbeat.Act.REFRESH, Heartbeat.decide(now, 0L, null))
        assertEquals(
            Heartbeat.Act.REFRESH,
            Heartbeat.decide(now, now - 130_000, LastPoint.Point(35.95, 39.01, atMs = 0L)),
        )
    }

    @Test
    fun `a mocked point is not laundered`() {
        assertEquals(Heartbeat.Act.WAIT, Heartbeat.decide(now, now - 130_000, point(10_000, mocked = true)))
    }
}
