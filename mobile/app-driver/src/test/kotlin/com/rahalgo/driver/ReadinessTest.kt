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

    /**
     * **DR-11 · والخلفيّةُ والإشعارُ لا يمنعان رحلةً في اليد.**
     *
     * **و`canWork` معناها الآن: أيُنتَج موقع؟** — **والدرجاتُ في
     * `allows`.**
     */
    @Test
    fun `الخلفيّةُ والإشعارُ لا يمنعان رحلةً في اليد`() {
        val s = Readiness.evaluate(
            permission = true, service = true, background = false, notifications = false,
        )
        assertFalse("**نقصٌ لم يُقَل**", s.ready)
        assertTrue("**مُنعت رحلةٌ في يده لأجل إشعارٍ وخلفيّة**", Readiness.canWork(s))
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

    // ══════════════ ودرجاتٌ لا رايةٌ واحدة (`DRF`) ══════════════
    //
    // **وقِيس في المستودع ٢٠٢٦-٠٩-١٥** — **لا يُقرَّر بالحدس**:
    //
    // **الإشعارُ**: **كشفُ الطلبات دفعٌ من `FCM`**،
    // **و`RahalPushService.onMessageReceived` لا تفعل غيرَ بناءِ
    // إشعارٍ** (`manager.notify`): **لا تُنعش حالاً ولا تُبلّغ بطريقٍ
    // آخر.** **فالإذنُ مرفوضٌ ⇒ تصل الرسالةُ ويُسقطها النظامُ صامتاً**
    // — **والجوّالُ في الجيب فلا يرى شيئاً.**
    //
    // **والخلفيّةُ**: **الخدمةُ أماميّةٌ من نوع `location`** وتُشغَّل
    // **والتطبيقُ مرئيّ** — **فيُبقي أندرويد الموقعَ بلا إذن
    // الخلفيّة**: **فالمسارُ الطبيعيُّ يعمل.**

    private fun full(
        permission: Boolean = true,
        service: Boolean = true,
        background: Boolean = true,
        notifications: Boolean = true,
    ) = Readiness.evaluate(permission, service, background, notifications)

    /** **DRF-01 · وثلاثُ درجاتٍ مسمّاةٌ لا رايةٌ عامّة.** */
    @Test
    fun `الدرجاتُ معدودةٌ ومسمّاة`() {
        assertEquals(3, Readiness.Level.values().size)
    }

    /**
     * **DRF-02 · والسجلُّ والإعداداتُ تُفتح على كلّ حال.**
     *
     * **ومن مُنع من قراءة سجلّه لأنّ موقعَه مطفأٌ عُوقب بلا سبب.**
     */
    @Test
    fun `التطبيقُ يُستعمل وإن نقص كلُّ شيء`() {
        val s = full(permission = false, service = false, background = false, notifications = false)
        assertTrue(
            "**مُنع من قراءة سجلّه لأنّ موقعَه مطفأ**",
            Readiness.allows(s, Readiness.Level.APP_USABLE),
        )
    }

    /**
     * **DRF-03 · ولا موقعَ ⇒ لا إعلانَ ولا رحلة.**
     *
     * **والإذنُ ممنوحٌ والخدمةُ مطفأةٌ داخلةٌ في هذا** — **وهي التي لا
     * تُرى.**
     */
    @Test
    fun `الخدمةُ المطفأةُ تمنع الدرجتين العمليّتين`() {
        val s = full(service = false)
        assertFalse(Readiness.allows(s, Readiness.Level.CAN_GO_ONLINE))
        assertFalse(Readiness.allows(s, Readiness.Level.CAN_RUN_ACTIVE_DELIVERY))
    }

    /** **DRF-03b · وغيابُ الإذن كذلك.** */
    @Test
    fun `غيابُ الإذن يمنع الدرجتين العمليّتين`() {
        val s = full(permission = false)
        assertFalse(Readiness.allows(s, Readiness.Level.CAN_GO_ONLINE))
        assertFalse(Readiness.allows(s, Readiness.Level.CAN_RUN_ACTIVE_DELIVERY))
    }

    /**
     * **DRF-04 · والإشعارُ يمنع الإعلانَ عن التوفّر — بقياس.**
     *
     * **ومن أُعلن متاحاً ولا يبلغه النداءُ يُحسَب رافضاً وهو لا يعلم** —
     * **فيُنقص تقييمُه بعملٍ لم يرَه.**
     *
     * **وكنتُ قلتُ في تقريرٍ سابقٍ إنّه «زينة»** — **ولم أكن قِستُ**:
     * **وقياسُ `onMessageReceived` أظهر أنّه طريقُ العمل نفسُه.**
     */
    @Test
    fun `الإشعارُ الناقصُ يمنع الإعلانَ عن التوفّر`() {
        val s = full(notifications = false)
        assertFalse(
            "**أُعلن متاحاً ولا نداءَ يبلغه** — **فيُحسَب رافضاً**",
            Readiness.allows(s, Readiness.Level.CAN_GO_ONLINE),
        )
    }

    /**
     * **DRF-05 · ولا يُوقَف عن رحلةٍ في يده لأجل الإشعار.**
     *
     * **والطلبُ مقبولٌ وهو يعرفه** — **فلا يحتاج نداءً يُوقظه**،
     * **ومن مُنع من إكمال تسليمٍ بيده حُبس زبونُه معه.**
     */
    @Test
    fun `الإشعارُ الناقصُ لا يوقف رحلةً جارية`() {
        val s = full(notifications = false)
        assertTrue(
            "**أُوقفت رحلةٌ في يده لأجل إذنِ إشعار**",
            Readiness.allows(s, Readiness.Level.CAN_RUN_ACTIVE_DELIVERY),
        )
    }

    /**
     * **DRF-06 · والخلفيّةُ لا تمنع درجةً — بقياس.**
     *
     * **والخدمةُ أماميّةٌ من نوع `location` تُشغَّل والتطبيقُ مرئيّ** —
     * **فالموقعُ يبقى بلا إذن الخلفيّة.** **ويبقى الإذنُ مهمّاً
     * للتعافي بعد موت العمليّة** — **فيُقال ولا يُمنَع.**
     */
    @Test
    fun `الخلفيّةُ الناقصةُ لا تمنع درجةً`() {
        val s = full(background = false)
        assertTrue(Readiness.allows(s, Readiness.Level.CAN_GO_ONLINE))
        assertTrue(Readiness.allows(s, Readiness.Level.CAN_RUN_ACTIVE_DELIVERY))
        // **ويُقال** — **فالحالُ ليست كاملة.**
        assertFalse(s.ready)
    }

    /** **DRF-07 · والكاملُ يُسمَح له بكلّ درجة.** */
    @Test
    fun `الكاملُ يُسمَح له بالدرجات كلِّها`() {
        val s = full()
        for (lvl in Readiness.Level.values()) {
            assertTrue(lvl.name, Readiness.allows(s, lvl))
        }
    }

    /**
     * **DRF-08 · والإعلانُ أشدُّ من الرحلة — لا العكس.**
     *
     * **وحالٌ تسمح بالإعلان ولا تسمح بمتابعة رحلةٍ محضُ تناقض** —
     * **ومن أرسلنا إليه عملاً ثمّ منعناه من إتمامه أسوأُ من ألّا
     * نرسل.**
     */
    @Test
    fun `الإعلانُ لا يُسمَح حيث تُمنَع الرحلة`() {
        val cases = listOf(
            full(permission = false),
            full(service = false),
            full(background = false),
            full(notifications = false),
            full(permission = false, notifications = false),
            full(service = false, background = false, notifications = false),
            full(),
        )
        for (s in cases) {
            if (Readiness.allows(s, Readiness.Level.CAN_GO_ONLINE)) {
                assertTrue(
                    "**أُعلن متاحاً وهو ممنوعٌ من إتمام رحلة** — " + s.blockers.toString(),
                    Readiness.allows(s, Readiness.Level.CAN_RUN_ACTIVE_DELIVERY),
                )
            }
        }
    }

    /**
     * **DRF-09 · و`canWork` هي درجةُ الرحلة نفسُها.**
     *
     * **ومعنىً واحدٌ في موضعين يفترق يوماً** — **فمن غيّر أحدَهما
     * وحدَه أسقط هذا.**
     */
    @Test
    fun `canWork هي درجةُ الرحلة`() {
        val cases = listOf(
            full(), full(permission = false), full(service = false),
            full(background = false), full(notifications = false),
        )
        for (s in cases) {
            assertEquals(
                s.blockers.toString(),
                Readiness.canWork(s),
                Readiness.allows(s, Readiness.Level.CAN_RUN_ACTIVE_DELIVERY),
            )
        }
    }
}
