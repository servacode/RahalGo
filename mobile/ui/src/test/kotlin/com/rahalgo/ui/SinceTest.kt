package com.rahalgo.ui

import java.time.Instant
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * **كم مضى — يُقاس ولا يُخمَّن.**
 *
 * (طلبُ المالك ٢٠٢٦-٠٩-٠٢: عمرُ الطلب على بطاقة المتجر.)
 */
class SinceTest {

    private val now = Instant.parse("2026-09-02T12:00:00Z")

    @Test
    fun `دقائقُ ومئاتُها تُقاس من الفارق`() {
        assertEquals(7L, Since.minutes("2026-09-02T11:53:00Z", now))
        assertEquals(180L, Since.minutes("2026-09-02T09:00:00Z", now))
    }

    /** **وما دون الدقيقة دقيقة** — «منذ ٠ د» لا تُكتب. */
    @Test
    fun `الطلبُ الآنَ عمرُه دقيقة`() {
        assertEquals(1L, Since.minutes("2026-09-02T11:59:40Z", now))
    }

    /**
     * **ووقتٌ في المستقبل دقيقة** — ساعةُ الجهاز قد تسبق ساعةَ
     * الخادم بثوانٍ، **ولا يُكتب «منذ ناقص ثلاث».**
     */
    @Test
    fun `المستقبلُ لا يُكتب سالباً`() {
        assertEquals(1L, Since.minutes("2026-09-02T12:03:00Z", now))
    }

    /** **وصيغةُ الإزاحة تُقرأ كذلك** — لا `Z` وحدَها. */
    @Test
    fun `الإزاحةُ الزمنيّة تُقرأ`() {
        assertEquals(60L, Since.minutes("2026-09-02T14:00:00+03:00", now))
    }

    /**
     * **وما لا يُقرأ يردّ فراغاً** — ومن اخترع قيمةً من نصٍّ فاسدٍ
     * كتب على الشاشة كذباً هادئاً.
     */
    @Test
    fun `نصٌّ فاسدٌ لا يُخمَّن`() {
        assertNull(Since.minutes("", now))
        assertNull(Since.minutes("لا وقت", now))
        assertNull(Since.minutes("2026-13-45", now))
    }
}
