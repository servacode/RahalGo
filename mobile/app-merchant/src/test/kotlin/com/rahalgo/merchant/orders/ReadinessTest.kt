package com.rahalgo.merchant.orders

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **«جاهز للاستلام» بعد الإرسال التلقائيّ** (٢٠٢٦-١٠-١٧).
 *
 * **وكان الزرُّ في «يُحضَّر» وحدَها** — وفي وضع المتاجر يقفز القبولُ إلى
 * «بانتظار سائق»، **فلم يكن يظهر أبداً.**
 */
class ReadinessTest {
    @Test
    fun readyShowsFromAcceptanceUntilPickup() {
        for (s in listOf("accepted", "preparing", "dispatching", "assigned", "at_pickup")) {
            assertTrue("**الزرُّ غائبٌ في $s**", canMarkReady(s, null))
            assertTrue("**الزرُّ غائبٌ في $s بوقتٍ فارغ**", canMarkReady(s, ""))
        }
    }

    @Test
    fun readyHidesOnceAnnounced() {
        assertFalse(canMarkReady("dispatching", "2026-10-17T10:00:00Z"))
        assertTrue(readyAnnounced("dispatching", "2026-10-17T10:00:00Z"))
        assertFalse(readyAnnounced("dispatching", null))
    }

    @Test
    fun readyNeverShowsBeforeAcceptOrAfterPickup() {
        for (s in listOf("pending", "picked_up", "on_the_way", "at_dropoff", "delivered", "rejected", "cancelled", "failed")) {
            assertFalse("**الزرُّ ظهر في $s**", canMarkReady(s, null))
        }
    }
}
