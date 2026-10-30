package com.rahalgo.customer.orders

/**
 * **فاتورةُ الزبون بالقاعدة المشتركة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — «شاورما دبل × 3»،
 * «بندورة × 1 كيلو»، «بوط أسود × 1 نمرة 40»؛ كلُّ صنفٍ ثمّ مشروباتُه، **ولا «الفاتورة كاملة».**
 */
import com.rahalgo.shared.customer.MyOrderLine
import com.rahalgo.shared.customer.MyOrderOption
import com.rahalgo.ui.invoiceOrder
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
            listOf(line("a"), line("b"), line("d1", "a"), line("d2", "b"), line("x", "gone")).map { it.toInvoiceLine() },
        ).map { it.id }
        assertEquals(listOf("a", "d1", "b", "d2", "x"), out)
    }

    @Test
    fun customerLinesUseTheSharedRule() {
        fun t(name: String, qty: Int, vararg o: Pair<String, String>) =
            MyOrderLine(name = name, qty = qty, unitPrice = 1, options = o.map { MyOrderOption(group = it.first, name = it.second) })
                .toInvoiceLine().text
        assertEquals("شاورما دبل × 3", t("شاورما", 3, "الحجم" to "دبل"))
        assertEquals("بندورة × 1 كيلو", t("بندورة", 1, "وزن ثقيل" to "كيلو"))
        assertEquals("كمون × 1 250 غ", t("كمون", 1, "وزن خفيف" to "250 غ"))
        assertEquals("بوط أسود × 1 نمرة 40", t("بوط", 1, "اللون" to "أسود", "نمرة رجالي" to "40"))
        assertEquals(30L, MyOrderLine(name = "عيران", qty = 3, unitPrice = 10, drinkFor = "a").toInvoiceLine().amount)
    }

    @Test
    fun wholeInvoiceOneLinePerElement() {
        val f = listOf(
            File("src/main/kotlin/com/rahalgo/customer/orders/OrderCard.kt"),
            File("app-customer/src/main/kotlin/com/rahalgo/customer/orders/OrderCard.kt"),
        ).first { it.exists() }
        val s = f.readText()
        assertTrue("عادت «الفاتورة كاملة»", !s.contains("ord_invoice_full") && !s.contains("INVOICE_PEEK"))
        assertTrue("عاد سطرُ «للواحد» أو عمودُ العدد", !s.contains("ord_unit_each") && !s.contains("INVOICE_QTY_COL"))
        assertTrue("الفاتورةُ لا تستعمل القاعدةَ المشتركة", s.contains("InvoiceTable(order)") && s.contains("com.rahalgo.ui.invoiceLineText("))
    }
}
