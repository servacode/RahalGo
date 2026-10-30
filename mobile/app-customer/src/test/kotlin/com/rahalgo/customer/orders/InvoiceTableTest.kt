package com.rahalgo.customer.orders

/**
 * **الفاتورةُ جدول** (قرارُ المالك ٢٠٢٦-١٠-٣٠، الثانية) — كلُّ صنفٍ ثمّ مشروباتُه،
 * والخياراتُ «المجموعة: الاسم» تحت الاسم، **ولا «الفاتورة كاملة» تطوي شيئاً.**
 */
import com.rahalgo.shared.customer.MyOrderLine
import com.rahalgo.shared.customer.MyOrderOption
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

class InvoiceTableTest {

    private fun line(id: String, drinkFor: String? = null) =
        MyOrderLine(id = id, name = id, unitPrice = 10, qty = 1, drinkFor = drinkFor)

    @Test
    fun drinksFollowTheirItem() {
        val out = invoiceOrder(
            listOf(line("a"), line("b"), line("d1", "a"), line("d2", "b"), line("x", "gone")),
        ).map { it.id }
        assertEquals(listOf("a", "d1", "b", "d2", "x"), out)
    }

    @Test
    fun optionsLineHasGroupAndName() {
        val l = MyOrderLine(
            name = "شاورما",
            options = listOf(
                MyOrderOption(group = "الحجم", name = "عادي"),
                MyOrderOption(group = "", name = "حار"),
                MyOrderOption(group = "إضافة", name = " "),
            ),
        )
        assertEquals("الحجم: عادي، حار", invoiceOptions(l))
    }

    @Test
    fun wholeInvoiceShownNoCollapse() {
        val f = listOf(
            File("src/main/kotlin/com/rahalgo/customer/orders/OrderCard.kt"),
            File("app-customer/src/main/kotlin/com/rahalgo/customer/orders/OrderCard.kt"),
        ).first { it.exists() }
        val s = f.readText()
        assertTrue("عادت «الفاتورة كاملة»", !s.contains("ord_invoice_full") && !s.contains("INVOICE_PEEK"))
        assertTrue("الفاتورةُ ليست جدولاً", s.contains("InvoiceTable(order)"))
    }
}
