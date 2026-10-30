package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverItem
import com.rahalgo.shared.model.DriverOption
import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.ui.invoiceOrder
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * **السائقُ يرى الفاتورةَ كاملة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — سطرٌ لكلّ عنصرٍ بسعره، والمشروبُ
 * بعد صنفه، ثمّ المجموع والتوصيل والخصم والإجمالي. **وخادمٌ قديمٌ بلا أسعار يُعرض كما كان.**
 */
class DriverInvoiceTest {

    private val order = DriverOrder(
        kind = "standard",
        subtotal = 9240,
        deliveryFee = 150,
        discount = 40,
        total = 9350,
        items = listOf(
            DriverItem(id = "c", name = "كولا", qty = 1, unitPrice = 50),
            DriverItem(id = "d", name = "عيران", qty = 2, unitPrice = 25, drinkFor = "s"),
            DriverItem(
                id = "s", name = "شاورما", qty = 3, unitPrice = 30, options = "دبل",
                optionList = listOf(DriverOption("الحجم", "دبل")),
            ),
            DriverItem(
                id = "b", name = "بوط أسود", qty = 1, unitPrice = 9100, options = "40",
                optionList = listOf(DriverOption("النمرة", "40")),
            ),
        ),
    )

    @Test
    fun `lines carry prices and drinks go below all items`() {
        val lines = invoiceOrder(driverInvoiceLines(order))
        assertEquals(
            listOf("كولا × 1", "شاورما دبل × 3", "بوط أسود × 1 نمرة 40", "عيران × 2"),
            lines.map { it.text },
        )
        assertEquals(listOf(50L, 90L, 9100L, 50L), lines.map { it.amount })
    }

    @Test
    fun `the same drink with two items is one combined row`() {
        val two = order.copy(
            items = order.items + listOf(
                DriverItem(id = "s2", name = "شاورما", qty = 1, unitPrice = 20, options = "عادي", optionList = listOf(DriverOption("الحجم", "عادي"))),
                DriverItem(id = "d2", name = "عيران", qty = 1, unitPrice = 25, drinkFor = "s2"),
            ),
        )
        val lines = invoiceOrder(driverInvoiceLines(two))
        assertEquals(
            listOf("كولا × 1", "شاورما دبل × 3", "بوط أسود × 1 نمرة 40", "شاورما عادي × 1", "عيران × 3"),
            lines.map { it.text },
        )
        assertEquals(75L, lines.last().amount)
    }

    @Test
    fun `totals block reads the server numbers`() {
        val t = driverInvoiceTotals(order)!!
        assertEquals(9240L, t.subtotal)
        assertEquals(150L, t.delivery)
        assertEquals(false, t.free)
        assertEquals(40L, t.discount)
        assertEquals(9350L, t.total)
    }

    @Test
    fun `free delivery is said free`() {
        val t = driverInvoiceTotals(order.copy(freeWaived = 150))!!
        assertEquals(0L, t.delivery)
        assertEquals(true, t.free)
    }

    @Test
    fun `an old server without prices renders as before`() {
        val old = DriverOrder(
            kind = "standard",
            items = listOf(DriverItem(name = "شاورما", qty = 2, options = "دبل، جبنة")),
        )
        val lines = driverInvoiceLines(old)
        assertEquals("شاورما دبل، جبنة × 2", lines.single().text)
        assertNull(lines.single().amount)
        assertNull(driverInvoiceTotals(old))
        assertEquals("شاورما دبل، جبنة × 2", itemsText(old))
    }

    @Test
    fun `custom orders have no totals block`() {
        assertNull(driverInvoiceTotals(order.copy(kind = "custom")))
    }
}
