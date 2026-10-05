package com.rahalgo.driver.trip

import com.rahalgo.shared.model.DriverOrder
import java.time.ZoneId
import org.junit.Assert.assertEquals
import org.junit.Test

/** **متى يجهز الطلب** — كانت حقولُه تصل ولا تُعرض (٢٠٢٦-١٠-٠٢). */
class PrepStateTest {

    private val damascus = ZoneId.of("Asia/Damascus")

    @Test
    fun `the store's ready stamp wins`() {
        val o = DriverOrder(readyAt = "2026-10-02T10:00:00Z", prepMinutes = 20, acceptedAt = "2026-10-02T09:50:00Z")
        assertEquals(PrepState.Ready, prepState(o, damascus))
    }

    @Test
    fun `accepted plus prep minutes is shown in local time`() {
        val o = DriverOrder(prepMinutes = 25, acceptedAt = "2026-10-02T09:50:00Z")
        // ٠٩:٥٠ عالميّاً + ٢٥ = ١٠:١٥ عالميّاً = ١٣:١٥ بدمشق.
        assertEquals(PrepState.Around("13:15"), prepState(o, damascus))
    }

    @Test
    fun `default zone is Syria not the device zone`() {
        val saved = java.util.TimeZone.getDefault()
        try {
            // **جهازٌ على غرينتش** — كالمحاكي: كان يقول ١٠:١٥ لا ١٣:١٥.
            java.util.TimeZone.setDefault(java.util.TimeZone.getTimeZone("UTC"))
            val o = DriverOrder(prepMinutes = 25, acceptedAt = "2026-10-02T09:50:00Z")
            assertEquals(PrepState.Around("13:15"), prepState(o))
        } finally {
            java.util.TimeZone.setDefault(saved)
        }
    }

    @Test
    fun `nothing known says nothing — and custom has no kitchen`() {
        assertEquals(PrepState.Unknown, prepState(DriverOrder(), damascus))
        assertEquals(PrepState.Unknown, prepState(DriverOrder(prepMinutes = 10), damascus))
        assertEquals(
            PrepState.Unknown,
            prepState(DriverOrder(kind = "custom", readyAt = "2026-10-02T10:00:00Z"), damascus),
        )
    }
}
