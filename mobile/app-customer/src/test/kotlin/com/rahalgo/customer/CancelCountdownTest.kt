package com.rahalgo.customer

import com.rahalgo.customer.orders.CancelCountdown
import org.junit.Assert.assertEquals
import org.junit.Test

/** **مهلةُ الإلغاء بكلامٍ مفهوم** (قرارُ المالك ٢٠٢٦-١٠-٣٠). */
class CancelCountdownTest {
    @Test
    fun wording() {
        assertEquals("46 ثانية", CancelCountdown.within(46))
        assertEquals("5 ثوانٍ", CancelCountdown.within(5))
        assertEquals("ثانيتين", CancelCountdown.within(2))
        assertEquals("دقيقة", CancelCountdown.within(60))
        assertEquals("دقيقة و10 ثوانٍ", CancelCountdown.within(70))
        assertEquals("دقيقتين و10 ثوانٍ", CancelCountdown.within(130))
        assertEquals("3 دقائق و15 ثانية", CancelCountdown.within(195))
        assertEquals("", CancelCountdown.within(0))
        assertEquals("", CancelCountdown.within(-1))
    }
}
