package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverOrder
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **الكابتن يطلب تعديل ثمن البضاعة بعد الاستلام والزبون يوافق** (قرار المالك ٢٠٢٦-١١-٠٦).
 *
 * الزر بالمشتريات بعد الاستلام بس، **وما في «تم التسليم» والسعر بانتظار الزبون**، وبعد
 * الرفض بيقول «رجع السعر القديم».
 */
class PriceEditTripTest {

    private fun purchase(
        status: String = "on_the_way",
        mode: String = "purchase",
        version: Long = 2,
        confirmed: Long? = 2,
        rejected: String? = null,
        goods: Long? = 1000,
        pending: Boolean = false,
    ) = DriverOrder(
        id = "p1", status = status, kind = "custom", customMode = mode,
        customGoods = goods, customFee = 1500, customGoodsPending = pending,
        quoteVersion = version, quoteConfirmedVersion = confirmed, quoteRejectedAt = rejected,
    )

    @Test
    fun `after pickup a confirmed purchase can request`() {
        for (st in listOf("picked_up", "on_the_way", "at_dropoff")) {
            assertEquals(st, PriceEditStep.CAN_REQUEST, priceEditStep(purchase(status = st)))
        }
    }

    @Test
    fun `before pickup, amanah and standard have no button`() {
        assertEquals(PriceEditStep.NONE, priceEditStep(purchase(status = "assigned")))
        assertEquals(PriceEditStep.NONE, priceEditStep(purchase(status = "at_pickup")))
        assertEquals(PriceEditStep.NONE, priceEditStep(purchase(mode = "amanah")))
        assertEquals(PriceEditStep.NONE, priceEditStep(purchase(status = "delivered")))
        assertEquals(PriceEditStep.NONE, priceEditStep(purchase(confirmed = null)))
        assertEquals(PriceEditStep.NONE, priceEditStep(purchase(pending = true)))
        assertEquals(
            PriceEditStep.NONE,
            priceEditStep(DriverOrder(id = "s", status = "on_the_way", kind = "standard")),
        )
    }

    @Test
    fun `a pending price blocks delivery`() {
        val o = purchase(version = 3, confirmed = 2)
        assertEquals(PriceEditStep.WAITING, priceEditStep(o))
        assertTrue(priceEditBlocksDelivery(o))
        assertFalse(priceEditBlocksDelivery(purchase()))
    }

    @Test
    fun `a rejected request reverts and may ask again`() {
        val o = purchase(version = 4, confirmed = 4, rejected = "2026-11-06T10:00:00Z")
        assertEquals(PriceEditStep.REVERTED, priceEditStep(o))
        assertFalse(priceEditBlocksDelivery(o))
    }
}
