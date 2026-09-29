package com.rahalgo.driver

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حارسُ عزل مختبر الرحلة التجريبيّة — لا وجودَ له في الإنتاج**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٩-٢٨، البند ١: «يجب ألّا يظهر أو يُوصَل إليه في
 *  الإنتاج/الإصدار… أضِف اختباراً يثبت أنّ الإنتاج لا يمكنه كشفَه».)
 *
 * # **ثلاثةُ أقفالٍ مستقلّة**
 *
 * ١) **الشيفرةُ في `src/debug` وحدَها** — فلا تُصرَّف في الإصدار البتّة،
 *    ولا سطرَ منها في `src/main`.
 * ٢) **النشاطُ مُعلَنٌ في بيان التطوير وحدَه** — وغائبٌ عن بيان الإصدار.
 * ٣) **لا مدخلَ له في واجهة السائق أصلاً** — لا في الرئيسيّة ولا على شاشة
 *    الرحلة الحقيقيّة؛ يُطلَق بـ`adb` للهندسة وحدَها.
 *
 * **وحارسٌ يقرأ الملفّاتِ لا النيّة**: من نقل ملفاً إلى `src/main`، أو
 * أعلن النشاطَ في البيان الرئيسيّ، أو نزع الحارسَ — يُسقط هذا الفحص.
 */
class ReplayLabIsolationTest {

    private val moduleDir = File("").absoluteFile
    private val debugKotlin = File(moduleDir, "src/debug/kotlin/com/rahalgo/driver/replaylab")
    private val mainKotlin = File(moduleDir, "src/main/kotlin/com/rahalgo/driver")

    /** ١) الشيفرةُ في `src/debug` وحدَها — وغائبةٌ عن `src/main`. */
    @Test
    fun `lab code lives only in the debug source set`() {
        assertTrue(
            "شيفرةُ المختبر ليست في src/debug: " + debugKotlin.absolutePath,
            File(debugKotlin, "ReplayLabActivity.kt").exists() &&
                File(debugKotlin, "ReplayLabController.kt").exists() &&
                File(debugKotlin, "ReplayLabScreen.kt").exists() &&
                File(debugKotlin, "ReplayTripActivity.kt").exists() &&
                File(debugKotlin, "ReplayTripController.kt").exists(),
        )
        // **ولا ملفَّ مختبرٍ في `src/main`** — فلا يدخل الإصدار.
        val leakedInMain = File(mainKotlin, "replaylab").exists()
        assertFalse("شيفرةُ المختبر تسرّبت إلى src/main فتدخل الإصدار", leakedInMain)
    }

    /** ٢) النشاطُ في بيان التطوير وحدَه — لا في البيان الرئيسيّ. */
    @Test
    fun `lab activity is declared in the debug manifest only`() {
        val debugManifest = File(moduleDir, "src/debug/AndroidManifest.xml").readText()
        assertTrue(
            "بيانُ التطوير لا يُعلن نشاطَي المختبر/الرحلة",
            debugManifest.contains("com.rahalgo.driver.replaylab.ReplayLabActivity") &&
                debugManifest.contains("com.rahalgo.driver.replaylab.ReplayTripActivity"),
        )
        assertTrue(
            "بيانُ التطوير لا يحمل فعلَي الإطلاق",
            debugManifest.contains("com.rahalgo.driver.debug.REPLAY_LAB") &&
                debugManifest.contains("com.rahalgo.driver.debug.REPLAY_TRIP"),
        )
        val mainManifest = File(moduleDir, "src/main/AndroidManifest.xml").readText()
        assertFalse(
            "**البيانُ الرئيسيُّ يُعلن نشاطَ المختبر/الرحلة** — فيدخل الإصدار",
            mainManifest.contains("ReplayLabActivity") || mainManifest.contains("ReplayTripActivity") ||
                mainManifest.contains("REPLAY_LAB") || mainManifest.contains("REPLAY_TRIP"),
        )
    }

    /**
     * ٣) **لا مدخلَ للمختبر/الإعادة في واجهة السائق البتّة** — حُذف من المسار
     *    العاديّ (قرارُ المالك ٢٠٢٦-٠٩-٢٨: «أزِل المداخلَ التي يبلغها سائقٌ
     *    حقيقيّ»). **والمحرّكُ يبقى في `src/debug` (يُطلَق بـ`adb`)، ولا زرَّ
     *    في التطبيق.**
     */
    @Test
    fun `no user-facing lab or replay entry in the driver UI`() {
        // **الرئيسيّة: لا فعلَ إطلاقٍ للإعادة/المختبر.**
        val home = File(mainKotlin, "home/HomeScreen.kt").readText()
        assertFalse(
            "**الرئيسيّة تحوي مدخلَ إعادة/مختبر** — يبلغه سائقٌ حقيقيّ",
            home.contains("com.rahalgo.driver.debug.REPLAY_TRIP") ||
                home.contains("com.rahalgo.driver.debug.REPLAY_LAB"),
        )
        // **وشاشةُ الرحلة الحقيقيّة: لا زرَّ «رحلة تجريبيّة» (`ReplayButton`).**
        val trip = File(mainKotlin, "trip/TripScreen.kt").readText()
        assertFalse(
            "**شاشةُ الرحلة تحوي زرَّ إعادةٍ** — يبلغه السائقُ على رحلةٍ حقيقيّة",
            trip.contains("ReplayButton("),
        )
    }
}
