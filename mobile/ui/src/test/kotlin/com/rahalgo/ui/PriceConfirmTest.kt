package com.rahalgo.ui

import org.junit.Assert.assertEquals
import org.junit.Test

/** **نافذةُ السعر: القبولُ أخضرُ والرفضُ أحمر** (طلبُ المالك ٢٠٢٦-١٠-١٨). */
class PriceConfirmTest {
    @Test
    fun acceptIsGreenRejectIsRed() {
        assertEquals(Tone.Success, PriceConfirm.Accept)
        assertEquals(Tone.Danger, PriceConfirm.Reject)
    }
}
