package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مفتاحُ المحاولة يثبت في كلّ تطبيق** — `DUP-LEAD`، ٢٠٢٦-٠٩-٣٠
 * ══════════════════════════════════════════════════════════════════════
 *
 * **رُئي على جهاز المالك**: عميلٌ أُرسل من تطبيق المندوب وانقطعت الشبكةُ
 * قبل الردّ، ثمّ أُعيد — **فسُجِّل مرّتين.**
 *
 * **والحراسةُ في الخادم سليمة** (`WithIdempotentTx`)، **والتطبيقُ يرسل
 * المفتاح** (`Attempt.LEAD`) — **لكنّ مخزنه كان يُركَّب في تطبيق الزبون
 * وحدَه**، **وبلا مخزنٍ يُولِّد `Attempt.key` مفتاحاً جديداً في كلّ نداء.**
 */
class AttemptWiringTest {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        while (dir.parentFile != null) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
            dir = dir.parentFile
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    /** **النواةُ المشتركةُ تُركّبه** — فيصل كلَّ تطبيقٍ يُقلع بها. */
    @Test
    fun coreInstallsTheAttemptStore() {
        val core = File(mobileRoot(), "ui/src/main/kotlin/com/rahalgo/ui/Core.kt").readText()
        assertTrue(
            "**`AppCore.install` لا يُركّب `Attempt`** — فتطبيقٌ غيرُ الزبون يولّد مفتاحاً جديداً لكلّ محاولة",
            core.contains("Attempt.install(context)"),
        )
    }

    /** **والشاهدُ السلوكيّ**: بلا مخزنٍ مفتاحان مختلفان، وبه مفتاحٌ واحد. */
    @Test
    fun keyIsStableOnlyWithAStore() {
        Attempt.resetForTest(null)
        assertNotEquals(Attempt.key(Attempt.LEAD), Attempt.key(Attempt.LEAD))

        val mem = HashMap<String, String>()
        Attempt.resetForTest(object : Attempt.Store {
            override fun get(key: String): String? = mem[key]
            override fun put(key: String, value: String) { mem[key] = value }
            override fun remove(key: String) { mem.remove(key) }
        })
        assertEquals(Attempt.key(Attempt.LEAD), Attempt.key(Attempt.LEAD))
        Attempt.resetForTest(null)
    }
}
