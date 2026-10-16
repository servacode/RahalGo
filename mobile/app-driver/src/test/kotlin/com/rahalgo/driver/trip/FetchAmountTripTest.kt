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
}
