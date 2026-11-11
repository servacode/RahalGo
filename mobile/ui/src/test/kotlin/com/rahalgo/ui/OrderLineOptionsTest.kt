package com.rahalgo.ui

import org.junit.Assert.assertEquals
import org.junit.Test

/** **الخياراتُ بمجموعاتها تحت الاسم** (قرارُ المالك ٢٠٢٦-١١-١٢) — «المقاس: 42 · اللون: أسود». */
class OrderLineOptionsTest {

    @Test
    fun groupedOptionsReadAsGroupColonValue() {
        assertEquals(
            "المقاس: 42 · اللون: أسود",
            invoiceOptionsLabel(listOf("المقاس" to "42", "اللون" to "أسود")),
        )
    }

    @Test
    fun optionsWithoutGroupOrValueAreSkipped() {
        assertEquals("", invoiceOptionsLabel(listOf("" to "دبل", "المقاس" to " ")))
        assertEquals("اللون: أسود", invoiceOptionsLabel(listOf("" to "دبل", "اللون" to "أسود")))
    }
}
