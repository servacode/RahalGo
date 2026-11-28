package com.rahalgo.customer.live

import com.rahalgo.shared.customer.PartyView
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/** **«الطرف الآخر يشارك موقعَه»** (قرارُ المالك ٢٠٢٦-١١-٢٨) — قراراتٌ صافية. */
class PartyPolicyTest {

    private fun p(id: String, answer: String = "", open: Boolean = true, kind: String = "amanah") =
        PartyView(orderId = id, kind = kind, answer = answer, open = open)

    @Test
    fun `banner shows the first open unanswered order only`() {
        assertEquals("b", PartyPolicy.bannerTarget(listOf(p("a", "yes"), p("b"), p("c")))?.orderId)
        assertNull(PartyPolicy.bannerTarget(listOf(p("a", "no"), p("b", open = false))))
        assertNull(PartyPolicy.bannerTarget(emptyList()))
    }

    @Test
    fun `prompt opens only for an open unanswered order that matches`() {
        val list = listOf(p("a"), p("b", "yes"), p("c", open = false))
        assertEquals("a", PartyPolicy.promptTarget(list, "a")?.orderId)
        assertNull(PartyPolicy.promptTarget(list, "b"))
        assertNull(PartyPolicy.promptTarget(list, "c"))
        assertNull(PartyPolicy.promptTarget(list, "zzz"))
        assertNull(PartyPolicy.promptTarget(list, null))
    }

    @Test
    fun `resume picks an open yes order with permission and no manual stop`() {
        val list = listOf(p("a", "no"), p("b", "yes"), p("c", "yes"))
        assertEquals("b", PartyPolicy.resumeTarget(list, emptyList(), null, true)?.orderId)
        // أوقفها بيده فتبقى موقوفةً ويُنتقل لغيرها
        assertEquals("c", PartyPolicy.resumeTarget(list, listOf(PartyPolicy.stopKey("b")), null, true)?.orderId)
    }

    @Test
    fun `resume never fires without permission or when already sharing or closed`() {
        val list = listOf(p("b", "yes"))
        assertNull(PartyPolicy.resumeTarget(list, emptyList(), null, false))
        assertNull(PartyPolicy.resumeTarget(list, emptyList(), "b", true))
        assertNull(PartyPolicy.resumeTarget(listOf(p("b", "yes", open = false)), emptyList(), null, true))
        assertNull(PartyPolicy.resumeTarget(listOf(p("b", "")), emptyList(), null, true))
    }

    @Test
    fun `owner stop keys never collide with party stop keys`() {
        assertEquals("party:x", PartyPolicy.stopKey("x"))
        assertFalse("x" in listOf(PartyPolicy.stopKey("x")))
    }

    @Test
    fun `service stops on the three terminal codes only`() {
        for (c in listOf("live_location_inactive", "not_found", "party_not_consented")) {
            assertTrue(c, PartyPolicy.shouldStop(c))
        }
        for (c in listOf("rate_limited", "validation", null, "")) assertFalse(c.toString(), PartyPolicy.shouldStop(c))
    }

    @Test
    fun `amanah wording only for amanah`() {
        assertTrue(PartyPolicy.isAmanah("amanah"))
        assertFalse(PartyPolicy.isAmanah("delivery"))
        assertFalse(PartyPolicy.isAmanah("order"))
    }
}
