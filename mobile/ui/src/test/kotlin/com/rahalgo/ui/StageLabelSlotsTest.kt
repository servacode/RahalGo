package com.rahalgo.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **أسماءُ مراحل الطلب سطرٌ واحدٌ لا تتراكب — والطريقُ من حافّةٍ إلى حافّة.**
 *
 * (فحصُ القبول ٢٠٢٦-١٠-٠٣: الأسماءُ تدخل بعضُها في بعض. **ثمّ قرارُ المالك
 *  ٢٠٢٦-١٠-١٥**: لا حشوةَ جانبيّة، والاسمُ سطرٌ واحد، والطرفان يحاذيان الحافّتين،
 *  **وإن لم تسع صغُر الخطُّ قليلاً لا التفّ.**)
 *
 * **والعرضُ عرضُ الشريط داخل البطاقة** — ٣٦٠ ناقصَ حشوة الشاشة والبطاقة.
 */
class StageLabelSlotsTest {

    private val inset = 6f

    /** **لا يتراكب جاران ولا يخرج اسمٌ عن الحافّتين.** */
    private fun assertLaidOut(width: Float, widths: List<Float>) {
        val x = stageLabelStarts(width, widths, inset)
        assertTrue("خرج الأوّلُ عن الحافّة: ${x[0]}", x[0] >= -0.001f)
        assertTrue("خرج الأخيرُ عن الحافّة", x.last() + widths.last() <= width + 0.001f)
        for (i in 0 until widths.size - 1) {
            assertTrue(
                "عرض $width: الاسم $i ينتهي عند ${x[i] + widths[i]} والتالي يبدأ عند ${x[i + 1]} — تتراكب",
                x[i] + widths[i] + LabelGap <= x[i + 1] + 0.001f,
            )
        }
    }

    /** **الطرفان على الحافّتين، والأوسطُ تحت عقدته** — حين لا زحام. */
    @Test
    fun `الطرفان يحاذيان الحافّتين والأوسط تحت عقدته`() {
        val w = 300f
        val ws = List(7) { 20f }
        val x = stageLabelStarts(w, ws, inset)
        assertEquals(0f, x[0], 0.001f)
        assertEquals(w, x[6] + 20f, 0.001f)
        val c = inset + (w - 2 * inset) * 3 / 6
        assertEquals(c - 10f, x[3], 0.001f)
    }

    /** **أسماءُ الزبون القصيرة بخطّها الكامل** — ثماني مراحل على أضيق بطاقة. */
    @Test
    fun `الأسماءُ القصيرة تسع بخطّها الكامل`() {
        // «بالانتظار · مقبول · سائق · عم يشتري · اشترى · بالطريق · وصل · تسلّم» تقريباً.
        val ws = listOf(46f, 26f, 22f, 42f, 28f, 36f, 18f, 26f)
        val (scale, got) = fitStageLabels(StageScales, 296f) { k -> ws.map { it * k } }
        assertEquals(1f, scale, 0f)
        assertEquals(ws, stageLabelWidths(296f, got))
        assertLaidOut(296f, got)
    }

    /** **وما لا يسع بالكامل يصغُر خطُّه قبل أيّ قصّ.** */
    @Test
    fun `الضيقُ يصغّر الخطّ أوّلاً`() {
        val ws = List(8) { 38f } // ٣٣٢ بفراغاتها على ٢٨٠ — وتسع بمقياس ٠٫٨.
        val (scale, got) = fitStageLabels(StageScales, 280f) { k -> ws.map { it * k } }
        assertTrue("لم يصغُر الخطّ", scale < 1f)
        assertTrue(stageLabelsFit(280f, got))
        assertEquals(got, stageLabelWidths(280f, got))
        assertLaidOut(280f, got)
    }

    /** **وآخرُ حيلةٍ القصُّ بنسبةٍ واحدة — ولا تراكبَ مهما ضاق.** */
    @Test
    fun `أضيقُ عرضٍ لا يتراكب`() {
        for (w in listOf(200f, 240f, 280f, 296f, 328f)) {
            for (n in 2..9) {
                val ws = List(n) { i -> 30f + (i % 3) * 25f }
                val (_, got) = fitStageLabels(StageScales, w) { k -> ws.map { it * k } }
                val final = stageLabelWidths(w, got)
                assertTrue("خانةٌ سالبة", final.all { it >= 0f })
                assertLaidOut(w, final)
            }
        }
    }
}
