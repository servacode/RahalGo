package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **عدّادُ الطلبات — البندان ٦ و٨**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك نصّاً**:
 *
 *	اختبار إلزامي: user selects alternative → verify: عدد alternative
 *	network requests يبقى = 1 لا 2.
 *
 * **والمعدودُ هنا هو الشرطُ الإنتاجيُّ نفسُه** — `shouldFetch` التي
 * تناديها [com.rahalgo.driver.trip.TripScreen] في أثرها، **لا نسخةً
 * منها.** والمزيَّفُ هو الشبكةُ وحدَها.
 *
 * (إغلاقُ صحّة واجهة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 */
class AltFetchCountTest {

    /**
     * **محاكي الأثر** — يستدعي الشرطَ عند كلّ تبدّل، ويعدّ.
     *
     * **والمفاتيحُ هي مفاتيحُ `LaunchedEffect` بعينها**: الطلبُ
     * والطورُ والجيلُ والصحّة. **فكلُّ تبدّلٍ في واحدٍ منها يعيد
     * التقييم** — وهذا أسوأُ ما قد يقع، لا أفضلُه.
     */
    private class Screen {
        var hasRoute = true
        var healthy = true
        var hasOrigin = true
        var reason = RouteInstallReason.INITIAL
        var generation = 1L
        var requests = 0
            private set

        /** **تبدّلَ مفتاحٌ فأُعيد التقييم.** */
        fun recompose() {
            if (RouteChoiceGate.shouldFetch(hasRoute, healthy, hasOrigin, reason)) requests++
        }

        /** **رُكّب مسارٌ بسببٍ ما** — الجيلُ يزيد ثمّ يُعاد التقييم. */
        fun install(next: RouteInstallReason) {
            generation++
            reason = next
            recompose()
        }
    }

    @Test
    fun `اختيارُ السائق لا يزيد الطلبات`() {
        val s = Screen()

        // ── ١ · بدءُ الساق: طلبٌ واحد ─────────────────────────────
        s.recompose()
        assertEquals("أوّلُ جلبة", 1, s.requests)

        // ── ٢ · واختار السائقُ بديلاً فرُكّب ──────────────────────
        s.install(RouteInstallReason.USER_SELECTION)

        assertEquals("**بقيت واحدة**", 1, s.requests)
    }

    @Test
    fun `ولا نبضاتُ إعادة التركيب بعده تزيدها`() {
        /**
         * **العيبُ الحقيقيُّ أعمقُ من نبضةٍ واحدة**: بعد الاختيار
         * تتبدّل مفاتيحُ أخرى — موقعٌ، صحّةٌ، طور — **وكلُّ تبدّلٍ
         * يعيد التقييم.**
         *
         * **فلو كان الشرطُ على الجيل وحدَه لعادت اللوحةُ.**
         */
        val s = Screen()
        s.recompose()
        s.install(RouteInstallReason.USER_SELECTION)

        repeat(20) { s.recompose() }

        assertEquals(1, s.requests)
    }

    @Test
    fun `وإعادةُ الحساب التلقائيّةُ تطلب`() {
        // **البند ٩** — يجوز لها أن تطلب، **وتتميّز عن الاختيار.**
        val s = Screen()
        s.recompose()
        s.install(RouteInstallReason.USER_SELECTION)
        assertEquals(1, s.requests)

        // ── خرج عن المسار فأُعيد حسابُه ونجح ──────────────────────
        s.healthy = false
        s.recompose()
        assertEquals("ولا تُطلب والحالُ سيّئ", 1, s.requests)

        s.healthy = true
        s.install(RouteInstallReason.AUTOMATIC_REROUTE)
        assertEquals("**فلمّا استقرّ طُلبت**", 2, s.requests)
    }

    @Test
    fun `وتبدّلُ الوجهة يطلب`() {
        val s = Screen()
        s.recompose()
        s.install(RouteInstallReason.USER_SELECTION)
        // ── استُلم الطلبُ فصارت الوجهةُ الزبون ────────────────────
        s.install(RouteInstallReason.TARGET_CHANGED)
        assertEquals(2, s.requests)
    }

    @Test
    fun `ولا جلبَ قبل تركيب المسار`() {
        val s = Screen()
        s.hasRoute = false
        repeat(5) { s.recompose() }
        assertEquals(0, s.requests)
    }

    @Test
    fun `ولا جلبَ بلا موضعٍ للسائق`() {
        val s = Screen()
        s.hasOrigin = false
        repeat(5) { s.recompose() }
        assertEquals(0, s.requests)
    }

    @Test
    fun `والحالُ السيّئُ يمنع كلَّ الأسباب`() {
        // **البند ١٠** — لا استثناءَ لسببٍ دون سبب.
        for (r in RouteInstallReason.entries) {
            val s = Screen()
            s.healthy = false
            s.reason = r
            repeat(5) { s.recompose() }
            assertEquals("$r", 0, s.requests)
        }
    }
}
