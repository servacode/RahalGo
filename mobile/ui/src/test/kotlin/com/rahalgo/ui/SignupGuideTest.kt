package com.rahalgo.ui

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شرحُ إنشاء الحساب — أيُّ خطوة، والتقدّم، والتخطّي، ومرّةً واحدة** (٢٠٢٦-١١-٠٧)
 * ══════════════════════════════════════════════════════════════════════
 */
class SignupGuideTest {

    private class FakeStore(var flag: Boolean = false, val broken: Boolean = false) : GuideStore {
        var writes = 0
        override fun seen(): Boolean {
            if (broken) error("storage")
            return flag
        }
        override fun markSeen() {
            writes++
            if (broken) error("storage")
            flag = true
        }
    }

    /** **G1 · الرقم ثمّ الرمز ثمّ الاسم** — تتبع الخطوةَ الحقيقيّة. */
    @Test
    fun followsTheRealSteps() {
        assertEquals(GuideSpot.PHONE, GuideSteps.spot(SignupStep.PHONE, false, false, null))
        assertEquals(GuideSpot.CODE, GuideSteps.spot(SignupStep.CODE, false, false, null))
        assertEquals(GuideSpot.NAME, GuideSteps.spot(SignupStep.DETAILS, false, false, null))
    }

    /** **G2 · تنتقل وحدها حين يكتمل الحقل** — الاسم ← كلمة السرّ ← رمز الدعوة. */
    @Test
    fun advancesWhenFilled() {
        assertEquals(GuideSpot.PASSWORD, GuideSteps.spot(SignupStep.DETAILS, true, false, null))
        assertEquals(GuideSpot.REFERRAL, GuideSteps.spot(SignupStep.DETAILS, true, true, null))
    }

    /** **G3 · الحقلُ الملموسُ يغلب** — ولا يُقبل حقلٌ من خطوةٍ أخرى. */
    @Test
    fun focusedFieldWins() {
        assertEquals(GuideSpot.REFERRAL, GuideSteps.spot(SignupStep.DETAILS, false, false, GuideSpot.REFERRAL))
        assertEquals(GuideSpot.NAME, GuideSteps.spot(SignupStep.DETAILS, true, true, GuideSpot.NAME))
        assertEquals(GuideSpot.NAME, GuideSteps.spot(SignupStep.DETAILS, false, false, GuideSpot.PHONE))
        assertEquals(GuideSpot.PHONE, GuideSteps.spot(SignupStep.PHONE, false, false, GuideSpot.NAME))
    }

    /** **G4 · كلمةُ السرّ تمّت بطولها وتأكيدِها.** */
    @Test
    fun passwordDoneNeedsLengthAndMatch() {
        assertFalse(GuideSteps.passwordDone("1234567", "1234567", 8))
        assertFalse(GuideSteps.passwordDone("12345678", "1234567", 8))
        assertTrue(GuideSteps.passwordDone("12345678", "12345678", 8))
    }

    /** **G5 · يُعرض أوّلَ مرّة، والتخطّي يُطفئه ويُحفظ مرّةً واحدة.** */
    @Test
    fun skipEndsItOnce() {
        val store = FakeStore()
        val g = SignupGuide(store)
        assertTrue(g.active)
        g.finish()
        assertFalse(g.active)
        g.finish()
        assertEquals(1, store.writes)
        assertTrue(store.flag)
    }

    /** **G6 · من رآه لا يُعرض له ثانية** — جهازٌ واحدٌ ومرّةٌ واحدة. */
    @Test
    fun shownOnlyOnce() {
        val store = FakeStore()
        SignupGuide(store).finish()
        assertFalse(SignupGuide(store).active)
    }

    /** **G7 · تخزينٌ عاطلٌ لا يُسقط شيئاً** — يظهر، والإطفاءُ يعمل في الذاكرة. */
    @Test
    fun brokenStorageIsHarmless() {
        val g = SignupGuide(FakeStore(broken = true))
        assertTrue(g.active)
        g.finish()
        assertFalse(g.active)
    }

    private fun mobileRoot(): File {
        var dir = File("").absoluteFile
        repeat(6) {
            if (File(dir, "settings.gradle.kts").exists() && File(dir, "app-customer").exists()) return dir
            dir = dir.parentFile ?: return@repeat
        }
        throw AssertionError("لم أجد جذرَ mobile من " + File("").absolutePath)
    }

    /** **G8 · أرقامُ الرمز في الفقاعة أرقامُ المحرّك** — `randomDigits(n)` في التسجيل. */
    @Test
    fun codeDigitsMatchTheEngine() {
        val go = File(mobileRoot().parentFile, "backend/internal/identity/service.go").readText()
        val ns = Regex("""randomDigits\((\d+)\)""").findAll(go).map { it.groupValues[1].toInt() }.toSet()
        assertEquals("أطوالُ الرمز في المحرّك", setOf(SIGNUP_CODE_DIGITS), ns)
    }

    /** **G9 · للزبون وحدَه** — يُقرأ داخل `signup` في الدخول، والتطبيقاتُ الأخرى لا تذكره. */
    @Test
    fun onlyTheCustomerSeesIt() {
        val root = mobileRoot()
        val auth = File(root, "ui/src/main/kotlin/com/rahalgo/ui/AuthScreen.kt").readText()
        val gate = auth.indexOf("if (signup) {")
        assertTrue("بابُ التسجيل في الدخول", gate > 0)
        assertTrue("الشرحُ خارج بابِ الزبون", auth.indexOf("SignupGuide.shared") > gate)
        for (app in listOf("app-driver", "app-merchant", "app-rep")) {
            File(root, "$app/src/main/kotlin").walkTopDown().filter { it.extension == "kt" }.forEach { f ->
                assertFalse("${f.name} يذكر الشرح", f.readText().contains("SignupGuide"))
            }
        }
    }
}
