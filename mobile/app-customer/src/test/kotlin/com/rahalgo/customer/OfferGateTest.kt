package com.rahalgo.customer

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بابُ العروض يمرّ ببوّابةِ الخدمة** (`CAF-12`، `CUST-ENG-005`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت إضافةُ العرض تنادي `Cart.add` مباشرةً بلا حارس** — **فيُضاف عرضٌ
 * إلى نقطةٍ لا نصلها ثمّ يُردُّ عند الإرسال.** فوجب أن تمرّ بالبوّابة نفسِها
 * التي تمرّ بها إضافةُ السوق (`rememberAddBlocked` ⇐ `Orderable`/`addActionFor`).
 *
 * **اختبارُ مصدرٍ**: العقدُ لا يُقاس بلقطة.
 */
class OfferGateTest {

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

    /** **البوّابةُ المشتركةُ موجودةٌ وتعتمد المسارَ نفسَه.** */
    @Test
    fun sharedGateReusesPreCartPath() {
        val s = read("app-customer/src/main/kotlin/com/rahalgo/customer/PreCart.kt")
        assertTrue(
            "**`rememberAddBlocked` غائبة**",
            s.contains("fun rememberAddBlocked("),
        )
        assertTrue(
            "**البوّابةُ لا تعتمد addActionFor المركزيّة**",
            s.contains("addActionFor(ctx0, availability) == AddAction.BLOCKED"),
        )
    }

    /** **العروضُ تحسب `blocked` وتحرس كلَّ إضافة.** */
    @Test
    fun offersGuardEveryAdd() {
        val s = read("app-customer/src/main/kotlin/com/rahalgo/customer/mine/MineScreens.kt")
        assertTrue(
            "**العروضُ لا تستدعي البوّابة**",
            s.contains("rememberAddBlocked(address)"),
        )
        // **ولا `Cart.add` بلا حارسٍ في العروض** — كلُّ استدعاءٍ داخل فرعِ !blocked.
        val offers = s.substringAfter("private fun Offers(")
        assertTrue(
            "**إضافةٌ في العروض بلا حارس `blocked`**",
            offers.contains("if (blocked)"),
        )
    }
}
