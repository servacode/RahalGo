package com.rahalgo.driver.orders

import com.rahalgo.shared.model.DriverOrder
import com.rahalgo.shared.model.DriverOutcome
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **طلبٌ خرج من يده — يُقال لماذا** (٢٠٢٦-١٠-٠٢).
 *
 * **كان يُغلَق ويقفز التطبيقُ صامتاً** — والآن ما اختفى من القائمة يُسأل عنه.
 */
class DeparturesTest {

    @Test
    fun `an order that left the list is detected`() {
        val now = listOf(DriverOrder(id = "b"))
        assertEquals(listOf("a"), Departures.departed(listOf("a", "b"), now))
        assertTrue(Departures.departed(listOf("b"), now).isEmpty())
        // **وطلبٌ جديدٌ لا يُعدّ خروجاً.**
        assertTrue(Departures.departed(emptyList(), listOf(DriverOrder(id = "c"))).isEmpty())
    }

    @Test
    fun `every server reason has its own sentence`() {
        for (code in listOf(
            "cancelled_customer", "cancelled_merchant", "cancelled_ops",
            "requeued_ops", "requeued_system", "merchant_blocked", "failed_ops",
        )) {
            assertNotNull(code, Departures.reasonRes(code))
        }
        // **ورمزٌ لا نعرفه تُقرأ له جملةُ الخادم** — لا يُبتلع.
        assertNull(Departures.reasonRes("something_new"))
    }

    @Test
    fun `the outcome contract is read`() {
        val o = Json { ignoreUnknownKeys = true }.decodeFromString(
            DriverOutcome.serializer(),
            """{"order_id":"x","number":1219,"status":"cancelled","reason":"cancelled_customer","message":"m","at":"2026-10-02T10:00:00Z"}""",
        )
        assertEquals(1219L, o.number)
        assertEquals("cancelled_customer", o.reason)
    }
}
