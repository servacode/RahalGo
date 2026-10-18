package com.rahalgo.customer.shop

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **«+» المطفأ في صفحة المتجر يقول لماذا** (٢٠٢٦-١٠-١٨) — باللافتة نفسِها
 * التي في السوق (`ServiceBlockNotice` ولافتةُ الاستعراض)، **لا نسخةٌ ثانية.**
 */
class StoreNoticeWiringTest {

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
    fun storePageShowsTheShopsNotice() {
        val shop = src("shop/ShopScreen.kt")
        val store = src("shop/StoreScreen.kt")
        // **لافتةٌ واحدةٌ تُركَّب مرّتين** — في السوق وفي صفحة المتجر.
        assertEquals("لافتة السوق كُتبت مرّتين", 1, Regex("""val addNotice\b""").findAll(shop).count())
        assertTrue("السوق لا يرسم اللافتة", shop.contains("            addNotice()"))
        assertTrue("صفحة المتجر لا تأخذ اللافتة", shop.contains("notice = addNotice"))
        val body = shop.substringAfter("val addNotice").substringBefore("// **صفحةُ المتجر فوق السوق**")
        assertTrue("اللافتة ليست ServiceBlockNotice", body.contains("ServiceBlockNotice("))
        assertTrue("لافتة الاستعراض غائبة", body.contains("preview_banner_named"))
        assertTrue("صفحة المتجر لا ترسم اللافتة", store.contains("notice()"))
        // **ولا نسخةَ من السبب في صفحة المتجر** — تقرأ اللافتةَ لا تكتبها.
        assertTrue("صفحة المتجر تكتب سبباً خاصّاً بها", !store.contains("ServiceBlockNotice("))
    }
}
