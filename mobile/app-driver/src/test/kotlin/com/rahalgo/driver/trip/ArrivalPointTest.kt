package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverOrder
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * **متجرٌ بلا دبّوس — «لا يُعرف» لا «بعيد»** (٢٠٢٦-١٠-٠٢).
 *
 * **كان يُقرأ بعيداً** فيُخفى زرُّ «وصلت المتجر» عن سائقٍ يقف أمامه، ولا آليّةَ
 * تضغطه عنه: **طلبٌ لا يتقدّم.**
 */
class ArrivalPointTest {

    @Test
    fun `a store without a pin is unknown so the arrive button shows`() {
        val o = DriverOrder(status = "assigned", navLat = null, navLng = null)
        assertEquals(ArrivalPoint.Unknown, arrivalPoint(o))
    }

    @Test
    fun `a pinned store is a point`() {
        val o = DriverOrder(status = "assigned", navLat = 35.95, navLng = 39.01)
        assertEquals(ArrivalPoint.At(35.95, 39.01), arrivalPoint(o))
    }

    @Test
    fun `states between arrivals are not arrivals`() {
        for (s in listOf("at_pickup", "picked_up", "at_dropoff", "delivered")) {
            assertEquals(s, ArrivalPoint.NotArriving, arrivalPoint(DriverOrder(status = s, navLat = 1.0, navLng = 1.0)))
        }
    }
}
