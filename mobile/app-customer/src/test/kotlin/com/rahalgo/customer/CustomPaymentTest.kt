package com.rahalgo.customer

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عقدُ دفعِ الطلب الخاصّ — يُؤجَّل إلى تأكيد العرض** (Batch 2c)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك — عقدُ Batch 2c.)
 *
 * **الطلبُ المخصَّصُ لا سعرَ له عند الإنشاء** — فلا تُختار طريقةُ الدفع حينها،
 * ولا تُعرَض المحفظةُ قبل معرفة المبلغ. **بل يُقال «تُحدَّد بعد التكلفة»**،
 * ويؤكّد الزبونُ ويختار الدفعَ حين يصل العرضُ (`confirm-quote`). **والمحفظةُ
 * لا تُعرَض إلّا إن غطّى رصيدُها المبلغَ.**
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

    /** **الإنشاءُ لا يعرض منتقيَ الدفع — بل يؤجّله إلى تأكيد العرض.** */
    @Test
    fun customCreationDefersPaymentChoice() {
        val s = read("app-customer/src/main/kotlin/com/rahalgo/customer/custom/CustomScreen.kt")
        // **لا منتقيَ دفعٍ عند الإنشاء** — لا اختيارُ الزبون يُرسَل.
        assertTrue("**عاد اختيارُ الدفع يُرسَل من الإنشاء**", !s.contains("if (wallet)"))
        // **بل يُقال إنّ الدفعَ يُحدَّد بعد التكلفة.**
        assertTrue(
            "**لا إشعارَ بتأجيل اختيار الدفع**",
            s.contains("R.string.cst_pay_after_cost"),
        )
        // **وسياسةُ الأجرة تُقرأ من الحال** — أجرةٌ محدَّدةٌ من المنصة أو بعد قبول السائق.
        assertTrue(
            "**لا تُقرأ سياسةُ أجرة المخصَّص عند الإنشاء**",
            s.contains("Serving.customFeeAdminDefined") && s.contains("R.string.cst_fee_on_deal"),
        )
    }

    /** **والحمولةُ تحمل طريقةَ الدفع** (يستعملها المحرّكُ افتراضاً آمناً). */
    @Test
    fun newCustomCarriesPaymentMethod() {
        val s = read("shared/src/main/kotlin/com/rahalgo/shared/customer/CustomerApi.kt")
        assertTrue(
            "**`NewCustom` بلا `payment_method`**",
            Regex("data class NewCustom\\b[\\s\\S]*?payment_method[\\s\\S]*?\\)").containsMatchIn(s),
        )
    }

    /** **وبابُ تأكيد العرض موصولٌ** — نسخةٌ ومبلغٌ متوقَّعان وطريقةُ دفع. */
    @Test
    fun customerApiConfirmsQuote() {
        val s = read("shared/src/main/kotlin/com/rahalgo/shared/customer/CustomerApi.kt")
        assertTrue("**لا `confirmQuote`**", s.contains("fun confirmQuote("))
        assertTrue("**لا نداءَ `confirm-quote`**", s.contains("/confirm-quote"))
        assertTrue(
            "**التأكيدُ بلا نسخةٍ ومبلغٍ متوقَّعين**",
            s.contains("expected_total") && s.contains("expected_quote_version"),
        )
    }

    /** **وبطاقةُ الطلب لا تعرض المحفظةَ إلّا إن غطّى رصيدُها المبلغ.** */
    @Test
    fun orderCardGatesWalletOnBalance() {
        val s = read("app-customer/src/main/kotlin/com/rahalgo/customer/orders/OrderCard.kt")
        assertTrue("**لا بوّابةَ رصيدٍ للمحفظة**", s.contains("walletBalance >= total"))
        assertTrue("**لا زرَّ تأكيدٍ للعرض**", s.contains("R.string.ord_confirm_action"))
        // **والرصيدُ غيرُ الكافي يُقال لا يُخفى بلا سبب.**
        assertTrue("**لا رسالةَ رصيدٍ غيرِ كافٍ**", s.contains("R.string.ord_wallet_short"))
    }
}
