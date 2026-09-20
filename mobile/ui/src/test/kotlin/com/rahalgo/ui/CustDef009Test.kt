package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **CUST-DEF-009 — حدُّ الجلسة الواحد: رفضُ الجلسة يمسح المحلّيَّ كالخروج**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العطب
 *
 * **إصلاحُ `CUST-DEF-004`** جعل الخروجَ الصريحَ يمسح **السلّةَ والوصلةَ**
 * (`AppCore.afterLogout()`)، **لكنّه في `logout()` وحدَه.** ومسارُ رفضِ
 * المحرّك عند الإقلاع (`sessionRejected`) كان يمسح التوكنَ **وحدَه**
 * (`session.clear()`) — بلا `afterLogout`. **فمن أُبطلت جلستُه بلا خروجٍ
 * صريح** (توكنٌ مرفوضٌ · جلسةٌ أُبطلت · حسابٌ حُذف · كلمةٌ أُعيدت) **بقيت
 * سلّتُه على الجهاز، فورثها الحسابُ التالي.**
 *
 * **وشوهد على الجهاز**: حُذف الحسابُ الأوّلُ من الخادم، فدخل الثاني على
 * الجهاز نفسِه **فرأى سلّةَ الأوّل** (عزلُ المحفظة/الوارد/الطلبات سليمٌ —
 * تُجلب بالجلسة؛ **السلّةُ محلّيّةٌ ولم تُمسح**).
 *
 * # العقدُ بعد الإصلاح
 *
 * **حدٌّ واحدٌ يمرّ به الجميع** — `detachSession()`: يمسح التوكنَ،
 * والهويّةَ المحلّيّةَ (`user=null` تُسقط الغلافَ ضيفاً فيُنعَش `Shell`)،
 * **والسلّةَ والوصلةَ** (`afterLogout`). **ويُنادى من `logout()` ومن
 * مسارِ الرفض معا.** **ولا يبقى `session.clear()` عارٍ في مكانٍ آخر** —
 * فمن مسح الجلسةَ مسح المحلّيَّ كلَّه، لا التوكنَ وحدَه.
 *
 * # لماذا حراسةُ مصدرٍ لا رسمُ شاشة
 *
 * **`AuthViewModel`/`AppCore` تحتاج `Context`/`Robolectric`** ولا نستعملها
 * (كـ`CustDef004Test`). **فالمقيسُ القرارُ**: أنّ كلَّ انتقالٍ إلى لا-جلسة
 * يمرّ بالحدّ الواحد، وأنّ الحدَّ يمسح الثلاثة — والآليّاتُ (`Cart.clear`،
 * `LiveSocket.stop`) مقيسةٌ سلوكيّاً في مواضعها، وخطّافُ الزبون في
 * `CustDef004Test`.
 */
class CustDef009Test {

    // **ونهاياتُ الأسطر تُوحَّد** — جيتٌ على ويندوز يكتب CRLF فتسقط المطابقةُ.
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

    /** **جسدُ مقطعٍ** — من علامةٍ إلى أوّلِ نهايةٍ بعدها. */
    private fun body(src: String, marker: String, vararg ends: String): String {
        val start = src.indexOf(marker)
        assertTrue("**غاب المقطع**: " + marker, start >= 0)
        val end = ends.mapNotNull { e -> src.indexOf(e, start + marker.length).takeIf { it >= 0 } }
            .minOrNull() ?: src.length
        return src.substring(start, end)
    }

    private val auth = "ui/src/main/kotlin/com/rahalgo/ui/AuthViewModel.kt"
    private val backend = "app-customer/src/main/kotlin/com/rahalgo/customer/Backend.kt"

    private fun detachBody() = body(read(auth), "private fun detachSession()", "\n    }")
    private fun logoutBody() = body(read(auth), "fun logout()", "\n    private fun ", "\n    fun ", "\n}")
    private fun restoreBody() = body(read(auth), "private fun restore()", "\n    /** يعيد محاولة الاستعادة")

    // ══════════════════════════════════════════════════════════════════════
    // **الحدُّ الواحدُ موجودٌ ويمسح الثلاثة**
    // ══════════════════════════════════════════════════════════════════════

    /** **CUST-DEF-009-01 · `detachSession()` هو الحدُّ المركزيّ.** */
    @Test
    fun centralBoundaryExists() {
        assertTrue(
            "**لا حدَّ مركزيّ `detachSession()`** — فكلُّ فرعٍ يمسح بعقده",
            read(auth).contains("private fun detachSession()"),
        )
    }

    /** **CUST-DEF-009-02 · والحدُّ يمسح التوكنَ.** */
    @Test
    fun boundaryClearsToken() {
        assertTrue(
            "**الحدُّ لا يمسح الجلسة** (`session.clear()`)",
            detachBody().contains("session.clear()"),
        )
    }

    /** **CUST-DEF-009-03 · والهويّةَ المحلّيّة — `user=null` تُسقط الغلافَ ضيفاً فيُنعَش `Shell`.** */
    @Test
    fun boundaryClearsIdentity() {
        assertTrue(
            "**الحدُّ لا يُسقط الهويّةَ** (`user = null`) — فلا يُنعَش الغلاف",
            detachBody().contains("user = null"),
        )
    }

    /** **CUST-DEF-009-04 · والسلّةَ والوصلةَ — عبر خطّاف `afterLogout` نفسِه (عقدُ `CUST-DEF-004`).** */
    @Test
    fun boundaryInvokesLocalCleanupHook() {
        assertTrue(
            "**الحدُّ لا يستدعي `AppCore.afterLogout()`** — فالسلّةُ والوصلةُ تبقيان",
            detachBody().contains("AppCore.afterLogout()"),
        )
    }

    // ══════════════════════════════════════════════════════════════════════
    // **وكلُّ انتقالٍ إلى لا-جلسة يمرّ به**
    // ══════════════════════════════════════════════════════════════════════

    /** **CUST-DEF-009-05 · الخروجُ الصريحُ يمرّ بالحدّ (بقاءُ عقد `CUST-DEF-004`).** */
    @Test
    fun logoutRoutesThroughBoundary() {
        assertTrue(
            "**`logout()` لا يمرّ بالحدّ** — عاد يمسح التوكنَ وحدَه",
            logoutBody().contains("detachSession()"),
        )
    }

    /** **CUST-DEF-009-06 · ورفضُ المحرّك يمرّ بالحدّ — لا `session.clear()` عارٍ. (الشاهدُ السالب.)** */
    @Test
    fun sessionRejectRoutesThroughBoundary() {
        val r = restoreBody()
        assertTrue(
            "**مسارُ الرفض في `restore()` لا يستدعي `detachSession()`** — فيمسح التوكنَ ويترك السلّة",
            r.contains("detachSession()"),
        )
        assertTrue(
            "**رفضُ الجلسةِ ما زال يمسح التوكنَ عارياً** (`session.clear()`) بلا الحدّ — تسرّبُ سلّةٍ",
            !r.contains("session.clear()"),
        )
    }

    /**
     * **CUST-DEF-009-07 · ولا `session.clear()` عارٍ في المِلفّ كلِّه —
     * الوحيدُ داخلَ الحدّ.**
     *
     * **فمن زاد مسارَ إبطالٍ ومسح التوكنَ فيه مباشرةً أسقط هذا الفحص** —
     * والحدُّ لا يُنسى.
     */
    @Test
    fun onlyOneSessionClearAndItIsTheBoundary() {
        val src = read(auth)
        // **النداءُ الحقيقيُّ `backend.session.clear()`** — لا ذكرُه في تعليقٍ
        // (السطر ٥٦٨ يشرح العطبَ بـ`session.clear()` عارية، وليس نداءً).
        val count = Regex(Regex.escape("backend.session.clear()")).findAll(src).count()
        assertEquals(
            "**`backend.session.clear()` يقع في غير الحدّ** — كلُّ مسحٍ يجب أن يمرّ بـ`detachSession()`",
            1, count,
        )
        assertTrue(
            "**النداءُ الوحيدُ ليس داخلَ الحدّ**",
            detachBody().contains("backend.session.clear()"),
        )
    }

    /**
     * **CUST-DEF-009-08 · وخطّافُ الزبون يمسح السلّةَ فعلاً** — فالحدُّ
     * الذي يستدعيه يُفرغها (تكاملُ العقد مع `CUST-DEF-004`).
     */
    @Test
    fun customerHookClearsCart() {
        val reg = body(read(backend), "afterLogout = {", "},")
        assertTrue(
            "**خطّافُ `afterLogout` في الزبون لا يمسح السلّة** — فالحدُّ لا يُفرغها",
            reg.contains("Cart.clear()"),
        )
    }
}
