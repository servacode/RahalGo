package com.rahalgo.driver.trip

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مسارُ الرحلة — الخطوةُ والفعلُ التالي والوصولُ التلقائيّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (فحصُ تطبيق السائق ٢٠٢٦-٠٨-٢٣، بأمر المالك.)
 *
 * # ولماذا كُتبت الآن
 *
 * **قِيس أنّ تطبيقَ السائق فيه ٢١ اختباراً في ثلاثة ملفّات، ووحدةَ
 * الملاحة ٣٣٥ اختباراً في ستّةٍ وعشرين.** والفجوةُ ليست في العدد بل
 * في الموضع: **`TripScreen` (٢٢٠٠ سطر) و`OrdersViewModel` (١٣٨٧)
 * بلا اختبارٍ واحد** — وفيهما منطقُ الخطوات وحالُ الرحلة.
 *
 * **وعطبُ اليوم كان فيهما بالضبط**: نداءُ المسار يسبق وصولَ الطلبات
 * فيخرج بلا نداء، **فيُرسم الخطُّ المستقيمُ من فوق البيوت** — ولم
 * يمسكه شيءٌ إلّا عينُ المالك في الشارع.
 *
 * # وما يُحرَس هنا
 *
 * **ثلاثةُ جداولَ تقود يدَ السائق**: أين هو من الرحلة، وما الزرُّ
 * الذي يراه، **ومتى يُسجَّل وصولُه بلا ضغطة.** والثالثُ أخطرُها:
 * **انتقالٌ لا رجعةَ فيه في جدول المحرّك.**
 */
class TripFlowTest {

    // ══════════════════════════════════════════════════════════════════
    // **١ · الخطوةُ تُقرأ من حال الطلب لا من رايةٍ في الجهاز**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `each engine status maps to its step`() {
        assertEquals(TripStep.TO_PICKUP, TripStep.of("assigned"))
        assertEquals(TripStep.AT_PICKUP, TripStep.of("at_pickup"))
        assertEquals(TripStep.PICKED_UP, TripStep.of("picked_up"))
        assertEquals(TripStep.TO_CUSTOMER, TripStep.of("on_the_way"))
        assertEquals(TripStep.AT_CUSTOMER, TripStep.of("at_dropoff"))
        assertEquals(TripStep.DELIVERED, TripStep.of("delivered"))
    }

    /**
     * **و`assigned` تُقرأ «في الطريق» لا «قبلت».**
     *
     * **القبولُ لحظةٌ مضت، وما يفعله الآن هو المشي** — ومن رأى
     * «قبلتَ الطلب» وهو يقود ظنّ أنّ عليه فعلاً لم يفعله بعد.
     */
    @Test
    fun `assigned reads as on the way to the merchant`() {
        assertEquals(TripStep.TO_PICKUP, TripStep.of("assigned"))
    }

    /**
     * **وحالٌ لا نعرفه لا يُسقط الشاشة.**
     *
     * **وحالٌ جديدٌ يُضاف في المحرّك يصل التطبيقاتِ القديمةَ قبل أن
     * تُحدَّث** — فيرتدّ إلى البداية ولا ينهار.
     */
    @Test
    fun `an unknown status falls back and does not crash`() {
        assertEquals(TripStep.ACCEPTED, TripStep.of("something_new"))
        assertEquals(TripStep.ACCEPTED, TripStep.of(""))
    }

    // ══════════════════════════════════════════════════════════════════
    // **٢ · الفعلُ التالي — وهو نصُّ الزرّ الذي تحت إبهامه**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `the ordinary order walks its steps in order`() {
        assertEquals("at_pickup", nextAction("assigned")?.status)
        assertEquals("picked_up", nextAction("at_pickup")?.status)
        assertEquals("on_the_way", nextAction("picked_up")?.status)
        assertEquals("at_dropoff", nextAction("on_the_way")?.status)
    }

    /**
     * **ولا «وصلتُ إلى المتجر» في الطلب الخاصّ.**
     *
     * (تصحيحُ المالك ٢٠٢٦-٠٨-٠٩: «ما في شيء اسمه وصلتُ للمتجر».)
     *
     * **ولا متجرَ يقف عنده** — ما يفعله محادثةٌ واتّفاقٌ ثمّ شراء.
     * **والمحرّكُ يفرضه في جدوله**: الخاصُّ يقفز من `assigned` إلى
     * `picked_up` رأساً.
     */
    @Test
    fun `a custom order skips arriving at the merchant`() {
        assertEquals("picked_up", nextAction("assigned", custom = true)?.status)
    }

    /** **والمسلَّمُ لا فعلَ بعده** — وزرٌّ في نهاية الرحلة يُضغط بلا معنى. */
    @Test
    fun `a delivered order has no next action`() {
        assertNull(nextAction("delivered"))
    }

    // ══════════════════════════════════════════════════════════════════
    // **٣ · الوصولُ التلقائيّ — وهو أخطرُها**
    // ══════════════════════════════════════════════════════════════════
    //
    // **انتقالٌ لا رجعةَ فيه**: جدولُ المحرّك لا يعرف
    // `at_pickup → assigned`. **فخطأٌ هنا يُسجَّل ولا يُردّ إلّا بيد
    // المكتب.**

    @Test
    fun `arrival auto-advances on both legs`() {
        assertEquals("at_pickup", autoArrivalTarget("assigned", custom = false))
        assertEquals("at_dropoff", autoArrivalTarget("on_the_way", custom = false))
    }

    /** **ولا وصولَ تلقائيٌّ في الطلب الخاصّ** — في طرفيه معاً. */
    @Test
    fun `a custom order never auto-arrives`() {
        assertNull(autoArrivalTarget("assigned", custom = true))
        assertNull(autoArrivalTarget("on_the_way", custom = true))
    }

    /**
     * **والحالُ يُطابَق تماماً لا يُقارَب.**
     *
     * **وما بين الطورين حالٌ بلغها بيده** (`at_pickup` · `picked_up`)
     * — **ولا يُعاد تسجيلُ وصولٍ سُجّل**، وإلّا رُدّ النداءُ بخطأٍ
     * يراه السائقُ ولا يفهمه.
     */
    @Test
    fun `states the driver already advanced are never re-fired`() {
        for (s in listOf("at_pickup", "picked_up", "at_dropoff", "delivered", "")) {
            assertNull("لا يُفجَّر عند: $s", autoArrivalTarget(s, custom = false))
        }
    }
}
