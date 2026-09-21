package com.rahalgo.customer

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقةُ الطلبِ تحمل ما يُعرِّفه** (`CUST-14-021`، قرارُ المالك §40.15-4)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **البطاقةُ هي سطحُ الطلبِ الأوّل (لا شاشةَ تفصيل)** — فوجب أن تحمل: وقتَ
 * الطلب، وطريقةَ الدفع، والعنوان. **كانت هذه في النموذج (`createdAt`/
 * `paymentMethod`/`addressText`) تُجلَب ولا تُعرَض** — فأُضيفت للبطاقة.
 *
 * **اختبارُ مصدرٍ**: عقدُ البطاقةِ لا يُقاس بلقطةِ شاشة.
 */
class OrderCardInfoTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("**ملفٌّ غائب**: " + rel, f.exists())
        return f.readText().replace("\r\n", "\n")
    }

    private val card = "app-customer/src/main/kotlin/com/rahalgo/customer/orders/OrderCard.kt"

    /** **وقتُ الطلب** — `ord_placed` + `Since.text(…, order.createdAt)`. */
    @Test
    fun showsPlacedTime() {
        val s = read(card)
        assertTrue(
            "**البطاقةُ لا تُظهر وقتَ الطلب**",
            s.contains("R.string.ord_placed") && s.contains("Since.text(ctx, order.createdAt)"),
        )
    }

    /** **طريقةُ الدفع** — `ord_payment` من `order.paymentMethod`. */
    @Test
    fun showsPaymentMethod() {
        val s = read(card)
        assertTrue(
            "**البطاقةُ لا تُظهر طريقةَ الدفع**",
            s.contains("R.string.ord_payment") && s.contains("order.paymentMethod == \"wallet\""),
        )
    }

    /** **العنوان** — `ord_address` من `order.addressText`. */
    @Test
    fun showsDeliveryAddress() {
        val s = read(card)
        assertTrue(
            "**البطاقةُ لا تُظهر عنوانَ التوصيل**",
            s.contains("R.string.ord_address") && s.contains("order.addressText"),
        )
    }
}
