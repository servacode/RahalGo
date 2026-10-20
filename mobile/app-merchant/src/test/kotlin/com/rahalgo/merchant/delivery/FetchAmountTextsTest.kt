package com.rahalgo.merchant.delivery

import com.rahalgo.merchant.R
import com.rahalgo.shared.merchant.Delivery
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«إحضار»: نافذةُ موافقة المتجر على مبلغ السائق، وكودُ الاستلام له** (قرارُ المالك ٢٠٢٦-١٠-١٦)
 * ══════════════════════════════════════════════════════════════════════
 */
class FetchAmountTextsTest {

    private val json = Json { ignoreUnknownKeys = true }

    private fun d(id: String, mode: String = "fetch", status: String = "at_pickup", st: String = "pending", amount: Long? = 21000) =
        Delivery(id = id, deliveryMode = mode, status = status, fetchAmountStatus = st, fetchAmount = amount, fetchAmountVersion = 1)

    @Test
    fun `the server fields are read — amount, version, status and the store's code`() {
        val x = json.decodeFromString(
            Delivery.serializer(),
            """{"id":"x","status":"picked_up","delivery_mode":"fetch","collect_amount":20000,
               "fetch_amount":21000,"fetch_amount_version":3,"fetch_amount_status":"approved","delivery_code":"0427"}""",
        )
        assertEquals(21000L, x.fetchAmount)
        assertEquals(3L, x.fetchAmountVersion)
        assertEquals("approved", x.fetchAmountStatus)
        assertEquals("0427", x.deliveryCode)
        // **وبلا عملةٍ ليرة** — خادمٌ أقدم لا يرسلها.
        assertEquals("SYP", x.amountCurrency)
    }

    /** **دولار** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — **المبلغُ بالسنتات** ويُعرض بعملته. */
    @Test
    fun `a dollar delivery reads its currency and shows cents as dollars`() {
        val x = json.decodeFromString(
            Delivery.serializer(),
            """{"id":"x","status":"at_pickup","delivery_mode":"fetch","collect_amount":1250,
               "fetch_amount":1325,"fetch_amount_version":1,"fetch_amount_status":"pending","amount_currency":"USD"}""",
        )
        assertEquals("USD", x.amountCurrency)
        assertEquals("12.50$", DeliveryTexts.amount(x, x.collectAmount!!))
        assertEquals("13.25$", DeliveryTexts.amount(x, x.fetchAmount!!))
    }

    @Test
    fun `only a pending amount at the pickup place in fetch opens the popup`() {
        assertTrue(DeliveryTexts.awaitsAmount(d("a")))
        assertFalse(DeliveryTexts.awaitsAmount(d("b", st = "approved")))
        assertFalse(DeliveryTexts.awaitsAmount(d("c", st = "rejected")))
        assertFalse(DeliveryTexts.awaitsAmount(d("d", mode = "send")))
        assertFalse(DeliveryTexts.awaitsAmount(d("e", status = "picked_up")))
        assertFalse(DeliveryTexts.awaitsAmount(d("f", amount = null)))
        assertNull(DeliveryTexts.pendingAmount(null))
        assertNull(DeliveryTexts.pendingAmount(listOf(d("b", st = "approved"))))
        assertEquals("a", DeliveryTexts.pendingAmount(listOf(d("b", st = "approved"), d("a")))?.id)
    }

    @Test
    fun `the details row follows the decision`() {
        assertEquals(R.string.md_amount_row_pending, DeliveryTexts.amountRow("pending"))
        assertEquals(R.string.md_amount_row_approved, DeliveryTexts.amountRow("approved"))
        assertEquals(R.string.md_amount_row_rejected, DeliveryTexts.amountRow("rejected"))
        assertNull(DeliveryTexts.amountRow(""))
    }
}
