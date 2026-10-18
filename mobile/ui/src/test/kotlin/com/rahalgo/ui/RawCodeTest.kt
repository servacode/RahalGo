package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ولا رمزَ آليٌّ أمام الناس — في التطبيقات الأربعة و`:ui`** (٢٠٢٦-١٠-١٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك: شكاوى المتجر تُعرض «late».) **وكانت ترجماتٌ كثيرةٌ تقول
 * «والمجهولُ يُعرض برمزه ليُبلَّغ عنه»** — فيقرأ الناسُ رمزاً إنكليزيّاً. **والآن
 * المجهولُ لفظٌ عامّ** («سبب آخر»، «قيد المعالجة»، «حركة تانية»)، **والحارسُ
 * يمنع عودةَ النمط.**
 */
class RawCodeTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-merchant").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    /** **ترجمةٌ تردّ مفتاحَها حين تجهله** — `else -> code` أو `?: status`. */
    private val raw = Regex(
        """(else\s*->\s*(code|status|key|kind|reason)\s*$)|(\?:\s*(code|status|key|kind|reason)\s*\)?\s*$)""",
        RegexOption.MULTILINE,
    )

    @Test
    fun noTranslatorReturnsItsRawCode() {
        val root = mobileRoot()
        val hits = mutableListOf<String>()
        for (mod in listOf("app-customer", "app-merchant", "app-driver", "app-rep", "ui")) {
            File(root, "$mod/src/main/kotlin").walkTopDown().filter { it.extension == "kt" }.forEach { f ->
                f.readLines().forEachIndexed { i, line ->
                    if (raw.containsMatchIn(line)) hits += "${f.relativeTo(root)}:${i + 1}: ${line.trim()}"
                }
            }
        }
        assertEquals("مواضعُ تعرض الرمزَ الخامّ", emptyList<String>(), hits)
    }

    private fun src(path: String) = File(mobileRoot(), path).readText()

    /** **شكاوى المتجر من الموضع المركزيّ** — والإنذارُ برمزه مترجَماً. */
    @Test
    fun merchantComplaintsAndWarningsAreTranslated() {
        val drawer = src("app-merchant/src/main/kotlin/com/rahalgo/merchant/drawer/DrawerScreens.kt")
        assertTrue("reasonAr لا يقرأ TicketReasons", drawer.contains("com.rahalgo.ui.TicketReasons.text("))
        assertTrue("الإنذار يُعرض برمزه", !drawer.contains("Text(w.reason"))
        assertTrue("الإنذار لا يقرأ WarnReasons", drawer.contains("WarnReasons.res(w.reason)"))
        val mine = src("app-customer/src/main/kotlin/com/rahalgo/customer/mine/MineScreens.kt")
        assertTrue("شكوى الزبون بلا موضوع تُعرض برمزها", !mine.contains("ifEmpty { t.reason }"))
    }

    /** **كلُّ حالٍ للطلب في المحرّك لها اسمٌ في بطاقة الزبون.** */
    @Test
    fun customerStatusCoversBackendStatuses() {
        val goSrc = File(mobileRoot().parentFile, "backend/internal/orders/statuses.go").readText()
        val statuses = Regex("""(?m)^\s*St[A-Z][A-Za-z]*\s*=\s*"([a-z_]+)"""").findAll(goSrc).map { it.groupValues[1] }.toSet()
        assertTrue("لم تُقرأ الحالات", statuses.size >= 14)
        val status = src("app-customer/src/main/kotlin/com/rahalgo/customer/orders/Status.kt")
            .substringAfter("fun statusText(").substringBefore("fun statusColor(")
        val missing = statuses.filter { !status.contains("\"$it\"") }
        assertEquals("حالاتٌ بلا اسمٍ عند الزبون", emptyList<String>(), missing)
    }
}
