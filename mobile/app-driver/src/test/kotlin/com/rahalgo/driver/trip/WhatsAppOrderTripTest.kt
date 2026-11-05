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
 * **«طلب واتساب» على بطاقة السائق** (قرارُ المالك ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * طلبٌ كتبه المكتبُ من رسالة واتساب — «لدي توصيلة» في المحرّك (`kind=merchant_delivery`) بأصله
 * `origin=whatsapp`. **فالبطاقةُ تقول «طلب واتساب» وصندوقُ ما يحمله «الطلب»** — وخادمٌ قديمٌ بلا
 * الحقل يبقى «لدي توصيلة».
 */
class WhatsAppOrderTripTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun `a whatsapp order is labelled as such`() {
        val o = json.decodeFromString(
            DriverOrder.serializer(),
            """{"id":"w1","status":"dispatching","kind":"merchant_delivery","origin":"whatsapp",
               "delivery_mode":"send","parcel_note":"2 شاورما","fee_payer":"recipient"}""",
        )
        assertTrue(isWhatsApp(o))
        assertFalse(isFetch(o))
        assertEquals(R.string.kind_whatsapp, kindLabel(o))
        assertEquals(R.string.detail_parcel_whatsapp, parcelLabel(o))
    }

    @Test
    fun `a store delivery and an old server keep their words`() {
        val o = json.decodeFromString(
            DriverOrder.serializer(),
            """{"id":"d1","status":"dispatching","kind":"merchant_delivery","delivery_mode":"send"}""",
        )
        assertFalse(isWhatsApp(o))
        assertEquals(R.string.kind_delivery, kindLabel(o))
        assertEquals(R.string.detail_parcel, parcelLabel(o))
    }
}
