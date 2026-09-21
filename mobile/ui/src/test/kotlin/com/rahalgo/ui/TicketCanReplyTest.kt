package com.rahalgo.ui

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * **بوّابةُ الردّ على الشكوى** — `SUP-014`.
 *
 * **المحلولةُ لا يُردُّ عليها** (المحرّكُ يردّ `409 ticket_resolved`)،
 * **والمفتوحةُ وقيدُ المعالجة تقبلان.** **والشاشةُ تتبع المحرّك**: تُخفي
 * الحقلَ وتقول لماذا، لا حقلاً باهتاً يُضغط فيُردّ.
 */
class TicketCanReplyTest {

    @Test
    fun openAndInProgressAcceptReplies() {
        assertTrue("**المفتوحةُ يجب أن تقبل ردّاً**", ticketCanReply("open"))
        assertTrue("**قيدُ المعالجة يجب أن تقبل ردّاً**", ticketCanReply("in_progress"))
    }

    @Test
    fun resolvedRejectsReplies() {
        assertFalse(
            "**المحلولةُ يجب ألّا تقبل ردّاً — يوافق المحرّكَ (409 ticket_resolved)**",
            ticketCanReply("resolved"),
        )
    }
}
