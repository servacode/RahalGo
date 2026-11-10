package com.rahalgo.driver.trip

import com.rahalgo.shared.driver.ShamcashPay
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **كيف قبضت؟ نقدي / شام كاش** (قرار المالك 2026-11-10) — النقد لا يُرسل شيئاً كما كان،
 * وشام كاش مبلغٌ صالحٌ أو لا تأكيد.
 */
class ShamcashDeliverTest {

    @Test
    fun `cash sends nothing and is always ready`() {
        assertNull(shamcashPayload(false, false, "150000"))
        assertNull(shamcashPayload(false, true, ""))
        assertTrue(shamcashReady(false, false, ""))
    }

    @Test
    fun `syp whole number`() {
        assertEquals(ShamcashPay("SYP", 150000.0), shamcashPayload(true, false, "150000"))
        assertTrue(shamcashReady(true, false, "150000"))
    }

    @Test
    fun `usd up to two decimals`() {
        assertEquals(ShamcashPay("USD", 12.5), shamcashPayload(true, true, "12.5"))
        assertEquals(ShamcashPay("USD", 12.34), shamcashPayload(true, true, "12.34"))
        assertNull(shamcashPayload(true, true, "12.345"))
    }

    @Test
    fun `empty zero and junk are invalid and block the confirm`() {
        for (t in listOf("", "0", "0.00", "abc", "-5", "12.")) {
            assertNull("«$t» USD", shamcashPayload(true, true, t))
            assertFalse("«$t» USD", shamcashReady(true, true, t))
        }
        for (t in listOf("", "0", "12.5", "1e3")) {
            assertNull("«$t» SYP", shamcashPayload(true, false, t))
            assertFalse("«$t» SYP", shamcashReady(true, false, t))
        }
    }

    @Test
    fun `syp above the amount due asks to confirm, equal or less does not`() {
        assertTrue(shamcashExceedsDue(ShamcashPay("SYP", 30000.0), 25000))
        assertFalse(shamcashExceedsDue(ShamcashPay("SYP", 25000.0), 25000))
        assertFalse(shamcashExceedsDue(ShamcashPay("SYP", 10000.0), 25000))
        // الدولار ما له مطلوب يُقارَن فيه، والنقد ما فيه مبلغ.
        assertFalse(shamcashExceedsDue(ShamcashPay("USD", 999.0), 25000))
        assertFalse(shamcashExceedsDue(null, 25000))
    }
}
