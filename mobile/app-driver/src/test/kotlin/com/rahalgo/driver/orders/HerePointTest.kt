package com.rahalgo.driver.orders

import com.rahalgo.ui.LastPoint
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
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

    /**
     * **المعلمُ من قراءةٍ تُقاس الآن قبل آخر موضعٍ وصل** (مراجعة ٢٠٢٦-١١-٠٥) — `atMs` وقتُ الوصول،
     * والقراءاتُ تُجمَّع حتّى دقيقتين: **موضعٌ «عمرُه» ٩٠ ثانيةً قد يكون قبل الجامع بمئات الأمتار.**
     */
    @Test
    fun landmarkPrefersAFixMeasuredNow() {
        val atMosque = 35.9700 to 39.0300
        assertEquals(
            "قراءةُ الآن تغلب آخرَ موضعٍ وصل قبل ٩٠ ثانية",
            atMosque,
            pickLandmarkHere(following = false, nav = delivery, fresh = atMosque, last = last(now - 90_000), nowElapsedMs = now),
        )
        assertEquals(
            "والملاحةُ الشغّالةُ تغلب — قراءتُها كلَّ ثانية",
            delivery,
            pickLandmarkHere(following = true, nav = delivery, fresh = atMosque, last = last(now - 5_000), nowElapsedMs = now),
        )
        assertEquals(
            "لا إشارةَ في عشر ثوانٍ ⇒ آخرُ موضعٍ حديث",
            35.9600 to 39.0200,
            pickLandmarkHere(following = false, nav = delivery, fresh = null, last = last(now - 30_000), nowElapsedMs = now),
        )
        assertNull(
            "ولا قراءةُ ملاحةٍ مطفأة",
            pickLandmarkHere(following = false, nav = delivery, fresh = null, last = last(now - HERE_MAX_AGE_MS - 1), nowElapsedMs = now),
        )
    }

    /** **عمرُ القراءة من لحظة قياسها** — مخبوءٌ قديمٌ من `getCurrentLocation` لا يُقبل. */
    @Test
    fun freshFixAgeIsFromMeasurement() {
        val nowNs = 5_000_000_000_000L
        assertTrue(measuredJustNow(nowNs - 3_000_000_000L, nowNs))
        assertFalse("عمرُها ١١ ثانية", measuredJustNow(nowNs - 11_000_000_000L, nowNs))
        assertFalse("بلا وقت", measuredJustNow(0L, nowNs))
        assertFalse("من المستقبل", measuredJustNow(nowNs + 1, nowNs))
    }
}
