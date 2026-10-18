package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **كلُّ رمزِ سببٍ يكتبه المحرّكُ في شكوى أو بلاغ أو إنذار له اسمٌ عربيّ** —
 * يُقرأ من ملفّات `backend/internal/support` لا من قائمةٍ منسوخة (بلاغُ المالك
 * ٢٠٢٦-١٠-١٨: شكاوى المتجر تُعرض «late»). **والمجهولُ «سبب آخر» لا رمزُه.**
 */
class TicketReasonsTest {

    private fun repoRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "backend/internal/support").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ المستودع من " + File("").absolutePath)
    }

    private fun go(name: String) = File(repoRoot(), "backend/internal/support/$name").readText()

    private fun codes(src: String): Set<String> =
        Regex("""Code:\s*"([a-z_]+)"""").findAll(src).map { it.groupValues[1] }.toSet()

    /** **أسبابُ الشكوى والبلاغين** — كلُّها تقع على طلبات المتجر. */
    private fun ticketCodes(): Set<String> {
        val complaint = codes(go("complaints.go").substringAfter("var ComplaintReasons").substringBefore("\n}\n"))
        val driver = codes(go("driver_report.go"))
        val merchant = Regex(""""([a-z_]+)",""").findAll(
            go("merchant_report.go").substringAfter("var MerchantReportReasons = []string{").substringBefore("}"),
        ).map { it.groupValues[1] }.toSet()
        assertTrue("لم تُقرأ أسبابُ الزبون", complaint.size >= 8)
        assertTrue("لم تُقرأ أسبابُ السائق", driver.size >= 9)
        assertTrue("لم تُقرأ أسبابُ المتجر", merchant.size >= 4)
        return complaint + driver + merchant
    }

    @Test
    fun everyBackendTicketReasonHasArabicName() {
        val all = ticketCodes()
        assertTrue("late من بلاغ المالك", "late" in all)
        val missing = all.filter { TicketReasons.known(it) == null }
        assertEquals("أسبابٌ بلا اسمٍ عربيّ", emptyList<String>(), missing)
        // **وكلُّ سببٍ غيرُ «أخرى» له اسمُه لا «سبب آخر».**
        val swallowed = all.filter { it != "other" && TicketReasons.res(it) == R.string.rs_other }
        assertEquals("أسبابٌ تُقرأ «سبب آخر»", emptyList<String>(), swallowed)
    }

    @Test
    fun everyBackendWarnReasonHasArabicName() {
        val all = codes(go("warn_reasons.go"))
        assertTrue("لم تُقرأ أسبابُ الإنذار", all.size >= 8)
        val missing = all.filter { WarnReasons.known(it) == null }
        assertEquals("أسبابُ إنذارٍ بلا اسم", emptyList<String>(), missing)
    }

    @Test
    fun unknownOrEmptyNeverRaw() {
        assertEquals(R.string.rs_other, TicketReasons.res("brand_new_code"))
        assertEquals(R.string.rs_other, TicketReasons.res(""))
        assertEquals(R.string.rs_other, WarnReasons.res("brand_new_code"))
        assertNotEquals(R.string.rs_other, TicketReasons.res("late"))
    }

    /** **والألفاظُ نفسُها عربيّةٌ** — لا حرفَ لاتينيّاً فيها. */
    @Test
    fun namesAreArabic() {
        val xml = File(repoRoot(), "mobile/ui/src/main/res/values/strings.xml").readText()
        val names = Regex("""<string name="((tr|wr)_[a-z_]+|rs_other|rs_driver_conduct|tik_progress|wal_k_other|ord_st_refunded)">([^<]*)</string>""")
            .findAll(xml).map { it.groupValues[1] to it.groupValues[3] }.toList()
        assertTrue("لم تُقرأ الألفاظ", names.size >= 20)
        val latin = names.filter { (_, v) -> v.any { it in 'a'..'z' || it in 'A'..'Z' } }
        assertEquals("ألفاظٌ فيها حروفٌ لاتينيّة", emptyList<Pair<String, String>>(), latin)
    }
}
