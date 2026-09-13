package com.rahalgo.ui

import androidx.compose.material3.DrawerState
import androidx.compose.material3.DrawerValue
import org.junit.After
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سياسةُ إيماءةِ الدرج — تُقاس بالنداء** (`DWR`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وهذه تقيس القاعدةَ نفسَها** — **لا نصَّ ملفٍّ ولا رسمَ شاشة.**
 * **وحالُ الدرج يُصنَع بـ`DrawerState` حقيقيّةٍ** كما يصنعها التطبيق،
 * **فالجوابُ جوابُ المنتَج لا محاكاةً له.**
 *
 * **وأثرُ الحضور والخروج** (`MapGestureLock`) **يُقاس على الجهاز** في
 * `DrawerGestureUiTest` — **لأنّه تركيبٌ لا حساب.**
 */
class DrawerGesturePolicyTest {

    private fun closed() = DrawerState(DrawerValue.Closed)
    private fun open() = DrawerState(DrawerValue.Open)

    @Before
    fun clean() = DrawerGestures.resetForTest()

    @After
    fun after() = DrawerGestures.resetForTest()

    /** **DWR-1 · شاشةٌ عاديّةٌ: الإيماءةُ مفتوحة.** */
    @Test
    fun `بلا خريطةٍ تُفتَح الإيماءة`() {
        assertFalse("المقدّمةُ مكسورة: خريطةٌ محسوبةٌ بلا سبب", DrawerGestures.mapActive)
        assertTrue(DrawerGestures.enabledFor(closed()))
    }

    /** **DWR-5 · خريطةٌ حاضرةٌ: إيماءةُ الفتح مقفلة.** */
    @Test
    fun `الخريطةُ تقفل إيماءةَ الفتح`() {
        DrawerGestures.enterMap()
        assertTrue(DrawerGestures.mapActive)
        assertFalse(
            "**سحبُ الحافّة يفتح الدرجَ والخريطةُ حاضرة** — وهو بلاغُ ٢٠٢٦-٠٩-٠١",
            DrawerGestures.enabledFor(closed()),
        )
    }

    /** **DWR-2 · والمفتوحُ يبقى يُغلَق بالسحب ولو كانت خريطةٌ حاضرة.** */
    @Test
    fun `الدرجُ المفتوحُ لا يُحبَس`() {
        DrawerGestures.enterMap()
        assertTrue(
            "**درجٌ مفتوحٌ لا يُغلَق بالسحب يحبس صاحبَه**",
            DrawerGestures.enabledFor(open()),
        )
    }

    /** **DWR-7 · وبالخروج من الخريطة ترجع الإيماءةُ بلا إعادة تشغيل.** */
    @Test
    fun `الخروجُ من الخريطة يُرجع الإيماءة`() {
        DrawerGestures.enterMap()
        DrawerGestures.exitMap()
        assertFalse(DrawerGestures.mapActive)
        assertTrue(DrawerGestures.enabledFor(closed()))
    }

    /**
     * **DWR-8 · وخريطتان متراكبتان لحظةَ الانتقال.**
     *
     * **ورايةٌ ثنائيّةٌ تُطفئها الأولى وهي تختفي فتفتح القفلَ والثانيةُ
     * حاضرة** — **والعدّادُ لا يُخطئ.**
     */
    @Test
    fun `خريطتان متراكبتان لا تفتحان القفلَ مبكّراً`() {
        DrawerGestures.enterMap()
        DrawerGestures.enterMap()
        DrawerGestures.exitMap()
        assertTrue("**القفلُ انفتح والخريطةُ الثانيةُ حاضرة**", DrawerGestures.mapActive)
        assertFalse(DrawerGestures.enabledFor(closed()))
        DrawerGestures.exitMap()
        assertTrue(DrawerGestures.enabledFor(closed()))
    }

    /** **ولا ينزل العدّادُ تحت الصفر** — فخروجٌ زائدٌ لا يفتح ما لا يُقفَل. */
    @Test
    fun `خروجٌ زائدٌ لا يُفسد العدّاد`() {
        DrawerGestures.exitMap()
        DrawerGestures.exitMap()
        DrawerGestures.enterMap()
        assertFalse(
            "**خروجٌ زائدٌ أنزل العدّادَ تحت الصفر فصار القفلُ لا يُقفل**",
            DrawerGestures.enabledFor(closed()),
        )
    }
}
