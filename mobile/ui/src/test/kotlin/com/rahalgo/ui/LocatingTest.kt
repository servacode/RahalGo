package com.rahalgo.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«موقعي» — حالُ الزرّ وسببُ التعذّر** (`ML`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والقياسُ الزمنيُّ على جهازٍ حقيقيٍّ لا هنا** — **وهذه تقيس
 * الانتقالات**: **أنّ الضغطةَ تُغيّر الحالَ في الحال، وأنّ ضغطتين
 * طلبٌ واحد، وأنّ لكلّ تعذّرٍ سببَه.**
 */
class LocatingTest {

    // ═════════════════ ML-01 ═════════════════

    /**
     * **ML-01 · والضغطةُ تُغيّر الحالَ في اللحظة.**
     *
     * **وزرٌّ يُضغط ولا يتبدّل شيءٌ يُقرأ معطوباً** — **فيُضغط خمساً.**
     */
    @Test
    fun `الضغطةُ تُبدّل الحالَ فوراً`() {
        val l = Locating()
        assertEquals(Locating.Phase.IDLE, l.phase)
        assertFalse(l.busy)

        assertTrue("**الضغطةُ الأولى رُدّت**", l.start())
        assertEquals(Locating.Phase.ACQUIRING, l.phase)
        assertTrue("**الزرُّ لا يقول إنّه يعمل**", l.busy)
    }

    // ═════════════════ ML-04 ═════════════════

    /**
     * **ML-04 · وضغطتان سريعتان طلبٌ واحد.**
     *
     * **ولا نداءان يتسابقان** — **فيصل أقدمُهما آخراً فيُوسَّط به.**
     */
    @Test
    fun `الضغطُ المتكرّرُ لا يفتح نداءً ثانياً`() {
        val l = Locating()
        assertTrue(l.start())
        assertFalse("**ضغطةٌ ثانيةٌ فتحت نداءً ثانياً**", l.start())
        assertFalse(l.start())

        // **وبعد أن يُحسَم الأمرُ يُقبل طلبٌ جديد.**
        l.fixed()
        assertTrue("**بعد النجاح لم يُقبل طلبٌ جديد**", l.start())
    }

    // ═════════════════ ML-02 · ML-03 ═════════════════

    /**
     * **ML-02 · ML-03 · والقديمُ يُوسَّط به ثمّ يُصحَّح.**
     *
     * **والطلبُ يبقى قائماً** — **فالطازجُ لم يصل بعد**، **والزرُّ
     * يبقى يدور.**
     */
    @Test
    fun `القديمُ يُوسَّط ثمّ يُدقَّق`() {
        val l = Locating()
        l.start()
        l.provisional()
        assertTrue("**لم يُعلَم أنّه موضعٌ مؤقّت**", l.provisional)
        assertEquals(
            "**القديمُ أنهى الطلبَ** — **فلا يُصحَّح بطازج**",
            Locating.Phase.ACQUIRING,
            l.phase,
        )
        assertTrue(l.busy)

        l.fixed()
        assertEquals(Locating.Phase.FIXED, l.phase)
        assertFalse("**بقي مؤقّتاً بعد وصول الطازج**", l.provisional)
        assertFalse(l.busy)
    }

    /** **ولا يُعلَّم موضعٌ مؤقّتٌ بلا طلبٍ قائم.** */
    @Test
    fun `لا موضعَ مؤقّتٌ بلا طلب`() {
        val l = Locating()
        l.provisional()
        assertFalse(l.provisional)
        assertEquals(Locating.Phase.IDLE, l.phase)
    }

    // ═════════════════ ML-05 … ML-09 ═════════════════

    /**
     * **ولكلّ تعذّرٍ سببُه** — **ولا يُجمَع الكلُّ في «حدث خطأ».**
     *
     * **وعلاجُ كلٍّ غيرُ علاج الآخر**: **الإذنُ يُطلَب، والنهائيُّ
     * يُفتَح له الإعدادات، والخدمةُ المطفأةُ مفتاحٌ في النظام.**
     */
    @Test
    fun `لكلّ تعذّرٍ سببُه`() {
        for (why in Locating.Problem.values()) {
            val l = Locating()
            l.start()
            l.failed(why)
            assertEquals(Locating.Phase.FAILED, l.phase)
            assertEquals("**السببُ ضاع**: $why", why, l.problem)
            assertFalse("**بقي يدور بعد التعذّر**: $why", l.busy)
        }
        // **وستّةُ أسبابٍ لا سببٌ واحد.**
        assertEquals(6, Locating.Problem.values().size)
    }

    /** **وبعد التعذّر يُقبل طلبٌ جديد** — **ولا يُحبَس على فشل.** */
    @Test
    fun `بعد التعذّر يُعاد المحاولة`() {
        val l = Locating()
        l.start()
        l.failed(Locating.Problem.TIMEOUT)
        assertTrue("**لم يُقبل طلبٌ بعد التعذّر**", l.start())
        assertNull("**سببٌ قديمٌ بقي معروضاً**", l.problem)
    }

    /** **والتصفيرُ يعيدها ساكنة.** */
    @Test
    fun `التصفيرُ يُسكّن الحال`() {
        val l = Locating()
        l.start()
        l.failed(Locating.Problem.SERVICE_OFF)
        l.reset()
        assertEquals(Locating.Phase.IDLE, l.phase)
        assertNull(l.problem)
        assertFalse(l.busy)
    }

    // ═════════════════ ML-05 — الدقّة ═════════════════

    /**
     * **وتوسيطُ خريطةٍ يقبل ما لا يقبله تأكيدُ نقطة.**
     *
     * **ومئةُ مترٍ لا تُرى على مستوى المدينة** — **وهي بابٌ آخرُ في
     * شارع.**
     */
    @Test
    fun `لكلّ غرضٍ حدُّ دقّته`() {
        assertTrue(Accuracy.enough(80f, Accuracy.CENTER_M))
        assertTrue(Accuracy.enough(80f, Accuracy.CONFIRM_M))
        assertTrue(Accuracy.enough(80f, Accuracy.DRIVE_M))

        // **ومئتان تكفي للتوسيط والتأكيد ولا تكفي للتتبّع.**
        assertTrue(Accuracy.enough(200f, Accuracy.CENTER_M))
        assertTrue(Accuracy.enough(200f, Accuracy.CONFIRM_M))
        assertFalse("**قياسٌ خشنٌ قُبل لتتبّع سائق**", Accuracy.enough(200f, Accuracy.DRIVE_M))

        // **وألفان لا تكفي لشيء.**
        assertFalse(Accuracy.enough(2000f, Accuracy.CENTER_M))
    }

    /**
     * **ودقّةٌ لا تُعرَف لا تكفي لشيء.**
     *
     * **ولا يُحكَم بقياسٍ لا يقول كم يُخطئ** — **والصمتُ أصدقُ من
     * حكمٍ بنقطةٍ مجهولة** (كنقطة الاستكشاف).
     */
    @Test
    fun `الدقّةُ المجهولةُ لا تكفي`() {
        assertFalse(Accuracy.enough(-1f, Accuracy.CENTER_M))
        assertFalse(Accuracy.enough(-1f, Accuracy.DRIVE_M))
    }

    /**
     * **ML-11 · وحدُّ السائق أشدُّ من حدّ الزبون.**
     *
     * **والمسافةُ والأجرةُ تُبنيان على أثره** — **وقياسٌ خشنٌ يُنتج
     * مساراً متعرّجاً ومسافةً كاذبة.**
     */
    @Test
    fun `حدُّ السائق أشدُّ`() {
        assertTrue(Accuracy.DRIVE_M < Accuracy.CONFIRM_M)
        assertTrue(Accuracy.CONFIRM_M < Accuracy.CENTER_M)
    }
}
