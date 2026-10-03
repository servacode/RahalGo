package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverOrder
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **مشوارُ إرجاع البضاعة** (قرارُ المالك ٢٠٢٦-١٠-٠٣).
 *
 * **الطلبُ أُنهي (`failed`) والبضاعةُ مع السائق** — فيبقى في قائمته بوجهةٍ هي المكتبُ أو
 * المتجر، **وتُرسم الخريطةُ إليها كما تُرسم إلى الزبون**، ولا وصولَ تلقائيّاً: الزرُّ بيده.
 */
class ReturnTripTest {

    @Test
    fun `a failed order with a return destination is a return trip`() {
        assertTrue(isReturnTrip(DriverOrder(status = "failed", returnTo = "office")))
        assertTrue(isReturnTrip(DriverOrder(status = "failed", returnTo = "store")))
    }

    @Test
    fun `a failed order without a destination or a live order is not`() {
        assertFalse(isReturnTrip(DriverOrder(status = "failed")))
        assertFalse(isReturnTrip(DriverOrder(status = "on_the_way")))
        // **والحقلُ وحدَه لا يكفي** — طلبٌ قائمٌ لا يُقرأ مشوارَ إرجاع.
        assertFalse(isReturnTrip(DriverOrder(status = "at_dropoff", returnTo = "office")))
    }

    @Test
    fun `the return leg is drawn like the leg to the customer`() {
        val o = DriverOrder(status = "failed", returnTo = "office", lat = 35.96, lng = 39.01)
        assertEquals(TripStep.TO_CUSTOMER, tripStepOf(o))
        // **والوجهةُ نقطةُ الباب نفسُها** — `lat`/`lng` التي كتبها الخادمُ للمكتب.
        assertEquals(35.96 to 39.01, dropoffPoint(o))
    }

    @Test
    fun `an office without a pin has no point on the map`() {
        val o = DriverOrder(status = "failed", returnTo = "office", dropoffKnown = false)
        assertNull(dropoffPoint(o))
    }

    @Test
    fun `there is no automatic arrival on a return trip`() {
        assertNull(autoArrivalTarget("failed", custom = false))
        assertEquals(ArrivalPoint.NotArriving, arrivalPoint(DriverOrder(status = "failed", returnTo = "office")))
    }

    @Test
    fun `other orders keep their usual step`() {
        assertEquals(TripStep.TO_PICKUP, tripStepOf(DriverOrder(status = "assigned")))
        assertEquals(TripStep.AT_CUSTOMER, tripStepOf(DriverOrder(status = "at_dropoff")))
    }
}
