package com.rahalgo.customer

import com.rahalgo.customer.mine.ticketKey
import com.rahalgo.shared.customer.TicketsPage
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * **رقمُ الشكوى كما يعرفه المكتب — لا ستّةُ أحرفٍ من معرّفها.**
 *
 * (فحصُ القبول ٢٠٢٦-١٠-٠٣: «شكاواي» عرضت «#08d6f3» والشكوى رقمُها ٥١١.)
 *
 * **والمحرّكُ يرسل `number` منذ بُني** (`handleMyTickets`) — **والنموذجُ
 * كان يقرأ `order_code` الذي لا يُرسَل**، فسقط إلى المعرّف.
 */
class TicketKeyTest {

    private val json = Json { ignoreUnknownKeys = true }

    /** **ردُّ `GET /my/tickets` كما يرسله المحرّك.** */
    private val body = """
        {"tickets":[{"id":"08d6f3a2-1b4c-4d5e-8f90-123456789abc","number":511,
          "order_number":null,"subject":"تأخّر الطلب","reason":"late","status":"open",
          "compensation":0,"resolution":"","created_at":"2026-10-03T16:51:00Z","resolved_at":null}]}
    """.trimIndent()

    @Test
    fun `رقمُ الشكوى يُعرض لا معرّفُها`() {
        val t = json.decodeFromString(TicketsPage.serializer(), body).tickets.single()
        assertEquals("#511", ticketKey(t))
    }
}
