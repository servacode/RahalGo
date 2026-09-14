package com.rahalgo.customer

import com.rahalgo.shared.model.Availability
import com.rahalgo.ui.ServiceReason
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الإتاحةُ قبل السلّة — سياسةُ الزرّ** (`PC`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والحكمُ في المحرّك ويُقاس هناك** (`internal/qa/precart_test.go`) —
 * **وهذه تقيس ما في الجهاز**: **أنّ الزرَّ لا يبدو صالحاً لعنوانٍ لا
 * يُخدَم**، **وأنّ حالَ عنوانٍ لا تُقرأ على عنوانٍ آخر.**
 */
class PreCartPolicyTest {

    @Before fun clean() = Orderable.resetForTest()

    @After fun after() = Orderable.resetForTest()

    private fun blocked(reason: String) = Availability(available = false, reason = reason)

    // ═════════════════ PC-01 · PC-02 ═════════════════

    /** **PC-01 · عنوانٌ مخدومٌ يُضاف إليه.** */
    @Test
    fun `المخدومُ يُضاف`() {
        assertEquals(
            AddAction.ADD,
            addAction(hasAddress = true, av = Availability(available = true)),
        )
    }

    /**
     * **PC-02 · ولا عنوانَ لا إضافة** — **يُسأل أوّلاً.**
     *
     * **ولا يُؤخذ موقعُ الجهاز عنوانَ توصيل** — **من تسوّق من عمله
     * يُوصَّل إلى بيته.**
     */
    @Test
    fun `بلا عنوانٍ يُسأل لا يُضاف`() {
        assertEquals(AddAction.NEED_ADDRESS, addAction(hasAddress = false, av = null))
        // **وحتّى لو كانت الحالُ متاحةً** — **الحالُ لعنوانٍ ليست عنوانا.**
        assertEquals(
            AddAction.NEED_ADDRESS,
            addAction(hasAddress = false, av = Availability(available = true)),
        )
    }

    // ═════════════════ PC-03 … PC-08 ═════════════════

    /**
     * **وكلُّ سببٍ يمنع الإضافة** — **زمنيّاً كان أو جغرافيّاً.**
     *
     * **وزرٌّ يبدو صالحاً ثمّ يُردّ أسوأُ من زرٍّ مُعطَّلٍ بسببٍ مكتوب.**
     */
    @Test
    fun `كلُّ منعٍ يُوقف الإضافة قبل السلّة`() {
        for (
            r in listOf(
                ServiceReason.CITY_NOT_SUPPORTED,
                ServiceReason.PROVINCE_NOT_SUPPORTED,
                ServiceReason.AREA_NOT_SUPPORTED,
                ServiceReason.ADDRESS_OUTSIDE_COVERAGE,
                ServiceReason.COVERAGE_UNAVAILABLE,
                ServiceReason.LAUNCH_CLOSED,
                ServiceReason.TEMPORARILY_UNAVAILABLE,
                ServiceReason.PLATFORM_CLOSED_NOW,
                ServiceReason.ZONE_CLOSED_NOW,
                ServiceReason.MERCHANT_CLOSED_NOW,
            )
        ) {
            assertEquals(
                "**سببٌ مانعٌ لم يُوقف الإضافة**: $r",
                AddAction.BLOCKED,
                addAction(hasAddress = true, av = blocked(r)),
            )
        }
    }

    /**
     * **وقبل أوّل قراءةٍ لا يُمنَع أحد.**
     *
     * **ومنعٌ بلا علمٍ أسوأُ من ردٍّ بعلم** — **والمحرّكُ يردّ عند
     * الإنشاء على كلّ حال.**
     */
    @Test
    fun `الغيابُ لا يمنع`() {
        assertEquals(AddAction.ADD, addAction(hasAddress = true, av = null))
    }

    // ═════════════════ PC-09 · PC-10 — تبديلُ العنوان ═════════════════

    /**
     * **PC-09 · وحالُ عنوانٍ لا تُقرأ على عنوانٍ آخر.**
     *
     * **ومن بدّل إلى عنوانٍ خارجَ النطاق ورأى الزرَّ صالحاً قرأ كذباً.**
     */
    @Test
    fun `حالُ عنوانٍ لا تُقرأ على غيره`() {
        val a = Orderable.pointKey(35.95, 39.00)
        val b = Orderable.pointKey(33.51, 36.27)
        Orderable.put(Availability(available = true), a, 1_000L)

        assertEquals(AddAction.ADD, addAction(true, Orderable.of(a)))
        // **والعنوانُ الثاني لا حالَ له** — **فلا يرث «متاح».**
        assertEquals(null, Orderable.of(b))
        assertTrue("**عنوانٌ جديدٌ لم يُطلَب له نداء**", Orderable.stale(b, 1_000L))
    }

    /** **PC-10 · والعودةُ إلى الصالح تُصلح الحالَ بعد قراءةٍ جديدة.** */
    @Test
    fun `العودةُ إلى عنوانٍ صالحٍ تُعيد الإضافة`() {
        val good = Orderable.pointKey(35.95, 39.00)
        val bad = Orderable.pointKey(33.51, 36.27)

        Orderable.put(blocked(ServiceReason.CITY_NOT_SUPPORTED), bad, 1_000L)
        assertEquals(AddAction.BLOCKED, addAction(true, Orderable.of(bad)))

        Orderable.put(Availability(available = true), good, 2_000L)
        assertEquals(AddAction.ADD, addAction(true, Orderable.of(good)))
        // **وما كان للعنوان الأوّل لم يبقَ.**
        assertEquals(null, Orderable.of(bad))
    }

    /** **وتبديلُ العنوان يُبطل ما قبله** — `invalidate`. */
    @Test
    fun `الإبطالُ ينسى كلَّ حال`() {
        val p = Orderable.pointKey(35.95, 39.00)
        Orderable.put(Availability(available = true), p, 1_000L)
        Orderable.invalidate()
        assertEquals(null, Orderable.of(p))
        assertEquals("", Orderable.forPoint)
    }

    /** **ولا تُسأل في كلّ رسمة** — **والحالُ الطازجةُ تكفي.** */
    @Test
    fun `الطازجةُ لا تُعاد`() {
        val p = Orderable.pointKey(35.95, 39.00)
        Orderable.put(Availability(available = true), p, 1_000L)
        assertFalse("**نداءٌ في كلّ رسمة**", Orderable.stale(p, 1_000L + 1))
        assertTrue("**حالٌ شاخت ولم تُجدَّد**", Orderable.stale(p, 1_000L + Orderable.FRESH_MS))
    }

    // ═════════════════ PC-11 · PC-12 · PC-13 — سياسةُ الزرّ ═════════════

    /**
     * **PC-11 · PC-12 · وزرُّ النيّة بالسياسة المركزيّة** (الدفعةُ
     * الرابعة).
     *
     * **PC-13 · ومن مُنع لسببٍ زمنيٍّ لا يُدعى إلى طلب منطقته** —
     * **ودعوةٌ إلى «أخبرني حين نصل» لمن نصله كلَّ صباحٍ كذب.**
     */
    @Test
    fun `زرُّ النيّة للجغرافيا وحدَها`() {
        assertEquals(
            ServiceReason.KIND_COVERAGE,
            ServiceReason.ctaKind(ServiceReason.ADDRESS_OUTSIDE_COVERAGE),
        )
        assertEquals(
            ServiceReason.KIND_INTEREST,
            ServiceReason.ctaKind(ServiceReason.CITY_NOT_SUPPORTED),
        )
        for (
            r in listOf(
                ServiceReason.PLATFORM_CLOSED_NOW,
                ServiceReason.ZONE_CLOSED_NOW,
                ServiceReason.MERCHANT_CLOSED_NOW,
                ServiceReason.TEMPORARILY_UNAVAILABLE,
                ServiceReason.COVERAGE_UNAVAILABLE,
                ServiceReason.LAUNCH_CLOSED,
                ServiceReason.INVALID_LOCATION,
            )
        ) {
            assertEquals("**سببٌ زمنيٌّ عرض زرَّ توسّع**: $r", "", ServiceReason.ctaKind(r))
        }
    }
}
