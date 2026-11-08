package com.rahalgo.customer.shamcash

import com.rahalgo.shared.customer.Bill
import com.rahalgo.shared.customer.ShamBillReply
import com.rahalgo.shared.customer.ShamConfig
import com.rahalgo.shared.customer.ShamStarted
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ShamCashTest {

    private val json = Json { ignoreUnknownKeys = true; explicitNulls = false }

    @Test
    fun configParses() {
        val c = json.decodeFromString<ShamConfig>("""{"enabled":true,"min":1000,"max":500000,"expires_minutes":10}""")
        assertTrue(c.enabled)
        assertEquals(1000L, c.min)
        assertEquals(500000L, c.max)
        assertEquals(10, c.expiresMinutes)
    }

    @Test
    fun configDefaultsToOff() {
        assertFalse(json.decodeFromString<ShamConfig>("{}").enabled)
    }

    @Test
    fun pendingBillParses() {
        val raw = """{"bill":{"id":"b1","number":"SC-1","purpose":"topup","amount":5000,"status":"pending",
            "created_at":"2026-11-09T10:00:00Z","expires_at":"2026-11-09T10:10:00Z","paid_at":null,
            "order_id":null,"order_number":null,"order_error":"","payment_url":"https://pay.example/x"},
            "payment_url":"https://pay.example/x"}"""
        val s = json.decodeFromString<ShamStarted>(raw)
        assertEquals("b1", s.bill.id)
        assertTrue(s.bill.pending)
        assertFalse(s.bill.isFinal)
        assertNull(s.bill.orderId)
        assertEquals("https://pay.example/x", s.paymentUrl)
    }

    @Test
    fun paidOrderBillParses() {
        val raw = """{"bill":{"id":"b2","purpose":"order","amount":9000,"status":"paid",
            "order_id":"o-9","order_number":123,"order_error":""}}"""
        val b = json.decodeFromString<ShamBillReply>(raw).bill
        assertTrue(b.paid)
        assertTrue(b.isFinal)
        assertEquals("o-9", b.orderId)
        assertEquals(123L, b.orderNumber)
        assertTrue(b.orderError.isEmpty())
    }

    @Test
    fun paidWithOrderErrorKeepsTheError() {
        val b = json.decodeFromString<Bill>(
            """{"id":"b3","purpose":"order","status":"paid","order_error":"out_of_stock"}""",
        )
        assertTrue(b.paid)
        assertEquals("out_of_stock", b.orderError)
    }

    @Test
    fun deepLinkExtractsBillId() {
        assertEquals("abc-123", ShamLink.billId("rahalgo.com", "/wallet/shamcash/abc-123"))
        assertEquals("abc-123", ShamLink.billId("www.rahalgo.com", "/wallet/shamcash/abc-123/"))
        assertEquals("abc-123", ShamLink.billId("RAHALGO.com", "/wallet/shamcash/abc-123/x"))
    }

    @Test
    fun deepLinkRejectsOthers() {
        assertNull(ShamLink.billId("rahalgo.com", "/signup"))
        assertNull(ShamLink.billId("rahalgo.com", "/wallet/shamcash"))
        assertNull(ShamLink.billId("rahalgo.com", "/wallet/shamcash/"))
        assertNull(ShamLink.billId("evil.com", "/wallet/shamcash/abc"))
        assertNull(ShamLink.billId(null, "/wallet/shamcash/abc"))
        assertNull(ShamLink.billId("rahalgo.com", "/wallet/shamcash/a b"))
        assertNull(ShamLink.billId("rahalgo.com", "/wallet/shamcashx/abc"))
    }
}
