package com.rahalgo.customer

import com.rahalgo.shared.model.Address
import com.rahalgo.shared.model.Availability
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **موقعٌ يستكشف وموقعٌ يُوصَّل إليه** (`DL`، ٢٠٢٦-٠٩-١٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ومن فتح التطبيقَ في دمشقَ ليطلب إلى بيت أهله في الرقّة** — **فلو
 * صار موقعُه عنوانَ توصيلِه لَذهب الطلبُ إلى حيث يقف لا إلى حيث أراد**،
 * **ولا يُكتشَف إلّا بعد أن يخرج السائق.**
 */
class DiscoveryPolicyTest {

    private val raqqa = Address(id = "a1", text = "الرقة", lat = 35.9506, lng = 39.0094)
    private val damascusPoint = Discovery(lat = 33.5138, lng = 36.2765, accuracyM = 20f)

    // ═════════════════ DL-01 · DL-02 ═════════════════

    /** **DL-01 · DL-02 · ولا عنوانَ مؤكَّدٌ ⇒ يُسأل عن موقع الجهاز.** */
    @Test
    fun `بلا عنوانٍ تُقرأ نقطةُ الاستكشاف`() {
        val p = contextPoint(delivery = null, discovery = damascusPoint)
        assertEquals(PointSource.DISCOVERY, p.source)
        assertEquals(33.5138, p.lat, 1e-9)
    }

    // ═════════════════ DL-03 · DL-04 ═════════════════

    /**
     * **DL-03 · DL-04 · والاستكشافُ لا يفتح الزرَّ.**
     *
     * **ولو قال «الخدمةُ متوفّرة»** — **نقطةٌ تُخبِر لا تُنشئ طلبا.**
     */
    @Test
    fun `الاستكشافُ لا يصير عنوانَ توصيلٍ ولو كان متاحاً`() {
        val p = contextPoint(delivery = null, discovery = damascusPoint)
        assertEquals(
            "**نقطةُ استكشافٍ فتحت الإضافة** — **فيُوصَّل إلى حيث يقف لا حيث أراد**",
            AddAction.NEED_ADDRESS,
            addActionFor(p, Availability(available = true)),
        )
    }

    // ═════════════════ DL-06 · DL-07 ═════════════════

    /**
     * **DL-06 · DL-07 · والمؤكَّدةُ تسبق دائماً.**
     *
     * **وهاتفُه في دمشقَ وعنوانُه الرقّة** — **فالرقّةُ تحكم.**
     */
    @Test
    fun `العنوانُ المختارُ يغلب موقعَ الجهاز`() {
        val p = contextPoint(delivery = raqqa, discovery = damascusPoint)
        assertEquals(PointSource.DELIVERY, p.source)
        assertEquals("**موقعُ الجهاز حكم بدل العنوان المختار**", 35.9506, p.lat, 1e-9)
        assertEquals(AddAction.ADD, addActionFor(p, Availability(available = true)))
    }

    // ═════════════════ DL-08 ═════════════════

    /** **DL-08 · وحركةُ الجهاز بعد التأكيد لا تبدّل العنوان.** */
    @Test
    fun `تحرّكُ الجهاز لا يبدّل عنوانَ التوصيل`() {
        val moved = Discovery(lat = 36.2021, lng = 37.1343, accuracyM = 5f)
        val p = contextPoint(delivery = raqqa, discovery = moved)
        assertEquals(PointSource.DELIVERY, p.source)
        assertEquals("**الجهازُ تحرّك فبدّل عنوانَ التوصيل**", 35.9506, p.lat, 1e-9)
    }

    // ═════════════════ DL-09 · DL-10 · DL-12 ═════════════════

    /**
     * **DL-09 · DL-10 · DL-12 · ولا موقعَ ⇒ لا حكمَ ولا منعَ تصفّح.**
     *
     * **ومن رُفض إذنُه أو أُطفئ موقعُه يتصفّح** — **ويُسأل عن عنوانه
     * حين يضيف.**
     */
    @Test
    fun `بلا موقعٍ ولا عنوانٍ لا يُدّعى شيء`() {
        val p = contextPoint(delivery = null, discovery = null)
        assertEquals(PointSource.NONE, p.source)
        assertEquals(AddAction.NEED_ADDRESS, addActionFor(p, null))
    }

    // ═════════════════ DL-11 ═════════════════

    /**
     * **DL-11 · ونقطةٌ لا تُعرَف دقّتُها لا يُحكَم بها.**
     *
     * **وحدُّ المنطقة يُقاس بمئات الأمتار** — **ونقطةٌ أخطأت كيلومترين
     * تقول «أنت خارجَ النطاق» لمن هو في قلبه.**
     */
    @Test
    fun `الدقّةُ الضعيفةُ لا تُنتج حكماً`() {
        assertFalse(Discovery(35.95, 39.00, accuracyM = 2000f).usable())
        assertFalse("**نقطةٌ بلا دقّةٍ حُكم بها**", Discovery(35.95, 39.00).usable())
        assertTrue(Discovery(35.95, 39.00, accuracyM = 30f).usable())

        val vague = contextPoint(delivery = null, discovery = Discovery(35.95, 39.00, 2000f))
        assertEquals(
            "**نقطةٌ غامضةٌ صارت مصدرَ حكم**",
            PointSource.NONE,
            vague.source,
        )
    }

    /** **وحدُّ القبول يُقاس لا يُظَنّ.** */
    @Test
    fun `حدُّ الدقّة عند خمس مئةِ متر`() {
        assertTrue(Discovery(1.0, 1.0, DISCOVERY_MAX_ACCURACY_M).usable())
        assertFalse(Discovery(1.0, 1.0, DISCOVERY_MAX_ACCURACY_M + 1f).usable())
    }

    // ═════════════════ DL-05 ═════════════════

    /**
     * **DL-05 · و«استخدم موقعي الحالي» تصنع عنواناً مؤكَّداً.**
     *
     * **وتُقرأ الإتاحةُ من جديدٍ عليه** — **ونتيجةُ استكشافٍ قديمةٌ
     * ليست حقيقةَ طلب.** **والمفتاحُ يتبدّل فتُطلَب قراءةٌ جديدة.**
     */
    @Test
    fun `تأكيدُ الموقع الحاليِّ يُنشئ نقطةَ توصيلٍ تُقرأ من جديد`() {
        Orderable.resetForTest()
        val discovered = Orderable.pointKey(damascusPoint.lat, damascusPoint.lng)
        // **حالُ استكشافٍ قديمةٌ محفوظة.**
        Orderable.put(Availability(available = true), discovered, 1_000L)

        // **ثمّ يُؤكَّد الموقعُ عنواناً** — **والنقطةُ هي هي.**
        val confirmed = Address(id = "b1", text = "موقعي", lat = damascusPoint.lat, lng = damascusPoint.lng)
        val p = contextPoint(delivery = confirmed, discovery = damascusPoint)
        assertEquals(PointSource.DELIVERY, p.source)

        // **ولا تُقرأ حالُ الاستكشاف حقيقةً للطلب** — **بل تُقرأ
        // الإتاحةُ بسلطة التوصيل**: **والحالُ تشيخ فتُجدَّد.**
        assertTrue(
            "**حالُ استكشافٍ صارت حقيقةَ طلبٍ بلا قراءةٍ جديدة**",
            Orderable.stale(Orderable.pointKey(p.lat, p.lng), 1_000L + Orderable.FRESH_MS),
        )
        Orderable.resetForTest()
    }
}
