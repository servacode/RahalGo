package com.rahalgo.ui

import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * **قلبُ صورة الصنف إلى وصفه** (قرارُ المالك ٢٠٢٦-١٠-٢١): ضغطةٌ تقلبها بنفس الحجم،
 * «لا يوجد وصف لهذا الصنف» حين يغيب، **وترجع وحدَها** — «ماتظل عالقة ع وصف».
 * حراسةُ مصدرٍ كأخواتها — الدورانُ يحتاج جهازاً.
 */
class ItemFlipTest {

    private fun read(rel: String): String {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "app-customer").exists()) return File(dir, rel).readText().replace("\r\n", "\n")
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile")
    }

    @Test
    fun cardFlipsToDescriptionAndReturnsByItself() {
        val s = read("app-customer/src/main/kotlin/com/rahalgo/customer/shop/ShopScreen.kt")
        assertTrue("**لا قلبَ للصورة**", s.contains("rotationY = turn"))
        assertTrue("**الظهرُ لا يقرأ الوصف**", s.contains("item.description.trim()"))
        assertTrue("**بلا وصفٍ لا يُقال شيء**", s.contains("R.string.shop_no_description"))
        assertTrue("**لا رجوعَ تلقائيّاً**", s.contains("coerceAtMost(10_000L)") && s.contains("flipped = false"))
    }
}
