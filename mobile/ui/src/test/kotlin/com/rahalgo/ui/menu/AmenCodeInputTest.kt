package com.rahalgo.ui.menu

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **رقمُ الأمين والباركود** (مراجعةُ ما قبل الإصدار ٢٠٢٦-١١-٠٥).
 *
 * الكيبوردُ العربيّ يكتب «٠١٧٠٦٤» وملفُّ الأمين «017064» — **فكان الصنفُ المربوطُ بيده
 * لا يطابق أبداً.** فيُحوَّل كما يُكتب (`amenCodeInput`)، **وكيبوردُ الحقلين لاتينيّ**،
 * **ورسالةُ رفضه بأرقامٍ لاتينيّة** (قرارُ المالك: الأرقامُ كلُّها بالأجنبيّة).
 */
class AmenCodeInputTest {

    @Test
    fun arabicAndPersianDigitsBecomeWestern() {
        assertEquals("017064", amenCodeInput("٠١٧٠٦٤"))
        assertEquals("6281000123", amenCodeInput("۶۲۸۱۰۰۰۱۲۳"))
        assertEquals("A-12", amenCodeInput("A-١٢"))
        assertEquals("017064", amenCodeInput("017064"))
    }

    @Test
    fun spacesDroppedAndCappedAtServerLimit() {
        assertEquals("017064", amenCodeInput(" ٠١٧ ٠٦٤ "))
        assertEquals("9".repeat(64), amenCodeInput("٩".repeat(70)))
    }

    /** **والحقلان يمرّان به وبكيبوردٍ لاتينيّ** — وإلّا بقي دالّةً لا تُنادى. */
    @Test
    fun editorFieldsUseItWithLatinKeyboard() {
        val src = source("ui/src/main/kotlin/com/rahalgo/ui/menu/ItemEditor.kt")
        val block = src.substringAfter("if (amenLinkAllowed) {").substringBefore("ModifiersEditor(")
        assertEquals(2, Regex("""amenCodeInput\(v\)""").findAll(block).count())
        assertEquals(2, Regex("""KeyboardType\.Ascii""").findAll(block).count())
        // **وبلا تصحيحٍ تلقائيّ** — مرآةُ `autoCorrect="off"` في `ItemEditor.tsx`.
        assertEquals(2, Regex("""autoCorrectEnabled = false""").findAll(block).count())
        assertFalse("حقلٌ يتجاوز التحويل", block.contains("v.filterNot("))
    }

    @Test
    fun rejectionMessageHasWesternDigits() {
        val xml = source("ui/src/main/res/values/strings.xml")
        val msg = Regex("""<string name="err_external_code_invalid">([^<]*)</string>""").find(xml)!!.groupValues[1]
        assertTrue(msg, msg.contains("64"))
        assertFalse(msg, msg.any { it in '٠'..'٩' || it in '۰'..'۹' })
    }

    /** **ورسائلُ مزامنة الأمين كلُّها بأرقامٍ لاتينيّة** — «حتى ٥٠ ألف مادة» كانت هنديّة. */
    @Test
    fun priceSyncMessagesHaveWesternDigits() {
        val xml = source("ui/src/main/res/values/strings.xml")
        val msgs = Regex("""<string name="err_(?:price_sync|external_code)_[a-z_]+">([^<]*)</string>""")
            .findAll(xml).map { it.groupValues[1] }.toList()
        assertTrue("لا رسائل", msgs.size >= 3)
        for (msg in msgs) assertFalse(msg, msg.any { it in '٠'..'٩' || it in '۰'..'۹' })
        val empty = Regex("""<string name="err_price_sync_empty">([^<]*)</string>""").find(xml)!!.groupValues[1]
        assertTrue(empty, empty.contains("50"))
    }

    private fun source(path: String): String {
        var dir = File("").absoluteFile
        while (!File(dir, "settings.gradle.kts").exists()) dir = dir.parentFile
        return File(dir, path).readText()
    }
}
