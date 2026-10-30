package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عقدُ واجهةِ الدفع بالمحفظة** (`CUST-WAL-011..014`, `CUST-12-023`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **الرصيدُ < الإجماليِّ المستحقِّ ⇒ خيارُ المحفظة معطّلٌ ظاهراً لا يُختار،
 * ويُقال الرصيدُ والسبب** — لا يُنتظَر رفضُ المحرّك. **والمساواةُ صالحة**
 * (`>=`). **ويُعاد التقييمُ متى تبدّل الإجماليّ أو الرصيد.** **ولا حسابَ
 * محلّيٌّ منفصل**: الرصيدُ من `ShellViewModel` (`me.wallet().balance`).
 * **والمحرّكُ يبقى الحاكمَ الأخير** (`CUST-12-023`: 409 `insufficient_balance`).
 */
class CustWalletUxTest {

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

    private val cart = "app-customer/src/main/kotlin/com/rahalgo/customer/cart/CartScreen.kt"
    private val main = "app-customer/src/main/kotlin/com/rahalgo/customer/MainActivity.kt"

    /**
     * **CUST-WAL-011 · المحفظةُ لا تُعرض حين لا يكفي رصيدُها** (قرارُ المالك ٢٠٢٦-١٠-٣٠:
     * «من محفظتي ما تطلع أبداً إلّا إذا الرصيد بيكفي» — كانت خياراً معطّلاً).
     */
    @Test
    fun walletOptionHiddenWhenInsufficient() {
        val s = read(cart)
        assertTrue(
            "**لا يُحسَب `walletBlocked` من الإجماليّ المستحقّ**",
            s.contains("val walletBlocked = payableTotal == null || walletBalance < payableTotal"),
        )
        assertTrue(
            "**المحفظةُ لا تغيب حين لا تكفي** — يجب سطرُ «نقداً عند الاستلام» بدل الخيارين",
            s.contains("if (walletBlocked) {") && s.contains("R.string.ord_pay_cash_only"),
        )
    }

    /**
     * **CUST-WAL-012 · ولا رسالةَ رصيدٍ تُخيف** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — لا «رصيد محفظتك
     * غير كافٍ» ولا «يمكنك الدفع نقدًا»: الزبونُ لا يظنّ أنّ عليه شحنَ محفظته ليطلب.
     */
    @Test
    fun noInsufficientBalanceMessage() {
        val s = read(cart)
        assertFalse("**عادت رسالةُ «رصيدك غير كافٍ»**", s.contains("R.string.cart_wallet_insufficient"))
        assertFalse("**عاد سطرُ «يمكنك الدفع نقدًا»**", s.contains("R.string.cart_wallet_cod_hint"))
    }

    /**
     * **CUST-WAL-013 · حدُّ المساواة صالح** — `balance == total` يُقبل
     * (المقارنةُ `<` لا `<=`).
     */
    @Test
    fun exactBoundaryIsValid() {
        val s = read(cart)
        assertTrue(
            "**المقارنةُ تمنع المساواة** — يجب `walletBalance < payableTotal` (فـ`==` يمرّ)",
            s.contains("walletBalance < payableTotal"),
        )
        assertFalse(
            "**استُعمل `<=` فمُنعت المساواة**",
            s.contains("walletBalance <= payableTotal"),
        )
    }

    /**
     * **CUST-WAL-014 · يُعاد التقييمُ من حالةٍ موثوقةٍ مُراقَبة** — لا حسابَ
     * محلّيٌّ منفصل؛ الرصيدُ من `ShellViewModel`، والإجماليُّ من التسعيرة.
     */
    @Test
    fun reevaluatesFromAuthoritativeState() {
        val c = read(cart)
        // الإجماليُّ المستحقُّ من التسعيرة (`vm.priced`) لا رقمٌ مُختلَق
        assertTrue(
            "**الإجماليُّ المستحقُّ لا يُشتقُّ من التسعيرة**",
            c.contains("val q0 = vm.priced") && c.contains("val payableTotal"),
        )
        // والرصيدُ مُمرَّرٌ من الغلاف (لا حساب محلّي)
        val m = read(main)
        assertTrue(
            "**الرصيدُ ليس من الحالة الموثوقة** (`shell.balance`)",
            m.contains("walletBalance = shell.balance"),
        )
        // وخيارٌ معطّلٌ لا يبقى مختاراً (يُعاد إلى النقد)
        assertTrue(
            "**لا يُعاد الخيارُ إلى النقد حين يُعطَّل**",
            c.contains("if (walletBlocked && vm.wallet) vm.wallet = false"),
        )
    }

    /** **CUST-WAL / CUST-12-023 · والمحرّكُ يبقى الحاكمَ — الإرسالُ يمرّ بالخادم.** */
    @Test
    fun backendRemainsFinalAuthority() {
        // الإرسالُ يُمرّر طريقةَ الدفع للخادم (الذي يرفض الناقص، `insufficient_balance`).
        val s = read(cart)
        assertTrue(
            "**الإرسالُ لا يُمرّر طريقةَ الدفع للخادم**",
            s.contains("if (vm.wallet) \"wallet\" else \"cash\""),
        )
    }
}
