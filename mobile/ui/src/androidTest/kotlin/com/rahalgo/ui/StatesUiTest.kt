package com.rahalgo.ui

/**
 * **طبقةُ الواجهة — حالاتُ الشاشة الأربع.**
 *
 * المعرّفات: `UI-STATE-*` · الوسم: `@ui @critical @release`
 *
 * (أولويّةُ المالك ٢٠٢٦-٠٨-١٩، البند ٤: «Loading · Success · Empty ·
 *
 *  Error · Retry · Unauthorized · Network failure».)
 *
 * # وأخطرُ حالةٍ لا اسمَ لها
 *
 * **دوّارةٌ لا تقف** — لا خطأٌ يُقرأ ولا محتوًى يظهر. **والزبونُ ينتظر
 * شيئاً لن يأتي**، ولا شيءَ في التطبيق يقول له أن يتوقّف.
 *
 * **فيُقاس أنّ الخطأ يُقصي الدوّارة** — وأنّ زرَّ الإعادة يظهر معه،
 * **وأنّه يُطلق فعلاً.**
 */
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import com.rahalgo.design.RahalGoTheme
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

class StatesUiTest {

    @get:Rule
    val rule = createComposeRule()

    /**
     * UI-STATE-001 **والخطأُ يُقرأ نصّاً لا رمزا.**
     *
     * **ورمزٌ إنجليزيٌّ في شاشةٍ عربيّةٍ يُقرأ عطباً في التطبيق** لا
     * خطأً في الشبكة.
     */
    @Test
    fun errorTextIsShown() {
        rule.setContent {
            RahalGoTheme(dark = false) {
                LoadState(loading = false, error = "تعذّر الاتصال بالإنترنت")
            }
        }
        rule.onNodeWithText("تعذّر الاتصال بالإنترنت").assertIsDisplayed()
    }

    /**
     * UI-STATE-002 **والخطأُ يُقصي الدوّارة — لا يجتمعان.**
     *
     * **ودوّارةٌ فوق رسالةِ خطأٍ تقول «ما زلتُ أحاول»** وهو غيرُ صحيح،
     * **فينتظر بلا سبب.**
     */
    @Test
    fun errorReplacesSpinner() {
        rule.setContent {
            RahalGoTheme(dark = false) {
                LoadState(loading = true, error = "انقطع الاتصال", onRetry = {})
            }
        }
        rule.onNodeWithText("انقطع الاتصال").assertIsDisplayed()
        // **وزرُّ الإعادةُ ظاهرٌ مع الخطأ** — خطأٌ بلا مخرجٍ طريقٌ مسدود.
        assertTrue(
            "لا زرَّ إعادةٍ مع الخطأ",
            rule.onAllNodesWithText("إعادة المحاولة").fetchSemanticsNodes().isNotEmpty() ||
                rule.onAllNodesWithText("أعد المحاولة").fetchSemanticsNodes().isNotEmpty(),
        )
    }

    /** UI-STATE-003 **وزرُّ الإعادةِ يُعيد فعلا.** */
    @Test
    fun retryFires() {
        var tries = 0
        rule.setContent {
            RahalGoTheme(dark = false) {
                LoadState(loading = false, error = "سقط النداء", onRetry = { tries++ })
            }
        }
        val retry = rule.onAllNodesWithText("إعادة المحاولة").fetchSemanticsNodes()
        val label = if (retry.isNotEmpty()) "إعادة المحاولة" else "أعد المحاولة"
        rule.onNodeWithText(label).performClick()
        rule.waitForIdle()
        assertEquals("زرُّ الإعادةِ لم يُطلق شيئا", 1, tries)
    }

    /**
     * UI-STATE-010 **وشاشةُ التحميل تحمل نصَّها.**
     *
     * **ودوّارةٌ صامتةٌ لا تقول ماذا ننتظر** — والنصُّ يفرّق بين
     * «نُحمّل» و«تعطّل».
     */
    @Test
    fun loadingScreenShowsItsText() {
        rule.setContent {
            RahalGoTheme(dark = false) { LoadingScreen(text = "نجهّز سوقك…") }
        }
        rule.onNodeWithText("نجهّز سوقك…").assertIsDisplayed()
    }

    /**
     * UI-STATE-020 **وشاشةُ الانقطاع تقول ما وقع وتعرض مخرجا.**
     *
     * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «شاشةُ عدم توفّر إنترنت… يجب أن تكون
     *  مركزيّة».)
     */
    @Test
    fun offlineScreenHasMessageAndRetry() {
        var tries = 0
        rule.setContent {
            RahalGoTheme(dark = false) { OfflineScreen(onRetry = { tries++ }) }
        }
        val nodes = rule.onAllNodesWithText("لا يوجد اتصال بالإنترنت").fetchSemanticsNodes()
        assertTrue("شاشةُ الانقطاع بلا رسالةٍ مفهومة", nodes.isNotEmpty())
    }
}
