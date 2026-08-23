package com.rahalgo.driver

import com.rahalgo.driver.home.failureMessage
import com.rahalgo.map.data.MapFailure
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **كلُّ سببٍ يجد عبارةً — إغلاقُ `TD-MAP-SILENT-RESOURCE-FAIL`**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٢، البندان ٣ و٦.)
 *
 * **والحارسُ الحقيقيُّ هو الأخير**: من أضاف سبباً جديداً إلى
 * `MapFailure` **ونسي أن يعطيه عبارة** لم يكسر بناءً — `when` عندها
 * `else`. **فيقع العطبُ الصامتُ نفسُه من بابٍ آخر.**
 *
 * **وهذا يمرّ على كلّ قيم التعداد** فيسقط إن ظهر واحدٌ بلا عبارة.
 *
 * **ولا يُختبر النصُّ نفسُه** — يتبدّل بقرار المالك. **يُختبر أنّ ثمّ
 * نصّاً، وأنّ التصنيفات الثلاثة لا تختلط.**
 */
class FailureMessageTest {

    @Test
    fun `الشبكةُ والمهلةُ عبارةٌ واحدة`() {
        val net = failureMessage(MapFailure.NETWORK)
        assertEquals(net, failureMessage(MapFailure.TIMEOUT))
        assertNotEquals(net, failureMessage(MapFailure.NO_SPACE))
    }

    /** **ونقصُ التخزين عبارتُه وحدَه** — فما يفعله السائقُ مختلف. */
    @Test
    fun `نقصُ التخزين عبارةٌ مستقلّة`() {
        val space = failureMessage(MapFailure.NO_SPACE)
        assertNotEquals(space, failureMessage(MapFailure.NETWORK))
        assertNotEquals(space, failureMessage(MapFailure.CHECKSUM_MISMATCH))
    }

    /**
     * **وما لا يملك له السائقُ حيلةً يُجمع في عبارةٍ عامّة.**
     *
     * `CHECKSUM_MISMATCH` و`INVALID_CONTRACT` و`SECURITY` **لا تُقال
     * له** — لا يفعل بها شيئاً، **وقولُها يخيفه ولا يفيده.**
     */
    @Test
    fun `الأسبابُ التقنيّةُ تُجمع في عبارةٍ عامّة`() {
        val other = failureMessage(MapFailure.CHECKSUM_MISMATCH)
        for (f in listOf(
            MapFailure.SIZE_MISMATCH,
            MapFailure.INVALID_CONTRACT,
            MapFailure.INCOMPATIBLE_SCHEMA,
            MapFailure.SECURITY,
            MapFailure.INCOMPLETE,
            MapFailure.WRITE_FAILED,
        )) {
            assertEquals("$f لا يشارك العبارةَ العامّة", other, failureMessage(f))
        }
    }

    /** **وسببٌ غيرُ مصنَّفٍ لا يترك الشاشةَ صامتة.** */
    @Test
    fun `غيابُ التصنيف يعطي عبارةً أيضاً`() {
        assertTrue(failureMessage(null) != 0)
        assertEquals(failureMessage(null), failureMessage(MapFailure.WRITE_FAILED))
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **ولا سببَ بلا عبارة — ولو أُضيف غداً**
     * ══════════════════════════════════════════════════════════════════
     *
     * **وهذا هو الحارسُ الذي يبقى**: `when` بـ`else` لا يكسر بناءً
     * عند إضافةِ قيمة، **فالمترجمُ لا ينبّه.** وهذا يمرّ على التعداد
     * كلِّه.
     */
    @Test
    fun `كلُّ قيمةٍ في التعداد لها عبارة`() {
        for (f in MapFailure.entries) {
            assertTrue("$f بلا عبارة", failureMessage(f) != 0)
        }
    }
}
