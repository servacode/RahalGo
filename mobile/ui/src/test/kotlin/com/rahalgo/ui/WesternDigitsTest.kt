package com.rahalgo.ui

import java.io.File
import java.util.Locale
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **الأرقامُ لاتينيّةٌ في كلّ شاشة.**
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٣: «الأرقام كلُّها بالأجنبيّة (0123456789) في كلّ
 *  مكان، مركزيّاً، وفاصلُ الآلاف «,»».)
 */
class WesternDigitsTest {

    /** **`%d` بلغةٍ عربيّةٍ يخرج لاتينيّاً** — وهو ما يمرّ به كلُّ نصٍّ من الموارد. */
    @Test
    fun `التنسيقُ بالعربيّة يُخرج أرقاماً لاتينيّة`() {
        val ar = WesternDigits.locale(Locale.forLanguageTag("ar-SY"))
        assertEquals("ar", ar.language)
        assertEquals("محادثة الطلب #511", String.format(ar, "محادثة الطلب #%d", 511))
        assertEquals("3 أصناف", String.format(ar, "%d أصناف", 3))
        // **وبلا التحويل كان يخرج هنديّاً** — وهو العطبُ نفسُه.
        assertEquals("٥١١", String.format(Locale.forLanguageTag("ar-EG"), "%d", 511))
    }

    @Test
    fun `النصُّ الواردُ بأرقامٍ هنديّةٍ يصير لاتينيّاً`() {
        assertEquals("12,500", WesternDigits.text("١٢٬٥٠٠"))
        assertEquals("2026-10-03", WesternDigits.text("۲۰۲۶-10-03"))
        assertEquals("بلا أرقام", WesternDigits.text("بلا أرقام"))
        // **والمبالغُ لاتينيّةٌ بفاصلة «,» أصلاً.**
        assertEquals("12,500 ل.س", money(12_500))
    }

    /** **والتطبيقُ يلفّ سياقَه ولغتَه** — وإلّا بقي كلُّ هذا دالّةً لا تُنادى. */
    @Test
    fun `تطبيقُ الزبون يلفّ سياقَه`() {
        var dir = File("").absoluteFile
        while (!File(dir, "settings.gradle.kts").exists()) dir = dir.parentFile
        val main = File(dir, "app-customer/src/main/kotlin/com/rahalgo/customer/MainActivity.kt").readText()
        val app = File(dir, "app-customer/src/main/kotlin/com/rahalgo/customer/CustomerApplication.kt").readText()
        assertTrue(main.contains("WesternDigits.wrap(newBase)"))
        assertTrue(app.contains("WesternDigits.applyDefault()"))
    }
}
