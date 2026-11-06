package com.rahalgo.merchant.orders

import com.rahalgo.shared.merchant.MerchantOrder
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class SharedStopsTest {
    @Test
    fun labelOnlyWhenTwoOrMore() {
        assertNull(sharedStopsCount(MerchantOrder()))
        assertNull(sharedStopsCount(MerchantOrder(sharedStops = 1)))
        assertEquals(2, sharedStopsCount(MerchantOrder(sharedStops = 2)))
        assertEquals(3, sharedStopsCount(MerchantOrder(sharedStops = 3)))
    }
}
