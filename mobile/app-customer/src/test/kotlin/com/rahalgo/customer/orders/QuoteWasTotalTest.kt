package com.rahalgo.customer.orders

import com.rahalgo.shared.customer.MyOrder
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * **«كان X وصار Y» بنافذة السعر** (قرار المالك ٢٠٢٦-١١-٠٦) — الكابتن طلب تعديل ثمن البضاعة
 * بعد الاستلام، **والزبون يشوف القديم والجديد** قبل ما يوافق أو يرفض.
 */
class QuoteWasTotalTest {

    private fun order(version: Long, confirmed: Long?, confirmedTotal: Long?, total: Long) = MyOrder(
        id = "o1", kind = "custom", status = "on_the_way", total = total,
        customGoodsAmount = 1400, quoteVersion = version,
        quoteConfirmedVersion = confirmed, quoteConfirmedTotal = confirmedTotal,
    )

    @Test
    fun `a pending new price shows the old total`() {
        assertEquals(2500L, quoteWasTotal(order(version = 2, confirmed = 1, confirmedTotal = 2500, total = 2900)))
    }

    @Test
    fun `usd goods keep the old total as the delivery fee only`() {
        val o = order(version = 2, confirmed = 1, confirmedTotal = 150, total = 200).copy(customGoodsUsd = 1250)
        assertEquals(150L, quoteWasTotal(o))
        assertEquals("12.50$", com.rahalgo.ui.amountText(o.customGoodsUsd!!, "USD"))
        // التوصيل ليرة والبضاعة دولار — لا يُجمعان: المجموع المؤكَّد هو `total` وحده.
        assertEquals(200L, o.total)
    }

    @Test
    fun `nothing pending or nothing changed shows nothing`() {
        assertNull(quoteWasTotal(order(version = 2, confirmed = 2, confirmedTotal = 2900, total = 2900)))
        assertNull(quoteWasTotal(order(version = 1, confirmed = null, confirmedTotal = null, total = 2900)))
        assertNull(quoteWasTotal(order(version = 3, confirmed = 2, confirmedTotal = 2900, total = 2900)))
    }
}
