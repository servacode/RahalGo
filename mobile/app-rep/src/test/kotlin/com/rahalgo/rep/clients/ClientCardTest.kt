package com.rahalgo.rep.clients

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/** **«10٪» معزولةً يساريّاً** — الرقمُ ثمّ ٪ بعده مهما كان اتّجاهُ الجملة حولها (بلاغُ المالك ٢٠٢٦-١١-١١: «%10»). */
class ClientCardTest {
    private val open = "⁦"
    private val close = "⁩"

    @Test
    fun `النسبةُ رقمٌ ثمّ علامةٌ بعده ومعزولة`() {
        assertEquals(open + "10٪" + close, pctLabel(10.0))
        assertEquals(open + "7.5٪" + close, pctLabel(7.5))
        assertEquals(open + "0٪" + close, pctLabel(0.0))
    }

    @Test
    fun `العلامةُ لا تسبق الرقم`() {
        val s = pctLabel(10.0)
        assertTrue(s.indexOf('1') < s.indexOf('٪'))
        assertFalse(s.contains('%'))
    }
}
