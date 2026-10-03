package com.rahalgo.driver

import com.rahalgo.ui.WesternDigits
import com.rahalgo.ui.dist
import com.rahalgo.ui.money
import org.junit.Assert.assertEquals
import org.junit.Assume.assumeTrue
import org.junit.Test
import java.util.Locale

/**
 * **الأرقامُ بالأجنبيّة في كلّ مكان** (قرارُ المالك ٢٠٢٦-١٠-٠٣) — قِيس على جهازه:
 * «الطلب #١٣٦٧ لم يعد معك». انظر `WesternDigits`.
 */
class WesternDigitsTest {

    private val ar = Locale("ar")

    @Test
    fun arabicFormatsHinduDigitsWithoutTheFix() {
        // **الضابط**: العربيّةُ بلا العلاج تكتب أرقاماً هنديّة — وهو ما رآه المالك.
        val raw = String.format(ar, "%d", 1367)
        assumeTrue("هذا الـJDK لا يكتب الهنديّة للعربيّة أصلاً", raw != "1367")
    }

    @Test
    fun arabicWithLatinNumberingWritesWesternDigits() {
        val l = WesternDigits.locale(ar)
        assertEquals("ar", l.language)
        assertEquals("الطلب #1367 لم يعد معك", String.format(l, "الطلب #%1\$d لم يعد معك", 1367))
        assertEquals("2:05", String.format(l, "%d:%02d", 2, 5))
        assertEquals("12.5", String.format(l, "%.1f", 12.5))
    }

    @Test
    fun centralFormattersUseWesternDigitsAndComma() {
        assertEquals("3,000 ل.س", money(3000))
        assertEquals("320 م", dist(320.0))
    }
}
