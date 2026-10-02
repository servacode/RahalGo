package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverOrder
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«لدي توصيلة» في يد السائق** (٢٠٢٦-١٠-٠٢)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **بلا نقطة تسليمٍ يُكتب مكانَها موقعُ المتجر** — فكانت الملاحةُ تقوده بعد
 * الاستلام إلى المتجر نفسِه، **ويُعلَن وصولُه إلى المستلِم وهو واقفٌ هناك.**
 * **و«أنا نقداً» يقبضها من المتجر** — والبطاقةُ كانت تطلبها من المستلِم.
 */
class MerchantDeliveryTripTest {

    private val json = Json { ignoreUnknownKeys = true }

    private fun delivery(status: String, known: Boolean, payer: String = "recipient", cash: Long = 5000) =
        DriverOrder(
            id = "d1", status = status, kind = "merchant_delivery",
            dropoffKnown = known, feePayer = payer, cashDue = cash,
            lat = 35.95, lng = 39.01, navLat = 35.95, navLng = 39.01,
        )

    @Test
    fun `the server field is read — and not defaulted to known`() {
        val o = json.decodeFromString(
            DriverOrder.serializer(),
            """{"id":"x","status":"on_the_way","kind":"merchant_delivery","dropoff_known":false,
               "parcel_note":"كيس","fee_payer":"merchant_cash"}""",
        )
        assertFalse(o.dropoffKnown)
        assertEquals("كيس", o.parcelNote)
        assertEquals("merchant_cash", o.feePayer)
    }

    @Test
    fun `no auto-arrival at an unknown drop-off`() {
        assertNull(autoArrivalTarget("on_the_way", custom = false, dropoffKnown = false))
        // **والمتجرُ معروفٌ** — الوصولُ إليه يبقى تلقائيّاً.
        assertEquals("at_pickup", autoArrivalTarget("assigned", custom = false, dropoffKnown = false))
        assertEquals("at_dropoff", autoArrivalTarget("on_the_way", custom = false, dropoffKnown = true))
    }

    @Test
    fun `an unknown drop-off is unknown — not a point at the store`() {
        assertEquals(ArrivalPoint.Unknown, arrivalPoint(delivery("on_the_way", known = false)))
        assertNull(dropoffPoint(delivery("on_the_way", known = false)))
        assertEquals(ArrivalPoint.At(35.95, 39.01), arrivalPoint(delivery("on_the_way", known = true)))
        assertEquals(35.95 to 39.01, dropoffPoint(delivery("on_the_way", known = true)))
    }

    @Test
    fun `merchant cash is collected at the store, never from the recipient`() {
        val o = delivery("assigned", known = true, payer = "merchant_cash")
        assertEquals(CashFrom.STORE, cashFrom(o, pickedUp = false))
        assertEquals(CashFrom.NONE, cashFrom(o, pickedUp = true))
    }

    @Test
    fun `the recipient pays at the door, the wallet payer pays nothing`() {
        assertEquals(CashFrom.NONE, cashFrom(delivery("assigned", true, "recipient"), pickedUp = false))
        assertEquals(CashFrom.RECIPIENT, cashFrom(delivery("on_the_way", true, "recipient"), pickedUp = true))
        assertEquals(CashFrom.NONE, cashFrom(delivery("on_the_way", true, "merchant", cash = 0), pickedUp = true))
        // **والطلبُ العاديُّ كما كان** — يقبض من الزبون بعد الاستلام.
        val std = DriverOrder(status = "on_the_way", cashDue = 9000)
        assertEquals(CashFrom.RECIPIENT, cashFrom(std, pickedUp = true))
    }
}
