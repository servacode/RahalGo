package com.rahalgo.shared.customer

import com.rahalgo.shared.merchant.MerchantOrder
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/** Multi-stop fields: new payloads parse, old payloads (no field) still parse. */
class MultiStopFieldsTest {
    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
    }

    @Test
    fun customerSectionsParse() {
        val o = json.decodeFromString<MyOrder>(
            """{"id":"a","sections":[{"seq":1,"status":"accepted","items":[{"name":"x","qty":2}]},
               {"seq":2,"status":"unavailable","items":[{"name":"y","qty":1}]}]}""",
        )
        assertEquals(2, o.sections!!.size)
        assertEquals("accepted", o.sections!![0].status)
        assertEquals(2, o.sections!![0].items[0].qty)
        assertEquals(2, o.sections!![1].seq)
    }

    @Test
    fun customerOldPayloadHasNoSections() {
        assertNull(json.decodeFromString<MyOrder>("""{"id":"a","status":"pending"}""").sections)
        assertNull(json.decodeFromString<MyOrder>("""{"id":"a","sections":null}""").sections)
    }

    @Test
    fun merchantSharedStopsParse() {
        assertEquals(3, json.decodeFromString<MerchantOrder>("""{"id":"a","shared_stops":3}""").sharedStops)
        assertNull(json.decodeFromString<MerchantOrder>("""{"id":"a"}""").sharedStops)
    }
}
