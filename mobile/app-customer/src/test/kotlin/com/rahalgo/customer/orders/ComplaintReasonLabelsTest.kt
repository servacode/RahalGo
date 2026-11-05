package com.rahalgo.customer.orders

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **كلُّ سببِ شكوى يعرضه المحرّكُ على الزبون له سطرٌ باسمه في شاشة الشكوى** —
 * يُقرأ من `backend/internal/support/complaints.go` لا من قائمةٍ منسوخة.
 *
 * **وسببٌ بلا سطرٍ يُقرأ «أخرى»** (`reasonLabel`): «السائق طلب رقمي» (قرارُ المالك
 * ٢٠٢٦-١١-٠٥) يصير «أخرى» ثانيةً — **فيختارها الزبونُ يريد «أخرى» فيفتح بلاغاً على
 * سائقٍ بالغلط.**
 */
class ComplaintReasonLabelsTest {

    private fun repoRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "backend/internal/support").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ المستودع من " + File("").absolutePath)
    }

    @Test
    fun everyComplaintReasonHasItsOwnLabel() {
        val root = repoRoot()
        val goSrc = File(root, "backend/internal/support/complaints.go").readText()
            .substringAfter("var ComplaintReasons").substringBefore("\n}\n")
        val codes = Regex("""Code:\s*"([a-z_]+)"""").findAll(goSrc).map { it.groupValues[1] }.toSet()
        assertTrue("لم تُقرأ أسبابُ الشكوى", codes.size >= 8)
        assertTrue("سببُ «طلب رقمي» ليس بين أسباب المحرّك", "off_platform" in codes)

        val screen = File(root, "mobile/app-customer/src/main/kotlin/com/rahalgo/customer/orders/OrdersScreen.kt")
            .readText().substringAfter("private fun reasonLabel(").substringBefore("\n}\n")
        val labelled = Regex(""""([a-z_]+)"\s*->""").findAll(screen).map { it.groupValues[1] }.toSet()
        assertEquals("أسبابٌ بلا سطرٍ في reasonLabel", emptySet<String>(), codes - labelled)
    }
}
