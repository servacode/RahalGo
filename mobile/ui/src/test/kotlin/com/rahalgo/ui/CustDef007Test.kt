package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **CUST-DEF-007 — حالُ الحساب ليست انقطاعاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العطب
 *
 * **٤٠٣ `forbidden` (حسابٌ مقيَّد) كان يسقط في `else` فيُعرَض «لا يوجد
 * اتصال»** — فمن عُلّق حسابُه يفحص الواي فاي. **رفضُ الشبكة ≠ رفضُ الجلسة
 * ≠ تقييدُ الحساب ≠ تبديلُ كلمةٍ مطلوب ≠ تعذّرُ خدمة.**
 *
 * # العقدُ بعد الإصلاح
 *
 * دوالُّ تصنيفٍ صافيةٌ مركزيّة: `sessionRejected` (401/`invalid_refresh`)،
 * `accountForbidden` (403 `forbidden`)، `passwordChangeRequired`
 * (403 `password_change_required`). ومسارُ الاستعادة يوزّع: الجلسةُ
 * المرفوضةُ ⇒ `detachSession` (عزلةُ `CUST-DEF-009`)، والمقيَّدُ ⇒
 * `accountRestricted` (شاشةُ حال، **بلا مسحِ جلسة**)، والتبديلُ ⇒
 * `mustChangePassword`، وما بقي (شبكة/5xx/`auth_unavailable`) ⇒ `offline`.
 */
class CustDef007Test {

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

    // ══════════════════════════════════════════════════════════════════
    // **دوالُّ التصنيف — تُقاس مباشرةً (بلا جهاز)**
    // ══════════════════════════════════════════════════════════════════

    /** **CUST-DEF-007-01 · الحسابُ المقيَّد يُصنَّف بـ403 `forbidden` وحدَه.** */
    @Test
    fun accountForbiddenClassifier() {
        assertTrue("403 forbidden ليس مقيَّداً", accountForbidden(403, "forbidden"))
        assertFalse("401 يُقرأ تقييداً", accountForbidden(401, "unauthorized"))
        assertFalse("تبديلُ الكلمة يُقرأ تقييداً", accountForbidden(403, "password_change_required"))
        assertFalse("503 يُقرأ تقييداً", accountForbidden(503, "auth_unavailable"))
    }

    /** **CUST-DEF-007-02 · تبديلُ الكلمة عقدٌ مستقلٌّ عن التقييد.** */
    @Test
    fun passwordChangeClassifierIsDistinct() {
        assertTrue(passwordChangeRequired(403, "password_change_required"))
        assertFalse("`forbidden` يُقرأ تبديلاً", passwordChangeRequired(403, "forbidden"))
    }

    /** **CUST-DEF-007-03 · الفئاتُ لا تتداخل** — كلُّ حالٍ تصنيفٌ واحد. */
    @Test
    fun categoriesDoNotOverlap() {
        // حسابٌ مقيَّد: ليس جلسةً مرفوضة
        assertFalse("مقيَّدٌ يُقرأ جلسةً مرفوضة", sessionRejected(403, "forbidden"))
        // جلسةٌ مرفوضة: ليست تقييداً ولا تبديلاً
        assertTrue(sessionRejected(401, "unauthorized"))
        assertFalse(accountForbidden(401, "unauthorized"))
        assertFalse(passwordChangeRequired(401, "unauthorized"))
    }

    // ══════════════════════════════════════════════════════════════════
    // **الوصلُ — الاستعادةُ توزّع، والغلافُ يرسم**
    // ══════════════════════════════════════════════════════════════════

    /** **CUST-DEF-007-04 · الاستعادةُ تُميّز المقيَّدَ عن الانقطاع.** */
    @Test
    fun restoreMapsForbiddenToAccountState() {
        val r = body(read(auth), "private fun restore()", "\n    /** يعيد محاولة")
        assertTrue(
            "**الاستعادةُ لا تستدعي `accountForbidden`** — 403 يسقط في «لا اتصال»",
            r.contains("accountForbidden("),
        )
        assertTrue(
            "**المقيَّدُ لا يرفع `accountRestricted`**",
            r.contains("accountRestricted = true"),
        )
        assertTrue(
            "**تبديلُ الكلمة لا يُميَّز في الاستعادة**",
            r.contains("passwordChangeRequired(") && r.contains("mustChangePassword = true"),
        )
    }

    /**
     * **CUST-DEF-007-05 · والمقيَّدُ لا يمسح الجلسة** — عزلةُ `CUST-DEF-009`
     * باقية: `detachSession` وحدَه لمسار الجلسة المرفوضة (401).
     */
    @Test
    fun accountRestrictedDoesNotDetachSession() {
        val r = body(read(auth), "private fun restore()", "\n    /** يعيد محاولة")
        // `detachSession()` يقع في فرع `sessionRejected` وحدَه، لا في فرع التقييد.
        val reject = r.indexOf("sessionRejected(")
        val detach = r.indexOf("detachSession()")
        val forbid = r.indexOf("accountRestricted = true")
        assertTrue("**غاب أحدُ الفروع**", reject >= 0 && detach >= 0 && forbid >= 0)
        // مسحُ الجلسة قبل فرع التقييد (في فرع الرفض) — لا داخلَه
        assertTrue("**`detachSession` ليس في فرع الرفض**", detach in reject until forbid)
        // وفرعُ التقييد نفسُه لا يستدعي مسحاً
        val forbidArm = r.substring(forbid, r.length.coerceAtMost(forbid + 200))
        assertFalse(
            "**فرعُ التقييد يمسح الجلسة** — تسرّبٌ محتمل / خروجٌ خاطئ",
            forbidArm.contains("detachSession()"),
        )
    }

    /** **CUST-DEF-007-06 · والغلافُ يرسم شاشةَ الحال قبل شاشة الانقطاع.** */
    @Test
    fun frameRendersAccountStateBeforeOffline() {
        val src = read(frame)
        val acct = src.indexOf("vm.accountRestricted ->")
        val offline = src.indexOf("vm.offline ->")
        assertTrue("**الغلافُ لا يرسم `AccountStateScreen`**", src.contains("AccountStateScreen("))
        assertTrue("**شاشةُ الحال بعد شاشة الانقطاع أو غائبة**", acct >= 0 && offline >= 0 && acct < offline)
    }

    /** **CUST-DEF-007-07 · والانقطاعُ ما زال انقطاعاً** (شبكة ⇒ `offline`). */
    @Test
    fun networkStillMapsToOffline() {
        val r = body(read(auth), "private fun restore()", "\n    /** يعيد محاولة")
        // فرعُ `catch (e: Exception)` (شبكة) يبقى `offline = true`
        val netCatch = r.indexOf("catch (e: Exception)")
        assertTrue("**غاب مسارُ الشبكة**", netCatch >= 0)
        assertTrue(
            "**الشبكةُ لم تعُد تُعرَض انقطاعاً**",
            r.substring(netCatch).contains("offline = true"),
        )
    }
}
