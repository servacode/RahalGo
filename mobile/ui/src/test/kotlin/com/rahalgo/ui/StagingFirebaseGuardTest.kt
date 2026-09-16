package com.rahalgo.ui

import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ======================================================================
 * **حارسُ الحارس — فايربيسُ التجهيز لا تُنزَع** (`P-8`، ٢٠٢٦-٠٩-١٦)
 * ======================================================================
 *
 * # ولمَ فحصٌ يقرأ نصَّ بناء
 *
 * **وسقوطُ البناء في `mobile/build.gradle.kts` هو الحارسُ الحقيقيّ** —
 * **وهذا يحرسه من أن يُنزَع.** **ومن حذف الكتلةَ ليمرّ بناؤه لا يُنبّهه
 * شيء**: **الحرّاسُ الساكتون يُحذفون أوّلاً.**
 *
 * # وما وقع قبله
 *
 * **وكلُّ تطبيقٍ يُطبّق إضافةَ فايربيس بشرط** — **وإن لم يصحّ مضى
 * البناءُ ناجحاً وأنتج أثراً بلا دفعٍ أصلاً.** (قِيس ٢٠٢٦-٠٩-١٦
 * بتنحية ملفّ المندوب: البناءُ لم يسقط، وتبخّرت مهمّةُ
 * `processDebugGoogleServices`.)
 *
 * **وأثرُ قبولٍ بلا دفعٍ يُنصَّب ويعمل** — **فتُجرَّب الإشعاراتُ فلا
 * يصل شيء، ويُقرأ عيباً في المنصّة وهو نقصُ إعداد.**
 */
class StagingFirebaseGuardTest {

    private fun rootScript(): String {
        var dir = java.io.File("").absoluteFile
        repeat(6) {
            val f = java.io.File(dir, "build.gradle.kts")
            if (f.exists() && java.io.File(dir, "settings.gradle.kts").exists()) {
                // **وgit على ويندوز يكتب `CRLF`** — **وحارسٌ يطابق `LF`
                // يحمرّ على شجرةٍ سليمة.**
                return f.readText().replace("\r\n", "\n")
            }
            dir = dir.parentFile ?: return@repeat
        }
        throw IllegalStateException("لم أجد نصَّ بناء الجذر")
    }

    /** **والحزمُ الأربعُ مذكورةٌ بأعيانها** — **ولا يُنسى الخامس.** */
    @Test
    fun `حزمُ التصحيح الأربعُ في حارس فايربيس`() {
        val script = rootScript()
        for (pkg in listOf(
            "com.rahalgo.customer.debug",
            "com.rahalgo.driver.debug",
            "com.rahalgo.merchant.debug",
            "com.rahalgo.rep.debug",
        )) {
            assertTrue(
                "**غابت " + pkg + " عن حارس فايربيس في نصّ الجذر**",
                script.contains(pkg),
            )
        }
    }

    /** **ويُسقط البناءَ ولا يحذّر.** */
    @Test
    fun `حارسُ فايربيس يرمي ولا يسجّل تحذيراً`() {
        val script = rootScript()
        assertTrue(
            "**ذهبت كتلةُ حارس فايربيس من نصّ الجذر**",
            script.contains("fun p8Halt(") && script.contains("throw GradleException("),
        )
        assertTrue(
            "**لم يعد الحارسُ يعمل عند رايةِ P-8**",
            script.contains("if (p8Build) {"),
        )
    }

    /** **وما يُتحقَّق منه في كلّ ملفّ** — **ولا يُكتفى بوجوده.** */
    @Test
    fun `حارسُ فايربيس يفحص المشروعَ والحزمةَ والمعرّف`() {
        val script = rootScript()
        for (key in listOf("project_id", "project_number", "mobilesdk_app_id", "package_name")) {
            assertTrue(
                "**لم يعد الحارسُ يفحص " + key + "**",
                script.contains(key),
            )
        }
        assertTrue(
            "**لم يعد مشروعُ الإنتاج مرفوضاً للتجهيز**",
            script.contains("rahalgo-prod"),
        )
        assertTrue(
            "**لم يعد المعرّفُ يُمرَّر خاصّيّةً — وربّما اختُلق ثابتاً**",
            script.contains("rahalgo.stagingFirebaseProject"),
        )
    }

    /** **وملفُّ التجهيز في `src/debug` لا فوقَ ملفِّ الإنتاج.** */
    @Test
    fun `ملفُّ التجهيز في مجلَّد التصحيح`() {
        assertTrue(
            "**تبدّل موضعُ ملفّ تجهيز فايربيس**",
            rootScript().contains("/src/debug/google-services.json"),
        )
    }
}
