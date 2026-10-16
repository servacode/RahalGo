package com.rahalgo.customer.live

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * **لا يُرسَل إلّا الدقيق — وأحسنُ ما وُجد بعد ثلاثين ثانية** (قرارُ المالك ٢٠٢٦-١٠-١٦).
 */
class LiveSendFilterTest {

    private fun r(atSec: Long, acc: Double?, lat: Double = 35.95) =
        LiveSendFilter.Reading(lat, 39.0, acc, atSec * 1000)

    @Test
    fun `a precise reading is sent at once`() {
        val f = LiveSendFilter()
        assertEquals(r(0, 8.0), f.offer(r(0, 8.0)))
        // **والحدُّ نفسُه دقيق.**
        assertEquals(r(10, 30.0), f.offer(r(10, 30.0)))
    }

    @Test
    fun `imprecise readings are held then the best of the window is sent after 30 s`() {
        val f = LiveSendFilter()
        assertNull(f.offer(r(0, 60.0, lat = 1.0)))
        assertNull(f.offer(r(10, 42.0, lat = 2.0)))
        assertNull(f.offer(r(20, 80.0, lat = 3.0)))
        // **ثلاثون ثانيةً بلا دقيق** — يُرسَل أحسنُها (٤٢م) لا آخرُها.
        assertEquals(r(10, 42.0, lat = 2.0), f.offer(r(30, 55.0, lat = 4.0)))
        // **ونافذةٌ جديدةٌ تبدأ من الإرسال** — لا يُرسَل التالي فوراً.
        assertNull(f.offer(r(40, 50.0)))
        assertNull(f.offer(r(50, 50.0)))
        assertEquals(r(60, 45.0), f.offer(r(60, 45.0)))
    }

    @Test
    fun `unknown accuracy is never precise and ranks last`() {
        val f = LiveSendFilter()
        assertNull(f.offer(r(0, null, lat = 1.0)))
        assertNull(f.offer(r(10, 90.0, lat = 2.0)))
        assertEquals(r(10, 90.0, lat = 2.0), f.offer(r(30, null, lat = 3.0)))
    }

    @Test
    fun `the threshold follows the server`() {
        val f = LiveSendFilter()
        assertNull(f.offer(r(0, 40.0)))
        f.maxAccuracyM = 50.0
        assertEquals(r(10, 40.0), f.offer(r(10, 40.0)))
    }

    @Test
    fun `no faster than every eight seconds and an early precise reading waits only for the gap`() {
        val f = LiveSendFilter()
        assertEquals(r(0, 5.0), f.offer(r(0, 5.0)))
        assertNull(f.offer(r(4, 6.0, lat = 9.0)))
        // **بعد الفجوة** — التقريبيّةُ الجديدةُ لا تُرسَل، **والدقيقةُ المحبوسةُ تُرسَل.**
        assertEquals(r(4, 6.0, lat = 9.0), f.offer(r(9, 70.0)))
    }

    @Test
    fun `reset forgets the window`() {
        val f = LiveSendFilter()
        assertNull(f.offer(r(0, 60.0)))
        f.reset()
        assertNull(f.offer(r(30, 70.0)))
        assertEquals(r(60, 65.0), f.offer(r(60, 65.0)))
    }
}
