package com.rahalgo.shared.merchant

import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **عمولةٌ بمنزلتين و«مفتوح ٢٤ ساعة»** — قرارُ المالك ٢٠٢٦-١٠-١٨.
 */
class CommissionAndHoursTest {

    /** **نسخةُ إعداد `ApiClient`.** */
    private val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
    }

    @Test
    fun exactPercentWinsAndOldServerFallsBack() {
        val now = json.decodeFromString<MerchantOrder>(
            """{"id":"a","commission_percent":2,"commission_percent_exact":1.5}""",
        )
        assertEquals("1.5", now.commissionLabel)
        // **وخادمٌ أقدمُ يرسل الصحيحةَ وحدَها** — تُعرض كما كانت.
        val old = json.decodeFromString<MerchantOrder>("""{"id":"a","commission_percent":10}""")
        assertEquals("10", old.commissionLabel)
        assertEquals("2.25", percentLabel(2.25, 2))
        assertEquals("10", percentLabel(10.0, 10))
        assertEquals("0.05", percentLabel(0.05, 0))
    }

    @Test
    fun open24hReadsFlagOrMidnightPair() {
        val flagged = json.decodeFromString<DayHours>(
            """{"day_of_week":1,"closed":false,"open_time":"00:00","close_time":"00:00","open_24h":true}""",
        )
        assertTrue(flagged.isOpen24h)
        // **خادمٌ أقدمُ بلا راية** — ٠٠:٠٠ ← ٠٠:٠٠ تُقرأ ٢٤ ساعة.
        val old = json.decodeFromString<DayHours>(
            """{"day_of_week":1,"closed":false,"open_time":"00:00:00","close_time":"00:00:00"}""",
        )
        assertTrue(old.isOpen24h)
        val normal = DayHours(1, false, "09:00", "23:00")
        assertFalse(normal.isOpen24h)
    }

    @Test
    fun toggling24hWritesAndClearsTheTimes() {
        val on = DayHours(2, true, "09:00", "23:00").with24h(true)
        assertEquals(DayHours(2, false, "00:00", "00:00", open24h = true), on)
        // **والإطفاءُ لا يترك ٠٠:٠٠** — وإلّا قُرئ ٢٤ ساعة ثانيةً.
        val off = on.with24h(false)
        assertFalse(off.isOpen24h)
        assertEquals("09:00", off.openTime)
        assertTrue(json.encodeToString(on).contains("\"open_24h\":true"))
    }
}
