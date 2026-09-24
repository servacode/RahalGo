package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **Obs 3 — جلسةُ زبونٍ واحدةٌ نشطة: إزاحةٌ فوريّةٌ بسببٍ صريح**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العقد
 *
 * **دخولٌ جديدٌ من نوعِ العميل نفسِه يُزيح القديم**: الجهازُ القديمُ يُرفَض
 * فوراً (بثٌّ خاصٌّ بالعائلة، ثمّ ٤٠١)، **ويخرج بسببٍ صريح**: «تم تسجيل خروجك…
 * من جهازٍ آخر» لا رسالةً عامّة، **ويدوم السببُ ولو عاد الجهازُ متأخّراً**
 * (القاعدةُ تحفظه، لا Redis وحدَها). **ورمزُ دفعِ الجهاز الحاليّ يُعاد تسجيلُه
 * مع كلّ دخول** فلا يبقى مقطوعاً بعد إعادةِ دخولٍ على الجهاز نفسِه.
 *
 * **حارسُ مصدرٍ**: يفحص الوصلَ لا المنطقَ الحيّ (شاهدُ الأثرِ الخادميُّ في
 * `identity/session_client_test.go`، والشاهدُ الحيُّ جهازان في الميدان).
 */
class SessionSupersededTest {

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

    private val client = "shared/src/main/kotlin/com/rahalgo/shared/net/ApiClient.kt"
    private val auth = "ui/src/main/kotlin/com/rahalgo/ui/AuthViewModel.kt"
    private val frame = "ui/src/main/kotlin/com/rahalgo/ui/AppFrame.kt"
    private val errs = "ui/src/main/kotlin/com/rahalgo/ui/ApiErrors.kt"
    private val strings = "ui/src/main/res/values/strings.xml"

    /** **Obs3-01 · العميلُ يملك خطّافَ الرفض ويُطلقه حين يفشل التجديد.** */
    @Test
    fun apiClientHasSessionRejectedHookAndFiresIt() {
        val src = read(client)
        assertTrue(
            "**لا خطّافَ `onSessionRejected`**",
            src.contains("var onSessionRejected: ((String) -> Unit)? = null"),
        )
        val call = body(src, "suspend inline fun <reified T> call(", "\n    /** نداء بلا تجديد")
        assertTrue(
            "**`call` لا يُطلق الخطّافَ حين يفشل التجديد** — فتبقى الشاشةُ تعرض خطأً وصاحبُها «داخلٌ»",
            call.contains("onSessionRejected?.invoke("),
        )
        // **ويُفضَّل سببُ «جهازٌ آخر»** من أيّ الإشارتين ظهر.
        assertTrue(
            "**لا يُميَّز `session_superseded`** — فيخرج برسالةٍ عامّة",
            call.contains("session_superseded"),
        )
    }

    /** **Obs3-02 · الغلافُ يوصل الخطّافَ بخروجٍ قسريّ.** */
    @Test
    fun authViewModelWiresForcedLogout() {
        val src = read(auth)
        assertTrue(
            "**لا يُوصَل `ApiClient.onSessionRejected` بالحالة**",
            src.contains("ApiClient.onSessionRejected = {") && src.contains("forcedLogout("),
        )
        val fl = body(src, "private fun forcedLogout(", "\n    fun ", "\n    private fun ")
        assertTrue(
            "**الخروجُ القسريُّ لا يمرّ بحدِّ `detachSession` الواحد** (عزلةُ CUST-DEF-009)",
            fl.contains("detachSession()"),
        )
        assertTrue(
            "**لا يُعرَض سببُ «جهازٌ آخر» الصريح**",
            fl.contains("R.string.err_session_superseded"),
        )
    }

    /** **Obs3-03 · «جهازٌ آخر» في المعجم، بنصِّه الصريح.** */
    @Test
    fun errorCodeMapsToAnotherDeviceString() {
        assertTrue(
            "**`session_superseded` غيرُ مربوطٍ في `ApiErrors`**",
            read(errs).contains("\"session_superseded\" to R.string.err_session_superseded"),
        )
        val s = read(strings)
        assertTrue("**غاب مفتاحُ `err_session_superseded`**", s.contains("name=\"err_session_superseded\""))
        assertTrue(
            "**الرسالةُ لا تذكر «جهاز آخر» صراحةً**",
            s.contains("تم تسجيل خروجك") && s.contains("من جهاز آخر"),
        )
    }

    /** **Obs3-04 · رمزُ الدفع يُعاد تسجيلُه مع كلّ دخول (لا مع تبدّلِ المستخدم وحدَه).** */
    @Test
    fun deviceTokenReRegistersEverySession() {
        assertTrue(
            "**`sessionEpoch` لا يزيد في `onSignedIn`** — فإعادةُ الدخول لا تُعيد تسجيلَ رمز الدفع",
            body(read(auth), "private fun onSignedIn(", "\n    private fun ", "\n}")
                .contains("sessionEpoch += 1"),
        )
        assertTrue(
            "**الغلافُ يعيد التسجيلَ على `user.id` وحدَه** — فرمزٌ قُطع لا يُعاد بعد دخولٍ للحساب نفسِه",
            read(frame).contains("LaunchedEffect(vm.user?.id, vm.sessionEpoch)"),
        )
    }
}
