package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **CUST-DEF-011 — رسالةُ الانقطاعِ الحقليّةُ لا تُمحى عند عودة الاتّصال**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العطب (شكوى المالك ٢٠٢٦-٠٩-٢٣)
 *
 * **في شاشة إنشاء الحساب**: طلبُ رمزٍ وهو منقطعٌ يكتب «لا اتصال بالإنترنت»
 * تحتَ حقل الرقم. **وعند عودة الاتّصال يزول شريطُ الأعلى** (يتبع
 * `Net.online`) **وتبقى هذه الرسالةُ الحقليّةُ ساكنةً** — فيرى صاحبُها
 * اتّصالاً عاد ورسالةَ انقطاعٍ لم تزُل.
 *
 * # العقدُ بعد الإصلاح
 *
 * **رسالةُ الحقلِ حالٌ غيرُ شريطِ الأعلى** — نصٌّ في `SignupState`.
 * **فتُوسَم أخطأُ الانقطاعِ وحدَها** (`offlineError`)، **وتُبلَّغ الشاشةُ
 * بعودة الاتّصال** (`Net.online` ⇒ `onReconnected`) **فيمحوها النموذجُ**
 * (`signupConnectivityRestored`) دون نقرةٍ ولا تنقّل. **وأخطاءُ التحقّق
 * تبقى** — الحارسُ `if (offlineError)`. **ونجاحُ نداءٍ لاحقٍ يمحو كلَّ
 * أثرٍ منقطع.** ورسالةٌ طافيةٌ تُطمئن («عاد الاتصال») تنصرف وحدَها.
 *
 * **وتُقاس من المصدر** — كأخواتها (`CustDef007Test`)، بلا جهاز.
 */
class CustDef011Test {

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    private fun read(rel: String): String {
        val f = File(mobileRoot(), rel)
        assertTrue("**ملفٌّ غائب**: " + rel, f.exists())
        return f.readText().replace("\r\n", "\n")
    }

    private fun body(src: String, marker: String, vararg ends: String): String {
        val start = src.indexOf(marker)
        assertTrue("**غاب المقطع**: " + marker, start >= 0)
        val end = ends.mapNotNull { e -> src.indexOf(e, start + marker.length).takeIf { it >= 0 } }
            .minOrNull() ?: src.length
        return src.substring(start, end)
    }

    private val auth = "ui/src/main/kotlin/com/rahalgo/ui/AuthViewModel.kt"
    private val screen = "ui/src/main/kotlin/com/rahalgo/ui/SignupScreen.kt"
    private val frame = "ui/src/main/kotlin/com/rahalgo/ui/AppFrame.kt"
    private val strings = "ui/src/main/res/values/strings.xml"

    // ══════════════════════════════════════════════════════════════════
    // **الحالُ — رسالةُ الحقلِ مُوسَمةٌ أهي من الشبكة**
    // ══════════════════════════════════════════════════════════════════

    /** **CUST-DEF-011-01 · `SignupState` يحمل رايةَ الانقطاعِ الحقليّة.** */
    @Test
    fun signupStateTagsOffline() {
        assertTrue(
            "**`SignupState` بلا `offlineError`** — فلا يُفرَّق انقطاعٌ عن تحقّق",
            read(screen).contains("val offlineError: Boolean = false"),
        )
    }

    /** **CUST-DEF-011-02 · الانقطاعُ وحدَه يُوسَم، وردُّ الخادمِ لا.** */
    @Test
    fun networkErrorTaggedOfflineNotApiError() {
        for (fn in listOf("fun sendSignupCode()", "fun verifySignupCode(")) {
            val b = body(read(auth), fn, "\n    fun ", "\n    private fun ")
            assertTrue(
                "**فرعُ الشبكة لا يُوسَم `offlineError = isOffline(e)`** في " + fn,
                b.contains("error = describe(e), offlineError = isOffline(e)"),
            )
            assertTrue(
                "**فرعُ ردِّ الخادمِ (`ApiException`) لا يُصفّر الوسمَ** في " + fn,
                b.contains("error = message(e), offlineError = false"),
            )
        }
    }

    /** **CUST-DEF-011-03 · `isOffline` يفرّق الانقطاعَ/المهلةَ عمّا سواه.** */
    @Test
    fun isOfflineDistinguishesNetwork() {
        val b = body(read(auth), "private fun isOffline(", "\n    fun ", "\n    private fun ")
        assertTrue(
            "**`isOffline` لا يقيس `IOException`/المهلة**",
            b.contains("IOException") && b.contains("HttpRequestTimeoutException"),
        )
    }

    // ══════════════════════════════════════════════════════════════════
    // **عودةُ الاتّصال — تمحو الانقطاعَ وحدَه، وتُبقي التحقّق**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **CUST-DEF-011-04 · المحو مشروطٌ بـ`offlineError`** — فأخطاءُ التحقّق
     * تبقى، ولا يُمحى ما ليس من الشبكة.
     */
    @Test
    fun restoredClearsOnlyOfflineError() {
        val b = body(read(auth), "fun signupConnectivityRestored()", "\n    fun ", "\n    private fun ")
        val guard = b.indexOf("if (current.offlineError)")
        assertTrue("**المحو غيرُ مشروطٍ بـ`offlineError`** — تُمحى أخطاءُ التحقّق", guard >= 0)
        val clear = b.indexOf("error = \"\", offlineError = false")
        assertTrue("**لا يُمحى الخطأُ ولا تُخفَض الرايةُ عند العودة**", clear > guard)
    }

    /** **CUST-DEF-011-05 · ورسالةٌ طافيةٌ تُطمئن عند العودة** («عاد الاتصال»). */
    @Test
    fun restoredShowsReconnectFlash() {
        val b = body(read(auth), "fun signupConnectivityRestored()", "\n    fun ", "\n    private fun ")
        assertTrue(
            "**لا رسالةَ عودةٍ طافية**",
            b.contains("Flash.ok(str(R.string.net_reconnected))"),
        )
        assertTrue(
            "**نصُّ العودةِ غائبٌ من الموارد**",
            read(strings).contains("name=\"net_reconnected\""),
        )
    }

    /** **CUST-DEF-011-06 · نجاحُ الطلبِ يمحو كلَّ أثرٍ منقطع** (العقدُ ٣). */
    @Test
    fun successClearsStaleOfflineError() {
        for (fn in listOf("fun sendSignupCode()", "fun verifySignupCode(")) {
            val b = body(read(auth), fn, "\n    fun ", "\n    private fun ")
            // فرعُ النجاح (داخلَ `try`، قبل أوّلِ `catch`) يُصفّر الاثنين.
            val successArm = b.substring(0, b.indexOf("catch").let { if (it < 0) b.length else it })
            assertTrue(
                "**فرعُ النجاحِ لا يُصفّر `error`+`offlineError`** في " + fn,
                successArm.contains("busy = false, error = \"\", offlineError = false") ||
                    successArm.contains("busy = false, error = \"\"") && successArm.contains("offlineError = false"),
            )
        }
    }

    // ══════════════════════════════════════════════════════════════════
    // **الوصلُ — الشاشةُ تُبلِّغ، والغلافُ يوصل، والقرارُ في النموذج**
    // ══════════════════════════════════════════════════════════════════

    /** **CUST-DEF-011-07 · الشاشةُ ترصد `Net.online` فتُبلِّغ عند العودة.** */
    @Test
    fun screenObservesConnectivity() {
        val s = read(screen)
        assertTrue("**الشاشةُ لا ترصد `Net.online`**", s.contains("Net.online"))
        val eff = body(s, "LaunchedEffect(online)", "}")
        assertTrue(
            "**الشاشةُ لا تُبلِّغ `onReconnected` عند العودة**",
            eff.contains("if (online) actions.onReconnected()"),
        )
        // والشاشةُ تُبلِّغ لا تقرّر: لا محوَ للحالِ في الشاشة نفسِها.
        assertFalse(
            "**الشاشةُ تمحو الحالَ بيدها** — القرارُ من حقِّ النموذج",
            s.contains("offlineError = false"),
        )
    }

    /** **CUST-DEF-011-08 · الغلافُ يوصل بلاغَ العودةِ إلى النموذج.** */
    @Test
    fun frameWiresReconnect() {
        assertTrue(
            "**`AppFrame` لا يوصل `onReconnected` بـ`signupConnectivityRestored`**",
            read(frame).contains("onReconnected = vm::signupConnectivityRestored"),
        )
    }
}
