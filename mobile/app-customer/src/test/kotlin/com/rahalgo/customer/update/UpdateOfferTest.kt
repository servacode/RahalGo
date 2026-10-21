package com.rahalgo.customer.update

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/** **«أأحدث؟» بالأرقام لا بالحروف** (قرارُ المالك ٢٠٢٦-١٠-٢١) — ١٫١٠ بعد ١٫٩. */
class UpdateOfferTest {
    @Test
    fun `numbers not letters`() {
        assertTrue(UpdateOffer.isNewer("1.10.0", "1.9.0"))
        assertTrue(UpdateOffer.isNewer("1.5.0", "1.4.0"))
        assertTrue(UpdateOffer.isNewer("1.4.1", "1.4"))
        assertFalse(UpdateOffer.isNewer("1.4.0", "1.4.0"))
        assertFalse(UpdateOffer.isNewer("1.1.0", "1.4.0"))
        // **ونسخةٌ لا تُقرأ لا تُعرَض** — فلا شريطَ بلا سبب.
        assertFalse(UpdateOffer.isNewer("", "1.4.0"))
        assertFalse(UpdateOffer.isNewer("beta", "1.4.0"))
    }
}
