package com.rahalgo.customer.orders

import com.rahalgo.shared.customer.MyOrder
import com.rahalgo.shared.customer.MyOrderLine
import org.junit.Assert.assertEquals
import org.junit.Test

/** **«+N» أصنافٌ أخرى لا قطع** (قرارُ المالك ٢٠٢٦-١١-١١) — توأمُه `check-order-title.mjs` في الويب. */
class ItemsTitleTest {
    private fun order(vararg lines: MyOrderLine, preview: String = "") =
        MyOrder(items = lines.toList(), itemsPreview = preview, itemsCount = lines.sumOf { it.qty })

    private fun line(id: String, name: String, qty: Int, drinkFor: String? = null) =
        MyOrderLine(id = "l$id$name", menuItemId = id, name = name, qty = qty, drinkFor = drinkFor)

    @Test
    fun oneLineWithQtyTwoIsTimes() {
        assertEquals("برغر كلاسيك × 2", ItemsTitle.of(order(line("a", "برغر كلاسيك", 2))).text())
    }

    @Test
    fun oneLineWithQtyOneIsPlainName() {
        assertEquals("برغر كلاسيك", ItemsTitle.of(order(line("a", "برغر كلاسيك", 1))).text())
    }

    @Test
    fun twoDifferentItemsIsPlusOne() {
        val o = order(line("a", "برغر كلاسيك", 2), line("b", "بطاطا", 1))
        assertEquals("برغر كلاسيك +1", ItemsTitle.of(o).text())
    }

    @Test
    fun sameItemTwoOptionLinesIsStillOneItem() {
        val o = order(line("a", "برغر كلاسيك", 1), line("a", "برغر كلاسيك", 1))
        assertEquals("برغر كلاسيك × 2", ItemsTitle.of(o).text())
    }

    @Test
    fun drinksAreNotItems() {
        val o = order(line("a", "برغر كلاسيك", 1), line("d", "كولا", 2, drinkFor = "la"))
        assertEquals("برغر كلاسيك", ItemsTitle.of(o).text())
    }

    @Test
    fun noLinesFallsBackToPreview() {
        assertEquals("شاورما +2", ItemsTitle.of(MyOrder(itemsPreview = "شاورما، فلافل، كولا")).text())
    }
}
