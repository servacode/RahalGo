package com.rahalgo.ui

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **أسماءُ مراحل الطلب لا تتراكب** — على هاتفٍ بعرض ٣٦٠ وفي العربيّة.
 *
 * (فحصُ القبول ٢٠٢٦-١٠-٠٣: «بانتظار القبول · مقبول · قيد التحضير · في
 *  الطريق · وصل إليك · تم التسليم» تدخل بعضُها في بعض، و«السائق في طريقه
 *  لشراء طلبك».)
 *
 * **والعرضُ عرضُ الشريط داخل البطاقة** — ٣٦٠ ناقصَ حشوة الشاشة والبطاقة.
 */
class StageLabelSlotsTest {

    /** **لا يتراكب جاران في الصفّ نفسِه** — بعد حشوةٍ نصفُها خانة. */
    private fun assertNoOverlap(width: Float, count: Int, fits: (Float) -> Boolean) {
        val slots = stageLabelSlots(width, count, fits)
        val step = (width - slots.width) / (count - 1)
        // **وفي الصفّين جارُ الاسم في صفّه على مسافة عقدتين.**
        val neighbour = if (slots.staggered) step * 2 else step
        assertTrue(
            "عرض $width و$count مراحل: خانةٌ ${slots.width} والمسافة $neighbour — تتراكب",
            slots.width + LabelGap <= neighbour + 0.001f,
        )
        assertTrue("خانةٌ بلا عرض", slots.width > 20f)
    }

    @Test
    fun `ستُّ مراحل على هاتف ٣٦٠ لا تتراكب`() {
        for (w in listOf(280f, 296f, 312f, 328f)) assertNoOverlap(w, 6) { true }
    }

    /** **وسبعُ مراحل الطلب الخاصّ بأسمائها الطويلة تنزل صفّين.** */
    @Test
    fun `أسماءُ الطلب الخاصّ الطويلة تنزل صفّين ولا تتراكب`() {
        for (w in listOf(280f, 296f, 328f)) {
            assertNoOverlap(w, 7) { false }
            assertTrue(stageLabelSlots(w, 7) { false }.staggered)
        }
    }

    /** **وما يسع صفّاً واحداً يبقى صفّاً واحداً** — لا يُنزَل بلا داعٍ. */
    @Test
    fun `ما يسع يبقى صفّاً واحداً`() {
        assertFalse(stageLabelSlots(296f, 6) { true }.staggered)
    }
}
