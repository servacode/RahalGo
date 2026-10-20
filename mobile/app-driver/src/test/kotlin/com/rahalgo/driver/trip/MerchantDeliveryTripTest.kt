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

    /** **«إحضار»: من يقف عند نقطة الاستلام يدفع عندها** (قرارُ المالك ٢٠٢٦-١٠-١٥). */
    @Test
    fun `fetch reverses who pays where — and the collect amount is read`() {
        val o = json.decodeFromString(
            DriverOrder.serializer(),
            """{"id":"x","status":"assigned","kind":"merchant_delivery","fee_payer":"recipient",
               "cash_due":500,"delivery_mode":"fetch","collect_amount":25000}""",
        )
        assertEquals(true, isFetch(o))
        assertEquals(25000L, collectForStore(o))
        assertEquals(CashFrom.STORE, cashFrom(o, pickedUp = false))
        assertEquals(CashFrom.NONE, cashFrom(o, pickedUp = true))
        val mc = o.copy(feePayer = "merchant_cash")
        assertEquals(CashFrom.NONE, cashFrom(mc, pickedUp = false))
        assertEquals(CashFrom.RECIPIENT, cashFrom(mc, pickedUp = true))
        assertNull(collectForStore(delivery("assigned", true)))
    }

    /** **«لدي توصيلة» بالدولار** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — «ادفع للمتجر 25$» لا «2,500 ل.س». */
    @Test
    fun `a dollar collect amount is shown in dollars`() {
        val o = json.decodeFromString(
            DriverOrder.serializer(),
            """{"id":"x","status":"assigned","kind":"merchant_delivery","fee_payer":"merchant",
               "delivery_mode":"send","collect_amount":2500,"amount_currency":"USD"}""",
        )
        assertEquals(2500L, collectForStore(o))
        assertEquals("25$", orderAmount(o, collectForStore(o)!!))
        assertEquals("25,000 ل.س", orderAmount(o.copy(amountCurrency = "SYP"), 25000))
    }

    /**
     * **«إحضار»: يدفع بمكان الاستلام ويقبضه من المتجر** (قرارُ المالك ٢٠٢٦-١٠-١٦) — و«توصيل»
     * كما كان: يدفع للمتجر ويقبض من المستلم.
     */
    @Test
    fun `the collect text follows the mode`() {
        val send = delivery("assigned", true).copy(collectAmount = 25000)
        val fetch = send.copy(deliveryMode = "fetch")
        assertEquals(com.rahalgo.driver.R.string.trip_collect_for_store, collectTextRes(send))
        assertEquals(com.rahalgo.driver.R.string.trip_collect_for_store_fetch, collectTextRes(fetch))
        val xml = listOf(
            java.io.File("src/main/res/values/strings.xml"),
            java.io.File("app-driver/src/main/res/values/strings.xml"),
        ).first { it.exists() }.readText(Charsets.UTF_8)
        val text = Regex("""<string name="trip_collect_for_store_fetch">(.*?)</string>""")
            .find(xml)?.groupValues?.get(1)
        assertEquals("ادفع بمكان الاستلام %1\$s من جيبك، واقبضه من المتجر لما توصل", text)
    }
}
