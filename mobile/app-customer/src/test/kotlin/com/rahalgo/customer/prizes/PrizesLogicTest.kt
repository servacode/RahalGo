package com.rahalgo.customer.prizes

import com.rahalgo.shared.customer.PrizeItem
import com.rahalgo.shared.customer.PrizesMe
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **جوائزُ رحّال غو — القرارُ الخالص** (قرارُ المالك ٢٠٢٦-١١-٢٣): متى اللافتة، وكيف تُقسم
 * الأصنافُ على الرصيد، وما الفرقُ الذي يدفعه، وما تقوله الشارة.
 */
class PrizesLogicTest {

    private fun item(id: String, price: Long) = PrizeItem(id = id, name = id, price = price)

    private val items = listOf(item("c", 1200), item("a", 500), item("b", 750), item("d", 2000))

    @Test
    fun `teaser while disabled or before launch day and open from the day itself`() {
        assertTrue(PrizesLogic.isTeaser(false, "2026-12-01", "2026-12-05"))
        assertTrue(PrizesLogic.isTeaser(true, "2026-12-01", "2026-11-30"))
        assertFalse(PrizesLogic.isTeaser(true, "2026-12-01", "2026-12-01"))
        assertFalse(PrizesLogic.isTeaser(true, "2026-12-01", "2027-01-02"))
        assertFalse(PrizesLogic.isTeaser(true, "", "2026-11-01"))
    }

    @Test
    fun `split puts items within the balance first and the rest over it`() {
        val s = PrizesLogic.split(items, 750)
        assertEquals(listOf("a", "b"), s.within.map { it.id })
        assertEquals(listOf("c", "d"), s.over.map { it.id })
        assertFalse(s.showcase)
    }

    @Test
    fun `zero balance is a showcase with nothing within`() {
        val s = PrizesLogic.split(items, 0)
        assertTrue(s.showcase)
        assertTrue(s.within.isEmpty())
        assertEquals(listOf("a", "b", "c", "d"), s.over.map { it.id })
    }

    @Test
    fun `an item priced exactly at the balance is within`() {
        assertEquals(listOf("a"), PrizesLogic.split(items, 500).within.map { it.id })
    }

    @Test
    fun `difference is the price above the balance and never negative`() {
        assertEquals(450L, PrizesLogic.difference(1200, 750))
        assertEquals(0L, PrizesLogic.difference(500, 750))
        assertEquals(0L, PrizesLogic.difference(750, 750))
        assertEquals(500L, PrizesLogic.difference(500, -10))
    }

    @Test
    fun `progress counts invites toward the next entry`() {
        val p = PrizesLogic.progress(PrizesMe(monthInvites = 8, invitesPerEntry = 5, invitesToNext = 2))
        assertEquals(3, p.current)
        assertEquals(5, p.perEntry)
        assertEquals(2, p.toNext)
    }

    @Test
    fun `progress computes the remainder when the engine omits it and survives a zero rate`() {
        val p = PrizesLogic.progress(PrizesMe(monthInvites = 1, invitesPerEntry = 0))
        assertEquals(1, p.current)
        assertEquals(5, p.perEntry)
        assertEquals(4, p.toNext)
    }

    @Test
    fun `badge reads current over rate and hides before launch`() {
        val me = PrizesMe(enabled = true, monthInvites = 3, invitesPerEntry = 5)
        assertEquals("3/5", PrizesLogic.badge(me, teaser = false))
        assertNull(PrizesLogic.badge(me, teaser = true))
        assertNull(PrizesLogic.badge(null, teaser = false))
    }

    @Test
    fun `pulse only when the counter really changes`() {
        assertFalse(PrizesLogic.shouldPulse(null, "3/5"))
        assertFalse(PrizesLogic.shouldPulse("3/5", "3/5"))
        assertTrue(PrizesLogic.shouldPulse("3/5", "4/5"))
        assertFalse(PrizesLogic.shouldPulse("3/5", null))
    }

    @Test
    fun `launch and draw dates are written for people`() {
        assertEquals("1/12/2026", PrizesLogic.launchLabel("2026-12-01"))
        assertEquals("x", PrizesLogic.launchLabel("x"))
        // 12:00 دمشق = 09:00 UTC (توقيتٌ دائمٌ +03).
        assertEquals("1/12/2026 12:00", PrizesLogic.drawLabel("2026-12-01T09:00:00Z"))
        assertEquals("", PrizesLogic.drawLabel("not a date"))
        assertEquals("30/11/2026", PrizesLogic.expiryLabel("2026-11-30T09:00:00Z"))
    }

    @Test
    fun `ladder numbers the amounts from first place`() {
        assertEquals(listOf(1 to 1500L, 2 to 1000L), PrizesLogic.ladder(listOf(1500, 1000)))
    }
}
