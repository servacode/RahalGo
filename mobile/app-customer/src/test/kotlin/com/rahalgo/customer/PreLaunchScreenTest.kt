package com.rahalgo.customer

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ما قبل الافتتاح في تطبيق الزبون** (`PL`، ٢٠٢٦-٠٩-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العطبُ الذي يُحرَس منه
 *
 * **والمنصّةُ تُنزَّل قبل أن تُفتح** — **وكان التطبيقُ لا يعلم**:
 * **يرسم شاشةَ سوقٍ، يطرق البابَ، فيُردّ ٥٠٣، فيبدّلها رسالةً حمراء.**
 *
 * **وحالٌ مقصودةٌ تُقرأ خطأً تُرى عطباً** — **ومن رآها حذف التطبيقَ
 * ولم يعد.**
 *
 * # ولمَ يُقرأ النصُّ لا تُشغَّل الشاشة
 *
 * **وشاشةُ Compose تحتاج جهازاً أو مُحاكياً** — **وهذه فحوصُ وحدة.**
 * **فيُقرأ المصدرُ نصّاً**: **يُثبت أنّ الفرعَ موجودٌ وأنّ ما فيه ليس
 * دوّاراً ولا خطأً.** **والسلوكُ على جهازٍ يبقى بندَ ميدان.**
 */
class PreLaunchScreenTest {

    private companion object {
        /** **وgit على ويندوز يكتب `CRLF`** — وحارسٌ يطابق `LF` يحمرّ باطلاً. */
        fun read(rel: String): String =
            java.io.File(appModuleDir(), rel).readText().replace("\r\n", "\n")

        fun appModuleDir(): java.io.File {
            var dir = java.io.File("").absoluteFile
            repeat(6) {
                if (java.io.File(dir, "build.gradle.kts").exists() &&
                    java.io.File(dir, "src/main").exists()
                ) {
                    return dir
                }
                dir = dir.parentFile ?: return@repeat
            }
            throw IllegalStateException("لم أجد مجلَّد الوحدة")
        }

        const val MAIN = "src/main/kotlin/com/rahalgo/customer/MainActivity.kt"
        const val SERVING = "src/main/kotlin/com/rahalgo/customer/Serving.kt"

        /** **جسمُ شاشة ما قبل الافتتاح وحدَه** — لا الملفُّ كلُّه. */
        fun preLaunchBody(): String {
            val src = read(MAIN)
            val i = src.indexOf("private fun PreLaunch(")
            check(i >= 0) { "**ذهبت شاشةُ ما قبل الافتتاح**" }
            val j = src.indexOf("\n}\n", i)
            check(j > i) { "**لم أجد نهايةَ الشاشة**" }
            return src.substring(i, j)
        }
    }

    // ── PL-08 ────────────────────────────────────────────────────────

    /**
     * **التصفّحُ مغلقٌ ⇒ شاشةُ حالٍ مقصودة — لا دوّار.**
     */
    @Test
    fun `التصفّحُ المغلقُ يرسم شاشةَ الحال لا دوّاراً`() {
        val main = read(MAIN)
        assertTrue(
            "**ذهب فرعُ ما قبل الافتتاح من موزّع الشاشات**",
            main.contains("Serving.preLaunch") && main.contains("PreLaunch("),
        )
        val body = preLaunchBody()
        for (spinner in listOf("CircularProgressIndicator", "LoadingState", "LinearProgressIndicator")) {
            assertFalse(
                "**دوّارٌ في شاشةِ حالٍ لا شيءَ يُنتظَر فيها**: $spinner",
                body.contains(spinner),
            )
        }
    }

    // ── PL-19 ────────────────────────────────────────────────────────

    /**
     * **وليست خطأً** — **ولا تُرسم بلون خطأٍ ولا تمرّ بمترجم الأخطاء.**
     */
    @Test
    fun `شاشةُ ما قبل الافتتاح ليست حالَ خطأ`() {
        val body = preLaunchBody()
        for (err in listOf("apiErrorText", "ApiErrors", "colors.danger", "colors.error", "IconWarning")) {
            assertFalse("**حالٌ مقصودةٌ رُسمت خطأً**: $err", body.contains(err))
        }
    }

    // ── PL-09 ────────────────────────────────────────────────────────

    /**
     * **ولا زرَّ إنشاءِ حساب** — **وبابٌ يردّ المحرّكُ طارقَه عبثٌ.**
     */
    @Test
    fun `لا بابَ إنشاءِ حسابٍ في ما قبل الافتتاح`() {
        val body = preLaunchBody()
        for (signup in listOf("signup", "Signup", "إنشاء حساب")) {
            assertFalse("**بابُ إنشاءٍ معروضٌ والتسجيلُ مغلق**: $signup", body.contains(signup))
        }
    }

    // ── PL-10 ────────────────────────────────────────────────────────

    /**
     * **والدخولُ يبقى** — **وإغلاقُ السوق ليس إغلاقَ الباب.**
     */
    @Test
    fun `بابُ الدخول يبقى في ما قبل الافتتاح`() {
        val body = preLaunchBody()
        assertTrue(
            "**ذهب بابُ الدخول من شاشة ما قبل الافتتاح**",
            body.contains("onAskLogin") && body.contains("prelaunch_login"),
        )
        // **وحسابُ من دخل يبقى مفتوحاً** — **الفرعُ يستثني تبويبَ الحساب.**
        assertTrue(
            "**شاشةُ الحال ابتلعت تبويبَ الحساب أيضاً**",
            read(MAIN).contains("Serving.preLaunch && tab != Tab.Account"),
        )
    }

    // ── PL-16 · PL-17 ────────────────────────────────────────────────

    /**
     * **وتُقرأ الحالُ من المنصّة وتُجدَّد** — **فلا يلزم تحديثُ حزمة.**
     *
     * **ومن فتح المالكُ السوقَ له وجب أن يراه بعد التجديد** — **لا بعد
     * تنصيبٍ جديد.**
     */
    @Test
    fun `حالُ الافتتاح تُقرأ من المنصّة وتُجدَّد`() {
        val serving = read(SERVING)
        assertTrue(
            "**لم تعد حالُ الافتتاح تُقرأ من ردّ المنصّة**",
            serving.contains("it.launch"),
        )
        assertTrue(
            "**ذهب التجديدُ الدوريّ** — **فحالٌ قديمةٌ تبقى إلى أن يُقتل التطبيق**",
            serving.contains("refreshIfStale") && serving.contains("stale("),
        )
        assertTrue(
            "**لم يعد نصُّ المالك يصل الشاشةَ**",
            serving.contains("launchNotice"),
        )
        // **والنصُّ من اللوحة يغلب نصَّ الحزمة** — **ونصٌّ في حزمةٍ لا
        // يُصحَّح إلّا بنشرٍ في المتجر.**
        assertTrue(
            "**نصُّ اللوحة لم يعد يغلب نصَّ الحزمة**",
            preLaunchBody().contains("Serving.launchNotice"),
        )
    }

    // ── PL-18 (الطرفُ الأندرويديّ) ───────────────────────────────────

    /**
     * **ولا رقمَ مراجِعٍ بعينه في التطبيق** — **وبابٌ خلفيٌّ لا يُغلق.**
     */
    @Test
    fun `لا استثناءَ لرقم مراجِع غوغل`() {
        val main = read(MAIN)
        val serving = read(SERVING)
        for (src in listOf(main, serving)) {
            assertFalse(
                "**رقمُ المراجِع مكتوبٌ في التطبيق** — **وهو بابٌ لا يُغلق.**",
                src.contains("963900000099"),
            )
        }
    }
}
