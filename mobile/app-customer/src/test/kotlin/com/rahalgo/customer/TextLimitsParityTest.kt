package com.rahalgo.customer

import com.rahalgo.shared.model.TextLimits
import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **حدودُ الحقول في التطبيق مرآةُ حدود المحرّك** — رقمٌ يتبدّل هناك يتبدّل هنا.
 *
 * (فحصُ القبول ٢٠٢٦-١٠-٠٣: لا حدَّ لنصوص الطلب.)
 */
class TextLimitsParityTest {

    private fun goConst(src: String, name: String): Int =
        Regex("""\b$name\s*=\s*(\d+)""").find(src)?.groupValues?.get(1)?.toInt()
            ?: error("لم أجد $name في text_limits.go")

    @Test
    fun `الأرقامُ نفسُها في الطرفين`() {
        var dir = File("").absoluteFile
        while (!File(dir, "settings.gradle.kts").exists()) dir = dir.parentFile
        val go = File(dir.parentFile, "backend/internal/server/text_limits.go").readText()
        assertEquals(goConst(go, "maxAddressText"), TextLimits.ADDRESS_TEXT)
        assertEquals(goConst(go, "maxOrderNotes"), TextLimits.ORDER_NOTES)
        assertEquals(goConst(go, "maxItemNote"), TextLimits.ITEM_NOTE)
        assertEquals(goConst(go, "maxCustomRequest"), TextLimits.CUSTOM_REQUEST)
        assertEquals(goConst(go, "maxComplaintNote"), TextLimits.COMPLAINT_NOTE)
        assertEquals(goConst(go, "maxTicketReply"), TextLimits.TICKET_REPLY)
        assertEquals(goConst(go, "maxAddressPart"), TextLimits.ADDRESS_PART)
        assertEquals(goConst(go, "maxAddressFloor"), TextLimits.ADDRESS_FLOOR)
        val comms = File(dir.parentFile, "backend/internal/comms/messages.go").readText()
        assertEquals(goConst(comms, "MaxBody"), TextLimits.CHAT_BODY)
    }

    /** **والقصُّ بالحرف** — العربيُّ لا يُكسر. */
    @Test
    fun `القصُّ يحفظ الحروف`() {
        assertEquals("عربي", TextLimits.fit("عربيّة", 4))
        assertEquals("قصير", TextLimits.fit("قصير", 10))
        assertTrue(TextLimits.fit("x".repeat(2000), TextLimits.CUSTOM_REQUEST).length == 1000)
    }
}
