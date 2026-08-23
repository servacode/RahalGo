package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **صياغةُ الأرقام — البندان ٨ و٩**
 * ══════════════════════════════════════════════════════════════════
 *
 * **أمرُ المالك نصّاً**: «ولا تنشئ `+-3` أو `- -` في حالات القيم
 * السالبة».
 *
 * **والخطرُ ليس نظريّاً**: `"+" + format(-3)` تعطي `+-3`، **وقيمةٌ
 * سالبةٌ تُنسَّق بعلامتها ثمّ تُسبَق بأخرى.**
 */
class RouteMetricTextTest {

    // ══════════════════════════════════════════════════════════════
    // **ولا علامةَ في المقدار**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `المقدارُ بلا علامةٍ مهما كان الفرق`() {
        for (v in listOf(-5000.0, -150.0, 150.0, 5000.0)) {
            val d = RouteMetricText.distanceDelta(v)!!
            assertFalse("«${d.magnitude}» فيها علامة", d.magnitude.contains('-'))
            assertFalse("«${d.magnitude}» فيها علامة", d.magnitude.contains('+'))
        }
        for (v in listOf(-600.0, -60.0, 60.0, 600.0)) {
            val d = RouteMetricText.durationDelta(v)!!
            assertFalse("«${d.magnitude}» فيها علامة", d.magnitude.contains('-'))
            assertFalse("«${d.magnitude}» فيها علامة", d.magnitude.contains('+'))
        }
    }

    @Test
    fun `والنوعُ يحمل الاتّجاه`() {
        assertEquals(
            RouteMetricText.Delta.Kind.SHORTER,
            RouteMetricText.distanceDelta(-4500.0)!!.kind,
        )
        assertEquals(
            RouteMetricText.Delta.Kind.LONGER,
            RouteMetricText.distanceDelta(4500.0)!!.kind,
        )
        assertEquals(
            RouteMetricText.Delta.Kind.FASTER,
            RouteMetricText.durationDelta(-420.0)!!.kind,
        )
        assertEquals(
            RouteMetricText.Delta.Kind.SLOWER,
            RouteMetricText.durationDelta(420.0)!!.kind,
        )
    }

    // ══════════════════════════════════════════════════════════════
    // **والصغيرُ لا يُعرض**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `فرقٌ تافهٌ لا يُعرض`() {
        // **«أقصرُ بـ٢٠ م» ليست معلومةً لسائق.**
        assertNull(RouteMetricText.distanceDelta(20.0))
        assertNull(RouteMetricText.distanceDelta(-99.0))
        assertNull(RouteMetricText.durationDelta(5.0))
        assertNull(RouteMetricText.durationDelta(-29.0))
    }

    @Test
    fun `وبديلٌ متقاربُ الأرقام لا فروقَ له`() {
        val option = RouteOption(
            routeId = "a",
            route = RouteFixtures.straight(),
            engineDurationS = 900.0,
            distanceM = 11000.0,
            deltaDistanceM = 40.0,
            deltaDurationS = 8.0,
        )
        assertTrue(RouteMetricText.deltasOf(option).isEmpty())
    }

    // ══════════════════════════════════════════════════════════════
    // **الصياغة**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `المسافةُ بالمتر دونَ الكيلومتر`() {
        assertEquals("850 م", RouteMetricText.distance(850.0))
        assertEquals("120 م", RouteMetricText.distance(-123.0))
    }

    @Test
    fun `وبمنزلةٍ فوق الكيلومتر`() {
        assertEquals("4٫5 كم", RouteMetricText.distance(4500.0))
        assertEquals("11 كم", RouteMetricText.distance(11020.0))
    }

    @Test
    fun `ولا صفرَ عالق`() {
        // **«٤٫٠ كم» تصير «٤ كم».**
        assertEquals("4 كم", RouteMetricText.distance(4000.0))
        assertFalse(RouteMetricText.distance(4000.0).contains('٫'))
    }

    @Test
    fun `والمسافةُ الكبيرةُ بلا كسر`() {
        assertEquals("37 كم", RouteMetricText.distance(36997.0))
        assertEquals("432 كم", RouteMetricText.distance(432396.0))
    }

    @Test
    fun `المدّةُ بالدقيقة`() {
        assertEquals("15 د", RouteMetricText.duration(900.0))
        assertEquals("12 د", RouteMetricText.duration(-732.0))
    }

    @Test
    fun `وبالساعة فوق الستّين`() {
        assertEquals("1 س", RouteMetricText.duration(3600.0))
        assertEquals("1 س 15 د", RouteMetricText.duration(4500.0))
        assertEquals("5 س 32 د", RouteMetricText.duration(19921.0))
    }

    // ══════════════════════════════════════════════════════════════
    // **والمقاييسُ من المحرّك لا من العرض** — البند ٧
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `المقاييسُ من مدّة المحرّك`() {
        val option = RouteOption(
            routeId = "rec",
            route = RouteFixtures.straight(),
            engineDurationS = 900.0,
            distanceM = 11000.0,
        )
        val m = RouteMetricText.metricsOf(option)
        assertEquals("15 د", m.duration)
        assertEquals("11 كم", m.distance)
    }

    @Test
    fun `والحالةُ المقيسةُ الحقيقيّة`() {
        // **البند ٣٥** — بديلٌ أقلُّ زمناً وأقصرُ من الموصى به.
        // **مقيسٌ فعلاً**: الأساس ٣٩٨٫٣كم/٣٠٦٫٤د ← البديل ٣٦١٫٣/٢٩٩٫١.
        val option = RouteOption(
            routeId = "a",
            route = RouteFixtures.straight(),
            engineDurationS = 17946.0,
            distanceM = 361340.0,
            deltaDistanceM = -36997.0,
            deltaDurationS = -438.0,
        )
        val deltas = RouteMetricText.deltasOf(option)
        assertEquals(2, deltas.size)
        assertEquals(RouteMetricText.Delta.Kind.SHORTER, deltas[0].kind)
        assertEquals("37 كم", deltas[0].magnitude)
        assertEquals(RouteMetricText.Delta.Kind.FASTER, deltas[1].kind)
        assertEquals("7 د", deltas[1].magnitude)
    }

    @Test
    fun `والمسافةُ أوّلاً ثمّ الزمن`() {
        // **٢٩ من ٣١ بديلاً غيرِ مهيمَنٍ عليه «أقصرُ وأبطأ»** —
        // فالمسافةُ هي المقايضةُ الغالبة.
        val option = RouteOption(
            routeId = "a",
            route = RouteFixtures.straight(),
            engineDurationS = 1620.0,
            distanceM = 6500.0,
            deltaDistanceM = -4500.0,
            deltaDurationS = 720.0,
        )
        val deltas = RouteMetricText.deltasOf(option)
        assertEquals(RouteMetricText.Delta.Kind.SHORTER, deltas[0].kind)
        assertEquals(RouteMetricText.Delta.Kind.SLOWER, deltas[1].kind)
    }
}
