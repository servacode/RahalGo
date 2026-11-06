package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverOrder
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **طلبُ الواتساب: ثمنُ الطلب يكتبه السائق** (قرارُ المالك ٢٠٢٦-١١-٠٦) — بمكان الاستلام قبل
 * «استلمت»، **بلا انتظار موافقة**، ثمّ يُقبض من الزبون الثمنُ المكتوب.
 */
class WhatsAppGoodsTripTest {

    private fun wa(status: String = "at_pickup", amount: Long? = null, st: String = "") =
        DriverOrder(
            id = "w1", status = status, kind = "merchant_delivery", deliveryMode = "send", origin = "whatsapp",
            fetchAmount = amount, fetchAmountStatus = st, collectAmount = amount,
        )

    @Test
    fun `at pickup the goods price is asked before pickup`() {
        assertEquals(FetchAmountStep.ENTER, fetchAmountStep(wa()))
        assertTrue(fetchBlocksPickup(wa()))
    }

    @Test
    fun `once typed it is done at once and pickup goes on`() {
        val o = wa(amount = 2400, st = "approved")
        assertEquals(FetchAmountStep.APPROVED, fetchAmountStep(o))
        assertFalse(fetchBlocksPickup(o))
        assertEquals(2400L, collectForStore(o))
    }

    @Test
    fun `before or after pickup there is no step`() {
        assertEquals(FetchAmountStep.NONE, fetchAmountStep(wa(status = "assigned")))
        assertEquals(FetchAmountStep.NONE, fetchAmountStep(wa(status = "picked_up", amount = 2400, st = "approved")))
        assertFalse(fetchBlocksPickup(wa(status = "picked_up")))
    }

    @Test
    fun `an ordinary send delivery is untouched`() {
        val send = DriverOrder(id = "s1", status = "at_pickup", kind = "merchant_delivery", deliveryMode = "send")
        assertEquals(FetchAmountStep.NONE, fetchAmountStep(send))
    }
}
