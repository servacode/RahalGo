package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverOrder
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **«التوصيل من المحفظة والباقي كاش»** (قرارُ المالك ٢٠٢٦-١١-٠٩) في يد الكابتن: يقبض المشتريات
 * وحدَها (`cash_due`)، **والأجرةُ تُقال مدفوعةً لا تُطالَب.** والخادمُ القديمُ بلا الحقل يبقى كما كان.
 */
class WalletScopeTripTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun `delivery scope collects goods only and says the fee is paid`() {
        val o = json.decodeFromString(
            DriverOrder.serializer(),
            """{"id":"c1","status":"on_the_way","kind":"custom","custom_mode":"purchase",
               "payment_method":"wallet","wallet_scope":"delivery","cash_due":12000,"total":17000}""",
        )
        assertEquals("delivery", o.walletScope)
        assertTrue(feePaidFromWallet(o))
        assertEquals(CashFrom.RECIPIENT, cashFrom(o, pickedUp = true))
        assertEquals(12000L, o.cashDue)
    }

    @Test
    fun `old server without the field keeps the old reading`() {
        val o = json.decodeFromString(
            DriverOrder.serializer(),
            """{"id":"c2","status":"on_the_way","kind":"custom","cash_due":17000,"total":17000}""",
        )
        assertEquals("", o.walletScope)
        assertFalse(feePaidFromWallet(o))
        assertFalse(feePaidFromWallet(DriverOrder(id = "c3", walletScope = "full")))
    }
}
