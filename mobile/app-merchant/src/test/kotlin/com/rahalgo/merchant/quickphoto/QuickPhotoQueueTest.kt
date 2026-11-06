package com.rahalgo.merchant.quickphoto

import com.rahalgo.shared.merchant.NoPhotoItem
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class QuickPhotoQueueTest {
    private fun items(vararg ids: String) = ids.map { NoPhotoItem(id = it, name = "item $it") }

    @Test
    fun walksInServerOrderAndCountsPosition() {
        var q = QuickPhotoQueue.of(items("a", "b", "c"))
        assertEquals(3, q.total)
        assertEquals("a", q.current?.id)
        assertEquals(1, q.position)
        q = q.advance()
        assertEquals("b", q.current?.id)
        assertEquals(2, q.position)
        assertEquals("c", q.nextId)
        q = q.advance().advance()
        assertTrue(q.finished)
        assertNull(q.current)
        assertEquals(3, q.position)
        assertEquals(3, q.advance().position)
    }

    @Test
    fun resumeStartsAtNextItemWithoutPhotoAndSkippedGoLast() {
        // a and b got photos (gone from the server list); c was skipped earlier in this session
        var q = QuickPhotoQueue.of(items("c", "d", "e"), skipped = setOf("c"))
        val seen = mutableListOf<String>()
        while (!q.finished) {
            seen += q.current!!.id
            q = q.advance()
        }
        assertEquals(listOf("d", "e", "c"), seen)
    }

    @Test
    fun itemsWithPhotoStillUploadingAreNotShownAgain() {
        val q = QuickPhotoQueue.of(items("a", "b", "c"), queuedForUpload = setOf("a", "b"))
        assertEquals(1, q.total)
        assertEquals("c", q.current?.id)
    }

    @Test
    fun emptyListIsFinished() {
        val q = QuickPhotoQueue.of(emptyList())
        assertTrue(q.finished)
        assertEquals(0, q.position)
        assertFalse(q.total > 0)
    }
}
