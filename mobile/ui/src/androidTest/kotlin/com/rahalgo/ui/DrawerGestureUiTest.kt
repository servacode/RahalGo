package com.rahalgo.ui

/**
 * **إيماءةُ الدرج — تُقاس بسحبٍ حقيقيٍّ على جهازٍ حقيقيّ.**
 *
 * المعرّفات: `DWR-*` · الوسم: `@ui @critical @release`
 *
 * (بلاغُ المالك ٢٠٢٦-٠٩-٠١: «عند سحب الخريطة تُفتح القائمةُ الجانبيّة —
 *  بكلّ التطبيقات». وقرارُه ٢٠٢٦-٠٩-١٤: **تُفتح بالسحب، وتُقفَل على
 *  الخريطة وحدَها.**)
 *
 * # ولمَ على الجهاز لا في الحساب
 *
 * **وقاعدةُ السياسة تُقاس بالنداء** في `DrawerGesturePolicyTest` —
 * **وهذه تقيس ما لا يُحسَب**: **أنّ `ModalNavigationDrawer` يستجيب
 * للسحب فعلاً**، **وأنّ حضورَ الخريطة يقفله**، **وأنّ الخروجَ يُرجعه.**
 *
 * # ولمَ يُقاس الحالُ لا وجودُ العقدة
 *
 * **و`ModalNavigationDrawer` يُركّب درجَه دائماً** — مفتوحاً كان أو
 * مغلقاً، **وإنّما يُزيحه خارجَ الشاشة.** **فاختبارٌ يسأل «أموجودةٌ
 * القائمة؟» يمرّ وإن لم يُفتَح شيء** — **وقد وقع ذلك فيّ أوّلَ مرّة.**
 *
 * **فيُقاس `drawerState.currentValue`** — **وهو ما يقرؤه المنتَج نفسُه.**
 *
 * # والاتّجاهُ عربيٌّ
 *
 * **والحافّةُ في العربيّة يمنى** — **فالسحبُ يميناً⇐يساراً يفتح.**
 */
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.DrawerState
import androidx.compose.material3.DrawerValue
import androidx.compose.material3.ModalDrawerSheet
import androidx.compose.material3.ModalNavigationDrawer
import androidx.compose.material3.Text
import androidx.compose.material3.rememberDrawerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.swipeLeft
import androidx.compose.ui.test.swipeRight
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Rule
import org.junit.Test

class DrawerGestureUiTest {

    @get:Rule
    val rule = createComposeRule()

    private var state: DrawerState? = null

    @Before
    fun clean() {
        DrawerGestures.resetForTest()
        state = null
    }

    @After
    fun after() = DrawerGestures.resetForTest()

    /**
     * Shell **قشرةٌ مصغَّرةٌ بالقاعدة عينِها** — **لا بقاعدةٍ ثانيةٍ
     * تُكتب للاختبار.** `gesturesEnabled = DrawerGestures.enabledFor(...)`
     * هو ما يكتبه التطبيقُ حرفاً.
     *
     * **ولا سمةَ هنا**: **السمةُ زينةٌ والقياسُ للإيماءة** — **وتحميلُ
     * خطوطِ التصميم في فحصٍ يُطيله ولا يُفيده.**
     */
    @Composable
    private fun Shell(withMap: Boolean) {
        val drawer = rememberDrawerState(DrawerValue.Closed)
        state = drawer
        ModalNavigationDrawer(
            drawerState = drawer,
            gesturesEnabled = DrawerGestures.enabledFor(drawer),
            drawerContent = { ModalDrawerSheet { Text("القائمة") } },
        ) {
            Column(Modifier.fillMaxSize().testTag("screen")) {
                Text("الشاشة")
                if (withMap) {
                    // **وسطحُ خريطةٍ يُعلن نفسَه كما يفعل `MapCanvas`.**
                    Box(Modifier.fillMaxSize().testTag("map")) {
                        MapGestureLock()
                        Text("خريطة")
                    }
                }
            }
        }
    }

    private fun value(): DrawerValue =
        requireNotNull(state) { "لم تُركَّب القشرة" }.currentValue

    /** **DWR-1 · سحبةٌ من الحافّة تفتح الدرج.** */
    @Test
    fun dwr1_edgeSwipeOpensDrawer() {
        rule.setContent { Shell(withMap = false) }
        rule.waitForIdle()
        assertEquals("المقدّمةُ مكسورة: الدرجُ مفتوحٌ قبل السحب",
            DrawerValue.Closed, value())

        rule.onNodeWithTag("screen").performTouchInput { swipeRight() }
        rule.waitForIdle()
        assertEquals("**سحبُ الحافّة لم يفتح الدرج**", DrawerValue.Open, value())
    }

    /** **DWR-2 · وسحبةٌ معاكسةٌ تُغلقه.** */
    @Test
    fun dwr2_reverseSwipeClosesDrawer() {
        rule.setContent { Shell(withMap = false) }
        rule.onNodeWithTag("screen").performTouchInput { swipeRight() }
        rule.waitForIdle()
        assertEquals(DrawerValue.Open, value())

        rule.onNodeWithTag("screen").performTouchInput { swipeLeft() }
        rule.waitForIdle()
        assertEquals("**الدرجُ المفتوحُ لم يُغلَق بالسحب**",
            DrawerValue.Closed, value())
    }

    /**
     * **DWR-5 · وخريطةٌ حاضرةٌ تمنع الفتحَ بالسحب.**
     *
     * **وهذا بلاغُ المالك بعينه** — **والسحبُ فوق الخريطة يبقى للخريطة.**
     */
    @Test
    fun dwr5_activeMapBlocksOpenSwipe() {
        rule.setContent { Shell(withMap = true) }
        rule.waitForIdle()
        assertEquals("القفلُ لم يُسجَّل", 1, DrawerGestures.interactiveMaps)

        rule.onNodeWithTag("map").performTouchInput { swipeRight() }
        rule.waitForIdle()
        assertEquals("**سحبُ الخريطة فتح الدرج** — وهو العطبُ نفسُه",
            DrawerValue.Closed, value())
    }

    /** **DWR-7 · والخروجُ من الخريطة يُرجع الإيماءةَ بلا إعادةِ تشغيل.** */
    @Test
    fun dwr7_leavingMapRestoresSwipe() {
        var hasMap by mutableStateOf(true)
        rule.setContent { Shell(withMap = hasMap) }
        rule.waitForIdle()
        assertEquals(1, DrawerGestures.interactiveMaps)

        rule.onNodeWithTag("map").performTouchInput { swipeRight() }
        rule.waitForIdle()
        assertEquals(DrawerValue.Closed, value())

        hasMap = false
        rule.waitForIdle()
        assertEquals("**القفلُ بقي بعد مغادرة الخريطة**",
            0, DrawerGestures.interactiveMaps)

        rule.onNodeWithTag("screen").performTouchInput { swipeRight() }
        rule.waitForIdle()
        assertEquals("**الإيماءةُ لم ترجع بعد الخريطة**",
            DrawerValue.Open, value())
    }

    /** **DWR-8 · وسحباتٌ متكرّرةٌ لا تُعلّق الحال.** */
    @Test
    fun dwr8_rapidSwipesLeaveNoStuckState() {
        rule.setContent { Shell(withMap = false) }
        repeat(4) {
            rule.onNodeWithTag("screen").performTouchInput { swipeRight() }
            rule.waitForIdle()
            rule.onNodeWithTag("screen").performTouchInput { swipeLeft() }
            rule.waitForIdle()
        }
        assertEquals("**الدرجُ عالقٌ بعد سحباتٍ متكرّرة**",
            DrawerValue.Closed, value())
        assertEquals(0, DrawerGestures.interactiveMaps)
    }
}
