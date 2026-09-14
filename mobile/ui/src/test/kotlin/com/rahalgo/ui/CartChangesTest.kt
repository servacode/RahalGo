package com.rahalgo.ui

import com.rahalgo.shared.model.CartChange
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الموافقةُ على حالٍ بعينها** (`CA`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والنصُّ يُقاس في الجهاز بالموارد** — **وهذه تقيس القرار**: **متى
 * يُمنَع الإرسال، ومتى تبطل موافقةٌ سابقة.**
 */
class CartChangesTest {

    private fun price(name: String, old: Long, new: Long) = CartChange(
        type = CartChanges.PRICE_CHANGED, name = name, oldValue = old, newValue = new,
    )

    private fun fp(total: Long, lines: List<Pair<String, Int>> = listOf("a" to 1)) =
        CartChanges.fingerprint(
            subtotal = total, deliveryFee = 0, total = total, discount = 0,
            lines = lines, point = "35.9,39.0", available = true,
        )

    // ═════════════════ CA-09 · CA-12 ═════════════════

    /** **CA-12 · ولا تبدّلَ ⇒ يُرسَل بلا مراجعةٍ زائدة.** */
    @Test
    fun `بلا تبدّلٍ لا تُطلَب موافقة`() {
        val gate = ReviewGate()
        assertTrue(
            "**سلّةٌ لم تتبدّل طُلبت لها موافقة** — **فيُقرأ كلُّ إنذارٍ بعدها ضجيجا**",
            gate.canSubmit(emptyList(), fp(10_000)),
        )
    }

    /** **CA-09 · وتبدّلٌ ⇒ لا يُرسَل قبل الموافقة.** */
    @Test
    fun `التبدّلُ يمنع الإرسال حتّى يُوافَق`() {
        val gate = ReviewGate()
        val changes = listOf(price("حمص", 10_000, 12_000))
        val f = fp(12_000)
        assertFalse("**أُرسل قبل المراجعة**", gate.canSubmit(changes, f))
        gate.accept(f)
        assertTrue("**وافق ولم يُسمَح له**", gate.canSubmit(changes, f))
    }

    // ═════════════════ CA-10 · CA-11 ═════════════════

    /**
     * **CA-11 · وتبدّلٌ ثانٍ يُبطل موافقةً أولى.**
     *
     * **ومن وافق على مجموعٍ ثمّ تبدّل قبل أن يضغط لم يوافق على
     * الجديد** — **وموافقةٌ تُجيز ما لم يُعرَض بعدُ ليست موافقة.**
     */
    @Test
    fun `تبدّلٌ ثانٍ يُبطل الموافقة`() {
        val gate = ReviewGate()
        val first = fp(12_000)
        gate.accept(first)
        assertTrue(gate.canSubmit(listOf(price("حمص", 10_000, 12_000)), first))

        // **ثمّ تبدّل ثانيةً قبل أن يضغط.**
        val second = fp(13_500)
        assertFalse(
            "**موافقةٌ قديمةٌ أجازت مجموعاً جديداً** — **فيُحاسَب بما لم يره**",
            gate.canSubmit(listOf(price("حمص", 12_000, 13_500)), second),
        )
    }

    /** **CA-10 · والموافقةُ تخصُّ حالَها لا غيرَها.** */
    @Test
    fun `البصمةُ تتبدّل بتبدّل أيّ رقمٍ رآه`() {
        val base = fp(12_000)
        assertNotEquals("**تبدّل الإجماليُّ ولم تتبدّل البصمة**", base, fp(12_500))
        assertNotEquals(
            "**تبدّل العددُ ولم تتبدّل البصمة**",
            base, fp(12_000, listOf("a" to 2)),
        )
        assertNotEquals(
            "**تبدّلت الأجورُ ولم تتبدّل البصمة**",
            base,
            CartChanges.fingerprint(12_000, 3_000, 12_000, 0, listOf("a" to 1), "35.9,39.0", true),
        )
        assertNotEquals(
            "**تبدّل العنوانُ ولم تتبدّل البصمة**",
            base,
            CartChanges.fingerprint(12_000, 0, 12_000, 0, listOf("a" to 1), "33.5,36.2", true),
        )
    }

    /** **وترتيبُ الأسطر ليس تبدّلاً** — **ولا تُطلَب موافقةٌ لأجله.** */
    @Test
    fun `ترتيبُ الأسطر لا يُبطل موافقة`() {
        assertEquals(
            fp(12_000, listOf("a" to 1, "b" to 2)),
            fp(12_000, listOf("b" to 2, "a" to 1)),
        )
    }

    // ═════════════════ CA-08 ═════════════════

    /** **CA-08 · وأربعةُ تبدّلاتٍ تُعرَض كلُّها.** */
    @Test
    fun `كلُّ التبدّلات تُقرأ لا أوّلُها`() {
        val changes = listOf(
            price("حمص", 10_000, 12_000),
            CartChange(type = CartChanges.PRODUCT_REMOVED, name = "فلافل"),
            CartChange(type = CartChanges.FEE_CHANGED, oldValue = 2_000, newValue = 3_000),
            CartChange(type = CartChanges.PROMO_CHANGED, oldValue = 3_000, newValue = 0),
        )
        assertTrue(CartChanges.needsReview(changes))
        assertEquals(4, changes.size)
        // **ولكلٍّ نوعُه** — **ولا يُجمَعان في واحد.**
        assertEquals(4, changes.map { it.type }.toSet().size)
    }

    // ═════════════════ QI-08 — والعقدُ حرفاً ═════════════════

    /**
     * **وأسماءُ الأنواع نصُّ المحرّك حرفاً** (`orders/changes.go`).
     *
     * **واسمٌ يتبدّل في المحرّك يُفرِغ الشاشةَ صامتاً** — **فلا
     * يُعرَض شيءٌ ولا يظهر خطأ**، **ويُظنّ أنّ لا تبدّلَ وقع.**
     *
     * **ويُقاس الطرفُ الآخرُ في `internal/qa`** — **وهذه تُثبّت ما
     * تقرؤه الشاشةُ منه.**
     */
    @Test
    fun `أسماءُ الأنواع كما يرسلها المحرّك`() {
        assertEquals("product_removed", CartChanges.PRODUCT_REMOVED)
        assertEquals("product_unavailable", CartChanges.PRODUCT_UNAVAILABLE)
        assertEquals("quantity_invalid", CartChanges.QUANTITY_INVALID)
        assertEquals("product_price_changed", CartChanges.PRICE_CHANGED)
        assertEquals("delivery_fee_changed", CartChanges.FEE_CHANGED)
        assertEquals("promo_or_discount_changed", CartChanges.PROMO_CHANGED)
    }

    /** **وعددٌ غيرُ مقبولٍ يوجب المراجعةَ كغيره.** */
    @Test
    fun `العددُ غيرُ المقبول يوجب المراجعة`() {
        val gate = ReviewGate()
        val qi = listOf(
            CartChange(type = CartChanges.QUANTITY_INVALID, name = "حمص", qty = 51),
        )
        assertTrue(CartChanges.needsReview(qi))
        assertFalse("**عددٌ غيرُ مقبولٍ لم يمنع الإرسال**", gate.canSubmit(qi, fp(10_000)))
        // **والعددُ المطلوبُ يبقى كما طلبه** — **لا يُقلَّم.**
        assertEquals(51, qi[0].qty)
    }

    // ═════════════════ CA-13 · CA-14 · CA-15 ═════════════════

    /**
     * **CA-13 · CA-14 · CA-15 · ولا يُمحى سطرٌ ولا يُقلَّم عددٌ ولا
     * يُستبدَل رقمٌ من تلقاء الشاشة.**
     *
     * **والتبدّلُ خبرٌ لا فعل** — **`ReviewGate` يمنع الإرسال ولا
     * يمسّ السلّة.**
     */
    @Test
    fun `التبدّلُ خبرٌ لا فعل`() {
        val gate = ReviewGate()
        val changes = listOf(
            CartChange(type = CartChanges.PRODUCT_REMOVED, name = "فلافل"),
            CartChange(type = CartChanges.QUANTITY_INVALID, name = "حمص", qty = 9),
        )
        // **ولا تملك البوّابةُ أن تحذف** — **لا واجهةَ لها إلى السلّة.**
        assertFalse(gate.canSubmit(changes, fp(10_000)))
        // **والعددُ المطلوبُ يبقى كما طلبه** — **يُعرَض ولا يُقلَّم.**
        assertEquals(9, changes[1].qty)
    }

    /** **وتفريغُ السلّة ينسى الموافقة.** */
    @Test
    fun `الإبطالُ ينسى الموافقة`() {
        val gate = ReviewGate()
        val f = fp(12_000)
        gate.accept(f)
        gate.reset()
        assertFalse(
            "**موافقةٌ نجت من تبديل الحال**",
            gate.canSubmit(listOf(price("حمص", 1, 2)), f),
        )
    }
}
