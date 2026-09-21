package com.rahalgo.customer

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«لا ندري» يُعرض لا يُبتلع** (`CUST-13-012`، `PC-8`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **انقطاعُ الشبكةِ في منتصف الإرسال ليس حسماً**: قد يكون الطلبُ قُيِّد
 * وضاع الجواب — فيُضبط `uncertain`. **وكان يُضبط ولا يُعرض** (`PC-8`):
 * يرى صاحبُه رسالةَ خطأٍ عامّةً فيعيد الضغطَ **فيصير طلبُه طلبين.**
 *
 * **فوجب أن تُعرض حالُ «لا ندري» صريحةً وتُوجِّه إلى «طلباتي»** قبل إعادة
 * الإرسال. **اختبارُ مصدرٍ**: العقدُ لا يُقاس بلقطةِ شاشة.
 */
class UncertainDisplayTest {

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

    /** **الشاشةُ تعرض `vm.uncertain` بنصِّ `cart_uncertain`.** */
    @Test
    fun cartDisplaysUncertainState() {
        val s = read("app-customer/src/main/kotlin/com/rahalgo/customer/cart/CartScreen.kt")
        assertTrue(
            "**حالُ «لا ندري» تُضبط ولا تُعرض** (PC-8)",
            s.contains("if (vm.uncertain)") && s.contains("R.string.cart_uncertain"),
        )
    }

    /** **والنصُّ موجودٌ ويوجّه إلى «طلباتي».** */
    @Test
    fun uncertainStringGuidesToOrders() {
        val s = read("app-customer/src/main/res/values/strings.xml")
        assertTrue(
            "**نصُّ `cart_uncertain` غائب**",
            s.contains("name=\"cart_uncertain\""),
        )
        assertTrue(
            "**نصُّ «لا ندري» لا يوجّه إلى «طلباتي»**",
            s.contains("طلباتي"),
        )
    }
}
