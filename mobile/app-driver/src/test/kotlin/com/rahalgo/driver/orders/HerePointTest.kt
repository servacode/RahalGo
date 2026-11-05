package com.rahalgo.driver.orders

import com.rahalgo.ui.LastPoint
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * **«أضف معلماً» من مكانه الآن لا من مكان آخر تسليم** (قرارُ المالك ٢٠٢٦-١١-٠٥) — انظر
 * `pickHere`. **كانت قراءةُ الملاحة تُؤخذ وهي مطفأة** فيُحفظ المعلمُ عند آخر زبون.
 */
class HerePointTest {

    private val delivery = 35.9500 to 39.0100
    private val now = 1_000_000L

    private fun last(atMs: Long) = LastPoint.Point(35.9600, 39.0200, atMs = atMs)

    @Test
    fun navigationFixOnlyWhileFollowing() {
        assertEquals(delivery, pickHere(following = true, nav = delivery, last = last(now - 5_000), nowElapsedMs = now))
        assertEquals(
            "الملاحةُ مطفأةٌ بين طلبين — وقراءتُها مكانُ آخر تسليم",
            35.9600 to 39.0200,
            pickHere(following = false, nav = delivery, last = last(now - 5_000), nowElapsedMs = now),
        )
    }

    @Test
    fun staleOrUntimedShiftPointAsksForAFreshOne() {
        assertNull(pickHere(false, delivery, last(now - HERE_MAX_AGE_MS - 1), now))
        assertNull("موضعٌ بلا وقتٍ لا يُعرف عمرُه", pickHere(false, delivery, last(0L), now))
        assertNull("موضعٌ من المستقبل ساعةٌ أُعيد ضبطُها", pickHere(false, delivery, last(now + 1), now))
        assertNull(pickHere(false, null, null, now))
        assertNull("الملاحةُ شغّالةٌ بلا قراءةٍ بعد", pickHere(true, null, last(now - HERE_MAX_AGE_MS - 1), now))
    }
}
