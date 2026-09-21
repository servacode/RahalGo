package com.rahalgo.customer

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الطلبُ الخاصُّ يعرض منتقيَ الدفع** (`CUST-CUSTOM-020`، §40.11)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **المحرّكُ يقبل نقداً أو محفظةً للطلب الخاصّ** (`orders/custom.go`)، **وكان
 * التطبيقُ لا يرسل الحقلَ فيقع كلُّ طلبٍ نقداً** — فلا يختار صاحبُه المحفظةَ
 * ولو أرادها. **فوجب أن تعرض الشاشةُ الخيارَين وترسل المختار.**
 *
 * **اختبارُ مصدرٍ**: عقدُ الشاشة لا يُقاس بلقطة.
 */
class CustomPaymentTest {

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

    /** **الشاشةُ تعرض خياريَ نقدٍ ومحفظةٍ وترسل المختار.** */
    @Test
    fun customScreenExposesPaymentSelector() {
        val s = read("app-customer/src/main/kotlin/com/rahalgo/customer/custom/CustomScreen.kt")
        assertTrue(
            "**لا خيارَ محفظةٍ في الطلب الخاصّ**",
            s.contains("R.string.cart_cash") && s.contains("R.string.cart_wallet"),
        )
        assertTrue(
            "**المختارُ لا يُرسَل**",
            s.contains("if (wallet) \"wallet\" else \"cash\""),
        )
    }

    /** **والحمولةُ تحمل طريقةَ الدفع.** */
    @Test
    fun newCustomCarriesPaymentMethod() {
        val s = read("shared/src/main/kotlin/com/rahalgo/shared/customer/CustomerApi.kt")
        assertTrue(
            "**`NewCustom` بلا `payment_method`**",
            Regex("data class NewCustom\\b[\\s\\S]*?payment_method[\\s\\S]*?\\)").containsMatchIn(s),
        )
    }
}
