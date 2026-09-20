package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **CUST-DEF-010 — تبديلُ كلمةِ المرور المطلوب: تدفّقٌ لا رسالةُ خطأ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العطب (CUST-06-027, CONTRACT_MISMATCH بلا معرّف)
 *
 * **المحرّكُ قد يفرض تبديلاً** (`must_change_password` / 403
 * `password_change_required`)، **والعميلُ لم يكن إلّا نصَّ خطإٍ** — لا شاشةَ
 * ولا تدفّق. **فلا مخرجَ**: أيُّ بابٍ يُردّ، ولا مكانَ يبدّل فيه.
 *
 * # العقدُ بعد الإصلاح
 *
 * **يُكشَف من كائن المستخدم** (`must_change_password`، فـ`/auth/me` مسموحٌ
 * لمن يبدّل) **ومن 403 دفاعاً**؛ **يُساق إلى `ForcedPasswordScreen` قبل فرع
 * المستخدم في الغلاف** (لا يدخل التطبيق)؛ **يكتب الحاليّةَ + الجديدةَ +
 * التأكيد** (لا يُكشَف القديم)؛ **يُرسَل عبر النقطة الموثوقة**
 * (`account.setPassword` ⇒ `POST /auth/password`)؛ **وعند النجاح يُرفع القيدُ
 * ويدخل بالجلسة نفسِها** (عزلةُ `CUST-DEF-004`/`009` قائمة)؛ **ولا يُتخطّى
 * بالرجوع/موتِ العمليّة** — كلُّ إقلاعٍ يُعيد كشفَه.
 */
class CustDef010Test {

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
    private val frame = "ui/src/main/kotlin/com/rahalgo/ui/AppFrame.kt"
    private val userModel = "shared/src/main/kotlin/com/rahalgo/shared/model/Auth.kt"
    private val screen = "ui/src/main/kotlin/com/rahalgo/ui/ForcedPasswordScreen.kt"

    /** **CUST-DEF-010-01 · كائنُ المستخدم يحمل `must_change_password`.** */
    @Test
    fun userModelHasMustChangeField() {
        val u = body(read(userModel), "data class User(", "\n)")
        assertTrue(
            "**`User` لا يقرأ `must_change_password`** — فلا يُكشَف القيدُ من `me()`",
            u.contains("must_change_password") && u.contains("mustChangePassword"),
        )
    }

    /** **CUST-DEF-010-02 · التصنيفُ الصافي — 403 `password_change_required`.** */
    @Test
    fun passwordChangeRequiredClassifier() {
        assertTrue(passwordChangeRequired(403, "password_change_required"))
        assertFalse("`forbidden` يُقرأ تبديلاً", passwordChangeRequired(403, "forbidden"))
        assertFalse(passwordChangeRequired(401, "unauthorized"))
    }

    /** **CUST-DEF-010-03 · يُكشَف عند الاستعادة والدخول (كائنُ المستخدم).** */
    @Test
    fun detectedOnRestoreAndSignIn() {
        val src = read(auth)
        val restore = body(src, "private fun restore()", "\n    /** يعيد محاولة")
        assertTrue(
            "**الاستعادةُ لا تكشف `mustChangePassword` من المستخدم**",
            restore.contains("mustChangePassword == true") && restore.contains("mustChangePassword = true"),
        )
        val signedIn = body(src, "private fun onSignedIn(", "\n    private fun ", "\n}")
        assertTrue(
            "**الدخولُ لا يكشف التبديلَ المطلوب** — يدخل التطبيقَ قبل التبديل",
            signedIn.contains("mustChangePassword == true") && signedIn.contains("mustChangePassword = true"),
        )
    }

    /** **CUST-DEF-010-04 · الإرسالُ عبر النقطة الموثوقة، والنجاحُ يرفع القيد.** */
    @Test
    fun submitUsesAuthoritativeEndpointAndClears() {
        val m = body(read(auth), "fun submitForcedPasswordChange(", "\n    fun ", "\n    private fun ")
        assertTrue(
            "**التبديلُ لا يمرّ بـ`account.setPassword`** (`POST /auth/password`)",
            m.contains("backend.account.setPassword("),
        )
        assertTrue(
            "**النجاحُ لا يرفع القيدَ** — يبقى محبوساً",
            m.contains("mustChangePassword = false"),
        )
        // **الكلمةُ الحاليّةُ تُرسَل (يعرفها صاحبُها) — والجديدةُ** — لا كشفَ لقديم
        assertTrue("**لا تُمرَّر الحاليّةُ+الجديدة**", m.contains("current") && m.contains("next"))
    }

    /** **CUST-DEF-010-05 · الغلافُ يرسم شاشةَ التبديل قبل فرع المستخدم.** */
    @Test
    fun frameRendersBeforeUserBranch() {
        val src = read(frame)
        val pw = src.indexOf("vm.mustChangePassword ->")
        val user = src.indexOf("vm.user != null ->")
        assertTrue("**لا شاشةَ `ForcedPasswordScreen` في الغلاف**", src.contains("ForcedPasswordScreen("))
        assertTrue(
            "**شاشةُ التبديل بعد فرع المستخدم أو غائبة** — يدخل التطبيقَ فيتخطّى القيد",
            pw >= 0 && user >= 0 && pw < user,
        )
    }

    /** **CUST-DEF-010-06 · الشاشةُ تطلب الحاليّةَ+الجديدةَ+التأكيد ولا تكشف القديم.** */
    @Test
    fun screenHasCurrentNewConfirmAndHidesOld() {
        val s = read(screen)
        assertTrue("**لا حقلَ للحاليّة**", s.contains("acc_pw_current"))
        assertTrue("**لا حقلَ للجديدة**", s.contains("acc_pw_new"))
        assertTrue("**لا حقلَ للتأكيد**", s.contains("acc_pw_confirm"))
        // **حقولُ كلمةِ مرورٍ مُخفاة** — `PasswordField` (بصريّةُ إخفاء)
        assertTrue("**الحقولُ ليست مخفيّة**", s.contains("PasswordField("))
        // **ولا يُعرَض أيُّ نصِّ كلمةٍ قديم** — لا حقلَ للقراءة فقط للقديم
        assertFalse("**يُكشَف قديمٌ**", s.contains("old_password") || s.contains("temp_password"))
    }
}
