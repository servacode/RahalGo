package com.rahalgo.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مهلةُ «أعد إرسال الرمز» — دقيقتان تُعدّان ثمّ زرّ** (قرارُ المالك ٢٠٢٦-١٠-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والعدُّ دالّةٌ صافية** — فيُقاس بلا جهازٍ ولا ساعة. **والربطُ يُقاس
 * من المصدر**: المواضعُ الأربعةُ التي تنتظر رمزاً كلُّها تعرض المكوّن.
 */
class ResendCodeTest {
    private val t0 = 1_700_000_000_000L

    /** **R1 · بعد الإرسال مباشرةً: دقيقتان كاملتان.** */
    @Test
    fun startsAtTwoMinutes() {
        assertEquals(120, resendSecondsLeft(t0, t0))
        assertEquals("2:00", resendClock(resendSecondsLeft(t0, t0)))
    }

    /** **R2 · يعدّ تنازليّاً ويُقرَّب إلى الأعلى** — لا «0:00» والزرُّ غائب. */
    @Test
    fun countsDownAndRoundsUp() {
        assertEquals(120, resendSecondsLeft(t0, t0 + 1))
        assertEquals(119, resendSecondsLeft(t0, t0 + 1_000))
        assertEquals("1:59", resendClock(resendSecondsLeft(t0, t0 + 1_000)))
        assertEquals(61, resendSecondsLeft(t0, t0 + 59_500))
        assertEquals("1:01", resendClock(61))
        assertEquals(1, resendSecondsLeft(t0, t0 + 119_999))
        assertEquals("0:01", resendClock(1))
    }

    /** **R3 · عند الصفر وبعده: الزرُّ متاح** — ولا سالب. */
    @Test
    fun reachesZeroAndStays() {
        assertEquals(0, resendSecondsLeft(t0, t0 + 120_000))
        assertEquals(0, resendSecondsLeft(t0, t0 + 3_600_000))
    }

    /** **R4 · لم يُرسَل شيء ⇒ الزرُّ فورا.** */
    @Test
    fun neverSentMeansAvailable() {
        assertEquals(0, resendSecondsLeft(0L, t0))
    }

    /** **R5 · ساعةٌ رجعت إلى الوراء لا تُطيل المهلةَ فوق دقيقتين.** */
    @Test
    fun clockSkewIsClamped() {
        assertEquals(120, resendSecondsLeft(t0, t0 - 600_000))
    }

    /** **R6 · إعادةُ الإرسال الناجحة تبدأ العدَّ من جديد.** */
    @Test
    fun newSendRestartsWindow() {
        val resentAt = t0 + 130_000
        assertEquals(0, resendSecondsLeft(t0, resentAt))
        assertEquals(120, resendSecondsLeft(resentAt, resentAt))
    }

    /** **R7 · الصيغةُ بأرقامٍ غربيّةٍ وخانتين للثواني.** */
    @Test
    fun clockFormat() {
        assertEquals("0:05", resendClock(5))
        assertEquals("1:30", resendClock(90))
        assertEquals("0:00", resendClock(-3))
    }

    /**
     * **R8 · المواضعُ الأربعةُ التي تنتظر رمزاً تعرض `ResendCodeRow`** —
     * والنماذجُ تضبط لحظةَ الإرسال عند النجاح وحدَه.
     */
    @Test
    fun everyCodeStepShowsResend() {
        var dir = File("").absoluteFile
        while (!File(dir, "settings.gradle.kts").exists()) {
            dir = dir.parentFile ?: throw AssertionError("mobile root not found")
        }
        val ui = File(dir, "ui/src/main/kotlin/com/rahalgo/ui")
        for (screen in listOf("SignupScreen.kt", "AuthScreen.kt", "ResetScreen.kt", "AccountScreen.kt")) {
            assertTrue(screen, File(ui, screen).readText().contains("ResendCodeRow("))
        }
        val auth = File(ui, "AuthViewModel.kt").readText()
        // **الدخولُ برمزٍ والاستعادةُ والتسجيل** — ثلاثةُ نجاحاتٍ تضبط اللحظة.
        assertEquals(3, Regex("codeSentAt = at,").findAll(auth).count())
        // **وخطوةُ الرمز تُستعاد بلحظتها إن أُغلق التطبيق** (`PendingCode`، ٢٠٢٦-١٠-٠٩).
        assertEquals(3, Regex("codeSentAt = p.sentAt").findAll(auth).count())
        val acc = File(ui, "AccountViewModel.kt").readText()
        assertTrue(acc.contains("waSentAt = System.currentTimeMillis()"))
    }
}
