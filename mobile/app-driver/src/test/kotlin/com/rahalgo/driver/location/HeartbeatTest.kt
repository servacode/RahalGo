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

    // ══════════════════════════════════════════════════════════════════
    // **موعدُ النبضة من آخر إرسال** (A6.5، 0.9.0)
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `the beat wakes when the silence reaches two minutes`() {
        // **أُرسل قبل مئة ثانية** — تبقى عشرون.
        assertEquals(20_000L, Heartbeat.waitMs(now, now - 100_000, 0L))
        // **أُرسل للتوّ** — دقيقتان كاملتان.
        assertEquals(Heartbeat.EVERY_MS, Heartbeat.waitMs(now, now, 0L))
        // **فات الموعد** — تصحو فوراً (لا تنام دقيقتين أخريين).
        assertEquals(Heartbeat.MIN_WAIT_MS, Heartbeat.waitMs(now, now - 130_000, 0L))
    }

    @Test
    fun `a beat that found nothing does not retry every second`() {
        // **صمتٌ طويلٌ ونبضةٌ للتوّ لم تجد موضعاً حيّاً** — المحاولةُ التالية بعد دقيقتين.
        assertEquals(Heartbeat.EVERY_MS, Heartbeat.waitMs(now, now - 600_000, now))
        // **وأوّلُ نبضةٍ بعد بدء الخدمة** — دقيقتان من البدء كما كان.
        assertEquals(Heartbeat.EVERY_MS, Heartbeat.waitMs(now, 0L, now))
    }

    /**
     * **حلقةُ الخدمة على ساعةٍ مزيّفة** — كما في `LocationService.heartbeat`.
     *
     * **الواقفُ يُسمع بعد آخر نقطةٍ بدقيقتين بالضبط** — كانت النبضةُ تنام دقيقتين على دورتها
     * هي، فنقطةٌ وصلت بعد بدء الحلقة بخمسين ثانيةً تُسمع بعد ثلاثِ دقائقَ وعشر.
     */
    @Test
    fun `a parked phone speaks two minutes after its last fix`() {
        val start = 1_000_000L
        val lastFix = start + 50_000 // **آخرُ نقطةٍ بعد بدء الحلقة بخمسين ثانية.**
        var lastSent = lastFix
        var lastBeat = start
        var clock = start
        val resends = mutableListOf<Long>()
        while (resends.size < 2) {
            clock += Heartbeat.waitMs(clock, lastSent, lastBeat)
            if (Heartbeat.due(clock, lastSent)) lastBeat = clock
            val p = LastPoint.Point(35.95, 39.01, atMs = clock - 1_000)
            if (Heartbeat.decide(clock, lastSent, p) == Heartbeat.Act.RESEND) {
                resends += clock
                lastSent = clock
            }
        }
        assertEquals(lastFix + Heartbeat.EVERY_MS, resends[0])
        assertEquals(resends[0] + Heartbeat.EVERY_MS, resends[1])
    }
}
