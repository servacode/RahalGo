package com.rahalgo.ui

import com.rahalgo.shared.net.ApiClient
import java.io.IOException
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **هويّةُ المحاولة — ألف تبقى بعد موت العمليّة؟** (`SR`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والمحرّكُ يحفظ الردَّ أربعاً وعشرين ساعةً لمن حمل المفتاحَ نفسَه**
 * (`idempotency.go`) — **وإنّما كان الجهازُ ينسى المفتاح.**
 *
 * **فهذه تقيس الذاكرةَ لا الحماية**: **أنّ المفتاحَ واحدٌ لمحاولةٍ
 * واحدةٍ مهما مات التطبيقُ بينهما.**
 */
class AttemptTest {

    /** **قرصٌ في الذاكرة** — **يبقى وإن «ماتت العمليّة».** */
    private class Mem : Attempt.Store {
        val rows = mutableMapOf<String, String>()
        override fun get(key: String): String? = rows[key]
        override fun put(key: String, value: String) { rows[key] = value }
        override fun remove(key: String) { rows.remove(key) }
    }

    private lateinit var disk: Mem

    @Before
    fun setUp() {
        disk = Mem()
        Attempt.resetForTest(disk)
    }

    /** **SR-03 · محاولةٌ واحدةٌ مفتاحٌ واحد** — ولو سُئل مرارا. */
    @Test
    fun `المحاولةُ الواحدةُ مفتاحٌ واحد`() {
        val a = Attempt.key(Attempt.ORDER)
        val b = Attempt.key(Attempt.ORDER)
        assertEquals("**ضغطتان ولّدتا مفتاحين**", a, b)
        assertTrue(Attempt.pending(Attempt.ORDER))
    }

    /**
     * **SR-05 · SR-08 · وموتُ العمليّة لا ينسى.**
     *
     * **وهذا هو العطبُ بعينه**: **أنشأ المحرّكُ الطلبَ ثمّ قُتل
     * التطبيقُ قبل الجواب** — **فلو وُلِّد مفتاحٌ جديدٌ لَصار طلبان.**
     */
    @Test
    fun `المفتاحُ يبقى بعد موت العمليّة`() {
        val before = Attempt.key(Attempt.ORDER)

        // **تموت العمليّةُ** — **وتُعاد التهيئةُ فوق القرص عينِه.**
        Attempt.resetForTest(null)
        Attempt.resetForTest(disk)

        val after = Attempt.key(Attempt.ORDER)
        assertEquals(
            "**التطبيقُ نسي مفتاحَه بعد القتل** — **فطلبان وسائقان وخصمان**",
            before,
            after,
        )
    }

    /** **ويُمحى بعد النجاح** — **ومفتاحٌ يبقى يردّ جوابَ ما قبله.** */
    @Test
    fun `المفتاحُ يُمحى بعد النجاح`() {
        val first = Attempt.key(Attempt.ORDER)
        Attempt.clear(Attempt.ORDER)
        assertFalse(Attempt.pending(Attempt.ORDER))
        val second = Attempt.key(Attempt.ORDER)
        assertNotEquals("**الطلبُ التالي حمل مفتاحَ سابقِه**", first, second)
    }

    /** **SR-13 · SR-14 · وخانةٌ لكلّ نوعِ إنشاء.** */
    @Test
    fun `العاديُّ والمخصَّصُ لا يتشاركان مفتاحاً`() {
        val order = Attempt.key(Attempt.ORDER)
        val custom = Attempt.key(Attempt.CUSTOM)
        assertNotEquals(
            "**مفتاحٌ واحدٌ للنوعين** — **فيردّ المخصَّصُ جوابَ العاديّ**",
            order,
            custom,
        )
        Attempt.clear(Attempt.ORDER)
        assertTrue("**مسحُ العاديّ محا المخصَّص**", Attempt.pending(Attempt.CUSTOM))
    }

    // ═════════════════ SR-07 — أحُسم الأمرُ أم لا يُدرى؟ ═════════════════

    /**
     * **وردُّ المحرّك حسمٌ** — **وصل النداءُ وأجاب.**
     *
     * **وانقطاعُ الشبكة ليس حسماً** — **قد يكون الطلبُ قُيِّد وضاع
     * الجواب**، **ومن قيل له «فشل» وهو لم يفشل يعيد الكرّة.**
     */
    @Test
    fun `الشبكةُ المقطوعةُ ليست حسماً`() {
        assertTrue(
            "**ردُّ المحرّك لم يُعَدّ حسماً**",
            isDecided(ApiClient.ApiException(400, com.rahalgo.shared.model.ApiErrorBody(code = "validation"))),
        )
        assertFalse(
            "**انقطاعُ الشبكة عُدّ فشلاً** — **فيُنشَأ طلبٌ ثانٍ**",
            isDecided(IOException("لا شبكة")),
        )
    }
}
