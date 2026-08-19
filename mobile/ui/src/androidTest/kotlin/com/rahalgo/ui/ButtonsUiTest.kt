package com.rahalgo.ui

/**
 * **طبقةُ الواجهة — الأزرارُ والنقرُ المزدوج.**
 *
 * المعرّفات: `UI-BTN-*` · الوسم: `@ui @critical @release`
 *
 * (أولويّةُ المالك ٢٠٢٦-٠٨-١٩، البند ١٥: «Double tap Create Order ·
 *
 *  Double tap Cancel · Double add to cart · Double retry».)
 *
 * # وهذه تقود Compose فعلاً
 *
 * **لا نموذجَ عرضٍ ولا محاكاة** — يُركَّب المكوّنُ في نشاطٍ حقيقيٍّ على
 * جهاز، **ويُنقر كما ينقر الإصبع.**
 *
 * # وما تحرسه
 *
 * **زرٌّ لا يُعطَّل أثناء الإرسال يُنشئ طلبين** — وهو نصفُ الحماية
 * الذي لا يراه مفتاحُ منع التكرار: **المفتاحُ يمنع التكرارَ في الخادم،
 * والزرُّ المعطَّل يمنعه قبل أن يُرسَل.**
 *
 * **ونقرتان في ٢٠٠ مللي ثانيةٍ ليستا افتراضا** — يقعان كلَّ يومٍ على
 * شبكةٍ بطيئةٍ حين لا يتغيّر شيءٌ في الشاشة.
 */
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertHasClickAction
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.junit4.createComposeRule
import com.rahalgo.design.RahalGoTheme
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test

class ButtonsUiTest {

    @get:Rule
    val rule = createComposeRule()

    /**
     * UI-BTN-001 **نقرتان متتاليتان على زرٍّ معطَّلٍ بعد الأولى = فعلٌ
     * واحد.**
     *
     * **وهذه هي حمايةُ الشاشة**: الزرُّ يُعطَّل لحظةَ الضغط،
     * **والثانيةُ تقع على زرٍّ لا يستجيب.**
     */
    @Test
    fun doubleTapFiresOnce() {
        var fired = 0
        rule.setContent {
            RahalGoTheme(dark = false) {
                var busy by remember { mutableStateOf(false) }
                RahalButton(enabled = !busy, onClick = { busy = true; fired++ }) {
                    Text("أرسل الطلب")
                }
            }
        }
        rule.onNodeWithText("أرسل الطلب").performClick()
        rule.onNodeWithText("أرسل الطلب").performClick()
        rule.onNodeWithText("أرسل الطلب").performClick()
        rule.waitForIdle()
        assertEquals("نقراتٌ متعدّدةٌ أطلقت الفعلَ أكثرَ من مرّة", 1, fired)
    }

    /**
     * UI-BTN-002 **وزرٌّ بلا تعطيلٍ يُطلق ثلاثاً — وهذا ما نتجنّبه.**
     *
     * **والحارسُ يُثبت أنّ الاختبارَ يقيس شيئا**: لو أطلق واحدةً هنا
     * أيضاً لكان `UI-BTN-001` ينجح على أيّ شيء.
     */
    @Test
    fun unguardedButtonFiresEachTap() {
        var fired = 0
        rule.setContent {
            RahalGoTheme(dark = false) {
                RahalButton(onClick = { fired++ }) { Text("بلا حراسة") }
            }
        }
        repeat(3) { rule.onNodeWithText("بلا حراسة").performClick() }
        rule.waitForIdle()
        assertEquals("الزرُّ المكشوفُ لم يُطلق ثلاثا — الاختبارُ لا يقيس", 3, fired)
    }

    /** UI-BTN-003 **والمعطَّلُ لا يُنقر أصلا.** */
    @Test
    fun disabledButtonDoesNotFire() {
        var fired = 0
        rule.setContent {
            RahalGoTheme(dark = false) {
                RahalButton(enabled = false, onClick = { fired++ }) { Text("معطّل") }
            }
        }
        rule.onNodeWithText("معطّل").assertIsNotEnabled()
        rule.onNodeWithText("معطّل").performClick()
        rule.waitForIdle()
        assertEquals("زرٌّ معطَّلٌ أطلق فعلَه", 0, fired)
    }

    /**
     * UI-BTN-010 **وزيادةُ الكمّيّةِ السريعةُ تُحسب كلُّها.**
     *
     * (أولويّةُ المالك، البند ١٥: «Rapid quantity changes».)
     *
     * **وضغطةٌ تضيع في العدّ تجعل الزبونَ يظنّ الزرَّ معطوباً فيضغط
     * أكثر** — ثمّ تصل كلُّها دفعةً فيصير خمسةً مكانَ اثنين.
     */
    @Test
    fun rapidQuantityTapsAllCount() {
        rule.setContent {
            RahalGoTheme(dark = false) {
                var qty by remember { mutableIntStateOf(1) }
                RahalButton(onClick = { qty++ }) { Text("زد · $qty") }
            }
        }
        repeat(7) {
            rule.onNodeWithText("زد · ${it + 1}").performClick()
        }
        rule.waitForIdle()
        rule.onNodeWithText("زد · 8").assertIsDisplayed()
    }

    /** UI-BTN-020 **وكلُّ زرٍّ ظاهرٍ قابلٌ للنقر فعلا.** */
    @Test
    fun buttonIsClickable() {
        rule.setContent {
            RahalGoTheme(dark = false) { RahalButton(onClick = {}) { Text("تأكيد") } }
        }
        rule.onNodeWithText("تأكيد").assertIsDisplayed().assertHasClickAction()
    }
}
