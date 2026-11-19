package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverOrder
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * **«أبعت أمانة»: الأجرةُ من المرسِل وقت الاستلام، وعند المستلِم ولا شي** (قرارُ المالك ٢٠٢٦-١١-١٩).
 * **و«أستقبل» كما كانت: عند التسليم.** فما بينطلب من حدا مرّتين.
 */
class AmanahSendCashTest {
    private fun amanah(dir: String) =
        DriverOrder(status = "assigned", kind = "custom", customMode = "amanah", amanahDirection = dir, cashDue = 300)

    @Test
    fun sendCollectsAtPickupOnly() {
        val o = amanah("send")
        assertEquals(CashFrom.STORE, cashFrom(o, pickedUp = false))
        assertEquals(CashFrom.NONE, cashFrom(o, pickedUp = true))
    }

    @Test
    fun receiveCollectsAtDoorOnly() {
        val o = amanah("receive")
        assertEquals(CashFrom.NONE, cashFrom(o, pickedUp = false))
        assertEquals(CashFrom.RECIPIENT, cashFrom(o, pickedUp = true))
    }

    @Test
    fun oldServerWithoutDirectionIsReceive() {
        val o = amanah("")
        assertEquals(CashFrom.RECIPIENT, cashFrom(o, pickedUp = true))
    }
}
