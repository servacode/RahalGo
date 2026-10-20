package com.rahalgo.driver.trip

import com.rahalgo.driver.R
import com.rahalgo.shared.model.DriverOrder
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«إحضار»: المبلغُ الحقيقيُّ وموافقةُ المتجر** (قرارُ المالك ٢٠٢٦-١٠-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * بمكان الاستلام يكتب ما دفعه، **ولا «استلمت» قبل أن يوافق المتجر.** وبعد الموافقة يُقبض من
 * المتجر المبلغُ الحقيقيُّ لا التقدير، **والكودُ يُطلب من المتجر لا من الزبون.**
 */
class FetchAmountTripTest {

    private val json = Json { ignoreUnknownKeys = true }

    private fun fetch(status: String = "at_pickup", estimate: Long? = 20000, amount: Long? = null, st: String = "") =
        DriverOrder(
            id = "f1", status = status, kind = "merchant_delivery", deliveryMode = "fetch",
            collectAmount = estimate, fetchAmount = amount, fetchAmountStatus = st,
        )

    @Test
    fun `the server fields are read`() {
        val o = json.decodeFromString(
            DriverOrder.serializer(),
            """{"id":"x","status":"at_pickup","kind":"merchant_delivery","delivery_mode":"fetch",
               "collect_amount":20000,"fetch_amount":21000,"fetch_amount_version":2,"fetch_amount_status":"pending"}""",
        )
        assertEquals(21000L, o.fetchAmount)
        assertEquals(2L, o.fetchAmountVersion)
        assertEquals("pending", o.fetchAmountStatus)
        // **وبلا عملةٍ ليرة** — خادمٌ أقدم لا يرسلها.
        assertEquals("SYP", o.amountCurrency)
    }

    /**
     * **دولار** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — المبلغان من الخادم **بالسنتات** ويُعرضان دولاراً،
     * **والمرفوضُ يُعاد إلى الحقل بالوحدة الكبرى** (١٣٫٢٥ لا ١٣٢٥).
     */
    @Test
    fun `a dollar order shows cents as dollars and drafts major units`() {
        val o = json.decodeFromString(
            DriverOrder.serializer(),
            """{"id":"x","status":"at_pickup","kind":"merchant_delivery","delivery_mode":"fetch",
               "collect_amount":1250,"fetch_amount":1325,"fetch_amount_status":"rejected","amount_currency":"USD"}""",
        )
        assertEquals("USD", o.amountCurrency)
        assertEquals("12.50$", orderAmount(o, o.collectAmount!!))
        assertEquals("13.25$", orderAmount(o, o.fetchAmount!!))
        assertEquals("13.25", fetchAmountDraft(1325, "USD"))
        assertEquals("13", fetchAmountDraft(1300, "USD"))
        assertEquals("0.05", fetchAmountDraft(5, "USD"))
        assertEquals("21000", fetchAmountDraft(21000, "SYP"))
        assertEquals("21,000 ل.س", orderAmount(fetch(), 21000))
    }

    @Test
    fun `an estimate means the amount must be entered and approved before pickup`() {
        assertEquals(FetchAmountStep.ENTER, fetchAmountStep(fetch()))
        assertTrue(fetchBlocksPickup(fetch()))
        assertEquals(FetchAmountStep.WAITING, fetchAmountStep(fetch(amount = 21000, st = "pending")))
        assertTrue(fetchBlocksPickup(fetch(amount = 21000, st = "pending")))
        assertEquals(FetchAmountStep.REJECTED, fetchAmountStep(fetch(amount = 21000, st = "rejected")))
        assertTrue(fetchBlocksPickup(fetch(amount = 21000, st = "rejected")))
        assertEquals(FetchAmountStep.APPROVED, fetchAmountStep(fetch(amount = 21000, st = "approved")))
        assertFalse(fetchBlocksPickup(fetch(amount = 21000, st = "approved")))
    }

    @Test
    fun `no estimate, not at pickup, or not fetch — no step`() {
        assertEquals(FetchAmountStep.NONE, fetchAmountStep(fetch(estimate = null)))
        assertFalse(fetchBlocksPickup(fetch(estimate = null)))
        assertEquals(FetchAmountStep.NONE, fetchAmountStep(fetch(status = "assigned")))
        assertEquals(FetchAmountStep.NONE, fetchAmountStep(fetch(status = "on_the_way", amount = 1, st = "approved")))
        val send = fetch().copy(deliveryMode = "send")
        assertEquals(FetchAmountStep.NONE, fetchAmountStep(send))
        assertFalse(fetchBlocksPickup(send))
    }

    @Test
    fun `after approval the real amount is collected from the store, not the estimate`() {
        assertEquals(20000L, collectForStore(fetch()))
        assertEquals(20000L, collectForStore(fetch(amount = 21000, st = "pending")))
        assertEquals(21000L, collectForStore(fetch(status = "on_the_way", amount = 21000, st = "approved")))
    }

    @Test
    fun `in fetch the code is asked from the store`() {
        assertEquals(R.string.deliver_code_ask_store, deliverCodeAskRes(fetch()))
        assertEquals(R.string.deliver_code_ask, deliverCodeAskRes(fetch().copy(deliveryMode = "send")))
    }
    @Test
    fun `fetch stages read pickup place first then the store`() {
        assertEquals(
            listOf(
                R.string.leg_to_pickup_place, R.string.leg_at_pickup_place, R.string.leg_picked,
                R.string.leg_to_store, R.string.trip_s_at_pickup, R.string.ord_st_delivered,
            ),
            FETCH_LEGS,
        )
        assertEquals(R.string.step_at_pickup_place, nextAction("assigned", fetch = true)?.label)
        assertEquals("at_pickup", nextAction("assigned", fetch = true)?.status)
        assertEquals(R.string.step_at_pickup, nextAction("on_the_way", fetch = true)?.label)
        assertEquals(R.string.step_at_dropoff, nextAction("on_the_way")?.label)
        assertEquals(R.string.leg_at_pickup_place, phaseLabel(TripStep.AT_PICKUP, fetch = true))
        assertEquals(R.string.trip_p_at_store, phaseLabel(TripStep.AT_CUSTOMER, fetch = true))
    }
}
