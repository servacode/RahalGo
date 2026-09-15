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
