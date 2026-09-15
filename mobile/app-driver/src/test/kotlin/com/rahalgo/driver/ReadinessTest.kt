package com.rahalgo.driver

import com.rahalgo.driver.location.Readiness
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **جاهزيّةُ السائق — ما يمنع العمل وما لا يمنعه** (`DR`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وقراءةُ النظام تُقاس على جهازٍ حقيقيّ** — **وهذه تقيس القرار**:
 * **أيُّ نقصٍ يمنع العمل وأيُّه يُقال ولا يمنع.**
 *
 * **والعطبُ الذي كان**: **الإذنُ ممنوحٌ وخدمةُ الموقع مطفأة** —
 * **فيبدو جاهزاً ويفتح ورديّتَه ولا موقعَ يُرسَل.**
 */
class ReadinessTest {

    private fun state(vararg b: Readiness.Blocker) = Readiness.State(b.toList())

    /** **ولا نقصَ يعني جاهز.** */
    @Test
    fun `الفارغةُ جاهزة`() {
        assertTrue(state().ready)
        assertEquals(null, state().first)
    }

    /**
     * **DR-05 · وخدمةُ الموقع المطفأةُ تمنع العمل.**
     *
     * **وهي أخطرُها لأنّها لا تُرى** — **لا نافذةَ تُرفَض ولا رسالةَ
     * تظهر**، **والإذنُ ممنوحٌ فيُظنّ كلُّ شيءٍ بخير.**
     */
    @Test
    fun `خدمةُ الموقع المطفأةُ تمنع العمل`() {
        val s = state(Readiness.Blocker.LOCATION_SERVICE_OFF)
        assertFalse("**مطفأةٌ وقيل جاهز**", s.ready)
        assertEquals(Readiness.Blocker.LOCATION_SERVICE_OFF, s.first)
    }

    /** **DR-03 · وغيابُ الإذن كذلك.** */
    @Test
    fun `غيابُ الإذن يمنع العمل`() {
        assertFalse(state(Readiness.Blocker.LOCATION_PERMISSION_REQUIRED).ready)
    }

    /**
     * **DR-11 · DR-12 · وما يُقال ولا يمنع.**
     *
     * **والخلفيّةُ والإشعارُ ينقصان الجودةَ** — **ومن مُنع من فتح
     * ورديّته لأنّ إشعاراً غيرُ مسموحٍ حُرم عملَه لأجل زينة.**
     */
    @Test
    fun `الخلفيّةُ والإشعارُ يُقالان ولا يمنعان`() {
        val s = state(
            Readiness.Blocker.BACKGROUND_LOCATION_REQUIRED,
            Readiness.Blocker.NOTIFICATION_PERMISSION_REQUIRED,
        )
        // **ويُقالان** — **فالحالُ ليست جاهزةً تماماً.**
        assertFalse("**نقصٌ لم يُقَل**", s.ready)
        // **ولا يمنعان** — **والقرارُ في `canWork` لا في `ready`.**
        assertFalse(s.blockers.contains(Readiness.Blocker.LOCATION_PERMISSION_REQUIRED))
        assertFalse(s.blockers.contains(Readiness.Blocker.LOCATION_SERVICE_OFF))
    }

    /** **وأوّلُ ما يُقال واحدٌ** — **وقائمةٌ من أربعةٍ لا تُقرأ.** */
    @Test
    fun `يُقال أوّلُ النقص لا كلُّه`() {
        val s = state(
            Readiness.Blocker.LOCATION_PERMISSION_REQUIRED,
            Readiness.Blocker.BACKGROUND_LOCATION_REQUIRED,
        )
        assertEquals(Readiness.Blocker.LOCATION_PERMISSION_REQUIRED, s.first)
    }

    // ═════════════════ والقرارُ نفسُه يُقاس ═════════════════

    /**
     * **DR-05 · والحكمُ يُقاس لا الصفُّ وحدَه.**
     *
     * **وكان الفحصُ يبني `State` بيده ويقرؤها** — **فرُفع فحصُ
     * الخدمة من القرار ولم يسقط شيء.** **وحارسٌ لا يحرس القرارَ
     * زينة.**
     */
    @Test
    fun `الخدمةُ المطفأةُ تُنتج مانعاً من القرار نفسِه`() {
        val s = Readiness.evaluate(
            permission = true, service = false, background = true, notifications = true,
        )
        assertTrue(
            "**الإذنُ ممنوحٌ والخدمةُ مطفأةٌ ولم يُقَل شيء** — " +
                "**فيبدو جاهزاً ولا موقعَ يُرسَل**",
            s.blockers.contains(Readiness.Blocker.LOCATION_SERVICE_OFF),
        )
        assertFalse("**قيل جاهزٌ والموقعُ مطفأ**", Readiness.canWork(s))
    }

    /** **والكاملُ جاهزٌ يعمل.** */
    @Test
    fun `الكاملُ يعمل`() {
        val s = Readiness.evaluate(
            permission = true, service = true, background = true, notifications = true,
        )
        assertTrue(s.ready)
        assertTrue(Readiness.canWork(s))
    }

    /** **وغيابُ الإذن يمنع من القرار نفسِه.** */
    @Test
    fun `غيابُ الإذن يمنع من القرار`() {
        val s = Readiness.evaluate(
            permission = false, service = true, background = true, notifications = true,
        )
        assertFalse(Readiness.canWork(s))
        // **ولا تُسأل الخدمةُ قبل الإذن** — **ولا يُقال «شغّل الخدمة»
        // لمن لم يمنح الإذنَ بعد.**
        assertFalse(s.blockers.contains(Readiness.Blocker.LOCATION_SERVICE_OFF))
    }

    /** **DR-11 · والخلفيّةُ والإشعارُ لا يمنعان العمل.** */
    @Test
    fun `الخلفيّةُ والإشعارُ لا يمنعان العمل من القرار`() {
        val s = Readiness.evaluate(
            permission = true, service = true, background = false, notifications = false,
        )
        assertFalse("**نقصٌ لم يُقَل**", s.ready)
        assertTrue("**مُنع العملُ لأجل إشعارٍ وخلفيّة**", Readiness.canWork(s))
    }

    /**
     * **DR-01 · وأربعةُ أسبابٍ معروفةٌ لا رايةٌ عامّة.**
     *
     * **ومن أضاف سبباً خامساً يضيفه هنا** — **موضعٌ واحدٌ تقرؤه
     * الشاشاتُ كلُّها.**
     */
    @Test
    fun `الأسبابُ معدودةٌ ومسمّاة`() {
        assertEquals(4, Readiness.Blocker.values().size)
        assertTrue(
            Readiness.Blocker.values().contains(Readiness.Blocker.LOCATION_SERVICE_OFF),
        )
    }
}
