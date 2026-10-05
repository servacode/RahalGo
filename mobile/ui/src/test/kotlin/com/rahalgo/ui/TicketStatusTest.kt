package com.rahalgo.ui

import org.junit.Assert.assertEquals
import org.junit.Test

/** **«بانتظار المالية» تُقرأ «قيد المعالجة» لا بمفتاحها الإنكليزيّ.** */
class TicketStatusTest {
    @Test
    fun `بانتظار المالية تقرأ قيد المعالجة`() {
        assertEquals(R.string.tik_progress, ticketStatusRes("awaiting_finance"))
        assertEquals(R.string.tik_progress, ticketStatusRes("in_progress"))
        assertEquals(R.string.tik_open, ticketStatusRes("open"))
        assertEquals(R.string.tik_resolved, ticketStatusRes("resolved"))
    }
}
