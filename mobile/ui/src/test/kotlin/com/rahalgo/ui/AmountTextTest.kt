package com.rahalgo.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * **مبلغُ «لدي توصيلة» و«إحضار» بعملته** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — **والدولارُ بالسنتات**،
 * فمن عرضه بـ`money` كتب «1,250 ل.س» لـ١٢٫٥٠$.
 */
class AmountTextTest {
    @Test
    fun usdIsCentsShownAsDollars() {
        assertEquals("25$", amountText(2500, "USD"))
        assertEquals("12.50$", amountText(1250, "USD"))
        assertEquals("0.05$", amountText(5, "USD"))
        assertEquals("1,200.75$", amountText(120075, "USD"))
    }

    @Test
    fun sypAndEmptyAreLira() {
        assertEquals(money(21000), amountText(21000, "SYP"))
        assertEquals(money(21000), amountText(21000, ""))
        assertEquals(money(21000), amountText(21000, null))
    }

    @Test
    fun inputKeepsDigitsForLiraAndTwoDecimalsForDollar() {
        assertEquals("21000", amountInput("21,000", "SYP"))
        assertEquals("12.5", amountInput("12,5", "USD"))
        assertEquals("12.50", amountInput("12٫505", "USD"))
        assertEquals("0.5", amountInput(".5", "USD"))
        assertEquals("1.23", amountInput("1.2.3", "USD"))
    }

    @Test
    fun valueIsMajorUnits() {
        assertEquals(12.5, amountValue("12.5")!!, 0.0)
        assertNull(amountValue(""))
        assertNull(amountValue("0"))
    }
}
