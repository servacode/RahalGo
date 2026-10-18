package com.rahalgo.customer.orders

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **كلُّ نافذةِ سعرٍ في تطبيق الزبون: قبولٌ أخضرُ ممتلئ ورفضٌ أحمر** (طلبُ المالك
 * ٢٠٢٦-١٠-١٨) — عرضُ الطلب الخاصّ وثمنُ بضاعته ورفعُ مبلغه (`QuoteConfirmPanel`،
 * في البطاقة والنافذة معاً)، **وقبولُ المبالغ المتبدّلة في السلّة.**
 */
class QuoteToneTest {

    private fun src(path: String): String {
        var dir = File("").absoluteFile
        repeat(6) {
            val f = File(dir, "app-customer/src/main/kotlin/com/rahalgo/customer/$path")
            if (f.exists()) return f.readText()
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد $path")
    }

    @Test
    fun quotePanelAcceptGreenRejectRed() {
        val panel = src("orders/OrderCard.kt")
            .substringAfter("private fun QuoteConfirmPanel(")
            .substringBefore("fun QuoteConfirmDialog(")
        val buttons = Regex("""RahalButton\((.*?)\)\s*\{""", RegexOption.DOT_MATCHES_ALL)
            .findAll(panel).map { it.groupValues[1] }.toList()
        assertEquals("زرّا القبول والرفض", 2, buttons.size)
        assertTrue("القبول ليس أخضر: " + buttons[0], buttons[0].contains("PriceConfirm.Accept"))
        assertTrue("الرفض ليس أحمر: " + buttons[1], buttons[1].contains("PriceConfirm.Reject"))
    }

    @Test
    fun cartChangedAmountsAcceptIsGreenFilled() {
        val cart = src("cart/CartScreen.kt")
        val call = cart.substringBefore("onClick = { vm.acceptChanges() }").takeLast(200) +
            cart.substringAfter("onClick = { vm.acceptChanges() }").take(200)
        assertTrue("قبول المبالغ زرٌّ نصّيّ", call.contains("RahalButton(") && !call.contains("RahalTextButton(onClick = { vm.acceptChanges"))
        assertTrue("قبول المبالغ ليس أخضر", call.contains("PriceConfirm.Accept"))
    }
}
