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
    fun drinksGoBelowAllItemsAndTotalIsUnitTimesQty() {
        val lines = listOf(
            OrderLine(id = "a", name = "شاورما", qty = 1, unitPrice = 25, options = listOf(LineOption("الحجم", "عادي"))),
            OrderLine(id = "b", name = "بطاطا", qty = 1, unitPrice = 10),
            OrderLine(id = "c", name = "عيران", qty = 2, unitPrice = 25, drinkFor = "a"),
        ).map { it.toInvoiceLine() }
        val out = invoiceOrder(lines)
        assertEquals(listOf("شاورما عادي × 1", "بطاطا × 1", "عيران × 2"), out.map { it.text })
        assertEquals(listOf(25L, 10L, 50L), out.map { it.amount })
    }

    /** **المشروبُ نفسُه مع صنفين سطرٌ واحدٌ بعدده ومجموعه** — تحت كلّ الأصناف. */
    @Test
    fun sameDrinkAcrossTwoItemsIsOneRow() {
        val lines = listOf(
            OrderLine(id = "a", name = "شاورما", qty = 2, unitPrice = 20, options = listOf(LineOption("الحجم", "عادي"))),
            OrderLine(id = "d1", name = "عيران", qty = 1, unitPrice = 5, drinkFor = "a"),
            OrderLine(id = "d2", name = "ببسي", qty = 1, unitPrice = 7, drinkFor = "a"),
            OrderLine(id = "b", name = "شاورما", qty = 3, unitPrice = 30, options = listOf(LineOption("الحجم", "دبل"))),
            OrderLine(id = "d3", name = "عيران", qty = 2, unitPrice = 5, drinkFor = "b"),
        ).map { it.toInvoiceLine() }
        val out = invoiceOrder(lines)
        assertEquals(listOf("شاورما عادي × 2", "شاورما دبل × 3", "عيران × 3", "ببسي × 1"), out.map { it.text })
        assertEquals(listOf(40L, 90L, 15L, 7L), out.map { it.amount })
    }
}
