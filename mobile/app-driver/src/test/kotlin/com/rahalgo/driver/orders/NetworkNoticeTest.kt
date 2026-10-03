package com.rahalgo.driver.orders

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **«لا اتصال بالإنترنت» تُمحى حين تعود الشبكة** (تجربةُ القبول ٢٠٢٦-١٠-٠٣) — وخطأُ العمل يبقى.
 */
class NetworkNoticeTest {
    private val net = "لا اتصال بالإنترنت"

    @Test
    fun networkErrorClearsOnSuccess() {
        assertTrue(NetworkNotice.clears(net, net))
    }

    @Test
    fun businessErrorStays() {
        assertFalse(NetworkNotice.clears("سبقك سائق آخر إلى هذا الطلب", net))
        assertFalse(NetworkNotice.clears("", net))
    }
}
