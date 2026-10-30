package com.rahalgo.ui

import com.rahalgo.shared.merchant.LineOption
import com.rahalgo.shared.merchant.OrderLine
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * **سطرُ الفاتورة بشكلٍ واحد** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — لا اسمَ مجموعة، والوزنُ والنمرةُ
 * بعد العدد، وكلُّ قيمةٍ غيرِهما بعد الاسم، والمشروبُ سطرٌ بعد صنفه.
 */
class InvoiceLineTextTest {
    @Test
    fun ownerExamples() {
        assertEquals("بندورة × 1 كيلو", invoiceLineText("بندورة", 1, listOf("وزن ثقيل" to "كيلو")))
        assertEquals("كمون × 1 250 غ", invoiceLineText("كمون", 1, listOf("وزن خفيف" to "250 غ")))
        assertEquals("بوط أسود × 1 نمرة 40", invoiceLineText("بوط", 1, listOf("النمرة" to "40", "اللون" to "أسود")))
        assertEquals("شاورما دبل × 3", invoiceLineText("شاورما", 3, listOf("الحجم" to "دبل")))
        assertEquals("ببسي × 1", invoiceLineText("ببسي", 1, emptyList()))
    }

    @Test
    fun weightValueWithoutGroupStillGoesAfterQty() {
        assertEquals("تفاح × 2 نصف كيلو", invoiceLineText("تفاح", 2, listOf("" to "نصف كيلو")))
        assertEquals("قميص L، أزرق × 1", invoiceLineText("قميص", 1, listOf("المقاس" to "L", "اللون" to "أزرق")))
        assertEquals("بنطال × 1 مقاس 42", invoiceLineText("بنطال", 1, listOf("المقاس" to "42")))
    }

    @Test
    fun drinkLineFollowsItsItemAndTotalIsUnitTimesQty() {
        val lines = listOf(
            OrderLine(id = "a", name = "شاورما", qty = 1, unitPrice = 25, options = listOf(LineOption("الحجم", "عادي"))),
            OrderLine(id = "b", name = "بطاطا", qty = 1, unitPrice = 10),
            OrderLine(id = "c", name = "عيران", qty = 2, unitPrice = 25, drinkFor = "a"),
        ).map { it.toInvoiceLine() }
        val out = invoiceOrder(lines)
        assertEquals(listOf("شاورما عادي × 1", "عيران × 2", "بطاطا × 1"), out.map { it.text })
        assertEquals(listOf(25L, 50L, 10L), out.map { it.amount })
    }
}
