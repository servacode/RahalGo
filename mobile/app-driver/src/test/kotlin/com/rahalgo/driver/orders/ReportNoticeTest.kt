package com.rahalgo.driver.orders

import com.rahalgo.shared.model.DriverOrder
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **«وصل بلاغك» لا يلحق السائقَ إلى المتجر الجديد** (تجربةُ القبول ٢٠٢٦-١٠-٠٣) — انظر
 * `ReportNotice`. **كان المفتاحُ `id/status`** فيطابق «وصلت المتجر» الجديدَ مفتاحَ القديم.
 */
class ReportNoticeTest {

    private val atOld = DriverOrder(id = "o1", status = "at_pickup", merchantName = "الأوّل")

    @Test
    fun arrivingAtTheNewStoreIsANewKey() {
        val atNew = atOld.copy(merchantName = "البديل")
        assertNotEquals(
            "الوصولُ إلى المتجر الجديد طابق مفتاحَ بلاغ القديم — فخُبّئ «استلمت الطلب»",
            ReportNotice.key(atOld), ReportNotice.key(atNew),
        )
        assertTrue(ReportNotice.stale(ReportNotice.key(atOld), listOf(atNew)))
    }

    @Test
    fun goingBackOnTheRoadForgetsTheNotice() {
        val back = atOld.copy(status = "assigned")
        assertTrue("رجوعُ الطلب إلى الطريق لم يمحُ الخبر", ReportNotice.stale(ReportNotice.key(atOld), listOf(back)))
    }

    @Test
    fun sameStoreSameStageKeepsTheNotice() {
        assertFalse(ReportNotice.stale(ReportNotice.key(atOld), listOf(atOld)))
        assertEquals(false, ReportNotice.stale("", listOf(atOld)))
    }

    @Test
    fun orderGoneForgetsTheNotice() {
        assertTrue(ReportNotice.stale(ReportNotice.key(atOld), emptyList()))
    }
}
