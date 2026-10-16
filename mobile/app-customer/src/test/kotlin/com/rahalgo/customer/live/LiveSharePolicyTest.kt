package com.rahalgo.customer.live

import com.rahalgo.shared.customer.MyOrder
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **«شارك موقعك المباشر» — مرّةً واحدةً لكلّ طلب** (قرارُ المالك ٢٠٢٦-١٠-١٦).
 *
 * **يُسأل حين يصير الطلبُ بالطريق** — عاديّاً أو مشترياتٍ أو أمانة — **ثمّ لا يُسأل
 * عنه ثانيةً** شارك أو رفض، **ولو أُعيد تشغيلُ التطبيق** (الذاكرةُ تُحفظ سطراً).
 */
class LiveSharePolicyTest {

    private fun o(id: String, status: String = "on_the_way", kind: String = "standard", mode: String = "purchase") =
        MyOrder(id = id, status = status, kind = kind, customMode = mode)

    @Test
    fun `asks when the order is on the way for every kind`() {
        for (order in listOf(o("a"), o("b", kind = "custom"), o("c", kind = "custom", mode = "amanah"))) {
            assertEquals(order.id, LiveSharePolicy.pending(listOf(order), emptyList())?.id)
        }
    }

    @Test
    fun `does not ask before or after on the way`() {
        for (st in listOf("pending", "accepted", "preparing", "dispatching", "assigned", "at_pickup", "picked_up", "at_dropoff")) {
            assertNull(st, LiveSharePolicy.pending(listOf(o("a", status = st)), emptyList()))
        }
    }

    @Test
    fun `the popup shows once per order`() {
        val open = listOf(o("a"))
        var decided = emptyList<String>()
        assertEquals("a", LiveSharePolicy.pending(open, decided)?.id)
        decided = LiveSharePolicy.remember(decided, "a")
        assertNull(LiveSharePolicy.pending(open, decided))
        // **وطلبٌ ثانٍ بالطريق يُسأل عنه هو** — القرارُ لطلبه لا للحساب.
        assertEquals("b", LiveSharePolicy.pending(open + o("b"), decided)?.id)
    }

    @Test
    fun `the choice survives a restart`() {
        val saved = LiveSharePolicy.encode(LiveSharePolicy.remember(emptyList(), "a"))
        val restored = LiveSharePolicy.decode(saved)
        assertNull(LiveSharePolicy.pending(listOf(o("a")), restored))
        assertEquals(emptyList<String>(), LiveSharePolicy.decode(null))
        assertEquals(emptyList<String>(), LiveSharePolicy.decode(""))
    }

    @Test
    fun `memory is bounded and keeps the newest`() {
        var d = emptyList<String>()
        for (i in 1..(LiveSharePolicy.MAX_REMEMBERED + 10)) d = LiveSharePolicy.remember(d, "o$i")
        assertEquals(LiveSharePolicy.MAX_REMEMBERED, d.size)
        assertTrue("o${LiveSharePolicy.MAX_REMEMBERED + 10}" in d)
        assertFalse("o1" in d)
        // **وإعادةُ الطلب نفسِه لا تكرّره.**
        assertEquals(d.size, LiveSharePolicy.remember(d, d.last()).size)
    }

    @Test
    fun `sharing stops when the order is delivered or ended`() {
        assertTrue(LiveSharePolicy.keepSharing("a", listOf(o("a", status = "at_dropoff"))))
        assertFalse(LiveSharePolicy.keepSharing("a", listOf(o("a", status = "delivered"))))
        // **وطلبٌ خرج من الجاري كلِّه انتهى.**
        assertFalse(LiveSharePolicy.keepSharing("a", emptyList()))
    }
}
