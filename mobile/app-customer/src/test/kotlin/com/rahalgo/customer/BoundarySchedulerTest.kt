package com.rahalgo.customer

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مُوقِّتُ الحدّ — حسابُ أقربِ حدّ** (Batch 5، C)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **يقيس المنطقَ الخالصَ**: أيُّ حدٍّ يُختار (مفتوح⇒إغلاق، مغلق⇒فتح)،
 * وأقربُ حدٍّ في المستقبل من بين المنصّةِ والمنطقة. **والمؤقّتُ نفسُه
 * (النوم/الساعةُ الرتيبة) يُشهَد على الجهاز** (FINAL-TIME).
 */
class BoundarySchedulerTest {

    // **مفتوحٌ ⇒ الحدُّ إغلاقُه (`next_close_at`).**
    @Test fun openUsesNextClose() {
        val ms = BoundaryScheduler.boundaryOf(
            available = true,
            nextClose = "2026-09-13T17:00:00+03:00",
            nextOpen = "2026-09-14T09:00:00+03:00",
        )
        assertEquals(BoundaryScheduler.parseIso("2026-09-13T17:00:00+03:00"), ms)
    }

    // **مغلقٌ ⇒ الحدُّ فتحُه (`next_available_at`).**
    @Test fun closedUsesNextOpen() {
        val ms = BoundaryScheduler.boundaryOf(
            available = false,
            nextClose = "",
            nextOpen = "2026-09-14T09:00:00+03:00",
        )
        assertEquals(BoundaryScheduler.parseIso("2026-09-14T09:00:00+03:00"), ms)
    }

    // **بلا حدٍّ معلوم ⇒ صفر** (مفتوحٌ بلا حدّ، أو نصٌّ فارغ/فاسد).
    @Test fun blankOrBadGivesZero() {
        assertEquals(0L, BoundaryScheduler.boundaryOf(true, "", ""))
        assertEquals(0L, BoundaryScheduler.boundaryOf(true, "not-a-time", ""))
        assertNull(BoundaryScheduler.parseIso(""))
        assertNull(BoundaryScheduler.parseIso("garbage"))
    }

    // **أقربُ حدٍّ في المستقبل** — يُختار الأصغرُ ممّا هو بعدَ المرجع.
    @Test fun earliestFuturePicksSoonest() {
        val now = 1000L
        // المنصّةُ ٥٠٠٠، المنطقةُ ٣٠٠٠ ⇒ الأقربُ ٣٠٠٠.
        assertEquals(3000L, BoundaryScheduler.earliestFutureBoundary(5000L, 3000L, now))
        // المنطقةُ صفرٌ (لا حدّ) ⇒ يُختار حدُّ المنصّة.
        assertEquals(5000L, BoundaryScheduler.earliestFutureBoundary(5000L, 0L, now))
    }

    // **ولا حدٌّ في الماضي** — حدٌّ مضى يُتجاهَل، ولا شيءَ ⇒ null.
    @Test fun pastBoundariesIgnored() {
        val now = 4000L
        // المنصّةُ ٢٠٠٠ (مضى)، المنطقةُ ٦٠٠٠ (قادم) ⇒ ٦٠٠٠.
        assertEquals(6000L, BoundaryScheduler.earliestFutureBoundary(2000L, 6000L, now))
        // كلاهما مضى ⇒ لا تسليح.
        assertNull(BoundaryScheduler.earliestFutureBoundary(2000L, 3000L, now))
        // لا حدَّ لأيٍّ (صفر) ⇒ null.
        assertNull(BoundaryScheduler.earliestFutureBoundary(0L, 0L, now))
    }
}
