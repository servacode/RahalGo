package com.rahalgo.merchant.orders

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **طلبٌ انتهى بلا بيعٍ لا يُري مستحقّاً · ومجموعاتُ شاشة الطلبات** (تدقيقُ ٢٠٢٦-١٠-٢٧).
 */
class PayoutGroupTest {
    @Test
    fun endedWithoutSaleHidesPayout() {
        for (s in listOf("rejected", "cancelled", "failed", "refunded")) {
            assertTrue("**مستحقٌّ يظهر في $s**", noPayout(s))
        }
        for (s in listOf("pending", "accepted", "preparing", "dispatching", "delivered")) {
            assertFalse("**المستحقُّ مخفيٌّ في $s**", noPayout(s))
        }
    }

    @Test
    fun ordersFallIntoTheirGroups() {
        assertEquals(OrderGroup.New, groupOf("pending", null))
        assertEquals(OrderGroup.Preparing, groupOf("accepted", null))
        assertEquals(OrderGroup.Preparing, groupOf("preparing", null))
        // **في وضع المتاجر القبولُ يقفز إلى «بانتظار سائق»** — وما زال يُحضَّر.
        assertEquals(OrderGroup.Preparing, groupOf("dispatching", null))
        assertEquals(OrderGroup.Waiting, groupOf("dispatching", "2026-10-27T10:00:00Z"))
        assertEquals(OrderGroup.Waiting, groupOf("picked_up", null))
        assertEquals(OrderGroup.Closed, groupOf("delivered", null))
        assertEquals(OrderGroup.Closed, groupOf("rejected", null))
    }
}
